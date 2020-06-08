package integrationTests

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/accumulator"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/crypto/signing"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/ed25519"
	ed25519SingleSig "github.com/ElrondNetwork/elrond-go/crypto/signing/ed25519/singlesig"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/mcl"
	"github.com/ElrondNetwork/elrond-go/data"
	dataBlock "github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/blockchain"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/state/factory"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/data/trie"
	"github.com/ElrondNetwork/elrond-go/data/trie/evictionWaitingList"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/dataPool"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/dataPool/headersCache"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/shardedData"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/txpool"
	"github.com/ElrondNetwork/elrond-go/display"
	genesisProcess "github.com/ElrondNetwork/elrond-go/genesis/process"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/hashing/sha256"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	procFactory "github.com/ElrondNetwork/elrond-go/process/factory"
	txProc "github.com/ElrondNetwork/elrond-go/process/transaction"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/memorydb"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/ElrondNetwork/elrond-go/vm/systemSmartContracts/defaults"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

// StepDelay is used so that transactions can disseminate properly
var StepDelay = time.Second

// SyncDelay is used so that nodes have enough time to sync
var SyncDelay = time.Second * 2

// P2pBootstrapDelay is used so that nodes have enough time to bootstrap
var P2pBootstrapDelay = 5 * time.Second

var log = logger.GetOrCreate("integrationtests")

// shuffler constants
const (
	shuffleBetweenShards = false
	adaptivity           = false
	hysteresis           = float32(0.2)
	maxTrieLevelInMemory = uint(5)
)

// Type defines account types to save in accounts trie
type Type uint8

const (
	// UserAccount identifies an account holding balance, storage updates, code
	UserAccount Type = 0
	// ValidatorAccount identifies an account holding stake, crypto public keys, assigned shard, rating
	ValidatorAccount Type = 1
)

const defaultChancesSelection = 1

// GetConnectableAddress returns a non circuit, non windows default connectable address for provided messenger
func GetConnectableAddress(mes p2p.Messenger) string {
	for _, addr := range mes.Addresses() {
		if strings.Contains(addr, "circuit") || strings.Contains(addr, "169.254") {
			continue
		}
		return addr
	}
	return ""
}

func createP2PConfig(initialPeerList []string) config.P2PConfig {
	return config.P2PConfig{
		Node: config.NodeConfig{
			Port: "0",
		},
		KadDhtPeerDiscovery: config.KadDhtPeerDiscoveryConfig{
			Enabled:                          true,
			RefreshIntervalInSec:             2,
			RandezVous:                       "/erd/kad/1.0.0",
			InitialPeerList:                  initialPeerList,
			BucketSize:                       100,
			RoutingTableRefreshIntervalInSec: 100,
		},
		Sharding: config.ShardingConfig{
			Type: p2p.NilListSharder,
		},
	}
}

// CreateMessengerWithKadDht creates a new libp2p messenger with kad-dht peer discovery
func CreateMessengerWithKadDht(initialAddr string) p2p.Messenger {
	initialAddresses := make([]string, 0)
	if len(initialAddr) > 0 {
		initialAddresses = append(initialAddresses, initialAddr)
	}
	arg := libp2p.ArgsNetworkMessenger{
		ListenAddress: libp2p.ListenLocalhostAddrWithIp4AndTcp,
		P2pConfig:     createP2PConfig(initialAddresses),
	}

	libP2PMes, err := libp2p.NewNetworkMessenger(arg)
	if err != nil {
		fmt.Println(err.Error())
	}

	return libP2PMes
}

// CreateMessengerWithKadDhtAndProtocolID creates a new libp2p messenger with kad-dht peer discovery and peer ID
func CreateMessengerWithKadDhtAndProtocolID(initialAddr string, randezVous string) p2p.Messenger {
	initialAddresses := make([]string, 0)
	if len(initialAddr) > 0 {
		initialAddresses = append(initialAddresses, initialAddr)
	}
	p2pConfig := createP2PConfig(initialAddresses)
	p2pConfig.KadDhtPeerDiscovery.RandezVous = randezVous
	arg := libp2p.ArgsNetworkMessenger{
		ListenAddress: libp2p.ListenLocalhostAddrWithIp4AndTcp,
		P2pConfig:     p2pConfig,
	}

	libP2PMes, err := libp2p.NewNetworkMessenger(arg)
	if err != nil {
		fmt.Println(err.Error())
	}

	return libP2PMes
}

// CreateMessengerFromConfig creates a new libp2p messenger with provided configuration
func CreateMessengerFromConfig(p2pConfig config.P2PConfig) p2p.Messenger {
	arg := libp2p.ArgsNetworkMessenger{
		ListenAddress: libp2p.ListenLocalhostAddrWithIp4AndTcp,
		P2pConfig:     p2pConfig,
	}

	libP2PMes, err := libp2p.NewNetworkMessenger(arg)
	if err != nil {
		fmt.Println(err.Error())
	}

	return libP2PMes
}

// CreateMessengerWithNoDiscovery creates a new libp2p messenger with no peer discovery
func CreateMessengerWithNoDiscovery() p2p.Messenger {
	p2pConfig := config.P2PConfig{
		Node: config.NodeConfig{
			Port: "0",
			Seed: "",
		},
		KadDhtPeerDiscovery: config.KadDhtPeerDiscoveryConfig{
			Enabled: false,
		},
		Sharding: config.ShardingConfig{
			Type: p2p.NilListSharder,
		},
	}

	arg := libp2p.ArgsNetworkMessenger{
		ListenAddress: libp2p.ListenLocalhostAddrWithIp4AndTcp,
		P2pConfig:     p2pConfig,
	}

	libP2PMes, err := libp2p.NewNetworkMessenger(arg)
	if err != nil {
		fmt.Println(err.Error())
	}

	return libP2PMes
}

// CreateFixedNetworkOf8Peers assembles a network as following:
//
//                             0------------------- 1
//                             |                    |
//        2 ------------------ 3 ------------------ 4
//        |                    |                    |
//        5                    6                    7
func CreateFixedNetworkOf8Peers() ([]p2p.Messenger, error) {
	peers := createMessengersWithNoDiscovery(8)

	connections := map[int][]int{
		0: {1, 3},
		1: {4},
		2: {5, 3},
		3: {4, 6},
		4: {7},
	}

	err := createConnections(peers, connections)
	if err != nil {
		return nil, err
	}

	return peers, nil
}

// CreateFixedNetworkOf14Peers assembles a network as following:
//
//                 0
//                 |
//                 1
//                 |
//  +--+--+--+--+--2--+--+--+--+--+
//  |  |  |  |  |  |  |  |  |  |  |
//  3  4  5  6  7  8  9  10 11 12 13
func CreateFixedNetworkOf14Peers() ([]p2p.Messenger, error) {
	peers := createMessengersWithNoDiscovery(14)

	connections := map[int][]int{
		0: {1},
		1: {2},
		2: {3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
	}

	err := createConnections(peers, connections)
	if err != nil {
		return nil, err
	}

	return peers, nil
}

func createMessengersWithNoDiscovery(numPeers int) []p2p.Messenger {
	peers := make([]p2p.Messenger, numPeers)

	for i := 0; i < numPeers; i++ {
		peers[i] = CreateMessengerWithNoDiscovery()
	}

	return peers
}

func createConnections(peers []p2p.Messenger, connections map[int][]int) error {
	for pid, connectTo := range connections {
		err := connectPeerToOthers(peers, pid, connectTo)
		if err != nil {
			return err
		}
	}

	return nil
}

func connectPeerToOthers(peers []p2p.Messenger, idx int, connectToIdxes []int) error {
	for _, connectToIdx := range connectToIdxes {
		err := peers[idx].ConnectToPeer(peers[connectToIdx].Addresses()[0])
		if err != nil {
			return fmt.Errorf("%w connecting %s to %s", err, peers[idx].ID(), peers[connectToIdx].ID())
		}
	}

	return nil
}

// ClosePeers calls Messenger.Close on the provided peers
func ClosePeers(peers []p2p.Messenger) {
	for _, p := range peers {
		_ = p.Close()
	}
}

// CreateTestDataPool creates a test data pool for shard nodes
func CreateTestDataPool(txPool dataRetriever.ShardedDataCacherNotifier, selfShardID uint32) dataRetriever.PoolsHolder {
	if txPool == nil {
		txPool, _ = createTxPool(selfShardID)
	}

	uTxPool, _ := shardedData.NewShardedData(storageUnit.CacheConfig{Capacity: 100000, Type: storageUnit.LRUCache, Shards: 1})
	rewardsTxPool, _ := shardedData.NewShardedData(storageUnit.CacheConfig{Capacity: 300, Type: storageUnit.LRUCache, Shards: 1})

	hdrPool, _ := headersCache.NewHeadersPool(config.HeadersPoolConfig{MaxHeadersPerShard: 1000, NumElementsToRemoveOnEviction: 100})

	cacherCfg := storageUnit.CacheConfig{Capacity: 100000, Type: storageUnit.LRUCache, Shards: 1}
	txBlockBody, _ := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Capacity, cacherCfg.Shards, cacherCfg.SizeInBytes)

	cacherCfg = storageUnit.CacheConfig{Capacity: 100000, Type: storageUnit.LRUCache, Shards: 1}
	peerChangeBlockBody, _ := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Capacity, cacherCfg.Shards, cacherCfg.SizeInBytes)

	cacherCfg = storageUnit.CacheConfig{Capacity: 50000, Type: storageUnit.LRUCache}
	trieNodes, _ := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Capacity, cacherCfg.Shards, cacherCfg.SizeInBytes)

	currTxs, _ := dataPool.NewCurrentBlockPool()

	dPool, _ := dataPool.NewDataPool(
		txPool,
		uTxPool,
		rewardsTxPool,
		hdrPool,
		txBlockBody,
		peerChangeBlockBody,
		trieNodes,
		currTxs,
	)

	return dPool
}

