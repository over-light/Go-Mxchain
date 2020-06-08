package integrationTests

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	arwenConfig "github.com/ElrondNetwork/arwen-wasm-vm/config"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/spos/sposFactory"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/accumulator"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/partitioning"
	"github.com/ElrondNetwork/elrond-go/core/pubkeyConverter"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/crypto/signing"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/ed25519"
	"github.com/ElrondNetwork/elrond-go/data"
	dataBlock "github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	dataTransaction "github.com/ElrondNetwork/elrond-go/data/transaction"
	trieFactory "github.com/ElrondNetwork/elrond-go/data/trie/factory"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters/uint64ByteSlice"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/factory/containers"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/factory/resolverscontainer"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/requestHandlers"
	"github.com/ElrondNetwork/elrond-go/epochStart/metachain"
	"github.com/ElrondNetwork/elrond-go/epochStart/notifier"
	"github.com/ElrondNetwork/elrond-go/epochStart/shardchain"
	"github.com/ElrondNetwork/elrond-go/genesis/process/disabled"
	"github.com/ElrondNetwork/elrond-go/hashing/sha256"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/ElrondNetwork/elrond-go/node/external"
	"github.com/ElrondNetwork/elrond-go/node/nodeDebugFactory"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/process/block/postprocess"
	"github.com/ElrondNetwork/elrond-go/process/block/preprocess"
	"github.com/ElrondNetwork/elrond-go/process/coordinator"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	procFactory "github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/factory/interceptorscontainer"
	metaProcess "github.com/ElrondNetwork/elrond-go/process/factory/metachain"
	"github.com/ElrondNetwork/elrond-go/process/factory/shard"
	"github.com/ElrondNetwork/elrond-go/process/headerCheck"
	"github.com/ElrondNetwork/elrond-go/process/interceptors"
	"github.com/ElrondNetwork/elrond-go/process/peer"
	"github.com/ElrondNetwork/elrond-go/process/rating"
	"github.com/ElrondNetwork/elrond-go/process/rewardTransaction"
	"github.com/ElrondNetwork/elrond-go/process/scToProtocol"
	"github.com/ElrondNetwork/elrond-go/process/smartContract"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/builtInFunctions"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/hooks"
	"github.com/ElrondNetwork/elrond-go/process/track"
	"github.com/ElrondNetwork/elrond-go/process/transaction"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/ElrondNetwork/elrond-go/storage/timecache"
	"github.com/ElrondNetwork/elrond-go/update"
	"github.com/ElrondNetwork/elrond-go/vm/systemSmartContracts/defaults"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm/iele/elrond/node/endpoint"
	"github.com/pkg/errors"
)

// TestHasher represents a Sha256 hasher
var TestHasher = sha256.Sha256{}

// TestMarshalizer represents the main marshalizer
var TestMarshalizer = &marshal.GogoProtoMarshalizer{}

// TestVmMarshalizer represents the marshalizer used in vm communication
var TestVmMarshalizer = &marshal.JsonMarshalizer{}

// TestTxSignMarshalizer represents the marshalizer used in vm communication
var TestTxSignMarshalizer = &marshal.TxJsonMarshalizer{}

// TestAddressPubkeyConverter represents an address public key converter
var TestAddressPubkeyConverter, _ = pubkeyConverter.NewBech32PubkeyConverter(32)

// TestValidatorPubkeyConverter represents an address public key converter
var TestValidatorPubkeyConverter, _ = pubkeyConverter.NewHexPubkeyConverter(96)

// TestMultiSig represents a mock multisig
var TestMultiSig = mock.NewMultiSigner(1)

// TestKeyGenForAccounts represents a mock key generator for balances
var TestKeyGenForAccounts = signing.NewKeyGenerator(ed25519.NewEd25519())

// TestUint64Converter represents an uint64 to byte slice converter
var TestUint64Converter = uint64ByteSlice.NewBigEndianConverter()

// TestBlockSizeThrottler represents a block size throttler used in adaptive block size computation
var TestBlockSizeThrottler = &mock.BlockSizeThrottlerStub{}

// TestBlockSizeComputation represents a block size computation handler
var TestBlockSizeComputationHandler, _ = preprocess.NewBlockSizeComputation(TestMarshalizer, TestBlockSizeThrottler, uint32(core.MegabyteSize*90/100))

// TestBalanceComputationHandler represents a balance computation handler
var TestBalanceComputationHandler, _ = preprocess.NewBalanceComputation()

// MinTxGasPrice defines minimum gas price required by a transaction
var MinTxGasPrice = uint64(10)

// MinTxGasLimit defines minimum gas limit required by a transaction
var MinTxGasLimit = uint64(1_000)

// MaxGasLimitPerBlock defines maximum gas limit allowed per one block
const MaxGasLimitPerBlock = uint64(3_000_000)

const maxTxNonceDeltaAllowed = 8000
const minConnectedPeers = 0

// OpGasValueForMockVm represents the gas value that it consumed by each operation called on the mock VM
// By operation, we mean each go function that is called on the VM implementation
const OpGasValueForMockVm = uint64(50)

// TimeSpanForBadHeaders is the expiry time for an added block header hash
var TimeSpanForBadHeaders = time.Second * 30

// roundDuration defines the duration of the round
const roundDuration = 5 * time.Second

// ChainID is the chain ID identifier used in integration tests, processing nodes
var ChainID = []byte("integration tests chain ID")

// SoftwareVersion is the software version identifier used in integration tests, processing nodes
var SoftwareVersion = []byte("intT")

var testCommunityAddress = "erd1932eft30w753xyvme8d49qejgkjc09n5e49w4mwdjtm0neld797su0dlxp"

// sizeCheckDelta the maximum allowed bufer overhead (p2p unmarshalling)
const sizeCheckDelta = 100

const stateCheckpointModulus = 100

// TestKeyPair holds a pair of private/public Keys
type TestKeyPair struct {
	Sk crypto.PrivateKey
	Pk crypto.PublicKey
}

//CryptoParams holds crypto parametres
type CryptoParams struct {
	KeyGen       crypto.KeyGenerator
	Keys         map[uint32][]*TestKeyPair
	SingleSigner crypto.SingleSigner
	TxKeyGen     crypto.KeyGenerator
	TxKeys       map[uint32][]*TestKeyPair
}

// TestProcessorNode represents a container type of class used in integration tests
// with all its fields exported
type TestProcessorNode struct {
	ShardCoordinator sharding.Coordinator
	NodesCoordinator sharding.NodesCoordinator
	NodesSetup       sharding.GenesisNodesSetupHandler
	Messenger        p2p.Messenger

	OwnAccount *TestWalletAccount
	NodeKeys   *TestKeyPair

	ExportFolder        string
	DataPool            dataRetriever.PoolsHolder
	Storage             dataRetriever.StorageService
	PeerState           state.AccountsAdapter
	AccntState          state.AccountsAdapter
	TrieStorageManagers map[string]data.StorageManager
	TrieContainer       state.TriesHolder
	BlockChain          data.ChainHandler
	GenesisBlocks       map[uint32]data.HeaderHandler

	EconomicsData *economics.TestEconomicsData
	RatingsData   *rating.RatingsData

	BlockBlackListHandler process.BlackListHandler
	HeaderValidator       process.HeaderConstructionValidator
	BlockTracker          process.BlockTracker
	InterceptorsContainer process.InterceptorsContainer
	ResolversContainer    dataRetriever.ResolversContainer
	ResolverFinder        dataRetriever.ResolversFinder
	RequestHandler        process.RequestHandler

	InterimProcContainer   process.IntermediateProcessorContainer
	TxProcessor            process.TransactionProcessor
	TxCoordinator          process.TransactionCoordinator
	ScrForwarder           process.IntermediateTransactionHandler
	BlockchainHook         *hooks.BlockChainHookImpl
	VMContainer            process.VirtualMachinesContainer
	ArgsParser             process.ArgumentsParser
	ScProcessor            process.SmartContractProcessor
	RewardsProcessor       process.RewardTransactionProcessor
	PreProcessorsContainer process.PreProcessorsContainer
	GasHandler             process.GasHandler
	FeeAccumulator         process.TransactionFeeHandler

	ForkDetector             process.ForkDetector
	BlockProcessor           process.BlockProcessor
	BroadcastMessenger       consensus.BroadcastMessenger
	MiniblocksProvider       process.MiniBlockProvider
	Bootstrapper             TestBootstrapper
	Rounder                  *mock.RounderMock
	BootstrapStorer          *mock.BoostrapStorerMock
	StorageBootstrapper      *mock.StorageBootstrapperMock
	RequestedItemsHandler    dataRetriever.RequestedItemsHandler
	WhiteListHandler         process.WhiteListHandler
	WhiteListerVerifiedTxs   process.WhiteListHandler
	NetworkShardingCollector consensus.NetworkShardingCollector

	EpochStartTrigger  TestEpochStartTrigger
	EpochStartNotifier notifier.EpochStartNotifier

	MultiSigner             crypto.MultiSigner
	HeaderSigVerifier       process.InterceptedHeaderSigVerifier
	HeaderIntegrityVerifier process.InterceptedHeaderIntegrityVerifier

	ValidatorStatisticsProcessor process.ValidatorStatisticsProcessor
	Rater                        sharding.PeerAccountListAndRatingHandler

	//Node is used to call the functionality already implemented in it
	Node           *node.Node
	SCQueryService external.SCQueryService

	CounterHdrRecv       int32
	CounterMbRecv        int32
	CounterTxRecv        int32
	CounterMetaRcv       int32
	ReceivedTransactions sync.Map

	InitialNodes []*sharding.InitialNode

	ChainID []byte

	ExportHandler update.ExportHandler
	WaitTime      time.Duration
}

