package bls

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/consensus/spos"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/display"
	"github.com/ElrondNetwork/elrond-go/statusHandler"
)

type subroundEndRound struct {
	*spos.Subround
	processingThresholdPercentage int
	getSubroundName               func(subroundId int) string
	displayStatistics             func()

	appStatusHandler core.AppStatusHandler

	mutProcessingEndRound sync.Mutex
}

// SetAppStatusHandler method set appStatusHandler
func (sr *subroundEndRound) SetAppStatusHandler(ash core.AppStatusHandler) error {
	if ash == nil || ash.IsInterfaceNil() {
		return spos.ErrNilAppStatusHandler
	}

	sr.appStatusHandler = ash
	return nil
}

// NewSubroundEndRound creates a subroundEndRound object
func NewSubroundEndRound(
	baseSubround *spos.Subround,
	extend func(subroundId int),
	processingThresholdPercentage int,
	getSubroundName func(subroundId int) string,
	displayStatistics func(),
) (*subroundEndRound, error) {
	err := checkNewSubroundEndRoundParams(
		baseSubround,
	)
	if err != nil {
		return nil, err
	}

	srEndRound := subroundEndRound{
		baseSubround,
		processingThresholdPercentage,
		getSubroundName,
		displayStatistics,
		statusHandler.NewNilStatusHandler(),
		sync.Mutex{},
	}
	srEndRound.Job = srEndRound.doEndRoundJob
	srEndRound.Check = srEndRound.doEndRoundConsensusCheck
	srEndRound.Extend = extend

	return &srEndRound, nil
}

