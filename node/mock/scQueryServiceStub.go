package mock

import (
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/process"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

// SCQueryServiceStub -
type SCQueryServiceStub struct {
	ExecuteQueryCalled       func(*process.SCQuery) (*vmcommon.VMOutput, error)
	ComputeScCallCostHandler func(tx *transaction.Transaction) (uint64, error)
}

// ExecuteQuery -
func (serviceStub *SCQueryServiceStub) ExecuteQuery(query *process.SCQuery) (*vmcommon.VMOutput, error) {
	return serviceStub.ExecuteQueryCalled(query)
}

// ComputeScCallCost -
func (serviceStub *SCQueryServiceStub) ComputeScCallCost(tx *transaction.Transaction) (uint64, error) {
	return serviceStub.ComputeScCallCostHandler(tx)
}

// IsInterfaceNil returns true if there is no value under the interface
func (serviceStub *SCQueryServiceStub) IsInterfaceNil() bool {
	return serviceStub == nil
}
