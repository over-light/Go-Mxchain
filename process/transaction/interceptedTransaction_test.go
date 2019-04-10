package transaction_test

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-sandbox/crypto"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/state"
	"github.com/ElrondNetwork/elrond-go-sandbox/process"
	"github.com/ElrondNetwork/elrond-go-sandbox/process/mock"
	"github.com/ElrondNetwork/elrond-go-sandbox/process/transaction"
	"github.com/stretchr/testify/assert"
)

//------- Integrity()

func TestInterceptedTransaction_IntegrityNilTransactionShouldErr(t *testing.T) {
	t.Parallel()

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)

	tx.Transaction = nil
	assert.Equal(t, process.ErrNilTransaction, tx.Integrity(nil))
}

func TestInterceptedTransaction_IntegrityNilSignatureShouldErr(t *testing.T) {
	t.Parallel()

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)

	tx.Signature = nil
	tx.Challenge = make([]byte, 0)
	tx.RcvAddr = make([]byte, 0)
	tx.SndAddr = make([]byte, 0)
	tx.Value = big.NewInt(1)

	assert.Equal(t, process.ErrNilSignature, tx.Integrity(nil))
}

func TestInterceptedTransaction_IntegrityNilRcvAddrShouldErr(t *testing.T) {
	t.Parallel()

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)

	tx.Signature = make([]byte, 0)
	tx.Challenge = make([]byte, 0)
	tx.RcvAddr = nil
	tx.SndAddr = make([]byte, 0)
	tx.Value = big.NewInt(1)

	assert.Equal(t, process.ErrNilRcvAddr, tx.Integrity(nil))
}

func TestInterceptedTransaction_IntegrityNilSndAddrShouldErr(t *testing.T) {
	t.Parallel()

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)

	tx.Signature = make([]byte, 0)
	tx.Challenge = make([]byte, 0)
	tx.RcvAddr = make([]byte, 0)
	tx.SndAddr = nil
	tx.Value = big.NewInt(1)

	assert.Equal(t, process.ErrNilSndAddr, tx.Integrity(nil))
}

func TestInterceptedTransaction_IntegrityNegativeValueShouldErr(t *testing.T) {
	t.Parallel()

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)

	tx.Signature = make([]byte, 0)
	tx.Challenge = make([]byte, 0)
	tx.RcvAddr = make([]byte, 0)
	tx.SndAddr = make([]byte, 0)
	tx.Value = big.NewInt(-1)

	assert.Equal(t, process.ErrNegativeValue, tx.Integrity(nil))
}

func TestInterceptedTransaction_IntegrityOkValsShouldWork(t *testing.T) {
	t.Parallel()

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)

	tx.Signature = make([]byte, 0)
	tx.Challenge = make([]byte, 0)
	tx.RcvAddr = make([]byte, 0)
	tx.SndAddr = make([]byte, 0)
	tx.Value = big.NewInt(0)

	assert.Nil(t, tx.Integrity(nil))
}

//------- IntegrityAndValidity()

func TestInterceptedTransaction_IntegrityAndValidityNilTransactionShouldErr(t *testing.T) {
	t.Parallel()

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)

	tx.Transaction = nil
	assert.Equal(t, process.ErrNilShardCoordinator, tx.IntegrityAndValidity(nil))
}

func TestInterceptedTransaction_IntegrityAndValidityIntegrityFailsShouldErr(t *testing.T) {
	t.Parallel()

	oneSharder := mock.NewOneShardCoordinatorMock()

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)

	tx.Transaction = nil
	assert.Equal(t, process.ErrNilTransaction, tx.IntegrityAndValidity(oneSharder))
}

func TestInterceptedTransaction_IntegrityAndValidityNilAddrConverterShouldErr(t *testing.T) {
	t.Parallel()

	oneSharder := mock.NewOneShardCoordinatorMock()

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)

	tx.Signature = make([]byte, 0)
	tx.Challenge = make([]byte, 0)
	tx.RcvAddr = make([]byte, 0)
	tx.SndAddr = make([]byte, 0)
	tx.Value = big.NewInt(1)

	assert.Equal(t, process.ErrNilAddressConverter, tx.IntegrityAndValidity(oneSharder))
}

