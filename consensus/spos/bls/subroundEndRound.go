package bls

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/consensus"
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

// receivedBlockHeaderFinalInfo method is called when a block header final info is received
func (sr *subroundEndRound) receivedBlockHeaderFinalInfo(cnsDta *consensus.Message) bool {
	node := string(cnsDta.PubKey)

	if !sr.IsConsensusDataSet() {
		return false
	}

	if !sr.IsNodeLeaderInCurrentRound(node) { // is NOT this node leader in current round?
		sr.PeerHonestyHandler().Decrease(
			node,
			spos.GetConsensusTopicID(sr.ShardCoordinator()),
			spos.LeaderPeerHonestyDecreaseFactor)

		return false
	}

	if sr.IsSelfLeaderInCurrentRound() {
		return false
	}

	if !sr.IsConsensusDataEqual(cnsDta.BlockHeaderHash) {
		return false
	}

	if !sr.CanProcessReceivedMessage(cnsDta, sr.Rounder().Index(), sr.Current()) {
		return false
	}

	log.Debug("step 3: block header final info has been received",
		"PubKeysBitmap", cnsDta.PubKeysBitmap,
		"AggregateSignature", cnsDta.AggregateSignature,
		"LeaderSignature", cnsDta.LeaderSignature)

	sr.PeerHonestyHandler().Increase(
		node,
		spos.GetConsensusTopicID(sr.ShardCoordinator()),
		spos.LeaderPeerHonestyIncreaseFactor)

	return sr.doEndRoundJobByParticipant(cnsDta)
}

func (sr *subroundEndRound) receivedHeader(headerHandler data.HeaderHandler) {
	if sr.ConsensusGroup() == nil || sr.IsSelfLeaderInCurrentRound() {
		return
	}

	sr.AddReceivedHeader(headerHandler)

	sr.doEndRoundJobByParticipant(nil)
}

// doEndRoundJob method does the job of the subround EndRound
func (sr *subroundEndRound) doEndRoundJob() bool {
	if !sr.IsSelfLeaderInCurrentRound() {
		return sr.doEndRoundJobByParticipant(nil)
	}

	return sr.doEndRoundJobByLeader()
}