// CreateMemUnit returns an in-memory storer implementation (the vast majority of tests do not require effective
// disk I/O)
func CreateMemUnit() storage.Storer {
	capacity := uint32(10)
	shards := uint32(1)
	sizeInBytes := uint64(0)
	cache, _ := storageUnit.NewCache(storageUnit.LRUCache, capacity, shards, sizeInBytes)
	persist, _ := memorydb.NewlruDB(100000)
	unit, _ := storageUnit.NewStorageUnit(cache, persist)

	return unit
}

// CreateStore creates a storage service for shard nodes
func CreateStore(numOfShards uint32) dataRetriever.StorageService {
	store := dataRetriever.NewChainStorer()
	store.AddStorer(dataRetriever.TransactionUnit, CreateMemUnit())
	store.AddStorer(dataRetriever.MiniBlockUnit, CreateMemUnit())
	store.AddStorer(dataRetriever.MetaBlockUnit, CreateMemUnit())
	store.AddStorer(dataRetriever.PeerChangesUnit, CreateMemUnit())
	store.AddStorer(dataRetriever.BlockHeaderUnit, CreateMemUnit())
	store.AddStorer(dataRetriever.UnsignedTransactionUnit, CreateMemUnit())
	store.AddStorer(dataRetriever.RewardTransactionUnit, CreateMemUnit())
	store.AddStorer(dataRetriever.MetaHdrNonceHashDataUnit, CreateMemUnit())
	store.AddStorer(dataRetriever.HeartbeatUnit, CreateMemUnit())
	store.AddStorer(dataRetriever.BootstrapUnit, CreateMemUnit())
	store.AddStorer(dataRetriever.StatusMetricsUnit, CreateMemUnit())
	store.AddStorer(dataRetriever.MetaHdrNonceHashDataUnit, CreateMemUnit())

	for i := uint32(0); i < numOfShards; i++ {
		hdrNonceHashDataUnit := dataRetriever.ShardHdrNonceHashDataUnit + dataRetriever.UnitType(i)
		store.AddStorer(hdrNonceHashDataUnit, CreateMemUnit())
	}

	return store
}

// CreateTrieStorageManager creates the trie storage manager for the tests
func CreateTrieStorageManager() (data.StorageManager, storage.Storer) {
	store := CreateMemUnit()
	ewl, _ := evictionWaitingList.NewEvictionWaitingList(100, memorydb.New(), TestMarshalizer)

	// TODO change this implementation with a factory
	tempDir, _ := ioutil.TempDir("", "integrationTests")
	cfg := config.DBConfig{
		FilePath:          tempDir,
		Type:              string(storageUnit.LvlDBSerial),
		BatchDelaySeconds: 4,
		MaxBatchSize:      10000,
		MaxOpenFiles:      10,
	}
	generalCfg := config.TrieStorageManagerConfig{
		PruningBufferLen:   1000,
		SnapshotsBufferLen: 10,
		MaxSnapshots:       2,
	}
	trieStorageManager, _ := trie.NewTrieStorageManager(store, TestMarshalizer, TestHasher, cfg, ewl, generalCfg)

	return trieStorageManager, store
}

// CreateAccountsDB creates an account state with a valid trie implementation but with a memory storage
func CreateAccountsDB(
	accountType Type,
	trieStorageManager data.StorageManager,
) (*state.AccountsDB, data.Trie) {
	tr, _ := trie.NewTrie(trieStorageManager, TestMarshalizer, TestHasher, maxTrieLevelInMemory)

	accountFactory := getAccountFactory(accountType)
	adb, _ := state.NewAccountsDB(tr, sha256.Sha256{}, TestMarshalizer, accountFactory)

	return adb, tr
}

func getAccountFactory(accountType Type) state.AccountFactory {
	switch accountType {
	case UserAccount:
		return factory.NewAccountCreator()
	case ValidatorAccount:
		return factory.NewPeerAccountCreator()
	default:
		return nil
	}
}

// CreateShardChain creates a blockchain implementation used by the shard nodes
func CreateShardChain() data.ChainHandler {
	blockChain := blockchain.NewBlockChain()
	_ = blockChain.SetGenesisHeader(&dataBlock.Header{})
	genesisHeaderM, _ := TestMarshalizer.Marshal(blockChain.GetGenesisHeader())

	blockChain.SetGenesisHeaderHash(TestHasher.Compute(string(genesisHeaderM)))

	return blockChain
}

// CreateMetaChain creates a blockchain implementation used by the meta nodes
func CreateMetaChain() data.ChainHandler {
	metaChain := blockchain.NewMetaChain()
	_ = metaChain.SetGenesisHeader(&dataBlock.MetaBlock{})
	genesisHeaderHash, _ := core.CalculateHash(TestMarshalizer, TestHasher, metaChain.GetGenesisHeader())
	metaChain.SetGenesisHeaderHash(genesisHeaderHash)

	return metaChain
}

// CreateSimpleGenesisBlocks creates empty genesis blocks for all known shards, including metachain
func CreateSimpleGenesisBlocks(shardCoordinator sharding.Coordinator) map[uint32]data.HeaderHandler {
	genesisBlocks := make(map[uint32]data.HeaderHandler)
	for shardId := uint32(0); shardId < shardCoordinator.NumberOfShards(); shardId++ {
		genesisBlocks[shardId] = CreateSimpleGenesisBlock(shardId)
	}

	genesisBlocks[core.MetachainShardId] = CreateSimpleGenesisMetaBlock()

	return genesisBlocks
}

// CreateSimpleGenesisBlock creates a new mock shard genesis block
func CreateSimpleGenesisBlock(shardId uint32) *dataBlock.Header {
	rootHash := []byte("root hash")

	return &dataBlock.Header{
		Nonce:           0,
		Round:           0,
		Signature:       rootHash,
		RandSeed:        rootHash,
		PrevRandSeed:    rootHash,
		ShardID:         shardId,
		PubKeysBitmap:   rootHash,
		RootHash:        rootHash,
		PrevHash:        rootHash,
		AccumulatedFees: big.NewInt(0),
		DeveloperFees:   big.NewInt(0),
	}
}

// CreateSimpleGenesisMetaBlock creates a new mock meta genesis block
func CreateSimpleGenesisMetaBlock() *dataBlock.MetaBlock {
	rootHash := []byte("root hash")

	return &dataBlock.MetaBlock{
		Nonce:                  0,
		Epoch:                  0,
		Round:                  0,
		TimeStamp:              0,
		ShardInfo:              nil,
		Signature:              rootHash,
		PubKeysBitmap:          rootHash,
		PrevHash:               rootHash,
		PrevRandSeed:           rootHash,
		RandSeed:               rootHash,
		RootHash:               rootHash,
		ValidatorStatsRootHash: rootHash,
		TxCount:                0,
		MiniBlockHeaders:       nil,
		AccumulatedFees:        big.NewInt(0),
		AccumulatedFeesInEpoch: big.NewInt(0),
		DeveloperFees:          big.NewInt(0),
		DevFeesInEpoch:         big.NewInt(0),
	}
}

// CreateGenesisBlocks creates empty genesis blocks for all known shards, including metachain
func CreateGenesisBlocks(
	accounts state.AccountsAdapter,
	validatorAccounts state.AccountsAdapter,
	trieStorageManagers map[string]data.StorageManager,
	pubkeyConv core.PubkeyConverter,
	nodesSetup sharding.GenesisNodesSetupHandler,
	shardCoordinator sharding.Coordinator,
	store dataRetriever.StorageService,
	blkc data.ChainHandler,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	uint64Converter typeConverters.Uint64ByteSliceConverter,
	dataPool dataRetriever.PoolsHolder,
	economics *economics.EconomicsData,
) map[uint32]data.HeaderHandler {

	genesisBlocks := make(map[uint32]data.HeaderHandler)
	for shardId := uint32(0); shardId < shardCoordinator.NumberOfShards(); shardId++ {
		genesisBlocks[shardId] = CreateSimpleGenesisBlock(shardId)
	}

	genesisBlocks[core.MetachainShardId] = CreateGenesisMetaBlock(
		accounts,
		validatorAccounts,
		trieStorageManagers,
		pubkeyConv,
		nodesSetup,
		shardCoordinator,
		store,
		blkc,
		marshalizer,
		hasher,
		uint64Converter,
		dataPool,
		economics,
	)

	return genesisBlocks
}

