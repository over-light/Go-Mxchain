package track

import (
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

const PercentToKeep = percentToKeep

// metaBlockTrack

func (sbt *shardBlockTrack) GetNumPendingMiniBlocks(shardID uint32) uint32 {
	return sbt.blockBalancer.GetNumPendingMiniBlocks(shardID)
}

// baseBlockTrack

func (bbt *baseBlockTrack) ReceivedHeader(headerHandler data.HeaderHandler, headerHash []byte) {
	bbt.receivedHeader(headerHandler, headerHash)
}

func CheckTrackerNilParameters(arguments ArgBaseTracker) error {
	return checkTrackerNilParameters(arguments)
}

func (bbt *baseBlockTrack) InitNotarizedHeaders(startHeaders map[uint32]data.HeaderHandler) error {
	return bbt.initNotarizedHeaders(startHeaders)
}

func (bbt *baseBlockTrack) ReceivedShardHeader(headerHandler data.HeaderHandler, shardHeaderHash []byte) {
	bbt.receivedShardHeader(headerHandler, shardHeaderHash)
}

func (bbt *baseBlockTrack) ReceivedMetaBlock(headerHandler data.HeaderHandler, metaBlockHash []byte) {
	bbt.receivedMetaBlock(headerHandler, metaBlockHash)
}

func (bbt *baseBlockTrack) GetMaxNumHeadersToKeepPerShard() int {
	return bbt.maxNumHeadersToKeepPerShard
}

func (bbt *baseBlockTrack) ShouldAddHeaderForCrossShard(headerHandler data.HeaderHandler) bool {
	return bbt.shouldAddHeaderForShard(headerHandler, bbt.crossNotarizer, headerHandler.GetShardID())
}

func (bbt *baseBlockTrack) ShouldAddHeaderForSelfShard(headerHandler data.HeaderHandler) bool {
	return bbt.shouldAddHeaderForShard(headerHandler, bbt.selfNotarizer, core.MetachainShardId)
}

func (bbt *baseBlockTrack) AddHeader(header data.HeaderHandler, hash []byte) {
	bbt.addHeader(header, hash)
}

func (bbt *baseBlockTrack) AppendTrackedHeader(headerHandler data.HeaderHandler) {
	bbt.mutHeaders.Lock()
	headersForShard, ok := bbt.headers[headerHandler.GetShardID()]
	if !ok {
		headersForShard = make(map[uint64][]*HeaderInfo)
		bbt.headers[headerHandler.GetShardID()] = headersForShard
	}

	headersForShard[headerHandler.GetNonce()] = append(headersForShard[headerHandler.GetNonce()], &HeaderInfo{Header: headerHandler})
	bbt.mutHeaders.Unlock()
}

func (bbt *baseBlockTrack) CleanupWhenMaxCapacityIsReached(shardID uint32) {
	bbt.cleanupWhenMaxCapacityIsReached(shardID)
}

func (bbt *baseBlockTrack) CleanupTrackedHeadersBehindNonce(shardID uint32, nonce uint64) {
	bbt.cleanupTrackedHeadersBehindNonce(shardID, nonce)
}

func (bbt *baseBlockTrack) DisplayTrackedHeadersForShard(shardID uint32, message string) {
	bbt.displayTrackedHeadersForShard(shardID, message)
}

func (bbt *baseBlockTrack) SetRounder(rounder process.Rounder) {
	bbt.rounder = rounder
}

func (bbt *baseBlockTrack) SetCrossNotarizer(notarizer blockNotarizerHandler) {
	bbt.crossNotarizer = notarizer
}

func (bbt *baseBlockTrack) SetSelfNotarizer(notarizer blockNotarizerHandler) {
	bbt.selfNotarizer = notarizer
}

func (bbt *baseBlockTrack) SetShardCoordinator(coordinator sharding.Coordinator) {
	bbt.shardCoordinator = coordinator
}

func NewBaseBlockTrack() *baseBlockTrack {
	return &baseBlockTrack{}
}

// blockNotifier

func (bn *blockNotifier) GetNotarizedHeadersHandlers() []func(shardID uint32, headers []data.HeaderHandler, headersHashes [][]byte) {
	bn.mutNotarizedHeadersHandlers.RLock()
	notarizedHeadersHandlers := bn.notarizedHeadersHandlers
	bn.mutNotarizedHeadersHandlers.RUnlock()

	return notarizedHeadersHandlers
}

// blockNotarizer

func (bn *blockNotarizer) AppendNotarizedHeader(headerHandler data.HeaderHandler) {
	bn.mutNotarizedHeaders.Lock()
	bn.notarizedHeaders[headerHandler.GetShardID()] = append(bn.notarizedHeaders[headerHandler.GetShardID()], &HeaderInfo{Header: headerHandler})
	bn.mutNotarizedHeaders.Unlock()
}

func (bn *blockNotarizer) CleanupWhenMaxCapacityIsReached(shardID uint32) {
	bn.cleanupWhenMaxCapacityIsReached(shardID)
}

func (bn *blockNotarizer) GetNotarizedHeaders() map[uint32][]*HeaderInfo {
	bn.mutNotarizedHeaders.RLock()
	notarizedHeaders := bn.notarizedHeaders
	bn.mutNotarizedHeaders.RUnlock()

	return notarizedHeaders
}

func (bn *blockNotarizer) GetNotarizedHeaderWithIndex(shardID uint32, index int) data.HeaderHandler {
	bn.mutNotarizedHeaders.RLock()
	notarizedHeader := bn.notarizedHeaders[shardID][index].Header
	bn.mutNotarizedHeaders.RUnlock()

	return notarizedHeader
}

func (bn *blockNotarizer) LastNotarizedHeaderInfo(shardID uint32) *HeaderInfo {
	return bn.lastNotarizedHeaderInfo(shardID)
}

func (bn *blockNotarizer) GetMaxNumHeadersToKeepPerShard() int {
	return bn.maxNumHeadersToKeepPerShard
}

// blockProcessor

func (bp *blockProcessor) DoJobOnReceivedHeader(shardID uint32) {
	bp.doJobOnReceivedHeader(shardID)
}

func (bp *blockProcessor) DoJobOnReceivedCrossNotarizedHeader(shardID uint32) {
	bp.doJobOnReceivedCrossNotarizedHeader(shardID)
}

func (bp *blockProcessor) ComputeLongestChainFromLastCrossNotarized(shardID uint32) (data.HeaderHandler, []byte, []data.HeaderHandler, [][]byte) {
	return bp.computeLongestChainFromLastCrossNotarized(shardID)
}

func (bp *blockProcessor) ComputeSelfNotarizedHeaders(headers []data.HeaderHandler) ([]data.HeaderHandler, [][]byte) {
	return bp.computeSelfNotarizedHeaders(headers)
}

func (bp *blockProcessor) GetNextHeader(longestChainHeadersIndexes *[]int, headersIndexes []int, prevHeader data.HeaderHandler, sortedHeaders []data.HeaderHandler, index int) {
	bp.getNextHeader(longestChainHeadersIndexes, headersIndexes, prevHeader, sortedHeaders, index)
}

func (bp *blockProcessor) CheckHeaderFinality(header data.HeaderHandler, sortedHeaders []data.HeaderHandler, index int) error {
	return bp.checkHeaderFinality(header, sortedHeaders, index)
}

func (bp *blockProcessor) RequestHeadersIfNeeded(lastNotarizedHeader data.HeaderHandler, sortedHeaders []data.HeaderHandler, longestChainHeaders []data.HeaderHandler) {
	bp.requestHeadersIfNeeded(lastNotarizedHeader, sortedHeaders, longestChainHeaders)
}

func (bp *blockProcessor) GetLatestValidHeader(lastNotarizedHeader data.HeaderHandler, longestChainHeaders []data.HeaderHandler) data.HeaderHandler {
	return bp.getLatestValidHeader(lastNotarizedHeader, longestChainHeaders)
}

func (bp *blockProcessor) GetHighestRoundInReceivedHeaders(latestValidHeader data.HeaderHandler, sortedReceivedHeaders []data.HeaderHandler) uint64 {
	return bp.getHighestRoundInReceivedHeaders(latestValidHeader, sortedReceivedHeaders)
}

func (bp *blockProcessor) RequestHeadersIfNothingNewIsReceived(lastNotarizedHeaderNonce uint64, latestValidHeader data.HeaderHandler, highestRoundInReceivedHeaders uint64) {
	bp.requestHeadersIfNothingNewIsReceived(lastNotarizedHeaderNonce, latestValidHeader, highestRoundInReceivedHeaders)
}
