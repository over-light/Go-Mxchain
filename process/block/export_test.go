package block

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/process/mock"
)

func (bp *baseProcessor) ComputeHeaderHash(hdr data.HeaderHandler) ([]byte, error) {
	return core.CalculateHash(bp.marshalizer, bp.hasher, hdr)
}

func (bp *baseProcessor) VerifyStateRoot(rootHash []byte) bool {
	return bp.verifyStateRoot(rootHash)
}

func (bp *baseProcessor) CheckBlockValidity(
	headerHandler data.HeaderHandler,
	bodyHandler data.BodyHandler,
) error {
	return bp.checkBlockValidity(headerHandler, bodyHandler)
}

func (bp *baseProcessor) RemoveHeadersBehindNonceFromPools(
	shouldRemoveBlockBody bool,
	headersPool dataRetriever.HeadersPool,
	shardId uint32,
	nonce uint64,
) {
	bp.removeHeadersBehindNonceFromPools(shouldRemoveBlockBody, headersPool, shardId, nonce)
}

func (sp *shardProcessor) ReceivedMetaBlock(header data.HeaderHandler, metaBlockHash []byte) {
	sp.receivedMetaBlock(header, metaBlockHash)
}

func (sp *shardProcessor) CreateMiniBlocks(haveTime func() bool) (*block.Body, error) {
	return sp.createMiniBlocks(haveTime)
}

func (sp *shardProcessor) GetOrderedProcessedMetaBlocksFromHeader(header *block.Header) ([]data.HeaderHandler, error) {
	return sp.getOrderedProcessedMetaBlocksFromHeader(header)
}

func (sp *shardProcessor) RemoveProcessedMetaBlocksFromPool(processedMetaHdrs []data.HeaderHandler) error {
	return sp.removeProcessedMetaBlocksFromPool(processedMetaHdrs)
}

func (sp *shardProcessor) UpdateStateStorage(finalHeaders []data.HeaderHandler) {
	sp.updateState(finalHeaders)
}

func NewShardProcessorEmptyWith3shards(
	tdp dataRetriever.PoolsHolder,
	genesisBlocks map[uint32]data.HeaderHandler,
	blockChain data.ChainHandler,
) (*shardProcessor, error) {
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(3)
	nodesCoordinator := mock.NewNodesCoordinatorMock()

	argsHeaderValidator := ArgsHeaderValidator{
		Hasher:      &mock.HasherMock{},
		Marshalizer: &mock.MarshalizerMock{},
	}
	hdrValidator, _ := NewHeaderValidator(argsHeaderValidator)

	accountsDb := make(map[state.AccountsDbIdentifier]state.AccountsAdapter)
	accountsDb[state.UserAccountsState] = &mock.AccountsStub{}

	arguments := ArgShardProcessor{
		ArgBaseProcessor: ArgBaseProcessor{
			AccountsDB:        accountsDb,
			ForkDetector:      &mock.ForkDetectorMock{},
			Hasher:            &mock.HasherMock{},
			Marshalizer:       &mock.MarshalizerMock{},
			Store:             &mock.ChainStorerMock{},
			ShardCoordinator:  shardCoordinator,
			NodesCoordinator:  nodesCoordinator,
			FeeHandler:        &mock.FeeAccumulatorStub{},
			Uint64Converter:   &mock.Uint64ByteSliceConverterMock{},
			RequestHandler:    &mock.RequestHandlerStub{},
			Core:              &mock.ServiceContainerMock{},
			BlockChainHook:    &mock.BlockChainHookHandlerMock{},
			TxCoordinator:     &mock.TransactionCoordinatorMock{},
			EpochStartTrigger: &mock.EpochStartTriggerStub{},
			HeaderValidator:   hdrValidator,
			Rounder:           &mock.RounderMock{},
			BootStorer: &mock.BoostrapStorerMock{
				PutCalled: func(round int64, bootData bootstrapStorage.BootstrapData) error {
					return nil
				},
			},
			BlockTracker: mock.NewBlockTrackerMock(shardCoordinator, genesisBlocks),
			DataPool:     tdp,
			BlockChain:   blockChain,
		},

		TxsPoolsCleaner: &mock.TxPoolsCleanerMock{},
	}
	shardProc, err := NewShardProcessor(arguments)
	return shardProc, err
}

func (mp *metaProcessor) RequestBlockHeaders(header *block.MetaBlock) (uint32, uint32) {
	return mp.requestShardHeaders(header)
}

func (mp *metaProcessor) RemoveBlockInfoFromPool(header *block.MetaBlock) error {
	return mp.removeBlockInfoFromPool(header)
}

func (mp *metaProcessor) ReceivedShardHeader(header data.HeaderHandler, shardHeaderHash []byte) {
	mp.receivedShardHeader(header, shardHeaderHash)
}

