package memp2p

import (
	"encoding/binary"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/p2p"
)

var _ p2p.MessageP2P = (*message)(nil)

// Message represents a message to be sent through the in-memory network
// simulated by the Network struct.
type message struct {
	// sending PeerID, converted to []byte
	from []byte

	// the payload
	data []byte

	// leave empty
	seqNo []byte

	// topics set by the sender
	topics []string

	// leave empty
	signature []byte

	// sending PeerID, converted to []byte
	key []byte

	// sending PeerID
	peer core.PeerID
}

// NewMessage constructs a new Message instance from arguments
func newMessage(topic string, data []byte, peerID core.PeerID, seqNo uint64) *message {
	empty := make([]byte, 0)
	seqNoBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(seqNoBytes, seqNo)

	return &message{
		from:      []byte(string(peerID)),
		data:      data,
		seqNo:     seqNoBytes,
		topics:    []string{topic},
		signature: empty,
		key:       []byte(string(peerID)),
		peer:      peerID,
	}
}

// From returns the message originator's peer ID
func (msg *message) From() []byte {
	return msg.from
}

// Data returns the message payload
func (msg *message) Data() []byte {
	return msg.data
}

// SeqNo returns the message sequence number
func (msg *message) SeqNo() []byte {
	return msg.seqNo
}

// Topics returns the topic on which the message was sent
func (msg *message) Topics() []string {
	return msg.topics
}

// Signature returns the message signature
func (msg *message) Signature() []byte {
	return msg.signature
}

// Key returns the message public key (if it can not be recovered from From field)
func (msg *message) Key() []byte {
	return msg.key
}

// Peer returns the peer that originated the message
func (msg *message) Peer() core.PeerID {
	return msg.peer
}

// IsInterfaceNil returns true if there is no value under the interface
func (msg *message) IsInterfaceNil() bool {
	return msg == nil
}
