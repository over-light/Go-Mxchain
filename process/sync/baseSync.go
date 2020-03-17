package sync

import (
	"bytes"
	"math"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/logger"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/statusHandler"
	"github.com/ElrondNetwork/elrond-go/storage"
)

var log = logger.GetOrCreate("process/sync")

// sleepTime defines the time in milliseconds between each iteration made in syncBlocks method
const sleepTime = 5 * time.Millisecond

// HdrInfo hold the data related to a header
type HdrInfo struct {
	Nonce uint64
	Hash  []byte
}

type notarizedInfo struct {
	lastNotarized           map[uint32]*HdrInfo
	finalNotarized          map[uint32]*HdrInfo
	blockWithLastNotarized  map[uint32]uint64
	blockWithFinalNotarized map[uint32]uint64
	startNonce              uint64
}

type baseBootstrap struct {
	headers dataRetriever.HeadersPool

	chainHandler   data.ChainHandler
	blockProcessor process.BlockProcessor
	store          dataRetriever.StorageService

	rounder           consensus.Rounder
	hasher            hashing.Hasher
	marshalizer       marshal.Marshalizer
	epochHandler      dataRetriever.EpochHandler
	forkDetector      process.ForkDetector
	requestHandler    process.RequestHandler
	shardCoordinator  sharding.Coordinator
	accounts          state.AccountsAdapter
	blockBootstrapper blockBootstrapper
	blackListHandler  process.BlackListHandler

	mutHeader     sync.RWMutex
	headerNonce   *uint64
	headerhash    []byte
	chRcvHdrNonce chan bool
	chRcvHdrHash  chan bool

	requestedHashes process.RequiredDataPool

	statusHandler core.AppStatusHandler

	chStopSync chan bool
	waitTime   time.Duration

	mutNodeState          sync.RWMutex
	isNodeSynchronized    bool
	isNodeStateCalculated bool
	hasLastBlock          bool
	roundIndex            int64

	forkInfo *process.ForkInfo

	mutRcvHdrNonce        sync.RWMutex
	mutRcvHdrHash         sync.RWMutex
	syncStateListeners    []func(bool)
	mutSyncStateListeners sync.RWMutex
	uint64Converter       typeConverters.Uint64ByteSliceConverter
	requestsWithTimeout   uint32
	syncWithErrors        uint32

	requestMiniBlocks func(headerHandler data.HeaderHandler)

	networkWatcher    process.NetworkConnectionWatcher
	getHeaderFromPool func([]byte) (data.HeaderHandler, error)

	headerStore          storage.Storer
	headerNonceHashStore storage.Storer
	syncStarter          syncStarter
	bootStorer           process.BootStorer
	storageBootstrapper  process.BootstrapperFromStorage

	chRcvMiniBlocks    chan bool
	mutRcvMiniBlocks   sync.Mutex
	miniBlocksResolver process.MiniBlocksResolver
	poolsHolder        dataRetriever.PoolsHolder
	mutRequestHeaders  sync.Mutex
}

// setRequestedHeaderNonce method sets the header nonce requested by the sync mechanism
func (boot *baseBootstrap) setRequestedHeaderNonce(nonce *uint64) {
	boot.mutHeader.Lock()
	boot.headerNonce = nonce
	boot.mutHeader.Unlock()
}

// setRequestedHeaderHash method sets the header hash requested by the sync mechanism
func (boot *baseBootstrap) setRequestedHeaderHash(hash []byte) {
	boot.mutHeader.Lock()
	boot.headerhash = hash
	boot.mutHeader.Unlock()
}

// requestedHeaderNonce method gets the header nonce requested by the sync mechanism
func (boot *baseBootstrap) requestedHeaderNonce() *uint64 {
	boot.mutHeader.RLock()
	defer boot.mutHeader.RUnlock()
	return boot.headerNonce
}

// requestedHeaderHash method gets the header hash requested by the sync mechanism
func (boot *baseBootstrap) requestedHeaderHash() []byte {
	boot.mutHeader.RLock()
	defer boot.mutHeader.RUnlock()
	return boot.headerhash
}

