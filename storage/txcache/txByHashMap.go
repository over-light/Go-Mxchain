package txcache

import (
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
)

// txByHashMap is a new map-like structure for holding and accessing transactions by txHash
type txByHashMap struct {
	backingMap *ConcurrentMap
	counter    core.AtomicCounter
}

// newTxByHashMap creates a new TxByHashMap instance
func newTxByHashMap(nChunksHint uint32) txByHashMap {
	backingMap := NewConcurrentMap(nChunksHint)

	return txByHashMap{
		backingMap: backingMap,
		counter:    0,
	}
}

// addTx adds a transaction to the map
func (txMap *txByHashMap) addTx(txHash []byte, tx data.TransactionHandler) bool {
	added := txMap.backingMap.SetIfAbsent(string(txHash), tx)
	if added {
		txMap.counter.Increment()
	}

	return added
}

// removeTx removes a transaction from the map
func (txMap *txByHashMap) removeTx(txHash string) (data.TransactionHandler, bool) {
	tx, ok := txMap.getTx(txHash)
	if !ok {
		return nil, false
	}

	txMap.backingMap.Remove(txHash)
	txMap.counter.Decrement()
	return tx, true
}

// getTx gets a transaction from the map
func (txMap *txByHashMap) getTx(txHash string) (data.TransactionHandler, bool) {
	txUntyped, ok := txMap.backingMap.Get(txHash)
	if !ok {
		return nil, false
	}

	tx := txUntyped.(data.TransactionHandler)
	return tx, true
}

// RemoveTxsBulk removes transactions, in bulk
func (txMap *txByHashMap) RemoveTxsBulk(txHashes [][]byte) uint32 {
	for _, txHash := range txHashes {
		txMap.backingMap.Remove(string(txHash))
	}

	oldCount := uint32(txMap.counter.Get())
	newCount := uint32(txMap.backingMap.Count())
	nRemoved := oldCount - newCount

	txMap.counter.Set(int64(newCount))
	return nRemoved
}

// ForEachTransaction is an iterator callback
type ForEachTransaction func(txHash []byte, value data.TransactionHandler)

// forEach iterates over the senders
func (txMap *txByHashMap) forEach(function ForEachTransaction) {
	txMap.backingMap.IterCb(func(key string, item interface{}) {
		tx := item.(data.TransactionHandler)
		function([]byte(key), tx)
	})
}

func (txMap *txByHashMap) clear() {
	txMap.backingMap.Clear()
}
