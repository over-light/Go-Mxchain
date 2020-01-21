package resolvers_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/mock"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/resolvers"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/stretchr/testify/assert"
)

//------- NewHeaderResolver

func TestNewHeaderResolver_NilSenderResolverShouldErr(t *testing.T) {
	t.Parallel()

	hdrRes, err := resolvers.NewHeaderResolver(
		nil,
		&mock.HeadersCacherStub{},
		&mock.StorerStub{},
		&mock.StorerStub{},
		&mock.MarshalizerMock{},
		mock.NewNonceHashConverterMock(),
	)

	assert.Equal(t, dataRetriever.ErrNilResolverSender, err)
	assert.Nil(t, hdrRes)
}

func TestNewHeaderResolver_NilHeadersPoolShouldErr(t *testing.T) {
	t.Parallel()

	hdrRes, err := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{},
		nil,
		&mock.StorerStub{},
		&mock.StorerStub{},
		&mock.MarshalizerMock{},
		mock.NewNonceHashConverterMock(),
	)

	assert.Equal(t, dataRetriever.ErrNilHeadersDataPool, err)
	assert.Nil(t, hdrRes)
}

func TestNewHeaderResolver_NilHeadersStorageShouldErr(t *testing.T) {
	t.Parallel()

	hdrRes, err := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{},
		&mock.HeadersCacherStub{},
		nil,
		&mock.StorerStub{},
		&mock.MarshalizerMock{},
		mock.NewNonceHashConverterMock(),
	)

	assert.Equal(t, dataRetriever.ErrNilHeadersStorage, err)
	assert.Nil(t, hdrRes)
}

func TestNewHeaderResolver_NilHeadersNoncesStorageShouldErr(t *testing.T) {
	t.Parallel()

	hdrRes, err := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{},
		&mock.HeadersCacherStub{},
		&mock.StorerStub{},
		nil,
		&mock.MarshalizerMock{},
		mock.NewNonceHashConverterMock(),
	)

	assert.Equal(t, dataRetriever.ErrNilHeadersNoncesStorage, err)
	assert.Nil(t, hdrRes)
}

func TestNewHeaderResolver_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	hdrRes, err := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{},
		&mock.HeadersCacherStub{},
		&mock.StorerStub{},
		&mock.StorerStub{},
		nil,
		mock.NewNonceHashConverterMock(),
	)

	assert.Equal(t, dataRetriever.ErrNilMarshalizer, err)
	assert.Nil(t, hdrRes)
}

func TestNewHeaderResolver_NilNonceConverterShouldErr(t *testing.T) {
	t.Parallel()

	hdrRes, err := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{},
		&mock.HeadersCacherStub{},
		&mock.StorerStub{},
		&mock.StorerStub{},
		&mock.MarshalizerMock{},
		nil,
	)

	assert.Equal(t, dataRetriever.ErrNilUint64ByteSliceConverter, err)
	assert.Nil(t, hdrRes)
}

func TestNewHeaderResolver_OkValsShouldWork(t *testing.T) {
	t.Parallel()

	hdrRes, err := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{},
		&mock.HeadersCacherStub{},
		&mock.StorerStub{},
		&mock.StorerStub{},
		&mock.MarshalizerMock{},
		mock.NewNonceHashConverterMock(),
	)

	assert.NotNil(t, hdrRes)
	assert.Nil(t, err)
	assert.False(t, hdrRes.IsInterfaceNil())
}

//------- ProcessReceivedMessage

func TestHeaderResolver_ProcessReceivedMessageNilValueShouldErr(t *testing.T) {
	t.Parallel()

	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{},
		&mock.HeadersCacherStub{},
		&mock.StorerStub{},
		&mock.StorerStub{},
		&mock.MarshalizerMock{},
		mock.NewNonceHashConverterMock(),
	)

	err := hdrRes.ProcessReceivedMessage(createRequestMsg(dataRetriever.NonceType, nil), nil)
	assert.Equal(t, dataRetriever.ErrNilValue, err)
}

