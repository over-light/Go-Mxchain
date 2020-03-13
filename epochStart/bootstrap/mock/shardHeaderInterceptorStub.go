package mock

import (
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/p2p"
)

// ShardHeaderInterceptorStub -
type ShardHeaderInterceptorStub struct {
	ProcessReceivedMessageCalled     func(message p2p.MessageP2P, broadcastHandler func(buffToSend []byte)) error
	GetAllReceivedShardHeadersCalled func() []block.ShardData
	GetShardHeaderCalled             func(hash []byte, target int) (*block.Header, error)
}

// GetShardHeader -
func (s *ShardHeaderInterceptorStub) GetShardHeader(hash []byte, target int) (*block.Header, error) {
	if s.GetShardHeaderCalled != nil {
		return s.GetShardHeaderCalled(hash, target)
	}

	return &block.Header{}, nil
}

// ProcessReceivedMessage -
func (s *ShardHeaderInterceptorStub) ProcessReceivedMessage(message p2p.MessageP2P, broadcastHandler func(buffToSend []byte)) error {
	if s.ProcessReceivedMessageCalled != nil {
		return s.ProcessReceivedMessageCalled(message, broadcastHandler)
	}

	return nil
}

// GetAllReceivedShardHeaders -
func (s *ShardHeaderInterceptorStub) GetAllReceivedShardHeaders() []block.ShardData {
	if s.GetAllReceivedShardHeadersCalled != nil {
		return s.GetAllReceivedShardHeadersCalled()
	}

	return nil
}

// IsInterfaceNil -
func (s *ShardHeaderInterceptorStub) IsInterfaceNil() bool {
	return s == nil
}
