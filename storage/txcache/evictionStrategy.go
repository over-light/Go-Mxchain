package txcache

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/logger"
)

var log = logger.GetOrCreate("txcache/eviction")

// EvictionStrategyConfig is a cache eviction model
type EvictionStrategyConfig struct {
	CountThreshold                 int
	NoOldestSendersToEvict         int
	ALotOfTransactionsForASender   int
	NoTxsToEvictForASenderWithALot int
}

// EvictionStrategy is a cache eviction model
type EvictionStrategy struct {
	Cache  *TxCache
	Config EvictionStrategyConfig
	mutex  sync.Mutex
}

// NewEvictionStrategy creates a new EvictionModel
func NewEvictionStrategy(cache *TxCache, config EvictionStrategyConfig) *EvictionStrategy {
	model := &EvictionStrategy{
		Cache:  cache,
		Config: config,
	}

	return model
}

// DoEviction does cache eviction
func (model *EvictionStrategy) DoEviction(incomingTx *transaction.Transaction) {
	if !model.areThereTooManyTxs() {
		return
	}

	// We do not allow more evictions to start concurrently
	model.mutex.Lock()

	// First pass
	// Senders capacity is close to be reached first (before txs capacity) when there are a lot of senders with little or one transaction
	if model.areThereTooManySenders() {
		countTxs, countSenders := model.EvictOldestSenders()
		log.Debug("DoEviction, 1st pass:", "countTxs", countTxs, "countSenders", countSenders)
	}

	// Second pass
	// For senders with many transactions (> "ManyTransactionsForASender"), evict "PartOfManyTransactionsOfASender" transactions
	if model.areThereTooManyTxs() {
		countTxs, countSenders := model.EvictHighNonceTransactions()
		log.Debug("DoEviction, 2nd pass:", "countTxs", countTxs, "countSenders", countSenders)
	}

	// Third pass
	if model.areThereTooManyTxs() {
		steps, countTxs, countSenders := model.EvictSendersWhileTooManyTxs()
		log.Debug("DoEviction, 3rd pass:", "steps", steps, "countTxs", countTxs, "countSenders", countSenders)
	}

	model.mutex.Unlock()
}

func (model *EvictionStrategy) areThereTooManySenders() bool {
	tooManySenders := model.Cache.txListBySender.Counter.Get() > int64(model.Config.CountThreshold)
	return tooManySenders
}

func (model *EvictionStrategy) areThereTooManyTxs() bool {
	tooManyTxs := model.Cache.txByHash.Counter.Get() > int64(model.Config.CountThreshold)
	return tooManyTxs
}

// EvictOldestSenders removes transactions from the cache
func (model *EvictionStrategy) EvictOldestSenders() (int, int) {
	listsOrdered := model.Cache.txListBySender.GetListsSortedByOrderNumber()
	sliceEnd := core.MinInt(model.Config.NoOldestSendersToEvict, len(listsOrdered))
	listsToEvict := listsOrdered[:sliceEnd]

	return model.evictSendersAndTheirTxs(listsToEvict)
}

func (model *EvictionStrategy) evictSendersAndTheirTxs(listsToEvict []*TxListForSender) (int, int) {
	sendersToEvict := make([]string, 0)
	txsToEvict := make([][]byte, 0)

	for _, txList := range listsToEvict {
		sendersToEvict = append(sendersToEvict, txList.sender)
		txsToEvict = append(txsToEvict, txList.GetTxHashes()...)
	}

	return model.doEvictItems(txsToEvict, sendersToEvict)
}

func (model *EvictionStrategy) doEvictItems(txsToEvict [][]byte, sendersToEvict []string) (int, int) {
	model.Cache.txByHash.RemoveTransactionsBulk(txsToEvict)
	model.Cache.txListBySender.removeSenders(sendersToEvict)

	return len(txsToEvict), len(sendersToEvict)
}

// EvictHighNonceTransactions removes transactions from the cache
func (model *EvictionStrategy) EvictHighNonceTransactions() (int, int) {
	txsToEvict := make([][]byte, 0)
	sendersToEvict := make([]string, 0)

	model.Cache.ForEachSender(func(key string, txList *TxListForSender) {
		aLot := model.Config.ALotOfTransactionsForASender
		toEvictForSenderCount := model.Config.NoTxsToEvictForASenderWithALot

		if txList.HasMoreThan(aLot) {
			txsToEvictForSender := txList.RemoveHighNonceTxs(toEvictForSenderCount)
			txsToEvict = append(txsToEvict, txsToEvictForSender...)
		}

		if txList.IsEmpty() {
			sendersToEvict = append(sendersToEvict, key)
		}
	})

	return model.doEvictItems(txsToEvict, sendersToEvict)
}

// EvictSendersWhileTooManyTxs removes transactions
func (model *EvictionStrategy) EvictSendersWhileTooManyTxs() (step int, countTxs int, countSenders int) {
	listsOrdered := model.Cache.txListBySender.GetListsSortedByOrderNumber()

	sliceStart := 0
	for step = 1; model.areThereTooManyTxs(); step++ {
		batchSize := model.Config.NoOldestSendersToEvict
		sliceEnd := core.MinInt(sliceStart+batchSize, len(listsOrdered))
		listsToEvict := listsOrdered[sliceStart:sliceEnd]

		stepCountTxs, stepCountSenders := model.evictSendersAndTheirTxs(listsToEvict)

		countTxs += stepCountTxs
		countSenders += stepCountSenders
		sliceStart += batchSize
	}

	return
}
