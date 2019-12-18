package block

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters/uint64ByteSlice"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/dataPool"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/resolvers"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/stretchr/testify/assert"
)

var durationTimeout = time.Second * 10

func TestNode_GenerateSendInterceptHeaderByNonceWithNetMessenger(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	hasher := integrationTests.TestHasher
	marshalizer := integrationTests.TestMarshalizer
	uint64Converter := uint64ByteSlice.NewBigEndianConverter()
	var nrOfShards uint32 = 1
	var shardID uint32 = 0
	var txSignPrivKeyShardId uint32 = 0
	requesterNodeAddr := "0"
	resolverNodeAddr := "1"

	fmt.Println("Requester:	")
	nRequester := integrationTests.NewTestProcessorNode(nrOfShards, shardID, txSignPrivKeyShardId, requesterNodeAddr)

	fmt.Println("Resolver:")
	nResolver := integrationTests.NewTestProcessorNode(nrOfShards, shardID, txSignPrivKeyShardId, resolverNodeAddr)
	_ = nRequester.Node.Start()
	_ = nResolver.Node.Start()
	defer func() {
		_ = nRequester.Node.Stop()
		_ = nResolver.Node.Stop()
	}()

	//connect messengers together
	time.Sleep(time.Second)
	err := nRequester.Messenger.ConnectToPeer(integrationTests.GetConnectableAddress(nResolver.Messenger))
	assert.Nil(t, err)

	time.Sleep(time.Second)

	hdr1, hdr2 := generateTwoHeaders(integrationTests.IntegrationTestsChainID)
	hdrBuff1, _ := marshalizer.Marshal(hdr1)
	hdrHash1 := hasher.Compute(string(hdrBuff1))
	hdrBuff2, _ := marshalizer.Marshal(hdr2)
	hdrHash2 := hasher.Compute(string(hdrBuff2))

	//resolver has the headers
	_, _ = nResolver.ShardDataPool.Headers().HasOrAdd(hdrHash1, &hdr1)

	syncMap := &dataPool.ShardIdHashSyncMap{}
	syncMap.Store(0, hdrHash1)
	nResolver.ShardDataPool.HeadersNonces().Merge(0, syncMap)
	_ = nResolver.Storage.GetStorer(dataRetriever.BlockHeaderUnit).Put(hdrHash2, hdrBuff2)
	_ = nResolver.Storage.GetStorer(dataRetriever.ShardHdrNonceHashDataUnit).Put(uint64Converter.ToByteSlice(1), hdrHash2)

	chanDone1, chanDone2 := wireUpHandler(nRequester, hdr1, hdr2)

	//request header from pool
	res, err := nRequester.ResolverFinder.IntraShardResolver(factory.HeadersTopic)
	assert.Nil(t, err)
	hdrResolver := res.(*resolvers.HeaderResolver)
	_ = hdrResolver.RequestDataFromNonce(0)

	//request header that is stored
	_ = hdrResolver.RequestDataFromNonce(1)

	testChansShouldReadBoth(t, chanDone1, chanDone2)
}

func TestNode_InterceptedHeaderWithWrongChainIDShouldBeDiscarded(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	hasher := integrationTests.TestHasher
	marshalizer := integrationTests.TestMarshalizer
	uint64Converter := uint64ByteSlice.NewBigEndianConverter()
	var nrOfShards uint32 = 1
	var shardID uint32 = 0
	var txSignPrivKeyShardId uint32 = 0
	requesterNodeAddr := "0"
	resolverNodeAddr := "1"

	fmt.Println("Requester:	")
	nRequester := integrationTests.NewTestProcessorNode(nrOfShards, shardID, txSignPrivKeyShardId, requesterNodeAddr)

	fmt.Println("Resolver:")
	nResolver := integrationTests.NewTestProcessorNode(nrOfShards, shardID, txSignPrivKeyShardId, resolverNodeAddr)
	_ = nRequester.Node.Start()
	_ = nResolver.Node.Start()
	defer func() {
		_ = nRequester.Node.Stop()
		_ = nResolver.Node.Stop()
	}()

	//connect messengers together
	time.Sleep(time.Second)
	err := nRequester.Messenger.ConnectToPeer(integrationTests.GetConnectableAddress(nResolver.Messenger))
	assert.Nil(t, err)

	time.Sleep(time.Second)

	wrongChainID := []byte("wrong chain ID")
	hdr1, hdr2 := generateTwoHeaders(wrongChainID)
	hdrBuff1, _ := marshalizer.Marshal(&hdr1)
	hdrHash1 := hasher.Compute(string(hdrBuff1))
	hdrBuff2, _ := marshalizer.Marshal(&hdr2)
	hdrHash2 := hasher.Compute(string(hdrBuff2))

	//resolver has the headers
	_, _ = nResolver.ShardDataPool.Headers().HasOrAdd(hdrHash1, &hdr1)

	syncMap := &dataPool.ShardIdHashSyncMap{}
	syncMap.Store(0, hdrHash1)
	nResolver.ShardDataPool.HeadersNonces().Merge(0, syncMap)
	_ = nResolver.Storage.GetStorer(dataRetriever.BlockHeaderUnit).Put(hdrHash2, hdrBuff2)
	_ = nResolver.Storage.GetStorer(dataRetriever.ShardHdrNonceHashDataUnit).Put(uint64Converter.ToByteSlice(1), hdrHash2)

	chanDone1, chanDone2 := wireUpHandler(nRequester, hdr1, hdr2)

	//request header from pool
	res, err := nRequester.ResolverFinder.IntraShardResolver(factory.HeadersTopic)
	assert.Nil(t, err)
	hdrResolver := res.(*resolvers.HeaderResolver)
	_ = hdrResolver.RequestDataFromNonce(0)

	//request header that is stored
	_ = hdrResolver.RequestDataFromNonce(1)

	testChansShouldReadNone(t, chanDone1, chanDone2)
}

