package node

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sync/atomic"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/chronology"
	"github.com/ElrondNetwork/elrond-go/consensus/spos"
	"github.com/ElrondNetwork/elrond-go/consensus/spos/sposFactory"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/indexer"
	"github.com/ElrondNetwork/elrond-go/core/partitioning"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/getters"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/node/heartbeat"
	"github.com/ElrondNetwork/elrond-go/node/heartbeat/storage"
	"github.com/ElrondNetwork/elrond-go/ntp"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/dataValidators"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/sync"
	"github.com/ElrondNetwork/elrond-go/process/sync/storageBootstrap"
	procTx "github.com/ElrondNetwork/elrond-go/process/transaction"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/statusHandler"
)

// SendTransactionsPipe is the pipe used for sending new transactions
const SendTransactionsPipe = "send transactions pipe"

var log = logger.GetOrCreate("node")

// Option represents a functional configuration parameter that can operate
//  over the None struct.
type Option func(*Node) error

// Node is a structure that passes the configuration parameters and initializes
//  required services as requested
type Node struct {
	internalMarshalizer           marshal.Marshalizer
	vmMarshalizer                 marshal.Marshalizer
	txSignMarshalizer             marshal.Marshalizer
	ctx                           context.Context
	hasher                        hashing.Hasher
	feeHandler                    process.FeeHandler
	initialNodesPubkeys           map[uint32][]string
	roundDuration                 uint64
	consensusGroupSize            int
	messenger                     P2PMessenger
	syncTimer                     ntp.SyncTimer
	rounder                       consensus.Rounder
	blockProcessor                process.BlockProcessor
	genesisTime                   time.Time
	epochStartTrigger             epochStart.TriggerHandler
	epochStartRegistrationHandler epochStart.RegistrationHandler
	accounts                      state.AccountsAdapter
	addrConverter                 state.AddressConverter
	uint64ByteSliceConverter      typeConverters.Uint64ByteSliceConverter
	interceptorsContainer         process.InterceptorsContainer
	resolversFinder               dataRetriever.ResolversFinder
	peerBlackListHandler          process.BlackListHandler
	heartbeatMonitor              *heartbeat.Monitor
	heartbeatSender               *heartbeat.Sender
	appStatusHandler              core.AppStatusHandler
	validatorStatistics           process.ValidatorStatisticsProcessor
	hardforkTrigger               HardforkTrigger
	validatorsProvider            process.ValidatorsProvider
	whiteListHandler              process.WhiteListHandler

	pubKey            crypto.PublicKey
	privKey           crypto.PrivateKey
	keyGen            crypto.KeyGenerator
	keyGenForAccounts crypto.KeyGenerator
	singleSigner      crypto.SingleSigner
	txSingleSigner    crypto.SingleSigner
	multiSigner       crypto.MultiSigner
	forkDetector      process.ForkDetector

	blkc             data.ChainHandler
	dataPool         dataRetriever.PoolsHolder
	store            dataRetriever.StorageService
	shardCoordinator sharding.Coordinator
	nodesCoordinator sharding.NodesCoordinator
	miniblocksGetter process.MiniBlockDataGetter

	networkShardingCollector NetworkShardingCollector

	consensusTopic string
	consensusType  string

	isRunning                bool
	currentSendingGoRoutines int32
	bootstrapRoundIndex      uint64

	indexer                indexer.Indexer
	blocksBlackListHandler process.BlackListHandler
	bootStorer             process.BootStorer
	requestedItemsHandler  dataRetriever.RequestedItemsHandler
	headerSigVerifier      spos.RandSeedVerifier

	chainID                  []byte
	blockTracker             process.BlockTracker
	pendingMiniBlocksHandler process.PendingMiniBlocksHandler

	txStorageSize  uint32
	sizeCheckDelta uint32

	requestHandler process.RequestHandler

	inputAntifloodHandler P2PAntifloodHandler
	txAcumulator          Accumulator

	signatureSize int
	publicKeySize int

	chanStopNodeProcess chan bool
}

// ApplyOptions can set up different configurable options of a Node instance
func (n *Node) ApplyOptions(opts ...Option) error {
	if n.IsRunning() {
		return errors.New("cannot apply options while node is running")
	}
	for _, opt := range opts {
		err := opt(n)
		if err != nil {
			return errors.New("error applying option: " + err.Error())
		}
	}
	return nil
}

