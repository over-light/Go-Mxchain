package broadcast

import (
	"bytes"
	"strings"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/spos"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/alarm"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

const prefixHeaderAlarm = "header"
const prefixDelayDataAlarm = "delay"

// DelayedBlockBroadcasterArgs holds the arguments to create a delayed block broadcaster
type DelayedBlockBroadcasterArgs struct {
	InterceptorsContainer process.InterceptorsContainer
	HeadersSubscriber     consensus.HeadersPoolSubscriber
	ShardCoordinator      sharding.Coordinator
	LeaderCacheSize       uint32
	ValidatorCacheSize    uint32
}

type delayedBroadcastData struct {
	headerHash           []byte
	header               data.HeaderHandler
	metaMiniBlocksData   map[uint32][]byte
	metaTransactionsData map[string][][]byte
	miniBlocksData       map[uint32][]byte
	miniBlockHashes      map[string]map[string]struct{}
	transactions         map[string][][]byte
	order                uint32
}

// delayedBroadcaster exposes functionality for handling the consensus members broadcasting of delay data
type delayedBroadcaster interface {
	SetLeaderData(data *delayedBroadcastData) error
	SetValidatorData(data *delayedBroadcastData) error
	SetBroadcastHandlers(
		mbBroadcast func(mbData map[uint32][]byte) error,
		txBroadcast func(txData map[string][][]byte) error,
		headerBroadcast func(header data.HeaderHandler) error,

	) error
	Close()
}

// timersScheduler exposes functionality for scheduling multiple timers
type timersScheduler interface {
	Add(callback func(alarmID string), duration time.Duration, alarmID string)
	Cancel(alarmID string)
	Close()
	IsInterfaceNil() bool
}

type headerDataForValidator struct {
	round        uint64
	prevRandSeed []byte
}

type delayedBlockBroadcaster struct {
	alarm                      timersScheduler
	interceptorsContainer      process.InterceptorsContainer
	shardCoordinator           sharding.Coordinator
	headersSubscriber          consensus.HeadersPoolSubscriber
	valBroadcastData           []*delayedBroadcastData
	delayedBroadcastData       []*delayedBroadcastData
	maxDelayCacheSize          uint32
	maxValidatorDelayCacheSize uint32
	mutDataForBroadcast        sync.RWMutex
	broadcastMiniblocksData    func(mbData map[uint32][]byte) error
	broadcastTxsData           func(txData map[string][][]byte) error
	broadcastHeader            func(header data.HeaderHandler) error
}

// NewDelayedBlockBroadcaster create a new instance of a delayed block data broadcaster
func NewDelayedBlockBroadcaster(args *DelayedBlockBroadcasterArgs) (*delayedBlockBroadcaster, error) {
	if check.IfNil(args.ShardCoordinator) {
		return nil, spos.ErrNilShardCoordinator
	}
	if check.IfNil(args.InterceptorsContainer) {
		return nil, spos.ErrNilInterceptorsContainer
	}
	if check.IfNil(args.HeadersSubscriber) {
		return nil, spos.ErrNilHeadersSubscriber
	}

	dbb := &delayedBlockBroadcaster{
		alarm:                      alarm.NewAlarmScheduler(),
		shardCoordinator:           args.ShardCoordinator,
		interceptorsContainer:      args.InterceptorsContainer,
		headersSubscriber:          args.HeadersSubscriber,
		valBroadcastData:           make([]*delayedBroadcastData, 0),
		delayedBroadcastData:       make([]*delayedBroadcastData, 0),
		maxDelayCacheSize:          args.LeaderCacheSize,
		maxValidatorDelayCacheSize: args.ValidatorCacheSize,
		mutDataForBroadcast:        sync.RWMutex{},
	}

	dbb.headersSubscriber.RegisterHandler(dbb.headerReceived)
	err := dbb.registerHeaderInterceptorCallback(dbb.interceptedHeaderData)
	if err != nil {
		return nil, err
	}

	err = dbb.registerMiniBlockInterceptorCallback(dbb.interceptedMiniBlockData)
	if err != nil {
		return nil, err
	}

	return dbb, nil
}

// SetLeaderData sets the data for consensus leader delayed broadcast
func (dbb *delayedBlockBroadcaster) SetLeaderData(broadcastData *delayedBroadcastData) error {
	if broadcastData == nil {
		return spos.ErrNilParameter
	}

	dbb.mutDataForBroadcast.Lock()
	defer dbb.mutDataForBroadcast.Unlock()

	dbb.delayedBroadcastData = append(dbb.delayedBroadcastData, broadcastData)
	if len(dbb.delayedBroadcastData) > int(dbb.maxDelayCacheSize) {
		dbb.broadcastDelayedData(dbb.delayedBroadcastData[:1])
		dbb.delayedBroadcastData = dbb.delayedBroadcastData[1:]
	}

	return nil
}

// SetValidatorData sets the data for cnsensus validator delayed broadcast
func (dbb *delayedBlockBroadcaster) SetValidatorData(broadcastData *delayedBroadcastData) error {
	if broadcastData == nil {
		return spos.ErrNilParameter
	}

	dbb.mutDataForBroadcast.Lock()
	defer dbb.mutDataForBroadcast.Unlock()

	// set alarm only for validators that are aware that the block was finalized
	if broadcastData.header.GetSignature() != nil {
		duration := validatorDelayPerOrder * time.Duration(broadcastData.order)
		dbb.alarm.Add(dbb.headerAlarmExpired, duration, prefixHeaderAlarm+string(broadcastData.headerHash))
	}

	dbb.valBroadcastData = append(dbb.valBroadcastData, broadcastData)
	if len(dbb.valBroadcastData) > int(dbb.maxValidatorDelayCacheSize) {
		dbb.alarm.Cancel(prefixHeaderAlarm + string(dbb.valBroadcastData[0].headerHash))
		dbb.alarm.Cancel(prefixDelayDataAlarm + string(dbb.valBroadcastData[0].headerHash))
		dbb.valBroadcastData = dbb.valBroadcastData[1:]
	}

	return nil
}

// SetBroadcastHandlers sets the broadcast handlers for miniBlocks and transactions
func (dbb *delayedBlockBroadcaster) SetBroadcastHandlers(
	mbBroadcast func(mbData map[uint32][]byte) error,
	txBroadcast func(txData map[string][][]byte) error,
	headerBroadcast func(header data.HeaderHandler) error,
) error {
	if mbBroadcast == nil || txBroadcast == nil || headerBroadcast == nil {
		return spos.ErrNilParameter
	}

	dbb.mutDataForBroadcast.RLock()
	defer dbb.mutDataForBroadcast.RUnlock()

	dbb.broadcastMiniblocksData = mbBroadcast
	dbb.broadcastTxsData = txBroadcast
	dbb.broadcastHeader = headerBroadcast

	return nil
}

// Close closes all the started infinite looping goroutines and subcomponents
func (dbb *delayedBlockBroadcaster) Close() {
	dbb.alarm.Close()
}

func (dbb *delayedBlockBroadcaster) headerReceived(headerHandler data.HeaderHandler, _ []byte) {
	dbb.mutDataForBroadcast.RLock()
	defer dbb.mutDataForBroadcast.RUnlock()

	if len(dbb.delayedBroadcastData) == 0 {
		return
	}
	if headerHandler.GetShardID() != core.MetachainShardId {
		return
	}

	headerHashes, dataForValidators, err := getShardDataFromMetaChainBlock(
		headerHandler,
		dbb.shardCoordinator.SelfId(),
	)
	if err != nil {
		log.Error("notifier headerReceived", "error", err.Error())
		return
	}
	if len(headerHashes) == 0 {
		return
	}

	go dbb.scheduleValidatorBroadcast(dataForValidators)
	go dbb.broadcastDataForHeaders(headerHashes)
}

func (dbb *delayedBlockBroadcaster) broadcastDataForHeaders(headerHashes [][]byte) {
	time.Sleep(core.ExtraDelayForBroadcastBlockInfo)

	dbb.mutDataForBroadcast.Lock()
	defer dbb.mutDataForBroadcast.Unlock()

	for i := len(dbb.delayedBroadcastData) - 1; i >= 0; i-- {
		for _, headerHash := range headerHashes {
			if bytes.Equal(dbb.delayedBroadcastData[i].headerHash, headerHash) {
				dbb.broadcastDelayedData(dbb.delayedBroadcastData[:i+1])
				dbb.delayedBroadcastData = dbb.delayedBroadcastData[i+1:]
				return
			}
		}
	}
}

func (dbb *delayedBlockBroadcaster) scheduleValidatorBroadcast(dataForValidators []*headerDataForValidator) {
	for _, headerData := range dataForValidators {
		for _, broadcastData := range dbb.valBroadcastData {
			sameRound := headerData.round == broadcastData.header.GetRound()
			samePrevRandomness := bytes.Equal(headerData.prevRandSeed, broadcastData.header.GetPrevRandSeed())
			if sameRound && samePrevRandomness {
				duration := validatorDelayPerOrder * time.Duration(broadcastData.order)
				dbb.alarm.Add(dbb.alarmExpired, duration, prefixDelayDataAlarm+string(broadcastData.headerHash))
			}
		}
	}
}

func (dbb *delayedBlockBroadcaster) alarmExpired(alarmID string) {
	headerHash := strings.TrimLeft(alarmID, prefixDelayDataAlarm)

	dbb.mutDataForBroadcast.Lock()
	defer dbb.mutDataForBroadcast.Unlock()
	for i, broadcastData := range dbb.valBroadcastData {
		if bytes.Equal(broadcastData.headerHash, []byte(headerHash)) {
			dbb.broadcastDelayedData([]*delayedBroadcastData{broadcastData})
			dbb.valBroadcastData = append(dbb.valBroadcastData[:i], dbb.valBroadcastData[i+1:]...)
			return
		}
	}
}

func (dbb *delayedBlockBroadcaster) headerAlarmExpired(alarmID string) {
	headerHash := strings.TrimLeft(alarmID, prefixHeaderAlarm)

	dbb.mutDataForBroadcast.Lock()
	defer dbb.mutDataForBroadcast.Unlock()

	var bData *delayedBroadcastData
	for _, broadcastData := range dbb.valBroadcastData {
		if bytes.Equal(broadcastData.headerHash, []byte(headerHash)) {
			bData = broadcastData
			break
		}
	}

	if bData == nil {
		log.Warn("headerAlarmExpired", "error", "alarm data is nil")
		return
	}

	// broadcast header
	err := dbb.broadcastHeader(bData.header)
	if err != nil {
		log.Warn("headerAlarmExpired", "error", err.Error())
	}

	// if metaChain broadcast meta data with extra delay
	if dbb.shardCoordinator.SelfId() == core.MetachainShardId {
		dbb.broadcastBlockData(bData.metaMiniBlocksData, bData.metaTransactionsData)
	}
}

func (dbb *delayedBlockBroadcaster) broadcastDelayedData(broadcastData []*delayedBroadcastData) {
	for _, bData := range broadcastData {
		dbb.broadcastBlockData(bData.miniBlocksData, bData.transactions)
	}
}

func (dbb *delayedBlockBroadcaster) broadcastBlockData(miniBlocks map[uint32][]byte, transactions map[string][][]byte) {
	err := dbb.broadcastMiniblocksData(miniBlocks)
	if err != nil {
		log.Error("broadcastBlockData miniblocks", "error", err.Error())
	}

	err = dbb.broadcastTxsData(transactions)
	if err != nil {
		log.Error("broadcastBlockData transactions", "error", err.Error())
	}
}

func getShardDataFromMetaChainBlock(
	headerHandler data.HeaderHandler,
	shardID uint32,
) ([][]byte, []*headerDataForValidator, error) {
	metaHeader, ok := headerHandler.(*block.MetaBlock)
	if !ok {
		return nil, nil, spos.ErrInvalidMetaHeader
	}

	dataForValidators := make([]*headerDataForValidator, 0)
	shardHeaderHashes := make([][]byte, 0)
	shardsInfo := metaHeader.GetShardInfo()
	for _, shardInfo := range shardsInfo {
		if shardInfo.ShardID == shardID {
			shardHeaderHashes = append(shardHeaderHashes, shardInfo.HeaderHash)
			headerData := &headerDataForValidator{
				round:        shardInfo.GetRound(),
				prevRandSeed: shardInfo.GetPrevRandSeed(),
			}
			dataForValidators = append(dataForValidators, headerData)
		}
	}
	return shardHeaderHashes, dataForValidators, nil
}

func (dbb *delayedBlockBroadcaster) registerHeaderInterceptorCallback(
	cb func(topic string, hash []byte, data interface{}),
) error {
	selfShardID := dbb.shardCoordinator.SelfId()

	if selfShardID == core.MetachainShardId {
		return dbb.registerInterceptorsCallbackForMeta(factory.MetachainBlocksTopic, cb)
	}

	identifierShardHeader := factory.ShardBlocksTopic + dbb.shardCoordinator.CommunicationIdentifier(selfShardID)
	interceptor, err := dbb.interceptorsContainer.Get(identifierShardHeader)
	if err != nil {
		return err
	}

	interceptor.RegisterHandler(cb)
	return nil
}

func (dbb *delayedBlockBroadcaster) registerMiniBlockInterceptorCallback(
	cb func(topic string, hash []byte, data interface{}),
) error {
	if dbb.shardCoordinator.SelfId() == core.MetachainShardId {
		return dbb.registerInterceptorsCallbackForMeta(factory.MiniBlocksTopic, cb)
	}

	return dbb.registerInterceptorsCallbackForShard(factory.MiniBlocksTopic, cb)
}

func (dbb *delayedBlockBroadcaster) registerInterceptorsCallbackForMeta(
	rootTopic string,
	cb func(topic string, hash []byte, data interface{}),
) error {
	identifierMiniBlocks := rootTopic + dbb.shardCoordinator.CommunicationIdentifier(core.AllShardId)
	interceptor, err := dbb.interceptorsContainer.Get(identifierMiniBlocks)
	if err != nil {
		return err
	}

	interceptor.RegisterHandler(cb)

	return nil
}

func (dbb *delayedBlockBroadcaster) registerInterceptorsCallbackForShard(
	rootTopic string,
	cb func(topic string, hash []byte, data interface{}),
) error {
	for idx := uint32(0); idx < dbb.shardCoordinator.NumberOfShards(); idx++ {
		// interested only in cross shard data
		if idx == dbb.shardCoordinator.SelfId() {
			continue
		}
		identifierMiniBlocks := rootTopic + dbb.shardCoordinator.CommunicationIdentifier(idx)
		interceptor, err := dbb.interceptorsContainer.Get(identifierMiniBlocks)
		if err != nil {
			return err
		}

		interceptor.RegisterHandler(cb)
	}

	return nil
}

func (dbb *delayedBlockBroadcaster) interceptedHeaderData(_ string, _ []byte, header interface{}) {
	dbb.mutDataForBroadcast.Lock()
	dbb.mutDataForBroadcast.Unlock()

	headerHandler, ok := header.(data.HeaderHandler)
	if !ok {
		log.Warn("interceptedHeaderData",
			"error", "not a header")
		return
	}

	for _, broadcastData := range dbb.valBroadcastData {
		samePrevRandSeed := bytes.Equal(broadcastData.header.GetPrevRandSeed(), headerHandler.GetPrevRandSeed())
		sameRound := broadcastData.header.GetRound() == headerHandler.GetRound()
		sameHeader := samePrevRandSeed && sameRound

		if sameHeader {
			// leader has broadcast the header so we can cancel the header alarm
			dbb.alarm.Cancel(prefixHeaderAlarm + string(broadcastData.headerHash))
		}
	}
}

func (dbb *delayedBlockBroadcaster) interceptedMiniBlockData(topic string, hash []byte, _ interface{}) {
	dbb.mutDataForBroadcast.Lock()
	defer dbb.mutDataForBroadcast.Unlock()

	for i, broadcastData := range dbb.valBroadcastData {
		mbHashesMap := broadcastData.miniBlockHashes
		if len(mbHashesMap) > 0 && len(mbHashesMap[topic]) > 0 {
			delete(broadcastData.miniBlockHashes[topic], string(hash))
			if len(mbHashesMap[topic]) == 0 {
				delete(mbHashesMap, topic)
			}
		}

		if len(mbHashesMap) == 0 {
			dbb.alarm.Cancel(string(broadcastData.headerHash))
			dbb.valBroadcastData = append(dbb.valBroadcastData[:i], dbb.valBroadcastData[i+1:]...)
			return
		}
	}
}

func (dbb *delayedBlockBroadcaster) extractMiniBlockHashesCrossFromMe(header data.HeaderHandler) map[string]map[string]struct{} {
	mbHashesForShards := make(map[string]map[string]struct{})
	for i := uint32(0); i < dbb.shardCoordinator.NumberOfShards(); i++ {
		if i == dbb.shardCoordinator.SelfId() {
			continue
		}
		topic := factory.MiniBlocksTopic + dbb.shardCoordinator.CommunicationIdentifier(i)

		mbHashesForShards[topic] = make(map[string]struct{})

		miniBlockHeaders := header.GetMiniBlockHeadersWithDst(i)
		for key := range miniBlockHeaders {
			mbHashesForShards[topic][key] = struct{}{}
		}
	}

	// TODO: Do we need to add also for metachain?

	return mbHashesForShards
}
