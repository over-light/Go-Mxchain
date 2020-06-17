package factory

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/blockchain"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	dataRetrieverFactory "github.com/ElrondNetwork/elrond-go/dataRetriever/factory"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/factory"
)

// DataComponentsFactoryArgs holds the arguments needed for creating a data components factory
type DataComponentsFactoryArgs struct {
	Config             config.Config
	EconomicsData      *economics.EconomicsData
	ShardCoordinator   sharding.Coordinator
	Core               *CoreComponents
	PathManager        storage.PathManagerHandler
	EpochStartNotifier EpochStartNotifier
	CurrentEpoch       uint32
	HealthService      HealthService
}

type dataComponentsFactory struct {
	config             config.Config
	economicsData      *economics.EconomicsData
	shardCoordinator   sharding.Coordinator
	core               *CoreComponents
	pathManager        storage.PathManagerHandler
	epochStartNotifier EpochStartNotifier
	currentEpoch       uint32
	healthService      HealthService
}

// NewDataComponentsFactory will return a new instance of dataComponentsFactory
func NewDataComponentsFactory(args DataComponentsFactoryArgs) (*dataComponentsFactory, error) {
	if args.EconomicsData == nil {
		return nil, ErrNilEconomicsData
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, ErrNilShardCoordinator
	}
	if args.Core == nil {
		return nil, ErrNilCoreComponents
	}
	if check.IfNil(args.PathManager) {
		return nil, ErrNilPathManager
	}
	if check.IfNil(args.EpochStartNotifier) {
		return nil, ErrNilEpochStartNotifier
	}
	if check.IfNil(args.HealthService) {
		return nil, ErrNilHealthService
	}

	return &dataComponentsFactory{
		config:             args.Config,
		economicsData:      args.EconomicsData,
		shardCoordinator:   args.ShardCoordinator,
		core:               args.Core,
		pathManager:        args.PathManager,
		epochStartNotifier: args.EpochStartNotifier,
		currentEpoch:       args.CurrentEpoch,
		healthService:      args.HealthService,
	}, nil
}

// Create will create and return the data components
func (dcf *dataComponentsFactory) Create() (*DataComponents, error) {
	var datapool dataRetriever.PoolsHolder
	blkc, err := dcf.createBlockChainFromConfig()
	if err != nil {
		return nil, err
	}

	store, err := dcf.createDataStoreFromConfig()
	if err != nil {
		return nil, err
	}

	dataPoolArgs := dataRetrieverFactory.ArgsDataPool{
		Config:           &dcf.config,
		EconomicsData:    dcf.economicsData,
		ShardCoordinator: dcf.shardCoordinator,
		HealthService:    dcf.healthService,
	}
	datapool, err = dataRetrieverFactory.NewDataPoolFromConfig(dataPoolArgs)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDataPoolCreation, err.Error())
	}

	return &DataComponents{
		Blkc:     blkc,
		Store:    store,
		Datapool: datapool,
	}, nil
}

func (dcf *dataComponentsFactory) createBlockChainFromConfig() (data.ChainHandler, error) {
	if dcf.shardCoordinator.SelfId() < dcf.shardCoordinator.NumberOfShards() {
		blockChain := blockchain.NewBlockChain()

		err := blockChain.SetAppStatusHandler(dcf.core.StatusHandler)
		if err != nil {
			return nil, err
		}

		return blockChain, nil
	}
	if dcf.shardCoordinator.SelfId() == core.MetachainShardId {
		blockChain := blockchain.NewMetaChain()

		err := blockChain.SetAppStatusHandler(dcf.core.StatusHandler)
		if err != nil {
			return nil, err
		}

		return blockChain, nil
	}
	return nil, ErrBlockchainCreation
}

func (dcf *dataComponentsFactory) createDataStoreFromConfig() (dataRetriever.StorageService, error) {
	storageServiceFactory, err := factory.NewStorageServiceFactory(
		&dcf.config,
		dcf.shardCoordinator,
		dcf.pathManager,
		dcf.epochStartNotifier,
		dcf.currentEpoch,
	)
	if err != nil {
		return nil, err
	}
	if dcf.shardCoordinator.SelfId() < dcf.shardCoordinator.NumberOfShards() {
		return storageServiceFactory.CreateForShard()
	}
	if dcf.shardCoordinator.SelfId() == core.MetachainShardId {
		return storageServiceFactory.CreateForMeta()
	}
	return nil, ErrDataStoreCreation
}
