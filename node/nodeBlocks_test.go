package node_test

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/api"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/ElrondNetwork/elrond-go/node/blockAPI"
	"github.com/ElrondNetwork/elrond-go/node/mock"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/stretchr/testify/assert"
)

func TestGetBlockByHash_InvalidShardShouldErr(t *testing.T) {
	t.Parallel()

	n, _ := node.NewNode()

	blk, err := n.GetBlockByHash("invalidHash", false)
	assert.Error(t, err)
	assert.Nil(t, blk)
}

func TestGetBlockByHash_NilStoreShouldErr(t *testing.T) {
	t.Parallel()

	historyProc := &testscommon.HistoryRepositoryStub{
		IsEnabledCalled: func() bool {
			return true
		},
		GetEpochByHashCalled: func(hash []byte) (uint32, error) {
			return 1, nil
		},
	}
	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")
	uint64Converter := mock.NewNonceHashConverterMock()
	n, _ := node.NewNode(
		node.WithInternalMarshalizer(&mock.MarshalizerFake{}, 90),
		node.WithHistoryRepository(historyProc),
		node.WithShardCoordinator(mock.NewOneShardCoordinatorMock()),
		node.WithUint64ByteSliceConverter(uint64Converter),
	)

	blk, err := n.GetBlockByHash(hex.EncodeToString(headerHash), false)
	assert.Error(t, err)
	assert.Nil(t, blk)
}

func TestGetBlockByHash_NilUint64ByteSliceConverterShouldErr(t *testing.T) {
	t.Parallel()

	historyProc := &testscommon.HistoryRepositoryStub{
		IsEnabledCalled: func() bool {
			return true
		},
		GetEpochByHashCalled: func(hash []byte) (uint32, error) {
			return 1, nil
		},
	}
	storerMock := mock.NewStorerMock()
	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")
	n, _ := node.NewNode(
		node.WithInternalMarshalizer(&mock.MarshalizerFake{}, 90),
		node.WithHistoryRepository(historyProc),
		node.WithShardCoordinator(mock.NewOneShardCoordinatorMock()),
		node.WithDataStore(&mock.ChainStorerMock{
			GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
				return storerMock
			},
			GetCalled: func(unitType dataRetriever.UnitType, key []byte) ([]byte, error) {
				return headerHash, nil
			},
		}),
	)

	blk, err := n.GetBlockByHash(hex.EncodeToString(headerHash), false)
	assert.Error(t, err)
	assert.Nil(t, blk)
}

func TestGetBlockByHashFromHistoryNode(t *testing.T) {
	t.Parallel()

	historyProc := &testscommon.HistoryRepositoryStub{
		IsEnabledCalled: func() bool {
			return true
		},
		GetEpochByHashCalled: func(hash []byte) (uint32, error) {
			return 1, nil
		},
	}
	nonce := uint64(1)
	round := uint64(2)
	epoch := uint32(1)
	shardID := uint32(5)
	miniblockHeader := []byte("mbHash")
	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")

	uint64Converter := mock.NewNonceHashConverterMock()
	storerMock := mock.NewStorerMock()
	n, _ := node.NewNode(
		node.WithInternalMarshalizer(&mock.MarshalizerFake{}, 90),
		node.WithHistoryRepository(historyProc),
		node.WithShardCoordinator(mock.NewOneShardCoordinatorMock()),
		node.WithDataStore(&mock.ChainStorerMock{
			GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
				return storerMock
			},
			GetCalled: func(unitType dataRetriever.UnitType, key []byte) ([]byte, error) {
				return headerHash, nil
			},
		}),
		node.WithUint64ByteSliceConverter(uint64Converter),
	)

	header := &block.Header{
		Nonce:   nonce,
		Round:   round,
		ShardID: shardID,
		Epoch:   epoch,
		MiniBlockHeaders: []block.MiniBlockHeader{
			{Hash: miniblockHeader},
		},
		AccumulatedFees: big.NewInt(0),
		DeveloperFees:   big.NewInt(0),
	}
	blockBytes, _ := json.Marshal(header)
	_ = storerMock.Put(headerHash, blockBytes)

	nonceBytes := uint64Converter.ToByteSlice(nonce)
	_ = storerMock.Put(nonceBytes, headerHash)

	expectedBlock := &api.Block{
		Nonce: nonce,
		Round: round,
		Shard: shardID,
		Epoch: epoch,
		Hash:  hex.EncodeToString(headerHash),
		MiniBlocks: []*api.MiniBlock{
			{
				Hash: hex.EncodeToString(miniblockHeader),
				Type: block.TxBlock.String(),
			},
		},
		AccumulatedFees: "0",
		DeveloperFees:   "0",
		Status:          blockAPI.BlockStatusOnChain,
	}

	blk, err := n.GetBlockByHash(hex.EncodeToString(headerHash), false)
	assert.Nil(t, err)
	assert.Equal(t, expectedBlock, blk)
}

