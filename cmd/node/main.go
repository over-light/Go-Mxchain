package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/ElrondNetwork/elrond-go-sandbox/cmd/facade"
	"github.com/ElrondNetwork/elrond-go-sandbox/config"
	"github.com/ElrondNetwork/elrond-go-sandbox/consensus/round"
	"github.com/ElrondNetwork/elrond-go-sandbox/core"
	"github.com/ElrondNetwork/elrond-go-sandbox/core/logger"
	"github.com/ElrondNetwork/elrond-go-sandbox/core/statistics"
	"github.com/ElrondNetwork/elrond-go-sandbox/crypto"
	"github.com/ElrondNetwork/elrond-go-sandbox/crypto/signing"
	"github.com/ElrondNetwork/elrond-go-sandbox/crypto/signing/kyber"
	"github.com/ElrondNetwork/elrond-go-sandbox/crypto/signing/kyber/multisig"
	"github.com/ElrondNetwork/elrond-go-sandbox/crypto/signing/kyber/singlesig"
	"github.com/ElrondNetwork/elrond-go-sandbox/data"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/blockchain"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/state"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/trie"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/typeConverters/uint64ByteSlice"
	"github.com/ElrondNetwork/elrond-go-sandbox/dataRetriever"
	"github.com/ElrondNetwork/elrond-go-sandbox/dataRetriever/dataPool"
	"github.com/ElrondNetwork/elrond-go-sandbox/dataRetriever/factory/containers"
	factoryDataRetriever "github.com/ElrondNetwork/elrond-go-sandbox/dataRetriever/factory/shard"
	"github.com/ElrondNetwork/elrond-go-sandbox/dataRetriever/shardedData"
	"github.com/ElrondNetwork/elrond-go-sandbox/hashing"
	"github.com/ElrondNetwork/elrond-go-sandbox/hashing/blake2b"
	"github.com/ElrondNetwork/elrond-go-sandbox/hashing/sha256"
	"github.com/ElrondNetwork/elrond-go-sandbox/marshal"
	"github.com/ElrondNetwork/elrond-go-sandbox/node"
	"github.com/ElrondNetwork/elrond-go-sandbox/ntp"
	"github.com/ElrondNetwork/elrond-go-sandbox/p2p"
	"github.com/ElrondNetwork/elrond-go-sandbox/p2p/libp2p"
	factoryP2P "github.com/ElrondNetwork/elrond-go-sandbox/p2p/libp2p/factory"
	"github.com/ElrondNetwork/elrond-go-sandbox/p2p/loadBalancer"
	"github.com/ElrondNetwork/elrond-go-sandbox/process/block"
	"github.com/ElrondNetwork/elrond-go-sandbox/process/factory"
	"github.com/ElrondNetwork/elrond-go-sandbox/process/factory/shard"
	processSync "github.com/ElrondNetwork/elrond-go-sandbox/process/sync"
	"github.com/ElrondNetwork/elrond-go-sandbox/process/transaction"
	"github.com/ElrondNetwork/elrond-go-sandbox/sharding"
	"github.com/ElrondNetwork/elrond-go-sandbox/storage"
	beevikntp "github.com/beevik/ntp"
	"github.com/btcsuite/btcd/btcec"
	crypto2 "github.com/libp2p/go-libp2p-crypto"
	"github.com/pkg/profile"
	"github.com/urfave/cli"
)

const (
	defaultLogPath   = "logs"
	defaultStatsPath = "stats"
)

var (
	nodeHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}
USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}
   {{if len .Authors}}
AUTHOR:
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Commands}}
GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}
VERSION:
   {{.Version}}
   {{end}}
