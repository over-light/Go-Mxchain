package txcache

import (
	"sync"
)

// TODO: ProtectItems(keys) -> current protection & future protection?
// UnprotectItems() = explicit removal, actually.
// Protection is just for eviction (RemoveOldest).
// For protecting existing txs: iterate over keys, fetch items, mark as protected
// For future protection... keep keys in secondary map? Append to that map, replace keys in that map?

// crossTxCache holds cross-shard transactions (where destination == me)
type crossTxCache struct {
	mutex   sync.RWMutex
	nChunks uint32
	chunks  []*crossTxChunk
}

func newCrossTxCache(nChunks uint32) *crossTxCache {
	if nChunks == 0 {
		nChunks = 1
	}

	cache := crossTxCache{
		nChunks: nChunks,
	}

	cache.initializeChunks()
	return &cache
}

func (cache *crossTxCache) initializeChunks() {
	// Assignment is not an atomic operation, so we have to wrap this in a critical section
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	cache.chunks = make([]*crossTxChunk, cache.nChunks)

	for i := uint32(0); i < cache.nChunks; i++ {
		cache.chunks[i] = newCrossTxChunk()
	}
}

// AddItem adds the item in the map
func (cache *crossTxCache) AddItem(item *WrappedTransaction) {
	key := string(item.TxHash)
	chunk := cache.getChunk(key)
	chunk.addItem(item)
}

// Get gets an item from the map
func (cache *crossTxCache) Get(key string) (*WrappedTransaction, bool) {
	chunk := cache.getChunk(key)
	return chunk.getItem(key)
}

// Has returns whether the item is in the map
func (cache *crossTxCache) Has(key string) bool {
	chunk := cache.getChunk(key)
	_, ok := chunk.getItem(key)
	return ok
}

// Remove removes an element from the map
func (cache *crossTxCache) Remove(key string) {
	chunk := cache.getChunk(key)
	chunk.removeItem(key)
}

func (cache *crossTxCache) RemoveOldest(numToRemove int) {
}

// getChunk returns the chunk holding the given key.
func (cache *crossTxCache) getChunk(key string) *crossTxChunk {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	return cache.chunks[fnv32Hash(key)%cache.nChunks]
}

// fnv32Hash implements https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function for 32 bits
func fnv32Hash(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}

// Clear clears the map
func (cache *crossTxCache) Clear() {
	// There is no need to explicitly remove each item for each chunk
	// The garbage collector will remove the data from memory
	cache.initializeChunks()
}

// Count returns the number of elements within the map
func (cache *crossTxCache) Count() uint32 {
	count := uint32(0)
	for _, chunk := range cache.getChunks() {
		count += chunk.countItems()
	}
	return count
}

// Keys returns all keys as []string
func (cache *crossTxCache) Keys() []string {
	count := cache.Count()
	// count is not exact anymore, since we are in a different lock than the one aquired by Count() (but is a good approximation)
	keys := make([]string, 0, count)

	for _, chunk := range cache.getChunks() {
		keys = chunk.appendKeys(keys)
	}

	return keys
}

func (cache *crossTxCache) getChunks() []*crossTxChunk {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	return cache.chunks
}
