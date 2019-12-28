package block

import (
	"bytes"
	"fmt"
	"sort"
	"sync"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/display"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/logger"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
)

var log = logger.GetOrCreate("process/block")

type hashAndHdr struct {
	hdr  data.HeaderHandler
	hash []byte
}

type nonceAndHashInfo struct {
	hash  []byte
	nonce uint64
}

type hdrInfo struct {
	usedInBlock bool
	hdr         data.HeaderHandler
}

type hdrForBlock struct {
	missingHdrs                  uint32
	missingFinalityAttestingHdrs uint32
	highestHdrNonce              map[uint32]uint64
	mutHdrsForBlock              sync.RWMutex
	hdrHashAndInfo               map[string]*hdrInfo
}

type baseProcessor struct {
	shardCoordinator             sharding.Coordinator
	nodesCoordinator             sharding.NodesCoordinator
	specialAddressHandler        process.SpecialAddressHandler
	accounts                     state.AccountsAdapter
	forkDetector                 process.ForkDetector
	validatorStatisticsProcessor process.ValidatorStatisticsProcessor
	hasher                       hashing.Hasher
	marshalizer                  marshal.Marshalizer
	store                        dataRetriever.StorageService
	uint64Converter              typeConverters.Uint64ByteSliceConverter
	blockSizeThrottler           process.BlockSizeThrottler
	epochStartTrigger            process.EpochStartTriggerHandler
	headerValidator              process.HeaderConstructionValidator
	blockChainHook               process.BlockChainHookHandler
	txCoordinator                process.TransactionCoordinator
	rounder                      consensus.Rounder
	bootStorer                   process.BootStorer
	requestBlockBodyHandler      process.RequestBlockBodyHandler
	requestHandler               process.RequestHandler
	blockTracker                 process.BlockTracker

	hdrsForCurrBlock hdrForBlock

	appStatusHandler core.AppStatusHandler
}

func checkForNils(
	chainHandler data.ChainHandler,
	headerHandler data.HeaderHandler,
	bodyHandler data.BodyHandler,
) error {

	if chainHandler == nil || chainHandler.IsInterfaceNil() {
		return process.ErrNilBlockChain
	}
	if headerHandler == nil || headerHandler.IsInterfaceNil() {
		return process.ErrNilBlockHeader
	}
	if bodyHandler == nil || bodyHandler.IsInterfaceNil() {
		return process.ErrNilBlockBody
	}
	return nil
}

// SetAppStatusHandler method is used to set appStatusHandler
func (bp *baseProcessor) SetAppStatusHandler(ash core.AppStatusHandler) error {
	if ash == nil || ash.IsInterfaceNil() {
		return process.ErrNilAppStatusHandler
	}

	bp.appStatusHandler = ash
	return nil
}

// RevertAccountState reverts the account state for cleanup failed process
func (bp *baseProcessor) RevertAccountState() {
	err := bp.accounts.RevertToSnapshot(0)
	if err != nil {
		log.Debug("RevertToSnapshot", "error", err.Error())
	}

	err = bp.validatorStatisticsProcessor.RevertPeerStateToSnapshot(0)
	if err != nil {
		log.Debug("RevertPeerStateToSnapshot", "error", err.Error())
	}
}

// RevertStateToBlock recreates the state tries to the root hashes indicated by the provided header
func (bp *baseProcessor) RevertStateToBlock(header data.HeaderHandler) error {
	err := bp.accounts.RecreateTrie(header.GetRootHash())
	if err != nil {
		log.Debug("recreate trie with error for header",
			"nonce", header.GetNonce(),
			"hash", header.GetRootHash(),
		)

		return err
	}

	err = bp.validatorStatisticsProcessor.RevertPeerState(header)
	if err != nil {
		log.Debug("revert peer state with error for header",
			"nonce", header.GetNonce(),
			"validators root hash", header.GetValidatorStatsRootHash(),
		)

		return err
	}

	return nil
}

