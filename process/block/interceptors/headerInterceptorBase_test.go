package interceptors_test

import (
	"errors"
	"testing"

	dataBlock "github.com/ElrondNetwork/elrond-go-sandbox/data/block"
	"github.com/ElrondNetwork/elrond-go-sandbox/process"
	"github.com/ElrondNetwork/elrond-go-sandbox/process/block"
	"github.com/ElrondNetwork/elrond-go-sandbox/process/block/interceptors"
	"github.com/ElrondNetwork/elrond-go-sandbox/process/mock"
	"github.com/stretchr/testify/assert"
)

//------- NewHeaderInterceptorBase

func TestNewHeaderInterceptorBase_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	storer := &mock.StorerStub{}
	hi, err := interceptors.NewHeaderInterceptorBase(
		nil,
		storer,
		mock.NewMultiSigner(),
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock())

	assert.Equal(t, process.ErrNilMarshalizer, err)
	assert.Nil(t, hi)
}

func TestNewHeaderInterceptorBase_NilStorerShouldErr(t *testing.T) {
	t.Parallel()

	hi, err := interceptors.NewHeaderInterceptorBase(
		&mock.MarshalizerMock{},
		nil,
		mock.NewMultiSigner(),
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock())

	assert.Equal(t, process.ErrNilHeadersStorage, err)
	assert.Nil(t, hi)
}

func TestNewHeaderInterceptorBase_NilMultiSignerShouldErr(t *testing.T) {
	t.Parallel()

	storer := &mock.StorerStub{}
	hi, err := interceptors.NewHeaderInterceptorBase(
		&mock.MarshalizerMock{},
		storer,
		nil,
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock())

	assert.Nil(t, hi)
	assert.Equal(t, process.ErrNilMultiSigVerifier, err)
}

func TestNewHeaderInterceptorBase_NilHasherShouldErr(t *testing.T) {
	t.Parallel()

	storer := &mock.StorerStub{}
	hi, err := interceptors.NewHeaderInterceptorBase(
		&mock.MarshalizerMock{},
		storer,
		mock.NewMultiSigner(),
		nil,
		mock.NewOneShardCoordinatorMock())

	assert.Equal(t, process.ErrNilHasher, err)
	assert.Nil(t, hi)
}

func TestNewHeaderInterceptorBase_NilShardCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	storer := &mock.StorerStub{}
	hi, err := interceptors.NewHeaderInterceptorBase(
		&mock.MarshalizerMock{},
		storer,
		mock.NewMultiSigner(),
		mock.HasherMock{},
		nil)

	assert.Equal(t, process.ErrNilShardCoordinator, err)
	assert.Nil(t, hi)
}

func TestNewHeaderInterceptorBase_OkValsShouldWork(t *testing.T) {
	t.Parallel()

	storer := &mock.StorerStub{}
	hib, err := interceptors.NewHeaderInterceptorBase(
		&mock.MarshalizerMock{},
		storer,
		mock.NewMultiSigner(),
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock())

	assert.Nil(t, err)
	assert.NotNil(t, hib)
}

//------- ProcessReceivedMessage

func TestHeaderInterceptorBase_ParseReceivedMessageNilMessageShouldErr(t *testing.T) {
	t.Parallel()

	storer := &mock.StorerStub{}
	hib, _ := interceptors.NewHeaderInterceptorBase(
		&mock.MarshalizerMock{},
		storer,
		mock.NewMultiSigner(),
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock())

	hdr, err := hib.ParseReceivedMessage(nil)

	assert.Nil(t, hdr)
	assert.Equal(t, process.ErrNilMessage, err)
}

func TestHeaderInterceptorBase_ParseReceivedMessageNilDataToProcessShouldErr(t *testing.T) {
	t.Parallel()

	storer := &mock.StorerStub{}
	hib, _ := interceptors.NewHeaderInterceptorBase(
		&mock.MarshalizerMock{},
		storer,
		mock.NewMultiSigner(),
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock())

	msg := &mock.P2PMessageMock{}
	hdr, err := hib.ParseReceivedMessage(msg)

	assert.Nil(t, hdr)
	assert.Equal(t, process.ErrNilDataToProcess, err)
}

