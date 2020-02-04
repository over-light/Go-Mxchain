package mock

import (
	"github.com/ElrondNetwork/elrond-go/crypto"
)

// PublicKeyMock -
type PublicKeyMock struct {
}

// PrivateKeyMock -
type PrivateKeyMock struct {
}

// KeyGenMock -
type KeyGenMock struct {
}

//------- PublicKeyMock

// ToByteArray -
func (sspk *PublicKeyMock) ToByteArray() ([]byte, error) {
	return []byte("pubKey"), nil
}

// Suite -
func (sspk *PublicKeyMock) Suite() crypto.Suite {
	return nil
}

// Point -
func (sspk *PublicKeyMock) Point() crypto.Point {
	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (sspk *PublicKeyMock) IsInterfaceNil() bool {
	return sspk == nil
}

//------- PrivateKeyMock

// ToByteArray -
func (sk *PrivateKeyMock) ToByteArray() ([]byte, error) {
	return []byte("privKey"), nil
}

// GeneratePublic -
func (sk *PrivateKeyMock) GeneratePublic() crypto.PublicKey {
	return &PublicKeyMock{}
}

// Suite -
func (sk *PrivateKeyMock) Suite() crypto.Suite {
	return nil
}

// Scalar -
func (sk *PrivateKeyMock) Scalar() crypto.Scalar {
	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (sk *PrivateKeyMock) IsInterfaceNil() bool {
	return sk == nil
}

//------KeyGenMock

// GeneratePair -
func (keyGen *KeyGenMock) GeneratePair() (crypto.PrivateKey, crypto.PublicKey) {
	return &PrivateKeyMock{}, &PublicKeyMock{}
}

// PrivateKeyFromByteArray -
func (keyGen *KeyGenMock) PrivateKeyFromByteArray(b []byte) (crypto.PrivateKey, error) {
	return &PrivateKeyMock{}, nil
}

// PublicKeyFromByteArray -
func (keyGen *KeyGenMock) PublicKeyFromByteArray(b []byte) (crypto.PublicKey, error) {
	return &PublicKeyMock{}, nil
}

// Suite -
func (keyGen *KeyGenMock) Suite() crypto.Suite {
	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (keyGen *KeyGenMock) IsInterfaceNil() bool {
	return keyGen == nil
}
