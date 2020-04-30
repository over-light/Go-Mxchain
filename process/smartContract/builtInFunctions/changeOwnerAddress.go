package builtInFunctions

import (
	"bytes"
	"fmt"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/process"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var _ process.BuiltinFunction = (*changeOwnerAddress)(nil)

type changeOwnerAddress struct {
	gasCost uint64
}

// NewChangeOwnerAddressFunc create a new change owner built in function
func NewChangeOwnerAddressFunc(gasCost uint64) *changeOwnerAddress {
	return &changeOwnerAddress{gasCost: gasCost}
}

// ProcessBuiltinFunction processes simple protocol built-in function
func (c *changeOwnerAddress) ProcessBuiltinFunction(
	_, acntDst state.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, process.ErrNilVmInput
	}
	if len(vmInput.Arguments) == 0 {
		return nil, process.ErrInvalidArguments
	}
	if check.IfNil(acntDst) {
		return nil, process.ErrNilSCDestAccount
	}

	if !bytes.Equal(vmInput.CallerAddr, acntDst.GetOwnerAddress()) {
		return nil, fmt.Errorf("%w not the owner of the account", process.ErrOperationNotPermitted)
	}
	if len(vmInput.Arguments[0]) != len(acntDst.AddressContainer().Bytes()) {
		return nil, process.ErrInvalidAddressLength
	}
	if vmInput.GasProvided < c.gasCost {
		return nil, process.ErrNotEnoughGas
	}

	err := acntDst.ChangeOwnerAddress(vmInput.CallerAddr, vmInput.Arguments[0])
	if err != nil {
		return nil, err
	}

	return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (c *changeOwnerAddress) IsInterfaceNil() bool {
	return c == nil
}
