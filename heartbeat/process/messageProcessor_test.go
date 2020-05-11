package process_test

import (
	"errors"
	"testing"

	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/heartbeat"
	"github.com/ElrondNetwork/elrond-go/heartbeat/data"
	"github.com/ElrondNetwork/elrond-go/heartbeat/mock"
	"github.com/ElrondNetwork/elrond-go/heartbeat/process"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/stretchr/testify/assert"
)

func CreateHeartbeat() *data.Heartbeat {
	hb := data.Heartbeat{
		Payload:         []byte("Payload"),
		Pubkey:          []byte("PubKey"),
		Signature:       []byte("Signature"),
		ShardID:         0,
		VersionNumber:   "VersionNumber",
		NodeDisplayName: "NodeDisplayName",
	}
	return &hb
}

func TestNewMessageProcessor_SingleSignerNilShouldErr(t *testing.T) {
	t.Parallel()

	mon, err := process.NewMessageProcessor(
		nil,
		&mock.KeyGenMock{},
		&mock.MarshalizerStub{},
		&mock.NetworkShardingCollectorStub{},
	)

	assert.Nil(t, mon)
	assert.Equal(t, heartbeat.ErrNilSingleSigner, err)
}

func TestNewMessageProcessor_KeyGeneratorNilShouldErr(t *testing.T) {
	t.Parallel()

	mon, err := process.NewMessageProcessor(
		&mock.SinglesignMock{},
		nil,
		&mock.MarshalizerStub{},
		&mock.NetworkShardingCollectorStub{},
	)

	assert.Nil(t, mon)
	assert.Equal(t, heartbeat.ErrNilKeyGenerator, err)
}

func TestNewMessageProcessor_MarshalizerNilShouldErr(t *testing.T) {
	t.Parallel()

	mon, err := process.NewMessageProcessor(
		&mock.SinglesignMock{},
		&mock.KeyGenMock{},
		nil,
		&mock.NetworkShardingCollectorStub{},
	)

	assert.Nil(t, mon)
	assert.Equal(t, heartbeat.ErrNilMarshalizer, err)
}

func TestNewMessageProcessor_NetworkShardingCollectorNilShouldErr(t *testing.T) {
	t.Parallel()

	mon, err := process.NewMessageProcessor(
		&mock.SinglesignMock{},
		&mock.KeyGenMock{},
		&mock.MarshalizerStub{},
		nil,
	)

	assert.Nil(t, mon)
	assert.Equal(t, heartbeat.ErrNilNetworkShardingCollector, err)
}

func TestNewMessageProcessor_ShouldWork(t *testing.T) {
	t.Parallel()

	mon, err := process.NewMessageProcessor(
		&mock.SinglesignMock{},
		&mock.KeyGenMock{},
		&mock.MarshalizerStub{},
		&mock.NetworkShardingCollectorStub{},
	)

	assert.Nil(t, err)
	assert.NotNil(t, mon)
	assert.False(t, mon.IsInterfaceNil())
}

func TestNewMessageProcessor_VerifyMessageAllSmallerShouldWork(t *testing.T) {
	t.Parallel()

	hbmi := CreateHeartbeat()
	err := process.VerifyLengths(hbmi)

	assert.Nil(t, err)
}

func TestNewMessageProcessor_VerifyMessageAllNilShouldWork(t *testing.T) {
	t.Parallel()

	hbmi := CreateHeartbeat()
	hbmi.Signature = nil
	hbmi.Payload = nil
	hbmi.Pubkey = nil
	hbmi.VersionNumber = ""
	hbmi.NodeDisplayName = ""

	err := process.VerifyLengths(hbmi)

	assert.Nil(t, err)
}

func TestNewMessageProcessor_VerifyMessageBiggerPublicKeyShouldErr(t *testing.T) {
	t.Parallel()

	hbmi := CreateHeartbeat()
	hbmi.Pubkey = make([]byte, process.GetMaxSizeInBytes()+1)
	err := process.VerifyLengths(hbmi)

	assert.NotNil(t, err)
}

func TestNewMessageProcessor_VerifyMessageAllSmallerPublicKeyShouldWork(t *testing.T) {
	t.Parallel()

	hbmi := CreateHeartbeat()
	hbmi.Pubkey = make([]byte, process.GetMaxSizeInBytes())
	err := process.VerifyLengths(hbmi)

	assert.Nil(t, err)
}

func TestNewMessageProcessor_VerifyMessageBiggerPayloadShouldErr(t *testing.T) {
	t.Parallel()

	hbmi := CreateHeartbeat()
	hbmi.Payload = make([]byte, process.GetMaxSizeInBytes()+1)
	err := process.VerifyLengths(hbmi)

	assert.NotNil(t, err)
}

