package mock

import (
	"github.com/ElrondNetwork/elrond-go/data/state"
)

// AddressConverterStub -
type AddressConverterStub struct {
	CreateAddressFromPublicKeyBytesCalled func(pubKey []byte) (state.AddressContainer, error)
	ConvertToHexCalled                    func(addressContainer state.AddressContainer) (string, error)
	CreateAddressFromHexCalled            func(hexAddress string) (state.AddressContainer, error)
	PrepareAddressBytesCalled             func(addressBytes []byte) ([]byte, error)
	AddressLenHandler                     func() int
}

// CreateAddressFromPublicKeyBytes -
func (acs *AddressConverterStub) CreateAddressFromPublicKeyBytes(pubKey []byte) (state.AddressContainer, error) {
	return acs.CreateAddressFromPublicKeyBytesCalled(pubKey)
}

// ConvertToHex -
func (acs *AddressConverterStub) ConvertToHex(addressContainer state.AddressContainer) (string, error) {
	return acs.ConvertToHexCalled(addressContainer)
}

// CreateAddressFromHex -
func (acs *AddressConverterStub) CreateAddressFromHex(hexAddress string) (state.AddressContainer, error) {
	return acs.CreateAddressFromHexCalled(hexAddress)
}

// PrepareAddressBytes -
func (acs *AddressConverterStub) PrepareAddressBytes(addressBytes []byte) ([]byte, error) {
	return acs.PrepareAddressBytesCalled(addressBytes)
}

// AddressLen -
func (acs AddressConverterStub) AddressLen() int {
	if acs.AddressLenHandler != nil {
		return acs.AddressLenHandler()
	}
	return 0
}

// IsInterfaceNil returns true if there is no value under the interface
func (acs *AddressConverterStub) IsInterfaceNil() bool {
	return acs == nil
}
