package processor

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	heartbeatMessages "github.com/ElrondNetwork/elrond-go/heartbeat"
	"github.com/ElrondNetwork/elrond-go/p2p/message"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/heartbeat"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/process/p2p"
	"github.com/stretchr/testify/assert"
)

func createMockArgPeerShardInterceptorProcessor() ArgPeerShardInterceptorProcessor {
	return ArgPeerShardInterceptorProcessor{
		PeerShardMapper: &mock.PeerShardMapperStub{},
	}
}

func TestNewPeerShardInterceptorProcessor(t *testing.T) {
	t.Parallel()

	t.Run("nil peer shard mapper should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgPeerShardInterceptorProcessor()
		args.PeerShardMapper = nil

		processor, err := NewPeerShardInterceptorProcessor(args)
		assert.Equal(t, process.ErrNilPeerShardMapper, err)
		assert.True(t, check.IfNil(processor))
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		processor, err := NewPeerShardInterceptorProcessor(createMockArgPeerShardInterceptorProcessor())
		assert.Nil(t, err)
		assert.False(t, check.IfNil(processor))
	})
}

func TestPeerShardInterceptorProcessor_Save(t *testing.T) {
	t.Parallel()

	t.Run("invalid message should error", func(t *testing.T) {
		t.Parallel()

		wasCalled := false
		args := createMockArgPeerShardInterceptorProcessor()
		args.PeerShardMapper = &mock.PeerShardMapperStub{
			PutPeerIdShardIdCalled: func(pid core.PeerID, shardId uint32) {
				wasCalled = true
			},
		}

		processor, err := NewPeerShardInterceptorProcessor(args)
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
		args := createMockArgPeerShardInterceptorProcessor()
		args.PeerShardMapper = &mock.PeerShardMapperStub{
			PutPeerIdShardIdCalled: func(pid core.PeerID, shardId uint32) {
				wasCalled = true
			},
		}

		processor, err := NewPeerShardInterceptorProcessor(args)
		assert.Nil(t, err)
		assert.False(t, check.IfNil(processor))

		msg := &message.PeerShard{
			ShardId: "invalid shard",
		}
		marshaller := marshal.GogoProtoMarshalizer{}
		dataBuff, _ := marshaller.Marshal(msg)
		arg := p2p.ArgInterceptedPeerShard{
			Marshaller:  &marshaller,
			DataBuff:    dataBuff,
			NumOfShards: 10,
		}
		data, _ := p2p.NewInterceptedPeerShard(arg)

		err = processor.Save(data, "", "")
		assert.NotNil(t, err)
		assert.False(t, wasCalled)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		wasCalled := false
		args := createMockArgPeerShardInterceptorProcessor()
		args.PeerShardMapper = &mock.PeerShardMapperStub{
			PutPeerIdShardIdCalled: func(pid core.PeerID, shardId uint32) {
				wasCalled = true
			},
		}

		processor, err := NewPeerShardInterceptorProcessor(args)
		assert.Nil(t, err)
		assert.False(t, check.IfNil(processor))

		msg := &message.PeerShard{
			ShardId: "5",
		}
		marshaller := marshal.GogoProtoMarshalizer{}
		dataBuff, _ := marshaller.Marshal(msg)
		arg := p2p.ArgInterceptedPeerShard{
			Marshaller:  &marshaller,
			DataBuff:    dataBuff,
			NumOfShards: 10,
		}
		data, _ := p2p.NewInterceptedPeerShard(arg)

		err = processor.Save(data, "", "")
		assert.Nil(t, err)
		assert.True(t, wasCalled)
	})
}

func TestPeerShardInterceptorProcessor_DisabledMethod(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r != nil {
			assert.Fail(t, "should not panic")
		}
	}()

	processor, err := NewPeerShardInterceptorProcessor(createMockArgPeerShardInterceptorProcessor())
	assert.Nil(t, err)
	assert.False(t, check.IfNil(processor))

	err = processor.Validate(nil, "")
	assert.Nil(t, err)

	processor.RegisterHandler(nil)

}
