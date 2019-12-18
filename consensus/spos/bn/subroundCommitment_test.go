package bn_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/mock"
	"github.com/ElrondNetwork/elrond-go/consensus/spos"
	"github.com/ElrondNetwork/elrond-go/consensus/spos/bn"
	"github.com/stretchr/testify/assert"
)

func initSubroundCommitment() bn.SubroundCommitment {
	container := mock.InitConsensusCore()

	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bn.SrBitmap,
		bn.SrCommitment,
		bn.SrSignature,
		int64(55*roundTimeDuration/100),
		int64(70*roundTimeDuration/100),
		"(COMMITMENT)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)

	srCommitment, _ := bn.NewSubroundCommitment(
		sr,
		extend,
	)

	return srCommitment
}

func TestSubroundCommitment_NewSubroundCommitmentNilSubroundShouldFail(t *testing.T) {
	t.Parallel()

	srCommitment, err := bn.NewSubroundCommitment(
		nil,
		extend,
	)

	assert.Nil(t, srCommitment)
	assert.Equal(t, spos.ErrNilSubround, err)
}

func TestSubroundCommitment_NewSubroundCommitmentNilConsensusStateShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bn.SrBitmap,
		bn.SrCommitment,
		bn.SrSignature,
		int64(55*roundTimeDuration/100),
		int64(70*roundTimeDuration/100),
		"(COMMITMENT)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)

	sr.ConsensusState = nil
	srCommitment, err := bn.NewSubroundCommitment(
		sr,
		extend,
	)

	assert.Nil(t, srCommitment)
	assert.Equal(t, spos.ErrNilConsensusState, err)
}

func TestSubroundCommitment_NewSubroundCommitmentNilMultisignerShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()

	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bn.SrBitmap,
		bn.SrCommitment,
		bn.SrSignature,
		int64(55*roundTimeDuration/100),
		int64(70*roundTimeDuration/100),
		"(COMMITMENT)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)
	container.SetMultiSigner(nil)
	srCommitment, err := bn.NewSubroundCommitment(
		sr,
		extend,
	)

	assert.Nil(t, srCommitment)
	assert.Equal(t, spos.ErrNilMultiSigner, err)
}

func TestSubroundCommitment_NewSubroundCommitmentNilRounderShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()

	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bn.SrBitmap,
		bn.SrCommitment,
		bn.SrSignature,
		int64(55*roundTimeDuration/100),
		int64(70*roundTimeDuration/100),
		"(COMMITMENT)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)
	container.SetRounder(nil)
	srCommitment, err := bn.NewSubroundCommitment(
		sr,
		extend,
	)

	assert.Nil(t, srCommitment)
	assert.Equal(t, spos.ErrNilRounder, err)
}

func TestSubroundCommitment_NewSubroundCommitmentNilSyncTimerShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()

	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bn.SrBitmap,
		bn.SrCommitment,
		bn.SrSignature,
		int64(55*roundTimeDuration/100),
		int64(70*roundTimeDuration/100),
		"(COMMITMENT)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)
	container.SetSyncTimer(nil)
	srCommitment, err := bn.NewSubroundCommitment(
		sr,
		extend,
	)

	assert.Nil(t, srCommitment)
	assert.Equal(t, spos.ErrNilSyncTimer, err)
}

func TestSubroundCommitment_NewSubroundCommitmentShouldWork(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()

	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bn.SrBitmap,
		bn.SrCommitment,
		bn.SrSignature,
		int64(55*roundTimeDuration/100),
		int64(70*roundTimeDuration/100),
		"(COMMITMENT)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)

	srCommitment, err := bn.NewSubroundCommitment(
		sr,
		extend,
	)

	assert.NotNil(t, srCommitment)
	assert.Nil(t, err)
}

func TestSubroundCommitment_DoCommitmentJob(t *testing.T) {
	t.Parallel()

	sr := *initSubroundCommitment()

	r := sr.DoCommitmentJob()
	assert.False(t, r)

	sr.SetStatus(bn.SrBitmap, spos.SsFinished)
	sr.SetStatus(bn.SrCommitment, spos.SsFinished)

	r = sr.DoCommitmentJob()
	assert.False(t, r)

	sr.SetStatus(bn.SrCommitment, spos.SsNotFinished)
	_ = sr.SetJobDone(sr.SelfPubKey(), bn.SrCommitment, true)

	r = sr.DoCommitmentJob()
	assert.False(t, r)

	_ = sr.SetJobDone(sr.SelfPubKey(), bn.SrCommitment, false)

	r = sr.DoCommitmentJob()
	assert.False(t, r)

	_ = sr.SetJobDone(sr.SelfPubKey(), bn.SrBitmap, true)
	sr.Data = nil

	r = sr.DoCommitmentJob()
	assert.False(t, r)

	dta := []byte("X")
	sr.Data = dta

	r = sr.DoCommitmentJob()
	assert.True(t, r)
}

