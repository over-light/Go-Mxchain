package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ElrondNetwork/elrond-go/cmd/node/factory"
	"github.com/ElrondNetwork/elrond-go/cmd/node/metrics"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/indexer"
	"github.com/ElrondNetwork/elrond-go/core/serviceContainer"
	"github.com/ElrondNetwork/elrond-go/core/statistics"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/kyber"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/display"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/epochStart/notifier"
	"github.com/ElrondNetwork/elrond-go/facade"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/logger"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/ElrondNetwork/elrond-go/node/external"
	"github.com/ElrondNetwork/elrond-go/ntp"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/ElrondNetwork/elrond-go/process/factory/metachain"
	"github.com/ElrondNetwork/elrond-go/process/factory/shard"
	"github.com/ElrondNetwork/elrond-go/process/smartContract"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/hooks"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/statusHandler"
	factoryViews "github.com/ElrondNetwork/elrond-go/statusHandler/factory"
	"github.com/google/gops/agent"
	"github.com/urfave/cli"
)

const (
	defaultLogPath     = "logs"
	defaultStatsPath   = "stats"
	defaultDBPath      = "db"
	defaultEpochString = "Epoch"
	defaultShardString = "Shard"
	metachainShardName = "metachain"
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
		Value: "./config/genesis.json",
	}
	// nodesFile defines a flag for the path of the initial nodes file.
	nodesFile = cli.StringFlag{
		Name:  "nodesSetup-file",
		Usage: "The node will extract initial nodes info from the nodesSetup.json",
		Value: "./config/nodesSetup.json",
	}
	// txSignSk defines a flag for the path of the single sign private key used when starting the node
	txSignSk = cli.StringFlag{
		Name:  "tx-sign-sk",
		Usage: "Private key that the node will load on startup and will sign transactions - temporary until we have a wallet that can do that",
		Value: "",
	}
	// sk defines a flag for the path of the multi sign private key used when starting the node
	sk = cli.StringFlag{
		Name:  "sk",
		Usage: "Private key that the node will load on startup and will sign blocks",
		Value: "",
	}
	// configurationFile defines a flag for the path to the main toml configuration file
	configurationFile = cli.StringFlag{
		Name:  "config",
		Usage: "The main configuration file to load",
		Value: "./config/config.toml",
	}
	// configurationEconomicsFile defines a flag for the path to the economics toml configuration file
	configurationEconomicsFile = cli.StringFlag{
		Name:  "configEconomics",
		Usage: "The economics configuration file to load",
		Value: "./config/economics.toml",
	}
	// configurationPreferencesFile defines a flag for the path to the preferences toml configuration file
	configurationPreferencesFile = cli.StringFlag{
		Name:  "configPreferences",
		Usage: "The preferences configuration file to load",
		Value: "./config/prefs.toml",
	}
	// p2pConfigurationFile defines a flag for the path to the toml file containing P2P configuration
	p2pConfigurationFile = cli.StringFlag{
		Name:  "p2pconfig",
		Usage: "The configuration file for P2P",
		Value: "./config/p2p.toml",
	}
	// p2pConfigurationFile defines a flag for the path to the toml file containing P2P configuration
	serversConfigurationFile = cli.StringFlag{
		Name:  "serversconfig",
		Usage: "The configuration file for servers confidential data",
		Value: "./config/server.toml",
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
		Value: 0,
	}
	// profileMode defines a flag for profiling the binary
	// If enabled, it will open the pprof routes over the default gin rest webserver.
	// There are several routes that will be available for profiling (profiling can be analyzed with: go tool pprof):
	//  /debug/pprof/ (can be accessed in the browser, will list the available options)
	//  /debug/pprof/goroutine
	//  /debug/pprof/heap
	//  /debug/pprof/threadcreate
	//  /debug/pprof/block
	//  /debug/pprof/mutex
	//  /debug/pprof/profile (CPU profile)
	//  /debug/pprof/trace?seconds=5 (CPU trace) -> being a trace, can be analyzed with: go tool trace
	// Usage: go tool pprof http(s)://ip.of.the.server/debug/pprof/xxxxx
	profileMode = cli.BoolFlag{
		Name:  "profile-mode",
		Usage: "Boolean profiling mode option. If set to true, the /debug/pprof routes will be available on the node for profiling the application.",
	}
	// txSignSkIndex defines a flag that specifies the 0-th based index of the private key to be used from initialBalancesSk.pem file
	txSignSkIndex = cli.IntFlag{
		Name:  "tx-sign-sk-index",
		Usage: "Single sign private key index specifies the 0-th based index of the private key to be used from initialBalancesSk.pem file.",
		Value: 0,
	}
	// skIndex defines a flag that specifies the 0-th based index of the private key to be used from initialNodesSk.pem file
	skIndex = cli.IntFlag{
		Name:  "sk-index",
		Usage: "Private key index specifies the 0-th based index of the private key to be used from initialNodesSk.pem file.",
		Value: 0,
	}
	// gopsEn used to enable diagnosis of running go processes
	gopsEn = cli.BoolFlag{
		Name:  "gops-enable",
		Usage: "Enables gops over the process. Stack can be viewed by calling 'gops stack <pid>'",
	}
	// numOfNodes defines a flag that specifies the maximum number of nodes which will be used from the initialNodes
	numOfNodes = cli.Uint64Flag{
		Name:  "num-of-nodes",
		Usage: "Number of nodes specifies the maximum number of nodes which will be used from initialNodes list exposed in nodesSetup.json file",
		Value: math.MaxUint64,
	}
	// storageCleanup defines a flag for choosing the option of starting the node from scratch. If it is not set (false)
	// it starts from the last state stored on disk
	storageCleanup = cli.BoolFlag{
		Name:  "storage-cleanup",
		Usage: "If set the node will start from scratch, otherwise it starts from the last state stored on disk",
	}

	// restApiInterface defines a flag for the interface on which the rest API will try to bind with
	restApiInterface = cli.StringFlag{
		Name: "rest-api-interface",
		Usage: "The interface address and port to which the REST API will attempt to bind. " +
			"To bind to all available interfaces, set this flag to :8080",
		Value: facade.DefaultRestInterface,
	}

	// restApiDebug defines a flag for starting the rest API engine in debug mode
	restApiDebug = cli.BoolFlag{
		Name:  "rest-api-debug",
		Usage: "Start the rest API engine in debug mode",
	}

	// networkID defines the version of the network. If set, will override the same parameter from config.toml
	networkID = cli.StringFlag{
		Name:  "network-id",
		Usage: "The network version, overriding the one from config.toml",
		Value: "",
	}

	// nodeDisplayName defines the friendly name used by a node in the public monitoring tools. If set, will override
	// the NodeDisplayName from config.toml
	nodeDisplayName = cli.StringFlag{
		Name:  "display-name",
		Usage: "This will represent the friendly name in the public monitoring tools. Will override the config.toml one",
		Value: "",
	}

	// usePrometheus joins the node for prometheus monitoring if set
	usePrometheus = cli.BoolFlag{
		Name:  "use-prometheus",
		Usage: "Will make the node available for prometheus and grafana monitoring",
	}

	//useLogView is used when termui interface is not needed.
	useLogView = cli.BoolFlag{
		Name:  "use-log-view",
		Usage: "will not enable the user-friendly terminal view of the node",
	}

	// initialBalancesSkPemFile defines a flag for the path to the ...
	initialBalancesSkPemFile = cli.StringFlag{
		Name:  "initialBalancesSkPemFile",
		Usage: "The file containing the secret keys which ...",
		Value: "./config/initialBalancesSk.pem",
	}

	// initialNodesSkPemFile defines a flag for the path to the ...
	initialNodesSkPemFile = cli.StringFlag{
		Name:  "initialNodesSkPemFile",
		Usage: "The file containing the secret keys which ...",
		Value: "./config/initialNodesSk.pem",
	}
	// logLevel defines the logger level
	logLevel = cli.StringFlag{
		Name:  "logLevel",
		Usage: "This flag specifies the logger level",
		Value: "*:" + logger.LogInfo.String(),
	}
	// bootstrapRoundIndex defines a flag that specifies the round index from which node should bootstrap from storage
	bootstrapRoundIndex = cli.Uint64Flag{
		Name:  "bootstrap-round-index",
		Usage: "Bootstrap round index specifies the round index from which node should bootstrap from storage",
		Value: math.MaxUint64,
	}
	// enableTxIndexing enables transaction indexing. There can be cases when it's too expensive to index all transactions
	//  so we provide the command line option to disable this behaviour
	enableTxIndexing = cli.BoolTFlag{
		Name:  "tx-indexing",
		Usage: "Enables transaction indexing. There can be cases when it's too expensive to index all transactions so we provide the command line option to disable this behaviour",
	}

	// workingDirectory defines a flag for the path for the working directory.
	workingDirectory = cli.StringFlag{
		Name:  "working-directory",
		Usage: "The node will store here DB, Logs and Stats",
		Value: "",
	}

	// destinationShardAsObserver defines a flag for the prefered shard to be assigned to as an observer.
	destinationShardAsObserver = cli.StringFlag{
		Name:  "destination-shard-as-observer",
		Usage: "The preferred shard as an observer",
		Value: "",
	}

	rm *statistics.ResourceMonitor
)

