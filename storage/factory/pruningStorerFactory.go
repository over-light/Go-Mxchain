package factory

import (
	"fmt"
	"path/filepath"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/clean"
	"github.com/ElrondNetwork/elrond-go/storage/databaseremover"
	"github.com/ElrondNetwork/elrond-go/storage/databaseremover/disabled"
	"github.com/ElrondNetwork/elrond-go/storage/databaseremover/factory"
	storageDisabled "github.com/ElrondNetwork/elrond-go/storage/disabled"
	"github.com/ElrondNetwork/elrond-go/storage/pruning"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
)

var log = logger.GetOrCreate("storage/factory")

const (
	minimumNumberOfActivePersisters = 1
	minimumNumberOfEpochsToKeep     = 2
)

// StorageServiceFactory handles the creation of storage services for both meta and shards
type StorageServiceFactory struct {
	generalConfig                 config.Config
	prefsConfig                   config.PreferencesConfig
	shardCoordinator              storage.ShardCoordinator
	pathManager                   storage.PathManagerHandler
	epochStartNotifier            storage.EpochStartNotifier
	oldDataCleanerProvider        clean.OldDataCleanerProvider
	createTrieEpochRootHashStorer bool
	currentEpoch                  uint32
}

// StorageServiceFactoryArgs holds the arguments needed for creating a new storage service factory
type StorageServiceFactoryArgs struct {
	Config                        config.Config
	PrefsConfig                   config.PreferencesConfig
	ShardCoordinator              storage.ShardCoordinator
	PathManager                   storage.PathManagerHandler
	EpochStartNotifier            storage.EpochStartNotifier
	NodeTypeProvider              NodeTypeProviderHandler
	CurrentEpoch                  uint32
	CreateTrieEpochRootHashStorer bool
}

// NewStorageServiceFactory will return a new instance of StorageServiceFactory
func NewStorageServiceFactory(args StorageServiceFactoryArgs) (*StorageServiceFactory, error) {
	err := checkArgs(args)
	if err != nil {
		return nil, err
	}

	oldDataCleanProvider, err := clean.NewOldDataCleanerProvider(
		args.NodeTypeProvider,
		args.Config.StoragePruning,
	)
	if err != nil {
		return nil, err
	}
	if args.Config.StoragePruning.NumEpochsToKeep < minimumNumberOfEpochsToKeep && oldDataCleanProvider.ShouldClean() {
		return nil, storage.ErrInvalidNumberOfEpochsToSave
	}

	return &StorageServiceFactory{
		generalConfig:                 args.Config,
		prefsConfig:                   args.PrefsConfig,
		shardCoordinator:              args.ShardCoordinator,
		pathManager:                   args.PathManager,
		epochStartNotifier:            args.EpochStartNotifier,
		currentEpoch:                  args.CurrentEpoch,
		createTrieEpochRootHashStorer: args.CreateTrieEpochRootHashStorer,
		oldDataCleanerProvider:        oldDataCleanProvider,
	}, nil
}

func checkArgs(args StorageServiceFactoryArgs) error {
	if args.Config.StoragePruning.NumActivePersisters < minimumNumberOfActivePersisters {
		return storage.ErrInvalidNumberOfActivePersisters
	}
	if check.IfNil(args.ShardCoordinator) {
		return storage.ErrNilShardCoordinator
	}
	if check.IfNil(args.PathManager) {
		return storage.ErrNilPathManager
	}
	if check.IfNil(args.EpochStartNotifier) {
		return storage.ErrNilEpochStartNotifier
	}

	return nil
}

