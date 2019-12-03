package factory

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"math/big"
	"path/filepath"
	"strings"
	"time"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/round"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/partitioning"
	"github.com/ElrondNetwork/elrond-go/core/serviceContainer"
	"github.com/ElrondNetwork/elrond-go/core/statistics/softwareVersion"
	factorySoftawareVersion "github.com/ElrondNetwork/elrond-go/core/statistics/softwareVersion/factory"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/crypto/signing"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/kyber"
	blsMultiSig "github.com/ElrondNetwork/elrond-go/crypto/signing/kyber/multisig"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/kyber/singlesig"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/multisig"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/address"
	dataBlock "github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/blockchain"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/state/addressConverters"
	factoryState "github.com/ElrondNetwork/elrond-go/data/state/factory"
	"github.com/ElrondNetwork/elrond-go/data/trie"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters/uint64ByteSlice"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/dataPool"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/factory/containers"
	metafactoryDataRetriever "github.com/ElrondNetwork/elrond-go/dataRetriever/factory/metachain"
	shardfactoryDataRetriever "github.com/ElrondNetwork/elrond-go/dataRetriever/factory/shard"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/requestHandlers"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/shardedData"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/epochStart/genesis"
	metachainEpochStart "github.com/ElrondNetwork/elrond-go/epochStart/metachain"
	"github.com/ElrondNetwork/elrond-go/epochStart/shardchain"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/hashing/blake2b"
	"github.com/ElrondNetwork/elrond-go/hashing/sha256"
	"github.com/ElrondNetwork/elrond-go/logger"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/ntp"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p"
	factoryP2P "github.com/ElrondNetwork/elrond-go/p2p/libp2p/factory"
	"github.com/ElrondNetwork/elrond-go/p2p/loadBalancer"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/process/block/poolsCleaner"
	"github.com/ElrondNetwork/elrond-go/process/block/preprocess"
	"github.com/ElrondNetwork/elrond-go/process/coordinator"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/factory/metachain"
	"github.com/ElrondNetwork/elrond-go/process/factory/shard"
	"github.com/ElrondNetwork/elrond-go/process/peer"
	"github.com/ElrondNetwork/elrond-go/process/rewardTransaction"
	"github.com/ElrondNetwork/elrond-go/process/scToProtocol"
	"github.com/ElrondNetwork/elrond-go/process/smartContract"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/hooks"
	processSync "github.com/ElrondNetwork/elrond-go/process/sync"
	"github.com/ElrondNetwork/elrond-go/process/transaction"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/statusHandler"
	factoryViews "github.com/ElrondNetwork/elrond-go/statusHandler/factory"
	"github.com/ElrondNetwork/elrond-go/statusHandler/view"
	"github.com/ElrondNetwork/elrond-go/storage"
	storageFactory "github.com/ElrondNetwork/elrond-go/storage/factory"
	"github.com/ElrondNetwork/elrond-go/storage/memorydb"
	"github.com/ElrondNetwork/elrond-go/storage/pruning"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/ElrondNetwork/elrond-go/storage/timecache"
	"github.com/btcsuite/btcd/btcec"
	libp2pCrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/urfave/cli"
)

const (
	// BlsHashSize specifies the hash size for using bls scheme
	BlsHashSize = 16

	// BlsConsensusType specifies te signature scheme used in the consensus
	BlsConsensusType = "bls"

	// BnConsensusType specifies te signature scheme used in the consensus
	BnConsensusType = "bn"

	// MaxTxsToRequest specifies the maximum number of txs to request
	MaxTxsToRequest = 100
)

//TODO remove this
var log = logger.GetOrCreate("main")

// MaxTxNonceDeltaAllowed specifies the maximum difference between an account's nonce and a received transaction's nonce
// in order to mark the transaction as valid.
const MaxTxNonceDeltaAllowed = 15000

// ErrCreateForkDetector signals that a fork detector could not be created
//TODO: Extract all others error messages from this file in some defined errors
var ErrCreateForkDetector = errors.New("could not create fork detector")

// timeSpanForBadHeaders is the expiry time for an added block header hash
var timeSpanForBadHeaders = time.Minute * 2

// Network struct holds the network components of the Elrond protocol
type Network struct {
	NetMessenger p2p.Messenger
}

// Core struct holds the core components of the Elrond protocol
type Core struct {
	Hasher                   hashing.Hasher
	Marshalizer              marshal.Marshalizer
	Trie                     data.Trie
	Uint64ByteSliceConverter typeConverters.Uint64ByteSliceConverter
	StatusHandler            core.AppStatusHandler
}

// State struct holds the state components of the Elrond protocol
type State struct {
	AddressConverter  state.AddressConverter
	PeerAccounts      state.AccountsAdapter
	AccountsAdapter   state.AccountsAdapter
	InBalanceForShard map[string]*big.Int
}

// Data struct holds the data components of the Elrond protocol
type Data struct {
	Blkc         data.ChainHandler
	Store        dataRetriever.StorageService
	Datapool     dataRetriever.PoolsHolder
	MetaDatapool dataRetriever.MetaPoolsHolder
}

// Crypto struct holds the crypto components of the Elrond protocol
type Crypto struct {
	TxSingleSigner  crypto.SingleSigner
	SingleSigner    crypto.SingleSigner
	MultiSigner     crypto.MultiSigner
	BlockSignKeyGen crypto.KeyGenerator
	TxSignKeyGen    crypto.KeyGenerator
	TxSignPrivKey   crypto.PrivateKey
	TxSignPubKey    crypto.PublicKey
	InitialPubKeys  map[uint32][]string
}

// Process struct holds the process components of the Elrond protocol
type Process struct {
	InterceptorsContainer process.InterceptorsContainer
	ResolversFinder       dataRetriever.ResolversFinder
	Rounder               consensus.Rounder
	EpochStartTrigger     epochStart.TriggerHandler
	ForkDetector          process.ForkDetector
	BlockProcessor        process.BlockProcessor
	BlackListHandler      process.BlackListHandler
	BootStorer            process.BootStorer
}

type coreComponentsFactoryArgs struct {
	config   *config.Config
	uniqueID string
}

// NewCoreComponentsFactoryArgs initializes the arguments necessary for creating the core components
func NewCoreComponentsFactoryArgs(config *config.Config, uniqueID string) *coreComponentsFactoryArgs {
	return &coreComponentsFactoryArgs{
		config:   config,
		uniqueID: uniqueID,
	}
}

// CoreComponentsFactory creates the core components
func CoreComponentsFactory(args *coreComponentsFactoryArgs) (*Core, error) {
	hasher, err := getHasherFromConfig(args.config)
	if err != nil {
		return nil, errors.New("could not create hasher: " + err.Error())
	}

	marshalizer, err := getMarshalizerFromConfig(args.config)
	if err != nil {
		return nil, errors.New("could not create marshalizer: " + err.Error())
	}

	merkleTrie, err := getTrie(args.config.AccountsTrieStorage, marshalizer, hasher, args.uniqueID)
	if err != nil {
		return nil, errors.New("error creating trie: " + err.Error())
	}
	uint64ByteSliceConverter := uint64ByteSlice.NewBigEndianConverter()

	return &Core{
		Hasher:                   hasher,
		Marshalizer:              marshalizer,
		Trie:                     merkleTrie,
		Uint64ByteSliceConverter: uint64ByteSliceConverter,
		StatusHandler:            statusHandler.NewNilStatusHandler(),
	}, nil
}

type stateComponentsFactoryArgs struct {
	config           *config.Config
	genesisConfig    *sharding.Genesis
	shardCoordinator sharding.Coordinator
	core             *Core
	uniqueID         string
}

// NewStateComponentsFactoryArgs initializes the arguments necessary for creating the state components
func NewStateComponentsFactoryArgs(
	config *config.Config,
	genesisConfig *sharding.Genesis,
	shardCoordinator sharding.Coordinator,
	core *Core,
	uniqueID string,
) *stateComponentsFactoryArgs {
	return &stateComponentsFactoryArgs{
		config:           config,
		genesisConfig:    genesisConfig,
		shardCoordinator: shardCoordinator,
		core:             core,
		uniqueID:         uniqueID,
	}
}

// StateComponentsFactory creates the state components
func StateComponentsFactory(args *stateComponentsFactoryArgs) (*State, error) {
	addressConverter, err := addressConverters.NewPlainAddressConverter(
		args.config.Address.Length,
		args.config.Address.Prefix,
	)

	if err != nil {
		return nil, errors.New("could not create address converter: " + err.Error())
	}

	accountFactory, err := factoryState.NewAccountFactoryCreator(factoryState.UserAccount)
	if err != nil {
		return nil, errors.New("could not create account factory: " + err.Error())
	}

	accountsAdapter, err := state.NewAccountsDB(args.core.Trie, args.core.Hasher, args.core.Marshalizer, accountFactory)
	if err != nil {
		return nil, errors.New("could not create accounts adapter: " + err.Error())
	}

	inBalanceForShard, err := args.genesisConfig.InitialNodesBalances(args.shardCoordinator, addressConverter)
	if err != nil {
		return nil, errors.New("initial balances could not be processed " + err.Error())
	}

	peerAccountsTrie, err := getTrie(
		args.config.PeerAccountsTrieStorage,
		args.core.Marshalizer,
		args.core.Hasher,
		args.uniqueID,
	)
	if err != nil {
		return nil, err
	}

	accountFactory, err = factoryState.NewAccountFactoryCreator(factoryState.ValidatorAccount)
	if err != nil {
		return nil, errors.New("could not create peer account factory: " + err.Error())
	}

	peerAdapter, err := state.NewPeerAccountsDB(peerAccountsTrie, args.core.Hasher, args.core.Marshalizer, accountFactory)
	if err != nil {
		return nil, err
	}

	return &State{
		PeerAccounts:      peerAdapter,
		AddressConverter:  addressConverter,
		AccountsAdapter:   accountsAdapter,
		InBalanceForShard: inBalanceForShard,
	}, nil
}

type dataComponentsFactoryArgs struct {
	config             *config.Config
	shardCoordinator   sharding.Coordinator
	core               *Core
	uniqueID           string
	epochStartNotifier EpochStartNotifier
}

// NewDataComponentsFactoryArgs initializes the arguments necessary for creating the data components
func NewDataComponentsFactoryArgs(
	config *config.Config,
	shardCoordinator sharding.Coordinator,
	core *Core,
	uniqueID string,
	epochStartNotifier EpochStartNotifier,
) *dataComponentsFactoryArgs {
	return &dataComponentsFactoryArgs{
		config:             config,
		shardCoordinator:   shardCoordinator,
		core:               core,
		uniqueID:           uniqueID,
		epochStartNotifier: epochStartNotifier,
	}
}

