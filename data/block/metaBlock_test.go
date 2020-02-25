package block_test

import (
	"errors"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/stretchr/testify/assert"
)

func TestMetaBlock_GetEpoch(t *testing.T) {
	t.Parallel()

	epoch := uint32(1)
	m := block.MetaBlock{
		Epoch: epoch,
	}

	assert.Equal(t, epoch, m.GetEpoch())
}

func TestMetaBlock_GetShard(t *testing.T) {
	t.Parallel()

	m := block.MetaBlock{}

	assert.Equal(t, core.MetachainShardId, m.GetShardID())
}

func TestMetaBlock_GetNonce(t *testing.T) {
	t.Parallel()

	nonce := uint64(2)
	m := block.MetaBlock{
		Nonce: nonce,
	}

	assert.Equal(t, nonce, m.GetNonce())
}

func TestMetaBlock_GetPrevHash(t *testing.T) {
	t.Parallel()

	prevHash := []byte("prev hash")
	m := block.MetaBlock{
		PrevHash: prevHash,
	}

	assert.Equal(t, prevHash, m.GetPrevHash())
}

func TestMetaBlock_GetPubKeysBitmap(t *testing.T) {
	t.Parallel()

	pubKeysBitmap := []byte{10, 11, 12, 13}
	m := block.MetaBlock{
		PubKeysBitmap: pubKeysBitmap,
	}

	assert.Equal(t, pubKeysBitmap, m.GetPubKeysBitmap())
}

func TestMetaBlock_GetPrevRandSeed(t *testing.T) {
	t.Parallel()

	prevRandSeed := []byte("previous random seed")
	m := block.MetaBlock{
		PrevRandSeed: prevRandSeed,
	}

	assert.Equal(t, prevRandSeed, m.GetPrevRandSeed())
}

func TestMetaBlock_GetRandSeed(t *testing.T) {
	t.Parallel()

	randSeed := []byte("random seed")
	m := block.MetaBlock{
		RandSeed: randSeed,
	}

	assert.Equal(t, randSeed, m.GetRandSeed())
}

func TestMetaBlock_GetRootHash(t *testing.T) {
	t.Parallel()

	rootHash := []byte("root hash")
	m := block.MetaBlock{
		RootHash: rootHash,
	}

	assert.Equal(t, rootHash, m.GetRootHash())
}

func TestMetaBlock_GetRound(t *testing.T) {
	t.Parallel()

	round := uint64(1234)
	m := block.MetaBlock{
		Round: round,
	}

	assert.Equal(t, round, m.GetRound())
}

func TestMetaBlock_GetTimestamp(t *testing.T) {
	t.Parallel()

	timestamp := uint64(1000000)
	m := block.MetaBlock{
		TimeStamp: timestamp,
	}

	assert.Equal(t, timestamp, m.GetTimeStamp())
}

func TestMetaBlock_GetSignature(t *testing.T) {
	t.Parallel()

	signature := []byte("signature")
	m := block.MetaBlock{
		Signature: signature,
	}

	assert.Equal(t, signature, m.GetSignature())
}

func TestMetaBlock_GetTxCount(t *testing.T) {
	t.Parallel()

	txCount := uint32(100)
	m := block.MetaBlock{
		TxCount: txCount,
	}

	assert.Equal(t, txCount, m.GetTxCount())
}

func TestMetaBlock_SetEpoch(t *testing.T) {
	t.Parallel()

	epoch := uint32(10)
	m := block.MetaBlock{}
	m.SetEpoch(epoch)

	assert.Equal(t, epoch, m.GetEpoch())
}

func TestMetaBlock_SetNonce(t *testing.T) {
	t.Parallel()

	nonce := uint64(11)
	m := block.MetaBlock{}
	m.SetNonce(nonce)

	assert.Equal(t, nonce, m.GetNonce())
}

func TestMetaBlock_SetPrevHash(t *testing.T) {
	t.Parallel()

	prevHash := []byte("prev hash")
	m := block.MetaBlock{}
	m.SetPrevHash(prevHash)

	assert.Equal(t, prevHash, m.GetPrevHash())
}