// NewNode creates a new Node instance
func NewNode(opts ...Option) (*Node, error) {
	node := &Node{
		ctx:                      context.Background(),
		currentSendingGoRoutines: 0,
		appStatusHandler:         statusHandler.NewNilStatusHandler(),
	}
	for _, opt := range opts {
		err := opt(node)
		if err != nil {
			return nil, errors.New("error applying option: " + err.Error())
		}
	}

	return node, nil
}

// GetAppStatusHandler will return the current status handler
func (n *Node) GetAppStatusHandler() core.AppStatusHandler {
	return n.appStatusHandler
}

// IsRunning will return the current state of the node
func (n *Node) IsRunning() bool {
	return n.isRunning
}

// TODO: delete useles IsRunning, Start and Stop - too many usages in tests for this PR.

// Start will set up the Node state as running
func (n *Node) Start() {
	n.isRunning = true
}

// Stop closes the messenger and undos everything done in Start
func (n *Node) Stop() error {
	if !n.IsRunning() {
		return nil
	}

	return nil
}

// CreateShardedStores instantiate sharded cachers for Transactions and Headers
func (n *Node) CreateShardedStores() error {
	if n.shardCoordinator == nil {
		return ErrNilShardCoordinator
	}

	if n.dataPool == nil {
		return ErrNilDataPool
	}

	transactionsDataStore := n.dataPool.Transactions()
	headersDataStore := n.dataPool.Headers()

	if transactionsDataStore == nil {
		return errors.New("nil transaction sharded data store")
	}

	if headersDataStore == nil {
		return errors.New("nil header sharded data store")
	}

	shards := n.shardCoordinator.NumberOfShards()
	currentShardId := n.shardCoordinator.SelfId()

	transactionsDataStore.CreateShardStore(process.ShardCacherIdentifier(currentShardId, currentShardId))
	for i := uint32(0); i < shards; i++ {
		if i == n.shardCoordinator.SelfId() {
			continue
		}
		transactionsDataStore.CreateShardStore(process.ShardCacherIdentifier(i, currentShardId))
		transactionsDataStore.CreateShardStore(process.ShardCacherIdentifier(currentShardId, i))
	}

	return nil
}

// StartConsensus will start the consensus service for the current node
func (n *Node) StartConsensus() error {
	isGenesisBlockNotInitialized := len(n.blkc.GetGenesisHeaderHash()) == 0 ||
		check.IfNil(n.blkc.GetGenesisHeader())
	if isGenesisBlockNotInitialized {
		return ErrGenesisBlockNotInitialized
	}

	chronologyHandler, err := n.createChronologyHandler(n.rounder, n.appStatusHandler)
	if err != nil {
		return err
	}

	bootstrapper, err := n.createBootstrapper(n.rounder)
	if err != nil {
		return err
	}

	err = bootstrapper.SetStatusHandler(n.GetAppStatusHandler())
	if err != nil {
		log.Debug("cannot set app status handler for shard bootstrapper")
	}

	bootstrapper.StartSync()
	epoch := uint32(0)
	crtBlockHeader := n.blkc.GetCurrentBlockHeader()
	if !check.IfNil(crtBlockHeader) {
		epoch = crtBlockHeader.GetEpoch()
	}
	log.Info("starting consensus", "epoch", epoch)

	consensusState, err := n.createConsensusState(epoch)
	if err != nil {
		return err
	}

	consensusService, err := sposFactory.GetConsensusCoreFactory(n.consensusType)
	if err != nil {
		return err
	}

	broadcastMessenger, err := sposFactory.GetBroadcastMessenger(
		n.internalMarshalizer,
		n.messenger,
		n.shardCoordinator,
		n.privKey,
		n.singleSigner)

	if err != nil {
		return err
	}

	netInputMarshalizer := n.internalMarshalizer
	if n.sizeCheckDelta > 0 {
		netInputMarshalizer = marshal.NewSizeCheckUnmarshalizer(n.internalMarshalizer, n.sizeCheckDelta)
	}

	workerArgs := &spos.WorkerArgs{
		ConsensusService:         consensusService,
		BlockChain:               n.blkc,
		BlockProcessor:           n.blockProcessor,
		Bootstrapper:             bootstrapper,
		BroadcastMessenger:       broadcastMessenger,
		ConsensusState:           consensusState,
		ForkDetector:             n.forkDetector,
		KeyGenerator:             n.keyGen,
		Marshalizer:              netInputMarshalizer,
		Hasher:                   n.hasher,
		Rounder:                  n.rounder,
		ShardCoordinator:         n.shardCoordinator,
		SingleSigner:             n.singleSigner,
		SyncTimer:                n.syncTimer,
		HeaderSigVerifier:        n.headerSigVerifier,
		ChainID:                  n.chainID,
		NetworkShardingCollector: n.networkShardingCollector,
		AntifloodHandler:         n.inputAntifloodHandler,
		PoolAdder:                n.dataPool.MiniBlocks(),
		SignatureSize:            n.signatureSize,
		PublicKeySize:            n.publicKeySize,
	}

	worker, err := spos.NewWorker(workerArgs)
	if err != nil {
		return err
	}

	n.dataPool.Headers().RegisterHandler(worker.ReceivedHeader)

	err = n.createConsensusTopic(worker)
	if err != nil {
		return err
	}

	consensusArgs := &spos.ConsensusCoreArgs{
		BlockChain:                    n.blkc,
		BlockProcessor:                n.blockProcessor,
		Bootstrapper:                  bootstrapper,
		BroadcastMessenger:            broadcastMessenger,
		ChronologyHandler:             chronologyHandler,
		Hasher:                        n.hasher,
		Marshalizer:                   n.internalMarshalizer,
		BlsPrivateKey:                 n.privKey,
		BlsSingleSigner:               n.singleSigner,
		MultiSigner:                   n.multiSigner,
		Rounder:                       n.rounder,
		ShardCoordinator:              n.shardCoordinator,
		NodesCoordinator:              n.nodesCoordinator,
		SyncTimer:                     n.syncTimer,
		EpochStartRegistrationHandler: n.epochStartRegistrationHandler,
		AntifloodHandler:              n.inputAntifloodHandler,
	}

	consensusDataContainer, err := spos.NewConsensusCore(
		consensusArgs,
	)
	if err != nil {
		return err
	}

	fct, err := sposFactory.GetSubroundsFactory(
		consensusDataContainer,
		consensusState,
		worker,
		n.consensusType,
		n.appStatusHandler,
		n.indexer,
		n.chainID,
	)
	if err != nil {
		return err
	}

	err = fct.GenerateSubrounds()
	if err != nil {
		return err
	}

	go chronologyHandler.StartRounds()

	return nil
}

