package sync_test

import (
	"math"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/process/sync"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/stretchr/testify/assert"
)

func TestNewBasicForkDetector_ShouldErrNilRounder(t *testing.T) {
	t.Parallel()

	bfd, err := sync.NewShardForkDetector(
		nil,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	assert.Equal(t, process.ErrNilRounder, err)
	assert.Nil(t, bfd)
}

func TestNewBasicForkDetector_ShouldErrNilBlackListHandler(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, err := sync.NewShardForkDetector(
		rounderMock,
		nil,
		&mock.BlockTrackerStub{},
		0,
	)
	assert.Equal(t, process.ErrNilBlackListHandler, err)
	assert.Nil(t, bfd)
}

func TestNewBasicForkDetector_ShouldErrNilBlockTracker(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, err := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		nil,
		0,
	)
	assert.Equal(t, process.ErrNilBlockTracker, err)
	assert.Nil(t, bfd)
}

func TestNewBasicForkDetector_ShouldWork(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, err := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	assert.Nil(t, err)
	assert.NotNil(t, bfd)
}

func TestBasicForkDetector_CheckBlockValidityShouldErrGenesisTimeMissmatch(t *testing.T) {
	t.Parallel()

	genesisTime := time.Now().Unix()
	roundTimeDuration := time.Duration(4 * time.Second)
	round := uint64(2)
	incorrectTimeStamp := uint64(genesisTime + int64(roundTimeDuration)*int64(round) - 1)

	rounderMock := &mock.RounderMock{RoundIndex: 1, RoundTimeDuration: roundTimeDuration}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		genesisTime,
	)

	err := bfd.CheckBlockValidity(&block.Header{Nonce: 1, Round: round, TimeStamp: incorrectTimeStamp}, []byte("hash"), process.BHProposed)
	assert.Equal(t, sync.ErrGenesisTimeMissmatch, err)

	err = bfd.CheckBlockValidity(&block.Header{Nonce: 1, Round: round, TimeStamp: incorrectTimeStamp}, []byte("hash"), process.BHReceived)
	assert.Equal(t, sync.ErrGenesisTimeMissmatch, err)
}

func TestBasicForkDetector_CheckBlockValidityShouldErrLowerRoundInBlock(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	bfd.SetFinalCheckpoint(1, 1)
	err := bfd.CheckBlockValidity(&block.Header{PubKeysBitmap: []byte("X")}, []byte("hash"), process.BHProcessed)
	assert.Equal(t, sync.ErrLowerRoundInBlock, err)
}

func TestBasicForkDetector_CheckBlockValidityShouldErrLowerNonceInBlock(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	bfd.SetFinalCheckpoint(2, 2)
	err := bfd.CheckBlockValidity(&block.Header{Nonce: 1, Round: 3, PubKeysBitmap: []byte("X")}, []byte("hash"), process.BHProcessed)
	assert.Equal(t, sync.ErrLowerNonceInBlock, err)
}

func TestBasicForkDetector_CheckBlockValidityShouldErrHigherRoundInBlock(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 0}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	err := bfd.CheckBlockValidity(&block.Header{Nonce: 1, Round: 2, PubKeysBitmap: []byte("X")}, []byte("hash"), process.BHProcessed)
	assert.Equal(t, sync.ErrHigherRoundInBlock, err)
}

func TestBasicForkDetector_CheckBlockValidityShouldErrHigherNonceInBlock(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 1}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	err := bfd.CheckBlockValidity(&block.Header{Nonce: 2, Round: 1, PubKeysBitmap: []byte("X")}, []byte("hash"), process.BHProcessed)
	assert.Equal(t, sync.ErrHigherNonceInBlock, err)
}

func TestBasicForkDetector_CheckBlockValidityShouldWork(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 1}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	err := bfd.CheckBlockValidity(&block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")}, []byte("hash"), process.BHProcessed)
	assert.Nil(t, err)
}

