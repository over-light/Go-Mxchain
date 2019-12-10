package txcache

import (
	"github.com/ElrondNetwork/elrond-go/data/transaction"
)

// TxCache is
type TxCache struct {
	txListBySender   TxListBySenderMap
	txByHash         TxByHashMap
	EvictionStrategy *EvictionStrategy
}

// NewTxCache creates a new transaction cache
// "size" dictates the maximum number of transactions to hold in this cache at a given time
// "noChunksHint" is used to configure the internal concurrent maps on which the implementation relies
func NewTxCache(size uint32, noChunksHint uint32) *TxCache {
	// Note: for simplicity, we use the same "noChunksHint" for both internal concurrent maps
	txCache := &TxCache{
		txListBySender: NewTxListBySenderMap(size, noChunksHint),
		txByHash:       NewTxByHashMap(size, noChunksHint),
	}

	return txCache
}

// AddTx adds a transaction in the cache
// Eviction happens if maximum capacity is reached
func (cache *TxCache) AddTx(txHash []byte, tx *transaction.Transaction) {
	if cache.EvictionStrategy != nil {
		cache.EvictionStrategy.DoEviction(tx)
	}

	cache.txByHash.AddTx(txHash, tx)
	cache.txListBySender.AddTx(txHash, tx)
}

// GetByTxHash gets the transaction by hash
func (cache *TxCache) GetByTxHash(txHash []byte) (*transaction.Transaction, bool) {
	tx, ok := cache.txByHash.GetTx(string(txHash))
	return tx, ok
}

// GetTransactions gets a reasonably fair list of transactions to be included in the next miniblock
// It returns at most "noRequested" transactions
// Each sender gets the chance to give at least "batchSizePerSender" transactions, unless "noRequested" limit is reached before iterating over all senders
func (cache *TxCache) GetTransactions(noRequested int, batchSizePerSender int) []*transaction.Transaction {
	result := make([]*transaction.Transaction, noRequested)
	resultFillIndex := 0
	resultIsFull := false

	for pass := 0; !resultIsFull; pass++ {
		copiedInThisPass := 0

		cache.ForEachSender(func(key string, txList *TxListForSender) {
			// Do this on first pass only
			if pass == 0 {
				txList.StartBatchCopying(batchSizePerSender)
			}

			copied := txList.CopyBatchTo(result[resultFillIndex:])

			resultFillIndex += copied
			copiedInThisPass += copied
			resultIsFull = resultFillIndex == noRequested
		})

		nothingCopiedThisPass := copiedInThisPass == 0

		// No more passes needed
		if nothingCopiedThisPass {
			break
		}
	}

	return result[:resultFillIndex]
}

// RemoveTxByHash removes
func (cache *TxCache) RemoveTxByHash(txHash []byte) {
	tx, ok := cache.txByHash.RemoveTx(string(txHash))
	if ok {
		cache.txListBySender.RemoveTx(tx)
	}
}

// CountTx gets the number of transactions in the cache
func (cache *TxCache) CountTx() int64 {
	return cache.txByHash.Counter.Get()
}

// ForEachSender iterates over the senders
func (cache *TxCache) ForEachSender(function ForEachSender) {
	cache.txListBySender.ForEach(function)
}
