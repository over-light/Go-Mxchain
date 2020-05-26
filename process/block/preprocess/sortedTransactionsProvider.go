package preprocess

import (
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/txcache"
)

// createSortedTransactionsProvider is a "simple factory" for "SortedTransactionsProvider" objects
func createSortedTransactionsProvider(transactionsPreprocessor *transactions, cache storage.Cacher, cacheKey string) SortedTransactionsProvider {
	txCache, isTxCache := cache.(TxCache)
	if isTxCache {
		return newAdapterTxCacheToSortedTransactionsProvider(txCache)
	}

	log.Error("Could not create a real [SortedTransactionsProvider], will create a disabled one")
	return &disabledSortedTransactionsProvider{}
}

// adapterTxCacheToSortedTransactionsProvider adapts a "TxCache" to the "SortedTransactionsProvider" interface
type adapterTxCacheToSortedTransactionsProvider struct {
	txCache TxCache
}

func newAdapterTxCacheToSortedTransactionsProvider(txCache TxCache) *adapterTxCacheToSortedTransactionsProvider {
	adapter := &adapterTxCacheToSortedTransactionsProvider{
		txCache: txCache,
	}

	return adapter
}

// GetSortedTransactions gets the transactions from the cache
func (adapter *adapterTxCacheToSortedTransactionsProvider) GetSortedTransactions() []*txcache.WrappedTransaction {
	txs := adapter.txCache.SelectTransactions(process.MaxNumOfTxsToSelect, process.NumTxPerSenderBatchForFillingMiniblock)
	return txs
}

// NotifyAccountNonce notifies the cache about the current nonce of an account
func (adapter *adapterTxCacheToSortedTransactionsProvider) NotifyAccountNonce(accountKey []byte, nonce uint64) {
	adapter.txCache.NotifyAccountNonce(accountKey, nonce)
}

// IsInterfaceNil returns true if there is no value under the interface
func (adapter *adapterTxCacheToSortedTransactionsProvider) IsInterfaceNil() bool {
	return adapter == nil
}

// disabledSortedTransactionsProvider is a disabled "SortedTransactionsProvider" (should never be used in practice)
type disabledSortedTransactionsProvider struct {
}

// GetSortedTransactions returns an empty slice
func (adapter *disabledSortedTransactionsProvider) GetSortedTransactions() []*txcache.WrappedTransaction {
	return make([]*txcache.WrappedTransaction, 0)
}

// NotifyAccountNonce does nothing
func (adapter *disabledSortedTransactionsProvider) NotifyAccountNonce(_ []byte, _ uint64) {
}

// IsInterfaceNil returns true if there is no value under the interface
func (adapter *disabledSortedTransactionsProvider) IsInterfaceNil() bool {
	return adapter == nil
}
