package maps

import (
	"sync"
)

// This implementation is a simplified version of:
// https://github.com/ElrondNetwork/concurrent-map, which is based on:
// https://github.com/orcaman/concurrent-map

// ConcurrentMap is a thread safe map of type string:Anything.
// To avoid lock bottlenecks this map is divided to several map chunks.
type ConcurrentMap struct {
	mutex   sync.Mutex
	nChunks uint32
	chunks  []*concurrentMapChunk
}

// concurrentMapChunk is a thread safe string to anything map.
type concurrentMapChunk struct {
	items map[string]interface{}
	sync.RWMutex
}

// NewConcurrentMap creates a new concurrent map.
func NewConcurrentMap(nChunks uint32) *ConcurrentMap {
	// We cannot have a map with no chunks
	if nChunks == 0 {
		nChunks = 1
	}

	m := ConcurrentMap{
		nChunks: nChunks,
	}

	m.initializeChunks()

	return &m
}

func (m *ConcurrentMap) initializeChunks() {
	// Assignment is not an atomic operation, so we have to wrap this in a critical section
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.chunks = make([]*concurrentMapChunk, m.nChunks)

	for i := uint32(0); i < m.nChunks; i++ {
		m.chunks[i] = &concurrentMapChunk{
			items: make(map[string]interface{}),
		}
	}
}

// getChunk returns the chunk holding the given key.
func (m *ConcurrentMap) getChunk(key string) *concurrentMapChunk {
	return m.chunks[fnv32(key)%m.nChunks]
}

// Set sets the given value under the specified key.
func (m *ConcurrentMap) Set(key string, value interface{}) {
	chunk := m.getChunk(key)
	chunk.Lock()
	chunk.items[key] = value
	chunk.Unlock()
}

// SetIfAbsent sets the given value under the specified key if no value was associated with it.
func (m *ConcurrentMap) SetIfAbsent(key string, value interface{}) bool {
	// Get map shard.
	chunk := m.getChunk(key)
	chunk.Lock()
	_, ok := chunk.items[key]
	if !ok {
		chunk.items[key] = value
	}
	chunk.Unlock()
	return !ok
}

// Get retrieves an element from map under given key.
func (m *ConcurrentMap) Get(key string) (interface{}, bool) {
	chunk := m.getChunk(key)
	chunk.RLock()
	val, ok := chunk.items[key]
	chunk.RUnlock()
	return val, ok
}

// Count returns the number of elements within the map.
func (m *ConcurrentMap) Count() int {
	count := 0
	for i := uint32(0); i < m.nChunks; i++ {
		chunk := m.chunks[i]
		chunk.RLock()
		count += len(chunk.items)
		chunk.RUnlock()
	}
	return count
}

// Has looks up an item under specified key.
func (m *ConcurrentMap) Has(key string) bool {
	chunk := m.getChunk(key)
	chunk.RLock()
	_, ok := chunk.items[key]
	chunk.RUnlock()
	return ok
}

// Remove removes an element from the map.
func (m *ConcurrentMap) Remove(key string) {
	chunk := m.getChunk(key)
	chunk.Lock()
	delete(chunk.items, key)
	chunk.Unlock()
}

// IterCb is an iterator callback
type IterCb func(key string, v interface{})

// IterCb iterates over the map (cheapest way to read all elements in a map)
func (m *ConcurrentMap) IterCb(fn IterCb) {
	for idx := range m.chunks {
		chunk := (m.chunks)[idx]
		chunk.RLock()
		for key, value := range chunk.items {
			fn(key, value)
		}
		chunk.RUnlock()
	}
}

// fnv32 implements https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function for 32 bits
func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}

// Clear clears the map
func (m *ConcurrentMap) Clear() {
	// There is no need to explicitly remove each item for each chunk
	// The garbage collector will remove the data from memory
	m.initializeChunks()
}

// Keys returns all keys as []string
func (m *ConcurrentMap) Keys() []string {
	count := m.Count()
	ch := make(chan string, count)
	go func() {
		// Foreach shard.
		wg := sync.WaitGroup{}
		wg.Add(int(m.nChunks))
		for _, shard := range m.chunks {
			go func(shard *concurrentMapChunk) {
				// Foreach key, value pair.
				shard.RLock()
				for key := range shard.items {
					ch <- key
				}
				shard.RUnlock()
				wg.Done()
			}(shard)
		}
		wg.Wait()
		close(ch)
	}()

	// Generate keys
	keys := make([]string, 0, count)
	for k := range ch {
		keys = append(keys, k)
	}
	return keys
}
