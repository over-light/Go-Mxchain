package heartbeat_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/node/heartbeat"
	"github.com/ElrondNetwork/elrond-go/node/mock"
	"github.com/stretchr/testify/assert"
)

//------- NewSender

func TestNewSender_NilP2PMessengerShouldErr(t *testing.T) {
	t.Parallel()

	sender, err := heartbeat.NewSender(
		nil,
		&mock.SinglesignStub{},
		&mock.PrivateKeyStub{},
		&mock.MarshalizerMock{},
		"",
		&mock.ShardCoordinatorMock{},
		&mock.PeerTypeProviderStub{},
		&mock.AppStatusHandlerStub{},
		"v0.1",
		"undefined",
	)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilMessenger, err)
}

func TestNewSender_NilSingleSignerShouldErr(t *testing.T) {
	t.Parallel()

	sender, err := heartbeat.NewSender(
		&mock.MessengerStub{},
		nil,
		&mock.PrivateKeyStub{},
		&mock.MarshalizerMock{},
		"",
		&mock.ShardCoordinatorMock{},
		&mock.PeerTypeProviderStub{},
		&mock.AppStatusHandlerStub{},
		"v0.1",
		"undefined",
	)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilSingleSigner, err)
}

func TestNewSender_NilShardCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	sender, err := heartbeat.NewSender(
		&mock.MessengerStub{},
		&mock.SinglesignStub{},
		&mock.PrivateKeyStub{},
		&mock.MarshalizerMock{},
		"",
		nil,
		&mock.PeerTypeProviderStub{},
		&mock.AppStatusHandlerStub{},
		"v0.1",
		"undefined",
	)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilShardCoordinator, err)
}

func TestNewSender_NilPrivateKeyShouldErr(t *testing.T) {
	t.Parallel()

	sender, err := heartbeat.NewSender(
		&mock.MessengerStub{},
		&mock.SinglesignStub{},
		nil,
		&mock.MarshalizerMock{},
		"",
		&mock.ShardCoordinatorMock{},
		&mock.PeerTypeProviderStub{},
		&mock.AppStatusHandlerStub{},
		"v0.1",
		"undefined",
	)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilPrivateKey, err)
}

func TestNewSender_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	sender, err := heartbeat.NewSender(
		&mock.MessengerStub{},
		&mock.SinglesignStub{},
		&mock.PrivateKeyStub{},
		nil,
		"",
		&mock.ShardCoordinatorMock{},
		&mock.PeerTypeProviderStub{},
		&mock.AppStatusHandlerStub{},
		"v0.1",
		"undefined",
	)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilMarshalizer, err)
}

func TestNewSender_NilPeerTypeProviderShouldErr(t *testing.T) {
	t.Parallel()

	sender, err := heartbeat.NewSender(
		&mock.MessengerStub{},
		&mock.SinglesignStub{},
		&mock.PrivateKeyStub{},
		&mock.MarshalizerMock{},
		"",
		&mock.ShardCoordinatorMock{},
		nil,
		&mock.AppStatusHandlerStub{},
		"v0.1",
		"undefined",
	)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilPeerTypeProvider, err)
}

func TestNewSender_NilStatusHandlerShouldErr(t *testing.T) {
	t.Parallel()

	sender, err := heartbeat.NewSender(
		&mock.MessengerStub{},
		&mock.SinglesignStub{},
		&mock.PrivateKeyStub{},
		&mock.MarshalizerMock{},
		"",
		&mock.ShardCoordinatorMock{},
		&mock.PeerTypeProviderStub{},
		nil,
		"v0.1",
		"undefined",
	)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilAppStatusHandler, err)
}

func TestNewSender_ShouldWork(t *testing.T) {
	t.Parallel()

	sender, err := heartbeat.NewSender(
		&mock.MessengerStub{},
		&mock.SinglesignStub{},
		&mock.PrivateKeyStub{},
		&mock.MarshalizerMock{},
		"",
		&mock.ShardCoordinatorMock{},
		&mock.PeerTypeProviderStub{},
		&mock.AppStatusHandlerStub{},
		"v0.1",
		"undefined",
	)

	assert.NotNil(t, sender)
	assert.Nil(t, err)
}

//------- SendHeartbeat

func TestSender_SendHeartbeatGeneratePublicKeyErrShouldErr(t *testing.T) {
	t.Parallel()

	errExpected := errors.New("expected error")
	pubKey := &mock.PublicKeyMock{
		ToByteArrayHandler: func() (i []byte, e error) {
			return nil, errExpected
		},
	}

	sender, _ := heartbeat.NewSender(
		&mock.MessengerStub{
			BroadcastCalled: func(topic string, buff []byte) {
			},
		},
		&mock.SinglesignStub{
			SignCalled: func(private crypto.PrivateKey, msg []byte) (i []byte, e error) {
				return nil, nil
			},
		},
		&mock.PrivateKeyStub{
			GeneratePublicHandler: func() crypto.PublicKey {
				return pubKey
			},
		},
		&mock.MarshalizerMock{
			MarshalHandler: func(obj interface{}) (i []byte, e error) {
				return nil, nil
			},
		},
		"",
		&mock.ShardCoordinatorMock{},
		&mock.PeerTypeProviderStub{},
		&mock.AppStatusHandlerStub{},
		"v0.1",
		"undefined",
	)

	err := sender.SendHeartbeat()

	assert.Equal(t, errExpected, err)
}

