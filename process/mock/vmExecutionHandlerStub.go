package mock

import (
	"github.com/ElrondNetwork/elrond-vm-common"
	"math/big"
)

type VMExecutionHandlerStub struct {
	RunSmartContractCreateCalled func(input *vmcommon.ContractCreateInput) (*vmcommon.VMOutput, error)
	RunSmartContractCallCalled   func(input *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error)
}

// RunSmartContractCreate computes how a smart contract creation should be performed
func (vm *VMExecutionHandlerStub) RunSmartContractCreate(input *vmcommon.ContractCreateInput) (*vmcommon.VMOutput, error) {
	if vm.RunSmartContractCreateCalled == nil {
		return &vmcommon.VMOutput{
			GasRefund:    big.NewInt(0),
			GasRemaining: big.NewInt(0),
		}, nil
	}

	return vm.RunSmartContractCreateCalled(input)
}

// RunSmartContractCall Computes the result of a smart contract call and how the system must change after the execution
func (vm *VMExecutionHandlerStub) RunSmartContractCall(input *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error) {
	if vm.RunSmartContractCallCalled == nil {
		return &vmcommon.VMOutput{
			GasRefund:    big.NewInt(0),
			GasRemaining: big.NewInt(0),
		}, nil
	}

	return vm.RunSmartContractCallCalled(input)
}