// checkBlockValidity method checks if the given block is valid
func (bp *baseProcessor) checkBlockValidity(
	chainHandler data.ChainHandler,
	headerHandler data.HeaderHandler,
	bodyHandler data.BodyHandler,
) error {

	err := checkForNils(chainHandler, headerHandler, bodyHandler)
	if err != nil {
		return err
	}

	currentBlockHeader := chainHandler.GetCurrentBlockHeader()

	if currentBlockHeader == nil {
		if headerHandler.GetNonce() == 1 { // first block after genesis
			if bytes.Equal(headerHandler.GetPrevHash(), chainHandler.GetGenesisHeaderHash()) {
				// TODO: add genesis block verification
				return nil
			}

			log.Debug("hash does not match",
				"local block hash", chainHandler.GetGenesisHeaderHash(),
				"received previous hash", headerHandler.GetPrevHash())

			return process.ErrBlockHashDoesNotMatch
		}

		log.Debug("nonce does not match",
			"local block nonce", 0,
			"received nonce", headerHandler.GetNonce())

		return process.ErrWrongNonceInBlock
	}

	if headerHandler.GetRound() <= currentBlockHeader.GetRound() {
		log.Debug("round does not match",
			"local block round", currentBlockHeader.GetRound(),
			"received block round", headerHandler.GetRound())

		return process.ErrLowerRoundInBlock
	}

	if headerHandler.GetNonce() != currentBlockHeader.GetNonce()+1 {
		log.Debug("nonce does not match",
			"local block nonce", currentBlockHeader.GetNonce(),
			"received nonce", headerHandler.GetNonce())

		return process.ErrWrongNonceInBlock
	}

	prevHeaderHash, err := core.CalculateHash(bp.marshalizer, bp.hasher, currentBlockHeader)
	if err != nil {
		return err
	}

	if !bytes.Equal(headerHandler.GetPrevHash(), prevHeaderHash) {
		log.Debug("hash does not match",
			"local block hash", prevHeaderHash,
			"received previous hash", headerHandler.GetPrevHash())

		return process.ErrBlockHashDoesNotMatch
	}

	if !bytes.Equal(headerHandler.GetPrevRandSeed(), currentBlockHeader.GetRandSeed()) {
		log.Debug("random seed does not match", ": local block random seed is %s and node received block with previous random seed %s\n",
			"local random seed", currentBlockHeader.GetRandSeed(),
			"received previous random seed", headerHandler.GetPrevRandSeed())

		return process.ErrRandSeedDoesNotMatch
	}

	if bodyHandler != nil {
		// TODO: add bodyHandler verification here
	}

	// verification of epoch
	if headerHandler.GetEpoch() < currentBlockHeader.GetEpoch() {
		return process.ErrEpochDoesNotMatch
	}

	// TODO: add signature validation as well, with randomness source and all
	return nil
}

// verifyStateRoot verifies the state root hash given as parameter against the
// Merkle trie root hash stored for accounts and returns if equal or not
func (bp *baseProcessor) verifyStateRoot(rootHash []byte) bool {
	trieRootHash, err := bp.accounts.RootHash()
	if err != nil {
		log.Debug("verify account.RootHash", "error", err.Error())
	}

	return bytes.Equal(trieRootHash, rootHash)
}

// getRootHash returns the accounts merkle tree root hash
func (bp *baseProcessor) getRootHash() []byte {
	rootHash, err := bp.accounts.RootHash()
	if err != nil {
		log.Trace("get account.RootHash", "error", err.Error())
	}

	return rootHash
}

func (bp *baseProcessor) checkHeaderTypeCorrect(shardId uint32, hdr data.HeaderHandler) error {
	if shardId >= bp.shardCoordinator.NumberOfShards() && shardId != sharding.MetachainShardId {
		return process.ErrShardIdMissmatch
	}

	if shardId < bp.shardCoordinator.NumberOfShards() {
		_, ok := hdr.(*block.Header)
		if !ok {
			return process.ErrWrongTypeAssertion
		}
	}

	if shardId == sharding.MetachainShardId {
		_, ok := hdr.(*block.MetaBlock)
		if !ok {
			return process.ErrWrongTypeAssertion
		}
	}

	return nil
}

