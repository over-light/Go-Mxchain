package preprocess

import (
	"sort"

	"github.com/ElrondNetwork/elrond-go/core/sliceUtil"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/txcache"
)

// getSortedTransactionsProvider gets a sorted transactions provider given a generic cache
func getSortedTransactionsProvider(transactionsPreprocessor *transactions, cache storage.Cacher, cacheKey string) SortedTransactionsProvider {
	txCache, isTxCache := cache.(*txcache.TxCache)
	if isTxCache {
		return newTxCacheToSortedTransactionsProviderAdapter(txCache)
	}

	return newCacheToSortedTransactionsProviderAdapter(transactionsPreprocessor, cache, cacheKey)
}

type txCacheToSortedTransactionsProviderAdapter struct {
	txCache *txcache.TxCache
}

func newTxCacheToSortedTransactionsProviderAdapter(txCache *txcache.TxCache) *txCacheToSortedTransactionsProviderAdapter {
	adapter := &txCacheToSortedTransactionsProviderAdapter{
		txCache: txCache,
	}

	return adapter
}

// GetSortedTransactions gets the transactions from the cache
func (adapter *txCacheToSortedTransactionsProviderAdapter) GetSortedTransactions() ([]data.TransactionHandler, [][]byte) {
	txs, txHashes := adapter.txCache.GetTransactions(process.MaxItemsInBlock, process.NumTxPerSenderBatchForFillingMiniblock)
	return txs, txHashes
}

// IsInterfaceNil returns true if there is no value under the interface
func (adapter *txCacheToSortedTransactionsProviderAdapter) IsInterfaceNil() bool {
	return adapter == nil
}

type cacheToSortedTransactionsProviderAdapter struct {
	transactionsPreprocessor *transactions
	cache                    storage.Cacher
	cacheKey                 string
}

func newCacheToSortedTransactionsProviderAdapter(transactionsPreprocessor *transactions, cache storage.Cacher, cacheKey string) *cacheToSortedTransactionsProviderAdapter {
	adapter := &cacheToSortedTransactionsProviderAdapter{
		transactionsPreprocessor: transactionsPreprocessor,
		cache:                    cache,
		cacheKey:                 cacheKey,
	}

	return adapter
}

// GetSortedTransactions gets the transactions from the cache
func (adapter *cacheToSortedTransactionsProviderAdapter) GetSortedTransactions() ([]data.TransactionHandler, [][]byte) {
	txs, txHashes := adapter.getOrderedTx()
	return txs, txHashes
}

// getOrderedTx was moved here from the previous implementation
func (adapter *cacheToSortedTransactionsProviderAdapter) getOrderedTx() ([]data.TransactionHandler, [][]byte) {
	txs := adapter.transactionsPreprocessor
	strCache := adapter.cacheKey

	txs.mutOrderedTxs.RLock()
	orderedTxs := txs.orderedTxs[strCache]
	orderedTxHashes := txs.orderedTxHashes[strCache]
	txs.mutOrderedTxs.RUnlock()

	alreadyOrdered := len(orderedTxs) > 0
	if !alreadyOrdered {
		orderedTxs, orderedTxHashes = sortTxByNonce(adapter.cache)

		log.Debug("creating mini blocks has been started",
			"have num txs", len(orderedTxs),
			"strCache", strCache,
		)

		txs.mutOrderedTxs.Lock()
		txs.orderedTxs[strCache] = orderedTxs
		txs.orderedTxHashes[strCache] = orderedTxHashes
		txs.mutOrderedTxs.Unlock()
	}

	return orderedTxs, orderedTxHashes
}

// sortTxByNonce was moved here from the previous implementation
func sortTxByNonce(cache storage.Cacher) ([]data.TransactionHandler, [][]byte) {
	txShardPool := cache

	keys := txShardPool.Keys()
	transactions := make([]data.TransactionHandler, 0, len(keys))
	txHashes := make([][]byte, 0, len(keys))

	mTxHashes := make(map[uint64][][]byte, len(keys))
	mTransactions := make(map[uint64][]data.TransactionHandler, len(keys))

	nonces := make([]uint64, 0, len(keys))

	for _, key := range keys {
		val, _ := txShardPool.Peek(key)
		if val == nil {
			continue
		}

		tx, ok := val.(data.TransactionHandler)
		if !ok {
			continue
		}

		nonce := tx.GetNonce()
		if mTxHashes[nonce] == nil {
			nonces = append(nonces, nonce)
			mTxHashes[nonce] = make([][]byte, 0)
			mTransactions[nonce] = make([]data.TransactionHandler, 0)
		}

		mTxHashes[nonce] = append(mTxHashes[nonce], key)
		mTransactions[nonce] = append(mTransactions[nonce], tx)
	}

	sort.Slice(nonces, func(i, j int) bool {
		return nonces[i] < nonces[j]
	})

	for _, nonce := range nonces {
		keys := mTxHashes[nonce]

		for idx, key := range keys {
			txHashes = append(txHashes, key)
			transactions = append(transactions, mTransactions[nonce][idx])
		}
	}

	return transaction.TrimSliceHandler(transactions), sliceUtil.TrimSliceSliceByte(txHashes)
}

// IsInterfaceNil returns true if there is no value under the interface
func (adapter *cacheToSortedTransactionsProviderAdapter) IsInterfaceNil() bool {
	return adapter == nil
}
