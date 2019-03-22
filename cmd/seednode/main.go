package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ElrondNetwork/elrond-go-sandbox/cmd/flags"
	"github.com/ElrondNetwork/elrond-go-sandbox/config"
	"github.com/ElrondNetwork/elrond-go-sandbox/core"
	"github.com/ElrondNetwork/elrond-go-sandbox/display"
	"github.com/ElrondNetwork/elrond-go-sandbox/hashing/sha256"
	"github.com/ElrondNetwork/elrond-go-sandbox/logger"
	"github.com/ElrondNetwork/elrond-go-sandbox/p2p"
	"github.com/ElrondNetwork/elrond-go-sandbox/p2p/libp2p"
	"github.com/ElrondNetwork/elrond-go-sandbox/p2p/libp2p/discovery"
	factoryP2P "github.com/ElrondNetwork/elrond-go-sandbox/p2p/libp2p/factory"
	"github.com/ElrondNetwork/elrond-go-sandbox/p2p/loadBalancer"
	"github.com/btcsuite/btcd/btcec"
	crypto2 "github.com/libp2p/go-libp2p-crypto"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var seedNodeHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}
USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}
   {{if len .Authors}}
GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}{{if .Copyright }}
VERSION:
   {{.Version}}
   {{end}}
`
var p2pConfigurationFile = "./config/p2p.toml"

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

func main() {
	log := logger.NewDefaultLogger()
	log.SetLevel(logger.LogInfo)

	app := cli.NewApp()
	cli.AppHelpTemplate = seedNodeHelpTemplate
	app.Name = "SeedNode CLI App"
	app.Usage = "This is the entry point for starting a new seed node - the app will help bootnodes connect to the network"
	app.Flags = []cli.Flag{flags.Port, flags.P2PSeed}

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
	log.Info("Starting node...")

	stop := make(chan bool, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	p2pConfig, err := loadP2PConfig(p2pConfigurationFile, log)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Initialized with p2p config from: %s", p2pConfigurationFile))
	if ctx.IsSet(flags.Port.Name) {
		p2pConfig.Node.Port = ctx.GlobalInt(flags.Port.Name)
	}
	if ctx.IsSet(flags.P2PSeed.Name) {
		p2pConfig.Node.Seed = ctx.GlobalString(flags.P2PSeed.Name)
	}

	log.Info("Seed node....")
	messenger, err := createNode(p2pConfig, log)
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

	log.Info("Application is now running...")

	for {
		select {
		case <-stop:
			return nil
		case <-time.After(time.Second):
			displayMessengerInfo(messenger)
		}
	}
}

func loadP2PConfig(filepath string, log *logger.Logger) (*config.P2PConfig, error) {
	cfg := &config.P2PConfig{}
	err := core.LoadTomlFile(cfg, filepath, log)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func createNode(
	p2pConfig *config.P2PConfig,
	log *logger.Logger,
) (p2p.Messenger, error) {

	hasher := sha256.Sha256{}

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

	return netMessenger, nil
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
	_, ok := pDiscoverer.(*discovery.KadDhtDiscoverer)
	if !ok {
		return nil, errors.New("kad-dht peer discovery should have been enabled")
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

func displayMessengerInfo(messenger p2p.Messenger) {
	header1 := []string{"Available addresses:"}
	addresses := make([]*display.LineData, 0)
	for _, address := range messenger.Addresses() {
		addresses = append(addresses, display.NewLineData(false, []string{address}))
	}
	fmt.Println(display.CreateTableString(header1, addresses))

	header1 := []string{"Available addresses:"}
	addresses := make([]*display.LineData, 0)
	for _, address := range messenger.Addresses() {
		addresses = append(addresses, display.NewLineData(false, []string{address}))
	}
	fmt.Println(display.CreateTableString(header1, addresses))

}
