package factory

import (
	"errors"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/indexer"
	"github.com/ElrondNetwork/elrond-go/core/partitioning"
	"github.com/ElrondNetwork/elrond-go/core/statistics"
	"github.com/ElrondNetwork/elrond-go/core/statistics/softwareVersion"
	factorySoftwareVersion "github.com/ElrondNetwork/elrond-go/core/statistics/softwareVersion/factory"
	"github.com/ElrondNetwork/elrond-go/data"
	dataBlock "github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/factory/containers"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/factory/resolverscontainer"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/requestHandlers"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	metachainEpochStart "github.com/ElrondNetwork/elrond-go/epochStart/metachain"
	"github.com/ElrondNetwork/elrond-go/epochStart/shardchain"
	mainFactory "github.com/ElrondNetwork/elrond-go/factory"
	"github.com/ElrondNetwork/elrond-go/genesis"
	"github.com/ElrondNetwork/elrond-go/genesis/checking"
	genesisProcess "github.com/ElrondNetwork/elrond-go/genesis/process"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/process/block/pendingMb"
	"github.com/ElrondNetwork/elrond-go/process/block/poolsCleaner"
	"github.com/ElrondNetwork/elrond-go/process/block/postprocess"
	"github.com/ElrondNetwork/elrond-go/process/block/preprocess"
	"github.com/ElrondNetwork/elrond-go/process/coordinator"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/factory/interceptorscontainer"
	"github.com/ElrondNetwork/elrond-go/process/factory/metachain"
	"github.com/ElrondNetwork/elrond-go/process/factory/shard"
	"github.com/ElrondNetwork/elrond-go/process/headerCheck"
	"github.com/ElrondNetwork/elrond-go/process/peer"
	"github.com/ElrondNetwork/elrond-go/process/rewardTransaction"
	"github.com/ElrondNetwork/elrond-go/process/scToProtocol"
	"github.com/ElrondNetwork/elrond-go/process/smartContract"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/builtInFunctions"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/hooks"
	processSync "github.com/ElrondNetwork/elrond-go/process/sync"
	"github.com/ElrondNetwork/elrond-go/process/throttle"
	"github.com/ElrondNetwork/elrond-go/process/track"
	"github.com/ElrondNetwork/elrond-go/process/transaction"
	"github.com/ElrondNetwork/elrond-go/process/transactionLog"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/sharding/networksharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	storageFactory "github.com/ElrondNetwork/elrond-go/storage/factory"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/ElrondNetwork/elrond-go/storage/timecache"
	"github.com/ElrondNetwork/elrond-go/update"
	"github.com/ElrondNetwork/elrond-go/vm"
	"github.com/ElrondNetwork/elrond-vm-common/parsers"
)

const (
	// MaxTxsToRequest specifies the maximum number of txs to request
	MaxTxsToRequest = 1000
)

//TODO remove this
var log = logger.GetOrCreate("main")

// timeSpanForBadHeaders is the expiry time for an added block header hash
var timeSpanForBadHeaders = time.Minute * 2

// EpochStartNotifier defines which actions should be done for handling new epoch's events
type EpochStartNotifier interface {
	RegisterHandler(handler epochStart.ActionHandler)
	UnregisterHandler(handler epochStart.ActionHandler)
	NotifyAll(hdr data.HeaderHandler)
	NotifyAllPrepare(metaHdr data.HeaderHandler, body data.BodyHandler)
	RegisterForEpochChangeConfirmed(handler func(epoch uint32))
	NotifyEpochChangeConfirmed(epoch uint32)
	IsInterfaceNil() bool
}

// Process struct holds the process components
type Process struct {
	InterceptorsContainer    process.InterceptorsContainer
	ResolversFinder          dataRetriever.ResolversFinder
	Rounder                  consensus.Rounder
	EpochStartTrigger        epochStart.TriggerHandler
	ForkDetector             process.ForkDetector
	BlockProcessor           process.BlockProcessor
	BlackListHandler         process.TimeCacher
	BootStorer               process.BootStorer
	HeaderSigVerifier        HeaderSigVerifierHandler
	HeaderIntegrityVerifier  HeaderIntegrityVerifierHandler
	ValidatorsStatistics     process.ValidatorStatisticsProcessor
	ValidatorsProvider       process.ValidatorsProvider
	BlockTracker             process.BlockTracker
	PendingMiniBlocksHandler process.PendingMiniBlocksHandler
	RequestHandler           process.RequestHandler
	TxLogsProcessor          process.TransactionLogProcessorDatabase
	HeaderValidator          epochStart.HeaderValidator
}

type processComponentsFactoryArgs struct {
	coreComponents            *mainFactory.CoreComponentsFactoryArgs
	accountsParser            genesis.AccountsParser
	smartContractParser       genesis.InitialSmartContractParser
	economicsData             *economics.EconomicsData
	nodesConfig               *sharding.NodesSetup
	gasSchedule               map[string]map[string]uint64
	rounder                   consensus.Rounder
	shardCoordinator          sharding.Coordinator
	nodesCoordinator          sharding.NodesCoordinator
	data                      *mainFactory.DataComponents
	coreData                  *mainFactory.CoreComponents
	crypto                    *mainFactory.CryptoComponents
	state                     *mainFactory.StateComponents
	network                   *mainFactory.NetworkComponents
	tries                     *mainFactory.TriesComponents
	requestedItemsHandler     dataRetriever.RequestedItemsHandler
	whiteListHandler          process.WhiteListHandler
	whiteListerVerifiedTxs    process.WhiteListHandler
	epochStartNotifier        EpochStartNotifier
	mainConfig                config.Config
	epochStart                *config.EpochStartConfig
	rater                     sharding.PeerAccountListAndRatingHandler
	ratingsData               process.RatingsInfoHandler
	startEpochNum             uint32
	sizeCheckDelta            uint32
	stateCheckpointModulus    uint
	maxComputableRounds       uint64
	numConcurrentResolverJobs int32
	minSizeInBytes            uint32
	maxSizeInBytes            uint32
	maxRating                 uint32
	validatorPubkeyConverter  core.PubkeyConverter
	systemSCConfig            *config.SystemSmartContractsConfig
	txLogsProcessor           process.TransactionLogProcessor
	version                   string
	importStartHandler        update.ImportStartHandler
	workingDir                string
	indexer                   indexer.Indexer
	tpsBenchmark              statistics.TPSBenchmark
}

// NewProcessComponentsFactoryArgs initializes the arguments necessary for creating the process components
func NewProcessComponentsFactoryArgs(
	coreComponents *mainFactory.CoreComponentsFactoryArgs,
	accountsParser genesis.AccountsParser,
	smartContractParser genesis.InitialSmartContractParser,
	economicsData *economics.EconomicsData,
	nodesConfig *sharding.NodesSetup,
	gasSchedule map[string]map[string]uint64,
	rounder consensus.Rounder,
	shardCoordinator sharding.Coordinator,
	nodesCoordinator sharding.NodesCoordinator,
	data *mainFactory.DataComponents,
	coreData *mainFactory.CoreComponents,
	crypto *mainFactory.CryptoComponents,
	state *mainFactory.StateComponents,
	network *mainFactory.NetworkComponents,
	tries *mainFactory.TriesComponents,
	requestedItemsHandler dataRetriever.RequestedItemsHandler,
	whiteListHandler process.WhiteListHandler,
	whiteListerVerifiedTxs process.WhiteListHandler,
	epochStartNotifier EpochStartNotifier,
	mainConfig config.Config,
	startEpochNum uint32,
	rater sharding.PeerAccountListAndRatingHandler,
	sizeCheckDelta uint32,
	stateCheckpointModulus uint,
	maxComputableRounds uint64,
	numConcurrentResolverJobs int32,
	minSizeInBytes uint32,
	maxSizeInBytes uint32,
	maxRating uint32,
	validatorPubkeyConverter core.PubkeyConverter,
	ratingsData process.RatingsInfoHandler,
	systemSCConfig *config.SystemSmartContractsConfig,
	version string,
	importStartHandler update.ImportStartHandler,
	workingDir string,
	indexer indexer.Indexer,
	tpsBenchmark statistics.TPSBenchmark,
) *processComponentsFactoryArgs {
	return &processComponentsFactoryArgs{
		coreComponents:            coreComponents,
		accountsParser:            accountsParser,
		smartContractParser:       smartContractParser,
		economicsData:             economicsData,
		nodesConfig:               nodesConfig,
		gasSchedule:               gasSchedule,
		rounder:                   rounder,
		shardCoordinator:          shardCoordinator,
		nodesCoordinator:          nodesCoordinator,
		data:                      data,
		coreData:                  coreData,
		crypto:                    crypto,
		state:                     state,
		network:                   network,
		tries:                     tries,
		requestedItemsHandler:     requestedItemsHandler,
		whiteListHandler:          whiteListHandler,
		whiteListerVerifiedTxs:    whiteListerVerifiedTxs,
		epochStartNotifier:        epochStartNotifier,
		mainConfig:                mainConfig,
		epochStart:                &mainConfig.EpochStartConfig,
		startEpochNum:             startEpochNum,
		rater:                     rater,
		ratingsData:               ratingsData,
		sizeCheckDelta:            sizeCheckDelta,
		stateCheckpointModulus:    stateCheckpointModulus,
		maxComputableRounds:       maxComputableRounds,
		numConcurrentResolverJobs: numConcurrentResolverJobs,
		minSizeInBytes:            minSizeInBytes,
		maxSizeInBytes:            maxSizeInBytes,
		maxRating:                 maxRating,
		validatorPubkeyConverter:  validatorPubkeyConverter,
		systemSCConfig:            systemSCConfig,
		version:                   version,
		importStartHandler:        importStartHandler,
		workingDir:                workingDir,
		indexer:                   indexer,
		tpsBenchmark:              tpsBenchmark,
	}
}