func (bp *baseProcessor) requestHeadersIfMissing(
	sortedHdrs []data.HeaderHandler,
	shardId uint32,
	maxRound uint64,
	cacher storage.Cacher,
) error {

	allowedSize := uint64(float64(cacher.MaxSize()) * process.MaxOccupancyPercentageAllowed)

	prevHdr, _, err := bp.blockTracker.GetLastCrossNotarizedHeader(shardId)
	if err != nil {
		return err
	}

	lastNotarizedHdrNonce := prevHdr.GetNonce()

	missingNonces := make([]uint64, 0)
	for i := 0; i < len(sortedHdrs); i++ {
		currHdr := sortedHdrs[i]
		if currHdr == nil {
			continue
		}

		if i > 0 {
			prevHdr = sortedHdrs[i-1]
		}

		hdrTooNew := currHdr.GetRound() > maxRound || prevHdr.GetRound() > maxRound
		if hdrTooNew {
			continue
		}

		if currHdr.GetNonce()-prevHdr.GetNonce() > 1 {
			for j := prevHdr.GetNonce() + 1; j < currHdr.GetNonce(); j++ {
				missingNonces = append(missingNonces, j)
			}
		}
	}

	requested := 0
	for _, nonce := range missingNonces {
		isHeaderOutOfRange := nonce > lastNotarizedHdrNonce+allowedSize
		if isHeaderOutOfRange {
			break
		}

		if requested >= process.MaxHeaderRequestsAllowed {
			break
		}

		requested++
		go bp.requestHeaderByShardAndNonce(shardId, nonce)
	}

	return nil
}

func displayHeader(headerHandler data.HeaderHandler) []*display.LineData {
	return []*display.LineData{
		display.NewLineData(false, []string{
			"",
			"ChainID",
			display.DisplayByteSlice(headerHandler.GetChainID())}),
		display.NewLineData(false, []string{
			"",
			"Epoch",
			fmt.Sprintf("%d", headerHandler.GetEpoch())}),
		display.NewLineData(false, []string{
			"",
			"Round",
			fmt.Sprintf("%d", headerHandler.GetRound())}),
		display.NewLineData(false, []string{
			"",
			"TimeStamp",
			fmt.Sprintf("%d", headerHandler.GetTimeStamp())}),
		display.NewLineData(false, []string{
			"",
			"Nonce",
			fmt.Sprintf("%d", headerHandler.GetNonce())}),
		display.NewLineData(false, []string{
			"",
			"Prev hash",
			display.DisplayByteSlice(headerHandler.GetPrevHash())}),
		display.NewLineData(false, []string{
			"",
			"Prev rand seed",
			display.DisplayByteSlice(headerHandler.GetPrevRandSeed())}),
		display.NewLineData(false, []string{
			"",
			"Rand seed",
			display.DisplayByteSlice(headerHandler.GetRandSeed())}),
		display.NewLineData(false, []string{
			"",
			"Pub keys bitmap",
			core.ToHex(headerHandler.GetPubKeysBitmap())}),
		display.NewLineData(false, []string{
			"",
			"Signature",
			display.DisplayByteSlice(headerHandler.GetSignature())}),
		display.NewLineData(false, []string{
			"",
			"Leader's Signature",
			display.DisplayByteSlice(headerHandler.GetLeaderSignature())}),
		display.NewLineData(false, []string{
			"",
			"Root hash",
			display.DisplayByteSlice(headerHandler.GetRootHash())}),
		display.NewLineData(false, []string{
			"",
			"Validator stats root hash",
			display.DisplayByteSlice(headerHandler.GetValidatorStatsRootHash())}),
		display.NewLineData(true, []string{
			"",
			"Receipts hash",
			display.DisplayByteSlice(headerHandler.GetReceiptsHash())}),
	}
}