func TestBasicForkDetector_RemoveHeadersShouldWork(t *testing.T) {
	t.Parallel()

	hdr1 := &block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")}
	hash1 := []byte("hash1")
	hdr2 := &block.Header{Nonce: 2, Round: 2, PubKeysBitmap: []byte("X")}
	hash2 := []byte("hash2")
	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)

	rounderMock.RoundIndex = 1
	_ = bfd.AddHeader(hdr1, hash1, process.BHProcessed, nil, nil)
	rounderMock.RoundIndex = 2
	_ = bfd.AddHeader(hdr2, hash2, process.BHProcessed, nil, nil)

	hInfos := bfd.GetHeaders(1)
	assert.Equal(t, 1, len(hInfos))

	hInfos = bfd.GetHeaders(2)
	assert.Equal(t, 1, len(hInfos))

	bfd.RemoveHeaders(1, hash1)

	hInfos = bfd.GetHeaders(1)
	assert.Nil(t, hInfos)

	hInfos = bfd.GetHeaders(2)
	assert.Equal(t, 1, len(hInfos))
}

func TestBasicForkDetector_CheckForkOnlyOneShardHeaderOnANonceShouldReturnFalse(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	_ = bfd.AddHeader(
		&block.Header{Nonce: 0, PubKeysBitmap: []byte("X")},
		[]byte("hash1"),
		process.BHProcessed,
		nil,
		nil)
	_ = bfd.AddHeader(
		&block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")},
		[]byte("hash2"),
		process.BHProcessed,
		nil,
		nil)
	forkInfo := bfd.CheckFork()
	assert.False(t, forkInfo.IsDetected)
	assert.Equal(t, uint64(math.MaxUint64), forkInfo.Nonce)
	assert.Nil(t, forkInfo.Hash)
}

func TestBasicForkDetector_CheckForkOnlyReceivedHeadersShouldReturnFalse(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	_ = bfd.AddHeader(
		&block.Header{Nonce: 0, PubKeysBitmap: []byte("X")},
		[]byte("hash1"),
		process.BHReceived,
		nil,
		nil,
	)
	_ = bfd.AddHeader(
		&block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")},
		[]byte("hash1"),
		process.BHReceived,
		nil,
		nil,
	)
	forkInfo := bfd.CheckFork()
	assert.False(t, forkInfo.IsDetected)
	assert.Equal(t, uint64(math.MaxUint64), forkInfo.Nonce)
	assert.Nil(t, forkInfo.Hash)
}

func TestBasicForkDetector_CheckForkOnlyOneShardHeaderOnANonceReceivedAndProcessedShouldReturnFalse(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	_ = bfd.AddHeader(
		&block.Header{Nonce: 0, PubKeysBitmap: []byte("X")},
		[]byte("hash1"),
		process.BHProcessed,
		nil,
		nil,
	)
	_ = bfd.AddHeader(
		&block.Header{Nonce: 1, Round: 1, PubKeysBitmap: []byte("X")},
		[]byte("hash2"),
		process.BHReceived,
		nil,
		nil,
	)
	forkInfo := bfd.CheckFork()
	assert.False(t, forkInfo.IsDetected)
	assert.Equal(t, uint64(math.MaxUint64), forkInfo.Nonce)
	assert.Nil(t, forkInfo.Hash)
}

func TestBasicForkDetector_CheckForkMetaHeaderProcessedShouldReturnFalse(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 99}
	bfd, _ := sync.NewMetaForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	_ = bfd.AddHeader(
		&block.MetaBlock{Nonce: 1, Round: 3, PubKeysBitmap: []byte("X")},
		[]byte("hash1"),
		process.BHProcessed,
		nil,
		nil)
	_ = bfd.AddHeader(
		&block.MetaBlock{Nonce: 1, Round: 2, PubKeysBitmap: []byte("X")},
		[]byte("hash2"),
		process.BHProcessed,
		nil,
		nil)
	forkInfo := bfd.CheckFork()
	assert.False(t, forkInfo.IsDetected)
	assert.Equal(t, uint64(math.MaxUint64), forkInfo.Nonce)
	assert.Nil(t, forkInfo.Hash)
}

