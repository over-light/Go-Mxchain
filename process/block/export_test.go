package block

import (
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/display"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

func (bp *baseProcessor) ComputeHeaderHash(hdr data.HeaderHandler) ([]byte, error) {
	return bp.computeHeaderHash(hdr)
}

func (bp *baseProcessor) VerifyStateRoot(rootHash []byte) bool {
	return bp.verifyStateRoot(rootHash)
}

func (bp *baseProcessor) CheckBlockValidity(
	chainHandler data.ChainHandler,
	headerHandler data.HeaderHandler,
	bodyHandler data.BodyHandler,
) error {
	return bp.checkBlockValidity(chainHandler, headerHandler, bodyHandler)
}

func DisplayHeader(headerHandler data.HeaderHandler) []*display.LineData {
	return displayHeader(headerHandler)
}

func (sp *shardProcessor) ReceivedMetaBlock(metaBlockHash []byte) {
	sp.receivedMetaBlock(metaBlockHash)
}

func (sp *shardProcessor) CreateMiniBlocks(noShards uint32, maxTxInBlock int, round uint32, haveTime func() bool) (block.Body, error) {
	return sp.createMiniBlocks(noShards, maxTxInBlock, round, haveTime)
}

func (sp *shardProcessor) GetProcessedMetaBlocksFromPool(body block.Body) ([]data.HeaderHandler, error) {
	return sp.getProcessedMetaBlocksFromPool(body)
}

func (sp *shardProcessor) RemoveProcessedMetablocksFromPool(processedMetaHdrs []data.HeaderHandler) error {
	return sp.removeProcessedMetablocksFromPool(processedMetaHdrs)
}

func (mp *metaProcessor) RequestBlockHeaders(header *block.MetaBlock) (uint32, uint32) {
	return mp.requestShardHeaders(header)
}

func (mp *metaProcessor) RemoveBlockInfoFromPool(header *block.MetaBlock) error {
	return mp.removeBlockInfoFromPool(header)
}

func (mp *metaProcessor) DisplayMetaBlock(header *block.MetaBlock) {
	mp.displayMetaBlock(header)
}

func (mp *metaProcessor) ReceivedHeader(hdrHash []byte) {
	mp.receivedHeader(hdrHash)
}

func (mp *metaProcessor) AddHdrHashToRequestedList(hdrHash []byte) {
	mp.mutRequestedShardHdrsHashes.Lock()
	defer mp.mutRequestedShardHdrsHashes.Unlock()

	if mp.requestedShardHdrsHashes == nil {
		mp.requestedShardHdrsHashes = make(map[string]bool)
		mp.allNeededShardHdrsFound = true
	}

	if mp.currHighestShardHdrsNonces == nil {
		mp.currHighestShardHdrsNonces = make(map[uint32]uint64, mp.shardCoordinator.NumberOfShards())
		for i := uint32(0); i < mp.shardCoordinator.NumberOfShards(); i++ {
			mp.currHighestShardHdrsNonces[i] = uint64(0)
		}
	}

	mp.requestedShardHdrsHashes[string(hdrHash)] = true
	mp.allNeededShardHdrsFound = false
}

func (mp *metaProcessor) IsHdrHashRequested(hdrHash []byte) bool {
	mp.mutRequestedShardHdrsHashes.Lock()
	defer mp.mutRequestedShardHdrsHashes.Unlock()

	_, found := mp.requestedShardHdrsHashes[string(hdrHash)]

	return found
}

func (mp *metaProcessor) CreateShardInfo(maxMiniBlocksInBlock uint32, round uint32, haveTime func() bool) ([]block.ShardData, error) {
	return mp.createShardInfo(maxMiniBlocksInBlock, round, haveTime)
}

func (bp *baseProcessor) LastNotarizedHdrs() map[uint32]data.HeaderHandler {
	return bp.lastNotarizedHdrs
}

func (bp *baseProcessor) SetMarshalizer(marshal marshal.Marshalizer) {
	bp.marshalizer = marshal
}

func (bp *baseProcessor) SetHasher(hasher hashing.Hasher) {
	bp.hasher = hasher
}

func (mp *metaProcessor) SetNextKValidity(val uint32) {
	mp.mutRequestedShardHdrsHashes.Lock()
	mp.nextKValidity = val
	mp.mutRequestedShardHdrsHashes.Unlock()
}

func (mp *metaProcessor) CreateLastNotarizedHdrs(header *block.MetaBlock) error {
	return mp.createLastNotarizedHdrs(header)
}

func (mp *metaProcessor) CheckShardHeadersValidity(header *block.MetaBlock) (mapShardLastHeaders, error) {
	return mp.checkShardHeadersValidity(header)
}

func (mp *metaProcessor) CheckShardHeadersFinality(header *block.MetaBlock, highestNonceHdrs mapShardLastHeaders) error {
	return mp.checkShardHeadersFinality(header, highestNonceHdrs)
}

func (bp *baseProcessor) IsHdrConstructionValid(currHdr, prevHdr data.HeaderHandler) error {
	return bp.isHdrConstructionValid(currHdr, prevHdr)
}

func (mp *metaProcessor) IsShardHeaderValidFinal(currHdr *block.Header, lastHdr *block.Header, sortedShardHdrs []*block.Header) (bool, []uint32) {
	return mp.isShardHeaderValidFinal(currHdr, lastHdr, sortedShardHdrs)
}

func (mp *metaProcessor) ChRcvAllHdrs() chan bool {
	return mp.chRcvAllHdrs
}

func NewBaseProcessor(shardCord sharding.Coordinator) *baseProcessor {
	return &baseProcessor{shardCoordinator: shardCord}
}

func (bp *baseProcessor) SaveLastNotarizedHeader(shardId uint32, processedHdrs []data.HeaderHandler) error {
	return bp.saveLastNotarizedHeader(shardId, processedHdrs)
}

func (sp *shardProcessor) CheckHeaderBodyCorrelation(hdr *block.Header, body block.Body) error {
	return sp.checkHeaderBodyCorrelation(hdr, body)
}

func (bp *baseProcessor) SetLastNotarizedHeadersSlice(startHeaders map[uint32]data.HeaderHandler, metaChainActive bool) error {
	return bp.setLastNotarizedHeadersSlice(startHeaders, metaChainActive)
}
