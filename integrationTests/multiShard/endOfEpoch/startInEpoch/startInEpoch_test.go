package startInEpoch

import (
	"context"
	"encoding/hex"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	triesFactory "github.com/ElrondNetwork/elrond-go/data/trie/factory"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/epochStart/bootstrap"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/integrationTests/multiShard/endOfEpoch"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/process/sync/storageBootstrap"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/factory"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/stretchr/testify/assert"
)

func TestStartInEpochForAShardNodeInMultiShardedEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 2
	totalNodesPerShard := 4
	numNodesPerShardOnline := totalNodesPerShard - 1
	shardCnsSize := 2
	metaCnsSize := 3
	numMetachainNodes := 3

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()

	nodesMap := integrationTests.CreateNodesWithNodesCoordinator(
		numNodesPerShardOnline,
		numMetachainNodes,
		numOfShards,
		shardCnsSize,
		metaCnsSize,
		integrationTests.GetConnectableAddress(advertiser),
	)

	nodes := convertToSlice(nodesMap)

	// TODO: refactor test - node to join late should be created late.
	nodeToJoinLate := nodes[numNodesPerShardOnline] // will return the last node in shard 0 which was not used in consensus
	_ = nodeToJoinLate.Messenger.Close()            // set not offline
	// TODO: call nodeToJoinLate.Messenger.Bootstrap() later in the test and followed by a time.sleep as to allow it to bootstrap its peers.

	nodes = append(nodes[:numNodesPerShardOnline], nodes[numNodesPerShardOnline+1:]...)
	nodes = append(nodes[:2*numNodesPerShardOnline], nodes[2*numNodesPerShardOnline+1:]...)

	roundsPerEpoch := uint64(10)
	for _, node := range nodes {
		node.EpochStartTrigger.SetRoundsPerEpoch(roundsPerEpoch)
	}

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * numNodesPerShardOnline
	}
	idxProposers[numOfShards] = numOfShards * numNodesPerShardOnline

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Node.Stop()
		}
	}()

	initialVal := big.NewInt(10000000)
	sendValue := big.NewInt(5)
	integrationTests.MintAllNodes(nodes, initialVal)
	receiverAddress := []byte("12345678901234567890123456789012")

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	time.Sleep(time.Second)

	/////////----- wait for epoch end period
	epoch := uint32(2)
	nrRoundsToPropagateMultiShard := uint64(5)
	for i := uint64(0); i <= (uint64(epoch)*roundsPerEpoch)+nrRoundsToPropagateMultiShard; i++ {
		integrationTests.UpdateRound(nodes, round)
		integrationTests.ProposeBlock(nodes, idxProposers, round, nonce)
		integrationTests.SyncBlock(t, nodes, idxProposers, round)
		round = integrationTests.IncrementAndPrintRound(round)
		nonce++

		for _, node := range nodes {
			integrationTests.CreateAndSendTransaction(node, sendValue, receiverAddress, "")
		}

		time.Sleep(time.Second)
	}

	time.Sleep(time.Second)

	endOfEpoch.VerifyThatNodesHaveCorrectEpoch(t, epoch, nodes)
	endOfEpoch.VerifyIfAddedShardHeadersAreWithNewEpoch(t, nodes)

	epochHandler := &mock.EpochStartTriggerStub{
		EpochCalled: func() uint32 {
			return epoch
		},
	}
	for _, node := range nodes {
		_ = dataRetriever.SetEpochHandlerToHdrResolver(node.ResolversContainer, epochHandler)
	}

	// TODO: refactor this test in another PR

	generalConfig := getGeneralConfig()
	roundDurationMillis := 4000
	epochDurationMillis := generalConfig.EpochStartConfig.RoundsPerEpoch * int64(roundDurationMillis)
	nodesConfig := sharding.NodesSetup{
		StartTime:     time.Now().Add(-time.Duration(epochDurationMillis) * time.Millisecond).Unix(),
		RoundDuration: 4000,
		InitialNodes:  getInitialNodes(nodesMap),
		ChainID:       string(integrationTests.ChainID),
	}
	nodesConfig.SetNumberOfShards(uint32(numOfShards))

	defer func() {
		errRemoveDir := os.RemoveAll("Epoch_0")
		assert.NoError(t, errRemoveDir)
	}()

	genesisShardCoordinator, _ := sharding.NewMultiShardCoordinator(nodesConfig.NumberOfShards(), 0)
	messenger := integrationTests.CreateMessengerWithKadDht(context.Background(), integrationTests.GetConnectableAddress(advertiser))
	_ = messenger.Bootstrap()
	time.Sleep(integrationTests.P2pBootstrapDelay)

	trieStorageManager, triesHolder, _ := createTries(getGeneralConfig(), integrationTests.TestMarshalizer, integrationTests.TestHasher, 0, &mock.PathManagerStub{})
	argsBootstrapHandler := bootstrap.ArgsEpochStartBootstrap{
		PublicKey:                  nodeToJoinLate.NodeKeys.Pk,
		Marshalizer:                integrationTests.TestMarshalizer,
		TxSignMarshalizer:          integrationTests.TestTxSignMarshalizer,
		Hasher:                     integrationTests.TestHasher,
		Messenger:                  messenger,
		GeneralConfig:              getGeneralConfig(),
		GenesisShardCoordinator:    genesisShardCoordinator,
		EconomicsData:              integrationTests.CreateEconomicsData(),
		SingleSigner:               &mock.SignerMock{},
		BlockSingleSigner:          &mock.SignerMock{},
		KeyGen:                     &mock.KeyGenMock{},
		BlockKeyGen:                &mock.KeyGenMock{},
		GenesisNodesConfig:         &nodesConfig,
		PathManager:                &mock.PathManagerStub{},
		WorkingDir:                 "test_directory",
		DefaultDBPath:              "test_db",
		DefaultEpochString:         "test_epoch",
		DefaultShardString:         "test_shard",
		Rater:                      &mock.RaterMock{},
		DestinationShardAsObserver: "0",
		TrieContainer:              triesHolder,
		TrieStorageManagers:        trieStorageManager,
	}
	epochStartBootstrap, err := bootstrap.NewEpochStartBootstrap(argsBootstrapHandler)
	assert.Nil(t, err)

	_, err = epochStartBootstrap.Bootstrap()
	assert.NoError(t, err)
	//assert.Equal(t, epoch, params.Epoch)
	//assert.Equal(t, uint32(0), params.SelfShardId)
	//assert.Equal(t, uint32(2), params.NumOfShards)

	shardC, _ := sharding.NewMultiShardCoordinator(2, 0)

	storageFactory, err := factory.NewStorageServiceFactory(
		&generalConfig,
		shardC,
		&mock.PathManagerStub{},
		&mock.EpochStartNotifierStub{},
		0)
	assert.NoError(t, err)
	storageServiceShard, err := storageFactory.CreateForShard()
	assert.NoError(t, err)
	assert.NotNil(t, storageServiceShard)

	bootstrapUnit := storageServiceShard.GetStorer(dataRetriever.BootstrapUnit)
	assert.NotNil(t, bootstrapUnit)

	bootstrapStorer, err := bootstrapStorage.NewBootstrapStorer(integrationTests.TestMarshalizer, bootstrapUnit)
	assert.NoError(t, err)
	assert.NotNil(t, bootstrapStorer)

	bootstrapperArgs := storageBootstrap.ArgsShardStorageBootstrapper{
		ArgsBaseStorageBootstrapper: storageBootstrap.ArgsBaseStorageBootstrapper{
			BootStorer: bootstrapStorer,
			ForkDetector: &mock.ForkDetectorMock{
				RestoreToGenesisCalled: func() {},
			},
			BlockProcessor:      &mock.BlockProcessorMock{},
			ChainHandler:        &mock.BlockChainMock{},
			Marshalizer:         integrationTests.TestMarshalizer,
			Store:               storageServiceShard,
			Uint64Converter:     &mock.Uint64ByteSliceConverterMock{},
			BootstrapRoundIndex: round + 3,
			ShardCoordinator:    shardC,
			NodesCoordinator:    &mock.NodesCoordinatorMock{},
			EpochStartTrigger:   &mock.EpochStartTriggerStub{},
			ResolversFinder: &mock.ResolversFinderStub{
				IntraShardResolverCalled: func(baseTopic string) (dataRetriever.Resolver, error) {
					return &mock.MiniBlocksResolverMock{
						GetMiniBlocksCalled: func(hashes [][]byte) (block.MiniBlockSlice, [][]byte) {
							return nil, nil
						},
					}, nil
				},
			},
			BlockTracker: &mock.BlockTrackerStub{
				RestoreToGenesisCalled: func() {},
			},
		},
	}

	bootstrapper, err := storageBootstrap.NewShardStorageBootstrapper(bootstrapperArgs)
	assert.NoError(t, err)
	assert.NotNil(t, bootstrapper)

	err = bootstrapper.LoadFromStorage()
	assert.NoError(t, err)
	highestNonce := bootstrapper.GetHighestBlockNonce()
	assert.Equal(t, uint64(22), highestNonce)
}