// CreateForShard will return the storage service which contains all storers needed for a shard
func (psf *StorageServiceFactory) CreateForShard() (dataRetriever.StorageService, error) {
	var headerUnit storage.Storer
	var peerBlockUnit storage.Storer
	var miniBlockUnit storage.Storer
	var txUnit storage.Storer
	var metachainHeaderUnit storage.Storer
	var unsignedTxUnit storage.Storer
	var rewardTxUnit storage.Storer
	var bootstrapUnit storage.Storer
	var receiptsUnit storage.Storer
	var userAccountsUnit storage.Storer
	var peerAccountsUnit storage.Storer
	var userAccountsCheckpointsUnit storage.Storer
	var peerAccountsCheckpointsUnit storage.Storer
	var scheduledSCRsUnit storage.Storer
	var err error

	// TODO: if there will be a differentiation between the creation or opening of a DB, the DBs could be destroyed on a defer
	// in case of a failure while creating (not opening).

	disabledCustomDatabaseRemover := disabled.NewDisabledCustomDatabaseRemover()
	customDatabaseRemover, err := factory.CreateCustomDatabaseRemover(psf.generalConfig.StoragePruning)
	if err != nil {
		return nil, err
	}

	txUnitStorerArgs := psf.createPruningStorerArgs(psf.generalConfig.TxStorage, disabledCustomDatabaseRemover)
	txUnit, err = psf.createPruningPersister(txUnitStorerArgs)
	if err != nil {
		return nil, err
	}

	unsignedTxUnitStorerArgs := psf.createPruningStorerArgs(psf.generalConfig.UnsignedTransactionStorage, disabledCustomDatabaseRemover)
	unsignedTxUnit, err = psf.createPruningPersister(unsignedTxUnitStorerArgs)
	if err != nil {
		return nil, err
	}

	rewardTxUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.RewardTxStorage, disabledCustomDatabaseRemover)
	rewardTxUnit, err = psf.createPruningPersister(rewardTxUnitArgs)
	if err != nil {
		return nil, err
	}

	miniBlockUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.MiniBlocksStorage, disabledCustomDatabaseRemover)
	miniBlockUnit, err = psf.createPruningPersister(miniBlockUnitArgs)
	if err != nil {
		return nil, err
	}

	peerBlockUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.PeerBlockBodyStorage, disabledCustomDatabaseRemover)
	peerBlockUnit, err = psf.createPruningPersister(peerBlockUnitArgs)
	if err != nil {
		return nil, err
	}

	headerUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.BlockHeaderStorage, disabledCustomDatabaseRemover)
	headerUnit, err = psf.createPruningPersister(headerUnitArgs)
	if err != nil {
		return nil, err
	}

	metaChainHeaderUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.MetaBlockStorage, disabledCustomDatabaseRemover)
	metachainHeaderUnit, err = psf.createPruningPersister(metaChainHeaderUnitArgs)
	if err != nil {
		return nil, err
	}

	userAccountsUnit, err = psf.createTriePersister(psf.generalConfig.AccountsTrieStorage, psf.generalConfig.StateTriesConfig, customDatabaseRemover)
	if err != nil {
		return nil, err
	}

	peerAccountsUnit, err = psf.createTriePersister(psf.generalConfig.PeerAccountsTrieStorage, psf.generalConfig.StateTriesConfig, customDatabaseRemover)
	if err != nil {
		return nil, err
	}

	userAccountsCheckpointsUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.AccountsTrieCheckpointsStorage, disabledCustomDatabaseRemover)
	userAccountsCheckpointsUnit, err = psf.createPruningPersister(userAccountsCheckpointsUnitArgs)
	if err != nil {
		return nil, err
	}

	peerAccountsCheckpointsUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.PeerAccountsTrieCheckpointsStorage, disabledCustomDatabaseRemover)
	peerAccountsCheckpointsUnit, err = psf.createPruningPersister(peerAccountsCheckpointsUnitArgs)
	if err != nil {
		return nil, err
	}

	// metaHdrHashNonce is static
	metaHdrHashNonceUnitConfig := GetDBFromConfig(psf.generalConfig.MetaHdrNonceHashStorage.DB)
	shardID := core.GetShardIDString(psf.shardCoordinator.SelfId())
	dbPath := psf.pathManager.PathForStatic(shardID, psf.generalConfig.MetaHdrNonceHashStorage.DB.FilePath)
	metaHdrHashNonceUnitConfig.FilePath = dbPath
	metaHdrHashNonceUnit, err := storageUnit.NewStorageUnitFromConf(
		GetCacherFromConfig(psf.generalConfig.MetaHdrNonceHashStorage.Cache),
		metaHdrHashNonceUnitConfig)
	if err != nil {
		return nil, err
	}

	// shardHdrHashNonce storer is static
	shardHdrHashNonceConfig := GetDBFromConfig(psf.generalConfig.ShardHdrNonceHashStorage.DB)
	shardID = core.GetShardIDString(psf.shardCoordinator.SelfId())
	dbPath = psf.pathManager.PathForStatic(shardID, psf.generalConfig.ShardHdrNonceHashStorage.DB.FilePath) + shardID
	shardHdrHashNonceConfig.FilePath = dbPath
	shardHdrHashNonceUnit, err := storageUnit.NewStorageUnitFromConf(
		GetCacherFromConfig(psf.generalConfig.ShardHdrNonceHashStorage.Cache),
		shardHdrHashNonceConfig)
	if err != nil {
		return nil, err
	}

	heartbeatDbConfig := GetDBFromConfig(psf.generalConfig.Heartbeat.HeartbeatStorage.DB)
	shardId := core.GetShardIDString(psf.shardCoordinator.SelfId())
	dbPath = psf.pathManager.PathForStatic(shardId, psf.generalConfig.Heartbeat.HeartbeatStorage.DB.FilePath)
	heartbeatDbConfig.FilePath = dbPath
	heartbeatStorageUnit, err := storageUnit.NewStorageUnitFromConf(
		GetCacherFromConfig(psf.generalConfig.Heartbeat.HeartbeatStorage.Cache),
		heartbeatDbConfig)
	if err != nil {
		return nil, err
	}

	statusMetricsDbConfig := GetDBFromConfig(psf.generalConfig.StatusMetricsStorage.DB)
	shardId = core.GetShardIDString(psf.shardCoordinator.SelfId())
	dbPath = psf.pathManager.PathForStatic(shardId, psf.generalConfig.StatusMetricsStorage.DB.FilePath)
	statusMetricsDbConfig.FilePath = dbPath
	statusMetricsStorageUnit, err := storageUnit.NewStorageUnitFromConf(
		GetCacherFromConfig(psf.generalConfig.StatusMetricsStorage.Cache),
		statusMetricsDbConfig)
	if err != nil {
		return nil, err
	}

	trieEpochRootHashStorageUnit, err := psf.createTrieEpochRootHashStorerIfNeeded()
	if err != nil {
		return nil, err
	}

	bootstrapUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.BootstrapStorage, disabledCustomDatabaseRemover)
	bootstrapUnit, err = psf.createPruningPersister(bootstrapUnitArgs)
	if err != nil {
		return nil, err
	}

	receiptsUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.ReceiptsStorage, disabledCustomDatabaseRemover)
	receiptsUnit, err = psf.createPruningPersister(receiptsUnitArgs)
	if err != nil {
		return nil, err
	}

	scheduledSCRsUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.ScheduledSCRsStorage, disabledCustomDatabaseRemover)
	scheduledSCRsUnit, err = psf.createPruningPersister(scheduledSCRsUnitArgs)
	if err != nil {
		return nil, err
	}

	store := dataRetriever.NewChainStorer()
	store.AddStorer(dataRetriever.TransactionUnit, txUnit)
	store.AddStorer(dataRetriever.MiniBlockUnit, miniBlockUnit)
	store.AddStorer(dataRetriever.PeerChangesUnit, peerBlockUnit)
	store.AddStorer(dataRetriever.BlockHeaderUnit, headerUnit)
	store.AddStorer(dataRetriever.MetaBlockUnit, metachainHeaderUnit)
	store.AddStorer(dataRetriever.UnsignedTransactionUnit, unsignedTxUnit)
	store.AddStorer(dataRetriever.RewardTransactionUnit, rewardTxUnit)
	store.AddStorer(dataRetriever.MetaHdrNonceHashDataUnit, metaHdrHashNonceUnit)
	hdrNonceHashDataUnit := dataRetriever.ShardHdrNonceHashDataUnit + dataRetriever.UnitType(psf.shardCoordinator.SelfId())
	store.AddStorer(hdrNonceHashDataUnit, shardHdrHashNonceUnit)
	store.AddStorer(dataRetriever.HeartbeatUnit, heartbeatStorageUnit)
	store.AddStorer(dataRetriever.BootstrapUnit, bootstrapUnit)
	store.AddStorer(dataRetriever.StatusMetricsUnit, statusMetricsStorageUnit)
	store.AddStorer(dataRetriever.ReceiptsUnit, receiptsUnit)
	store.AddStorer(dataRetriever.TrieEpochRootHashUnit, trieEpochRootHashStorageUnit)
	store.AddStorer(dataRetriever.UserAccountsUnit, userAccountsUnit)
	store.AddStorer(dataRetriever.PeerAccountsUnit, peerAccountsUnit)
	store.AddStorer(dataRetriever.UserAccountsCheckpointsUnit, userAccountsCheckpointsUnit)
	store.AddStorer(dataRetriever.PeerAccountsCheckpointsUnit, peerAccountsCheckpointsUnit)
	store.AddStorer(dataRetriever.ScheduledSCRsUnit, scheduledSCRsUnit)

	err = psf.setupDbLookupExtensions(store)
	if err != nil {
		return nil, err
	}

	err = psf.setupLogsAndEventsStorer(store)
	if err != nil {
		return nil, err
	}

	err = psf.initOldDatabasesCleaningIfNeeded(store)
	if err != nil {
		return nil, err
	}

	return store, err
}

