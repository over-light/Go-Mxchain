package singlesig_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/crypto/mock"
	"github.com/ElrondNetwork/elrond-go/crypto/signing"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/kyber"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/kyber/singlesig"
	"github.com/stretchr/testify/assert"
)

func TestSchnorrSigner_SignNilPrivateKeyShouldErr(t *testing.T) {
	t.Parallel()

	msg := []byte("message to be signed")
	signer := &singlesig.SchnorrSigner{}
	signature, err := signer.Sign(nil, msg)

	assert.Nil(t, signature)
	assert.Equal(t, crypto.ErrNilPrivateKey, err)
}

func TestSchnorrSigner_SignPrivateKeyNilSuiteShouldErr(t *testing.T) {
	t.Parallel()

	suite := kyber.NewBlakeSHA256Ed25519()
	kg := signing.NewKeyGenerator(suite)
	privKey, _ := kg.GeneratePair()

	privKeyNilSuite := &mock.PrivateKeyStub{
		SuiteStub: func() crypto.Suite {
			return nil
		},
		ToByteArrayStub:    privKey.ToByteArray,
		ScalarStub:         privKey.Scalar,
		GeneratePublicStub: privKey.GeneratePublic,
	}

	msg := []byte("message to be signed")
	signer := &singlesig.SchnorrSigner{}
	signature, err := signer.Sign(privKeyNilSuite, msg)

	assert.Nil(t, signature)
	assert.Equal(t, crypto.ErrNilSuite, err)
}

func TestSchnorrSigner_SignPrivateKeyNilScalarShouldErr(t *testing.T) {
	t.Parallel()

	suite := kyber.NewBlakeSHA256Ed25519()
	kg := signing.NewKeyGenerator(suite)
	privKey, _ := kg.GeneratePair()

	privKeyNilSuite := &mock.PrivateKeyStub{
		SuiteStub:       privKey.Suite,
		ToByteArrayStub: privKey.ToByteArray,
		ScalarStub: func() crypto.Scalar {
			return nil
		},
		GeneratePublicStub: privKey.GeneratePublic,
	}

	msg := []byte("message to be signed")
	signer := &singlesig.SchnorrSigner{}
	signature, err := signer.Sign(privKeyNilSuite, msg)

	assert.Nil(t, signature)
	assert.Equal(t, crypto.ErrNilPrivateKeyScalar, err)
}

func TestSchnorrSigner_SignInvalidScalarShouldErr(t *testing.T) {
	t.Parallel()

	suite := kyber.NewBlakeSHA256Ed25519()
	kg := signing.NewKeyGenerator(suite)
	privKey, _ := kg.GeneratePair()

	privKeyNilSuite := &mock.PrivateKeyStub{
		SuiteStub:       privKey.Suite,
		ToByteArrayStub: privKey.ToByteArray,
		ScalarStub: func() crypto.Scalar {
			return &mock.ScalarMock{}
		},
		GeneratePublicStub: privKey.GeneratePublic,
	}

	msg := []byte("message to be signed")
	signer := &singlesig.SchnorrSigner{}
	signature, err := signer.Sign(privKeyNilSuite, msg)

	assert.Nil(t, signature)
	assert.Equal(t, crypto.ErrInvalidPrivateKey, err)
}

func TestSchnorrSigner_SignInvalidSuiteShouldErr(t *testing.T) {
	t.Parallel()

	suite := kyber.NewBlakeSHA256Ed25519()
	kg := signing.NewKeyGenerator(suite)
	privKey, _ := kg.GeneratePair()

	invalidSuite := &mock.SuiteMock{
		GetUnderlyingSuiteStub: func() interface{} {
			return 0
		},
	}

	privKeyNilSuite := &mock.PrivateKeyStub{
		SuiteStub: func() crypto.Suite {
			return invalidSuite
		},
		ToByteArrayStub:    privKey.ToByteArray,
		ScalarStub:         privKey.Scalar,
		GeneratePublicStub: privKey.GeneratePublic,
	}

	msg := []byte("message to be signed")
	signer := &singlesig.SchnorrSigner{}

	signature, err := signer.Sign(privKeyNilSuite, msg)

	assert.Nil(t, signature)
	assert.Equal(t, crypto.ErrInvalidSuite, err)
}

func signSchnorr(msg []byte, signer crypto.SingleSigner, t *testing.T) (
	pubKey crypto.PublicKey,
	privKey crypto.PrivateKey,
	signature []byte,
	err error) {

	suite := kyber.NewBlakeSHA256Ed25519()
	kg := signing.NewKeyGenerator(suite)
	privKey, pubKey = kg.GeneratePair()

	signature, err = signer.Sign(privKey, msg)

	assert.NotNil(t, signature)
	assert.Nil(t, err)

	return pubKey, privKey, signature, err
}

