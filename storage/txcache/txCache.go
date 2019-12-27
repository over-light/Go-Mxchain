package txcache

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/logger"
)

var log = logger.GetOrCreate("txcache")

// TxCache represents a cache-like structure (it has a fixed capacity and implements an eviction mechanism) for holding transactions
type TxCache struct {
	txListBySender txListBySenderMap
	txByHash       txByHashMap
	evictionConfig EvictionConfig
	evictionMutex  sync.Mutex
}

// NewTxCache creates a new transaction cache
// "nChunksHint" is used to configure the internal concurrent maps on which the implementation relies
func NewTxCache(nChunksHint uint32) *TxCache {
	// Note: for simplicity, we use the same "nChunksHint" for both internal concurrent maps
	txCache := &TxCache{
		txListBySender: newTxListBySenderMap(nChunksHint),
		txByHash:       newTxByHashMap(nChunksHint),
		evictionConfig: EvictionConfig{Enabled: false},
	}

	return txCache
}

// NewTxCacheWithEviction creates a new transaction cache with eviction
func NewTxCacheWithEviction(nChunksHint uint32, evictionConfig EvictionConfig) *TxCache {
	txCache := NewTxCache(nChunksHint)
	txCache.evictionConfig = evictionConfig

	return txCache
}

// AddTx adds a transaction in the cache
// Eviction happens if maximum capacity is reached
func (cache *TxCache) AddTx(txHash []byte, tx data.TransactionHandler) (ok bool, added bool) {
	ok = false
	added = false

	if check.IfNil(tx) {
		return
	}

	if cache.evictionConfig.Enabled {
		cache.doEviction()
	}

	ok = true
	added = cache.txByHash.addTx(txHash, tx)
	if added {
		cache.txListBySender.addTx(txHash, tx)
	}

	return
}

// GetByTxHash gets the transaction by hash
func (cache *TxCache) GetByTxHash(txHash []byte) (data.TransactionHandler, bool) {
	tx, ok := cache.txByHash.getTx(string(txHash))
	return tx, ok
}

// GetTransactions gets a reasonably fair list of transactions to be included in the next miniblock
// It returns at most "numRequested" transactions
// Each sender gets the chance to give at least "batchSizePerSender" transactions, unless "numRequested" limit is reached before iterating over all senders
func (cache *TxCache) GetTransactions(numRequested int, batchSizePerSender int) ([]data.TransactionHandler, [][]byte) {
	result := make([]data.TransactionHandler, numRequested)
	resultHashes := make([][]byte, numRequested)
	resultFillIndex := 0
	resultIsFull := false

	for pass := 0; !resultIsFull; pass++ {
		copiedInThisPass := 0

		cache.forEachSender(func(key string, txList *txListForSender) {
			// Reset happens on first pass only
			shouldResetCopy := pass == 0
			copied := txList.copyBatchTo(shouldResetCopy, result[resultFillIndex:], resultHashes[resultFillIndex:], batchSizePerSender)

			resultFillIndex += copied
			copiedInThisPass += copied
			resultIsFull = resultFillIndex == numRequested
		})

		nothingCopiedThisPass := copiedInThisPass == 0

		// No more passes needed
		if nothingCopiedThisPass {
			break
		}
	}

	return result[:resultFillIndex], resultHashes
}

// RemoveTxByHash removes
func (cache *TxCache) RemoveTxByHash(txHash []byte) error {
	tx, ok := cache.txByHash.removeTx(string(txHash))
	if !ok {
		return ErrTxNotFound
	}

	found := cache.txListBySender.removeTx(tx)
	if !found {
		// This should never happen (eviction should never cause this kind of inconsistency between the two internal maps)
		log.Error("RemoveTxByHash detected maps sync inconsistency", "tx", txHash)
		return ErrMapsSyncInconsistency
	}

	return nil
}

// CountTx gets the number of transactions in the cache
func (cache *TxCache) CountTx() int64 {
	return cache.txByHash.counter.Get()
}

// Len is an alias for CountTx
func (cache *TxCache) Len() int {
	return int(cache.CountTx())
}

// CountSenders gets the number of senders in the cache
func (cache *TxCache) CountSenders() int64 {
	return cache.txListBySender.counter.Get()
}

// forEachSender iterates over the senders in the cache
func (cache *TxCache) forEachSender(function ForEachSender) {
	cache.txListBySender.forEach(function)
}

// ForEachTransaction iterates over the transactions in the cache
func (cache *TxCache) ForEachTransaction(function ForEachTransaction) {
	cache.txByHash.forEach(function)
}

// Clear clears the cache
func (cache *TxCache) Clear() {
	cache.txListBySender.clear()
	cache.txByHash.clear()
}

// Put is not implemented
func (cache *TxCache) Put(key []byte, value interface{}) (evicted bool) {
	return false
}

// Get gets a transaction by hash
func (cache *TxCache) Get(key []byte) (value interface{}, ok bool) {
	tx, ok := cache.GetByTxHash(key)
	return tx, ok
}

// Has is not implemented
func (cache *TxCache) Has(key []byte) bool {
	log.Error("TxCache.Has is not implemented")
	return false
}

// Peek gets a transaction by hash
func (cache *TxCache) Peek(key []byte) (value interface{}, ok bool) {
	tx, ok := cache.GetByTxHash(key)
	return tx, ok
}

// HasOrAdd is not implemented
func (cache *TxCache) HasOrAdd(key []byte, value interface{}) (ok, evicted bool) {
	log.Error("TxCache.HasOrAdd is not implemented")
	return false, false
}

// Remove is not implemented
func (cache *TxCache) Remove(key []byte) {
	log.Error("TxCache.Remove is not implemented")
}

// RemoveOldest is not implemented
func (cache *TxCache) RemoveOldest() {
	log.Error("TxCache.RemoveOldest is not implemented")
}

// Keys is not implemented
func (cache *TxCache) Keys() [][]byte {
	log.Error("TxCache.Keys is not implemented")
	return [][]byte{}
}

// MaxSize is not implemented
func (cache *TxCache) MaxSize() int {
	log.Error("TxCache.MaxSize is not implemented")
	return 0
}

// RegisterHandler is not implemented
func (cache *TxCache) RegisterHandler(func(key []byte)) {
	log.Error("TxCache.RegisterHandler is not implemented")
}

// IsInterfaceNil returns true if there is no value under the interface
func (cache *TxCache) IsInterfaceNil() bool {
	return cache == nil
}
