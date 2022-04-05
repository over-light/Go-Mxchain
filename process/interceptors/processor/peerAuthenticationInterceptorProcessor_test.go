package processor_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go-core/core"
	heartbeatMessages "github.com/ElrondNetwork/elrond-go/heartbeat"
	heartbeatMocks "github.com/ElrondNetwork/elrond-go/heartbeat/mock"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/heartbeat"
	"github.com/ElrondNetwork/elrond-go/process/interceptors/processor"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/ElrondNetwork/elrond-go/testscommon/p2pmocks"
	"github.com/stretchr/testify/assert"
)

type interceptedDataHandler interface {
	SizeInBytes() int
	Message() interface{}
}

func createPeerAuthenticationInterceptorProcessArg() processor.ArgPeerAuthenticationInterceptorProcessor {
	return processor.ArgPeerAuthenticationInterceptorProcessor{
		PeerAuthenticationCacher: testscommon.NewCacherStub(),
		PeerShardMapper:          &p2pmocks.NetworkShardingCollectorStub{},
		Marshaller:               testscommon.MarshalizerMock{},
		HardforkTrigger:          &heartbeatMocks.HardforkTriggerStub{},
	}
}

func createInterceptedPeerAuthentication() *heartbeatMessages.PeerAuthentication {
	payload := &heartbeatMessages.Payload{
		Timestamp:       time.Now().Unix(),
		HardforkMessage: "hardfork message",
	}
	marshalizer := mock.MarshalizerMock{}
	payloadBytes, _ := marshalizer.Marshal(payload)

	return &heartbeatMessages.PeerAuthentication{
		Pubkey:           []byte("public key"),
		Signature:        []byte("signature"),
		Pid:              []byte("peer id"),
		Payload:          payloadBytes,
		PayloadSignature: []byte("payload signature"),
	}
}

func createMockInterceptedPeerAuthentication() process.InterceptedData {
	arg := heartbeat.ArgInterceptedPeerAuthentication{
		ArgBaseInterceptedHeartbeat: heartbeat.ArgBaseInterceptedHeartbeat{
			Marshalizer: &mock.MarshalizerMock{},
		},
		NodesCoordinator:      &mock.NodesCoordinatorStub{},
		SignaturesHandler:     &mock.SignaturesHandlerStub{},
		PeerSignatureHandler:  &mock.PeerSignatureHandlerStub{},
		ExpiryTimespanInSec:   30,
		HardforkTriggerPubKey: []byte("provided hardfork pub key"),
	}
	arg.DataBuff, _ = arg.Marshalizer.Marshal(createInterceptedPeerAuthentication())
	ipa, _ := heartbeat.NewInterceptedPeerAuthentication(arg)

	return ipa
}

func TestNewPeerAuthenticationInterceptorProcessor(t *testing.T) {
	t.Parallel()

	t.Run("nil cacher should error", func(t *testing.T) {
		t.Parallel()

		arg := createPeerAuthenticationInterceptorProcessArg()
		arg.PeerAuthenticationCacher = nil
		paip, err := processor.NewPeerAuthenticationInterceptorProcessor(arg)
		assert.Equal(t, process.ErrNilPeerAuthenticationCacher, err)
		assert.Nil(t, paip)
	})
	t.Run("nil peer shard mapper should error", func(t *testing.T) {
		t.Parallel()

		arg := createPeerAuthenticationInterceptorProcessArg()
		arg.PeerShardMapper = nil
		paip, err := processor.NewPeerAuthenticationInterceptorProcessor(arg)
		assert.Equal(t, process.ErrNilPeerShardMapper, err)
		assert.Nil(t, paip)
	})
	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		arg := createPeerAuthenticationInterceptorProcessArg()
		arg.Marshaller = nil
		paip, err := processor.NewPeerAuthenticationInterceptorProcessor(arg)
		assert.Equal(t, heartbeatMessages.ErrNilMarshaller, err)
		assert.Nil(t, paip)
	})
	t.Run("nil hardfork trigger should error", func(t *testing.T) {
		t.Parallel()

		arg := createPeerAuthenticationInterceptorProcessArg()
		arg.HardforkTrigger = nil
		paip, err := processor.NewPeerAuthenticationInterceptorProcessor(arg)
		assert.Equal(t, heartbeatMessages.ErrNilHardforkTrigger, err)
		assert.Nil(t, paip)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		paip, err := processor.NewPeerAuthenticationInterceptorProcessor(createPeerAuthenticationInterceptorProcessArg())
		assert.Nil(t, err)
		assert.False(t, paip.IsInterfaceNil())
	})
}

