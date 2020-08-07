package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/cmd/storer2elastic/config"
	"github.com/ElrondNetwork/elrond-go/cmd/storer2elastic/databasereader"
	"github.com/ElrondNetwork/elrond-go/cmd/storer2elastic/dataprocessor"
	"github.com/ElrondNetwork/elrond-go/cmd/storer2elastic/elastic"
	nodeConfigPackage "github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	stateFactory "github.com/ElrondNetwork/elrond-go/data/state/factory"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters/uint64ByteSlice"
	"github.com/ElrondNetwork/elrond-go/hashing"
	hasherFactory "github.com/ElrondNetwork/elrond-go/hashing/factory"
	"github.com/ElrondNetwork/elrond-go/marshal"
	marshalFactory "github.com/ElrondNetwork/elrond-go/marshal/factory"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage/factory"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/urfave/cli"
)

type flags struct {
	dbPathWithChainID  string
	configFilePath     string
	nodeConfigFilePath string
	nodesSetupFilePath string
	numShards          int
	timeout            int
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
	// configFilePathFlag defines a flag which holds the configuration file path
	configFilePathFlag = cli.StringFlag{
		Name:        "config",
		Usage:       "This string file specifies the `filepath` for the toml configuration file",
		Value:       "./config.toml",
		Destination: &flagsValues.configFilePath,
	}

	// address defines a flag for setting the address and port on which the node will listen for connections
	dbPathWithChainIDFlag = cli.StringFlag{
		Name:        "db-path",
		Usage:       "This string flag specifies the path for the database directory, the chain ID directory",
		Value:       "db/chainID",
		Destination: &flagsValues.dbPathWithChainID,
	}

	// nodeConfigFilePathFlag defines a flag which holds the configuration file path
	nodeConfigFilePathFlag = cli.StringFlag{
		Name:        "node-config",
		Usage:       "This string file specifies the `filepath` for the node's toml configuration file",
		Value:       "../node/config/config.toml",
		Destination: &flagsValues.nodeConfigFilePath,
	}

	// nodesSetupFilePathFlag defines a flag which holds the nodes setup json file path
	nodesSetupFilePathFlag = cli.StringFlag{
		Name:        "nodes-setup",
		Usage:       "This string file specifies the `filepath` for the node's nodes setup json configuration file",
		Value:       "config/nodesSetup.json",
		Destination: &flagsValues.nodesSetupFilePath,
	}

	// numOfShardsFlag defines the flag which holds the number of shards
	numOfShardsFlag = cli.IntFlag{
		Name:        "num-shards",
		Usage:       "This int flag specifies the number of shards",
		Value:       2,
		Destination: &flagsValues.numShards,
	}

	flagsValues = &flags{}

	log                      = logger.GetOrCreate("storer2elastic")
	cliApp                   *cli.App
	marshalizer              marshal.Marshalizer
	hasher                   hashing.Hasher
	uint64ByteSliceConverter typeConverters.Uint64ByteSliceConverter
	shardCoordinator         sharding.Coordinator
	nodeConfig               nodeConfigPackage.Config
	addressPubKeyConverter   core.PubkeyConverter
	validatorPubKeyConverter core.PubkeyConverter
)