func TestBasicForkDetector_CheckForkMetaHeaderProcessedShouldReturnFalseWhenLowerRound(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewMetaForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	rounderMock.RoundIndex = 5
	_ = bfd.AddHeader(
		&block.MetaBlock{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")},
		[]byte("hash1"),
		process.BHReceived,
		nil,
		nil)
	rounderMock.RoundIndex = 4
	_ = bfd.AddHeader(
		&block.MetaBlock{Nonce: 1, Round: 3, PubKeysBitmap: []byte("X")},
		[]byte("hash2"),
		process.BHReceived,
		nil,
		nil)
	rounderMock.RoundIndex = 3
	_ = bfd.AddHeader(
		&block.MetaBlock{Nonce: 1, Round: 2, PubKeysBitmap: []byte("X")},
		[]byte("hash3"),
		process.BHProcessed,
		nil,
		nil)

	hInfos := bfd.GetHeaders(1)
	assert.Equal(t, 3, len(hInfos))

	forkInfo := bfd.CheckFork()
	assert.False(t, forkInfo.IsDetected)
	assert.Equal(t, uint64(math.MaxUint64), forkInfo.Nonce)
	assert.Nil(t, forkInfo.Hash)

	hInfos = bfd.GetHeaders(1)
	assert.Equal(t, 3, len(hInfos))
}

func TestBasicForkDetector_CheckForkMetaHeaderProcessedShouldReturnFalseWhenEqualRoundWithLowerHash(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewMetaForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	rounderMock.RoundIndex = 5
	_ = bfd.AddHeader(
		&block.MetaBlock{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")},
		[]byte("hash1"),
		process.BHProcessed,
		nil,
		nil)
	_ = bfd.AddHeader(
		&block.MetaBlock{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")},
		[]byte("hash2"),
		process.BHReceived,
		nil,
		nil)
	_ = bfd.AddHeader(
		&block.MetaBlock{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")},
		[]byte("hash3"),
		process.BHReceived,
		nil,
		nil)

	hInfos := bfd.GetHeaders(1)
	assert.Equal(t, 3, len(hInfos))

	forkInfo := bfd.CheckFork()
	assert.False(t, forkInfo.IsDetected)
	assert.Equal(t, uint64(math.MaxUint64), forkInfo.Nonce)
	assert.Nil(t, forkInfo.Hash)

	hInfos = bfd.GetHeaders(1)
	assert.Equal(t, 3, len(hInfos))
	assert.Equal(t, []byte("hash1"), hInfos[0].Hash())
}

func TestBasicForkDetector_CheckForkShardHeaderProcessedShouldReturnTrueWhenEqualRoundWithLowerHash(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)

	hdr1 := &block.Header{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")}
	hash1 := []byte("hash1")
	hdr2 := &block.Header{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")}
	hash2 := []byte("hash2")
	hdr3 := &block.Header{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")}
	hash3 := []byte("hash3")

	notarizedHeaders2 := []data.HeaderHandler{
		hdr2,
	}
	notarizedHeadersHashes2 := [][]byte{
		hash2,
	}
	notarizedHeaders3 := []data.HeaderHandler{
		hdr3,
	}
	notarizedHeadersHashes3 := [][]byte{
		hash3,
	}

	rounderMock.RoundIndex = 5
	_ = bfd.AddHeader(
		hdr1,
		hash1,
		process.BHProcessed,
		nil,
		nil)
	_ = bfd.AddHeader(
		hdr2,
		hash2,
		process.BHNotarized,
		notarizedHeaders2,
		notarizedHeadersHashes2)
	_ = bfd.AddHeader(
		hdr3,
		hash3,
		process.BHNotarized,
		notarizedHeaders3,
		notarizedHeadersHashes3)

	hInfos := bfd.GetHeaders(1)
	assert.Equal(t, 3, len(hInfos))

	forkInfo := bfd.CheckFork()
	assert.True(t, forkInfo.IsDetected)
	assert.Equal(t, uint64(1), forkInfo.Nonce)
	assert.Equal(t, hash2, forkInfo.Hash)

	hInfos = bfd.GetHeaders(1)
	assert.Equal(t, 3, len(hInfos))
	assert.Equal(t, hash1, hInfos[0].Hash())
}

func TestBasicForkDetector_CheckForkMetaHeaderProcessedShouldReturnTrueWhenEqualRoundWithHigherHash(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewMetaForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	rounderMock.RoundIndex = 5
	_ = bfd.AddHeader(
		&block.MetaBlock{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")},
		[]byte("hash2"),
		process.BHProcessed,
		nil,
		nil)
	_ = bfd.AddHeader(
		&block.MetaBlock{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")},
		[]byte("hash3"),
		process.BHReceived,
		nil,
		nil)
	_ = bfd.AddHeader(
		&block.MetaBlock{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")},
		[]byte("hash1"),
		process.BHReceived,
		nil,
		nil)

	hInfos := bfd.GetHeaders(1)
	assert.Equal(t, 3, len(hInfos))

	forkInfo := bfd.CheckFork()
	assert.True(t, forkInfo.IsDetected)
	assert.Equal(t, uint64(1), forkInfo.Nonce)
	assert.Equal(t, []byte("hash1"), forkInfo.Hash)

	hInfos = bfd.GetHeaders(1)
	assert.Equal(t, 3, len(hInfos))
}

func TestBasicForkDetector_CheckForkShardHeaderProcessedShouldReturnTrueWhenEqualRoundWithHigherHash(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)

	hdr1 := &block.Header{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")}
	hash1 := []byte("hash1")
	hdr2 := &block.Header{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")}
	hash2 := []byte("hash2")
	hdr3 := &block.Header{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")}
	hash3 := []byte("hash3")

	notarizedHeaders1 := []data.HeaderHandler{
		hdr1,
	}
	notarizedHeadersHashes1 := [][]byte{
		hash1,
	}
	notarizedHeaders3 := []data.HeaderHandler{
		hdr3,
	}
	notarizedHeadersHashes3 := [][]byte{
		hash3,
	}

	rounderMock.RoundIndex = 5
	_ = bfd.AddHeader(
		hdr2,
		hash2,
		process.BHProcessed,
		nil,
		nil)
	_ = bfd.AddHeader(
		hdr3,
		hash3,
		process.BHNotarized,
		notarizedHeaders3,
		notarizedHeadersHashes3)
	_ = bfd.AddHeader(
		hdr1,
		hash1,
		process.BHNotarized,
		notarizedHeaders1,
		notarizedHeadersHashes1)

	hInfos := bfd.GetHeaders(1)
	assert.Equal(t, 3, len(hInfos))

	forkInfo := bfd.CheckFork()
	assert.True(t, forkInfo.IsDetected)
	assert.Equal(t, uint64(1), forkInfo.Nonce)
	assert.Equal(t, hash1, forkInfo.Hash)

	hInfos = bfd.GetHeaders(1)
	assert.Equal(t, 3, len(hInfos))
}

func TestBasicForkDetector_CheckForkShouldReturnTrue(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewMetaForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	rounderMock.RoundIndex = 4
	_ = bfd.AddHeader(
		&block.MetaBlock{Nonce: 1, Round: 3, PubKeysBitmap: []byte("X")},
		[]byte("hash1"),
		process.BHReceived,
		nil,
		nil)
	rounderMock.RoundIndex = 3
	_ = bfd.AddHeader(
		&block.MetaBlock{Nonce: 1, Round: 2, PubKeysBitmap: []byte("X")},
		[]byte("hash2"),
		process.BHReceived,
		nil,
		nil)
	rounderMock.RoundIndex = 4
	_ = bfd.AddHeader(
		&block.MetaBlock{Nonce: 1, Round: 3, PubKeysBitmap: []byte("X")},
		[]byte("hash3"),
		process.BHProcessed,
		nil,
		nil)

	hInfos := bfd.GetHeaders(1)
	assert.Equal(t, 3, len(hInfos))

	forkInfo := bfd.CheckFork()
	assert.True(t, forkInfo.IsDetected)
	assert.Equal(t, uint64(1), forkInfo.Nonce)
	assert.Equal(t, []byte("hash2"), forkInfo.Hash)

	hInfos = bfd.GetHeaders(1)
	assert.Equal(t, 3, len(hInfos))
}

func TestBasicForkDetector_RemovePastHeadersShouldWork(t *testing.T) {
	t.Parallel()

	hdr1 := &block.Header{Nonce: 1, PubKeysBitmap: []byte("X")}
	hash1 := []byte("hash1")
	hdr2 := &block.Header{Nonce: 2, PubKeysBitmap: []byte("X")}
	hash2 := []byte("hash2")
	hdr3 := &block.Header{Nonce: 3, PubKeysBitmap: []byte("X")}
	hash3 := []byte("hash3")
	rounderMock := &mock.RounderMock{RoundIndex: 100}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	_ = bfd.AddHeader(hdr1, hash1, process.BHReceived, nil, nil)
	_ = bfd.AddHeader(hdr2, hash2, process.BHReceived, nil, nil)
	_ = bfd.AddHeader(hdr3, hash3, process.BHReceived, nil, nil)
	bfd.SetFinalCheckpoint(4, 4)
	bfd.RemovePastHeaders()

	hInfos := bfd.GetHeaders(3)
	assert.Nil(t, hInfos)

	hInfos = bfd.GetHeaders(2)
	assert.Nil(t, hInfos)

	hInfos = bfd.GetHeaders(1)
	assert.Nil(t, hInfos)
}

func TestBasicForkDetector_RemoveInvalidReceivedHeadersShouldWork(t *testing.T) {
	t.Parallel()

	hdr0 := &block.Header{PubKeysBitmap: []byte("X"), Nonce: 8, Round: 10}
	hash0 := []byte("hash0")
	hdr1 := &block.Header{PubKeysBitmap: []byte("X"), Nonce: 9, Round: 12}
	hash1 := []byte("hash1")
	hdr2 := &block.Header{PubKeysBitmap: []byte("X"), Nonce: 13, Round: 15}
	hash2 := []byte("hash2")
	hdr3 := &block.Header{PubKeysBitmap: []byte("X"), Nonce: 10, Round: 14}
	hash3 := []byte("hash3")
	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)
	rounderMock.RoundIndex = 11
	_ = bfd.AddHeader(hdr0, hash0, process.BHReceived, nil, nil)
	rounderMock.RoundIndex = 13
	_ = bfd.AddHeader(hdr1, hash1, process.BHReceived, nil, nil)
	rounderMock.RoundIndex = 16
	_ = bfd.AddHeader(hdr2, hash2, process.BHReceived, nil, nil)
	rounderMock.RoundIndex = 15
	_ = bfd.AddHeader(hdr3, hash3, process.BHReceived, nil, nil)
	bfd.SetFinalCheckpoint(9, 12)
	bfd.RemoveInvalidReceivedHeaders()

	hInfos := bfd.GetHeaders(8)
	assert.Nil(t, hInfos)

	hInfos = bfd.GetHeaders(9)
	assert.NotNil(t, hInfos)

	hInfos = bfd.GetHeaders(13)
	assert.Nil(t, hInfos)

	hInfos = bfd.GetHeaders(10)
	assert.NotNil(t, hInfos)
	assert.Equal(t, uint64(13), bfd.ProbableHighestNonce())
	assert.Equal(t, uint64(10), bfd.ComputeProbableHighestNonce())
}

func TestBasicForkDetector_RemoveCheckpointHeaderNonceShouldResetCheckpoint(t *testing.T) {
	t.Parallel()

	hdr1 := &block.Header{Nonce: 2, Round: 2, PubKeysBitmap: []byte("X")}
	hash1 := []byte("hash1")
	rounderMock := &mock.RounderMock{RoundIndex: 2}
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)

	_ = bfd.AddHeader(hdr1, hash1, process.BHProcessed, nil, nil)
	assert.Equal(t, uint64(2), bfd.LastCheckpointNonce())

	bfd.RemoveHeaders(2, hash1)
	assert.Equal(t, uint64(0), bfd.LastCheckpointNonce())
	assert.Equal(t, uint64(0), bfd.LastCheckpointRound())
}

func TestBasicForkDetector_GetHighestFinalBlockNonce(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewMetaForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)

	hdr1 := &block.MetaBlock{Nonce: 2, Round: 1, PubKeysBitmap: []byte("X")}
	hash1 := []byte("hash1")
	rounderMock.RoundIndex = 1
	_ = bfd.AddHeader(hdr1, hash1, process.BHProcessed, nil, nil)
	assert.Equal(t, uint64(0), bfd.GetHighestFinalBlockNonce())

	hdr2 := &block.MetaBlock{Nonce: 3, Round: 3, PubKeysBitmap: []byte("X")}
	hash2 := []byte("hash2")
	rounderMock.RoundIndex = 3
	_ = bfd.AddHeader(hdr2, hash2, process.BHProcessed, nil, nil)
	assert.Equal(t, uint64(0), bfd.GetHighestFinalBlockNonce())

	hdr3 := &block.MetaBlock{Nonce: 4, Round: 4, PubKeysBitmap: []byte("X")}
	hash3 := []byte("hash3")
	rounderMock.RoundIndex = 4
	_ = bfd.AddHeader(hdr3, hash3, process.BHProcessed, nil, nil)
	assert.Equal(t, uint64(3), bfd.GetHighestFinalBlockNonce())

	hdr4 := &block.MetaBlock{Nonce: 6, Round: 5, PubKeysBitmap: []byte("X")}
	hash4 := []byte("hash4")
	rounderMock.RoundIndex = 5
	_ = bfd.AddHeader(hdr4, hash4, process.BHProcessed, nil, nil)
	assert.Equal(t, uint64(3), bfd.GetHighestFinalBlockNonce())
}

func TestBasicForkDetector_ProbableHighestNonce(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewMetaForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)

	rounderMock.RoundIndex = 11
	_ = bfd.AddHeader(
		&block.MetaBlock{PubKeysBitmap: []byte("X"), Nonce: 8, Round: 10},
		[]byte("hash0"),
		process.BHReceived,
		nil,
		nil)
	assert.Equal(t, uint64(8), bfd.ProbableHighestNonce())

	rounderMock.RoundIndex = 13
	_ = bfd.AddHeader(
		&block.MetaBlock{PubKeysBitmap: []byte("X"), Nonce: 9, Round: 12},
		[]byte("hash1"),
		process.BHProcessed,
		nil,
		nil)
	assert.Equal(t, uint64(9), bfd.ProbableHighestNonce())

	rounderMock.RoundIndex = 16
	_ = bfd.AddHeader(
		&block.MetaBlock{PubKeysBitmap: []byte("X"), Nonce: 13, Round: 15},
		[]byte("hash2"),
		process.BHReceived,
		nil,
		nil)
	assert.Equal(t, uint64(13), bfd.ProbableHighestNonce())

	rounderMock.RoundIndex = 15
	_ = bfd.AddHeader(
		&block.MetaBlock{PubKeysBitmap: []byte("X"), Nonce: 10, Round: 14},
		[]byte("hash3"),
		process.BHProcessed,
		nil,
		nil)
	assert.Equal(t, uint64(10), bfd.ProbableHighestNonce())

	rounderMock.RoundIndex = 16
	_ = bfd.AddHeader(
		&block.MetaBlock{PubKeysBitmap: []byte("X"), Nonce: 11, Round: 15},
		[]byte("hash3"),
		process.BHReceived,
		nil,
		nil)
	assert.Equal(t, uint64(11), bfd.ProbableHighestNonce())
}

func TestShardForkDetector_ShouldAddBlockInForkDetectorShouldWork(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 10}
	sfd, _ := sync.NewShardForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)

	hdr := &block.Header{Nonce: 1, Round: 1}
	receivedTooLate := sfd.IsHeaderReceivedTooLate(hdr, process.BHProcessed, process.BlockFinality)
	assert.False(t, receivedTooLate)

	receivedTooLate = sfd.IsHeaderReceivedTooLate(hdr, process.BHReceived, process.BlockFinality)
	assert.True(t, receivedTooLate)

	receivedTooLate = sfd.IsHeaderReceivedTooLate(hdr, process.BHProposed, process.BlockFinality)
	assert.True(t, receivedTooLate)

	hdr.Round = uint64(rounderMock.RoundIndex - process.BlockFinality)
	receivedTooLate = sfd.IsHeaderReceivedTooLate(hdr, process.BHReceived, process.BlockFinality)
	assert.False(t, receivedTooLate)

	receivedTooLate = sfd.IsHeaderReceivedTooLate(hdr, process.BHProposed, process.BlockFinality)
	assert.False(t, receivedTooLate)
}

func TestShardForkDetector_ShouldAddBlockInForkDetectorShouldErrLowerRoundInBlock(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 10}
	sfd, _ := sync.NewShardForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)
	hdr := &block.Header{Nonce: 1, Round: 1}

	hdr.Round = uint64(rounderMock.RoundIndex - process.BlockFinality - 1)
	receivedTooLate := sfd.IsHeaderReceivedTooLate(hdr, process.BHReceived, process.BlockFinality)
	assert.True(t, receivedTooLate)

	sfd.AddCheckPoint(2, hdr.GetNonce()+process.NonceDifferenceWhenSynced)
	sfd.SetProbableHighestNonce(hdr.GetNonce() + process.NonceDifferenceWhenSynced)
	receivedTooLate = sfd.IsHeaderReceivedTooLate(hdr, process.BHProposed, process.BlockFinality)
	assert.True(t, receivedTooLate)
}

func TestMetaForkDetector_ShouldAddBlockInForkDetectorShouldWork(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 10}
	mfd, _ := sync.NewMetaForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)

	hdr := &block.MetaBlock{Nonce: 1, Round: 1}
	receivedTooLate := mfd.IsHeaderReceivedTooLate(hdr, process.BHProcessed, process.BlockFinality)
	assert.False(t, receivedTooLate)

	receivedTooLate = mfd.IsHeaderReceivedTooLate(hdr, process.BHReceived, process.BlockFinality)
	assert.True(t, receivedTooLate)

	receivedTooLate = mfd.IsHeaderReceivedTooLate(hdr, process.BHProposed, process.BlockFinality)
	assert.True(t, true)

	hdr.Round = uint64(rounderMock.RoundIndex - process.BlockFinality)
	receivedTooLate = mfd.IsHeaderReceivedTooLate(hdr, process.BHReceived, process.BlockFinality)
	assert.False(t, receivedTooLate)

	receivedTooLate = mfd.IsHeaderReceivedTooLate(hdr, process.BHProposed, process.BlockFinality)
	assert.False(t, receivedTooLate)
}

func TestMetaForkDetector_ShouldAddBlockInForkDetectorShouldErrLowerRoundInBlock(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 10}
	mfd, _ := sync.NewMetaForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)
	hdr := &block.MetaBlock{Nonce: 1, Round: 1}

	hdr.Round = uint64(rounderMock.RoundIndex - process.BlockFinality - 1)
	receivedTooLate := mfd.IsHeaderReceivedTooLate(hdr, process.BHReceived, process.BlockFinality)
	assert.True(t, receivedTooLate)

	mfd.AddCheckPoint(2, hdr.GetNonce()+process.NonceDifferenceWhenSynced)
	mfd.SetProbableHighestNonce(hdr.GetNonce() + process.NonceDifferenceWhenSynced)
	receivedTooLate = mfd.IsHeaderReceivedTooLate(hdr, process.BHProposed, process.BlockFinality)
	assert.True(t, receivedTooLate)
}

