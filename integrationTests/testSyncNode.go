package integrationTests

import (
	"fmt"
	syncGo "sync"

	arwenConfig "github.com/ElrondNetwork/arwen-wasm-vm/v1_4/config"
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/common/enablers"
	"github.com/ElrondNetwork/elrond-go/common/forking"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/consensus/spos/sposFactory"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/provider"
	"github.com/ElrondNetwork/elrond-go/epochStart/bootstrap/disabled"
	"github.com/ElrondNetwork/elrond-go/epochStart/notifier"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/p2p/rating"
	"github.com/ElrondNetwork/elrond-go/process/block"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/process/smartContract"
	"github.com/ElrondNetwork/elrond-go/process/sync"
	"github.com/ElrondNetwork/elrond-go/process/transactionLog"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/sharding/nodesCoordinator"
	"github.com/ElrondNetwork/elrond-go/state"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/ElrondNetwork/elrond-go/testscommon/dblookupext"
	"github.com/ElrondNetwork/elrond-go/testscommon/shardingMocks"
)

// NewTestSyncNode returns a new TestProcessorNode instance with sync capabilities
func NewTestSyncNode(
	maxShards uint32,
	nodeShardId uint32,
	txSignPrivKeyShardId uint32,
) *TestProcessorNode {

	shardCoordinator, _ := sharding.NewMultiShardCoordinator(maxShards, nodeShardId)
	pkBytes := []byte("afafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafaf")
	address := []byte("afafafafafafafafafafafafafafafaf")

	nodesSetup := &mock.NodesSetupStub{
		InitialNodesInfoCalled: func() (m map[uint32][]nodesCoordinator.GenesisNodeInfoHandler, m2 map[uint32][]nodesCoordinator.GenesisNodeInfoHandler) {
			oneMap := make(map[uint32][]nodesCoordinator.GenesisNodeInfoHandler)
			oneMap[0] = append(oneMap[0], mock.NewNodeInfo(address, pkBytes, 0, InitialRating))
			return oneMap, nil
		},
		InitialNodesInfoForShardCalled: func(shardId uint32) (handlers []nodesCoordinator.GenesisNodeInfoHandler, handlers2 []nodesCoordinator.GenesisNodeInfoHandler, err error) {
			list := make([]nodesCoordinator.GenesisNodeInfoHandler, 0)
			list = append(list, mock.NewNodeInfo(address, pkBytes, 0, InitialRating))
			return list, nil, nil
		},
		GetMinTransactionVersionCalled: func() uint32 {
			return MinTransactionVersion
		},
	}

	nodesCoordinatorInstance := &shardingMocks.NodesCoordinatorStub{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validators []nodesCoordinator.Validator, err error) {
			v, _ := nodesCoordinator.NewValidator(pkBytes, 1, defaultChancesSelection)
			return []nodesCoordinator.Validator{v}, nil
		},
		GetAllValidatorsPublicKeysCalled: func() (map[uint32][][]byte, error) {
			keys := make(map[uint32][][]byte)
			keys[0] = make([][]byte, 0)
			keys[0] = append(keys[0], pkBytes)
			return keys, nil
		},
		GetValidatorWithPublicKeyCalled: func(publicKey []byte) (nodesCoordinator.Validator, uint32, error) {
			validator, _ := nodesCoordinator.NewValidator(publicKey, defaultChancesSelection, 1)
			return validator, 0, nil
		},
	}

	peersRatingHandler, _ := rating.NewPeersRatingHandler(
		rating.ArgPeersRatingHandler{
			TopRatedCache: testscommon.NewCacherMock(),
			BadRatedCache: testscommon.NewCacherMock(),
		})

	messenger := CreateMessengerWithNoDiscoveryAndPeersRatingHandler(peersRatingHandler)
	genericEpochNotifier := forking.NewGenericEpochNotifier()
	enableEpochsHandler, _ := enablers.NewEnableEpochsHandler(config.EnableEpochs{}, genericEpochNotifier)

	logsProcessor, _ := transactionLog.NewTxLogProcessor(transactionLog.ArgTxLogProcessor{Marshalizer: TestMarshalizer})
	tpn := &TestProcessorNode{
		ShardCoordinator: shardCoordinator,
		Messenger:        messenger,
		NodesCoordinator: nodesCoordinatorInstance,
		BootstrapStorer: &mock.BoostrapStorerMock{
			PutCalled: func(round int64, bootData bootstrapStorage.BootstrapData) error {
				return nil
			},
		},
		StorageBootstrapper:     &mock.StorageBootstrapperMock{},
		HeaderSigVerifier:       &mock.HeaderSigVerifierStub{},
		HeaderIntegrityVerifier: CreateHeaderIntegrityVerifier(),
		ChainID:                 ChainID,
		EpochStartTrigger:       &mock.EpochStartTriggerStub{},
		NodesSetup:              nodesSetup,
		MinTransactionVersion:   MinTransactionVersion,
		HistoryRepository:       &dblookupext.HistoryRepositoryStub{},
		EpochNotifier:           forking.NewGenericEpochNotifier(),
		ArwenChangeLocker:       &syncGo.RWMutex{},
		TransactionLogProcessor: logsProcessor,
		PeersRatingHandler:      peersRatingHandler,
		PeerShardMapper:         disabled.NewPeerShardMapper(),
		EnableEpochsHandler:     enableEpochsHandler,
	}

	kg := &mock.KeyGenMock{}
	sk, pk := kg.GeneratePair()
	tpn.NodeKeys = &TestKeyPair{
		Sk: sk,
		Pk: pk,
	}

	tpn.MultiSigner = TestMultiSig
	tpn.OwnAccount = CreateTestWalletAccount(shardCoordinator, txSignPrivKeyShardId)
	tpn.initDataPools()
	tpn.initTestNodeWithSync()

	return tpn
}