func (boot *baseBootstrap) processReceivedHeader(headerHandler data.HeaderHandler, headerHash []byte) {
	if boot.shardCoordinator.SelfId() != headerHandler.GetShardID() {
		return
	}

	log.Trace("received header from network",
		"shard", headerHandler.GetShardID(),
		"round", headerHandler.GetRound(),
		"nonce", headerHandler.GetNonce(),
		"hash", headerHash,
	)

	err := boot.forkDetector.AddHeader(headerHandler, headerHash, process.BHReceived, nil, nil)
	if err != nil {
		log.Debug("forkDetector.AddHeader", "error", err.Error())
	}

	go boot.requestMiniBlocks(headerHandler)

	boot.confirmHeaderReceivedByNonce(headerHandler, headerHash)
	boot.confirmHeaderReceivedByHash(headerHandler, headerHash)
}

func (boot *baseBootstrap) confirmHeaderReceivedByNonce(headerHandler data.HeaderHandler, hdrHash []byte) {
	boot.mutRcvHdrNonce.Lock()
	n := boot.requestedHeaderNonce()
	if n != nil && *n == headerHandler.GetNonce() {
		log.Debug("received requested header from network",
			"shard", headerHandler.GetShardID(),
			"round", headerHandler.GetRound(),
			"nonce", headerHandler.GetNonce(),
			"hash", hdrHash,
		)
		boot.setRequestedHeaderNonce(nil)
		boot.mutRcvHdrNonce.Unlock()
		boot.chRcvHdrNonce <- true

		return
	}

	boot.mutRcvHdrNonce.Unlock()
}

func (boot *baseBootstrap) confirmHeaderReceivedByHash(headerHandler data.HeaderHandler, hdrHash []byte) {
	boot.mutRcvHdrHash.Lock()
	hash := boot.requestedHeaderHash()
	if hash != nil && bytes.Equal(hash, hdrHash) {
		log.Debug("received requested header from network",
			"shard", headerHandler.GetShardID(),
			"round", headerHandler.GetRound(),
			"nonce", headerHandler.GetNonce(),
			"hash", hash,
		)
		boot.setRequestedHeaderHash(nil)
		boot.mutRcvHdrHash.Unlock()
		boot.chRcvHdrHash <- true

		return
	}
	boot.mutRcvHdrHash.Unlock()
}

// AddSyncStateListener adds a syncStateListener that get notified each time the sync status of the node changes
func (boot *baseBootstrap) AddSyncStateListener(syncStateListener func(isSyncing bool)) {
	boot.mutSyncStateListeners.Lock()
	boot.syncStateListeners = append(boot.syncStateListeners, syncStateListener)
	boot.mutSyncStateListeners.Unlock()
}

// SetStatusHandler will set the instance of the AppStatusHandler
func (boot *baseBootstrap) SetStatusHandler(handler core.AppStatusHandler) error {
	if handler == nil || handler.IsInterfaceNil() {
		return process.ErrNilAppStatusHandler
	}
	boot.statusHandler = handler

	return nil
}

func (boot *baseBootstrap) notifySyncStateListeners(isNodeSynchronized bool) {
	boot.mutSyncStateListeners.RLock()
	for i := 0; i < len(boot.syncStateListeners); i++ {
		go boot.syncStateListeners[i](isNodeSynchronized)
	}
	boot.mutSyncStateListeners.RUnlock()
}

// getNonceForNextBlock will get the nonce for the next block we should request
func (boot *baseBootstrap) getNonceForNextBlock() uint64 {
	nonce := uint64(1) // first block nonce after genesis block
	currentBlockHeader := boot.chainHandler.GetCurrentBlockHeader()
	if !check.IfNil(currentBlockHeader) {
		nonce = currentBlockHeader.GetNonce() + 1
	}
	return nonce
}

// waitForHeaderNonce method wait for header with the requested nonce to be received
func (boot *baseBootstrap) waitForHeaderNonce() error {
	select {
	case <-boot.chRcvHdrNonce:
		return nil
	case <-time.After(boot.waitTime):
		return process.ErrTimeIsOut
	}
}

