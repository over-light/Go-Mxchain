package mock

import (
	vmcommon "github.com/ElrondNetwork/elrond-go/core/vm-common"
	"github.com/ElrondNetwork/elrond-go/vm"
)

// SystemSCStub -
type SystemSCStub struct {
	ExecuteCalled       func(args *vmcommon.ContractCallInput) vmcommon.ReturnCode
	SetNewGasCostCalled func(gasCost vm.GasCost)
}

// SetNewGasCost -
func (s *SystemSCStub) SetNewGasCost(gasCost vm.GasCost) {
	if s.SetNewGasCostCalled != nil {
		s.SetNewGasCostCalled(gasCost)
	}
}

// Execute -
func (s *SystemSCStub) Execute(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if s.ExecuteCalled != nil {
		return s.ExecuteCalled(args)
	}
	return 0
}

// IsInterfaceNil -
func (s *SystemSCStub) IsInterfaceNil() bool {
	return s == nil
}