func (tpn *TestProcessorNode) initTestNodeWithSync() {
	tpn.EnableEpochs.ScheduledMiniBlocksEnableEpoch = UnreachableEpoch
	tpn.EnableEpochsHandler, _ = enablers.NewEnableEpochsHandler(tpn.EnableEpochs, tpn.EpochNotifier)
	tpn.NetworkShardingCollector = mock.NewNetworkShardingCollectorMock()
	tpn.initChainHandler()
	tpn.initHeaderValidator()
	tpn.initRoundHandler()
	tpn.initStorage()
	tpn.EpochStartNotifier = notifier.NewEpochStartSubscriptionHandler()
	tpn.initAccountDBsWithPruningStorer()
	tpn.GenesisBlocks = CreateSimpleGenesisBlocks(tpn.ShardCoordinator)
	tpn.initEconomicsData(tpn.createDefaultEconomicsConfig())
	tpn.initRatingsData()
	tpn.initRequestedItemsHandler()
	tpn.initResolvers()
	tpn.initBlockTracker()
	tpn.initInterceptors("")
	tpn.initInnerProcessors(arwenConfig.MakeGasMapForTests())
	tpn.initBlockProcessorWithSync()
	tpn.BroadcastMessenger, _ = sposFactory.GetBroadcastMessenger(
		TestMarshalizer,
		TestHasher,
		tpn.Messenger,
		tpn.ShardCoordinator,
		tpn.OwnAccount.SkTxSign,
		tpn.OwnAccount.PeerSigHandler,
		tpn.DataPool.Headers(),
		tpn.InterceptorsContainer,
		&testscommon.AlarmSchedulerStub{},
	)
	tpn.initBootstrapper()
	tpn.setGenesisBlock()
	tpn.initNode()
	argsNewScQueryService := smartContract.ArgsNewSCQueryService{
		VmContainer:              tpn.VMContainer,
		EconomicsFee:             tpn.EconomicsData,
		BlockChainHook:           tpn.BlockchainHook,
		BlockChain:               tpn.BlockChain,
		ArwenChangeLocker:        tpn.ArwenChangeLocker,
		Bootstrapper:             tpn.Bootstrapper,
		AllowExternalQueriesChan: common.GetClosedUnbufferedChannel(),
	}
	tpn.SCQueryService, _ = smartContract.NewSCQueryService(argsNewScQueryService)
	tpn.addHandlersForCounters()
	tpn.addGenesisBlocksIntoStorage()
}