// CreateForMeta will return the storage service which contains all storers needed for metachain
func (psf *StorageServiceFactory) CreateForMeta() (dataRetriever.StorageService, error) {
	var metaBlockUnit storage.Storer
	var headerUnit storage.Storer
	var txUnit storage.Storer
	var miniBlockUnit storage.Storer
	var unsignedTxUnit storage.Storer
	var rewardTxUnit storage.Storer
	var bootstrapUnit storage.Storer
	var receiptsUnit storage.Storer
	var userAccountsUnit storage.Storer
	var peerAccountsUnit storage.Storer
	var userAccountsCheckpointsUnit storage.Storer
	var peerAccountsCheckpointsUnit storage.Storer
	var scheduledSCRsUnit storage.Storer
	var err error

	// TODO: if there will be a differentiation between the creation or opening of a DB, the DBs could be destroyed on a defer
	// in case of a failure while creating (not opening)

	disabledCustomDatabaseRemover := disabled.NewDisabledCustomDatabaseRemover()
	customDatabaseRemover, err := databaseremover.NewCustomDatabaseRemover(psf.generalConfig.StoragePruning)
	if err != nil {
		return nil, err
	}

	metaBlockUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.MetaBlockStorage, disabledCustomDatabaseRemover)
	metaBlockUnit, err = psf.createPruningPersister(metaBlockUnitArgs)
	if err != nil {
		return nil, err
	}

	headerUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.BlockHeaderStorage, disabledCustomDatabaseRemover)
	headerUnit, err = psf.createPruningPersister(headerUnitArgs)
	if err != nil {
		return nil, err
	}

	userAccountsUnit, err = psf.createTriePersister(psf.generalConfig.AccountsTrieStorage, psf.generalConfig.StateTriesConfig, customDatabaseRemover)
	if err != nil {
		return nil, err
	}

	peerAccountsUnit, err = psf.createTriePersister(psf.generalConfig.PeerAccountsTrieStorage, psf.generalConfig.StateTriesConfig, customDatabaseRemover)
	if err != nil {
		return nil, err
	}

	userAccountsCheckpointsUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.AccountsTrieCheckpointsStorage, disabledCustomDatabaseRemover)
	userAccountsCheckpointsUnit, err = psf.createPruningPersister(userAccountsCheckpointsUnitArgs)
	if err != nil {
		return nil, err
	}

	peerAccountsCheckpointsUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.PeerAccountsTrieCheckpointsStorage, disabledCustomDatabaseRemover)
	peerAccountsCheckpointsUnit, err = psf.createPruningPersister(peerAccountsCheckpointsUnitArgs)
	if err != nil {
		return nil, err
	}

	// metaHdrHashNonce is static
	metaHdrHashNonceUnitConfig := GetDBFromConfig(psf.generalConfig.MetaHdrNonceHashStorage.DB)
	shardID := core.GetShardIDString(core.MetachainShardId)
	dbPath := psf.pathManager.PathForStatic(shardID, psf.generalConfig.MetaHdrNonceHashStorage.DB.FilePath)
	metaHdrHashNonceUnitConfig.FilePath = dbPath
	metaHdrHashNonceUnit, err := storageUnit.NewStorageUnitFromConf(
		GetCacherFromConfig(psf.generalConfig.MetaHdrNonceHashStorage.Cache),
		metaHdrHashNonceUnitConfig)
	if err != nil {
		return nil, err
	}

	shardHdrHashNonceUnits := make([]*storageUnit.Unit, psf.shardCoordinator.NumberOfShards())
	for i := uint32(0); i < psf.shardCoordinator.NumberOfShards(); i++ {
		shardHdrHashNonceConfig := GetDBFromConfig(psf.generalConfig.ShardHdrNonceHashStorage.DB)
		shardID = core.GetShardIDString(core.MetachainShardId)
		dbPath = psf.pathManager.PathForStatic(shardID, psf.generalConfig.ShardHdrNonceHashStorage.DB.FilePath) + fmt.Sprintf("%d", i)
		shardHdrHashNonceConfig.FilePath = dbPath
		shardHdrHashNonceUnits[i], err = storageUnit.NewStorageUnitFromConf(
			GetCacherFromConfig(psf.generalConfig.ShardHdrNonceHashStorage.Cache),
			shardHdrHashNonceConfig)
		if err != nil {
			return nil, err
		}
	}

	shardId := core.GetShardIDString(psf.shardCoordinator.SelfId())
	heartbeatDbConfig := GetDBFromConfig(psf.generalConfig.Heartbeat.HeartbeatStorage.DB)
	dbPath = psf.pathManager.PathForStatic(shardId, psf.generalConfig.Heartbeat.HeartbeatStorage.DB.FilePath)
	heartbeatDbConfig.FilePath = dbPath
	heartbeatStorageUnit, err := storageUnit.NewStorageUnitFromConf(
		GetCacherFromConfig(psf.generalConfig.Heartbeat.HeartbeatStorage.Cache),
		heartbeatDbConfig)
	if err != nil {
		return nil, err
	}

	statusMetricsDbConfig := GetDBFromConfig(psf.generalConfig.StatusMetricsStorage.DB)
	shardId = core.GetShardIDString(psf.shardCoordinator.SelfId())
	dbPath = psf.pathManager.PathForStatic(shardId, psf.generalConfig.StatusMetricsStorage.DB.FilePath)
	statusMetricsDbConfig.FilePath = dbPath
	statusMetricsStorageUnit, err := storageUnit.NewStorageUnitFromConf(
		GetCacherFromConfig(psf.generalConfig.StatusMetricsStorage.Cache),
		statusMetricsDbConfig)
	if err != nil {
		return nil, err
	}

	trieEpochRootHashStorageUnit, err := psf.createTrieEpochRootHashStorerIfNeeded()
	if err != nil {
		return nil, err
	}

	txUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.TxStorage, disabledCustomDatabaseRemover)
	txUnit, err = psf.createPruningPersister(txUnitArgs)
	if err != nil {
		return nil, err
	}

	unsignedTxUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.UnsignedTransactionStorage, disabledCustomDatabaseRemover)
	unsignedTxUnit, err = psf.createPruningPersister(unsignedTxUnitArgs)
	if err != nil {
		return nil, err
	}

	rewardTxUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.RewardTxStorage, disabledCustomDatabaseRemover)
	rewardTxUnit, err = psf.createPruningPersister(rewardTxUnitArgs)
	if err != nil {
		return nil, err
	}

	miniBlockUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.MiniBlocksStorage, disabledCustomDatabaseRemover)
	miniBlockUnit, err = psf.createPruningPersister(miniBlockUnitArgs)
	if err != nil {
		return nil, err
	}

	bootstrapUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.BootstrapStorage, disabledCustomDatabaseRemover)
	bootstrapUnit, err = psf.createPruningPersister(bootstrapUnitArgs)
	if err != nil {
		return nil, err
	}

	receiptsUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.ReceiptsStorage, disabledCustomDatabaseRemover)
	receiptsUnit, err = psf.createPruningPersister(receiptsUnitArgs)
	if err != nil {
		return nil, err
	}

	scheduledSCRsUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.ScheduledSCRsStorage, disabledCustomDatabaseRemover)
	scheduledSCRsUnit, err = pruning.NewPruningStorer(scheduledSCRsUnitArgs)
	if err != nil {
		return nil, err
	}

	store := dataRetriever.NewChainStorer()
	store.AddStorer(dataRetriever.MetaBlockUnit, metaBlockUnit)
	store.AddStorer(dataRetriever.BlockHeaderUnit, headerUnit)
	store.AddStorer(dataRetriever.MetaHdrNonceHashDataUnit, metaHdrHashNonceUnit)
	store.AddStorer(dataRetriever.TransactionUnit, txUnit)
	store.AddStorer(dataRetriever.UnsignedTransactionUnit, unsignedTxUnit)
	store.AddStorer(dataRetriever.MiniBlockUnit, miniBlockUnit)
	store.AddStorer(dataRetriever.RewardTransactionUnit, rewardTxUnit)
	for i := uint32(0); i < psf.shardCoordinator.NumberOfShards(); i++ {
		hdrNonceHashDataUnit := dataRetriever.ShardHdrNonceHashDataUnit + dataRetriever.UnitType(i)
		store.AddStorer(hdrNonceHashDataUnit, shardHdrHashNonceUnits[i])
	}
	store.AddStorer(dataRetriever.HeartbeatUnit, heartbeatStorageUnit)
	store.AddStorer(dataRetriever.BootstrapUnit, bootstrapUnit)
	store.AddStorer(dataRetriever.StatusMetricsUnit, statusMetricsStorageUnit)
	store.AddStorer(dataRetriever.ReceiptsUnit, receiptsUnit)
	store.AddStorer(dataRetriever.TrieEpochRootHashUnit, trieEpochRootHashStorageUnit)
	store.AddStorer(dataRetriever.UserAccountsUnit, userAccountsUnit)
	store.AddStorer(dataRetriever.PeerAccountsUnit, peerAccountsUnit)
	store.AddStorer(dataRetriever.UserAccountsCheckpointsUnit, userAccountsCheckpointsUnit)
	store.AddStorer(dataRetriever.PeerAccountsCheckpointsUnit, peerAccountsCheckpointsUnit)
	store.AddStorer(dataRetriever.ScheduledSCRsUnit, scheduledSCRsUnit)

	err = psf.setupDbLookupExtensions(store)
	if err != nil {
		return nil, err
	}

	err = psf.setupLogsAndEventsStorer(store)
	if err != nil {
		return nil, err
	}

	err = psf.initOldDatabasesCleaningIfNeeded(store)
	if err != nil {
		return nil, err
	}

	return store, err
}