// checkProcessorNilParameters will check the imput parameters for nil values
func checkProcessorNilParameters(arguments ArgBaseProcessor) error {

	if check.IfNil(arguments.Accounts) {
		return process.ErrNilAccountsAdapter
	}
	if check.IfNil(arguments.ForkDetector) {
		return process.ErrNilForkDetector
	}
	if check.IfNil(arguments.Hasher) {
		return process.ErrNilHasher
	}
	if check.IfNil(arguments.Marshalizer) {
		return process.ErrNilMarshalizer
	}
	if check.IfNil(arguments.Store) {
		return process.ErrNilStorage
	}
	if check.IfNil(arguments.ShardCoordinator) {
		return process.ErrNilShardCoordinator
	}
	if check.IfNil(arguments.NodesCoordinator) {
		return process.ErrNilNodesCoordinator
	}
	if check.IfNil(arguments.SpecialAddressHandler) {
		return process.ErrNilSpecialAddressHandler
	}
	if check.IfNil(arguments.Uint64Converter) {
		return process.ErrNilUint64Converter
	}
	if check.IfNil(arguments.RequestHandler) {
		return process.ErrNilRequestHandler
	}
	if check.IfNil(arguments.EpochStartTrigger) {
		return process.ErrNilEpochStartTrigger
	}
	if check.IfNil(arguments.Rounder) {
		return process.ErrNilRounder
	}
	if check.IfNil(arguments.BootStorer) {
		return process.ErrNilStorage
	}
	if check.IfNil(arguments.BlockChainHook) {
		return process.ErrNilBlockChainHook
	}
	if check.IfNil(arguments.TxCoordinator) {
		return process.ErrNilTransactionCoordinator
	}
	if check.IfNil(arguments.HeaderValidator) {
		return process.ErrNilHeaderValidator
	}
	if check.IfNil(arguments.BlockTracker) {
		return process.ErrNilBlockTracker
	}

	return nil
}

func (bp *baseProcessor) createBlockStarted() {
	bp.resetMissingHdrs()
	bp.hdrsForCurrBlock.mutHdrsForBlock.Lock()
	bp.hdrsForCurrBlock.hdrHashAndInfo = make(map[string]*hdrInfo)
	bp.hdrsForCurrBlock.highestHdrNonce = make(map[uint32]uint64)
	bp.hdrsForCurrBlock.mutHdrsForBlock.Unlock()
	bp.txCoordinator.CreateBlockStarted()
}

func (bp *baseProcessor) resetMissingHdrs() {
	bp.hdrsForCurrBlock.mutHdrsForBlock.Lock()
	bp.hdrsForCurrBlock.missingHdrs = 0
	bp.hdrsForCurrBlock.missingFinalityAttestingHdrs = 0
	bp.hdrsForCurrBlock.mutHdrsForBlock.Unlock()
}

//TODO: remove bool parameter and give instead the set to sort
func (bp *baseProcessor) sortHeadersForCurrentBlockByNonce(usedInBlock bool) map[uint32][]data.HeaderHandler {
	hdrsForCurrentBlock := make(map[uint32][]data.HeaderHandler)

	bp.hdrsForCurrBlock.mutHdrsForBlock.RLock()
	for _, headerInfo := range bp.hdrsForCurrBlock.hdrHashAndInfo {
		if headerInfo.usedInBlock != usedInBlock {
			continue
		}

		hdrsForCurrentBlock[headerInfo.hdr.GetShardID()] = append(hdrsForCurrentBlock[headerInfo.hdr.GetShardID()], headerInfo.hdr)
	}
	bp.hdrsForCurrBlock.mutHdrsForBlock.RUnlock()

	// sort headers for each shard
	for _, hdrsForShard := range hdrsForCurrentBlock {
		process.SortHeadersByNonce(hdrsForShard)
	}

	return hdrsForCurrentBlock
}

//TODO: remove bool parameter and give instead the set to sort
func (bp *baseProcessor) sortHeaderHashesForCurrentBlockByNonce(usedInBlock bool) map[uint32][][]byte {
	hdrsForCurrentBlockInfo := make(map[uint32][]*nonceAndHashInfo)

	bp.hdrsForCurrBlock.mutHdrsForBlock.RLock()
	for metaBlockHash, headerInfo := range bp.hdrsForCurrBlock.hdrHashAndInfo {
		if headerInfo.usedInBlock != usedInBlock {
			continue
		}

		hdrsForCurrentBlockInfo[headerInfo.hdr.GetShardID()] = append(hdrsForCurrentBlockInfo[headerInfo.hdr.GetShardID()],
			&nonceAndHashInfo{nonce: headerInfo.hdr.GetNonce(), hash: []byte(metaBlockHash)})
	}
	bp.hdrsForCurrBlock.mutHdrsForBlock.RUnlock()

	for _, hdrsForShard := range hdrsForCurrentBlockInfo {
		if len(hdrsForShard) > 1 {
			sort.Slice(hdrsForShard, func(i, j int) bool {
				return hdrsForShard[i].nonce < hdrsForShard[j].nonce
			})
		}
	}

	hdrsHashesForCurrentBlock := make(map[uint32][][]byte, len(hdrsForCurrentBlockInfo))
	for shardId, hdrsForShard := range hdrsForCurrentBlockInfo {
		for _, hdrForShard := range hdrsForShard {
			hdrsHashesForCurrentBlock[shardId] = append(hdrsHashesForCurrentBlock[shardId], hdrForShard.hash)
		}
	}

	return hdrsHashesForCurrentBlock
}