func (tpn *TestProcessorNode) addGenesisBlocksIntoStorage() {
	for shardId, header := range tpn.GenesisBlocks {
		buffHeader, _ := TestMarshalizer.Marshal(header)
		headerHash := TestHasher.Compute(string(buffHeader))

		if shardId == core.MetachainShardId {
			metablockStorer := tpn.Storage.GetStorer(dataRetriever.MetaBlockUnit)
			_ = metablockStorer.Put(headerHash, buffHeader)
		} else {
			shardblockStorer := tpn.Storage.GetStorer(dataRetriever.BlockHeaderUnit)
			_ = shardblockStorer.Put(headerHash, buffHeader)
		}
	}
}

func (tpn *TestProcessorNode) initBlockProcessorWithSync() {
	var err error

	accountsDb := make(map[state.AccountsDbIdentifier]state.AccountsAdapter)
	accountsDb[state.UserAccountsState] = tpn.AccntState
	accountsDb[state.PeerAccountsState] = tpn.PeerState

	coreComponents := GetDefaultCoreComponents()
	coreComponents.InternalMarshalizerField = TestMarshalizer
	coreComponents.HasherField = TestHasher
	coreComponents.Uint64ByteSliceConverterField = TestUint64Converter
	coreComponents.EpochNotifierField = tpn.EpochNotifier
	coreComponents.EnableEpochsHandlerField = &testscommon.EnableEpochsHandlerStub{}

	dataComponents := GetDefaultDataComponents()
	dataComponents.Store = tpn.Storage
	dataComponents.DataPool = tpn.DataPool
	dataComponents.BlockChain = tpn.BlockChain

	bootstrapComponents := getDefaultBootstrapComponents(tpn.ShardCoordinator)
	bootstrapComponents.HdrIntegrityVerifier = tpn.HeaderIntegrityVerifier

	statusComponents := GetDefaultStatusComponents()

	triesConfig := config.Config{
		StateTriesConfig: config.StateTriesConfig{
			CheckpointRoundsModulus: stateCheckpointModulus,
		},
	}

	argumentsBase := block.ArgBaseProcessor{
		CoreComponents:      coreComponents,
		DataComponents:      dataComponents,
		BootstrapComponents: bootstrapComponents,
		StatusComponents:    statusComponents,
		Config:              triesConfig,
		AccountsDB:          accountsDb,
		ForkDetector:        nil,
		NodesCoordinator:    tpn.NodesCoordinator,
		FeeHandler:          tpn.FeeAccumulator,
		RequestHandler:      tpn.RequestHandler,
		BlockChainHook:      &testscommon.BlockChainHookStub{},
		EpochStartTrigger:   &mock.EpochStartTriggerStub{},
		HeaderValidator:     tpn.HeaderValidator,
		BootStorer: &mock.BoostrapStorerMock{
			PutCalled: func(round int64, bootData bootstrapStorage.BootstrapData) error {
				return nil
			},
		},
		BlockTracker:                 tpn.BlockTracker,
		BlockSizeThrottler:           TestBlockSizeThrottler,
		HistoryRepository:            tpn.HistoryRepository,
		EnableRoundsHandler:          coreComponents.EnableRoundsHandler(),
		GasHandler:                   tpn.GasHandler,
		ScheduledTxsExecutionHandler: &testscommon.ScheduledTxsExecutionStub{},
		ProcessedMiniBlocksTracker:   &testscommon.ProcessedMiniBlocksTrackerStub{},
	}

	if tpn.ShardCoordinator.SelfId() == core.MetachainShardId {
		tpn.ForkDetector, _ = sync.NewMetaForkDetector(tpn.RoundHandler, tpn.BlockBlackListHandler, tpn.BlockTracker, 0)
		argumentsBase.ForkDetector = tpn.ForkDetector
		argumentsBase.TxCoordinator = &mock.TransactionCoordinatorMock{}
		arguments := block.ArgMetaProcessor{
			ArgBaseProcessor:             argumentsBase,
			SCToProtocol:                 &mock.SCToProtocolStub{},
			PendingMiniBlocksHandler:     &mock.PendingMiniBlocksHandlerStub{},
			EpochStartDataCreator:        &mock.EpochStartDataCreatorStub{},
			EpochEconomics:               &mock.EpochEconomicsStub{},
			EpochRewardsCreator:          &mock.EpochRewardsCreatorStub{},
			EpochValidatorInfoCreator:    &mock.EpochValidatorInfoCreatorStub{},
			ValidatorStatisticsProcessor: &mock.ValidatorStatisticsProcessorStub{},
			EpochSystemSCProcessor:       &mock.EpochStartSystemSCStub{},
		}

		tpn.BlockProcessor, err = block.NewMetaProcessor(arguments)
	} else {
		tpn.ForkDetector, _ = sync.NewShardForkDetector(tpn.RoundHandler, tpn.BlockBlackListHandler, tpn.BlockTracker, 0)
		argumentsBase.ForkDetector = tpn.ForkDetector
		argumentsBase.BlockChainHook = tpn.BlockchainHook
		argumentsBase.TxCoordinator = tpn.TxCoordinator
		argumentsBase.ScheduledTxsExecutionHandler = &testscommon.ScheduledTxsExecutionStub{}
		arguments := block.ArgShardProcessor{
			ArgBaseProcessor: argumentsBase,
		}

		tpn.BlockProcessor, err = block.NewShardProcessor(arguments)
	}

	if err != nil {
		panic(fmt.Sprintf("Error creating blockprocessor: %s", err.Error()))
	}
}

