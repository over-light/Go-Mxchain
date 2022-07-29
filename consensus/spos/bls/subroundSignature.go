package bls

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/spos"
)

type subroundSignature struct {
	*spos.Subround

	appStatusHandler core.AppStatusHandler
}

// NewSubroundSignature creates a subroundSignature object
func NewSubroundSignature(
	baseSubround *spos.Subround,
	extend func(subroundId int),
	appStatusHandler core.AppStatusHandler,
) (*subroundSignature, error) {
	err := checkNewSubroundSignatureParams(
		baseSubround,
	)
	if err != nil {
		return nil, err
	}
	if check.IfNil(appStatusHandler) {
		return nil, spos.ErrNilAppStatusHandler
	}

	srSignature := subroundSignature{
		Subround:         baseSubround,
		appStatusHandler: appStatusHandler,
	}
	srSignature.Job = srSignature.doSignatureJob
	srSignature.Check = srSignature.doSignatureConsensusCheck
	srSignature.Extend = extend

	return &srSignature, nil
}

func checkNewSubroundSignatureParams(
	baseSubround *spos.Subround,
) error {
	if baseSubround == nil {
		return spos.ErrNilSubround
	}
	if baseSubround.ConsensusState == nil {
		return spos.ErrNilConsensusState
	}

	err := spos.ValidateConsensusCore(baseSubround.ConsensusCoreHandler)

	return err
}

// doSignatureJob method does the job of the subround Signature
func (sr *subroundSignature) doSignatureJob(_ context.Context) bool {
	if !sr.IsNodeInConsensusGroup(sr.SelfPubKey()) {
		return true
	}
	if !sr.CanDoSubroundJob(sr.Current()) {
		return false
	}
	if check.IfNil(sr.Header) {
		log.Error("doSignatureJob", "error", spos.ErrNilHeader)
		return false
	}

	multiSigner, err := sr.MultiSignerContainer().GetMultiSigner(sr.Header.GetEpoch())
	if err != nil {
		log.Error("doSignatureJob.GetMultiSigner", "error", err.Error())
		return false
	}

	// TODO: keep private key as byte array in consensus
	privateKey, _ := sr.PrivateKey().ToByteArray()
	signatureShare, err := multiSigner.CreateSignatureShare(privateKey, sr.GetData())
	if err != nil {
		log.Debug("doSignatureJob.CreateSignatureShare", "error", err.Error())
		return false
	}

	isSelfLeader := sr.IsSelfLeaderInCurrentRound()

	if !isSelfLeader {
		// TODO: Analyze it is possible to send message only to leader with O(1) instead of O(n)
		cnsMsg := consensus.NewConsensusMessage(
			sr.GetData(),
			signatureShare,
			nil,
			nil,
			[]byte(sr.SelfPubKey()),
			nil,
			int(MtSignature),
			sr.RoundHandler().Index(),
			sr.ChainID(),
			nil,
			nil,
			nil,
			sr.CurrentPid(),
		)

		err = sr.BroadcastMessenger().BroadcastConsensusMessage(cnsMsg)
		if err != nil {
			log.Debug("doSignatureJob.BroadcastConsensusMessage", "error", err.Error())
			return false
		}

		log.Debug("step 2: signature has been sent")
	}

	err = sr.SetSelfJobDone(sr.Current(), true)
	if err != nil {
		log.Debug("doSignatureJob.SetSelfJobDone",
			"subround", sr.Name(),
			"error", err.Error())
		return false
	}

	if isSelfLeader {
		go sr.waitAllSignatures()
	}

	return true
}

// receivedSignature method is called when a signature is received through the signature channel.
// If the signature is valid, than the jobDone map corresponding to the node which sent it,
// is set on true for the subround Signature
func (sr *subroundSignature) receivedSignature(_ context.Context, cnsDta *consensus.Message) bool {
	node := string(cnsDta.PubKey)
	pkForLogs := core.GetTrimmedPk(hex.EncodeToString(cnsDta.PubKey))

	if !sr.IsConsensusDataSet() {
		return false
	}

	if !sr.IsNodeInConsensusGroup(node) {
		sr.PeerHonestyHandler().ChangeScore(
			node,
			spos.GetConsensusTopicID(sr.ShardCoordinator()),
			spos.ValidatorPeerHonestyDecreaseFactor,
		)

		return false
	}

	if !sr.IsSelfLeaderInCurrentRound() {
		return false
	}

	if !sr.IsConsensusDataEqual(cnsDta.BlockHeaderHash) {
		return false
	}

	if !sr.CanProcessReceivedMessage(cnsDta, sr.RoundHandler().Index(), sr.Current()) {
		return false
	}

	index, err := sr.ConsensusGroupIndex(node)
	if err != nil {
		log.Debug("receivedSignature.ConsensusGroupIndex",
			"node", pkForLogs,
			"error", err.Error())
		return false
	}

	if check.IfNil(sr.Header) {
		log.Error("receivedSignature", "error", spos.ErrNilHeader)
		return false
	}

	currentMultiSigner, err := sr.MultiSignerContainer().GetMultiSigner(sr.Header.GetEpoch())
	if err != nil {
		log.Error("receivedSignature.GetMultiSigner", "error", err.Error())
		return false
	}

	err = currentMultiSigner.VerifySignatureShare([]byte(node), sr.GetData(), cnsDta.SignatureShare)
	if err != nil {
		log.Debug("receivedSignature.VerifySignatureShare",
			"node", pkForLogs,
			"index", index,
			"error", err.Error())
		return false
	}

	// TODO: store here the signature in consensus state
	//err = currentMultiSigner.StoreSignatureShare(uint16(index), cnsDta.SignatureShare)
	//if err != nil {
	//	log.Debug("receivedSignature.StoreSignatureShare",
	//		"node", pkForLogs,
	//		"index", index,
	//		"error", err.Error())
	//	return false
	//}

	err = sr.SetJobDone(node, sr.Current(), true)
	if err != nil {
		log.Debug("receivedSignature.SetJobDone",
			"node", pkForLogs,
			"subround", sr.Name(),
			"error", err.Error())
		return false
	}

	sr.PeerHonestyHandler().ChangeScore(
		node,
		spos.GetConsensusTopicID(sr.ShardCoordinator()),
		spos.ValidatorPeerHonestyIncreaseFactor,
	)

	sr.appStatusHandler.SetStringValue(common.MetricConsensusRoundState, "signed")
	return true
}