// ProcessComponentsFactory creates the process components
func ProcessComponentsFactory(args *processComponentsFactoryArgs) (*Process, error) {
	argsHeaderSig := &headerCheck.ArgsHeaderSigVerifier{
		Marshalizer:       args.coreData.InternalMarshalizer,
		Hasher:            args.coreData.Hasher,
		NodesCoordinator:  args.nodesCoordinator,
		MultiSigVerifier:  args.crypto.MultiSigner,
		SingleSigVerifier: args.crypto.SingleSigner,
		KeyGen:            args.crypto.BlockSignKeyGen,
	}
	headerSigVerifier, err := headerCheck.NewHeaderSigVerifier(argsHeaderSig)
	if err != nil {
		return nil, err
	}
	headerIntegrityVerifier, err := headerCheck.NewHeaderIntegrityVerifier([]byte(args.nodesConfig.ChainID))
	if err != nil {
		return nil, err
	}

	resolversContainerFactory, err := newResolverContainerFactory(
		args.shardCoordinator,
		args.data,
		args.coreData,
		args.network,
		args.tries,
		args.sizeCheckDelta,
		args.numConcurrentResolverJobs,
	)
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

	requestHandler, err := requestHandlers.NewResolverRequestHandler(
		resolversFinder,
		args.requestedItemsHandler,
		args.whiteListHandler,
		MaxTxsToRequest,
		args.shardCoordinator.SelfId(),
		time.Second,
	)
	if err != nil {
		return nil, err
	}

	txLogsStorage := args.data.Store.GetStorer(dataRetriever.TxLogsUnit)
	txLogsProcessor, err := transactionLog.NewTxLogProcessor(transactionLog.ArgTxLogProcessor{
		Storer:      txLogsStorage,
		Marshalizer: args.coreData.InternalMarshalizer,
	})
	if err != nil {
		return nil, err
	}

	args.txLogsProcessor = txLogsProcessor
	genesisBlocks, err := generateGenesisHeadersAndApplyInitialBalances(args, args.workingDir)
	if err != nil {
		return nil, err
	}

	err = setGenesisHeader(args, genesisBlocks)
	if err != nil {
		return nil, err
	}

	validatorStatisticsProcessor, err := newValidatorStatisticsProcessor(args)
	if err != nil {
		return nil, err
	}

	cacheRefreshDuration := time.Duration(args.mainConfig.ValidatorStatistics.CacheRefreshIntervalInSec) * time.Second
	argVSP := peer.ArgValidatorsProvider{
		NodesCoordinator:                  args.nodesCoordinator,
		StartEpoch:                        args.startEpochNum,
		EpochStartEventNotifier:           args.epochStartNotifier,
		CacheRefreshIntervalDurationInSec: cacheRefreshDuration,
		ValidatorStatistics:               validatorStatisticsProcessor,
		MaxRating:                         args.maxRating,
		PubKeyConverter:                   args.validatorPubkeyConverter,
	}

	validatorsProvider, err := peer.NewValidatorsProvider(argVSP)
	if err != nil {
		return nil, err
	}

	epochStartTrigger, err := newEpochStartTrigger(args, requestHandler)
	if err != nil {
		return nil, err
	}

	requestHandler.SetEpoch(epochStartTrigger.Epoch())

	err = dataRetriever.SetEpochHandlerToHdrResolver(resolversContainer, epochStartTrigger)
	if err != nil {
		return nil, err
	}

	validatorStatsRootHash, err := validatorStatisticsProcessor.RootHash()
	if err != nil {
		return nil, err
	}

	log.Debug("Validator stats created", "validatorStatsRootHash", validatorStatsRootHash)

	genesisMetaBlock, ok := genesisBlocks[core.MetachainShardId]
	if !ok {
		return nil, errors.New("genesis meta block does not exist")
	}

	genesisMetaBlock.SetValidatorStatsRootHash(validatorStatsRootHash)
	err = prepareGenesisBlock(args, genesisBlocks)
	if err != nil {
		return nil, err
	}

	bootStr := args.data.Store.GetStorer(dataRetriever.BootstrapUnit)
	bootStorer, err := bootstrapStorage.NewBootstrapStorer(args.coreData.InternalMarshalizer, bootStr)
	if err != nil {
		return nil, err
	}

	argsHeaderValidator := block.ArgsHeaderValidator{
		Hasher:      args.coreData.Hasher,
		Marshalizer: args.coreData.InternalMarshalizer,
	}
	headerValidator, err := block.NewHeaderValidator(argsHeaderValidator)
	if err != nil {
		return nil, err
	}

	blockTracker, err := newBlockTracker(
		args,
		headerValidator,
		requestHandler,
		args.rounder,
		genesisBlocks,
	)
	if err != nil {
		return nil, err
	}

	mbsPoolsCleaner, err := poolsCleaner.NewMiniBlocksPoolsCleaner(
		args.data.Datapool.MiniBlocks(),
		args.rounder,
		args.shardCoordinator,
	)
	if err != nil {
		return nil, err
	}

	mbsPoolsCleaner.StartCleaning()

	txsPoolsCleaner, err := poolsCleaner.NewTxsPoolsCleaner(
		args.state.AddressPubkeyConverter,
		args.data.Datapool,
		args.rounder,
		args.shardCoordinator,
	)
	if err != nil {
		return nil, err
	}

	txsPoolsCleaner.StartCleaning()

	_, err = track.NewMiniBlockTrack(args.data.Datapool, args.shardCoordinator, args.whiteListHandler)
	if err != nil {
		return nil, err
	}

	interceptorContainerFactory, blackListHandler, err := newInterceptorContainerFactory(
		args.shardCoordinator,
		args.nodesCoordinator,
		args.data,
		args.coreData,
		args.crypto,
		args.state,
		args.network,
		args.economicsData,
		headerSigVerifier,
		headerIntegrityVerifier,
		args.sizeCheckDelta,
		blockTracker,
		epochStartTrigger,
		args.whiteListHandler,
		args.whiteListerVerifiedTxs,
	)
	if err != nil {
		return nil, err
	}

	//TODO refactor all these factory calls
	interceptorsContainer, err := interceptorContainerFactory.Create()
	if err != nil {
		return nil, err
	}

	var pendingMiniBlocksHandler process.PendingMiniBlocksHandler
	if args.shardCoordinator.SelfId() == core.MetachainShardId {
		pendingMiniBlocksHandler, err = pendingMb.NewPendingMiniBlocks()
		if err != nil {
			return nil, err
		}
	}

	forkDetector, err := newForkDetector(
		args.rounder,
		args.shardCoordinator,
		blackListHandler,
		blockTracker,
		args.nodesConfig.StartTime,
	)
	if err != nil {
		return nil, err
	}

	blockProcessor, err := newBlockProcessor(
		args,
		requestHandler,
		forkDetector,
		epochStartTrigger,
		bootStorer,
		validatorStatisticsProcessor,
		headerValidator,
		blockTracker,
		pendingMiniBlocksHandler,
	)
	if err != nil {
		return nil, err
	}

	nodesSetupChecker, err := checking.NewNodesSetupChecker(
		args.accountsParser,
		args.economicsData.GenesisNodePrice(),
		args.validatorPubkeyConverter,
		args.crypto.BlockSignKeyGen,
	)
	if err != nil {
		return nil, err
	}

	err = nodesSetupChecker.Check(args.nodesConfig.AllInitialNodes())
	if err != nil {
		return nil, err
	}

	return &Process{
		InterceptorsContainer:    interceptorsContainer,
		ResolversFinder:          resolversFinder,
		Rounder:                  args.rounder,
		ForkDetector:             forkDetector,
		BlockProcessor:           blockProcessor,
		EpochStartTrigger:        epochStartTrigger,
		BlackListHandler:         blackListHandler,
		BootStorer:               bootStorer,
		HeaderSigVerifier:        headerSigVerifier,
		HeaderIntegrityVerifier:  headerIntegrityVerifier,
		ValidatorsStatistics:     validatorStatisticsProcessor,
		ValidatorsProvider:       validatorsProvider,
		BlockTracker:             blockTracker,
		PendingMiniBlocksHandler: pendingMiniBlocksHandler,
		RequestHandler:           requestHandler,
		TxLogsProcessor:          txLogsProcessor,
		HeaderValidator:          headerValidator,
	}, nil
}