// dbIndexer will hold the database indexer. Defined globally so it can be initialised only in
//  certain conditions. If those conditions will not be met, it will stay as nil
var dbIndexer indexer.Indexer

// coreServiceContainer is defined globally so it can be injected with appropriate
//  params depending on the type of node we are starting
var coreServiceContainer serviceContainer.Core

// appVersion should be populated at build time using ldflags
// Usage examples:
// linux/mac:
//            go build -i -v -ldflags="-X main.appVersion=$(git describe --tags --long --dirty)"
// windows:
//            for /f %i in ('git describe --tags --long --dirty') do set VERS=%i
//            go build -i -v -ldflags="-X main.appVersion=%VERS%"
var appVersion = core.UnVersionedAppString

// EpochStartSubscriber provides Register and Unregister functionality for the end of epoch events
type EpochStartSubscriber interface {
	RegisterHandler(handler epochStart.EpochStartHandler)
	UnregisterHandler(handler epochStart.EpochStartHandler)
}

func main() {
	_ = display.SetDisplayByteSlice(display.ToHexShort)
	log := logger.GetOrCreate("main")

	app := cli.NewApp()
	cli.AppHelpTemplate = nodeHelpTemplate
	app.Name = "Elrond Node CLI App"
	app.Version = fmt.Sprintf("%s/%s/%s-%s", appVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	app.Usage = "This is the entry point for starting a new Elrond node - the app will start after the genesis timestamp"
	app.Flags = []cli.Flag{
		genesisFile,
		nodesFile,
		port,
		configurationFile,
		configurationEconomicsFile,
		configurationPreferencesFile,
		p2pConfigurationFile,
		txSignSk,
		sk,
		profileMode,
		txSignSkIndex,
		skIndex,
		numOfNodes,
		storageCleanup,
		initialBalancesSkPemFile,
		initialNodesSkPemFile,
		gopsEn,
		serversConfigurationFile,
		networkID,
		nodeDisplayName,
		restApiInterface,
		restApiDebug,
		logLevel,
		usePrometheus,
		useLogView,
		bootstrapRoundIndex,
		enableTxIndexing,
		workingDirectory,
		destinationShardAsObserver,
	}
	app.Authors = []cli.Author{
		{
			Name:  "The Elrond Team",
			Email: "contact@elrond.com",
		},
	}

	//TODO: The next line should be removed when the write in batches is done
	// set the maximum allowed OS threads (not go routines) which can run in the same time (the default is 10000)
	debug.SetMaxThreads(100000)

	app.Action = func(c *cli.Context) error {
		return startNode(c, log, app.Version)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func getSuite(config *config.Config) (crypto.Suite, error) {
	switch config.Consensus.Type {
	case factory.BlsConsensusType:
		return kyber.NewSuitePairingBn256(), nil
	case factory.BnConsensusType:
		return kyber.NewBlakeSHA256Ed25519(), nil
	}

	return nil, errors.New("no consensus provided in config file")
}

func startNode(ctx *cli.Context, log logger.Logger, version string) error {
	logLevel := ctx.GlobalString(logLevel.Name)
	err := logger.SetLogLevel(logLevel)
	if err != nil {
		return err
	}

	enableGopsIfNeeded(ctx, log)

	stop := make(chan bool, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	log.Info("starting node", "version", version, "pid", os.Getpid())

	configurationFileName := ctx.GlobalString(configurationFile.Name)
	generalConfig, err := loadMainConfig(configurationFileName)
	if err != nil {
		return err
	}
	log.Debug("config", "file", configurationFileName)

	configurationEconomicsFileName := ctx.GlobalString(configurationEconomicsFile.Name)
	economicsConfig, err := loadEconomicsConfig(configurationEconomicsFileName)
	if err != nil {
		return err
	}
	log.Debug("config", "file", configurationEconomicsFileName)

	configurationPreferencesFileName := ctx.GlobalString(configurationPreferencesFile.Name)
	preferencesConfig, err := loadPreferencesConfig(configurationPreferencesFileName)
	if err != nil {
		return err
	}
	log.Debug("config", "file", configurationPreferencesFileName)

	p2pConfigurationFileName := ctx.GlobalString(p2pConfigurationFile.Name)
	p2pConfig, err := core.LoadP2PConfig(p2pConfigurationFileName)
	if err != nil {
		return err
	}

	log.Debug("config", "file", p2pConfigurationFileName)
	if ctx.IsSet(port.Name) {
		p2pConfig.Node.Port = ctx.GlobalInt(port.Name)
	}

	genesisConfig, err := sharding.NewGenesisConfig(ctx.GlobalString(genesisFile.Name))
	if err != nil {
		return err
	}
	log.Debug("config", "file", ctx.GlobalString(genesisFile.Name))

	nodesConfig, err := sharding.NewNodesSetup(ctx.GlobalString(nodesFile.Name), ctx.GlobalUint64(numOfNodes.Name))
	if err != nil {
		return err
	}
	log.Debug("config", "file", ctx.GlobalString(nodesFile.Name))

	syncer := ntp.NewSyncTime(generalConfig.NTPConfig, time.Hour, nil)
	go syncer.StartSync()

	log.Debug("NTP average clock offset", "value", syncer.ClockOffset())

	//TODO: The next 5 lines should be deleted when we are done testing from a precalculated (not hard coded) timestamp
	if nodesConfig.StartTime == 0 {
		time.Sleep(1000 * time.Millisecond)
		ntpTime := syncer.CurrentTime()
		nodesConfig.StartTime = (ntpTime.Unix()/60 + 1) * 60
	}

	startTime := time.Unix(nodesConfig.StartTime, 0)

	log.Info("start time",
		"formatted", startTime.Format("Mon Jan 2 15:04:05 MST 2006"),
		"seconds", startTime.Unix())

	suite, err := getSuite(generalConfig)
	if err != nil {
		return err
	}

	initialNodesSkPemFileName := ctx.GlobalString(initialNodesSkPemFile.Name)
	keyGen, privKey, pubKey, err := factory.GetSigningParams(
		ctx,
		sk.Name,
		skIndex.Name,
		initialNodesSkPemFileName,
		suite)
	if err != nil {
		return err
	}
	log.Debug("block sign pubkey", "hex", factory.GetPkEncoded(pubKey))

	if ctx.IsSet(destinationShardAsObserver.Name) {
		generalConfig.GeneralSettings.DestinationShardAsObserver = ctx.GlobalString(destinationShardAsObserver.Name)
	}

	if ctx.IsSet(networkID.Name) {
		generalConfig.GeneralSettings.NetworkID = ctx.GlobalString(networkID.Name)
	}

	if ctx.IsSet(nodeDisplayName.Name) {
		preferencesConfig.Preferences.NodeDisplayName = ctx.GlobalString(nodeDisplayName.Name)
	}

	shardCoordinator, nodeType, err := createShardCoordinator(nodesConfig, pubKey, generalConfig.GeneralSettings, log)
	if err != nil {
		return err
	}

	var workingDir = ""
	if ctx.IsSet(workingDirectory.Name) {
		workingDir = ctx.GlobalString(workingDirectory.Name)
	} else {
		workingDir, err = os.Getwd()
		if err != nil {
			log.LogIfError(err)
			workingDir = ""
		}
	}

	var shardId = "metachain"
	if shardCoordinator.SelfId() != core.MetachainShardId {
		shardId = fmt.Sprintf("%d", shardCoordinator.SelfId())
	}

	uniqueDBFolder := filepath.Join(
		workingDir,
		defaultDBPath,
		fmt.Sprintf("%s_%d", defaultEpochString, 0),
		fmt.Sprintf("%s_%s", defaultShardString, shardId))

	storageCleanup := ctx.GlobalBool(storageCleanup.Name)
	if storageCleanup {
		err = os.RemoveAll(uniqueDBFolder)
		if err != nil {
			return err
		}
	}

	logDirectory := filepath.Join(workingDir, defaultLogPath)

	err = os.MkdirAll(logDirectory, os.ModePerm)
	if err != nil {
		return err
	}

	coreArgs := factory.NewCoreComponentsFactoryArgs(generalConfig, uniqueDBFolder)
	coreComponents, err := factory.CoreComponentsFactory(coreArgs)
	if err != nil {
		return err
	}

	epochStartNotifier := notifier.NewEpochStartSubscriptionHandler()
	// TODO: use epochStartNotifier in nodes coordinator
	nodesCoordinator, err := createNodesCoordinator(
		nodesConfig,
		generalConfig.GeneralSettings,
		epochStartNotifier,
		pubKey,
		coreComponents.Hasher)
	if err != nil {
		return err
	}

	stateArgs := factory.NewStateComponentsFactoryArgs(
		generalConfig,
		genesisConfig,
		shardCoordinator,
		coreComponents,
		uniqueDBFolder,
	)
	stateComponents, err := factory.StateComponentsFactory(stateArgs)
	if err != nil {
		return err
	}

	err = initLogFileAndStatsMonitor(generalConfig, pubKey, log, workingDir)
	if err != nil {
		return err
	}

	var appStatusHandlers []core.AppStatusHandler
	var views []factoryViews.Viewer

	prometheusJoinUrl, usePrometheusBool := getPrometheusJoinURLIfAvailable(ctx)
	if usePrometheusBool {
		prometheusStatusHandler := statusHandler.NewPrometheusStatusHandler()
		appStatusHandlers = append(appStatusHandlers, prometheusStatusHandler)
	}

	presenterStatusHandler := factory.CreateStatusHandlerPresenter()

	useTermui := !ctx.GlobalBool(useLogView.Name)
	if useTermui {

		views, err = factory.CreateViews(presenterStatusHandler)
		if err != nil {
			return err
		}

		writer, ok := presenterStatusHandler.(io.Writer)
		if ok {
			logger.ClearLogObservers()
			err = logger.AddLogObserver(writer, &logger.PlainFormatter{})
			if err != nil {
				return err
			}
		}

		appStatusHandler, ok := presenterStatusHandler.(core.AppStatusHandler)
		if ok {
			appStatusHandlers = append(appStatusHandlers, appStatusHandler)
		}
	}

	if views == nil {
		log.Warn("no views for current node")
	}

	statusMetrics := statusHandler.NewStatusMetrics()
	appStatusHandlers = append(appStatusHandlers, statusMetrics)

	if len(appStatusHandlers) > 0 {
		coreComponents.StatusHandler, err = statusHandler.NewAppStatusFacadeWithHandlers(appStatusHandlers...)
		if err != nil {
			log.Debug("cannot init AppStatusFacade",
				"error", err.Error(),
			)
		}
	} else {
		coreComponents.StatusHandler = statusHandler.NewNilStatusHandler()
		log.Debug("no AppStatusHandler used, started with NilStatusHandler")
	}

	metrics.InitMetrics(coreComponents.StatusHandler, pubKey, nodeType, shardCoordinator, nodesConfig, version, economicsConfig)

	dataArgs := factory.NewDataComponentsFactoryArgs(generalConfig, shardCoordinator, coreComponents, uniqueDBFolder)
	dataComponents, err := factory.DataComponentsFactory(dataArgs)
	if err != nil {
		return err
	}

	cryptoArgs := factory.NewCryptoComponentsFactoryArgs(
		ctx,
		generalConfig,
		nodesConfig,
		shardCoordinator,
		keyGen,
		privKey,
		log,
		initialBalancesSkPemFile.Name,
		txSignSk.Name,
		txSignSkIndex.Name,
	)
	cryptoComponents, err := factory.CryptoComponentsFactory(cryptoArgs)
	if err != nil {
		return err
	}

	txSignPk := factory.GetPkEncoded(cryptoComponents.TxSignPubKey)
	metrics.SaveCurrentNodeNameAndPubKey(coreComponents.StatusHandler, txSignPk, preferencesConfig.Preferences.NodeDisplayName)

	sessionInfoFileOutput := fmt.Sprintf("%s:%s\n%s:%s\n%s:%s\n%s:%v\n%s:%s\n%s:%v\n",
		"PkBlockSign", factory.GetPkEncoded(pubKey),
		"PkAccount", factory.GetPkEncoded(cryptoComponents.TxSignPubKey),
		"ShardId", shardId,
		"TotalShards", shardCoordinator.NumberOfShards(),
		"AppVersion", version,
		"GenesisTimeStamp", startTime.Unix(),
	)

	sessionInfoFileOutput += fmt.Sprintf("\nStarted with parameters:\n")
	for _, flag := range ctx.App.Flags {
		flagValue := fmt.Sprintf("%v", ctx.GlobalGeneric(flag.GetName()))
		if flagValue != "" {
			sessionInfoFileOutput += fmt.Sprintf("%s = %v\n", flag.GetName(), flagValue)
		}
	}

	err = ioutil.WriteFile(filepath.Join(logDirectory, "session.info"), []byte(sessionInfoFileOutput), os.ModePerm)
	log.LogIfError(err)

	networkComponents, err := factory.NetworkComponentsFactory(p2pConfig, log, coreComponents)
	if err != nil {
		return err
	}

	tpsBenchmark, err := statistics.NewTPSBenchmark(shardCoordinator.NumberOfShards(), nodesConfig.RoundDuration/1000)
	if err != nil {
		return err
	}

	if generalConfig.Explorer.Enabled {
		serversConfigurationFileName := ctx.GlobalString(serversConfigurationFile.Name)
		dbIndexer, err = createElasticIndexer(
			ctx,
			serversConfigurationFileName,
			generalConfig.Explorer.IndexerURL,
			shardCoordinator,
			coreComponents.Marshalizer,
			coreComponents.Hasher,
		)
		if err != nil {
			return err
		}

		err = setServiceContainer(shardCoordinator, tpsBenchmark)
		if err != nil {
			return err
		}
	}

	economicsData, err := economics.NewEconomicsData(economicsConfig)
	if err != nil {
		return err
	}

	processArgs := factory.NewProcessComponentsFactoryArgs(
		coreArgs,
		genesisConfig,
		economicsData,
		nodesConfig,
		syncer,
		shardCoordinator,
		nodesCoordinator,
		dataComponents,
		coreComponents,
		cryptoComponents,
		stateComponents,
		networkComponents,
		coreServiceContainer,
		epochStartNotifier,
		&generalConfig.EpochStartConfig,
		0,
	)
	processComponents, err := factory.ProcessComponentsFactory(processArgs)
	if err != nil {
		return err
	}

	var elasticIndexer indexer.Indexer
	if coreServiceContainer == nil || coreServiceContainer.IsInterfaceNil() {
		elasticIndexer = nil
	} else {
		elasticIndexer = coreServiceContainer.Indexer()
	}

	currentNode, err := createNode(
		generalConfig,
		preferencesConfig,
		nodesConfig,
		economicsData,
		syncer,
		keyGen,
		privKey,
		pubKey,
		shardCoordinator,
		nodesCoordinator,
		coreComponents,
		stateComponents,
		dataComponents,
		cryptoComponents,
		processComponents,
		networkComponents,
		ctx.GlobalUint64(bootstrapRoundIndex.Name),
		version,
		elasticIndexer,
	)
	if err != nil {
		return err
	}

	softwareVersionChecker, err := factory.CreateSoftwareVersionChecker(coreComponents.StatusHandler)
	if err != nil {
		log.Debug("nil software version checker", "error", err.Error())
	} else {
		softwareVersionChecker.StartCheckSoftwareVersion()
	}

	if shardCoordinator.SelfId() == core.MetachainShardId {
		indexValidatorsListIfNeeded(elasticIndexer, nodesCoordinator)
	}

	apiResolver, err := createApiResolver(
		stateComponents.AccountsAdapter,
		stateComponents.AddressConverter,
		dataComponents.Store,
		dataComponents.Blkc,
		coreComponents.Marshalizer,
		coreComponents.Uint64ByteSliceConverter,
		shardCoordinator,
		statusMetrics,
		economicsData,
	)
	if err != nil {
		return err
	}

	err = metrics.StartStatusPolling(
		currentNode.GetAppStatusHandler(),
		generalConfig.GeneralSettings.StatusPollingIntervalSec,
		networkComponents,
		processComponents,
	)
	if err != nil {
		return err
	}

	updateMachineStatisticsDurationSec := 1
	err = metrics.StartMachineStatisticsPolling(coreComponents.StatusHandler, updateMachineStatisticsDurationSec)
	if err != nil {
		return err
	}

	restAPIServerDebugMode := ctx.GlobalBool(restApiDebug.Name)
	ef := facade.NewElrondNodeFacade(currentNode, apiResolver, restAPIServerDebugMode)

	efConfig := &config.FacadeConfig{
		RestApiInterface:  ctx.GlobalString(restApiInterface.Name),
		PprofEnabled:      ctx.GlobalBool(profileMode.Name),
		Prometheus:        usePrometheusBool,
		PrometheusJoinURL: prometheusJoinUrl,
		PrometheusJobName: generalConfig.GeneralSettings.NetworkID,
	}

	ef.SetSyncer(syncer)
	ef.SetTpsBenchmark(tpsBenchmark)
	ef.SetConfig(efConfig)

	wg := sync.WaitGroup{}
	go ef.StartBackgroundServices(&wg)
	wg.Wait()

	if !ctx.Bool(withUI.Name) {
		log.Debug("bootstrapping node...")
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

	log.Info("application is now running")
	<-stop

	if rm != nil {
		err = rm.Close()
		log.LogIfError(err)
	}
	return nil
}

func indexValidatorsListIfNeeded(elasticIndexer indexer.Indexer, coordinator sharding.NodesCoordinator) {
	if elasticIndexer == nil || elasticIndexer.IsInterfaceNil() {
		return
	}

	validatorsPubKeys, _ := coordinator.GetAllValidatorsPublicKeys(0)

	if validatorsPubKeys != nil {
		go elasticIndexer.SaveValidatorsPubKeys(validatorsPubKeys)
	}
}

func getPrometheusJoinURLIfAvailable(ctx *cli.Context) (string, bool) {
	prometheusURLAvailable := true
	prometheusJoinUrl, err := getPrometheusJoinURL(ctx.GlobalString(serversConfigurationFile.Name))
	if err != nil || prometheusJoinUrl == "" {
		prometheusURLAvailable = false
	}
	usePrometheusBool := ctx.GlobalBool(usePrometheus.Name) && prometheusURLAvailable

	return prometheusJoinUrl, usePrometheusBool
}

func getPrometheusJoinURL(serversConfigurationFileName string) (string, error) {
	serversConfig, err := core.LoadServersPConfig(serversConfigurationFileName)
	if err != nil {
		return "", err
	}
	baseURL := serversConfig.Prometheus.PrometheusBaseURL
	statusURL := baseURL + serversConfig.Prometheus.StatusRoute
	resp, err := http.Get(statusURL)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == http.StatusNotFound {
		return "", errors.New("prometheus URL not available")
	}
	joinURL := baseURL + serversConfig.Prometheus.JoinRoute
	return joinURL, nil
}

func enableGopsIfNeeded(ctx *cli.Context, log logger.Logger) {
	var gopsEnabled bool
	if ctx.IsSet(gopsEn.Name) {
		gopsEnabled = ctx.GlobalBool(gopsEn.Name)
	}

	if gopsEnabled {
		if err := agent.Listen(agent.Options{}); err != nil {
			log.Error("failure to init gops", "error", err.Error())
		}
	}
}

func loadMainConfig(filepath string) (*config.Config, error) {
	cfg := &config.Config{}
	err := core.LoadTomlFile(cfg, filepath)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadEconomicsConfig(filepath string) (*config.ConfigEconomics, error) {
	cfg := &config.ConfigEconomics{}
	err := core.LoadTomlFile(cfg, filepath)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadPreferencesConfig(filepath string) (*config.ConfigPreferences, error) {
	cfg := &config.ConfigPreferences{}
	err := core.LoadTomlFile(cfg, filepath)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func getShardIdFromNodePubKey(pubKey crypto.PublicKey, nodesConfig *sharding.NodesSetup) (uint32, error) {
	if pubKey == nil {
		return 0, errors.New("nil public key")
	}

	publicKey, err := pubKey.ToByteArray()
	if err != nil {
		return 0, err
	}

	selfShardId, err := nodesConfig.GetShardIDForPubKey(publicKey)
	if err != nil {
		return 0, err
	}

	return selfShardId, err
}

func createShardCoordinator(
	nodesConfig *sharding.NodesSetup,
	pubKey crypto.PublicKey,
	settingsConfig config.GeneralSettingsConfig,
	log logger.Logger,
) (sharding.Coordinator, core.NodeType, error) {
	selfShardId, err := getShardIdFromNodePubKey(pubKey, nodesConfig)
	nodeType := core.NodeTypeValidator
	if err == sharding.ErrPublicKeyNotFoundInGenesis {
		nodeType = core.NodeTypeObserver
		log.Info("starting as observer node")

		selfShardId, err = processDestinationShardAsObserver(settingsConfig)
	}
	if err != nil {
		return nil, "", err
	}

	var shardName string
	if selfShardId == core.MetachainShardId {
		shardName = metachainShardName
	} else {
		shardName = fmt.Sprintf("%d", selfShardId)
	}
	log.Info("shard info", "started in shard", shardName)

	shardCoordinator, err := sharding.NewMultiShardCoordinator(nodesConfig.NumberOfShards(), selfShardId)
	if err != nil {
		return nil, "", err
	}

	return shardCoordinator, nodeType, nil
}

func createNodesCoordinator(
	nodesConfig *sharding.NodesSetup,
	settingsConfig config.GeneralSettingsConfig,
	epochStartSubscriber EpochStartSubscriber,
	pubKey crypto.PublicKey,
	hasher hashing.Hasher,
) (sharding.NodesCoordinator, error) {

	shardId, err := getShardIdFromNodePubKey(pubKey, nodesConfig)
	if err == sharding.ErrPublicKeyNotFoundInGenesis {
		shardId, err = processDestinationShardAsObserver(settingsConfig)
	}
	if err != nil {
		return nil, err
	}

	nbShards := nodesConfig.NumberOfShards()
	shardConsensusGroupSize := int(nodesConfig.ConsensusGroupSize)
	metaConsensusGroupSize := int(nodesConfig.MetaChainConsensusGroupSize)
	eligibleNodesInfo, waitingNodesInfo := nodesConfig.InitialNodesInfo()

	eligibleValidators, err := nodesInfoToValidators(eligibleNodesInfo)
	if err != nil {
		return nil, err
	}

	waitingValidators, err := nodesInfoToValidators(waitingNodesInfo)
	if err != nil {
		return nil, err
	}

	pubKeyBytes, err := pubKey.ToByteArray()
	if err != nil {
		return nil, err
	}

	nodeShuffler := sharding.NewXorValidatorsShuffler(
		nodesConfig.MinNodesPerShard,
		nodesConfig.MetaChainMinNodes,
		nodesConfig.Hysteresis,
		nodesConfig.Adaptivity,
	)

	argumentsNodesCoordinator := sharding.ArgNodesCoordinator{
		ShardConsensusGroupSize: shardConsensusGroupSize,
		MetaConsensusGroupSize:  metaConsensusGroupSize,
		Hasher:                  hasher,
		Shuffler:                nodeShuffler,
		EpochStartSubscriber:    epochStartSubscriber,
		ShardId:                 shardId,
		NbShards:                nbShards,
		EligibleNodes:           eligibleValidators,
		WaitingNodes:            waitingValidators,
		SelfPublicKey:           pubKeyBytes,
	}
	nodesCoordinator, err := sharding.NewIndexHashedNodesCoordinator(argumentsNodesCoordinator)
	if err != nil {
		return nil, err
	}

	return nodesCoordinator, nil
}

func nodesInfoToValidators(nodesInfo map[uint32][]*sharding.NodeInfo) (map[uint32][]sharding.Validator, error) {
	validatorsMap := make(map[uint32][]sharding.Validator)

	for shId, nodeInfoList := range nodesInfo {
		validators := make([]sharding.Validator, 0)
		for _, nodeInfo := range nodeInfoList {
			validator, err := sharding.NewValidator(big.NewInt(0), 0, nodeInfo.PubKey(), nodeInfo.Address())
			if err != nil {
				return nil, err
			}

			validators = append(validators, validator)
		}
		validatorsMap[shId] = validators
	}

	return validatorsMap, nil
}

func processDestinationShardAsObserver(settingsConfig config.GeneralSettingsConfig) (uint32, error) {
	destShard := strings.ToLower(settingsConfig.DestinationShardAsObserver)
	if len(destShard) == 0 {
		return 0, errors.New("option DestinationShardAsObserver is not set in config.toml")
	}
	if destShard == metachainShardName {
		return core.MetachainShardId, nil
	}

	val, err := strconv.ParseUint(destShard, 10, 32)
	if err != nil {
		return 0, errors.New("error parsing DestinationShardAsObserver option: " + err.Error())
	}

	return uint32(val), err
}

// createElasticIndexer creates a new elasticIndexer where the server listens on the url,
// authentication for the server is using the username and password
func createElasticIndexer(
	ctx *cli.Context,
	serversConfigurationFileName string,
	url string,
	coordinator sharding.Coordinator,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
) (indexer.Indexer, error) {
	serversConfig, err := core.LoadServersPConfig(serversConfigurationFileName)
	if err != nil {
		return nil, err
	}

	dbIndexer, err = indexer.NewElasticIndexer(
		url,
		serversConfig.ElasticSearch.Username,
		serversConfig.ElasticSearch.Password,
		coordinator,
		marshalizer,
		hasher,
		&indexer.Options{TxIndexingEnabled: ctx.GlobalBoolT(enableTxIndexing.Name)})
	if err != nil {
		return nil, err
	}

	return dbIndexer, nil
}

func getConsensusGroupSize(nodesConfig *sharding.NodesSetup, shardCoordinator sharding.Coordinator) (uint32, error) {
	if shardCoordinator.SelfId() == core.MetachainShardId {
		return nodesConfig.MetaChainConsensusGroupSize, nil
	}
	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		return nodesConfig.ConsensusGroupSize, nil
	}

	return 0, state.ErrUnknownShardId
}

func createNode(
	config *config.Config,
	preferencesConfig *config.ConfigPreferences,
	nodesConfig *sharding.NodesSetup,
	economicsData process.FeeHandler,
	syncer ntp.SyncTimer,
	keyGen crypto.KeyGenerator,
	privKey crypto.PrivateKey,
	pubKey crypto.PublicKey,
	shardCoordinator sharding.Coordinator,
	nodesCoordinator sharding.NodesCoordinator,
	coreData *factory.Core,
	state *factory.State,
	data *factory.Data,
	crypto *factory.Crypto,
	process *factory.Process,
	network *factory.Network,
	bootstrapRoundIndex uint64,
	version string,
	indexer indexer.Indexer,
) (*node.Node, error) {
	consensusGroupSize, err := getConsensusGroupSize(nodesConfig, shardCoordinator)
	if err != nil {
		return nil, err
	}

	nd, err := node.NewNode(
		node.WithMessenger(network.NetMessenger),
		node.WithHasher(coreData.Hasher),
		node.WithMarshalizer(coreData.Marshalizer),
		node.WithTxFeeHandler(economicsData),
		node.WithInitialNodesPubKeys(crypto.InitialPubKeys),
		node.WithAddressConverter(state.AddressConverter),
		node.WithAccountsAdapter(state.AccountsAdapter),
		node.WithBlockChain(data.Blkc),
		node.WithDataStore(data.Store),
		node.WithRoundDuration(nodesConfig.RoundDuration),
		node.WithConsensusGroupSize(int(consensusGroupSize)),
		node.WithSyncer(syncer),
		node.WithBlockProcessor(process.BlockProcessor),
		node.WithGenesisTime(time.Unix(nodesConfig.StartTime, 0)),
		node.WithRounder(process.Rounder),
		node.WithShardCoordinator(shardCoordinator),
		node.WithNodesCoordinator(nodesCoordinator),
		node.WithUint64ByteSliceConverter(coreData.Uint64ByteSliceConverter),
		node.WithSingleSigner(crypto.SingleSigner),
		node.WithMultiSigner(crypto.MultiSigner),
		node.WithKeyGen(keyGen),
		node.WithKeyGenForAccounts(crypto.TxSignKeyGen),
		node.WithTxSignPubKey(crypto.TxSignPubKey),
		node.WithTxSignPrivKey(crypto.TxSignPrivKey),
		node.WithPubKey(pubKey),
		node.WithPrivKey(privKey),
		node.WithForkDetector(process.ForkDetector),
		node.WithInterceptorsContainer(process.InterceptorsContainer),
		node.WithResolversFinder(process.ResolversFinder),
		node.WithConsensusType(config.Consensus.Type),
		node.WithTxSingleSigner(crypto.TxSingleSigner),
		node.WithTxStorageSize(config.TxStorage.Cache.Size),
		node.WithBootstrapRoundIndex(bootstrapRoundIndex),
		node.WithAppStatusHandler(coreData.StatusHandler),
		node.WithIndexer(indexer),
		node.WithEpochStartTrigger(process.EpochStartTrigger),
		node.WithBlackListHandler(process.BlackListHandler),
	)
	if err != nil {
		return nil, errors.New("error creating node: " + err.Error())
	}

	err = nd.StartHeartbeat(config.Heartbeat, version, preferencesConfig.Preferences.NodeDisplayName)
	if err != nil {
		return nil, err
	}

	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		err = nd.ApplyOptions(
			node.WithInitialNodesBalances(state.InBalanceForShard),
			node.WithDataPool(data.Datapool),
		)
		if err != nil {
			return nil, errors.New("error creating node: " + err.Error())
		}
		err = nd.CreateShardedStores()
		if err != nil {
			return nil, err
		}
	}
	if shardCoordinator.SelfId() == core.MetachainShardId {
		err = nd.ApplyOptions(node.WithMetaDataPool(data.MetaDatapool))
		if err != nil {
			return nil, errors.New("error creating meta-node: " + err.Error())
		}
	}
	return nd, nil
}

func initLogFileAndStatsMonitor(config *config.Config, pubKey crypto.PublicKey, log logger.Logger,
	workingDir string) error {
	publicKey, err := pubKey.ToByteArray()
	if err != nil {
		return err
	}

	hexPublicKey := core.GetTrimmedPk(hex.EncodeToString(publicKey))

	statsFile, err := core.CreateFile(hexPublicKey, filepath.Join(workingDir, defaultStatsPath), "txt")
	if err != nil {
		return err
	}
	err = startStatisticsMonitor(statsFile, config.ResourceStats, log)
	if err != nil {
		return err
	}
	return nil
}

func setServiceContainer(shardCoordinator sharding.Coordinator, tpsBenchmark *statistics.TpsBenchmark) error {
	var err error
	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		coreServiceContainer, err = serviceContainer.NewServiceContainer(serviceContainer.WithIndexer(dbIndexer))
		if err != nil {
			return err
		}
		return nil
	}
	if shardCoordinator.SelfId() == core.MetachainShardId {
		coreServiceContainer, err = serviceContainer.NewServiceContainer(
			serviceContainer.WithIndexer(dbIndexer),
			serviceContainer.WithTPSBenchmark(tpsBenchmark))
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("could not init core service container")
}

func startStatisticsMonitor(file *os.File, config config.ResourceStatsConfig, log logger.Logger) error {
	if !config.Enabled {
		return nil
	}

	if config.RefreshIntervalInSec < 1 {
		return errors.New("invalid RefreshIntervalInSec in section [ResourceStats]. Should be an integer higher than 1")
	}

	resMon, err := statistics.NewResourceMonitor(file)
	if err != nil {
		return err
	}

	go func() {
		for {
			err = resMon.SaveStatistics()
			log.LogIfError(err)
			time.Sleep(time.Second * time.Duration(config.RefreshIntervalInSec))
		}
	}()

	return nil
}

func createApiResolver(
	accnts state.AccountsAdapter,
	addrConv state.AddressConverter,
	storageService dataRetriever.StorageService,
	blockChain data.ChainHandler,
	marshalizer marshal.Marshalizer,
	uint64Converter typeConverters.Uint64ByteSliceConverter,
	shardCoordinator sharding.Coordinator,
	statusMetrics external.StatusMetricsHandler,
	economics *economics.EconomicsData,
) (facade.ApiResolver, error) {
	var vmFactory process.VirtualMachinesContainerFactory
	var err error

	argsHook := hooks.ArgBlockChainHook{
		Accounts:         accnts,
		AddrConv:         addrConv,
		StorageService:   storageService,
		BlockChain:       blockChain,
		ShardCoordinator: shardCoordinator,
		Marshalizer:      marshalizer,
		Uint64Converter:  uint64Converter,
	}

	if shardCoordinator.SelfId() == core.MetachainShardId {
		vmFactory, err = metachain.NewVMContainerFactory(argsHook, economics)
		if err != nil {
			return nil, err
		}
	} else {
		vmFactory, err = shard.NewVMContainerFactory(argsHook)
		if err != nil {
			return nil, err
		}
	}

	vmContainer, err := vmFactory.Create()
	if err != nil {
		return nil, err
	}

	scDataGetter, err := smartContract.NewSCDataGetter(addrConv, vmContainer)
	if err != nil {
		return nil, err
	}

	return external.NewNodeApiResolver(scDataGetter, statusMetrics)
}
