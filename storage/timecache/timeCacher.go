package timecache

import (
	"context"
	"math"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/storage"
)

var log = logger.GetOrCreate("storage/maptimecache")

const minDuration = time.Second

// ArgTimeCacher is the argument used to create a new timeCacher instance
type ArgTimeCacher struct {
	DefaultSpan time.Duration
	CacheExpiry time.Duration
}

// timeCacher implements a time cacher with automatic sweeping mechanism
type timeCacher struct {
	timeCache   *timeCacheCore
	cacheExpiry time.Duration
	cancelFunc  func()
}

// NewTimeCacher creates a new timeCacher
func NewTimeCacher(arg ArgTimeCacher) (*timeCacher, error) {
	err := checkArg(arg)
	if err != nil {
		return nil, err
	}

	tc := &timeCacher{
		timeCache:   newTimeCacheCore(arg.DefaultSpan),
		cacheExpiry: arg.CacheExpiry,
	}

	var ctx context.Context
	ctx, tc.cancelFunc = context.WithCancel(context.Background())
	go tc.startSweeping(ctx)

	return tc, nil
}

func checkArg(arg ArgTimeCacher) error {
	if arg.DefaultSpan < minDuration {
		return storage.ErrInvalidDefaultSpan
	}
	if arg.CacheExpiry < minDuration {
		return storage.ErrInvalidCacheExpiry
	}

	return nil
}

// startSweeping handles sweeping the time cache
func (tc *timeCacher) startSweeping(ctx context.Context) {
	timer := time.NewTimer(tc.cacheExpiry)
	defer timer.Stop()

	for {
		timer.Reset(tc.cacheExpiry)

		select {
		case <-timer.C:
			tc.timeCache.sweep()
		case <-ctx.Done():
			log.Info("closing mapTimeCacher's sweep go routine...")
			return
		}
	}
}

// Clear deletes all stored data
func (tc *timeCacher) Clear() {
	tc.timeCache.clear()
}

// Put adds a value to the cache. It will always return false since the eviction did not occur
func (tc *timeCacher) Put(key []byte, value interface{}, _ int) (evicted bool) {
	_, err := tc.timeCache.put(string(key), value, tc.timeCache.defaultSpan)
	if err != nil {
		log.Error("mapTimeCacher.Put", "error", key)
	}

	return false
}

// Get returns a key's value from the cache
func (tc *timeCacher) Get(key []byte) (interface{}, bool) {
	tc.timeCache.RLock()
	defer tc.timeCache.RUnlock()

	v, ok := tc.timeCache.data[string(key)]
	if !ok {
		return nil, ok
	}

	return v.value, ok
}

// Has checks if a key is in the cache
func (tc *timeCacher) Has(key []byte) bool {
	return tc.timeCache.has(string(key))
}

// Peek returns a key's value from the cache
func (tc *timeCacher) Peek(key []byte) (value interface{}, ok bool) {
	return tc.Get(key)
}

// HasOrAdd checks if a key is in the cache.
// If key exists, does not update the value. Otherwise, adds the key-value in the cache
func (tc *timeCacher) HasOrAdd(key []byte, value interface{}, _ int) (has, added bool) {
	return tc.timeCache.hasOrAdd(string(key), value, tc.timeCache.defaultSpan)
}

// Remove removes the key from cache
func (tc *timeCacher) Remove(key []byte) {
	if key == nil {
		return
	}

	tc.timeCache.Lock()
	defer tc.timeCache.Unlock()

	delete(tc.timeCache.data, string(key))
}

// Keys returns all keys from cache
func (tc *timeCacher) Keys() [][]byte {
	tc.timeCache.RLock()
	defer tc.timeCache.RUnlock()

	keys := make([][]byte, len(tc.timeCache.data))
	idx := 0
	for k := range tc.timeCache.data {
		keys[idx] = []byte(k)
		idx++
	}

	return keys
}

// Len returns the size of the cache
func (tc *timeCacher) Len() int {
	return tc.timeCache.len()
}

// SizeInBytesContained will always return 0
func (tc *timeCacher) SizeInBytesContained() uint64 {
	return 0
}

// MaxSize returns the maximum number of items which can be stored in cache.
func (tc *timeCacher) MaxSize() int {
	return math.MaxInt32
}

// RegisterHandler registers a handler, currently not needed
func (tc *timeCacher) RegisterHandler(_ func(key []byte, value interface{}), _ string) {
}

// UnRegisterHandler unregisters a handler, currently not needed
func (tc *timeCacher) UnRegisterHandler(_ string) {
}

// Close will close the internal sweep go routine
func (tc *timeCacher) Close() error {
	if tc.cancelFunc != nil {
		tc.cancelFunc()
	}

	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (tc *timeCacher) IsInterfaceNil() bool {
	return tc == nil
}