func setGenesisHeader(args *processComponentsFactoryArgs, genesisBlocks map[uint32]data.HeaderHandler) error {
	genesisBlock, ok := genesisBlocks[args.shardCoordinator.SelfId()]
	if !ok {
		return errors.New("genesis block does not exist")
	}

	err := args.data.Blkc.SetGenesisHeader(genesisBlock)
	if err != nil {
		return err
	}

	return nil
}

func prepareGenesisBlock(args *processComponentsFactoryArgs, genesisBlocks map[uint32]data.HeaderHandler) error {
	genesisBlock, ok := genesisBlocks[args.shardCoordinator.SelfId()]
	if !ok {
		return errors.New("genesis block does not exist")
	}

	genesisBlockHash, err := core.CalculateHash(args.coreData.InternalMarshalizer, args.coreData.Hasher, genesisBlock)
	if err != nil {
		return err
	}

	err = args.data.Blkc.SetGenesisHeader(genesisBlock)
	if err != nil {
		return err
	}

	args.data.Blkc.SetGenesisHeaderHash(genesisBlockHash)

	marshalizedBlock, err := args.coreData.InternalMarshalizer.Marshal(genesisBlock)
	if err != nil {
		return err
	}

	if args.shardCoordinator.SelfId() == core.MetachainShardId {
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

func newEpochStartTrigger(
	args *processComponentsFactoryArgs,
	requestHandler process.RequestHandler,
) (epochStart.TriggerHandler, error) {
	if args.shardCoordinator.SelfId() < args.shardCoordinator.NumberOfShards() {
		argsHeaderValidator := block.ArgsHeaderValidator{
			Hasher:      args.coreData.Hasher,
			Marshalizer: args.coreData.InternalMarshalizer,
		}
		headerValidator, err := block.NewHeaderValidator(argsHeaderValidator)
		if err != nil {
			return nil, err
		}

		argsPeerMiniBlockSyncer := shardchain.ArgPeerMiniBlockSyncer{
			MiniBlocksPool: args.data.Datapool.MiniBlocks(),
			Requesthandler: requestHandler,
		}

		peerMiniBlockSyncer, err := shardchain.NewPeerMiniBlockSyncer(argsPeerMiniBlockSyncer)
		if err != nil {
			return nil, err
		}

		argEpochStart := &shardchain.ArgsShardEpochStartTrigger{
			Marshalizer:          args.coreData.InternalMarshalizer,
			Hasher:               args.coreData.Hasher,
			HeaderValidator:      headerValidator,
			Uint64Converter:      args.coreData.Uint64ByteSliceConverter,
			DataPool:             args.data.Datapool,
			Storage:              args.data.Store,
			RequestHandler:       requestHandler,
			Epoch:                args.startEpochNum,
			EpochStartNotifier:   args.epochStartNotifier,
			Validity:             process.MetaBlockValidity,
			Finality:             process.BlockFinality,
			PeerMiniBlocksSyncer: peerMiniBlockSyncer,
			Rounder:              args.rounder,
		}
		epochStartTrigger, err := shardchain.NewEpochStartTrigger(argEpochStart)
		if err != nil {
			return nil, errors.New("error creating new start of epoch trigger" + err.Error())
		}
		err = epochStartTrigger.SetAppStatusHandler(args.coreData.StatusHandler)
		if err != nil {
			return nil, err
		}

		return epochStartTrigger, nil
	}

	if args.shardCoordinator.SelfId() == core.MetachainShardId {
		argEpochStart := &metachainEpochStart.ArgsNewMetaEpochStartTrigger{
			GenesisTime:        time.Unix(args.nodesConfig.StartTime, 0),
			Settings:           args.epochStart,
			Epoch:              args.startEpochNum,
			EpochStartRound:    args.data.Blkc.GetGenesisHeader().GetRound(),
			EpochStartNotifier: args.epochStartNotifier,
			Storage:            args.data.Store,
			Marshalizer:        args.coreData.InternalMarshalizer,
			Hasher:             args.coreData.Hasher,
		}
		epochStartTrigger, err := metachainEpochStart.NewEpochStartTrigger(argEpochStart)
		if err != nil {
			return nil, errors.New("error creating new start of epoch trigger" + err.Error())
		}
		err = epochStartTrigger.SetAppStatusHandler(args.coreData.StatusHandler)
		if err != nil {
			return nil, err
		}

		return epochStartTrigger, nil
	}

	return nil, errors.New("error creating new start of epoch trigger because of invalid shard id")
}

// CreateSoftwareVersionChecker will create a new software version checker and will start check if a new software version
// is available
func CreateSoftwareVersionChecker(
	statusHandler core.AppStatusHandler,
	config config.SoftwareVersionConfig,
) (*softwareVersion.SoftwareVersionChecker, error) {
	softwareVersionCheckerFactory, err := factorySoftwareVersion.NewSoftwareVersionFactory(statusHandler, config)
	if err != nil {
		return nil, err
	}

	softwareVersionChecker, err := softwareVersionCheckerFactory.Create()
	if err != nil {
		return nil, err
	}

	return softwareVersionChecker, nil
}

func newInterceptorContainerFactory(
	shardCoordinator sharding.Coordinator,
	nodesCoordinator sharding.NodesCoordinator,
	data *mainFactory.DataComponents,
	coreData *mainFactory.CoreComponents,
	crypto *mainFactory.CryptoComponents,
	state *mainFactory.StateComponents,
	network *mainFactory.NetworkComponents,
	economics *economics.EconomicsData,
	headerSigVerifier HeaderSigVerifierHandler,
	headerIntegrityVerifier HeaderIntegrityVerifierHandler,
	sizeCheckDelta uint32,
	validityAttester process.ValidityAttester,
	epochStartTrigger process.EpochStartTriggerHandler,
	whiteListHandler process.WhiteListHandler,
	whiteListerVerifiedTxs process.WhiteListHandler,
) (process.InterceptorsContainerFactory, process.TimeCacher, error) {
	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		return newShardInterceptorContainerFactory(
			shardCoordinator,
			nodesCoordinator,
			data,
			coreData,
			crypto,
			state,
			network,
			economics,
			headerSigVerifier,
			headerIntegrityVerifier,
			sizeCheckDelta,
			validityAttester,
			epochStartTrigger,
			whiteListHandler,
			whiteListerVerifiedTxs,
		)
	}
	if shardCoordinator.SelfId() == core.MetachainShardId {
		return newMetaInterceptorContainerFactory(
			shardCoordinator,
			nodesCoordinator,
			data,
			coreData,
			crypto,
			network,
			state,
			economics,
			headerSigVerifier,
			headerIntegrityVerifier,
			sizeCheckDelta,
			validityAttester,
			epochStartTrigger,
			whiteListHandler,
			whiteListerVerifiedTxs,
		)
	}

	return nil, nil, errors.New("could not create interceptor container factory")
}

func newResolverContainerFactory(
	shardCoordinator sharding.Coordinator,
	data *mainFactory.DataComponents,
	coreData *mainFactory.CoreComponents,
	network *mainFactory.NetworkComponents,
	tries *mainFactory.TriesComponents,
	sizeCheckDelta uint32,
	numConcurrentResolverJobs int32,
) (dataRetriever.ResolversContainerFactory, error) {

	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		return newShardResolverContainerFactory(
			shardCoordinator,
			data,
			coreData,
			network,
			tries,
			sizeCheckDelta,
			numConcurrentResolverJobs,
		)
	}
	if shardCoordinator.SelfId() == core.MetachainShardId {
		return newMetaResolverContainerFactory(
			shardCoordinator,
			data,
			coreData,
			network,
			tries,
			sizeCheckDelta,
			numConcurrentResolverJobs,
		)
	}

	return nil, errors.New("could not create interceptor and resolver container factory")
}