// CreateGenesisMetaBlock creates a new mock meta genesis block
func CreateGenesisMetaBlock(
	accounts state.AccountsAdapter,
	validatorAccounts state.AccountsAdapter,
	trieStorageManagers map[string]data.StorageManager,
	pubkeyConv core.PubkeyConverter,
	nodesSetup sharding.GenesisNodesSetupHandler,
	shardCoordinator sharding.Coordinator,
	store dataRetriever.StorageService,
	blkc data.ChainHandler,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	uint64Converter typeConverters.Uint64ByteSliceConverter,
	dataPool dataRetriever.PoolsHolder,
	economics *economics.EconomicsData,
) data.HeaderHandler {
	gasSchedule := make(map[string]map[string]uint64)
	defaults.FillGasMapInternal(gasSchedule, 1)

	argsMetaGenesis := genesisProcess.ArgsGenesisBlockCreator{
		GenesisTime:              0,
		Accounts:                 accounts,
		TrieStorageManagers:      trieStorageManagers,
		PubkeyConv:               pubkeyConv,
		InitialNodesSetup:        nodesSetup,
		ShardCoordinator:         shardCoordinator,
		Store:                    store,
		Blkc:                     blkc,
		Marshalizer:              marshalizer,
		Hasher:                   hasher,
		Uint64ByteSliceConverter: uint64Converter,
		DataPool:                 dataPool,
		Economics:                economics,
		ValidatorAccounts:        validatorAccounts,
		GasMap:                   gasSchedule,
		TxLogsProcessor:          &mock.TxLogsProcessorStub{},
		VirtualMachineConfig:     config.VirtualMachineConfig{},
		HardForkConfig:           config.HardforkConfig{},
		SystemSCConfig: config.SystemSmartContractsConfig{
			ESDTSystemSCConfig: config.ESDTSystemSCConfig{
				BaseIssuingCost: "1000",
				OwnerAddress:    "aaaaaa",
			},
		},
		BlockSignKeyGen: &mock.KeyGenMock{},
	}

	if shardCoordinator.SelfId() != core.MetachainShardId {
		newShardCoordinator, _ := sharding.NewMultiShardCoordinator(
			shardCoordinator.NumberOfShards(),
			core.MetachainShardId,
		)

		newDataPool := CreateTestDataPool(nil, shardCoordinator.SelfId())

		newBlkc := blockchain.NewMetaChain()
		trieStorage, _ := CreateTrieStorageManager()
		newAccounts, _ := CreateAccountsDB(UserAccount, trieStorage)

		argsMetaGenesis.ShardCoordinator = newShardCoordinator
		argsMetaGenesis.Accounts = newAccounts
		argsMetaGenesis.Blkc = newBlkc
		argsMetaGenesis.DataPool = newDataPool
	}

	nodesHandler, err := mock.NewNodesHandlerMock(nodesSetup)
	log.LogIfError(err)

	metaHdr, err := genesisProcess.CreateMetaGenesisBlock(argsMetaGenesis, nodesHandler)
	log.LogIfError(err)

	log.Info("meta genesis root hash", "hash", hex.EncodeToString(metaHdr.GetRootHash()))
	log.Info("meta genesis validatorStatistics",
		"shardID", shardCoordinator.SelfId(),
		"hash", hex.EncodeToString(metaHdr.GetValidatorStatsRootHash()),
	)

	return metaHdr
}

// CreateRandomAddress creates a random byte array with fixed size
func CreateRandomAddress() []byte {
	return CreateRandomBytes(32)
}

// MintAddress will create an account (if it does not exists), update the balance with required value,
// save the account and commit the trie.
func MintAddress(accnts state.AccountsAdapter, addressBytes []byte, value *big.Int) {
	accnt, _ := accnts.LoadAccount(addressBytes)
	_ = accnt.(state.UserAccountHandler).AddToBalance(value)
	_ = accnts.SaveAccount(accnt)
	_, _ = accnts.Commit()
}

// CreateAccount creates a new account and returns the address
func CreateAccount(accnts state.AccountsAdapter, nonce uint64, balance *big.Int) []byte {
	address := CreateRandomBytes(32)
	account, _ := accnts.LoadAccount(address)
	account.(state.UserAccountHandler).IncreaseNonce(nonce)
	_ = account.(state.UserAccountHandler).AddToBalance(balance)
	_ = accnts.SaveAccount(account)

	return address
}

// MakeDisplayTable will output a string containing counters for received transactions, headers, miniblocks and
// meta headers for all provided test nodes
func MakeDisplayTable(nodes []*TestProcessorNode) string {
	header := []string{"pk", "shard ID", "txs", "miniblocks", "headers", "metachain headers", "connections"}
	dataLines := make([]*display.LineData, len(nodes))

	for idx, n := range nodes {
		dataLines[idx] = display.NewLineData(
			false,
			[]string{
				hex.EncodeToString(n.OwnAccount.PkTxSignBytes),
				fmt.Sprintf("%d", n.ShardCoordinator.SelfId()),
				fmt.Sprintf("%d", atomic.LoadInt32(&n.CounterTxRecv)),
				fmt.Sprintf("%d", atomic.LoadInt32(&n.CounterMbRecv)),
				fmt.Sprintf("%d", atomic.LoadInt32(&n.CounterHdrRecv)),
				fmt.Sprintf("%d", atomic.LoadInt32(&n.CounterMetaRcv)),
				fmt.Sprintf("%d", len(n.Messenger.ConnectedPeers())),
			},
		)
	}
	table, _ := display.CreateTableString(header, dataLines)
	return table
}

// PrintShardAccount outputs on console a shard account data contained
func PrintShardAccount(accnt state.UserAccountHandler, tag string) {
	str := fmt.Sprintf("%s Address: %s\n", tag, base64.StdEncoding.EncodeToString(accnt.AddressBytes()))
	str += fmt.Sprintf("  Nonce: %d\n", accnt.GetNonce())
	str += fmt.Sprintf("  Balance: %d\n", accnt.GetBalance().Uint64())
	str += fmt.Sprintf("  Code hash: %s\n", base64.StdEncoding.EncodeToString(accnt.GetCodeHash()))
	str += fmt.Sprintf("  Root hash: %s\n", base64.StdEncoding.EncodeToString(accnt.GetRootHash()))

	fmt.Println(str)
}

// CreateRandomBytes returns a random byte slice with the given size
func CreateRandomBytes(chars int) []byte {
	buff := make([]byte, chars)
	_, _ = rand.Reader.Read(buff)

	return buff
}

// GenerateAddressJournalAccountAccountsDB returns an account, the accounts address, and the accounts database
func GenerateAddressJournalAccountAccountsDB() ([]byte, state.UserAccountHandler, *state.AccountsDB) {
	adr := CreateRandomAddress()
	trieStorage, _ := CreateTrieStorageManager()
	adb, _ := CreateAccountsDB(UserAccount, trieStorage)
	account, _ := state.NewUserAccount(adr)

	return adr, account, adb
}

// AdbEmulateBalanceTxSafeExecution emulates a tx execution by altering the accounts
// balance and nonce, and printing any encountered error
func AdbEmulateBalanceTxSafeExecution(acntSrc, acntDest state.UserAccountHandler, accounts state.AccountsAdapter, value *big.Int) {

	snapshot := accounts.JournalLen()
	err := AdbEmulateBalanceTxExecution(accounts, acntSrc, acntDest, value)

	if err != nil {
		fmt.Printf("Error executing tx (value: %v), reverting...\n", value)
		err = accounts.RevertToSnapshot(snapshot)

		if err != nil {
			panic(err)
		}
	}
}

// AdbEmulateBalanceTxExecution emulates a tx execution by altering the accounts
// balance and nonce, and printing any encountered error
func AdbEmulateBalanceTxExecution(accounts state.AccountsAdapter, acntSrc, acntDest state.UserAccountHandler, value *big.Int) error {

	srcVal := acntSrc.GetBalance()
	if srcVal.Cmp(value) < 0 {
		return errors.New("not enough funds")
	}

	err := acntSrc.SubFromBalance(value)
	if err != nil {
		return err
	}

	err = acntDest.AddToBalance(value)
	if err != nil {
		return err
	}

	acntSrc.IncreaseNonce(1)

	err = accounts.SaveAccount(acntSrc)
	if err != nil {
		return err
	}

	err = accounts.SaveAccount(acntDest)
	if err != nil {
		return err
	}

	return nil
}