func TestSender_SendHeartbeatSignErrShouldErr(t *testing.T) {
	t.Parallel()

	errExpected := errors.New("expected error")
	pubKey := &mock.PublicKeyMock{
		ToByteArrayHandler: func() (i []byte, e error) {
			return nil, nil
		},
	}

	sender, _ := heartbeat.NewSender(
		&mock.MessengerStub{
			BroadcastCalled: func(topic string, buff []byte) {
			},
		},
		&mock.SinglesignStub{
			SignCalled: func(private crypto.PrivateKey, msg []byte) (i []byte, e error) {
				return nil, errExpected
			},
		},
		&mock.PrivateKeyStub{
			GeneratePublicHandler: func() crypto.PublicKey {
				return pubKey
			},
		},
		&mock.MarshalizerMock{
			MarshalHandler: func(obj interface{}) (i []byte, e error) {
				return nil, nil
			},
		},
		"",
		&mock.ShardCoordinatorMock{},
		&mock.PeerTypeProviderStub{},
		&mock.AppStatusHandlerStub{},
		"v0.1",
		"undefined",
	)

	err := sender.SendHeartbeat()

	assert.Equal(t, errExpected, err)
}

func TestSender_SendHeartbeatMarshalizerErrShouldErr(t *testing.T) {
	t.Parallel()

	errExpected := errors.New("expected error")
	pubKey := &mock.PublicKeyMock{
		ToByteArrayHandler: func() (i []byte, e error) {
			return nil, nil
		},
	}

	sender, _ := heartbeat.NewSender(
		&mock.MessengerStub{
			BroadcastCalled: func(topic string, buff []byte) {
			},
		},
		&mock.SinglesignStub{
			SignCalled: func(private crypto.PrivateKey, msg []byte) (i []byte, e error) {
				return nil, nil
			},
		},
		&mock.PrivateKeyStub{
			GeneratePublicHandler: func() crypto.PublicKey {
				return pubKey
			},
		},
		&mock.MarshalizerMock{
			MarshalHandler: func(obj interface{}) (i []byte, e error) {
				return nil, errExpected
			},
		},
		"",
		&mock.ShardCoordinatorMock{},
		&mock.PeerTypeProviderStub{},
		&mock.AppStatusHandlerStub{},
		"v0.1",
		"undefined",
	)

	err := sender.SendHeartbeat()

	assert.Equal(t, errExpected, err)
}

func TestSender_SendHeartbeatShouldWork(t *testing.T) {
	t.Parallel()

	testTopic := "topic"
	marshaledBuff := []byte("marshalBuff")
	pubKey := &mock.PublicKeyMock{
		ToByteArrayHandler: func() (i []byte, e error) {
			return []byte("pub key"), nil
		},
	}
	signature := []byte("signature")

	broadcastCalled := false
	signCalled := false
	genPubKeyClled := false
	marshalCalled := false

	sender, _ := heartbeat.NewSender(
		&mock.MessengerStub{
			BroadcastCalled: func(topic string, buff []byte) {
				if topic == testTopic && bytes.Equal(buff, marshaledBuff) {
					broadcastCalled = true
				}
			},
		},
		&mock.SinglesignStub{
			SignCalled: func(private crypto.PrivateKey, msg []byte) (i []byte, e error) {
				signCalled = true
				return signature, nil
			},
		},
		&mock.PrivateKeyStub{
			GeneratePublicHandler: func() crypto.PublicKey {
				genPubKeyClled = true
				return pubKey
			},
		},
		&mock.MarshalizerMock{
			MarshalHandler: func(obj interface{}) (i []byte, e error) {
				hb, ok := obj.(*heartbeat.Heartbeat)
				if ok {
					pubkeyBytes, _ := pubKey.ToByteArray()
					if bytes.Equal(hb.Signature, signature) &&
						bytes.Equal(hb.Pubkey, pubkeyBytes) {

						marshalCalled = true
						return marshaledBuff, nil
					}
				}
				return nil, nil
			},
		},
		testTopic,
		&mock.ShardCoordinatorMock{},
		&mock.PeerTypeProviderStub{},
		&mock.AppStatusHandlerStub{},
		"v0.1",
		"undefined",
	)

	err := sender.SendHeartbeat()

	assert.Nil(t, err)
	assert.True(t, broadcastCalled)
	assert.True(t, signCalled)
	assert.True(t, genPubKeyClled)
	assert.True(t, marshalCalled)
}
