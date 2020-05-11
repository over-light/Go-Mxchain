package mock

import (
	"github.com/ElrondNetwork/elrond-go/p2p"
)

// MessengerStub -
type MessengerStub struct {
	IDCalled                         func() p2p.PeerID
	CloseCalled                      func() error
	CreateTopicCalled                func(name string, createChannelForTopic bool) error
	HasTopicCalled                   func(name string) bool
	HasTopicValidatorCalled          func(name string) bool
	BroadcastOnChannelCalled         func(channel string, topic string, buff []byte)
	BroadcastCalled                  func(topic string, buff []byte)
	RegisterMessageProcessorCalled   func(topic string, handler p2p.MessageProcessor) error
	BootstrapCalled                  func() error
	PeerAddressCalled                func(pid p2p.PeerID) string
	BroadcastOnChannelBlockingCalled func(channel string, topic string, buff []byte) error
	IsConnectedToTheNetworkCalled    func() bool
}

// ID -
func (ms *MessengerStub) ID() p2p.PeerID {
	if ms.IDCalled != nil {
		return ms.IDCalled()
	}

	return ""
}

// RegisterMessageProcessor -
func (ms *MessengerStub) RegisterMessageProcessor(topic string, handler p2p.MessageProcessor) error {
	if ms.RegisterMessageProcessorCalled != nil {
		return ms.RegisterMessageProcessorCalled(topic, handler)
	}
	return nil
}

// Broadcast -
func (ms *MessengerStub) Broadcast(topic string, buff []byte) {
	if ms.BroadcastCalled != nil {
		ms.BroadcastCalled(topic, buff)
	}
}

// Close -
func (ms *MessengerStub) Close() error {
	if ms.CloseCalled != nil {
		return ms.CloseCalled()
	}

	return nil
}

// CreateTopic -
func (ms *MessengerStub) CreateTopic(name string, createChannelForTopic bool) error {
	if ms.CreateTopicCalled != nil {
		return ms.CreateTopicCalled(name, createChannelForTopic)
	}

	return nil
}

// HasTopic -
func (ms *MessengerStub) HasTopic(name string) bool {
	if ms.HasTopicCalled != nil {
		return ms.HasTopicCalled(name)
	}

	return false
}

// HasTopicValidator -
func (ms *MessengerStub) HasTopicValidator(name string) bool {
	if ms.HasTopicValidatorCalled != nil {
		return ms.HasTopicValidatorCalled(name)
	}

	return false
}

// BroadcastOnChannel -
func (ms *MessengerStub) BroadcastOnChannel(channel string, topic string, buff []byte) {
	ms.BroadcastOnChannelCalled(channel, topic, buff)
}

// Bootstrap -
func (ms *MessengerStub) Bootstrap() error {
	return ms.BootstrapCalled()
}

// PeerAddress -
func (ms *MessengerStub) PeerAddress(pid p2p.PeerID) string {
	return ms.PeerAddressCalled(pid)
}

// BroadcastOnChannelBlocking -
func (ms *MessengerStub) BroadcastOnChannelBlocking(channel string, topic string, buff []byte) error {
	return ms.BroadcastOnChannelBlockingCalled(channel, topic, buff)
}

// IsConnectedToTheNetwork -
func (ms *MessengerStub) IsConnectedToTheNetwork() bool {
	return ms.IsConnectedToTheNetworkCalled()
}

// IsInterfaceNil returns true if there is no value under the interface
func (ms *MessengerStub) IsInterfaceNil() bool {
	return ms == nil
}