func (psf *StorageServiceFactory) setupLogsAndEventsStorer(chainStorer *dataRetriever.ChainStorer) error {
	var txLogsUnit storage.Storer
	txLogsUnit = storageDisabled.NewStorer()

	// Should not create logs and events storer in the next case:
	// - LogsAndEvents.Enabled = false and DbLookupExtensions.Enabled = false
	// If we have DbLookupExtensions ACTIVE node by default should save logs no matter if is enabled or not
	shouldCreateStorer := psf.generalConfig.LogsAndEvents.SaveInStorageEnabled || psf.generalConfig.DbLookupExtensions.Enabled
	if shouldCreateStorer {
		var err error
		txLogsUnitArgs := psf.createPruningStorerArgs(psf.generalConfig.LogsAndEvents.TxLogsStorage, disabled.NewDisabledCustomDatabaseRemover())
		txLogsUnit, err = psf.createPruningPersister(txLogsUnitArgs)
		if err != nil {
			return err
		}
	}

	chainStorer.AddStorer(dataRetriever.TxLogsUnit, txLogsUnit)

	return nil
}

func (psf *StorageServiceFactory) setupDbLookupExtensions(chainStorer *dataRetriever.ChainStorer) error {
	if !psf.generalConfig.DbLookupExtensions.Enabled {
		return nil
	}

	shardID := core.GetShardIDString(psf.shardCoordinator.SelfId())

	// Create the eventsHashesByTxHash (PRUNING) storer
	eventsHashesByTxHashConfig := psf.generalConfig.DbLookupExtensions.ResultsHashesByTxHashStorageConfig
	eventsHashesByTxHashStorerArgs := psf.createPruningStorerArgs(eventsHashesByTxHashConfig, disabled.NewDisabledCustomDatabaseRemover())
	eventsHashesByTxHashPruningStorer, err := psf.createPruningPersister(eventsHashesByTxHashStorerArgs)
	if err != nil {
		return err
	}

	chainStorer.AddStorer(dataRetriever.ResultsHashesByTxHashUnit, eventsHashesByTxHashPruningStorer)

	// Create the miniblocksMetadata (PRUNING) storer
	miniblocksMetadataConfig := psf.generalConfig.DbLookupExtensions.MiniblocksMetadataStorageConfig
	miniblocksMetadataPruningStorerArgs := psf.createPruningStorerArgs(miniblocksMetadataConfig, disabled.NewDisabledCustomDatabaseRemover())
	miniblocksMetadataPruningStorer, err := psf.createPruningPersister(miniblocksMetadataPruningStorerArgs)
	if err != nil {
		return err
	}

	chainStorer.AddStorer(dataRetriever.MiniblocksMetadataUnit, miniblocksMetadataPruningStorer)

	// Create the miniblocksHashByTxHash (STATIC) storer
	miniblockHashByTxHashConfig := psf.generalConfig.DbLookupExtensions.MiniblockHashByTxHashStorageConfig
	miniblockHashByTxHashDbConfig := GetDBFromConfig(miniblockHashByTxHashConfig.DB)
	miniblockHashByTxHashDbConfig.FilePath = psf.pathManager.PathForStatic(shardID, miniblockHashByTxHashConfig.DB.FilePath)
	miniblockHashByTxHashCacherConfig := GetCacherFromConfig(miniblockHashByTxHashConfig.Cache)
	miniblockHashByTxHashUnit, err := storageUnit.NewStorageUnitFromConf(miniblockHashByTxHashCacherConfig, miniblockHashByTxHashDbConfig)
	if err != nil {
		return err
	}

	chainStorer.AddStorer(dataRetriever.MiniblockHashByTxHashUnit, miniblockHashByTxHashUnit)

	// Create the blockHashByRound (STATIC) storer
	blockHashByRoundConfig := psf.generalConfig.DbLookupExtensions.RoundHashStorageConfig
	blockHashByRoundDBConfig := GetDBFromConfig(blockHashByRoundConfig.DB)
	blockHashByRoundDBConfig.FilePath = psf.pathManager.PathForStatic(shardID, blockHashByRoundConfig.DB.FilePath)
	blockHashByRoundCacherConfig := GetCacherFromConfig(blockHashByRoundConfig.Cache)
	blockHashByRoundUnit, err := storageUnit.NewStorageUnitFromConf(blockHashByRoundCacherConfig, blockHashByRoundDBConfig)
	if err != nil {
		return err
	}

	chainStorer.AddStorer(dataRetriever.RoundHdrHashDataUnit, blockHashByRoundUnit)

	// Create the epochByHash (STATIC) storer
	epochByHashConfig := psf.generalConfig.DbLookupExtensions.EpochByHashStorageConfig
	epochByHashDbConfig := GetDBFromConfig(epochByHashConfig.DB)
	epochByHashDbConfig.FilePath = psf.pathManager.PathForStatic(shardID, epochByHashConfig.DB.FilePath)
	epochByHashCacherConfig := GetCacherFromConfig(epochByHashConfig.Cache)
	epochByHashUnit, err := storageUnit.NewStorageUnitFromConf(epochByHashCacherConfig, epochByHashDbConfig)
	if err != nil {
		return err
	}

	chainStorer.AddStorer(dataRetriever.EpochByHashUnit, epochByHashUnit)

	esdtSuppliesConfig := psf.generalConfig.DbLookupExtensions.ESDTSuppliesStorageConfig
	esdtSuppliesDbConfig := GetDBFromConfig(esdtSuppliesConfig.DB)
	esdtSuppliesDbConfig.FilePath = psf.pathManager.PathForStatic(shardID, esdtSuppliesConfig.DB.FilePath)
	esdtSuppliesCacherConfig := GetCacherFromConfig(esdtSuppliesConfig.Cache)
	esdtSuppliesUnit, err := storageUnit.NewStorageUnitFromConf(esdtSuppliesCacherConfig, esdtSuppliesDbConfig)
	if err != nil {
		return err
	}

	chainStorer.AddStorer(dataRetriever.ESDTSuppliesUnit, esdtSuppliesUnit)

	return nil
}

