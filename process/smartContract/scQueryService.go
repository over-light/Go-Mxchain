package smartContract

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/process"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/pkg/errors"
)

var _ process.SCQueryService = (*SCQueryService)(nil)

// SCQueryService can execute Get functions over SC to fetch stored values
type SCQueryService struct {
	vmContainer  process.VirtualMachinesContainer
	economicsFee process.FeeHandler
	mutRunSc     sync.Mutex
}

// NewSCQueryService returns a new instance of SCQueryService
func NewSCQueryService(
	vmContainer process.VirtualMachinesContainer,
	economicsFee process.FeeHandler,
) (*SCQueryService, error) {
	if check.IfNil(vmContainer) {
		return nil, process.ErrNoVM
	}
	if check.IfNil(economicsFee) {
		return nil, process.ErrNilEconomicsFeeHandler
	}

	return &SCQueryService{
		vmContainer:  vmContainer,
		economicsFee: economicsFee,
	}, nil
}

// ExecuteQuery returns the VMOutput resulted upon running the function on the smart contract
func (service *SCQueryService) ExecuteQuery(query *process.SCQuery) (*vmcommon.VMOutput, error) {
	if query.ScAddress == nil {
		return nil, process.ErrNilScAddress
	}
	if len(query.FuncName) == 0 {
		return nil, process.ErrEmptyFunctionName
	}

	service.mutRunSc.Lock()
	defer service.mutRunSc.Unlock()

	return service.executeScCall(query, 0)
}

func (service *SCQueryService) executeScCall(query *process.SCQuery, gasPrice uint64) (*vmcommon.VMOutput, error) {
	vm, err := findVMByScAddress(service.vmContainer, query.ScAddress)
	if err != nil {
		return nil, err
	}

	vmInput := service.createVMCallInput(query, gasPrice)
	vmOutput, err := vm.RunSmartContractCall(vmInput)
	if err != nil {
		return nil, err
	}

	err = service.checkVMOutput(vmOutput)
	if err != nil {
		return nil, err
	}

	return vmOutput, nil
}

func (service *SCQueryService) createVMCallInput(query *process.SCQuery, gasPrice uint64) *vmcommon.ContractCallInput {
	vmInput := vmcommon.VMInput{
		CallerAddr:  query.ScAddress,
		CallValue:   big.NewInt(0),
		GasPrice:    gasPrice,
		GasProvided: service.economicsFee.MaxGasLimitPerBlock(0),
		Arguments:   query.Arguments,
		CallType:    vmcommon.DirectCall,
	}

	vmContractCallInput := &vmcommon.ContractCallInput{
		RecipientAddr: query.ScAddress,
		Function:      query.FuncName,
		VMInput:       vmInput,
	}

	return vmContractCallInput
}

func (service *SCQueryService) checkVMOutput(vmOutput *vmcommon.VMOutput) error {
	if vmOutput.ReturnCode != vmcommon.Ok {
		return errors.New(fmt.Sprintf("error running vm func: code: %d, %s", vmOutput.ReturnCode, vmOutput.ReturnCode))
	}

	return nil
}

// ComputeScCallGasLimit will estimate how many gas a transaction will consume
func (service *SCQueryService) ComputeScCallGasLimit(tx *transaction.Transaction) (uint64, error) {
	argumentParser := vmcommon.NewAtArgumentParser()

	err := argumentParser.ParseData(string(tx.Data))
	if err != nil {
		return 0, err
	}

	function, err := argumentParser.GetFunction()
	if err != nil {
		return 0, err
	}

	arguments, err := argumentParser.GetFunctionArguments()
	if err != nil {
		return 0, err
	}

	query := &process.SCQuery{
		ScAddress: tx.RcvAddr,
		FuncName:  function,
		Arguments: arguments,
	}

	service.mutRunSc.Lock()
	defer service.mutRunSc.Unlock()

	vmOutput, err := service.executeScCall(query, 1)
	if err != nil {
		return 0, err
	}

	gasConsumed := service.economicsFee.MaxGasLimitPerBlock(0) - vmOutput.GasRemaining

	return gasConsumed, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (service *SCQueryService) IsInterfaceNil() bool {
	return service == nil
}
