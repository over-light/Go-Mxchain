package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/cmd/termui/provider"
	"github.com/ElrondNetwork/elrond-go/statusHandler/presenter"
	"github.com/ElrondNetwork/elrond-go/statusHandler/view/termuic"
	"github.com/urfave/cli"
)

type config struct {
	logWithCorrelation bool
	logWithLoggerName  bool
	useWss             bool
	interval           int
	address            string
	logLevel           string
}

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
	// address defines a flag for setting the address and port on which the node will listen for connections
	address = cli.StringFlag{
		Name:        "address",
		Usage:       "Address and port number on which the application will try to connect to the elrond-go node",
		Value:       "127.0.0.1:8080",
		Destination: &argsConfig.address,
	}

	// logLevel defines the logger levels and patterns
	logLevel = cli.StringFlag{
		Name:        "log-level",
		Usage:       "This flag specifies the logger level",
		Value:       "*:" + logger.LogInfo.String(),
		Destination: &argsConfig.logLevel,
	}

	// logWithCorrelation is used when the user wants to include the log correlation elements in logs
	logWithCorrelation = cli.BoolFlag{
		Name:        "log-correlation",
		Usage:       "Will include log correlation elements",
		Destination: &argsConfig.logWithCorrelation,
	}

	// logWithLoggerName is used when the user wants to include the logger name in logs
	logWithLoggerName = cli.BoolFlag{
		Name:        "log-logger-name",
		Usage:       "Will include logger name",
		Destination: &argsConfig.logWithLoggerName,
	}

	// fetchInterval configures polling period
	fetchInterval = cli.IntFlag{
		Name:        "interval",
		Usage:       "This flag specifies the duration in seconds until new data is fetched from the node",
		Value:       2,
		Destination: &argsConfig.interval,
	}

	//useWss is used when the user require connection through wss
	useWss = cli.BoolFlag{
		Name:        "use-wss",
		Usage:       "Will use wss instead of ws when creating the web socket",
		Destination: &argsConfig.useWss,
	}
	argsConfig = &config{}

	log    = logger.GetOrCreate("termui")
	cliApp *cli.App
)

func main() {
	initCliFlags()

	cliApp.Action = func(c *cli.Context) error {
		return startTermuiViewer()
	}

	err := cliApp.Run(os.Args)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func startTermuiViewer() error {
	nodeAddress := argsConfig.address
	fetchIntervalFlagValue := argsConfig.interval

	presenterStatusHandler := presenter.NewPresenterStatusHandler()
	statusMetricsProvider, err := provider.NewStatusMetricsProvider(presenterStatusHandler, nodeAddress, fetchIntervalFlagValue)
	if err != nil {
		return err
	}

	termuiConsole, err := termuic.NewTermuiConsole(presenterStatusHandler)
	if err != nil {
		return err
	}

	statusMetricsProvider.StartUpdatingData()

	loggerProfile := logger.Profile{
		LogLevelPatterns: argsConfig.logLevel,
		WithCorrelation:  argsConfig.logWithCorrelation,
		WithLoggerName:   argsConfig.logWithLoggerName,
	}

	err = provider.InitLogHandler(presenterStatusHandler, nodeAddress, loggerProfile, argsConfig.useWss)
	if err != nil {
		return err
	}

	chanStartTermUI := make(chan struct{})
	err = termuiConsole.Start(chanStartTermUI)
	if err != nil {
		return err
	}
	chanStartTermUI <- struct{}{}

	waitForUserToTerminateApp()

	return nil
}

func initCliFlags() {
	cliApp = cli.NewApp()
	cli.AppHelpTemplate = nodeHelpTemplate
	cliApp.Name = "Elrond Terminal UI App"
	cliApp.Version = fmt.Sprintf("%s/%s/%s-%s", "1.0.0", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	cliApp.Usage = "Terminal UI application used to display metrics from the node"
	cliApp.Flags = []cli.Flag{
		address,
		logLevel,
		logWithCorrelation,
		logWithLoggerName,
		fetchInterval,
		useWss,
	}
	cliApp.Authors = []cli.Author{
		{
			Name:  "The Elrond Team",
			Email: "contact@elrond.com",
		},
	}
}

func waitForUserToTerminateApp() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

	log.Info("terminating terminal ui app at user's signal...")

	provider.StopWebSocket()
}
