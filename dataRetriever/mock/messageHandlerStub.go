package mock

import (
	"github.com/ElrondNetwork/elrond-go/p2p"
)

// MessageHandlerStub -
type MessageHandlerStub struct {
	ConnectedPeersOnTopicCalled func(topic string) []p2p.PeerID
	SendToConnectedPeerCalled   func(topic string, buff []byte, peerID p2p.PeerID) error
}

// ConnectedPeersOnTopic -
func (mhs *MessageHandlerStub) ConnectedPeersOnTopic(topic string) []p2p.PeerID {
	return mhs.ConnectedPeersOnTopicCalled(topic)
}

// SendToConnectedPeer -
func (mhs *MessageHandlerStub) SendToConnectedPeer(topic string, buff []byte, peerID p2p.PeerID) error {
	return mhs.SendToConnectedPeerCalled(topic, buff, peerID)
}

// IsInterfaceNil returns true if there is no value under the interface
func (mhs *MessageHandlerStub) IsInterfaceNil() bool {
	return mhs == nil
}