func TestShardForkDetector_AddNotarizedHeadersShouldNotChangeTheFinalCheckpoint(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{RoundIndex: 10}
	sfd, _ := sync.NewShardForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)
	hdr1 := &block.Header{Nonce: 3, Round: 3}
	hash1 := []byte("hash1")
	hdr2 := &block.Header{Nonce: 3, Round: 3}
	hash2 := []byte("hash2")
	hdr3 := &block.Header{Nonce: 4, Round: 5}
	hash3 := []byte("hash3")

	hdrs := make([]data.HeaderHandler, 0)
	hashes := make([][]byte, 0)
	hdrs = append(hdrs, hdr1)
	hashes = append(hashes, hash1)

	sfd.AddSelfNotarizedHeaders(hdrs, hashes)
	assert.Equal(t, uint64(0), sfd.FinalCheckpointNonce())

	sfd.AddHeader(hdr1, hash1, process.BHProcessed, hdrs, hashes)
	assert.Equal(t, hdr1.Nonce, sfd.FinalCheckpointNonce())

	hdrs = make([]data.HeaderHandler, 0)
	hashes = make([][]byte, 0)
	hdrs = append(hdrs, hdr2)
	hashes = append(hashes, hash2)

	sfd.AddSelfNotarizedHeaders(hdrs, hashes)
	assert.Equal(t, hdr1.Nonce, sfd.FinalCheckpointNonce())

	sfd.AddHeader(hdr2, hash2, process.BHProcessed, hdrs, hashes)
	assert.Equal(t, hdr2.Nonce, sfd.FinalCheckpointNonce())

	hdrs = make([]data.HeaderHandler, 0)
	hashes = make([][]byte, 0)
	hdrs = append(hdrs, hdr3)
	hashes = append(hashes, hash3)

	sfd.AddSelfNotarizedHeaders(hdrs, hashes)
	assert.Equal(t, hdr2.Nonce, sfd.FinalCheckpointNonce())

	sfd.AddHeader(hdr3, hash3, process.BHProcessed, hdrs, hashes)
	assert.Equal(t, hdr3.Nonce, sfd.FinalCheckpointNonce())
}

