package intermediate

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"strings"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/genesis"
	"github.com/ElrondNetwork/elrond-go/process"
)

type baseDeploy struct {
	genesis.TxExecutionProcessor
	pubkeyConv     core.PubkeyConverter
	blockchainHook process.BlockChainHookHandler
	emptyAddress   []byte
}

func (dp *baseDeploy) deployForOneAddress(
	sc genesis.InitialSmartContractHandler,
	ownerAddress []byte,
	code string,
	initParams string,
) ([]byte, error) {
	nonce, err := dp.GetNonce(ownerAddress)
	if err != nil {
		return nil, err
	}

	scResultingAddressBytes, err := dp.blockchainHook.NewAddress(
		ownerAddress,
		nonce,
		sc.VmTypeBytes(),
	)
	if err != nil {
		return nil, err
	}

	sc.SetAddressBytes(scResultingAddressBytes)
	sc.SetAddress(dp.pubkeyConv.Encode(scResultingAddressBytes))

	vmType := sc.GetVmType()
	arguments := []string{code, vmType, codeMetadataHexForInitialSC}
	if len(initParams) > 0 {
		arguments = append(arguments, initParams)
	}
	deployTxData := strings.Join(arguments, "@")

	log.Trace("deploying genesis SC",
		"SC owner", sc.GetOwner(),
		"SC ownerAddress", sc.Address(),
		"type", sc.GetType(),
		"VM type", sc.GetVmType(),
		"init params", initParams,
	)

	_, accountExists := dp.AccountExists(scResultingAddressBytes)
	if accountExists {
		return nil, fmt.Errorf("%w for SC ownerAddress %s, owner %s with nonce %d",
			genesis.ErrAccountAlreadyExists,
			sc.Address(),
			ownerAddress,
			nonce,
		)
	}

	err = dp.ExecuteTransaction(
		nonce,
		ownerAddress,
		dp.emptyAddress,
		big.NewInt(0),
		[]byte(deployTxData),
	)
	if err != nil {
		return nil, err
	}

	_, accountExists = dp.AccountExists(scResultingAddressBytes)
	if !accountExists {
		return nil, fmt.Errorf("%w for SC ownerAddress %s, owner %s with nonce %d",
			genesis.ErrAccountNotCreated,
			sc.Address(),
			ownerAddress,
			nonce,
		)
	}

	return scResultingAddressBytes, nil
}

func getSCCodeAsHex(filename string) (string, error) {
	code, err := ioutil.ReadFile(filepath.Clean(filename))
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(code), nil
}