// DataComponentsFactory creates the data components
func DataComponentsFactory(args *dataComponentsFactoryArgs) (*Data, error) {
	var datapool dataRetriever.PoolsHolder
	var metaDatapool dataRetriever.MetaPoolsHolder
	blkc, err := createBlockChainFromConfig(args.config, args.shardCoordinator, args.core.StatusHandler)
	if err != nil {
		return nil, errors.New("could not create block chain: " + err.Error())
	}

	store, err := createDataStoreFromConfig(args.config, args.shardCoordinator, args.uniqueID, args.epochStartNotifier)
	if err != nil {
		return nil, errors.New("could not create local data store: " + err.Error())
	}

	if args.shardCoordinator.SelfId() < args.shardCoordinator.NumberOfShards() {
		datapool, err = createShardDataPoolFromConfig(args.config, args.core.Uint64ByteSliceConverter)
		if err != nil {
			return nil, errors.New("could not create shard data pools: " + err.Error())
		}
	}
	if args.shardCoordinator.SelfId() == sharding.MetachainShardId {
		metaDatapool, err = createMetaDataPoolFromConfig(args.config, args.core.Uint64ByteSliceConverter)
		if err != nil {
			return nil, errors.New("could not create shard data pools: " + err.Error())
		}
	}
	if datapool == nil && metaDatapool == nil {
		return nil, errors.New("could not create data pools: ")
	}

	return &Data{
		Blkc:         blkc,
		Store:        store,
		Datapool:     datapool,
		MetaDatapool: metaDatapool,
	}, nil
}

type cryptoComponentsFactoryArgs struct {
	ctx                          *cli.Context
	config                       *config.Config
	nodesConfig                  *sharding.NodesSetup
	shardCoordinator             sharding.Coordinator
	keyGen                       crypto.KeyGenerator
	privKey                      crypto.PrivateKey
	log                          logger.Logger
	initialBalancesSkPemFileName string
	txSignSkName                 string
	txSignSkIndexName            string
}

// NewCryptoComponentsFactoryArgs initializes the arguments necessary for creating the crypto components
func NewCryptoComponentsFactoryArgs(
	ctx *cli.Context,
	config *config.Config,
	nodesConfig *sharding.NodesSetup,
	shardCoordinator sharding.Coordinator,
	keyGen crypto.KeyGenerator,
	privKey crypto.PrivateKey,
	log logger.Logger,
	initialBalancesSkPemFileName string,
	txSignSkName string,
	txSignSkIndexName string,
) *cryptoComponentsFactoryArgs {
	return &cryptoComponentsFactoryArgs{
		ctx:                          ctx,
		config:                       config,
		nodesConfig:                  nodesConfig,
		shardCoordinator:             shardCoordinator,
		keyGen:                       keyGen,
		privKey:                      privKey,
		log:                          log,
		initialBalancesSkPemFileName: initialBalancesSkPemFileName,
		txSignSkName:                 txSignSkName,
		txSignSkIndexName:            txSignSkIndexName,
	}
}

// CryptoComponentsFactory creates the crypto components
func CryptoComponentsFactory(args *cryptoComponentsFactoryArgs) (*Crypto, error) {
	initialPubKeys := args.nodesConfig.InitialNodesPubKeys()
	txSingleSigner := &singlesig.SchnorrSigner{}
	singleSigner, err := createSingleSigner(args.config)
	if err != nil {
		return nil, errors.New("could not create singleSigner: " + err.Error())
	}

	multisigHasher, err := getMultisigHasherFromConfig(args.config)
	if err != nil {
		return nil, errors.New("could not create multisig hasher: " + err.Error())
	}

	currentShardNodesPubKeys, err := args.nodesConfig.InitialNodesPubKeysForShard(args.shardCoordinator.SelfId())
	if err != nil {
		return nil, errors.New("could not start creation of multiSigner: " + err.Error())
	}

	multiSigner, err := createMultiSigner(args.config, multisigHasher, currentShardNodesPubKeys, args.privKey, args.keyGen)
	if err != nil {
		return nil, err
	}

	initialBalancesSkPemFileName := args.ctx.GlobalString(args.initialBalancesSkPemFileName)
	txSignKeyGen, txSignPrivKey, txSignPubKey, err := GetSigningParams(
		args.ctx,
		args.txSignSkName,
		args.txSignSkIndexName,
		initialBalancesSkPemFileName,
		kyber.NewBlakeSHA256Ed25519())
	if err != nil {
		return nil, err
	}
	args.log.Debug("starting with", "tx sign pubkey", GetPkEncoded(txSignPubKey))

	return &Crypto{
		TxSingleSigner:  txSingleSigner,
		SingleSigner:    singleSigner,
		MultiSigner:     multiSigner,
		BlockSignKeyGen: args.keyGen,
		TxSignKeyGen:    txSignKeyGen,
		TxSignPrivKey:   txSignPrivKey,
		TxSignPubKey:    txSignPubKey,
		InitialPubKeys:  initialPubKeys,
	}, nil
}

// NetworkComponentsFactory creates the network components
func NetworkComponentsFactory(p2pConfig *config.P2PConfig, log logger.Logger, core *Core) (*Network, error) {
	var randReader io.Reader
	if p2pConfig.Node.Seed != "" {
		randReader = NewSeedRandReader(core.Hasher.Compute(p2pConfig.Node.Seed))
	} else {
		randReader = rand.Reader
	}

	netMessenger, err := createNetMessenger(p2pConfig, log, randReader)
	if err != nil {
		return nil, err
	}

	return &Network{
		NetMessenger: netMessenger,
	}, nil
}

type processComponentsFactoryArgs struct {
	coreComponents       *coreComponentsFactoryArgs
	genesisConfig        *sharding.Genesis
	economicsData        *economics.EconomicsData
	nodesConfig          *sharding.NodesSetup
	gasSchedule          map[string]map[string]uint64
	syncer               ntp.SyncTimer
	shardCoordinator     sharding.Coordinator
	nodesCoordinator     sharding.NodesCoordinator
	data                 *Data
	core                 *Core
	crypto               *Crypto
	state                *State
	network              *Network
	coreServiceContainer serviceContainer.Core
	epochStartNotifier   EpochStartNotifier
	epochStart           *config.EpochStartConfig
	startEpochNum        uint32
}

// NewProcessComponentsFactoryArgs initializes the arguments necessary for creating the process components
func NewProcessComponentsFactoryArgs(
	coreComponents *coreComponentsFactoryArgs,
	genesisConfig *sharding.Genesis,
	economicsData *economics.EconomicsData,
	nodesConfig *sharding.NodesSetup,
	gasSchedule map[string]map[string]uint64,
	syncer ntp.SyncTimer,
	shardCoordinator sharding.Coordinator,
	nodesCoordinator sharding.NodesCoordinator,
	data *Data,
	core *Core,
	crypto *Crypto,
	state *State,
	network *Network,
	coreServiceContainer serviceContainer.Core,
	epochStartNotifier EpochStartNotifier,
	epochStart *config.EpochStartConfig,
	startEpochNum uint32,
) *processComponentsFactoryArgs {
	return &processComponentsFactoryArgs{
		coreComponents:       coreComponents,
		genesisConfig:        genesisConfig,
		economicsData:        economicsData,
		nodesConfig:          nodesConfig,
		gasSchedule:          gasSchedule,
		syncer:               syncer,
		shardCoordinator:     shardCoordinator,
		nodesCoordinator:     nodesCoordinator,
		data:                 data,
		core:                 core,
		crypto:               crypto,
		state:                state,
		network:              network,
		coreServiceContainer: coreServiceContainer,
		epochStartNotifier:   epochStartNotifier,
		epochStart:           epochStart,
		startEpochNum:        startEpochNum,
	}
}

// ProcessComponentsFactory creates the process components
func ProcessComponentsFactory(args *processComponentsFactoryArgs) (*Process, error) {
	rounder, err := round.NewRound(
		time.Unix(args.nodesConfig.StartTime, 0),
		args.syncer.CurrentTime(),
		time.Millisecond*time.Duration(args.nodesConfig.RoundDuration),
		args.syncer)
	if err != nil {
		return nil, err
	}

	interceptorContainerFactory, resolversContainerFactory, blackListHandler, err := newInterceptorAndResolverContainerFactory(
		args.shardCoordinator,
		args.nodesCoordinator,
		args.data, args.core,
		args.crypto,
		args.state,
		args.network,
		args.economicsData,
	)
	if err != nil {
		return nil, err
	}

	//TODO refactor all these factory calls
	interceptorsContainer, err := interceptorContainerFactory.Create()
	if err != nil {
		return nil, err
	}

	resolversContainer, err := resolversContainerFactory.Create()
	if err != nil {
		return nil, err
	}

	resolversFinder, err := containers.NewResolversFinder(resolversContainer, args.shardCoordinator)
	if err != nil {
		return nil, err
	}

	requestHandler, err := newRequestHandler(resolversFinder, args.shardCoordinator)
	if err != nil {
		return nil, err
	}

	epochStartTrigger, err := newEpochStartTrigger(args, requestHandler)
	if err != nil {
		return nil, err
	}

	forkDetector, err := newForkDetector(rounder, args.shardCoordinator, blackListHandler)
	if err != nil {
		return nil, err
	}

	genesisBlocks, err := generateGenesisHeadersAndApplyInitialBalances(
		args.core,
		args.state,
		args.data,
		args.shardCoordinator,
		args.nodesConfig,
		args.genesisConfig,
		args.economicsData,
	)
	if err != nil {
		return nil, err
	}

	err = prepareGenesisBlock(args, genesisBlocks)
	if err != nil {
		return nil, err
	}

	bootStr := args.data.Store.GetStorer(dataRetriever.BootstrapUnit)
	bootStorer, err := bootstrapStorage.NewBootstrapStorer(args.core.Marshalizer, bootStr)
	if err != nil {
		return nil, err
	}

	blockProcessor, err := newBlockProcessor(
		args,
		requestHandler,
		forkDetector,
		genesisBlocks,
		rounder,
		epochStartTrigger,
		bootStorer,
	)
	if err != nil {
		return nil, err
	}

	return &Process{
		InterceptorsContainer: interceptorsContainer,
		ResolversFinder:       resolversFinder,
		Rounder:               rounder,
		ForkDetector:          forkDetector,
		BlockProcessor:        blockProcessor,
		EpochStartTrigger:     epochStartTrigger,
		BlackListHandler:      blackListHandler,
		BootStorer:            bootStorer,
	}, nil
}

func prepareGenesisBlock(args *processComponentsFactoryArgs, genesisBlocks map[uint32]data.HeaderHandler) error {
	genesisBlock, ok := genesisBlocks[args.shardCoordinator.SelfId()]
	if !ok {
		return errors.New("genesis block does not exists")
	}

	genesisBlockHash, err := core.CalculateHash(args.core.Marshalizer, args.core.Hasher, genesisBlock)
	if err != nil {
		return err
	}

	err = args.data.Blkc.SetGenesisHeader(genesisBlock)
	if err != nil {
		return err
	}

	args.data.Blkc.SetGenesisHeaderHash(genesisBlockHash)

	marshalizedBlock, err := args.core.Marshalizer.Marshal(genesisBlock)
	if err != nil {
		return err
	}

	if args.shardCoordinator.SelfId() == sharding.MetachainShardId {
		errNotCritical := args.data.Store.Put(dataRetriever.MetaBlockUnit, genesisBlockHash, marshalizedBlock)
		if errNotCritical != nil {
			log.Error("error storing genesis metablock", "error", errNotCritical.Error())
		}
	} else {
		errNotCritical := args.data.Store.Put(dataRetriever.BlockHeaderUnit, genesisBlockHash, marshalizedBlock)
		if errNotCritical != nil {
			log.Error("error storing genesis shardblock", "error", errNotCritical.Error())
		}
	}

	return nil
}

