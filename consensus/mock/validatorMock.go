package mock

import "sync"

// ValidatorMock -
type ValidatorMock struct {
	pubKey     []byte
	address    []byte
	mutChances sync.RWMutex
	chances    uint32
}

// NewValidatorMock -
func NewValidatorMock(pubKey []byte, address []byte, chances uint32) *ValidatorMock {
	return &ValidatorMock{pubKey: pubKey, address: address, chances: chances}
}

// PubKey -
func (vm *ValidatorMock) PubKey() []byte {
	return vm.pubKey
}

// Address -
func (vm *ValidatorMock) Address() []byte {
	return vm.address
}

// Chances -
func (vm *ValidatorMock) Chances() uint32 {
	vm.mutChances.RLock()
	defer vm.mutChances.RUnlock()

	return vm.chances
}

// SetChances -
func (vm *ValidatorMock) SetChances(chances uint32) {
	vm.mutChances.Lock()
	vm.chances = chances
	vm.mutChances.Unlock()
}

// IsInterfaceNil returns true if there is no value under the interface
func (vm *ValidatorMock) IsInterfaceNil() bool {
	return vm == nil
}
