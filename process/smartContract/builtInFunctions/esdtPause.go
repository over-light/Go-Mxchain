package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/vm"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var _ process.BuiltinFunction = (*esdtPause)(nil)

type esdtPause struct {
	funcGasCost uint64
	marshalizer marshal.Marshalizer
	keyPrefix   []byte
	pause       bool
}

// NewESDTPauseFunc returns the esdt pause/un-pause built-in function component
func NewESDTPauseFunc(
	funcGasCost uint64,
	marshalizer marshal.Marshalizer,
	pause bool,
) (*esdtPause, error) {
	if check.IfNil(marshalizer) {
		return nil, process.ErrNilMarshalizer
	}

	e := &esdtPause{
		funcGasCost: funcGasCost,
		marshalizer: marshalizer,
		keyPrefix:   []byte(core.ElrondProtectedKeyPrefix + esdtKeyIdentifier),
		pause:       pause,
	}

	return e, nil
}

// ProcessBuiltinFunction resolve ESDT function calls
func (e *esdtPause) ProcessBuiltinFunction(
	_, acntDst state.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, process.ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, process.ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != 2 {
		return nil, process.ErrInvalidArguments
	}
	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	if value.Cmp(zero) <= 0 {
		return nil, process.ErrNegativeValue
	}
	if !bytes.Equal(vmInput.CallerAddr, vm.ESDTSCAddress) {
		return nil, process.ErrAddressIsNotESDTSystemSC
	}
	if check.IfNil(acntDst) {
		return nil, process.ErrNilUserAccount
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	log.Trace(vmInput.Function, "sender", vmInput.CallerAddr, "receiver", vmInput.RecipientAddr, "value", value, "token", esdtTokenKey)

	err := e.togglePause(acntDst, esdtTokenKey)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{}
	return vmOutput, nil
}

func (e *esdtPause) togglePause(_ state.UserAccountHandler, _ []byte) error {

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtPause) IsInterfaceNil() bool {
	return e == nil
}