func TestHeaderResolver_ProcessReceivedMessage_WrongIdentifierStartBlock(t *testing.T) {
	t.Parallel()

	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{},
		&mock.HeadersCacherStub{},
		&mock.StorerStub{},
		&mock.StorerStub{},
		&mock.MarshalizerMock{},
		mock.NewNonceHashConverterMock(),
	)

	requestedData := []byte("request")
	err := hdrRes.ProcessReceivedMessage(createRequestMsg(dataRetriever.EpochType, requestedData), nil)
	assert.Equal(t, core.ErrInvalidIdentifierForEpochStartBlockRequest, err)
}

func TestHeaderResolver_ProcessReceivedMessage_Ok(t *testing.T) {
	t.Parallel()

	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{},
		&mock.HeadersCacherStub{},
		&mock.StorerStub{
			GetCalled: func(key []byte) (i []byte, err error) {
				return nil, nil
			},
		},
		&mock.StorerStub{},
		&mock.MarshalizerMock{},
		mock.NewNonceHashConverterMock(),
	)

	requestedData := []byte("request_1")
	err := hdrRes.ProcessReceivedMessage(createRequestMsg(dataRetriever.EpochType, requestedData), nil)
	assert.Nil(t, err)
}

func TestHeaderResolver_RequestDataFromEpoch(t *testing.T) {
	t.Parallel()

	called := false
	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{
			SendOnRequestTopicCalled: func(rd *dataRetriever.RequestData) error {
				called = true
				return nil
			},
		},
		&mock.HeadersCacherStub{},
		&mock.StorerStub{
			GetCalled: func(key []byte) (i []byte, err error) {
				return nil, nil
			},
		},
		&mock.StorerStub{},
		&mock.MarshalizerMock{},
		mock.NewNonceHashConverterMock(),
	)

	requestedData := []byte("request_1")
	err := hdrRes.RequestDataFromEpoch(requestedData)
	assert.Nil(t, err)
	assert.True(t, called)
}

func TestHeaderResolver_ProcessReceivedMessageRequestUnknownTypeShouldErr(t *testing.T) {
	t.Parallel()

	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{},
		&mock.HeadersCacherStub{},
		&mock.StorerStub{},
		&mock.StorerStub{},
		&mock.MarshalizerMock{},
		mock.NewNonceHashConverterMock(),
	)

	err := hdrRes.ProcessReceivedMessage(createRequestMsg(254, make([]byte, 0)), nil)
	assert.Equal(t, dataRetriever.ErrResolveTypeUnknown, err)

}

func TestHeaderResolver_ValidateRequestHashTypeFoundInHdrPoolShouldSearchAndSend(t *testing.T) {
	t.Parallel()

	requestedData := []byte("aaaa")

	searchWasCalled := false
	sendWasCalled := false

	headers := &mock.HeadersCacherStub{}

	headers.GetHeaderByHashCalled = func(hash []byte) (handler data.HeaderHandler, e error) {
		if bytes.Equal(requestedData, hash) {
			searchWasCalled = true
			return &block.Header{}, nil
		}
		return nil, errors.New("0")
	}

	marshalizer := &mock.MarshalizerMock{}

	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{
			SendCalled: func(buff []byte, peer p2p.PeerID) error {
				sendWasCalled = true
				return nil
			},
		},
		headers,
		&mock.StorerStub{},
		&mock.StorerStub{},
		marshalizer,
		mock.NewNonceHashConverterMock(),
	)

	err := hdrRes.ProcessReceivedMessage(createRequestMsg(dataRetriever.HashType, requestedData), nil)
	assert.Nil(t, err)
	assert.True(t, searchWasCalled)
	assert.True(t, sendWasCalled)
}

func TestHeaderResolver_ProcessReceivedMessageRequestHashTypeFoundInHdrPoolMarshalizerFailsShouldErr(t *testing.T) {
	t.Parallel()

	requestedData := []byte("aaaa")

	errExpected := errors.New("MarshalizerMock generic error")

	headers := &mock.HeadersCacherStub{}

	headers.GetHeaderByHashCalled = func(hash []byte) (handler data.HeaderHandler, e error) {
		if bytes.Equal(requestedData, hash) {
			return &block.Header{}, nil
		}
		return nil, errors.New("err")
	}

	marshalizerMock := &mock.MarshalizerMock{}
	marshalizerStub := &mock.MarshalizerStub{
		MarshalCalled: func(obj interface{}) (i []byte, e error) {
			return nil, errExpected
		},
		UnmarshalCalled: func(obj interface{}, buff []byte) error {
			return marshalizerMock.Unmarshal(obj, buff)
		},
	}

	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{
			SendCalled: func(buff []byte, peer p2p.PeerID) error {
				return nil
			},
		},
		headers,
		&mock.StorerStub{},
		&mock.StorerStub{},
		marshalizerStub,
		mock.NewNonceHashConverterMock(),
	)

	err := hdrRes.ProcessReceivedMessage(createRequestMsg(dataRetriever.HashType, requestedData), nil)
	assert.Equal(t, errExpected, err)
}