// CreateSimpleTxProcessor returns a transaction processor
func CreateSimpleTxProcessor(accnts state.AccountsAdapter) process.TransactionProcessor {
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(1)
	txProcessor, _ := txProc.NewTxProcessor(
		accnts,
		TestHasher,
		TestAddressPubkeyConverter,
		TestMarshalizer,
		shardCoordinator,
		&mock.SCProcessorMock{},
		&mock.UnsignedTxHandlerMock{},
		&mock.TxTypeHandlerMock{},
		&mock.FeeHandlerStub{
			ComputeGasLimitCalled: func(tx process.TransactionWithFeeHandler) uint64 {
				return tx.GetGasLimit()
			},
			CheckValidityTxValuesCalled: func(tx process.TransactionWithFeeHandler) error {
				return nil
			},
			ComputeFeeCalled: func(tx process.TransactionWithFeeHandler) *big.Int {
				fee := big.NewInt(0).SetUint64(tx.GetGasLimit())
				fee.Mul(fee, big.NewInt(0).SetUint64(tx.GetGasPrice()))

				return fee
			},
		},
		&mock.IntermediateTransactionHandlerMock{},
		&mock.IntermediateTransactionHandlerMock{},
		vmcommon.NewAtArgumentParser(),
		&mock.IntermediateTransactionHandlerMock{},
	)

	return txProcessor
}

// CreateNewDefaultTrie returns a new trie with test hasher and marsahalizer
func CreateNewDefaultTrie() data.Trie {
	ewl, _ := evictionWaitingList.NewEvictionWaitingList(100, memorydb.New(), TestMarshalizer)
	generalCfg := config.TrieStorageManagerConfig{
		PruningBufferLen:   1000,
		SnapshotsBufferLen: 10,
		MaxSnapshots:       2,
	}
	trieStorage, _ := trie.NewTrieStorageManager(CreateMemUnit(), TestMarshalizer, TestHasher, config.DBConfig{}, ewl, generalCfg)

	tr, _ := trie.NewTrie(trieStorage, TestMarshalizer, TestHasher, maxTrieLevelInMemory)
	return tr
}

// GenerateRandomSlice returns a random byte slice with the given size
func GenerateRandomSlice(size int) []byte {
	buff := make([]byte, size)
	_, _ = rand.Reader.Read(buff)

	return buff
}

// MintAllNodes will take each shard node (n) and will mint all nodes that have their pk managed by the iterating node n
func MintAllNodes(nodes []*TestProcessorNode, value *big.Int) {
	for idx, n := range nodes {
		if n.ShardCoordinator.SelfId() == core.MetachainShardId {
			continue
		}

		mintAddressesFromSameShard(nodes, idx, value)
	}
}

func mintAddressesFromSameShard(nodes []*TestProcessorNode, targetNodeIdx int, value *big.Int) {
	targetNode := nodes[targetNodeIdx]

	for _, n := range nodes {
		shardId := targetNode.ShardCoordinator.ComputeId(n.OwnAccount.Address)
		if shardId != targetNode.ShardCoordinator.SelfId() {
			continue
		}

		n.OwnAccount.Balance = big.NewInt(0).Set(value)
		MintAddress(targetNode.AccntState, n.OwnAccount.Address, value)
	}
}

// MintAllPlayers mints addresses for all players
func MintAllPlayers(nodes []*TestProcessorNode, players []*TestWalletAccount, value *big.Int) {
	shardCoordinator := nodes[0].ShardCoordinator

	for _, player := range players {
		pShardId := shardCoordinator.ComputeId(player.Address)

		for _, n := range nodes {
			if pShardId != n.ShardCoordinator.SelfId() {
				continue
			}

			MintAddress(n.AccntState, player.Address, value)
			player.Balance = big.NewInt(0).Set(value)
		}
	}
}

// IncrementAndPrintRound increments the given variable, and prints the message for the beginning of the round
func IncrementAndPrintRound(round uint64) uint64 {
	round++
	fmt.Printf("#################################### ROUND %d BEGINS ####################################\n\n", round)

	return round
}

// ProposeBlock proposes a block for every shard
func ProposeBlock(nodes []*TestProcessorNode, idxProposers []int, round uint64, nonce uint64) {
	fmt.Println("All shards propose blocks...")

	stepDelayAdjustment := StepDelay * time.Duration(1+len(nodes)/3)

	for idx, n := range nodes {
		if !IsIntInSlice(idx, idxProposers) {
			continue
		}

		body, header, _ := n.ProposeBlock(round, nonce)
		n.BroadcastBlock(body, header)
		n.CommitBlock(body, header)
	}

	fmt.Println("Delaying for disseminating headers and miniblocks...")
	time.Sleep(stepDelayAdjustment)
	fmt.Println(MakeDisplayTable(nodes))
}

// SyncBlock synchronizes the proposed block in all the other shard nodes
func SyncBlock(
	t *testing.T,
	nodes []*TestProcessorNode,
	idxProposers []int,
	round uint64,
) {

	fmt.Println("All other shard nodes sync the proposed block...")
	for idx, n := range nodes {
		if IsIntInSlice(idx, idxProposers) {
			continue
		}

		err := n.SyncNode(round)
		if err != nil {
			log.Warn(fmt.Sprintf("SyncNode on round %v could not be synced. Error: %s", round, err.Error()))
			assert.Fail(t, err.Error())
			return
		}
	}

	time.Sleep(StepDelay)
	fmt.Println(MakeDisplayTable(nodes))
}

// IsIntInSlice returns true if idx is found on any position in the provided slice
func IsIntInSlice(idx int, slice []int) bool {
	for _, value := range slice {
		if value == idx {
			return true
		}
	}

	return false
}

// Uint32InSlice checks if a uint32 value is in a slice
func Uint32InSlice(searched uint32, list []uint32) bool {
	for _, val := range list {
		if val == searched {
			return true
		}
	}
	return false
}

// CheckRootHashes checks the root hash of the proposer in every shard
func CheckRootHashes(t *testing.T, nodes []*TestProcessorNode, idxProposers []int) {
	for _, idx := range idxProposers {
		checkRootHashInShard(t, nodes, idx)
	}
}

func checkRootHashInShard(t *testing.T, nodes []*TestProcessorNode, idxProposer int) {
	proposerNode := nodes[idxProposer]
	proposerRootHash, _ := proposerNode.AccntState.RootHash()

	for i := 0; i < len(nodes); i++ {
		n := nodes[i]

		if n.ShardCoordinator.SelfId() != proposerNode.ShardCoordinator.SelfId() {
			continue
		}

		fmt.Printf("Testing roothash for node index %d, shard ID %d...\n", i, n.ShardCoordinator.SelfId())
		nodeRootHash, _ := n.AccntState.RootHash()
		assert.Equal(t, proposerRootHash, nodeRootHash)
	}
}

// CheckTxPresentAndRightNonce verifies that the nonce was updated correctly after the exec of bulk txs
func CheckTxPresentAndRightNonce(
	t *testing.T,
	startingNonce uint64,
	noOfTxs int,
	txHashes [][]byte,
	txs []data.TransactionHandler,
	cache dataRetriever.ShardedDataCacherNotifier,
	shardCoordinator sharding.Coordinator,
) {

	if noOfTxs != len(txHashes) {
		for i := startingNonce; i < startingNonce+uint64(noOfTxs); i++ {
			found := false

			for _, txHandler := range txs {
				nonce := extractUint64ValueFromTxHandler(txHandler)
				if nonce == i {
					found = true
					break
				}
			}

			if !found {
				fmt.Printf("unsigned tx with nonce %d is missing\n", i)
			}
		}
		assert.Fail(t, fmt.Sprintf("should have been %d, got %d", noOfTxs, len(txHashes)))

		return
	}

	bitmap := make([]bool, noOfTxs+int(startingNonce))
	//set for each nonce from found tx a true flag in bitmap
	for i := 0; i < noOfTxs; i++ {
		selfId := shardCoordinator.SelfId()
		shardDataStore := cache.ShardDataStore(process.ShardCacherIdentifier(selfId, selfId))
		val, _ := shardDataStore.Get(txHashes[i])
		if val == nil {
			continue
		}

		nonce := extractUint64ValueFromTxHandler(val.(data.TransactionHandler))
		bitmap[nonce] = true
	}

	//for the first startingNonce values, the bitmap should be false
	//for the rest, true
	for i := 0; i < noOfTxs+int(startingNonce); i++ {
		if i < int(startingNonce) {
			assert.False(t, bitmap[i])
			continue
		}

		assert.True(t, bitmap[i])
	}
}

func extractUint64ValueFromTxHandler(txHandler data.TransactionHandler) uint64 {
	tx, ok := txHandler.(*transaction.Transaction)
	if ok {
		return tx.Nonce
	}

	buff := txHandler.GetData()
	return binary.BigEndian.Uint64(buff)
}

// CreateNodes creates multiple nodes in different shards
func CreateNodes(
	numOfShards int,
	nodesPerShard int,
	numMetaChainNodes int,
	serviceID string,
) []*TestProcessorNode {
	nodes := make([]*TestProcessorNode, numOfShards*nodesPerShard+numMetaChainNodes)

	idx := 0
	for shardId := uint32(0); shardId < uint32(numOfShards); shardId++ {
		for j := 0; j < nodesPerShard; j++ {
			n := NewTestProcessorNode(uint32(numOfShards), shardId, shardId, serviceID)
			nodes[idx] = n
			idx++
		}
	}

	for i := 0; i < numMetaChainNodes; i++ {
		metaNode := NewTestProcessorNode(uint32(numOfShards), core.MetachainShardId, 0, serviceID)
		idx = i + numOfShards*nodesPerShard
		nodes[idx] = metaNode
	}

	return nodes
}

