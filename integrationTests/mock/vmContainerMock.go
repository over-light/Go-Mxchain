package mock

import vmcommon "github.com/ElrondNetwork/elrond-vm-common"

// VMContainerMock -
type VMContainerMock struct {
	GetCalled         func(key []byte) (vmcommon.VMExecutionHandler, error)
	AddCalled         func(key []byte, val vmcommon.VMExecutionHandler) error
	AddMultipleCalled func(keys [][]byte, preprocessors []vmcommon.VMExecutionHandler) error
	ReplaceCalled     func(key []byte, val vmcommon.VMExecutionHandler) error
	RemoveCalled      func(key []byte)
	LenCalled         func() int
	KeysCalled        func() [][]byte
}

// Get -
func (V *VMContainerMock) Get(key []byte) (vmcommon.VMExecutionHandler, error) {
	if V.GetCalled == nil {
		return &VMExecutionHandlerStub{}, nil
	}
	return V.GetCalled(key)
}

// Add -
func (V *VMContainerMock) Add(key []byte, val vmcommon.VMExecutionHandler) error {
	if V.AddCalled == nil {
		return nil
	}
	return V.AddCalled(key, val)
}

// AddMultiple -
func (V *VMContainerMock) AddMultiple(keys [][]byte, vms []vmcommon.VMExecutionHandler) error {
	if V.AddMultipleCalled == nil {
		return nil
	}
	return V.AddMultipleCalled(keys, vms)
}

// Replace -
func (V *VMContainerMock) Replace(key []byte, val vmcommon.VMExecutionHandler) error {
	if V.ReplaceCalled == nil {
		return nil
	}
	return V.ReplaceCalled(key, val)
}

// Remove -
func (V *VMContainerMock) Remove(key []byte) {
	if V.RemoveCalled == nil {
		return
	}
	V.RemoveCalled(key)
}

// Len -
func (V *VMContainerMock) Len() int {
	if V.LenCalled == nil {
		return 0
	}
	return V.LenCalled()
}

// Keys -
func (V *VMContainerMock) Keys() [][]byte {
	if V.KeysCalled == nil {
		return nil
	}
	return V.KeysCalled()
}

// IsInterfaceNil returns true if there is no value under the interface
func (V *VMContainerMock) IsInterfaceNil() bool {
	return V == nil
}
