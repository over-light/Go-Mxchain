package hardFork

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	arwenConfig "github.com/ElrondNetwork/arwen-wasm-vm/config"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/genesis/process"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/update/factory"
	"github.com/ElrondNetwork/elrond-go/vm/systemSmartContracts/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var log = logger.GetOrCreate("integrationTests/hardfork")

func TestHardForkWithoutTransactionInMultiShardedEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 1
	nodesPerShard := 1
	numMetachainNodes := 1

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	roundsPerEpoch := uint64(10)
	for _, node := range nodes {
		node.EpochStartTrigger.SetRoundsPerEpoch(roundsPerEpoch)
	}

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	time.Sleep(time.Second)

	nrRoundsToPropagateMultiShard := 5
	/////////----- wait for epoch end period
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, int(roundsPerEpoch), nonce, round, idxProposers)

	time.Sleep(time.Second)

	nonce, _ = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)

	time.Sleep(time.Second)

	epoch := uint32(1)
	verifyIfNodesHaveCorrectEpoch(t, epoch, nodes)
	verifyIfNodesHaveCorrectNonce(t, nonce-1, nodes)
	verifyIfAddedShardHeadersAreWithNewEpoch(t, nodes)

	defer func() {
		for _, node := range nodes {
			_ = os.RemoveAll(node.ExportFolder)
		}
	}()

	exportStorageConfigs := hardForkExport(t, nodes)
	hardForkImport(t, nodes, exportStorageConfigs)
	checkGenesisBlocksStateIsEqual(t, nodes)
}

func TestEHardForkWithContinuousTransactionsInMultiShardedEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 2
	nodesPerShard := 1
	numMetachainNodes := 1

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
		integrationTests.GetConnectableAddress(advertiser),
	)

	roundsPerEpoch := uint64(10)
	for _, node := range nodes {
		node.EpochStartTrigger.SetRoundsPerEpoch(roundsPerEpoch)
	}

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	integrationTests.DisplayAndStartNodes(nodes)

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(10000000)
	sendValue := big.NewInt(5)
	integrationTests.MintAllNodes(nodes, initialVal)
	receiverAddress1 := []byte("12345678901234567890123456789012")
	receiverAddress2 := []byte("12345678901234567890123456789011")

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	time.Sleep(time.Second)

	/////////----- wait for epoch end period
	epoch := uint32(1)
	nrRoundsToPropagateMultiShard := uint64(6)
	for i := uint64(0); i <= (uint64(epoch)*roundsPerEpoch)+nrRoundsToPropagateMultiShard; i++ {
		round, nonce = integrationTests.ProposeAndSyncOneBlock(t, nodes, idxProposers, round, nonce)

		for _, node := range nodes {
			integrationTests.CreateAndSendTransaction(node, sendValue, receiverAddress1, "")
			integrationTests.CreateAndSendTransaction(node, sendValue, receiverAddress2, "")
		}

		time.Sleep(time.Second)
	}

	time.Sleep(time.Second)

	verifyIfNodesHaveCorrectEpoch(t, epoch, nodes)
	verifyIfNodesHaveCorrectNonce(t, nonce-1, nodes)
	verifyIfAddedShardHeadersAreWithNewEpoch(t, nodes)

	defer func() {
		for _, node := range nodes {
			_ = os.RemoveAll(node.ExportFolder)
			_ = os.RemoveAll("./Static")
		}
	}()

	exportStorageConfigs := hardForkExport(t, nodes)
	hardForkImport(t, nodes, exportStorageConfigs)
	checkGenesisBlocksStateIsEqual(t, nodes)
}

func hardForkExport(t *testing.T, nodes []*integrationTests.TestProcessorNode) []*config.StorageConfig {
	exportStorageConfigs := createHardForkExporter(t, nodes)
	for _, node := range nodes {
		log.Warn("***********************************************************************************")
		log.Warn("starting to export for node with shard", "id", node.ShardCoordinator.SelfId())
		err := node.ExportHandler.ExportAll(1)
		assert.Nil(t, err)
		log.Warn("***********************************************************************************")
	}
	return exportStorageConfigs
}

