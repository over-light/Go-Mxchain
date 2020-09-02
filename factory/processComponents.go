package factory

import (
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/dblookupext"
	"github.com/ElrondNetwork/elrond-go/core/indexer"
	"github.com/ElrondNetwork/elrond-go/core/partitioning"
	"github.com/ElrondNetwork/elrond-go/core/statistics"
	"github.com/ElrondNetwork/elrond-go/data"
	dataBlock "github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/endProcess"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/factory/containers"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/factory/resolverscontainer"
	storageResolversContainers "github.com/ElrondNetwork/elrond-go/dataRetriever/factory/storageResolversContainer"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/requestHandlers"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/epochStart/metachain"
	"github.com/ElrondNetwork/elrond-go/epochStart/notifier"
	"github.com/ElrondNetwork/elrond-go/epochStart/shardchain"
	errErd "github.com/ElrondNetwork/elrond-go/errors"
	"github.com/ElrondNetwork/elrond-go/genesis"
	"github.com/ElrondNetwork/elrond-go/genesis/checking"
	processGenesis "github.com/ElrondNetwork/elrond-go/genesis/process"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/process/block/pendingMb"
	"github.com/ElrondNetwork/elrond-go/process/block/poolsCleaner"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/ElrondNetwork/elrond-go/process/factory/interceptorscontainer"
	"github.com/ElrondNetwork/elrond-go/process/headerCheck"
	"github.com/ElrondNetwork/elrond-go/process/peer"
	"github.com/ElrondNetwork/elrond-go/process/smartContract"
	"github.com/ElrondNetwork/elrond-go/process/sync"
	"github.com/ElrondNetwork/elrond-go/process/track"
	"github.com/ElrondNetwork/elrond-go/process/transactionLog"
	"github.com/ElrondNetwork/elrond-go/process/txsimulator"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/sharding/networksharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	storageFactory "github.com/ElrondNetwork/elrond-go/storage/factory"
	"github.com/ElrondNetwork/elrond-go/storage/pathmanager"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/ElrondNetwork/elrond-go/storage/timecache"
	"github.com/ElrondNetwork/elrond-go/update"
)

// TODO: check underlying components if there are goroutines with infinite for loops

var log = logger.GetOrCreate("factory")

// timeSpanForBadHeaders is the expiry time for an added block header hash
var timeSpanForBadHeaders = time.Minute * 2

// processComponents struct holds the process components
type processComponents struct {
	nodesCoordinator            sharding.NodesCoordinator
	shardCoordinator            sharding.Coordinator
	interceptorsContainer       process.InterceptorsContainer
	resolversFinder             dataRetriever.ResolversFinder
	rounder                     consensus.Rounder
	epochStartTrigger           epochStart.TriggerHandler
	epochStartNotifier          EpochStartNotifier
	forkDetector                process.ForkDetector
	blockProcessor              process.BlockProcessor
	blackListHandler            process.TimeCacher
	bootStorer                  process.BootStorer
	headerSigVerifier           process.InterceptedHeaderSigVerifier
	headerIntegrityVerifier     HeaderIntegrityVerifierHandler
	validatorsStatistics        process.ValidatorStatisticsProcessor
	validatorsProvider          process.ValidatorsProvider
	blockTracker                process.BlockTracker
	pendingMiniBlocksHandler    process.PendingMiniBlocksHandler
	requestHandler              process.RequestHandler
	txLogsProcessor             process.TransactionLogProcessorDatabase
	headerConstructionValidator process.HeaderConstructionValidator
	// TODO: maybe move PeerShardMapper to network components
	peerShardMapper      process.NetworkShardingCollector
	txSimulatorProcessor TransactionSimulatorProcessor
}

// ProcessComponentsFactoryArgs holds the arguments needed to create a process components factory
type ProcessComponentsFactoryArgs struct {
	Config                    config.Config
	AccountsParser            genesis.AccountsParser
	SmartContractParser       genesis.InitialSmartContractParser
	EconomicsData             *economics.EconomicsData
	NodesConfig               NodesSetupHandler
	GasSchedule               map[string]map[string]uint64
	Rounder                   consensus.Rounder
	ShardCoordinator          sharding.Coordinator
	NodesCoordinator          sharding.NodesCoordinator
	Data                      DataComponentsHolder
	CoreData                  CoreComponentsHolder
	Crypto                    CryptoComponentsHolder
	State                     StateComponentsHolder
	Network                   NetworkComponentsHolder
	RequestedItemsHandler     dataRetriever.RequestedItemsHandler
	WhiteListHandler          process.WhiteListHandler
	WhiteListerVerifiedTxs    process.WhiteListHandler
	EpochStartNotifier        EpochStartNotifier
	EpochStart                *config.EpochStartConfig
	Rater                     sharding.PeerAccountListAndRatingHandler
	RatingsData               process.RatingsInfoHandler
	StartEpochNum             uint32
	SizeCheckDelta            uint32
	StateCheckpointModulus    uint
	MaxComputableRounds       uint64
	NumConcurrentResolverJobs int32
	MinSizeInBytes            uint32
	MaxSizeInBytes            uint32
	MaxRating                 uint32
	ValidatorPubkeyConverter  core.PubkeyConverter
	SystemSCConfig            *config.SystemSmartContractsConfig
	Version                   string
	ImportStartHandler        update.ImportStartHandler
	WorkingDir                string
	Indexer                   indexer.Indexer
	TpsBenchmark              statistics.TPSBenchmark
	HistoryRepo               dblookupext.HistoryRepository
	EpochNotifier             process.EpochNotifier
	HeaderIntegrityVerifier   HeaderIntegrityVerifierHandler
	StorageResolverImportPath string
	ChanGracefullyClose       chan endProcess.ArgEndProcess
}