func TestPeerAuthenticationInterceptorProcessor_Save(t *testing.T) {
	t.Parallel()

	t.Run("invalid data should error", func(t *testing.T) {
		t.Parallel()

		paip, err := processor.NewPeerAuthenticationInterceptorProcessor(createPeerAuthenticationInterceptorProcessArg())
		assert.Nil(t, err)
		assert.False(t, paip.IsInterfaceNil())
		assert.Equal(t, process.ErrWrongTypeAssertion, paip.Save(nil, "", ""))
	})
	t.Run("invalid peer auth data should error", func(t *testing.T) {
		t.Parallel()

		providedData := createMockInterceptedHeartbeat() // unable to cast to intercepted peer auth
		wasCalled := false
		args := createPeerAuthenticationInterceptorProcessArg()
		args.PeerShardMapper = &p2pmocks.NetworkShardingCollectorStub{
			UpdatePeerIDPublicKeyPairCalled: func(pid core.PeerID, pk []byte) {
				wasCalled = true
			},
		}

		paip, err := processor.NewPeerAuthenticationInterceptorProcessor(args)
		assert.Nil(t, err)
		assert.False(t, paip.IsInterfaceNil())
		assert.Equal(t, process.ErrWrongTypeAssertion, paip.Save(providedData, "", ""))
		assert.False(t, wasCalled)
	})
	t.Run("unmarshal returns error", func(t *testing.T) {
		t.Parallel()

		expectedError := errors.New("expected error")
		args := createPeerAuthenticationInterceptorProcessArg()
		args.Marshaller = &testscommon.MarshalizerStub{
			UnmarshalCalled: func(obj interface{}, buff []byte) error {
				return expectedError
			},
		}
		paip, err := processor.NewPeerAuthenticationInterceptorProcessor(args)
		assert.Nil(t, err)
		assert.False(t, paip.IsInterfaceNil())

		err = paip.Save(createMockInterceptedPeerAuthentication(), "", "")
		assert.Equal(t, expectedError, err)
	})
	t.Run("trigger received returns error", func(t *testing.T) {
		t.Parallel()

		expectedError := errors.New("expected error")
		args := createPeerAuthenticationInterceptorProcessArg()
		args.HardforkTrigger = &heartbeatMocks.HardforkTriggerStub{
			TriggerReceivedCalled: func(payload []byte, data []byte, pkBytes []byte) (bool, error) {
				return true, expectedError
			},
		}
		paip, err := processor.NewPeerAuthenticationInterceptorProcessor(args)
		assert.Nil(t, err)
		assert.False(t, paip.IsInterfaceNil())

		err = paip.Save(createMockInterceptedPeerAuthentication(), "", "")
		assert.Equal(t, expectedError, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		providedIPA := createMockInterceptedPeerAuthentication()
		providedIPAHandler := providedIPA.(interceptedDataHandler)
		providedIPAMessage := providedIPAHandler.Message().(heartbeatMessages.PeerAuthentication)
		wasPutCalled := false
		providedPid := core.PeerID("pid")
		arg := createPeerAuthenticationInterceptorProcessArg()
		arg.PeerAuthenticationCacher = &testscommon.CacherStub{
			PutCalled: func(key []byte, value interface{}, sizeInBytes int) (evicted bool) {
				assert.True(t, bytes.Equal(providedPid.Bytes(), key))
				ipa := value.(heartbeatMessages.PeerAuthentication)
				assert.Equal(t, providedIPAMessage.Pid, ipa.Pid)
				assert.Equal(t, providedIPAMessage.Payload, ipa.Payload)
				assert.Equal(t, providedIPAMessage.Signature, ipa.Signature)
				assert.Equal(t, providedIPAMessage.PayloadSignature, ipa.PayloadSignature)
				assert.Equal(t, providedIPAMessage.Pubkey, ipa.Pubkey)
				wasPutCalled = true
				return false
			},
		}
		wasUpdatePeerIDPublicKeyPairCalled := false
		arg.PeerShardMapper = &p2pmocks.NetworkShardingCollectorStub{
			UpdatePeerIDPublicKeyPairCalled: func(pid core.PeerID, pk []byte) {
				wasUpdatePeerIDPublicKeyPairCalled = true
				assert.Equal(t, providedIPAMessage.Pid, pid.Bytes())
				assert.Equal(t, providedIPAMessage.Pubkey, pk)
			},
		}

		paip, err := processor.NewPeerAuthenticationInterceptorProcessor(arg)
		assert.Nil(t, err)
		assert.False(t, paip.IsInterfaceNil())

		err = paip.Save(providedIPA, providedPid, "")
		assert.Nil(t, err)
		assert.True(t, wasPutCalled)
		assert.True(t, wasUpdatePeerIDPublicKeyPairCalled)
	})
}

func TestPeerAuthenticationInterceptorProcessor_Validate(t *testing.T) {
	t.Parallel()

	paip, err := processor.NewPeerAuthenticationInterceptorProcessor(createPeerAuthenticationInterceptorProcessArg())
	assert.Nil(t, err)
	assert.False(t, paip.IsInterfaceNil())
	assert.Nil(t, paip.Validate(nil, ""))
}

func TestPeerAuthenticationInterceptorProcessor_RegisterHandler(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			assert.Fail(t, "should not panic")
		}
	}()

	paip, err := processor.NewPeerAuthenticationInterceptorProcessor(createPeerAuthenticationInterceptorProcessArg())
	assert.Nil(t, err)
	assert.False(t, paip.IsInterfaceNil())
	paip.RegisterHandler(nil)
}
