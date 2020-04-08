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

func createMockArgHeartbeatSender() heartbeat.ArgHeartbeatSender {
	return heartbeat.ArgHeartbeatSender{
		PeerMessenger: &mock.MessengerStub{
			BroadcastCalled: func(topic string, buff []byte) {},
		},
		SingleSigner: &mock.SinglesignStub{
			SignCalled: func(private crypto.PrivateKey, msg []byte) (i []byte, e error) {
				return nil, nil
			},
		},
		PrivKey: &mock.PrivateKeyStub{},
		Marshalizer: &mock.MarshalizerMock{
			MarshalHandler: func(obj interface{}) (i []byte, e error) {
				return nil, nil
			},
		},
		Topic:            "",
		ShardCoordinator: &mock.ShardCoordinatorMock{},
		PeerTypeProvider: &mock.PeerTypeProviderStub{},
		StatusHandler:    &mock.AppStatusHandlerStub{},
		VersionNumber:    "v0.1",
		NodeDisplayName:  "undefined",
		HardforkTrigger:  &mock.HardforkTriggerStub{},
	}
}

func TestNewSender_NilP2PMessengerShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHeartbeatSender()
	arg.PeerMessenger = nil
	sender, err := heartbeat.NewSender(arg)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilMessenger, err)
}

func TestNewSender_NilSingleSignerShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHeartbeatSender()
	arg.SingleSigner = nil
	sender, err := heartbeat.NewSender(arg)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilSingleSigner, err)
}

func TestNewSender_NilShardCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHeartbeatSender()
	arg.ShardCoordinator = nil
	sender, err := heartbeat.NewSender(arg)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilShardCoordinator, err)
}

func TestNewSender_NilPrivateKeyShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHeartbeatSender()
	arg.PrivKey = nil
	sender, err := heartbeat.NewSender(arg)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilPrivateKey, err)
}

func TestNewSender_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHeartbeatSender()
	arg.Marshalizer = nil
	sender, err := heartbeat.NewSender(arg)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilMarshalizer, err)
}

func TestNewSender_NilPeerTypeProviderShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHeartbeatSender()
	arg.PeerTypeProvider = nil
	sender, err := heartbeat.NewSender(arg)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilPeerTypeProvider, err)
}

func TestNewSender_NilStatusHandlerShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHeartbeatSender()
	arg.StatusHandler = nil
	sender, err := heartbeat.NewSender(arg)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilAppStatusHandler, err)
}

func TestNewSender_NilHardforkTriggerShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHeartbeatSender()
	arg.HardforkTrigger = nil
	sender, err := heartbeat.NewSender(arg)

	assert.Nil(t, sender)
	assert.Equal(t, heartbeat.ErrNilHardforkTrigger, err)
}

func TestNewSender_ShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockArgHeartbeatSender()
	sender, err := heartbeat.NewSender(arg)

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

	arg := createMockArgHeartbeatSender()
	arg.PrivKey = &mock.PrivateKeyStub{
		GeneratePublicHandler: func() crypto.PublicKey {
			return pubKey
		},
	}
	sender, _ := heartbeat.NewSender(arg)

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

	arg := createMockArgHeartbeatSender()
	arg.SingleSigner = &mock.SinglesignStub{
		SignCalled: func(private crypto.PrivateKey, msg []byte) (i []byte, e error) {
			return nil, errExpected
		},
	}
	arg.PrivKey = &mock.PrivateKeyStub{
		GeneratePublicHandler: func() crypto.PublicKey {
			return pubKey
		},
	}
	sender, _ := heartbeat.NewSender(arg)

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

	arg := createMockArgHeartbeatSender()
	arg.PrivKey = &mock.PrivateKeyStub{
		GeneratePublicHandler: func() crypto.PublicKey {
			return pubKey
		},
	}
	arg.Marshalizer = &mock.MarshalizerMock{
		MarshalHandler: func(obj interface{}) (i []byte, e error) {
			return nil, errExpected
		},
	}
	sender, _ := heartbeat.NewSender(arg)

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

	arg := createMockArgHeartbeatSender()
	arg.Topic = testTopic
	arg.PeerMessenger = &mock.MessengerStub{
		BroadcastCalled: func(topic string, buff []byte) {
			if topic == testTopic && bytes.Equal(buff, marshaledBuff) {
				broadcastCalled = true
			}
		},
	}
	arg.SingleSigner = &mock.SinglesignStub{
		SignCalled: func(private crypto.PrivateKey, msg []byte) (i []byte, e error) {
			signCalled = true
			return signature, nil
		},
	}
	arg.PrivKey = &mock.PrivateKeyStub{
		GeneratePublicHandler: func() crypto.PublicKey {
			genPubKeyClled = true
			return pubKey
		},
	}
	arg.Marshalizer = &mock.MarshalizerMock{
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
	}
	sender, _ := heartbeat.NewSender(arg)

	err := sender.SendHeartbeat()

	assert.Nil(t, err)
	assert.True(t, broadcastCalled)
	assert.True(t, signCalled)
	assert.True(t, genPubKeyClled)
	assert.True(t, marshalCalled)
}