// waitForHeaderHash method wait for header with the requested hash to be received
func (boot *baseBootstrap) waitForHeaderHash() error {
	select {
	case <-boot.chRcvHdrHash:
		return nil
	case <-time.After(boot.waitTime):
		return process.ErrTimeIsOut
	}
}

func (boot *baseBootstrap) computeNodeState() {
	boot.mutNodeState.Lock()
	defer boot.mutNodeState.Unlock()

	isNodeStateCalculatedInCurrentRound := boot.roundIndex == boot.rounder.Index() && boot.isNodeStateCalculated
	if isNodeStateCalculatedInCurrentRound {
		return
	}

	boot.forkInfo = boot.forkDetector.CheckFork()

	if check.IfNil(boot.chainHandler.GetCurrentBlockHeader()) {
		boot.hasLastBlock = boot.forkDetector.ProbableHighestNonce() == 0
	} else {
		boot.hasLastBlock = boot.forkDetector.ProbableHighestNonce() <= boot.chainHandler.GetCurrentBlockHeader().GetNonce()
	}

	isNodeConnectedToTheNetwork := boot.networkWatcher.IsConnectedToTheNetwork()
	isNodeSynchronized := !boot.forkInfo.IsDetected && boot.hasLastBlock && isNodeConnectedToTheNetwork
	if isNodeSynchronized != boot.isNodeSynchronized {
		log.Debug("node has changed its synchronized state",
			"state", isNodeSynchronized,
		)
	}

	boot.isNodeSynchronized = isNodeSynchronized
	boot.isNodeStateCalculated = true
	boot.roundIndex = boot.rounder.Index()
	boot.notifySyncStateListeners(isNodeSynchronized)

	result := uint64(1)
	if isNodeSynchronized {
		result = uint64(0)
	}

	boot.statusHandler.SetUInt64Value(core.MetricIsSyncing, result)

	if boot.shouldTryToRequestHeaders() {
		go boot.requestHeadersIfSyncIsStuck()
	}
}

func (boot *baseBootstrap) shouldTryToRequestHeaders() bool {
	if boot.rounder.Index() < 0 {
		return false
	}
	if boot.isForcedRollBackOneBlock() {
		return false
	}
	if boot.isForcedRollBackToNonce() {
		return false
	}
	if !boot.isNodeSynchronized {
		return true
	}

	return boot.rounder.Index()%process.RoundModulusTriggerWhenSyncIsStuck == 0
}

func (boot *baseBootstrap) requestHeadersIfSyncIsStuck() {
	lastSyncedRound := uint64(0)
	currHeader := boot.chainHandler.GetCurrentBlockHeader()
	if !check.IfNil(currHeader) {
		lastSyncedRound = currHeader.GetRound()
	}

	roundDiff := uint64(boot.rounder.Index()) - lastSyncedRound
	if roundDiff <= process.MaxRoundsWithoutNewBlockReceived {
		return
	}

	fromNonce := boot.getNonceForNextBlock()
	numHeadersToRequest := core.MinUint64(process.MaxHeadersToRequestInAdvance, roundDiff-1)
	toNonce := fromNonce + numHeadersToRequest - 1

	if fromNonce > toNonce {
		return
	}

	log.Debug("requestHeadersIfSyncIsStuck",
		"from nonce", fromNonce,
		"to nonce", toNonce,
		"probable highest nonce", boot.forkDetector.ProbableHighestNonce())

	boot.requestHeaders(fromNonce, toNonce)
}

func (boot *baseBootstrap) removeHeaderFromPools(header data.HeaderHandler) []byte {
	hash, err := core.CalculateHash(boot.marshalizer, boot.hasher, header)
	if err != nil {
		log.Debug("CalculateHash", "error", err.Error())
		return nil
	}

	log.Debug("removeHeaderFromPools",
		"shard", header.GetShardID(),
		"epoch", header.GetEpoch(),
		"round", header.GetRound(),
		"nonce", header.GetNonce(),
		"hash", hash)

	boot.headers.RemoveHeaderByHash(hash)

	return hash
}