// GetBalance gets the balance for a specific address
func (n *Node) GetBalance(addressHex string) (*big.Int, error) {
	if n.addrConverter == nil || n.addrConverter.IsInterfaceNil() || n.accounts == nil || n.accounts.IsInterfaceNil() {
		return nil, errors.New("initialize AccountsAdapter and AddressConverter first")
	}

	address, err := n.addrConverter.CreateAddressFromHex(addressHex)
	if err != nil {
		return nil, errors.New("invalid address, could not decode from hex: " + err.Error())
	}
	accWrp, err := n.accounts.GetExistingAccount(address)
	if err != nil {
		return nil, errors.New("could not fetch sender address from provided param: " + err.Error())
	}

	if accWrp == nil || accWrp.IsInterfaceNil() {
		return big.NewInt(0), nil
	}

	account, ok := accWrp.(state.UserAccountHandler)
	if !ok {
		return big.NewInt(0), nil
	}

	return account.GetBalance(), nil
}

// createChronologyHandler method creates a chronology object
func (n *Node) createChronologyHandler(rounder consensus.Rounder, appStatusHandler core.AppStatusHandler) (consensus.ChronologyHandler, error) {
	chr, err := chronology.NewChronology(
		n.genesisTime,
		rounder,
		n.syncTimer,
	)

	if err != nil {
		return nil, err
	}

	err = chr.SetAppStatusHandler(appStatusHandler)
	if err != nil {
		return nil, err
	}

	return chr, nil
}

//TODO move this func in structs.go
func (n *Node) createBootstrapper(rounder consensus.Rounder) (process.Bootstrapper, error) {
	miniblocksGetter, err := n.createMiniblocksGetter()
	if err != nil {
		return nil, err
	}

	n.miniblocksGetter = miniblocksGetter

	if n.shardCoordinator.SelfId() < n.shardCoordinator.NumberOfShards() {
		return n.createShardBootstrapper(rounder)
	}

	if n.shardCoordinator.SelfId() == core.MetachainShardId {
		return n.createMetaChainBootstrapper(rounder)
	}

	return nil, sharding.ErrShardIdOutOfRange
}

