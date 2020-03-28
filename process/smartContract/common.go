package smartContract

import (
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/process"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

func findVMByTransaction(container process.VirtualMachinesContainer, tx data.TransactionHandler) (vmcommon.VMExecutionHandler, error) {
	scAddress := tx.GetRcvAddr()
	return findVMByScAddress(container, scAddress)
}

func findVMByScAddress(container process.VirtualMachinesContainer, scAddress []byte) (vmcommon.VMExecutionHandler, error) {
	vmType, err := parseVMTypeFromContractAddress(scAddress)
	if err != nil {
		return nil, err
	}

	vm, err := container.Get(vmType)
	if err != nil {
		return nil, err
	}

	return vm, nil
}

func parseVMTypeFromContractAddress(contractAddress []byte) ([]byte, error) {
	if len(contractAddress) < core.NumInitCharactersForScAddress {
		return nil, vmcommon.ErrInvalidVMType
	}

	startIndex := core.NumInitCharactersForScAddress - vmcommon.VMTypeLen
	endIndex := core.NumInitCharactersForScAddress
	return contractAddress[startIndex:endIndex], nil
}
