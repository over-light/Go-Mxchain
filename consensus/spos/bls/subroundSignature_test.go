package bls_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/mock"
	"github.com/ElrondNetwork/elrond-go/consensus/spos"
	"github.com/ElrondNetwork/elrond-go/consensus/spos/bls"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func initSubroundSignatureWithContainer(container *mock.ConsensusCoreMock) bls.SubroundSignature {
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)

	srSignature, _ := bls.NewSubroundSignature(
		sr,
		extend,
	)

	return srSignature
}

func initSubroundSignature() bls.SubroundSignature {
	container := mock.InitConsensusCore()
	return initSubroundSignatureWithContainer(container)
}

func TestSubroundSignature_NewSubroundSignatureNilSubroundShouldFail(t *testing.T) {
	t.Parallel()

	srSignature, err := bls.NewSubroundSignature(
		nil,
		extend,
	)

	assert.Nil(t, srSignature)
	assert.Equal(t, spos.ErrNilSubround, err)
}

func TestSubroundSignature_NewSubroundSignatureNilConsensusStateShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)

	sr.ConsensusState = nil
	srSignature, err := bls.NewSubroundSignature(
		sr,
		extend,
	)

	assert.Nil(t, srSignature)
	assert.Equal(t, spos.ErrNilConsensusState, err)
}

func TestSubroundSignature_NewSubroundSignatureNilHasherShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)
	container.SetHasher(nil)
	srSignature, err := bls.NewSubroundSignature(
		sr,
		extend,
	)

	assert.Nil(t, srSignature)
	assert.Equal(t, spos.ErrNilHasher, err)
}

func TestSubroundSignature_NewSubroundSignatureNilMultisignerShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)
	container.SetMultiSigner(nil)
	srSignature, err := bls.NewSubroundSignature(
		sr,
		extend,
	)

	assert.Nil(t, srSignature)
	assert.Equal(t, spos.ErrNilMultiSigner, err)
}

func TestSubroundSignature_NewSubroundSignatureNilRounderShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)
	container.SetRounder(nil)

	srSignature, err := bls.NewSubroundSignature(
		sr,
		extend,
	)

	assert.Nil(t, srSignature)
	assert.Equal(t, spos.ErrNilRounder, err)
}

func TestSubroundSignature_NewSubroundSignatureNilSyncTimerShouldFail(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)
	container.SetSyncTimer(nil)
	srSignature, err := bls.NewSubroundSignature(
		sr,
		extend,
	)

	assert.Nil(t, srSignature)
	assert.Equal(t, spos.ErrNilSyncTimer, err)
}

func TestSubroundSignature_NewSubroundSignatureShouldWork(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	consensusState := initConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
	)

	srSignature, err := bls.NewSubroundSignature(
		sr,
		extend,
	)

	assert.NotNil(t, srSignature)
	assert.Nil(t, err)
}

func TestSubroundSignature_DoSignatureJob(t *testing.T) {
	t.Parallel()

	container := mock.InitConsensusCore()
	sr := *initSubroundSignatureWithContainer(container)

	sr.Data = nil
	r := sr.DoSignatureJob()
	assert.False(t, r)

	sr.Data = []byte("X")

	multiSignerMock := mock.InitMultiSignerMock()

	err := errors.New("create signature share error")
	multiSignerMock.CreateSignatureShareMock = func(msg []byte, bitmap []byte) ([]byte, error) {
		return nil, err
	}

	container.SetMultiSigner(multiSignerMock)

	r = sr.DoSignatureJob()
	assert.False(t, r)

	multiSignerMock = mock.InitMultiSignerMock()

	multiSignerMock.CreateSignatureShareMock = func(msg []byte, bitmap []byte) ([]byte, error) {
		return []byte("SIG"), nil
	}
	container.SetMultiSigner(multiSignerMock)

	r = sr.DoSignatureJob()
	assert.True(t, r)

	_ = sr.SetJobDone(sr.SelfPubKey(), bls.SrSignature, false)
	sr.RoundCanceled = false
	sr.SetSelfPubKey(sr.ConsensusGroup()[0])
	r = sr.DoSignatureJob()
	assert.True(t, r)
	assert.False(t, sr.RoundCanceled)
}