func TestSchnorrSigner_SignOK(t *testing.T) {
	t.Parallel()

	msg := []byte("message to be signed")
	signer := &singlesig.SchnorrSigner{}
	pubKey, _, signature, _ := signSchnorr(msg, signer, t)

	err := signer.Verify(pubKey, msg, signature)

	assert.Nil(t, err)
}

func TestSchnorrSigner_VerifyNilSuiteShouldErr(t *testing.T) {
	t.Parallel()

	msg := []byte("message to be signed")
	signer := &singlesig.SchnorrSigner{}
	pubKey, _, signature, _ := signSchnorr(msg, signer, t)

	pubKeyNilSuite := &mock.PublicKeyStub{
		SuiteStub: func() crypto.Suite {
			return nil
		},
		ToByteArrayStub: pubKey.ToByteArray,
		PointStub:       pubKey.Point,
	}

	err := signer.Verify(pubKeyNilSuite, msg, signature)

	assert.Equal(t, crypto.ErrNilSuite, err)
}

func TestSchnorrSigner_VerifyNilPublicKeyShouldErr(t *testing.T) {
	t.Parallel()

	msg := []byte("message to be signed")
	signer := &singlesig.SchnorrSigner{}
	_, _, signature, _ := signSchnorr(msg, signer, t)

	err := signer.Verify(nil, msg, signature)

	assert.Equal(t, crypto.ErrNilPublicKey, err)
}

func TestSchnorrSigner_VerifyNilMessageShouldErr(t *testing.T) {
	t.Parallel()

	msg := []byte("message to be signed")
	signer := &singlesig.SchnorrSigner{}
	pubKey, _, signature, _ := signSchnorr(msg, signer, t)

	err := signer.Verify(pubKey, nil, signature)

	assert.Equal(t, crypto.ErrNilMessage, err)
}

func TestSchnorrSigner_VerifyNilSignatureShouldErr(t *testing.T) {
	t.Parallel()

	msg := []byte("message to be signed")
	signer := &singlesig.SchnorrSigner{}
	pubKey, _, _, _ := signSchnorr(msg, signer, t)

	err := signer.Verify(pubKey, msg, nil)

	assert.Equal(t, crypto.ErrNilSignature, err)
}

func TestSchnorrSigner_VerifyInvalidSuiteShouldErr(t *testing.T) {
	t.Parallel()

	msg := []byte("message to be signed")
	signer := &singlesig.SchnorrSigner{}
	pubKey, _, signature, _ := signSchnorr(msg, signer, t)

	invalidSuite := &mock.SuiteMock{
		GetUnderlyingSuiteStub: func() interface{} {
			return 0
		},
	}

	pubKeyInvalidSuite := &mock.PublicKeyStub{
		SuiteStub: func() crypto.Suite {
			return invalidSuite
		},
		ToByteArrayStub: pubKey.ToByteArray,
		PointStub:       pubKey.Point,
	}

	err := signer.Verify(pubKeyInvalidSuite, msg, signature)

	assert.Equal(t, crypto.ErrInvalidSuite, err)
}

func TestSchnorrSigner_VerifyPublicKeyInvalidPointShouldErr(t *testing.T) {
	t.Parallel()

	msg := []byte("message to be signed")
	signer := &singlesig.SchnorrSigner{}
	pubKey, _, signature, _ := signSchnorr(msg, signer, t)

	pubKeyInvalidSuite := &mock.PublicKeyStub{
		SuiteStub:       pubKey.Suite,
		ToByteArrayStub: pubKey.ToByteArray,
		PointStub: func() crypto.Point {
			return nil
		},
	}

	err := signer.Verify(pubKeyInvalidSuite, msg, signature)

	assert.Equal(t, crypto.ErrNilPublicKeyPoint, err)
}

func TestSchnorrSigner_VerifyInvalidPublicKeyShouldErr(t *testing.T) {
	t.Parallel()

	msg := []byte("message to be signed")
	signer := &singlesig.SchnorrSigner{}
	pubKey, _, signature, _ := signSchnorr(msg, signer, t)

	pubKeyInvalidSuite := &mock.PublicKeyStub{
		SuiteStub:       pubKey.Suite,
		ToByteArrayStub: pubKey.ToByteArray,
		PointStub: func() crypto.Point {
			return &mock.PointMock{}
		},
	}

	err := signer.Verify(pubKeyInvalidSuite, msg, signature)

	assert.Equal(t, crypto.ErrInvalidPublicKey, err)
}

func TestSchnorrSigner_VerifyOK(t *testing.T) {
	t.Parallel()

	msg := []byte("message to be signed")
	signer := &singlesig.SchnorrSigner{}
	pubKey, _, signature, _ := signSchnorr(msg, signer, t)

	err := signer.Verify(pubKey, msg, signature)

	assert.Nil(t, err)
}
