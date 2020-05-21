package mock

import "github.com/ElrondNetwork/elrond-go/data"

// HeaderSigVerifierStub -
type HeaderSigVerifierStub struct {
	VerifyRandSeedCaller                   func(header data.HeaderHandler) error
	VerifyRandSeedAndLeaderSignatureCalled func(header data.HeaderHandler) error
	VerifySignatureCalled                  func(header data.HeaderHandler) error
}

// VerifyRandSeed -
func (hsvm *HeaderSigVerifierStub) VerifyRandSeed(header data.HeaderHandler) error {
	if hsvm.VerifyRandSeedCaller != nil {
		return hsvm.VerifyRandSeedCaller(header)
	}

	return nil
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
