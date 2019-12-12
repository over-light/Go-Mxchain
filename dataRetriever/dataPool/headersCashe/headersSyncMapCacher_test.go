package headersCashe_test

import (
	"fmt"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/dataPool/headersCashe"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func createASliceOfHeaders(numHeaders int, shardId uint32) ([]block.Header, [][]byte) {
	headers := make([]block.Header, 0)
	headersHashes := make([][]byte, 0)
	for i := 0; i < numHeaders; i++ {
		headers = append(headers, block.Header{Nonce: uint64(i), ShardId: shardId})
		headersHashes = append(headersHashes, []byte(fmt.Sprintf("%d", i)))
	}

	return headers, headersHashes
}

func TestNewHeadersCacher_AddHeadersInCache(t *testing.T) {
	t.Parallel()

	hdrsCacher, _ := headersCashe.NewHeadersCacher(1000, 100)

	nonce := uint64(1)
	shardId := uint32(0)

	hdrHash1 := []byte("hash1")
	hdrHash2 := []byte("hash2")
	testHdr1 := &block.Header{Nonce: nonce, ShardId: shardId}
	testHdr2 := &block.Header{Nonce: nonce, ShardId: shardId, Round: 100}

	hdrsCacher.Add(hdrHash1, testHdr1)
	hdrsCacher.Add(hdrHash2, testHdr2)

	hdr, err := hdrsCacher.GetHeaderByHash(hdrHash1)
	assert.Nil(t, err)
	assert.Equal(t, testHdr1, hdr)

	hdr, err = hdrsCacher.GetHeaderByHash(hdrHash2)
	assert.Nil(t, err)
	assert.Equal(t, testHdr2, hdr)

	expectedHeaders := []data.HeaderHandler{testHdr1, testHdr2}
	hdrs, err := hdrsCacher.GetHeaderByNonceAndShardId(nonce, shardId)
	assert.Nil(t, err)
	assert.Equal(t, expectedHeaders, hdrs)
}

func TestHeadersCacher_AddHeadersInCacheAndRemoveByHash(t *testing.T) {
	t.Parallel()

	hdrsCacher, _ := headersCashe.NewHeadersCacher(1000, 100)

	nonce := uint64(1)
	shardId := uint32(0)

	hdrHash1 := []byte("hash1")
	hdrHash2 := []byte("hash2")
	testHdr1 := &block.Header{Nonce: nonce, ShardId: shardId}
	testHdr2 := &block.Header{Nonce: nonce, ShardId: shardId, Round: 100}

	hdrsCacher.Add(hdrHash1, testHdr1)
	hdrsCacher.Add(hdrHash2, testHdr2)

	hdrsCacher.RemoveHeaderByHash(hdrHash1)
	hdr, err := hdrsCacher.GetHeaderByHash(hdrHash1)
	assert.Nil(t, hdr)
	assert.Equal(t, headersCashe.ErrHeaderNotFound, err)

	hdrsCacher.RemoveHeaderByHash(hdrHash2)
	hdr, err = hdrsCacher.GetHeaderByHash(hdrHash2)
	assert.Nil(t, hdr)
	assert.Equal(t, headersCashe.ErrHeaderNotFound, err)
}

func TestHeadersCacher_AddHeadersInCacheAndRemoveByNonceAndShadId(t *testing.T) {
	t.Parallel()

	hdrsCacher, _ := headersCashe.NewHeadersCacher(1000, 100)

	nonce := uint64(1)
	shardId := uint32(0)

	hdrHash1 := []byte("hash1")
	hdrHash2 := []byte("hash2")
	testHdr1 := &block.Header{Nonce: nonce, ShardId: shardId}
	testHdr2 := &block.Header{Nonce: nonce, ShardId: shardId, Round: 100}

	hdrsCacher.Add(hdrHash1, testHdr1)
	hdrsCacher.Add(hdrHash2, testHdr2)

	hdrsCacher.RemoveHeaderByNonceAndShardId(nonce, shardId)
	hdr, err := hdrsCacher.GetHeaderByHash(hdrHash1)
	assert.Nil(t, hdr)
	assert.Equal(t, headersCashe.ErrHeaderNotFound, err)

	hdr, err = hdrsCacher.GetHeaderByHash(hdrHash2)
	assert.Nil(t, hdr)
	assert.Equal(t, headersCashe.ErrHeaderNotFound, err)
}

func TestHeadersCacher_EvictionShouldWork(t *testing.T) {
	t.Parallel()

	hdrs, hdrsHashes := createASliceOfHeaders(1000, 0)
	hdrsCacher, _ := headersCashe.NewHeadersCacher(900, 100)

	for i := 0; i < 1000; i++ {
		hdrsCacher.Add(hdrsHashes[i], &hdrs[i])
	}

	// Cache will do eviction 2 times, in headers cache will be 800 headers
	for i := 200; i < 1000; i++ {
		hdr, err := hdrsCacher.GetHeaderByHash(hdrsHashes[i])
		assert.Nil(t, err)
		assert.Equal(t, &hdrs[i], hdr)
	}
}

func TestHeadersCacher_ConcurrentRequestsShouldWorkNoEviction(t *testing.T) {
	t.Parallel()

	numHeadersToGenerate := 500

	hdrs, hdrsHashes := createASliceOfHeaders(numHeadersToGenerate, 0)
	hdrsCacher, _ := headersCashe.NewHeadersCacher(numHeadersToGenerate+1, 10)

	for i := 0; i < numHeadersToGenerate; i++ {
		go func(index int) {
			hdrsCacher.Add(hdrsHashes[index], &hdrs[index])
			hdr, err := hdrsCacher.GetHeaderByHash(hdrsHashes[index])

			assert.Nil(t, err)
			assert.Equal(t, &hdrs[index], hdr)
		}(i)
	}
}

