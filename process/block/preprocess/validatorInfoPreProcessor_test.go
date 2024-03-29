package preprocess

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/ElrondNetwork/elrond-go/testscommon/hashingMocks"
	"github.com/stretchr/testify/assert"
)

func TestNewValidatorInfoPreprocessor_NilHasherShouldErr(t *testing.T) {
	t.Parallel()

	rtp, err := NewValidatorInfoPreprocessor(
		nil,
		&testscommon.MarshalizerMock{},
		&testscommon.BlockSizeComputationStub{},
	)

	assert.Nil(t, rtp)
	assert.Equal(t, process.ErrNilHasher, err)
}

func TestNewValidatorInfoPreprocessor_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	rtp, err := NewValidatorInfoPreprocessor(
		&hashingMocks.HasherMock{},
		nil,
		&testscommon.BlockSizeComputationStub{},
	)

	assert.Nil(t, rtp)
	assert.Equal(t, process.ErrNilMarshalizer, err)
}

func TestNewValidatorInfoPreprocessor_NilBlockSizeComputationHandlerShouldErr(t *testing.T) {
	t.Parallel()

	rtp, err := NewValidatorInfoPreprocessor(
		&hashingMocks.HasherMock{},
		&testscommon.MarshalizerMock{},
		nil,
	)

	assert.Nil(t, rtp)
	assert.Equal(t, process.ErrNilBlockSizeComputationHandler, err)
}

func TestNewValidatorInfoPreprocessor_OkValsShouldWork(t *testing.T) {
	t.Parallel()

	rtp, err := NewValidatorInfoPreprocessor(
		&hashingMocks.HasherMock{},
		&testscommon.MarshalizerMock{},
		&testscommon.BlockSizeComputationStub{},
	)
	assert.Nil(t, err)
	assert.NotNil(t, rtp)
}

func TestNewValidatorInfoPreprocessor_CreateMarshalizedDataShouldWork(t *testing.T) {
	t.Parallel()

	rtp, _ := NewValidatorInfoPreprocessor(
		&hashingMocks.HasherMock{},
		&testscommon.MarshalizerMock{},
		&testscommon.BlockSizeComputationStub{},
	)

	hash := make([][]byte, 0)
	res, err := rtp.CreateMarshalizedData(hash)

	assert.Nil(t, err)
	assert.Equal(t, 0, len(res))
}

func TestNewValidatorInfoPreprocessor_ProcessMiniBlockInvalidMiniBlockTypeShouldErr(t *testing.T) {
	t.Parallel()

	rtp, _ := NewValidatorInfoPreprocessor(
		&hashingMocks.HasherMock{},
		&testscommon.MarshalizerMock{},
		&testscommon.BlockSizeComputationStub{},
	)

	txHashes := make([][]byte, 0)
	mb1 := block.MiniBlock{
		TxHashes:        txHashes,
		ReceiverShardID: 1,
		SenderShardID:   0,
		Type:            0,
	}

	preProcessorExecutionInfoHandlerMock := &testscommon.PreProcessorExecutionInfoHandlerMock{
		GetNumOfCrossInterMbsAndTxsCalled: getNumOfCrossInterMbsAndTxsZero,
	}

	_, _, _, err := rtp.ProcessMiniBlock(&mb1, haveTimeTrue, haveAdditionalTimeFalse, false, false, -1, preProcessorExecutionInfoHandlerMock)
	assert.Equal(t, process.ErrWrongTypeInMiniBlock, err)
}