func generateTwoHeaders(chainID []byte) (data.HeaderHandler, data.HeaderHandler) {
	hdr1 := &block.Header{
		Nonce:            0,
		PubKeysBitmap:    []byte{255, 0},
		Signature:        []byte("signature"),
		PrevHash:         []byte("prev hash"),
		TimeStamp:        uint64(time.Now().Unix()),
		Round:            1,
		Epoch:            2,
		ShardId:          0,
		BlockBodyType:    block.TxBlock,
		RootHash:         []byte{255, 255},
		PrevRandSeed:     make([]byte, 0),
		RandSeed:         make([]byte, 0),
		MiniBlockHeaders: make([]block.MiniBlockHeader, 0),
		ChainID:          chainID,
	}

	hdr2 := &block.Header{
		Nonce:            1,
		PubKeysBitmap:    []byte{255, 0},
		Signature:        []byte("signature"),
		PrevHash:         []byte("prev hash"),
		TimeStamp:        uint64(time.Now().Unix()),
		Round:            1,
		Epoch:            2,
		ShardId:          0,
		BlockBodyType:    block.TxBlock,
		RootHash:         []byte{255, 255},
		PrevRandSeed:     make([]byte, 0),
		RandSeed:         make([]byte, 0),
		MiniBlockHeaders: make([]block.MiniBlockHeader, 0),
		ChainID:          chainID,
	}

	return hdr1, hdr2
}

func wireUpHandler(
	nRequester *integrationTests.TestProcessorNode,
	hdr1 data.HeaderHandler,
	hdr2 data.HeaderHandler,
) (chan struct{}, chan struct{}) {

	//wire up a received handler
	chanDone1 := make(chan struct{}, 1)
	chanDone2 := make(chan struct{}, 1)
	nRequester.ShardDataPool.Headers().RegisterHandler(func(key []byte) {
		hdrStored, _ := nRequester.ShardDataPool.Headers().Peek(key)
		fmt.Printf("Received hash %v\n", base64.StdEncoding.EncodeToString(key))

		if reflect.DeepEqual(hdrStored, hdr1) && hdr1.GetSignature() != nil {
			fmt.Printf("Received header with hash %v\n", base64.StdEncoding.EncodeToString(key))
			chanDone1 <- struct{}{}
		}

		if reflect.DeepEqual(hdrStored, hdr2) && hdr2.GetSignature() != nil {
			fmt.Printf("Received header with hash %v\n", base64.StdEncoding.EncodeToString(key))
			chanDone2 <- struct{}{}
		}
	})

	return chanDone1, chanDone2
}

func testChansShouldReadBoth(t *testing.T, chanDone1 chan struct{}, chanDone2 chan struct{}) {
	select {
	case <-chanDone1:
	case <-time.After(durationTimeout):
		assert.Fail(t, "timeout")
		return
	}

	select {
	case <-chanDone2:
	case <-time.After(durationTimeout):
		assert.Fail(t, "timeout")
		return
	}
}

func testChansShouldReadNone(t *testing.T, chanDone1 chan struct{}, chanDone2 chan struct{}) {
	select {
	case <-chanDone1:
	case <-chanDone2:
	case <-time.After(durationTimeout):
		return
	}
	assert.Fail(t, "should have not received header with wrong chain ID")
}