type processComponentsFactory struct {
	config                    config.Config
	accountsParser            genesis.AccountsParser
	smartContractParser       genesis.InitialSmartContractParser
	economicsData             *economics.EconomicsData
	nodesConfig               NodesSetupHandler
	gasSchedule               map[string]map[string]uint64
	rounder                   consensus.Rounder
	shardCoordinator          sharding.Coordinator
	nodesCoordinator          sharding.NodesCoordinator
	data                      DataComponentsHolder
	coreData                  CoreComponentsHolder
	crypto                    CryptoComponentsHolder
	state                     StateComponentsHolder
	network                   NetworkComponentsHolder
	requestedItemsHandler     dataRetriever.RequestedItemsHandler
	whiteListHandler          process.WhiteListHandler
	whiteListerVerifiedTxs    process.WhiteListHandler
	epochStartNotifier        EpochStartNotifier
	startEpochNum             uint32
	rater                     sharding.PeerAccountListAndRatingHandler
	sizeCheckDelta            uint32
	stateCheckpointModulus    uint
	maxComputableRounds       uint64
	numConcurrentResolverJobs int32
	minSizeInBytes            uint32
	maxSizeInBytes            uint32
	maxRating                 uint32
	validatorPubkeyConverter  core.PubkeyConverter
	ratingsData               process.RatingsInfoHandler
	systemSCConfig            *config.SystemSmartContractsConfig
	txLogsProcessor           process.TransactionLogProcessor
	version                   string
	importStartHandler        update.ImportStartHandler
	workingDir                string
	indexer                   indexer.Indexer
	tpsBenchmark              statistics.TPSBenchmark
	historyRepo               dblookupext.HistoryRepository
	epochNotifier             process.EpochNotifier
	headerIntegrityVerifier   HeaderIntegrityVerifierHandler
	storageResolverImportPath string
	chanGracefullyClose       chan endProcess.ArgEndProcess
}

// NewProcessComponentsFactory will return a new instance of processComponentsFactory
func NewProcessComponentsFactory(args ProcessComponentsFactoryArgs) (*processComponentsFactory, error) {
	err := checkArgs(args)
	if err != nil {
		return nil, err
	}

	return &processComponentsFactory{
		config:                    args.Config,
		accountsParser:            args.AccountsParser,
		smartContractParser:       args.SmartContractParser,
		economicsData:             args.EconomicsData,
		nodesConfig:               args.NodesConfig,
		gasSchedule:               args.GasSchedule,
		rounder:                   args.Rounder,
		shardCoordinator:          args.ShardCoordinator,
		nodesCoordinator:          args.NodesCoordinator,
		data:                      args.Data,
		coreData:                  args.CoreData,
		crypto:                    args.Crypto,
		state:                     args.State,
		network:                   args.Network,
		requestedItemsHandler:     args.RequestedItemsHandler,
		whiteListHandler:          args.WhiteListHandler,
		whiteListerVerifiedTxs:    args.WhiteListerVerifiedTxs,
		epochStartNotifier:        args.EpochStartNotifier,
		rater:                     args.Rater,
		ratingsData:               args.RatingsData,
		sizeCheckDelta:            args.SizeCheckDelta,
		stateCheckpointModulus:    args.StateCheckpointModulus,
		startEpochNum:             args.StartEpochNum,
		maxComputableRounds:       args.MaxComputableRounds,
		numConcurrentResolverJobs: args.NumConcurrentResolverJobs,
		minSizeInBytes:            args.MinSizeInBytes,
		maxSizeInBytes:            args.MaxSizeInBytes,
		maxRating:                 args.MaxRating,
		validatorPubkeyConverter:  args.ValidatorPubkeyConverter,
		systemSCConfig:            args.SystemSCConfig,
		version:                   args.Version,
		importStartHandler:        args.ImportStartHandler,
		workingDir:                args.WorkingDir,
		indexer:                   args.Indexer,
		tpsBenchmark:              args.TpsBenchmark,
		historyRepo:               args.HistoryRepo,
		headerIntegrityVerifier:   args.HeaderIntegrityVerifier,
		epochNotifier:             args.EpochNotifier,
		storageResolverImportPath: args.StorageResolverImportPath,
		chanGracefullyClose:       args.ChanGracefullyClose,
	}, nil
}

