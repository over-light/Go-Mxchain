package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/crypto/signing"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/ed25519"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/mcl"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/state/factory"
	"github.com/urfave/cli"
)

type cfg struct {
	numKeys   int
	keyType   string
	keyFormat string
}

var (
	fileGenHelpTemplate = `NAME:
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

	// numKeys defines a flag for setting how many keys should generate
	numKeys = cli.IntFlag{
		Name:        "numKeys",
		Usage:       "How many keys should generate. Example: 1",
		Value:       1,
		Destination: &argsConfig.numKeys,
	}

	// keyType defines a flag for setting what keys should generate
	keyType = cli.StringFlag{
		Name:        "keyType",
		Usage:       "What king of keys should generate. Available options: block, tx, both",
		Value:       "both",
		Destination: &argsConfig.keyType,
	}

	// keyFormat defines a flag for setting the format for the keys to generate
	keyFormat = cli.StringFlag{
		Name:        "keyFormat",
		Usage:       "Defines the key format. Available options: hex and bech32",
		Value:       "hex",
		Destination: &argsConfig.keyFormat,
	}

	argsConfig = &cfg{}

	initialBalancesSkFileName = "initialBalancesSk.pem"
	initialNodesSkFileName    = "initialNodesSk.pem"

	log = logger.GetOrCreate("keygenerator")
)

func main() {
	app := cli.NewApp()
	cli.AppHelpTemplate = fileGenHelpTemplate
	app.Name = "Key generation Tool"
	app.Version = "v1.0.0"
	app.Usage = "This binary will generate a initialBalancesSk.pem and initialNodesSk.pem, each containing one private key"
	app.Authors = []cli.Author{
		{
			Name:  "The Elrond Team",
			Email: "contact@elrond.com",
		},
	}
	app.Flags = []cli.Flag{
		numKeys,
		keyType,
		keyFormat,
	}

	app.Action = func(_ *cli.Context) error {
		return generateAllFiles()
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Error("error generating files", "error", err)

		os.Exit(1)
	}
}

func generateFolder(index int) (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	absPath := filepath.Join(workingDir, fmt.Sprintf("node-%d", index))

	log.Info("generating files in", "folder", absPath)

	err = os.MkdirAll(absPath, os.ModePerm)
	if err != nil {
		return "", err
	}

	return absPath, nil
}

func generateKeys(keyGen crypto.KeyGenerator) ([]byte, []byte, error) {
	sk, pk := keyGen.GeneratePair()
	skBytes, err := sk.ToByteArray()
	if err != nil {
		return nil, nil, err
	}

	pkBytes, err := pk.ToByteArray()
	if err != nil {
		return nil, nil, err
	}

	return skBytes, pkBytes, nil
}

func backupFileIfExists(filename string) {
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return
		}
	}
	//if we reached here the file probably exists, make a timestamped backup
	_ = os.Rename(filename, filename+"."+fmt.Sprintf("%d", time.Now().Unix()))
}

func generateAllFiles() error {
	for i := 0; i < argsConfig.numKeys; i++ {
		err := generateOneSetOfFiles(i)
		if err != nil {
			return err
		}
	}

	return nil
}

func generateOneSetOfFiles(index int) error {
	switch argsConfig.keyType {
	case "block":
		return generateBlockKey(index)
	case "tx":
		return generateTxKey(index)
	case "both":
		err := generateBlockKey(index)
		if err != nil {
			return err
		}

		return generateTxKey(index)
	default:
		return fmt.Errorf("unknown key type %s", argsConfig.keyType)
	}
}

func generateBlockKey(index int) error {
	pubkeyConverter, err := factory.NewPubkeyConverter(
		config.PubkeyConfig{
			Length: 96,
			Type:   argsConfig.keyFormat,
		},
	)
	if err != nil {
		return err
	}

	genForBlockSigningSk := signing.NewKeyGenerator(mcl.NewSuiteBLS12())

	return generateAndSave(index, initialNodesSkFileName, genForBlockSigningSk, pubkeyConverter)
}

func generateTxKey(index int) error {
	pubkeyConverter, err := factory.NewPubkeyConverter(
		config.PubkeyConfig{
			Length: 32,
			Type:   argsConfig.keyFormat,
		},
	)
	if err != nil {
		return err
	}

	genForBlockSigningSk := signing.NewKeyGenerator(ed25519.NewEd25519())

	return generateAndSave(index, initialBalancesSkFileName, genForBlockSigningSk, pubkeyConverter)
}

func generateAndSave(index int, baseFilename string, genForBlockSigningSk crypto.KeyGenerator, pubkeyConverter state.PubkeyConverter) error {
	folder, err := generateFolder(index)
	if err != nil {
		return err
	}

	filename := filepath.Join(folder, baseFilename)
	backupFileIfExists(filename)

	err = os.Remove(filename)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, core.FileModeUserReadWrite)
	defer func() {
		_ = file.Close()
	}()

	sk, pk, err := generateKeys(genForBlockSigningSk)

	pkString, err := pubkeyConverter.String(pk)
	if err != nil {
		return err
	}

	return core.SaveSkToPemFile(file, pkString, []byte(hex.EncodeToString(sk)))
}