// CreatePkBytes creates 'numShards' public key-like byte slices
func CreatePkBytes(numShards uint32) map[uint32][]byte {
	pk := []byte("afafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafafaf")
	pksbytes := make(map[uint32][]byte, numShards+1)
	for i := uint32(0); i < numShards; i++ {
		pksbytes[i] = make([]byte, 128)
		pksbytes[i] = pk
		pksbytes[i][0] = byte(i)
	}

	pksbytes[core.MetachainShardId] = make([]byte, 128)
	pksbytes[core.MetachainShardId] = pk
	pksbytes[core.MetachainShardId][0] = byte(numShards)

	return pksbytes
}

// NewTestProcessorNode returns a new TestProcessorNode instance with a libp2p messenger
func NewTestProcessorNode(
	maxShards uint32,
	nodeShardId uint32,
	txSignPrivKeyShardId uint32,
	initialNodeAddr string,
) *TestProcessorNode {

	shardCoordinator, _ := sharding.NewMultiShardCoordinator(maxShards, nodeShardId)

	kg := &mock.KeyGenMock{}
	sk, pk := kg.GeneratePair()

	pksBytes := CreatePkBytes(maxShards)
	address := make([]byte, 32)
	address = []byte("afafafafafafafafafafafafafafafaf")

	nodesSetup := &mock.NodesSetupStub{
		InitialNodesInfoCalled: func() (m map[uint32][]sharding.GenesisNodeInfoHandler, m2 map[uint32][]sharding.GenesisNodeInfoHandler) {
			oneMap := make(map[uint32][]sharding.GenesisNodeInfoHandler)
			for i := uint32(0); i < maxShards; i++ {
				oneMap[i] = append(oneMap[i], mock.NewNodeInfo(address, pksBytes[i], i))
			}
			oneMap[core.MetachainShardId] = append(oneMap[core.MetachainShardId], mock.NewNodeInfo(address, pksBytes[core.MetachainShardId], core.MetachainShardId))
			return oneMap, nil
		},
		InitialNodesInfoForShardCalled: func(shardId uint32) (handlers []sharding.GenesisNodeInfoHandler, handlers2 []sharding.GenesisNodeInfoHandler, err error) {
			list := make([]sharding.GenesisNodeInfoHandler, 0)
			list = append(list, mock.NewNodeInfo(address, pksBytes[shardId], shardId))
			return list, nil, nil
		},
	}
	nodesCoordinator := &mock.NodesCoordinatorMock{
		ComputeValidatorsGroupCalled: func(randomness []byte, round uint64, shardId uint32, epoch uint32) (validators []sharding.Validator, err error) {
			v, _ := sharding.NewValidator(pksBytes[shardId], 1, defaultChancesSelection)
			return []sharding.Validator{v}, nil
		},
		GetAllValidatorsPublicKeysCalled: func() (map[uint32][][]byte, error) {
			keys := make(map[uint32][][]byte)
			for shardID := uint32(0); shardID < maxShards; shardID++ {
				keys[shardID] = append(keys[shardID], pksBytes[shardID])
			}

			shardID := core.MetachainShardId
			keys[shardID] = append(keys[shardID], pksBytes[shardID])

			return keys, nil
		},
		GetValidatorWithPublicKeyCalled: func(publicKey []byte) (sharding.Validator, uint32, error) {
			validator, _ := sharding.NewValidator(publicKey, defaultChancesSelection, 1)
			return validator, 0, nil
		},
	}

	messenger := CreateMessengerWithKadDht(initialNodeAddr)
	headerIntegrityVerifier, _ := headerCheck.NewHeaderIntegrityVerifier(ChainID)
	tpn := &TestProcessorNode{
		ShardCoordinator:        shardCoordinator,
		Messenger:               messenger,
		NodesCoordinator:        nodesCoordinator,
		HeaderSigVerifier:       &mock.HeaderSigVerifierStub{},
		HeaderIntegrityVerifier: headerIntegrityVerifier,
		ChainID:                 ChainID,
		NodesSetup:              nodesSetup,
	}

	tpn.NodeKeys = &TestKeyPair{
		Sk: sk,
		Pk: pk,
	}
	tpn.MultiSigner = TestMultiSig
	tpn.OwnAccount = CreateTestWalletAccount(shardCoordinator, txSignPrivKeyShardId)
	tpn.initDataPools()
	tpn.initTestNode()

	tpn.StorageBootstrapper = &mock.StorageBootstrapperMock{}
	tpn.BootstrapStorer = &mock.BoostrapStorerMock{}

	return tpn
}

// NewTestProcessorNodeWithCustomDataPool returns a new TestProcessorNode instance with the given data pool
func NewTestProcessorNodeWithCustomDataPool(maxShards uint32, nodeShardId uint32, txSignPrivKeyShardId uint32, initialNodeAddr string, dPool dataRetriever.PoolsHolder) *TestProcessorNode {
	shardCoordinator, _ := sharding.NewMultiShardCoordinator(maxShards, nodeShardId)

	messenger := CreateMessengerWithKadDht(initialNodeAddr)
	_ = messenger.SetThresholdMinConnectedPeers(minConnectedPeers)
	nodesCoordinator := &mock.NodesCoordinatorMock{}
	kg := &mock.KeyGenMock{}
	sk, pk := kg.GeneratePair()

	tpn := &TestProcessorNode{
		ShardCoordinator:        shardCoordinator,
		Messenger:               messenger,
		NodesCoordinator:        nodesCoordinator,
		HeaderSigVerifier:       &mock.HeaderSigVerifierStub{},
		HeaderIntegrityVerifier: &mock.HeaderIntegrityVerifierStub{},
		ChainID:                 ChainID,
		NodesSetup:              &mock.NodesSetupStub{},
	}

	tpn.NodeKeys = &TestKeyPair{
		Sk: sk,
		Pk: pk,
	}
	tpn.MultiSigner = TestMultiSig
	tpn.OwnAccount = CreateTestWalletAccount(shardCoordinator, txSignPrivKeyShardId)

	tpn.initDataPools()
	if tpn.ShardCoordinator.SelfId() != core.MetachainShardId {
		tpn.DataPool = dPool
	}

	tpn.initTestNode()

	return tpn
}

func (tpn *TestProcessorNode) initAccountDBs() {
	trieStorageManager, _ := CreateTrieStorageManager()
	tpn.TrieContainer = state.NewDataTriesHolder()
	var stateTrie data.Trie
	tpn.AccntState, stateTrie = CreateAccountsDB(UserAccount, trieStorageManager)
	tpn.TrieContainer.Put([]byte(trieFactory.UserAccountTrie), stateTrie)

	var peerTrie data.Trie
	tpn.PeerState, peerTrie = CreateAccountsDB(ValidatorAccount, trieStorageManager)
	tpn.TrieContainer.Put([]byte(trieFactory.PeerAccountTrie), peerTrie)

	tpn.TrieStorageManagers = make(map[string]data.StorageManager)
	tpn.TrieStorageManagers[trieFactory.UserAccountTrie] = trieStorageManager
	tpn.TrieStorageManagers[trieFactory.PeerAccountTrie] = trieStorageManager
}

func (tpn *TestProcessorNode) initValidatorStatistics() {
	rater, _ := rating.NewBlockSigningRater(tpn.RatingsData)

	if check.IfNil(tpn.NodesSetup) {
		tpn.NodesSetup = &mock.NodesSetupStub{}
	}

	arguments := peer.ArgValidatorStatisticsProcessor{
		PeerAdapter:         tpn.PeerState,
		PubkeyConv:          TestValidatorPubkeyConverter,
		NodesCoordinator:    tpn.NodesCoordinator,
		ShardCoordinator:    tpn.ShardCoordinator,
		DataPool:            tpn.DataPool,
		StorageService:      tpn.Storage,
		Marshalizer:         TestMarshalizer,
		Rater:               rater,
		MaxComputableRounds: 1000,
		RewardsHandler:      tpn.EconomicsData,
		NodesSetup:          tpn.NodesSetup,
	}

	tpn.ValidatorStatisticsProcessor, _ = peer.NewValidatorStatisticsProcessor(arguments)
}

