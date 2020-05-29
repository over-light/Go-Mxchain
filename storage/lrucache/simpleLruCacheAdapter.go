package lrucache

import "github.com/ElrondNetwork/elrond-go/storage"

// simpleLRUCacheAdapter provides an adapter between LRUCacheHandler and SizeLRUCacheHandler
type simpleLRUCacheAdapter struct {
	storage.LRUCacheHandler
}

// AddSized calls the Add method without the size in bytes parameter
func (slca *simpleLRUCacheAdapter) AddSized(key, value interface{}, _ int64) bool {
	return slca.Add(key, value)
}

// ContainsOrAddSized calls ContainsOrAdd without the size in bytes parameter
func (slca *simpleLRUCacheAdapter) ContainsOrAddSized(key, value interface{}, _ int64) (ok, evicted bool) {
	return slca.ContainsOrAdd(key, value)
}
