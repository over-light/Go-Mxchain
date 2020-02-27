package factory

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block/interceptedBlocks"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/stretchr/testify/assert"
)

var errSingleSignKeyGenMock = errors.New("errSingleSignKeyGenMock")
var errSignerMockVerifySigFails = errors.New("errSignerMockVerifySigFails")
var sigOk = []byte("signature")

func createMockKeyGen() crypto.KeyGenerator {
	return &mock.SingleSignKeyGenMock{
		PublicKeyFromByteArrayCalled: func(b []byte) (key crypto.PublicKey, e error) {
			if string(b) == "" {
				return nil, errSingleSignKeyGenMock
			}

			return &mock.SingleSignPublicKey{}, nil
		},
	}
}

func createMockSigner() crypto.SingleSigner {
	return &mock.SignerMock{
		VerifyStub: func(public crypto.PublicKey, msg []byte, sig []byte) error {
			if !bytes.Equal(sig, sigOk) {
				return errSignerMockVerifySigFails
			}
			return nil
		},
	}
}

func createMockAddressConverter() state.AddressConverter {
	return &mock.AddressConverterStub{
		CreateAddressFromPublicKeyBytesCalled: func(pubKey []byte) (container state.AddressContainer, e error) {
			return mock.NewAddressMock(pubKey), nil
		},
	}
}

func createMockFeeHandler() process.FeeHandler {
	return &mock.FeeHandlerStub{}
}

func createMockArgument() *ArgInterceptedDataFactory {
	return &ArgInterceptedDataFactory{
		Marshalizer:       &mock.MarshalizerMock{},
		Hasher:            mock.HasherMock{},
		ShardCoordinator:  mock.NewOneShardCoordinatorMock(),
		MultiSigVerifier:  mock.NewMultiSigner(),
		NodesCoordinator:  mock.NewNodesCoordinatorMock(),
		KeyGen:            createMockKeyGen(),
		BlockKeyGen:       createMockKeyGen(),
		Signer:            createMockSigner(),
		BlockSigner:       createMockSigner(),
		AddrConv:          createMockAddressConverter(),
		FeeHandler:        createMockFeeHandler(),
		HeaderSigVerifier: &mock.HeaderSigVerifierStub{},
		ChainID:           []byte("chain ID"),
		ValidityAttester:  &mock.ValidityAttesterStub{},
		EpochStartTrigger: &mock.EpochStartTriggerStub{},
	}
}

func TestNewInterceptedMetaHeaderDataFactory_NilArgumentShouldErr(t *testing.T) {
	t.Parallel()

	imh, err := NewInterceptedMetaHeaderDataFactory(nil)

	assert.True(t, check.IfNil(imh))
	assert.Equal(t, process.ErrNilArgumentStruct, err)
}

func TestNewInterceptedMetaHeaderDataFactory_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()
	arg.Marshalizer = nil

	imh, err := NewInterceptedMetaHeaderDataFactory(arg)
	assert.True(t, check.IfNil(imh))
	assert.Equal(t, process.ErrNilMarshalizer, err)
}

func TestNewInterceptedMetaHeaderDataFactory_NilHasherShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()
	arg.Hasher = nil

	imh, err := NewInterceptedMetaHeaderDataFactory(arg)
	assert.True(t, check.IfNil(imh))
	assert.Equal(t, process.ErrNilHasher, err)
}

func TestNewInterceptedMetaHeaderDataFactory_NilHeaderSigVerifierShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()
	arg.HeaderSigVerifier = nil

	imh, err := NewInterceptedMetaHeaderDataFactory(arg)
	assert.True(t, check.IfNil(imh))
	assert.Equal(t, process.ErrNilHeaderSigVerifier, err)
}

func TestNewInterceptedMetaHeaderDataFactory_NilShardCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()
	arg.ShardCoordinator = nil

	imh, err := NewInterceptedMetaHeaderDataFactory(arg)
	assert.True(t, check.IfNil(imh))
	assert.Equal(t, process.ErrNilShardCoordinator, err)
}

func TestNewInterceptedMetaHeaderDataFactory_NilChainIdShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()
	arg.ChainID = nil

	imh, err := NewInterceptedMetaHeaderDataFactory(arg)
	assert.True(t, check.IfNil(imh))
	assert.Equal(t, process.ErrInvalidChainID, err)
}

func TestNewInterceptedMetaHeaderDataFactory_NilValidityAttesterShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()
	arg.ValidityAttester = nil

	imh, err := NewInterceptedMetaHeaderDataFactory(arg)
	assert.True(t, check.IfNil(imh))
	assert.Equal(t, process.ErrNilValidityAttester, err)
}

func TestNewInterceptedMetaHeaderDataFactory_ShouldWorkAndCreate(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()

	imh, err := NewInterceptedMetaHeaderDataFactory(arg)
	assert.False(t, check.IfNil(imh))
	assert.Nil(t, err)
	assert.False(t, imh.IsInterfaceNil())

	marshalizer := &mock.MarshalizerMock{}
	emptyMetaHeader := &block.Header{}
	emptyMetaHeaderBuff, _ := marshalizer.Marshal(emptyMetaHeader)
	interceptedData, err := imh.Create(emptyMetaHeaderBuff)

	_, ok := interceptedData.(*interceptedBlocks.InterceptedMetaHeader)
	assert.True(t, ok)
}