func newShardInterceptorContainerFactory(
	shardCoordinator sharding.Coordinator,
	nodesCoordinator sharding.NodesCoordinator,
	data *mainFactory.DataComponents,
	dataCore *mainFactory.CoreComponents,
	crypto *mainFactory.CryptoComponents,
	state *mainFactory.StateComponents,
	network *mainFactory.NetworkComponents,
	economics *economics.EconomicsData,
	headerSigVerifier HeaderSigVerifierHandler,
	headerIntegrityVerifier HeaderIntegrityVerifierHandler,
	sizeCheckDelta uint32,
	validityAttester process.ValidityAttester,
	epochStartTrigger process.EpochStartTriggerHandler,
	whiteListHandler process.WhiteListHandler,
	whiteListerVerifiedTxs process.WhiteListHandler,
) (process.InterceptorsContainerFactory, process.TimeCacher, error) {
	headerBlackList := timecache.NewTimeCache(timeSpanForBadHeaders)
	shardInterceptorsContainerFactoryArgs := interceptorscontainer.ShardInterceptorsContainerFactoryArgs{
		Accounts:                state.AccountsAdapter,
		ShardCoordinator:        shardCoordinator,
		NodesCoordinator:        nodesCoordinator,
		Messenger:               network.NetMessenger,
		Store:                   data.Store,
		ProtoMarshalizer:        dataCore.InternalMarshalizer,
		TxSignMarshalizer:       dataCore.TxSignMarshalizer,
		Hasher:                  dataCore.Hasher,
		KeyGen:                  crypto.TxSignKeyGen,
		BlockSignKeyGen:         crypto.BlockSignKeyGen,
		SingleSigner:            crypto.TxSingleSigner,
		BlockSingleSigner:       crypto.SingleSigner,
		MultiSigner:             crypto.MultiSigner,
		DataPool:                data.Datapool,
		AddressPubkeyConverter:  state.AddressPubkeyConverter,
		MaxTxNonceDeltaAllowed:  core.MaxTxNonceDeltaAllowed,
		TxFeeHandler:            economics,
		BlockBlackList:          headerBlackList,
		HeaderSigVerifier:       headerSigVerifier,
		HeaderIntegrityVerifier: headerIntegrityVerifier,
		SizeCheckDelta:          sizeCheckDelta,
		ValidityAttester:        validityAttester,
		EpochStartTrigger:       epochStartTrigger,
		WhiteListHandler:        whiteListHandler,
		WhiteListerVerifiedTxs:  whiteListerVerifiedTxs,
		AntifloodHandler:        network.InputAntifloodHandler,
		ArgumentsParser:         smartContract.NewArgumentParser(),
		ChainID:                 dataCore.ChainID,
		MinTransactionVersion:   dataCore.MinTransactionVersion,
	}
	interceptorContainerFactory, err := interceptorscontainer.NewShardInterceptorsContainerFactory(shardInterceptorsContainerFactoryArgs)
	if err != nil {
		return nil, nil, err
	}

	return interceptorContainerFactory, headerBlackList, nil
}

func newMetaInterceptorContainerFactory(
	shardCoordinator sharding.Coordinator,
	nodesCoordinator sharding.NodesCoordinator,
	data *mainFactory.DataComponents,
	dataCore *mainFactory.CoreComponents,
	crypto *mainFactory.CryptoComponents,
	network *mainFactory.NetworkComponents,
	state *mainFactory.StateComponents,
	economics *economics.EconomicsData,
	headerSigVerifier HeaderSigVerifierHandler,
	headerIntegrityVerifier HeaderIntegrityVerifierHandler,
	sizeCheckDelta uint32,
	validityAttester process.ValidityAttester,
	epochStartTrigger process.EpochStartTriggerHandler,
	whiteListHandler process.WhiteListHandler,
	whiteListerVerifiedTxs process.WhiteListHandler,
) (process.InterceptorsContainerFactory, process.TimeCacher, error) {
	headerBlackList := timecache.NewTimeCache(timeSpanForBadHeaders)
	metaInterceptorsContainerFactoryArgs := interceptorscontainer.MetaInterceptorsContainerFactoryArgs{
		ShardCoordinator:        shardCoordinator,
		NodesCoordinator:        nodesCoordinator,
		Messenger:               network.NetMessenger,
		Store:                   data.Store,
		ProtoMarshalizer:        dataCore.InternalMarshalizer,
		TxSignMarshalizer:       dataCore.TxSignMarshalizer,
		Hasher:                  dataCore.Hasher,
		MultiSigner:             crypto.MultiSigner,
		DataPool:                data.Datapool,
		Accounts:                state.AccountsAdapter,
		AddressPubkeyConverter:  state.AddressPubkeyConverter,
		SingleSigner:            crypto.TxSingleSigner,
		BlockSingleSigner:       crypto.SingleSigner,
		KeyGen:                  crypto.TxSignKeyGen,
		BlockKeyGen:             crypto.BlockSignKeyGen,
		MaxTxNonceDeltaAllowed:  core.MaxTxNonceDeltaAllowed,
		TxFeeHandler:            economics,
		BlackList:               headerBlackList,
		HeaderSigVerifier:       headerSigVerifier,
		HeaderIntegrityVerifier: headerIntegrityVerifier,
		SizeCheckDelta:          sizeCheckDelta,
		ValidityAttester:        validityAttester,
		EpochStartTrigger:       epochStartTrigger,
		WhiteListHandler:        whiteListHandler,
		WhiteListerVerifiedTxs:  whiteListerVerifiedTxs,
		AntifloodHandler:        network.InputAntifloodHandler,
		ArgumentsParser:         smartContract.NewArgumentParser(),
		ChainID:                 dataCore.ChainID,
		MinTransactionVersion:   dataCore.MinTransactionVersion,
	}
	interceptorContainerFactory, err := interceptorscontainer.NewMetaInterceptorsContainerFactory(metaInterceptorsContainerFactoryArgs)
	if err != nil {
		return nil, nil, err
	}

	return interceptorContainerFactory, headerBlackList, nil
}

func newShardResolverContainerFactory(
	shardCoordinator sharding.Coordinator,
	data *mainFactory.DataComponents,
	core *mainFactory.CoreComponents,
	network *mainFactory.NetworkComponents,
	tries *mainFactory.TriesComponents,
	sizeCheckDelta uint32,
	numConcurrentResolverJobs int32,
) (dataRetriever.ResolversContainerFactory, error) {

	dataPacker, err := partitioning.NewSimpleDataPacker(core.InternalMarshalizer)
	if err != nil {
		return nil, err
	}

	resolversContainerFactoryArgs := resolverscontainer.FactoryArgs{
		ShardCoordinator:           shardCoordinator,
		Messenger:                  network.NetMessenger,
		Store:                      data.Store,
		Marshalizer:                core.InternalMarshalizer,
		DataPools:                  data.Datapool,
		Uint64ByteSliceConverter:   core.Uint64ByteSliceConverter,
		DataPacker:                 dataPacker,
		TriesContainer:             tries.TriesContainer,
		SizeCheckDelta:             sizeCheckDelta,
		InputAntifloodHandler:      network.InputAntifloodHandler,
		OutputAntifloodHandler:     network.OutputAntifloodHandler,
		NumConcurrentResolvingJobs: numConcurrentResolverJobs,
	}
	resolversContainerFactory, err := resolverscontainer.NewShardResolversContainerFactory(resolversContainerFactoryArgs)
	if err != nil {
		return nil, err
	}

	return resolversContainerFactory, nil
}

func newMetaResolverContainerFactory(
	shardCoordinator sharding.Coordinator,
	data *mainFactory.DataComponents,
	core *mainFactory.CoreComponents,
	network *mainFactory.NetworkComponents,
	tries *mainFactory.TriesComponents,
	sizeCheckDelta uint32,
	numConcurrentResolverJobs int32,
) (dataRetriever.ResolversContainerFactory, error) {
	dataPacker, err := partitioning.NewSimpleDataPacker(core.InternalMarshalizer)
	if err != nil {
		return nil, err
	}

	resolversContainerFactoryArgs := resolverscontainer.FactoryArgs{
		ShardCoordinator:           shardCoordinator,
		Messenger:                  network.NetMessenger,
		Store:                      data.Store,
		Marshalizer:                core.InternalMarshalizer,
		DataPools:                  data.Datapool,
		Uint64ByteSliceConverter:   core.Uint64ByteSliceConverter,
		DataPacker:                 dataPacker,
		TriesContainer:             tries.TriesContainer,
		SizeCheckDelta:             sizeCheckDelta,
		InputAntifloodHandler:      network.InputAntifloodHandler,
		OutputAntifloodHandler:     network.OutputAntifloodHandler,
		NumConcurrentResolvingJobs: numConcurrentResolverJobs,
	}
	resolversContainerFactory, err := resolverscontainer.NewMetaResolversContainerFactory(resolversContainerFactoryArgs)
	if err != nil {
		return nil, err
	}
	return resolversContainerFactory, nil
}

