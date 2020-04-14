package stateTrieSync

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/trie"
	factory2 "github.com/ElrondNetwork/elrond-go/data/trie/factory"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/requestHandlers"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/interceptors"
	"github.com/stretchr/testify/assert"
)

func TestNode_RequestInterceptTrieNodesWithMessenger(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	var nrOfShards uint32 = 1
	var shardID uint32 = 0
	var txSignPrivKeyShardId uint32 = 0
	requesterNodeAddr := "0"
	resolverNodeAddr := "1"

	fmt.Println("Requester:	")
	nRequester := integrationTests.NewTestProcessorNode(nrOfShards, shardID, txSignPrivKeyShardId, requesterNodeAddr)

	fmt.Println("Resolver:")
	nResolver := integrationTests.NewTestProcessorNode(nrOfShards, shardID, txSignPrivKeyShardId, resolverNodeAddr)
	nRequester.Node.Start()
	nResolver.Node.Start()
	defer func() {
		_ = nRequester.Node.Stop()
		_ = nResolver.Node.Stop()
	}()

	time.Sleep(time.Second)
	err := nRequester.Messenger.ConnectToPeer(integrationTests.GetConnectableAddress(nResolver.Messenger))
	assert.Nil(t, err)

	time.Sleep(integrationTests.SyncDelay)

	resolverTrie := nResolver.TrieContainer.Get([]byte(factory2.UserAccountTrie))
	for i := 0; i < 100; i++ {
		_ = resolverTrie.Update([]byte(strconv.Itoa(i)), []byte(strconv.Itoa(i)))
	}
	_ = resolverTrie.Commit()
	rootHash, _ := resolverTrie.Root()

	requesterTrie := nRequester.TrieContainer.Get([]byte(factory2.UserAccountTrie))
	nilRootHash, _ := requesterTrie.Root()
	whiteListHandler, _ := interceptors.NewWhiteListDataVerifier(&mock.CacherStub{PutCalled: func(key []byte, value interface{}) (evicted bool) {
		return false
	}})
	requestHandler, _ := requestHandlers.NewResolverRequestHandler(
		nRequester.ResolverFinder,
		&mock.RequestedItemsHandlerStub{},
		whiteListHandler,
		10000,
		nRequester.ShardCoordinator.SelfId(),
		time.Second,
	)

	waitTime := 10 * time.Second
	trieSyncer, _ := trie.NewTrieSyncer(requestHandler, nRequester.DataPool.TrieNodes(), requesterTrie, core.MetachainShardId, factory.AccountTrieNodesTopic)
	ctx, cancel := context.WithTimeout(context.Background(), waitTime)
	defer cancel()

	err = trieSyncer.StartSyncing(rootHash, ctx)
	assert.Nil(t, err)

	newRootHash, _ := requesterTrie.Root()
	assert.NotEqual(t, nilRootHash, newRootHash)
	assert.Equal(t, rootHash, newRootHash)
	_, err = requesterTrie.GetAllLeaves()
	assert.Nil(t, requesterTrie)
}
