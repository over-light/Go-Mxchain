package pruning

import (
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
)

// StorerArgs will hold the arguments needed for PruningStorer
type StorerArgs struct {
	Identifier            string
	ShardCoordinator      storage.ShardCoordinator
	CacheConf             storageUnit.CacheConfig
	PathManager           storage.PathManagerHandler
	DbPath                string
	PersisterFactory      DbFactoryHandler
	BloomFilterConf       storageUnit.BloomConfig
	Notifier              EpochStartNotifier
	NumOfEpochsToKeep     uint32
	NumOfActivePersisters uint32
	StartingEpoch         uint32
	PruningEnabled        bool
	FullArchive           bool
	MaxBatchSize          int
}
