package interceptors

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/storage"
)

type whiteListDataVerifier struct {
	cache storage.Cacher
}

// NewWhiteListDataVerifier returns a default data verifier
func NewWhiteListDataVerifier(cacher storage.Cacher) (*whiteListDataVerifier, error) {
	if check.IfNil(cacher) {
		return nil, process.ErrNilCacher
	}

	return &whiteListDataVerifier{cache: cacher}, nil
}

// IsForCurrentShard return true if intercepted data is for shard
func (w *whiteListDataVerifier) IsForCurrentShard(interceptedData process.InterceptedData) bool {
	if check.IfNil(interceptedData) {
		return false
	}

	return w.cache.Has(interceptedData.Hash())
}

func (w *whiteListDataVerifier) Add(keys [][]byte) {
	for _, key := range keys {
		_ = w.cache.Put(key, struct{}{})
	}
}

func (w *whiteListDataVerifier) Remove(keys [][]byte) {
	for _, key := range keys {
		w.cache.Remove(key)
	}
}

// IsInterfaceNil returns true if underlying object is nil
func (w *whiteListDataVerifier) IsInterfaceNil() bool {
	return w == nil
}
