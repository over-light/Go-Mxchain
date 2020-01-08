package txcache

import (
	"sort"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
)

// txListBySenderMap is a map-like structure for holding and accessing transactions by sender
type txListBySenderMap struct {
	backingMap      *ConcurrentMap
	counter         core.AtomicCounter
	nextOrderNumber core.AtomicCounter
}

// newTxListBySenderMap creates a new instance of TxListBySenderMap
func newTxListBySenderMap(nChunksHint uint32) txListBySenderMap {
	backingMap := NewConcurrentMap(nChunksHint)

	return txListBySenderMap{
		backingMap: backingMap,
		counter:    0,
	}
}

// addTx adds a transaction in the map, in the corresponding list (selected by its sender)
func (txMap *txListBySenderMap) addTx(txHash []byte, tx data.TransactionHandler) {
	sender := string(tx.GetSndAddress())
	listForSender := txMap.getOrAddListForSender(sender)
	listForSender.AddTx(txHash, tx)
}

func (txMap *txListBySenderMap) getOrAddListForSender(sender string) *txListForSender {
	listForSender, ok := txMap.getListForSender(sender)
	if !ok {
		listForSender = txMap.addSender(sender)
	}

	return listForSender
}

func (txMap *txListBySenderMap) getListForSender(sender string) (*txListForSender, bool) {
	listForSenderUntyped, ok := txMap.backingMap.Get(sender)
	if !ok {
		return nil, false
	}

	listForSender := listForSenderUntyped.(*txListForSender)
	return listForSender, true
}

func (txMap *txListBySenderMap) addSender(sender string) *txListForSender {
	orderNumber := txMap.nextOrderNumber.Get()
	listForSender := newTxListForSender(sender, orderNumber)

	txMap.backingMap.Set(sender, listForSender)
	txMap.counter.Increment()
	txMap.nextOrderNumber.Increment()

	return listForSender
}

// removeTx removes a transaction from the map
func (txMap *txListBySenderMap) removeTx(tx data.TransactionHandler) bool {
	sender := string(tx.GetSndAddress())

	listForSender, ok := txMap.getListForSender(sender)
	if !ok {
		log.Error("txListBySenderMap.removeTx() detected inconsistency: sender of tx not in cache", "sender", sender)
		return false
	}

	isFound := listForSender.RemoveTx(tx)

	if listForSender.IsEmpty() {
		txMap.removeSender(sender)
	}

	return isFound
}

func (txMap *txListBySenderMap) removeSender(sender string) {
	if !txMap.backingMap.Has(sender) {
		return
	}

	txMap.backingMap.Remove(sender)
	txMap.counter.Decrement()
}

// RemoveSendersBulk removes senders, in bulk
func (txMap *txListBySenderMap) RemoveSendersBulk(senders []string) uint32 {
	oldCount := uint32(txMap.counter.Get())

	for _, senderKey := range senders {
		txMap.removeSender(senderKey)
	}

	newCount := uint32(txMap.counter.Get())
	nRemoved := oldCount - newCount
	return nRemoved
}

type txListBySenderSortKind string

// LRUCache is currently the only supported Cache type
const (
	SortByOrderNumberAsc txListBySenderSortKind = "SortByOrderNumberAsc"
	SortByTotalBytesDesc txListBySenderSortKind = "SortByTotalBytesDesc"
	SortByTotalGas       txListBySenderSortKind = "SortByTotalGas"
	SortBySmartScore     txListBySenderSortKind = "SortBySmartScore"
)

func (txMap *txListBySenderMap) GetListsSortedBy(sortKind txListBySenderSortKind) []*txListForSender {
	// TODO-TXCACHE: do partial sort? optimization.

	switch sortKind {
	case SortByOrderNumberAsc:
		return txMap.GetListsSortedByOrderNumber()
	case SortByTotalBytesDesc:
		return txMap.GetListsSortedByTotalBytes()
	case SortByTotalGas:
		return txMap.GetListsSortedByTotalGas()
	case SortBySmartScore:
		return txMap.GetListsSortedBySmartScore()
	default:
		return txMap.GetListsSortedByOrderNumber()
	}
}

// GetListsSortedByOrderNumber gets the list of sender addreses, sorted by the global order number, ascending
func (txMap *txListBySenderMap) GetListsSortedByOrderNumber() []*txListForSender {
	snapshot := txMap.getListsSnapshot()

	sort.Slice(snapshot, func(i, j int) bool {
		return snapshot[i].orderNumber < snapshot[j].orderNumber
	})

	return snapshot
}

// GetListsSortedByTotalBytes gets the list of sender addreses, sorted by the total amount of bytes, descending
func (txMap *txListBySenderMap) GetListsSortedByTotalBytes() []*txListForSender {
	snapshot := txMap.getListsSnapshot()

	sort.Slice(snapshot, func(i, j int) bool {
		return snapshot[i].totalBytes > snapshot[j].totalBytes
	})

	return snapshot
}

// GetListsSortedByTotalGas gets the list of sender addreses, sorted by the total amoung of gas, ascending
func (txMap *txListBySenderMap) GetListsSortedByTotalGas() []*txListForSender {
	snapshot := txMap.getListsSnapshot()

	sort.Slice(snapshot, func(i, j int) bool {
		return snapshot[i].totalGas < snapshot[j].totalGas
	})

	return snapshot
}

// GetListsSortedBySmartScore gets the list of sender addreses, sorted by a smart score
func (txMap *txListBySenderMap) GetListsSortedBySmartScore() []*txListForSender {
	snapshot := txMap.getListsSnapshot()
	computer := newEvictionScoreComputer(snapshot)

	// Scores are quantized
	// This way, sort is also a bit more optimized (less item movement)
	// And partial, approximate sort is sufficient
	sort.Slice(snapshot, func(i, j int) bool {
		return computer.quantizedScores[i] < computer.quantizedScores[j]
	})

	return snapshot
}

func (txMap *txListBySenderMap) getListsSnapshot() []*txListForSender {
	counter := txMap.counter.Get()
	if counter < 1 {
		return make([]*txListForSender, 0)
	}

	snapshot := make([]*txListForSender, 0, counter)

	txMap.forEach(func(key string, item *txListForSender) {
		snapshot = append(snapshot, item)
	})

	return snapshot
}

// func (txMap *txListBySenderMap) getListsSortedByFunc(less func(txListA, txListB *txListForSender) bool) []*txListForSender {

// 	sort.Slice(lists, func(i, j int) bool {
// 		return less(lists[i], lists[j])
// 	})

// 	return lists
// }

// ForEachSender is an iterator callback
type ForEachSender func(key string, value *txListForSender)

// forEach iterates over the senders
func (txMap *txListBySenderMap) forEach(function ForEachSender) {
	txMap.backingMap.IterCb(func(key string, item interface{}) {
		txList := item.(*txListForSender)
		function(key, txList)
	})
}

func (txMap *txListBySenderMap) clear() {
	txMap.backingMap.Clear()
	txMap.counter.Set(0)
}

func (txMap *txListBySenderMap) countTxBySender(sender string) int64 {
	listForSender, ok := txMap.getListForSender(sender)
	if !ok {
		return 0
	}

	return listForSender.countTx()
}