func (psf *StorageServiceFactory) createPruningStorerArgs(
	storageConfig config.StorageConfig,
	customDatabaseRemover storage.CustomDatabaseRemoverHandler,
) *pruning.StorerArgs {
	numOfEpochsToKeep := uint32(psf.generalConfig.StoragePruning.NumEpochsToKeep)
	numOfActivePersisters := uint32(psf.generalConfig.StoragePruning.NumActivePersisters)
	pruningEnabled := psf.generalConfig.StoragePruning.Enabled
	shardId := core.GetShardIDString(psf.shardCoordinator.SelfId())
	dbPath := filepath.Join(psf.pathManager.PathForEpoch(shardId, psf.currentEpoch, storageConfig.DB.FilePath))
	args := &pruning.StorerArgs{
		Identifier:                storageConfig.DB.FilePath,
		PruningEnabled:            pruningEnabled,
		StartingEpoch:             psf.currentEpoch,
		OldDataCleanerProvider:    psf.oldDataCleanerProvider,
		CustomDatabaseRemover:     customDatabaseRemover,
		ShardCoordinator:          psf.shardCoordinator,
		CacheConf:                 GetCacherFromConfig(storageConfig.Cache),
		PathManager:               psf.pathManager,
		DbPath:                    dbPath,
		PersisterFactory:          NewPersisterFactory(storageConfig.DB),
		NumOfEpochsToKeep:         numOfEpochsToKeep,
		NumOfActivePersisters:     numOfActivePersisters,
		Notifier:                  psf.epochStartNotifier,
		MaxBatchSize:              storageConfig.DB.MaxBatchSize,
		EnabledDbLookupExtensions: psf.generalConfig.DbLookupExtensions.Enabled,
	}

	return args
}

