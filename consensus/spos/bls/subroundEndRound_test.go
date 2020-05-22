package bls_test

import (
	"errors"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/mock"
	"github.com/ElrondNetwork/elrond-go/consensus/spos"
	"github.com/ElrondNetwork/elrond-go/consensus/spos/bls"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/blockchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initSubroundEndRoundWithContainer(container *mock.ConsensusCoreMock) bls.SubroundEndRound {
	ch := make(chan bool, 1)
	consensusState := initConsensusState()
	sr, _ := spos.NewSubround(
		bls.SrSignature,
		bls.SrEndRound,
		-1,
		int64(85*roundTimeDuration/100),
		int64(95*roundTimeDuration/100),
		"(END_ROUND)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)

	srEndRound, _ := bls.NewSubroundEndRound(
		sr,
		extend,
		bls.ProcessingThresholdPercent,
		displayStatistics,
	)

	return srEndRound
}

func initSubroundEndRound() bls.SubroundEndRound {
	container := mock.InitConsensusCore()
	return initSubroundEndRoundWithContainer(container)
}

func TestSubroundEndRound_NewSubroundEndRoundNilSubroundShouldFail(t *testing.T) {
	t.Parallel()
	srEndRound, err := bls.NewSubroundEndRound(
		nil,
		extend,
		bls.ProcessingThresholdPercent,
		displayStatistics,
	)

	assert.Nil(t, srEndRound)
	assert.Equal(t, spos.ErrNilSubround, err)
}

func TestSubroundEndRound_NewSubroundEndRoundNilBlockChainShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrSignature,
		bls.SrEndRound,
		-1,
		int64(85*roundTimeDuration/100),
		int64(95*roundTimeDuration/100),
		"(END_ROUND)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)
	container.SetBlockchain(nil)
	srEndRound, err := bls.NewSubroundEndRound(
		sr,
		extend,
		bls.ProcessingThresholdPercent,
		displayStatistics,
	)

	assert.Nil(t, srEndRound)
	assert.Equal(t, spos.ErrNilBlockChain, err)
}

func TestSubroundEndRound_NewSubroundEndRoundNilBlockProcessorShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrSignature,
		bls.SrEndRound,
		-1,
		int64(85*roundTimeDuration/100),
		int64(95*roundTimeDuration/100),
		"(END_ROUND)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)
	container.SetBlockProcessor(nil)
	srEndRound, err := bls.NewSubroundEndRound(
		sr,
		extend,
		bls.ProcessingThresholdPercent,
		displayStatistics,
	)

	assert.Nil(t, srEndRound)
	assert.Equal(t, spos.ErrNilBlockProcessor, err)
}

func TestSubroundEndRound_NewSubroundEndRoundNilConsensusStateShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrSignature,
		bls.SrEndRound,
		-1,
		int64(85*roundTimeDuration/100),
		int64(95*roundTimeDuration/100),
		"(END_ROUND)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)

	sr.ConsensusState = nil
	srEndRound, err := bls.NewSubroundEndRound(
		sr,
		extend,
		bls.ProcessingThresholdPercent,
		displayStatistics,
	)

	assert.Nil(t, srEndRound)
	assert.Equal(t, spos.ErrNilConsensusState, err)
}

func TestSubroundEndRound_NewSubroundEndRoundNilMultisignerShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrSignature,
		bls.SrEndRound,
		-1,
		int64(85*roundTimeDuration/100),
		int64(95*roundTimeDuration/100),
		"(END_ROUND)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)
	container.SetMultiSigner(nil)
	srEndRound, err := bls.NewSubroundEndRound(
		sr,
		extend,
		bls.ProcessingThresholdPercent,
		displayStatistics,
	)

	assert.Nil(t, srEndRound)
	assert.Equal(t, spos.ErrNilMultiSigner, err)
}

func TestSubroundEndRound_NewSubroundEndRoundNilRounderShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrSignature,
		bls.SrEndRound,
		-1,
		int64(85*roundTimeDuration/100),
		int64(95*roundTimeDuration/100),
		"(END_ROUND)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)
	container.SetRounder(nil)
	srEndRound, err := bls.NewSubroundEndRound(
		sr,
		extend,
		bls.ProcessingThresholdPercent,
		displayStatistics,
	)

	assert.Nil(t, srEndRound)
	assert.Equal(t, spos.ErrNilRounder, err)
}