func generateGenesisHeadersAndApplyInitialBalances(args *processComponentsFactoryArgs, workingDir string) (map[uint32]data.HeaderHandler, error) {
	coreComponents := args.coreData
	stateComponents := args.state
	dataComponents := args.data
	shardCoordinator := args.shardCoordinator
	nodesSetup := args.nodesConfig
	accountsParser := args.accountsParser
	smartContractParser := args.smartContractParser
	economicsData := args.economicsData

	genesisVmConfig := args.mainConfig.VirtualMachineConfig
	genesisVmConfig.OutOfProcessEnabled = false

	arg := genesisProcess.ArgsGenesisBlockCreator{
		GenesisTime:              uint64(nodesSetup.StartTime),
		StartEpochNum:            args.startEpochNum,
		Accounts:                 stateComponents.AccountsAdapter,
		PubkeyConv:               stateComponents.AddressPubkeyConverter,
		InitialNodesSetup:        nodesSetup,
		Economics:                economicsData,
		ShardCoordinator:         shardCoordinator,
		Store:                    dataComponents.Store,
		Blkc:                     dataComponents.Blkc,
		Marshalizer:              coreComponents.InternalMarshalizer,
		SignMarshalizer:          coreComponents.TxSignMarshalizer,
		Hasher:                   coreComponents.Hasher,
		Uint64ByteSliceConverter: coreComponents.Uint64ByteSliceConverter,
		DataPool:                 dataComponents.Datapool,
		AccountsParser:           accountsParser,
		SmartContractParser:      smartContractParser,
		ValidatorAccounts:        stateComponents.PeerAccounts,
		GasMap:                   args.gasSchedule,
		VirtualMachineConfig:     genesisVmConfig,
		TxLogsProcessor:          args.txLogsProcessor,
		HardForkConfig:           args.mainConfig.Hardfork,
		TrieStorageManagers:      args.tries.TrieStorageManagers,
		ChainID:                  string(args.coreComponents.ChainID),
		SystemSCConfig:           *args.systemSCConfig,
		BlockSignKeyGen:          args.crypto.BlockSignKeyGen,
		ImportStartHandler:       args.importStartHandler,
		WorkingDir:               workingDir,
	}

	gbc, err := genesisProcess.NewGenesisBlockCreator(arg)
	if err != nil {
		return nil, err
	}

	return gbc.CreateGenesisBlocks()
}

func newBlockTracker(
	processArgs *processComponentsFactoryArgs,
	headerValidator process.HeaderConstructionValidator,
	requestHandler process.RequestHandler,
	rounder process.Rounder,
	genesisBlocks map[uint32]data.HeaderHandler,
) (process.BlockTracker, error) {

	argBaseTracker := track.ArgBaseTracker{
		Hasher:           processArgs.coreData.Hasher,
		HeaderValidator:  headerValidator,
		Marshalizer:      processArgs.coreData.InternalMarshalizer,
		RequestHandler:   requestHandler,
		Rounder:          rounder,
		ShardCoordinator: processArgs.shardCoordinator,
		Store:            processArgs.data.Store,
		StartHeaders:     genesisBlocks,
		PoolsHolder:      processArgs.data.Datapool,
		WhitelistHandler: processArgs.whiteListHandler,
	}

	if processArgs.shardCoordinator.SelfId() < processArgs.shardCoordinator.NumberOfShards() {
		arguments := track.ArgShardTracker{
			ArgBaseTracker: argBaseTracker,
		}

		return track.NewShardBlockTrack(arguments)
	}

	if processArgs.shardCoordinator.SelfId() == core.MetachainShardId {
		arguments := track.ArgMetaTracker{
			ArgBaseTracker: argBaseTracker,
		}

		return track.NewMetaBlockTrack(arguments)
	}

	return nil, errors.New("could not create block tracker")
}

func newForkDetector(
	rounder consensus.Rounder,
	shardCoordinator sharding.Coordinator,
	headerBlackList process.TimeCacher,
	blockTracker process.BlockTracker,
	genesisTime int64,
) (process.ForkDetector, error) {
	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		return processSync.NewShardForkDetector(rounder, headerBlackList, blockTracker, genesisTime)
	}
	if shardCoordinator.SelfId() == core.MetachainShardId {
		return processSync.NewMetaForkDetector(rounder, headerBlackList, blockTracker, genesisTime)
	}

	return nil, errors.New("could not create fork detector")
}

func newBlockProcessor(
	processArgs *processComponentsFactoryArgs,
	requestHandler process.RequestHandler,
	forkDetector process.ForkDetector,
	epochStartTrigger epochStart.TriggerHandler,
	bootStorer process.BootStorer,
	validatorStatisticsProcessor process.ValidatorStatisticsProcessor,
	headerValidator process.HeaderConstructionValidator,
	blockTracker process.BlockTracker,
	pendingMiniBlocksHandler process.PendingMiniBlocksHandler,
) (process.BlockProcessor, error) {

	shardCoordinator := processArgs.shardCoordinator

	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		return newShardBlockProcessor(
			&processArgs.coreComponents.Config,
			requestHandler,
			processArgs.shardCoordinator,
			processArgs.nodesCoordinator,
			processArgs.data,
			processArgs.coreData,
			processArgs.state,
			forkDetector,
			processArgs.economicsData,
			processArgs.rounder,
			epochStartTrigger,
			bootStorer,
			processArgs.gasSchedule,
			processArgs.stateCheckpointModulus,
			headerValidator,
			blockTracker,
			processArgs.minSizeInBytes,
			processArgs.maxSizeInBytes,
			processArgs.txLogsProcessor,
			processArgs.smartContractParser,
			processArgs.indexer,
			processArgs.tpsBenchmark,
			processArgs.version,
		)
	}
	if shardCoordinator.SelfId() == core.MetachainShardId {
		return newMetaBlockProcessor(
			requestHandler,
			processArgs.shardCoordinator,
			processArgs.nodesCoordinator,
			processArgs.data,
			processArgs.coreData,
			processArgs.state,
			forkDetector,
			processArgs.economicsData,
			validatorStatisticsProcessor,
			processArgs.rounder,
			epochStartTrigger,
			bootStorer,
			headerValidator,
			blockTracker,
			pendingMiniBlocksHandler,
			processArgs.stateCheckpointModulus,
			processArgs.crypto.MessageSignVerifier,
			processArgs.gasSchedule,
			processArgs.minSizeInBytes,
			processArgs.maxSizeInBytes,
			processArgs.ratingsData,
			processArgs.nodesConfig,
			processArgs.txLogsProcessor,
			processArgs.systemSCConfig,
			processArgs.indexer,
			processArgs.tpsBenchmark,
			processArgs.version,
		)
	}

	return nil, errors.New("could not create block processor")
}