func TestNewMessageProcessor_VerifyMessageSmallerPayloadShouldWork(t *testing.T) {
	t.Parallel()

	hbmi := CreateHeartbeat()
	hbmi.Payload = make([]byte, process.GetMaxSizeInBytes())
	err := process.VerifyLengths(hbmi)

	assert.Nil(t, err)
}

func TestNewMessageProcessor_VerifyMessageBiggerSignatureShouldErr(t *testing.T) {
	t.Parallel()

	hbmi := CreateHeartbeat()
	hbmi.Signature = make([]byte, process.GetMaxSizeInBytes()+1)
	err := process.VerifyLengths(hbmi)

	assert.NotNil(t, err)
}

func TestNewMessageProcessor_VerifyMessageSignatureShouldWork(t *testing.T) {
	t.Parallel()

	hbmi := CreateHeartbeat()
	hbmi.Signature = make([]byte, process.GetMaxSizeInBytes()-1)
	err := process.VerifyLengths(hbmi)

	assert.Nil(t, err)
}

func TestNewMessageProcessor_VerifyMessageBiggerNodeDisplayNameShouldErr(t *testing.T) {
	t.Parallel()

	hbmi := CreateHeartbeat()
	hbmi.NodeDisplayName = string(make([]byte, process.GetMaxSizeInBytes()+1))
	err := process.VerifyLengths(hbmi)

	assert.NotNil(t, err)
}

func TestNewMessageProcessor_VerifyMessageNodeDisplayNameShouldWork(t *testing.T) {
	t.Parallel()

	hbmi := CreateHeartbeat()
	hbmi.NodeDisplayName = string(make([]byte, process.GetMaxSizeInBytes()))
	err := process.VerifyLengths(hbmi)

	assert.Nil(t, err)
}

func TestNewMessageProcessor_VerifyMessageBiggerVersionNumberShouldErr(t *testing.T) {
	t.Parallel()

	hbmi := CreateHeartbeat()
	hbmi.VersionNumber = string(make([]byte, process.GetMaxSizeInBytes()+1))
	err := process.VerifyLengths(hbmi)

	assert.NotNil(t, err)
}

func TestNewMessageProcessor_VerifyMessageVersionNumberShouldWork(t *testing.T) {
	t.Parallel()

	hbmi := CreateHeartbeat()
	hbmi.VersionNumber = string(make([]byte, process.GetMaxSizeInBytes()))
	err := process.VerifyLengths(hbmi)

	assert.Nil(t, err)
}

func TestNewMessageProcessor_CreateHeartbeatFromP2PMessage(t *testing.T) {
	t.Parallel()

	hb := data.Heartbeat{
		Payload:         []byte("Payload"),
		Pubkey:          []byte("PubKey"),
		Signature:       []byte("signed"),
		ShardID:         0,
		VersionNumber:   "VersionNumber",
		NodeDisplayName: "NodeDisplayName",
	}

	marshalizer := &mock.MarshalizerStub{}

	marshalizer.UnmarshalHandler = func(obj interface{}, buff []byte) error {
		(obj.(*data.Heartbeat)).Pubkey = hb.Pubkey
		(obj.(*data.Heartbeat)).Payload = hb.Payload
		(obj.(*data.Heartbeat)).Signature = hb.Signature
		(obj.(*data.Heartbeat)).ShardID = hb.ShardID
		(obj.(*data.Heartbeat)).VersionNumber = hb.VersionNumber
		(obj.(*data.Heartbeat)).NodeDisplayName = hb.NodeDisplayName

		return nil
	}

	singleSigner := &mock.SinglesignMock{}
	keyGen := &mock.KeyGenMock{
		PublicKeyFromByteArrayMock: func(b []byte) (key crypto.PublicKey, e error) {
			return &mock.PublicKeyMock{}, nil
		},
	}

	updatePubKeyWasCalled := false
	updatePidShardIdCalled := false
	mon, err := process.NewMessageProcessor(
		singleSigner,
		keyGen,
		marshalizer,
		&mock.NetworkShardingCollectorStub{
			UpdatePeerIdPublicKeyCalled: func(pid p2p.PeerID, pk []byte) {
				updatePubKeyWasCalled = true
			},
			UpdatePeerIdShardIdCalled: func(pid p2p.PeerID, shardId uint32) {
				updatePidShardIdCalled = true
			},
		},
	)
	assert.Nil(t, err)

	message := &mock.P2PMessageStub{
		FromField:      nil,
		DataField:      make([]byte, 5),
		SeqNoField:     nil,
		TopicsField:    nil,
		SignatureField: nil,
		KeyField:       nil,
		PeerField:      "",
	}

	ret, err := mon.CreateHeartbeatFromP2PMessage(message)

	assert.Nil(t, err)
	assert.NotNil(t, ret)
	assert.True(t, updatePubKeyWasCalled)
	assert.True(t, updatePidShardIdCalled)
}