func TestSubroundEndRound_NewSubroundEndRoundNilSyncTimerShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrSignature,
		bls.SrEndRound,
		-1,
		int64(85*roundTimeDuration/100),
		int64(95*roundTimeDuration/100),
		"(END_ROUND)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)
	container.SetSyncTimer(nil)
	srEndRound, err := bls.NewSubroundEndRound(
		sr,
		extend,
		bls.ProcessingThresholdPercent,
		displayStatistics,
	)

	assert.Nil(t, srEndRound)
	assert.Equal(t, spos.ErrNilSyncTimer, err)
}

func TestSubroundEndRound_NewSubroundEndRoundShouldWork(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrSignature,
		bls.SrEndRound,
		-1,
		int64(85*roundTimeDuration/100),
		int64(95*roundTimeDuration/100),
		"(END_ROUND)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)

	srEndRound, err := bls.NewSubroundEndRound(
		sr,
		extend,
		bls.ProcessingThresholdPercent,
		displayStatistics,
	)

	assert.NotNil(t, srEndRound)
	assert.Nil(t, err)
}

func TestSubroundEndRound_SetAppStatusHandlerNilAshShouldErr(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()

	err := sr.SetAppStatusHandler(nil)
	assert.Equal(t, spos.ErrNilAppStatusHandler, err)
}

func TestSubroundEndRound_SetAppStatusHandlerOkAshShouldWork(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()

	err := sr.SetAppStatusHandler(&mock.AppStatusHandlerStub{})
	assert.Nil(t, err)
}

func TestSubroundEndRound_DoEndRoundJobErrAggregatingSigShouldFail(t *testing.T) {
	t.Parallel()
	container := mock.InitConsensusCore()
	sr := *initSubroundEndRoundWithContainer(container)
	multiSignerMock := mock.InitMultiSignerMock()
	multiSignerMock.AggregateSigsMock = func(bitmap []byte) ([]byte, error) {
		return nil, crypto.ErrNilHasher
	}

	container.SetMultiSigner(multiSignerMock)
	sr.Header = &block.Header{}

	sr.SetSelfPubKey("A")

	assert.True(t, sr.IsSelfLeaderInCurrentRound())
	r := sr.DoEndRoundJob()
	assert.False(t, r)
}

func TestSubroundEndRound_DoEndRoundJobErrCommitBlockShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	sr := *initSubroundEndRoundWithContainer(container)
	sr.SetSelfPubKey("A")

	blProcMock := mock.InitBlockProcessorMock()
	blProcMock.CommitBlockCalled = func(
		header data.HeaderHandler,
		body data.BodyHandler,
	) error {
		return blockchain.ErrHeaderUnitNil
	}

	container.SetBlockProcessor(blProcMock)
	sr.Header = &block.Header{}

	r := sr.DoEndRoundJob()
	assert.False(t, r)
}

func TestSubroundEndRound_DoEndRoundJobErrBroadcastBlockOK(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	bm := &mock.BroadcastMessengerMock{
		BroadcastBlockCalled: func(handler data.BodyHandler, handler2 data.HeaderHandler) error {
			return errors.New("error")
		},
	}
	container.SetBroadcastMessenger(bm)
	sr := *initSubroundEndRoundWithContainer(container)
	sr.SetSelfPubKey("A")

	sr.Header = &block.Header{}

	r := sr.DoEndRoundJob()
	assert.True(t, r)
}

func TestSubroundEndRound_DoEndRoundJobErrMarshalizedDataToBroadcastOK(t *testing.T) {
	t.Parallel()

	err := errors.New("")
	container := mock.InitConsensusCore()

	bpm := mock.InitBlockProcessorMock()
	bpm.MarshalizedDataToBroadcastCalled = func(header data.HeaderHandler, body data.BodyHandler) (map[uint32][]byte, map[string][][]byte, error) {
		err = errors.New("error marshalized data to broadcast")
		return make(map[uint32][]byte), make(map[string][][]byte), err
	}
	container.SetBlockProcessor(bpm)

	bm := &mock.BroadcastMessengerMock{
		BroadcastBlockCalled: func(handler data.BodyHandler, handler2 data.HeaderHandler) error {
			return nil
		},
		BroadcastMiniBlocksCalled: func(bytes map[uint32][]byte) error {
			return nil
		},
		BroadcastTransactionsCalled: func(bytes map[string][][]byte) error {
			return nil
		},
	}
	container.SetBroadcastMessenger(bm)
	sr := *initSubroundEndRoundWithContainer(container)
	sr.SetSelfPubKey("A")

	sr.Header = &block.Header{}

	r := sr.DoEndRoundJob()
	assert.True(t, r)
	assert.Equal(t, errors.New("error marshalized data to broadcast"), err)
}