func (boot *baseBootstrap) removeHeadersWithNonceFromPool(nonce uint64) {
	log.Debug("removeHeadersWithNonceFromPool",
		"shard", boot.shardCoordinator.SelfId(),
		"nonce", nonce)

	boot.headers.RemoveHeaderByNonceAndShardId(nonce, boot.shardCoordinator.SelfId())
}

func (boot *baseBootstrap) cleanCachesAndStorageOnRollback(header data.HeaderHandler) {
	hash := boot.removeHeaderFromPools(header)
	boot.forkDetector.RemoveHeader(header.GetNonce(), hash)
	nonceToByteSlice := boot.uint64Converter.ToByteSlice(header.GetNonce())
	_ = boot.headerNonceHashStore.Remove(nonceToByteSlice)
}

// checkBootstrapNilParameters will check the imput parameters for nil values
func checkBootstrapNilParameters(arguments ArgBaseBootstrapper) error {
	if check.IfNil(arguments.ChainHandler) {
		return process.ErrNilBlockChain
	}
	if check.IfNil(arguments.Rounder) {
		return process.ErrNilRounder
	}
	if check.IfNil(arguments.BlockProcessor) {
		return process.ErrNilBlockProcessor
	}
	if check.IfNil(arguments.Hasher) {
		return process.ErrNilHasher
	}
	if check.IfNil(arguments.Marshalizer) {
		return process.ErrNilMarshalizer
	}
	if check.IfNil(arguments.ForkDetector) {
		return process.ErrNilForkDetector
	}
	if check.IfNil(arguments.RequestHandler) {
		return process.ErrNilRequestHandler
	}
	if check.IfNil(arguments.ShardCoordinator) {
		return process.ErrNilShardCoordinator
	}
	if check.IfNil(arguments.Accounts) {
		return process.ErrNilAccountsAdapter
	}
	if check.IfNil(arguments.Store) {
		return process.ErrNilStore
	}
	if check.IfNil(arguments.BlackListHandler) {
		return process.ErrNilBlackListHandler
	}
	if check.IfNil(arguments.NetworkWatcher) {
		return process.ErrNilNetworkWatcher
	}
	if check.IfNil(arguments.BootStorer) {
		return process.ErrNilBootStorer
	}
	if check.IfNil(arguments.MiniBlocksResolver) {
		return process.ErrNilMiniBlocksResolver
	}

	return nil
}

func (boot *baseBootstrap) requestHeadersFromNonceIfMissing(fromNonce uint64) {
	toNonce := core.MinUint64(fromNonce+process.MaxHeadersToRequestInAdvance-1, boot.forkDetector.ProbableHighestNonce())

	if fromNonce > toNonce {
		return
	}

	log.Debug("requestHeadersFromNonceIfMissing",
		"from nonce", fromNonce,
		"to nonce", toNonce,
		"probable highest nonce", boot.forkDetector.ProbableHighestNonce())

	boot.requestHeaders(fromNonce, toNonce)
}

// StopSync method will stop SyncBlocks
func (boot *baseBootstrap) StopSync() {
	boot.chStopSync <- true
}

// syncBlocks method calls repeatedly synchronization method SyncBlock
func (boot *baseBootstrap) syncBlocks() {
	for {
		time.Sleep(sleepTime)

		if !boot.networkWatcher.IsConnectedToTheNetwork() {
			continue
		}

		select {
		case <-boot.chStopSync:
			return
		default:
			err := boot.syncStarter.SyncBlock()
			if err != nil {
				log.Debug("SyncBlock", "error", err.Error())
			}
		}
	}
}