func (n *Node) createShardBootstrapper(rounder consensus.Rounder) (process.Bootstrapper, error) {
	argsBaseStorageBootstrapper := storageBootstrap.ArgsBaseStorageBootstrapper{
		BootStorer:          n.bootStorer,
		ForkDetector:        n.forkDetector,
		BlockProcessor:      n.blockProcessor,
		ChainHandler:        n.blkc,
		Marshalizer:         n.internalMarshalizer,
		Store:               n.store,
		Uint64Converter:     n.uint64ByteSliceConverter,
		BootstrapRoundIndex: n.bootstrapRoundIndex,
		ShardCoordinator:    n.shardCoordinator,
		NodesCoordinator:    n.nodesCoordinator,
		EpochStartTrigger:   n.epochStartTrigger,
		BlockTracker:        n.blockTracker,
	}

	argsShardStorageBootstrapper := storageBootstrap.ArgsShardStorageBootstrapper{
		ArgsBaseStorageBootstrapper: argsBaseStorageBootstrapper,
	}

	shardStorageBootstrapper, err := storageBootstrap.NewShardStorageBootstrapper(argsShardStorageBootstrapper)
	if err != nil {
		return nil, err
	}

	argsBaseBootstrapper := sync.ArgBaseBootstrapper{
		PoolsHolder:         n.dataPool,
		Store:               n.store,
		ChainHandler:        n.blkc,
		Rounder:             rounder,
		BlockProcessor:      n.blockProcessor,
		WaitTime:            n.rounder.TimeDuration(),
		Hasher:              n.hasher,
		Marshalizer:         n.internalMarshalizer,
		ForkDetector:        n.forkDetector,
		RequestHandler:      n.requestHandler,
		ShardCoordinator:    n.shardCoordinator,
		Accounts:            n.accounts,
		BlackListHandler:    n.blocksBlackListHandler,
		NetworkWatcher:      n.messenger,
		BootStorer:          n.bootStorer,
		StorageBootstrapper: shardStorageBootstrapper,
		EpochHandler:        n.epochStartTrigger,
		MiniblocksGetter:    n.miniblocksGetter,
		Uint64Converter:     n.uint64ByteSliceConverter,
	}

	argsShardBootstrapper := sync.ArgShardBootstrapper{
		ArgBaseBootstrapper: argsBaseBootstrapper,
	}

	bootstrap, err := sync.NewShardBootstrap(argsShardBootstrapper)
	if err != nil {
		return nil, err
	}

	return bootstrap, nil
}

func (n *Node) createMetaChainBootstrapper(rounder consensus.Rounder) (process.Bootstrapper, error) {
	argsBaseStorageBootstrapper := storageBootstrap.ArgsBaseStorageBootstrapper{
		BootStorer:          n.bootStorer,
		ForkDetector:        n.forkDetector,
		BlockProcessor:      n.blockProcessor,
		ChainHandler:        n.blkc,
		Marshalizer:         n.internalMarshalizer,
		Store:               n.store,
		Uint64Converter:     n.uint64ByteSliceConverter,
		BootstrapRoundIndex: n.bootstrapRoundIndex,
		ShardCoordinator:    n.shardCoordinator,
		NodesCoordinator:    n.nodesCoordinator,
		EpochStartTrigger:   n.epochStartTrigger,
		BlockTracker:        n.blockTracker,
	}

	argsMetaStorageBootstrapper := storageBootstrap.ArgsMetaStorageBootstrapper{
		ArgsBaseStorageBootstrapper: argsBaseStorageBootstrapper,
		PendingMiniBlocksHandler:    n.pendingMiniBlocksHandler,
	}

	metaStorageBootstrapper, err := storageBootstrap.NewMetaStorageBootstrapper(argsMetaStorageBootstrapper)
	if err != nil {
		return nil, err
	}

	argsBaseBootstrapper := sync.ArgBaseBootstrapper{
		PoolsHolder:         n.dataPool,
		Store:               n.store,
		ChainHandler:        n.blkc,
		Rounder:             rounder,
		BlockProcessor:      n.blockProcessor,
		WaitTime:            n.rounder.TimeDuration(),
		Hasher:              n.hasher,
		Marshalizer:         n.internalMarshalizer,
		ForkDetector:        n.forkDetector,
		RequestHandler:      n.requestHandler,
		ShardCoordinator:    n.shardCoordinator,
		Accounts:            n.accounts,
		BlackListHandler:    n.blocksBlackListHandler,
		NetworkWatcher:      n.messenger,
		BootStorer:          n.bootStorer,
		StorageBootstrapper: metaStorageBootstrapper,
		EpochHandler:        n.epochStartTrigger,
		MiniblocksGetter:    n.miniblocksGetter,
		Uint64Converter:     n.uint64ByteSliceConverter,
	}

	argsMetaBootstrapper := sync.ArgMetaBootstrapper{
		ArgBaseBootstrapper: argsBaseBootstrapper,
		EpochBootstrapper:   n.epochStartTrigger,
	}

	bootstrap, err := sync.NewMetaBootstrap(argsMetaBootstrapper)
	if err != nil {
		return nil, err
	}

	return bootstrap, nil
}

