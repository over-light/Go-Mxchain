package sync_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/process/sync"
	"github.com/stretchr/testify/assert"
)

func TestNewShardForkDetector_NilRounderShouldErr(t *testing.T) {
	t.Parallel()

	sfd, err := sync.NewShardForkDetector(nil, &mock.BlackListHandlerStub{})
	assert.Nil(t, sfd)
	assert.Equal(t, process.ErrNilRounder, err)
}

func TestNewShardForkDetector_NilBlackListShouldErr(t *testing.T) {
	t.Parallel()

	sfd, err := sync.NewShardForkDetector(&mock.RounderMock{}, nil)
	assert.Nil(t, sfd)
	assert.Equal(t, process.ErrNilBlackListHandler, err)
}

func TestNewShardForkDetector_OkParamsShouldWork(t *testing.T) {
	t.Parallel()

	sfd, err := sync.NewShardForkDetector(&mock.RounderMock{}, &mock.BlackListHandlerStub{})
	assert.Nil(t, err)
	assert.NotNil(t, sfd)

	assert.Equal(t, uint64(0), sfd.LastCheckpointNonce())
	assert.Equal(t, uint64(0), sfd.LastCheckpointRound())
	assert.Equal(t, uint64(0), sfd.FinalCheckpointNonce())
	assert.Equal(t, uint64(0), sfd.FinalCheckpointRound())
}

func TestShardForkDetector_AddHeaderNilHeaderShouldErr(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, _ := sync.NewShardForkDetector(rounderMock, &mock.BlackListHandlerStub{})
	err := bfd.AddHeader(nil, make([]byte, 0), process.BHProcessed, nil, nil, false)
	assert.Equal(t, sync.ErrNilHeader, err)
}

func TestShardForkDetector_AddHeaderNilHashShouldErr(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, _ := sync.NewShardForkDetector(rounderMock, &mock.BlackListHandlerStub{})
	err := bfd.AddHeader(&block.Header{}, nil, process.BHProcessed, nil, nil, false)
	assert.Equal(t, sync.ErrNilHash, err)
}

func TestShardForkDetector_AddHeaderUnsignedBlockShouldErr(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 1}
	bfd, _ := sync.NewShardForkDetector(rounderMock, &mock.BlackListHandlerStub{
		HasCalled: func(key string) bool {
			return false
		},
		AddCalled: func(key string) error {
			return nil
		},
		SweepCalled: func() {},
	})
	err := bfd.AddHeader(
		&block.Header{Nonce: 1, Round: 1},
		make([]byte, 0),
		process.BHProcessed,
		nil,
		nil,
		false)
	assert.Equal(t, sync.ErrBlockIsNotSigned, err)
}

func TestShardForkDetector_AddHeaderNotPresentShouldWork(t *testing.T) {
	t.Parallel()

	hdr := &block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")}
	hash := make([]byte, 0)
	rounderMock := &mock.RounderMock{RoundIndex: 1}
	bfd, _ := sync.NewShardForkDetector(rounderMock, &mock.BlackListHandlerStub{
		HasCalled: func(key string) bool {
			return false
		},
		AddCalled: func(key string) error {
			return nil
		},
		SweepCalled: func() {},
	})

	err := bfd.AddHeader(hdr, hash, process.BHProcessed, nil, nil, false)
	assert.Nil(t, err)

	hInfos := bfd.GetHeaders(1)
	assert.Equal(t, 1, len(hInfos))
	assert.Equal(t, hash, hInfos[0].Hash())
}

func TestShardForkDetector_AddHeaderPresentShouldAppend(t *testing.T) {
	t.Parallel()

	hdr1 := &block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")}
	hash1 := []byte("hash1")
	hdr2 := &block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")}
	hash2 := []byte("hash2")
	rounderMock := &mock.RounderMock{RoundIndex: 1}
	bfd, _ := sync.NewShardForkDetector(rounderMock, &mock.BlackListHandlerStub{
		HasCalled: func(key string) bool {
			return false
		},
		AddCalled: func(key string) error {
			return nil
		},
		SweepCalled: func() {},
	})

	_ = bfd.AddHeader(hdr1, hash1, process.BHProcessed, nil, nil, false)
	err := bfd.AddHeader(hdr2, hash2, process.BHProcessed, nil, nil, false)
	assert.Nil(t, err)

	hInfos := bfd.GetHeaders(1)
	assert.Equal(t, 2, len(hInfos))
	assert.Equal(t, hash1, hInfos[0].Hash())
	assert.Equal(t, hash2, hInfos[1].Hash())
}

func TestShardForkDetector_AddHeaderWithProcessedBlockShouldSetCheckpoint(t *testing.T) {
	t.Parallel()

	hdr1 := &block.Header{Nonce: 69, Round: 72, PubKeysBitmap: []byte("X")}
	hash1 := []byte("hash1")
	rounderMock := &mock.RounderMock{RoundIndex: 73}
	bfd, _ := sync.NewShardForkDetector(rounderMock, &mock.BlackListHandlerStub{
		HasCalled: func(key string) bool {
			return false
		},
		AddCalled: func(key string) error {
			return nil
		},
		SweepCalled: func() {},
	})
	_ = bfd.AddHeader(hdr1, hash1, process.BHProcessed, nil, nil, false)
	assert.Equal(t, hdr1.Nonce, bfd.LastCheckpointNonce())
}

func TestShardForkDetector_AddHeaderPresentShouldNotRewriteState(t *testing.T) {
	t.Parallel()

	hdr1 := &block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")}
	hash := []byte("hash1")
	hdr2 := &block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")}
	rounderMock := &mock.RounderMock{RoundIndex: 1}
	bfd, _ := sync.NewShardForkDetector(rounderMock, &mock.BlackListHandlerStub{
		HasCalled: func(key string) bool {
			return false
		},
		AddCalled: func(key string) error {
			return nil
		},
		SweepCalled: func() {},
	})

	_ = bfd.AddHeader(hdr1, hash, process.BHReceived, nil, nil, false)
	err := bfd.AddHeader(hdr2, hash, process.BHProcessed, nil, nil, false)
	assert.Nil(t, err)

	hInfos := bfd.GetHeaders(1)
	assert.Equal(t, 2, len(hInfos))
	assert.Equal(t, hash, hInfos[0].Hash())
	assert.Equal(t, process.BHReceived, hInfos[0].GetBlockHeaderState())
	assert.Equal(t, process.BHProcessed, hInfos[1].GetBlockHeaderState())
}

func TestShardForkDetector_AddHeaderHigherNonceThanRoundShouldErr(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, _ := sync.NewShardForkDetector(rounderMock, &mock.BlackListHandlerStub{
		HasCalled: func(key string) bool {
			return false
		},
		AddCalled: func(key string) error {
			return nil
		},
		SweepCalled: func() {},
	})
	err := bfd.AddHeader(
		&block.Header{Nonce: 1, Round: 0, PubKeysBitmap: []byte("X")},
		[]byte("hash1"),
		process.BHProcessed,
		nil,
		nil,
		false,
	)
	assert.Equal(t, sync.ErrHigherNonceInBlock, err)
}
