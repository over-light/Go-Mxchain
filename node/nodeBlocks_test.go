package node_test

import (
	"encoding/hex"
	"encoding/json"
	apiBlock "github.com/ElrondNetwork/elrond-go/api/block"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"testing"

	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/ElrondNetwork/elrond-go/node/mock"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/stretchr/testify/assert"
)

func TestGetBlockByHash_InvalidShardShouldErr(t *testing.T) {
	t.Parallel()

	n, _ := node.NewNode()

	blk, err := n.GetBlockByHash("invalidHash", false)
	assert.Error(t, err)
	assert.Nil(t, blk)
}

func TestGetBlockByHashFromHistoryNode(t *testing.T) {
	t.Parallel()

	historyProc := &mock.HistoryProcessorStub{
		IsEnabledCalled: func() bool {
			return true
		},
		GetEpochForHashCalled: func(hash []byte) (uint32, error) {
			return 1, nil
		},
	}

	nonce := uint64(1)
	round := uint64(2)
	epoch := uint32(1)
	shardID := uint32(5)
	miniblockHeader := []byte("mbHash")
	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")
	n, _ := node.NewNode(
		node.WithInternalMarshalizer(&mock.MarshalizerFake{}, 90),
		node.WithHistoryProcessor(historyProc),
		node.WithShardCoordinator(mock.NewOneShardCoordinatorMock()),
		node.WithDataStore(&mock.ChainStorerMock{
			GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
				return &mock.StorerStub{
					GetFromEpochCalled: func(key []byte, epoch uint32) ([]byte, error) {
						blk := &block.Header{
							Nonce:   nonce,
							Round:   round,
							ShardID: shardID,
							Epoch:   epoch,
							MiniBlockHeaders: []block.MiniBlockHeader{
								{Hash: miniblockHeader},
							},
						}
						blockBytes, _ := json.Marshal(blk)
						return blockBytes, nil
					},
				}
			},
		}),
	)

	expectedBlock := &apiBlock.APIBlock{
		Nonce:   nonce,
		Round:   round,
		ShardID: shardID,
		Epoch:   epoch,
		Hash:    hex.EncodeToString(headerHash),
		MiniBlocks: []*apiBlock.APIMiniBlock{
			{
				Hash: hex.EncodeToString(miniblockHeader),
				Type: block.TxBlock.String(),
			},
		},
	}

	blk, err := n.GetBlockByHash(hex.EncodeToString(headerHash), false)
	assert.Nil(t, err)
	assert.Equal(t, expectedBlock, blk)
}

func TestGetBlockByHashFromNormalNode(t *testing.T) {
	t.Parallel()

	nonce := uint64(1)
	round := uint64(2)
	epoch := uint32(1)
	miniblockHeader := []byte("mbHash")
	headerHash := []byte("d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00")
	n, _ := node.NewNode(
		node.WithInternalMarshalizer(&mock.MarshalizerFake{}, 90),
		node.WithHistoryProcessor(&mock.HistoryProcessorStub{
			IsEnabledCalled: func() bool {
				return false
			},
		}),
		node.WithShardCoordinator(&mock.ShardCoordinatorMock{SelfShardId: core.MetachainShardId}),
		node.WithDataStore(&mock.ChainStorerMock{
			GetCalled: func(unitType dataRetriever.UnitType, key []byte) ([]byte, error) {
				blk := &block.MetaBlock{
					Nonce: nonce,
					Round: round,
					Epoch: epoch,
					MiniBlockHeaders: []block.MiniBlockHeader{
						{Hash: miniblockHeader},
					},
				}
				blockBytes, _ := json.Marshal(blk)
				return blockBytes, nil
			},
		}),
	)

	expectedBlock := &apiBlock.APIBlock{
		Nonce:   nonce,
		Round:   round,
		ShardID: core.MetachainShardId,
		Epoch:   epoch,
		Hash:    hex.EncodeToString(headerHash),
		MiniBlocks: []*apiBlock.APIMiniBlock{
			{
				Hash: hex.EncodeToString(miniblockHeader),
				Type: block.TxBlock.String(),
			},
		},
		NotarizedBlockHashes: []string{},
	}

	blk, err := n.GetBlockByHash(hex.EncodeToString(headerHash), false)
	assert.Nil(t, err)
	assert.Equal(t, expectedBlock, blk)
}

func TestGetBlockByNonceFromHistoryNode(t *testing.T) {
	t.Parallel()

	historyProc := &mock.HistoryProcessorStub{
		IsEnabledCalled: func() bool {
			return true
		},
		GetEpochForHashCalled: func(hash []byte) (uint32, error) {
			return 1, nil
		},
	}

	nonce := uint64(1)
	round := uint64(2)
	epoch := uint32(1)
	shardID := uint32(5)
	miniblockHeader := []byte("mbHash")
	headerHash := "d08089f2ab739520598fd7aeed08c427460fe94f286383047f3f61951afc4e00"
	n, _ := node.NewNode(
		node.WithUint64ByteSliceConverter(mock.NewNonceHashConverterMock()),
		node.WithInternalMarshalizer(&mock.MarshalizerFake{}, 90),
		node.WithHistoryProcessor(historyProc),
		node.WithShardCoordinator(mock.NewOneShardCoordinatorMock()),
		node.WithDataStore(&mock.ChainStorerMock{
			GetCalled: func(unitType dataRetriever.UnitType, key []byte) ([]byte, error) {
				return hex.DecodeString(headerHash)
			},
			GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
				return &mock.StorerStub{
					GetFromEpochCalled: func(key []byte, epoch uint32) ([]byte, error) {
						blk := &block.Header{
							Nonce:   nonce,
							Round:   round,
							ShardID: shardID,
							Epoch:   epoch,
							MiniBlockHeaders: []block.MiniBlockHeader{
								{Hash: miniblockHeader},
							},
						}
						blockBytes, _ := json.Marshal(blk)
						return blockBytes, nil
					},
				}
			},
		}),
	)

	expectedBlock := &apiBlock.APIBlock{
		Nonce:   nonce,
		Round:   round,
		ShardID: shardID,
		Epoch:   epoch,
		Hash:    headerHash,
		MiniBlocks: []*apiBlock.APIMiniBlock{
			{
				Hash: hex.EncodeToString(miniblockHeader),
				Type: block.TxBlock.String(),
			},
		},
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
		node.WithHistoryProcessor(&mock.HistoryProcessorStub{
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
				}
				blockBytes, _ := json.Marshal(blk)
				return blockBytes, nil
			},
		}),
	)

	expectedBlock := &apiBlock.APIBlock{
		Nonce:   nonce,
		Round:   round,
		ShardID: shardID,
		Epoch:   epoch,
		Hash:    headerHash,
		MiniBlocks: []*apiBlock.APIMiniBlock{
			{
				Hash: hex.EncodeToString(miniblockHeader),
				Type: block.TxBlock.String(),
			},
		},
	}

	blk, err := n.GetBlockByNonce(1, false)
	assert.Nil(t, err)
	assert.Equal(t, expectedBlock, blk)
}