func (n *Node) createMiniblocksGetter() (process.MiniBlockDataGetter, error) {
	if check.IfNil(n.dataPool) {
		return nil, process.ErrNilPoolsHolder
	}
	if check.IfNil(n.store) {
		return nil, process.ErrNilStorage
	}

	arg := getters.ArgMiniBlockDataGetter{
		MiniBlockPool:    n.dataPool.MiniBlocks(),
		MiniBlockStorage: n.store.GetStorer(dataRetriever.MiniBlockUnit),
		Marshalizer:      n.internalMarshalizer,
	}

	return getters.NewMiniBlockDataGetter(arg)
}

// createConsensusState method creates a consensusState object
func (n *Node) createConsensusState(epoch uint32) (*spos.ConsensusState, error) {
	selfId, err := n.pubKey.ToByteArray()
	if err != nil {
		return nil, err
	}

	eligibleNodesPubKeys, err := n.nodesCoordinator.GetConsensusWhitelistedNodes(epoch)
	if err != nil {
		return nil, err
	}

	roundConsensus := spos.NewRoundConsensus(
		eligibleNodesPubKeys,
		n.consensusGroupSize,
		string(selfId))

	roundConsensus.ResetRoundState()

	roundThreshold := spos.NewRoundThreshold()

	roundStatus := spos.NewRoundStatus()
	roundStatus.ResetRoundStatus()

	consensusState := spos.NewConsensusState(
		roundConsensus,
		roundThreshold,
		roundStatus)

	return consensusState, nil
}

// createConsensusTopic creates a consensus topic for node
func (n *Node) createConsensusTopic(messageProcessor p2p.MessageProcessor) error {
	if check.IfNil(n.shardCoordinator) {
		return ErrNilShardCoordinator
	}
	if check.IfNil(messageProcessor) {
		return ErrNilMessenger
	}

	n.consensusTopic = core.ConsensusTopic + n.shardCoordinator.CommunicationIdentifier(n.shardCoordinator.SelfId())
	if n.messenger.HasTopicValidator(n.consensusTopic) {
		return ErrValidatorAlreadySet
	}

	if !n.messenger.HasTopic(n.consensusTopic) {
		err := n.messenger.CreateTopic(n.consensusTopic, true)
		if err != nil {
			return err
		}
	}

	return n.messenger.RegisterMessageProcessor(n.consensusTopic, messageProcessor)
}

// SendBulkTransactions sends the provided transactions as a bulk, optimizing transfer between nodes
func (n *Node) SendBulkTransactions(txs []*transaction.Transaction) (uint64, error) {
	if len(txs) == 0 {
		return 0, ErrNoTxToProcess
	}

	n.addTransactionsToSendPipe(txs)

	return uint64(len(txs)), nil
}

func (n *Node) addTransactionsToSendPipe(txs []*transaction.Transaction) {
	if check.IfNil(n.txAcumulator) {
		log.Error("node has a nil tx accumulator instance")
		return
	}

	for _, tx := range txs {
		n.txAcumulator.AddData(tx)
	}
}

func (n *Node) sendFromTxAccumulator() {
	outputChannel := n.txAcumulator.OutputChannel()

	for objs := range outputChannel {
		//this will read continuously until the chan is closed

		if len(objs) == 0 {
			continue
		}

		txs := make([]*transaction.Transaction, 0, len(objs))
		for _, obj := range objs {
			tx, ok := obj.(*transaction.Transaction)
			if !ok {
				continue
			}

			txs = append(txs, tx)
		}

		n.sendBulkTransactions(txs)
	}
}