func checkGenesisBlocksStateIsEqual(t *testing.T, nodes []*integrationTests.TestProcessorNode) {
	for _, nodeA := range nodes {
		for _, nodeB := range nodes {
			for _, genesisBlockA := range nodeA.GenesisBlocks {
				genesisBlockB := nodeB.GenesisBlocks[genesisBlockA.GetShardID()]
				assert.True(t, bytes.Equal(genesisBlockA.GetRootHash(), genesisBlockB.GetRootHash()))
			}
		}
	}
}

func hardForkImport(
	t *testing.T,
	nodes []*integrationTests.TestProcessorNode,
	importStorageConfigs []*config.StorageConfig,
) {
	for id, node := range nodes {
		gasSchedule := arwenConfig.MakeGasMap(1)
		defaults.FillGasMapInternal(gasSchedule, 1)
		log.Warn("started import process")

		argsGenesis := process.ArgsGenesisBlockCreator{
			GenesisTime:              0,
			StartEpochNum:            0,
			Accounts:                 node.AccntState,
			PubkeyConv:               integrationTests.TestAddressPubkeyConverter,
			InitialNodesSetup:        node.NodesSetup,
			Economics:                node.EconomicsData.EconomicsData,
			ShardCoordinator:         node.ShardCoordinator,
			Store:                    node.Storage,
			Blkc:                     node.BlockChain,
			Marshalizer:              integrationTests.TestMarshalizer,
			Hasher:                   integrationTests.TestHasher,
			Uint64ByteSliceConverter: integrationTests.TestUint64Converter,
			DataPool:                 node.DataPool,
			ValidatorAccounts:        node.PeerState,
			GasMap:                   gasSchedule,
			TxLogsProcessor:          &mock.TxLogsProcessorStub{},
			VirtualMachineConfig:     config.VirtualMachineConfig{},
			HardForkConfig: config.HardforkConfig{
				MustImport:               true,
				ImportFolder:             node.ExportFolder,
				StartEpoch:               1000,
				StartNonce:               1000,
				StartRound:               1000,
				ImportStateStorageConfig: *importStorageConfigs[id],
			},
			TrieStorageManagers: node.TrieStorageManagers,
			ChainID:             string(node.ChainID),
			SystemSCConfig: config.SystemSmartContractsConfig{
				ESDTSystemSCConfig: config.ESDTSystemSCConfig{
					BaseIssuingCost: "1000",
					OwnerAddress:    "aaaaaa",
				},
			},
			AccountsParser:      &mock.AccountsParserStub{},
			SmartContractParser: &mock.SmartContractParserStub{},
			Version:             "version2",
		}

		genesisProcessor, err := process.NewGenesisBlockCreator(argsGenesis)
		require.Nil(t, err)
		genesisBlocks, err := genesisProcessor.CreateGenesisBlocks()
		require.Nil(t, err)
		require.NotNil(t, genesisBlocks)

		node.GenesisBlocks = genesisBlocks
		for _, genesisBlock := range genesisBlocks {
			log.Info("hardfork genesisblock roothash", "shardID", genesisBlock.GetShardID(), "rootHash", genesisBlock.GetRootHash())
		}
	}
}