func createTries(
	config config.Config,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	shardId uint32,
	pathManager storage.PathManagerHandler,
) (map[string]data.StorageManager, state.TriesHolder, error) {

	trieContainer := state.NewDataTriesHolder()
	trieFactoryArgs := triesFactory.TrieFactoryArgs{
		EvictionWaitingListCfg: config.EvictionWaitingList,
		SnapshotDbCfg:          config.TrieSnapshotDB,
		Marshalizer:            marshalizer,
		Hasher:                 hasher,
		PathManager:            pathManager,
		ShardId:                core.GetShardIdString(shardId),
	}
	trieFactory, err := triesFactory.NewTrieFactory(trieFactoryArgs)
	if err != nil {
		return nil, nil, err
	}

	trieStorageManagers := make(map[string]data.StorageManager)
	userStorageManager, userAccountTrie, err := trieFactory.Create(config.AccountsTrieStorage, config.StateTriesConfig.AccountsStatePruningEnabled)
	if err != nil {
		return nil, nil, err
	}
	trieContainer.Put([]byte(triesFactory.UserAccountTrie), userAccountTrie)
	trieStorageManagers[triesFactory.UserAccountTrie] = userStorageManager

	peerStorageManager, peerAccountsTrie, err := trieFactory.Create(config.PeerAccountsTrieStorage, config.StateTriesConfig.PeerStatePruningEnabled)
	if err != nil {
		return nil, nil, err
	}
	trieContainer.Put([]byte(triesFactory.PeerAccountTrie), peerAccountsTrie)
	trieStorageManagers[triesFactory.PeerAccountTrie] = peerStorageManager

	return trieStorageManagers, trieContainer, nil
}

func convertToSlice(originalMap map[uint32][]*integrationTests.TestProcessorNode) []*integrationTests.TestProcessorNode {
	sliceToRet := make([]*integrationTests.TestProcessorNode, 0)
	for _, nodesPerShard := range originalMap {
		for _, node := range nodesPerShard {
			sliceToRet = append(sliceToRet, node)
		}
	}

	return sliceToRet
}

func getInitialNodes(nodesMap map[uint32][]*integrationTests.TestProcessorNode) []*sharding.InitialNode {
	sliceToRet := make([]*sharding.InitialNode, 0)
	for _, nodesPerShard := range nodesMap {
		for _, node := range nodesPerShard {
			pubKeyBytes, _ := node.NodeKeys.Pk.ToByteArray()
			addressBytes := node.OwnAccount.Address.Bytes()
			entry := &sharding.InitialNode{
				PubKey:   hex.EncodeToString(pubKeyBytes),
				Address:  hex.EncodeToString(addressBytes),
				NodeInfo: sharding.NodeInfo{},
			}
			sliceToRet = append(sliceToRet, entry)
		}
	}

	return sliceToRet
}

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
	}
}