// CreateNodesWithCustomStateCheckpointModulus creates multiple nodes in different shards with custom stateCheckpointModulus
func CreateNodesWithCustomStateCheckpointModulus(
	numOfShards int,
	nodesPerShard int,
	numMetaChainNodes int,
	serviceID string,
	stateCheckpointModulus uint,
) []*TestProcessorNode {
	nodes := make([]*TestProcessorNode, numOfShards*nodesPerShard+numMetaChainNodes)

	idx := 0
	for shardId := uint32(0); shardId < uint32(numOfShards); shardId++ {
		for j := 0; j < nodesPerShard; j++ {
			n := NewTestProcessorNodeWithStateCheckpointModulus(uint32(numOfShards), shardId, shardId, serviceID, stateCheckpointModulus)

			nodes[idx] = n
			idx++
		}
	}

	for i := 0; i < numMetaChainNodes; i++ {
		metaNode := NewTestProcessorNodeWithStateCheckpointModulus(uint32(numOfShards), core.MetachainShardId, 0, serviceID, stateCheckpointModulus)
		idx = i + numOfShards*nodesPerShard
		nodes[idx] = metaNode
	}

	return nodes
}

// DisplayAndStartNodes prints each nodes shard ID, sk and pk, and then starts the node
func DisplayAndStartNodes(nodes []*TestProcessorNode) {
	for _, n := range nodes {
		skTxBuff, _ := n.OwnAccount.SkTxSign.ToByteArray()
		pkTxBuff, _ := n.OwnAccount.PkTxSign.ToByteArray()
		pkNode := n.NodesCoordinator.GetOwnPublicKey()

		fmt.Printf("Shard ID: %v, pkNode: %s\n",
			n.ShardCoordinator.SelfId(),
			TestValidatorPubkeyConverter.Encode(pkNode))

		fmt.Printf("skTx: %s, pkTx: %s\n",
			hex.EncodeToString(skTxBuff),
			TestAddressPubkeyConverter.Encode(pkTxBuff),
		)
		_ = n.Messenger.Bootstrap()
	}

	fmt.Println("Delaying for node bootstrap and topic announcement...")
	time.Sleep(P2pBootstrapDelay)
}

// SetEconomicsParameters will set maxGasLimitPerBlock, minGasPrice and minGasLimits to provided nodes
func SetEconomicsParameters(nodes []*TestProcessorNode, maxGasLimitPerBlock uint64, minGasPrice uint64, minGasLimit uint64) {
	for _, n := range nodes {
		n.EconomicsData.SetMaxGasLimitPerBlock(maxGasLimitPerBlock)
		n.EconomicsData.SetMinGasPrice(minGasPrice)
		n.EconomicsData.SetMinGasLimit(minGasLimit)
	}
}

// GenerateAndDisseminateTxs generates and sends multiple txs
func GenerateAndDisseminateTxs(
	n *TestProcessorNode,
	senders []crypto.PrivateKey,
	receiversPublicKeysMap map[uint32][]crypto.PublicKey,
	valToTransfer *big.Int,
	gasPrice uint64,
	gasLimit uint64,
) {

	for i := 0; i < len(senders); i++ {
		senderKey := senders[i]
		incrementalNonce := make([]uint64, len(senders))
		for _, shardReceiversPublicKeys := range receiversPublicKeysMap {
			receiverPubKey := shardReceiversPublicKeys[i]
			tx := GenerateTransferTx(incrementalNonce[i], senderKey, receiverPubKey, valToTransfer, gasPrice, gasLimit)
			_, _ = n.SendTransaction(tx)
			incrementalNonce[i]++
		}
	}
}

// CreateSendersWithInitialBalances creates a map of 1 sender per shard with an initial balance
func CreateSendersWithInitialBalances(
	nodesMap map[uint32][]*TestProcessorNode,
	mintValue *big.Int,
) map[uint32][]crypto.PrivateKey {

	sendersPrivateKeys := make(map[uint32][]crypto.PrivateKey)
	for shardId, nodes := range nodesMap {
		if shardId == core.MetachainShardId {
			continue
		}

		sendersPrivateKeys[shardId], _ = CreateSendersAndReceiversInShard(
			nodes[0],
			1,
		)

		fmt.Println("Minting sender addresses...")
		CreateMintingForSenders(
			nodes,
			shardId,
			sendersPrivateKeys[shardId],
			mintValue,
		)
	}

	return sendersPrivateKeys
}

// CreateAndSendTransaction will generate a transaction with provided parameters, sign it with the provided
// node's tx sign private key and send it on the transaction topic
func CreateAndSendTransaction(
	node *TestProcessorNode,
	txValue *big.Int,
	rcvAddress []byte,
	txData string,
) {
	tx := &transaction.Transaction{
		Nonce:    node.OwnAccount.Nonce,
		Value:    new(big.Int).Set(txValue),
		SndAddr:  node.OwnAccount.Address,
		RcvAddr:  rcvAddress,
		Data:     []byte(txData),
		GasPrice: MinTxGasPrice,
		GasLimit: MinTxGasLimit*1000 + uint64(len(txData)),
	}

	txBuff, _ := tx.GetDataForSigning(TestAddressPubkeyConverter, TestTxSignMarshalizer)
	tx.Signature, _ = node.OwnAccount.SingleSigner.Sign(node.OwnAccount.SkTxSign, txBuff)

	_, err := node.SendTransaction(tx)
	if err != nil {
		log.Warn("could not create transaction", "address", node.OwnAccount.Address, "error", err)
	}
	node.OwnAccount.Nonce++
}

// CreateAndSendTransactionWithGasLimit generates and send a transaction with provided gas limit/gas price
func CreateAndSendTransactionWithGasLimit(
	node *TestProcessorNode,
	txValue *big.Int,
	gasLimit uint64,
	rcvAddress []byte,
	txData []byte,
) {
	tx := &transaction.Transaction{
		Nonce:    node.OwnAccount.Nonce,
		Value:    txValue,
		SndAddr:  node.OwnAccount.Address,
		RcvAddr:  rcvAddress,
		Data:     txData,
		GasPrice: MinTxGasPrice,
		GasLimit: gasLimit,
	}

	txBuff, _ := tx.GetDataForSigning(TestAddressPubkeyConverter, TestTxSignMarshalizer)
	tx.Signature, _ = node.OwnAccount.SingleSigner.Sign(node.OwnAccount.SkTxSign, txBuff)

	_, _ = node.SendTransaction(tx)
	node.OwnAccount.Nonce++
}

type txArgs struct {
	nonce    uint64
	value    *big.Int
	rcvAddr  []byte
	sndAddr  []byte
	data     string
	gasPrice uint64
	gasLimit uint64
}

// GenerateTransferTx will generate a move balance transaction
func GenerateTransferTx(
	nonce uint64,
	senderPrivateKey crypto.PrivateKey,
	receiverPublicKey crypto.PublicKey,
	valToTransfer *big.Int,
	gasPrice uint64,
	gasLimit uint64,
) *transaction.Transaction {

	receiverPubKeyBytes, _ := receiverPublicKey.ToByteArray()
	tx := transaction.Transaction{
		Nonce:    nonce,
		Value:    new(big.Int).Set(valToTransfer),
		RcvAddr:  receiverPubKeyBytes,
		SndAddr:  skToPk(senderPrivateKey),
		Data:     []byte(""),
		GasLimit: gasLimit,
		GasPrice: gasPrice,
	}
	txBuff, _ := tx.GetDataForSigning(TestAddressPubkeyConverter, TestTxSignMarshalizer)
	signer := &ed25519SingleSig.Ed25519Signer{}
	tx.Signature, _ = signer.Sign(senderPrivateKey, txBuff)

	return &tx
}

func generateTx(
	skSign crypto.PrivateKey,
	signer crypto.SingleSigner,
	args *txArgs,
) *transaction.Transaction {
	tx := &transaction.Transaction{
		Nonce:    args.nonce,
		Value:    new(big.Int).Set(args.value),
		RcvAddr:  args.rcvAddr,
		SndAddr:  args.sndAddr,
		GasPrice: args.gasPrice,
		GasLimit: args.gasLimit,
		Data:     []byte(args.data),
	}
	txBuff, _ := tx.GetDataForSigning(TestAddressPubkeyConverter, TestTxSignMarshalizer)
	tx.Signature, _ = signer.Sign(skSign, txBuff)

	return tx
}

func skToPk(sk crypto.PrivateKey) []byte {
	pkBuff, _ := sk.GeneratePublic().ToByteArray()
	return pkBuff
}