func TestGetBlockByHashFromNormalNode(t *testing.T) {
	t.Parallel()

	uint64Converter := mock.NewNonceHashConverterMock()

	nonce := uint64(1)
	round := uint64(2)
	epoch := uint32(1)
	miniblockHeader := []byte("mbHash")
	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")
	storerMock := mock.NewStorerMock()
	n, _ := node.NewNode(
		node.WithInternalMarshalizer(&mock.MarshalizerFake{}, 90),
		node.WithHistoryRepository(&testscommon.HistoryRepositoryStub{
			IsEnabledCalled: func() bool {
				return false
			},
		}),
		node.WithShardCoordinator(&mock.ShardCoordinatorMock{SelfShardId: core.MetachainShardId}),
		node.WithDataStore(&mock.ChainStorerMock{
			GetCalled: func(unitType dataRetriever.UnitType, key []byte) ([]byte, error) {
				return storerMock.Get(key)
			},
		}),
		node.WithUint64ByteSliceConverter(uint64Converter),
	)

	header := &block.MetaBlock{
		Nonce: nonce,
		Round: round,
		Epoch: epoch,
		MiniBlockHeaders: []block.MiniBlockHeader{
			{Hash: miniblockHeader},
		},
		AccumulatedFees:        big.NewInt(100),
		DeveloperFees:          big.NewInt(10),
		AccumulatedFeesInEpoch: big.NewInt(2000),
		DevFeesInEpoch:         big.NewInt(49),
	}
	headerBytes, _ := json.Marshal(header)
	_ = storerMock.Put(headerHash, headerBytes)

	nonceBytes := uint64Converter.ToByteSlice(nonce)
	_ = storerMock.Put(nonceBytes, headerHash)

	expectedBlock := &api.Block{
		Nonce: nonce,
		Round: round,
		Shard: core.MetachainShardId,
		Epoch: epoch,
		Hash:  hex.EncodeToString(headerHash),
		MiniBlocks: []*api.MiniBlock{
			{
				Hash: hex.EncodeToString(miniblockHeader),
				Type: block.TxBlock.String(),
			},
		},
		NotarizedBlocks:        []*api.NotarizedBlock{},
		Status:                 blockAPI.BlockStatusOnChain,
		AccumulatedFees:        "100",
		DeveloperFees:          "10",
		AccumulatedFeesInEpoch: "2000",
		DeveloperFeesInEpoch:   "49",
	}

	blk, err := n.GetBlockByHash(hex.EncodeToString(headerHash), false)
	assert.Nil(t, err)
	assert.Equal(t, expectedBlock, blk)
}

func TestGetBlockByNonce_NilStoreShouldErr(t *testing.T) {
	t.Parallel()

	historyProc := &testscommon.HistoryRepositoryStub{
		IsEnabledCalled: func() bool {
			return true
		},
		GetEpochByHashCalled: func(hash []byte) (uint32, error) {
			return 1, nil
		},
	}
	nonce := uint64(1)
	uint64Converter := mock.NewNonceHashConverterMock()
	n, _ := node.NewNode(
		node.WithInternalMarshalizer(&mock.MarshalizerFake{}, 90),
		node.WithHistoryRepository(historyProc),
		node.WithShardCoordinator(mock.NewOneShardCoordinatorMock()),
		node.WithUint64ByteSliceConverter(uint64Converter),
	)

	blk, err := n.GetBlockByNonce(nonce, false)
	assert.Error(t, err)
	assert.Nil(t, blk)
}

func TestGetBlockByNonceFromHistoryNode(t *testing.T) {
	t.Parallel()

	historyProc := &testscommon.HistoryRepositoryStub{
		IsEnabledCalled: func() bool {
			return true
		},
		GetEpochByHashCalled: func(hash []byte) (uint32, error) {
			return 1, nil
		},
	}

	nonce := uint64(1)
	round := uint64(2)
	epoch := uint32(1)
	shardID := uint32(5)
	miniblockHeader := []byte("mbHash")
	storerMock := mock.NewStorerMock()
	headerHash := "d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00"
	n, _ := node.NewNode(
		node.WithUint64ByteSliceConverter(mock.NewNonceHashConverterMock()),
		node.WithInternalMarshalizer(&mock.MarshalizerFake{}, 90),
		node.WithHistoryRepository(historyProc),
		node.WithShardCoordinator(mock.NewOneShardCoordinatorMock()),
		node.WithDataStore(&mock.ChainStorerMock{
			GetCalled: func(unitType dataRetriever.UnitType, key []byte) ([]byte, error) {
				return hex.DecodeString(headerHash)
			},
			GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
				return storerMock
			},
		}),
	)

	header := &block.Header{
		Nonce:   nonce,
		Round:   round,
		ShardID: shardID,
		Epoch:   epoch,
		MiniBlockHeaders: []block.MiniBlockHeader{
			{Hash: miniblockHeader},
		},
		AccumulatedFees: big.NewInt(1000),
		DeveloperFees:   big.NewInt(50),
	}
	headerBytes, _ := json.Marshal(header)
	_ = storerMock.Put(func() []byte { hashBytes, _ := hex.DecodeString(headerHash); return hashBytes }(), headerBytes)

	expectedBlock := &api.Block{
		Nonce: nonce,
		Round: round,
		Shard: shardID,
		Epoch: epoch,
		Hash:  headerHash,
		MiniBlocks: []*api.MiniBlock{
			{
				Hash: hex.EncodeToString(miniblockHeader),
				Type: block.TxBlock.String(),
			},
		},
		AccumulatedFees: "1000",
		DeveloperFees:   "50",
		Status:          blockAPI.BlockStatusOnChain,
	}

	blk, err := n.GetBlockByNonce(1, false)
	assert.Nil(t, err)
	assert.Equal(t, expectedBlock, blk)
}

