package libp2p_test

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/p2p/data"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/btcsuite/btcd/btcec"
	libp2pCrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/stretchr/testify/assert"
)

func TestMessage_ShouldErrBecauseOfFromField(t *testing.T) {
	from := []byte("dummy from")

	mes := &pubsubpb.Message{
		From: from,
	}

	pMes := &pubsub.Message{Message: mes}
	m, err := libp2p.NewMessage(pMes, &testscommon.ProtoMarshalizerMock{})
	assert.Nil(t, m)
	assert.NotNil(t, err)
}

func TestMessage_ShouldWork(t *testing.T) {
	marshalizer := &testscommon.ProtoMarshalizerMock{}

	topicMessage := &data.TopicMessage{
		Timestamp: time.Now().Unix(),
		Payload:   []byte("data"),
	}

	buff, _ := marshalizer.Marshal(topicMessage)

	mes := &pubsubpb.Message{
		From: getRandomID(),
		Data: buff,
	}

	pMes := &pubsub.Message{Message: mes}
	m, err := libp2p.NewMessage(pMes, marshalizer)
	assert.Nil(t, err)
	assert.False(t, check.IfNil(m))
}

func TestMessage_From(t *testing.T) {
	from := getRandomID()

	mes := &pubsubpb.Message{
		From: from,
	}
	pMes := &pubsub.Message{Message: mes}
	m, err := libp2p.NewMessage(pMes, &testscommon.ProtoMarshalizerMock{})
	assert.Nil(t, err)

	assert.Equal(t, m.From(), from)
}

func TestMessage_Peer(t *testing.T) {
	id := getRandomID()

	mes := &pubsubpb.Message{
		From: id,
	}
	pMes := &pubsub.Message{Message: mes}

	m, _ := libp2p.NewMessage(pMes, &testscommon.ProtoMarshalizerMock{})

	assert.Equal(t, core.PeerID(id), m.Peer())
}

func getRandomID() []byte {
	prvKey, _ := ecdsa.GenerateKey(btcec.S256(), rand.Reader)
	sk := (*libp2pCrypto.Secp256k1PrivateKey)(prvKey)
	id, _ := peer.IDFromPublicKey(sk.GetPublic())

	return []byte(id)
}
