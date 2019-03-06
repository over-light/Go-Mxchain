package bn_test

import (
	"github.com/ElrondNetwork/elrond-go-sandbox/data"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go-sandbox/consensus/spos"
	"github.com/ElrondNetwork/elrond-go-sandbox/consensus/spos/bn"
	"github.com/ElrondNetwork/elrond-go-sandbox/consensus/spos/mock"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/block"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/blockchain"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func initSubroundBlock() bn.SubroundBlock {
	blockChain := blockchain.BlockChain{}
	blockProcessorMock := initBlockProcessorMock()
	consensusState := initConsensusState()
	hasherMock := mock.HasherMock{}
	marshalizerMock := mock.MarshalizerMock{}
	multiSignerMock := initMultiSignerMock()
	rounderMock := initRounderMock()
	shardCoordinatorMock := mock.ShardCoordinatorMock{}
	syncTimerMock := mock.SyncTimerMock{}

	ch := make(chan bool, 1)

	sr, _ := bn.NewSubround(
		int(bn.SrStartRound),
		int(bn.SrBlock),
		int(bn.SrCommitmentHash),
		int64(5*roundTimeDuration/100),
		int64(25*roundTimeDuration/100),
		"(BLOCK)",
		ch,
	)

	srBlock, _ := bn.NewSubroundBlock(
		sr,
		&blockChain,
		blockProcessorMock,
		consensusState,
		hasherMock,
		marshalizerMock,
		multiSignerMock,
		rounderMock,
		shardCoordinatorMock,
		syncTimerMock,
		sendConsensusMessage,
		extend,
	)

	return srBlock
}

func TestSubroundBlock_NewSubroundBlockNilSubroundShouldFail(t *testing.T) {
	t.Parallel()

	blockChain := blockchain.BlockChain{}
	blockProcessorMock := initBlockProcessorMock()
	consensusState := initConsensusState()
	hasherMock := mock.HasherMock{}
	marshalizerMock := mock.MarshalizerMock{}
	multiSignerMock := initMultiSignerMock()
	rounderMock := initRounderMock()
	shardCoordinatorMock := mock.ShardCoordinatorMock{}
	syncTimerMock := mock.SyncTimerMock{}

	srBlock, err := bn.NewSubroundBlock(
		nil,
		&blockChain,
		blockProcessorMock,
		consensusState,
		hasherMock,
		marshalizerMock,
		multiSignerMock,
		rounderMock,
		shardCoordinatorMock,
		syncTimerMock,
		sendConsensusMessage,
		extend,
	)

	assert.Nil(t, srBlock)
	assert.Equal(t, err, spos.ErrNilSubround)
}

func TestSubroundBlock_NewSubroundBlockNilBlockchainShouldFail(t *testing.T) {
	t.Parallel()

	blockProcessorMock := initBlockProcessorMock()
	consensusState := initConsensusState()
	hasherMock := mock.HasherMock{}
	marshalizerMock := mock.MarshalizerMock{}
	multiSignerMock := initMultiSignerMock()
	rounderMock := initRounderMock()
	shardCoordinatorMock := mock.ShardCoordinatorMock{}
	syncTimerMock := mock.SyncTimerMock{}

	ch := make(chan bool, 1)

	sr, _ := bn.NewSubround(
		int(bn.SrStartRound),
		int(bn.SrBlock),
		int(bn.SrCommitmentHash),
		int64(5*roundTimeDuration/100),
		int64(25*roundTimeDuration/100),
		"(BLOCK)",
		ch,
	)

	srBlock, err := bn.NewSubroundBlock(
		sr,
		nil,
		blockProcessorMock,
		consensusState,
		hasherMock,
		marshalizerMock,
		multiSignerMock,
		rounderMock,
		shardCoordinatorMock,
		syncTimerMock,
		sendConsensusMessage,
		extend,
	)

	assert.Nil(t, srBlock)
	assert.Equal(t, err, spos.ErrNilBlockChain)
}

func TestSubroundBlock_NewSubroundBlockNilBlockProcessorShouldFail(t *testing.T) {
	t.Parallel()

	blockChain := blockchain.BlockChain{}
	consensusState := initConsensusState()
	hasherMock := mock.HasherMock{}
	marshalizerMock := mock.MarshalizerMock{}
	multiSignerMock := initMultiSignerMock()
	rounderMock := initRounderMock()
	shardCoordinatorMock := mock.ShardCoordinatorMock{}
	syncTimerMock := mock.SyncTimerMock{}

	ch := make(chan bool, 1)

	sr, _ := bn.NewSubround(
		int(bn.SrStartRound),
		int(bn.SrBlock),
		int(bn.SrCommitmentHash),
		int64(5*roundTimeDuration/100),
		int64(25*roundTimeDuration/100),
		"(BLOCK)",
		ch,
	)

	srBlock, err := bn.NewSubroundBlock(
		sr,
		&blockChain,
		nil,
		consensusState,
		hasherMock,
		marshalizerMock,
		multiSignerMock,
		rounderMock,
		shardCoordinatorMock,
		syncTimerMock,
		sendConsensusMessage,
		extend,
	)

	assert.Nil(t, srBlock)
	assert.Equal(t, err, spos.ErrNilBlockProcessor)
}

func TestSubroundBlock_NewSubroundBlockNilConsensusStateShouldFail(t *testing.T) {
	t.Parallel()

	blockChain := blockchain.BlockChain{}
	blockProcessorMock := initBlockProcessorMock()
	hasherMock := mock.HasherMock{}
	marshalizerMock := mock.MarshalizerMock{}
	multiSignerMock := initMultiSignerMock()
	rounderMock := initRounderMock()
	shardCoordinatorMock := mock.ShardCoordinatorMock{}
	syncTimerMock := mock.SyncTimerMock{}

	ch := make(chan bool, 1)

	sr, _ := bn.NewSubround(
		int(bn.SrStartRound),
		int(bn.SrBlock),
		int(bn.SrCommitmentHash),
		int64(5*roundTimeDuration/100),
		int64(25*roundTimeDuration/100),
		"(BLOCK)",
		ch,
	)

	srBlock, err := bn.NewSubroundBlock(
		sr,
		&blockChain,
		blockProcessorMock,
		nil,
		hasherMock,
		marshalizerMock,
		multiSignerMock,
		rounderMock,
		shardCoordinatorMock,
		syncTimerMock,
		sendConsensusMessage,
		extend,
	)

	assert.Nil(t, srBlock)
	assert.Equal(t, err, spos.ErrNilConsensusState)
}

func TestSubroundBlock_NewSubroundBlockNilHasherShouldFail(t *testing.T) {
	t.Parallel()

	blockChain := blockchain.BlockChain{}
	blockProcessorMock := initBlockProcessorMock()
	consensusState := initConsensusState()
	marshalizerMock := mock.MarshalizerMock{}
	multiSignerMock := initMultiSignerMock()
	rounderMock := initRounderMock()
	shardCoordinatorMock := mock.ShardCoordinatorMock{}
	syncTimerMock := mock.SyncTimerMock{}

	ch := make(chan bool, 1)

	sr, _ := bn.NewSubround(
		int(bn.SrStartRound),
		int(bn.SrBlock),
		int(bn.SrCommitmentHash),
		int64(5*roundTimeDuration/100),
		int64(25*roundTimeDuration/100),
		"(BLOCK)",
		ch,
	)

	srBlock, err := bn.NewSubroundBlock(
		sr,
		&blockChain,
		blockProcessorMock,
		consensusState,
		nil,
		marshalizerMock,
		multiSignerMock,
		rounderMock,
		shardCoordinatorMock,
		syncTimerMock,
		sendConsensusMessage,
		extend,
	)

	assert.Nil(t, srBlock)
	assert.Equal(t, err, spos.ErrNilHasher)
}

func TestSubroundBlock_NewSubroundBlockNilMarshalizerShouldFail(t *testing.T) {
	t.Parallel()

	blockChain := blockchain.BlockChain{}
	blockProcessorMock := initBlockProcessorMock()
	consensusState := initConsensusState()
	hasherMock := mock.HasherMock{}
	multiSignerMock := initMultiSignerMock()
	rounderMock := initRounderMock()
	shardCoordinatorMock := mock.ShardCoordinatorMock{}
	syncTimerMock := mock.SyncTimerMock{}

	ch := make(chan bool, 1)

	sr, _ := bn.NewSubround(
		int(bn.SrStartRound),
		int(bn.SrBlock),
		int(bn.SrCommitmentHash),
		int64(5*roundTimeDuration/100),
		int64(25*roundTimeDuration/100),
		"(BLOCK)",
		ch,
	)

	srBlock, err := bn.NewSubroundBlock(
		sr,
		&blockChain,
		blockProcessorMock,
		consensusState,
		hasherMock,
		nil,
		multiSignerMock,
		rounderMock,
		shardCoordinatorMock,
		syncTimerMock,
		sendConsensusMessage,
		extend,
	)

	assert.Nil(t, srBlock)
	assert.Equal(t, err, spos.ErrNilMarshalizer)
}

func TestSubroundBlock_NewSubroundBlockNilMultisignerShouldFail(t *testing.T) {
	t.Parallel()

	blockChain := blockchain.BlockChain{}
	blockProcessorMock := initBlockProcessorMock()
	consensusState := initConsensusState()
	hasherMock := mock.HasherMock{}
	marshalizerMock := mock.MarshalizerMock{}
	rounderMock := initRounderMock()
	shardCoordinatorMock := mock.ShardCoordinatorMock{}
	syncTimerMock := mock.SyncTimerMock{}

	ch := make(chan bool, 1)

	sr, _ := bn.NewSubround(
		int(bn.SrStartRound),
		int(bn.SrBlock),
		int(bn.SrCommitmentHash),
		int64(5*roundTimeDuration/100),
		int64(25*roundTimeDuration/100),
		"(BLOCK)",
		ch,
	)

	srBlock, err := bn.NewSubroundBlock(
		sr,
		&blockChain,
		blockProcessorMock,
		consensusState,
		hasherMock,
		marshalizerMock,
		nil,
		rounderMock,
		shardCoordinatorMock,
		syncTimerMock,
		sendConsensusMessage,
		extend,
	)

	assert.Nil(t, srBlock)
	assert.Equal(t, err, spos.ErrNilMultiSigner)
}

func TestSubroundBlock_NewSubroundBlockNilRounderShouldFail(t *testing.T) {
	t.Parallel()

	blockChain := blockchain.BlockChain{}
	blockProcessorMock := initBlockProcessorMock()
	consensusState := initConsensusState()
	hasherMock := mock.HasherMock{}
	marshalizerMock := mock.MarshalizerMock{}
	multiSignerMock := initMultiSignerMock()
	shardCoordinatorMock := mock.ShardCoordinatorMock{}
	syncTimerMock := mock.SyncTimerMock{}

	ch := make(chan bool, 1)

	sr, _ := bn.NewSubround(
		int(bn.SrStartRound),
		int(bn.SrBlock),
		int(bn.SrCommitmentHash),
		int64(5*roundTimeDuration/100),
		int64(25*roundTimeDuration/100),
		"(BLOCK)",
		ch,
	)

	srBlock, err := bn.NewSubroundBlock(
		sr,
		&blockChain,
		blockProcessorMock,
		consensusState,
		hasherMock,
		marshalizerMock,
		multiSignerMock,
		nil,
		shardCoordinatorMock,
		syncTimerMock,
		sendConsensusMessage,
		extend,
	)

	assert.Nil(t, srBlock)
	assert.Equal(t, err, spos.ErrNilRounder)
}

func TestSubroundBlock_NewSubroundBlockNilShardCoordinatorShouldFail(t *testing.T) {
	t.Parallel()

	blockChain := blockchain.BlockChain{}
	blockProcessorMock := initBlockProcessorMock()
	consensusState := initConsensusState()
	hasherMock := mock.HasherMock{}
	marshalizerMock := mock.MarshalizerMock{}
	multiSignerMock := initMultiSignerMock()
	rounderMock := initRounderMock()
	syncTimerMock := mock.SyncTimerMock{}

	ch := make(chan bool, 1)

	sr, _ := bn.NewSubround(
		int(bn.SrStartRound),
		int(bn.SrBlock),
		int(bn.SrCommitmentHash),
		int64(5*roundTimeDuration/100),
		int64(25*roundTimeDuration/100),
		"(BLOCK)",
		ch,
	)

	srBlock, err := bn.NewSubroundBlock(
		sr,
		&blockChain,
		blockProcessorMock,
		consensusState,
		hasherMock,
		marshalizerMock,
		multiSignerMock,
		rounderMock,
		nil,
		syncTimerMock,
		sendConsensusMessage,
		extend,
	)

	assert.Nil(t, srBlock)
	assert.Equal(t, err, spos.ErrNilShardCoordinator)
}

func TestSubroundBlock_NewSubroundBlockNilSyncTimerShouldFail(t *testing.T) {
	t.Parallel()

	blockChain := blockchain.BlockChain{}
	blockProcessorMock := initBlockProcessorMock()
	consensusState := initConsensusState()
	hasherMock := mock.HasherMock{}
	marshalizerMock := mock.MarshalizerMock{}
	multiSignerMock := initMultiSignerMock()
	rounderMock := initRounderMock()
	shardCoordinatorMock := mock.ShardCoordinatorMock{}

	ch := make(chan bool, 1)

	sr, _ := bn.NewSubround(
		int(bn.SrStartRound),
		int(bn.SrBlock),
		int(bn.SrCommitmentHash),
		int64(5*roundTimeDuration/100),
		int64(25*roundTimeDuration/100),
		"(BLOCK)",
		ch,
	)

	srBlock, err := bn.NewSubroundBlock(
		sr,
		&blockChain,
		blockProcessorMock,
		consensusState,
		hasherMock,
		marshalizerMock,
		multiSignerMock,
		rounderMock,
		shardCoordinatorMock,
		nil,
		sendConsensusMessage,
		extend,
	)

	assert.Nil(t, srBlock)
	assert.Equal(t, err, spos.ErrNilSyncTimer)
}

func TestSubroundBlock_NewSubroundBlockNilSendConsensusMessageFunctionShouldFail(t *testing.T) {
	t.Parallel()

	blockChain := blockchain.BlockChain{}
	blockProcessorMock := initBlockProcessorMock()
	consensusState := initConsensusState()
	hasherMock := mock.HasherMock{}
	marshalizerMock := mock.MarshalizerMock{}
	multiSignerMock := initMultiSignerMock()
	rounderMock := initRounderMock()
	shardCoordinatorMock := mock.ShardCoordinatorMock{}
	syncTimerMock := mock.SyncTimerMock{}

	ch := make(chan bool, 1)

	sr, _ := bn.NewSubround(
		int(bn.SrStartRound),
		int(bn.SrBlock),
		int(bn.SrCommitmentHash),
		int64(5*roundTimeDuration/100),
		int64(25*roundTimeDuration/100),
		"(BLOCK)",
		ch,
	)

	srBlock, err := bn.NewSubroundBlock(
		sr,
		&blockChain,
		blockProcessorMock,
		consensusState,
		hasherMock,
		marshalizerMock,
		multiSignerMock,
		rounderMock,
		shardCoordinatorMock,
		syncTimerMock,
		nil,
		extend,
	)

	assert.Nil(t, srBlock)
	assert.Equal(t, err, spos.ErrNilSendConsensusMessageFunction)
}

func TestSubroundBlock_NewSubroundBlockShouldWork(t *testing.T) {
	t.Parallel()

	blockChain := blockchain.BlockChain{}
	blockProcessorMock := initBlockProcessorMock()
	consensusState := initConsensusState()
	hasherMock := mock.HasherMock{}
	marshalizerMock := mock.MarshalizerMock{}
	multiSignerMock := initMultiSignerMock()
	rounderMock := initRounderMock()
	shardCoordinatorMock := mock.ShardCoordinatorMock{}
	syncTimerMock := mock.SyncTimerMock{}

	ch := make(chan bool, 1)

	sr, _ := bn.NewSubround(
		int(bn.SrStartRound),
		int(bn.SrBlock),
		int(bn.SrCommitmentHash),
		int64(5*roundTimeDuration/100),
		int64(25*roundTimeDuration/100),
		"(BLOCK)",
		ch,
	)

	srBlock, err := bn.NewSubroundBlock(
		sr,
		&blockChain,
		blockProcessorMock,
		consensusState,
		hasherMock,
		marshalizerMock,
		multiSignerMock,
		rounderMock,
		shardCoordinatorMock,
		syncTimerMock,
		sendConsensusMessage,
		extend,
	)

	assert.NotNil(t, srBlock)
	assert.Nil(t, err)
}

func TestSubroundBlock_DoBlockJob(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()

	r := sr.DoBlockJob()
	assert.False(t, r)

	sr.ConsensusState().SetSelfPubKey(sr.ConsensusState().ConsensusGroup()[0])
	sr.ConsensusState().SetJobDone(sr.ConsensusState().SelfPubKey(), bn.SrBlock, true)
	r = sr.DoBlockJob()
	assert.False(t, r)

	sr.ConsensusState().SetJobDone(sr.ConsensusState().SelfPubKey(), bn.SrBlock, false)
	sr.ConsensusState().SetStatus(bn.SrBlock, spos.SsFinished)
	r = sr.DoBlockJob()
	assert.False(t, r)

	sr.ConsensusState().SetStatus(bn.SrBlock, spos.SsNotFinished)

	bpm := &mock.BlockProcessorMock{
		GetRootHashCalled: func() []byte {
			return []byte{}
		},
	}
	err := errors.New("error")
	bpm.CreateBlockCalled = func(shardId uint32, maxTxInBlock int, round int32, remainingTime func() bool) (data.BodyHandler, error) {
		return nil, err
	}
	sr.SetBlockProcessor(bpm)

	r = sr.DoBlockJob()
	assert.False(t, r)

	bpm = initBlockProcessorMock()
	sr.SetBlockProcessor(bpm)

	r = sr.DoBlockJob()
	assert.True(t, r)
	assert.Equal(t, uint64(1), sr.ConsensusState().Header.Nonce)
}

func TestSubroundBlock_ReceivedBlock(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()

	blockProcessorMock := initBlockProcessorMock()

	blBody := make(block.Body, 0)

	blBodyStr, _ := mock.MarshalizerMock{}.Marshal(blBody)

	cnsMsg := spos.NewConsensusMessage(
		nil,
		blBodyStr,
		[]byte(sr.ConsensusState().ConsensusGroup()[0]),
		[]byte("sig"),
		int(bn.MtBlockBody),
		uint64(sr.Rounder().TimeStamp().Unix()),
		0,
	)

	sr.ConsensusState().BlockBody = make(block.Body, 0)
	r := sr.ReceivedBlockBody(cnsMsg)
	assert.False(t, r)

	sr.ConsensusState().BlockBody = nil
	cnsMsg.PubKey = []byte(sr.ConsensusState().ConsensusGroup()[1])
	r = sr.ReceivedBlockBody(cnsMsg)
	assert.False(t, r)

	cnsMsg.PubKey = []byte(sr.ConsensusState().ConsensusGroup()[0])
	sr.ConsensusState().SetStatus(bn.SrBlock, spos.SsFinished)
	r = sr.ReceivedBlockBody(cnsMsg)
	assert.False(t, r)

	sr.ConsensusState().SetStatus(bn.SrBlock, spos.SsNotFinished)
	r = sr.ReceivedBlockBody(cnsMsg)
	assert.False(t, r)

	hdr := &block.Header{}
	hdr.Nonce = 2

	hdrStr, _ := mock.MarshalizerMock{}.Marshal(hdr)
	hdrHash := mock.HasherMock{}.Compute(string(hdrStr))

	cnsMsg = spos.NewConsensusMessage(
		hdrHash,
		hdrStr,
		[]byte(sr.ConsensusState().ConsensusGroup()[0]),
		[]byte("sig"),
		int(bn.MtBlockHeader),
		uint64(sr.Rounder().TimeStamp().Unix()),
		0,
	)

	r = sr.ReceivedBlockHeader(cnsMsg)
	assert.False(t, r)

	sr.ConsensusState().Data = nil
	sr.ConsensusState().Header = hdr
	r = sr.ReceivedBlockHeader(cnsMsg)
	assert.False(t, r)

	sr.ConsensusState().Header = nil
	cnsMsg.PubKey = []byte(sr.ConsensusState().ConsensusGroup()[1])
	r = sr.ReceivedBlockHeader(cnsMsg)
	assert.False(t, r)

	cnsMsg.PubKey = []byte(sr.ConsensusState().ConsensusGroup()[0])
	sr.ConsensusState().SetStatus(bn.SrBlock, spos.SsFinished)
	r = sr.ReceivedBlockHeader(cnsMsg)
	assert.False(t, r)

	blockProcessorMock.CheckBlockValidityCalled = func(blockChain *blockchain.BlockChain, header data.HeaderHandler, body data.BodyHandler) bool {
		return false
	}

	sr.SetBlockProcessor(blockProcessorMock)

	sr.ConsensusState().SetStatus(bn.SrBlock, spos.SsNotFinished)
	r = sr.ReceivedBlockHeader(cnsMsg)
	assert.False(t, r)

	blockProcessorMock.CheckBlockValidityCalled = func(blockChain *blockchain.BlockChain, header data.HeaderHandler, body data.BodyHandler) bool {
		return true
	}

	sr.SetBlockProcessor(blockProcessorMock)

	sr.ConsensusState().Data = nil
	sr.ConsensusState().Header = nil

	hdr = &block.Header{}
	hdr.Nonce = 1

	hdrStr, _ = mock.MarshalizerMock{}.Marshal(hdr)
	hdrHash = mock.HasherMock{}.Compute(string(hdrStr))
	cnsMsg.BlockHeaderHash = hdrHash
	cnsMsg.SubRoundData = hdrStr

	r = sr.ReceivedBlockHeader(cnsMsg)
	assert.True(t, r)
}

func TestSubroundBlock_DecodeBlockBody(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()

	body := make(block.Body, 0)
	body = append(body, &block.MiniBlock{ShardID: 69})

	message, err := mock.MarshalizerMock{}.Marshal(body)

	assert.Nil(t, err)

	dcdBlk := sr.DecodeBlockBody(nil)

	assert.Nil(t, dcdBlk)

	dcdBlk = sr.DecodeBlockBody(message)

	assert.Equal(t, body, dcdBlk)
	assert.Equal(t, uint32(69), body[0].ShardID)
}

func TestSubroundBlock_DecodeBlockHeader(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()

	hdr := &block.Header{}
	hdr.Nonce = 1
	hdr.TimeStamp = uint64(sr.Rounder().TimeStamp().Unix())
	hdr.Signature = []byte(sr.ConsensusState().SelfPubKey())

	message, err := mock.MarshalizerMock{}.Marshal(hdr)

	assert.Nil(t, err)

	message, err = mock.MarshalizerMock{}.Marshal(hdr)

	assert.Nil(t, err)

	dcdHdr := sr.DecodeBlockHeader(nil)

	assert.Nil(t, dcdHdr)

	dcdHdr = sr.DecodeBlockHeader(message)

	assert.Equal(t, hdr, dcdHdr)
	assert.Equal(t, []byte(sr.ConsensusState().SelfPubKey()), dcdHdr.Signature)
}

func TestSubroundBlock_ProcessReceivedBlockShouldReturnFalseWhenBodyAndHeaderAreNotSet(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()

	blk := make(block.Body, 0)
	message, _ := mock.MarshalizerMock{}.Marshal(blk)

	cnsMsg := spos.NewConsensusMessage(
		message,
		nil,
		[]byte(sr.ConsensusState().ConsensusGroup()[0]),
		[]byte("sig"),
		int(bn.MtBlockBody),
		uint64(sr.Rounder().TimeStamp().Unix()),
		0,
	)

	assert.False(t, sr.ProcessReceivedBlock(cnsMsg))
}

func TestSubroundBlock_ProcessReceivedBlockShouldReturnFalseWhenProcessBlockFails(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()

	blProcMock := initBlockProcessorMock()

	err := errors.New("error process block")
	blProcMock.ProcessBlockCalled = func(*blockchain.BlockChain, data.HeaderHandler, data.BodyHandler, func() time.Duration) error {
		return err
	}

	sr.SetBlockProcessor(blProcMock)

	hdr := &block.Header{}
	blk := make(block.Body, 0)
	message, _ := mock.MarshalizerMock{}.Marshal(blk)

	cnsMsg := spos.NewConsensusMessage(
		message,
		nil,
		[]byte(sr.ConsensusState().ConsensusGroup()[0]),
		[]byte("sig"),
		int(bn.MtBlockBody),
		uint64(sr.Rounder().TimeStamp().Unix()),
		0,
	)

	sr.ConsensusState().Header = hdr
	sr.ConsensusState().BlockBody = blk

	assert.False(t, sr.ProcessReceivedBlock(cnsMsg))
}

func TestSubroundBlock_ProcessReceivedBlockShouldReturnFalseWhenProcessBlockReturnsInNextRound(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()

	hdr := &block.Header{}
	blk := make(block.Body, 0)
	message, _ := mock.MarshalizerMock{}.Marshal(blk)

	cnsMsg := spos.NewConsensusMessage(
		message,
		nil,
		[]byte(sr.ConsensusState().ConsensusGroup()[0]),
		[]byte("sig"),
		int(bn.MtBlockBody),
		uint64(sr.Rounder().TimeStamp().Unix()),
		0,
	)

	sr.ConsensusState().Header = hdr
	sr.ConsensusState().BlockBody = blk

	sr.SetRounder(&mock.RounderMock{RemainingTimeInRoundCalled: func(safeThresholdPercent uint32) time.Duration {
		return time.Duration(-1)
	}})

	assert.False(t, sr.ProcessReceivedBlock(cnsMsg))
}

func TestSubroundBlock_ProcessReceivedBlockShouldReturnTrue(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()

	hdr := &block.Header{}
	blk := make(block.Body, 0)
	message, _ := mock.MarshalizerMock{}.Marshal(blk)

	cnsMsg := spos.NewConsensusMessage(
		message,
		nil,
		[]byte(sr.ConsensusState().ConsensusGroup()[0]),
		[]byte("sig"),
		int(bn.MtBlockBody),
		uint64(sr.Rounder().TimeStamp().Unix()),
		0,
	)

	sr.ConsensusState().Header = hdr
	sr.ConsensusState().BlockBody = blk

	assert.True(t, sr.ProcessReceivedBlock(cnsMsg))
}

func TestSubroundBlock_RemainingTimeShouldReturnNegativeValue(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()

	remainingTimeInThisRound := func() time.Duration {
		roundStartTime := sr.Rounder().TimeStamp()
		currentTime := sr.SyncTimer().CurrentTime()
		elapsedTime := currentTime.Sub(roundStartTime)
		remainingTime := sr.Rounder().TimeDuration()*85/100 - elapsedTime

		return time.Duration(remainingTime)
	}

	sr.SetSyncTimer(mock.SyncTimerMock{CurrentTimeCalled: func() time.Time {
		return time.Unix(0, 0).Add(roundTimeDuration * 84 / 100)
	}})

	ret := remainingTimeInThisRound()
	assert.True(t, ret > 0)

	sr.SetSyncTimer(mock.SyncTimerMock{CurrentTimeCalled: func() time.Time {
		return time.Unix(0, 0).Add(roundTimeDuration * 85 / 100)
	}})

	ret = remainingTimeInThisRound()
	assert.True(t, ret == 0)

	sr.SetSyncTimer(mock.SyncTimerMock{CurrentTimeCalled: func() time.Time {
		return time.Unix(0, 0).Add(roundTimeDuration * 86 / 100)
	}})

	ret = remainingTimeInThisRound()
	assert.True(t, ret < 0)

}

func TestSubroundBlock_DoBlockConsensusCheckShouldReturnFalseWhenRoundIsCanceled(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()
	sr.ConsensusState().RoundCanceled = true
	assert.False(t, sr.DoBlockConsensusCheck())
}

func TestSubroundBlock_DoBlockConsensusCheckShouldReturnTrueWhenSubroundIsFinished(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()
	sr.ConsensusState().SetStatus(bn.SrBlock, spos.SsFinished)
	assert.True(t, sr.DoBlockConsensusCheck())
}

func TestSubroundBlock_DoBlockConsensusCheckShouldReturnTrueWhenBlockIsReceivedReturnTrue(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()

	for i := 0; i < sr.ConsensusState().Threshold(bn.SrBlock); i++ {
		sr.ConsensusState().SetJobDone(sr.ConsensusState().ConsensusGroup()[i], bn.SrBlock, true)
	}

	assert.True(t, sr.DoBlockConsensusCheck())
}

func TestSubroundBlock_DoBlockConsensusCheckShouldReturnFalseWhenBlockIsReceivedReturnFalse(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()
	assert.False(t, sr.DoBlockConsensusCheck())
}

func TestSubroundBlock_IsBlockReceived(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()

	for i := 0; i < len(sr.ConsensusState().ConsensusGroup()); i++ {
		sr.ConsensusState().SetJobDone(sr.ConsensusState().ConsensusGroup()[i], bn.SrBlock, false)
		sr.ConsensusState().SetJobDone(sr.ConsensusState().ConsensusGroup()[i], bn.SrCommitmentHash, false)
		sr.ConsensusState().SetJobDone(sr.ConsensusState().ConsensusGroup()[i], bn.SrBitmap, false)
		sr.ConsensusState().SetJobDone(sr.ConsensusState().ConsensusGroup()[i], bn.SrCommitment, false)
		sr.ConsensusState().SetJobDone(sr.ConsensusState().ConsensusGroup()[i], bn.SrSignature, false)
	}

	ok := sr.IsBlockReceived(1)
	assert.False(t, ok)

	sr.ConsensusState().SetJobDone("A", bn.SrBlock, true)
	isJobDone, _ := sr.ConsensusState().JobDone("A", bn.SrBlock)

	assert.True(t, isJobDone)

	ok = sr.IsBlockReceived(1)
	assert.True(t, ok)

	ok = sr.IsBlockReceived(2)
	assert.False(t, ok)
}

func TestSubroundBlock_HaveTimeInCurrentSubroundShouldReturnTrue(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()

	haveTimeInCurrentSubound := func() bool {
		roundStartTime := sr.Rounder().TimeStamp()
		currentTime := sr.SyncTimer().CurrentTime()
		elapsedTime := currentTime.Sub(roundStartTime)
		remainingTime := sr.EndTime() - int64(elapsedTime)

		return time.Duration(remainingTime) > 0
	}

	rounderMock := &mock.RounderMock{}
	rounderMock.RoundTimeDuration = time.Duration(4000 * time.Millisecond)
	rounderMock.RoundTimeStamp = time.Unix(0, 0)

	syncTimerMock := &mock.SyncTimerMock{}

	timeElapsed := int64(sr.EndTime() - 1)

	syncTimerMock.CurrentTimeCalled = func() time.Time {
		return time.Unix(0, timeElapsed)
	}

	sr.SetRounder(rounderMock)
	sr.SetSyncTimer(syncTimerMock)

	assert.True(t, haveTimeInCurrentSubound())
}

func TestSubroundBlock_HaveTimeInCurrentSuboundShouldReturnFalse(t *testing.T) {
	t.Parallel()

	sr := *initSubroundBlock()

	haveTimeInCurrentSubound := func() bool {
		roundStartTime := sr.Rounder().TimeStamp()
		currentTime := sr.SyncTimer().CurrentTime()
		elapsedTime := currentTime.Sub(roundStartTime)
		remainingTime := sr.EndTime() - int64(elapsedTime)

		return time.Duration(remainingTime) > 0
	}

	rounderMock := &mock.RounderMock{}
	rounderMock.RoundTimeDuration = time.Duration(4000 * time.Millisecond)
	rounderMock.RoundTimeStamp = time.Unix(0, 0)

	syncTimerMock := &mock.SyncTimerMock{}

	timeElapsed := int64(sr.EndTime() + 1)

	syncTimerMock.CurrentTimeCalled = func() time.Time {
		return time.Unix(0, timeElapsed)
	}

	sr.SetRounder(rounderMock)
	sr.SetSyncTimer(syncTimerMock)

	assert.False(t, haveTimeInCurrentSubound())
}