// TestPublicKeyHasBalance checks if the account corresponding to the given public key has the expected balance
func TestPublicKeyHasBalance(t *testing.T, n *TestProcessorNode, pk crypto.PublicKey, expectedBalance *big.Int) {
	pkBuff, _ := pk.ToByteArray()
	account, _ := n.AccntState.GetExistingAccount(pkBuff)
	assert.Equal(t, expectedBalance, account.(state.UserAccountHandler).GetBalance())
}

// TestPrivateKeyHasBalance checks if the private key has the expected balance
func TestPrivateKeyHasBalance(t *testing.T, n *TestProcessorNode, sk crypto.PrivateKey, expectedBalance *big.Int) {
	pkBuff, _ := sk.GeneratePublic().ToByteArray()
	account, _ := n.AccntState.GetExistingAccount(pkBuff)
	assert.Equal(t, expectedBalance, account.(state.UserAccountHandler).GetBalance())
}

// GetMiniBlocksHashesFromShardIds returns miniblock hashes from body
func GetMiniBlocksHashesFromShardIds(body *dataBlock.Body, shardIds ...uint32) [][]byte {
	var hashes [][]byte

	for _, miniblock := range body.MiniBlocks {
		for _, shardId := range shardIds {
			if miniblock.ReceiverShardID == shardId {
				buff, _ := TestMarshalizer.Marshal(miniblock)
				hashes = append(hashes, TestHasher.Compute(string(buff)))
			}
		}
	}

	return hashes
}

// GenerateIntraShardTransactions generates intra shard transactions
func GenerateIntraShardTransactions(
	nodesMap map[uint32][]*TestProcessorNode,
	nbTxsPerShard uint32,
	mintValue *big.Int,
	valToTransfer *big.Int,
	gasPrice uint64,
	gasLimit uint64,
) {
	sendersPrivateKeys := make(map[uint32][]crypto.PrivateKey)
	receiversPublicKeys := make(map[uint32][]crypto.PublicKey)

	for shardId, nodes := range nodesMap {
		if shardId == core.MetachainShardId {
			continue
		}

		sendersPrivateKeys[shardId], receiversPublicKeys[shardId] = CreateSendersAndReceiversInShard(
			nodes[0],
			nbTxsPerShard,
		)

		fmt.Println("Minting sender addresses...")
		CreateMintingForSenders(
			nodes,
			shardId,
			sendersPrivateKeys[shardId],
			mintValue,
		)
	}

	CreateAndSendTransactions(
		nodesMap,
		sendersPrivateKeys,
		receiversPublicKeys,
		gasPrice,
		gasLimit,
		valToTransfer,
	)
}

// GenerateSkAndPkInShard generates and returns a private and a public key that reside in a given shard.
// It also returns the key generator
func GenerateSkAndPkInShard(
	coordinator sharding.Coordinator,
	shardId uint32,
) (crypto.PrivateKey, crypto.PublicKey, crypto.KeyGenerator) {
	suite := ed25519.NewEd25519()
	keyGen := signing.NewKeyGenerator(suite)
	sk, pk := keyGen.GeneratePair()

	if shardId == core.MetachainShardId {
		// for metachain generate in shard 0
		shardId = 0
	}

	for {
		pkBytes, _ := pk.ToByteArray()
		if coordinator.ComputeId(pkBytes) == shardId {
			break
		}
		sk, pk = keyGen.GeneratePair()
	}

	return sk, pk, keyGen
}

// CreateSendersAndReceiversInShard creates given number of sender private key and receiver public key pairs,
// with account in same shard as given node
func CreateSendersAndReceiversInShard(
	nodeInShard *TestProcessorNode,
	nbSenderReceiverPairs uint32,
) ([]crypto.PrivateKey, []crypto.PublicKey) {
	shardId := nodeInShard.ShardCoordinator.SelfId()
	receiversPublicKeys := make([]crypto.PublicKey, nbSenderReceiverPairs)
	sendersPrivateKeys := make([]crypto.PrivateKey, nbSenderReceiverPairs)

	for i := uint32(0); i < nbSenderReceiverPairs; i++ {
		sendersPrivateKeys[i], _, _ = GenerateSkAndPkInShard(nodeInShard.ShardCoordinator, shardId)
		_, receiversPublicKeys[i], _ = GenerateSkAndPkInShard(nodeInShard.ShardCoordinator, shardId)
	}

	return sendersPrivateKeys, receiversPublicKeys
}

// CreateAndSendTransactions creates and sends transactions between given senders and receivers.
func CreateAndSendTransactions(
	nodes map[uint32][]*TestProcessorNode,
	sendersPrivKeysMap map[uint32][]crypto.PrivateKey,
	receiversPubKeysMap map[uint32][]crypto.PublicKey,
	gasPricePerTx uint64,
	gasLimitPerTx uint64,
	valueToTransfer *big.Int,
) {
	for shardId := range nodes {
		if shardId == core.MetachainShardId {
			continue
		}

		nodeInShard := nodes[shardId][0]

		fmt.Println("Generating transactions...")
		GenerateAndDisseminateTxs(
			nodeInShard,
			sendersPrivKeysMap[shardId],
			receiversPubKeysMap,
			valueToTransfer,
			gasPricePerTx,
			gasLimitPerTx,
		)
	}

	fmt.Println("Delaying for disseminating transactions...")
	time.Sleep(time.Second * 5)
}

// CreateMintingForSenders creates account with balances for every node in a given shard
func CreateMintingForSenders(
	nodes []*TestProcessorNode,
	senderShard uint32,
	sendersPrivateKeys []crypto.PrivateKey,
	value *big.Int,
) {

	for _, n := range nodes {
		//only sender shard nodes will be minted
		if n.ShardCoordinator.SelfId() != senderShard {
			continue
		}

		for _, sk := range sendersPrivateKeys {
			pkBuff, _ := sk.GeneratePublic().ToByteArray()
			account, _ := n.AccntState.LoadAccount(pkBuff)
			_ = account.(state.UserAccountHandler).AddToBalance(value)
			_ = n.AccntState.SaveAccount(account)
		}

		_, _ = n.AccntState.Commit()
	}
}

// CreateMintingFromAddresses creates account with balances for given address
func CreateMintingFromAddresses(
	nodes []*TestProcessorNode,
	addresses [][]byte,
	value *big.Int,
) {
	for _, n := range nodes {
		for _, address := range addresses {
			MintAddress(n.AccntState, address, value)
		}
	}
}

// ProposeBlockSignalsEmptyBlock proposes and broadcasts a block
func ProposeBlockSignalsEmptyBlock(
	node *TestProcessorNode,
	round uint64,
	nonce uint64,
) (data.HeaderHandler, data.BodyHandler, bool) {

	fmt.Println("Proposing block without commit...")

	body, header, txHashes := node.ProposeBlock(round, nonce)
	node.BroadcastBlock(body, header)
	isEmptyBlock := len(txHashes) == 0

	fmt.Println("Delaying for disseminating headers and miniblocks...")
	time.Sleep(StepDelay)

	return header, body, isEmptyBlock
}

// CreateAccountForNodes creates accounts for each node and commits the accounts state
func CreateAccountForNodes(nodes []*TestProcessorNode) {
	for i := 0; i < len(nodes); i++ {
		CreateAccountForNode(nodes[i])
	}
}

// CreateAccountForNode creates an account for the given node
func CreateAccountForNode(node *TestProcessorNode) {
	acc, _ := node.AccntState.LoadAccount(node.OwnAccount.PkTxSignBytes)
	_ = node.AccntState.SaveAccount(acc)
	_, _ = node.AccntState.Commit()
}

// ComputeAndRequestMissingTransactions computes missing transactions for each node, and requests them
func ComputeAndRequestMissingTransactions(
	nodes []*TestProcessorNode,
	generatedTxHashes [][]byte,
	shardResolver uint32,
	shardRequesters ...uint32,
) {
	for _, n := range nodes {
		if !Uint32InSlice(n.ShardCoordinator.SelfId(), shardRequesters) {
			continue
		}

		neededTxs := getMissingTxsForNode(n, generatedTxHashes)
		requestMissingTransactions(n, shardResolver, neededTxs)
	}
}

func getMissingTxsForNode(n *TestProcessorNode, generatedTxHashes [][]byte) [][]byte {
	var neededTxs [][]byte

	for i := 0; i < len(generatedTxHashes); i++ {
		_, ok := n.DataPool.Transactions().SearchFirstData(generatedTxHashes[i])
		if !ok {
			neededTxs = append(neededTxs, generatedTxHashes[i])
		}
	}

	return neededTxs
}

func requestMissingTransactions(n *TestProcessorNode, shardResolver uint32, neededTxs [][]byte) {
	txResolver, _ := n.ResolverFinder.CrossShardResolver(procFactory.TransactionTopic, shardResolver)

	for i := 0; i < len(neededTxs); i++ {
		_ = txResolver.RequestDataFromHash(neededTxs[i], 0)
	}
}