func (boot *baseBootstrap) doJobOnSyncBlockFail(headerHandler data.HeaderHandler, err error) {
	boot.syncWithErrors++

	if err == process.ErrTimeIsOut {
		boot.requestsWithTimeout++
	}

	allowedRequestsWithTimeOutLimitReached := boot.requestsWithTimeout >= process.MaxRequestsWithTimeoutAllowed
	isInProperRound := process.IsInProperRound(boot.rounder.Index())

	shouldRollBack := err != process.ErrTimeIsOut || (allowedRequestsWithTimeOutLimitReached && isInProperRound)
	if shouldRollBack {
		if !check.IfNil(headerHandler) {
			hash := boot.removeHeaderFromPools(headerHandler)
			boot.forkDetector.RemoveHeader(headerHandler.GetNonce(), hash)
		}

		errNotCritical := boot.rollBack(false)
		if errNotCritical != nil {
			log.Debug("rollBack", "error", errNotCritical.Error())
		}
	}

	allowedSyncWithErrorsLimitReached := boot.syncWithErrors >= process.MaxSyncWithErrorsAllowed
	if allowedSyncWithErrorsLimitReached && isInProperRound {
		boot.forkDetector.ResetProbableHighestNonce()
		boot.removeHeadersWithNonceFromPool(boot.getNonceForNextBlock())
	}
}

// syncBlock method actually does the synchronization. It requests the next block header from the pool
// and if it is not found there it will be requested from the network. After the header is received,
// it requests the block body in the same way(pool and than, if it is not found in the pool, from network).
// If either header and body are received the ProcessBlock and CommitBlock method will be called successively.
// These methods will execute the block and its transactions. Finally if everything works, the block will be committed
// in the blockchain, and all this mechanism will be reiterated for the next block.
func (boot *baseBootstrap) syncBlock() error {
	boot.computeNodeState()
	nodeState := boot.GetNodeState()
	if nodeState != core.NsNotSynchronized {
		return nil
	}

	defer func() {
		boot.mutNodeState.Lock()
		boot.isNodeStateCalculated = false
		boot.mutNodeState.Unlock()
	}()

	if boot.forkInfo.IsDetected {
		boot.statusHandler.Increment(core.MetricNumTimesInForkChoice)

		if boot.isForcedRollBackOneBlock() {
			log.Debug("roll back one block has been forced")
			boot.rollBackOneBlockForced()
			return nil
		}

		if boot.isForcedRollBackToNonce() {
			log.Debug("roll back to nonce has been forced", "nonce", boot.forkInfo.Nonce)
			boot.rollBackToNonceForced()
			return nil
		}

		log.Debug("fork detected",
			"nonce", boot.forkInfo.Nonce,
			"hash", boot.forkInfo.Hash,
		)
		err := boot.rollBack(true)
		if err != nil {
			return err
		}
	}

	var hdr data.HeaderHandler
	var err error

	defer func() {
		if err != nil {
			boot.doJobOnSyncBlockFail(hdr, err)
		}
	}()

	hdr, err = boot.getNextHeaderRequestingIfMissing()
	if err != nil {
		return err
	}

	go boot.requestHeadersFromNonceIfMissing(hdr.GetNonce() + 1)

	blockBody, err := boot.blockBootstrapper.getBlockBodyRequestingIfMissing(hdr)
	if err != nil {
		return err
	}

	haveTime := func() time.Duration {
		return boot.rounder.TimeDuration()
	}

	startTime := time.Now()
	err = boot.blockProcessor.ProcessBlock(hdr, blockBody, haveTime)
	elapsedTime := time.Since(startTime)
	log.Debug("elapsed time to process block",
		"time [s]", elapsedTime,
	)
	if err != nil {
		return err
	}

	startTime = time.Now()
	err = boot.blockProcessor.CommitBlock(hdr, blockBody)
	elapsedTime = time.Since(startTime)
	log.Debug("elapsed time to commit block",
		"time [s]", elapsedTime,
	)
	if err != nil {
		return err
	}

	log.Debug("block has been synced successfully",
		"nonce", hdr.GetNonce(),
	)

	boot.syncWithErrors = 0
	boot.requestsWithTimeout = 0

	return nil
}