func TestHeaderResolver_ProcessReceivedMessageRequestRetFromStorageShouldRetValAndSend(t *testing.T) {
	t.Parallel()

	requestedData := []byte("aaaa")

	headers := &mock.HeadersCacherStub{}

	headers.GetHeaderByHashCalled = func(hash []byte) (handler data.HeaderHandler, e error) {
		return nil, errors.New("err")
	}

	wasGotFromStorage := false
	wasSent := false

	store := &mock.StorerStub{}
	store.SearchFirstCalled = func(key []byte) (i []byte, e error) {
		if bytes.Equal(key, requestedData) {
			wasGotFromStorage = true
			return make([]byte, 0), nil
		}

		return nil, errors.New("should have not reach this point")
	}

	marshalizer := &mock.MarshalizerMock{}

	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{
			SendCalled: func(buff []byte, peer p2p.PeerID) error {
				wasSent = true
				return nil
			},
		},
		headers,
		store,
		&mock.StorerStub{},
		marshalizer,
		mock.NewNonceHashConverterMock(),
	)

	err := hdrRes.ProcessReceivedMessage(createRequestMsg(dataRetriever.HashType, requestedData), nil)
	assert.Nil(t, err)
	assert.True(t, wasGotFromStorage)
	assert.True(t, wasSent)
}

func TestHeaderResolver_ProcessReceivedMessageRequestNonceTypeInvalidSliceShouldErr(t *testing.T) {
	t.Parallel()

	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{},
		&mock.HeadersCacherStub{},
		&mock.StorerStub{},
		&mock.StorerStub{
			GetFromEpochCalled: func(key []byte, epoch uint32) ([]byte, error) {
				return nil, errors.New("key not found")
			},
		},
		&mock.MarshalizerMock{},
		mock.NewNonceHashConverterMock(),
	)

	err := hdrRes.ProcessReceivedMessage(createRequestMsg(dataRetriever.NonceType, []byte("aaa")), nil)
	assert.Equal(t, dataRetriever.ErrInvalidNonceByteSlice, err)
}

func TestHeaderResolver_ProcessReceivedMessageRequestNonceShouldCallWithTheCorrectEpoch(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	expectedEpoch := uint32(7)
	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{},
		&mock.HeadersCacherStub{},
		&mock.StorerStub{},
		&mock.StorerStub{
			GetFromEpochCalled: func(key []byte, epoch uint32) ([]byte, error) {
				assert.Equal(t, expectedEpoch, epoch)
				return nil, nil
			},
		},
		marshalizer,
		mock.NewNonceHashConverterMock(),
	)

	buff, _ := marshalizer.Marshal(
		&dataRetriever.RequestData{
			Type:  dataRetriever.NonceType,
			Value: []byte("aaa"),
			Epoch: expectedEpoch,
		},
	)
	msg := &mock.P2PMessageMock{DataField: buff}
	_ = hdrRes.ProcessReceivedMessage(msg, nil)
}

