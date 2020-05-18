package trigger_test

import (
	"encoding/hex"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/endProcess"
	"github.com/ElrondNetwork/elrond-go/update"
	"github.com/ElrondNetwork/elrond-go/update/mock"
	"github.com/ElrondNetwork/elrond-go/update/trigger"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/assert"
)

func createMockArgHardforkTrigger() trigger.ArgHardforkTrigger {
	return trigger.ArgHardforkTrigger{
		TriggerPubKeyBytes:        []byte("trigger"),
		SelfPubKeyBytes:           []byte("self"),
		Enabled:                   true,
		EnabledAuthenticated:      true,
		ArgumentParser:            vmcommon.NewAtArgumentParser(),
		EpochProvider:             &mock.EpochHandlerStub{},
		ExportFactoryHandler:      &mock.ExportFactoryHandlerStub{},
		CloseAfterExportInMinutes: 0,
		ChanStopNodeProcess:       make(chan endProcess.ArgEndProcess),
		EpochConfirmedNotifier:    &mock.EpochStartNotifierStub{},
	}
}

func TestNewTrigger_EmptyTriggerPubKeyBytesShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	arg.TriggerPubKeyBytes = nil
	trig, err := trigger.NewTrigger(arg)

	assert.True(t, errors.Is(err, update.ErrInvalidValue))
	assert.True(t, check.IfNil(trig))
}

func TestNewTrigger_EmptySelfPubKeyBytesShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	arg.SelfPubKeyBytes = nil
	trig, err := trigger.NewTrigger(arg)

	assert.True(t, errors.Is(err, update.ErrInvalidValue))
	assert.True(t, check.IfNil(trig))
}

func TestNewTrigger_ShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	trig, err := trigger.NewTrigger(arg)

	assert.Nil(t, err)
	assert.False(t, check.IfNil(trig))
}

//------- Trigger

func TestTrigger_TriggerNotEnabledShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	arg.Enabled = false
	trig, _ := trigger.NewTrigger(arg)

	err := trig.Trigger(0)
	assert.Equal(t, update.ErrTriggerNotEnabled, err)

	_, wasTriggered := trig.RecordedTriggerMessage()
	assert.False(t, wasTriggered)
}

func TestTrigger_TriggerEnabledShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	trig, _ := trigger.NewTrigger(arg)

	payload, wasTriggered := trig.RecordedTriggerMessage()
	assert.Nil(t, payload)
	assert.False(t, wasTriggered)

	err := trig.Trigger(0)

	// delay as to execute the async calls
	time.Sleep(time.Second)

	payload, wasTriggered = trig.RecordedTriggerMessage()

	assert.Nil(t, err)
	assert.Nil(t, payload)
	assert.True(t, wasTriggered)
}

func TestTrigger_TriggerCalledTwiceShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	trig, _ := trigger.NewTrigger(arg)

	payload, wasTriggered := trig.RecordedTriggerMessage()
	assert.Nil(t, payload)
	assert.False(t, wasTriggered)

	err := trig.Trigger(0)

	// delay as to execute the async calls
	time.Sleep(time.Second)

	payload, wasTriggered = trig.RecordedTriggerMessage()

	assert.Nil(t, err)
	assert.Nil(t, payload)
	assert.True(t, wasTriggered)

	err = trig.Trigger(0)

	assert.Equal(t, update.ErrTriggerAlreadyDone, err)
}

func TestTrigger_TriggerReceivedNotAHardforkTriggerRetNilButNotCall(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	arg.Enabled = false
	trig, _ := trigger.NewTrigger(arg)

	isHardfork, err := trig.TriggerReceived(nil, make([]byte, 0), nil)
	assert.Nil(t, err)
	assert.False(t, isHardfork)

	_, wasTriggered := trig.RecordedTriggerMessage()
	assert.False(t, wasTriggered)
}

func TestTrigger_TriggerReceivedNotEnabledShouldRetNilButNotCall(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	arg.Enabled = false
	trig, _ := trigger.NewTrigger(arg)
	data := []byte(trigger.HardforkTriggerString + trigger.PayloadSeparator + fmt.Sprintf("%d", 0))

	isHardfork, err := trig.TriggerReceived(nil, data, nil)
	assert.Nil(t, err)
	assert.True(t, isHardfork)

	_, wasTriggered := trig.RecordedTriggerMessage()
	assert.False(t, wasTriggered)
}

func TestTrigger_TriggerReceivedNotEnabledAuthenticatedShouldRetNilButNotCall(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	arg.EnabledAuthenticated = false
	trig, _ := trigger.NewTrigger(arg)
	data := []byte(trigger.HardforkTriggerString + trigger.PayloadSeparator + fmt.Sprintf("%d", 0))

	isHardfork, err := trig.TriggerReceived(nil, data, nil)
	assert.Nil(t, err)
	assert.True(t, isHardfork)

	_, wasTriggered := trig.RecordedTriggerMessage()
	assert.False(t, wasTriggered)
}

func TestTrigger_TriggerReceivedPubkeysMismatchShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	trig, _ := trigger.NewTrigger(arg)
	data := []byte(trigger.HardforkTriggerString + trigger.PayloadSeparator + fmt.Sprintf("%d", 0))

	pubkey := []byte("invalid pubkey")
	isHardfork, err := trig.TriggerReceived(nil, data, pubkey)
	assert.Equal(t, update.ErrTriggerPubKeyMismatch, err)
	assert.True(t, isHardfork)

	_, wasTriggered := trig.RecordedTriggerMessage()
	assert.False(t, wasTriggered)
}