func TestGetBlockByNonceFromNormalNode(t *testing.T) {
	t.Parallel()

	nonce := uint64(1)
	round := uint64(2)
	epoch := uint32(1)
	shardID := uint32(5)
	miniblockHeader := []byte("mbHash")
	headerHash := "d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00"
	n, _ := node.NewNode(
		node.WithUint64ByteSliceConverter(mock.NewNonceHashConverterMock()),
		node.WithInternalMarshalizer(&mock.MarshalizerFake{}, 90),
		node.WithHistoryRepository(&testscommon.HistoryRepositoryStub{
			IsEnabledCalled: func() bool {
				return false
			},
		}),
		node.WithShardCoordinator(&mock.ShardCoordinatorMock{SelfShardId: 0}),
		node.WithDataStore(&mock.ChainStorerMock{
			GetCalled: func(unitType dataRetriever.UnitType, key []byte) ([]byte, error) {
				if unitType == dataRetriever.ShardHdrNonceHashDataUnit {
					return hex.DecodeString(headerHash)
				}
				blk := &block.Header{
					Nonce:   nonce,
					Round:   round,
					ShardID: shardID,
					Epoch:   epoch,
					MiniBlockHeaders: []block.MiniBlockHeader{
						{Hash: miniblockHeader},
					},
					AccumulatedFees: big.NewInt(1000),
					DeveloperFees:   big.NewInt(50),
				}
				blockBytes, _ := json.Marshal(blk)
				return blockBytes, nil
			},
		}),
	)

	expectedBlock := &api.Block{
		Nonce: nonce,
		Round: round,
		Shard: shardID,
		Epoch: epoch,
		Hash:  headerHash,
		MiniBlocks: []*api.MiniBlock{
			{
				Hash: hex.EncodeToString(miniblockHeader),
				Type: block.TxBlock.String(),
			},
		},
		AccumulatedFees: "1000",
		DeveloperFees:   "50",
		Status:          blockAPI.BlockStatusOnChain,
	}

	blk, err := n.GetBlockByNonce(1, false)
	assert.Nil(t, err)
	assert.Equal(t, expectedBlock, blk)
}

func TestGetBlockByHashFromHistoryNode_StatusReverted(t *testing.T) {
	t.Parallel()

	historyProc := &testscommon.HistoryRepositoryStub{
		IsEnabledCalled: func() bool {
			return true
		},
		GetEpochByHashCalled: func(hash []byte) (uint32, error) {
			return 1, nil
		},
	}
	nonce := uint64(1)
	round := uint64(2)
	epoch := uint32(1)
	shardID := uint32(5)
	miniblockHeader := []byte("mbHash")
	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")

	uint64Converter := mock.NewNonceHashConverterMock()
	storerMock := mock.NewStorerMock()
	n, _ := node.NewNode(
		node.WithInternalMarshalizer(&mock.MarshalizerFake{}, 90),
		node.WithHistoryRepository(historyProc),
		node.WithShardCoordinator(mock.NewOneShardCoordinatorMock()),
		node.WithDataStore(&mock.ChainStorerMock{
			GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
				return storerMock
			},
			GetCalled: func(unitType dataRetriever.UnitType, key []byte) ([]byte, error) {
				return storerMock.Get(key)
			},
		}),
		node.WithUint64ByteSliceConverter(uint64Converter),
	)

	header := &block.Header{
		Nonce:   nonce,
		Round:   round,
		ShardID: shardID,
		Epoch:   epoch,
		MiniBlockHeaders: []block.MiniBlockHeader{
			{Hash: miniblockHeader},
		},
		AccumulatedFees: big.NewInt(500),
		DeveloperFees:   big.NewInt(55),
	}
	blockBytes, _ := json.Marshal(header)
	_ = storerMock.Put(headerHash, blockBytes)

	nonceBytes := uint64Converter.ToByteSlice(nonce)
	correctHash := []byte("correct-hash")
	_ = storerMock.Put(nonceBytes, correctHash)

	expectedBlock := &api.Block{
		Nonce: nonce,
		Round: round,
		Shard: shardID,
		Epoch: epoch,
		Hash:  hex.EncodeToString(headerHash),
		MiniBlocks: []*api.MiniBlock{
			{
				Hash: hex.EncodeToString(miniblockHeader),
				Type: block.TxBlock.String(),
			},
		},
		AccumulatedFees: "500",
		DeveloperFees:   "55",
		Status:          blockAPI.BlockStatusReverted,
	}

	blk, err := n.GetBlockByHash(hex.EncodeToString(headerHash), false)
	assert.Nil(t, err)
	assert.Equal(t, expectedBlock, blk)
}