func (mp *metaProcessor) AddHdrHashToRequestedList(hdr *block.Header, hdrHash []byte) {
	mp.hdrsForCurrBlock.mutHdrsForBlock.Lock()
	defer mp.hdrsForCurrBlock.mutHdrsForBlock.Unlock()

	if mp.hdrsForCurrBlock.hdrHashAndInfo == nil {
		mp.hdrsForCurrBlock.hdrHashAndInfo = make(map[string]*hdrInfo)
	}

	if mp.hdrsForCurrBlock.highestHdrNonce == nil {
		mp.hdrsForCurrBlock.highestHdrNonce = make(map[uint32]uint64, mp.shardCoordinator.NumberOfShards())
	}

	mp.hdrsForCurrBlock.hdrHashAndInfo[string(hdrHash)] = &hdrInfo{hdr: hdr, usedInBlock: true}
	mp.hdrsForCurrBlock.missingHdrs++
}

func (mp *metaProcessor) IsHdrMissing(hdrHash []byte) bool {
	mp.hdrsForCurrBlock.mutHdrsForBlock.RLock()
	defer mp.hdrsForCurrBlock.mutHdrsForBlock.RUnlock()

	hdrInfoValue, ok := mp.hdrsForCurrBlock.hdrHashAndInfo[string(hdrHash)]
	if !ok {
		return true
	}

	return check.IfNil(hdrInfoValue.hdr)
}

func (mp *metaProcessor) CreateShardInfo() ([]block.ShardData, error) {
	return mp.createShardInfo()
}

func (mp *metaProcessor) RequestMissingFinalityAttestingShardHeaders() uint32 {
	mp.hdrsForCurrBlock.mutHdrsForBlock.Lock()
	defer mp.hdrsForCurrBlock.mutHdrsForBlock.Unlock()

	return mp.requestMissingFinalityAttestingShardHeaders()
}

func (bp *baseProcessor) NotarizedHdrs() map[uint32][]data.HeaderHandler {
	lastCrossNotarizedHeaders := make(map[uint32][]data.HeaderHandler)
	for shardID := uint32(0); shardID < bp.shardCoordinator.NumberOfShards(); shardID++ {
		lastCrossNotarizedHeaderForShard := bp.LastNotarizedHdrForShard(shardID)
		if !check.IfNil(lastCrossNotarizedHeaderForShard) {
			lastCrossNotarizedHeaders[shardID] = append(lastCrossNotarizedHeaders[shardID], lastCrossNotarizedHeaderForShard)
		}
	}

	lastCrossNotarizedHeaderForShard := bp.LastNotarizedHdrForShard(core.MetachainShardId)
	if !check.IfNil(lastCrossNotarizedHeaderForShard) {
		lastCrossNotarizedHeaders[core.MetachainShardId] = append(lastCrossNotarizedHeaders[core.MetachainShardId], lastCrossNotarizedHeaderForShard)
	}

	return lastCrossNotarizedHeaders
}

func (bp *baseProcessor) LastNotarizedHdrForShard(shardID uint32) data.HeaderHandler {
	lastCrossNotarizedHeaderForShard, _, _ := bp.blockTracker.GetLastCrossNotarizedHeader(shardID)
	if check.IfNil(lastCrossNotarizedHeaderForShard) {
		return nil
	}

	return lastCrossNotarizedHeaderForShard
}

func (bp *baseProcessor) SetMarshalizer(marshal marshal.Marshalizer) {
	bp.marshalizer = marshal
}

func (bp *baseProcessor) SetHasher(hasher hashing.Hasher) {
	bp.hasher = hasher
}

func (bp *baseProcessor) SetHeaderValidator(validator process.HeaderConstructionValidator) {
	bp.headerValidator = validator
}

func (mp *metaProcessor) SetShardBlockFinality(val uint32) {
	mp.hdrsForCurrBlock.mutHdrsForBlock.Lock()
	mp.shardBlockFinality = val
	mp.hdrsForCurrBlock.mutHdrsForBlock.Unlock()
}

func (mp *metaProcessor) SaveLastNotarizedHeader(header *block.MetaBlock) error {
	return mp.saveLastNotarizedHeader(header)
}

func (mp *metaProcessor) CheckShardHeadersValidity(header *block.MetaBlock) (map[uint32]data.HeaderHandler, error) {
	return mp.checkShardHeadersValidity(header)
}

func (mp *metaProcessor) CheckShardHeadersFinality(highestNonceHdrs map[uint32]data.HeaderHandler) error {
	return mp.checkShardHeadersFinality(highestNonceHdrs)
}

func (mp *metaProcessor) CheckHeaderBodyCorrelation(hdr *block.Header, body *block.Body) error {
	return mp.checkHeaderBodyCorrelation(hdr.MiniBlockHeaders, body)
}

func (bp *baseProcessor) IsHdrConstructionValid(currHdr, prevHdr data.HeaderHandler) error {
	return bp.headerValidator.IsHeaderConstructionValid(currHdr, prevHdr)
}

func (mp *metaProcessor) ChRcvAllHdrs() chan bool {
	return mp.chRcvAllHdrs
}

func (mp *metaProcessor) UpdateShardsHeadersNonce(key uint32, value uint64) {
	mp.updateShardHeadersNonce(key, value)
}

func (mp *metaProcessor) GetShardsHeadersNonce() *sync.Map {
	return mp.shardsHeadersNonce
}