func TestNewValidatorInfoPreprocessor_ProcessMiniBlockShouldWork(t *testing.T) {
	t.Parallel()

	rtp, _ := NewValidatorInfoPreprocessor(
		&hashingMocks.HasherMock{},
		&testscommon.MarshalizerMock{},
		&testscommon.BlockSizeComputationStub{},
	)

	txHashes := make([][]byte, 0)
	mb1 := block.MiniBlock{
		TxHashes:        txHashes,
		ReceiverShardID: 1,
		SenderShardID:   core.MetachainShardId,
		Type:            block.PeerBlock,
	}

	preProcessorExecutionInfoHandlerMock := &testscommon.PreProcessorExecutionInfoHandlerMock{
		GetNumOfCrossInterMbsAndTxsCalled: getNumOfCrossInterMbsAndTxsZero,
	}

	_, _, _, err := rtp.ProcessMiniBlock(&mb1, haveTimeTrue, haveAdditionalTimeFalse, false, false, -1, preProcessorExecutionInfoHandlerMock)
	assert.Nil(t, err)
}

func TestNewValidatorInfoPreprocessor_ProcessMiniBlockNotFromMeta(t *testing.T) {
	t.Parallel()

	rtp, _ := NewValidatorInfoPreprocessor(
		&hashingMocks.HasherMock{},
		&testscommon.MarshalizerMock{},
		&testscommon.BlockSizeComputationStub{},
	)

	txHashes := make([][]byte, 0)
	mb1 := block.MiniBlock{
		TxHashes:        txHashes,
		ReceiverShardID: 1,
		SenderShardID:   0,
		Type:            block.PeerBlock,
	}

	preProcessorExecutionInfoHandlerMock := &testscommon.PreProcessorExecutionInfoHandlerMock{
		GetNumOfCrossInterMbsAndTxsCalled: getNumOfCrossInterMbsAndTxsZero,
	}

	_, _, _, err := rtp.ProcessMiniBlock(&mb1, haveTimeTrue, haveAdditionalTimeFalse, false, false, -1, preProcessorExecutionInfoHandlerMock)
	assert.Equal(t, process.ErrValidatorInfoMiniBlockNotFromMeta, err)
}

func TestNewValidatorInfoPreprocessor_RestorePeerBlockIntoPools(t *testing.T) {
	t.Parallel()

	hasher := &hashingMocks.HasherMock{}
	marshalizer := &testscommon.MarshalizerMock{}
	blockSizeComputation := &testscommon.BlockSizeComputationStub{}

	rtp, _ := NewValidatorInfoPreprocessor(
		hasher,
		marshalizer,
		blockSizeComputation,
	)

	txHashes := [][]byte{[]byte("tx_hash1")}
	mb1 := block.MiniBlock{
		TxHashes:        txHashes,
		ReceiverShardID: core.AllShardId,
		SenderShardID:   core.MetachainShardId,
		Type:            block.PeerBlock,
	}

	blockBody := &block.Body{}
	blockBody.MiniBlocks = append(blockBody.MiniBlocks, &mb1)
	miniBlockPool := testscommon.NewCacherMock()

	marshalizedMb, _ := marshalizer.Marshal(mb1)
	mbHash := hasher.Compute(string(marshalizedMb))

	foundMb, ok := miniBlockPool.Get(mbHash)
	assert.Nil(t, foundMb)
	assert.False(t, ok)

	numRestoredTxs, err := rtp.RestoreBlockDataIntoPools(blockBody, miniBlockPool)
	assert.Equal(t, 1, numRestoredTxs)
	assert.Nil(t, err)

	foundMb, ok = miniBlockPool.Get(mbHash)
	assert.NotNil(t, foundMb)
	assert.True(t, ok)
}