func newShardBlockProcessor(
	config *config.Config,
	requestHandler process.RequestHandler,
	shardCoordinator sharding.Coordinator,
	nodesCoordinator sharding.NodesCoordinator,
	data *mainFactory.DataComponents,
	core *mainFactory.CoreComponents,
	stateComponents *mainFactory.StateComponents,
	forkDetector process.ForkDetector,
	economics *economics.EconomicsData,
	rounder consensus.Rounder,
	epochStartTrigger epochStart.TriggerHandler,
	bootStorer process.BootStorer,
	gasSchedule map[string]map[string]uint64,
	stateCheckpointModulus uint,
	headerValidator process.HeaderConstructionValidator,
	blockTracker process.BlockTracker,
	minSizeInBytes uint32,
	maxSizeInBytes uint32,
	txLogsProcessor process.TransactionLogProcessor,
	smartContractParser genesis.InitialSmartContractParser,
	indexer indexer.Indexer,
	tpsBenchmark statistics.TPSBenchmark,
	version string,
) (process.BlockProcessor, error) {
	argsParser := smartContract.NewArgumentParser()

	mapDNSAddresses, err := smartContractParser.GetDeployedSCAddresses(genesis.DNSType)
	if err != nil {
		return nil, err
	}

	argsBuiltIn := builtInFunctions.ArgsCreateBuiltInFunctionContainer{
		GasMap:          gasSchedule,
		MapDNSAddresses: mapDNSAddresses,
		Marshalizer:     core.InternalMarshalizer,
	}
	builtInFuncs, err := builtInFunctions.CreateBuiltInFunctionContainer(argsBuiltIn)
	if err != nil {
		return nil, err
	}

	argsHook := hooks.ArgBlockChainHook{
		Accounts:         stateComponents.AccountsAdapter,
		PubkeyConv:       stateComponents.AddressPubkeyConverter,
		StorageService:   data.Store,
		BlockChain:       data.Blkc,
		ShardCoordinator: shardCoordinator,
		Marshalizer:      core.InternalMarshalizer,
		Uint64Converter:  core.Uint64ByteSliceConverter,
		BuiltInFunctions: builtInFuncs,
	}
	vmFactory, err := shard.NewVMContainerFactory(
		config.VirtualMachineConfig,
		economics.MaxGasLimitPerBlock(shardCoordinator.SelfId()),
		gasSchedule,
		argsHook)
	if err != nil {
		return nil, err
	}

	vmContainer, err := vmFactory.Create()
	if err != nil {
		return nil, err
	}

	interimProcFactory, err := shard.NewIntermediateProcessorsContainerFactory(
		shardCoordinator,
		core.InternalMarshalizer,
		core.Hasher,
		stateComponents.AddressPubkeyConverter,
		data.Store,
		data.Datapool,
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

	receiptTxInterim, err := interimProcContainer.Get(dataBlock.ReceiptBlock)
	if err != nil {
		return nil, err
	}

	badTxInterim, err := interimProcContainer.Get(dataBlock.InvalidBlock)
	if err != nil {
		return nil, err
	}

	argsTxTypeHandler := coordinator.ArgNewTxTypeHandler{
		PubkeyConverter:  stateComponents.AddressPubkeyConverter,
		ShardCoordinator: shardCoordinator,
		BuiltInFuncNames: builtInFuncs.Keys(),
		ArgumentParser:   parsers.NewCallArgsParser(),
	}
	txTypeHandler, err := coordinator.NewTxTypeHandler(argsTxTypeHandler)
	if err != nil {
		return nil, err
	}

	gasHandler, err := preprocess.NewGasComputation(economics, txTypeHandler)
	if err != nil {
		return nil, err
	}

	txFeeHandler, err := postprocess.NewFeeAccumulator()
	if err != nil {
		return nil, err
	}

	argsNewScProcessor := smartContract.ArgsNewSmartContractProcessor{
		VmContainer:      vmContainer,
		ArgsParser:       argsParser,
		Hasher:           core.Hasher,
		Marshalizer:      core.InternalMarshalizer,
		AccountsDB:       stateComponents.AccountsAdapter,
		TempAccounts:     vmFactory.BlockChainHookImpl(),
		PubkeyConv:       stateComponents.AddressPubkeyConverter,
		Coordinator:      shardCoordinator,
		ScrForwarder:     scForwarder,
		TxFeeHandler:     txFeeHandler,
		EconomicsFee:     economics,
		GasHandler:       gasHandler,
		BuiltInFunctions: vmFactory.BlockChainHookImpl().GetBuiltInFunctions(),
		TxLogsProcessor:  txLogsProcessor,
		TxTypeHandler:    txTypeHandler,
		DisableDeploy:    config.GeneralSettings.DisableDeploy,
		DisableBuiltIn:   config.GeneralSettings.DisableBuiltInFunctions,
		BadTxForwarder:   badTxInterim,
	}
	scProcessor, err := smartContract.NewSmartContractProcessor(argsNewScProcessor)
	if err != nil {
		return nil, err
	}

	rewardsTxProcessor, err := rewardTransaction.NewRewardTxProcessor(
		stateComponents.AccountsAdapter,
		stateComponents.AddressPubkeyConverter,
		shardCoordinator,
	)
	if err != nil {
		return nil, err
	}

	transactionProcessor, err := transaction.NewTxProcessor(
		stateComponents.AccountsAdapter,
		core.Hasher,
		stateComponents.AddressPubkeyConverter,
		core.InternalMarshalizer,
		core.TxSignMarshalizer,
		shardCoordinator,
		scProcessor,
		txFeeHandler,
		txTypeHandler,
		economics,
		receiptTxInterim,
		badTxInterim,
		argsParser,
		scForwarder,
	)
	if err != nil {
		return nil, errors.New("could not create transaction statisticsProcessor: " + err.Error())
	}

	blockSizeThrottler, err := throttle.NewBlockSizeThrottle(minSizeInBytes, maxSizeInBytes)
	if err != nil {
		return nil, err
	}

	blockSizeComputationHandler, err := preprocess.NewBlockSizeComputation(core.InternalMarshalizer, blockSizeThrottler, maxSizeInBytes)
	if err != nil {
		return nil, err
	}

	balanceComputationHandler, err := preprocess.NewBalanceComputation()
	if err != nil {
		return nil, err
	}

	preProcFactory, err := shard.NewPreProcessorsContainerFactory(
		shardCoordinator,
		data.Store,
		core.InternalMarshalizer,
		core.Hasher,
		data.Datapool,
		stateComponents.AddressPubkeyConverter,
		stateComponents.AccountsAdapter,
		requestHandler,
		transactionProcessor,
		scProcessor,
		scProcessor,
		rewardsTxProcessor,
		economics,
		gasHandler,
		blockTracker,
		blockSizeComputationHandler,
		balanceComputationHandler,
	)
	if err != nil {
		return nil, err
	}

	preProcContainer, err := preProcFactory.Create()
	if err != nil {
		return nil, err
	}

	txCoordinator, err := coordinator.NewTransactionCoordinator(
		core.Hasher,
		core.InternalMarshalizer,
		shardCoordinator,
		stateComponents.AccountsAdapter,
		data.Datapool.MiniBlocks(),
		requestHandler,
		preProcContainer,
		interimProcContainer,
		gasHandler,
		txFeeHandler,
		blockSizeComputationHandler,
		balanceComputationHandler,
	)
	if err != nil {
		return nil, err
	}

	accountsDb := make(map[state.AccountsDbIdentifier]state.AccountsAdapter)
	accountsDb[state.UserAccountsState] = stateComponents.AccountsAdapter

	argumentsBaseProcessor := block.ArgBaseProcessor{
		Version:                version,
		AccountsDB:             accountsDb,
		ForkDetector:           forkDetector,
		Hasher:                 core.Hasher,
		Marshalizer:            core.InternalMarshalizer,
		Store:                  data.Store,
		ShardCoordinator:       shardCoordinator,
		NodesCoordinator:       nodesCoordinator,
		Uint64Converter:        core.Uint64ByteSliceConverter,
		RequestHandler:         requestHandler,
		BlockChainHook:         vmFactory.BlockChainHookImpl(),
		TxCoordinator:          txCoordinator,
		Rounder:                rounder,
		EpochStartTrigger:      epochStartTrigger,
		HeaderValidator:        headerValidator,
		BootStorer:             bootStorer,
		BlockTracker:           blockTracker,
		DataPool:               data.Datapool,
		FeeHandler:             txFeeHandler,
		BlockChain:             data.Blkc,
		StateCheckpointModulus: stateCheckpointModulus,
		BlockSizeThrottler:     blockSizeThrottler,
		Indexer:                indexer,
		TpsBenchmark:           tpsBenchmark,
	}
	arguments := block.ArgShardProcessor{
		ArgBaseProcessor: argumentsBaseProcessor,
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
	data *mainFactory.DataComponents,
	core *mainFactory.CoreComponents,
	stateComponents *mainFactory.StateComponents,
	forkDetector process.ForkDetector,
	economicsData *economics.EconomicsData,
	validatorStatisticsProcessor process.ValidatorStatisticsProcessor,
	rounder consensus.Rounder,
	epochStartTrigger epochStart.TriggerHandler,
	bootStorer process.BootStorer,
	headerValidator process.HeaderConstructionValidator,
	blockTracker process.BlockTracker,
	pendingMiniBlocksHandler process.PendingMiniBlocksHandler,
	stateCheckpointModulus uint,
	messageSignVerifier vm.MessageSignVerifier,
	gasSchedule map[string]map[string]uint64,
	minSizeInBytes uint32,
	maxSizeInBytes uint32,
	ratingsData process.RatingsInfoHandler,
	nodesSetup sharding.GenesisNodesSetupHandler,
	txLogsProcessor process.TransactionLogProcessor,
	systemSCConfig *config.SystemSmartContractsConfig,
	indexer indexer.Indexer,
	tpsBenchmark statistics.TPSBenchmark,
	version string,
) (process.BlockProcessor, error) {

	builtInFuncs := builtInFunctions.NewBuiltInFunctionContainer()
	argsHook := hooks.ArgBlockChainHook{
		Accounts:         stateComponents.AccountsAdapter,
		PubkeyConv:       stateComponents.AddressPubkeyConverter,
		StorageService:   data.Store,
		BlockChain:       data.Blkc,
		ShardCoordinator: shardCoordinator,
		Marshalizer:      core.InternalMarshalizer,
		Uint64Converter:  core.Uint64ByteSliceConverter,
		BuiltInFunctions: builtInFuncs, // no built-in functions for meta.
	}
	vmFactory, err := metachain.NewVMContainerFactory(
		argsHook,
		economicsData,
		messageSignVerifier,
		gasSchedule,
		nodesSetup,
		core.Hasher,
		core.InternalMarshalizer,
		systemSCConfig,
		stateComponents.PeerAccounts,
	)
	if err != nil {
		return nil, err
	}

	argsParser := smartContract.NewArgumentParser()

	vmContainer, err := vmFactory.Create()
	if err != nil {
		return nil, err
	}

	interimProcFactory, err := metachain.NewIntermediateProcessorsContainerFactory(
		shardCoordinator,
		core.InternalMarshalizer,
		core.Hasher,
		stateComponents.AddressPubkeyConverter,
		data.Store,
		data.Datapool,
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

	badTxForwarder, err := interimProcContainer.Get(dataBlock.InvalidBlock)
	if err != nil {
		return nil, err
	}

	argsTxTypeHandler := coordinator.ArgNewTxTypeHandler{
		PubkeyConverter:  stateComponents.AddressPubkeyConverter,
		ShardCoordinator: shardCoordinator,
		BuiltInFuncNames: builtInFuncs.Keys(),
		ArgumentParser:   parsers.NewCallArgsParser(),
	}
	txTypeHandler, err := coordinator.NewTxTypeHandler(argsTxTypeHandler)
	if err != nil {
		return nil, err
	}

	gasHandler, err := preprocess.NewGasComputation(economicsData, txTypeHandler)
	if err != nil {
		return nil, err
	}

	txFeeHandler, err := postprocess.NewFeeAccumulator()
	if err != nil {
		return nil, err
	}

	argsNewScProcessor := smartContract.ArgsNewSmartContractProcessor{
		VmContainer:      vmContainer,
		ArgsParser:       argsParser,
		Hasher:           core.Hasher,
		Marshalizer:      core.InternalMarshalizer,
		AccountsDB:       stateComponents.AccountsAdapter,
		TempAccounts:     vmFactory.BlockChainHookImpl(),
		PubkeyConv:       stateComponents.AddressPubkeyConverter,
		Coordinator:      shardCoordinator,
		ScrForwarder:     scForwarder,
		TxFeeHandler:     txFeeHandler,
		EconomicsFee:     economicsData,
		TxTypeHandler:    txTypeHandler,
		GasHandler:       gasHandler,
		BuiltInFunctions: vmFactory.BlockChainHookImpl().GetBuiltInFunctions(),
		TxLogsProcessor:  txLogsProcessor,
		BadTxForwarder:   badTxForwarder,
	}
	scProcessor, err := smartContract.NewSmartContractProcessor(argsNewScProcessor)
	if err != nil {
		return nil, err
	}

	transactionProcessor, err := transaction.NewMetaTxProcessor(
		core.Hasher,
		core.InternalMarshalizer,
		stateComponents.AccountsAdapter,
		stateComponents.AddressPubkeyConverter,
		shardCoordinator,
		scProcessor,
		txTypeHandler,
		economicsData,
	)
	if err != nil {
		return nil, errors.New("could not create transaction processor: " + err.Error())
	}

	blockSizeThrottler, err := throttle.NewBlockSizeThrottle(minSizeInBytes, maxSizeInBytes)
	if err != nil {
		return nil, err
	}

	blockSizeComputationHandler, err := preprocess.NewBlockSizeComputation(core.InternalMarshalizer, blockSizeThrottler, maxSizeInBytes)
	if err != nil {
		return nil, err
	}

	balanceComputationHandler, err := preprocess.NewBalanceComputation()
	if err != nil {
		return nil, err
	}

	preProcFactory, err := metachain.NewPreProcessorsContainerFactory(
		shardCoordinator,
		data.Store,
		core.InternalMarshalizer,
		core.Hasher,
		data.Datapool,
		stateComponents.AccountsAdapter,
		requestHandler,
		transactionProcessor,
		scProcessor,
		economicsData,
		gasHandler,
		blockTracker,
		stateComponents.AddressPubkeyConverter,
		blockSizeComputationHandler,
		balanceComputationHandler,
	)
	if err != nil {
		return nil, err
	}

	preProcContainer, err := preProcFactory.Create()
	if err != nil {
		return nil, err
	}

	txCoordinator, err := coordinator.NewTransactionCoordinator(
		core.Hasher,
		core.InternalMarshalizer,
		shardCoordinator,
		stateComponents.AccountsAdapter,
		data.Datapool.MiniBlocks(),
		requestHandler,
		preProcContainer,
		interimProcContainer,
		gasHandler,
		txFeeHandler,
		blockSizeComputationHandler,
		balanceComputationHandler,
	)
	if err != nil {
		return nil, err
	}

	scDataGetter, err := smartContract.NewSCQueryService(vmContainer, economicsData)
	if err != nil {
		return nil, err
	}

	argsStaking := scToProtocol.ArgStakingToPeer{
		PubkeyConv:  stateComponents.ValidatorPubkeyConverter,
		Hasher:      core.Hasher,
		Marshalizer: core.InternalMarshalizer,
		PeerState:   stateComponents.PeerAccounts,
		BaseState:   stateComponents.AccountsAdapter,
		ArgParser:   argsParser,
		CurrTxs:     data.Datapool.CurrentBlockTxs(),
		ScQuery:     scDataGetter,
		RatingsData: ratingsData,
	}
	smartContractToProtocol, err := scToProtocol.NewStakingToPeer(argsStaking)
	if err != nil {
		return nil, err
	}

	genesisHdr := data.Blkc.GetGenesisHeader()
	argsEpochStartData := metachainEpochStart.ArgsNewEpochStartData{
		Marshalizer:       core.InternalMarshalizer,
		Hasher:            core.Hasher,
		Store:             data.Store,
		DataPool:          data.Datapool,
		BlockTracker:      blockTracker,
		ShardCoordinator:  shardCoordinator,
		EpochStartTrigger: epochStartTrigger,
		RequestHandler:    requestHandler,
		GenesisEpoch:      genesisHdr.GetEpoch(),
	}
	epochStartDataCreator, err := metachainEpochStart.NewEpochStartData(argsEpochStartData)
	if err != nil {
		return nil, err
	}

	argsEpochEconomics := metachainEpochStart.ArgsNewEpochEconomics{
		Marshalizer:        core.InternalMarshalizer,
		Hasher:             core.Hasher,
		Store:              data.Store,
		ShardCoordinator:   shardCoordinator,
		RewardsHandler:     economicsData,
		RoundTime:          rounder,
		GenesisNonce:       genesisHdr.GetNonce(),
		GenesisEpoch:       genesisHdr.GetEpoch(),
		GenesisTotalSupply: economicsData.GenesisTotalSupply(),
	}
	epochEconomics, err := metachainEpochStart.NewEndOfEpochEconomicsDataCreator(argsEpochEconomics)
	if err != nil {
		return nil, err
	}

	rewardsStorage := data.Store.GetStorer(dataRetriever.RewardTransactionUnit)
	miniBlockStorage := data.Store.GetStorer(dataRetriever.MiniBlockUnit)
	argsEpochRewards := metachainEpochStart.ArgsNewRewardsCreator{
		ShardCoordinator:    shardCoordinator,
		PubkeyConverter:     stateComponents.AddressPubkeyConverter,
		RewardsStorage:      rewardsStorage,
		MiniBlockStorage:    miniBlockStorage,
		Hasher:              core.Hasher,
		Marshalizer:         core.InternalMarshalizer,
		DataPool:            data.Datapool,
		CommunityAddress:    economicsData.CommunityAddress(),
		NodesConfigProvider: nodesCoordinator,
	}
	epochRewards, err := metachainEpochStart.NewEpochStartRewardsCreator(argsEpochRewards)
	if err != nil {
		return nil, err
	}

	argsEpochValidatorInfo := metachainEpochStart.ArgsNewValidatorInfoCreator{
		ShardCoordinator: shardCoordinator,
		MiniBlockStorage: miniBlockStorage,
		Hasher:           core.Hasher,
		Marshalizer:      core.InternalMarshalizer,
		DataPool:         data.Datapool,
	}
	validatorInfoCreator, err := metachainEpochStart.NewValidatorInfoCreator(argsEpochValidatorInfo)
	if err != nil {
		return nil, err
	}

	accountsDb := make(map[state.AccountsDbIdentifier]state.AccountsAdapter)
	accountsDb[state.UserAccountsState] = stateComponents.AccountsAdapter
	accountsDb[state.PeerAccountsState] = stateComponents.PeerAccounts

	argumentsBaseProcessor := block.ArgBaseProcessor{
		Version:                version,
		AccountsDB:             accountsDb,
		ForkDetector:           forkDetector,
		Hasher:                 core.Hasher,
		Marshalizer:            core.InternalMarshalizer,
		Store:                  data.Store,
		ShardCoordinator:       shardCoordinator,
		NodesCoordinator:       nodesCoordinator,
		Uint64Converter:        core.Uint64ByteSliceConverter,
		RequestHandler:         requestHandler,
		BlockChainHook:         vmFactory.BlockChainHookImpl(),
		TxCoordinator:          txCoordinator,
		EpochStartTrigger:      epochStartTrigger,
		Rounder:                rounder,
		HeaderValidator:        headerValidator,
		BootStorer:             bootStorer,
		BlockTracker:           blockTracker,
		DataPool:               data.Datapool,
		FeeHandler:             txFeeHandler,
		BlockChain:             data.Blkc,
		StateCheckpointModulus: stateCheckpointModulus,
		BlockSizeThrottler:     blockSizeThrottler,
		Indexer:                indexer,
		TpsBenchmark:           tpsBenchmark,
	}
	arguments := block.ArgMetaProcessor{
		ArgBaseProcessor:             argumentsBaseProcessor,
		SCDataGetter:                 scDataGetter,
		SCToProtocol:                 smartContractToProtocol,
		PendingMiniBlocksHandler:     pendingMiniBlocksHandler,
		EpochStartDataCreator:        epochStartDataCreator,
		EpochEconomics:               epochEconomics,
		EpochRewardsCreator:          epochRewards,
		EpochValidatorInfoCreator:    validatorInfoCreator,
		ValidatorStatisticsProcessor: validatorStatisticsProcessor,
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

	storageService := processComponents.data.Store

	var peerDataPool peer.DataPool = processComponents.data.Datapool
	if processComponents.shardCoordinator.SelfId() < processComponents.shardCoordinator.NumberOfShards() {
		peerDataPool = processComponents.data.Datapool
	}

	hardForkConfig := processComponents.mainConfig.Hardfork
	ratingEnabledEpoch := uint32(0)

	if hardForkConfig.AfterHardFork {
		ratingEnabledEpoch = hardForkConfig.StartEpoch + hardForkConfig.ValidatorGracePeriodInEpochs
	}
	arguments := peer.ArgValidatorStatisticsProcessor{
		PeerAdapter:         processComponents.state.PeerAccounts,
		PubkeyConv:          processComponents.state.ValidatorPubkeyConverter,
		NodesCoordinator:    processComponents.nodesCoordinator,
		ShardCoordinator:    processComponents.shardCoordinator,
		DataPool:            peerDataPool,
		StorageService:      storageService,
		Marshalizer:         processComponents.coreData.InternalMarshalizer,
		Rater:               processComponents.rater,
		MaxComputableRounds: processComponents.maxComputableRounds,
		RewardsHandler:      processComponents.economicsData,
		NodesSetup:          processComponents.nodesConfig,
		RatingEnableEpoch:   ratingEnabledEpoch,
		GenesisNonce:        processComponents.data.Blkc.GetGenesisHeader().GetNonce(),
	}

	validatorStatisticsProcessor, err := peer.NewValidatorStatisticsProcessor(arguments)
	if err != nil {
		return nil, err
	}

	return validatorStatisticsProcessor, nil
}

// PrepareOpenTopics will set to the anti flood handler the topics for which
// the node can receive messages from others than validators
func PrepareOpenTopics(
	antiflood mainFactory.P2PAntifloodHandler,
	shardCoordinator sharding.Coordinator,
) {
	selfID := shardCoordinator.SelfId()
	if selfID == core.MetachainShardId {
		antiflood.SetTopicsForAll(core.HeartbeatTopic)
		return
	}

	selfShardTxTopic := factory.TransactionTopic + core.CommunicationIdentifierBetweenShards(selfID, selfID)
	antiflood.SetTopicsForAll(core.HeartbeatTopic, selfShardTxTopic)
}

// PrepareNetworkShardingCollector will create the network sharding collector and apply it to
// the network messenger and antiflood handler
func PrepareNetworkShardingCollector(
	network *mainFactory.NetworkComponents,
	config *config.Config,
	nodesCoordinator sharding.NodesCoordinator,
	coordinator sharding.Coordinator,
	epochStartRegistrationHandler epochStart.RegistrationHandler,
	epochStart uint32,
) (*networksharding.PeerShardMapper, error) {

	networkShardingCollector, err := createNetworkShardingCollector(config, nodesCoordinator, epochStartRegistrationHandler, epochStart)
	if err != nil {
		return nil, err
	}

	localID := network.NetMessenger.ID()
	networkShardingCollector.UpdatePeerIdShardId(localID, coordinator.SelfId())

	err = network.NetMessenger.SetPeerShardResolver(networkShardingCollector)
	if err != nil {
		return nil, err
	}

	err = network.InputAntifloodHandler.SetPeerValidatorMapper(networkShardingCollector)
	if err != nil {
		return nil, err
	}

	return networkShardingCollector, nil
}

func createNetworkShardingCollector(
	config *config.Config,
	nodesCoordinator sharding.NodesCoordinator,
	epochStartRegistrationHandler epochStart.RegistrationHandler,
	epochStart uint32,
) (*networksharding.PeerShardMapper, error) {

	cacheConfig := config.PublicKeyPeerId
	cachePkPid, err := createCache(cacheConfig)
	if err != nil {
		return nil, err
	}

	cacheConfig = config.PublicKeyShardId
	cachePkShardID, err := createCache(cacheConfig)
	if err != nil {
		return nil, err
	}

	cacheConfig = config.PeerIdShardId
	cachePidShardID, err := createCache(cacheConfig)
	if err != nil {
		return nil, err
	}

	psm, err := networksharding.NewPeerShardMapper(
		cachePkPid,
		cachePkShardID,
		cachePidShardID,
		nodesCoordinator,
		epochStart,
	)
	if err != nil {
		return nil, err
	}

	epochStartRegistrationHandler.RegisterHandler(psm)

	return psm, nil
}

func createCache(cacheConfig config.CacheConfig) (storage.Cacher, error) {
	return storageUnit.NewCache(storageFactory.GetCacherFromConfig(cacheConfig))
}

// CreateLatestStorageDataProvider will create a latest storage data provider handler
func CreateLatestStorageDataProvider(
	bootstrapDataProvider storageFactory.BootstrapDataProviderHandler,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	generalConfig config.Config,
	chainID string,
	workingDir string,
	defaultDBPath string,
	defaultEpochString string,
	defaultShardString string,
) (storage.LatestStorageDataProviderHandler, error) {
	directoryReader := storageFactory.NewDirectoryReader()

	latestStorageDataArgs := storageFactory.ArgsLatestDataProvider{
		GeneralConfig:         generalConfig,
		Marshalizer:           marshalizer,
		Hasher:                hasher,
		BootstrapDataProvider: bootstrapDataProvider,
		DirectoryReader:       directoryReader,
		WorkingDir:            workingDir,
		ChainID:               chainID,
		DefaultDBPath:         defaultDBPath,
		DefaultEpochString:    defaultEpochString,
		DefaultShardString:    defaultShardString,
	}
	return storageFactory.NewLatestDataProvider(latestStorageDataArgs)
}

// CreateUnitOpener will create a new unit opener handler
func CreateUnitOpener(
	bootstrapDataProvider storageFactory.BootstrapDataProviderHandler,
	latestDataFromStorageProvider storage.LatestStorageDataProviderHandler,
	internalMarshalizer marshal.Marshalizer,
	generalConfig config.Config,
	chainID string,
	workingDir string,
	defaultDBPath string,
	defaultEpochString string,
	defaultShardString string,
) (storage.UnitOpenerHandler, error) {
	argsStorageUnitOpener := storageFactory.ArgsNewOpenStorageUnits{
		GeneralConfig:             generalConfig,
		Marshalizer:               internalMarshalizer,
		BootstrapDataProvider:     bootstrapDataProvider,
		LatestStorageDataProvider: latestDataFromStorageProvider,
		WorkingDir:                workingDir,
		ChainID:                   chainID,
		DefaultDBPath:             defaultDBPath,
		DefaultEpochString:        defaultEpochString,
		DefaultShardString:        defaultShardString,
	}

	return storageFactory.NewStorageUnitOpenHandler(argsStorageUnitOpener)
}
