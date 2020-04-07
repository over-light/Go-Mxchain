package disabled

import (
	"github.com/ElrondNetwork/elrond-go/data"
)

type headerSigVerifier struct {
}

// NewHeaderSigVerifier returns a new instance of headerSigVerifier
func NewHeaderSigVerifier() *headerSigVerifier {
	return &headerSigVerifier{}
}

// VerifyRandSeedAndLeaderSignature -
func (h *headerSigVerifier) VerifyRandSeedAndLeaderSignature(_ data.HeaderHandler) error {
	return nil
}

// VerifySignature -
func (h *headerSigVerifier) VerifySignature(_ data.HeaderHandler) error {
	return nil
}

// IsInterfaceNil -
func (h *headerSigVerifier) IsInterfaceNil() bool {
	return h == nil
}