// sendBulkTransactions sends the provided transactions as a bulk, optimizing transfer between nodes
func (n *Node) sendBulkTransactions(txs []*transaction.Transaction) {
	transactionsByShards := make(map[uint32][][]byte)

	for _, tx := range txs {
		senderShardId, err := n.getSenderShardId(tx)
		if err != nil {
			continue
		}

		marshalizedTx, err := n.internalMarshalizer.Marshal(tx)
		if err != nil {
			continue
		}

		transactionsByShards[senderShardId] = append(transactionsByShards[senderShardId], marshalizedTx)
	}

	numOfSentTxs := uint64(0)
	for shardId, txsForShard := range transactionsByShards {
		err := n.sendBulkTransactionsFromShard(txsForShard, shardId)
		if err != nil {
			log.Debug("sendBulkTransactionsFromShard", "error", err.Error())
		} else {
			numOfSentTxs += uint64(len(txsForShard))
		}
	}
}

func (n *Node) getSenderShardId(tx *transaction.Transaction) (uint32, error) {
	senderBytes, err := n.addrConverter.CreateAddressFromPublicKeyBytes(tx.SndAddr)
	if err != nil {
		return 0, err
	}

	senderShardId := n.shardCoordinator.ComputeId(senderBytes)
	if senderShardId != n.shardCoordinator.SelfId() {
		return senderShardId, nil
	}

	//tx is cross-shard with self, send it on the [transaction topic]_self_cross directly so it will
	//traverse the network only once
	recvBytes, err := n.addrConverter.CreateAddressFromPublicKeyBytes(tx.RcvAddr)
	if err != nil {
		return 0, err
	}

	return n.shardCoordinator.ComputeId(recvBytes), nil
}

// ValidateTransaction will validate a transaction
func (n *Node) ValidateTransaction(tx *transaction.Transaction) error {
	txValidator, err := dataValidators.NewTxValidator(
		n.accounts,
		n.shardCoordinator,
		n.whiteListHandler,
		core.MaxTxNonceDeltaAllowed,
	)
	if err != nil {
		return nil
	}

	marshalizedTx, err := n.internalMarshalizer.Marshal(tx)
	if err != nil {
		return err
	}

	intTx, err := procTx.NewInterceptedTransaction(
		marshalizedTx,
		n.internalMarshalizer,
		n.txSignMarshalizer,
		n.hasher,
		n.keyGenForAccounts,
		n.txSingleSigner,
		n.addrConverter,
		n.shardCoordinator,
		n.feeHandler,
	)
	if err != nil {
		return err
	}

	err = intTx.CheckValidity()
	if err != nil {
		return err
	}

	err = txValidator.CheckTxValidity(intTx)
	if errors.Is(err, process.ErrAccountNotFound) {
		// we allow the broadcast of provided transaction even if that transaction is not targeted on the current shard
		return nil
	}

	return err
}

func (n *Node) sendBulkTransactionsFromShard(transactions [][]byte, senderShardId uint32) error {
	dataPacker, err := partitioning.NewSimpleDataPacker(n.internalMarshalizer)
	if err != nil {
		return err
	}

	//the topic identifier is made of the current shard id and sender's shard id
	identifier := factory.TransactionTopic + n.shardCoordinator.CommunicationIdentifier(senderShardId)

	packets, err := dataPacker.PackDataInChunks(transactions, core.MaxBulkTransactionSize)
	if err != nil {
		return err
	}

	atomic.AddInt32(&n.currentSendingGoRoutines, int32(len(packets)))
	for _, buff := range packets {
		go func(bufferToSend []byte) {
			err = n.messenger.BroadcastOnChannelBlocking(
				SendTransactionsPipe,
				identifier,
				bufferToSend,
			)
			if err != nil {
				log.Debug("BroadcastOnChannelBlocking", "error", err.Error())
			}

			atomic.AddInt32(&n.currentSendingGoRoutines, -1)
		}(buff)
	}

	return nil
}