func TestSubroundEndRound_DoEndRoundJobErrBroadcastMiniBlocksOK(t *testing.T) {
	t.Parallel()

	err := errors.New("")
	container := mock.InitConsensusCore()

	bpm := mock.InitBlockProcessorMock()
	bpm.MarshalizedDataToBroadcastCalled = func(header data.HeaderHandler, body data.BodyHandler) (map[uint32][]byte, map[string][][]byte, error) {
		return make(map[uint32][]byte), make(map[string][][]byte), nil
	}
	container.SetBlockProcessor(bpm)

	bm := &mock.BroadcastMessengerMock{
		BroadcastBlockCalled: func(handler data.BodyHandler, handler2 data.HeaderHandler) error {
			return nil
		},
		BroadcastMiniBlocksCalled: func(bytes map[uint32][]byte) error {
			err = errors.New("error broadcast miniblocks")
			return err
		},
		BroadcastTransactionsCalled: func(bytes map[string][][]byte) error {
			return nil
		},
	}
	container.SetBroadcastMessenger(bm)
	sr := *initSubroundEndRoundWithContainer(container)
	sr.SetSelfPubKey("A")

	sr.Header = &block.Header{}

	r := sr.DoEndRoundJob()
	assert.True(t, r)
	assert.Equal(t, errors.New("error broadcast miniblocks"), err)
}

func TestSubroundEndRound_DoEndRoundJobErrBroadcastTransactionsOK(t *testing.T) {
	t.Parallel()

	err := errors.New("")
	container := mock.InitConsensusCore()

	bpm := mock.InitBlockProcessorMock()
	bpm.MarshalizedDataToBroadcastCalled = func(header data.HeaderHandler, body data.BodyHandler) (map[uint32][]byte, map[string][][]byte, error) {
		return make(map[uint32][]byte), make(map[string][][]byte), nil
	}
	container.SetBlockProcessor(bpm)

	bm := &mock.BroadcastMessengerMock{
		BroadcastBlockCalled: func(handler data.BodyHandler, handler2 data.HeaderHandler) error {
			return nil
		},
		BroadcastMiniBlocksCalled: func(bytes map[uint32][]byte) error {
			return nil
		},
		BroadcastTransactionsCalled: func(bytes map[string][][]byte) error {
			err = errors.New("error broadcast transactions")
			return err
		},
	}
	container.SetBroadcastMessenger(bm)
	sr := *initSubroundEndRoundWithContainer(container)
	sr.SetSelfPubKey("A")

	sr.Header = &block.Header{}

	r := sr.DoEndRoundJob()
	assert.True(t, r)
	assert.Equal(t, errors.New("error broadcast transactions"), err)
}

func TestSubroundEndRound_DoEndRoundJobAllOK(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	bm := &mock.BroadcastMessengerMock{
		BroadcastBlockCalled: func(handler data.BodyHandler, handler2 data.HeaderHandler) error {
			return errors.New("error")
		},
	}
	container.SetBroadcastMessenger(bm)
	sr := *initSubroundEndRoundWithContainer(container)
	sr.SetSelfPubKey("A")

	sr.Header = &block.Header{}

	r := sr.DoEndRoundJob()
	assert.True(t, r)
}

func TestSubroundEndRound_CheckIfSignatureIsFilled(t *testing.T) {
	t.Parallel()

	expectedSignature := []byte("signature")
	container := mock.InitConsensusCore()
	singleSigner := &mock.SingleSignerMock{
		SignStub: func(private crypto.PrivateKey, msg []byte) ([]byte, error) {
			var receivedHdr block.Header
			_ = container.Marshalizer().Unmarshal(&receivedHdr, msg)
			return expectedSignature, nil
		},
	}
	container.SetSingleSigner(singleSigner)
	bm := &mock.BroadcastMessengerMock{
		BroadcastBlockCalled: func(handler data.BodyHandler, handler2 data.HeaderHandler) error {
			return errors.New("error")
		},
	}
	container.SetBroadcastMessenger(bm)
	sr := *initSubroundEndRoundWithContainer(container)
	sr.SetSelfPubKey("A")

	sr.Header = &block.Header{Nonce: 5}

	r := sr.DoEndRoundJob()
	assert.True(t, r)
	assert.Equal(t, expectedSignature, sr.Header.GetLeaderSignature())
}

