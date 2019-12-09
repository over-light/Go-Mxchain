package sync_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/process/sync"
	"github.com/stretchr/testify/assert"
)

func TestNewMetaForkDetector_NilRounderShouldErr(t *testing.T) {
	t.Parallel()

	sfd, err := sync.NewMetaForkDetector(
		nil,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	assert.Nil(t, sfd)
	assert.Equal(t, process.ErrNilRounder, err)
}

func TestNewMetaForkDetector_NilBlackListShouldErr(t *testing.T) {
	t.Parallel()

	sfd, err := sync.NewMetaForkDetector(
		&mock.RounderMock{},
		nil,
		&mock.BlockTrackerStub{},
		0,
	)
	assert.Nil(t, sfd)
	assert.Equal(t, process.ErrNilBlackListHandler, err)
}

func TestNewMetaForkDetector_NilBlockTrackerShouldErr(t *testing.T) {
	t.Parallel()

	sfd, err := sync.NewMetaForkDetector(
		&mock.RounderMock{},
		&mock.BlackListHandlerStub{},
		nil,
		0,
	)
	assert.Nil(t, sfd)
	assert.Equal(t, process.ErrNilBlockTracker, err)
}

func TestNewMetaForkDetector_OkParamsShouldWork(t *testing.T) {
	t.Parallel()

	sfd, err := sync.NewMetaForkDetector(
		&mock.RounderMock{},
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	assert.Nil(t, err)
	assert.NotNil(t, sfd)

	assert.Equal(t, uint64(0), sfd.LastCheckpointNonce())
	assert.Equal(t, uint64(0), sfd.LastCheckpointRound())
	assert.Equal(t, uint64(0), sfd.FinalCheckpointNonce())
	assert.Equal(t, uint64(0), sfd.FinalCheckpointRound())
}

func TestMetaForkDetector_AddHeaderNilHeaderShouldErr(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, _ := sync.NewMetaForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)
	err := bfd.AddHeader(nil, make([]byte, 0), process.BHProcessed, nil, nil)
	assert.Equal(t, sync.ErrNilHeader, err)
}

func TestMetaForkDetector_AddHeaderNilHashShouldErr(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, _ := sync.NewMetaForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)
	err := bfd.AddHeader(&block.Header{}, nil, process.BHProcessed, nil, nil)
	assert.Equal(t, sync.ErrNilHash, err)
}

func TestMetaForkDetector_AddHeaderNotPresentShouldWork(t *testing.T) {
	t.Parallel()

	hdr := &block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")}
	hash := make([]byte, 0)
	rounderMock := &mock.RounderMock{RoundIndex: 1}
	bfd, _ := sync.NewMetaForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)

	err := bfd.AddHeader(hdr, hash, process.BHProcessed, nil, nil)
	assert.Nil(t, err)

	hInfos := bfd.GetHeaders(1)
	assert.Equal(t, 1, len(hInfos))
	assert.Equal(t, hash, hInfos[0].Hash())
}

func TestMetaForkDetector_AddHeaderPresentShouldAppend(t *testing.T) {
	t.Parallel()

	hdr1 := &block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")}
	hash1 := []byte("hash1")
	hdr2 := &block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")}
	hash2 := []byte("hash2")
	rounderMock := &mock.RounderMock{RoundIndex: 1}
	bfd, _ := sync.NewMetaForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)

	_ = bfd.AddHeader(hdr1, hash1, process.BHProcessed, nil, nil)
	err := bfd.AddHeader(hdr2, hash2, process.BHProcessed, nil, nil)
	assert.Nil(t, err)

	hInfos := bfd.GetHeaders(1)
	assert.Equal(t, 2, len(hInfos))
	assert.Equal(t, hash1, hInfos[0].Hash())
	assert.Equal(t, hash2, hInfos[1].Hash())
}

func TestMetaForkDetector_AddHeaderWithProcessedBlockShouldSetCheckpoint(t *testing.T) {
	t.Parallel()

	hdr1 := &block.Header{Nonce: 69, Round: 72, PubKeysBitmap: []byte("X")}
	hash1 := []byte("hash1")
	rounderMock := &mock.RounderMock{RoundIndex: 73}
	bfd, _ := sync.NewMetaForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)
	_ = bfd.AddHeader(hdr1, hash1, process.BHProcessed, nil, nil)
	assert.Equal(t, hdr1.Nonce, bfd.LastCheckpointNonce())
}

func TestMetaForkDetector_AddHeaderPresentShouldNotRewriteState(t *testing.T) {
	t.Parallel()

	hdr1 := &block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")}
	hash := []byte("hash1")
	hdr2 := &block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")}
	rounderMock := &mock.RounderMock{RoundIndex: 1}
	bfd, _ := sync.NewMetaForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)

	_ = bfd.AddHeader(hdr1, hash, process.BHReceived, nil, nil)
	err := bfd.AddHeader(hdr2, hash, process.BHProcessed, nil, nil)
	assert.Nil(t, err)

	hInfos := bfd.GetHeaders(1)
	assert.Equal(t, 2, len(hInfos))
	assert.Equal(t, hash, hInfos[0].Hash())
	assert.Equal(t, process.BHReceived, hInfos[0].GetBlockHeaderState())
	assert.Equal(t, process.BHProcessed, hInfos[1].GetBlockHeaderState())
}

func TestMetaForkDetector_AddHeaderHigherNonceThanRoundShouldErr(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, _ := sync.NewMetaForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)
	err := bfd.AddHeader(
		&block.Header{Nonce: 1, Round: 0, PubKeysBitmap: []byte("X")},
		[]byte("hash1"),
		process.BHProcessed,
		nil,
		nil,
	)
	assert.Equal(t, sync.ErrHigherNonceInBlock, err)
}