func newRequestHandler(
	resolversFinder dataRetriever.ResolversFinder,
	shardCoordinator sharding.Coordinator,
) (process.RequestHandler, error) {
	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		requestHandler, err := requestHandlers.NewShardResolverRequestHandler(
			resolversFinder,
			factory.TransactionTopic,
			factory.UnsignedTransactionTopic,
			factory.RewardsTransactionTopic,
			factory.MiniBlocksTopic,
			factory.HeadersTopic,
			factory.MetachainBlocksTopic,
			MaxTxsToRequest,
		)
		if err != nil {
			return nil, err
		}

		return requestHandler, nil
	}

	if shardCoordinator.SelfId() == sharding.MetachainShardId {
		requestHandler, err := requestHandlers.NewMetaResolverRequestHandler(
			resolversFinder,
			factory.ShardHeadersForMetachainTopic,
			factory.MetachainBlocksTopic,
			factory.TransactionTopic,
			factory.UnsignedTransactionTopic,
			factory.MiniBlocksTopic,
			MaxTxsToRequest,
		)
		if err != nil {
			return nil, err
		}

		return requestHandler, nil
	}

	return nil, errors.New("could not create new request handler because of wrong shard id")
}

func newEpochStartTrigger(
	args *processComponentsFactoryArgs,
	requestHandler epochStart.RequestHandler,
) (epochStart.TriggerHandler, error) {
	if args.shardCoordinator.SelfId() < args.shardCoordinator.NumberOfShards() {
		argsHeaderValidator := block.ArgsHeaderValidator{
			Hasher:      args.core.Hasher,
			Marshalizer: args.core.Marshalizer,
		}
		headerValidator, err := block.NewHeaderValidator(argsHeaderValidator)
		if err != nil {
			return nil, err
		}

		argEpochStart := &shardchain.ArgsShardEpochStartTrigger{
			Marshalizer:        args.core.Marshalizer,
			Hasher:             args.core.Hasher,
			HeaderValidator:    headerValidator,
			Uint64Converter:    args.core.Uint64ByteSliceConverter,
			DataPool:           args.data.Datapool,
			Storage:            args.data.Store,
			RequestHandler:     requestHandler,
			Epoch:              args.startEpochNum,
			EpochStartNotifier: args.epochStartNotifier,
			Validity:           process.MetaBlockValidity,
			Finality:           process.MetaBlockFinality,
		}
		epochStartTrigger, err := shardchain.NewEpochStartTrigger(argEpochStart)
		if err != nil {
			return nil, errors.New("error creating new start of epoch trigger" + err.Error())
		}

		return epochStartTrigger, nil
	}

	if args.shardCoordinator.SelfId() == sharding.MetachainShardId {
		argEpochStart := &metachainEpochStart.ArgsNewMetaEpochStartTrigger{
			GenesisTime:        time.Unix(args.nodesConfig.StartTime, 0),
			Settings:           args.epochStart,
			Epoch:              args.startEpochNum,
			EpochStartNotifier: args.epochStartNotifier,
		}
		epochStartTrigger, err := metachainEpochStart.NewEpochStartTrigger(argEpochStart)
		if err != nil {
			return nil, errors.New("error creating new start of epoch trigger" + err.Error())
		}

		return epochStartTrigger, nil
	}

	return nil, errors.New("error creating new start of epoch trigger because of invalid shard id")
}

type seedRandReader struct {
	index int
	seed  []byte
}

// NewSeedRandReader will return a new instance of a seed-based reader
func NewSeedRandReader(seed []byte) *seedRandReader {
	return &seedRandReader{seed: seed, index: 0}
}

func (srr *seedRandReader) Read(p []byte) (n int, err error) {
	if srr.seed == nil {
		return 0, errors.New("nil seed")
	}
	if len(srr.seed) == 0 {
		return 0, errors.New("empty seed")
	}
	if p == nil {
		return 0, errors.New("nil buffer")
	}
	if len(p) == 0 {
		return 0, errors.New("empty buffer")
	}

	for i := 0; i < len(p); i++ {
		p[i] = srr.seed[srr.index]

		srr.index++
		srr.index = srr.index % len(srr.seed)
	}

	return len(p), nil
}

// CreateStatusHandlerPresenter will return an instance of PresenterStatusHandler
func CreateStatusHandlerPresenter() view.Presenter {
	presenterStatusHandlerFactory := factoryViews.NewPresenterFactory()

	return presenterStatusHandlerFactory.Create()
}

// CreateViews will start an termui console  and will return an object if cannot create and start termuiConsole
func CreateViews(presenter view.Presenter) ([]factoryViews.Viewer, error) {
	viewsFactory, err := factoryViews.NewViewsFactory(presenter)
	if err != nil {
		return nil, err
	}

	views, err := viewsFactory.Create()
	if err != nil {
		return nil, err
	}

	for _, v := range views {
		err = v.Start()
		if err != nil {
			return nil, err
		}
	}

	return views, nil
}

// CreateSoftwareVersionChecker will create a new software version checker and will start check if a new software version
// is available
func CreateSoftwareVersionChecker(statusHandler core.AppStatusHandler) (*softwareVersion.SoftwareVersionChecker, error) {
	softwareVersionCheckerFactory, err := factorySoftawareVersion.NewSoftwareVersionFactory(statusHandler)
	if err != nil {
		return nil, err
	}

	softwareVersionChecker, err := softwareVersionCheckerFactory.Create()
	if err != nil {
		return nil, err
	}

	return softwareVersionChecker, nil
}

func getHasherFromConfig(cfg *config.Config) (hashing.Hasher, error) {
	switch cfg.Hasher.Type {
	case "sha256":
		return sha256.Sha256{}, nil
	case "blake2b":
		return blake2b.Blake2b{}, nil
	}

	return nil, errors.New("no hasher provided in config file")
}

func getMarshalizerFromConfig(cfg *config.Config) (marshal.Marshalizer, error) {
	switch cfg.Marshalizer.Type {
	case "json":
		return &marshal.JsonMarshalizer{}, nil
	}

	return nil, errors.New("no marshalizer provided in config file")
}

func getTrie(
	cfg config.StorageConfig,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	uniqueID string,
) (data.Trie, error) {

	uniqueID = strings.Replace(uniqueID, "Epoch_0", "Static", 1)
	accountsTrieStorage, err := storageUnit.NewStorageUnitFromConf(
		getCacherFromConfig(cfg.Cache),
		getDBFromConfig(cfg.DB, uniqueID),
		getBloomFromConfig(cfg.Bloom),
	)
	if err != nil {
		return nil, errors.New("error creating accountsTrieStorage: " + err.Error())
	}

	return trie.NewTrie(accountsTrieStorage, marshalizer, hasher)
}

func createBlockChainFromConfig(config *config.Config, coordinator sharding.Coordinator, ash core.AppStatusHandler) (data.ChainHandler, error) {
	badBlockCache, err := storageUnit.NewCache(
		storageUnit.CacheType(config.BadBlocksCache.Type),
		config.BadBlocksCache.Size,
		config.BadBlocksCache.Shards)
	if err != nil {
		return nil, err
	}

	if coordinator == nil {
		return nil, state.ErrNilShardCoordinator
	}

	if coordinator.SelfId() < coordinator.NumberOfShards() {
		blockChain, err := blockchain.NewBlockChain(badBlockCache)
		if err != nil {
			return nil, err
		}

		err = blockChain.SetAppStatusHandler(ash)
		if err != nil {
			return nil, err
		}

		return blockChain, nil
	}
	if coordinator.SelfId() == sharding.MetachainShardId {
		blockChain, err := blockchain.NewMetaChain(badBlockCache)
		if err != nil {
			return nil, err
		}

		err = blockChain.SetAppStatusHandler(ash)
		if err != nil {
			return nil, err
		}

		return blockChain, nil
	}
	return nil, errors.New("can not create blockchain")
}

func createDataStoreFromConfig(
	config *config.Config,
	shardCoordinator sharding.Coordinator,
	uniqueID string,
	epochStartNotifier EpochStartNotifier,
) (dataRetriever.StorageService, error) {
	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		return createShardDataStoreFromConfig(config, shardCoordinator, uniqueID, epochStartNotifier)
	}
	if shardCoordinator.SelfId() == sharding.MetachainShardId {
		return createMetaChainDataStoreFromConfig(config, shardCoordinator, uniqueID, epochStartNotifier)
	}
	return nil, errors.New("can not create data store")
}

