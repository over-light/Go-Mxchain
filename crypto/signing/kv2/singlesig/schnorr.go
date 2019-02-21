package singlesig

import (
	"github.com/ElrondNetwork/elrond-go-sandbox/crypto"
	"gopkg.in/dedis/kyber.v2"
	"gopkg.in/dedis/kyber.v2/sign/schnorr"
)

type SchnorrSigner struct {
}

// Sign Signs a message with using a single signature schnorr scheme
func (s *SchnorrSigner) Sign(private crypto.PrivateKey, msg []byte) ([]byte, error) {
	if private == nil {
		return nil, crypto.ErrNilPrivateKey
	}

	scalar := private.Scalar()
	if scalar == nil {
		return nil, crypto.ErrNilPrivateKeyScalar
	}

	kScalar, ok := scalar.GetUnderlyingObj().(kyber.Scalar)

	if !ok {
		return nil, crypto.ErrInvalidPrivateKey
	}

	suite := private.Suite()
	if suite == nil {
		return nil, crypto.ErrNilSuite
	}

	kSuite, ok := suite.GetUnderlyingSuite().(schnorr.Suite)

	if !ok {
		return nil, crypto.ErrInvalidSuite
	}

	return schnorr.Sign(kSuite, kScalar, msg)
}

// Verify verifies a signature using a single signature schnorr scheme
func (s *SchnorrSigner) Verify(public crypto.PublicKey, msg []byte, sig []byte) error {
	if public == nil {
		return crypto.ErrNilPublicKey
	}

	if msg == nil {
		return crypto.ErrNilMessage
	}

	if sig == nil {
		return crypto.ErrNilSignature
	}

	suite := public.Suite()
	if suite == nil {
		return crypto.ErrNilSuite
	}

	kSuite, ok := suite.GetUnderlyingSuite().(schnorr.Suite)

	if !ok {
		return crypto.ErrInvalidSuite
	}

	point := public.Point()
	if point == nil {
		return crypto.ErrNilPublicKeyPoint
	}

	kPoint, ok := point.GetUnderlyingObj().(kyber.Point)

	if !ok {
		return crypto.ErrInvalidPublicKey
	}

	return schnorr.Verify(kSuite, kPoint, msg, sig)
}