func TestSubroundEndRound_DoEndRoundConsensusCheckShouldReturnFalseWhenRoundIsCanceled(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()
	sr.RoundCanceled = true

	ok := sr.DoEndRoundConsensusCheck()
	assert.False(t, ok)
}

func TestSubroundEndRound_DoEndRoundConsensusCheckShouldReturnTrueWhenRoundIsFinished(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()
	sr.SetStatus(bls.SrEndRound, spos.SsFinished)

	ok := sr.DoEndRoundConsensusCheck()
	assert.True(t, ok)
}

func TestSubroundEndRound_DoEndRoundConsensusCheckShouldReturnFalseWhenRoundIsNotFinished(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()

	ok := sr.DoEndRoundConsensusCheck()
	assert.False(t, ok)
}

func TestSubroundEndRound_CheckSignaturesValidityShouldErrNilSignature(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()

	err := sr.CheckSignaturesValidity([]byte(string(2)))
	assert.Equal(t, spos.ErrNilSignature, err)
}

func TestSubroundEndRound_CheckSignaturesValidityShouldErrIndexOutOfBounds(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	sr := *initSubroundEndRoundWithContainer(container)
	_, _ = sr.MultiSigner().Create(nil, 0)
	_ = sr.SetJobDone(sr.ConsensusGroup()[0], bls.SrSignature, true)

	multiSignerMock := mock.InitMultiSignerMock()
	multiSignerMock.SignatureShareMock = func(index uint16) ([]byte, error) {
		return nil, crypto.ErrIndexOutOfBounds
	}
	container.SetMultiSigner(multiSignerMock)

	err := sr.CheckSignaturesValidity([]byte(string(1)))
	assert.Equal(t, crypto.ErrIndexOutOfBounds, err)
}

func TestSubroundEndRound_CheckSignaturesValidityShouldErrInvalidSignatureShare(t *testing.T) {
	t.Parallel()
	container := mock.InitConsensusCore()
	sr := *initSubroundEndRoundWithContainer(container)
	multiSignerMock := mock.InitMultiSignerMock()
	err := errors.New("invalid signature share")
	multiSignerMock.VerifySignatureShareMock = func(index uint16, sig []byte, msg []byte, bitmap []byte) error {
		return err
	}
	container.SetMultiSigner(multiSignerMock)

	_ = sr.SetJobDone(sr.ConsensusGroup()[0], bls.SrSignature, true)

	err2 := sr.CheckSignaturesValidity([]byte(string(1)))
	assert.Equal(t, err, err2)
}

func TestSubroundEndRound_CheckSignaturesValidityShouldReturnNil(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()

	_ = sr.SetJobDone(sr.ConsensusGroup()[0], bls.SrSignature, true)

	err := sr.CheckSignaturesValidity([]byte(string(1)))
	assert.Equal(t, nil, err)
}

func TestSubroundEndRound_DoEndRoundJobByParticipant_RoundCanceledShouldReturnFalse(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()
	sr.RoundCanceled = true

	cnsData := consensus.Message{}
	res := sr.DoEndRoundJobByParticipant(&cnsData)
	assert.False(t, res)
}

func TestSubroundEndRound_DoEndRoundJobByParticipant_ConsensusDataNotSetShouldReturnFalse(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()
	sr.Data = nil

	cnsData := consensus.Message{}
	res := sr.DoEndRoundJobByParticipant(&cnsData)
	assert.False(t, res)
}

func TestSubroundEndRound_DoEndRoundJobByParticipant_PreviousSubroundNotFinishedShouldReturnFalse(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()
	sr.SetStatus(2, spos.SsNotFinished)
	cnsData := consensus.Message{}
	res := sr.DoEndRoundJobByParticipant(&cnsData)
	assert.False(t, res)
}