func TestHeaderInterceptorBase_ParseReceivedMessageMarshalizerErrorsAtUnmarshalingShouldErr(t *testing.T) {
	t.Parallel()

	errMarshalizer := errors.New("marshalizer error")

	storer := &mock.StorerStub{}
	hib, _ := interceptors.NewHeaderInterceptorBase(
		&mock.MarshalizerStub{
			UnmarshalCalled: func(obj interface{}, buff []byte) error {
				return errMarshalizer
			},
		},
		storer,
		mock.NewMultiSigner(),
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock())

	msg := &mock.P2PMessageMock{
		DataField: make([]byte, 0),
	}
	hdr, err := hib.ParseReceivedMessage(msg)

	assert.Nil(t, hdr)
	assert.Equal(t, errMarshalizer, err)
}

func TestHeaderInterceptorBase_ParseReceivedMessageSanityCheckFailedShouldErr(t *testing.T) {
	t.Parallel()

	storer := &mock.StorerStub{}
	marshalizer := &mock.MarshalizerMock{}
	multisigner := mock.NewMultiSigner()
	hib, _ := interceptors.NewHeaderInterceptorBase(
		marshalizer,
		storer,
		multisigner,
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock())

	hdr := block.NewInterceptedHeader(multisigner)
	buff, _ := marshalizer.Marshal(hdr)
	msg := &mock.P2PMessageMock{
		DataField: buff,
	}
	hdr, err := hib.ParseReceivedMessage(msg)

	assert.Nil(t, hdr)
	assert.Equal(t, process.ErrNilPubKeysBitmap, err)
}

func TestHeaderInterceptorBase_ParseReceivedMessageValsOkShouldWork(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	testedNonce := uint64(67)
	multisigner := mock.NewMultiSigner()
	storer := &mock.StorerStub{}
	storer.HasCalled = func(key []byte) (bool, error) {
		return false, nil
	}
	hib, _ := interceptors.NewHeaderInterceptorBase(
		marshalizer,
		storer,
		multisigner,
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock())

	hdr := block.NewInterceptedHeader(multisigner)
	hdr.Nonce = testedNonce
	hdr.ShardId = 0
	hdr.PrevHash = make([]byte, 0)
	hdr.PubKeysBitmap = make([]byte, 0)
	hdr.BlockBodyType = dataBlock.TxBlock
	hdr.Signature = make([]byte, 0)
	hdr.SetHash([]byte("aaa"))
	hdr.RootHash = make([]byte, 0)
	hdr.MiniBlockHeaders = make([]dataBlock.MiniBlockHeader, 0)

	buff, _ := marshalizer.Marshal(hdr)
	msg := &mock.P2PMessageMock{
		DataField: buff,
	}

	hdrIntercepted, err := hib.ParseReceivedMessage(msg)
	if hdrIntercepted != nil {
		//hdrIntercepted will have a "real" hash computed
		hdrIntercepted.SetHash(hdr.Hash())
	}

	assert.Equal(t, hdr, hdrIntercepted)
	assert.Nil(t, err)
}

func TestHeaderInterceptorBase_ParseReceivedMessageFoundInStorageShouldErr(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	testedNonce := uint64(67)
	multisigner := mock.NewMultiSigner()
	storer := &mock.StorerStub{}
	storer.HasCalled = func(key []byte) (bool, error) {
		return true, nil
	}
	hib, _ := interceptors.NewHeaderInterceptorBase(
		marshalizer,
		storer,
		multisigner,
		mock.HasherMock{},
		mock.NewOneShardCoordinatorMock())

	hdr := block.NewInterceptedHeader(multisigner)
	hdr.Nonce = testedNonce
	hdr.ShardId = 0
	hdr.PrevHash = make([]byte, 0)
	hdr.PubKeysBitmap = make([]byte, 0)
	hdr.BlockBodyType = dataBlock.TxBlock
	hdr.Signature = make([]byte, 0)
	hdr.SetHash([]byte("aaa"))
	hdr.RootHash = make([]byte, 0)
	hdr.MiniBlockHeaders = make([]dataBlock.MiniBlockHeader, 0)

	buff, _ := marshalizer.Marshal(hdr)
	msg := &mock.P2PMessageMock{
		DataField: buff,
	}

	hdrIntercepted, err := hib.ParseReceivedMessage(msg)

	assert.Nil(t, hdrIntercepted)
	assert.Equal(t, process.ErrHeaderIsInStorage, err)
}