func TestHeaderResolver_ProcessReceivedMessageRequestNonceTypeNotFoundInHdrNoncePoolAndStorageShouldRetNilAndNotSend(t *testing.T) {
	t.Parallel()

	requestedNonce := uint64(67)
	nonceConverter := mock.NewNonceHashConverterMock()

	expectedErr := errors.New("err")
	wasSent := false

	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{
			SendCalled: func(buff []byte, peer p2p.PeerID) error {
				wasSent = true
				return nil
			},
			TargetShardIDCalled: func() uint32 {
				return 1
			},
		},
		&mock.HeadersCacherStub{
			GetHeaderByNonceAndShardIdCalled: func(hdrNonce uint64, shardId uint32) (handlers []data.HeaderHandler, i [][]byte, e error) {
				return nil, nil, expectedErr
			},
		},
		&mock.StorerStub{
			SearchFirstCalled: func(key []byte) (i []byte, e error) {
				return nil, errors.New("key not found")
			},
		},
		&mock.StorerStub{
			GetFromEpochCalled: func(key []byte, epoch uint32) (i []byte, e error) {
				return nil, errors.New("key not found")
			},
			SearchFirstCalled: func(key []byte) (i []byte, e error) {
				return nil, errors.New("key not found")
			},
		},
		&mock.MarshalizerMock{},
		nonceConverter,
	)

	err := hdrRes.ProcessReceivedMessage(
		createRequestMsg(dataRetriever.NonceType, nonceConverter.ToByteSlice(requestedNonce)),
		nil,
	)
	assert.Equal(t, expectedErr, err)
	assert.False(t, wasSent)
}

func TestHeaderResolver_ProcessReceivedMessageRequestNonceTypeFoundInHdrNoncePoolShouldRetFromPoolAndSend(t *testing.T) {
	t.Parallel()

	requestedNonce := uint64(67)
	targetShardId := uint32(9)
	wasResolved := false
	wasSent := false

	headers := &mock.HeadersCacherStub{}
	headers.GetHeaderByNonceAndShardIdCalled = func(hdrNonce uint64, shardId uint32) (handlers []data.HeaderHandler, i [][]byte, e error) {
		wasResolved = true
		return []data.HeaderHandler{&block.Header{}, &block.Header{}}, [][]byte{[]byte("1"), []byte("2")}, nil
	}

	nonceConverter := mock.NewNonceHashConverterMock()
	marshalizer := &mock.MarshalizerMock{}

	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{
			SendCalled: func(buff []byte, peer p2p.PeerID) error {
				wasSent = true
				return nil
			},
			TargetShardIDCalled: func() uint32 {
				return targetShardId
			},
		},
		headers,
		&mock.StorerStub{},
		&mock.StorerStub{
			GetFromEpochCalled: func(key []byte, epoch uint32) ([]byte, error) {
				return nil, errors.New("key not found")
			},
			SearchFirstCalled: func(key []byte) ([]byte, error) {
				return nil, errors.New("key not found")
			},
		},
		marshalizer,
		nonceConverter,
	)

	err := hdrRes.ProcessReceivedMessage(
		createRequestMsg(dataRetriever.NonceType, nonceConverter.ToByteSlice(requestedNonce)),
		nil,
	)

	assert.Nil(t, err)
	assert.True(t, wasResolved)
	assert.True(t, wasSent)
}

func TestHeaderResolver_ProcessReceivedMessageRequestNonceTypeFoundInHdrNoncePoolShouldRetFromStorageAndSend(t *testing.T) {
	t.Parallel()

	requestedNonce := uint64(67)
	targetShardId := uint32(9)
	wasResolved := false
	wasSend := false
	hash := []byte("aaaa")

	headers := &mock.HeadersCacherStub{}
	headers.GetHeaderByHashCalled = func(hash []byte) (handler data.HeaderHandler, e error) {
		return nil, errors.New("err")
	}
	headers.GetHeaderByNonceAndShardIdCalled = func(hdrNonce uint64, shardId uint32) (handlers []data.HeaderHandler, i [][]byte, e error) {
		wasResolved = true
		return []data.HeaderHandler{&block.Header{}, &block.Header{}}, [][]byte{[]byte("1"), []byte("2")}, nil
	}

	nonceConverter := mock.NewNonceHashConverterMock()
	marshalizer := &mock.MarshalizerMock{}

	store := &mock.StorerStub{}
	store.GetFromEpochCalled = func(key []byte, epoch uint32) (i []byte, e error) {
		if bytes.Equal(key, hash) {
			wasResolved = true
			return make([]byte, 0), nil
		}

		return nil, errors.New("should have not reach this point")
	}

	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{
			SendCalled: func(buff []byte, peer p2p.PeerID) error {
				wasSend = true
				return nil
			},
			TargetShardIDCalled: func() uint32 {
				return targetShardId
			},
		},
		headers,
		store,
		&mock.StorerStub{
			GetFromEpochCalled: func(key []byte, epoch uint32) ([]byte, error) {
				return nil, errors.New("key not found")
			},
			SearchFirstCalled: func(key []byte) (i []byte, e error) {
				return nil, errors.New("key not found")
			},
		},
		marshalizer,
		nonceConverter,
	)

	err := hdrRes.ProcessReceivedMessage(
		createRequestMsg(dataRetriever.NonceType, nonceConverter.ToByteSlice(requestedNonce)),
		nil,
	)

	assert.Nil(t, err)
	assert.True(t, wasResolved)
	assert.True(t, wasSend)
}