func (tpn *TestProcessorNode) initTestNode() {
	tpn.initChainHandler()
	tpn.initHeaderValidator()
	tpn.initRounder()
	tpn.NetworkShardingCollector = mock.NewNetworkShardingCollectorMock()
	tpn.initStorage()
	tpn.initAccountDBs()
	tpn.initEconomicsData()
	tpn.initRatingsData()
	tpn.initRequestedItemsHandler()
	tpn.initResolvers()
	tpn.initValidatorStatistics()

	tpn.GenesisBlocks = CreateGenesisBlocks(
		tpn.AccntState,
		tpn.PeerState,
		tpn.TrieStorageManagers,
		TestAddressPubkeyConverter,
		tpn.NodesSetup,
		tpn.ShardCoordinator,
		tpn.Storage,
		tpn.BlockChain,
		TestMarshalizer,
		TestHasher,
		TestUint64Converter,
		tpn.DataPool,
		tpn.EconomicsData.EconomicsData,
	)
	tpn.initBlockTracker()
	tpn.initInterceptors()
	tpn.initInnerProcessors()
	tpn.SCQueryService, _ = smartContract.NewSCQueryService(tpn.VMContainer, tpn.EconomicsData)
	tpn.initBlockProcessor(stateCheckpointModulus)
	tpn.BroadcastMessenger, _ = sposFactory.GetBroadcastMessenger(
		TestMarshalizer,
		tpn.Messenger,
		tpn.ShardCoordinator,
		tpn.OwnAccount.SkTxSign,
		tpn.OwnAccount.SingleSigner,
		tpn.DataPool.Headers(),
	)
	tpn.setGenesisBlock()
	tpn.initNode()
	tpn.addHandlersForCounters()
	tpn.addGenesisBlocksIntoStorage()
}

func (tpn *TestProcessorNode) initDataPools() {
	tpn.DataPool = CreateTestDataPool(nil, tpn.ShardCoordinator.SelfId())
	cacherCfg := storageUnit.CacheConfig{Capacity: 10000, Type: storageUnit.LRUCache, Shards: 1}
	cache, _ := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Capacity, cacherCfg.Shards, cacherCfg.SizeInBytes)
	tpn.WhiteListHandler, _ = interceptors.NewWhiteListDataVerifier(cache)

	cacherVerifiedCfg := storageUnit.CacheConfig{Capacity: 5000, Type: storageUnit.LRUCache, Shards: 1}
	cacheVerified, _ := storageUnit.NewCache(cacherVerifiedCfg.Type, cacherVerifiedCfg.Capacity, cacherVerifiedCfg.Shards, cacherVerifiedCfg.SizeInBytes)
	tpn.WhiteListerVerifiedTxs, _ = interceptors.NewWhiteListDataVerifier(cacheVerified)
}

func (tpn *TestProcessorNode) initStorage() {
	tpn.Storage = CreateStore(tpn.ShardCoordinator.NumberOfShards())
}

func (tpn *TestProcessorNode) initChainHandler() {
	if tpn.ShardCoordinator.SelfId() == core.MetachainShardId {
		tpn.BlockChain = CreateMetaChain()
	} else {
		tpn.BlockChain = CreateShardChain()
	}
}

func (tpn *TestProcessorNode) initEconomicsData() {
	economicsData := CreateEconomicsData()

	tpn.EconomicsData = &economics.TestEconomicsData{
		EconomicsData: economicsData,
	}
}

func (tpn *TestProcessorNode) initRatingsData() {
	if tpn.RatingsData == nil {
		tpn.RatingsData = CreateRatingsData()
	}
}

// CreateEconomicsData creates a mock EconomicsData object
func CreateEconomicsData() *economics.EconomicsData {
	maxGasLimitPerBlock := strconv.FormatUint(MaxGasLimitPerBlock, 10)
	minGasPrice := strconv.FormatUint(MinTxGasPrice, 10)
	minGasLimit := strconv.FormatUint(MinTxGasLimit, 10)

	economicsData, _ := economics.NewEconomicsData(
		&config.EconomicsConfig{
			GlobalSettings: config.GlobalSettings{
				TotalSupply:      "2000000000000000000000",
				MinimumInflation: 0,
				MaximumInflation: 0.05,
			},
			RewardsSettings: config.RewardsSettings{
				LeaderPercentage:    0.1,
				DeveloperPercentage: 0.1,
				CommunityAddress:    testCommunityAddress,
			},
			FeeSettings: config.FeeSettings{
				MaxGasLimitPerBlock:     maxGasLimitPerBlock,
				MaxGasLimitPerMetaBlock: maxGasLimitPerBlock,
				MinGasPrice:             minGasPrice,
				MinGasLimit:             minGasLimit,
				GasPerDataByte:          "1",
				DataLimitForBaseCalc:    "10000",
			},
			ValidatorSettings: config.ValidatorSettings{
				GenesisNodePrice:         "500000000",
				UnBondPeriod:             "5",
				TotalSupply:              "200000000000",
				MinStepValue:             "100000",
				AuctionEnableNonce:       "100000",
				StakeEnableNonce:         "0",
				NumRoundsWithoutBleed:    "1000",
				MaximumPercentageToBleed: "0.5",
				BleedPercentagePerRound:  "0.00001",
				UnJailValue:              "1000",
			},
		},
	)
	return economicsData
}

// CreateRatingsData creates a mock RatingsData object
func CreateRatingsData() *rating.RatingsData {
	ratingsConfig := config.RatingsConfig{
		ShardChain: config.ShardChain{
			RatingSteps: config.RatingSteps{
				ProposerValidatorImportance:    1,
				ProposerDecreaseFactor:         -4,
				ValidatorDecreaseFactor:        -4,
				ConsecutiveMissedBlocksPenalty: 1.1,
			},
		},
		MetaChain: config.MetaChain{
			RatingSteps: config.RatingSteps{
				ProposerValidatorImportance:    1,
				ProposerDecreaseFactor:         -4,
				ValidatorDecreaseFactor:        -4,
				ConsecutiveMissedBlocksPenalty: 1.1,
			},
		},
		General: config.General{
			StartRating:                     500000,
			MaxRating:                       1000000,
			MinRating:                       1,
			HoursToMaxRatingFromStartRating: 50,
			SignedBlocksThreshold:           0.025,
			SelectionChances: []*config.SelectionChance{
				{
					MaxThreshold:  0,
					ChancePercent: 5,
				},
				{
					MaxThreshold:  100000,
					ChancePercent: 0,
				},
				{
					MaxThreshold:  200000,
					ChancePercent: 16,
				},
				{
					MaxThreshold:  300000,
					ChancePercent: 17,
				},
				{
					MaxThreshold:  400000,
					ChancePercent: 18,
				},
				{
					MaxThreshold:  500000,
					ChancePercent: 19,
				},
				{
					MaxThreshold:  600000,
					ChancePercent: 20,
				},
				{
					MaxThreshold:  700000,
					ChancePercent: 21,
				},
				{
					MaxThreshold:  800000,
					ChancePercent: 22,
				},
				{
					MaxThreshold:  900000,
					ChancePercent: 23,
				},
				{
					MaxThreshold:  1000000,
					ChancePercent: 24,
				},
			},
		},
	}

	ratingDataArgs := rating.RatingsDataArg{
		Config:                   ratingsConfig,
		ShardConsensusSize:       63,
		MetaConsensusSize:        400,
		ShardMinNodes:            400,
		MetaMinNodes:             400,
		RoundDurationMiliseconds: 6000,
	}

	ratingsData, _ := rating.NewRatingsData(ratingDataArgs)
	return ratingsData
}