// Create will create and return a struct containing process components
func (pcf *processComponentsFactory) Create() (*processComponents, error) {
	argsHeaderSig := &headerCheck.ArgsHeaderSigVerifier{
		Marshalizer:       pcf.coreData.InternalMarshalizer(),
		Hasher:            pcf.coreData.Hasher(),
		NodesCoordinator:  pcf.nodesCoordinator,
		MultiSigVerifier:  pcf.crypto.MultiSigner(),
		SingleSigVerifier: pcf.crypto.BlockSigner(),
		KeyGen:            pcf.crypto.BlockSignKeyGen(),
	}
	headerSigVerifier, err := headerCheck.NewHeaderSigVerifier(argsHeaderSig)
	if err != nil {
		return nil, err
	}

	resolversContainerFactory, err := pcf.newResolverContainerFactory()
	if err != nil {
		return nil, err
	}

	resolversContainer, err := resolversContainerFactory.Create()
	if err != nil {
		return nil, err
	}

	resolversFinder, err := containers.NewResolversFinder(resolversContainer, pcf.shardCoordinator)
	if err != nil {
		return nil, err
	}

	requestHandler, err := requestHandlers.NewResolverRequestHandler(
		resolversFinder,
		pcf.requestedItemsHandler,
		pcf.whiteListHandler,
		core.MaxTxsToRequest,
		pcf.shardCoordinator.SelfId(),
		time.Second,
	)
	if err != nil {
		return nil, err
	}

	txLogsStorage := pcf.data.StorageService().GetStorer(dataRetriever.TxLogsUnit)
	txLogsProcessor, err := transactionLog.NewTxLogProcessor(transactionLog.ArgTxLogProcessor{
		Storer:      txLogsStorage,
		Marshalizer: pcf.coreData.InternalMarshalizer(),
	})
	if err != nil {
		return nil, err
	}

	pcf.txLogsProcessor = txLogsProcessor
	genesisBlocks, err := pcf.generateGenesisHeadersAndApplyInitialBalances()
	if err != nil {
		return nil, err
	}

	if pcf.startEpochNum == 0 {
		err = pcf.indexGenesisBlocks(genesisBlocks)
		if err != nil {
			return nil, err
		}
	}

	err = pcf.setGenesisHeader(genesisBlocks)
	if err != nil {
		return nil, err
	}

	validatorStatisticsProcessor, err := pcf.newValidatorStatisticsProcessor()
	if err != nil {
		return nil, err
	}

	cacheRefreshDuration := time.Duration(pcf.config.ValidatorStatistics.CacheRefreshIntervalInSec) * time.Second
	argVSP := peer.ArgValidatorsProvider{
		NodesCoordinator:                  pcf.nodesCoordinator,
		StartEpoch:                        pcf.startEpochNum,
		EpochStartEventNotifier:           pcf.epochStartNotifier,
		CacheRefreshIntervalDurationInSec: cacheRefreshDuration,
		ValidatorStatistics:               validatorStatisticsProcessor,
		MaxRating:                         pcf.maxRating,
		PubKeyConverter:                   pcf.validatorPubkeyConverter,
	}

	validatorsProvider, err := peer.NewValidatorsProvider(argVSP)
	if err != nil {
		return nil, err
	}

	epochStartTrigger, err := pcf.newEpochStartTrigger(requestHandler)
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
	err = pcf.prepareGenesisBlock(genesisBlocks)
	if err != nil {
		return nil, err
	}

	bootStr := pcf.data.StorageService().GetStorer(dataRetriever.BootstrapUnit)
	bootStorer, err := bootstrapStorage.NewBootstrapStorer(pcf.coreData.InternalMarshalizer(), bootStr)
	if err != nil {
		return nil, err
	}

	argsHeaderValidator := block.ArgsHeaderValidator{
		Hasher:      pcf.coreData.Hasher(),
		Marshalizer: pcf.coreData.InternalMarshalizer(),
	}
	headerValidator, err := block.NewHeaderValidator(argsHeaderValidator)
	if err != nil {
		return nil, err
	}

	blockTracker, err := pcf.newBlockTracker(
		headerValidator,
		requestHandler,
		genesisBlocks,
	)
	if err != nil {
		return nil, err
	}

	mbsPoolsCleaner, err := poolsCleaner.NewMiniBlocksPoolsCleaner(
		pcf.data.Datapool().MiniBlocks(),
		pcf.rounder,
		pcf.shardCoordinator,
	)
	if err != nil {
		return nil, err
	}

	mbsPoolsCleaner.StartCleaning()

	txsPoolsCleaner, err := poolsCleaner.NewTxsPoolsCleaner(
		pcf.coreData.AddressPubKeyConverter(),
		pcf.data.Datapool(),
		pcf.rounder,
		pcf.shardCoordinator,
	)
	if err != nil {
		return nil, err
	}

	txsPoolsCleaner.StartCleaning()

	_, err = track.NewMiniBlockTrack(pcf.data.Datapool(), pcf.shardCoordinator, pcf.whiteListHandler)
	if err != nil {
		return nil, err
	}

	interceptorContainerFactory, blackListHandler, err := pcf.newInterceptorContainerFactory(
		headerSigVerifier,
		pcf.headerIntegrityVerifier,
		blockTracker,
		epochStartTrigger,
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
	pendingMiniBlocksHandler, err = pendingMb.NewNilPendingMiniBlocks()
	if err != nil {
		return nil, err
	}
	if pcf.shardCoordinator.SelfId() == core.MetachainShardId {
		pendingMiniBlocksHandler, err = pendingMb.NewPendingMiniBlocks()
		if err != nil {
			return nil, err
		}
	}

	forkDetector, err := pcf.newForkDetector(blackListHandler, blockTracker)
	if err != nil {
		return nil, err
	}

	txSimulatorProcessorArgs := &txsimulator.ArgsTxSimulator{
		AddressPubKeyConverter: pcf.coreData.AddressPubKeyConverter(),
		ShardCoordinator:       pcf.shardCoordinator,
	}

	blockProcessor, err := pcf.newBlockProcessor(
		requestHandler,
		forkDetector,
		epochStartTrigger,
		bootStorer,
		validatorStatisticsProcessor,
		headerValidator,
		blockTracker,
		pendingMiniBlocksHandler,
		txSimulatorProcessorArgs,
	)
	if err != nil {
		return nil, err
	}

	conversionBase := 10
	genesisNodePrice, ok := big.NewInt(0).SetString(pcf.systemSCConfig.StakingSystemSCConfig.GenesisNodePrice, conversionBase)
	if !ok {
		return nil, errors.New("invalid genesis node price")
	}

	nodesSetupChecker, err := checking.NewNodesSetupChecker(
		pcf.accountsParser,
		genesisNodePrice,
		pcf.validatorPubkeyConverter,
		pcf.crypto.BlockSignKeyGen(),
	)
	if err != nil {
		return nil, err
	}

	err = nodesSetupChecker.Check(pcf.nodesConfig.AllInitialNodes())
	if err != nil {
		return nil, err
	}

	peerShardMapper, err := pcf.prepareNetworkShardingCollector()
	if err != nil {
		return nil, err
	}

	txSimulator, err := txsimulator.NewTransactionSimulator(*txSimulatorProcessorArgs)

	return &processComponents{
		nodesCoordinator:            pcf.nodesCoordinator,
		shardCoordinator:            pcf.shardCoordinator,
		interceptorsContainer:       interceptorsContainer,
		resolversFinder:             resolversFinder,
		rounder:                     pcf.rounder,
		forkDetector:                forkDetector,
		blockProcessor:              blockProcessor,
		epochStartTrigger:           epochStartTrigger,
		epochStartNotifier:          pcf.epochStartNotifier,
		blackListHandler:            blackListHandler,
		bootStorer:                  bootStorer,
		headerSigVerifier:           headerSigVerifier,
		validatorsStatistics:        validatorStatisticsProcessor,
		validatorsProvider:          validatorsProvider,
		blockTracker:                blockTracker,
		pendingMiniBlocksHandler:    pendingMiniBlocksHandler,
		requestHandler:              requestHandler,
		txLogsProcessor:             txLogsProcessor,
		headerConstructionValidator: headerValidator,
		headerIntegrityVerifier:     pcf.headerIntegrityVerifier,
		peerShardMapper:             peerShardMapper,
		txSimulatorProcessor:        txSimulator,
	}, nil
}

func (pcf *processComponentsFactory) newValidatorStatisticsProcessor() (process.ValidatorStatisticsProcessor, error) {

	storageService := pcf.data.StorageService()

	var peerDataPool peer.DataPool = pcf.data.Datapool()
	if pcf.shardCoordinator.SelfId() < pcf.shardCoordinator.NumberOfShards() {
		peerDataPool = pcf.data.Datapool()
	}

	hardForkConfig := pcf.config.Hardfork
	ratingEnabledEpoch := uint32(0)
	if hardForkConfig.AfterHardFork {
		ratingEnabledEpoch = hardForkConfig.StartEpoch + hardForkConfig.ValidatorGracePeriodInEpochs
	}
	arguments := peer.ArgValidatorStatisticsProcessor{
		PeerAdapter:         pcf.state.PeerAccounts(),
		PubkeyConv:          pcf.coreData.ValidatorPubKeyConverter(),
		NodesCoordinator:    pcf.nodesCoordinator,
		ShardCoordinator:    pcf.shardCoordinator,
		DataPool:            peerDataPool,
		StorageService:      storageService,
		Marshalizer:         pcf.coreData.InternalMarshalizer(),
		Rater:               pcf.rater,
		MaxComputableRounds: pcf.maxComputableRounds,
		RewardsHandler:      pcf.economicsData,
		NodesSetup:          pcf.nodesConfig,
		RatingEnableEpoch:   ratingEnabledEpoch,
		GenesisNonce:        pcf.data.Blockchain().GetGenesisHeader().GetNonce(),
	}

	validatorStatisticsProcessor, err := peer.NewValidatorStatisticsProcessor(arguments)
	if err != nil {
		return nil, err
	}

	return validatorStatisticsProcessor, nil
}

func (pcf *processComponentsFactory) newEpochStartTrigger(requestHandler process.RequestHandler) (epochStart.TriggerHandler, error) {
	if pcf.shardCoordinator.SelfId() < pcf.shardCoordinator.NumberOfShards() {
		argsHeaderValidator := block.ArgsHeaderValidator{
			Hasher:      pcf.coreData.Hasher(),
			Marshalizer: pcf.coreData.InternalMarshalizer(),
		}
		headerValidator, err := block.NewHeaderValidator(argsHeaderValidator)
		if err != nil {
			return nil, err
		}

		argsPeerMiniBlockSyncer := shardchain.ArgPeerMiniBlockSyncer{
			MiniBlocksPool: pcf.data.Datapool().MiniBlocks(),
			Requesthandler: requestHandler,
		}

		peerMiniBlockSyncer, err := shardchain.NewPeerMiniBlockSyncer(argsPeerMiniBlockSyncer)
		if err != nil {
			return nil, err
		}

		argEpochStart := &shardchain.ArgsShardEpochStartTrigger{
			Marshalizer:          pcf.coreData.InternalMarshalizer(),
			Hasher:               pcf.coreData.Hasher(),
			HeaderValidator:      headerValidator,
			Uint64Converter:      pcf.coreData.Uint64ByteSliceConverter(),
			DataPool:             pcf.data.Datapool(),
			Storage:              pcf.data.StorageService(),
			RequestHandler:       requestHandler,
			Epoch:                pcf.startEpochNum,
			EpochStartNotifier:   pcf.epochStartNotifier,
			Validity:             process.MetaBlockValidity,
			Finality:             process.BlockFinality,
			PeerMiniBlocksSyncer: peerMiniBlockSyncer,
			Rounder:              pcf.rounder,
		}
		epochStartTrigger, err := shardchain.NewEpochStartTrigger(argEpochStart)
		if err != nil {
			return nil, errors.New("error creating new start of epoch trigger" + err.Error())
		}
		err = epochStartTrigger.SetAppStatusHandler(pcf.coreData.StatusHandler())
		if err != nil {
			return nil, err
		}

		return epochStartTrigger, nil
	}

	if pcf.shardCoordinator.SelfId() == core.MetachainShardId {
		argEpochStart := &metachain.ArgsNewMetaEpochStartTrigger{
			GenesisTime:        time.Unix(pcf.nodesConfig.GetStartTime(), 0),
			Settings:           &pcf.config.EpochStartConfig,
			Epoch:              pcf.startEpochNum,
			EpochStartRound:    pcf.data.Blockchain().GetGenesisHeader().GetRound(),
			EpochStartNotifier: pcf.epochStartNotifier,
			Storage:            pcf.data.StorageService(),
			Marshalizer:        pcf.coreData.InternalMarshalizer(),
			Hasher:             pcf.coreData.Hasher(),
		}
		epochStartTrigger, err := metachain.NewEpochStartTrigger(argEpochStart)
		if err != nil {
			return nil, errors.New("error creating new start of epoch trigger" + err.Error())
		}
		err = epochStartTrigger.SetAppStatusHandler(pcf.coreData.StatusHandler())
		if err != nil {
			return nil, err
		}

		return epochStartTrigger, nil
	}

	return nil, errors.New("error creating new start of epoch trigger because of invalid shard id")
}

func (pcf *processComponentsFactory) generateGenesisHeadersAndApplyInitialBalances() (map[uint32]data.HeaderHandler, error) {
	genesisVmConfig := pcf.config.VirtualMachineConfig
	genesisVmConfig.OutOfProcessConfig.MaxLoopTime = 5000 // 5 seconds
	conversionBase := 10
	genesisNodePrice, ok := big.NewInt(0).SetString(pcf.systemSCConfig.StakingSystemSCConfig.GenesisNodePrice, conversionBase)
	if !ok {
		return nil, errors.New("invalid genesis node price")
	}

	arg := processGenesis.ArgsGenesisBlockCreator{
		Core:                 pcf.coreData,
		Data:                 pcf.data,
		GenesisTime:          uint64(pcf.nodesConfig.GetStartTime()),
		StartEpochNum:        pcf.startEpochNum,
		Accounts:             pcf.state.AccountsAdapter(),
		InitialNodesSetup:    pcf.nodesConfig,
		Economics:            pcf.economicsData,
		ShardCoordinator:     pcf.shardCoordinator,
		AccountsParser:       pcf.accountsParser,
		SmartContractParser:  pcf.smartContractParser,
		ValidatorAccounts:    pcf.state.PeerAccounts(),
		GasMap:               pcf.gasSchedule,
		VirtualMachineConfig: genesisVmConfig,
		TxLogsProcessor:      pcf.txLogsProcessor,
		HardForkConfig:       pcf.config.Hardfork,
		TrieStorageManagers:  pcf.state.TrieStorageManagers(),
		SystemSCConfig:       *pcf.systemSCConfig,
		ImportStartHandler:   pcf.importStartHandler,
		WorkingDir:           pcf.workingDir,
		BlockSignKeyGen:      pcf.crypto.BlockSignKeyGen(),
		GenesisString:        pcf.config.GeneralSettings.GenesisString,
		GenesisNodePrice:     genesisNodePrice,
		GeneralConfig:        &pcf.config.GeneralSettings,
	}

	gbc, err := processGenesis.NewGenesisBlockCreator(arg)
	if err != nil {
		return nil, err
	}

	return gbc.CreateGenesisBlocks()
}

func (pcf *processComponentsFactory) setGenesisHeader(genesisBlocks map[uint32]data.HeaderHandler) error {
	genesisBlock, ok := genesisBlocks[pcf.shardCoordinator.SelfId()]
	if !ok {
		return errors.New("genesis block does not exist")
	}

	err := pcf.data.Blockchain().SetGenesisHeader(genesisBlock)
	if err != nil {
		return err
	}

	return nil
}

func (pcf *processComponentsFactory) prepareGenesisBlock(genesisBlocks map[uint32]data.HeaderHandler) error {
	genesisBlock, ok := genesisBlocks[pcf.shardCoordinator.SelfId()]
	if !ok {
		return errors.New("genesis block does not exist")
	}

	genesisBlockHash, err := core.CalculateHash(pcf.coreData.InternalMarshalizer(), pcf.coreData.Hasher(), genesisBlock)
	if err != nil {
		return err
	}

	err = pcf.data.Blockchain().SetGenesisHeader(genesisBlock)
	if err != nil {
		return err
	}

	pcf.data.Blockchain().SetGenesisHeaderHash(genesisBlockHash)

	marshalizedBlock, err := pcf.coreData.InternalMarshalizer().Marshal(genesisBlock)
	if err != nil {
		return err
	}

	nonceToByteSlice := pcf.coreData.Uint64ByteSliceConverter().ToByteSlice(genesisBlock.GetNonce())
	if pcf.shardCoordinator.SelfId() == core.MetachainShardId {
		errNotCritical := pcf.data.StorageService().Put(dataRetriever.MetaBlockUnit, genesisBlockHash, marshalizedBlock)
		if errNotCritical != nil {
			log.Error("error storing genesis metablock", "error", errNotCritical.Error())
		}
		errNotCritical = pcf.data.StorageService().Put(dataRetriever.MetaHdrNonceHashDataUnit, nonceToByteSlice, genesisBlockHash)
		if errNotCritical != nil {
			log.Error("error storing genesis metablock (nonce-hash)", "error", errNotCritical.Error())
		}
	} else {
		errNotCritical := pcf.data.StorageService().Put(dataRetriever.BlockHeaderUnit, genesisBlockHash, marshalizedBlock)
		if errNotCritical != nil {
			log.Error("error storing genesis shardblock", "error", errNotCritical.Error())
		}
		hdrNonceHashDataUnit := dataRetriever.ShardHdrNonceHashDataUnit + dataRetriever.UnitType(genesisBlock.GetShardID())
		errNotCritical = pcf.data.StorageService().Put(hdrNonceHashDataUnit, nonceToByteSlice, genesisBlockHash)
		if errNotCritical != nil {
			log.Error("error storing genesis shard header (nonce-hash)", "error", errNotCritical.Error())
		}
	}

	return nil
}

func (pcf *processComponentsFactory) indexGenesisBlocks(genesisBlocks map[uint32]data.HeaderHandler) error {
	// In Elastic Indexer, only index the metachain block
	genesisBlockHeader := genesisBlocks[core.MetachainShardId]
	genesisBlockHash, err := core.CalculateHash(pcf.coreData.InternalMarshalizer(), pcf.coreData.Hasher(), genesisBlockHeader)
	if err != nil {
		return err
	}

	log.Info("indexGenesisBlocks(): indexer.SaveBlock", "hash", genesisBlockHash)

	pcf.indexer.SaveBlock(&dataBlock.Body{}, genesisBlockHeader, nil, nil, nil)

	// In "dblookupext" index, record both the metachain and the shard blocks
	var shardID uint32
	for shardID, genesisBlockHeader = range genesisBlocks {
		if pcf.shardCoordinator.SelfId() != shardID {
			continue
		}

		genesisBlockHash, err = core.CalculateHash(pcf.coreData.InternalMarshalizer(), pcf.coreData.Hasher(), genesisBlockHeader)
		if err != nil {
			return err
		}

		log.Info("indexGenesisBlocks(): historyRepo.RecordBlock", "shard", shardID, "hash", genesisBlockHash)
		err = pcf.historyRepo.RecordBlock(genesisBlockHash, genesisBlockHeader, &dataBlock.Body{})
		if err != nil {
			return err
		}

		nonceByHashDataUnit := dataRetriever.GetHdrNonceHashDataUnit(shardID)
		nonceAsBytes := pcf.coreData.Uint64ByteSliceConverter().ToByteSlice(genesisBlockHeader.GetNonce())
		err = pcf.data.StorageService().Put(nonceByHashDataUnit, nonceAsBytes, genesisBlockHash)
		if err != nil {
			return err
		}
	}

	return nil
}

func (pcf *processComponentsFactory) newBlockTracker(
	headerValidator process.HeaderConstructionValidator,
	requestHandler process.RequestHandler,
	genesisBlocks map[uint32]data.HeaderHandler,
) (process.BlockTracker, error) {
	argBaseTracker := track.ArgBaseTracker{
		Hasher:           pcf.coreData.Hasher(),
		HeaderValidator:  headerValidator,
		Marshalizer:      pcf.coreData.InternalMarshalizer(),
		RequestHandler:   requestHandler,
		Rounder:          pcf.rounder,
		ShardCoordinator: pcf.shardCoordinator,
		Store:            pcf.data.StorageService(),
		StartHeaders:     genesisBlocks,
		PoolsHolder:      pcf.data.Datapool(),
		WhitelistHandler: pcf.whiteListHandler,
	}

	if pcf.shardCoordinator.SelfId() < pcf.shardCoordinator.NumberOfShards() {
		arguments := track.ArgShardTracker{
			ArgBaseTracker: argBaseTracker,
		}

		return track.NewShardBlockTrack(arguments)
	}

	if pcf.shardCoordinator.SelfId() == core.MetachainShardId {
		arguments := track.ArgMetaTracker{
			ArgBaseTracker: argBaseTracker,
		}

		return track.NewMetaBlockTrack(arguments)
	}

	return nil, errors.New("could not create block tracker")
}

// -- Resolvers container Factory begin
func (pcf *processComponentsFactory) newResolverContainerFactory() (dataRetriever.ResolversContainerFactory, error) {
	if len(pcf.storageResolverImportPath) > 0 {
		log.Debug("starting with storage resolvers", "path", pcf.storageResolverImportPath)
		return pcf.newStorageResolver()
	}
	if pcf.shardCoordinator.SelfId() < pcf.shardCoordinator.NumberOfShards() {
		return pcf.newShardResolverContainerFactory()
	}
	if pcf.shardCoordinator.SelfId() == core.MetachainShardId {
		return pcf.newMetaResolverContainerFactory()
	}

	return nil, errors.New("could not create interceptor and resolver container factory")
}

func (pcf *processComponentsFactory) newShardResolverContainerFactory() (dataRetriever.ResolversContainerFactory, error) {

	dataPacker, err := partitioning.NewSimpleDataPacker(pcf.coreData.InternalMarshalizer())
	if err != nil {
		return nil, err
	}

	resolversContainerFactoryArgs := resolverscontainer.FactoryArgs{
		ShardCoordinator:           pcf.shardCoordinator,
		Messenger:                  pcf.network.NetworkMessenger(),
		Store:                      pcf.data.StorageService(),
		Marshalizer:                pcf.coreData.InternalMarshalizer(),
		DataPools:                  pcf.data.Datapool(),
		Uint64ByteSliceConverter:   pcf.coreData.Uint64ByteSliceConverter(),
		DataPacker:                 dataPacker,
		TriesContainer:             pcf.state.TriesContainer(),
		SizeCheckDelta:             pcf.sizeCheckDelta,
		InputAntifloodHandler:      pcf.network.InputAntiFloodHandler(),
		OutputAntifloodHandler:     pcf.network.OutputAntiFloodHandler(),
		NumConcurrentResolvingJobs: pcf.numConcurrentResolverJobs,
	}
	resolversContainerFactory, err := resolverscontainer.NewShardResolversContainerFactory(resolversContainerFactoryArgs)
	if err != nil {
		return nil, err
	}

	return resolversContainerFactory, nil
}

func (pcf *processComponentsFactory) newMetaResolverContainerFactory() (dataRetriever.ResolversContainerFactory, error) {
	dataPacker, err := partitioning.NewSimpleDataPacker(pcf.coreData.InternalMarshalizer())
	if err != nil {
		return nil, err
	}

	resolversContainerFactoryArgs := resolverscontainer.FactoryArgs{
		ShardCoordinator:           pcf.shardCoordinator,
		Messenger:                  pcf.network.NetworkMessenger(),
		Store:                      pcf.data.StorageService(),
		Marshalizer:                pcf.coreData.InternalMarshalizer(),
		DataPools:                  pcf.data.Datapool(),
		Uint64ByteSliceConverter:   pcf.coreData.Uint64ByteSliceConverter(),
		DataPacker:                 dataPacker,
		TriesContainer:             pcf.state.TriesContainer(),
		SizeCheckDelta:             pcf.sizeCheckDelta,
		InputAntifloodHandler:      pcf.network.InputAntiFloodHandler(),
		OutputAntifloodHandler:     pcf.network.OutputAntiFloodHandler(),
		NumConcurrentResolvingJobs: pcf.numConcurrentResolverJobs,
	}
	resolversContainerFactory, err := resolverscontainer.NewMetaResolversContainerFactory(resolversContainerFactoryArgs)
	if err != nil {
		return nil, err
	}
	return resolversContainerFactory, nil
}

func (pcf *processComponentsFactory) newInterceptorContainerFactory(
	headerSigVerifier process.InterceptedHeaderSigVerifier,
	headerIntegrityVerifier HeaderIntegrityVerifierHandler,
	validityAttester process.ValidityAttester,
	epochStartTrigger process.EpochStartTriggerHandler,
) (process.InterceptorsContainerFactory, process.TimeCacher, error) {
	if pcf.shardCoordinator.SelfId() < pcf.shardCoordinator.NumberOfShards() {
		return pcf.newShardInterceptorContainerFactory(
			headerSigVerifier,
			headerIntegrityVerifier,
			validityAttester,
			epochStartTrigger,
		)
	}
	if pcf.shardCoordinator.SelfId() == core.MetachainShardId {
		return pcf.newMetaInterceptorContainerFactory(
			headerSigVerifier,
			headerIntegrityVerifier,
			validityAttester,
			epochStartTrigger,
		)
	}

	return nil, nil, errors.New("could not create interceptor container factory")
}

func (pcf *processComponentsFactory) newStorageResolver() (dataRetriever.ResolversContainerFactory, error) {
	pathManager, err := createPathManager(pcf.storageResolverImportPath, pcf.coreData.ChainID())
	if err != nil {
		return nil, err
	}

	manualEpochStartNotifier := notifier.NewManualEpochStartNotifier()
	storageServiceCreator, err := storageFactory.NewStorageServiceFactory(
		&pcf.config,
		pcf.shardCoordinator,
		pathManager,
		manualEpochStartNotifier,
		pcf.startEpochNum,
	)
	if err != nil {
		return nil, err
	}

	if pcf.shardCoordinator.SelfId() == core.MetachainShardId {
		store, errStore := storageServiceCreator.CreateForMeta()
		if errStore != nil {
			return nil, errStore
		}

		manualEpochStartNotifier.NewEpoch(pcf.startEpochNum + 1)

		return pcf.createStorageResolversForMeta(
			store,
			manualEpochStartNotifier,
		)
	}

	store, err := storageServiceCreator.CreateForShard()
	if err != nil {
		return nil, err
	}

	manualEpochStartNotifier.NewEpoch(pcf.startEpochNum + 1)

	return pcf.createStorageResolversForShard(
		store,
		manualEpochStartNotifier,
	)
}

func createPathManager(
	storageResolverImportPath string,
	chainID string,
) (storage.PathManagerHandler, error) {
	pathTemplateForPruningStorer := filepath.Join(
		storageResolverImportPath,
		core.DefaultDBPath,
		chainID,
		fmt.Sprintf("%s_%s", core.DefaultEpochString, core.PathEpochPlaceholder),
		fmt.Sprintf("%s_%s", core.DefaultShardString, core.PathShardPlaceholder),
		core.PathIdentifierPlaceholder)

	pathTemplateForStaticStorer := filepath.Join(
		storageResolverImportPath,
		core.DefaultDBPath,
		chainID,
		core.DefaultStaticDbString,
		fmt.Sprintf("%s_%s", core.DefaultShardString, core.PathShardPlaceholder),
		core.PathIdentifierPlaceholder)

	return pathmanager.NewPathManager(pathTemplateForPruningStorer, pathTemplateForStaticStorer)
}

func (pcf *processComponentsFactory) createStorageResolversForMeta(
	store dataRetriever.StorageService,
	manualEpochStartNotifier dataRetriever.ManualEpochStartNotifier,
) (dataRetriever.ResolversContainerFactory, error) {
	dataPacker, err := partitioning.NewSimpleDataPacker(pcf.coreData.InternalMarshalizer())
	if err != nil {
		return nil, err
	}

	resolversContainerFactoryArgs := storageResolversContainers.FactoryArgs{
		ShardCoordinator:         pcf.shardCoordinator,
		Messenger:                pcf.network.NetworkMessenger(),
		Store:                    store,
		Marshalizer:              pcf.coreData.InternalMarshalizer(),
		Uint64ByteSliceConverter: pcf.coreData.Uint64ByteSliceConverter(),
		DataPacker:               dataPacker,
		ManualEpochStartNotifier: manualEpochStartNotifier,
		ChanGracefullyClose:      pcf.chanGracefullyClose,
	}
	resolversContainerFactory, err := storageResolversContainers.NewMetaResolversContainerFactory(resolversContainerFactoryArgs)
	if err != nil {
		return nil, err
	}

	return resolversContainerFactory, nil
}

func (pcf *processComponentsFactory) createStorageResolversForShard(
	store dataRetriever.StorageService,
	manualEpochStartNotifier dataRetriever.ManualEpochStartNotifier,
) (dataRetriever.ResolversContainerFactory, error) {
	dataPacker, err := partitioning.NewSimpleDataPacker(pcf.coreData.InternalMarshalizer())
	if err != nil {
		return nil, err
	}

	resolversContainerFactoryArgs := storageResolversContainers.FactoryArgs{
		ShardCoordinator:         pcf.shardCoordinator,
		Messenger:                pcf.network.NetworkMessenger(),
		Store:                    store,
		Marshalizer:              pcf.coreData.InternalMarshalizer(),
		Uint64ByteSliceConverter: pcf.coreData.Uint64ByteSliceConverter(),
		DataPacker:               dataPacker,
		ManualEpochStartNotifier: manualEpochStartNotifier,
		ChanGracefullyClose:      pcf.chanGracefullyClose,
	}
	resolversContainerFactory, err := storageResolversContainers.NewShardResolversContainerFactory(resolversContainerFactoryArgs)
	if err != nil {
		return nil, err
	}

	return resolversContainerFactory, nil
}

func (pcf *processComponentsFactory) newShardInterceptorContainerFactory(
	headerSigVerifier process.InterceptedHeaderSigVerifier,
	headerIntegrityVerifier HeaderIntegrityVerifierHandler,
	validityAttester process.ValidityAttester,
	epochStartTrigger process.EpochStartTriggerHandler,
) (process.InterceptorsContainerFactory, process.TimeCacher, error) {
	headerBlackList := timecache.NewTimeCache(timeSpanForBadHeaders)
	shardInterceptorsContainerFactoryArgs := interceptorscontainer.ShardInterceptorsContainerFactoryArgs{
		CoreComponents:          pcf.coreData,
		CryptoComponents:        pcf.crypto,
		Accounts:                pcf.state.AccountsAdapter(),
		ShardCoordinator:        pcf.shardCoordinator,
		NodesCoordinator:        pcf.nodesCoordinator,
		Messenger:               pcf.network.NetworkMessenger(),
		Store:                   pcf.data.StorageService(),
		DataPool:                pcf.data.Datapool(),
		MaxTxNonceDeltaAllowed:  core.MaxTxNonceDeltaAllowed,
		TxFeeHandler:            pcf.economicsData,
		BlockBlackList:          headerBlackList,
		HeaderSigVerifier:       headerSigVerifier,
		HeaderIntegrityVerifier: headerIntegrityVerifier,
		SizeCheckDelta:          pcf.sizeCheckDelta,
		ValidityAttester:        validityAttester,
		EpochStartTrigger:       epochStartTrigger,
		WhiteListHandler:        pcf.whiteListHandler,
		WhiteListerVerifiedTxs:  pcf.whiteListerVerifiedTxs,
		AntifloodHandler:        pcf.network.InputAntiFloodHandler(),
		ArgumentsParser:         smartContract.NewArgumentParser(),
	}
	interceptorContainerFactory, err := interceptorscontainer.NewShardInterceptorsContainerFactory(shardInterceptorsContainerFactoryArgs)
	if err != nil {
		return nil, nil, err
	}

	return interceptorContainerFactory, headerBlackList, nil
}

func (pcf *processComponentsFactory) newMetaInterceptorContainerFactory(
	headerSigVerifier process.InterceptedHeaderSigVerifier,
	headerIntegrityVerifier HeaderIntegrityVerifierHandler,
	validityAttester process.ValidityAttester,
	epochStartTrigger process.EpochStartTriggerHandler,
) (process.InterceptorsContainerFactory, process.TimeCacher, error) {
	headerBlackList := timecache.NewTimeCache(timeSpanForBadHeaders)
	metaInterceptorsContainerFactoryArgs := interceptorscontainer.MetaInterceptorsContainerFactoryArgs{
		CoreComponents:          pcf.coreData,
		CryptoComponents:        pcf.crypto,
		ShardCoordinator:        pcf.shardCoordinator,
		NodesCoordinator:        pcf.nodesCoordinator,
		Messenger:               pcf.network.NetworkMessenger(),
		Store:                   pcf.data.StorageService(),
		DataPool:                pcf.data.Datapool(),
		Accounts:                pcf.state.AccountsAdapter(),
		MaxTxNonceDeltaAllowed:  core.MaxTxNonceDeltaAllowed,
		TxFeeHandler:            pcf.economicsData,
		BlackList:               headerBlackList,
		HeaderSigVerifier:       headerSigVerifier,
		HeaderIntegrityVerifier: headerIntegrityVerifier,
		SizeCheckDelta:          pcf.sizeCheckDelta,
		ValidityAttester:        validityAttester,
		EpochStartTrigger:       epochStartTrigger,
		WhiteListHandler:        pcf.whiteListHandler,
		WhiteListerVerifiedTxs:  pcf.whiteListerVerifiedTxs,
		AntifloodHandler:        pcf.network.InputAntiFloodHandler(),
		ArgumentsParser:         smartContract.NewArgumentParser(),
	}
	interceptorContainerFactory, err := interceptorscontainer.NewMetaInterceptorsContainerFactory(metaInterceptorsContainerFactoryArgs)
	if err != nil {
		return nil, nil, err
	}

	return interceptorContainerFactory, headerBlackList, nil
}

func (pcf *processComponentsFactory) newForkDetector(
	headerBlackList process.TimeCacher,
	blockTracker process.BlockTracker,
) (process.ForkDetector, error) {
	if pcf.shardCoordinator.SelfId() < pcf.shardCoordinator.NumberOfShards() {
		return sync.NewShardForkDetector(pcf.rounder, headerBlackList, blockTracker, pcf.nodesConfig.GetStartTime())
	}
	if pcf.shardCoordinator.SelfId() == core.MetachainShardId {
		return sync.NewMetaForkDetector(pcf.rounder, headerBlackList, blockTracker, pcf.nodesConfig.GetStartTime())
	}

	return nil, errors.New("could not create fork detector")
}

// PrepareNetworkShardingCollector will create the network sharding collector and apply it to the network messenger
func (pcf *processComponentsFactory) prepareNetworkShardingCollector() (*networksharding.PeerShardMapper, error) {

	networkShardingCollector, err := createNetworkShardingCollector(
		&pcf.config,
		pcf.nodesCoordinator,
		pcf.epochStartNotifier,
		pcf.startEpochNum,
	)
	if err != nil {
		return nil, err
	}

	localID := pcf.network.NetworkMessenger().ID()
	networkShardingCollector.UpdatePeerIdShardId(localID, pcf.shardCoordinator.SelfId())

	err = pcf.network.NetworkMessenger().SetPeerShardResolver(networkShardingCollector)
	if err != nil {
		return nil, err
	}

	err = pcf.network.InputAntiFloodHandler().SetPeerValidatorMapper(networkShardingCollector)
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

func checkArgs(args ProcessComponentsFactoryArgs) error {
	baseErrMessage := "error creating process components"
	if check.IfNil(args.AccountsParser) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilAccountsParser)
	}
	if check.IfNil(args.SmartContractParser) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilSmartContractParser)
	}
	if args.EconomicsData == nil {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilEconomicsData)
	}
	if args.NodesConfig == nil {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilNodesConfig)
	}
	if args.GasSchedule == nil {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilGasSchedule)
	}
	if check.IfNil(args.Rounder) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilRounder)
	}
	if check.IfNil(args.ShardCoordinator) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilShardCoordinator)
	}
	if check.IfNil(args.NodesCoordinator) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilNodesCoordinator)
	}
	if check.IfNil(args.Data) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilDataComponentsHolder)
	}
	if check.IfNil(args.CoreData) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilCoreComponentsHolder)
	}
	if check.IfNil(args.Crypto) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilCryptoComponentsHolder)
	}
	if check.IfNil(args.State) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilStateComponentsHolder)
	}
	if check.IfNil(args.Network) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilNetworkComponentsHolder)
	}
	if check.IfNil(args.RequestedItemsHandler) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilRequestedItemHandler)
	}
	if check.IfNil(args.WhiteListHandler) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilWhiteListHandler)
	}
	if check.IfNil(args.WhiteListerVerifiedTxs) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilWhiteListVerifiedTxs)
	}
	if check.IfNil(args.EpochStartNotifier) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilEpochStartNotifier)
	}
	if args.EpochStart == nil {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilEpochStartConfig)
	}
	if check.IfNil(args.EpochStartNotifier) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilEpochStartNotifier)
	}
	if check.IfNil(args.Rater) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilRater)
	}
	if check.IfNil(args.RatingsData) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilRatingData)
	}
	if check.IfNil(args.ValidatorPubkeyConverter) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilPubKeyConverter)
	}
	if args.SystemSCConfig == nil {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilSystemSCConfig)
	}
	if check.IfNil(args.EpochNotifier) {
		return fmt.Errorf("%s: %w", baseErrMessage, errErd.ErrNilEpochNotifier)
	}

	return nil
}

// Close closes all underlying components that need closing
func (pc *processComponents) Close() error {
	// TODO: close all components

	return nil
}