func TestHeaderResolver_ProcessReceivedMessageRequestNonceTypeFoundInHdrNoncePoolCheckRetErr(t *testing.T) {
	t.Parallel()

	requestedNonce := uint64(67)
	targetShardId := uint32(9)
	errExpected := errors.New("expected error")

	headers := &mock.HeadersCacherStub{}
	headers.GetHeaderByHashCalled = func(hash []byte) (handler data.HeaderHandler, e error) {
		return nil, errors.New("err")
	}
	headers.GetHeaderByNonceAndShardIdCalled = func(hdrNonce uint64, shardId uint32) (handlers []data.HeaderHandler, i [][]byte, e error) {
		return nil, nil, errExpected
	}

	nonceConverter := mock.NewNonceHashConverterMock()
	marshalizer := &mock.MarshalizerMock{}

	store := &mock.StorerStub{}
	store.GetFromEpochCalled = func(key []byte, epoch uint32) (i []byte, e error) {
		if bytes.Equal(key, []byte("aaaa")) {
			return nil, errExpected
		}

		return nil, errors.New("should have not reach this point")
	}

	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{
			SendCalled: func(buff []byte, peer p2p.PeerID) error {
				return nil
			},
			TargetShardIDCalled: func() uint32 {
				return targetShardId
			},
		},
		headers,
		store,
		&mock.StorerStub{
			GetFromEpochCalled: func(key []byte, epoch uint32) ([]byte, error) {
				return nil, errors.New("key not found")
			},
			SearchFirstCalled: func(key []byte) (i []byte, e error) {
				return nil, errors.New("key not found")
			},
		},
		marshalizer,
		nonceConverter,
	)

	err := hdrRes.ProcessReceivedMessage(
		createRequestMsg(dataRetriever.NonceType, nonceConverter.ToByteSlice(requestedNonce)),
		nil,
	)

	assert.Equal(t, errExpected, err)
}

//------- Requests

func TestHeaderResolver_RequestDataFromNonceShouldWork(t *testing.T) {
	t.Parallel()

	nonceRequested := uint64(67)
	wasRequested := false

	nonceConverter := mock.NewNonceHashConverterMock()

	buffToExpect := nonceConverter.ToByteSlice(nonceRequested)

	hdrRes, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{
			SendOnRequestTopicCalled: func(rd *dataRetriever.RequestData) error {
				if bytes.Equal(rd.Value, buffToExpect) {
					wasRequested = true
				}
				return nil
			},
		},
		&mock.HeadersCacherStub{},
		&mock.StorerStub{},
		&mock.StorerStub{},
		&mock.MarshalizerMock{},
		nonceConverter,
	)

	assert.Nil(t, hdrRes.RequestDataFromNonce(nonceRequested, 0))
	assert.True(t, wasRequested)
}

func TestHeaderResolverBase_RequestDataFromHashShouldWork(t *testing.T) {
	t.Parallel()

	buffRequested := []byte("aaaa")
	wasRequested := false
	nonceConverter := mock.NewNonceHashConverterMock()
	hdrResBase, _ := resolvers.NewHeaderResolver(
		&mock.TopicResolverSenderStub{
			SendOnRequestTopicCalled: func(rd *dataRetriever.RequestData) error {
				if bytes.Equal(rd.Value, buffRequested) {
					wasRequested = true
				}

				return nil
			},
		},
		&mock.HeadersCacherStub{},
		&mock.StorerStub{},
		&mock.StorerStub{},
		&mock.MarshalizerMock{},
		nonceConverter,
	)

	assert.Nil(t, hdrResBase.RequestDataFromHash(buffRequested, 0))
	assert.True(t, wasRequested)
}