`

	// genesisFile defines a flag for the path of the bootstrapping file.
	genesisFile = cli.StringFlag{
		Name:  "genesis-file",
		Usage: "The node will extract bootstrapping info from the genesis.json",
		Value: "genesis.json",
	}
	// privateKey defines a flag for the path of the private key used when starting the node
	privateKey = cli.StringFlag{
		Name:  "private-key",
		Usage: "Private key that the node will load on startup and will sign transactions - temporary until we have a wallet that can do that",
		Value: "",
	}
	// withUI defines a flag for choosing the option of starting with/without UI. If false, the node will start automatically
	withUI = cli.BoolTFlag{
		Name:  "with-ui",
		Usage: "If true, the application will be accompanied by a UI. The node will have to be manually started from the UI",
	}
	// port defines a flag for setting the port on which the node will listen for connections
	port = cli.IntFlag{
		Name:  "port",
		Usage: "Port number on which the application will start",
		Value: 32000,
	}
	// profileMode defines a flag for profiling the binary
	profileMode = cli.StringFlag{
		Name:  "profile-mode",
		Usage: "Profiling mode. Available options: cpu, mem, mutex, block",
		Value: "",
	}
	// privateKeyIndex defines a flag that specify the 0-th based index of the private key to be used from privkeys.pem file.
	privateKeyIndex = cli.IntFlag{
		Name:  "private-key-index",
		Usage: "PrivateKeyIndex defines a flag that specify the 0-th based index of the private key to be used from privkeys.pem file.",
		Value: 0,
	}

	configurationFile    = "./config/config.toml"
	p2pConfigurationFile = "./config/p2p.toml"
	privKeysPemFile      = "./config/privkeys.pem"

	//TODO remove uniqueID
	uniqueID = ""

	rm *statistics.ResourceMonitor
)

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

type nullChronologyValidator struct {
}

func (*nullChronologyValidator) ValidateReceivedBlock(shardID uint32, epoch uint32, nonce uint64, round uint32) error {
	return nil
}

func main() {
	log := logger.DefaultLogger()
	log.SetLevel(logger.LogInfo)

	app := cli.NewApp()
	cli.AppHelpTemplate = nodeHelpTemplate
	app.Name = "Elrond Node CLI App"
	app.Version = "v0.0.1"
	app.Usage = "This is the entry point for starting a new Elrond node - the app will start after the genesis timestamp"
	app.Flags = []cli.Flag{genesisFile, port, privateKey, profileMode, privateKeyIndex}
	app.Authors = []cli.Author{
		{
			Name:  "The Elrond Team",
			Email: "contact@elrond.com",
		},
	}

	app.Action = func(c *cli.Context) error {
		return startNode(c, log)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func startNode(ctx *cli.Context, log *logger.Logger) error {
	profileMode := ctx.GlobalString(profileMode.Name)
	switch profileMode {
	case "cpu":
		p := profile.Start(profile.CPUProfile, profile.ProfilePath("."), profile.NoShutdownHook)
		defer p.Stop()
	case "mem":
		p := profile.Start(profile.MemProfile, profile.ProfilePath("."), profile.NoShutdownHook)
		defer p.Stop()
	case "mutex":
		p := profile.Start(profile.MutexProfile, profile.ProfilePath("."), profile.NoShutdownHook)
		defer p.Stop()
	case "block":
		p := profile.Start(profile.BlockProfile, profile.ProfilePath("."), profile.NoShutdownHook)
		defer p.Stop()
	}

	log.Info("Starting node...")

	stop := make(chan bool, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	generalConfig, err := loadMainConfig(configurationFile, log)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Initialized with config from: %s", configurationFile))

	p2pConfig, err := core.LoadP2PConfig(p2pConfigurationFile)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Initialized with p2p config from: %s", p2pConfigurationFile))
	if ctx.IsSet(port.Name) {
		p2pConfig.Node.Port = ctx.GlobalInt(port.Name)
	}
	uniqueID = strconv.Itoa(p2pConfig.Node.Port)

	err = os.RemoveAll(config.DefaultPath() + uniqueID)
	if err != nil {
		return err
	}

	genesisConfig, err := sharding.NewGenesisConfig(ctx.GlobalString(genesisFile.Name))
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Initialized with genesis config from: %s", ctx.GlobalString(genesisFile.Name)))

	syncer := ntp.NewSyncTime(time.Hour, beevikntp.Query)
	go syncer.StartSync()

	// TODO: The next 5 lines should be deleted when we are done testing from a precalculated (not hard coded)
	//  timestamp
	if genesisConfig.StartTime == 0 {
		time.Sleep(1000 * time.Millisecond)
		ntpTime := syncer.CurrentTime()
		genesisConfig.StartTime = (ntpTime.Unix()/60 + 1) * 60
	}

	startTime := time.Unix(genesisConfig.StartTime, 0)
	log.Info(fmt.Sprintf("Start time in seconds: %d", startTime.Unix()))

	currentNode, err := createNode(ctx, generalConfig, genesisConfig, p2pConfig, syncer, log)

	if err != nil {
		return err
	}

	ef := facade.NewElrondNodeFacade(currentNode)

	ef.SetLogger(log)
	ef.SetSyncer(syncer)

	wg := sync.WaitGroup{}
	go ef.StartBackgroundServices(&wg)
	wg.Wait()

	if !ctx.Bool(withUI.Name) {
		log.Info("Bootstrapping node....")
		err = ef.StartNode()
		if err != nil {
			log.Error("starting node failed", err.Error())
			return err
		}
	}

	go func() {
		<-sigs
		log.Info("terminating at user's signal...")
		stop <- true
	}()

	log.Info("Application is now running...")
	<-stop

	if rm != nil {
		err = rm.Close()
		log.LogIfError(err)
	}
	return nil
}

func loadMainConfig(filepath string, log *logger.Logger) (*config.Config, error) {
	cfg := &config.Config{}
	err := core.LoadTomlFile(cfg, filepath, log)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func createNode(
	ctx *cli.Context,
	config *config.Config,
	genesisConfig *sharding.Genesis,
	p2pConfig *config.P2PConfig,
	syncer ntp.SyncTimer,
	log *logger.Logger,
) (*node.Node, error) {

	hasher, err := getHasherFromConfig(config)
	if err != nil {
		return nil, errors.New("could not create hasher: " + err.Error())
	}

	marshalizer, err := getMarshalizerFromConfig(config)
	if err != nil {
		return nil, errors.New("could not create marshalizer: " + err.Error())
	}

	tr, err := getTrie(config.AccountsTrieStorage, hasher)
	if err != nil {
		return nil, errors.New("error creating node: " + err.Error())
	}

	addressConverter, err := state.NewPlainAddressConverter(config.Address.Length, config.Address.Prefix)
	if err != nil {
		return nil, errors.New("could not create address converter: " + err.Error())
	}

	accountsAdapter, err := state.NewAccountsDB(tr, hasher, marshalizer)
	if err != nil {
		return nil, errors.New("could not create accounts adapter: " + err.Error())
	}

	keyGen, privKey, pubKey, err := getSigningParams(ctx, log)
	if err != nil {
		return nil, err
	}

	initialPubKeys := genesisConfig.InitialNodesPubKeys()

	publickKey, err := pubKey.ToByteArray()
	if err != nil {
		return nil, err
	}

	hexPublicKey := hex.EncodeToString(publickKey)
	logFile, err := core.CreateFile(hexPublicKey, defaultLogPath, "log")
	if err != nil {
		return nil, err
	}

	err = log.ApplyOptions(logger.WithFile(logFile))
	if err != nil {
		return nil, err
	}

	statsFile, err := core.CreateFile(hexPublicKey, defaultStatsPath, "txt")
	if err != nil {
		return nil, err
	}
	err = startStatisticsMonitor(statsFile, config.ResourceStats, log)
	if err != nil {
		return nil, err
	}

	selfShardId, err := genesisConfig.GetShardIDFromPubKey(publickKey)
	if err != nil {
		return nil, err
	}

	shardCoordinator, err := sharding.NewMultiShardCoordinator(genesisConfig.NumberOfShards(), selfShardId)
	if err != nil {
		return nil, err
	}

	transactionProcessor, err := transaction.NewTxProcessor(accountsAdapter, hasher, addressConverter, marshalizer, shardCoordinator)
	if err != nil {
		return nil, errors.New("could not create transaction processor: " + err.Error())
	}

	blkc, err := createBlockChainFromConfig(config)
	if err != nil {
		return nil, errors.New("could not create block chain: " + err.Error())
	}

	store, err := createShardDataStoreFromConfig(config)
	if err != nil {
		return nil, errors.New("could not create local data store: " + err.Error())
	}

	uint64ByteSliceConverter := uint64ByteSlice.NewBigEndianConverter()
	datapool, err := createShardDataPoolFromConfig(config, uint64ByteSliceConverter)
	if err != nil {
		return nil, errors.New("could not create shard data pools: " + err.Error())
	}

	// TODO create metachain / blockchain
	// TODO save config, and move this creation into another place for node movement
	// TODO call createMetaChainFromConfig and createMetaDataPoolFromConfig and createMetaChainDataStoreFromConfig

	inBalanceForShard, err := genesisConfig.InitialNodesBalances(shardCoordinator, addressConverter)
	if err != nil {
		return nil, errors.New("initial balances could not be processed " + err.Error())
	}

	singlesigner := &singlesig.SchnorrSigner{}

	multisigHasher, err := getMultisigHasherFromConfig(config)
	if err != nil {
		return nil, errors.New("could not create multisig hasher: " + err.Error())
	}

	currentShardPubKeys, err := genesisConfig.InitialNodesPubKeysForShard(shardCoordinator.SelfId())
	if err != nil {
		return nil, errors.New("could not start creation of multisigner: " + err.Error())
	}

	multisigner, err := multisig.NewBelNevMultisig(multisigHasher, currentShardPubKeys, privKey, keyGen, uint16(0))
	if err != nil {
		return nil, err
	}

	var randReader io.Reader
	if p2pConfig.Node.Seed != "" {
		randReader = NewSeedRandReader(hasher.Compute(p2pConfig.Node.Seed))
	} else {
		randReader = rand.Reader
	}

	netMessenger, err := createNetMessenger(p2pConfig, log, randReader)
	if err != nil {
		return nil, err
	}

	//TODO add a real chronology validator and remove null chronology validator
	interceptorContainerFactory, err := shard.NewInterceptorsContainerFactory(
		shardCoordinator,
		netMessenger,
		store,
		marshalizer,
		hasher,
		keyGen,
		singlesigner,
		multisigner,
		datapool,
		addressConverter,
		&nullChronologyValidator{},
	)
	if err != nil {
		return nil, err
	}

	//TODO refactor all these factory calls
	interceptorsContainer, err := interceptorContainerFactory.Create()
	if err != nil {
		return nil, err
	}

	resolversContainerFactory, err := factoryDataRetriever.NewResolversContainerFactory(
		shardCoordinator,
		netMessenger,
		store,
		marshalizer,
		datapool,
		uint64ByteSliceConverter,
	)
	if err != nil {
		return nil, err
	}

	resolversContainer, err := resolversContainerFactory.Create()
	if err != nil {
		return nil, err
	}

	resolversFinder, err := containers.NewResolversFinder(resolversContainer, shardCoordinator)
	if err != nil {
		return nil, err
	}

	rounder, err := round.NewRound(
		time.Unix(genesisConfig.StartTime, 0),
		syncer.CurrentTime(),
		time.Millisecond*time.Duration(genesisConfig.RoundDuration),
		syncer)
	if err != nil {
		return nil, err
	}

	forkDetector, err := processSync.NewBasicForkDetector(rounder)
	if err != nil {
		return nil, err
	}

	blockProcessor, err := block.NewShardProcessor(
		datapool,
		store,
		hasher,
		marshalizer,
		transactionProcessor,
		accountsAdapter,
		shardCoordinator,
		forkDetector,
		createRequestTransactionHandler(resolversFinder, log),
		createRequestMiniBlocksHandler(resolversFinder, log),
	)

	if err != nil {
		return nil, errors.New("could not create block processor: " + err.Error())
	}

	nd, err := node.NewNode(
		node.WithMessenger(netMessenger),
		node.WithHasher(hasher),
		node.WithMarshalizer(marshalizer),
		node.WithInitialNodesPubKeys(initialPubKeys),
		node.WithInitialNodesBalances(inBalanceForShard),
		node.WithAddressConverter(addressConverter),
		node.WithAccountsAdapter(accountsAdapter),
		node.WithBlockChain(blkc),
		node.WithDataStore(store),
		node.WithRoundDuration(genesisConfig.RoundDuration),
		node.WithConsensusGroupSize(int(genesisConfig.ConsensusGroupSize)),
		node.WithSyncer(syncer),
		node.WithBlockProcessor(blockProcessor),
		node.WithGenesisTime(time.Unix(genesisConfig.StartTime, 0)),
		node.WithRounder(rounder),
		node.WithDataPool(datapool),
		node.WithShardCoordinator(shardCoordinator),
		node.WithUint64ByteSliceConverter(uint64ByteSliceConverter),
		node.WithSinglesig(singlesigner),
		node.WithMultisig(multisigner),
		node.WithKeyGenerator(keyGen),
		node.WithPublicKey(pubKey),
		node.WithPrivateKey(privKey),
		node.WithForkDetector(forkDetector),
		node.WithInterceptorsContainer(interceptorsContainer),
		node.WithResolversFinder(resolversFinder),
	)

	if err != nil {
		return nil, errors.New("error creating node: " + err.Error())
	}

	err = nd.CreateShardedStores()
	if err != nil {
		return nil, err
	}

	return nd, nil
}

func createRequestTransactionHandler(resolversFinder dataRetriever.ResolversFinder, log *logger.Logger) func(destShardID uint32, txHash []byte) {
	return func(destShardID uint32, txHash []byte) {
		log.Debug(fmt.Sprintf("Requesting tx from shard %d with hash %s from network\n", destShardID, toB64(txHash)))
		resolver, err := resolversFinder.CrossShardResolver(factory.TransactionTopic, destShardID)
		if err != nil {
			log.Error(fmt.Sprintf("missing resolver to transaction topic to shard %d", destShardID))
			return
		}

		err = resolver.RequestDataFromHash(txHash)
		if err != nil {
			log.Debug(err.Error())
		}
	}
}

func createRequestMiniBlocksHandler(resolversFinder dataRetriever.ResolversFinder, log *logger.Logger) func(destShardID uint32, txHash []byte) {
	return func(shardId uint32, mbHash []byte) {
		log.Debug(fmt.Sprintf("Requesting miniblock from shard %d with hash %s from network\n", shardId, toB64(mbHash)))
		resolver, err := resolversFinder.CrossShardResolver(factory.MiniBlocksTopic, shardId)
		if err != nil {
			log.Error(fmt.Sprintf("missing resolver to miniblock topic to shard %d", shardId))
			return
		}

		err = resolver.RequestDataFromHash(mbHash)
		if err != nil {
			log.Debug(err.Error())
		}
	}
}

func createNetMessenger(
	p2pConfig *config.P2PConfig,
	log *logger.Logger,
	randReader io.Reader,
) (p2p.Messenger, error) {

	if p2pConfig.Node.Port <= 0 {
		return nil, errors.New("cannot start node on port <= 0")
	}

	pDiscoveryFactory := factoryP2P.NewPeerDiscovererCreator(*p2pConfig)
	pDiscoverer, err := pDiscoveryFactory.CreatePeerDiscoverer()

	if err != nil {
		return nil, err
	}

	log.Info(fmt.Sprintf("Starting with peer discovery: %s", pDiscoverer.Name()))

	prvKey, _ := ecdsa.GenerateKey(btcec.S256(), randReader)
	sk := (*crypto2.Secp256k1PrivateKey)(prvKey)

	nm, err := libp2p.NewNetworkMessenger(
		context.Background(),
		p2pConfig.Node.Port,
		sk,
		nil,
		loadBalancer.NewOutgoingChannelLoadBalancer(),
		pDiscoverer,
	)

	if err != nil {
		return nil, err
	}
	return nm, nil
}

func getSk(ctx *cli.Context, log *logger.Logger) ([]byte, error) {
	//if flag is defined, it shall overwrite what was read from pem file
	if ctx.GlobalIsSet(privateKey.Name) {
		encodedSk := []byte(ctx.GlobalString(privateKey.Name))
		return decodeAddress(string(encodedSk))
	}

	privateKeyIndex := ctx.GlobalInt(privateKeyIndex.Name)
	encodedSk, err := core.LoadSkFromPemFile(privKeysPemFile, log, privateKeyIndex)
	if err != nil {
		return nil, err
	}

	return decodeAddress(string(encodedSk))
}

func getSigningParams(ctx *cli.Context, log *logger.Logger) (
	keyGen crypto.KeyGenerator,
	privKey crypto.PrivateKey,
	pubKey crypto.PublicKey,
	err error,
) {
	sk, err := getSk(ctx, log)

	if err != nil {
		return nil, nil, nil, err
	}

	suite := kyber.NewBlakeSHA256Ed25519()
	keyGen = signing.NewKeyGenerator(suite)
	privKey, err = keyGen.PrivateKeyFromByteArray(sk)

	if err != nil {
		return nil, nil, nil, err
	}

	pubKey = privKey.GeneratePublic()

	pk, _ := pubKey.ToByteArray()

	pkEncoded := encodeAddress(pk)
	log.Info("starting with public key: " + pkEncoded)

	return keyGen, privKey, pubKey, err
}

func getTrie(cfg config.StorageConfig, hasher hashing.Hasher) (*trie.Trie, error) {
	accountsTrieStorage, err := storage.NewStorageUnitFromConf(
		getCacherFromConfig(cfg.Cache),
		getDBFromConfig(cfg.DB),
		getBloomFromConfig(cfg.Bloom),
	)
	if err != nil {
		return nil, errors.New("error creating node: " + err.Error())
	}

	dbWriteCache, err := trie.NewDBWriteCache(accountsTrieStorage)
	if err != nil {
		return nil, errors.New("error creating node: " + err.Error())
	}

	return trie.NewTrie(make([]byte, 32), dbWriteCache, hasher)
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

func getMultisigHasherFromConfig(cfg *config.Config) (hashing.Hasher, error) {
	switch cfg.MultisigHasher.Type {
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
		return marshal.JsonMarshalizer{}, nil
	}

	return nil, errors.New("no marshalizer provided in config file")
}

func getCacherFromConfig(cfg config.CacheConfig) storage.CacheConfig {
	return storage.CacheConfig{
		Size: cfg.Size,
		Type: storage.CacheType(cfg.Type),
	}
}

func getDBFromConfig(cfg config.DBConfig) storage.DBConfig {
	return storage.DBConfig{
		FilePath: filepath.Join(config.DefaultPath()+uniqueID, cfg.FilePath),
		Type:     storage.DBType(cfg.Type),
	}
}

func getBloomFromConfig(cfg config.BloomFilterConfig) storage.BloomConfig {
	var hashFuncs []storage.HasherType
	if cfg.HashFunc != nil {
		hashFuncs = make([]storage.HasherType, 0)
		for _, hf := range cfg.HashFunc {
			hashFuncs = append(hashFuncs, storage.HasherType(hf))
		}
	}

	return storage.BloomConfig{
		Size:     cfg.Size,
		HashFunc: hashFuncs,
	}
}

func createShardDataPoolFromConfig(
	config *config.Config,
	uint64ByteSliceConverter typeConverters.Uint64ByteSliceConverter,
) (dataRetriever.PoolsHolder, error) {

	txPool, err := shardedData.NewShardedData(getCacherFromConfig(config.TxDataPool))
	if err != nil {
		return nil, err
	}

	cacherCfg := getCacherFromConfig(config.BlockHeaderDataPool)
	hdrPool, err := storage.NewCache(cacherCfg.Type, cacherCfg.Size)
	if err != nil {
		return nil, err
	}

	cacherCfg = getCacherFromConfig(config.MetaBlockBodyDataPool)
	metaBlockBody, err := storage.NewCache(cacherCfg.Type, cacherCfg.Size)
	if err != nil {
		return nil, err
	}

	cacherCfg = getCacherFromConfig(config.BlockHeaderNoncesDataPool)
	hdrNoncesCacher, err := storage.NewCache(cacherCfg.Type, cacherCfg.Size)
	if err != nil {
		return nil, err
	}
	hdrNonces, err := dataPool.NewNonceToHashCacher(hdrNoncesCacher, uint64ByteSliceConverter)
	if err != nil {
		return nil, err
	}

	cacherCfg = getCacherFromConfig(config.TxBlockBodyDataPool)
	txBlockBody, err := storage.NewCache(cacherCfg.Type, cacherCfg.Size)
	if err != nil {
		return nil, err
	}

	cacherCfg = getCacherFromConfig(config.PeerBlockBodyDataPool)
	peerChangeBlockBody, err := storage.NewCache(cacherCfg.Type, cacherCfg.Size)
	if err != nil {
		return nil, err
	}

	cacherCfg = getCacherFromConfig(config.MetaHeaderNoncesDataPool)
	metaBlockNoncesCacher, err := storage.NewCache(cacherCfg.Type, cacherCfg.Size)
	if err != nil {
		return nil, err
	}
	metaBlockNonces, err := dataPool.NewNonceToHashCacher(metaBlockNoncesCacher, uint64ByteSliceConverter)
	if err != nil {
		return nil, err
	}

	return dataPool.NewShardedDataPool(
		txPool,
		hdrPool,
		hdrNonces,
		txBlockBody,
		peerChangeBlockBody,
		metaBlockBody,
		metaBlockNonces,
	)
}

func createBlockChainFromConfig(config *config.Config) (data.ChainHandler, error) {
	badBlockCache, err := storage.NewCache(
		storage.CacheType(config.BadBlocksCache.Type),
		config.BadBlocksCache.Size)
	if err != nil {
		return nil, err
	}

	blockChain, err := blockchain.NewBlockChain(
		badBlockCache,
	)
	if err != nil {
		return nil, err
	}

	return blockChain, err
}

func createShardDataStoreFromConfig(config *config.Config) (dataRetriever.StorageService, error) {
	var headerUnit, peerBlockUnit, miniBlockUnit, txUnit, metachainHeaderUnit *storage.Unit
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
			if metachainHeaderUnit != nil {
				_ = metachainHeaderUnit.DestroyUnit()
			}
		}
	}()

	txUnit, err = storage.NewStorageUnitFromConf(
		getCacherFromConfig(config.TxStorage.Cache),
		getDBFromConfig(config.TxStorage.DB),
		getBloomFromConfig(config.TxStorage.Bloom))
	if err != nil {
		return nil, err
	}

	miniBlockUnit, err = storage.NewStorageUnitFromConf(
		getCacherFromConfig(config.MiniBlocksStorage.Cache),
		getDBFromConfig(config.MiniBlocksStorage.DB),
		getBloomFromConfig(config.MiniBlocksStorage.Bloom))
	if err != nil {
		return nil, err
	}

	peerBlockUnit, err = storage.NewStorageUnitFromConf(
		getCacherFromConfig(config.PeerBlockBodyStorage.Cache),
		getDBFromConfig(config.PeerBlockBodyStorage.DB),
		getBloomFromConfig(config.PeerBlockBodyStorage.Bloom))
	if err != nil {
		return nil, err
	}

	headerUnit, err = storage.NewStorageUnitFromConf(
		getCacherFromConfig(config.BlockHeaderStorage.Cache),
		getDBFromConfig(config.BlockHeaderStorage.DB),
		getBloomFromConfig(config.BlockHeaderStorage.Bloom))
	if err != nil {
		return nil, err
	}

	metachainHeaderUnit, err = storage.NewStorageUnitFromConf(
		getCacherFromConfig(config.MetaBlockStorage.Cache),
		getDBFromConfig(config.MetaBlockStorage.DB),
		getBloomFromConfig(config.MetaBlockStorage.Bloom))
	if err != nil {
		return nil, err
	}

	store := dataRetriever.NewChainStorer()
	store.AddStorer(dataRetriever.TransactionUnit, txUnit)
	store.AddStorer(dataRetriever.MiniBlockUnit, miniBlockUnit)
	store.AddStorer(dataRetriever.PeerChangesUnit, peerBlockUnit)
	store.AddStorer(dataRetriever.BlockHeaderUnit, headerUnit)
	store.AddStorer(dataRetriever.MetaBlockUnit, metachainHeaderUnit)

	return store, err
}

func createMetaDataPoolFromConfig(
	config *config.Config,
	uint64ByteSliceConverter typeConverters.Uint64ByteSliceConverter,
) (dataRetriever.MetaPoolsHolder, error) {
	cacherCfg := getCacherFromConfig(config.MetaBlockBodyDataPool)
	metaBlockBody, err := storage.NewCache(cacherCfg.Type, cacherCfg.Size)
	if err != nil {
		return nil, err
	}

	miniBlockHashes, err := shardedData.NewShardedData(getCacherFromConfig(config.MiniBlockHeaderHashesDataPool))
	if err != nil {
		return nil, err
	}

	cacherCfg = getCacherFromConfig(config.ShardHeadersDataPool)
	shardHeaders, err := storage.NewCache(cacherCfg.Type, cacherCfg.Size)
	if err != nil {
		return nil, err
	}

	cacherCfg = getCacherFromConfig(config.MetaHeaderNoncesDataPool)
	metaBlockNoncesCacher, err := storage.NewCache(cacherCfg.Type, cacherCfg.Size)
	if err != nil {
		return nil, err
	}
	metaBlockNonces, err := dataPool.NewNonceToHashCacher(metaBlockNoncesCacher, uint64ByteSliceConverter)
	if err != nil {
		return nil, err
	}

	return dataPool.NewMetaDataPool(metaBlockBody, miniBlockHashes, shardHeaders, metaBlockNonces)
}

func createMetaChainFromConfig(config *config.Config) (*blockchain.MetaChain, error) {
	badBlockCache, err := storage.NewCache(
		storage.CacheType(config.BadBlocksCache.Type),
		config.BadBlocksCache.Size)
	if err != nil {
		return nil, err
	}

	metaChain, err := blockchain.NewMetaChain(
		badBlockCache,
	)
	if err != nil {
		return nil, err
	}

	return metaChain, err
}

func createMetaChainDataStoreFromConfig(config *config.Config) (dataRetriever.StorageService, error) {
	var peerDataUnit, shardDataUnit, metaBlockUnit *storage.Unit
	var err error

	defer func() {
		// cleanup
		if err != nil {
			if peerDataUnit != nil {
				_ = peerDataUnit.DestroyUnit()
			}
			if shardDataUnit != nil {
				_ = shardDataUnit.DestroyUnit()
			}
			if metaBlockUnit != nil {
				_ = metaBlockUnit.DestroyUnit()
			}
		}
	}()

	metaBlockUnit, err = storage.NewStorageUnitFromConf(
		getCacherFromConfig(config.MetaBlockStorage.Cache),
		getDBFromConfig(config.MetaBlockStorage.DB),
		getBloomFromConfig(config.MetaBlockStorage.Bloom))
	if err != nil {
		return nil, err
	}

	shardDataUnit, err = storage.NewStorageUnitFromConf(
		getCacherFromConfig(config.ShardDataStorage.Cache),
		getDBFromConfig(config.ShardDataStorage.DB),
		getBloomFromConfig(config.ShardDataStorage.Bloom))
	if err != nil {
		return nil, err
	}

	peerDataUnit, err = storage.NewStorageUnitFromConf(
		getCacherFromConfig(config.PeerDataStorage.Cache),
		getDBFromConfig(config.PeerDataStorage.DB),
		getBloomFromConfig(config.PeerDataStorage.Bloom))
	if err != nil {
		return nil, err
	}

	store := dataRetriever.NewChainStorer()
	store.AddStorer(dataRetriever.MetaBlockUnit, metaBlockUnit)
	store.AddStorer(dataRetriever.MetaShardDataUnit, shardDataUnit)
	store.AddStorer(dataRetriever.MetaPeerDataUnit, peerDataUnit)

	return store, err
}

func decodeAddress(address string) ([]byte, error) {
	return hex.DecodeString(address)
}

func encodeAddress(address []byte) string {
	return hex.EncodeToString(address)
}

func toB64(buff []byte) string {
	if buff == nil {
		return "<NIL>"
	}
	return base64.StdEncoding.EncodeToString(buff)
}

func startStatisticsMonitor(file *os.File, config config.ResourceStatsConfig, log *logger.Logger) error {
	if !config.Enabled {
		return nil
	}

	if config.RefreshIntervalInSec < 1 {
		return errors.New("invalid RefreshIntervalInSec in section [ResourceStats]. Should be an integer higher than 1")
	}

	rm, err := statistics.NewResourceMonitor(file)
	if err != nil {
		return err
	}

	go func() {
		for {
			err = rm.SaveStatistics()
			log.LogIfError(err)
			time.Sleep(time.Second * time.Duration(config.RefreshIntervalInSec))
		}
	}()

	return nil
}