func (sr *subroundEndRound) doEndRoundJobByLeader() bool {
	bitmap := sr.GenerateBitmap(SrSignature)
	err := sr.checkSignaturesValidity(bitmap)
	if err != nil {
		log.Debug("doEndRoundJob.checkSignaturesValidity", "error", err.Error())
		return false
	}

	// Aggregate sig and add it to the block
	sig, err := sr.MultiSigner().AggregateSigs(bitmap)
	if err != nil {
		log.Debug("doEndRoundJob.AggregateSigs", "error", err.Error())
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

	// broadcast section

	// create and broadcast header final info
	sr.createAndBroadcastHeaderFinalInfo()

	// broadcast header
	err = sr.BroadcastMessenger().BroadcastHeader(sr.Header)
	if err != nil {
		log.Debug("doEndRoundJob.BroadcastHeader", "error", err.Error())
	}

	startTime := time.Now()
	err = sr.BlockProcessor().CommitBlock(sr.Header, sr.Body)
	elapsedTime := time.Since(startTime)
	if elapsedTime >= core.CommitMaxTime {
		log.Warn("doEndRoundJobByLeader.CommitBlock", "elapsed time", elapsedTime)
	} else {
		log.Debug("elapsed time to commit block",
			"time [s]", elapsedTime,
		)
	}
	if err != nil {
		log.Debug("doEndRoundJob.CommitBlock", "error", err)
		return false
	}

	sr.SetStatus(sr.Current(), spos.SsFinished)

	sr.displayStatistics()

	log.Debug("step 3: Body and Header have been committed and header has been broadcast")

	err = sr.broadcastMiniBlocksAndTransactions()
	if err != nil {
		log.Debug("doEndRoundJob.broadcastMiniBlocksAndTransactions", "error", err.Error())
	}

	msg := fmt.Sprintf("Added proposed block with nonce  %d  in blockchain", sr.Header.GetNonce())
	log.Debug(display.Headline(msg, sr.SyncTimer().FormattedCurrentTime(), "+"))

	sr.updateMetricsForLeader()

	return true
}

func (sr *subroundEndRound) createAndBroadcastHeaderFinalInfo() {
	cnsMsg := consensus.NewConsensusMessage(
		sr.GetData(),
		nil,
		nil,
		nil,
		[]byte(sr.SelfPubKey()),
		nil,
		int(MtBlockHeaderFinalInfo),
		sr.Rounder().Index(),
		sr.ChainID(),
		sr.Header.GetPubKeysBitmap(),
		sr.Header.GetSignature(),
		sr.Header.GetLeaderSignature(),
	)

	err := sr.BroadcastMessenger().BroadcastConsensusMessage(cnsMsg)
	if err != nil {
		log.Debug("doEndRoundJob.BroadcastConsensusMessage", "error", err.Error())
		return
	}

	log.Debug("step 3: block header final info has been sent",
		"PubKeysBitmap", sr.Header.GetPubKeysBitmap(),
		"AggregateSignature", sr.Header.GetSignature(),
		"LeaderSignature", sr.Header.GetLeaderSignature())
}

func (sr *subroundEndRound) doEndRoundJobByParticipant(cnsDta *consensus.Message) bool {
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

	haveHeader, header := sr.haveConsensusHeaderWithFullInfo(cnsDta)
	if !haveHeader {
		return false
	}

	defer func() {
		sr.SetProcessingBlock(false)
	}()

	sr.SetProcessingBlock(true)

	shouldNotCommitBlock := sr.ExtendedCalled || int64(header.GetRound()) < sr.Rounder().Index()
	if shouldNotCommitBlock {
		log.Debug("canceled round, extended has been called or round index has been changed",
			"round", sr.Rounder().Index(),
			"subround", sr.Name(),
			"header round", header.GetRound(),
			"extended called", sr.ExtendedCalled,
		)
		return false
	}

	if sr.isOutOfTime() {
		return false
	}

	startTime := time.Now()
	err := sr.BlockProcessor().CommitBlock(header, sr.Body)
	elapsedTime := time.Since(startTime)
	if elapsedTime >= core.CommitMaxTime {
		log.Warn("doEndRoundJobByParticipant.CommitBlock", "elapsed time", elapsedTime)
	} else {
		log.Debug("elapsed time to commit block",
			"time [s]", elapsedTime,
		)
	}
	if err != nil {
		log.Debug("doEndRoundJobByParticipant.CommitBlock", "error", err.Error())
		return false
	}

	sr.SetStatus(sr.Current(), spos.SsFinished)

	sr.displayStatistics()

	log.Debug("step 3: Body and Header have been committed")

	headerTypeMsg := "received"
	if cnsDta != nil {
		headerTypeMsg = "assembled"
	}

	msg := fmt.Sprintf("Added %s block with nonce  %d  in blockchain", headerTypeMsg, header.GetNonce())
	log.Debug(display.Headline(msg, sr.SyncTimer().FormattedCurrentTime(), "-"))
	return true
}

func (sr *subroundEndRound) haveConsensusHeaderWithFullInfo(cnsDta *consensus.Message) (bool, data.HeaderHandler) {
	if cnsDta == nil {
		return sr.isConsensusHeaderReceived()
	}

	if check.IfNil(sr.Header) {
		return false, nil
	}

	header := sr.Header.Clone()
	header.SetPubKeysBitmap(cnsDta.PubKeysBitmap)
	header.SetSignature(cnsDta.AggregateSignature)
	header.SetLeaderSignature(cnsDta.LeaderSignature)

	return true, header
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

	var receivedHeaderHash []byte
	for index := range receivedHeaders {
		receivedHeader := receivedHeaders[index].Clone()
		receivedHeader.SetLeaderSignature(nil)
		receivedHeader.SetPubKeysBitmap(nil)
		receivedHeader.SetSignature(nil)

		receivedHeaderHash, err = core.CalculateHash(sr.Marshalizer(), sr.Hasher(), receivedHeader)
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
		fmt.Sprintf("valid block produced in %f sec", time.Since(sr.Rounder().TimeStamp()).Seconds()))
}

func (sr *subroundEndRound) broadcastMiniBlocksAndTransactions() error {
	miniBlocks, transactions, err := sr.BlockProcessor().MarshalizedDataToBroadcast(sr.Header, sr.Body)
	if err != nil {
		return err
	}

	if sr.ShardCoordinator().SelfId() != core.MetachainShardId {
		var headerHash []byte
		headerHash, err = core.CalculateHash(sr.Marshalizer(), sr.Hasher(), sr.Header)
		if err != nil {
			return err
		}

		metaMiniBlocks, metaTransactions := sr.extractMetaMiniBlocksAndTransactions(miniBlocks, transactions)

		err = sr.BroadcastMessenger().SetDataForDelayBroadcast(headerHash, miniBlocks, transactions)
		if err != nil {
			return err
		}

		go sr.broadcast(metaMiniBlocks, metaTransactions, 0)
		return nil
	}

	go sr.broadcast(miniBlocks, transactions, core.ExtraDelayForBroadcastBlockInfo)
	return nil
}

func (sr *subroundEndRound) extractMetaMiniBlocksAndTransactions(
	miniBlocks map[uint32][]byte,
	transactions map[string][][]byte,
) (map[uint32][]byte, map[string][][]byte) {

	metaMiniBlocks := make(map[uint32][]byte, 0)
	metaTransactions := make(map[string][][]byte, 0)

	for shardID, mbsMarshalized := range miniBlocks {
		if shardID != core.MetachainShardId {
			continue
		}

		metaMiniBlocks[shardID] = mbsMarshalized
		delete(miniBlocks, shardID)
	}

	identifier := sr.ShardCoordinator().CommunicationIdentifier(core.MetachainShardId)

	for broadcastTopic, txsMarshalized := range transactions {
		if !strings.Contains(broadcastTopic, identifier) {
			continue
		}

		metaTransactions[broadcastTopic] = txsMarshalized
		delete(transactions, broadcastTopic)
	}

	return metaMiniBlocks, metaTransactions
}

func (sr *subroundEndRound) broadcast(
	miniBlocks map[uint32][]byte,
	transactions map[string][][]byte,
	extraDelayForBroadcast time.Duration,
) {
	time.Sleep(extraDelayForBroadcast)

	if len(miniBlocks) > 0 {
		err := sr.BroadcastMessenger().BroadcastMiniBlocks(miniBlocks)
		if err != nil {
			log.Warn("broadcast.BroadcastMiniBlocks", "error", err.Error())
		}
	}

	if len(transactions) > 0 {
		err := sr.BroadcastMessenger().BroadcastTransactions(transactions)
		if err != nil {
			log.Warn("broadcast.BroadcastTransactions", "error", err.Error())
		}
	}
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
	startTime := sr.RoundTimeStamp
	maxTime := sr.Rounder().TimeDuration() * time.Duration(sr.processingThresholdPercentage) / 100
	if sr.Rounder().RemainingTime(startTime, maxTime) < 0 {
		log.Debug("canceled round, time is out",
			"round", sr.SyncTimer().FormattedCurrentTime(), sr.Rounder().Index(),
			"subround", sr.Name())

		sr.RoundCanceled = true
		return true
	}

	return false
}