func (bp *baseProcessor) getMaxMiniBlocksSpaceRemained(
	maxItemsInBlock uint32,
	itemsAddedInBlock uint32,
	miniBlocksAddedInBlock uint32,
) int32 {
	mbSpaceRemainedInBlock := int32(maxItemsInBlock) - int32(itemsAddedInBlock)
	mbSpaceRemainedInCache := int32(core.MaxMiniBlocksInBlock) - int32(miniBlocksAddedInBlock)
	maxMbSpaceRemained := core.MinInt32(mbSpaceRemainedInBlock, mbSpaceRemainedInCache)

	return maxMbSpaceRemained
}

func (bp *baseProcessor) createMiniBlockHeaders(body block.Body) (int, []block.MiniBlockHeader, error) {
	totalTxCount := 0
	miniBlockHeaders := make([]block.MiniBlockHeader, len(body))

	for i := 0; i < len(body); i++ {
		txCount := len(body[i].TxHashes)
		totalTxCount += txCount

		miniBlockHash, err := core.CalculateHash(bp.marshalizer, bp.hasher, body[i])
		if err != nil {
			return 0, nil, err
		}

		miniBlockHeaders[i] = block.MiniBlockHeader{
			Hash:            miniBlockHash,
			SenderShardID:   body[i].SenderShardID,
			ReceiverShardID: body[i].ReceiverShardID,
			TxCount:         uint32(txCount),
			Type:            body[i].Type,
		}
	}

	return totalTxCount, miniBlockHeaders, nil
}

// check if header has the same miniblocks as presented in body
func (bp *baseProcessor) checkHeaderBodyCorrelation(miniBlockHeaders []block.MiniBlockHeader, body block.Body) error {
	mbHashesFromHdr := make(map[string]*block.MiniBlockHeader, len(miniBlockHeaders))
	for i := 0; i < len(miniBlockHeaders); i++ {
		mbHashesFromHdr[string(miniBlockHeaders[i].Hash)] = &miniBlockHeaders[i]
	}

	if len(miniBlockHeaders) != len(body) {
		return process.ErrHeaderBodyMismatch
	}

	for i := 0; i < len(body); i++ {
		miniBlock := body[i]

		mbHash, err := core.CalculateHash(bp.marshalizer, bp.hasher, miniBlock)
		if err != nil {
			return err
		}

		mbHdr, ok := mbHashesFromHdr[string(mbHash)]
		if !ok {
			return process.ErrHeaderBodyMismatch
		}

		if mbHdr.TxCount != uint32(len(miniBlock.TxHashes)) {
			return process.ErrHeaderBodyMismatch
		}

		if mbHdr.ReceiverShardID != miniBlock.ReceiverShardID {
			return process.ErrHeaderBodyMismatch
		}

		if mbHdr.SenderShardID != miniBlock.SenderShardID {
			return process.ErrHeaderBodyMismatch
		}
	}

	return nil
}

func (bp *baseProcessor) isHeaderOutOfRange(header data.HeaderHandler, cacher storage.Cacher) bool {
	lastCrossNotarizedHeader, _, _ := bp.blockTracker.GetLastCrossNotarizedHeader(header.GetShardID())
	if check.IfNil(lastCrossNotarizedHeader) {
		return false
	}

	allowedSize := uint64(float64(cacher.MaxSize()) * process.MaxOccupancyPercentageAllowed)
	isHeaderOutOfRange := header.GetNonce() > lastCrossNotarizedHeader.GetNonce()+allowedSize

	return isHeaderOutOfRange
}