func TestTransactionInterceptor_IntegrityAndValidityInvalidSenderAddrShouldRetFalse(t *testing.T) {
	t.Parallel()

	oneSharder := mock.NewOneShardCoordinatorMock()
	signer := &mock.SignerMock{}

	tx := transaction.NewInterceptedTransaction(signer)
	tx.Signature = make([]byte, 0)
	tx.Challenge = make([]byte, 0)
	tx.RcvAddr = make([]byte, 0)
	tx.SndAddr = []byte("please fail, addrConverter!")
	tx.Value = big.NewInt(0)

	addrConv := &mock.AddressConverterMock{}
	addrConv.CreateAddressFromPublicKeyBytesRetErrForValue = []byte("please fail, addrConverter!")
	tx.SetAddressConverter(addrConv)

	assert.Equal(t, process.ErrInvalidSndAddr, tx.IntegrityAndValidity(oneSharder))
}

func TestTransactionInterceptor_IntegrityAndValidityInvalidReceiverAddrShouldRetFalse(t *testing.T) {
	t.Parallel()

	oneSharder := mock.NewOneShardCoordinatorMock()
	signer := &mock.SignerMock{}

	tx := transaction.NewInterceptedTransaction(signer)
	tx.Signature = make([]byte, 0)
	tx.Challenge = make([]byte, 0)
	tx.RcvAddr = []byte("please fail, addrConverter!")
	tx.SndAddr = make([]byte, 0)
	tx.Value = big.NewInt(0)

	addrConv := &mock.AddressConverterMock{}
	addrConv.CreateAddressFromPublicKeyBytesRetErrForValue = []byte("please fail, addrConverter!")
	tx.SetAddressConverter(addrConv)

	assert.Equal(t, process.ErrInvalidRcvAddr, tx.IntegrityAndValidity(oneSharder))
}

func TestTransactionInterceptor_IntegrityAndValiditySameShardShouldWork(t *testing.T) {
	t.Parallel()

	oneSharder := mock.NewOneShardCoordinatorMock()
	signer := &mock.SignerMock{}

	tx := transaction.NewInterceptedTransaction(signer)
	tx.Signature = make([]byte, 0)
	tx.Challenge = make([]byte, 0)
	tx.RcvAddr = make([]byte, 0)
	tx.SndAddr = make([]byte, 0)
	tx.Value = big.NewInt(0)

	addrConv := &mock.AddressConverterMock{}
	tx.SetAddressConverter(addrConv)

	assert.Nil(t, tx.IntegrityAndValidity(oneSharder))
	assert.Equal(t, uint32(0), tx.RcvShard())
	assert.Equal(t, uint32(0), tx.SndShard())
	assert.False(t, tx.IsAddressedToOtherShards())
}

func TestTransactionInterceptor_IntegrityAndValidityOtherShardsShouldWork(t *testing.T) {
	t.Parallel()

	multiSharder := mock.NewMultipleShardsCoordinatorMock()
	multiSharder.ComputeIdCalled = func(address state.AddressContainer) uint32 {
		if len(address.Bytes()) == 0 {
			return uint32(5)
		}

		if len(address.Bytes()) == 1 {
			return uint32(6)
		}

		return uint32(0)
	}
	multiSharder.CurrentShard = 10
	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)
	tx.Signature = make([]byte, 0)
	tx.Challenge = make([]byte, 0)
	tx.RcvAddr = make([]byte, 1)
	tx.SndAddr = make([]byte, 0)
	tx.Value = big.NewInt(0)

	addrConv := &mock.AddressConverterMock{}
	tx.SetAddressConverter(addrConv)

	assert.Nil(t, tx.IntegrityAndValidity(multiSharder))
	assert.Equal(t, uint32(6), tx.RcvShard())
	assert.Equal(t, uint32(5), tx.SndShard())
	assert.True(t, tx.IsAddressedToOtherShards())
}

//------- VerifySig()

func TestInterceptedTransaction_VerifySigNilTransactionShouldErr(t *testing.T) {
	t.Parallel()

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)
	tx.Transaction = nil

	tx.SetSingleSignKeyGen(&mock.SingleSignKeyGenMock{})

	assert.Equal(t, process.ErrNilTransaction, tx.VerifySig())
}

func TestInterceptedTransaction_VerifySigNilSingleSignKeyGenShouldErr(t *testing.T) {
	t.Parallel()

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)

	assert.Equal(t, process.ErrNilKeyGen, tx.VerifySig())
}