// rollBack decides if rollBackOneBlock must be called
func (boot *baseBootstrap) rollBack(revertUsingForkNonce bool) error {
	if boot.headerStore == nil {
		return process.ErrNilHeadersStorage
	}
	if boot.headerNonceHashStore == nil {
		return process.ErrNilHeadersNonceHashStorage
	}

	log.Debug("starting roll back")
	for {
		currHeaderHash := boot.chainHandler.GetCurrentBlockHeaderHash()
		currHeader, err := boot.blockBootstrapper.getCurrHeader()
		if err != nil {
			return err
		}
		if !revertUsingForkNonce && currHeader.GetNonce() <= boot.forkDetector.GetHighestFinalBlockNonce() {
			return ErrRollBackBehindFinalHeader
		}

		shouldEndRollBack := revertUsingForkNonce && currHeader.GetNonce() < boot.forkInfo.Nonce
		if shouldEndRollBack {
			return ErrRollBackBehindForkNonce
		}

		currBlockBody, err := boot.blockBootstrapper.getBlockBody(currHeader)
		if err != nil {
			return err
		}
		prevHeaderHash := currHeader.GetPrevHash()
		prevHeader, err := boot.blockBootstrapper.getPrevHeader(currHeader, boot.headerStore)
		if err != nil {
			return err
		}
		prevBlockBody, err := boot.blockBootstrapper.getBlockBody(prevHeader)
		if err != nil {
			return err
		}

		log.Debug("roll back to block",
			"nonce", currHeader.GetNonce()-1,
			"hash", currHeader.GetPrevHash(),
		)
		log.Debug("highest final block nonce",
			"nonce", boot.forkDetector.GetHighestFinalBlockNonce(),
		)

		err = boot.rollBackOneBlock(
			currHeaderHash,
			currHeader,
			currBlockBody,
			prevHeaderHash,
			prevHeader,
			prevBlockBody)

		if err != nil {
			return err
		}

		_, _ = updateMetricsFromStorage(boot.store, boot.uint64Converter, boot.marshalizer, boot.statusHandler, prevHeader.GetNonce())

		err = boot.bootStorer.SaveLastRound(int64(prevHeader.GetRound()))
		if err != nil {
			log.Debug("save last round in storage",
				"error", err.Error(),
				"round", prevHeader.GetRound(),
			)
		}

		shouldAddHeaderToBlackList := revertUsingForkNonce && boot.blockBootstrapper.isForkTriggeredByMeta()
		if shouldAddHeaderToBlackList {
			process.AddHeaderToBlackList(boot.blackListHandler, currHeaderHash)
		}

		shouldContinueRollBack := revertUsingForkNonce && currHeader.GetNonce() > boot.forkInfo.Nonce
		if shouldContinueRollBack {
			continue
		}

		break
	}

	log.Debug("ending roll back")
	return nil
}

func (boot *baseBootstrap) rollBackOneBlock(
	currHeaderHash []byte,
	currHeader data.HeaderHandler,
	currBlockBody data.BodyHandler,
	prevHeaderHash []byte,
	prevHeader data.HeaderHandler,
	prevBlockBody data.BodyHandler,
) error {

	var err error

	defer func() {
		if err != nil {
			boot.restoreState(currHeaderHash, currHeader, currBlockBody)
		}
	}()

	if currHeader.GetNonce() > 1 {
		err = boot.setCurrentBlockInfo(prevHeaderHash, prevHeader, prevBlockBody)
		if err != nil {
			return err
		}
	} else {
		err = boot.setCurrentBlockInfo(nil, nil, nil)
		if err != nil {
			return err
		}
	}

	err = boot.blockProcessor.RevertStateToBlock(prevHeader)
	if err != nil {
		return err
	}
	boot.blockProcessor.PruneStateOnRollback(currHeader, prevHeader)

	err = boot.blockProcessor.RestoreBlockIntoPools(currHeader, currBlockBody)
	if err != nil {
		return err
	}

	boot.cleanCachesAndStorageOnRollback(currHeader)

	return nil
}

func (boot *baseBootstrap) getNextHeaderRequestingIfMissing() (data.HeaderHandler, error) {
	nonce := boot.getNonceForNextBlock()

	boot.setRequestedHeaderHash(nil)
	boot.setRequestedHeaderNonce(nil)

	hash := boot.forkDetector.GetNotarizedHeaderHash(nonce)
	if boot.forkInfo.IsDetected {
		hash = boot.forkInfo.Hash
	}

	if hash != nil {
		return boot.blockBootstrapper.getHeaderWithHashRequestingIfMissing(hash)
	}

	return boot.blockBootstrapper.getHeaderWithNonceRequestingIfMissing(nonce)
}

