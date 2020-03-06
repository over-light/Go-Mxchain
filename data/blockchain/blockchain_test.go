package blockchain_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/blockchain"
	"github.com/ElrondNetwork/elrond-go/data/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewBlockChain_NilBadBlockCacheShouldError(t *testing.T) {
	t.Parallel()

	_, err := blockchain.NewBlockChain(
		nil,
	)

	assert.Equal(t, err, blockchain.ErrBadBlocksCacheNil)
}

func TestNewBlockChain_ShouldWork(t *testing.T) {
	t.Parallel()

	badBlocksStub := &mock.CacherStub{}
	blck, err := blockchain.NewBlockChain(badBlocksStub)

	assert.Nil(t, err)
	assert.False(t, check.IfNil(blck))
}

func TestBlockChain_IsBadBlock(t *testing.T) {
	t.Parallel()

	badBlocksStub := &mock.CacherStub{}
	hasReturns := true
	badBlocksStub.HasCalled = func(key []byte) bool {
		return hasReturns
	}

	b, _ := blockchain.NewBlockChain(
		badBlocksStub,
	)

	hasBadBlock := b.HasBadBlock([]byte("test"))
	assert.True(t, hasBadBlock)
}

func TestBlockChain_PutBadBlock(t *testing.T) {
	t.Parallel()

	badBlocksStub := &mock.CacherStub{}
	putCalled := false
	badBlocksStub.PutCalled = func(key []byte, value interface{}) bool {
		putCalled = true
		return true
	}

	b, _ := blockchain.NewBlockChain(
		badBlocksStub,
	)

	b.PutBadBlock([]byte("test"))
	assert.True(t, putCalled)
}

func TestBlockChain_SetNilAppStatusHandlerShouldErr(t *testing.T) {
	t.Parallel()

	b, _ := blockchain.NewBlockChain(
		&mock.CacherStub{},
	)

	err := b.SetAppStatusHandler(nil)

	assert.Equal(t, blockchain.ErrNilAppStatusHandler, err)
}

func TestBlockChain_SettersAndGetters(t *testing.T) {
	t.Parallel()

	b, _ := blockchain.NewBlockChain(
		&mock.CacherStub{},
	)

	hdr := &block.Header{}
	body := &block.Body{}
	height := int64(37)
	hdrHash := []byte("hash")

	b.SetCurrentBlockHeaderHash(hdrHash)
	b.SetNetworkHeight(height)
	b.SetGenesisHeaderHash(hdrHash)
	b.SetLocalHeight(height)
	_ = b.SetGenesisHeader(hdr)
	_ = b.SetCurrentBlockBody(body)
	_ = b.SetCurrentBlockHeader(hdr)

	assert.Equal(t, hdr, b.GetCurrentBlockHeader())
	assert.Equal(t, hdr, b.GetGenesisHeader())
	assert.Equal(t, hdrHash, b.GetCurrentBlockHeaderHash())
	assert.Equal(t, hdrHash, b.GetGenesisHeaderHash())
	assert.Equal(t, body, b.GetCurrentBlockBody())
	assert.Equal(t, height, b.GetNetworkHeight())
	assert.Equal(t, height, b.GetLocalHeight())
}