func TestSubroundEndRound_DoEndRoundJobByParticipant_CurrentSubroundFinishedShouldReturnFalse(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()

	// set previous as finished
	sr.SetStatus(2, spos.SsFinished)

	// set current as finished
	sr.SetStatus(3, spos.SsFinished)

	cnsData := consensus.Message{}
	res := sr.DoEndRoundJobByParticipant(&cnsData)
	assert.False(t, res)
}

func TestSubroundEndRound_DoEndRoundJobByParticipant_ConsensusHeaderNotReceivedShouldReturnFalse(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()

	// set previous as finished
	sr.SetStatus(2, spos.SsFinished)

	// set current as not finished
	sr.SetStatus(3, spos.SsNotFinished)

	cnsData := consensus.Message{}
	res := sr.DoEndRoundJobByParticipant(&cnsData)
	assert.False(t, res)
}

func TestSubroundEndRound_DoEndRoundJobByParticipant_ShouldReturnTrue(t *testing.T) {
	t.Parallel()

	hdr := &block.Header{Nonce: 37}
	sr := *initSubroundEndRound()
	sr.Header = hdr
	sr.AddReceivedHeader(hdr)

	// set previous as finished
	sr.SetStatus(2, spos.SsFinished)

	// set current as not finished
	sr.SetStatus(3, spos.SsNotFinished)

	cnsData := consensus.Message{}
	res := sr.DoEndRoundJobByParticipant(&cnsData)
	assert.True(t, res)
}

func TestSubroundEndRound_IsConsensusHeaderReceived_NoReceivedHeadersShouldReturnFalse(t *testing.T) {
	t.Parallel()

	hdr := &block.Header{Nonce: 37}
	sr := *initSubroundEndRound()
	sr.Header = hdr

	res, retHdr := sr.IsConsensusHeaderReceived()
	assert.False(t, res)
	assert.Nil(t, retHdr)
}

func TestSubroundEndRound_IsConsensusHeaderReceived_HeaderNotReceivedShouldReturnFalse(t *testing.T) {
	t.Parallel()

	hdr := &block.Header{Nonce: 37}
	hdrToSearchFor := &block.Header{Nonce: 38}
	sr := *initSubroundEndRound()
	sr.AddReceivedHeader(hdr)
	sr.Header = hdrToSearchFor

	res, retHdr := sr.IsConsensusHeaderReceived()
	assert.False(t, res)
	assert.Nil(t, retHdr)
}

func TestSubroundEndRound_IsConsensusHeaderReceivedShouldReturnTrue(t *testing.T) {
	t.Parallel()

	hdr := &block.Header{Nonce: 37}
	sr := *initSubroundEndRound()
	sr.Header = hdr
	sr.AddReceivedHeader(hdr)

	res, retHdr := sr.IsConsensusHeaderReceived()
	assert.True(t, res)
	assert.Equal(t, hdr, retHdr)
}

func TestSubroundEndRound_HaveConsensusHeaderWithFullInfoNilHdrShouldNotWork(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()

	cnsData := consensus.Message{}

	haveHdr, hdr := sr.HaveConsensusHeaderWithFullInfo(&cnsData)
	assert.False(t, haveHdr)
	assert.Nil(t, hdr)
}

func TestSubroundEndRound_HaveConsensusHeaderWithFullInfoShouldWork(t *testing.T) {
	t.Parallel()

	originalPubKeyBitMap := []byte{0, 1, 2}
	newPubKeyBitMap := []byte{3, 4, 5}
	originalLeaderSig := []byte{6, 7, 8}
	newLeaderSig := []byte{9, 10, 11}
	originalSig := []byte{12, 13, 14}
	newSig := []byte{15, 16, 17}
	hdr := block.Header{
		PubKeysBitmap:   originalPubKeyBitMap,
		Signature:       originalSig,
		LeaderSignature: originalLeaderSig,
	}
	sr := *initSubroundEndRound()
	sr.Header = &hdr

	cnsData := consensus.Message{
		PubKeysBitmap:      newPubKeyBitMap,
		LeaderSignature:    newLeaderSig,
		AggregateSignature: newSig,
	}
	haveHdr, newHdr := sr.HaveConsensusHeaderWithFullInfo(&cnsData)
	assert.True(t, haveHdr)
	require.NotNil(t, newHdr)
	assert.Equal(t, newPubKeyBitMap, newHdr.GetPubKeysBitmap())
	assert.Equal(t, newLeaderSig, newHdr.GetLeaderSignature())
	assert.Equal(t, newSig, newHdr.GetSignature())
}