// CreateRequesterDataPool creates a datapool with a mock txPool
func CreateRequesterDataPool(recvTxs map[int]map[string]struct{}, mutRecvTxs *sync.Mutex, nodeIndex int, selfShardID uint32) dataRetriever.PoolsHolder {
	//not allowed to request data from the same shard
	return CreateTestDataPool(&mock.ShardedDataStub{
		SearchFirstDataCalled: func(key []byte) (value interface{}, ok bool) {
			return nil, false
		},
		ShardDataStoreCalled: func(cacheId string) (c storage.Cacher) {
			return nil
		},
		AddDataCalled: func(key []byte, data interface{}, sizeInBytes int, cacheId string) {
			mutRecvTxs.Lock()
			defer mutRecvTxs.Unlock()

			txMap := recvTxs[nodeIndex]
			if txMap == nil {
				txMap = make(map[string]struct{})
				recvTxs[nodeIndex] = txMap
			}

			txMap[string(key)] = struct{}{}
		},
		RegisterHandlerCalled: func(i func(key []byte, value interface{})) {
		},
	}, selfShardID)
}

// CreateResolversDataPool creates a datapool containing a given number of transactions
func CreateResolversDataPool(
	t *testing.T,
	maxTxs int,
	senderShardID uint32,
	recvShardId uint32,
	shardCoordinator sharding.Coordinator,
) (dataRetriever.PoolsHolder, [][]byte, [][]byte) {

	txHashes := make([][]byte, maxTxs)
	txsSndAddr := make([][]byte, 0)
	txPool, _ := createTxPool(shardCoordinator.SelfId())

	for i := 0; i < maxTxs; i++ {
		tx, txHash := generateValidTx(t, shardCoordinator, senderShardID, recvShardId)
		cacherIdentifier := process.ShardCacherIdentifier(1, 0)
		txPool.AddData(txHash, tx, tx.Size(), cacherIdentifier)
		txHashes[i] = txHash
		txsSndAddr = append(txsSndAddr, tx.SndAddr)
	}

	return CreateTestDataPool(txPool, shardCoordinator.SelfId()), txHashes, txsSndAddr
}

func generateValidTx(
	t *testing.T,
	shardCoordinator sharding.Coordinator,
	senderShardId uint32,
	receiverShardId uint32,
) (*transaction.Transaction, []byte) {

	skSender, pkSender, _ := GenerateSkAndPkInShard(shardCoordinator, senderShardId)
	pkSenderBuff, _ := pkSender.ToByteArray()

	_, pkRecv, _ := GenerateSkAndPkInShard(shardCoordinator, receiverShardId)
	pkRecvBuff, _ := pkRecv.ToByteArray()

	trieStorage, _ := CreateTrieStorageManager()
	accnts, _ := CreateAccountsDB(UserAccount, trieStorage)
	acc, _ := accnts.LoadAccount(pkSenderBuff)
	_ = accnts.SaveAccount(acc)
	_, _ = accnts.Commit()

	txAccumulator, _ := accumulator.NewTimeAccumulator(time.Millisecond*10, time.Millisecond)
	mockNode, _ := node.NewNode(
		node.WithInternalMarshalizer(TestMarshalizer, 100),
		node.WithVmMarshalizer(TestVmMarshalizer),
		node.WithTxSignMarshalizer(TestTxSignMarshalizer),
		node.WithHasher(TestHasher),
		node.WithAddressPubkeyConverter(TestAddressPubkeyConverter),
		node.WithValidatorPubkeyConverter(TestValidatorPubkeyConverter),
		node.WithKeyGen(signing.NewKeyGenerator(ed25519.NewEd25519())),
		node.WithTxSingleSigner(&ed25519SingleSig.Ed25519Signer{}),
		node.WithAccountsAdapter(accnts),
		node.WithTxAccumulator(txAccumulator),
	)

	tx, err := mockNode.GenerateTransaction(
		TestAddressPubkeyConverter.Encode(pkSenderBuff),
		TestAddressPubkeyConverter.Encode(pkRecvBuff),
		big.NewInt(1),
		"",
		skSender,
	)
	assert.Nil(t, err)

	txBuff, _ := TestMarshalizer.Marshal(tx)
	txHash := TestHasher.Compute(string(txBuff))

	return tx, txHash
}

// GetNumTxsWithDst returns the total number of transactions that have a certain destination shard
func GetNumTxsWithDst(dstShardId uint32, dataPool dataRetriever.PoolsHolder, nrShards uint32) int {
	txPool := dataPool.Transactions()
	if txPool == nil {
		return 0
	}

	sumTxs := 0

	for i := uint32(0); i < nrShards; i++ {
		strCache := process.ShardCacherIdentifier(i, dstShardId)
		txStore := txPool.ShardDataStore(strCache)
		if txStore == nil {
			continue
		}
		sumTxs += txStore.Len()
	}

	return sumTxs
}

// ProposeAndSyncBlocks proposes and syncs blocks until all transaction pools are empty
func ProposeAndSyncBlocks(
	t *testing.T,
	nodes []*TestProcessorNode,
	idxProposers []int,
	round uint64,
	nonce uint64,
) (uint64, uint64) {

	// if there are many transactions, they might not fit into the block body in only one round
	for {
		numTxsInPool := 0
		round, nonce = ProposeAndSyncOneBlock(t, nodes, idxProposers, round, nonce)

		for _, idProposer := range idxProposers {
			proposerNode := nodes[idProposer]
			numTxsInPool = GetNumTxsWithDst(
				proposerNode.ShardCoordinator.SelfId(),
				proposerNode.DataPool,
				proposerNode.ShardCoordinator.NumberOfShards(),
			)

			if numTxsInPool > 0 {
				break
			}
		}

		if numTxsInPool == 0 {
			break
		}
	}

	if nodes[0].ShardCoordinator.NumberOfShards() == 1 {
		return round, nonce
	}

	// cross shard smart contract call is first processed at sender shard, notarized by metachain, processed at
	// shard with smart contract, smart contract result is notarized by metachain, then finally processed at the
	// sender shard
	numberToPropagateToEveryShard := 5
	for i := 0; i < numberToPropagateToEveryShard; i++ {
		round, nonce = ProposeAndSyncOneBlock(t, nodes, idxProposers, round, nonce)
	}

	return round, nonce
}

// ProposeAndSyncOneBlock proposes a block, syncs the block and then increments the round
func ProposeAndSyncOneBlock(
	t *testing.T,
	nodes []*TestProcessorNode,
	idxProposers []int,
	round uint64,
	nonce uint64,
) (uint64, uint64) {

	UpdateRound(nodes, round)
	ProposeBlock(nodes, idxProposers, round, nonce)
	SyncBlock(t, nodes, idxProposers, round)
	round = IncrementAndPrintRound(round)
	nonce++

	return round, nonce
}

// WaitForBootstrapAndShowConnected will delay a given duration in order to wait for bootstraping  and print the
// number of peers that each node is connected to
func WaitForBootstrapAndShowConnected(peers []p2p.Messenger, durationBootstrapingTime time.Duration) {
	fmt.Printf("Waiting %v for peer discovery...\n", durationBootstrapingTime)
	time.Sleep(durationBootstrapingTime)

	fmt.Println("Connected peers:")
	for _, peer := range peers {
		fmt.Printf("Peer %s is connected to %d peers\n", peer.ID().Pretty(), len(peer.ConnectedPeers()))
	}
}

// PubKeysMapFromKeysMap returns a map of public keys per shard from the key pairs per shard map.
func PubKeysMapFromKeysMap(keyPairMap map[uint32][]*TestKeyPair) map[uint32][]string {
	keysMap := make(map[uint32][]string)

	for shardId, pairList := range keyPairMap {
		shardKeys := make([]string, len(pairList))
		for i, pair := range pairList {
			b, _ := pair.Pk.ToByteArray()
			shardKeys[i] = string(b)
		}
		keysMap[shardId] = shardKeys
	}

	return keysMap
}

// GenValidatorsFromPubKeys generates a map of validators per shard out of public keys map
func GenValidatorsFromPubKeys(pubKeysMap map[uint32][]string, _ uint32) map[uint32][]sharding.GenesisNodeInfoHandler {
	validatorsMap := make(map[uint32][]sharding.GenesisNodeInfoHandler)

	for shardId, shardNodesPks := range pubKeysMap {
		var shardValidators []sharding.GenesisNodeInfoHandler
		for i := 0; i < len(shardNodesPks); i++ {
			v := mock.NewNodeInfo([]byte(shardNodesPks[i][:32]), []byte(shardNodesPks[i]), shardId)
			shardValidators = append(shardValidators, v)
		}
		validatorsMap[shardId] = shardValidators
	}

	return validatorsMap
}

// GenValidatorsFromPubKeysAndTxPubKeys generates a map of validators per shard out of public keys map
func GenValidatorsFromPubKeysAndTxPubKeys(
	blsPubKeysMap map[uint32][]string,
	txPubKeysMap map[uint32][]string,
) map[uint32][]sharding.GenesisNodeInfoHandler {
	validatorsMap := make(map[uint32][]sharding.GenesisNodeInfoHandler)

	for shardId, shardNodesPks := range blsPubKeysMap {
		var shardValidators []sharding.GenesisNodeInfoHandler
		for i := 0; i < len(shardNodesPks); i++ {
			v := mock.NewNodeInfo([]byte(txPubKeysMap[shardId][i]), []byte(shardNodesPks[i]), shardId)
			shardValidators = append(shardValidators, v)
		}
		validatorsMap[shardId] = shardValidators
	}

	return validatorsMap
}

