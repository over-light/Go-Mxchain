package mock

import (
	"encoding/hex"
	"errors"
	"strings"

	"github.com/ElrondNetwork/elrond-go/data/state"
)

// AddressConverterFake -
type AddressConverterFake struct {
	addressLen int
	prefix     string
}

// NewAddressConverterFake -
func NewAddressConverterFake(addressLen int, prefix string) *AddressConverterFake {
	return &AddressConverterFake{
		addressLen: addressLen,
		prefix:     prefix,
	}
}

// CreateAddressFromPublicKeyBytes -
func (acf *AddressConverterFake) CreateAddressFromPublicKeyBytes(pubKey []byte) (state.AddressContainer, error) {
	newPubKey := make([]byte, len(pubKey))
	copy(newPubKey, pubKey)

	if len(newPubKey) > acf.addressLen {
		newPubKey = newPubKey[len(newPubKey)-acf.addressLen:]
	}

	return state.NewAddress(newPubKey), nil
}

// ConvertToHex -
func (acf *AddressConverterFake) ConvertToHex(addressContainer state.AddressContainer) (string, error) {
	return acf.prefix + hex.EncodeToString(addressContainer.Bytes()), nil
}

// CreateAddressFromHex -
func (acf *AddressConverterFake) CreateAddressFromHex(hexAddress string) (state.AddressContainer, error) {
	hexAddress = strings.ToLower(hexAddress)

	if strings.HasPrefix(hexAddress, strings.ToLower(acf.prefix)) {
		hexAddress = hexAddress[len(acf.prefix):]
	}

	hexValsInByte := 2
	if len(hexAddress) != acf.addressLen*hexValsInByte {
		return nil, errors.New("wrong size")
	}

	buff := make([]byte, acf.addressLen)
	_, err := hex.Decode(buff, []byte(hexAddress))

	if err != nil {
		return nil, err
	}

	return state.NewAddress(buff), nil
}

// PrepareAddressBytes -
func (acf *AddressConverterFake) PrepareAddressBytes(addressBytes []byte) ([]byte, error) {
	return addressBytes, nil
}

// AddressLen -
func (acf *AddressConverterFake) AddressLen() int {
	return acf.addressLen
}

// IsInterfaceNil returns true if there is no value under the interface
func (acf *AddressConverterFake) IsInterfaceNil() bool {
	if acf == nil {
		return true
	}
	return false
}
