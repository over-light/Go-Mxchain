package integrationTests

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go/consensus/spos/sposFactory"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/provider"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/process/block"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/process/smartContract"
	"github.com/ElrondNetwork/elrond-go/process/sync"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// NewTestSyncNode returns a new TestProcessorNode instance with sync capabilities
func NewTestSyncNode(
	maxShards uint32,
	nodeShardId uint32,
	txSignPrivKeyShardId uint32,
	initialNodeAddr string,
) *TestProcessorNode {

	shardCoordinator, _ := sharding.NewMultiShardCoordinator(maxShards, nodeShardId)
	pkBytes := make([]byte, 128)
	pkBytes = []byte("afafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafaf")
	address := make([]byte, 32)
	address = []byte("afafafafafafafafafafafafafafafaf")

	nodesSetup := &mock.NodesSetupStub{
		InitialNodesInfoCalled: func() (m map[uint32][]sharding.GenesisNodeInfoHandler, m2 map[uint32][]sharding.GenesisNodeInfoHandler) {
			oneMap := make(map[uint32][]sharding.GenesisNodeInfoHandler)
			oneMap[0] = append(oneMap[0], mock.NewNodeInfo(address, pkBytes, 0))
			return oneMap, nil
		},
		InitialNodesInfoForShardCalled: func(shardId uint32) (handlers []sharding.GenesisNodeInfoHandler, handlers2 []sharding.GenesisNodeInfoHandler, err error) {
			list := make([]sharding.GenesisNodeInfoHandler, 0)
			list = append(list, mock.NewNodeInfo(address, pkBytes, 0))
			return list, nil, nil
		},
		GetMinTransactionVersionCalled: func() uint32 {
			return MinTransactionVersion
		},
	}

	nodesCoordinator := &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validators []sharding.Validator, err error) {
			v, _ := sharding.NewValidator(pkBytes, 1, defaultChancesSelection)
			return []sharding.Validator{v}, nil
		},
		GetAllValidatorsPublicKeysCalled: func() (map[uint32][][]byte, error) {
			keys := make(map[uint32][][]byte)
			keys[0] = make([][]byte, 0)
			keys[0] = append(keys[0], pkBytes)
			return keys, nil
		},
		GetValidatorWithPublicKeyCalled: func(publicKey []byte) (sharding.Validator, uint32, error) {
			validator, _ := sharding.NewValidator(publicKey, defaultChancesSelection, 1)
			return validator, 0, nil
		},
	}

	messenger := CreateMessengerWithKadDht(initialNodeAddr)

	tpn := &TestProcessorNode{
		ShardCoordinator: shardCoordinator,
		Messenger:        messenger,
		NodesCoordinator: nodesCoordinator,
		BootstrapStorer: &mock.BoostrapStorerMock{
			PutCalled: func(round int64, bootData bootstrapStorage.BootstrapData) error {
				return nil
			},
		},
		StorageBootstrapper:     &mock.StorageBootstrapperMock{},
		HeaderSigVerifier:       &mock.HeaderSigVerifierStub{},
		HeaderIntegrityVerifier: &mock.HeaderIntegrityVerifierStub{},
		ChainID:                 ChainID,
		EpochStartTrigger:       &mock.EpochStartTriggerStub{},
		NodesSetup:              nodesSetup,
		MinTransactionVersion:   MinTransactionVersion,
		HistoryProcessor:        &mock.HistoryProcessorStub{},
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
	tpn.NetworkShardingCollector = mock.NewNetworkShardingCollectorMock()
	tpn.initChainHandler()
	tpn.initHeaderValidator()
	tpn.initRounder()
	tpn.initStorage()
	tpn.initAccountDBs()
	tpn.GenesisBlocks = CreateSimpleGenesisBlocks(tpn.ShardCoordinator)
	tpn.initEconomicsData()
	tpn.initRatingsData()
	tpn.initRequestedItemsHandler()
	tpn.initResolvers()
	tpn.initBlockTracker()
	tpn.initInterceptors()
	tpn.initInnerProcessors()
	tpn.initBlockProcessorWithSync()
	tpn.BroadcastMessenger, _ = sposFactory.GetBroadcastMessenger(
		TestMarshalizer,
		TestHasher,
		tpn.Messenger,
		tpn.ShardCoordinator,
		tpn.OwnAccount.SkTxSign,
		tpn.OwnAccount.SingleSigner,
		tpn.DataPool.Headers(),
		tpn.InterceptorsContainer,
	)
	tpn.initBootstrapper()
	tpn.setGenesisBlock()
	tpn.initNode()
	tpn.SCQueryService, _ = smartContract.NewSCQueryService(tpn.VMContainer, tpn.EconomicsData)
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

	argumentsBase := block.ArgBaseProcessor{
		AccountsDB:        accountsDb,
		ForkDetector:      nil,
		Hasher:            TestHasher,
		Marshalizer:       TestMarshalizer,
		Store:             tpn.Storage,
		ShardCoordinator:  tpn.ShardCoordinator,
		NodesCoordinator:  tpn.NodesCoordinator,
		FeeHandler:        tpn.FeeAccumulator,
		Uint64Converter:   TestUint64Converter,
		RequestHandler:    tpn.RequestHandler,
		Core:              nil,
		BlockChainHook:    &mock.BlockChainHookHandlerMock{},
		EpochStartTrigger: &mock.EpochStartTriggerStub{},
		HeaderValidator:   tpn.HeaderValidator,
		Rounder:           &mock.RounderMock{},
		BootStorer: &mock.BoostrapStorerMock{
			PutCalled: func(round int64, bootData bootstrapStorage.BootstrapData) error {
				return nil
			},
		},
		BlockTracker:           tpn.BlockTracker,
		DataPool:               tpn.DataPool,
		StateCheckpointModulus: stateCheckpointModulus,
		BlockChain:             tpn.BlockChain,
		BlockSizeThrottler:     TestBlockSizeThrottler,
		Version:                string(SoftwareVersion),
		HistoryProcessor:       tpn.HistoryProcessor,
	}

	if tpn.ShardCoordinator.SelfId() == core.MetachainShardId {
		tpn.ForkDetector, _ = sync.NewMetaForkDetector(tpn.Rounder, tpn.BlockBlackListHandler, tpn.BlockTracker, 0)
		argumentsBase.Core = &mock.ServiceContainerMock{}
		argumentsBase.ForkDetector = tpn.ForkDetector
		argumentsBase.TxCoordinator = &mock.TransactionCoordinatorMock{}
		arguments := block.ArgMetaProcessor{
			ArgBaseProcessor:             argumentsBase,
			SCDataGetter:                 &mock.ScQueryStub{},
			SCToProtocol:                 &mock.SCToProtocolStub{},
			PendingMiniBlocksHandler:     &mock.PendingMiniBlocksHandlerStub{},
			EpochStartDataCreator:        &mock.EpochStartDataCreatorStub{},
			EpochEconomics:               &mock.EpochEconomicsStub{},
			EpochRewardsCreator:          &mock.EpochRewardsCreatorStub{},
			EpochValidatorInfoCreator:    &mock.EpochValidatorInfoCreatorStub{},
			ValidatorStatisticsProcessor: &mock.ValidatorStatisticsProcessorStub{},
		}

		tpn.BlockProcessor, err = block.NewMetaProcessor(arguments)
	} else {
		tpn.ForkDetector, _ = sync.NewShardForkDetector(tpn.Rounder, tpn.BlockBlackListHandler, tpn.BlockTracker, 0)
		argumentsBase.ForkDetector = tpn.ForkDetector
		argumentsBase.BlockChainHook = tpn.BlockchainHook
		argumentsBase.TxCoordinator = tpn.TxCoordinator
		arguments := block.ArgShardProcessor{
			ArgBaseProcessor: argumentsBase,
		}

		tpn.BlockProcessor, err = block.NewShardProcessor(arguments)
	}

	if err != nil {
		fmt.Printf("Error creating blockprocessor: %s\n", err.Error())
	}
}

