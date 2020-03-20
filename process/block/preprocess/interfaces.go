package preprocess

import (
	"github.com/ElrondNetwork/elrond-go/storage/txcache"
)

// SortedTransactionsProvider defines the public API of the transactions cache
type SortedTransactionsProvider interface {
	GetSortedTransactions() []*txcache.WrappedTransaction
	NotifyAccountNonce(accountKey []byte, nonce uint64)
	IsInterfaceNil() bool
}

// BlockTracker defines the functionality for node to track the blocks which are received from network
type BlockTracker interface {
	IsShardStuck(shardID uint32) bool
	IsInterfaceNil() bool
}

// BlockSizeComputationHandler defines the functionality for block size computation
type BlockSizeComputationHandler interface {
	Init()
	AddNumMiniBlocks(numMiniBlocks int)
	AddNumTxs(numTxs int)
	IsMaxBlockSizeReached(numNewMiniBlocks int, numNewTxs int) bool
	IsInterfaceNil() bool
}

// BlockSizeThrottler defines the functionality of adapting the node to the network speed/latency when it should send a
// block to its peers which should be received in a limited time frame
type BlockSizeThrottler interface {
	GetMaxSize() uint32
	IsInterfaceNil() bool
}