func TestMetaBlock_SetPubKeysBitmap(t *testing.T) {
	t.Parallel()

	pubKeysBitmap := []byte{12, 13, 14, 15}
	m := block.MetaBlock{}
	m.SetPubKeysBitmap(pubKeysBitmap)

	assert.Equal(t, pubKeysBitmap, m.GetPubKeysBitmap())
}

func TestMetaBlock_SetPrevRandSeed(t *testing.T) {
	t.Parallel()

	prevRandSeed := []byte("previous random seed")
	m := block.MetaBlock{}
	m.SetPrevRandSeed(prevRandSeed)

	assert.Equal(t, prevRandSeed, m.GetPrevRandSeed())
}

func TestMetaBlock_SetRandSeed(t *testing.T) {
	t.Parallel()

	randSeed := []byte("random seed")
	m := block.MetaBlock{}
	m.SetRandSeed(randSeed)

	assert.Equal(t, randSeed, m.GetRandSeed())
}

func TestMetaBlock_SetRootHash(t *testing.T) {
	t.Parallel()

	rootHash := []byte("root hash")
	m := block.MetaBlock{}
	m.SetRootHash(rootHash)

	assert.Equal(t, rootHash, m.GetRootHash())
}

func TestMetaBlock_SetRound(t *testing.T) {
	t.Parallel()

	rootHash := []byte("root hash")
	m := block.MetaBlock{}
	m.SetRootHash(rootHash)

	assert.Equal(t, rootHash, m.GetRootHash())
}

func TestMetaBlock_SetSignature(t *testing.T) {
	t.Parallel()

	signature := []byte("signature")
	m := block.MetaBlock{}
	m.SetSignature(signature)

	assert.Equal(t, signature, m.GetSignature())
}

func TestMetaBlock_SetTimeStamp(t *testing.T) {
	t.Parallel()

	timestamp := uint64(100000)
	m := block.MetaBlock{}
	m.SetTimeStamp(timestamp)

	assert.Equal(t, timestamp, m.GetTimeStamp())
}

func TestMetaBlock_SetTxCount(t *testing.T) {
	t.Parallel()

	txCount := uint32(100)
	m := block.MetaBlock{}
	m.SetTxCount(txCount)

	assert.Equal(t, txCount, m.GetTxCount())
}

func TestMetaBlock_GetMiniBlockHeadersWithDst(t *testing.T) {
	t.Parallel()

	metaHdr := &block.MetaBlock{Round: 15}
	metaHdr.ShardInfo = make([]block.ShardData, 0)

	shardMBHeader := make([]block.ShardMiniBlockHeader, 0)
	shMBHdr1 := block.ShardMiniBlockHeader{SenderShardID: 0, ReceiverShardID: 1, Hash: []byte("hash1")}
	shMBHdr2 := block.ShardMiniBlockHeader{SenderShardID: 0, ReceiverShardID: 1, Hash: []byte("hash2")}
	shardMBHeader = append(shardMBHeader, shMBHdr1, shMBHdr2)

	shData1 := block.ShardData{ShardID: 0, HeaderHash: []byte("sh"), ShardMiniBlockHeaders: shardMBHeader}
	metaHdr.ShardInfo = append(metaHdr.ShardInfo, shData1)

	shData2 := block.ShardData{ShardID: 1, HeaderHash: []byte("sh"), ShardMiniBlockHeaders: shardMBHeader}
	metaHdr.ShardInfo = append(metaHdr.ShardInfo, shData2)

	mbDst0 := metaHdr.GetMiniBlockHeadersWithDst(0)
	assert.Equal(t, 0, len(mbDst0))
	mbDst1 := metaHdr.GetMiniBlockHeadersWithDst(1)
	assert.Equal(t, len(shardMBHeader), len(mbDst1))
}

func TestMetaBlock_IsChainIDValid(t *testing.T) {
	t.Parallel()

	chainID := []byte("chainID")
	okChainID := []byte("chainID")
	wrongChainID := []byte("wrong chain ID")
	metablock := &block.MetaBlock{
		ChainID: chainID,
	}

	assert.Nil(t, metablock.CheckChainID(okChainID))
	assert.True(t, errors.Is(metablock.CheckChainID(wrongChainID), data.ErrInvalidChainID))
}