func TestNewValidatorInfoPreprocessor_RestoreOtherBlockTypeIntoPoolsShouldNotRestore(t *testing.T) {
	t.Parallel()

	hasher := &hashingMocks.HasherMock{}
	marshalizer := &testscommon.MarshalizerMock{}
	blockSizeComputation := &testscommon.BlockSizeComputationStub{}

	rtp, _ := NewValidatorInfoPreprocessor(
		hasher,
		marshalizer,
		blockSizeComputation,
	)

	txHashes := [][]byte{[]byte("tx_hash1")}
	mb1 := block.MiniBlock{
		TxHashes:        txHashes,
		ReceiverShardID: core.AllShardId,
		SenderShardID:   core.MetachainShardId,
		Type:            block.TxBlock,
	}

	blockBody := &block.Body{}
	blockBody.MiniBlocks = append(blockBody.MiniBlocks, &mb1)
	miniBlockPool := testscommon.NewCacherMock()

	marshalizedMb, _ := marshalizer.Marshal(mb1)
	mbHash := hasher.Compute(string(marshalizedMb))

	foundMb, ok := miniBlockPool.Get(mbHash)
	assert.Nil(t, foundMb)
	assert.False(t, ok)

	numRestoredTxs, err := rtp.RestoreBlockDataIntoPools(blockBody, miniBlockPool)
	assert.Equal(t, 0, numRestoredTxs)
	assert.Nil(t, err)

	foundMb, ok = miniBlockPool.Get(mbHash)
	assert.Nil(t, foundMb)
	assert.False(t, ok)
}

func TestNewValidatorInfoPreprocessor_RemovePeerBlockFromPool(t *testing.T) {
	t.Parallel()

	hasher := &hashingMocks.HasherMock{}
	marshalizer := &testscommon.MarshalizerMock{}
	blockSizeComputation := &testscommon.BlockSizeComputationStub{}

	rtp, _ := NewValidatorInfoPreprocessor(
		hasher,
		marshalizer,
		blockSizeComputation,
	)

	txHashes := [][]byte{[]byte("tx_hash1")}
	mb1 := block.MiniBlock{
		TxHashes:        txHashes,
		ReceiverShardID: core.AllShardId,
		SenderShardID:   core.MetachainShardId,
		Type:            block.PeerBlock,
	}

	marshalizedMb, _ := marshalizer.Marshal(mb1)
	mbHash := hasher.Compute(string(marshalizedMb))

	blockBody := &block.Body{}
	blockBody.MiniBlocks = append(blockBody.MiniBlocks, &mb1)
	miniBlockPool := testscommon.NewCacherMock()
	miniBlockPool.Put(mbHash, marshalizedMb, len(marshalizedMb))

	foundMb, ok := miniBlockPool.Get(mbHash)
	assert.NotNil(t, foundMb)
	assert.True(t, ok)

	err := rtp.RemoveBlockDataFromPools(blockBody, miniBlockPool)
	assert.Nil(t, err)

	foundMb, ok = miniBlockPool.Get(mbHash)
	assert.Nil(t, foundMb)
	assert.False(t, ok)
}

func TestNewValidatorInfoPreprocessor_RemoveOtherBlockTypeFromPoolShouldNotRemove(t *testing.T) {
	t.Parallel()

	hasher := &hashingMocks.HasherMock{}
	marshalizer := &testscommon.MarshalizerMock{}
	blockSizeComputation := &testscommon.BlockSizeComputationStub{}

	rtp, _ := NewValidatorInfoPreprocessor(
		hasher,
		marshalizer,
		blockSizeComputation,
	)

	txHashes := [][]byte{[]byte("tx_hash1")}
	mb1 := block.MiniBlock{
		TxHashes:        txHashes,
		ReceiverShardID: core.AllShardId,
		SenderShardID:   core.MetachainShardId,
		Type:            block.TxBlock,
	}

	marshalizedMb, _ := marshalizer.Marshal(mb1)
	mbHash := hasher.Compute(string(marshalizedMb))

	blockBody := &block.Body{}
	blockBody.MiniBlocks = append(blockBody.MiniBlocks, &mb1)
	miniBlockPool := testscommon.NewCacherMock()
	miniBlockPool.Put(mbHash, marshalizedMb, len(marshalizedMb))

	foundMb, ok := miniBlockPool.Get(mbHash)
	assert.NotNil(t, foundMb)
	assert.True(t, ok)

	err := rtp.RemoveBlockDataFromPools(blockBody, miniBlockPool)
	assert.Nil(t, err)

	foundMb, ok = miniBlockPool.Get(mbHash)
	assert.NotNil(t, foundMb)
	assert.True(t, ok)
}