func TestSubroundSignature_ReceivedSignature(t *testing.T) {
	t.Parallel()

	sr := *initSubroundSignature()
	signature := []byte("signature")
	cnsMsg := consensus.NewConsensusMessage(
		sr.Data,
		signature,
		nil,
		nil,
		[]byte(sr.ConsensusGroup()[1]),
		[]byte("sig"),
		int(bls.MtSignature),
		0,
		chainID,
		nil,
		nil,
		nil,
	)

	sr.Data = nil
	r := sr.ReceivedSignature(cnsMsg)
	assert.False(t, r)

	sr.Data = []byte("Y")
	r = sr.ReceivedSignature(cnsMsg)
	assert.False(t, r)

	sr.Data = []byte("X")
	r = sr.ReceivedSignature(cnsMsg)
	assert.False(t, r)

	sr.SetSelfPubKey(sr.ConsensusGroup()[0])

	cnsMsg.PubKey = []byte("X")
	r = sr.ReceivedSignature(cnsMsg)
	assert.False(t, r)

	cnsMsg.PubKey = []byte(sr.ConsensusGroup()[1])
	maxCount := len(sr.ConsensusGroup()) * 2 / 3
	count := 0
	for i := 0; i < len(sr.ConsensusGroup()); i++ {
		if sr.ConsensusGroup()[i] != string(cnsMsg.PubKey) {
			_ = sr.SetJobDone(sr.ConsensusGroup()[i], bls.SrSignature, true)
			count++
			if count == maxCount {
				break
			}
		}
	}
	r = sr.ReceivedSignature(cnsMsg)
	assert.True(t, r)
}

func TestSubroundSignature_SignaturesCollected(t *testing.T) {
	t.Parallel()

	sr := *initSubroundSignature()

	for i := 0; i < len(sr.ConsensusGroup()); i++ {
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bls.SrBlock, false)
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bls.SrSignature, false)
	}

	ok, n := sr.SignaturesCollected(2)
	assert.False(t, ok)
	assert.Equal(t, 0, n)

	ok, _ = sr.SignaturesCollected(2)
	assert.False(t, ok)

	_ = sr.SetJobDone("B", bls.SrSignature, true)
	isJobDone, _ := sr.JobDone("B", bls.SrSignature)
	assert.True(t, isJobDone)

	ok, _ = sr.SignaturesCollected(2)
	assert.False(t, ok)

	_ = sr.SetJobDone("C", bls.SrSignature, true)
	ok, _ = sr.SignaturesCollected(2)
	assert.True(t, ok)
}

func TestSubroundSignature_DoSignatureConsensusCheckShouldReturnFalseWhenRoundIsCanceled(t *testing.T) {
	t.Parallel()

	sr := *initSubroundSignature()
	sr.RoundCanceled = true
	assert.False(t, sr.DoSignatureConsensusCheck())
}

func TestSubroundSignature_DoSignatureConsensusCheckShouldReturnTrueWhenSubroundIsFinished(t *testing.T) {
	t.Parallel()

	sr := *initSubroundSignature()
	sr.SetStatus(bls.SrSignature, spos.SsFinished)
	assert.True(t, sr.DoSignatureConsensusCheck())
}

func TestSubroundSignature_DoSignatureConsensusCheckShouldReturnTrueWhenSignaturesCollectedReturnTrue(t *testing.T) {
	t.Parallel()

	sr := *initSubroundSignature()

	for i := 0; i < sr.Threshold(bls.SrSignature); i++ {
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bls.SrSignature, true)
	}

	assert.True(t, sr.DoSignatureConsensusCheck())
}

func TestSubroundSignature_DoSignatureConsensusCheckShouldReturnFalseWhenSignaturesCollectedReturnFalse(t *testing.T) {
	t.Parallel()

	sr := *initSubroundSignature()
	assert.False(t, sr.DoSignatureConsensusCheck())
}

func TestSubroundSignature_ReceivedSignatureReturnFalseWhenConsensusDataIsNotEqual(t *testing.T) {
	t.Parallel()

	sr := *initSubroundSignature()

	cnsMsg := consensus.NewConsensusMessage(
		append(sr.Data, []byte("X")...),
		[]byte("signature"),
		nil,
		nil,
		[]byte(sr.ConsensusGroup()[0]),
		[]byte("sig"),
		int(bls.MtSignature),
		0,
		chainID,
		nil,
		nil,
		nil,
	)

	assert.False(t, sr.ReceivedSignature(cnsMsg))
}
