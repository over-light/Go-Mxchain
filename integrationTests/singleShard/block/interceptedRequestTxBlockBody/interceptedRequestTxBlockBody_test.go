package interceptedRequestTxBlockBody

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/stretchr/testify/assert"
)

func TestNode_GenerateSendInterceptTxBlockBodyWithNetMessenger(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	hasher := integrationTests.TestHasher
	marshalizer := integrationTests.TestMarshalizer
	var nrOfShards uint32 = 1
	var shardID uint32 = 0
	var txSignPrivKeyShardId uint32 = 0
	requesterNodeAddr := "0"
	resolverNodeAddr := "1"

	fmt.Println("Requester:	")
	nRequester := integrationTests.NewTestProcessorNode(nrOfShards, shardID, txSignPrivKeyShardId, requesterNodeAddr)

	fmt.Println("Resolver:")
	nResolver := integrationTests.NewTestProcessorNode(nrOfShards, shardID, txSignPrivKeyShardId, resolverNodeAddr)

	defer func() {
		_ = nRequester.Messenger.Close()
		_ = nResolver.Messenger.Close()
	}()

	//connect messengers together
	time.Sleep(time.Second)
	err := nRequester.Messenger.ConnectToPeer(integrationTests.GetConnectableAddress(nResolver.Messenger))
	assert.Nil(t, err)

	time.Sleep(time.Second)

	//Step 1. Generate a block body
	body := block.Body{
		MiniBlocks: []*block.MiniBlock{
			{
				ReceiverShardID: 0,
				SenderShardID:   0,
				TxHashes: [][]byte{
					hasher.Compute("tx1"),
				},
			},
		},
	}

	miniBlock := body.MiniBlocks[0]
	miniBlockHashes := make([][]byte, 1)

	txBlockBodyBuff, _ := marshalizer.Marshal(miniBlock)
	txBlockBodyHash := hasher.Compute(string(txBlockBodyBuff))

	//Step 2. resolver has the tx block body
	nResolver.DataPool.MiniBlocks().HasOrAdd(txBlockBodyHash, miniBlock, miniBlock.Size())
	fmt.Printf("Added %s to dPoolResolver\n", base64.StdEncoding.EncodeToString(txBlockBodyHash))

	//Step 3. wire up a received handler
	chanDone := make(chan bool)

	nRequester.DataPool.MiniBlocks().RegisterHandler(func(key []byte, value interface{}) {
		txBlockBodyStored, _ := nRequester.DataPool.MiniBlocks().Get(key)

		if reflect.DeepEqual(txBlockBodyStored, miniBlock) {
			chanDone <- true
		}

		assert.Equal(t, txBlockBodyStored, miniBlock)
	})

	//Step 4. request tx block body
	txBlockBodyRequester, _ := nRequester.ResolverFinder.IntraShardResolver(factory.MiniBlocksTopic)
	miniBlockRequester := txBlockBodyRequester.(dataRetriever.MiniBlocksResolver)
	miniBlockHashes[0] = txBlockBodyHash
	_ = miniBlockRequester.RequestDataFromHashArray(miniBlockHashes, 0)

	select {
	case <-chanDone:
	case <-time.After(time.Second * 10):
		assert.Fail(t, "timeout")
	}
}