func TestHeadersCacher_ConcurrentRequestsShouldWorkWithEviction(t *testing.T) {
	shardId := uint32(0)
	cacheSize := 2
	numHeadersToGenerate := 500

	hdrs, hdrsHashes := createASliceOfHeaders(numHeadersToGenerate, shardId)
	hdrsCacher, _ := headersCashe.NewHeadersCacher(cacheSize, 1)

	for i := 0; i < numHeadersToGenerate; i++ {
		go func(index int) {
			hdrsCacher.Add(hdrsHashes[index], &hdrs[index])
		}(i)
	}
	time.Sleep(time.Second)
	// cache size after all eviction is finish should be 2
	assert.Equal(t, 2, hdrsCacher.GetNumHeadersFromCacheShard(shardId))

	numHeadersToGenerate = 3
	hdrs, hdrsHashes = createASliceOfHeaders(3, shardId)
	for i := 0; i < numHeadersToGenerate; i++ {
		hdrsCacher.Add(hdrsHashes[i], &hdrs[i])
	}
	time.Sleep(time.Second)

	assert.Equal(t, 2, hdrsCacher.GetNumHeadersFromCacheShard(shardId))
	hdr1, err := hdrsCacher.GetHeaderByHash(hdrsHashes[1])
	assert.Nil(t, err)
	assert.Equal(t, &hdrs[1], hdr1)

	hdr2, err := hdrsCacher.GetHeaderByHash(hdrsHashes[2])
	assert.Nil(t, err)
	assert.Equal(t, &hdrs[2], hdr2)
}

func TestHeadersCacher_AddHeadersWithSameNonceShouldBeRemovedAtEviction(t *testing.T) {
	t.Parallel()

	shardId := uint32(0)
	cacheSize := 2

	hash1, hash2, hash3 := []byte("hash1"), []byte("hash2"), []byte("hash3")
	hdr1, hdr2, hdr3 := &block.Header{Nonce: 0}, &block.Header{Nonce: 0}, &block.Header{Nonce: 1}

	hdrsCacher, _ := headersCashe.NewHeadersCacher(cacheSize, 1)
	hdrsCacher.Add(hash1, hdr1)
	hdrsCacher.Add(hash2, hdr2)
	hdrsCacher.Add(hash3, hdr3)

	time.Sleep(time.Second)
	assert.Equal(t, 1, hdrsCacher.GetNumHeadersFromCacheShard(shardId))

	hdr, err := hdrsCacher.GetHeaderByHash(hash3)
	assert.Nil(t, err)
	assert.Equal(t, hdr3, hdr)
}

func TestHeadersCacher_AddALotOfHeadersAndCheckEviction(t *testing.T) {
	t.Parallel()

	cacheSize := 100
	numHeaders := 500
	shardId := uint32(0)
	hdrs, hdrsHash := createASliceOfHeaders(numHeaders, shardId)
	hdrsCacher, _ := headersCashe.NewHeadersCacher(cacheSize, 50)

	for i := 0; i < numHeaders; i++ {
		go func(index int) {
			hdrsCacher.Add(hdrsHash[index], &hdrs[index])
		}(i)
	}

	time.Sleep(time.Second)
	assert.Equal(t, 100, hdrsCacher.GetNumHeadersFromCacheShard(shardId))
}

func TestHeadersCacher_BigCacheALotOfHeadersShouldWork(t *testing.T) {
	t.Parallel()

	cacheSize := 100000
	numHeadersToGenerate := cacheSize
	shardId := uint32(0)

	hdrs, hdrsHash := createASliceOfHeaders(numHeadersToGenerate, shardId)
	hdrsCacher, _ := headersCashe.NewHeadersCacher(cacheSize, 50)

	start := time.Now()
	for i := 0; i < numHeadersToGenerate; i++ {
		hdrsCacher.Add(hdrsHash[i], &hdrs[i])
	}
	elapsed := time.Since(start)
	fmt.Printf("insert %d took %s \n", numHeadersToGenerate, elapsed)

	start = time.Now()
	hdr, _ := hdrsCacher.GetHeaderByHash(hdrsHash[100])
	elapsed = time.Since(start)
	assert.Equal(t, &hdrs[100], hdr)
	fmt.Printf("get header by hash took %s \n", elapsed)

	start = time.Now()
	d, _ := hdrsCacher.GetHeaderByNonceAndShardId(uint64(100), shardId)
	elapsed = time.Since(start)
	fmt.Printf("get header by shard id and nonce took %s \n", elapsed)
	assert.Equal(t, &hdrs[100], d[0])

	start = time.Now()
	hdrsCacher.RemoveHeaderByNonceAndShardId(uint64(500), shardId)
	elapsed = time.Since(start)
	fmt.Printf("remove header by shard id and nonce took %s \n", elapsed)

	hdr, err := hdrsCacher.GetHeaderByHash(hdrsHash[500])
	assert.Error(t, headersCashe.ErrHeaderNotFound, err)

	start = time.Now()
	hdrsCacher.RemoveHeaderByHash(hdrsHash[2012])
	elapsed = time.Since(start)
	fmt.Printf("remove header by hash took %s \n", elapsed)

	hdr, err = hdrsCacher.GetHeaderByHash(hdrsHash[2012])
	assert.Error(t, headersCashe.ErrHeaderNotFound, err)
}
