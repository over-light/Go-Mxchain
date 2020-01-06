package headersCache

import (
	"bytes"
	"github.com/ElrondNetwork/elrond-go/core"
	"time"

	"github.com/ElrondNetwork/elrond-go/data"
)

type headersCache struct {
	headersNonceCache map[uint32]listOfHeadersByNonces

	headersByHash  headersByHashMap
	headersCounter numHeadersByShard

	numHeadersToRemove int
	maxHeadersPerShard int
}

func newHeadersCache(numMaxHeaderPerShard int, numHeadersToRemove int) *headersCache {
	return &headersCache{
		headersNonceCache:  make(map[uint32]listOfHeadersByNonces),
		headersCounter:     make(numHeadersByShard),
		headersByHash:      make(headersByHashMap),
		numHeadersToRemove: numHeadersToRemove,
		maxHeadersPerShard: numMaxHeaderPerShard,
	}
}

func (cache *headersCache) addHeader(headerHash []byte, header data.HeaderHandler) bool {
	headerShardId := header.GetShardID()
	headerNonce := header.GetNonce()

	//check if pool is full and if it is do eviction
	cache.tryToDoEviction(headerShardId)

	// add header info in second map
	added := cache.headersByHash.addElement(headerHash, headerInfo{headerNonce, headerShardId})
	if added {
		return false
	}

	shard := cache.getShardMap(headerShardId)
	shard.appendHeaderToList(headerHash, header)

	cache.headersCounter.increment(headerShardId)

	return true

}

func (cache *headersCache) tryToDoEviction(shardId uint32) {
	numHeaders := cache.getNumHeaders(shardId)
	if int(numHeaders) >= cache.maxHeadersPerShard {
		cache.lruEviction(shardId)
	}

	return
}

func (cache *headersCache) lruEviction(shardId uint32) {
	shard, ok := cache.headersNonceCache[shardId]
	if !ok {
		return
	}

	nonces := shard.getNoncesSortedByTimestamp()

	numHashes := 0
	maxItemsToRemove := core.MinInt(cache.numHeadersToRemove, len(nonces))
	for i := 0; i < maxItemsToRemove; i++ {
		numHashes += cache.removeHeaderByNonceAndShardId(nonces[i], shardId)

		if numHashes >= maxItemsToRemove {
			break
		}
	}
}

func (cache *headersCache) getShardMap(shardId uint32) listOfHeadersByNonces {
	if _, ok := cache.headersNonceCache[shardId]; !ok {
		cache.headersNonceCache[shardId] = make(listOfHeadersByNonces)
	}

	return cache.headersNonceCache[shardId]
}

func (cache *headersCache) getNumHeaders(shardId uint32) int64 {
	return cache.headersCounter.getCount(shardId)
}

func (cache *headersCache) removeHeaderByNonceAndShardId(headerNonce uint64, shardId uint32) int {
	shard, ok := cache.headersNonceCache[shardId]
	if !ok {
		return 0
	}

	headers, ok := shard.getHeadersByNonce(headerNonce)
	if !ok {
		return 0
	}
	headersHashes := headers.getHashes()

	//remove items from nonce map
	shard.removeListOfHeaders(headerNonce)
	//remove elements from hashes map
	cache.headersByHash.deleteBulk(headersHashes)

	cache.headersCounter.decrement(shardId, len(headersHashes))

	return len(headersHashes)
}

func (cache *headersCache) removeHeaderByHash(hash []byte) {
	info, ok := cache.headersByHash.getElement(hash)
	if !ok {
		return
	}

	//remove header from first map
	cache.removeHeaderFromNonceMap(info, hash)
	//remove header from second map
	cache.headersByHash.deleteElement(hash)
}

// removeHeaderFromNonceMap will remove a header from headerWithTimestamp
// when a header is removed by hash we need to remove also header from the map where is stored with nonce
func (cache *headersCache) removeHeaderFromNonceMap(headerInfo headerInfo, headerHash []byte) {
	shard, ok := cache.headersNonceCache[headerInfo.headerShardId]
	if !ok {
		return
	}

	headers, ok := shard.getHeadersByNonce(headerInfo.headerNonce)
	if !ok {
		return
	}

	//remove header from header list
	for index, header := range headers.items {
		if !bytes.Equal(header.headerHash, headerHash) {
			continue
		}

		headers.removeHeader(index)
		cache.headersCounter.decrement(headerInfo.headerShardId, 1)

		if headers.isEmpty() {
			shard.removeListOfHeaders(headerInfo.headerNonce)
			return
		}

		shard.setListOfHeaders(headerInfo.headerNonce, headers)
		return
	}
}

func (cache *headersCache) getHeaderByHash(hash []byte) (data.HeaderHandler, error) {
	info, ok := cache.headersByHash.getElement(hash)
	if !ok {
		return nil, ErrHeaderNotFound
	}

	shard, ok := cache.headersNonceCache[info.headerShardId]
	if !ok {
		return nil, ErrHeaderNotFound
	}

	headers := shard.getListOfHeaders(info.headerNonce)
	if headers.isEmpty() {
		return nil, ErrHeaderNotFound
	}

	// update headers timestamp for lru
	headers.timestamp = time.Now()
	shard.setListOfHeaders(info.headerNonce, headers)

	if header, ok := headers.findHeaderByHash(hash); ok {
		return header, nil
	}

	return nil, ErrHeaderNotFound
}

func (cache *headersCache) getHeadersByNonceAndShardId(headerNonce uint64, shardId uint32) ([]headerDetails, bool) {
	shard, ok := cache.headersNonceCache[shardId]
	if !ok {
		return nil, false
	}

	headersList, ok := shard.getHeadersByNonce(headerNonce)
	if !ok {
		return nil, false
	}

	return headersList.items, true
}

func (cache *headersCache) getHeadersAndHashesByNonceAndShardId(nonce uint64, shardId uint32) ([]data.HeaderHandler, [][]byte, bool) {
	headersList, ok := cache.getHeadersByNonceAndShardId(nonce, shardId)
	if !ok || len(headersList) == 0 {
		return nil, nil, false
	}

	headers := make([]data.HeaderHandler, 0, len(headersList))
	hashes := make([][]byte, 0, len(headersList))
	for _, headerDetails := range headersList {
		headers = append(headers, headerDetails.header)
		hashes = append(hashes, headerDetails.headerHash)
	}

	return headers, hashes, true
}

func (cache *headersCache) keys(shardId uint32) []uint64 {
	shardMap := cache.getShardMap(shardId)

	return shardMap.keys()
}

func (cache *headersCache) totalHeaders() int {
	return cache.headersCounter.totalHeaders()
}

func (cache *headersCache) clear() {
	cache.headersNonceCache = make(map[uint32]listOfHeadersByNonces)
	cache.headersCounter = make(numHeadersByShard)
	cache.headersByHash = make(headersByHashMap)
}