// CreateTransaction will return a transaction from all the required fields
func (n *Node) CreateTransaction(
	nonce uint64,
	value string,
	receiverHex string,
	senderHex string,
	gasPrice uint64,
	gasLimit uint64,
	dataField []byte,
	signatureHex string,
) (*transaction.Transaction, []byte, error) {

	if check.IfNil(n.addrConverter) {
		return nil, nil, ErrNilAddressConverter
	}
	if check.IfNil(n.accounts) {
		return nil, nil, ErrNilAccountsAdapter
	}

	receiverAddress, err := n.addrConverter.CreateAddressFromHex(receiverHex)
	if err != nil {
		return nil, nil, errors.New("could not create receiver address from provided param")
	}

	senderAddress, err := n.addrConverter.CreateAddressFromHex(senderHex)
	if err != nil {
		return nil, nil, errors.New("could not create sender address from provided param")
	}

	signatureBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return nil, nil, errors.New("could not fetch signature bytes")
	}

	valAsBigInt, ok := big.NewInt(0).SetString(value, 10)
	if !ok {
		return nil, nil, ErrInvalidValue
	}

	tx := &transaction.Transaction{
		Nonce:     nonce,
		Value:     valAsBigInt,
		RcvAddr:   receiverAddress.Bytes(),
		SndAddr:   senderAddress.Bytes(),
		GasPrice:  gasPrice,
		GasLimit:  gasLimit,
		Data:      dataField,
		Signature: signatureBytes,
	}

	var txHash []byte
	txHash, err = core.CalculateHash(n.internalMarshalizer, n.hasher, tx)
	if err != nil {
		return nil, nil, err
	}

	return tx, txHash, nil
}