func TestBaseForkDetector_ActivateForcedForkIfNeededStateNotProposedShouldNotActivate(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewShardForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)

	state := process.BHReceived
	hdr1 := &block.Header{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")}

	bfd.ActivateForcedForkIfNeeded(hdr1, state, sharding.MetachainShardId)
	assert.False(t, bfd.ShouldForceFork())
}

func TestBaseForkDetector_ActivateForcedForkIfNeededNotSyncingShouldNotActivate(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewShardForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)

	state := process.BHProposed
	hdr1 := &block.Header{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")}

	bfd.ActivateForcedForkIfNeeded(hdr1, state, sharding.MetachainShardId)
	assert.False(t, bfd.ShouldForceFork())
}

func TestBaseForkDetector_ActivateForcedForkIfNeededDifferencesNotEnoughShouldNotActivate(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewMetaForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)

	_ = bfd.AddHeader(
		&block.Header{PubKeysBitmap: []byte("X"), Nonce: 9, Round: 3},
		[]byte("hash1"),
		process.BHProcessed,
		nil,
		nil)

	state := process.BHProposed
	hdr1 := &block.Header{Nonce: 1, Round: 4, PubKeysBitmap: []byte("X")}
	rounderMock.RoundIndex = 5
	bfd.ActivateForcedForkIfNeeded(hdr1, state, sharding.MetachainShardId)
	assert.False(t, bfd.ShouldForceFork())
}