func TestSender_SendHeartbeatAfterTriggerShouldWork(t *testing.T) {
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

	dataPayload := []byte("payload")
	arg := createMockArgHeartbeatSender()
	arg.Topic = testTopic
	arg.PeerMessenger = &mock.MessengerStub{
		BroadcastCalled: func(topic string, buff []byte) {
			if topic != testTopic {
				return
			}
			if bytes.Equal(buff, marshaledBuff) {
				broadcastCalled = true
			}
		},
	}
	arg.SingleSigner = &mock.SinglesignStub{
		SignCalled: func(private crypto.PrivateKey, msg []byte) (i []byte, e error) {
			signCalled = true
			return signature, nil
		},
	}
	arg.PrivKey = &mock.PrivateKeyStub{
		GeneratePublicHandler: func() crypto.PublicKey {
			genPubKeyClled = true
			return pubKey
		},
	}
	arg.Marshalizer = &mock.MarshalizerMock{
		MarshalHandler: func(obj interface{}) (i []byte, e error) {
			hb, ok := obj.(*heartbeat.Heartbeat)
			if ok {
				pubkeyBytes, _ := pubKey.ToByteArray()
				if bytes.Equal(hb.Signature, signature) &&
					bytes.Equal(hb.Pubkey, pubkeyBytes) &&
					bytes.Equal(hb.Payload, dataPayload) {

					marshalCalled = true
					return marshaledBuff, nil
				}
			}

			return nil, nil
		},
	}
	arg.HardforkTrigger = &mock.HardforkTriggerStub{
		RecordedTriggerMessageCalled: func() (i []byte, b bool) {
			return nil, true
		},
		CreateDataCalled: func() []byte {
			return dataPayload
		},
	}
	sender, _ := heartbeat.NewSender(arg)

	err := sender.SendHeartbeat()

	assert.Nil(t, err)
	assert.True(t, broadcastCalled)
	assert.True(t, signCalled)
	assert.True(t, genPubKeyClled)
	assert.True(t, marshalCalled)
}

func TestSender_SendHeartbeatAfterTriggerWithRecorededPayloadShouldWork(t *testing.T) {
	t.Parallel()

	testTopic := "topic"
	marshaledBuff := []byte("marshalBuff")
	pubKey := &mock.PublicKeyMock{
		ToByteArrayHandler: func() (i []byte, e error) {
			return []byte("pub key"), nil
		},
	}
	signature := []byte("signature")
	originalTriggerPayload := []byte("original trigger payload")

	broadcastCalled := false
	broadcastTriggerCalled := false
	signCalled := false
	genPubKeyClled := false
	marshalCalled := false

	arg := createMockArgHeartbeatSender()
	arg.Topic = testTopic
	arg.PeerMessenger = &mock.MessengerStub{
		BroadcastCalled: func(topic string, buff []byte) {
			if topic != testTopic {
				return
			}
			if bytes.Equal(buff, marshaledBuff) {
				broadcastCalled = true
			}
			if bytes.Equal(buff, originalTriggerPayload) {
				broadcastTriggerCalled = true
			}
		},
	}
	arg.SingleSigner = &mock.SinglesignStub{
		SignCalled: func(private crypto.PrivateKey, msg []byte) (i []byte, e error) {
			signCalled = true
			return signature, nil
		},
	}
	arg.PrivKey = &mock.PrivateKeyStub{
		GeneratePublicHandler: func() crypto.PublicKey {
			genPubKeyClled = true
			return pubKey
		},
	}
	arg.Marshalizer = &mock.MarshalizerMock{
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
	}
	arg.HardforkTrigger = &mock.HardforkTriggerStub{
		RecordedTriggerMessageCalled: func() (i []byte, b bool) {
			return originalTriggerPayload, true
		},
	}
	sender, _ := heartbeat.NewSender(arg)

	err := sender.SendHeartbeat()

	assert.Nil(t, err)
	assert.True(t, broadcastCalled)
	assert.True(t, broadcastTriggerCalled)
	assert.True(t, signCalled)
	assert.True(t, genPubKeyClled)
	assert.True(t, marshalCalled)
}