//GetTransaction gets the transaction
func (n *Node) GetTransaction(_ string) (*transaction.Transaction, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// GetAccount will return account details for a given address
func (n *Node) GetAccount(address string) (state.UserAccountHandler, error) {
	if n.addrConverter == nil || n.addrConverter.IsInterfaceNil() {
		return nil, ErrNilAddressConverter
	}
	if n.accounts == nil || n.accounts.IsInterfaceNil() {
		return nil, ErrNilAccountsAdapter
	}

	addr, err := n.addrConverter.CreateAddressFromHex(address)
	if err != nil {
		return nil, err
	}

	accWrp, err := n.accounts.GetExistingAccount(addr)
	if err != nil {
		if err == state.ErrAccNotFound {
			return state.NewUserAccount(addr)
		}
		return nil, errors.New("could not fetch sender address from provided param: " + err.Error())
	}

	account, ok := accWrp.(state.UserAccountHandler)
	if !ok {
		return nil, errors.New("account is not of type with balance and nonce")
	}

	return account, nil
}

// StartHeartbeat starts the node's heartbeat processing/signaling module
func (n *Node) StartHeartbeat(hbConfig config.HeartbeatConfig, versionNumber string, nodeDisplayName string) error {
	if !hbConfig.Enabled {
		return nil
	}

	err := n.checkConfigParams(hbConfig)
	if err != nil {
		return err
	}

	if n.messenger.HasTopicValidator(core.HeartbeatTopic) {
		return ErrValidatorAlreadySet
	}

	if !n.messenger.HasTopic(core.HeartbeatTopic) {
		err = n.messenger.CreateTopic(core.HeartbeatTopic, true)
		if err != nil {
			return err
		}
	}

	peerTypeProvider, err := sharding.NewPeerTypeProvider(n.nodesCoordinator, n.epochStartTrigger, n.epochStartRegistrationHandler)
	if err != nil {
		return err
	}

	argSender := heartbeat.ArgHeartbeatSender{
		PeerMessenger:    n.messenger,
		SingleSigner:     n.singleSigner,
		PrivKey:          n.privKey,
		Marshalizer:      n.internalMarshalizer,
		Topic:            core.HeartbeatTopic,
		ShardCoordinator: n.shardCoordinator,
		PeerTypeProvider: peerTypeProvider,
		StatusHandler:    n.appStatusHandler,
		VersionNumber:    versionNumber,
		NodeDisplayName:  nodeDisplayName,
		HardforkTrigger:  n.hardforkTrigger,
	}

	n.heartbeatSender, err = heartbeat.NewSender(argSender)
	if err != nil {
		return err
	}

	heartbeatStorageUnit := n.store.GetStorer(dataRetriever.HeartbeatUnit)

	heartBeatMsgProcessor, err := heartbeat.NewMessageProcessor(
		n.singleSigner,
		n.keyGen,
		n.internalMarshalizer,
		n.networkShardingCollector,
	)
	if err != nil {
		return err
	}

	heartbeatStorer, err := storage.NewHeartbeatDbStorer(heartbeatStorageUnit, n.internalMarshalizer)
	if err != nil {
		return err
	}

	timer := &heartbeat.RealTimer{}
	netInputMarshalizer := n.internalMarshalizer
	if n.sizeCheckDelta > 0 {
		netInputMarshalizer = marshal.NewSizeCheckUnmarshalizer(n.internalMarshalizer, n.sizeCheckDelta)
	}

	allValidators, _, _ := n.getLatestValidators()
	pubKeysMap := make(map[uint32][]string)
	for shardID, valsInShard := range allValidators {
		for _, val := range valsInShard {
			pubKeysMap[shardID] = append(pubKeysMap[shardID], string(val.PublicKey))
		}
	}

	argMonitor := heartbeat.ArgHeartbeatMonitor{
		Marshalizer:                 netInputMarshalizer,
		MaxDurationPeerUnresponsive: time.Second * time.Duration(hbConfig.DurationInSecToConsiderUnresponsive),
		PubKeysMap:                  pubKeysMap,
		GenesisTime:                 n.genesisTime,
		MessageHandler:              heartBeatMsgProcessor,
		Storer:                      heartbeatStorer,
		PeerTypeProvider:            peerTypeProvider,
		Timer:                       timer,
		AntifloodHandler:            n.inputAntifloodHandler,
		HardforkTrigger:             n.hardforkTrigger,
		PeerBlackListHandler:        n.peerBlackListHandler,
	}
	n.heartbeatMonitor, err = heartbeat.NewMonitor(argMonitor)
	if err != nil {
		return err
	}

	err = n.heartbeatMonitor.SetAppStatusHandler(n.appStatusHandler)
	if err != nil {
		return err
	}

	err = n.messenger.RegisterMessageProcessor(core.HeartbeatTopic, n.heartbeatMonitor)
	if err != nil {
		return err
	}

	go n.startSendingHeartbeats(hbConfig)

	return nil
}

func (n *Node) checkConfigParams(config config.HeartbeatConfig) error {
	if config.DurationInSecToConsiderUnresponsive < 1 {
		return ErrNegativeDurationInSecToConsiderUnresponsive
	}
	if config.MaxTimeToWaitBetweenBroadcastsInSec < 1 {
		return ErrNegativeMaxTimeToWaitBetweenBroadcastsInSec
	}
	if config.MinTimeToWaitBetweenBroadcastsInSec < 1 {
		return ErrNegativeMinTimeToWaitBetweenBroadcastsInSec
	}
	if config.MaxTimeToWaitBetweenBroadcastsInSec <= config.MinTimeToWaitBetweenBroadcastsInSec {
		return ErrWrongValues
	}
	if config.DurationInSecToConsiderUnresponsive <= config.MaxTimeToWaitBetweenBroadcastsInSec {
		return ErrWrongValues
	}

	return nil
}

func (n *Node) startSendingHeartbeats(config config.HeartbeatConfig) {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	for {
		diffSeconds := config.MaxTimeToWaitBetweenBroadcastsInSec - config.MinTimeToWaitBetweenBroadcastsInSec
		diffNanos := int64(diffSeconds) * time.Second.Nanoseconds()
		randomNanos := r.Int63n(diffNanos)
		timeToWait := time.Second*time.Duration(config.MinTimeToWaitBetweenBroadcastsInSec) + time.Duration(randomNanos)

		time.Sleep(timeToWait)

		err := n.heartbeatSender.SendHeartbeat()
		if err != nil {
			log.Debug("SendHeartbeat", "error", err.Error())
		}
	}
}

// GetHeartbeats returns the heartbeat status for each public key defined in genesis.json
func (n *Node) GetHeartbeats() []heartbeat.PubKeyHeartbeat {
	if n.heartbeatMonitor == nil {
		return nil
	}
	return n.heartbeatMonitor.GetHeartbeats()
}

// ValidatorStatisticsApi will return the statistics for all the validators from the initial nodes pub keys
func (n *Node) ValidatorStatisticsApi() (map[string]*state.ValidatorApiResponse, error) {
	return n.validatorsProvider.GetLatestValidators(), nil
}

func (n *Node) getLatestValidators() (map[uint32][]*state.ValidatorInfo, map[string]*state.ValidatorApiResponse, error) {
	latestHash, err := n.validatorStatistics.RootHash()
	if err != nil {
		return nil, nil, err
	}

	validators, err := n.validatorStatistics.GetValidatorInfoForRootHash(latestHash)
	if err != nil {
		return nil, nil, err
	}

	return validators, nil, nil
}

// DirectTrigger will start the hardfork trigger
func (n *Node) DirectTrigger() error {
	return n.hardforkTrigger.Trigger()
}

// IsSelfTrigger returns true if the trigger's registered public key matches the self public key
func (n *Node) IsSelfTrigger() bool {
	return n.hardforkTrigger.IsSelfTrigger()
}

// IsInterfaceNil returns true if there is no value under the interface
func (n *Node) IsInterfaceNil() bool {
	return n == nil
}