func main() {
	initCliFlags()

	cliApp.Action = func(c *cli.Context) error {
		return startStorer2Elastic(c)
	}

	err := cliApp.Run(os.Args)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func initCliFlags() {
	cliApp = cli.NewApp()
	cli.AppHelpTemplate = nodeHelpTemplate
	cliApp.Name = "Elrond storer to Elastic Search App"
	cliApp.Version = fmt.Sprintf("%s/%s/%s-%s", "1.0.0", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	cliApp.Usage = "Elrond storer2elastic application is used to index all data inside storers to elastic search"
	cliApp.Flags = []cli.Flag{
		dbPathWithChainIDFlag,
		configFilePathFlag,
		nodeConfigFilePathFlag,
		nodesSetupFilePathFlag,
		numOfShardsFlag,
	}
	cliApp.Authors = []cli.Author{
		{
			Name:  "The Elrond Team",
			Email: "contact@elrond.com",
		},
	}
}

func startStorer2Elastic(ctx *cli.Context) error {
	log.Info("storer2elastic application started", "version", cliApp.Version)

	uint64ByteSliceConverter = uint64ByteSlice.NewBigEndianConverter()
	var err error
	shardCoordinator, err = sharding.NewMultiShardCoordinator(uint32(flagsValues.numShards), 0)
	if err != nil {
		return err
	}

	configuration := &config.Config{}
	err = core.LoadTomlFile(configuration, flagsValues.configFilePath)
	if err != nil {
		return err
	}
	if ctx.IsSet(dbPathWithChainIDFlag.Name) {
		configuration.General.DBPathWithChainID = ctx.GlobalString(dbPathWithChainIDFlag.Name)
	}
	if ctx.IsSet(nodeConfigFilePathFlag.Name) {
		configuration.General.NodeConfigFilePath = ctx.GlobalString(nodeConfigFilePathFlag.Name)
	}

	err = core.LoadTomlFile(&nodeConfig, configuration.General.NodeConfigFilePath)
	if err != nil {
		return err
	}

	marshalizer, err = marshalFactory.NewMarshalizer(nodeConfig.Marshalizer.Type)
	if err != nil {
		return err
	}
	hasher, err = hasherFactory.NewHasher(nodeConfig.Hasher.Type)
	if err != nil {
		return err
	}
	addressPubKeyConverter, err = stateFactory.NewPubkeyConverter(nodeConfig.AddressPubkeyConverter)
	if err != nil {
		return err
	}
	validatorPubKeyConverter, err = stateFactory.NewPubkeyConverter(nodeConfig.ValidatorPubkeyConverter)
	if err != nil {
		return err
	}

	elasticConnectorFactoryArgs := elastic.ConnectorFactoryArgs{
		ElasticConfig:            configuration.ElasticSearch,
		Marshalizer:              marshalizer,
		Hasher:                   hasher,
		ValidatorPubKeyConverter: validatorPubKeyConverter,
		AddressPubKeyConverter:   addressPubKeyConverter,
	}

	elasticConnectorFactory, err := elastic.NewConnectorFactory(elasticConnectorFactoryArgs)
	if err != nil {
		return err
	}

	elasticIndexer, err := elasticConnectorFactory.Create()
	if err != nil {
		return fmt.Errorf("error connecting to elastic: %w", err)
	}

	// TODO: maybe use custom configs from node config instead of a general configuration
	generalDBConfig := config.DBConfig{
		Type:              string(storageUnit.LvlDBSerial),
		BatchDelaySeconds: 2,
		MaxBatchSize:      30000,
		MaxOpenFiles:      20,
	}

	persisterFactory := factory.NewPersisterFactory(nodeConfigPackage.DBConfig(generalDBConfig))
	dbReaderArgs := databasereader.Args{
		DirectoryReader:   factory.NewDirectoryReader(),
		Marshalizer:       marshalizer,
		PersisterFactory:  persisterFactory,
		DbPathWithChainID: configuration.General.DBPathWithChainID,
	}
	dbReader, err := databasereader.New(dbReaderArgs)
	if err != nil {
		return err
	}

	ratingsProcessor, err := dataprocessor.NewRatingsProcessor(
		dataprocessor.RatingProcessorArgs{
			ShardCoordinator:         shardCoordinator,
			ValidatorPubKeyConverter: validatorPubKeyConverter,
			DbPathWithChainID:        configuration.General.DBPathWithChainID,
			GeneralConfig:            nodeConfig,
			Marshalizer:              marshalizer,
			Hasher:                   hasher,
			ElasticIndexer:           elasticIndexer,
		},
	)
	if err != nil {
		return err
	}

	genesisNodesConfig, err := sharding.NewNodesSetup(
		flagsValues.nodesSetupFilePath,
		addressPubKeyConverter,
		validatorPubKeyConverter,
	)
	if err != nil {
		return err
	}

	headerMarshalizer, err := databasereader.NewHeaderMarshalizer(marshalizer)
	if err != nil {
		return err
	}

	dataReplayerArgs := dataprocessor.DataReplayerArgs{
		DatabaseReader:           dbReader,
		ShardCoordinator:         shardCoordinator,
		Marshalizer:              marshalizer,
		Hasher:                   hasher,
		Uint64ByteSliceConverter: uint64ByteSliceConverter,
		HeaderMarshalizer:        headerMarshalizer,
	}

	dataReplayer, err := dataprocessor.NewDataReplayer(dataReplayerArgs)
	if err != nil {
		return err
	}

	dataProcessor2, err := dataprocessor.NewDataProcessor(
		dataprocessor.ArgsDataProcessor{
			ElasticIndexer:    elasticIndexer,
			DataReplayer:      dataReplayer,
			GenesisNodesSetup: genesisNodesConfig,
			Marshalizer:       marshalizer,
			Hasher:            hasher,
			ShardCoordinator:  shardCoordinator,
			RatingsProcessor:  ratingsProcessor,
		})
	if err != nil {
		return err
	}
	err = dataProcessor2.Index()
	if err != nil {
		return err
	}

	log.Info("finished indexing. app will close")

	return nil
}