func checkNewSubroundEndRoundParams(
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

func (sr *subroundEndRound) receivedHeader(headerHandler data.HeaderHandler) {
	sr.AddReceivedHeader(headerHandler)

	if !sr.IsSelfLeaderInCurrentRound() {
		sr.doEndRoundJobByParticipant()
	}
}

// doEndRoundJob method does the job of the subround EndRound
func (sr *subroundEndRound) doEndRoundJob() bool {
	if !sr.IsSelfLeaderInCurrentRound() {
		return sr.doEndRoundJobByParticipant()
	}

	return sr.doEndRoundJobByLeader()
}

func (sr *subroundEndRound) doEndRoundJobByLeader() bool {
	bitmap := sr.GenerateBitmap(SrSignature)
	err := sr.checkSignaturesValidity(bitmap)
	if err != nil {
		debugError("checkSignaturesValidity", err)
		return false
	}

	// Aggregate sig and add it to the block
	sig, err := sr.MultiSigner().AggregateSigs(bitmap)
	if err != nil {
		debugError("multisigner.AggregateSigs", err)
		return false
	}

	sr.Header.SetPubKeysBitmap(bitmap)
	sr.Header.SetSignature(sig)

	// Header is complete so the leader can sign it
	leaderSignature, err := sr.signBlockHeader()
	if err != nil {
		log.Error(err.Error())
		return false
	}
	sr.Header.SetLeaderSignature(leaderSignature)

	startTime := time.Now()
	err = sr.BlockProcessor().CommitBlock(sr.Blockchain(), sr.Header, sr.BlockBody)
	elapsedTime := time.Since(startTime)
	log.Debug("elapsed time to commit block",
		"time [s]", elapsedTime,
	)
	if err != nil {
		debugError("CommitBlock", err)
		return false
	}

	sr.SetStatus(sr.Current(), spos.SsFinished)

	// broadcast section

	// broadcast block body and header
	err = sr.BroadcastMessenger().BroadcastBlock(sr.BlockBody, sr.Header)
	if err != nil {
		debugError("BroadcastBlock", err)
	}

	sr.displayStatistics()

	log.Debug("step 3: BlockBody and Header has been committed and broadcast",
		"type", "spos/bls",
		"time [s]", sr.SyncTimer().FormattedCurrentTime())

	err = sr.broadcastMiniBlocksAndTransactions()
	if err != nil {
		debugError("broadcastMiniBlocksAndTransactions", err)
	}

	msg := fmt.Sprintf("Added proposed block with nonce  %d  in blockchain", sr.Header.GetNonce())
	log.Debug(display.Headline(msg, sr.SyncTimer().FormattedCurrentTime(), "+"))

	sr.updateMetricsForLeader()

	return true
}

func (sr *subroundEndRound) doEndRoundJobByParticipant() bool {
	sr.mutProcessingEndRound.Lock()
	defer sr.mutProcessingEndRound.Unlock()

	if sr.RoundCanceled {
		return false
	}
	if !sr.IsConsensusDataSet() {
		return false
	}
	if !sr.IsSubroundFinished(sr.Previous()) {
		return false
	}
	if sr.IsSubroundFinished(sr.Current()) {
		return false
	}
	isConsensusHeaderReceived, header := sr.isConsensusHeaderReceived()
	if !isConsensusHeaderReceived {
		return false
	}

	defer func() {
		sr.SetProcessingBlock(false)
	}()

	sr.SetProcessingBlock(true)

	if sr.isOutOfTime() {
		return false
	}

	startTime := time.Now()
	err := sr.BlockProcessor().CommitBlock(sr.Blockchain(), header, sr.BlockBody)
	elapsedTime := time.Since(startTime)
	log.Debug("elapsed time to commit block",
		"time [s]", elapsedTime,
	)
	if err != nil {
		debugError("CommitBlock", err)
		return false
	}

	sr.SetStatus(sr.Current(), spos.SsFinished)

	sr.displayStatistics()

	log.Debug("step 3: BlockBody and Header has been committed",
		"type", "spos/bls",
		"time [s]", sr.SyncTimer().FormattedCurrentTime())

	msg := fmt.Sprintf("Added received block with nonce  %d  in blockchain", header.GetNonce())
	log.Debug(display.Headline(msg, sr.SyncTimer().FormattedCurrentTime(), "-"))
	return true
}

func (sr *subroundEndRound) isConsensusHeaderReceived() (bool, data.HeaderHandler) {
	if check.IfNil(sr.Header) {
		return false, nil
	}

	consensusHeaderHash, err := core.CalculateHash(sr.Marshalizer(), sr.Hasher(), sr.Header)
	if err != nil {
		log.Debug("isConsensusHeaderReceived: calculate consensus header hash", "error", err.Error())
		return false, nil
	}

	receivedHeaders := sr.GetReceivedHeaders()

	for index := range receivedHeaders {
		receivedHeader := receivedHeaders[index].Clone()
		receivedHeader.SetLeaderSignature(nil)
		receivedHeader.SetPubKeysBitmap(nil)
		receivedHeader.SetSignature(nil)

		receivedHeaderHash, err := core.CalculateHash(sr.Marshalizer(), sr.Hasher(), receivedHeader)
		if err != nil {
			log.Debug("isConsensusHeaderReceived: calculate received header hash", "error", err.Error())
			return false, nil
		}

		if bytes.Equal(receivedHeaderHash, consensusHeaderHash) {
			return true, receivedHeaders[index]
		}
	}

	return false, nil
}

func (sr *subroundEndRound) signBlockHeader() ([]byte, error) {
	headerClone := sr.Header.Clone()
	headerClone.SetLeaderSignature(nil)

	marshalizedHdr, err := sr.Marshalizer().Marshal(headerClone)
	if err != nil {
		return nil, err
	}

	return sr.SingleSigner().Sign(sr.PrivateKey(), marshalizedHdr)
}

func (sr *subroundEndRound) updateMetricsForLeader() {
	sr.appStatusHandler.Increment(core.MetricCountAcceptedBlocks)
	sr.appStatusHandler.SetStringValue(core.MetricConsensusRoundState,
		fmt.Sprintf("valid block produced in %f sec", time.Now().Sub(sr.Rounder().TimeStamp()).Seconds()))
}

func (sr *subroundEndRound) broadcastMiniBlocksAndTransactions() error {
	miniBlocks, transactions, err := sr.BlockProcessor().MarshalizedDataToBroadcast(sr.Header, sr.BlockBody)
	if err != nil {
		return err
	}

	err = sr.BroadcastMessenger().BroadcastMiniBlocks(miniBlocks)
	if err != nil {
		return err
	}

	err = sr.BroadcastMessenger().BroadcastTransactions(transactions)
	if err != nil {
		return err
	}

	return nil
}

// doEndRoundConsensusCheck method checks if the consensus is achieved
func (sr *subroundEndRound) doEndRoundConsensusCheck() bool {
	if sr.RoundCanceled {
		return false
	}

	if sr.IsSubroundFinished(sr.Current()) {
		return true
	}

	return false
}

func (sr *subroundEndRound) checkSignaturesValidity(bitmap []byte) error {
	nbBitsBitmap := len(bitmap) * 8
	consensusGroup := sr.ConsensusGroup()
	consensusGroupSize := len(consensusGroup)
	size := consensusGroupSize

	if consensusGroupSize > nbBitsBitmap {
		size = nbBitsBitmap
	}

	for i := 0; i < size; i++ {
		indexRequired := (bitmap[i/8] & (1 << uint16(i%8))) > 0
		if !indexRequired {
			continue
		}

		pubKey := consensusGroup[i]
		isSigJobDone, err := sr.JobDone(pubKey, SrSignature)
		if err != nil {
			return err
		}

		if !isSigJobDone {
			return spos.ErrNilSignature
		}

		signature, err := sr.MultiSigner().SignatureShare(uint16(i))
		if err != nil {
			return err
		}

		err = sr.MultiSigner().VerifySignatureShare(uint16(i), signature, sr.GetData(), bitmap)
		if err != nil {
			return err
		}
	}

	return nil
}

func (sr *subroundEndRound) isOutOfTime() bool {
	startTime := time.Time{}
	startTime = sr.RoundTimeStamp
	maxTime := sr.Rounder().TimeDuration() * time.Duration(sr.processingThresholdPercentage) / 100
	if sr.Rounder().RemainingTime(startTime, maxTime) < 0 {
		log.Debug("canceled round, time is out",
			"time [s]", sr.SyncTimer().FormattedCurrentTime(),
			"round", sr.SyncTimer().FormattedCurrentTime(), sr.Rounder().Index(),
			"subround", sr.getSubroundName(sr.Current()))

		sr.RoundCanceled = true

		return true
	}

	return false
}