func (boot *baseBootstrap) isForcedRollBackOneBlock() bool {
	return boot.forkInfo.IsDetected &&
		boot.forkInfo.Nonce == math.MaxUint64 &&
		boot.forkInfo.Hash == nil
}

func (boot *baseBootstrap) isForcedRollBackToNonce() bool {
	return boot.forkInfo.IsDetected &&
		boot.forkInfo.Round == math.MaxUint64 &&
		boot.forkInfo.Hash == nil
}

func (boot *baseBootstrap) rollBackOneBlockForced() {
	err := boot.rollBack(false)
	if err != nil {
		log.Debug("rollBackOneBlockForced", "error", err.Error())
	}

	boot.forkDetector.ResetFork()
	boot.removeHeadersWithNonceFromPool(boot.getNonceForNextBlock())
}

func (boot *baseBootstrap) rollBackToNonceForced() {
	err := boot.rollBack(true)
	if err != nil {
		log.Debug("rollBackToNonceForced", "error", err.Error())
	}

	boot.forkDetector.ResetProbableHighestNonce()
	boot.removeHeadersWithNonceFromPool(boot.getNonceForNextBlock())
}

func (boot *baseBootstrap) restoreState(
	currHeaderHash []byte,
	currHeader data.HeaderHandler,
	currBlockBody data.BodyHandler,
) {
	log.Debug("revert state to header",
		"nonce", currHeader.GetNonce(),
		"hash", currHeaderHash)

	err := boot.chainHandler.SetCurrentBlockHeader(currHeader)
	if err != nil {
		log.Debug("SetCurrentBlockHeader", "error", err.Error())
	}

	err = boot.chainHandler.SetCurrentBlockBody(currBlockBody)
	if err != nil {
		log.Debug("SetCurrentBlockBody", "error", err.Error())
	}

	boot.chainHandler.SetCurrentBlockHeaderHash(currHeaderHash)

	err = boot.blockProcessor.RevertStateToBlock(currHeader)
	if err != nil {
		log.Debug("RevertState", "error", err.Error())
	}
}

func (boot *baseBootstrap) setCurrentBlockInfo(
	headerHash []byte,
	header data.HeaderHandler,
	body data.BodyHandler,
) error {

	err := boot.chainHandler.SetCurrentBlockHeader(header)
	if err != nil {
		return err
	}

	err = boot.chainHandler.SetCurrentBlockBody(body)
	if err != nil {
		return err
	}

	boot.chainHandler.SetCurrentBlockHeaderHash(headerHash)

	return nil
}

// setRequestedMiniBlocks method sets the body hash requested by the sync mechanism
func (boot *baseBootstrap) setRequestedMiniBlocks(hashes [][]byte) {
	boot.requestedHashes.SetHashes(hashes)
}

// receivedBodyHash method is a call back function which is called when a new body is added
// in the block bodies pool
func (boot *baseBootstrap) receivedBodyHash(hash []byte) {
	boot.mutRcvMiniBlocks.Lock()
	if len(boot.requestedHashes.ExpectedData()) == 0 {
		boot.mutRcvMiniBlocks.Unlock()
		return
	}

	boot.requestedHashes.SetReceivedHash(hash)
	if boot.requestedHashes.ReceivedAll() {
		log.Debug("received all the requested mini blocks from network")
		boot.setRequestedMiniBlocks(nil)
		boot.mutRcvMiniBlocks.Unlock()
		boot.chRcvMiniBlocks <- true
	} else {
		boot.mutRcvMiniBlocks.Unlock()
	}
}

// requestMiniBlocksByHashes method requests a block body from network when it is not found in the pool
func (boot *baseBootstrap) requestMiniBlocksByHashes(hashes [][]byte) {
	boot.setRequestedMiniBlocks(hashes)
	log.Debug("requesting mini blocks from network",
		"num miniblocks", len(hashes),
	)
	boot.requestHandler.RequestMiniBlocks(boot.shardCoordinator.SelfId(), hashes)
}