func createShardDataStoreFromConfig(
	config *config.Config,
	shardCoordinator sharding.Coordinator,
	uniqueID string,
	epochStartNotifier EpochStartNotifier,
) (dataRetriever.StorageService, error) {

	var headerUnit *pruning.PruningStorer
	var peerBlockUnit *pruning.PruningStorer
	var miniBlockUnit *pruning.PruningStorer
	var txUnit *pruning.PruningStorer
	var metachainHeaderUnit *pruning.PruningStorer
	var unsignedTxUnit *pruning.PruningStorer
	var rewardTxUnit *pruning.PruningStorer
	var metaHdrHashNonceUnit *pruning.PruningStorer
	var shardHdrHashNonceUnit *pruning.PruningStorer
	//	var heartbeatStorageUnit storageUnit.Unit
	var bootstrapUnit *storageUnit.Unit
	var err error

	defer func() {
		// cleanup
		if err != nil {
			if headerUnit != nil {
				_ = headerUnit.DestroyUnit()
			}
			if peerBlockUnit != nil {
				_ = peerBlockUnit.DestroyUnit()
			}
			if miniBlockUnit != nil {
				_ = miniBlockUnit.DestroyUnit()
			}
			if txUnit != nil {
				_ = txUnit.DestroyUnit()
			}
			if unsignedTxUnit != nil {
				_ = unsignedTxUnit.DestroyUnit()
			}
			if rewardTxUnit != nil {
				_ = rewardTxUnit.DestroyUnit()
			}
			if metachainHeaderUnit != nil {
				_ = metachainHeaderUnit.DestroyUnit()
			}
			if metaHdrHashNonceUnit != nil {
				_ = metaHdrHashNonceUnit.DestroyUnit()
			}
			if shardHdrHashNonceUnit != nil {
				_ = shardHdrHashNonceUnit.DestroyUnit()
			}
			if bootstrapUnit != nil {
				_ = bootstrapUnit.DestroyUnit()
			}
		}
	}()

	fullArchiveMode := config.StoragePruning.FullArchive
	numOfEpochsToKeep := uint32(config.StoragePruning.NumOfEpochsToKeep)
	numOfActivePersisters := uint32(config.StoragePruning.NumOfActivePersisters)

	txUnitStorerArgs := &pruning.PruningStorerArgs{
		Identifier:            "txUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.TxStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.TxStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.TxStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.TxStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	txUnit, err = pruning.NewPruningStorer(txUnitStorerArgs)
	if err != nil {
		return nil, err
	}

	unsignedTxUnitStorerArgs := &pruning.PruningStorerArgs{
		Identifier:            "unsignedTxUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.UnsignedTransactionStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.UnsignedTransactionStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.UnsignedTransactionStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.UnsignedTransactionStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	unsignedTxUnit, err = pruning.NewPruningStorer(unsignedTxUnitStorerArgs)
	if err != nil {
		return nil, err
	}

	rewardTxUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "rewardTxUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.RewardTxStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.RewardTxStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.RewardTxStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.RewardTxStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	rewardTxUnit, err = pruning.NewPruningStorer(rewardTxUnitArgs)
	if err != nil {
		return nil, err
	}

	miniBlockUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "miniBlockUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.MiniBlocksStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.MiniBlocksStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.MiniBlocksStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.MiniBlocksStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	miniBlockUnit, err = pruning.NewPruningStorer(miniBlockUnitArgs)
	if err != nil {
		return nil, err
	}

	peerBlockUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "peerBlockUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.PeerBlockBodyStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.PeerBlockBodyStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.PeerBlockBodyStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.PeerBlockBodyStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	peerBlockUnit, err = pruning.NewPruningStorer(peerBlockUnitArgs)
	if err != nil {
		return nil, err
	}

	headerUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "headerUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.BlockHeaderStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.BlockHeaderStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.BlockHeaderStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.BlockHeaderStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	headerUnit, err = pruning.NewPruningStorer(headerUnitArgs)
	if err != nil {
		return nil, err
	}

	metaChainHeaderUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "metachainHeaderUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.MetaBlockStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.MetaBlockStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.MetaBlockStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.MetaBlockStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	metachainHeaderUnit, err = pruning.NewPruningStorer(metaChainHeaderUnitArgs)
	if err != nil {
		return nil, err
	}

	metaHdrHashNonceUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "metaHdrHashNonceUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.MetaHdrNonceHashStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.MetaHdrNonceHashStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.MetaHdrNonceHashStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.MetaHdrNonceHashStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	metaHdrHashNonceUnit, err = pruning.NewPruningStorer(metaHdrHashNonceUnitArgs)
	if err != nil {
		return nil, err
	}

	shardHdrHashNonceUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "shardHrHashNonceUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.ShardHdrNonceHashStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.ShardHdrNonceHashStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.ShardHdrNonceHashStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.ShardHdrNonceHashStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	shardHdrHashNonceUnit, err = pruning.NewShardedPruningStorer(shardHdrHashNonceUnitArgs, shardCoordinator.SelfId())
	if err != nil {
		return nil, err
	}

	//heartbeatStorageUnitArgs := &pruning.PruningStorerArgs{
	//	Identifier:            "heartbeatStorageUnit",
	//	FullArchive:           fullArchiveMode,
	//	CacheConf:             getCacherFromConfig(config.Heartbeat.HeartbeatStorage.Cache),
	//	DbPath:                filepath.Join(uniqueID, config.Heartbeat.HeartbeatStorage.DB.FilePath),
	//	PersisterFactory:      storageFactory.NewPersisterFactory(config.Heartbeat.HeartbeatStorage.DB),
	//	BloomFilterConf:       getBloomFromConfig(config.Heartbeat.HeartbeatStorage.Bloom),
	//	NumOfEpochsToKeep:     numOfEpochsToKeep,
	//	NumOfActivePersisters: numOfActivePersisters,
	//	Notifier:              epochStartNotifier,
	//}
	heartbeatUniqueID := strings.Replace(uniqueID, "Epoch_0", "Static", 1)
	heartbeatStorageUnit, err := storageUnit.NewStorageUnitFromConf(
		getCacherFromConfig(config.Heartbeat.HeartbeatStorage.Cache),
		getDBFromConfig(config.Heartbeat.HeartbeatStorage.DB, heartbeatUniqueID),
		getBloomFromConfig(config.Heartbeat.HeartbeatStorage.Bloom))
	if err != nil {
		return nil, err
	}

	bootstrapUnit, err = storageUnit.NewStorageUnitFromConf(
		getCacherFromConfig(config.BootstrapStorage.Cache),
		getDBFromConfig(config.BootstrapStorage.DB, uniqueID),
		getBloomFromConfig(config.BootstrapStorage.Bloom))
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
	hdrNonceHashDataUnit := dataRetriever.ShardHdrNonceHashDataUnit + dataRetriever.UnitType(shardCoordinator.SelfId())
	store.AddStorer(hdrNonceHashDataUnit, shardHdrHashNonceUnit)
	store.AddStorer(dataRetriever.HeartbeatUnit, heartbeatStorageUnit)
	store.AddStorer(dataRetriever.BootstrapUnit, bootstrapUnit)

	return store, err
}

func createMetaChainDataStoreFromConfig(
	config *config.Config,
	shardCoordinator sharding.Coordinator,
	uniqueID string,
	epochStartNotifier pruning.EpochStartNotifier,
) (dataRetriever.StorageService, error) {
	var shardDataUnit *pruning.PruningStorer
	var metaBlockUnit *pruning.PruningStorer
	var headerUnit *pruning.PruningStorer
	var txUnit *pruning.PruningStorer
	var peerDataUnit *pruning.PruningStorer
	var metaHdrHashNonceUnit *pruning.PruningStorer
	var miniBlockUnit *pruning.PruningStorer
	var unsignedTxUnit *pruning.PruningStorer
	var miniBlockHeadersUnit *pruning.PruningStorer
	var shardHdrHashNonceUnits []*pruning.PruningStorer
	//var heartbeatStorageUnit *pruning.PruningStorer
	var err error

	defer func() {
		// cleanup
		if err != nil {
			log.Error("create meta store", "error", err.Error())
			if peerDataUnit != nil {
				_ = peerDataUnit.DestroyUnit()
			}
			if shardDataUnit != nil {
				_ = shardDataUnit.DestroyUnit()
			}
			if metaBlockUnit != nil {
				_ = metaBlockUnit.DestroyUnit()
			}
			if headerUnit != nil {
				_ = headerUnit.DestroyUnit()
			}
			if metaHdrHashNonceUnit != nil {
				_ = metaHdrHashNonceUnit.DestroyUnit()
			}
			if shardHdrHashNonceUnits != nil {
				for i := uint32(0); i < shardCoordinator.NumberOfShards(); i++ {
					_ = shardHdrHashNonceUnits[i].DestroyUnit()
				}
			}
			if txUnit != nil {
				_ = txUnit.DestroyUnit()
			}
			if unsignedTxUnit != nil {
				_ = unsignedTxUnit.DestroyUnit()
			}
			if miniBlockUnit != nil {
				_ = miniBlockUnit.DestroyUnit()
			}
			if miniBlockHeadersUnit != nil {
				_ = miniBlockHeadersUnit.DestroyUnit()
			}
			if bootstrapUnit != nil {
				_ = bootstrapUnit.DestroyUnit()
			}
		}
	}()

	fullArchiveMode := config.StoragePruning.FullArchive
	numOfEpochsToKeep := uint32(config.StoragePruning.NumOfEpochsToKeep)
	numOfActivePersisters := uint32(config.StoragePruning.NumOfActivePersisters)

	metaBlockUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "metaBlockUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.MetaBlockStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.MetaBlockStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.MetaBlockStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.MetaBlockStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	metaBlockUnit, err = pruning.NewPruningStorer(metaBlockUnitArgs)
	if err != nil {
		return nil, err
	}

	shardDataUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "shardDataUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.ShardDataStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.ShardDataStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.ShardDataStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.ShardDataStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	shardDataUnit, err = pruning.NewPruningStorer(shardDataUnitArgs)
	if err != nil {
		return nil, err
	}

	peerDataUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "peerDataUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.PeerDataStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.PeerDataStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.PeerDataStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.PeerDataStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	peerDataUnit, err = pruning.NewPruningStorer(peerDataUnitArgs)
	if err != nil {
		return nil, err
	}

	headerUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "headerUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.BlockHeaderStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.BlockHeaderStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.BlockHeaderStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.BlockHeaderStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	headerUnit, err = pruning.NewPruningStorer(headerUnitArgs)
	if err != nil {
		return nil, err
	}

	metaHdrHashNonceUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "metaHdrHashNonceUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.MetaHdrNonceHashStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.MetaHdrNonceHashStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.MetaHdrNonceHashStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.MetaHdrNonceHashStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	metaHdrHashNonceUnit, err = pruning.NewPruningStorer(metaHdrHashNonceUnitArgs)
	if err != nil {
		return nil, err
	}

	shardHdrHashNonceUnits = make([]*pruning.PruningStorer, shardCoordinator.NumberOfShards())
	for i := uint32(0); i < shardCoordinator.NumberOfShards(); i++ {
		shardHdrHashNonceUnitArgs := &pruning.PruningStorerArgs{
			Identifier:            "shardHdrHashNonceUnit",
			FullArchive:           fullArchiveMode,
			CacheConf:             getCacherFromConfig(config.ShardHdrNonceHashStorage.Cache),
			DbPath:                filepath.Join(uniqueID, config.ShardHdrNonceHashStorage.DB.FilePath),
			PersisterFactory:      storageFactory.NewPersisterFactory(config.ShardHdrNonceHashStorage.DB),
			BloomFilterConf:       getBloomFromConfig(config.ShardHdrNonceHashStorage.Bloom),
			NumOfEpochsToKeep:     numOfEpochsToKeep,
			NumOfActivePersisters: numOfActivePersisters,
			Notifier:              epochStartNotifier,
		}
		shardHdrHashNonceUnits[i], err = pruning.NewShardedPruningStorer(shardHdrHashNonceUnitArgs, i)
		if err != nil {
			return nil, err
		}
	}

	//heartbeatStorageUnitArgs := &pruning.PruningStorerArgs{
	//	Identifier:            "heartbeatStorageUnit",
	//	FullArchive:           fullArchiveMode,
	//	CacheConf:             getCacherFromConfig(config.Heartbeat.HeartbeatStorage.Cache),
	//	DbPath:                filepath.Join(uniqueID, config.Heartbeat.HeartbeatStorage.DB.FilePath),
	//	PersisterFactory:      storageFactory.NewPersisterFactory(config.Heartbeat.HeartbeatStorage.DB),
	//	BloomFilterConf:       getBloomFromConfig(config.Heartbeat.HeartbeatStorage.Bloom),
	//	NumOfEpochsToKeep:     numOfEpochsToKeep,
	//	NumOfActivePersisters: numOfActivePersisters,
	//	Notifier:              epochStartNotifier,
	//}
	//heartbeatStorageUnit, err = pruning.NewPruningStorer(heartbeatStorageUnitArgs)
	//if err != nil {
	//	return nil, err
	//}

	heartbeatUniqueID := strings.Replace(uniqueID, "Epoch_0", "Static", 1)
	heartbeatStorageUnit, err := storageUnit.NewStorageUnitFromConf(
		getCacherFromConfig(config.Heartbeat.HeartbeatStorage.Cache),
		getDBFromConfig(config.Heartbeat.HeartbeatStorage.DB, heartbeatUniqueID),
		getBloomFromConfig(config.Heartbeat.HeartbeatStorage.Bloom))
	if err != nil {
		return nil, err
	}

	txUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "txUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.TxStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.TxStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.TxStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.TxStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	txUnit, err = pruning.NewPruningStorer(txUnitArgs)
	if err != nil {
		return nil, err
	}

	unsignedTxUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "unsignedTxUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.UnsignedTransactionStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.UnsignedTransactionStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.UnsignedTransactionStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.UnsignedTransactionStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	unsignedTxUnit, err = pruning.NewPruningStorer(unsignedTxUnitArgs)
	if err != nil {
		return nil, err
	}

	miniBlockUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "miniBlockUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.MiniBlocksStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.MiniBlocksStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.MiniBlocksStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.MiniBlocksStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	miniBlockUnit, err = pruning.NewPruningStorer(miniBlockUnitArgs)
	if err != nil {
		return nil, err
	}

	miniBlockHeadersUnitArgs := &pruning.PruningStorerArgs{
		Identifier:            "miniBlockHeadersUnit",
		FullArchive:           fullArchiveMode,
		CacheConf:             getCacherFromConfig(config.MiniBlockHeadersStorage.Cache),
		DbPath:                filepath.Join(uniqueID, config.MiniBlockHeadersStorage.DB.FilePath),
		PersisterFactory:      storageFactory.NewPersisterFactory(config.MiniBlockHeadersStorage.DB),
		BloomFilterConf:       getBloomFromConfig(config.MiniBlockHeadersStorage.Bloom),
		NumOfEpochsToKeep:     numOfEpochsToKeep,
		NumOfActivePersisters: numOfActivePersisters,
		Notifier:              epochStartNotifier,
	}
	miniBlockHeadersUnit, err = pruning.NewPruningStorer(miniBlockHeadersUnitArgs)
	if err != nil {
		return nil, err
	}

	bootstrapUnit, err = storageUnit.NewStorageUnitFromConf(
		getCacherFromConfig(config.BootstrapStorage.Cache),
		getDBFromConfig(config.BootstrapStorage.DB, uniqueID),
		getBloomFromConfig(config.BootstrapStorage.Bloom))
	if err != nil {
		return nil, err
	}

	store := dataRetriever.NewChainStorer()
	store.AddStorer(dataRetriever.MetaBlockUnit, metaBlockUnit)
	store.AddStorer(dataRetriever.MetaShardDataUnit, shardDataUnit)
	store.AddStorer(dataRetriever.MetaPeerDataUnit, peerDataUnit)
	store.AddStorer(dataRetriever.BlockHeaderUnit, headerUnit)
	store.AddStorer(dataRetriever.MetaHdrNonceHashDataUnit, metaHdrHashNonceUnit)
	store.AddStorer(dataRetriever.TransactionUnit, txUnit)
	store.AddStorer(dataRetriever.UnsignedTransactionUnit, unsignedTxUnit)
	store.AddStorer(dataRetriever.MiniBlockUnit, miniBlockUnit)
	store.AddStorer(dataRetriever.MiniBlockHeaderUnit, miniBlockHeadersUnit)
	for i := uint32(0); i < shardCoordinator.NumberOfShards(); i++ {
		hdrNonceHashDataUnit := dataRetriever.ShardHdrNonceHashDataUnit + dataRetriever.UnitType(i)
		store.AddStorer(hdrNonceHashDataUnit, shardHdrHashNonceUnits[i])
	}
	store.AddStorer(dataRetriever.HeartbeatUnit, heartbeatStorageUnit)
	store.AddStorer(dataRetriever.BootstrapUnit, bootstrapUnit)

	return store, err
}

func createShardDataPoolFromConfig(
	config *config.Config,
	uint64ByteSliceConverter typeConverters.Uint64ByteSliceConverter,
) (dataRetriever.PoolsHolder, error) {

	log.Debug("creatingShardDataPool from config")

	txPool, err := shardedData.NewShardedData(getCacherFromConfig(config.TxDataPool))
	if err != nil {
		log.Error("error creating txpool")
		return nil, err
	}

	uTxPool, err := shardedData.NewShardedData(getCacherFromConfig(config.UnsignedTransactionDataPool))
	if err != nil {
		log.Error("error creating smart contract result pool")
		return nil, err
	}

	rewardTxPool, err := shardedData.NewShardedData(getCacherFromConfig(config.RewardTransactionDataPool))
	if err != nil {
		log.Error("error creating reward transaction pool")
		return nil, err
	}

	cacherCfg := getCacherFromConfig(config.BlockHeaderDataPool)
	hdrPool, err := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	if err != nil {
		log.Error("error creating hdrpool")
		return nil, err
	}

	cacherCfg = getCacherFromConfig(config.MetaBlockBodyDataPool)
	metaBlockBody, err := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	if err != nil {
		log.Error("error creating metaBlockBody")
		return nil, err
	}

	cacherCfg = getCacherFromConfig(config.BlockHeaderNoncesDataPool)
	hdrNoncesCacher, err := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	if err != nil {
		log.Error("error creating hdrNoncesCacher")
		return nil, err
	}
	hdrNonces, err := dataPool.NewNonceSyncMapCacher(hdrNoncesCacher, uint64ByteSliceConverter)
	if err != nil {
		log.Error("error creating hdrNonces")
		return nil, err
	}

	cacherCfg = getCacherFromConfig(config.TxBlockBodyDataPool)
	txBlockBody, err := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	if err != nil {
		log.Error("error creating txBlockBody")
		return nil, err
	}

	cacherCfg = getCacherFromConfig(config.PeerBlockBodyDataPool)
	peerChangeBlockBody, err := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	if err != nil {
		log.Error("error creating peerChangeBlockBody")
		return nil, err
	}

	currBlockTxs, err := dataPool.NewCurrentBlockPool()
	if err != nil {
		return nil, err
	}

	return dataPool.NewShardedDataPool(
		txPool,
		uTxPool,
		rewardTxPool,
		hdrPool,
		hdrNonces,
		txBlockBody,
		peerChangeBlockBody,
		metaBlockBody,
		currBlockTxs,
	)
}

func createMetaDataPoolFromConfig(
	config *config.Config,
	uint64ByteSliceConverter typeConverters.Uint64ByteSliceConverter,
) (dataRetriever.MetaPoolsHolder, error) {
	cacherCfg := getCacherFromConfig(config.MetaBlockBodyDataPool)
	metaBlockBody, err := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	if err != nil {
		log.Error("error creating metaBlockBody")
		return nil, err
	}

	cacherCfg = getCacherFromConfig(config.TxBlockBodyDataPool)
	txBlockBody, err := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	if err != nil {
		log.Error("error creating txBlockBody")
		return nil, err
	}

	cacherCfg = getCacherFromConfig(config.ShardHeadersDataPool)
	shardHeaders, err := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	if err != nil {
		log.Error("error creating shardHeaders")
		return nil, err
	}

	headersNoncesCacher, err := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	if err != nil {
		log.Error("error creating shard headers nonces pool")
		return nil, err
	}
	headersNonces, err := dataPool.NewNonceSyncMapCacher(headersNoncesCacher, uint64ByteSliceConverter)
	if err != nil {
		log.Error("error creating shard headers nonces pool")
		return nil, err
	}

	txPool, err := shardedData.NewShardedData(getCacherFromConfig(config.TxDataPool))
	if err != nil {
		log.Error("error creating txpool")
		return nil, err
	}

	uTxPool, err := shardedData.NewShardedData(getCacherFromConfig(config.UnsignedTransactionDataPool))
	if err != nil {
		log.Error("error creating smart contract result pool")
		return nil, err
	}

	currBlockTxs, err := dataPool.NewCurrentBlockPool()
	if err != nil {
		return nil, err
	}

	return dataPool.NewMetaDataPool(
		metaBlockBody,
		txBlockBody,
		shardHeaders,
		headersNonces,
		txPool,
		uTxPool,
		currBlockTxs,
	)
}

func createSingleSigner(config *config.Config) (crypto.SingleSigner, error) {
	switch config.Consensus.Type {
	case BlsConsensusType:
		return &singlesig.BlsSingleSigner{}, nil
	case BnConsensusType:
		return &singlesig.SchnorrSigner{}, nil
	}

	return nil, errors.New("no consensus type provided in config file")
}

func getMultisigHasherFromConfig(cfg *config.Config) (hashing.Hasher, error) {
	if cfg.Consensus.Type == BlsConsensusType && cfg.MultisigHasher.Type != "blake2b" {
		return nil, errors.New("wrong multisig hasher provided for bls consensus type")
	}

	switch cfg.MultisigHasher.Type {
	case "sha256":
		return sha256.Sha256{}, nil
	case "blake2b":
		if cfg.Consensus.Type == BlsConsensusType {
			return blake2b.Blake2b{HashSize: BlsHashSize}, nil
		}
		return blake2b.Blake2b{}, nil
	}

	return nil, errors.New("no multisig hasher provided in config file")
}

func createMultiSigner(
	config *config.Config,
	hasher hashing.Hasher,
	pubKeys []string,
	privateKey crypto.PrivateKey,
	keyGen crypto.KeyGenerator,
) (crypto.MultiSigner, error) {

	switch config.Consensus.Type {
	case BlsConsensusType:
		blsSigner := &blsMultiSig.KyberMultiSignerBLS{}
		return multisig.NewBLSMultisig(blsSigner, hasher, pubKeys, privateKey, keyGen, uint16(0))
	case BnConsensusType:
		return multisig.NewBelNevMultisig(hasher, pubKeys, privateKey, keyGen, uint16(0))
	}

	return nil, errors.New("no consensus type provided in config file")
}

func createNetMessenger(
	p2pConfig *config.P2PConfig,
	log logger.Logger,
	randReader io.Reader,
) (p2p.Messenger, error) {

	if p2pConfig.Node.Port < 0 {
		return nil, errors.New("cannot start node on port < 0")
	}

	pDiscoveryFactory := factoryP2P.NewPeerDiscovererCreator(*p2pConfig)
	pDiscoverer, err := pDiscoveryFactory.CreatePeerDiscoverer()

	if err != nil {
		return nil, err
	}

	log.Debug("peer discovery", "method", pDiscoverer.Name())

	prvKey, _ := ecdsa.GenerateKey(btcec.S256(), randReader)
	sk := (*libp2pCrypto.Secp256k1PrivateKey)(prvKey)

	nm, err := libp2p.NewNetworkMessenger(
		context.Background(),
		p2pConfig.Node.Port,
		sk,
		nil,
		loadBalancer.NewOutgoingChannelLoadBalancer(),
		pDiscoverer,
		libp2p.ListenAddrWithIp4AndTcp,
		p2pConfig.Node.TargetPeerCount,
	)
	if err != nil {
		return nil, err
	}

	return nm, nil
}

func newInterceptorAndResolverContainerFactory(
	shardCoordinator sharding.Coordinator,
	nodesCoordinator sharding.NodesCoordinator,
	data *Data,
	core *Core,
	crypto *Crypto,
	state *State,
	network *Network,
	economics *economics.EconomicsData,
) (process.InterceptorsContainerFactory, dataRetriever.ResolversContainerFactory, process.BlackListHandler, error) {

	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		return newShardInterceptorAndResolverContainerFactory(
			shardCoordinator,
			nodesCoordinator,
			data,
			core,
			crypto,
			state,
			network,
			economics,
		)
	}
	if shardCoordinator.SelfId() == sharding.MetachainShardId {
		return newMetaInterceptorAndResolverContainerFactory(
			shardCoordinator,
			nodesCoordinator,
			data,
			core,
			crypto,
			network,
			state,
			economics,
		)
	}

	return nil, nil, nil, errors.New("could not create interceptor and resolver container factory")
}

func newShardInterceptorAndResolverContainerFactory(
	shardCoordinator sharding.Coordinator,
	nodesCoordinator sharding.NodesCoordinator,
	data *Data,
	core *Core,
	crypto *Crypto,
	state *State,
	network *Network,
	economics *economics.EconomicsData,
) (process.InterceptorsContainerFactory, dataRetriever.ResolversContainerFactory, process.BlackListHandler, error) {

	headerBlackList := timecache.NewTimeCache(timeSpanForBadHeaders)
	interceptorContainerFactory, err := shard.NewInterceptorsContainerFactory(
		state.AccountsAdapter,
		shardCoordinator,
		nodesCoordinator,
		network.NetMessenger,
		data.Store,
		core.Marshalizer,
		core.Hasher,
		crypto.TxSignKeyGen,
		crypto.BlockSignKeyGen,
		crypto.TxSingleSigner,
		crypto.SingleSigner,
		crypto.MultiSigner,
		data.Datapool,
		state.AddressConverter,
		MaxTxNonceDeltaAllowed,
		economics,
		headerBlackList,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	dataPacker, err := partitioning.NewSimpleDataPacker(core.Marshalizer)
	if err != nil {
		return nil, nil, nil, err
	}

	resolversContainerFactory, err := shardfactoryDataRetriever.NewResolversContainerFactory(
		shardCoordinator,
		network.NetMessenger,
		data.Store,
		core.Marshalizer,
		data.Datapool,
		core.Uint64ByteSliceConverter,
		dataPacker,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	return interceptorContainerFactory, resolversContainerFactory, headerBlackList, nil
}

func newMetaInterceptorAndResolverContainerFactory(
	shardCoordinator sharding.Coordinator,
	nodesCoordinator sharding.NodesCoordinator,
	data *Data,
	core *Core,
	crypto *Crypto,
	network *Network,
	state *State,
	economics *economics.EconomicsData,
) (process.InterceptorsContainerFactory, dataRetriever.ResolversContainerFactory, process.BlackListHandler, error) {

	headerBlackList := timecache.NewTimeCache(timeSpanForBadHeaders)
	interceptorContainerFactory, err := metachain.NewInterceptorsContainerFactory(
		shardCoordinator,
		nodesCoordinator,
		network.NetMessenger,
		data.Store,
		core.Marshalizer,
		core.Hasher,
		crypto.MultiSigner,
		data.MetaDatapool,
		state.AccountsAdapter,
		state.AddressConverter,
		crypto.TxSingleSigner,
		crypto.SingleSigner,
		crypto.TxSignKeyGen,
		crypto.BlockSignKeyGen,
		MaxTxNonceDeltaAllowed,
		economics,
		headerBlackList,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	dataPacker, err := partitioning.NewSimpleDataPacker(core.Marshalizer)
	if err != nil {
		return nil, nil, nil, err
	}

	resolversContainerFactory, err := metafactoryDataRetriever.NewResolversContainerFactory(
		shardCoordinator,
		network.NetMessenger,
		data.Store,
		core.Marshalizer,
		data.MetaDatapool,
		core.Uint64ByteSliceConverter,
		dataPacker,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	return interceptorContainerFactory, resolversContainerFactory, headerBlackList, nil
}

func generateGenesisHeadersAndApplyInitialBalances(
	coreComponents *Core,
	stateComponents *State,
	dataComponents *Data,
	shardCoordinator sharding.Coordinator,
	nodesSetup *sharding.NodesSetup,
	genesisConfig *sharding.Genesis,
	economics *economics.EconomicsData,
) (map[uint32]data.HeaderHandler, error) {
	//TODO change this rudimentary startup for metachain nodes
	// Talk between Adrian, Robert and Iulian, did not want it to be discarded:
	// --------------------------------------------------------------------
	// Adrian: "This looks like a workaround as the metchain should not deal with individual accounts, but shards data.
	// What I was thinking was that the genesis on metachain (or pre-genesis block) is the nodes allocation to shards,
	// with 0 state root for every shard, as there is no balance yet.
	// Then the shards start operating as they get the initial node allocation, maybe we can do consensus on the
	// genesis as well, I think this would be actually good as then everything is signed and agreed upon.
	// The genesis shard blocks need to be then just the state root, I think we already have that in genesis,
	// so shard nodes can go ahead with individually creating the block, but then run consensus on this.
	// Then this block is sent to metachain who updates the state root of every shard and creates the metablock for
	// the genesis of each of the shards (this is actually the same thing that would happen at new epoch start)."

	genesisBlocks := make(map[uint32]data.HeaderHandler)

	for shardId := uint32(0); shardId < shardCoordinator.NumberOfShards(); shardId++ {
		isCurrentShard := shardId == shardCoordinator.SelfId()
		if isCurrentShard {
			continue
		}

		newShardCoordinator, account, err := createInMemoryShardCoordinatorAndAccount(
			coreComponents,
			shardCoordinator.NumberOfShards(),
			shardId,
		)
		if err != nil {
			return nil, err
		}

		genesisBlock, err := createGenesisBlockAndApplyInitialBalances(
			account,
			newShardCoordinator,
			stateComponents.AddressConverter,
			genesisConfig,
			uint64(nodesSetup.StartTime),
		)
		if err != nil {
			return nil, err
		}

		genesisBlocks[shardId] = genesisBlock
	}

	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		genesisBlockForCurrentShard, err := createGenesisBlockAndApplyInitialBalances(
			stateComponents.AccountsAdapter,
			shardCoordinator,
			stateComponents.AddressConverter,
			genesisConfig,
			uint64(nodesSetup.StartTime),
		)
		if err != nil {
			return nil, err
		}

		genesisBlocks[shardCoordinator.SelfId()] = genesisBlockForCurrentShard
	}

	argsMetaGenesis := genesis.ArgsMetaGenesisBlockCreator{
		GenesisTime:              uint64(nodesSetup.StartTime),
		Accounts:                 stateComponents.AccountsAdapter,
		AddrConv:                 stateComponents.AddressConverter,
		NodesSetup:               nodesSetup,
		ShardCoordinator:         shardCoordinator,
		Store:                    dataComponents.Store,
		Blkc:                     dataComponents.Blkc,
		Marshalizer:              coreComponents.Marshalizer,
		Hasher:                   coreComponents.Hasher,
		Uint64ByteSliceConverter: coreComponents.Uint64ByteSliceConverter,
		MetaDatapool:             dataComponents.MetaDatapool,
		Economics:                economics,
	}

	if shardCoordinator.SelfId() != sharding.MetachainShardId {
		newShardCoordinator, newAccounts, err := createInMemoryShardCoordinatorAndAccount(
			coreComponents,
			shardCoordinator.NumberOfShards(),
			sharding.MetachainShardId,
		)
		if err != nil {
			return nil, err
		}

		newStore, newBlkc, newMetaDataPool, err := createInMemoryStoreBlkcAndMetaDataPool(newShardCoordinator)

		argsMetaGenesis.ShardCoordinator = newShardCoordinator
		argsMetaGenesis.Accounts = newAccounts
		argsMetaGenesis.Store = newStore
		argsMetaGenesis.Blkc = newBlkc
		argsMetaGenesis.MetaDatapool = newMetaDataPool
	}

	genesisBlock, err := genesis.CreateMetaGenesisBlock(
		argsMetaGenesis,
	)
	if err != nil {
		return nil, err
	}

	log.Debug("MetaGenesisBlock created",
		"roothash", genesisBlock.GetRootHash(),
	)
	genesisBlocks[sharding.MetachainShardId] = genesisBlock

	return genesisBlocks, nil
}

func createInMemoryStoreBlkcAndMetaDataPool(
	shardCoordinator sharding.Coordinator,
) (dataRetriever.StorageService, data.ChainHandler, dataRetriever.MetaPoolsHolder, error) {

	cache, _ := storageUnit.NewCache(storageUnit.LRUCache, 10, 1)
	blkc, err := blockchain.NewMetaChain(cache)
	if err != nil {
		return nil, nil, nil, err
	}

	metaDataPool, err := createMemMetaDataPool()
	if err != nil {
		return nil, nil, nil, err
	}

	store := dataRetriever.NewChainStorer()
	store.AddStorer(dataRetriever.MetaBlockUnit, createMemUnit())
	store.AddStorer(dataRetriever.MetaShardDataUnit, createMemUnit())
	store.AddStorer(dataRetriever.MetaPeerDataUnit, createMemUnit())
	store.AddStorer(dataRetriever.BlockHeaderUnit, createMemUnit())
	store.AddStorer(dataRetriever.MetaHdrNonceHashDataUnit, createMemUnit())
	store.AddStorer(dataRetriever.TransactionUnit, createMemUnit())
	store.AddStorer(dataRetriever.UnsignedTransactionUnit, createMemUnit())
	store.AddStorer(dataRetriever.MiniBlockUnit, createMemUnit())
	for i := uint32(0); i < shardCoordinator.NumberOfShards(); i++ {
		hdrNonceHashDataUnit := dataRetriever.ShardHdrNonceHashDataUnit + dataRetriever.UnitType(i)
		store.AddStorer(hdrNonceHashDataUnit, createMemUnit())
	}
	store.AddStorer(dataRetriever.HeartbeatUnit, createMemUnit())

	return store, blkc, metaDataPool, nil
}

func createGenesisBlockAndApplyInitialBalances(
	accounts state.AccountsAdapter,
	shardCoordinator sharding.Coordinator,
	addressConverter state.AddressConverter,
	genesisConfig *sharding.Genesis,
	startTime uint64,
) (data.HeaderHandler, error) {

	initialBalances, err := genesisConfig.InitialNodesBalances(shardCoordinator, addressConverter)
	if err != nil {
		return nil, err
	}

	return genesis.CreateShardGenesisBlockFromInitialBalances(
		accounts,
		shardCoordinator,
		addressConverter,
		initialBalances,
		startTime,
	)
}

func createInMemoryShardCoordinatorAndAccount(
	coreComponents *Core,
	numOfShards uint32,
	shardId uint32,
) (sharding.Coordinator, state.AccountsAdapter, error) {

	newShardCoordinator, err := sharding.NewMultiShardCoordinator(numOfShards, shardId)
	if err != nil {
		return nil, nil, err
	}

	accountFactory, err := factoryState.NewAccountFactoryCreator(factoryState.UserAccount)
	if err != nil {
		return nil, nil, err
	}

	accounts, err := generateInMemoryAccountsAdapter(
		accountFactory,
		coreComponents.Hasher,
		coreComponents.Marshalizer,
	)
	if err != nil {
		return nil, nil, err
	}

	return newShardCoordinator, accounts, nil
}

func newForkDetector(
	rounder consensus.Rounder,
	shardCoordinator sharding.Coordinator,
	headerBlackList process.BlackListHandler,
) (process.ForkDetector, error) {
	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		return processSync.NewShardForkDetector(rounder, headerBlackList)
	}
	if shardCoordinator.SelfId() == sharding.MetachainShardId {
		return processSync.NewMetaForkDetector(rounder, headerBlackList)
	}

	return nil, ErrCreateForkDetector
}

func newBlockProcessor(
	processArgs *processComponentsFactoryArgs,
	requestHandler process.RequestHandler,
	forkDetector process.ForkDetector,
	genesisBlocks map[uint32]data.HeaderHandler,
	rounder consensus.Rounder,
	epochStartTrigger epochStart.TriggerHandler,
	bootStorer process.BootStorer,
) (process.BlockProcessor, error) {

	shardCoordinator := processArgs.shardCoordinator
	nodesCoordinator := processArgs.nodesCoordinator

	communityAddr := processArgs.economicsData.CommunityAddress()
	burnAddr := processArgs.economicsData.BurnAddress()
	if communityAddr == "" || burnAddr == "" {
		return nil, errors.New("rewards configuration missing")
	}

	communityAddress, err := hex.DecodeString(communityAddr)
	if err != nil {
		return nil, err
	}

	burnAddress, err := hex.DecodeString(burnAddr)
	if err != nil {
		return nil, err
	}

	specialAddressHolder, err := address.NewSpecialAddressHolder(
		communityAddress,
		burnAddress,
		processArgs.state.AddressConverter,
		shardCoordinator,
		nodesCoordinator,
	)
	if err != nil {
		return nil, err
	}

	validatorStatisticsProcessor, err := newValidatorStatisticsProcessor(processArgs)
	if err != nil {
		return nil, err
	}

	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		return newShardBlockProcessor(
			requestHandler,
			processArgs.shardCoordinator,
			processArgs.nodesCoordinator,
			specialAddressHolder,
			processArgs.data,
			processArgs.core,
			processArgs.state,
			forkDetector,
			genesisBlocks,
			processArgs.coreServiceContainer,
			processArgs.economicsData,
			rounder,
			epochStartTrigger,
			validatorStatisticsProcessor,
			bootStorer,
			processArgs.gasSchedule,
		)
	}
	if shardCoordinator.SelfId() == sharding.MetachainShardId {

		return newMetaBlockProcessor(
			requestHandler,
			processArgs.shardCoordinator,
			processArgs.nodesCoordinator,
			specialAddressHolder,
			processArgs.data,
			processArgs.core,
			processArgs.state,
			forkDetector,
			genesisBlocks,
			processArgs.coreServiceContainer,
			processArgs.economicsData,
			validatorStatisticsProcessor,
			rounder,
			epochStartTrigger,
			bootStorer,
		)
	}

	return nil, errors.New("could not create block processor and tracker")
}

func newShardBlockProcessor(
	requestHandler process.RequestHandler,
	shardCoordinator sharding.Coordinator,
	nodesCoordinator sharding.NodesCoordinator,
	specialAddressHandler process.SpecialAddressHandler,
	data *Data,
	core *Core,
	state *State,
	forkDetector process.ForkDetector,
	genesisBlocks map[uint32]data.HeaderHandler,
	coreServiceContainer serviceContainer.Core,
	economics *economics.EconomicsData,
	rounder consensus.Rounder,
	epochStartTrigger epochStart.TriggerHandler,
	statisticsProcessor process.ValidatorStatisticsProcessor,
	bootStorer process.BootStorer,
	gasSchedule map[string]map[string]uint64,
) (process.BlockProcessor, error) {
	argsParser, err := smartContract.NewAtArgumentParser()
	if err != nil {
		return nil, err
	}

	argsHook := hooks.ArgBlockChainHook{
		Accounts:         state.AccountsAdapter,
		AddrConv:         state.AddressConverter,
		StorageService:   data.Store,
		BlockChain:       data.Blkc,
		ShardCoordinator: shardCoordinator,
		Marshalizer:      core.Marshalizer,
		Uint64Converter:  core.Uint64ByteSliceConverter,
	}
	vmFactory, err := shard.NewVMContainerFactory(economics.MaxGasLimitPerBlock(), gasSchedule, argsHook)
	if err != nil {
		return nil, err
	}

	vmContainer, err := vmFactory.Create()
	if err != nil {
		return nil, err
	}

	interimProcFactory, err := shard.NewIntermediateProcessorsContainerFactory(
		shardCoordinator,
		core.Marshalizer,
		core.Hasher,
		state.AddressConverter,
		specialAddressHandler,
		data.Store,
		data.Datapool,
		economics,
	)
	if err != nil {
		return nil, err
	}

	interimProcContainer, err := interimProcFactory.Create()
	if err != nil {
		return nil, err
	}

	scForwarder, err := interimProcContainer.Get(dataBlock.SmartContractResultBlock)
	if err != nil {
		return nil, err
	}

	rewardsTxInterim, err := interimProcContainer.Get(dataBlock.RewardsBlock)
	if err != nil {
		return nil, err
	}

	rewardsTxHandler, ok := rewardsTxInterim.(process.TransactionFeeHandler)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	internalTransactionProducer, ok := rewardsTxInterim.(process.InternalTransactionProducer)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	gasHandler, err := preprocess.NewGasComputation(economics)
	if err != nil {
		return nil, err
	}

	scProcessor, err := smartContract.NewSmartContractProcessor(
		vmContainer,
		argsParser,
		core.Hasher,
		core.Marshalizer,
		state.AccountsAdapter,
		vmFactory.BlockChainHookImpl(),
		state.AddressConverter,
		shardCoordinator,
		scForwarder,
		rewardsTxHandler,
		economics,
		gasHandler,
	)
	if err != nil {
		return nil, err
	}

	rewardsTxProcessor, err := rewardTransaction.NewRewardTxProcessor(
		state.AccountsAdapter,
		state.AddressConverter,
		shardCoordinator,
		rewardsTxInterim,
	)
	if err != nil {
		return nil, err
	}

	txTypeHandler, err := coordinator.NewTxTypeHandler(state.AddressConverter, shardCoordinator, state.AccountsAdapter)
	if err != nil {
		return nil, err
	}

	transactionProcessor, err := transaction.NewTxProcessor(
		state.AccountsAdapter,
		core.Hasher,
		state.AddressConverter,
		core.Marshalizer,
		shardCoordinator,
		scProcessor,
		rewardsTxHandler,
		txTypeHandler,
		economics,
	)
	if err != nil {
		return nil, errors.New("could not create transaction statisticsProcessor: " + err.Error())
	}

	miniBlocksCompacter, err := preprocess.NewMiniBlocksCompaction(economics, shardCoordinator, gasHandler)
	if err != nil {
		return nil, err
	}

	preProcFactory, err := shard.NewPreProcessorsContainerFactory(
		shardCoordinator,
		data.Store,
		core.Marshalizer,
		core.Hasher,
		data.Datapool,
		state.AddressConverter,
		state.AccountsAdapter,
		requestHandler,
		transactionProcessor,
		scProcessor,
		scProcessor,
		rewardsTxProcessor,
		internalTransactionProducer,
		economics,
		miniBlocksCompacter,
		gasHandler,
	)
	if err != nil {
		return nil, err
	}

	preProcContainer, err := preProcFactory.Create()
	if err != nil {
		return nil, err
	}

	txCoordinator, err := coordinator.NewTransactionCoordinator(
		shardCoordinator,
		state.AccountsAdapter,
		data.Datapool.MiniBlocks(),
		requestHandler,
		preProcContainer,
		interimProcContainer,
		gasHandler,
	)
	if err != nil {
		return nil, err
	}

	txPoolsCleaner, err := poolsCleaner.NewTxsPoolsCleaner(
		state.AccountsAdapter,
		shardCoordinator,
		data.Datapool,
		state.AddressConverter,
	)
	if err != nil {
		return nil, err
	}

	argsHeaderValidator := block.ArgsHeaderValidator{
		Hasher:      core.Hasher,
		Marshalizer: core.Marshalizer,
	}
	headerValidator, err := block.NewHeaderValidator(argsHeaderValidator)
	if err != nil {
		return nil, err
	}

	argumentsBaseProcessor := block.ArgBaseProcessor{
		Accounts:              state.AccountsAdapter,
		ForkDetector:          forkDetector,
		Hasher:                core.Hasher,
		Marshalizer:           core.Marshalizer,
		Store:                 data.Store,
		ShardCoordinator:      shardCoordinator,
		NodesCoordinator:      nodesCoordinator,
		SpecialAddressHandler: specialAddressHandler,
		Uint64Converter:       core.Uint64ByteSliceConverter,
		StartHeaders:          genesisBlocks,
		RequestHandler:        requestHandler,
		Core:                  coreServiceContainer,
		BlockChainHook:        vmFactory.BlockChainHookImpl(),
		TxCoordinator:         txCoordinator,
		Rounder:               rounder,
		EpochStartTrigger:     epochStartTrigger,
		HeaderValidator:       headerValidator,
		ValidatorStatisticsProcessor: statisticsProcessor,
		BootStorer:                   bootStorer,
	}
	arguments := block.ArgShardProcessor{
		ArgBaseProcessor: argumentsBaseProcessor,
		DataPool:         data.Datapool,
		TxsPoolsCleaner:  txPoolsCleaner,
	}

	blockProcessor, err := block.NewShardProcessor(arguments)
	if err != nil {
		return nil, errors.New("could not create block statisticsProcessor: " + err.Error())
	}

	err = blockProcessor.SetAppStatusHandler(core.StatusHandler)
	if err != nil {
		return nil, err
	}

	return blockProcessor, nil
}

func newMetaBlockProcessor(
	requestHandler process.RequestHandler,
	shardCoordinator sharding.Coordinator,
	nodesCoordinator sharding.NodesCoordinator,
	specialAddressHandler process.SpecialAddressHandler,
	data *Data,
	core *Core,
	state *State,
	forkDetector process.ForkDetector,
	genesisBlocks map[uint32]data.HeaderHandler,
	coreServiceContainer serviceContainer.Core,
	economics *economics.EconomicsData,
	validatorStatisticsProcessor process.ValidatorStatisticsProcessor,
	rounder consensus.Rounder,
	epochStartTrigger epochStart.TriggerHandler,
	bootStorer process.BootStorer,
) (process.BlockProcessor, error) {

	argsHook := hooks.ArgBlockChainHook{
		Accounts:         state.AccountsAdapter,
		AddrConv:         state.AddressConverter,
		StorageService:   data.Store,
		BlockChain:       data.Blkc,
		ShardCoordinator: shardCoordinator,
		Marshalizer:      core.Marshalizer,
		Uint64Converter:  core.Uint64ByteSliceConverter,
	}
	vmFactory, err := metachain.NewVMContainerFactory(argsHook, economics)
	if err != nil {
		return nil, err
	}

	argsParser, err := smartContract.NewAtArgumentParser()
	if err != nil {
		return nil, err
	}

	vmContainer, err := vmFactory.Create()
	if err != nil {
		return nil, err
	}

	interimProcFactory, err := metachain.NewIntermediateProcessorsContainerFactory(
		shardCoordinator,
		core.Marshalizer,
		core.Hasher,
		state.AddressConverter,
		data.Store,
		data.MetaDatapool,
	)
	if err != nil {
		return nil, err
	}

	interimProcContainer, err := interimProcFactory.Create()
	if err != nil {
		return nil, err
	}

	scForwarder, err := interimProcContainer.Get(dataBlock.SmartContractResultBlock)
	if err != nil {
		return nil, err
	}

	gasHandler, err := preprocess.NewGasComputation(economics)
	if err != nil {
		return nil, err
	}

	scProcessor, err := smartContract.NewSmartContractProcessor(
		vmContainer,
		argsParser,
		core.Hasher,
		core.Marshalizer,
		state.AccountsAdapter,
		vmFactory.BlockChainHookImpl(),
		state.AddressConverter,
		shardCoordinator,
		scForwarder,
		&metachain.TransactionFeeHandler{},
		economics,
		gasHandler,
	)
	if err != nil {
		return nil, err
	}

	txTypeHandler, err := coordinator.NewTxTypeHandler(state.AddressConverter, shardCoordinator, state.AccountsAdapter)
	if err != nil {
		return nil, err
	}

	transactionProcessor, err := transaction.NewMetaTxProcessor(
		state.AccountsAdapter,
		state.AddressConverter,
		shardCoordinator,
		scProcessor,
		txTypeHandler,
	)
	if err != nil {
		return nil, errors.New("could not create transaction processor: " + err.Error())
	}

	miniBlocksCompacter, err := preprocess.NewMiniBlocksCompaction(economics, shardCoordinator, gasHandler)
	if err != nil {
		return nil, err
	}

	preProcFactory, err := metachain.NewPreProcessorsContainerFactory(
		shardCoordinator,
		data.Store,
		core.Marshalizer,
		core.Hasher,
		data.MetaDatapool,
		state.AccountsAdapter,
		requestHandler,
		transactionProcessor,
		economics,
		miniBlocksCompacter,
		gasHandler,
	)
	if err != nil {
		return nil, err
	}

	preProcContainer, err := preProcFactory.Create()
	if err != nil {
		return nil, err
	}

	txCoordinator, err := coordinator.NewTransactionCoordinator(
		shardCoordinator,
		state.AccountsAdapter,
		data.MetaDatapool.MiniBlocks(),
		requestHandler,
		preProcContainer,
		interimProcContainer,
		gasHandler,
	)
	if err != nil {
		return nil, err
	}

	scDataGetter, err := smartContract.NewSCQueryService(vmContainer, economics.MaxGasLimitPerBlock())
	if err != nil {
		return nil, err
	}

	argsStaking := scToProtocol.ArgStakingToPeer{
		AdrConv:     state.AddressConverter,
		Hasher:      core.Hasher,
		Marshalizer: core.Marshalizer,
		PeerState:   state.PeerAccounts,
		BaseState:   state.AccountsAdapter,
		ArgParser:   argsParser,
		CurrTxs:     data.MetaDatapool.CurrentBlockTxs(),
		ScQuery:     scDataGetter,
	}
	smartContractToProtocol, err := scToProtocol.NewStakingToPeer(argsStaking)
	if err != nil {
		return nil, err
	}

	miniBlockHeaderStore := data.Store.GetStorer(dataRetriever.MiniBlockHeaderUnit)
	if check.IfNil(miniBlockHeaderStore) {
		return nil, errors.New("could not create pending miniblocks handler because of empty miniblock header store")
	}

	metaBlocksStore := data.Store.GetStorer(dataRetriever.MetaBlockUnit)
	if check.IfNil(metaBlocksStore) {
		return nil, errors.New("could not create pending miniblocks handler because of empty metablock store")
	}

	argsPendingMiniBlocks := &metachainEpochStart.ArgsPendingMiniBlocks{
		Marshalizer:      core.Marshalizer,
		Storage:          miniBlockHeaderStore,
		MetaBlockPool:    data.MetaDatapool.MetaBlocks(),
		MetaBlockStorage: metaBlocksStore,
	}
	pendingMiniBlocks, err := metachainEpochStart.NewPendingMiniBlocks(argsPendingMiniBlocks)
	if err != nil {
		return nil, err
	}

	argsHeaderValidator := block.ArgsHeaderValidator{
		Hasher:      core.Hasher,
		Marshalizer: core.Marshalizer,
	}
	headerValidator, err := block.NewHeaderValidator(argsHeaderValidator)
	if err != nil {
		return nil, err
	}

	argumentsBaseProcessor := block.ArgBaseProcessor{
		Accounts:                     state.AccountsAdapter,
		ForkDetector:                 forkDetector,
		Hasher:                       core.Hasher,
		Marshalizer:                  core.Marshalizer,
		Store:                        data.Store,
		ShardCoordinator:             shardCoordinator,
		NodesCoordinator:             nodesCoordinator,
		SpecialAddressHandler:        specialAddressHandler,
		Uint64Converter:              core.Uint64ByteSliceConverter,
		StartHeaders:                 genesisBlocks,
		RequestHandler:               requestHandler,
		Core:                         coreServiceContainer,
		BlockChainHook:               vmFactory.BlockChainHookImpl(),
		TxCoordinator:                txCoordinator,
		ValidatorStatisticsProcessor: validatorStatisticsProcessor,
		EpochStartTrigger:            epochStartTrigger,
		Rounder:                      rounder,
		HeaderValidator:              headerValidator,
		BootStorer:                   bootStorer,
	}
	arguments := block.ArgMetaProcessor{
		ArgBaseProcessor:   argumentsBaseProcessor,
		DataPool:           data.MetaDatapool,
		SCDataGetter:       scDataGetter,
		SCToProtocol:       smartContractToProtocol,
		PeerChangesHandler: smartContractToProtocol,
		PendingMiniBlocks:  pendingMiniBlocks,
	}

	metaProcessor, err := block.NewMetaProcessor(arguments)
	if err != nil {
		return nil, errors.New("could not create block processor: " + err.Error())
	}

	err = metaProcessor.SetAppStatusHandler(core.StatusHandler)
	if err != nil {
		return nil, err
	}

	return metaProcessor, nil
}

func newValidatorStatisticsProcessor(
	processComponents *processComponentsFactoryArgs,
) (process.ValidatorStatisticsProcessor, error) {

	initialNodes := processComponents.nodesConfig.InitialNodes
	storageService := processComponents.data.Store

	var peerDataPool peer.DataPool = processComponents.data.MetaDatapool
	if processComponents.shardCoordinator.SelfId() < processComponents.shardCoordinator.NumberOfShards() {
		peerDataPool = processComponents.data.Datapool
	}

	arguments := peer.ArgValidatorStatisticsProcessor{
		InitialNodes:     initialNodes,
		PeerAdapter:      processComponents.state.PeerAccounts,
		AdrConv:          processComponents.state.AddressConverter,
		NodesCoordinator: processComponents.nodesCoordinator,
		ShardCoordinator: processComponents.shardCoordinator,
		DataPool:         peerDataPool,
		StorageService:   storageService,
		Marshalizer:      processComponents.core.Marshalizer,
		Economics:        processComponents.economicsData,
	}

	validatorStatisticsProcessor, err := peer.NewValidatorStatisticsProcessor(arguments)
	if err != nil {
		return nil, err
	}

	return validatorStatisticsProcessor, nil
}

func getCacherFromConfig(cfg config.CacheConfig) storageUnit.CacheConfig {
	return storageUnit.CacheConfig{
		Size:   cfg.Size,
		Type:   storageUnit.CacheType(cfg.Type),
		Shards: cfg.Shards,
	}
}

func getDBFromConfig(cfg config.DBConfig, uniquePath string) storageUnit.DBConfig {
	return storageUnit.DBConfig{
		FilePath:          filepath.Join(uniquePath, cfg.FilePath),
		Type:              storageUnit.DBType(cfg.Type),
		MaxBatchSize:      cfg.MaxBatchSize,
		BatchDelaySeconds: cfg.BatchDelaySeconds,
		MaxOpenFiles:      cfg.MaxOpenFiles,
	}
}

func getBloomFromConfig(cfg config.BloomFilterConfig) storageUnit.BloomConfig {
	var hashFuncs []storageUnit.HasherType
	if cfg.HashFunc != nil {
		hashFuncs = make([]storageUnit.HasherType, 0)
		for _, hf := range cfg.HashFunc {
			hashFuncs = append(hashFuncs, storageUnit.HasherType(hf))
		}
	}

	return storageUnit.BloomConfig{
		Size:     cfg.Size,
		HashFunc: hashFuncs,
	}
}

func generateInMemoryAccountsAdapter(
	accountFactory state.AccountFactory,
	hasher hashing.Hasher,
	marshalizer marshal.Marshalizer,
) (state.AccountsAdapter, error) {

	tr, err := trie.NewTrie(createMemUnit(), marshalizer, hasher)
	if err != nil {
		return nil, err
	}

	adb, err := state.NewAccountsDB(tr, hasher, marshalizer, accountFactory)
	if err != nil {
		return nil, err
	}

	return adb, nil
}

func createMemUnit() storage.Storer {
	cache, err := storageUnit.NewCache(storageUnit.LRUCache, 10, 1)
	if err != nil {
		log.Error("error creating cache for mem unit " + err.Error())
		return nil
	}

	persist, err := memorydb.New()
	if err != nil {
		log.Error("error creating persister for mem unit " + err.Error())
		return nil
	}

	unit, err := storageUnit.NewStorageUnit(cache, persist)
	if err != nil {
		log.Error("error creating unit " + err.Error())
		return nil
	}

	return unit
}

func createMemMetaDataPool() (dataRetriever.MetaPoolsHolder, error) {
	cacherCfg := storageUnit.CacheConfig{Size: 10, Type: storageUnit.LRUCache}
	metaBlocks, err := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	if err != nil {
		return nil, err
	}

	cacherCfg = storageUnit.CacheConfig{Size: 10, Type: storageUnit.LRUCache, Shards: 1}
	txBlockBody, err := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	if err != nil {
		return nil, err
	}

	cacherCfg = storageUnit.CacheConfig{Size: 10, Type: storageUnit.LRUCache}
	shardHeaders, err := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	if err != nil {
		return nil, err
	}

	shardHeadersNoncesCacher, err := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	if err != nil {
		return nil, err
	}

	shardHeadersNonces, err := dataPool.NewNonceSyncMapCacher(shardHeadersNoncesCacher, uint64ByteSlice.NewBigEndianConverter())
	if err != nil {
		return nil, err
	}

	txPool, err := shardedData.NewShardedData(storageUnit.CacheConfig{Size: 1000, Type: storageUnit.LRUCache, Shards: 1})
	if err != nil {
		return nil, err
	}

	uTxPool, err := shardedData.NewShardedData(storageUnit.CacheConfig{Size: 1000, Type: storageUnit.LRUCache, Shards: 1})
	if err != nil {
		return nil, err
	}

	currTxs, err := dataPool.NewCurrentBlockPool()
	if err != nil {
		return nil, err
	}

	dPool, err := dataPool.NewMetaDataPool(
		metaBlocks,
		txBlockBody,
		shardHeaders,
		shardHeadersNonces,
		txPool,
		uTxPool,
		currTxs,
	)
	if err != nil {
		return nil, err
	}

	return dPool, nil
}

// GetSigningParams returns a key generator, a private key, and a public key
func GetSigningParams(
	ctx *cli.Context,
	skName string,
	skIndexName string,
	skPemFileName string,
	suite crypto.Suite,
) (keyGen crypto.KeyGenerator, privKey crypto.PrivateKey, pubKey crypto.PublicKey, err error) {

	sk, err := getSk(ctx, skName, skIndexName, skPemFileName)
	if err != nil {
		return nil, nil, nil, err
	}

	keyGen = signing.NewKeyGenerator(suite)

	privKey, err = keyGen.PrivateKeyFromByteArray(sk)
	if err != nil {
		return nil, nil, nil, err
	}

	pubKey = privKey.GeneratePublic()

	return keyGen, privKey, pubKey, err
}

// GetPkEncoded returns the encoded public key
func GetPkEncoded(pubKey crypto.PublicKey) string {
	pk, err := pubKey.ToByteArray()
	if err != nil {
		return err.Error()
	}

	return encodeAddress(pk)
}

func encodeAddress(address []byte) string {
	return hex.EncodeToString(address)
}

func decodeAddress(address string) ([]byte, error) {
	return hex.DecodeString(address)
}

func getSk(
	ctx *cli.Context,
	skName string,
	skIndexName string,
	skPemFileName string,
) ([]byte, error) {

	//if flag is defined, it shall overwrite what was read from pem file
	if ctx.GlobalIsSet(skName) {
		encodedSk := []byte(ctx.GlobalString(skName))
		return decodeAddress(string(encodedSk))
	}

	skIndex := ctx.GlobalInt(skIndexName)
	encodedSk, err := core.LoadSkFromPemFile(skPemFileName, skIndex)
	if err != nil {
		return nil, err
	}

	return decodeAddress(string(encodedSk))
}