func createHardForkExporter(
	t *testing.T,
	nodes []*integrationTests.TestProcessorNode,
) []*config.StorageConfig {
	exportConfigs := make([]*config.StorageConfig, 0, len(nodes))

	for id, node := range nodes {
		accountsDBs := make(map[state.AccountsDbIdentifier]state.AccountsAdapter)
		accountsDBs[state.UserAccountsState] = node.AccntState
		accountsDBs[state.PeerAccountsState] = node.PeerState

		node.ExportFolder = "./export" + fmt.Sprintf("%d", id)
		exportConfig := config.StorageConfig{
			Cache: config.CacheConfig{
				Size: 100000, Type: "LRU", Shards: 1,
			},
			DB: config.DBConfig{
				FilePath:          "ExportState" + fmt.Sprintf("%d", id),
				Type:              "LvlDBSerial",
				BatchDelaySeconds: 30,
				MaxBatchSize:      6,
				MaxOpenFiles:      10,
			},
		}
		exportConfigs = append(exportConfigs, &exportConfig)

		argsExportHandler := factory.ArgsExporter{
			TxSignMarshalizer: integrationTests.TestTxSignMarshalizer,
			Marshalizer:       integrationTests.TestMarshalizer,
			Hasher:            integrationTests.TestHasher,
			HeaderValidator:   node.HeaderValidator,
			Uint64Converter:   integrationTests.TestUint64Converter,
			DataPool:          node.DataPool,
			StorageService:    node.Storage,
			RequestHandler:    node.RequestHandler,
			ShardCoordinator:  node.ShardCoordinator,
			Messenger:         node.Messenger,
			ActiveAccountsDBs: accountsDBs,
			ExportFolder:      node.ExportFolder,
			ExportTriesStorageConfig: config.StorageConfig{
				Cache: config.CacheConfig{
					Size: 10000, Type: "LRU", Shards: 1,
				},
				DB: config.DBConfig{
					FilePath:          "ExportTrie" + fmt.Sprintf("%d", id),
					Type:              "MemoryDB",
					BatchDelaySeconds: 30,
					MaxBatchSize:      6,
					MaxOpenFiles:      10,
				},
			},
			ExportStateStorageConfig: exportConfig,
			WhiteListHandler:         node.WhiteListHandler,
			WhiteListerVerifiedTxs:   node.WhiteListerVerifiedTxs,
			InterceptorsContainer:    node.InterceptorsContainer,
			ExistingResolvers:        node.ResolversContainer,
			MultiSigner:              node.MultiSigner,
			NodesCoordinator:         node.NodesCoordinator,
			SingleSigner:             node.OwnAccount.SingleSigner,
			AddressPubkeyConverter:   integrationTests.TestAddressPubkeyConverter,
			BlockKeyGen:              node.OwnAccount.KeygenBlockSign,
			KeyGen:                   node.OwnAccount.KeygenTxSign,
			BlockSigner:              node.OwnAccount.BlockSingleSigner,
			HeaderSigVerifier:        node.HeaderSigVerifier,
			ChainID:                  node.ChainID,
			ValidityAttester:         node.BlockTracker,
			OutputAntifloodHandler:   &mock.NilAntifloodHandler{},
			InputAntifloodHandler:    &mock.NilAntifloodHandler{},
		}

		exportHandler, err := factory.NewExportHandlerFactory(argsExportHandler)
		assert.Nil(t, err)
		require.NotNil(t, exportHandler)

		node.ExportHandler, err = exportHandler.Create()
		assert.Nil(t, err)
		require.NotNil(t, node.ExportHandler)
	}

	return exportConfigs
}

func verifyIfNodesHaveCorrectEpoch(
	t *testing.T,
	epoch uint32,
	nodes []*integrationTests.TestProcessorNode,
) {
	for _, node := range nodes {
		currentHeader := node.BlockChain.GetCurrentBlockHeader()
		assert.Equal(t, epoch, currentHeader.GetEpoch())
	}
}

func verifyIfNodesHaveCorrectNonce(
	t *testing.T,
	nonce uint64,
	nodes []*integrationTests.TestProcessorNode,
) {
	for _, node := range nodes {
		currentHeader := node.BlockChain.GetCurrentBlockHeader()
		assert.Equal(t, nonce, currentHeader.GetNonce())
	}
}

func verifyIfAddedShardHeadersAreWithNewEpoch(
	t *testing.T,
	nodes []*integrationTests.TestProcessorNode,
) {
	for _, node := range nodes {
		if node.ShardCoordinator.SelfId() != core.MetachainShardId {
			continue
		}

		currentMetaHdr, ok := node.BlockChain.GetCurrentBlockHeader().(*block.MetaBlock)
		if !ok {
			assert.Fail(t, "metablock should have been in current block header")
		}

		shardHDrStorage := node.Storage.GetStorer(dataRetriever.BlockHeaderUnit)
		for _, shardInfo := range currentMetaHdr.ShardInfo {
			value, err := node.DataPool.Headers().GetHeaderByHash(shardInfo.HeaderHash)
			if err == nil {
				header, headerOk := value.(data.HeaderHandler)
				if !headerOk {
					assert.Fail(t, "wrong type in shard header pool")
				}

				assert.Equal(t, currentMetaHdr.GetEpoch(), header.GetEpoch())
				continue
			}

			buff, err := shardHDrStorage.Get(shardInfo.HeaderHash)
			assert.Nil(t, err)

			shardHeader := block.Header{}
			err = integrationTests.TestMarshalizer.Unmarshal(&shardHeader, buff)
			assert.Nil(t, err)
			assert.Equal(t, shardHeader.GetEpoch(), currentMetaHdr.GetEpoch())
		}
	}
}
