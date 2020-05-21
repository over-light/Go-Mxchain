package process

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/update/mock"
	"github.com/stretchr/testify/assert"
)

func createMockArgsNewShardBlockCreatorAfterHardFork() ArgsNewShardBlockCreatorAfterHardFork {
	return ArgsNewShardBlockCreatorAfterHardFork{
		ShardCoordinator:   mock.NewOneShardCoordinatorMock(),
		TxCoordinator:      &mock.TransactionCoordinatorMock{},
		PendingTxProcessor: &mock.PendingTransactionProcessorStub{},
		ImportHandler:      &mock.ImportHandlerStub{},
		Marshalizer:        &mock.MarshalizerMock{},
		Hasher:             &mock.HasherMock{},
	}
}

func TestNewShardBlockCreatorAfterHardFork(t *testing.T) {
	t.Parallel()

	args := createMockArgsNewShardBlockCreatorAfterHardFork()

	shardBlockCreator, err := NewShardBlockCreatorAfterHardFork(args)
	assert.NoError(t, err)
	assert.False(t, check.IfNil(shardBlockCreator))
}

func TestCreateBody(t *testing.T) {
	t.Parallel()

	mb1, mb2 := &block.MiniBlock{SenderShardID: 0}, &block.MiniBlock{SenderShardID: 1}
	mb3, mb4 := &block.MiniBlock{SenderShardID: 2}, &block.MiniBlock{SenderShardID: 3}

	args := createMockArgsNewShardBlockCreatorAfterHardFork()
	args.PendingTxProcessor = &mock.PendingTransactionProcessorStub{
		ProcessTransactionsDstMeCalled: func(mapTxs map[string]data.TransactionHandler) (block.MiniBlockSlice, error) {
			return block.MiniBlockSlice{mb1, mb2}, nil
		},
	}
	args.TxCoordinator = &mock.TransactionCoordinatorMock{
		CreatePostProcessMiniBlocksCalled: func() block.MiniBlockSlice {
			return block.MiniBlockSlice{mb3, mb4}
		},
	}

	shardBlockCreator, _ := NewShardBlockCreatorAfterHardFork(args)

	expectedBody := &block.Body{
		MiniBlocks: []*block.MiniBlock{mb1, mb2, mb3, mb4},
	}
	body, err := shardBlockCreator.createBody()
	assert.NoError(t, err)
	assert.Equal(t, expectedBody, body)
}

func TestCreateMiniBlockHeader(t *testing.T) {
	t.Parallel()

	args := createMockArgsNewShardBlockCreatorAfterHardFork()
	shardBlockCreator, _ := NewShardBlockCreatorAfterHardFork(args)

	hashTx1, hashTx2 := []byte("hash1"), []byte("hash2")
	mb1 := &block.MiniBlock{SenderShardID: 1, ReceiverShardID: 2, TxHashes: [][]byte{hashTx1}}
	mb2 := &block.MiniBlock{SenderShardID: 3, ReceiverShardID: 4, TxHashes: [][]byte{hashTx2}}
	body := &block.Body{
		MiniBlocks: []*block.MiniBlock{mb1, mb2},
	}

	mb1Hash, _ := core.CalculateHash(args.Marshalizer, args.Hasher, mb1)
	mb2Hash, _ := core.CalculateHash(args.Marshalizer, args.Hasher, mb2)

	expectedMbHeaders := []block.MiniBlockHeader{
		{
			Hash:            mb1Hash,
			TxCount:         1,
			Type:            0,
			SenderShardID:   1,
			ReceiverShardID: 2,
		},
		{
			Hash:            mb2Hash,
			TxCount:         1,
			Type:            0,
			SenderShardID:   3,
			ReceiverShardID: 4,
		},
	}
	totalTxs, mbHeaders, err := shardBlockCreator.createMiniBlockHeaders(body)
	assert.NoError(t, err)
	assert.Equal(t, 2, totalTxs)
	assert.Equal(t, expectedMbHeaders, mbHeaders)
}

func TestCreateNewBlock(t *testing.T) {
	t.Parallel()

	rootHash := []byte("rotHash")
	hashTx1, hashTx2 := []byte("hash1"), []byte("hash2")
	mb1 := &block.MiniBlock{SenderShardID: 1, ReceiverShardID: 2, TxHashes: [][]byte{hashTx1}}
	mb2 := &block.MiniBlock{SenderShardID: 3, ReceiverShardID: 4, TxHashes: [][]byte{hashTx2}}
	args := createMockArgsNewShardBlockCreatorAfterHardFork()
	args.PendingTxProcessor = &mock.PendingTransactionProcessorStub{
		ProcessTransactionsDstMeCalled: func(mapTxs map[string]data.TransactionHandler) (block.MiniBlockSlice, error) {
			return block.MiniBlockSlice{mb1}, nil
		},
		RootHashCalled: func() ([]byte, error) {
			return rootHash, nil
		},
	}
	args.TxCoordinator = &mock.TransactionCoordinatorMock{
		CreatePostProcessMiniBlocksCalled: func() block.MiniBlockSlice {
			return block.MiniBlockSlice{mb2}
		},
	}

	meta := &block.MetaBlock{}
	metaHash, _ := core.CalculateHash(args.Marshalizer, args.Hasher, meta)

	args.ImportHandler = &mock.ImportHandlerStub{
		GetHardForkMetaBlockCalled: func() *block.MetaBlock {
			return &block.MetaBlock{}
		},
	}

	shardBlockCreator, _ := NewShardBlockCreatorAfterHardFork(args)

	chainID, round, nonce, epoch := "chainId", uint64(100), uint64(90), uint32(2)

	expectedBody := &block.Body{
		MiniBlocks: []*block.MiniBlock{mb1, mb2},
	}
	mb1Hash, _ := core.CalculateHash(args.Marshalizer, args.Hasher, mb1)
	mb2Hash, _ := core.CalculateHash(args.Marshalizer, args.Hasher, mb2)

	expectedMbHeaders := []block.MiniBlockHeader{
		{
			Hash:            mb1Hash,
			TxCount:         1,
			Type:            0,
			SenderShardID:   1,
			ReceiverShardID: 2,
		},
		{
			Hash:            mb2Hash,
			TxCount:         1,
			Type:            0,
			SenderShardID:   3,
			ReceiverShardID: 4,
		},
	}

	expectedHeader := &block.Header{
		ChainID: []byte(chainID), Round: round, Nonce: nonce, Epoch: epoch,
		RandSeed: rootHash, RootHash: rootHash, PrevHash: rootHash, PrevRandSeed: rootHash,
		ReceiptsHash: []byte("receiptHash"), TxCount: 2,
		MetaBlockHashes: [][]byte{metaHash}, MiniBlockHeaders: expectedMbHeaders,
		AccumulatedFees: big.NewInt(0),
		PubKeysBitmap:   []byte{1},
		SoftwareVersion: []byte(""),
	}
	header, body, err := shardBlockCreator.CreateNewBlock(chainID, round, nonce, epoch)
	assert.NoError(t, err)
	assert.Equal(t, expectedBody, body)
	assert.Equal(t, expectedHeader, header)
}