func TestSubroundCommitment_ReceivedCommitment(t *testing.T) {
	t.Parallel()

	sr := *initSubroundCommitment()

	commitment := []byte("commitment")
	cnsMsg := consensus.NewConsensusMessage(
		sr.Data,
		commitment,
		[]byte(sr.ConsensusGroup()[0]),
		[]byte("sig"),
		int(bn.MtCommitment),
		uint64(sr.Rounder().TimeStamp().Unix()),
		0,
		chainID,
	)

	sr.Data = nil
	r := sr.ReceivedCommitment(cnsMsg)
	assert.False(t, r)

	sr.Data = []byte("X")
	sr.SetStatus(bn.SrCommitment, spos.SsFinished)

	r = sr.ReceivedCommitment(cnsMsg)
	assert.False(t, r)

	_ = sr.SetJobDone(sr.ConsensusGroup()[0], bn.SrBitmap, true)
	sr.SetStatus(bn.SrCommitment, spos.SsFinished)

	r = sr.ReceivedCommitment(cnsMsg)
	assert.False(t, r)

	sr.SetStatus(bn.SrCommitment, spos.SsNotFinished)

	r = sr.ReceivedCommitment(cnsMsg)
	assert.True(t, r)
	isCommJobDone, _ := sr.JobDone(sr.ConsensusGroup()[0], bn.SrCommitment)
	assert.True(t, isCommJobDone)
}

func TestSubroundCommitment_DoCommitmentConsensusCheckShouldReturnFalseWhenRoundIsCanceled(t *testing.T) {
	t.Parallel()

	sr := *initSubroundCommitment()
	sr.RoundCanceled = true
	assert.False(t, sr.DoCommitmentConsensusCheck())
}

func TestSubroundCommitment_DoCommitmentConsensusCheckShouldReturnTrueWhenSubroundIsFinished(t *testing.T) {
	t.Parallel()

	sr := *initSubroundCommitment()
	sr.SetStatus(bn.SrCommitment, spos.SsFinished)
	assert.True(t, sr.DoCommitmentConsensusCheck())
}

func TestSubroundCommitment_DoCommitmentConsensusCheckShouldReturnTrueWhenCommitmentsCollectedReturnTrue(t *testing.T) {
	t.Parallel()

	sr := *initSubroundCommitment()

	for i := 0; i < sr.Threshold(bn.SrBitmap); i++ {
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bn.SrBitmap, true)
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bn.SrCommitment, true)
	}

	assert.True(t, sr.DoCommitmentConsensusCheck())
}

func TestSubroundCommitment_DoCommitmentConsensusCheckShouldReturnFalseWhenCommitmentsCollectedReturnFalse(t *testing.T) {
	t.Parallel()

	sr := *initSubroundCommitment()
	assert.False(t, sr.DoCommitmentConsensusCheck())
}

func TestSubroundCommitment_CommitmentsCollected(t *testing.T) {
	t.Parallel()

	sr := *initSubroundCommitment()

	for i := 0; i < len(sr.ConsensusGroup()); i++ {
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bn.SrBlock, false)
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bn.SrCommitmentHash, false)
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bn.SrBitmap, false)
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bn.SrCommitment, false)
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bn.SrSignature, false)
	}

	ok := sr.CommitmentsCollected(2)
	assert.False(t, ok)

	_ = sr.SetJobDone("A", bn.SrBitmap, true)
	_ = sr.SetJobDone("C", bn.SrBitmap, true)
	isJobDone, _ := sr.JobDone("C", bn.SrBitmap)
	assert.True(t, isJobDone)

	ok = sr.CommitmentsCollected(2)
	assert.False(t, ok)

	_ = sr.SetJobDone("B", bn.SrCommitment, true)
	isJobDone, _ = sr.JobDone("B", bn.SrCommitment)
	assert.True(t, isJobDone)

	ok = sr.CommitmentsCollected(2)
	assert.False(t, ok)

	_ = sr.SetJobDone("C", bn.SrCommitment, true)
	ok = sr.CommitmentsCollected(2)
	assert.False(t, ok)

	_ = sr.SetJobDone("A", bn.SrCommitment, true)
	ok = sr.CommitmentsCollected(2)
	assert.True(t, ok)
}

func TestSubroundCommitment_ReceivedCommitmentReturnFalseWhenConsensusDataIsNotEqual(t *testing.T) {
	t.Parallel()

	sr := *initSubroundCommitment()

	cnsMsg := consensus.NewConsensusMessage(
		append(sr.Data, []byte("X")...),
		[]byte("commitment"),
		[]byte(sr.ConsensusGroup()[0]),
		[]byte("sig"),
		int(bn.MtCommitment),
		uint64(sr.Rounder().TimeStamp().Unix()),
		0,
		chainID,
	)

	_ = sr.SetJobDone(sr.ConsensusGroup()[0], bn.SrBitmap, true)

	assert.False(t, sr.ReceivedCommitment(cnsMsg))
}
