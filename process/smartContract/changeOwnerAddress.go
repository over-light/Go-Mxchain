package smartContract

import (
	"bytes"
	"math/big"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/process"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

type changeOwnerAddress struct {
	gasCost uint64
}

// ProcessBuiltinFunction processes simple protocol built-in function
func (c *changeOwnerAddress) ProcessBuiltinFunction(
	tx data.TransactionHandler,
	_, acntDst state.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*big.Int, error) {
	if len(vmInput.Arguments) == 0 {
		return nil, process.ErrInvalidArguments
	}
	if check.IfNil(tx) {
		return nil, process.ErrNilTransaction
	}
	if check.IfNil(acntDst) {
		return nil, process.ErrNilSCDestAccount
	}

	if !bytes.Equal(tx.GetSndAddress(), acntDst.GetOwnerAddress()) {
		return nil, process.ErrOperationNotPermitted
	}
	if len(vmInput.Arguments[0]) != len(acntDst.AddressContainer().Bytes()) {
		return nil, process.ErrInvalidAddressLength
	}

	err := acntDst.ChangeOwnerAddress(tx.GetSndAddr(), vmInput.Arguments[0])
	if err != nil {
		return nil, err
	}

	return big.NewInt(0), nil
}

// GasUsed returns the gas used for processing the change
func (c *changeOwnerAddress) GasUsed() uint64 {
	return c.gasCost
}

// IsInterfaceNil returns true if underlying object in nil
func (c *changeOwnerAddress) IsInterfaceNil() bool {
	return c == nil
}