func TestBaseForkDetector_ActivateForcedForkIfNeededShouldActivate(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewMetaForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		0,
	)

	bfd.SetFinalCheckpoint(0, 0)
	_ = bfd.AddHeader(
		&block.Header{PubKeysBitmap: []byte("X"), Nonce: 0, Round: 28},
		[]byte("hash1"),
		process.BHProcessed,
		nil,
		nil)

	// last checkpoint will be (round = 0 , nonce = 0)
	// round difference is higher than 20
	// nonce difference is 1
	// round index is divisible by 5
	// => should activate force fork
	state := process.BHProposed
	hdr1 := &block.Header{Nonce: 1, Round: 29, PubKeysBitmap: []byte("X")}
	rounderMock.RoundIndex = 30
	bfd.ActivateForcedForkIfNeeded(hdr1, state, sharding.MetachainShardId)
	assert.True(t, bfd.ShouldForceFork())
}

func TestBaseForkDetector_ResetFork(t *testing.T) {
	t.Parallel()

	rounderMock := &mock.RounderMock{}
	bfd, _ := sync.NewShardForkDetector(rounderMock, &mock.BlackListHandlerStub{}, &mock.BlockTrackerStub{}, 0)

	bfd.SetShouldForceFork(true)
	assert.True(t, bfd.ShouldForceFork())
	bfd.ResetFork()
	assert.False(t, bfd.ShouldForceFork())
}

func TestBaseForkDetector_ComputeTimeDuration(t *testing.T) {
	t.Parallel()

	roundDuration := uint64(1)
	rounderMock := &mock.RounderMock{
		RoundTimeDuration: time.Second,
	}

	genesisTime := int64(9000)
	hdrTimeStamp := uint64(10000)
	hdrRound := uint64(20)
	bfd, _ := sync.NewShardForkDetector(
		rounderMock,
		&mock.BlackListHandlerStub{},
		&mock.BlockTrackerStub{},
		genesisTime,
	)

	hdr1 := &block.Header{Nonce: 1, Round: hdrRound, PubKeysBitmap: []byte("X"), TimeStamp: hdrTimeStamp}

	expectedTimeStamp := hdrTimeStamp - (hdrRound * roundDuration)
	timeDuration := bfd.ComputeGenesisTimeFromHeader(hdr1)
	assert.Equal(t, int64(expectedTimeStamp), timeDuration)
}
