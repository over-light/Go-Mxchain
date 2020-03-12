//go:generate protoc -I=proto -I=$GOPATH/src -I=$GOPATH/src/github.com/gogo/protobuf/protobuf  --gogoslick_out=. message.proto
package consensus

// MessageType specifies what type of message was received
type MessageType int

// NewConsensusMessage creates a new Message object
func NewConsensusMessage(
	blHeaderHash []byte,
	signatureShare []byte,
	body []byte,
	header []byte,
	pubKey []byte,
	sig []byte,
	msg int,
	roundIndex int64,
	chainID []byte,
	pubKeysBitmap []byte,
	aggregateSignature []byte,
	leaderSignature []byte,
) *Message {
	return &Message{
		BlockHeaderHash:    blHeaderHash,
		SignatureShare:     signatureShare,
		Body:               body,
		Header:             header,
		PubKey:             pubKey,
		Signature:          sig,
		MsgType:            int64(msg),
		RoundIndex:         roundIndex,
		ChainID:            chainID,
		PubKeysBitmap:      pubKeysBitmap,
		AggregateSignature: aggregateSignature,
		LeaderSignature:    leaderSignature,
	}
}