func (sp *shardProcessor) SaveLastNotarizedHeader(shardId uint32, processedHdrs []data.HeaderHandler) error {
	return sp.saveLastNotarizedHeader(shardId, processedHdrs)
}

func (sp *shardProcessor) CheckHeaderBodyCorrelation(hdr *block.Header, body *block.Body) error {
	return sp.checkHeaderBodyCorrelation(hdr.MiniBlockHeaders, body)
}

func (sp *shardProcessor) CheckAndRequestIfMetaHeadersMissing(round uint64) {
	sp.checkAndRequestIfMetaHeadersMissing(round)
}

func (sp *shardProcessor) GetHashAndHdrStruct(header data.HeaderHandler, hash []byte) *hashAndHdr {
	return &hashAndHdr{header, hash}
}

func (sp *shardProcessor) RequestMissingFinalityAttestingHeaders() uint32 {
	sp.hdrsForCurrBlock.mutHdrsForBlock.Lock()
	defer sp.hdrsForCurrBlock.mutHdrsForBlock.Unlock()

	return sp.requestMissingFinalityAttestingHeaders(
		core.MetachainShardId,
		sp.metaBlockFinality,
	)
}

func (sp *shardProcessor) CheckMetaHeadersValidityAndFinality() error {
	return sp.checkMetaHeadersValidityAndFinality()
}

func (sp *shardProcessor) CreateAndProcessMiniBlocksDstMe(
	haveTime func() bool,
) (block.MiniBlockSlice, uint32, uint32, error) {
	return sp.createAndProcessMiniBlocksDstMe(haveTime)
}

func (sp *shardProcessor) DisplayLogInfo(
	header *block.Header,
	body *block.Body,
	headerHash []byte,
	numShards uint32,
	selfId uint32,
	dataPool dataRetriever.PoolsHolder,
	statusHandler core.AppStatusHandler,
	blockTracker process.BlockTracker,
) {
	sp.txCounter.displayLogInfo(header, body, headerHash, numShards, selfId, dataPool, statusHandler, blockTracker)
}

func (sp *shardProcessor) GetHighestHdrForOwnShardFromMetachain(processedHdrs []data.HeaderHandler) ([]data.HeaderHandler, [][]byte, error) {
	return sp.getHighestHdrForOwnShardFromMetachain(processedHdrs)
}

func (sp *shardProcessor) RestoreMetaBlockIntoPool(
	miniBlockHashes map[string]uint32,
	metaBlockHashes [][]byte,
) error {
	return sp.restoreMetaBlockIntoPool(miniBlockHashes, metaBlockHashes)
}

func (sp *shardProcessor) GetAllMiniBlockDstMeFromMeta(
	header *block.Header,
) (map[string][]byte, error) {
	return sp.getAllMiniBlockDstMeFromMeta(header)
}

func (bp *baseProcessor) SetHdrForCurrentBlock(headerHash []byte, headerHandler data.HeaderHandler, usedInBlock bool) {
	bp.hdrsForCurrBlock.mutHdrsForBlock.Lock()
	bp.hdrsForCurrBlock.hdrHashAndInfo[string(headerHash)] = &hdrInfo{hdr: headerHandler, usedInBlock: usedInBlock}
	bp.hdrsForCurrBlock.mutHdrsForBlock.Unlock()
}

func (bp *baseProcessor) SetHighestHdrNonceForCurrentBlock(shardId uint32, value uint64) {
	bp.hdrsForCurrBlock.mutHdrsForBlock.Lock()
	bp.hdrsForCurrBlock.highestHdrNonce[shardId] = value
	bp.hdrsForCurrBlock.mutHdrsForBlock.Unlock()
}

func (bp *baseProcessor) CreateBlockStarted() {
	bp.createBlockStarted()
}

func (sp *shardProcessor) AddProcessedCrossMiniBlocksFromHeader(header *block.Header) error {
	return sp.addProcessedCrossMiniBlocksFromHeader(header)
}

func (mp *metaProcessor) VerifyCrossShardMiniBlockDstMe(header *block.MetaBlock) error {
	return mp.verifyCrossShardMiniBlockDstMe(header)
}

func (mp *metaProcessor) ApplyBodyToHeader(metaHdr *block.MetaBlock, body *block.Body) (data.BodyHandler, error) {
	return mp.applyBodyToHeader(metaHdr, body)
}

func (sp *shardProcessor) ApplyBodyToHeader(shardHdr *block.Header, body *block.Body) (*block.Body, error) {
	return sp.applyBodyToHeader(shardHdr, body)
}

func (mp *metaProcessor) CreateBlockBody(metaBlock *block.MetaBlock, haveTime func() bool) (data.BodyHandler, error) {
	return mp.createBlockBody(metaBlock, haveTime)
}

func (sp *shardProcessor) CreateBlockBody(shardHdr *block.Header, haveTime func() bool) (data.BodyHandler, error) {
	return sp.createBlockBody(shardHdr, haveTime)
}

func (sp *shardProcessor) CheckEpochCorrectnessCrossChain() error {
	return sp.checkEpochCorrectnessCrossChain()
}
