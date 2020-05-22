package trie

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/mock"
	"github.com/stretchr/testify/assert"
)

func TestTrieSync_InterceptedNodeShouldNotBeAddedToNodesForTrieIfNodeReceived(t *testing.T) {
	t.Parallel()

	marsh, hasher := getTestMarshAndHasher()
	ts, err := NewTrieSyncer(&mock.RequestHandlerStub{}, &mock.CacherMock{}, &patriciaMerkleTrie{}, 0, "trieNodes")
	assert.Nil(t, err)
	assert.NotNil(t, ts)

	bn, collapsedBn := getBnAndCollapsedBn(marsh, hasher)
	encodedNode, err := collapsedBn.getEncodedNode()
	assert.Nil(t, err)

	interceptedNode, err := NewInterceptedTrieNode(encodedNode, marsh, hasher)
	assert.Nil(t, err)

	hash := "nodeHash"
	ts.nodesForTrie[hash] = trieNodeInfo{
		trieNode: bn,
		received: true,
	}

	ts.trieNodeIntercepted([]byte(hash), interceptedNode)

	nodeInfo, ok := ts.nodesForTrie[hash]
	assert.True(t, ok)
	assert.Equal(t, bn, nodeInfo.trieNode)
}