func TestTrigger_TriggerReceivedNotEnoughTokensShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	trig, _ := trigger.NewTrigger(arg)
	data := []byte(trigger.HardforkTriggerString)

	isHardfork, err := trig.TriggerReceived(nil, data, arg.TriggerPubKeyBytes)
	assert.True(t, errors.Is(err, update.ErrIncorrectHardforkMessage))
	assert.True(t, isHardfork)

	_, wasTriggered := trig.RecordedTriggerMessage()
	assert.False(t, wasTriggered)
}

func TestTrigger_TriggerReceivedNotAnIntShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	trig, _ := trigger.NewTrigger(arg)
	data := []byte(trigger.HardforkTriggerString + trigger.PayloadSeparator + hex.EncodeToString([]byte("not-an-int")))

	isHardfork, err := trig.TriggerReceived(nil, data, arg.TriggerPubKeyBytes)
	assert.True(t, errors.Is(err, update.ErrIncorrectHardforkMessage))
	assert.True(t, isHardfork)

	_, wasTriggered := trig.RecordedTriggerMessage()
	assert.False(t, wasTriggered)
}

func TestTrigger_TriggerReceivedOutOfGracePeriodShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	trig, _ := trigger.NewTrigger(arg)
	currentTimeStamp := time.Now().Unix()
	trig.SetTimeHandler(func() int64 {
		return currentTimeStamp
	})
	messageTimeStamp := currentTimeStamp - int64(trigger.HardforkGracePeriod.Seconds()) - 1
	data := []byte(trigger.HardforkTriggerString + trigger.PayloadSeparator + fmt.Sprintf("%d", messageTimeStamp))

	isHardfork, err := trig.TriggerReceived(nil, data, arg.TriggerPubKeyBytes)
	assert.True(t, errors.Is(err, update.ErrIncorrectHardforkMessage))
	assert.True(t, isHardfork)

	_, wasTriggered := trig.RecordedTriggerMessage()
	assert.False(t, wasTriggered)
}

func TestTrigger_TriggerReceivedShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	trig, _ := trigger.NewTrigger(arg)
	payloadReceived := []byte("original message")
	currentTimeStamp := time.Now().Unix()
	trig.SetTimeHandler(func() int64 {
		return currentTimeStamp
	})
	messageTimeStamp := currentTimeStamp - int64(trigger.HardforkGracePeriod.Seconds())
	data := []byte(trigger.HardforkTriggerString +
		trigger.PayloadSeparator + hex.EncodeToString([]byte(fmt.Sprintf("%d", messageTimeStamp))) +
		trigger.PayloadSeparator + hex.EncodeToString([]byte(fmt.Sprintf("%d", 0))))

	payload, wasTriggered := trig.RecordedTriggerMessage()
	assert.Nil(t, payload)
	assert.False(t, wasTriggered)

	isHardfork, err := trig.TriggerReceived(payloadReceived, data, arg.TriggerPubKeyBytes)
	assert.True(t, isHardfork)

	// delay as to execute the async calls
	time.Sleep(time.Second)

	payload, wasTriggered = trig.RecordedTriggerMessage()

	assert.Nil(t, err)
	assert.Equal(t, payloadReceived, payload)
	assert.True(t, wasTriggered)
}

func TestTrigger_TriggerReceivedCreatePayloadShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	trig, _ := trigger.NewTrigger(arg)

	data := trig.CreateData()
	payloadReceived := []byte("original message")

	numCloseCalled := int32(0)
	cs := &mock.CloserStub{
		CloseCalled: func() error {
			atomic.AddInt32(&numCloseCalled, 1)
			return nil
		},
	}
	_ = trig.AddCloser(cs)

	isHardfork, err := trig.TriggerReceived(payloadReceived, data, arg.TriggerPubKeyBytes)

	assert.True(t, isHardfork)
	assert.Nil(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&numCloseCalled))
}

func TestTrigger_TriggerReceivedOneCloseErrorsShouldContinueCalling(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	trig, _ := trigger.NewTrigger(arg)

	data := trig.CreateData()
	payloadReceived := []byte("original message")

	numCloseCalled := int32(0)
	cs1 := &mock.CloserStub{
		CloseCalled: func() error {
			atomic.AddInt32(&numCloseCalled, 1)
			return errors.New("expected error")
		},
	}
	cs2 := &mock.CloserStub{
		CloseCalled: func() error {
			atomic.AddInt32(&numCloseCalled, 1)
			return nil
		},
	}

	_ = trig.AddCloser(cs1)
	_ = trig.AddCloser(cs2)

	_, _ = trig.TriggerReceived(payloadReceived, data, arg.TriggerPubKeyBytes)

	assert.Equal(t, int32(2), atomic.LoadInt32(&numCloseCalled))
}

//------- IsSelfTrigger

func TestTrigger_IsSelfTrigger(t *testing.T) {
	t.Parallel()

	arg1 := createMockArgHardforkTrigger()
	trig1, _ := trigger.NewTrigger(arg1)

	assert.False(t, trig1.IsSelfTrigger())

	arg2 := createMockArgHardforkTrigger()
	arg2.SelfPubKeyBytes = arg2.TriggerPubKeyBytes
	trig2, _ := trigger.NewTrigger(arg2)

	assert.True(t, trig2.IsSelfTrigger())
}

//------- Close

func TestTrigger_AddCloserNilInstanceShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	trig, _ := trigger.NewTrigger(arg)

	err := trig.AddCloser(nil)

	assert.Equal(t, update.ErrNilCloser, err)
}

func TestTrigger_AddCloserShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockArgHardforkTrigger()
	trig, _ := trigger.NewTrigger(arg)

	cs := &mock.CloserStub{}
	err := trig.AddCloser(cs)

	assert.Nil(t, err)
	assert.True(t, trig.Closers()[0] == cs) //pointer testing
}
