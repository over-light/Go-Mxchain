package mock

import (
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/process"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

// ScQueryStub -
type ScQueryStub struct {
	ExecuteQueryCalled      func(query *process.SCQuery) (*vmcommon.VMOutput, error)
	ComputeScCallCostCalled func(tx *transaction.Transaction) (uint64, error)
}

// ExecuteQuery -
func (s *ScQueryStub) ExecuteQuery(query *process.SCQuery) (*vmcommon.VMOutput, error) {
	if s.ExecuteQueryCalled != nil {
		return s.ExecuteQueryCalled(query)
	}
	return &vmcommon.VMOutput{}, nil
}

// ComputeScCallCost --
func (s *ScQueryStub) ComputeScCallCost(tx *transaction.Transaction) (uint64, error) {
	if s.ComputeScCallCostCalled != nil {
		return s.ComputeScCallCostCalled(tx)
	}
	return 100, nil
}

// IsInterfaceNil -
func (s *ScQueryStub) IsInterfaceNil() bool {
	return s == nil
}
