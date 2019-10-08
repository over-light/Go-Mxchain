package process

import (
	"bytes"
	"testing"

	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/vm"
	"github.com/ElrondNetwork/elrond-go/vm/mock"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/assert"
)

func TestNewSystemVM_NilSystemEI(t *testing.T) {
	t.Parallel()

	systemVM, err := NewSystemVM(nil, &mock.SystemSCContainerStub{}, factory.SystemVirtualMachine)

	assert.Nil(t, systemVM)
	assert.Equal(t, vm.ErrNilSystemEnvironmentInterface, err)
}

func TestNewSystemVM_NilContainer(t *testing.T) {
	t.Parallel()

	systemVM, err := NewSystemVM(&mock.SystemEIStub{}, nil, factory.SystemVirtualMachine)

	assert.Nil(t, systemVM)
	assert.Equal(t, vm.ErrNilSystemContractsContainer, err)
}

func TestNewSystemVM_NilVMType(t *testing.T) {
	t.Parallel()

	systemVM, err := NewSystemVM(&mock.SystemEIStub{}, &mock.SystemSCContainerStub{}, nil)

	assert.Nil(t, systemVM)
	assert.Equal(t, vm.ErrNilVMType, err)
}

func TestNewSystemVM_Ok(t *testing.T) {
	t.Parallel()

	systemVM, err := NewSystemVM(&mock.SystemEIStub{}, &mock.SystemSCContainerStub{}, factory.SystemVirtualMachine)

	assert.Nil(t, err)
	assert.NotNil(t, systemVM)
}

func TestSystemVM_RunSmartContractCreate(t *testing.T) {
	t.Parallel()

	systemVM, _ := NewSystemVM(&mock.SystemEIStub{}, &mock.SystemSCContainerStub{}, factory.SystemVirtualMachine)

	vmOutput, err := systemVM.RunSmartContractCreate(nil)
	assert.Nil(t, vmOutput)
	assert.Equal(t, vm.ErrCannotCreateNewSystemSmartContract, err)
}

func TestSystemVM_RunSmartContractCallWrongSmartContract(t *testing.T) {
	t.Parallel()

	systemVM, _ := NewSystemVM(&mock.SystemEIStub{}, &mock.SystemSCContainerStub{}, factory.SystemVirtualMachine)

	vmOutput, err := systemVM.RunSmartContractCall(&vmcommon.ContractCallInput{RecipientAddr: []byte("tralala")})
	assert.Nil(t, vmOutput)
	assert.Equal(t, vm.ErrUnknownSystemSmartContract, err)
}

func TestSystemVM_RunSmartContractCall(t *testing.T) {
	t.Parallel()

	sc := &mock.SystemSCStub{ExecuteCalled: func(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
		return vmcommon.Ok
	}}
	scAddress := []byte("tralala")

	container := &mock.SystemSCContainerStub{GetCalled: func(key []byte) (contract vm.SystemSmartContract, e error) {
		if bytes.Equal(scAddress, key) {
			return sc, nil
		}
		return nil, vm.ErrUnknownSystemSmartContract
	}}

	systemVM, _ := NewSystemVM(&mock.SystemEIStub{}, container, factory.SystemVirtualMachine)

	vmOutput, err := systemVM.RunSmartContractCall(&vmcommon.ContractCallInput{RecipientAddr: scAddress})
	assert.Nil(t, err)
	assert.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
}
