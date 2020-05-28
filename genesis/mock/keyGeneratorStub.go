package mock

import "github.com/ElrondNetwork/elrond-go/crypto"

// KeyGeneratorStub -
type KeyGeneratorStub struct {
	GeneratePairCalled            func() (crypto.PrivateKey, crypto.PublicKey)
	PrivateKeyFromByteArrayCalled func(b []byte) (crypto.PrivateKey, error)
	PublicKeyFromByteArrayCalled  func(b []byte) (crypto.PublicKey, error)
	SuiteCalled                   func() crypto.Suite
}

// GeneratePair -
func (kgs *KeyGeneratorStub) GeneratePair() (crypto.PrivateKey, crypto.PublicKey) {
	if kgs.GeneratePairCalled != nil {
		return kgs.GeneratePairCalled()
	}

	return nil, nil
}

// PrivateKeyFromByteArray -
func (kgs *KeyGeneratorStub) PrivateKeyFromByteArray(b []byte) (crypto.PrivateKey, error) {
	if kgs.PrivateKeyFromByteArrayCalled != nil {
		return kgs.PrivateKeyFromByteArrayCalled(b)
	}

	return nil, nil
}

// PublicKeyFromByteArray -
func (kgs *KeyGeneratorStub) PublicKeyFromByteArray(b []byte) (crypto.PublicKey, error) {
	if kgs.PublicKeyFromByteArrayCalled != nil {
		return kgs.PublicKeyFromByteArrayCalled(b)
	}

	return nil, nil
}

// Suite -
func (kgs *KeyGeneratorStub) Suite() crypto.Suite {
	if kgs.SuiteCalled != nil {
		return kgs.SuiteCalled()
	}

	return nil
}

// IsInterfaceNil -
func (kgs *KeyGeneratorStub) IsInterfaceNil() bool {
	return kgs == nil
}
