package bootstrap

import (
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
)

func getGeneralConfig() config.Config {
	return config.Config{
		EpochStartConfig: config.EpochStartConfig{
			MinRoundsBetweenEpochs: 5,
			RoundsPerEpoch:         10,
		},
		WhiteListPool: config.CacheConfig{
			Size:   10000,
			Type:   "LRU",
			Shards: 1,
		},
		StoragePruning: config.StoragePruningConfig{
			Enabled:             false,
			FullArchive:         true,
			NumEpochsToKeep:     3,
			NumActivePersisters: 3,
		},
		EvictionWaitingList: config.EvictionWaitingListConfig{
			Size: 100,
			DB: config.DBConfig{
				FilePath:          "EvictionWaitingList",
				Type:              "MemoryDB",
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		TrieSnapshotDB: config.DBConfig{
			FilePath:          "TrieSnapshot",
			Type:              "MemoryDB",
			BatchDelaySeconds: 30,
			MaxBatchSize:      6,
			MaxOpenFiles:      10,
		},
		AccountsTrieStorage: config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 10000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "AccountsTrie/MainDB",
				Type:              "MemoryDB",
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		PeerAccountsTrieStorage: config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 10000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "PeerAccountsTrie/MainDB",
				Type:              "MemoryDB",
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		TxDataPool: config.CacheConfig{
			Size: 10000, Type: "LRU", Shards: 1,
		},
		UnsignedTransactionDataPool: config.CacheConfig{
			Size: 10000, Type: "LRU", Shards: 1,
		},
		RewardTransactionDataPool: config.CacheConfig{
			Size: 10000, Type: "LRU", Shards: 1,
		},
		HeadersPoolConfig: config.HeadersPoolConfig{
			MaxHeadersPerShard:            100,
			NumElementsToRemoveOnEviction: 1,
		},
		TxBlockBodyDataPool: config.CacheConfig{
			Size: 10000, Type: "LRU", Shards: 1,
		},
		PeerBlockBodyDataPool: config.CacheConfig{
			Size: 10000, Type: "LRU", Shards: 1,
		},
		TrieNodesDataPool: config.CacheConfig{
			Size: 10000, Type: "LRU", Shards: 1,
		},
		TxStorage: config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 10000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "Transactions",
				Type:              "MemoryDB",
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		MiniBlocksStorage: config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 10000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "MiniBlocks",
				Type:              string(storageUnit.LvlDBSerial),
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		MiniBlockHeadersStorage: config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 10000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "MiniBlockHeaders",
				Type:              string(storageUnit.LvlDBSerial),
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		ShardHdrNonceHashStorage: config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 10000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "ShardHdrHashNonce",
				Type:              string(storageUnit.LvlDBSerial),
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		MetaBlockStorage: config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 10000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "MetaBlock",
				Type:              string(storageUnit.LvlDBSerial),
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		MetaHdrNonceHashStorage: config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 10000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "MetaHdrHashNonce",
				Type:              string(storageUnit.LvlDBSerial),
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		UnsignedTransactionStorage: config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 10000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "UnsignedTransactions",
				Type:              "MemoryDB",
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		RewardTxStorage: config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 10000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "RewardTransactions",
				Type:              "MemoryDB",
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		BlockHeaderStorage: config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 10000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "BlockHeaders",
				Type:              string(storageUnit.LvlDBSerial),
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		Heartbeat: config.HeartbeatConfig{
			HeartbeatStorage: config.StorageConfig{
				Cache: config.CacheConfig{
					Size: 10000, Type: "LRU", Shards: 1,
				},
				DB: config.DBConfig{
					FilePath:          "HeartbeatStorage",
					Type:              "MemoryDB",
					BatchDelaySeconds: 30,
					MaxBatchSize:      6,
					MaxOpenFiles:      10,
				},
			},
		},
		StatusMetricsStorage: config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 10000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "StatusMetricsStorageDB",
				Type:              "MemoryDB",
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		PeerBlockBodyStorage: config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 10000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "PeerBlocks",
				Type:              string(storageUnit.LvlDBSerial),
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		BootstrapStorage: config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 10000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "BootstrapData",
				Type:              string(storageUnit.LvlDBSerial),
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		},
		GeneralSettings: config.GeneralSettingsConfig{
			StartInEpochEnabled: true,
		},
	}
}

func (e *epochStartMetaBlockProcessor) GetMapMetaBlock() map[string]*block.MetaBlock {
	e.mutReceivedMetaBlocks.RLock()
	defer e.mutReceivedMetaBlocks.RUnlock()

	return e.mapReceivedMetaBlocks
}

const DurationBetweenChecksForEpochStartMetaBlock = durationBetweenChecks

const DurationBetweenReRequest = durationBetweenReRequests