func (psf *StorageServiceFactory) createTrieEpochRootHashStorerIfNeeded() (storage.Storer, error) {
	if !psf.createTrieEpochRootHashStorer {
		return storageUnit.NewNilStorer(), nil
	}

	trieEpochRootHashDbConfig := GetDBFromConfig(psf.generalConfig.TrieEpochRootHashStorage.DB)
	shardId := core.GetShardIDString(psf.shardCoordinator.SelfId())
	dbPath := psf.pathManager.PathForStatic(shardId, psf.generalConfig.TrieEpochRootHashStorage.DB.FilePath)
	trieEpochRootHashDbConfig.FilePath = dbPath
	trieEpochRootHashStorageUnit, err := storageUnit.NewStorageUnitFromConf(
		GetCacherFromConfig(psf.generalConfig.TrieEpochRootHashStorage.Cache),
		trieEpochRootHashDbConfig)
	if err != nil {
		return nil, err
	}

	return trieEpochRootHashStorageUnit, nil
}

func (psf *StorageServiceFactory) createTriePersister(
	storageConfig config.StorageConfig,
	triesConfig config.StateTriesConfig,
	customDatabaseRemover storage.CustomDatabaseRemoverHandler,
) (storage.Storer, error) {
	if triesConfig.SnapshotsEnabled {
		pruningPersisterArgs := psf.createPruningStorerArgs(storageConfig, customDatabaseRemover)
		return psf.createTriePruningPersister(pruningPersisterArgs)
	}

	trieDBConfig := GetDBFromConfig(storageConfig.DB)
	shardID := core.GetShardIDString(psf.shardCoordinator.SelfId())
	dbPath := psf.pathManager.PathForStatic(shardID, storageConfig.DB.FilePath)
	trieDBConfig.FilePath = dbPath
	trieUnit, err := storageUnit.NewStorageUnitFromConf(
		GetCacherFromConfig(storageConfig.Cache),
		trieDBConfig)
	if err != nil {
		return nil, err
	}

	return trieUnit, nil
}

