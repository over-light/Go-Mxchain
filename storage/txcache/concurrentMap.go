package txcache

import (
	"sync"
)

// This implementation is a simplified version of:
// https://github.com/ElrondNetwork/concurrent-map

// ConcurrentMap is a thread safe map of type string:Anything.
// To avoid lock bottlenecks this map is divided to several map chunks.
type ConcurrentMap struct {
	noChunks uint32
	chunks   []*concurrentMapChunk
}

// concurrentMapChunk is a thread safe string to anything map.
type concurrentMapChunk struct {
	maxSize uint32
	items   map[string]interface{}
	sync.RWMutex
}

// NewConcurrentMap creates a new concurrent map.
func NewConcurrentMap(maxSize uint32, noChunks uint32) *ConcurrentMap {
	m := ConcurrentMap{
		noChunks: noChunks,
		chunks:   make([]*concurrentMapChunk, noChunks),
	}

	chunkSize := maxSize / noChunks
	if chunkSize == 0 {
		chunkSize = 1
	}
	if maxSize%noChunks != 0 {
		chunkSize++
	}

	for i := uint32(0); i < noChunks; i++ {
		m.chunks[i] = &concurrentMapChunk{
			maxSize: chunkSize,
			items:   make(map[string]interface{}),
		}
	}

	return &m
}

// getChunk returns the chunk holding the given key.
func (m *ConcurrentMap) getChunk(key string) *concurrentMapChunk {
	return m.chunks[fnv32(key)%m.noChunks]
}

// Set sets the given value under the specified key.
func (m *ConcurrentMap) Set(key string, value interface{}) {
	chunk := m.getChunk(key)
	chunk.Lock()
	chunk.items[key] = value
	chunk.Unlock()
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
	for i := uint32(0); i < m.noChunks; i++ {
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

// IsEmpty checks if map is empty.
func (m *ConcurrentMap) IsEmpty() bool {
	return m.Count() == 0
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

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}