func (tpn *TestProcessorNode) createShardBootstrapper() (TestBootstrapper, error) {
	argsBaseBootstrapper := sync.ArgBaseBootstrapper{
		PoolsHolder:         tpn.DataPool,
		Store:               tpn.Storage,
		ChainHandler:        tpn.BlockChain,
		Rounder:             tpn.Rounder,
		BlockProcessor:      tpn.BlockProcessor,
		WaitTime:            tpn.Rounder.TimeDuration(),
		Hasher:              TestHasher,
		Marshalizer:         TestMarshalizer,
		ForkDetector:        tpn.ForkDetector,
		RequestHandler:      tpn.RequestHandler,
		ShardCoordinator:    tpn.ShardCoordinator,
		Accounts:            tpn.AccntState,
		BlackListHandler:    tpn.BlockBlackListHandler,
		NetworkWatcher:      tpn.Messenger,
		BootStorer:          tpn.BootstrapStorer,
		StorageBootstrapper: tpn.StorageBootstrapper,
		EpochHandler:        tpn.EpochStartTrigger,
		MiniblocksProvider:  tpn.MiniblocksProvider,
		Uint64Converter:     TestUint64Converter,
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
		PoolsHolder:         tpn.DataPool,
		Store:               tpn.Storage,
		ChainHandler:        tpn.BlockChain,
		Rounder:             tpn.Rounder,
		BlockProcessor:      tpn.BlockProcessor,
		WaitTime:            tpn.Rounder.TimeDuration(),
		Hasher:              TestHasher,
		Marshalizer:         TestMarshalizer,
		ForkDetector:        tpn.ForkDetector,
		RequestHandler:      tpn.RequestHandler,
		ShardCoordinator:    tpn.ShardCoordinator,
		Accounts:            tpn.AccntState,
		BlackListHandler:    tpn.BlockBlackListHandler,
		NetworkWatcher:      tpn.Messenger,
		BootStorer:          tpn.BootstrapStorer,
		StorageBootstrapper: tpn.StorageBootstrapper,
		EpochHandler:        tpn.EpochStartTrigger,
		MiniblocksProvider:  tpn.MiniblocksProvider,
		Uint64Converter:     TestUint64Converter,
	}

	argsMetaBootstrapper := sync.ArgMetaBootstrapper{
		ArgBaseBootstrapper: argsBaseBootstrapper,
		EpochBootstrapper:   tpn.EpochStartTrigger,
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