// requestMissingFinalityAttestingHeaders requests the headers needed to accept the current selected headers for
// processing the current block. It requests the finality headers greater than the highest header, for given shard,
// related to the block which should be processed
func (bp *baseProcessor) requestMissingFinalityAttestingHeaders(
	shardId uint32,
	finality uint32,
) uint32 {
	requestedHeaders := uint32(0)
	missingFinalityAttestingHeaders := uint32(0)

	highestHdrNonce := bp.hdrsForCurrBlock.highestHdrNonce[shardId]
	if highestHdrNonce == uint64(0) {
		return missingFinalityAttestingHeaders
	}

	lastFinalityAttestingHeader := highestHdrNonce + uint64(finality)
	for i := highestHdrNonce + 1; i <= lastFinalityAttestingHeader; i++ {
		headers, headersHashes := bp.blockTracker.GetTrackedHeadersForShardWithNonce(shardId, i)

		if len(headers) == 0 {
			missingFinalityAttestingHeaders++
			requestedHeaders++
			go bp.requestHeaderByShardAndNonce(shardId, i)
			continue
		}

		for index := range headers {
			bp.hdrsForCurrBlock.hdrHashAndInfo[string(headersHashes[index])] = &hdrInfo{hdr: headers[index], usedInBlock: false}
		}
	}

	if requestedHeaders > 0 {
		log.Debug("requested missing finality attesting headers",
			"num headers", requestedHeaders,
			"shard", shardId)
	}

	return missingFinalityAttestingHeaders
}

func (bp *baseProcessor) requestHeaderByShardAndNonce(targetShardID uint32, nonce uint64) {
	if targetShardID == sharding.MetachainShardId {
		bp.requestHandler.RequestMetaHeaderByNonce(nonce)
	} else {
		bp.requestHandler.RequestShardHeaderByNonce(targetShardID, nonce)
	}
}

func (bp *baseProcessor) cleanupPools(
	headersNoncesPool dataRetriever.Uint64SyncMapCacher,
	headersPool storage.Cacher,
	notarizedHeadersPool storage.Cacher,
) {
	bp.removeHeadersBehindNonceFromPools(
		true,
		headersPool,
		headersNoncesPool,
		bp.shardCoordinator.SelfId(),
		bp.forkDetector.GetHighestFinalBlockNonce())

	for shardID := uint32(0); shardID < bp.shardCoordinator.NumberOfShards(); shardID++ {
		if bp.shardCoordinator.SelfId() == shardID {
			continue
		}

		bp.cleanupPoolsForShard(shardID, headersNoncesPool, notarizedHeadersPool)
	}

	if bp.shardCoordinator.SelfId() != sharding.MetachainShardId {
		bp.cleanupPoolsForShard(sharding.MetachainShardId, headersNoncesPool, notarizedHeadersPool)
	}
}

func (bp *baseProcessor) cleanupPoolsForShard(
	shardID uint32,
	headersNoncesPool dataRetriever.Uint64SyncMapCacher,
	notarizedHeadersPool storage.Cacher,
) {
	lastCrossNotarizedHeader, _, _ := bp.blockTracker.GetLastCrossNotarizedHeader(shardID)
	if check.IfNil(lastCrossNotarizedHeader) {
		return
	}

	bp.removeHeadersBehindNonceFromPools(
		false,
		notarizedHeadersPool,
		headersNoncesPool,
		shardID,
		lastCrossNotarizedHeader.GetNonce())
}

func (bp *baseProcessor) cleanupBlockTrackerPools(chainHandler data.ChainHandler) {
	noncesToFinal := bp.getNoncesToFinal(chainHandler)

	for shardID := uint32(0); shardID < bp.shardCoordinator.NumberOfShards(); shardID++ {
		bp.cleanupBlockTrackerPoolsForShard(shardID, noncesToFinal)
	}

	bp.cleanupBlockTrackerPoolsForShard(sharding.MetachainShardId, noncesToFinal)
}

