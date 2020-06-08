package mock

import (
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/dataPool"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/dataPool/headersCache"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/shardedData"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/txpool"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
)

// PoolsHolderMock -
type PoolsHolderMock struct {
	transactions         dataRetriever.ShardedDataCacherNotifier
	unsignedTransactions dataRetriever.ShardedDataCacherNotifier
	rewardTransactions   dataRetriever.ShardedDataCacherNotifier
	headers              dataRetriever.HeadersPool
	miniBlocks           storage.Cacher
	peerChangesBlocks    storage.Cacher
	trieNodes            storage.Cacher
	currBlockTxs         dataRetriever.TransactionCacher
}

// NewPoolsHolderMock -
func NewPoolsHolderMock() *PoolsHolderMock {
	phf := &PoolsHolderMock{}

	phf.transactions, _ = txpool.NewShardedTxPool(
		txpool.ArgShardedTxPool{
			Config: storageUnit.CacheConfig{
				Capacity:             100000,
				SizePerSender:        1000,
				SizeInBytes:          1000000000,
				SizeInBytesPerSender: 10000000,
				Shards:               16,
			},
			MinGasPrice:    200000000000,
			NumberOfShards: 1,
		},
	)

	phf.unsignedTransactions, _ = shardedData.NewShardedData(storageUnit.CacheConfig{Capacity: 10000, Type: storageUnit.FIFOShardedWithImmunityCache})
	phf.rewardTransactions, _ = shardedData.NewShardedData(storageUnit.CacheConfig{Capacity: 100, Type: storageUnit.FIFOShardedWithImmunityCache})
	phf.headers, _ = headersCache.NewHeadersPool(config.HeadersPoolConfig{MaxHeadersPerShard: 1000, NumElementsToRemoveOnEviction: 100})
	phf.miniBlocks, _ = storageUnit.NewCache(storageUnit.LRUCache, 10000, 1, 0)
	phf.peerChangesBlocks, _ = storageUnit.NewCache(storageUnit.LRUCache, 10000, 1, 0)
	phf.currBlockTxs, _ = dataPool.NewCurrentBlockPool()
	phf.trieNodes, _ = storageUnit.NewCache(storageUnit.LRUCache, 10000, 1, 0)

	return phf
}

// CurrentBlockTxs -
func (phm *PoolsHolderMock) CurrentBlockTxs() dataRetriever.TransactionCacher {
	return phm.currBlockTxs
}

// Transactions -
func (phm *PoolsHolderMock) Transactions() dataRetriever.ShardedDataCacherNotifier {
	return phm.transactions
}

// UnsignedTransactions -
func (phm *PoolsHolderMock) UnsignedTransactions() dataRetriever.ShardedDataCacherNotifier {
	return phm.unsignedTransactions
}

// RewardTransactions -
func (phm *PoolsHolderMock) RewardTransactions() dataRetriever.ShardedDataCacherNotifier {
	return phm.rewardTransactions
}

// Headers -
func (phm *PoolsHolderMock) Headers() dataRetriever.HeadersPool {
	return phm.headers
}

// MiniBlocks -
func (phm *PoolsHolderMock) MiniBlocks() storage.Cacher {
	return phm.miniBlocks
}

// PeerChangesBlocks -
func (phm *PoolsHolderMock) PeerChangesBlocks() storage.Cacher {
	return phm.peerChangesBlocks
}

// SetTransactions -
func (phm *PoolsHolderMock) SetTransactions(transactions dataRetriever.ShardedDataCacherNotifier) {
	phm.transactions = transactions
}

// SetUnsignedTransactions -
func (phm *PoolsHolderMock) SetUnsignedTransactions(scrs dataRetriever.ShardedDataCacherNotifier) {
	phm.unsignedTransactions = scrs
}

// TrieNodes -
func (phm *PoolsHolderMock) TrieNodes() storage.Cacher {
	return phm.trieNodes
}

// IsInterfaceNil returns true if there is no value under the interface
func (phm *PoolsHolderMock) IsInterfaceNil() bool {
	return phm == nil
}