// CreateCryptoParams generates the crypto parameters (key pairs, key generator and suite) for multiple nodes
func CreateCryptoParams(nodesPerShard int, nbMetaNodes int, nbShards uint32) *CryptoParams {
	txSuite := ed25519.NewEd25519()
	txKeyGen := signing.NewKeyGenerator(txSuite)
	suite := mcl.NewSuiteBLS12()
	singleSigner := &ed25519SingleSig.Ed25519Signer{}
	keyGen := signing.NewKeyGenerator(suite)

	txKeysMap := make(map[uint32][]*TestKeyPair)
	keysMap := make(map[uint32][]*TestKeyPair)
	for shardId := uint32(0); shardId < nbShards; shardId++ {
		txKeyPairs := make([]*TestKeyPair, nodesPerShard)
		keyPairs := make([]*TestKeyPair, nodesPerShard)
		for n := 0; n < nodesPerShard; n++ {
			kp := &TestKeyPair{}
			kp.Sk, kp.Pk = keyGen.GeneratePair()
			keyPairs[n] = kp

			txKp := &TestKeyPair{}
			txKp.Sk, txKp.Pk = txKeyGen.GeneratePair()
			txKeyPairs[n] = txKp
		}
		keysMap[shardId] = keyPairs
		txKeysMap[shardId] = txKeyPairs
	}

	txKeyPairs := make([]*TestKeyPair, nbMetaNodes)
	keyPairs := make([]*TestKeyPair, nbMetaNodes)
	for n := 0; n < nbMetaNodes; n++ {
		kp := &TestKeyPair{}
		kp.Sk, kp.Pk = keyGen.GeneratePair()
		keyPairs[n] = kp

		txKp := &TestKeyPair{}
		txKp.Sk, txKp.Pk = txKeyGen.GeneratePair()
		txKeyPairs[n] = txKp
	}
	keysMap[core.MetachainShardId] = keyPairs
	txKeysMap[core.MetachainShardId] = txKeyPairs

	params := &CryptoParams{
		Keys:         keysMap,
		KeyGen:       keyGen,
		SingleSigner: singleSigner,
		TxKeyGen:     txKeyGen,
		TxKeys:       txKeysMap,
	}

	return params
}

// CloseProcessorNodes closes the used TestProcessorNodes and advertiser
func CloseProcessorNodes(nodes []*TestProcessorNode, advertiser p2p.Messenger) {
	_ = advertiser.Close()
	for _, n := range nodes {
		_ = n.Messenger.Close()
	}
}

// StartP2PBootstrapOnProcessorNodes will start the p2p discovery on processor nodes and wait a predefined time
func StartP2PBootstrapOnProcessorNodes(nodes []*TestProcessorNode) {
	for _, n := range nodes {
		_ = n.Messenger.Bootstrap()
	}

	fmt.Println("Delaying for nodes p2p bootstrap...")
	time.Sleep(P2pBootstrapDelay)
}

// SetupSyncNodesOneShardAndMeta creates nodes with sync capabilities divided into one shard and a metachain
func SetupSyncNodesOneShardAndMeta(
	numNodesPerShard int,
	numNodesMeta int,
) ([]*TestProcessorNode, p2p.Messenger, []int) {

	maxShards := uint32(1)
	shardId := uint32(0)

	advertiser := CreateMessengerWithKadDht("")
	_ = advertiser.Bootstrap()
	advertiserAddr := GetConnectableAddress(advertiser)

	var nodes []*TestProcessorNode
	for i := 0; i < numNodesPerShard; i++ {
		shardNode := NewTestSyncNode(
			maxShards,
			shardId,
			shardId,
			advertiserAddr,
		)
		nodes = append(nodes, shardNode)
	}
	idxProposerShard0 := 0

	for i := 0; i < numNodesMeta; i++ {
		metaNode := NewTestSyncNode(
			maxShards,
			core.MetachainShardId,
			shardId,
			advertiserAddr,
		)
		nodes = append(nodes, metaNode)
	}
	idxProposerMeta := len(nodes) - 1

	idxProposers := []int{idxProposerShard0, idxProposerMeta}

	return nodes, advertiser, idxProposers
}

// StartSyncingBlocks starts the syncing process of all the nodes
func StartSyncingBlocks(nodes []*TestProcessorNode) {
	for _, n := range nodes {
		_ = n.StartSync()
	}

	fmt.Println("Delaying for nodes to start syncing blocks...")
	time.Sleep(StepDelay)
}

// ForkChoiceOneBlock rollbacks a block from the given shard
func ForkChoiceOneBlock(nodes []*TestProcessorNode, shardId uint32) {
	for idx, n := range nodes {
		if n.ShardCoordinator.SelfId() != shardId {
			continue
		}
		err := n.Bootstrapper.RollBack(false)
		if err != nil {
			fmt.Println(err)
		}

		newNonce := n.BlockChain.GetCurrentBlockHeader().GetNonce()
		fmt.Printf("Node's id %d is at block height %d\n", idx, newNonce)
	}
}

// ResetHighestProbableNonce resets the highest probable nonce
func ResetHighestProbableNonce(nodes []*TestProcessorNode, shardId uint32, targetNonce uint64) {
	for _, n := range nodes {
		if n.ShardCoordinator.SelfId() != shardId {
			continue
		}
		if n.BlockChain.GetCurrentBlockHeader().GetNonce() != targetNonce {
			continue
		}

		n.Bootstrapper.SetProbableHighestNonce(targetNonce)
	}
}

// EmptyDataPools clears all the data pools
func EmptyDataPools(nodes []*TestProcessorNode, shardId uint32) {
	for _, n := range nodes {
		if n.ShardCoordinator.SelfId() != shardId {
			continue
		}

		emptyNodeDataPool(n)
	}
}

func emptyNodeDataPool(node *TestProcessorNode) {
	if node.DataPool != nil {
		emptyDataPool(node.DataPool)
	}
}

func emptyDataPool(sdp dataRetriever.PoolsHolder) {
	sdp.Headers().Clear()
	sdp.UnsignedTransactions().Clear()
	sdp.Transactions().Clear()
	sdp.MiniBlocks().Clear()
	sdp.PeerChangesBlocks().Clear()
}

// UpdateRound updates the round for every node
func UpdateRound(nodes []*TestProcessorNode, round uint64) {
	for _, n := range nodes {
		n.Rounder.IndexField = int64(round)
	}
}

// ProposeBlocks proposes blocks for a given number of rounds
func ProposeBlocks(
	nodes []*TestProcessorNode,
	round *uint64,
	idxProposers []int,
	nonces []*uint64,
	numOfRounds int,
) {

	for i := 0; i < numOfRounds; i++ {
		crtRound := atomic.LoadUint64(round)
		proposeBlocks(nodes, idxProposers, nonces, crtRound)

		time.Sleep(SyncDelay)

		crtRound = IncrementAndPrintRound(crtRound)
		atomic.StoreUint64(round, crtRound)
		UpdateRound(nodes, crtRound)
		IncrementNonces(nonces)
	}
	time.Sleep(SyncDelay)
}

// IncrementNonces increments all the nonces
func IncrementNonces(nonces []*uint64) {
	for i := 0; i < len(nonces); i++ {
		atomic.AddUint64(nonces[i], 1)
	}
}

func proposeBlocks(
	nodes []*TestProcessorNode,
	idxProposers []int,
	nonces []*uint64,
	crtRound uint64,
) {
	for idx, proposer := range idxProposers {
		crtNonce := atomic.LoadUint64(nonces[idx])
		ProposeBlock(nodes, []int{proposer}, crtRound, crtNonce)
	}
}

func createTxPool(selfShardID uint32) (dataRetriever.ShardedDataCacherNotifier, error) {
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
			NumberOfShards: 1,
			SelfShardID:    selfShardID,
		},
	)
}

// WaitOperationToBeDone -
func WaitOperationToBeDone(t *testing.T, nodes []*TestProcessorNode, nrOfRounds int, nonce, round uint64, idxProposers []int) (uint64, uint64) {
	for i := 0; i < nrOfRounds; i++ {
		round, nonce = ProposeAndSyncOneBlock(t, nodes, idxProposers, round, nonce)
	}

	return nonce, round
}

// AddSelfNotarizedHeaderByMetachain -
func AddSelfNotarizedHeaderByMetachain(nodes []*TestProcessorNode) {
	for _, n := range nodes {
		if n.ShardCoordinator.SelfId() == core.MetachainShardId {
			continue
		}

		header := n.BlockChain.GetCurrentBlockHeader()
		if check.IfNil(header) {
			continue
		}

		n.BlockTracker.AddSelfNotarizedHeader(core.MetachainShardId, header, nil)
	}
}
