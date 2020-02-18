package mock

import "github.com/ElrondNetwork/elrond-go/data"

// HeaderSigVerifierStub -
type HeaderSigVerifierStub struct {
	VerifyRandSeedAndLeaderSignatureCalled func(header data.HeaderHandler) error
	VerifySignatureCalled                  func(header data.HeaderHandler) error
}

// VerifyRandSeedAndLeaderSignature -
func (hsvm *HeaderSigVerifierStub) VerifyRandSeedAndLeaderSignature(header data.HeaderHandler) error {
	if hsvm.VerifyRandSeedAndLeaderSignatureCalled != nil {
		return hsvm.VerifyRandSeedAndLeaderSignatureCalled(header)
	}

	return nil
}

// VerifySignature -
func (hsvm *HeaderSigVerifierStub) VerifySignature(header data.HeaderHandler) error {
	if hsvm.VerifySignatureCalled != nil {
		return hsvm.VerifySignatureCalled(header)
	}

	return nil
}

// IsInterfaceNil -
func (hsvm *HeaderSigVerifierStub) IsInterfaceNil() bool {
	return hsvm == nil
}