// getMiniBlocksRequestingIfMissing method gets the body with given nonce from pool, if it exist there,
// and if not it will be requested from network
// the func returns interface{} as to match the next implementations for block body fetchers
// that will be added. The block executor should decide by parsing the header block body type value
// what kind of block body received.
func (boot *baseBootstrap) getMiniBlocksRequestingIfMissing(hashes [][]byte) (block.MiniBlockSlice, error) {
	miniBlocks, missingMiniBlocksHashes := boot.miniBlocksResolver.GetMiniBlocksFromPool(hashes)
	if len(missingMiniBlocksHashes) > 0 {
		_ = process.EmptyChannel(boot.chRcvMiniBlocks)
		boot.requestMiniBlocksByHashes(missingMiniBlocksHashes)
		err := boot.waitForMiniBlocks()
		if err != nil {
			return nil, err
		}

		receivedMiniBlocks, unreceivedMiniBlocksHashes := boot.miniBlocksResolver.GetMiniBlocksFromPool(missingMiniBlocksHashes)
		if len(unreceivedMiniBlocksHashes) > 0 {
			return nil, process.ErrMissingBody
		}

		miniBlocks = append(miniBlocks, receivedMiniBlocks...)
	}

	return miniBlocks, nil
}

// waitForMiniBlocks method wait for body with the requested nonce to be received
func (boot *baseBootstrap) waitForMiniBlocks() error {
	select {
	case <-boot.chRcvMiniBlocks:
		return nil
	case <-time.After(boot.waitTime):
		return process.ErrTimeIsOut
	}
}

func (boot *baseBootstrap) init() {
	boot.forkInfo = process.NewForkInfo()

	boot.chRcvHdrNonce = make(chan bool)
	boot.chRcvHdrHash = make(chan bool)
	boot.chRcvMiniBlocks = make(chan bool)

	boot.setRequestedHeaderNonce(nil)
	boot.setRequestedHeaderHash(nil)
	boot.setRequestedMiniBlocks(nil)

	boot.poolsHolder.MiniBlocks().RegisterHandler(boot.receivedBodyHash)
	boot.headers.RegisterHandler(boot.processReceivedHeader)

	boot.chStopSync = make(chan bool)

	boot.statusHandler = statusHandler.NewNilStatusHandler()

	boot.syncStateListeners = make([]func(bool), 0)
	boot.requestedHashes = process.RequiredDataPool{}
}

func (boot *baseBootstrap) requestHeaders(fromNonce uint64, toNonce uint64) {
	boot.mutRequestHeaders.Lock()
	defer boot.mutRequestHeaders.Unlock()

	for currentNonce := fromNonce; currentNonce <= toNonce; currentNonce++ {
		haveHeader := boot.blockBootstrapper.haveHeaderInPoolWithNonce(currentNonce)
		if haveHeader {
			continue
		}

		boot.blockBootstrapper.requestHeaderByNonce(currentNonce)
	}
}

// GetNodeState method returns the sync state of the node. If it returns 'NsNotSynchronized', this means that the node
// is not synchronized yet and it has to continue the bootstrapping mechanism. If it returns 'NsSynchronized', this means
// that the node is already synced and it can participate to the consensus. This method could also returns 'NsNotCalculated'
// which means that the state of the node in the current round is not calculated yet. Note that when the node is not
// connected to the network, GetNodeState could return 'NsNotSynchronized' but the SyncBlock is not automatically called.
func (boot *baseBootstrap) GetNodeState() core.NodeState {
	boot.mutNodeState.RLock()
	isNodeStateCalculatedInCurrentRound := boot.roundIndex == boot.rounder.Index() && boot.isNodeStateCalculated
	isNodeSynchronized := boot.isNodeSynchronized
	boot.mutNodeState.RUnlock()

	if !isNodeStateCalculatedInCurrentRound {
		return core.NsNotCalculated
	}

	if isNodeSynchronized {
		return core.NsSynchronized
	}

	return core.NsNotSynchronized
}
