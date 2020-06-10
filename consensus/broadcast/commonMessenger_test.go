package broadcast_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/broadcast"
	"github.com/ElrondNetwork/elrond-go/consensus/mock"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBlockBody() *block.Body {
	return &block.Body{
		MiniBlocks: []*block.MiniBlock{
			{
				TxHashes:        [][]byte{[]byte("txh1"), []byte("txh2")},
				ReceiverShardID: 0,
				SenderShardID:   0,
				Type:            0,
			},
		},
	}
}

func TestCommonMessenger_BroadcastConsensusMessageShouldErrWhenSignMessageFail(t *testing.T) {
	err := errors.New("sign message error")
	marshalizerMock := &mock.MarshalizerMock{}
	messengerMock := &mock.MessengerStub{}
	privateKeyMock := &mock.PrivateKeyMock{}
	shardCoordinatorMock := &mock.ShardCoordinatorMock{}
	singleSignerMock := &mock.SingleSignerMock{
		SignStub: func(private crypto.PrivateKey, msg []byte) ([]byte, error) {
			return nil, err
		},
	}

	cm, _ := broadcast.NewCommonMessenger(
		marshalizerMock,
		messengerMock,
		privateKeyMock,
		shardCoordinatorMock,
		singleSignerMock,
	)

	msg := &consensus.Message{}
	err2 := cm.BroadcastConsensusMessage(msg)
	assert.Equal(t, err, err2)
}

func TestCommonMessenger_BroadcastConsensusMessageShouldWork(t *testing.T) {
	marshalizerMock := &mock.MarshalizerMock{}
	messengerMock := &mock.MessengerStub{
		BroadcastCalled: func(topic string, buff []byte) {
		},
	}
	privateKeyMock := &mock.PrivateKeyMock{}
	shardCoordinatorMock := &mock.ShardCoordinatorMock{}
	singleSignerMock := &mock.SingleSignerMock{
		SignStub: func(private crypto.PrivateKey, msg []byte) ([]byte, error) {
			return []byte(""), nil
		},
	}

	cm, _ := broadcast.NewCommonMessenger(
		marshalizerMock,
		messengerMock,
		privateKeyMock,
		shardCoordinatorMock,
		singleSignerMock,
	)

	msg := &consensus.Message{}
	err := cm.BroadcastConsensusMessage(msg)
	assert.Nil(t, err)
}

func TestCommonMessenger_SignMessageShouldErrWhenMarshalFail(t *testing.T) {
	marshalizerMock := &mock.MarshalizerMock{}
	messengerMock := &mock.MessengerStub{}
	privateKeyMock := &mock.PrivateKeyMock{}
	shardCoordinatorMock := &mock.ShardCoordinatorMock{}
	singleSignerMock := &mock.SingleSignerMock{
		SignStub: func(private crypto.PrivateKey, msg []byte) ([]byte, error) {
			return []byte(""), nil
		},
	}
	marshalizerMock.Fail = true

	cm, _ := broadcast.NewCommonMessenger(
		marshalizerMock,
		messengerMock,
		privateKeyMock,
		shardCoordinatorMock,
		singleSignerMock,
	)

	msg := &consensus.Message{}
	_, err := cm.SignMessage(msg)
	assert.Equal(t, err, mock.ErrMockMarshalizer)
}

func TestCommonMessenger_SignMessageShouldErrWhenSignFail(t *testing.T) {
	err := errors.New("sign message error")
	marshalizerMock := &mock.MarshalizerMock{}
	messengerMock := &mock.MessengerStub{}
	privateKeyMock := &mock.PrivateKeyMock{}
	shardCoordinatorMock := &mock.ShardCoordinatorMock{}
	singleSignerMock := &mock.SingleSignerMock{
		SignStub: func(private crypto.PrivateKey, msg []byte) ([]byte, error) {
			return nil, err
		},
	}

	cm, _ := broadcast.NewCommonMessenger(
		marshalizerMock,
		messengerMock,
		privateKeyMock,
		shardCoordinatorMock,
		singleSignerMock,
	)

	msg := &consensus.Message{}
	_, err2 := cm.SignMessage(msg)
	assert.Equal(t, err, err2)
}

func TestSubroundEndRound_ExtractMiniBlocksAndTransactionsShouldWork(t *testing.T) {
	t.Parallel()

	miniBlocks := make(map[uint32][]byte, 0)
	transactions := make(map[string][][]byte, 0)

	miniBlocks[1] = []byte("mbs_shard_1")
	miniBlocks[core.MetachainShardId] = []byte("mbs_shard_meta")
	miniBlocks[2] = []byte("mbs_shard_2")

	transactions["transactions_0_1"] = [][]byte{
		[]byte("tx1_shard_1"),
		[]byte("tx2_shard_1"),
		[]byte("tx3_shard_1"),
	}

	transactions["transactions_0_META"] = [][]byte{
		[]byte("tx1_shard_meta"),
		[]byte("tx2_shard_meta"),
		[]byte("tx3_shard_meta"),
	}

	transactions["transactions_0_2"] = [][]byte{
		[]byte("tx1_shard_2"),
		[]byte("tx2_shard_2"),
		[]byte("tx3_shard_2"),
	}

	marshalizerMock := &mock.MarshalizerMock{}
	messengerMock := &mock.MessengerStub{
		BroadcastCalled: func(topic string, buff []byte) {
		},
	}
	privateKeyMock := &mock.PrivateKeyMock{}
	shardCoordinatorMock := &mock.ShardCoordinatorMock{}
	singleSignerMock := &mock.SingleSignerMock{
		SignStub: func(private crypto.PrivateKey, msg []byte) ([]byte, error) {
			return []byte(""), nil
		},
	}

	cm, _ := broadcast.NewCommonMessenger(
		marshalizerMock,
		messengerMock,
		privateKeyMock,
		shardCoordinatorMock,
		singleSignerMock,
	)

	metaMiniBlocks, metaTransactions := cm.ExtractMetaMiniBlocksAndTransactions(miniBlocks, transactions)

	require.Equal(t, 2, len(miniBlocks))
	require.Equal(t, 2, len(transactions))
	require.Equal(t, 1, len(metaMiniBlocks))
	require.Equal(t, 1, len(metaTransactions))

	assert.Nil(t, miniBlocks[core.MetachainShardId])
	assert.Nil(t, transactions["transactions_0_META"])
	assert.NotNil(t, metaMiniBlocks[core.MetachainShardId])
	assert.NotNil(t, metaTransactions["transactions_0_META"])
}