func TestNewMessageProcessor_CreateHeartbeatFromP2pMessageWithNilDataShouldErr(t *testing.T) {
	t.Parallel()

	message := &mock.P2PMessageStub{
		FromField:      nil,
		DataField:      nil,
		SeqNoField:     nil,
		TopicsField:    nil,
		SignatureField: nil,
		KeyField:       nil,
		PeerField:      "",
	}

	mon, _ := process.NewMessageProcessor(
		&mock.SinglesignMock{},
		&mock.KeyGenMock{},
		&mock.MarshalizerStub{},
		&mock.NetworkShardingCollectorStub{
			UpdatePeerIdPublicKeyCalled: func(pid p2p.PeerID, pk []byte) {},
		},
	)

	ret, err := mon.CreateHeartbeatFromP2PMessage(message)

	assert.Nil(t, ret)
	assert.Equal(t, heartbeat.ErrNilDataToProcess, err)
}

func TestNewMessageProcessor_CreateHeartbeatFromP2pMessageWithUnmarshaliableDataShouldErr(t *testing.T) {
	t.Parallel()

	message := &mock.P2PMessageStub{
		FromField:      nil,
		DataField:      []byte("hello"),
		SeqNoField:     nil,
		TopicsField:    nil,
		SignatureField: nil,
		KeyField:       nil,
		PeerField:      "",
	}

	expectedErr := errors.New("marshal didn't work")

	mon, _ := process.NewMessageProcessor(
		&mock.SinglesignMock{},
		&mock.KeyGenMock{},
		&mock.MarshalizerStub{
			UnmarshalHandler: func(obj interface{}, buff []byte) error {
				return expectedErr
			},
		},
		&mock.NetworkShardingCollectorStub{
			UpdatePeerIdPublicKeyCalled: func(pid p2p.PeerID, pk []byte) {},
		},
	)

	ret, err := mon.CreateHeartbeatFromP2PMessage(message)

	assert.Nil(t, ret)
	assert.Equal(t, expectedErr, err)
}

func TestNewMessageProcessor_CreateHeartbeatFromP2PMessageWithTooLongLengthsShouldErr(t *testing.T) {
	t.Parallel()

	length := 129
	buff := make([]byte, length)

	for i := 0; i < length; i++ {
		buff[i] = byte(97)
	}
	bigNodeName := string(buff)

	hb := data.Heartbeat{
		Payload:         []byte("Payload"),
		Pubkey:          []byte("PubKey"),
		Signature:       []byte("signed"),
		ShardID:         0,
		VersionNumber:   "VersionNumber",
		NodeDisplayName: bigNodeName,
	}

	marshalizer := &mock.MarshalizerStub{}

	marshalizer.UnmarshalHandler = func(obj interface{}, buff []byte) error {
		(obj.(*data.Heartbeat)).Pubkey = hb.Pubkey
		(obj.(*data.Heartbeat)).Payload = hb.Payload
		(obj.(*data.Heartbeat)).Signature = hb.Signature
		(obj.(*data.Heartbeat)).ShardID = hb.ShardID
		(obj.(*data.Heartbeat)).VersionNumber = hb.VersionNumber
		(obj.(*data.Heartbeat)).NodeDisplayName = hb.NodeDisplayName

		return nil
	}

	singleSigner := &mock.SinglesignMock{}

	keyGen := &mock.KeyGenMock{
		PublicKeyFromByteArrayMock: func(b []byte) (key crypto.PublicKey, e error) {
			return &mock.PublicKeyMock{}, nil
		},
	}

	mon, err := process.NewMessageProcessor(
		singleSigner,
		keyGen,
		marshalizer,
		&mock.NetworkShardingCollectorStub{
			UpdatePeerIdPublicKeyCalled: func(pid p2p.PeerID, pk []byte) {},
		},
	)
	assert.Nil(t, err)

	message := &mock.P2PMessageStub{
		FromField:      nil,
		DataField:      make([]byte, 5),
		SeqNoField:     nil,
		TopicsField:    nil,
		SignatureField: nil,
		KeyField:       nil,
		PeerField:      "",
	}

	ret, err := mon.CreateHeartbeatFromP2PMessage(message)

	assert.Nil(t, ret)
	assert.True(t, errors.Is(err, heartbeat.ErrPropertyTooLong))
}

func TestNewMessageProcessor_CreateHeartbeatFromP2pNilMessageShouldErr(t *testing.T) {
	t.Parallel()

	mon, _ := process.NewMessageProcessor(
		&mock.SinglesignMock{},
		&mock.KeyGenMock{},
		&mock.MarshalizerStub{},
		&mock.NetworkShardingCollectorStub{
			UpdatePeerIdPublicKeyCalled: func(pid p2p.PeerID, pk []byte) {},
		},
	)

	ret, err := mon.CreateHeartbeatFromP2PMessage(nil)

	assert.Nil(t, ret)
	assert.Equal(t, heartbeat.ErrNilMessage, err)
}
