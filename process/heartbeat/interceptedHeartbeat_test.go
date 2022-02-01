package heartbeat

import (
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/mock"
	"github.com/ElrondNetwork/elrond-go/heartbeat"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/testscommon/hashingMocks"
	"github.com/stretchr/testify/assert"
)

func createDefaultInterceptedHeartbeat() *heartbeat.HeartbeatV2 {
	return &heartbeat.HeartbeatV2{
		Payload:         []byte("payload"),
		VersionNumber:   "version number",
		NodeDisplayName: "node display name",
		Identity:        "identity",
		Nonce:           123,
		PeerSubType:     uint32(core.RegularPeer),
	}
}

func createMockInterceptedHeartbeatArg(interceptedData *heartbeat.HeartbeatV2) ArgInterceptedHeartbeat {
	arg := ArgInterceptedHeartbeat{}
	arg.Marshalizer = &mock.MarshalizerMock{}
	arg.Hasher = &hashingMocks.HasherMock{}
	arg.DataBuff, _ = arg.Marshalizer.Marshal(interceptedData)

	return arg
}

func TestNewInterceptedHeartbeat(t *testing.T) {
	t.Parallel()

	t.Run("nil data buff should error", func(t *testing.T) {
		t.Parallel()

		arg := createMockInterceptedHeartbeatArg(createDefaultInterceptedHeartbeat())
		arg.DataBuff = nil

		ihb, err := NewInterceptedHeartbeat(arg)
		assert.Nil(t, ihb)
		assert.Equal(t, process.ErrNilBuffer, err)
	})
	t.Run("nil marshalizer should error", func(t *testing.T) {
		t.Parallel()

		arg := createMockInterceptedHeartbeatArg(createDefaultInterceptedHeartbeat())
		arg.Marshalizer = nil

		ihb, err := NewInterceptedHeartbeat(arg)
		assert.Nil(t, ihb)
		assert.Equal(t, process.ErrNilMarshalizer, err)
	})
	t.Run("nil hasher should error", func(t *testing.T) {
		t.Parallel()

		arg := createMockInterceptedHeartbeatArg(createDefaultInterceptedHeartbeat())
		arg.Hasher = nil

		ihb, err := NewInterceptedHeartbeat(arg)
		assert.Nil(t, ihb)
		assert.Equal(t, process.ErrNilHasher, err)
	})
	t.Run("unmarshal returns error", func(t *testing.T) {
		t.Parallel()

		arg := createMockInterceptedHeartbeatArg(createDefaultInterceptedHeartbeat())
		arg.Marshalizer = &mock.MarshalizerStub{
			UnmarshalCalled: func(obj interface{}, buff []byte) error {
				return expectedErr
			},
		}

		ihb, err := NewInterceptedHeartbeat(arg)
		assert.Nil(t, ihb)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		arg := createMockInterceptedHeartbeatArg(createDefaultInterceptedHeartbeat())

		ihb, err := NewInterceptedHeartbeat(arg)
		assert.False(t, ihb.IsInterfaceNil())
		assert.Nil(t, err)
	})
}

func Test_interceptedHeartbeat_CheckValidity(t *testing.T) {
	t.Parallel()
	t.Run("payloadProperty too short", testInterceptedHeartbeatPropertyLen(payloadProperty, false))
	t.Run("payloadProperty too short", testInterceptedHeartbeatPropertyLen(payloadProperty, true))

	t.Run("versionNumberProperty too short", testInterceptedHeartbeatPropertyLen(versionNumberProperty, false))
	t.Run("versionNumberProperty too short", testInterceptedHeartbeatPropertyLen(versionNumberProperty, true))

	t.Run("nodeDisplayNameProperty too short", testInterceptedHeartbeatPropertyLen(nodeDisplayNameProperty, false))
	t.Run("nodeDisplayNameProperty too short", testInterceptedHeartbeatPropertyLen(nodeDisplayNameProperty, true))

	t.Run("identityProperty too short", testInterceptedHeartbeatPropertyLen(identityProperty, false))
	t.Run("identityProperty too short", testInterceptedHeartbeatPropertyLen(identityProperty, true))

	t.Run("invalid peer subtype should error", func(t *testing.T) {
		t.Parallel()

		arg := createMockInterceptedHeartbeatArg(createDefaultInterceptedHeartbeat())
		ihb, _ := NewInterceptedHeartbeat(arg)
		ihb.heartbeat.PeerSubType = 123
		err := ihb.CheckValidity()
		assert.Equal(t, process.ErrInvalidPeerSubType, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		arg := createMockInterceptedHeartbeatArg(createDefaultInterceptedHeartbeat())
		ihb, _ := NewInterceptedHeartbeat(arg)
		err := ihb.CheckValidity()
		assert.Nil(t, err)
	})
}

func testInterceptedHeartbeatPropertyLen(property string, tooLong bool) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		value := []byte("")
		expectedError := process.ErrPropertyTooShort
		if tooLong {
			value = make([]byte, 130)
			expectedError = process.ErrPropertyTooLong
		}

		arg := createMockInterceptedHeartbeatArg(createDefaultInterceptedHeartbeat())
		ihb, _ := NewInterceptedHeartbeat(arg)
		switch property {
		case payloadProperty:
			ihb.heartbeat.Payload = value
		case versionNumberProperty:
			ihb.heartbeat.VersionNumber = string(value)
		case nodeDisplayNameProperty:
			ihb.heartbeat.NodeDisplayName = string(value)
		case identityProperty:
			ihb.heartbeat.Identity = string(value)
		default:
			assert.True(t, false)
		}

		err := ihb.CheckValidity()
		assert.True(t, strings.Contains(err.Error(), expectedError.Error()))
	}
}

func Test_interceptedHeartbeat_Hash(t *testing.T) {
	t.Parallel()

	arg := createMockInterceptedHeartbeatArg(createDefaultInterceptedHeartbeat())
	ihb, _ := NewInterceptedHeartbeat(arg)
	hash := ihb.Hash()
	expectedHash := arg.Hasher.Compute(string(arg.DataBuff))
	assert.Equal(t, expectedHash, hash)

	identifiers := ihb.Identifiers()
	assert.Equal(t, 1, len(identifiers))
	assert.Equal(t, expectedHash, identifiers[0])
}

func Test_interceptedHeartbeat_Getters(t *testing.T) {
	t.Parallel()

	arg := createMockInterceptedHeartbeatArg(createDefaultInterceptedHeartbeat())
	ihb, _ := NewInterceptedHeartbeat(arg)
	expectedHeartbeat := &heartbeat.HeartbeatV2{}
	err := arg.Marshalizer.Unmarshal(expectedHeartbeat, arg.DataBuff)
	assert.Nil(t, err)
	assert.True(t, ihb.IsForCurrentShard())
	assert.Equal(t, interceptedHeartbeatType, ihb.Type())
}