func (psf *StorageServiceFactory) createTriePruningPersister(arg *pruning.StorerArgs) (storage.Storer, error) {
	isFullArchive := psf.prefsConfig.FullArchive
	isDBLookupExtension := psf.generalConfig.DbLookupExtensions.Enabled
	if !isFullArchive && !isDBLookupExtension {
		return pruning.NewTriePruningStorer(arg)
	}

	numOldActivePersisters := psf.getNumActivePersistersForFullHistoryStorer(isFullArchive, isDBLookupExtension)
	historyArgs := &pruning.FullHistoryStorerArgs{
		StorerArgs:               arg,
		NumOfOldActivePersisters: numOldActivePersisters,
	}

	return pruning.NewFullHistoryTriePruningStorer(historyArgs)
}

func (psf *StorageServiceFactory) createPruningPersister(arg *pruning.StorerArgs) (storage.Storer, error) {
	isFullArchive := psf.prefsConfig.FullArchive
	isDBLookupExtension := psf.generalConfig.DbLookupExtensions.Enabled
	if !isFullArchive && !isDBLookupExtension {
		return pruning.NewPruningStorer(arg)
	}

	numOldActivePersisters := psf.getNumActivePersistersForFullHistoryStorer(isFullArchive, isDBLookupExtension)
	historyArgs := &pruning.FullHistoryStorerArgs{
		StorerArgs:               arg,
		NumOfOldActivePersisters: numOldActivePersisters,
	}

	return pruning.NewFullHistoryPruningStorer(historyArgs)
}

