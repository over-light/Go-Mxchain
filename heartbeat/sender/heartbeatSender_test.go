package sender

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go/heartbeat"
	"github.com/ElrondNetwork/elrond-go/heartbeat/mock"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/stretchr/testify/assert"
)

var expectedErr = errors.New("expected error")

func createMockBaseArgs() argBaseSender {
	return argBaseSender{
		messenger:                 &mock.MessengerStub{},
		marshaller:                &mock.MarshallerMock{},
		topic:                     "topic",
		timeBetweenSends:          time.Second,
		timeBetweenSendsWhenError: time.Second,
	}
}

func createMockHeartbeatSenderArgs(argBase argBaseSender) argHeartbeatSender {
	return argHeartbeatSender{
		argBaseSender:        argBase,
		versionNumber:        "v1",
		nodeDisplayName:      "node",
		identity:             "identity",
		peerSubType:          core.RegularPeer,
		currentBlockProvider: &mock.CurrentBlockProviderStub{},
	}
}

func TestNewHeartbeatSender(t *testing.T) {
	t.Parallel()

	t.Run("nil peer messenger should error", func(t *testing.T) {
		t.Parallel()

		argBase := createMockBaseArgs()
		argBase.messenger = nil
		args := createMockHeartbeatSenderArgs(argBase)
		sender, err := newHeartbeatSender(args)

		assert.Nil(t, sender)
		assert.Equal(t, heartbeat.ErrNilMessenger, err)
	})
	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		argBase := createMockBaseArgs()
		argBase.marshaller = nil
		args := createMockHeartbeatSenderArgs(argBase)
		sender, err := newHeartbeatSender(args)

		assert.Nil(t, sender)
		assert.Equal(t, heartbeat.ErrNilMarshaller, err)
	})
	t.Run("empty topic should error", func(t *testing.T) {
		t.Parallel()

		argBase := createMockBaseArgs()
		argBase.topic = ""
		args := createMockHeartbeatSenderArgs(argBase)
		sender, err := newHeartbeatSender(args)

		assert.Nil(t, sender)
		assert.Equal(t, heartbeat.ErrEmptySendTopic, err)
	})
	t.Run("invalid time between sends should error", func(t *testing.T) {
		t.Parallel()

		argBase := createMockBaseArgs()
		argBase.timeBetweenSends = time.Second - time.Nanosecond
		args := createMockHeartbeatSenderArgs(argBase)
		sender, err := newHeartbeatSender(args)

		assert.Nil(t, sender)
		assert.True(t, errors.Is(err, heartbeat.ErrInvalidTimeDuration))
		assert.True(t, strings.Contains(err.Error(), "timeBetweenSends"))
		assert.False(t, strings.Contains(err.Error(), "timeBetweenSendsWhenError"))
	})
	t.Run("invalid time between sends should error", func(t *testing.T) {
		t.Parallel()

		argBase := createMockBaseArgs()
		argBase.timeBetweenSendsWhenError = time.Second - time.Nanosecond
		args := createMockHeartbeatSenderArgs(argBase)
		sender, err := newHeartbeatSender(args)

		assert.Nil(t, sender)
		assert.True(t, errors.Is(err, heartbeat.ErrInvalidTimeDuration))
		assert.True(t, strings.Contains(err.Error(), "timeBetweenSendsWhenError"))
	})
	t.Run("empty version number should error", func(t *testing.T) {
		t.Parallel()

		args := createMockHeartbeatSenderArgs(createMockBaseArgs())
		args.versionNumber = ""
		sender, err := newHeartbeatSender(args)

		assert.Nil(t, sender)
		assert.Equal(t, heartbeat.ErrEmptyVersionNumber, err)
	})
	t.Run("empty node display name should error", func(t *testing.T) {
		t.Parallel()

		args := createMockHeartbeatSenderArgs(createMockBaseArgs())
		args.nodeDisplayName = ""
		sender, err := newHeartbeatSender(args)

		assert.Nil(t, sender)
		assert.Equal(t, heartbeat.ErrEmptyNodeDisplayName, err)
	})
	t.Run("empty identity should error", func(t *testing.T) {
		t.Parallel()

		args := createMockHeartbeatSenderArgs(createMockBaseArgs())
		args.identity = ""
		sender, err := newHeartbeatSender(args)

		assert.Nil(t, sender)
		assert.Equal(t, heartbeat.ErrEmptyIdentity, err)
	})
	t.Run("nil current block provider should error", func(t *testing.T) {
		t.Parallel()

		args := createMockHeartbeatSenderArgs(createMockBaseArgs())
		args.currentBlockProvider = nil
		sender, err := newHeartbeatSender(args)

		assert.Nil(t, sender)
		assert.Equal(t, heartbeat.ErrNilCurrentBlockProvider, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		args := createMockHeartbeatSenderArgs(createMockBaseArgs())
		sender, err := newHeartbeatSender(args)

		assert.False(t, check.IfNil(sender))
		assert.Nil(t, err)
	})
}