func (tpn *TestProcessorNode) initInterceptors() {
	var err error
	tpn.BlockBlackListHandler = timecache.NewTimeCache(TimeSpanForBadHeaders)
	if check.IfNil(tpn.EpochStartNotifier) {
		tpn.EpochStartNotifier = &mock.EpochStartNotifierStub{}
	}
	if tpn.ShardCoordinator.SelfId() == core.MetachainShardId {

		argsEpochStart := &metachain.ArgsNewMetaEpochStartTrigger{
			GenesisTime: tpn.Rounder.TimeStamp(),
			Settings: &config.EpochStartConfig{
				MinRoundsBetweenEpochs: 1000,
				RoundsPerEpoch:         10000,
			},
			Epoch:              0,
			EpochStartNotifier: tpn.EpochStartNotifier,
			Storage:            tpn.Storage,
			Marshalizer:        TestMarshalizer,
			Hasher:             TestHasher,
		}
		epochStartTrigger, _ := metachain.NewEpochStartTrigger(argsEpochStart)
		tpn.EpochStartTrigger = &metachain.TestTrigger{}
		tpn.EpochStartTrigger.SetTrigger(epochStartTrigger)
		metaIntercContFactArgs := interceptorscontainer.MetaInterceptorsContainerFactoryArgs{
			ShardCoordinator:        tpn.ShardCoordinator,
			NodesCoordinator:        tpn.NodesCoordinator,
			Messenger:               tpn.Messenger,
			Store:                   tpn.Storage,
			ProtoMarshalizer:        TestMarshalizer,
			TxSignMarshalizer:       TestTxSignMarshalizer,
			Hasher:                  TestHasher,
			MultiSigner:             TestMultiSig,
			DataPool:                tpn.DataPool,
			Accounts:                tpn.AccntState,
			AddressPubkeyConverter:  TestAddressPubkeyConverter,
			SingleSigner:            tpn.OwnAccount.SingleSigner,
			BlockSingleSigner:       tpn.OwnAccount.BlockSingleSigner,
			KeyGen:                  tpn.OwnAccount.KeygenTxSign,
			BlockKeyGen:             tpn.OwnAccount.KeygenBlockSign,
			MaxTxNonceDeltaAllowed:  maxTxNonceDeltaAllowed,
			TxFeeHandler:            tpn.EconomicsData,
			BlackList:               tpn.BlockBlackListHandler,
			HeaderSigVerifier:       tpn.HeaderSigVerifier,
			HeaderIntegrityVerifier: tpn.HeaderIntegrityVerifier,
			SizeCheckDelta:          sizeCheckDelta,
			ValidityAttester:        tpn.BlockTracker,
			EpochStartTrigger:       tpn.EpochStartTrigger,
			WhiteListHandler:        tpn.WhiteListHandler,
			WhiteListerVerifiedTxs:  tpn.WhiteListerVerifiedTxs,
			AntifloodHandler:        &mock.NilAntifloodHandler{},
			NonceConverter:          TestUint64Converter,
		}
		interceptorContainerFactory, _ := interceptorscontainer.NewMetaInterceptorsContainerFactory(metaIntercContFactArgs)

		tpn.InterceptorsContainer, err = interceptorContainerFactory.Create()
		if err != nil {
			log.Debug("interceptor container factory Create", "error", err.Error())
		}
	} else {
		argsPeerMiniBlocksSyncer := shardchain.ArgPeerMiniBlockSyncer{
			MiniBlocksPool: tpn.DataPool.MiniBlocks(),
			Requesthandler: tpn.RequestHandler,
		}
		peerMiniBlockSyncer, _ := shardchain.NewPeerMiniBlockSyncer(argsPeerMiniBlocksSyncer)
		argsShardEpochStart := &shardchain.ArgsShardEpochStartTrigger{
			Marshalizer:          TestMarshalizer,
			Hasher:               TestHasher,
			HeaderValidator:      tpn.HeaderValidator,
			Uint64Converter:      TestUint64Converter,
			DataPool:             tpn.DataPool,
			Storage:              tpn.Storage,
			RequestHandler:       tpn.RequestHandler,
			Epoch:                0,
			Validity:             1,
			Finality:             1,
			EpochStartNotifier:   tpn.EpochStartNotifier,
			PeerMiniBlocksSyncer: peerMiniBlockSyncer,
		}
		epochStartTrigger, _ := shardchain.NewEpochStartTrigger(argsShardEpochStart)
		tpn.EpochStartTrigger = &shardchain.TestTrigger{}
		tpn.EpochStartTrigger.SetTrigger(epochStartTrigger)

		shardInterContFactArgs := interceptorscontainer.ShardInterceptorsContainerFactoryArgs{
			Accounts:                tpn.AccntState,
			ShardCoordinator:        tpn.ShardCoordinator,
			NodesCoordinator:        tpn.NodesCoordinator,
			Messenger:               tpn.Messenger,
			Store:                   tpn.Storage,
			ProtoMarshalizer:        TestMarshalizer,
			TxSignMarshalizer:       TestTxSignMarshalizer,
			Hasher:                  TestHasher,
			KeyGen:                  tpn.OwnAccount.KeygenTxSign,
			BlockSignKeyGen:         tpn.OwnAccount.KeygenBlockSign,
			SingleSigner:            tpn.OwnAccount.SingleSigner,
			BlockSingleSigner:       tpn.OwnAccount.BlockSingleSigner,
			MultiSigner:             TestMultiSig,
			DataPool:                tpn.DataPool,
			AddressPubkeyConverter:  TestAddressPubkeyConverter,
			MaxTxNonceDeltaAllowed:  maxTxNonceDeltaAllowed,
			TxFeeHandler:            tpn.EconomicsData,
			BlackList:               tpn.BlockBlackListHandler,
			HeaderSigVerifier:       tpn.HeaderSigVerifier,
			HeaderIntegrityVerifier: tpn.HeaderIntegrityVerifier,
			SizeCheckDelta:          sizeCheckDelta,
			ValidityAttester:        tpn.BlockTracker,
			EpochStartTrigger:       tpn.EpochStartTrigger,
			WhiteListHandler:        tpn.WhiteListHandler,
			WhiteListerVerifiedTxs:  tpn.WhiteListerVerifiedTxs,
			AntifloodHandler:        &mock.NilAntifloodHandler{},
			NonceConverter:          TestUint64Converter,
		}
		interceptorContainerFactory, _ := interceptorscontainer.NewShardInterceptorsContainerFactory(shardInterContFactArgs)

		tpn.InterceptorsContainer, err = interceptorContainerFactory.Create()
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

func (tpn *TestProcessorNode) initResolvers() {
	dataPacker, _ := partitioning.NewSimpleDataPacker(TestMarshalizer)

	_ = tpn.Messenger.CreateTopic(core.ConsensusTopic+tpn.ShardCoordinator.CommunicationIdentifier(tpn.ShardCoordinator.SelfId()), true)

	resolverContainerFactory := resolverscontainer.FactoryArgs{
		ShardCoordinator:           tpn.ShardCoordinator,
		Messenger:                  tpn.Messenger,
		Store:                      tpn.Storage,
		Marshalizer:                TestMarshalizer,
		DataPools:                  tpn.DataPool,
		Uint64ByteSliceConverter:   TestUint64Converter,
		DataPacker:                 dataPacker,
		TriesContainer:             tpn.TrieContainer,
		SizeCheckDelta:             100,
		InputAntifloodHandler:      &mock.NilAntifloodHandler{},
		OutputAntifloodHandler:     &mock.NilAntifloodHandler{},
		NumConcurrentResolvingJobs: 10,
	}

	var err error
	if tpn.ShardCoordinator.SelfId() == core.MetachainShardId {
		resolversContainerFactory, _ := resolverscontainer.NewMetaResolversContainerFactory(resolverContainerFactory)

		tpn.ResolversContainer, err = resolversContainerFactory.Create()
		log.LogIfError(err)

		tpn.ResolverFinder, _ = containers.NewResolversFinder(tpn.ResolversContainer, tpn.ShardCoordinator)
		tpn.RequestHandler, _ = requestHandlers.NewResolverRequestHandler(
			tpn.ResolverFinder,
			tpn.RequestedItemsHandler,
			tpn.WhiteListHandler,
			100,
			tpn.ShardCoordinator.SelfId(),
			time.Second,
		)
	} else {
		resolversContainerFactory, _ := resolverscontainer.NewShardResolversContainerFactory(resolverContainerFactory)

		tpn.ResolversContainer, err = resolversContainerFactory.Create()
		log.LogIfError(err)

		tpn.ResolverFinder, _ = containers.NewResolversFinder(tpn.ResolversContainer, tpn.ShardCoordinator)
		tpn.RequestHandler, _ = requestHandlers.NewResolverRequestHandler(
			tpn.ResolverFinder,
			tpn.RequestedItemsHandler,
			tpn.WhiteListHandler,
			100,
			tpn.ShardCoordinator.SelfId(),
			time.Second,
		)
	}
}

func createAndAddIeleVM(
	vmContainer process.VirtualMachinesContainer,
	blockChainHook vmcommon.BlockchainHook,
) {
	cryptoHook := hooks.NewVMCryptoHook()
	ieleVM := endpoint.NewElrondIeleVM(factory.IELEVirtualMachine, endpoint.ElrondTestnet, blockChainHook, cryptoHook)
	_ = vmContainer.Add(factory.IELEVirtualMachine, ieleVM)
}

func (tpn *TestProcessorNode) initInnerProcessors() {
	if tpn.ShardCoordinator.SelfId() == core.MetachainShardId {
		tpn.initMetaInnerProcessors()
		return
	}

	if tpn.ValidatorStatisticsProcessor == nil {
		tpn.ValidatorStatisticsProcessor = &mock.ValidatorStatisticsProcessorStub{}
	}

	interimProcFactory, _ := shard.NewIntermediateProcessorsContainerFactory(
		tpn.ShardCoordinator,
		TestMarshalizer,
		TestHasher,
		TestAddressPubkeyConverter,
		tpn.Storage,
		tpn.DataPool,
	)

	tpn.InterimProcContainer, _ = interimProcFactory.Create()
	tpn.ScrForwarder, _ = tpn.InterimProcContainer.Get(dataBlock.SmartContractResultBlock)

	tpn.RewardsProcessor, _ = rewardTransaction.NewRewardTxProcessor(
		tpn.AccntState,
		TestAddressPubkeyConverter,
		tpn.ShardCoordinator,
	)

	gasSchedule := arwenConfig.MakeGasMapForTests()
	defaults.FillGasMapInternal(gasSchedule, 1)
	argsBuiltIn := builtInFunctions.ArgsCreateBuiltInFunctionContainer{
		GasMap:          gasSchedule,
		MapDNSAddresses: make(map[string]struct{}),
		Marshalizer:     TestMarshalizer,
	}
	builtInFuncs, _ := builtInFunctions.CreateBuiltInFunctionContainer(argsBuiltIn)

	argsHook := hooks.ArgBlockChainHook{
		Accounts:         tpn.AccntState,
		PubkeyConv:       TestAddressPubkeyConverter,
		StorageService:   tpn.Storage,
		BlockChain:       tpn.BlockChain,
		ShardCoordinator: tpn.ShardCoordinator,
		Marshalizer:      TestMarshalizer,
		Uint64Converter:  TestUint64Converter,
		BuiltInFunctions: builtInFuncs,
	}
	maxGasLimitPerBlock := uint64(0xFFFFFFFFFFFFFFFF)
	vmFactory, _ := shard.NewVMContainerFactory(
		config.VirtualMachineConfig{
			OutOfProcessEnabled: true,
			OutOfProcessConfig:  config.VirtualMachineOutOfProcessConfig{MaxLoopTime: 1000},
		},
		maxGasLimitPerBlock,
		gasSchedule,
		argsHook,
	)

	var err error
	tpn.VMContainer, err = vmFactory.Create()
	if err != nil {
		panic(err)
	}

	tpn.BlockchainHook, _ = vmFactory.BlockChainHookImpl().(*hooks.BlockChainHookImpl)
	createAndAddIeleVM(tpn.VMContainer, tpn.BlockchainHook)

	mockVM, _ := mock.NewOneSCExecutorMockVM(tpn.BlockchainHook, TestHasher)
	mockVM.GasForOperation = OpGasValueForMockVm
	_ = tpn.VMContainer.Add(procFactory.InternalTestingVM, mockVM)

	tpn.FeeAccumulator, _ = postprocess.NewFeeAccumulator()
	tpn.ArgsParser = vmcommon.NewAtArgumentParser()
	argsTxTypeHandler := coordinator.ArgNewTxTypeHandler{
		PubkeyConverter:  TestAddressPubkeyConverter,
		ShardCoordinator: tpn.ShardCoordinator,
		BuiltInFuncNames: builtInFuncs.Keys(),
		ArgumentParser:   tpn.ArgsParser,
	}
	txTypeHandler, _ := coordinator.NewTxTypeHandler(argsTxTypeHandler)
	tpn.GasHandler, _ = preprocess.NewGasComputation(tpn.EconomicsData, txTypeHandler)

	argsNewScProcessor := smartContract.ArgsNewSmartContractProcessor{
		VmContainer:      tpn.VMContainer,
		ArgsParser:       tpn.ArgsParser,
		Hasher:           TestHasher,
		Marshalizer:      TestMarshalizer,
		AccountsDB:       tpn.AccntState,
		TempAccounts:     vmFactory.BlockChainHookImpl(),
		PubkeyConv:       TestAddressPubkeyConverter,
		Coordinator:      tpn.ShardCoordinator,
		ScrForwarder:     tpn.ScrForwarder,
		TxFeeHandler:     tpn.FeeAccumulator,
		EconomicsFee:     tpn.EconomicsData,
		TxTypeHandler:    txTypeHandler,
		GasHandler:       tpn.GasHandler,
		BuiltInFunctions: tpn.BlockchainHook.GetBuiltInFunctions(),
		TxLogsProcessor:  &mock.TxLogsProcessorStub{},
	}
	tpn.ScProcessor, _ = smartContract.NewSmartContractProcessor(argsNewScProcessor)

	receiptsHandler, _ := tpn.InterimProcContainer.Get(dataBlock.ReceiptBlock)
	badBlocskHandler, _ := tpn.InterimProcContainer.Get(dataBlock.InvalidBlock)
	tpn.TxProcessor, _ = transaction.NewTxProcessor(
		tpn.AccntState,
		TestHasher,
		TestAddressPubkeyConverter,
		TestMarshalizer,
		tpn.ShardCoordinator,
		tpn.ScProcessor,
		tpn.FeeAccumulator,
		txTypeHandler,
		tpn.EconomicsData,
		receiptsHandler,
		badBlocskHandler,
	)

	fact, _ := shard.NewPreProcessorsContainerFactory(
		tpn.ShardCoordinator,
		tpn.Storage,
		TestMarshalizer,
		TestHasher,
		tpn.DataPool,
		TestAddressPubkeyConverter,
		tpn.AccntState,
		tpn.RequestHandler,
		tpn.TxProcessor,
		tpn.ScProcessor,
		tpn.ScProcessor.(process.SmartContractResultProcessor),
		tpn.RewardsProcessor,
		tpn.EconomicsData,
		tpn.GasHandler,
		tpn.BlockTracker,
		TestBlockSizeComputationHandler,
		TestBalanceComputationHandler,
	)
	tpn.PreProcessorsContainer, _ = fact.Create()

	tpn.TxCoordinator, _ = coordinator.NewTransactionCoordinator(
		TestHasher,
		TestMarshalizer,
		tpn.ShardCoordinator,
		tpn.AccntState,
		tpn.DataPool.MiniBlocks(),
		tpn.RequestHandler,
		tpn.PreProcessorsContainer,
		tpn.InterimProcContainer,
		tpn.GasHandler,
		tpn.FeeAccumulator,
		TestBlockSizeComputationHandler,
		TestBalanceComputationHandler,
	)
}

func (tpn *TestProcessorNode) initMetaInnerProcessors() {
	interimProcFactory, _ := metaProcess.NewIntermediateProcessorsContainerFactory(
		tpn.ShardCoordinator,
		TestMarshalizer,
		TestHasher,
		TestAddressPubkeyConverter,
		tpn.Storage,
		tpn.DataPool,
	)

	tpn.InterimProcContainer, _ = interimProcFactory.Create()
	tpn.ScrForwarder, _ = tpn.InterimProcContainer.Get(dataBlock.SmartContractResultBlock)

	builtInFuncs := builtInFunctions.NewBuiltInFunctionContainer()
	argsHook := hooks.ArgBlockChainHook{
		Accounts:         tpn.AccntState,
		PubkeyConv:       TestAddressPubkeyConverter,
		StorageService:   tpn.Storage,
		BlockChain:       tpn.BlockChain,
		ShardCoordinator: tpn.ShardCoordinator,
		Marshalizer:      TestMarshalizer,
		Uint64Converter:  TestUint64Converter,
		BuiltInFunctions: builtInFuncs,
	}
	gasSchedule := make(map[string]map[string]uint64)
	defaults.FillGasMapInternal(gasSchedule, 1)
	signVerifer, _ := disabled.NewMessageSignVerifier(&mock.KeyGenMock{})
	vmFactory, _ := metaProcess.NewVMContainerFactory(
		argsHook,
		tpn.EconomicsData.EconomicsData,
		signVerifer,
		gasSchedule,
		tpn.NodesSetup,
		TestHasher,
		TestMarshalizer,
		&config.SystemSmartContractsConfig{
			ESDTSystemSCConfig: config.ESDTSystemSCConfig{
				BaseIssuingCost: "1000",
				OwnerAddress:    "aaaaaa",
			},
		},
		tpn.PeerState,
	)

	tpn.VMContainer, _ = vmFactory.Create()
	tpn.BlockchainHook, _ = vmFactory.BlockChainHookImpl().(*hooks.BlockChainHookImpl)

	tpn.addMockVm(tpn.BlockchainHook)

	tpn.FeeAccumulator, _ = postprocess.NewFeeAccumulator()
	tpn.ArgsParser = vmcommon.NewAtArgumentParser()
	argsTxTypeHandler := coordinator.ArgNewTxTypeHandler{
		PubkeyConverter:  TestAddressPubkeyConverter,
		ShardCoordinator: tpn.ShardCoordinator,
		BuiltInFuncNames: builtInFuncs.Keys(),
		ArgumentParser:   tpn.ArgsParser,
	}
	txTypeHandler, _ := coordinator.NewTxTypeHandler(argsTxTypeHandler)
	tpn.GasHandler, _ = preprocess.NewGasComputation(tpn.EconomicsData, txTypeHandler)
	argsNewScProcessor := smartContract.ArgsNewSmartContractProcessor{
		VmContainer:      tpn.VMContainer,
		ArgsParser:       tpn.ArgsParser,
		Hasher:           TestHasher,
		Marshalizer:      TestMarshalizer,
		AccountsDB:       tpn.AccntState,
		TempAccounts:     vmFactory.BlockChainHookImpl(),
		PubkeyConv:       TestAddressPubkeyConverter,
		Coordinator:      tpn.ShardCoordinator,
		ScrForwarder:     tpn.ScrForwarder,
		TxFeeHandler:     tpn.FeeAccumulator,
		EconomicsFee:     tpn.EconomicsData,
		TxTypeHandler:    txTypeHandler,
		GasHandler:       tpn.GasHandler,
		BuiltInFunctions: tpn.BlockchainHook.GetBuiltInFunctions(),
		TxLogsProcessor:  &mock.TxLogsProcessorStub{},
	}
	scProcessor, _ := smartContract.NewSmartContractProcessor(argsNewScProcessor)
	tpn.ScProcessor = scProcessor
	tpn.TxProcessor, _ = transaction.NewMetaTxProcessor(
		TestHasher,
		TestMarshalizer,
		tpn.AccntState,
		TestAddressPubkeyConverter,
		tpn.ShardCoordinator,
		tpn.ScProcessor,
		txTypeHandler,
		tpn.EconomicsData,
	)

	fact, _ := metaProcess.NewPreProcessorsContainerFactory(
		tpn.ShardCoordinator,
		tpn.Storage,
		TestMarshalizer,
		TestHasher,
		tpn.DataPool,
		tpn.AccntState,
		tpn.RequestHandler,
		tpn.TxProcessor,
		scProcessor,
		tpn.EconomicsData.EconomicsData,
		tpn.GasHandler,
		tpn.BlockTracker,
		TestAddressPubkeyConverter,
		TestBlockSizeComputationHandler,
		TestBalanceComputationHandler,
	)
	tpn.PreProcessorsContainer, _ = fact.Create()

	tpn.TxCoordinator, _ = coordinator.NewTransactionCoordinator(
		TestHasher,
		TestMarshalizer,
		tpn.ShardCoordinator,
		tpn.AccntState,
		tpn.DataPool.MiniBlocks(),
		tpn.RequestHandler,
		tpn.PreProcessorsContainer,
		tpn.InterimProcContainer,
		tpn.GasHandler,
		tpn.FeeAccumulator,
		TestBlockSizeComputationHandler,
		TestBalanceComputationHandler,
	)
}

func (tpn *TestProcessorNode) addMockVm(blockchainHook vmcommon.BlockchainHook) {
	mockVM, _ := mock.NewOneSCExecutorMockVM(blockchainHook, TestHasher)
	mockVM.GasForOperation = OpGasValueForMockVm

	_ = tpn.VMContainer.Add(factory.InternalTestingVM, mockVM)
}

func (tpn *TestProcessorNode) initBlockProcessor(stateCheckpointModulus uint) {
	var err error

	tpn.ForkDetector = &mock.ForkDetectorStub{
		AddHeaderCalled: func(header data.HeaderHandler, hash []byte, state process.BlockHeaderState, selfNotarizedHeaders []data.HeaderHandler, selfNotarizedHeadersHashes [][]byte) error {
			return nil
		},
		GetHighestFinalBlockNonceCalled: func() uint64 {
			return 0
		},
		ProbableHighestNonceCalled: func() uint64 {
			return 0
		},
		GetHighestFinalBlockHashCalled: func() []byte {
			return nil
		},
	}

	accountsDb := make(map[state.AccountsDbIdentifier]state.AccountsAdapter)
	accountsDb[state.UserAccountsState] = tpn.AccntState
	accountsDb[state.PeerAccountsState] = tpn.PeerState

	argumentsBase := block.ArgBaseProcessor{
		AccountsDB:       accountsDb,
		ForkDetector:     tpn.ForkDetector,
		Hasher:           TestHasher,
		Marshalizer:      TestMarshalizer,
		Store:            tpn.Storage,
		ShardCoordinator: tpn.ShardCoordinator,
		NodesCoordinator: tpn.NodesCoordinator,
		FeeHandler:       tpn.FeeAccumulator,
		Uint64Converter:  TestUint64Converter,
		RequestHandler:   tpn.RequestHandler,
		Core:             nil,
		BlockChainHook:   tpn.BlockchainHook,
		HeaderValidator:  tpn.HeaderValidator,
		Rounder:          tpn.Rounder,
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
	}

	if check.IfNil(tpn.EpochStartNotifier) {
		tpn.EpochStartNotifier = &mock.EpochStartNotifierStub{}
	}

	if tpn.ShardCoordinator.SelfId() == core.MetachainShardId {
		argsEpochStart := &metachain.ArgsNewMetaEpochStartTrigger{
			GenesisTime: argumentsBase.Rounder.TimeStamp(),
			Settings: &config.EpochStartConfig{
				MinRoundsBetweenEpochs: 1000,
				RoundsPerEpoch:         10000,
			},
			Epoch:              0,
			EpochStartNotifier: tpn.EpochStartNotifier,
			Storage:            tpn.Storage,
			Marshalizer:        TestMarshalizer,
			Hasher:             TestHasher,
		}
		epochStartTrigger, _ := metachain.NewEpochStartTrigger(argsEpochStart)
		tpn.EpochStartTrigger = &metachain.TestTrigger{}
		tpn.EpochStartTrigger.SetTrigger(epochStartTrigger)

		argumentsBase.EpochStartTrigger = tpn.EpochStartTrigger
		argumentsBase.TxCoordinator = tpn.TxCoordinator

		argsStakingToPeer := scToProtocol.ArgStakingToPeer{
			PubkeyConv:       TestValidatorPubkeyConverter,
			Hasher:           TestHasher,
			ProtoMarshalizer: TestMarshalizer,
			VmMarshalizer:    TestVmMarshalizer,
			PeerState:        tpn.PeerState,
			BaseState:        tpn.AccntState,
			ArgParser:        tpn.ArgsParser,
			CurrTxs:          tpn.DataPool.CurrentBlockTxs(),
			ScQuery:          tpn.SCQueryService,
			RatingsData:      tpn.RatingsData,
		}
		scToProtocolInstance, _ := scToProtocol.NewStakingToPeer(argsStakingToPeer)

		argsEpochStartData := metachain.ArgsNewEpochStartData{
			Marshalizer:       TestMarshalizer,
			Hasher:            TestHasher,
			Store:             tpn.Storage,
			DataPool:          tpn.DataPool,
			BlockTracker:      tpn.BlockTracker,
			ShardCoordinator:  tpn.ShardCoordinator,
			EpochStartTrigger: tpn.EpochStartTrigger,
			RequestHandler:    tpn.RequestHandler,
		}
		epochStartDataCreator, _ := metachain.NewEpochStartData(argsEpochStartData)

		argsEpochEconomics := metachain.ArgsNewEpochEconomics{
			Marshalizer:         TestMarshalizer,
			Hasher:              TestHasher,
			Store:               tpn.Storage,
			ShardCoordinator:    tpn.ShardCoordinator,
			NodesConfigProvider: tpn.NodesCoordinator,
			RewardsHandler:      tpn.EconomicsData,
			RoundTime:           tpn.Rounder,
		}
		epochEconomics, _ := metachain.NewEndOfEpochEconomicsDataCreator(argsEpochEconomics)

		rewardsStorage := tpn.Storage.GetStorer(dataRetriever.RewardTransactionUnit)
		miniBlockStorage := tpn.Storage.GetStorer(dataRetriever.MiniBlockUnit)
		argsEpochRewards := metachain.ArgsNewRewardsCreator{
			ShardCoordinator: tpn.ShardCoordinator,
			PubkeyConverter:  TestAddressPubkeyConverter,
			RewardsStorage:   rewardsStorage,
			MiniBlockStorage: miniBlockStorage,
			Hasher:           TestHasher,
			Marshalizer:      TestMarshalizer,
			DataPool:         tpn.DataPool,
			CommunityAddress: testCommunityAddress,
		}
		epochStartRewards, _ := metachain.NewEpochStartRewardsCreator(argsEpochRewards)

		argsEpochValidatorInfo := metachain.ArgsNewValidatorInfoCreator{
			ShardCoordinator: tpn.ShardCoordinator,
			MiniBlockStorage: miniBlockStorage,
			Hasher:           TestHasher,
			Marshalizer:      TestMarshalizer,
			DataPool:         tpn.DataPool,
		}

		epochStartValidatorInfo, _ := metachain.NewValidatorInfoCreator(argsEpochValidatorInfo)

		arguments := block.ArgMetaProcessor{
			ArgBaseProcessor:             argumentsBase,
			SCDataGetter:                 tpn.SCQueryService,
			SCToProtocol:                 scToProtocolInstance,
			PendingMiniBlocksHandler:     &mock.PendingMiniBlocksHandlerStub{},
			EpochEconomics:               epochEconomics,
			EpochStartDataCreator:        epochStartDataCreator,
			EpochRewardsCreator:          epochStartRewards,
			EpochValidatorInfoCreator:    epochStartValidatorInfo,
			ValidatorStatisticsProcessor: tpn.ValidatorStatisticsProcessor,
		}

		tpn.BlockProcessor, err = block.NewMetaProcessor(arguments)
	} else {
		if check.IfNil(tpn.EpochStartTrigger) {
			argsPeerMiniBlocksSyncer := shardchain.ArgPeerMiniBlockSyncer{
				MiniBlocksPool: tpn.DataPool.MiniBlocks(),
				Requesthandler: tpn.RequestHandler,
			}
			peerMiniBlocksSyncer, _ := shardchain.NewPeerMiniBlockSyncer(argsPeerMiniBlocksSyncer)
			argsShardEpochStart := &shardchain.ArgsShardEpochStartTrigger{
				Marshalizer:          TestMarshalizer,
				Hasher:               TestHasher,
				HeaderValidator:      tpn.HeaderValidator,
				Uint64Converter:      TestUint64Converter,
				DataPool:             tpn.DataPool,
				Storage:              tpn.Storage,
				RequestHandler:       tpn.RequestHandler,
				Epoch:                0,
				Validity:             1,
				Finality:             1,
				EpochStartNotifier:   tpn.EpochStartNotifier,
				PeerMiniBlocksSyncer: peerMiniBlocksSyncer,
			}
			epochStartTrigger, _ := shardchain.NewEpochStartTrigger(argsShardEpochStart)
			tpn.EpochStartTrigger = &shardchain.TestTrigger{}
			tpn.EpochStartTrigger.SetTrigger(epochStartTrigger)
		}

		argumentsBase.EpochStartTrigger = tpn.EpochStartTrigger
		argumentsBase.BlockChainHook = tpn.BlockchainHook
		argumentsBase.TxCoordinator = tpn.TxCoordinator
		arguments := block.ArgShardProcessor{
			ArgBaseProcessor: argumentsBase,
		}

		tpn.BlockProcessor, err = block.NewShardProcessor(arguments)
	}

	if err != nil {
		log.Error("error creating blockprocessor", "error", err.Error())
	}
}

func (tpn *TestProcessorNode) setGenesisBlock() {
	genesisBlock := tpn.GenesisBlocks[tpn.ShardCoordinator.SelfId()]
	_ = tpn.BlockChain.SetGenesisHeader(genesisBlock)
	hash, _ := core.CalculateHash(TestMarshalizer, TestHasher, genesisBlock)
	tpn.BlockChain.SetGenesisHeaderHash(hash)
	log.Info("set genesis",
		"shard ID", tpn.ShardCoordinator.SelfId(),
		"hash", hex.EncodeToString(hash),
	)
}

func (tpn *TestProcessorNode) initNode() {
	var err error

	txAccumulator, _ := accumulator.NewTimeAccumulator(time.Millisecond*10, time.Millisecond)
	tpn.Node, err = node.NewNode(
		node.WithMessenger(tpn.Messenger),
		node.WithInternalMarshalizer(TestMarshalizer, 100),
		node.WithVmMarshalizer(TestVmMarshalizer),
		node.WithTxSignMarshalizer(TestTxSignMarshalizer),
		node.WithHasher(TestHasher),
		node.WithAddressPubkeyConverter(TestAddressPubkeyConverter),
		node.WithValidatorPubkeyConverter(TestValidatorPubkeyConverter),
		node.WithAccountsAdapter(tpn.AccntState),
		node.WithKeyGen(tpn.OwnAccount.KeygenTxSign),
		node.WithKeyGenForAccounts(TestKeyGenForAccounts),
		node.WithTxFeeHandler(tpn.EconomicsData),
		node.WithShardCoordinator(tpn.ShardCoordinator),
		node.WithNodesCoordinator(tpn.NodesCoordinator),
		node.WithBlockChain(tpn.BlockChain),
		node.WithUint64ByteSliceConverter(TestUint64Converter),
		node.WithMultiSigner(tpn.MultiSigner),
		node.WithSingleSigner(tpn.OwnAccount.SingleSigner),
		node.WithPrivKey(tpn.NodeKeys.Sk),
		node.WithPubKey(tpn.NodeKeys.Pk),
		node.WithInterceptorsContainer(tpn.InterceptorsContainer),
		node.WithHeaderIntegrityVerifier(tpn.HeaderIntegrityVerifier),
		node.WithResolversFinder(tpn.ResolverFinder),
		node.WithBlockProcessor(tpn.BlockProcessor),
		node.WithTxSingleSigner(tpn.OwnAccount.SingleSigner),
		node.WithDataStore(tpn.Storage),
		node.WithSyncer(&mock.SyncTimerMock{}),
		node.WithBlockBlackListHandler(tpn.BlockBlackListHandler),
		node.WithPeerBlackListHandler(&mock.PeerBlackListHandlerStub{}),
		node.WithDataPool(tpn.DataPool),
		node.WithNetworkShardingCollector(tpn.NetworkShardingCollector),
		node.WithTxAccumulator(txAccumulator),
		node.WithHardforkTrigger(&mock.HardforkTriggerStub{}),
	)
	log.LogIfError(err)

	err = nodeDebugFactory.CreateInterceptedDebugHandler(
		tpn.Node,
		tpn.InterceptorsContainer,
		tpn.ResolverFinder,
		config.InterceptorResolverDebugConfig{
			Enabled:                    true,
			CacheSize:                  1000,
			EnablePrint:                true,
			IntervalAutoPrintInSeconds: 1,
			NumRequestsThreshold:       1,
			NumResolveFailureThreshold: 1,
			DebugLineExpiration:        1000,
		},
	)
	log.LogIfError(err)
}

// SendTransaction can send a transaction (it does the dispatching)
func (tpn *TestProcessorNode) SendTransaction(tx *dataTransaction.Transaction) (string, error) {
	tx, txHash, err := tpn.Node.CreateTransaction(
		tx.Nonce,
		tx.Value.String(),
		TestAddressPubkeyConverter.Encode(tx.RcvAddr),
		TestAddressPubkeyConverter.Encode(tx.SndAddr),
		tx.GasPrice,
		tx.GasLimit,
		string(tx.Data),
		hex.EncodeToString(tx.Signature),
	)
	if err != nil {
		return "", err
	}

	err = tpn.Node.ValidateTransaction(tx)
	if err != nil {
		return "", err
	}

	_, err = tpn.Node.SendBulkTransactions([]*dataTransaction.Transaction{tx})
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(txHash), err
}

func (tpn *TestProcessorNode) addHandlersForCounters() {
	hdrHandlers := func(header data.HeaderHandler, key []byte) {
		atomic.AddInt32(&tpn.CounterHdrRecv, 1)
	}

	if tpn.ShardCoordinator.SelfId() == core.MetachainShardId {
		tpn.DataPool.Headers().RegisterHandler(hdrHandlers)
	} else {
		txHandler := func(key []byte, value interface{}) {
			tx, _ := tpn.DataPool.Transactions().SearchFirstData(key)
			tpn.ReceivedTransactions.Store(string(key), tx)
			atomic.AddInt32(&tpn.CounterTxRecv, 1)
		}
		mbHandlers := func(key []byte, value interface{}) {
			atomic.AddInt32(&tpn.CounterMbRecv, 1)
		}

		tpn.DataPool.UnsignedTransactions().RegisterHandler(txHandler)
		tpn.DataPool.Transactions().RegisterHandler(txHandler)
		tpn.DataPool.RewardTransactions().RegisterHandler(txHandler)
		tpn.DataPool.Headers().RegisterHandler(hdrHandlers)
		tpn.DataPool.MiniBlocks().RegisterHandler(mbHandlers, core.UniqueIdentifier())
	}
}

// StartSync calls Bootstrapper.StartSync. Errors if bootstrapper is not set
func (tpn *TestProcessorNode) StartSync() error {
	if tpn.Bootstrapper == nil {
		return errors.New("no bootstrapper available")
	}

	tpn.Bootstrapper.StartSyncingBlocks()

	return nil
}

// LoadTxSignSkBytes alters the already generated sk/pk pair
func (tpn *TestProcessorNode) LoadTxSignSkBytes(skBytes []byte) {
	tpn.OwnAccount.LoadTxSignSkBytes(skBytes)
}

// ProposeBlock proposes a new block
func (tpn *TestProcessorNode) ProposeBlock(round uint64, nonce uint64) (data.BodyHandler, data.HeaderHandler, [][]byte) {
	startTime := time.Now()
	maxTime := time.Second * 2

	haveTime := func() bool {
		elapsedTime := time.Since(startTime)
		remainingTime := maxTime - elapsedTime
		return remainingTime > 0
	}

	blockHeader := tpn.BlockProcessor.CreateNewHeader(round, nonce)

	blockHeader.SetShardID(tpn.ShardCoordinator.SelfId())
	blockHeader.SetPubKeysBitmap([]byte{1})

	currHdr := tpn.BlockChain.GetCurrentBlockHeader()
	currHdrHash := tpn.BlockChain.GetCurrentBlockHeaderHash()
	if check.IfNil(currHdr) {
		currHdr = tpn.BlockChain.GetGenesisHeader()
		currHdrHash = tpn.BlockChain.GetGenesisHeaderHash()
	}

	blockHeader.SetPrevHash(currHdrHash)
	blockHeader.SetPrevRandSeed(currHdr.GetRandSeed())
	sig, _ := TestMultiSig.AggregateSigs(nil)
	blockHeader.SetSignature(sig)
	blockHeader.SetRandSeed(sig)
	blockHeader.SetLeaderSignature([]byte("leader sign"))
	blockHeader.SetChainID(tpn.ChainID)
	blockHeader.SetSoftwareVersion(SoftwareVersion)
	blockHeader.SetTimeStamp(round * uint64(tpn.Rounder.TimeDuration().Seconds()))

	blockHeader, blockBody, err := tpn.BlockProcessor.CreateBlock(blockHeader, haveTime)
	if err != nil {
		log.Warn("createBlockBody", "error", err.Error())
		return nil, nil, nil
	}

	shardBlockBody, ok := blockBody.(*dataBlock.Body)
	txHashes := make([][]byte, 0)
	if !ok {
		return blockBody, blockHeader, txHashes
	}

	for _, mb := range shardBlockBody.MiniBlocks {
		if mb.Type == dataBlock.PeerBlock {
			continue
		}
		for _, hash := range mb.TxHashes {
			copiedHash := make([]byte, len(hash))
			copy(copiedHash, hash)
			txHashes = append(txHashes, copiedHash)
		}
	}

	return blockBody, blockHeader, txHashes
}

// BroadcastBlock broadcasts the block and body to the connected peers
func (tpn *TestProcessorNode) BroadcastBlock(body data.BodyHandler, header data.HeaderHandler) {
	_ = tpn.BroadcastMessenger.BroadcastBlock(body, header)

	time.Sleep(tpn.WaitTime)

	miniBlocks, transactions, _ := tpn.BlockProcessor.MarshalizedDataToBroadcast(header, body)
	_ = tpn.BroadcastMessenger.BroadcastMiniBlocks(miniBlocks)
	_ = tpn.BroadcastMessenger.BroadcastTransactions(transactions)
}

// CommitBlock commits the block and body
func (tpn *TestProcessorNode) CommitBlock(body data.BodyHandler, header data.HeaderHandler) {
	_ = tpn.BlockProcessor.CommitBlock(header, body)
}

// GetShardHeader returns the first *dataBlock.Header stored in datapools having the nonce provided as parameter
func (tpn *TestProcessorNode) GetShardHeader(nonce uint64) (*dataBlock.Header, error) {
	invalidCachers := tpn.DataPool == nil || tpn.DataPool.Headers() == nil
	if invalidCachers {
		return nil, errors.New("invalid data pool")
	}

	headerObjects, _, err := tpn.DataPool.Headers().GetHeadersByNonceAndShardId(nonce, tpn.ShardCoordinator.SelfId())
	if err != nil {
		return nil, errors.New(fmt.Sprintf("no headers found for nonce %d and shard id %d %s", nonce, tpn.ShardCoordinator.SelfId(), err.Error()))
	}

	headerObject := headerObjects[len(headerObjects)-1]

	header, ok := headerObject.(*dataBlock.Header)
	if !ok {
		return nil, errors.New(fmt.Sprintf("not a *dataBlock.Header stored in headers found for nonce and shard id %d %d", nonce, tpn.ShardCoordinator.SelfId()))
	}

	return header, nil
}

// GetBlockBody returns the body for provided header parameter
func (tpn *TestProcessorNode) GetBlockBody(header *dataBlock.Header) (*dataBlock.Body, error) {
	invalidCachers := tpn.DataPool == nil || tpn.DataPool.MiniBlocks() == nil
	if invalidCachers {
		return nil, errors.New("invalid data pool")
	}

	body := &dataBlock.Body{}
	for _, miniBlockHeader := range header.MiniBlockHeaders {
		miniBlockHash := miniBlockHeader.Hash

		mbObject, ok := tpn.DataPool.MiniBlocks().Get(miniBlockHash)
		if !ok {
			return nil, errors.New(fmt.Sprintf("no miniblock found for hash %s", hex.EncodeToString(miniBlockHash)))
		}

		mb, ok := mbObject.(*dataBlock.MiniBlock)
		if !ok {
			return nil, errors.New(fmt.Sprintf("not a *dataBlock.MiniBlock stored in miniblocks found for hash %s", hex.EncodeToString(miniBlockHash)))
		}

		body.MiniBlocks = append(body.MiniBlocks, mb)
	}

	return body, nil
}

// GetMetaBlockBody returns the body for provided header parameter
func (tpn *TestProcessorNode) GetMetaBlockBody(header *dataBlock.MetaBlock) (*dataBlock.Body, error) {
	invalidCachers := tpn.DataPool == nil || tpn.DataPool.MiniBlocks() == nil
	if invalidCachers {
		return nil, errors.New("invalid data pool")
	}

	body := &dataBlock.Body{}
	for _, miniBlockHeader := range header.MiniBlockHeaders {
		miniBlockHash := miniBlockHeader.Hash

		mbObject, ok := tpn.DataPool.MiniBlocks().Get(miniBlockHash)
		if !ok {
			return nil, errors.New(fmt.Sprintf("no miniblock found for hash %s", hex.EncodeToString(miniBlockHash)))
		}

		mb, ok := mbObject.(*dataBlock.MiniBlock)
		if !ok {
			return nil, errors.New(fmt.Sprintf("not a *dataBlock.MiniBlock stored in miniblocks found for hash %s", hex.EncodeToString(miniBlockHash)))
		}

		body.MiniBlocks = append(body.MiniBlocks, mb)
	}

	return body, nil
}

// GetMetaHeader returns the first *dataBlock.MetaBlock stored in datapools having the nonce provided as parameter
func (tpn *TestProcessorNode) GetMetaHeader(nonce uint64) (*dataBlock.MetaBlock, error) {
	invalidCachers := tpn.DataPool == nil || tpn.DataPool.Headers() == nil
	if invalidCachers {
		return nil, errors.New("invalid data pool")
	}

	headerObjects, _, err := tpn.DataPool.Headers().GetHeadersByNonceAndShardId(nonce, core.MetachainShardId)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("no headers found for nonce and shard id %d %d %s", nonce, core.MetachainShardId, err.Error()))
	}

	headerObject := headerObjects[len(headerObjects)-1]

	header, ok := headerObject.(*dataBlock.MetaBlock)
	if !ok {
		return nil, errors.New(fmt.Sprintf("not a *dataBlock.MetaBlock stored in headers found for nonce and shard id %d %d", nonce, core.MetachainShardId))
	}

	return header, nil
}