func (tpn *TestProcessorNode) createShardBootstrapper() (TestBootstrapper, error) {
	argsBaseBootstrapper := sync.ArgBaseBootstrapper{
		PoolsHolder:                  tpn.DataPool,
		Store:                        tpn.Storage,
		ChainHandler:                 tpn.BlockChain,
		RoundHandler:                 tpn.RoundHandler,
		BlockProcessor:               tpn.BlockProcessor,
		WaitTime:                     tpn.RoundHandler.TimeDuration(),
		Hasher:                       TestHasher,
		Marshalizer:                  TestMarshalizer,
		ForkDetector:                 tpn.ForkDetector,
		RequestHandler:               tpn.RequestHandler,
		ShardCoordinator:             tpn.ShardCoordinator,
		Accounts:                     tpn.AccntState,
		BlackListHandler:             tpn.BlockBlackListHandler,
		NetworkWatcher:               tpn.Messenger,
		BootStorer:                   tpn.BootstrapStorer,
		StorageBootstrapper:          tpn.StorageBootstrapper,
		EpochHandler:                 tpn.EpochStartTrigger,
		MiniblocksProvider:           tpn.MiniblocksProvider,
		Uint64Converter:              TestUint64Converter,
		AppStatusHandler:             TestAppStatusHandler,
		OutportHandler:               mock.NewNilOutport(),
		AccountsDBSyncer:             &mock.AccountsDBSyncerStub{},
		CurrentEpochProvider:         &testscommon.CurrentEpochProviderStub{},
		IsInImportMode:               false,
		HistoryRepo:                  &dblookupext.HistoryRepositoryStub{},
		ScheduledTxsExecutionHandler: &testscommon.ScheduledTxsExecutionStub{},
		ProcessWaitTime:              tpn.RoundHandler.TimeDuration(),
	}

	argsShardBootstrapper := sync.ArgShardBootstrapper{
		ArgBaseBootstrapper: argsBaseBootstrapper,
	}

	bootstrap, err := sync.NewShardBootstrap(argsShardBootstrapper)
	if err != nil {
		return nil, err
	}

	return &sync.TestShardBootstrap{
		ShardBootstrap: bootstrap,
	}, nil
}

