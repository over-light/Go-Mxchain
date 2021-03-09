package builtInFunctions

import (
	"bytes"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/vmcommon"
	"github.com/ElrondNetwork/elrond-go/data/esdt"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
)

var _ process.BuiltinFunction = (*esdtNFTCreate)(nil)

type esdtNFTCreate struct {
	keyPrefix    []byte
	noncePrefix  []byte
	marshalizer  marshal.Marshalizer
	pauseHandler process.ESDTPauseHandler
	rolesHandler process.ESDTRoleHandler
	funcGasCost  uint64
	gasConfig    process.BaseOperationCost
	mutExecution sync.RWMutex
}

// NewESDTNFTCreateFunc returns the esdt nft create built-in function component
func NewESDTNFTCreateFunc(
	funcGasCost uint64,
	gasConfig process.BaseOperationCost,
	marshalizer marshal.Marshalizer,
	pauseHandler process.ESDTPauseHandler,
	rolesHandler process.ESDTRoleHandler,
) (*esdtNFTCreate, error) {
	if check.IfNil(marshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(pauseHandler) {
		return nil, process.ErrNilPauseHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, process.ErrNilRolesHandler
	}

	e := &esdtNFTCreate{
		keyPrefix:    []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier),
		noncePrefix:  []byte(core.ElrondProtectedKeyPrefix + core.ESDTNFTLatestNonceIdentifier),
		marshalizer:  marshalizer,
		pauseHandler: pauseHandler,
		rolesHandler: rolesHandler,
		funcGasCost:  funcGasCost,
		gasConfig:    gasConfig,
		mutExecution: sync.RWMutex{},
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtNFTCreate) SetNewGasConfig(gasCost *process.GasCost) {
	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTTransfer
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT NFT create function call
func (e *esdtNFTCreate) ProcessBuiltinFunction(
	acntSnd, _ state.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkESDTNFTCreateBurnAddInput(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) < 7 {
		return nil, process.ErrInvalidArguments
	}

	tokenID := vmInput.Arguments[0]
	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[0], []byte(core.ESDTRoleNFTCreate))
	if err != nil {
		return nil, err
	}

	nonce, err := e.getLatestNonce(acntSnd, tokenID)
	if err != nil {
		return nil, err
	}

	royalties := uint32(big.NewInt(0).SetBytes(vmInput.Arguments[3]).Uint64())
	if royalties > core.MaxRoyalty {
		return nil, process.ErrInvalidArguments
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	quantity := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	if quantity.Cmp(zero) == 0 {
		return nil, process.ErrInvalidArguments
	}
	if quantity.Cmp(big.NewInt(1)) > 0 {
		err = e.rolesHandler.CheckAllowedToExecute(acntSnd, esdtTokenKey, []byte(core.ESDTRoleNFTAddQuantity))
		if err != nil {
			return nil, err
		}
	}

	nextNonce := nonce + 1
	esdtData := &esdt.ESDigitalToken{
		Type:  uint32(core.NonFungible),
		Value: quantity,
		TokenMetaData: &esdt.MetaData{
			Nonce:      nextNonce,
			Name:       vmInput.Arguments[2],
			Creator:    vmInput.CallerAddr,
			Royalties:  royalties,
			Hash:       vmInput.Arguments[4],
			URIs:       vmInput.Arguments[6:],
			Attributes: vmInput.Arguments[5],
		},
	}

	totalLength := uint64(0)
	for _, arg := range vmInput.Arguments {
		totalLength += uint64(len(arg))
	}
	gasToUse := totalLength*e.gasConfig.StorePerByte + e.funcGasCost
	if vmInput.GasProvided < gasToUse {
		return nil, process.ErrNotEnoughGas
	}

	err = saveESDTNFTToken(acntSnd, esdtTokenKey, esdtData, e.marshalizer, e.pauseHandler)
	if err != nil {
		return nil, err
	}

	err = e.saveLatestNonce(acntSnd, tokenID, nextNonce)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - gasToUse,
	}
	return vmOutput, nil
}

func (e *esdtNFTCreate) getLatestNonce(acnt state.UserAccountHandler, tokenID []byte) (uint64, error) {
	nonceKey := e.getNonceKey(tokenID)
	nonceData, err := acnt.DataTrieTracker().RetrieveValue(nonceKey)
	if err != nil {
		return 0, err
	}

	if len(nonceData) == 0 {
		return 0, nil
	}

	return big.NewInt(0).SetBytes(nonceData).Uint64(), nil
}

func (e *esdtNFTCreate) saveLatestNonce(acnt state.UserAccountHandler, tokenID []byte, nonce uint64) error {
	nonceKey := e.getNonceKey(tokenID)
	return acnt.DataTrieTracker().SaveKeyValue(nonceKey, big.NewInt(0).SetUint64(nonce).Bytes())
}

func computeESDTNFTTokenKey(esdtTokenKey []byte, nonce uint64) []byte {
	return append(esdtTokenKey, big.NewInt(0).SetUint64(nonce).Bytes()...)
}

func getESDTNFTToken(
	acnt state.UserAccountHandler,
	esdtTokenKey []byte,
	nonce uint64,
	marshalizer marshal.Marshalizer,
) (*esdt.ESDigitalToken, error) {
	esdtNFTTokenKey := computeESDTNFTTokenKey(esdtTokenKey, nonce)

	marshaledData, err := acnt.DataTrieTracker().RetrieveValue(esdtNFTTokenKey)
	if err != nil {
		return nil, err
	}
	if len(marshaledData) == 0 {
		return nil, process.ErrNFTTokenDoesNotExist
	}

	esdtData := &esdt.ESDigitalToken{}
	err = marshalizer.Unmarshal(esdtData, marshaledData)
	if err != nil {
		return nil, err
	}

	if esdtData.TokenMetaData == nil {
		return nil, process.ErrNFTDoesNotHaveMetadata
	}

	return esdtData, nil
}

func saveESDTNFTToken(
	acnt state.UserAccountHandler,
	esdtTokenKey []byte,
	esdtData *esdt.ESDigitalToken,
	marshalizer marshal.Marshalizer,
	pauseHandler process.ESDTPauseHandler,
) error {
	if esdtData.TokenMetaData == nil {
		return process.ErrNFTDoesNotHaveMetadata
	}

	err := checkFrozeAndPause(acnt.AddressBytes(), esdtTokenKey, esdtData, pauseHandler)
	if err != nil {
		return err
	}

	nonce := esdtData.TokenMetaData.Nonce
	esdtNFTTokenKey := computeESDTNFTTokenKey(esdtTokenKey, nonce)

	if esdtData.Value.Cmp(zero) <= 0 {
		return acnt.DataTrieTracker().SaveKeyValue(esdtNFTTokenKey, nil)
	}

	marshaledData, err := marshalizer.Marshal(esdtData)
	if err != nil {
		return err
	}

	return acnt.DataTrieTracker().SaveKeyValue(esdtNFTTokenKey, marshaledData)
}

func checkESDTNFTCreateBurnAddInput(
	account state.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
	funcGasCost uint64) error {
	err := checkBasicESDTArguments(vmInput)
	if err != nil {
		return err
	}
	if !bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return process.ErrInvalidRcvAddr
	}
	if check.IfNil(account) {
		return process.ErrNilUserAccount
	}
	if vmInput.GasProvided < funcGasCost {
		return process.ErrNotEnoughGas
	}
	return nil
}

func (e *esdtNFTCreate) getNonceKey(tokenID []byte) []byte {
	return append(e.noncePrefix, tokenID...)
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtNFTCreate) IsInterfaceNil() bool {
	return e == nil
}
