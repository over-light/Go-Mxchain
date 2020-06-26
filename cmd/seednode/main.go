package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go-logger/redirects"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/display"
	"github.com/ElrondNetwork/elrond-go/marshal"
	factoryMarshalizer "github.com/ElrondNetwork/elrond-go/marshal/factory"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p"
	"github.com/urfave/cli"
)

const defaultLogsPath = "logs"
const filePathPlaceholder = "[path]"

var (
	seedNodeHelpTemplate = `NAME:
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
	// port defines a flag for setting the port on which the node will listen for connections
	port = cli.StringFlag{
		Name: "port",
		Usage: "The `[p2p port]` number on which the application will start. Can use single values such as " +
			"`0, 10230, 15670` or range of ports such as `5000-10000`",
		Value: "10000",
	}
	// p2pSeed defines a flag to be used as a seed when generating P2P credentials. Useful for seed nodes.
	p2pSeed = cli.StringFlag{
		Name:  "p2p-seed",
		Usage: "P2P seed will be used when generating credentials for p2p component. Can be any string.",
		Value: "seed",
	}
	// logLevel defines the logger level
	logLevel = cli.StringFlag{
		Name: "log-level",
		Usage: "This flag specifies the logger `level(s)`. It can contain multiple comma-separated value. For example" +
			", if set to *:INFO the logs for all packages will have the INFO level. However, if set to *:INFO,api:DEBUG" +
			" the logs for all packages will have the INFO level, excepting the api package which will receive a DEBUG" +
			" log level.",
		Value: "*:" + logger.LogInfo.String(),
	}
	//logFile is used when the log output needs to be logged in a file
	logSaveFile = cli.BoolFlag{
		Name:  "log-save",
		Usage: "Boolean option for enabling log saving. If set, it will automatically save all the logs into a file.",
	}
	// configurationFile defines a flag for the path to the main toml configuration file
	configurationFile = cli.StringFlag{
		Name: "config",
		Usage: "The `" + filePathPlaceholder + "` for the main configuration file. This TOML file contain the main " +
			"configurations such as the marshalizer type",
		Value: "./config/config.toml",
	}
	p2pConfigurationFile = "./config/p2p.toml"
)

var log = logger.GetOrCreate("main")

func main() {
	app := cli.NewApp()
	cli.AppHelpTemplate = seedNodeHelpTemplate
	app.Name = "SeedNode CLI App"
	app.Usage = "This is the entry point for starting a new seed node - the app will help bootnodes connect to the network"
	app.Flags = []cli.Flag{
		port,
		p2pSeed,
		logLevel,
		logSaveFile,
		configurationFile,
	}
	app.Version = "v0.0.1"
	app.Authors = []cli.Author{
		{
			Name:  "The Elrond Team",
			Email: "contact@elrond.com",
		},
	}

	app.Action = func(c *cli.Context) error {
		return startNode(c)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func startNode(ctx *cli.Context) error {
	var err error

	configurationFileName := ctx.GlobalString(configurationFile.Name)
	generalConfig, err := loadMainConfig(configurationFileName)
	if err != nil {
		return err
	}

	internalMarshalizer, err := factoryMarshalizer.NewMarshalizer(generalConfig.Marshalizer.Type)
	if err != nil {
		return fmt.Errorf("error creating marshalizer (internal): %s", err.Error())
	}

	withLogFile := ctx.GlobalBool(logSaveFile.Name)
	if withLogFile {
		var fileForLogs *os.File
		workingDir := getWorkingDir(log)
		fileForLogs, err = prepareLogFile(workingDir)
		if err != nil {
			return fmt.Errorf("%w creating a log file", err)
		}

		defer func() {
			_ = fileForLogs.Close()
		}()
	}

	logLevelFlagValue := ctx.GlobalString(logLevel.Name)
	err = logger.SetLogLevel(logLevelFlagValue)
	if err != nil {
		return err
	}

	log.Info("starting seednode...")

	stop := make(chan bool, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	p2pConfig, err := core.LoadP2PConfig(p2pConfigurationFile)
	if err != nil {
		return err
	}
	log.Info("initialized with p2p config",
		"filename", p2pConfigurationFile,
	)
	if ctx.IsSet(port.Name) {
		p2pConfig.Node.Port = ctx.GlobalString(port.Name)
	}
	if ctx.IsSet(p2pSeed.Name) {
		p2pConfig.Node.Seed = ctx.GlobalString(p2pSeed.Name)
	}

	err = checkExpectedPeerCount(*p2pConfig)
	if err != nil {
		return err
	}

	messenger, err := createNode(*p2pConfig, internalMarshalizer)
	if err != nil {
		return err
	}

	err = messenger.Bootstrap()
	if err != nil {
		return err
	}

	go func() {
		<-sigs
		log.Info("terminating at user's signal...")
		stop <- true
	}()

	log.Info("application is now running...")
	displayMessengerInfo(messenger)
	for {
		select {
		case <-stop:
			return nil
		case <-time.After(time.Second * 5):
			displayMessengerInfo(messenger)
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

func createNode(p2pConfig config.P2PConfig, marshalizer marshal.Marshalizer) (p2p.Messenger, error) {
	arg := libp2p.ArgsNetworkMessenger{
		Marshalizer:   marshalizer,
		ListenAddress: libp2p.ListenAddrWithIp4AndTcp,
		P2pConfig:     p2pConfig,
	}

	return libp2p.NewNetworkMessenger(arg)
}

func displayMessengerInfo(messenger p2p.Messenger) {
	headerSeedAddresses := []string{"Seednode addresses:"}
	addresses := make([]*display.LineData, 0)

	for _, address := range messenger.Addresses() {
		addresses = append(addresses, display.NewLineData(false, []string{address}))
	}

	tbl, _ := display.CreateTableString(headerSeedAddresses, addresses)
	log.Info("\n" + tbl)

	mesConnectedAddrs := messenger.ConnectedAddresses()
	sort.Slice(mesConnectedAddrs, func(i, j int) bool {
		return strings.Compare(mesConnectedAddrs[i], mesConnectedAddrs[j]) < 0
	})

	log.Info("known peers", "num peers", len(messenger.Peers()))
	headerConnectedAddresses := []string{fmt.Sprintf("Seednode is connected to %d peers:", len(mesConnectedAddrs))}
	connAddresses := make([]*display.LineData, len(mesConnectedAddrs))

	for idx, address := range mesConnectedAddrs {
		connAddresses[idx] = display.NewLineData(false, []string{address})
	}

	tbl2, _ := display.CreateTableString(headerConnectedAddresses, connAddresses)
	log.Info("\n" + tbl2)
}

func prepareLogFile(workingDir string) (*os.File, error) {
	logDirectory := filepath.Join(workingDir, defaultLogsPath)
	fileForLog, err := core.CreateFile("elrond-go", logDirectory, "log")
	if err != nil {
		return nil, err
	}

	//we need this function as to close file.Close() when the code panics and the defer func associated
	//with the file pointer in the main func will never be reached
	runtime.SetFinalizer(fileForLog, func(f *os.File) {
		_ = f.Close()
	})

	err = redirects.RedirectStderr(fileForLog)
	if err != nil {
		return nil, err
	}

	err = logger.AddLogObserver(fileForLog, &logger.PlainFormatter{})
	if err != nil {
		return nil, fmt.Errorf("%w adding file log observer", err)
	}

	return fileForLog, nil
}

func getWorkingDir(log logger.Logger) string {
	workingDir, err := os.Getwd()
	if err != nil {
		log.LogIfError(err)
		workingDir = ""
	}

	log.Trace("working directory", "path", workingDir)

	return workingDir
}

func checkExpectedPeerCount(p2pConfig config.P2PConfig) error {
	maxExpectedPeerCount := p2pConfig.Node.MaximumExpectedPeerCount

	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return fmt.Errorf("%w while getting RLimits", err)
	}

	log.Info("file limits",
		"current", rLimit.Cur,
		"max", rLimit.Max,
		"expected", maxExpectedPeerCount,
	)

	if maxExpectedPeerCount > rLimit.Cur {
		return fmt.Errorf("provided maxExpectedPeerCount is less than the current OS configured value")
	}

	return nil
}