func (tpn *TestProcessorNode) createMetaChainBootstrapper() (TestBootstrapper, error) {
	argsBaseBootstrapper := sync.ArgBaseBootstrapper{
		PoolsHolder:                  tpn.DataPool,
		Store:                        tpn.Storage,
		ChainHandler:                 tpn.BlockChain,
		RoundHandler:                 tpn.RoundHandler,
		BlockProcessor:               tpn.BlockProcessor,
		WaitTime:                     tpn.RoundHandler.TimeDuration(),
		Hasher:                       TestHasher,
		Marshalizer:                  TestMarshalizer,
		ForkDetector:                 tpn.ForkDetector,
		RequestHandler:               tpn.RequestHandler,
		ShardCoordinator:             tpn.ShardCoordinator,
		Accounts:                     tpn.AccntState,
		BlackListHandler:             tpn.BlockBlackListHandler,
		NetworkWatcher:               tpn.Messenger,
		BootStorer:                   tpn.BootstrapStorer,
		StorageBootstrapper:          tpn.StorageBootstrapper,
		EpochHandler:                 tpn.EpochStartTrigger,
		MiniblocksProvider:           tpn.MiniblocksProvider,
		Uint64Converter:              TestUint64Converter,
		AppStatusHandler:             TestAppStatusHandler,
		OutportHandler:               mock.NewNilOutport(),
		AccountsDBSyncer:             &mock.AccountsDBSyncerStub{},
		CurrentEpochProvider:         &testscommon.CurrentEpochProviderStub{},
		IsInImportMode:               false,
		HistoryRepo:                  &dblookupext.HistoryRepositoryStub{},
		ScheduledTxsExecutionHandler: &testscommon.ScheduledTxsExecutionStub{},
		ProcessWaitTime:              tpn.RoundHandler.TimeDuration(),
	}

	argsMetaBootstrapper := sync.ArgMetaBootstrapper{
		ArgBaseBootstrapper:         argsBaseBootstrapper,
		EpochBootstrapper:           tpn.EpochStartTrigger,
		ValidatorAccountsDB:         tpn.PeerState,
		ValidatorStatisticsDBSyncer: &mock.AccountsDBSyncerStub{},
	}

	bootstrap, err := sync.NewMetaBootstrap(argsMetaBootstrapper)
	if err != nil {
		return nil, err
	}

	return &sync.TestMetaBootstrap{
		MetaBootstrap: bootstrap,
	}, nil
}

func (tpn *TestProcessorNode) initBootstrapper() {
	tpn.createMiniblocksProvider()

	if tpn.ShardCoordinator.SelfId() < tpn.ShardCoordinator.NumberOfShards() {
		tpn.Bootstrapper, _ = tpn.createShardBootstrapper()
	} else {
		tpn.Bootstrapper, _ = tpn.createMetaChainBootstrapper()
	}
}

func (tpn *TestProcessorNode) createMiniblocksProvider() {
	arg := provider.ArgMiniBlockProvider{
		MiniBlockPool:    tpn.DataPool.MiniBlocks(),
		MiniBlockStorage: tpn.Storage.GetStorer(dataRetriever.MiniBlockUnit),
		Marshalizer:      TestMarshalizer,
	}

	miniblockGetter, err := provider.NewMiniBlockProvider(arg)
	log.LogIfError(err)

	tpn.MiniblocksProvider = miniblockGetter
}