func TestHeartbeatSender_Execute(t *testing.T) {
	t.Parallel()

	t.Run("execute errors, should set the error time duration value", func(t *testing.T) {
		t.Parallel()

		wasCalled := false
		argsBase := createMockBaseArgs()
		argsBase.timeBetweenSendsWhenError = time.Second * 3
		argsBase.timeBetweenSends = time.Second * 2
		argsBase.marshaller = &mock.MarshallerStub{
			MarshalHandler: func(obj interface{}) ([]byte, error) {
				return nil, expectedErr
			},
		}

		args := createMockHeartbeatSenderArgs(argsBase)
		sender, _ := newHeartbeatSender(args)
		sender.timerHandler = &mock.TimerHandlerStub{
			CreateNewTimerCalled: func(duration time.Duration) {
				assert.Equal(t, argsBase.timeBetweenSendsWhenError, duration)
				wasCalled = true
			},
		}

		sender.Execute()
		assert.True(t, wasCalled)
	})
	t.Run("execute worked, should set the normal time duration value", func(t *testing.T) {
		t.Parallel()

		wasCalled := false
		argsBase := createMockBaseArgs()
		argsBase.timeBetweenSendsWhenError = time.Second * 3
		argsBase.timeBetweenSends = time.Second * 2

		args := createMockHeartbeatSenderArgs(argsBase)
		sender, _ := newHeartbeatSender(args)
		sender.timerHandler = &mock.TimerHandlerStub{
			CreateNewTimerCalled: func(duration time.Duration) {
				assert.Equal(t, argsBase.timeBetweenSends, duration)
				wasCalled = true
			},
		}

		sender.Execute()
		assert.True(t, wasCalled)
	})
}

func TestHeartbeatSender_execute(t *testing.T) {
	t.Parallel()

	t.Run("marshal returns error first time", func(t *testing.T) {
		t.Parallel()

		argsBase := createMockBaseArgs()
		argsBase.marshaller = &mock.MarshallerStub{
			MarshalHandler: func(obj interface{}) ([]byte, error) {
				return nil, expectedErr
			},
		}

		args := createMockHeartbeatSenderArgs(argsBase)
		sender, _ := newHeartbeatSender(args)
		assert.False(t, check.IfNil(sender))

		err := sender.execute()
		assert.Equal(t, expectedErr, err)
	})
	t.Run("marshal returns error second time", func(t *testing.T) {
		t.Parallel()

		argsBase := createMockBaseArgs()
		numOfCalls := 0
		argsBase.marshaller = &mock.MarshallerStub{
			MarshalHandler: func(obj interface{}) ([]byte, error) {
				if numOfCalls < 1 {
					numOfCalls++
					return []byte(""), nil
				}

				return nil, expectedErr
			},
		}

		args := createMockHeartbeatSenderArgs(argsBase)
		sender, _ := newHeartbeatSender(args)
		assert.False(t, check.IfNil(sender))

		err := sender.execute()
		assert.Equal(t, expectedErr, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		argsBase := createMockBaseArgs()
		broadcastCalled := false
		argsBase.messenger = &mock.MessengerStub{
			BroadcastCalled: func(topic string, buff []byte) {
				assert.Equal(t, argsBase.topic, topic)
				broadcastCalled = true
			},
		}

		args := createMockHeartbeatSenderArgs(argsBase)

		args.currentBlockProvider = &mock.CurrentBlockProviderStub{
			GetCurrentBlockHeaderCalled: func() data.HeaderHandler {
				return &testscommon.HeaderHandlerStub{}
			},
		}

		sender, _ := newHeartbeatSender(args)
		assert.False(t, check.IfNil(sender))

		err := sender.execute()
		assert.Nil(t, err)
		assert.True(t, broadcastCalled)
		assert.Equal(t, uint64(1), args.currentBlockProvider.GetCurrentBlockHeader().GetNonce())
	})
}
