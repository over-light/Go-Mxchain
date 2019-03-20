package block

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go-sandbox/data"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/block"
	"github.com/stretchr/testify/assert"
	"github.com/whyrusleeping/go-logging"
)

func init() {
	logging.SetLevel(logging.ERROR, "pubsub")
}

func TestHeaderAndMiniBlocksAreRoutedCorrectly(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	numOfShards := 6
	startingPort := 36000
	nodesPerShard := 3

	senderShard := uint32(0)
	recvShards := []uint32{1, 2}

	advertiser := createMessengerWithKadDht(context.Background(), startingPort, "")
	advertiser.Bootstrap()
	startingPort++

	nodes := createNodes(
		startingPort,
		numOfShards,
		nodesPerShard,
		getConnectableAddress(advertiser),
	)
	displayAndStartNodes(nodes)

	defer func() {
		advertiser.Close()
		for _, n := range nodes {
			n.node.Stop()
		}
	}()

	// delay for bootstrapping and topic announcement
	fmt.Println("Delaying for node bootstrap and topic announcement...")
	time.Sleep(time.Second * 5)

	body, hdr := generateHeaderAndBody(senderShard, recvShards...)
	err := nodes[0].node.BroadcastBlock(body, hdr)
	assert.Nil(t, err)

	//display, can be removed
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)

		fmt.Println(makeDisplayTable(nodes))
	}

	for _, n := range nodes {
		if n.shardId == senderShard {
			//sender shard nodes
			assert.Equal(t, int32(1), n.headersRecv)

			shards := []uint32{senderShard}
			shards = append(shards, recvShards...)

			expectedMiniblocks := getMiniBlocksHashesFromShardIds(body.(block.Body), shards...)

			assert.True(t, equalSlices(expectedMiniblocks, n.miniblocksHashes))
			continue
		}

		//all other nodes should have not got the header
		assert.Equal(t, int32(0), n.headersRecv)

		if uint32InSlice(n.shardId, recvShards) {
			//receiver shard nodes
			expectedMiniblocks := getMiniBlocksHashesFromShardIds(body.(block.Body), n.shardId)

			assert.True(t, equalSlices(expectedMiniblocks, n.miniblocksHashes))
			continue
		}

		//other shard nodes
		assert.Equal(t, int32(0), n.miniblocksRecv)
	}
}

func generateHeaderAndBody(senderShard uint32, recvShards ...uint32) (data.BodyHandler, data.HeaderHandler) {
	hdr := block.Header{
		Nonce:            0,
		PubKeysBitmap:    []byte{255, 0},
		Signature:        []byte("signature"),
		PrevHash:         []byte("prev hash"),
		TimeStamp:        uint64(time.Now().Unix()),
		Round:            1,
		Epoch:            2,
		ShardId:          senderShard,
		BlockBodyType:    block.TxBlock,
		RootHash:         []byte{255, 255},
		MiniBlockHeaders: make([]block.MiniBlockHeader, 0),
	}

	body := block.Body{
		&block.MiniBlock{
			ShardID: senderShard,
			TxHashes: [][]byte{
				testHasher.Compute("tx1"),
			},
		},
	}

	for i, recvShard := range recvShards {
		body = append(
			body,
			&block.MiniBlock{
				ShardID: recvShard,
				TxHashes: [][]byte{
					testHasher.Compute(fmt.Sprintf("tx%d", i)),
				},
			},
		)
	}

	return body, &hdr
}