func TestSubroundEndRound_CreateAndBroadcastHeaderFinalInfoBroadcastShouldBeCalled(t *testing.T) {
	t.Parallel()

	chanRcv := make(chan bool, 1)
	leaderSigInHdr := []byte("leader sig")
	container := mock.InitConsensusCore()
	messenger := &mock.BroadcastMessengerMock{
		BroadcastConsensusMessageCalled: func(message *consensus.Message) error {
			chanRcv <- true
			assert.Equal(t, message.LeaderSignature, leaderSigInHdr)
			return nil
		},
	}
	container.SetBroadcastMessenger(messenger)
	sr := *initSubroundEndRoundWithContainer(container)
	sr.Header = &block.Header{LeaderSignature: leaderSigInHdr}

	sr.CreateAndBroadcastHeaderFinalInfo()

	select {
	case <-chanRcv:
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "broadcast not called")
	}
}

func TestSubroundEndRound_ReceivedBlockHeaderFinalInfoShouldWork(t *testing.T) {
	t.Parallel()

	hdr := &block.Header{Nonce: 37}
	sr := *initSubroundEndRound()
	sr.Header = hdr
	sr.AddReceivedHeader(hdr)

	sr.SetStatus(2, spos.SsFinished)
	sr.SetStatus(3, spos.SsNotFinished)

	cnsData := consensus.Message{
		// apply the data which is mocked in consensus state so the checks will pass
		BlockHeaderHash: []byte("X"),
		PubKey:          []byte("A"),
	}
	res := sr.ReceivedBlockHeaderFinalInfo(&cnsData)
	assert.True(t, res)
}

func TestSubroundEndRound_IsOutOfTimeShouldReturnFalse(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()

	res := sr.IsOutOfTime()
	assert.False(t, res)
}

func TestSubroundEndRound_IsOutOfTimeShouldReturnTrue(t *testing.T) {
	t.Parallel()

	// update rounder's mock so it will calculate for real the duration
	container := mock.InitConsensusCore()
	rounder := mock.RounderMock{RemainingTimeCalled: func(startTime time.Time, maxTime time.Duration) time.Duration {
		currentTime := time.Now()
		elapsedTime := currentTime.Sub(startTime)
		remainingTime := maxTime - elapsedTime

		return remainingTime
	}}
	container.SetRounder(&rounder)
	sr := *initSubroundEndRoundWithContainer(container)

	sr.RoundTimeStamp = time.Now().AddDate(0, 0, -1)

	res := sr.IsOutOfTime()
	assert.True(t, res)
}

func TestSubroundEndRound_ExtractMiniBlocksAndTransactionsShouldWork(t *testing.T) {
	t.Parallel()

	sr := *initSubroundEndRound()

	miniBlocks := make(map[uint32][]byte, 0)
	transactions := make(map[string][][]byte, 0)

	miniBlocks[1] = []byte("mbs_shard_1")
	miniBlocks[core.MetachainShardId] = []byte("mbs_shard_meta")
	miniBlocks[2] = []byte("mbs_shard_2")

	transactions["transactions_0_1"] = [][]byte{
		[]byte("tx1_shard_1"),
		[]byte("tx2_shard_1"),
		[]byte("tx3_shard_1"),
	}

	transactions["transactions_0_META"] = [][]byte{
		[]byte("tx1_shard_meta"),
		[]byte("tx2_shard_meta"),
		[]byte("tx3_shard_meta"),
	}

	transactions["transactions_0_2"] = [][]byte{
		[]byte("tx1_shard_2"),
		[]byte("tx2_shard_2"),
		[]byte("tx3_shard_2"),
	}

	metaMiniBlocks, metaTransactions := sr.ExtractMetaMiniBlocksAndTransactions(miniBlocks, transactions)

	require.Equal(t, 2, len(miniBlocks))
	require.Equal(t, 2, len(transactions))
	require.Equal(t, 1, len(metaMiniBlocks))
	require.Equal(t, 1, len(metaTransactions))

	assert.Nil(t, miniBlocks[core.MetachainShardId])
	assert.Nil(t, transactions["transactions_0_META"])
	assert.NotNil(t, metaMiniBlocks[core.MetachainShardId])
	assert.NotNil(t, metaTransactions["transactions_0_META"])
}