// SyncNode tries to process and commit a block already stored in data pool with provided nonce
func (tpn *TestProcessorNode) SyncNode(nonce uint64) error {
	if tpn.ShardCoordinator.SelfId() == core.MetachainShardId {
		return tpn.syncMetaNode(nonce)
	} else {
		return tpn.syncShardNode(nonce)
	}
}

func (tpn *TestProcessorNode) syncShardNode(nonce uint64) error {
	header, err := tpn.GetShardHeader(nonce)
	if err != nil {
		return err
	}

	body, err := tpn.GetBlockBody(header)
	if err != nil {
		return err
	}

	err = tpn.BlockProcessor.ProcessBlock(
		header,
		body,
		func() time.Duration {
			return time.Second * 5
		},
	)
	if err != nil {
		return err
	}

	err = tpn.BlockProcessor.CommitBlock(header, body)
	if err != nil {
		return err
	}

	return nil
}

func (tpn *TestProcessorNode) syncMetaNode(nonce uint64) error {
	header, err := tpn.GetMetaHeader(nonce)
	if err != nil {
		return err
	}

	body, err := tpn.GetMetaBlockBody(header)
	if err != nil {
		return err
	}

	err = tpn.BlockProcessor.ProcessBlock(
		header,
		body,
		func() time.Duration {
			return time.Second * 2
		},
	)
	if err != nil {
		return err
	}

	err = tpn.BlockProcessor.CommitBlock(header, body)
	if err != nil {
		return err
	}

	return nil
}