func (psf *StorageServiceFactory) getNumActivePersistersForFullHistoryStorer(isFullArchive bool, isDBLookupExtension bool) uint32 {
	if isFullArchive && !isDBLookupExtension {
		return psf.generalConfig.StoragePruning.FullArchiveNumActivePersisters
	}

	if !isFullArchive && isDBLookupExtension {
		return psf.generalConfig.DbLookupExtensions.DbLookupMaxActivePersisters
	}

	if psf.generalConfig.DbLookupExtensions.DbLookupMaxActivePersisters != psf.generalConfig.StoragePruning.FullArchiveNumActivePersisters {
		log.Warn("node is started with both Full Archive and DB Lookup Extension modes and have different values " +
			"for the number of active persisters. It will use NumOfOldActivePersisters from full archive's settings")
	}

	return psf.generalConfig.StoragePruning.FullArchiveNumActivePersisters
}

func (psf *StorageServiceFactory) initOldDatabasesCleaningIfNeeded(store dataRetriever.StorageService) error {
	isFullArchive := psf.prefsConfig.FullArchive
	if isFullArchive {
		return nil
	}
	_, err := clean.NewOldDatabaseCleaner(clean.ArgsOldDatabaseCleaner{
		DatabasePath:           psf.pathManager.DatabasePath(),
		StorageListProvider:    store,
		EpochStartNotifier:     psf.epochStartNotifier,
		OldDataCleanerProvider: psf.oldDataCleanerProvider,
	})

	return err
}