func (bp *baseProcessor) getNoncesToFinal(chainHandler data.ChainHandler) uint64 {
	currentBlockNonce := uint64(0)
	headerHandler := chainHandler.GetCurrentBlockHeader()
	if !check.IfNil(headerHandler) {
		currentBlockNonce = headerHandler.GetNonce()
	}

	noncesToFinal := uint64(0)
	finalBlockNonce := bp.forkDetector.GetHighestFinalBlockNonce()
	if currentBlockNonce > finalBlockNonce {
		noncesToFinal = currentBlockNonce - finalBlockNonce
	}

	return noncesToFinal
}

func (bp *baseProcessor) cleanupBlockTrackerPoolsForShard(shardID uint32, noncesToFinal uint64) {
	lastCrossNotarizedHeader, _, _ := bp.blockTracker.GetLastCrossNotarizedHeader(shardID)
	if check.IfNil(lastCrossNotarizedHeader) {
		return
	}

	crossNotarizedNonce := uint64(0)
	if lastCrossNotarizedHeader.GetNonce() > noncesToFinal {
		crossNotarizedNonce = lastCrossNotarizedHeader.GetNonce() - noncesToFinal
	}

	bp.blockTracker.CleanupHeadersForShardBehindNonce(
		shardID,
		bp.forkDetector.GetHighestFinalBlockNonce(),
		crossNotarizedNonce,
	)
}

func (bp *baseProcessor) removeHeadersBehindNonceFromPools(
	shouldRemoveBlockBody bool,
	cacher storage.Cacher,
	uint64SyncMapCacher dataRetriever.Uint64SyncMapCacher,
	shardId uint32,
	nonce uint64,
) {
	if nonce <= 1 {
		return
	}

	if check.IfNil(cacher) {
		return
	}

	for _, key := range cacher.Keys() {
		val, _ := cacher.Peek(key)
		if val == nil {
			continue
		}

		hdr, ok := val.(data.HeaderHandler)
		if !ok {
			continue
		}

		if hdr.GetShardID() != shardId || hdr.GetNonce() >= nonce {
			continue
		}

		if shouldRemoveBlockBody {
			errNotCritical := bp.removeBlockBodyOfHeader(hdr)
			if errNotCritical != nil {
				log.Debug("RemoveBlockDataFromPool", "error", errNotCritical.Error())
			}
		}

		cacher.Remove(key)

		if check.IfNil(uint64SyncMapCacher) {
			continue
		}

		uint64SyncMapCacher.Remove(hdr.GetNonce(), hdr.GetShardID())
	}
}

func (bp *baseProcessor) removeBlockBodyOfHeader(headerHandler data.HeaderHandler) error {
	bodyHandler, err := bp.requestBlockBodyHandler.GetBlockBodyFromPool(headerHandler)
	if err != nil {
		return err
	}

	body, ok := bodyHandler.(block.Body)
	if !ok {
		return process.ErrWrongTypeAssertion
	}

	err = bp.txCoordinator.RemoveBlockDataFromPool(body)
	if err != nil {
		return err
	}

	return nil
}

//func (bp *baseProcessor) removeHeaderFromPools(
//	header data.HeaderHandler,
//	cacher storage.Cacher,
//	uint64SyncMapCacher dataRetriever.Uint64SyncMapCacher,
//) {
//
//	if check.IfNil(header) {
//		return
//	}
//
//	headerHash, err := core.CalculateHash(bp.marshalizer, bp.hasher, header)
//	if err != nil {
//		return
//	}
//
//	if !check.IfNil(cacher) {
//		cacher.Remove(headerHash)
//	}
//
//	if !check.IfNil(uint64SyncMapCacher) {
//		syncMap, ok := uint64SyncMapCacher.Get(header.GetNonce())
//		if !ok {
//			return
//		}
//
//		hash, ok := syncMap.Load(header.GetShardID())
//		if hash == nil || !ok {
//			return
//		}
//
//		if bytes.Equal(headerHash, hash) {
//			uint64SyncMapCacher.Remove(header.GetNonce(), header.GetShardID())
//		}
//	}
//}