// SetAccountNonce sets the account nonce with journal
func (tpn *TestProcessorNode) SetAccountNonce(nonce uint64) error {
	nodeAccount, _ := tpn.AccntState.LoadAccount(tpn.OwnAccount.Address)
	nodeAccount.(state.UserAccountHandler).IncreaseNonce(nonce)

	err := tpn.AccntState.SaveAccount(nodeAccount)
	if err != nil {
		return err
	}

	_, err = tpn.AccntState.Commit()
	if err != nil {
		return err
	}

	return nil
}

// MiniBlocksPresent checks if the all the miniblocks are present in the pool
func (tpn *TestProcessorNode) MiniBlocksPresent(hashes [][]byte) bool {
	mbCacher := tpn.DataPool.MiniBlocks()
	for i := 0; i < len(hashes); i++ {
		ok := mbCacher.Has(hashes[i])
		if !ok {
			return false
		}
	}

	return true
}

func (tpn *TestProcessorNode) initRounder() {
	tpn.Rounder = &mock.RounderMock{TimeDurationField: 5 * time.Second}
}

func (tpn *TestProcessorNode) initRequestedItemsHandler() {
	tpn.RequestedItemsHandler = timecache.NewTimeCache(roundDuration)
}

func (tpn *TestProcessorNode) initBlockTracker() {
	argBaseTracker := track.ArgBaseTracker{
		Hasher:           TestHasher,
		HeaderValidator:  tpn.HeaderValidator,
		Marshalizer:      TestMarshalizer,
		RequestHandler:   tpn.RequestHandler,
		Rounder:          tpn.Rounder,
		ShardCoordinator: tpn.ShardCoordinator,
		Store:            tpn.Storage,
		StartHeaders:     tpn.GenesisBlocks,
		PoolsHolder:      tpn.DataPool,
		WhitelistHandler: tpn.WhiteListHandler,
	}

	if tpn.ShardCoordinator.SelfId() != core.MetachainShardId {
		arguments := track.ArgShardTracker{
			ArgBaseTracker: argBaseTracker,
		}

		tpn.BlockTracker, _ = track.NewShardBlockTrack(arguments)
	} else {
		arguments := track.ArgMetaTracker{
			ArgBaseTracker: argBaseTracker,
		}

		tpn.BlockTracker, _ = track.NewMetaBlockTrack(arguments)
	}
}

func (tpn *TestProcessorNode) initHeaderValidator() {
	argsHeaderValidator := block.ArgsHeaderValidator{
		Hasher:      TestHasher,
		Marshalizer: TestMarshalizer,
	}

	tpn.HeaderValidator, _ = block.NewHeaderValidator(argsHeaderValidator)
}
