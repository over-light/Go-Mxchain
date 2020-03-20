package block

import (
	"bytes"
	"fmt"
	"sort"

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

type baseProcessor struct {
	shardCoordinator        sharding.Coordinator
	nodesCoordinator        sharding.NodesCoordinator
	accountsDB              map[state.AccountsDbIdentifier]state.AccountsAdapter
	forkDetector            process.ForkDetector
	hasher                  hashing.Hasher
	marshalizer             marshal.Marshalizer
	store                   dataRetriever.StorageService
	uint64Converter         typeConverters.Uint64ByteSliceConverter
	blockSizeThrottler      process.BlockSizeThrottler
	epochStartTrigger       process.EpochStartTriggerHandler
	headerValidator         process.HeaderConstructionValidator
	blockChainHook          process.BlockChainHookHandler
	txCoordinator           process.TransactionCoordinator
	rounder                 consensus.Rounder
	bootStorer              process.BootStorer
	requestBlockBodyHandler process.RequestBlockBodyHandler
	requestHandler          process.RequestHandler
	blockTracker            process.BlockTracker
	dataPool                dataRetriever.PoolsHolder
	feeHandler              process.TransactionFeeHandler
	blockChain              data.ChainHandler
	hdrsForCurrBlock        *hdrForBlock

	appStatusHandler       core.AppStatusHandler
	stateCheckpointModulus uint
}

type bootStorerDataArgs struct {
	headerInfo                 bootstrapStorage.BootstrapHeaderInfo
	lastSelfNotarizedHeaders   []bootstrapStorage.BootstrapHeaderInfo
	round                      uint64
	highestFinalBlockNonce     uint64
	pendingMiniBlocks          []bootstrapStorage.PendingMiniBlocksInfo
	processedMiniBlocks        []bootstrapStorage.MiniBlocksInMeta
	nodesCoordinatorConfigKey  []byte
	epochStartTriggerConfigKey []byte
}

func checkForNils(
	headerHandler data.HeaderHandler,
	bodyHandler data.BodyHandler,
) error {
	if check.IfNil(headerHandler) {
		return process.ErrNilBlockHeader
	}
	if check.IfNil(bodyHandler) {
		return process.ErrNilBlockBody
	}
	return nil
}

// SetAppStatusHandler method is used to set appStatusHandler
func (bp *baseProcessor) SetAppStatusHandler(ash core.AppStatusHandler) error {
	if check.IfNil(ash) {
		return process.ErrNilAppStatusHandler
	}

	bp.appStatusHandler = ash
	return nil
}

// checkBlockValidity method checks if the given block is valid
func (bp *baseProcessor) checkBlockValidity(
	headerHandler data.HeaderHandler,
	bodyHandler data.BodyHandler,
) error {

	err := checkForNils(headerHandler, bodyHandler)
	if err != nil {
		return err
	}

	currentBlockHeader := bp.blockChain.GetCurrentBlockHeader()

	if check.IfNil(currentBlockHeader) {
		if headerHandler.GetNonce() == 1 { // first block after genesis
			if bytes.Equal(headerHandler.GetPrevHash(), bp.blockChain.GetGenesisHeaderHash()) {
				// TODO: add genesis block verification
				return nil
			}

			log.Debug("hash does not match",
				"local block hash", bp.blockChain.GetGenesisHeaderHash(),
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

	if !bytes.Equal(headerHandler.GetPrevHash(), bp.blockChain.GetCurrentBlockHeaderHash()) {
		log.Debug("hash does not match",
			"local block hash", bp.blockChain.GetCurrentBlockHeaderHash(),
			"received previous hash", headerHandler.GetPrevHash())

		return process.ErrBlockHashDoesNotMatch
	}

	if !bytes.Equal(headerHandler.GetPrevRandSeed(), currentBlockHeader.GetRandSeed()) {
		log.Debug("random seed does not match",
			"local random seed", currentBlockHeader.GetRandSeed(),
			"received previous random seed", headerHandler.GetPrevRandSeed())

		return process.ErrRandSeedDoesNotMatch
	}

	// verification of epoch
	if headerHandler.GetEpoch() < currentBlockHeader.GetEpoch() {
		return process.ErrEpochDoesNotMatch
	}

	return nil
}

// verifyStateRoot verifies the state root hash given as parameter against the
// Merkle trie root hash stored for accounts and returns if equal or not
func (bp *baseProcessor) verifyStateRoot(rootHash []byte) bool {
	trieRootHash, err := bp.accountsDB[state.UserAccountsState].RootHash()
	if err != nil {
		log.Debug("verify account.RootHash", "error", err.Error())
	}

	return bytes.Equal(trieRootHash, rootHash)
}

// getRootHash returns the accounts merkle tree root hash
func (bp *baseProcessor) getRootHash() []byte {
	rootHash, err := bp.accountsDB[state.UserAccountsState].RootHash()
	if err != nil {
		log.Trace("get account.RootHash", "error", err.Error())
	}

	return rootHash
}

func (bp *baseProcessor) requestHeadersIfMissing(
	sortedHdrs []data.HeaderHandler,
	shardId uint32,
	maxRound uint64,
) error {

	prevHdr, _, err := bp.blockTracker.GetLastCrossNotarizedHeader(shardId)
	if err != nil {
		return err
	}

	lastNotarizedHdrRound := prevHdr.GetRound()

	missingNonces := make([]uint64, 0)
	for i := 0; i < len(sortedHdrs); i++ {
		currHdr := sortedHdrs[i]
		if currHdr == nil {
			continue
		}

		hdrTooOld := currHdr.GetRound() <= lastNotarizedHdrRound
		if hdrTooOld {
			continue
		}

		hdrTooNew := currHdr.GetRound() > maxRound
		if hdrTooNew {
			break
		}

		if !bp.blockTracker.ShouldAddHeader(currHdr) {
			break
		}

		if currHdr.GetNonce()-prevHdr.GetNonce() > 1 {
			for j := prevHdr.GetNonce() + 1; j < currHdr.GetNonce(); j++ {
				missingNonces = append(missingNonces, j)
			}
		}

		prevHdr = currHdr
	}

	requested := 0
	for _, nonce := range missingNonces {
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
		display.NewLineData(false, []string{
			"",
			"Receipts hash",
			display.DisplayByteSlice(headerHandler.GetReceiptsHash())}),
		display.NewLineData(true, []string{
			"",
			"Epoch start meta hash",
			display.DisplayByteSlice(headerHandler.GetEpochStartMetaHash())}),
	}
}

// checkProcessorNilParameters will check the imput parameters for nil values
func checkProcessorNilParameters(arguments ArgBaseProcessor) error {

	for key := range arguments.AccountsDB {
		if check.IfNil(arguments.AccountsDB[key]) {
			return process.ErrNilAccountsAdapter
		}
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
	if check.IfNil(arguments.FeeHandler) {
		return process.ErrNilEconomicsFeeHandler
	}
	if check.IfNil(arguments.BlockChain) {
		return process.ErrNilBlockChain
	}
	if check.IfNil(arguments.BlockSizeThrottler) {
		return process.ErrNilBlockSizeThrottler
	}

	return nil
}

func (bp *baseProcessor) createBlockStarted() {
	bp.hdrsForCurrBlock.resetMissingHdrs()
	bp.hdrsForCurrBlock.initMaps()
	bp.txCoordinator.CreateBlockStarted()
	bp.feeHandler.CreateBlockStarted()
}

func (bp *baseProcessor) verifyAccumulatedFees(header data.HeaderHandler) error {
	if header.GetAccumulatedFees().Cmp(bp.feeHandler.GetAccumulatedFees()) != 0 {
		return process.ErrAccumulatedFeesDoNotMatch
	}
	return nil
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

func (bp *baseProcessor) createMiniBlockHeaders(body *block.Body) (int, []block.MiniBlockHeader, error) {
	if len(body.MiniBlocks) == 0 {
		return 0, nil, nil
	}

	totalTxCount := 0
	miniBlockHeaders := make([]block.MiniBlockHeader, len(body.MiniBlocks))

	for i := 0; i < len(body.MiniBlocks); i++ {
		txCount := len(body.MiniBlocks[i].TxHashes)
		totalTxCount += txCount

		miniBlockHash, err := core.CalculateHash(bp.marshalizer, bp.hasher, body.MiniBlocks[i])
		if err != nil {
			return 0, nil, err
		}

		miniBlockHeaders[i] = block.MiniBlockHeader{
			Hash:            miniBlockHash,
			SenderShardID:   body.MiniBlocks[i].SenderShardID,
			ReceiverShardID: body.MiniBlocks[i].ReceiverShardID,
			TxCount:         uint32(txCount),
			Type:            body.MiniBlocks[i].Type,
		}
	}

	return totalTxCount, miniBlockHeaders, nil
}

// check if header has the same miniblocks as presented in body
func (bp *baseProcessor) checkHeaderBodyCorrelation(miniBlockHeaders []block.MiniBlockHeader, body *block.Body) error {
	mbHashesFromHdr := make(map[string]*block.MiniBlockHeader, len(miniBlockHeaders))
	for i := 0; i < len(miniBlockHeaders); i++ {
		mbHashesFromHdr[string(miniBlockHeaders[i].Hash)] = &miniBlockHeaders[i]
	}

	if len(miniBlockHeaders) != len(body.MiniBlocks) {
		return process.ErrHeaderBodyMismatch
	}

	for i := 0; i < len(body.MiniBlocks); i++ {
		miniBlock := body.MiniBlocks[i]
		if miniBlock == nil {
			return process.ErrNilMiniBlock
		}

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
		headers, headersHashes := bp.blockTracker.GetTrackedHeadersWithNonce(shardId, i)

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
	if targetShardID == core.MetachainShardId {
		bp.requestHandler.RequestMetaHeaderByNonce(nonce)
	} else {
		bp.requestHandler.RequestShardHeaderByNonce(targetShardID, nonce)
	}
}

func (bp *baseProcessor) cleanupPools(headerHandler data.HeaderHandler) {
	headersPool := bp.dataPool.Headers()
	noncesToFinal := bp.getNoncesToFinal(headerHandler)

	bp.removeHeadersBehindNonceFromPools(
		true,
		headersPool,
		bp.shardCoordinator.SelfId(),
		bp.forkDetector.GetHighestFinalBlockNonce())

	if bp.shardCoordinator.SelfId() == core.MetachainShardId {
		for shardID := uint32(0); shardID < bp.shardCoordinator.NumberOfShards(); shardID++ {
			bp.cleanupPoolsForShard(shardID, headersPool, noncesToFinal)
		}
	} else {
		bp.cleanupPoolsForShard(core.MetachainShardId, headersPool, noncesToFinal)
	}
}

func (bp *baseProcessor) cleanupPoolsForShard(
	shardID uint32,
	headersPool dataRetriever.HeadersPool,
	noncesToFinal uint64,
) {
	crossNotarizedHeader, _, err := bp.blockTracker.GetCrossNotarizedHeader(shardID, noncesToFinal)
	if err != nil {
		log.Warn("cleanupPoolsForShard",
			"shard", shardID,
			"nonces to final", noncesToFinal,
			"error", err.Error())
		return
	}

	bp.removeHeadersBehindNonceFromPools(
		false,
		headersPool,
		shardID,
		crossNotarizedHeader.GetNonce(),
	)
}

func (bp *baseProcessor) removeHeadersBehindNonceFromPools(
	shouldRemoveBlockBody bool,
	headersPool dataRetriever.HeadersPool,
	shardId uint32,
	nonce uint64,
) {
	if nonce <= 1 {
		return
	}

	if check.IfNil(headersPool) {
		return
	}

	nonces := headersPool.Nonces(shardId)
	for _, nonceFromCache := range nonces {
		if nonceFromCache >= nonce {
			continue
		}

		if shouldRemoveBlockBody {
			bp.removeBlocksBody(nonceFromCache, shardId, headersPool)
		}

		headersPool.RemoveHeaderByNonceAndShardId(nonceFromCache, shardId)
	}
}

func (bp *baseProcessor) removeBlocksBody(nonce uint64, shardId uint32, headersPool dataRetriever.HeadersPool) {
	headers, _, err := headersPool.GetHeadersByNonceAndShardId(nonce, shardId)
	if err != nil {
		return
	}

	for _, header := range headers {
		errNotCritical := bp.removeBlockBodyOfHeader(header)
		if errNotCritical != nil {
			log.Debug("RemoveBlockDataFromPool", "error", errNotCritical.Error())
		}
	}
}

func (bp *baseProcessor) removeBlockBodyOfHeader(headerHandler data.HeaderHandler) error {
	bodyHandler, err := bp.requestBlockBodyHandler.GetBlockBodyFromPool(headerHandler)
	if err != nil {
		return err
	}

	body, ok := bodyHandler.(*block.Body)
	if !ok {
		return process.ErrWrongTypeAssertion
	}

	err = bp.txCoordinator.RemoveBlockDataFromPool(body)
	if err != nil {
		return err
	}

	return nil
}

func (bp *baseProcessor) cleanupBlockTrackerPools(headerHandler data.HeaderHandler) {
	noncesToFinal := bp.getNoncesToFinal(headerHandler)

	bp.cleanupBlockTrackerPoolsForShard(bp.shardCoordinator.SelfId(), noncesToFinal)

	if bp.shardCoordinator.SelfId() == core.MetachainShardId {
		for shardID := uint32(0); shardID < bp.shardCoordinator.NumberOfShards(); shardID++ {
			bp.cleanupBlockTrackerPoolsForShard(shardID, noncesToFinal)
		}
	} else {
		bp.cleanupBlockTrackerPoolsForShard(core.MetachainShardId, noncesToFinal)
	}
}

func (bp *baseProcessor) cleanupBlockTrackerPoolsForShard(shardID uint32, noncesToFinal uint64) {
	shardForSelfNotarized := bp.getShardForSelfNotarized(shardID)
	selfNotarizedHeader, _, err := bp.blockTracker.GetLastSelfNotarizedHeader(shardForSelfNotarized)
	if err != nil {
		log.Warn("cleanupBlockTrackerPoolsForShard.GetLastSelfNotarizedHeader",
			"shard", shardForSelfNotarized,
			"error", err.Error())
		return
	}

	selfNotarizedNonce := selfNotarizedHeader.GetNonce()
	crossNotarizedNonce := uint64(0)

	if shardID != bp.shardCoordinator.SelfId() {
		crossNotarizedHeader, _, errNotCritical := bp.blockTracker.GetCrossNotarizedHeader(shardID, noncesToFinal)
		if errNotCritical != nil {
			log.Warn("cleanupBlockTrackerPoolsForShard.GetCrossNotarizedHeader",
				"shard", shardID,
				"nonces to final", noncesToFinal,
				"error", errNotCritical.Error())
			return
		}

		crossNotarizedNonce = crossNotarizedHeader.GetNonce()
	}

	bp.blockTracker.CleanupHeadersBehindNonce(
		shardID,
		selfNotarizedNonce,
		crossNotarizedNonce,
	)

	log.Trace("cleanupBlockTrackerPoolsForShard.CleanupHeadersBehindNonce",
		"shard", shardID,
		"self notarized nonce", selfNotarizedNonce,
		"cross notarized nonce", crossNotarizedNonce)
}

func (bp *baseProcessor) getShardForSelfNotarized(shardID uint32) uint32 {
	isSelfShard := shardID == bp.shardCoordinator.SelfId()
	if isSelfShard && bp.shardCoordinator.SelfId() != core.MetachainShardId {
		return core.MetachainShardId
	}

	return shardID
}

func (bp *baseProcessor) prepareDataForBootStorer(args bootStorerDataArgs) {
	lastCrossNotarizedHeaders := bp.getLastCrossNotarizedHeaders()

	bootData := bootstrapStorage.BootstrapData{
		LastHeader:                 args.headerInfo,
		LastCrossNotarizedHeaders:  lastCrossNotarizedHeaders,
		LastSelfNotarizedHeaders:   args.lastSelfNotarizedHeaders,
		PendingMiniBlocks:          args.pendingMiniBlocks,
		ProcessedMiniBlocks:        args.processedMiniBlocks,
		HighestFinalBlockNonce:     args.highestFinalBlockNonce,
		NodesCoordinatorConfigKey:  args.nodesCoordinatorConfigKey,
		EpochStartTriggerConfigKey: args.epochStartTriggerConfigKey,
	}

	err := bp.bootStorer.Put(int64(args.round), bootData)
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

	bootstrapHeaderInfo := bp.getLastCrossNotarizedHeadersForShard(core.MetachainShardId)
	if bootstrapHeaderInfo != nil {
		lastCrossNotarizedHeaders = append(lastCrossNotarizedHeaders, *bootstrapHeaderInfo)
	}

	return bootstrapStorage.TrimHeaderInfoSlice(lastCrossNotarizedHeaders)
}

func (bp *baseProcessor) getLastCrossNotarizedHeadersForShard(shardID uint32) *bootstrapStorage.BootstrapHeaderInfo {
	lastCrossNotarizedHeader, lastCrossNotarizedHeaderHash, err := bp.blockTracker.GetLastCrossNotarizedHeader(shardID)
	if err != nil {
		log.Warn("getLastCrossNotarizedHeadersForShard",
			"shard", shardID,
			"error", err.Error())
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

func deleteSelfReceiptsMiniBlocks(body *block.Body) *block.Body {
	newBody := &block.Body{}
	for _, mb := range body.MiniBlocks {
		isInShardUnsignedMB := mb.ReceiverShardID == mb.SenderShardID &&
			(mb.Type == block.ReceiptBlock || mb.Type == block.SmartContractResultBlock)
		if isInShardUnsignedMB {
			continue
		}

		newBody.MiniBlocks = append(newBody.MiniBlocks, mb)
	}

	return newBody
}

func (bp *baseProcessor) getNoncesToFinal(headerHandler data.HeaderHandler) uint64 {
	currentBlockNonce := uint64(0)
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

// DecodeBlockBody method decodes block body from a given byte array
func (bp *baseProcessor) DecodeBlockBody(dta []byte) data.BodyHandler {
	body := &block.Body{}
	if dta == nil {
		return body
	}

	err := bp.marshalizer.Unmarshal(body, dta)
	if err != nil {
		log.Debug("DecodeBlockBody.Unmarshal", "error", err.Error())
		return nil
	}

	return body
}

// DecodeBlockHeader method decodes block header from a given byte array
func (bp *baseProcessor) DecodeBlockHeader(dta []byte) data.HeaderHandler {
	if dta == nil {
		return nil
	}

	header := bp.blockChain.CreateNewHeader()

	err := bp.marshalizer.Unmarshal(header, dta)
	if err != nil {
		log.Debug("DecodeBlockHeader.Unmarshal", "error", err.Error())
		return nil
	}

	return header
}

func (bp *baseProcessor) saveBody(body *block.Body) {
	errNotCritical := bp.txCoordinator.SaveBlockDataToStorage(body)
	if errNotCritical != nil {
		log.Warn("saveBody.SaveBlockDataToStorage", "error", errNotCritical.Error())
	}

	for i := 0; i < len(body.MiniBlocks); i++ {
		marshalizedMiniBlock, errNotCritical := bp.marshalizer.Marshal(body.MiniBlocks[i])
		if errNotCritical != nil {
			log.Warn("saveBody.Marshal", "error", errNotCritical.Error())
			continue
		}

		miniBlockHash := bp.hasher.Compute(string(marshalizedMiniBlock))
		errNotCritical = bp.store.Put(dataRetriever.MiniBlockUnit, miniBlockHash, marshalizedMiniBlock)
		if errNotCritical != nil {
			log.Warn("saveBody.Put -> MiniBlockUnit", "error", errNotCritical.Error())
		}
	}
}

func (bp *baseProcessor) saveShardHeader(header data.HeaderHandler, headerHash []byte, marshalizedHeader []byte) {
	nonceToByteSlice := bp.uint64Converter.ToByteSlice(header.GetNonce())
	hdrNonceHashDataUnit := dataRetriever.ShardHdrNonceHashDataUnit + dataRetriever.UnitType(header.GetShardID())

	errNotCritical := bp.store.Put(hdrNonceHashDataUnit, nonceToByteSlice, headerHash)
	if errNotCritical != nil {
		log.Warn(fmt.Sprintf("saveHeader.Put -> ShardHdrNonceHashDataUnit_%d", header.GetShardID()),
			"error", errNotCritical.Error(),
		)
	}

	errNotCritical = bp.store.Put(dataRetriever.BlockHeaderUnit, headerHash, marshalizedHeader)
	if errNotCritical != nil {
		log.Warn("saveHeader.Put -> BlockHeaderUnit", "error", errNotCritical.Error())
	}
}

func (bp *baseProcessor) saveMetaHeader(header data.HeaderHandler, headerHash []byte, marshalizedHeader []byte) {
	nonceToByteSlice := bp.uint64Converter.ToByteSlice(header.GetNonce())

	errNotCritical := bp.store.Put(dataRetriever.MetaHdrNonceHashDataUnit, nonceToByteSlice, headerHash)
	if errNotCritical != nil {
		log.Warn("saveMetaHeader.Put -> MetaHdrNonceHashDataUnit", "error", errNotCritical.Error())
	}

	errNotCritical = bp.store.Put(dataRetriever.MetaBlockUnit, headerHash, marshalizedHeader)
	if errNotCritical != nil {
		log.Warn("saveMetaHeader.Put -> MetaBlockUnit", "error", errNotCritical.Error())
	}
}

func getLastSelfNotarizedHeaderByItself(chainHandler data.ChainHandler) (data.HeaderHandler, []byte) {
	if check.IfNil(chainHandler.GetCurrentBlockHeader()) {
		return chainHandler.GetGenesisHeader(), chainHandler.GetGenesisHeaderHash()
	}

	return chainHandler.GetCurrentBlockHeader(), chainHandler.GetCurrentBlockHeaderHash()
}

func (bp *baseProcessor) updateStateStorage(
	finalHeader data.HeaderHandler,
	rootHash []byte,
	prevRootHash []byte,
	accounts state.AccountsAdapter,
) {
	if !accounts.IsPruningEnabled() {
		return
	}

	accounts.CancelPrune(rootHash, data.NewRoot)

	if finalHeader.IsStartOfEpochBlock() {
		log.Debug("trie snapshot", "rootHash", rootHash)
		accounts.SnapshotState(rootHash)
	}

	// TODO generate checkpoint on a trigger
	if bp.stateCheckpointModulus != 0 {
		if finalHeader.GetNonce()%uint64(bp.stateCheckpointModulus) == 0 {
			log.Debug("trie checkpoint", "rootHash", rootHash)
			accounts.SetStateCheckpoint(rootHash)
		}
	}

	if bytes.Equal(prevRootHash, rootHash) {
		return
	}

	errNotCritical := accounts.PruneTrie(prevRootHash, data.OldRoot)
	if errNotCritical != nil {
		log.Debug(errNotCritical.Error())
	}
}

// RevertAccountState reverts the account state for cleanup failed process
func (bp *baseProcessor) RevertAccountState(_ data.HeaderHandler) {
	for key := range bp.accountsDB {
		err := bp.accountsDB[key].RevertToSnapshot(0)
		if err != nil {
			log.Debug("RevertToSnapshot", "error", err.Error())
		}
	}
}

func (bp *baseProcessor) commitAll() error {
	for key := range bp.accountsDB {
		_, err := bp.accountsDB[key].Commit()
		if err != nil {
			return err
		}
	}

	return nil
}

// PruneStateOnRollback recreates the state tries to the root hashes indicated by the provided header
func (bp *baseProcessor) PruneStateOnRollback(currHeader data.HeaderHandler, prevHeader data.HeaderHandler) {
	for key := range bp.accountsDB {
		if !bp.accountsDB[key].IsPruningEnabled() {
			return
		}

		rootHash, prevRootHash := bp.getRootHashes(currHeader, prevHeader, key)

		if bytes.Equal(rootHash, prevRootHash) {
			return
		}

		bp.accountsDB[key].CancelPrune(prevRootHash, data.OldRoot)

		errNotCritical := bp.accountsDB[key].PruneTrie(rootHash, data.NewRoot)
		if errNotCritical != nil {
			log.Debug(errNotCritical.Error())
		}
	}
}

func (bp *baseProcessor) getRootHashes(currHeader data.HeaderHandler, prevHeader data.HeaderHandler, identifier state.AccountsDbIdentifier) ([]byte, []byte) {
	switch identifier {
	case state.UserAccountsState:
		return currHeader.GetRootHash(), prevHeader.GetRootHash()
	case state.PeerAccountsState:
		return currHeader.GetValidatorStatsRootHash(), prevHeader.GetValidatorStatsRootHash()
	default:
		return []byte{}, []byte{}
	}
}