func TestInterceptedTransaction_VerifySigKeyGenRetErrShouldErr(t *testing.T) {
	t.Parallel()

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)

	keyGen := &mock.SingleSignKeyGenMock{}
	keyGen.PublicKeyFromByteArrayCalled = func(b []byte) (key crypto.PublicKey, e error) {
		return nil, errors.New("failure")
	}
	tx.SetSingleSignKeyGen(keyGen)

	assert.Equal(t, "failure", tx.VerifySig().Error())
}

func TestInterceptedTransaction_VerifySigKeyGenShouldReceiveSenderAddr(t *testing.T) {
	t.Parallel()

	signer := &mock.SignerMock{
		VerifyStub: func(public crypto.PublicKey, msg []byte, sig []byte) error {
			return nil
		},
		SignStub: func(private crypto.PrivateKey, msg []byte) ([]byte, error) {
			return []byte("signed"), nil
		},
	}
	tx := transaction.NewInterceptedTransaction(signer)
	senderBytes := []byte("sender")

	tx.SndAddr = senderBytes
	tx.RcvAddr = []byte("receiver")

	keyGen := &mock.SingleSignKeyGenMock{}
	keyGen.PublicKeyFromByteArrayCalled = func(b []byte) (key crypto.PublicKey, e error) {
		if !bytes.Equal(b, senderBytes) {
			assert.Fail(t, "publickey from byte array should have been called for sender bytes")
		}

		return nil, errors.New("failure")
	}
	tx.SetSingleSignKeyGen(keyGen)

	tx.VerifySig()
}

func TestInterceptedTransaction_VerifySigVerifyDoesNotPassShouldErr(t *testing.T) {
	t.Parallel()

	signer := &mock.SignerMock{
		VerifyStub: func(public crypto.PublicKey, msg []byte, sig []byte) error {
			return errors.New("sig not valid")
		},
	}
	tx := transaction.NewInterceptedTransaction(signer)

	pubKey := &mock.SingleSignPublicKey{}

	keyGen := &mock.SingleSignKeyGenMock{}
	keyGen.PublicKeyFromByteArrayCalled = func(b []byte) (key crypto.PublicKey, e error) {
		return pubKey, nil
	}
	tx.SetSingleSignKeyGen(keyGen)

	assert.Equal(t, "sig not valid", tx.VerifySig().Error())
}

func TestInterceptedTransaction_VerifySigVerifyDoesPassShouldRetNil(t *testing.T) {
	t.Parallel()

	signer := &mock.SignerMock{
		VerifyStub: func(public crypto.PublicKey, msg []byte, sig []byte) error {
			return nil
		},
		SignStub: func(private crypto.PrivateKey, msg []byte) ([]byte, error) {
			return []byte("signed"), nil
		},
	}
	tx := transaction.NewInterceptedTransaction(signer)

	pubKey := &mock.SingleSignPublicKey{}

	keyGen := &mock.SingleSignKeyGenMock{}
	keyGen.PublicKeyFromByteArrayCalled = func(b []byte) (key crypto.PublicKey, e error) {
		return pubKey, nil
	}
	tx.SetSingleSignKeyGen(keyGen)

	assert.Nil(t, tx.VerifySig())
}

//------- Getters and Setters

func TestTransactionInterceptor_GetterSetterAddrConv(t *testing.T) {
	t.Parallel()

	addrConv := &mock.AddressConverterMock{}

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)
	tx.SetAddressConverter(addrConv)

	assert.True(t, addrConv == tx.AddressConverter())
}

func TestTransactionInterceptor_GetterSetterHash(t *testing.T) {
	t.Parallel()

	hash := []byte("hash")

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)
	tx.SetHash(hash)

	assert.Equal(t, string(hash), string(tx.Hash()))
}

func TestTransactionInterceptor_GetterSetterTxBuffWithoutSig(t *testing.T) {
	t.Parallel()

	txBuffWithoutSig := []byte("txBuffWithoutSig")

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)
	tx.SetTxBuffWithoutSig(txBuffWithoutSig)

	assert.Equal(t, txBuffWithoutSig, tx.TxBuffWithoutSig())
}

func TestTransactionInterceptor_GetterSetterKeyGen(t *testing.T) {
	t.Parallel()

	keyGen := &mock.SingleSignKeyGenMock{}

	signer := &mock.SignerMock{}
	tx := transaction.NewInterceptedTransaction(signer)
	tx.SetSingleSignKeyGen(keyGen)

	assert.True(t, keyGen == tx.SingleSignKeyGen())
}
