package processor

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	heartbeatMessages "github.com/ElrondNetwork/elrond-go/heartbeat"
	p2pFactory "github.com/ElrondNetwork/elrond-go/p2p/factory"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/heartbeat"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	processP2P "github.com/ElrondNetwork/elrond-go/process/p2p"
	"github.com/stretchr/testify/assert"
)

func createMockArgDirectConnectionInfoInterceptorProcessor() ArgDirectConnectionInfoInterceptorProcessor {
	return ArgDirectConnectionInfoInterceptorProcessor{
		PeerShardMapper: &mock.PeerShardMapperStub{},
	}
}

func TestNewDirectConnectionInfoInterceptorProcessor(t *testing.T) {
	t.Parallel()

	t.Run("nil peer shard mapper should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgDirectConnectionInfoInterceptorProcessor()
		args.PeerShardMapper = nil

		processor, err := NewDirectConnectionInfoInterceptorProcessor(args)
		assert.Equal(t, process.ErrNilPeerShardMapper, err)
		assert.True(t, check.IfNil(processor))
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		processor, err := NewDirectConnectionInfoInterceptorProcessor(createMockArgDirectConnectionInfoInterceptorProcessor())
		assert.Nil(t, err)
		assert.False(t, check.IfNil(processor))
	})
}

func TestDirectConnectionInfoInterceptorProcessor_Save(t *testing.T) {
	t.Parallel()

	t.Run("invalid message should error", func(t *testing.T) {
		t.Parallel()

		wasCalled := false
		args := createMockArgDirectConnectionInfoInterceptorProcessor()
		args.PeerShardMapper = &mock.PeerShardMapperStub{
			PutPeerIdShardIdCalled: func(pid core.PeerID, shardId uint32) {
				wasCalled = true
			},
		}

		processor, err := NewDirectConnectionInfoInterceptorProcessor(args)
		assert.Nil(t, err)
		assert.False(t, check.IfNil(processor))

		// provide heartbeat as intercepted data
		arg := heartbeat.ArgBaseInterceptedHeartbeat{
			Marshaller: &marshal.GogoProtoMarshalizer{},
		}
		arg.DataBuff, _ = arg.Marshaller.Marshal(&heartbeatMessages.HeartbeatV2{})
		ihb, _ := heartbeat.NewInterceptedHeartbeat(arg)

		err = processor.Save(ihb, "", "")
		assert.Equal(t, process.ErrWrongTypeAssertion, err)
		assert.False(t, wasCalled)
	})
	t.Run("invalid shard should error", func(t *testing.T) {
		t.Parallel()

		wasCalled := false
		args := createMockArgDirectConnectionInfoInterceptorProcessor()
		args.PeerShardMapper = &mock.PeerShardMapperStub{
			PutPeerIdShardIdCalled: func(pid core.PeerID, shardId uint32) {
				wasCalled = true
			},
		}

		processor, err := NewDirectConnectionInfoInterceptorProcessor(args)
		assert.Nil(t, err)
		assert.False(t, check.IfNil(processor))

		msg := &p2pFactory.DirectConnectionInfo{
			ShardId: "invalid shard",
		}
		marshaller := marshal.GogoProtoMarshalizer{}
		dataBuff, _ := marshaller.Marshal(msg)
		arg := processP2P.ArgInterceptedDirectConnectionInfo{
			Marshaller:  &marshaller,
			DataBuff:    dataBuff,
			NumOfShards: 10,
		}
		data, _ := processP2P.NewInterceptedDirectConnectionInfo(arg)

		err = processor.Save(data, "", "")
		assert.NotNil(t, err)
		assert.False(t, wasCalled)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		wasCalled := false
		args := createMockArgDirectConnectionInfoInterceptorProcessor()
		args.PeerShardMapper = &mock.PeerShardMapperStub{
			PutPeerIdShardIdCalled: func(pid core.PeerID, shardId uint32) {
				wasCalled = true
			},
		}

		processor, err := NewDirectConnectionInfoInterceptorProcessor(args)
		assert.Nil(t, err)
		assert.False(t, check.IfNil(processor))

		msg := &p2pFactory.DirectConnectionInfo{
			ShardId: "5",
		}
		marshaller := marshal.GogoProtoMarshalizer{}
		dataBuff, _ := marshaller.Marshal(msg)
		arg := processP2P.ArgInterceptedDirectConnectionInfo{
			Marshaller:  &marshaller,
			DataBuff:    dataBuff,
			NumOfShards: 10,
		}
		data, _ := processP2P.NewInterceptedDirectConnectionInfo(arg)

		err = processor.Save(data, "", "")
		assert.Nil(t, err)
		assert.True(t, wasCalled)
	})
}

func TestDirectConnectionInfoInterceptorProcessor_DisabledMethod(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r != nil {
			assert.Fail(t, "should not panic")
		}
	}()

	processor, err := NewDirectConnectionInfoInterceptorProcessor(createMockArgDirectConnectionInfoInterceptorProcessor())
	assert.Nil(t, err)
	assert.False(t, check.IfNil(processor))

	err = processor.Validate(nil, "")
	assert.Nil(t, err)

	processor.RegisterHandler(nil)

}
