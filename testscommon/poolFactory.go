package testscommon

import (
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/dataPool"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/dataPool/headersCache"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/shardedData"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/txpool"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
)

// CreateTxPool -
func CreateTxPool(numShards uint32, selfShard uint32) (dataRetriever.ShardedDataCacherNotifier, error) {
	return txpool.NewShardedTxPool(
		txpool.ArgShardedTxPool{
			Config: storageUnit.CacheConfig{
				Capacity:             100_000,
				SizePerSender:        1_000_000_000,
				SizeInBytes:          1_000_000_000,
				SizeInBytesPerSender: 33_554_432,
				Shards:               16,
			},
			MinGasPrice:    200000000000,
			NumberOfShards: numShards,
			SelfShardID:    selfShard,
		},
	)
}

// CreatePoolsHolder -
func CreatePoolsHolder(numShards uint32, selfShard uint32) dataRetriever.PoolsHolder {
	var err error

	txPool, err := CreateTxPool(numShards, selfShard)

	unsignedTxPool, err := shardedData.NewShardedData(storageUnit.CacheConfig{
		Capacity:    100000,
		SizeInBytes: 1000000000,
		Shards:      1,
	})
	panicIfError("CreatePoolsHolderWithTxPool", err)

	rewardsTxPool, err := shardedData.NewShardedData(storageUnit.CacheConfig{
		Capacity:    300,
		SizeInBytes: 300000,
		Shards:      1,
	})
	panicIfError("CreatePoolsHolderWithTxPool", err)

	headersPool, err := headersCache.NewHeadersPool(config.HeadersPoolConfig{
		MaxHeadersPerShard:            1000,
		NumElementsToRemoveOnEviction: 100,
	})
	panicIfError("CreatePoolsHolderWithTxPool", err)

	cacherConfig := storageUnit.CacheConfig{Capacity: 100000, Type: storageUnit.LRUCache, Shards: 1}
	txBlockBody, err := storageUnit.NewCache(cacherConfig.Type, cacherConfig.Capacity, cacherConfig.Shards, cacherConfig.SizeInBytes)
	panicIfError("CreatePoolsHolderWithTxPool", err)

	cacherConfig = storageUnit.CacheConfig{Capacity: 100000, Type: storageUnit.LRUCache, Shards: 1}
	peerChangeBlockBody, err := storageUnit.NewCache(cacherConfig.Type, cacherConfig.Capacity, cacherConfig.Shards, cacherConfig.SizeInBytes)
	panicIfError("CreatePoolsHolderWithTxPool", err)

	cacherConfig = storageUnit.CacheConfig{Capacity: 50000, Type: storageUnit.LRUCache}
	trieNodes, err := storageUnit.NewCache(cacherConfig.Type, cacherConfig.Capacity, cacherConfig.Shards, cacherConfig.SizeInBytes)
	panicIfError("CreatePoolsHolderWithTxPool", err)

	currentTx, err := dataPool.NewCurrentBlockPool()
	panicIfError("CreatePoolsHolderWithTxPool", err)

	holder, err := dataPool.NewDataPool(
		txPool,
		unsignedTxPool,
		rewardsTxPool,
		headersPool,
		txBlockBody,
		peerChangeBlockBody,
		trieNodes,
		currentTx,
	)
	panicIfError("CreatePoolsHolderWithTxPool", err)

	return holder
}

// CreatePoolsHolderWithTxPool -
func CreatePoolsHolderWithTxPool(txPool dataRetriever.ShardedDataCacherNotifier) dataRetriever.PoolsHolder {
	var err error

	unsignedTxPool, err := shardedData.NewShardedData(storageUnit.CacheConfig{
		Capacity:    100000,
		SizeInBytes: 1000000000,
		Shards:      1,
	})
	panicIfError("CreatePoolsHolderWithTxPool", err)

	rewardsTxPool, err := shardedData.NewShardedData(storageUnit.CacheConfig{
		Capacity:    300,
		SizeInBytes: 300000,
		Shards:      1,
	})
	panicIfError("CreatePoolsHolderWithTxPool", err)

	headersPool, err := headersCache.NewHeadersPool(config.HeadersPoolConfig{
		MaxHeadersPerShard:            1000,
		NumElementsToRemoveOnEviction: 100,
	})
	panicIfError("CreatePoolsHolderWithTxPool", err)

	cacherConfig := storageUnit.CacheConfig{Capacity: 100000, Type: storageUnit.LRUCache, Shards: 1}
	txBlockBody, err := storageUnit.NewCache(cacherConfig.Type, cacherConfig.Capacity, cacherConfig.Shards, cacherConfig.SizeInBytes)
	panicIfError("CreatePoolsHolderWithTxPool", err)

	cacherConfig = storageUnit.CacheConfig{Capacity: 100000, Type: storageUnit.LRUCache, Shards: 1}
	peerChangeBlockBody, err := storageUnit.NewCache(cacherConfig.Type, cacherConfig.Capacity, cacherConfig.Shards, cacherConfig.SizeInBytes)
	panicIfError("CreatePoolsHolderWithTxPool", err)

	cacherConfig = storageUnit.CacheConfig{Capacity: 50000, Type: storageUnit.LRUCache}
	trieNodes, err := storageUnit.NewCache(cacherConfig.Type, cacherConfig.Capacity, cacherConfig.Shards, cacherConfig.SizeInBytes)
	panicIfError("CreatePoolsHolderWithTxPool", err)

	currentTx, err := dataPool.NewCurrentBlockPool()
	panicIfError("CreatePoolsHolderWithTxPool", err)

	holder, err := dataPool.NewDataPool(
		txPool,
		unsignedTxPool,
		rewardsTxPool,
		headersPool,
		txBlockBody,
		peerChangeBlockBody,
		trieNodes,
		currentTx,
	)
	panicIfError("CreatePoolsHolderWithTxPool", err)

	return holder
}
