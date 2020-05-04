//go:generate protoc -I=proto -I=$GOPATH/src -I=$GOPATH/src/github.com/gogo/protobuf/protobuf  --gogoslick_out=. esdt.proto
package systemSmartContracts

import (
	"bytes"
	"encoding/hex"
	"math/big"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/vm"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

const minLengthForTokenName = 10
const maxLengthForTokenName = 20
const configKeyPrefix = "esdtConfig"
const burnable = "burnable"
const mintable = "mintable"
const canPause = "canPause"
const canFreeze = "canFreeze"
const canWipe = "canWipe"

const conversionBase = 10

//TODO: think about how should we enable issuing of shorter names - maybe only a special address can do it
// a little bit centralized but it resolves the problem of issuing commonly known token names

type esdt struct {
	eei             vm.SystemEI
	gasCost         vm.GasCost
	baseIssuingCost *big.Int
	ownerAddress    []byte
	eSDTSCAddress   []byte
	marshalizer     marshal.Marshalizer
	hasher          hashing.Hasher
}

// ArgsNewESDTSmartContract defines the arguments needed for the esdt contract
type ArgsNewESDTSmartContract struct {
	Eei           vm.SystemEI
	GasCost       vm.GasCost
	ESDTSCConfig  config.ESDTSystemSCConfig
	ESDTSCAddress []byte
	Marshalizer   marshal.Marshalizer
	Hasher        hashing.Hasher
}

// NewESDTSmartContract creates the esdt smart contract, which controls the issuing of tokens
func NewESDTSmartContract(args ArgsNewESDTSmartContract) (*esdt, error) {
	if check.IfNil(args.Eei) {
		return nil, vm.ErrNilSystemEnvironmentInterface
	}
	if check.IfNil(args.Marshalizer) {
		return nil, vm.ErrNilMarshalizer
	}
	if check.IfNil(args.Hasher) {
		return nil, vm.ErrNilHasher
	}

	baseIssuingCost, ok := big.NewInt(0).SetString(args.ESDTSCConfig.BaseIssuingCost, conversionBase)
	if !ok || baseIssuingCost.Cmp(big.NewInt(0)) < 0 {
		return nil, vm.ErrInvalidBaseIssuingCost
	}

	return &esdt{
		eei:             args.Eei,
		gasCost:         args.GasCost,
		baseIssuingCost: baseIssuingCost,
		ownerAddress:    []byte(args.ESDTSCConfig.OwnerAddress),
		eSDTSCAddress:   args.ESDTSCAddress,
		hasher:          args.Hasher,
		marshalizer:     args.Marshalizer,
	}, nil
}

// Execute calls one of the functions from the esdt smart contract and runs the code according to the input
func (e *esdt) Execute(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if CheckIfNil(args) != nil {
		return vmcommon.UserError
	}

	switch args.Function {
	case core.SCDeployInitFunctionName:
		return e.init(args)
	case "issue":
		return e.issue(args)
	case "issueProtected":
		return e.issueProtected(args)
	case "burn":
		return e.burn(args)
	case "mint":
		return e.mint(args)
	case "freeze":
		return e.freeze(args)
	case "wipe":
		return e.wipe(args)
	case "pause":
		return e.pause(args)
	case "unPause":
		return e.unpause(args)
	case "claim":
		return e.claim(args)
	case "configChange":
		return e.configChange(args)
	case "esdtControlChanges":
		return e.esdtControlChanges(args)
	}

	return vmcommon.Ok
}

func (e *esdt) init(_ *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	scConfig := &ESDTConfig{
		OwnerAddress:       e.ownerAddress,
		BaseIssuingCost:    e.baseIssuingCost,
		MinTokenNameLength: minLengthForTokenName,
		MaxTokenNameLength: maxLengthForTokenName,
	}
	marshaledData, err := e.marshalizer.Marshal(scConfig)
	log.LogIfError(err, "marshal error on esdt init function")

	e.eei.SetStorage([]byte(configKeyPrefix), marshaledData)
	return vmcommon.Ok
}

func (e *esdt) issueProtected(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if !bytes.Equal(args.CallerAddr, e.ownerAddress) {
		return vmcommon.UserError
	}
	if len(args.Arguments) < 3 {
		return vmcommon.FunctionWrongSignature
	}
	if len(args.Arguments[0]) < len(args.CallerAddr) {
		return vmcommon.FunctionWrongSignature
	}
	if args.CallValue.Cmp(e.baseIssuingCost) != 0 {
		return vmcommon.OutOfFunds
	}
	err := e.eei.UseGas(e.gasCost.MetaChainSystemSCsCost.ESDTIssue)
	if err != nil {
		return vmcommon.OutOfGas
	}

	err = e.issueToken(args.Arguments[0], args.Arguments[1:])
	if err != nil {
		return vmcommon.UserError
	}

	return vmcommon.Ok
}

func (e *esdt) issue(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if len(args.Arguments) < 2 {
		return vmcommon.FunctionWrongSignature
	}
	if len(args.Arguments[0]) < minLengthForTokenName || len(args.Arguments[0]) > maxLengthForTokenName {
		return vmcommon.FunctionWrongSignature
	}
	if args.CallValue.Cmp(e.baseIssuingCost) != 0 {
		return vmcommon.OutOfFunds
	}
	err := e.eei.UseGas(e.gasCost.MetaChainSystemSCsCost.ESDTIssue)
	if err != nil {
		return vmcommon.OutOfGas
	}

	err = e.issueToken(args.CallerAddr, args.Arguments)
	if err != nil {
		return vmcommon.UserError
	}

	return vmcommon.Ok
}

func (e *esdt) issueToken(owner []byte, arguments [][]byte) error {
	tokenName := arguments[0]
	initialSupply := big.NewInt(0).SetBytes(arguments[1])
	if initialSupply.Cmp(big.NewInt(0)) < 0 {
		return vm.ErrNegativeInitialSupply
	}

	data := e.eei.GetStorage(tokenName)
	if len(data) > 0 {
		return vm.ErrTokenAlreadyRegistered
	}

	newESDTToken := ESDTData{
		IssuerAddress: owner,
		TokenName:     tokenName,
		Mintable:      false,
		Burnable:      false,
		CanPause:      false,
		Paused:        false,
		CanFreeze:     false,
		CanWipe:       false,
		MintedValue:   initialSupply,
		BurntValue:    big.NewInt(0),
	}
	for i := 2; i < len(arguments); i++ {
		optionalArg := string(arguments[i])
		switch optionalArg {
		case burnable:
			newESDTToken.Burnable = true
		case mintable:
			newESDTToken.Mintable = true
		case canPause:
			newESDTToken.CanPause = true
		case canFreeze:
			newESDTToken.CanFreeze = true
		case canWipe:
			newESDTToken.CanWipe = true
		}
	}

	marshalledData, err := e.marshalizer.Marshal(newESDTToken)
	if err != nil {
		return err
	}

	e.eei.SetStorage(tokenName, marshalledData)

	esdtTransferData := core.BuiltInFunctionESDTTransfer + "@" + hex.EncodeToString(initialSupply.Bytes())
	err = e.eei.Transfer(owner, e.eSDTSCAddress, big.NewInt(0), []byte(esdtTransferData))
	if err != nil {
		return err
	}

	return nil
}

func (e *esdt) burn(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	//TODO: implement me
	return vmcommon.Ok
}

func (e *esdt) mint(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	//TODO: implement me
	return vmcommon.Ok
}

func (e *esdt) freeze(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	//TODO: implement me
	return vmcommon.Ok
}

func (e *esdt) wipe(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	//TODO: implement me
	return vmcommon.Ok
}

func (e *esdt) pause(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	//TODO: implement me
	return vmcommon.Ok
}

func (e *esdt) unpause(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	//TODO: implement me
	return vmcommon.Ok
}

func (e *esdt) configChange(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	//TODO: implement me
	return vmcommon.Ok
}

func (e *esdt) claim(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	//TODO: implement me
	return vmcommon.Ok
}

func (e *esdt) esdtControlChanges(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	//TODO: implement me
	return vmcommon.Ok
}

// IsInterfaceNil returns true if underlying object is nil
func (e *esdt) IsInterfaceNil() bool {
	return e == nil
}