func (bp *baseProcessor) prepareDataForBootStorer(
	headerInfo bootstrapStorage.BootstrapHeaderInfo,
	round uint64,
	lastFinalHdrs []data.HeaderHandler,
	lastFinalHashes [][]byte,
	processedMiniBlocks []bootstrapStorage.MiniBlocksInMeta,
) {
	lastFinals := make([]bootstrapStorage.BootstrapHeaderInfo, 0, len(lastFinalHdrs))

	//TODO add end of epoch stuff

	lastCrossNotarizedHeaders := bp.getLastCrossNotarizedHeaders()
	highestFinalNonce := bp.forkDetector.GetHighestFinalBlockNonce()

	for i := range lastFinalHdrs {
		headerInfo := bootstrapStorage.BootstrapHeaderInfo{
			ShardId: lastFinalHdrs[i].GetShardID(),
			Nonce:   lastFinalHdrs[i].GetNonce(),
			Hash:    lastFinalHashes[i],
		}

		lastFinals = append(lastFinals, headerInfo)
	}

	bootData := bootstrapStorage.BootstrapData{
		LastHeader:                headerInfo,
		LastCrossNotarizedHeaders: lastCrossNotarizedHeaders,
		LastSelfNotarizedHeaders:  lastFinals,
		HighestFinalNonce:         highestFinalNonce,
		ProcessedMiniBlocks:       processedMiniBlocks,
	}

	err := bp.bootStorer.Put(int64(round), bootData)
	if err != nil {
		log.Warn("cannot save boot data in storage",
			"error", err.Error())
	}
}

func (bp *baseProcessor) getLastCrossNotarizedHeaders() []bootstrapStorage.BootstrapHeaderInfo {
	lastCrossNotarizedHeaders := make([]bootstrapStorage.BootstrapHeaderInfo, 0, bp.shardCoordinator.NumberOfShards()+1)

	for shardID := uint32(0); shardID < bp.shardCoordinator.NumberOfShards(); shardID++ {
		bootstrapHeaderInfo := bp.getLastCrossNotarizedHeadersForShard(shardID)
		if bootstrapHeaderInfo != nil {
			lastCrossNotarizedHeaders = append(lastCrossNotarizedHeaders, *bootstrapHeaderInfo)
		}
	}

	bootstrapHeaderInfo := bp.getLastCrossNotarizedHeadersForShard(sharding.MetachainShardId)
	if bootstrapHeaderInfo != nil {
		lastCrossNotarizedHeaders = append(lastCrossNotarizedHeaders, *bootstrapHeaderInfo)
	}

	return bootstrapStorage.TrimHeaderInfoSlice(lastCrossNotarizedHeaders)
}

func (bp baseProcessor) getLastCrossNotarizedHeadersForShard(shardID uint32) *bootstrapStorage.BootstrapHeaderInfo {
	lastCrossNotarizedHeader, lastCrossNotarizedHeaderHash, _ := bp.blockTracker.GetLastCrossNotarizedHeader(shardID)
	if check.IfNil(lastCrossNotarizedHeader) {
		return nil
	}

	if lastCrossNotarizedHeader.GetNonce() == 0 {
		return nil
	}

	headerInfo := &bootstrapStorage.BootstrapHeaderInfo{
		ShardId: lastCrossNotarizedHeader.GetShardID(),
		Nonce:   lastCrossNotarizedHeader.GetNonce(),
		Hash:    lastCrossNotarizedHeaderHash,
	}

	return headerInfo
}

func (bp *baseProcessor) commitAll() error {
	_, err := bp.accounts.Commit()
	if err != nil {
		return err
	}

	_, err = bp.validatorStatisticsProcessor.Commit()
	if err != nil {
		return err
	}

	return nil
}

func deleteSelfReceiptsMiniBlocks(body block.Body) block.Body {
	for i := 0; i < len(body); {
		mb := body[i]
		if mb.ReceiverShardID != mb.SenderShardID {
			i++
			continue
		}

		if mb.Type != block.ReceiptBlock && mb.Type != block.SmartContractResultBlock {
			i++
			continue
		}

		body[i] = body[len(body)-1]
		body = body[:len(body)-1]
		if i == len(body)-1 {
			break
		}
	}

	return body
}
