package node_test

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/ElrondNetwork/elrond-go/node/mock"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/ElrondNetwork/elrond-go/testscommon/dblookupext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInternalMiniBlock_NotFoundInStorerShouldErr(t *testing.T) {
	t.Parallel()

	uint64Converter := mock.NewNonceHashConverterMock()
	coreComponentsMock := getDefaultCoreComponents()
	coreComponentsMock.UInt64ByteSliceConv = uint64Converter
	processComponentsMock := getDefaultProcessComponents()
	processComponentsMock.HistoryRepositoryInternal = &dblookupext.HistoryRepositoryStub{}
	processComponentsMock.ShardCoord = &testscommon.ShardsCoordinatorMock{}
	dataComponentsMock := getDefaultDataComponents()
	storerMock := mock.NewStorerMock()
	dataComponentsMock.Store = &mock.ChainStorerMock{
		GetStorerCalled: func(_ dataRetriever.UnitType) storage.Storer {
			return storerMock
		},
		GetCalled: func(_ dataRetriever.UnitType, _ []byte) ([]byte, error) {
			return nil, nil
		},
	}

	n, _ := node.NewNode(
		node.WithCoreComponents(coreComponentsMock),
		node.WithProcessComponents(processComponentsMock),
		node.WithDataComponents(dataComponentsMock),
	)

	miniBlock, err := n.GetInternalMiniBlock(common.Proto, hex.EncodeToString([]byte("dummyhash")))
	assert.Error(t, err)
	assert.Nil(t, miniBlock)
}

func TestInternalMiniBlock_NilUint64ByteSliceConverterShouldErr(t *testing.T) {
	t.Parallel()

	coreComponentsMock := getDefaultCoreComponents()
	processComponentsMock := getDefaultProcessComponents()
	processComponentsMock.HistoryRepositoryInternal = &dblookupext.HistoryRepositoryStub{}
	processComponentsMock.ShardCoord = &testscommon.ShardsCoordinatorMock{}
	dataComponentsMock := getDefaultDataComponents()
	dataComponentsMock.Store = &mock.ChainStorerMock{}

	n, _ := node.NewNode(
		node.WithCoreComponents(coreComponentsMock),
		node.WithProcessComponents(processComponentsMock),
		node.WithDataComponents(dataComponentsMock),
	)

	miniBlock, err := n.GetInternalMiniBlock(common.Proto, hex.EncodeToString([]byte("dummyhash")))
	assert.Nil(t, err)
	assert.Nil(t, miniBlock)
}

func TestGetInternalMiniBlock_WrongEncodedHashShoudFail(t *testing.T) {
	t.Parallel()

	historyProc := &dblookupext.HistoryRepositoryStub{}
	uint64Converter := mock.NewNonceHashConverterMock()
	coreComponentsMock := getDefaultCoreComponents()
	coreComponentsMock.UInt64ByteSliceConv = uint64Converter
	processComponentsMock := getDefaultProcessComponents()
	processComponentsMock.HistoryRepositoryInternal = historyProc
	processComponentsMock.ShardCoord = &testscommon.ShardsCoordinatorMock{
		NoShards:     1,
		CurrentShard: 1,
	}
	dataComponentsMock := getDefaultDataComponents()
	dataComponentsMock.Store = &mock.ChainStorerMock{}

	n, _ := node.NewNode(
		node.WithCoreComponents(coreComponentsMock),
		node.WithProcessComponents(processComponentsMock),
		node.WithDataComponents(dataComponentsMock),
	)

	blk, err := n.GetInternalMiniBlock(common.Proto, "wronghashformat")
	assert.Error(t, err)
	assert.Nil(t, blk)
}

func TestGetInternalMiniBlock_InvalidOutportFormat_ShouldFail(t *testing.T) {
	t.Parallel()

	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")

	n, _ := prepareInternalMiniBlockNode(headerHash)

	blk, err := n.GetInternalMiniBlock(2, hex.EncodeToString(headerHash))
	assert.Equal(t, node.ErrInvalidOutportFormat, err)
	assert.Nil(t, blk)
}

func TestGetInternalMiniBlock_InvalidOutportFormat_ShouldWork(t *testing.T) {
	t.Parallel()

	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")

	n, miniBlockBytes := prepareInternalMiniBlockNode(headerHash)

	blk, err := n.GetInternalMiniBlock(common.Proto, hex.EncodeToString(headerHash))
	assert.Nil(t, err)
	assert.Equal(t, miniBlockBytes, blk)
}

func TestGetInternalMiniBlock__InvalidOutportFormat_ShouldWork(t *testing.T) {
	t.Parallel()

	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")

	n, miniBlockBytes := prepareInternalMiniBlockNode(headerHash)
	miniBlock := &block.MiniBlock{}
	err := json.Unmarshal(miniBlockBytes, miniBlock)
	require.Nil(t, err)

	blk, err := n.GetInternalMiniBlock(common.Internal, hex.EncodeToString(headerHash))
	assert.Nil(t, err)
	assert.Equal(t, miniBlock, blk)
}

func prepareInternalMiniBlockNode(headerHash []byte) (*node.Node, []byte) {
	nonce := uint64(1)

	historyProc := &dblookupext.HistoryRepositoryStub{
		IsEnabledCalled: func() bool {
			return true
		},
		GetEpochByHashCalled: func(_ []byte) (uint32, error) {
			return 1, nil
		},
	}
	uint64Converter := mock.NewNonceHashConverterMock()
	coreComponentsMock := getDefaultCoreComponents()
	storerMock := mock.NewStorerMock()
	coreComponentsMock.UInt64ByteSliceConv = uint64Converter
	processComponentsMock := getDefaultProcessComponents()
	processComponentsMock.HistoryRepositoryInternal = historyProc
	processComponentsMock.ShardCoord = &testscommon.ShardsCoordinatorMock{
		NoShards:     1,
		CurrentShard: 1,
	}
	dataComponentsMock := getDefaultDataComponents()
	dataComponentsMock.Store = &mock.ChainStorerMock{
		GetStorerCalled: func(_ dataRetriever.UnitType) storage.Storer {
			return storerMock
		},
		GetCalled: func(_ dataRetriever.UnitType, _ []byte) ([]byte, error) {
			return headerHash, nil
		},
	}

	miniBlock := &block.MiniBlock{
		ReceiverShardID: 1,
		SenderShardID:   1,
	}

	blockBytes, _ := json.Marshal(miniBlock)
	_ = storerMock.Put(headerHash, blockBytes)

	nonceBytes := uint64Converter.ToByteSlice(nonce)
	_ = storerMock.Put(nonceBytes, headerHash)

	n, _ := node.NewNode(
		node.WithCoreComponents(coreComponentsMock),
		node.WithProcessComponents(processComponentsMock),
		node.WithDataComponents(dataComponentsMock),
	)

	return n, blockBytes
}