// doSignatureConsensusCheck method checks if the consensus in the subround Signature is achieved
func (sr *subroundSignature) doSignatureConsensusCheck() bool {
	if sr.RoundCanceled {
		return false
	}

	if sr.IsSubroundFinished(sr.Current()) {
		sr.appStatusHandler.SetStringValue(common.MetricConsensusRoundState, "signed")

		return true
	}

	isSelfLeader := sr.IsSelfLeaderInCurrentRound()
	isSelfInConsensusGroup := sr.IsNodeInConsensusGroup(sr.SelfPubKey())

	threshold := sr.Threshold(sr.Current())
	if sr.FallbackHeaderValidator().ShouldApplyFallbackValidation(sr.Header) {
		threshold = sr.FallbackThreshold(sr.Current())
		log.Warn("subroundSignature.doSignatureConsensusCheck: fallback validation has been applied",
			"minimum number of signatures required", threshold,
			"actual number of signatures received", sr.getNumOfSignaturesCollected(),
		)
	}

	areSignaturesCollected, numSigs := sr.areSignaturesCollected(threshold)
	areAllSignaturesCollected := numSigs == sr.ConsensusGroupSize()

	isJobDoneByLeader := isSelfLeader && (areAllSignaturesCollected || (areSignaturesCollected && sr.WaitingAllSignaturesTimeOut))
	isJobDoneByConsensusNode := !isSelfLeader && isSelfInConsensusGroup && sr.IsSelfJobDone(sr.Current())

	isSubroundFinished := !isSelfInConsensusGroup || isJobDoneByConsensusNode || isJobDoneByLeader

	if isSubroundFinished {
		if isSelfLeader {
			log.Debug("step 2: signatures",
				"received", numSigs,
				"total", len(sr.ConsensusGroup()))
		}

		log.Debug("step 2: subround has been finished",
			"subround", sr.Name())
		sr.SetStatus(sr.Current(), spos.SsFinished)

		sr.appStatusHandler.SetStringValue(common.MetricConsensusRoundState, "signed")

		return true
	}

	return false
}

// areSignaturesCollected method checks if the signatures received from the nodes, belonging to the current
// jobDone group, are more than the necessary given threshold
func (sr *subroundSignature) areSignaturesCollected(threshold int) (bool, int) {
	n := sr.getNumOfSignaturesCollected()
	return n >= threshold, n
}

func (sr *subroundSignature) getNumOfSignaturesCollected() int {
	n := 0

	for i := 0; i < len(sr.ConsensusGroup()); i++ {
		node := sr.ConsensusGroup()[i]

		isSignJobDone, err := sr.JobDone(node, sr.Current())
		if err != nil {
			log.Debug("getNumOfSignaturesCollected.JobDone",
				"node", node,
				"subround", sr.Name(),
				"error", err.Error())
			continue
		}

		if isSignJobDone {
			n++
		}
	}

	return n
}

func (sr *subroundSignature) waitAllSignatures() {
	remainingTime := sr.remainingTime()
	time.Sleep(remainingTime)

	if sr.IsSubroundFinished(sr.Current()) {
		return
	}

	sr.WaitingAllSignaturesTimeOut = true

	select {
	case sr.ConsensusChannel() <- true:
	default:
	}
}

func (sr *subroundSignature) remainingTime() time.Duration {
	startTime := sr.RoundHandler().TimeStamp()
	maxTime := time.Duration(float64(sr.StartTime()) + float64(sr.EndTime()-sr.StartTime())*waitingAllSigsMaxTimeThreshold)
	remainigTime := sr.RoundHandler().RemainingTime(startTime, maxTime)

	return remainigTime
}
