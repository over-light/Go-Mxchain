package bls

import (
	"time"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/spos"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
)

// subroundBlock defines the data needed by the subround Block
type subroundBlock struct {
	*spos.Subround

	processingThresholdPercentage int
}

// NewSubroundBlock creates a subroundBlock object
func NewSubroundBlock(
	baseSubround *spos.Subround,
	extend func(subroundId int),
	processingThresholdPercentage int,
) (*subroundBlock, error) {
	err := checkNewSubroundBlockParams(baseSubround)
	if err != nil {
		return nil, err
	}

	srBlock := subroundBlock{
		Subround:                      baseSubround,
		processingThresholdPercentage: processingThresholdPercentage,
	}

	srBlock.Job = srBlock.doBlockJob
	srBlock.Check = srBlock.doBlockConsensusCheck
	srBlock.Extend = extend

	return &srBlock, nil
}

func checkNewSubroundBlockParams(
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

// doBlockJob method does the job of the subround Block
func (sr *subroundBlock) doBlockJob() bool {
	if !sr.IsSelfLeaderInCurrentRound() { // is NOT self leader in this round?
		return false
	}

	if sr.Rounder().Index() <= sr.getRoundInLastCommittedBlock() {
		return false
	}

	if sr.IsSelfJobDone(sr.Current()) {
		return false
	}

	if sr.IsSubroundFinished(sr.Current()) {
		return false
	}

	metricStatTime := time.Now()
	defer sr.computeSubroundProcessingMetric(metricStatTime, core.MetricCreatedProposedBlock)

	header, err := sr.createHeader()
	if err != nil {
		log.Debug("doBlockJob.createHeader", "error", err.Error())
		return false
	}

	body, err := sr.createBody(header)
	if err != nil {
		log.Debug("doBlockJob.createBody", "error", err.Error())
		return false
	}

	body, err = sr.BlockProcessor().ApplyBodyToHeader(sr.Blockchain(), header, body)
	if err != nil {
		log.Debug("doBlockJob.ApplyBodyToHeader", "error", err.Error())
		return false
	}

	sentWithSuccess := sr.sendBlock(body, header)
	if !sentWithSuccess {
		return false
	}

	err = sr.SetSelfJobDone(sr.Current(), true)
	if err != nil {
		log.Debug("doBlockJob.SetSelfJobDone", "error", err.Error())
		return false
	}

	return true
}

func (sr *subroundBlock) sendBlock(body data.BodyHandler, header data.HeaderHandler) bool {
	marshalizedBody, err := sr.Marshalizer().Marshal(body)
	if err != nil {
		log.Debug("sendBlock.Marshal: body", "error", err.Error())
		return false
	}

	marshalizedHeader, err := sr.Marshalizer().Marshal(header)
	if err != nil {
		log.Debug("sendBlock.Marshal: header", "error", err.Error())
		return false
	}

	if sr.canBeSentTogether(marshalizedBody, marshalizedHeader) {
		return sr.sendBlockBodyAndHeader(body, header, marshalizedBody, marshalizedHeader)
	}

	if !sr.sendBlockBody(body, marshalizedBody) || !sr.sendBlockHeader(header, marshalizedHeader) {
		return false
	}

	return true
}

func (sr *subroundBlock) canBeSentTogether(marshalizedBody []byte, marshalizedHeader []byte) bool {
	bodyAndHeaderSize := len(marshalizedBody) + len(marshalizedHeader)
	log.Trace("consensus block message",
		"body size", len(marshalizedBody),
		"header size", len(marshalizedHeader),
		"body and header size", bodyAndHeaderSize)
	return bodyAndHeaderSize <= core.MegabyteSize
}

func (sr *subroundBlock) createBody(header data.HeaderHandler) (data.BodyHandler, error) {
	startTime := sr.RoundTimeStamp
	maxTime := time.Duration(sr.EndTime())
	haveTimeInCurrentSubround := func() bool {
		return sr.Rounder().RemainingTime(startTime, maxTime) > 0
	}

	blockBody, err := sr.BlockProcessor().CreateBlockBody(
		header,
		haveTimeInCurrentSubround,
	)
	if err != nil {
		return nil, err
	}

	return blockBody, nil
}

// sendBlockBodyAndHeader method sends the proposed block body and header in the subround Block
func (sr *subroundBlock) sendBlockBodyAndHeader(
	body data.BodyHandler,
	header data.HeaderHandler,
	marshalizedBody []byte,
	marshalizedHeader []byte,
) bool {

	marshalizedBodyAndHeader := data.MarshalizedBodyAndHeader{
		Body:   marshalizedBody,
		Header: marshalizedHeader,
	}

	subRoundData, err := sr.Marshalizer().Marshal(&marshalizedBodyAndHeader)
	if err != nil {
		log.Debug("sendBlockBodyAndHeader.Marshal: marshalizedBodyAndHeader", "error", err.Error())
		return false
	}

	headerHash, err := core.CalculateHash(sr.Marshalizer(), sr.Hasher(), header)
	if err != nil {
		log.Debug("sendBlockBodyAndHeader.CalculateHash", "error", err.Error())
		return false
	}

	cnsMsg := consensus.NewConsensusMessage(
		headerHash,
		subRoundData,
		[]byte(sr.SelfPubKey()),
		nil,
		int(MtBlockBodyAndHeader),
		sr.Rounder().Index(),
		sr.ChainID(),
		nil,
		nil,
		nil,
	)

	err = sr.BroadcastMessenger().BroadcastConsensusMessage(cnsMsg)
	if err != nil {
		log.Debug("sendBlockBodyAndHeader.BroadcastConsensusMessage", "error", err.Error())
		return false
	}

	log.Debug("step 1: block body and header have been sent",
		"nonce", header.GetNonce(),
		"hash", headerHash)

	sr.Data = headerHash
	sr.Body = body
	sr.Header = header

	return true
}

// sendBlockBody method sends the proposed block body in the subround Block
func (sr *subroundBlock) sendBlockBody(body data.BodyHandler, marshalizedBody []byte) bool {
	cnsMsg := consensus.NewConsensusMessage(
		nil,
		marshalizedBody,
		[]byte(sr.SelfPubKey()),
		nil,
		int(MtBlockBody),
		sr.Rounder().Index(),
		sr.ChainID(),
		nil,
		nil,
		nil,
	)

	err := sr.BroadcastMessenger().BroadcastConsensusMessage(cnsMsg)
	if err != nil {
		log.Debug("sendBlockBody.BroadcastConsensusMessage", "error", err.Error())
		return false
	}

	log.Debug("step 1: block body has been sent")

	sr.Body = body

	return true
}

// sendBlockHeader method sends the proposed block header in the subround Block
func (sr *subroundBlock) sendBlockHeader(header data.HeaderHandler, marshalizedHeader []byte) bool {
	headerHash := sr.Hasher().Compute(string(marshalizedHeader))

	cnsMsg := consensus.NewConsensusMessage(
		headerHash,
		marshalizedHeader,
		[]byte(sr.SelfPubKey()),
		nil,
		int(MtBlockHeader),
		sr.Rounder().Index(),
		sr.ChainID(),
		nil,
		nil,
		nil,
	)

	err := sr.BroadcastMessenger().BroadcastConsensusMessage(cnsMsg)
	if err != nil {
		log.Debug("sendBlockHeader.BroadcastConsensusMessage", "error", err.Error())
		return false
	}

	log.Debug("step 1: block header has been sent",
		"nonce", header.GetNonce(),
		"hash", headerHash)

	sr.Data = headerHash
	sr.Header = header

	return true
}

func (sr *subroundBlock) createHeader() (data.HeaderHandler, error) {
	hdr := sr.BlockProcessor().CreateNewHeader()

	var prevRandSeed []byte
	if sr.Blockchain().GetCurrentBlockHeader() == nil {
		hdr.SetNonce(1)
		hdr.SetPrevHash(sr.Blockchain().GetGenesisHeaderHash())

		prevRandSeed = sr.Blockchain().GetGenesisHeader().GetRandSeed()
	} else {
		hdr.SetNonce(sr.Blockchain().GetCurrentBlockHeader().GetNonce() + 1)
		hdr.SetPrevHash(sr.Blockchain().GetCurrentBlockHeaderHash())

		prevRandSeed = sr.Blockchain().GetCurrentBlockHeader().GetRandSeed()
	}

	randSeed, err := sr.SingleSigner().Sign(sr.PrivateKey(), prevRandSeed)
	if err != nil {
		return nil, err
	}

	hdr.SetShardID(sr.ShardCoordinator().SelfId())
	hdr.SetRound(uint64(sr.Rounder().Index()))
	hdr.SetTimeStamp(uint64(sr.Rounder().TimeStamp().Unix()))
	hdr.SetPrevRandSeed(prevRandSeed)
	hdr.SetRandSeed(randSeed)
	hdr.SetChainID(sr.ChainID())

	return hdr, nil
}

// receivedBlockBodyAndHeader method is called when a block body and a block header is received
func (sr *subroundBlock) receivedBlockBodyAndHeader(cnsDta *consensus.Message) bool {
	node := string(cnsDta.PubKey)

	if sr.IsConsensusDataSet() {
		return false
	}

	if sr.IsBlockBodyAlreadyReceived() {
		return false
	}

	if sr.IsHeaderAlreadyReceived() {
		return false
	}

	if !sr.IsNodeLeaderInCurrentRound(node) { // is NOT this node leader in current round?
		return false
	}

	if !sr.CanProcessReceivedMessage(cnsDta, sr.Rounder().Index(), sr.Current()) {
		return false
	}

	sr.Data = cnsDta.BlockHeaderHash
	sr.Body, sr.Header = sr.BlockProcessor().DecodeBlockBodyAndHeader(cnsDta.SubRoundData)

	if check.IfNil(sr.Body) || check.IfNil(sr.Header) {
		return false
	}

	log.Debug("step 1: block body and header have been received",
		"nonce", sr.Header.GetNonce(),
		"hash", cnsDta.BlockHeaderHash)
	blockProcessedWithSuccess := sr.processReceivedBlock(cnsDta)

	return blockProcessedWithSuccess
}

// receivedBlockBody method is called when a block body is received through the block body channel
func (sr *subroundBlock) receivedBlockBody(cnsDta *consensus.Message) bool {
	node := string(cnsDta.PubKey)

	if sr.IsBlockBodyAlreadyReceived() {
		return false
	}

	if !sr.IsNodeLeaderInCurrentRound(node) { // is NOT this node leader in current round?
		return false
	}

	if !sr.CanProcessReceivedMessage(cnsDta, sr.Rounder().Index(), sr.Current()) {
		return false
	}

	sr.Body = sr.BlockProcessor().DecodeBlockBody(cnsDta.SubRoundData)

	if check.IfNil(sr.Body) {
		return false
	}

	log.Debug("step 1: block body has been received")

	blockProcessedWithSuccess := sr.processReceivedBlock(cnsDta)

	return blockProcessedWithSuccess
}

// receivedBlockHeader method is called when a block header is received through the block header channel.
// If the block header is valid, than the validatorRoundStates map corresponding to the node which sent it,
// is set on true for the subround Block
func (sr *subroundBlock) receivedBlockHeader(cnsDta *consensus.Message) bool {
	node := string(cnsDta.PubKey)

	if sr.IsConsensusDataSet() {
		return false
	}

	if sr.IsHeaderAlreadyReceived() {
		return false
	}

	if !sr.IsNodeLeaderInCurrentRound(node) { // is NOT this node leader in current round?
		return false
	}

	if !sr.CanProcessReceivedMessage(cnsDta, sr.Rounder().Index(), sr.Current()) {
		return false
	}

	sr.Data = cnsDta.BlockHeaderHash
	sr.Header = sr.BlockProcessor().DecodeBlockHeader(cnsDta.SubRoundData)

	if check.IfNil(sr.Header) {
		return false
	}

	log.Debug("step 1: block header has been received",
		"nonce", sr.Header.GetNonce(),
		"hash", cnsDta.BlockHeaderHash)
	blockProcessedWithSuccess := sr.processReceivedBlock(cnsDta)

	return blockProcessedWithSuccess
}

func (sr *subroundBlock) processReceivedBlock(cnsDta *consensus.Message) bool {
	if check.IfNil(sr.Body) {
		return false
	}
	if check.IfNil(sr.Header) {
		return false
	}

	defer func() {
		sr.SetProcessingBlock(false)
	}()

	sr.SetProcessingBlock(true)

	node := string(cnsDta.PubKey)

	startTime := sr.RoundTimeStamp
	maxTime := sr.Rounder().TimeDuration() * time.Duration(sr.processingThresholdPercentage) / 100
	remainingTimeInCurrentRound := func() time.Duration {
		return sr.Rounder().RemainingTime(startTime, maxTime)
	}

	metricStatTime := time.Now()
	defer sr.computeSubroundProcessingMetric(metricStatTime, core.MetricProcessedProposedBlock)

	err := sr.BlockProcessor().ProcessBlock(
		sr.Blockchain(),
		sr.Header,
		sr.Body,
		remainingTimeInCurrentRound,
	)

	if cnsDta.RoundIndex < sr.Rounder().Index() {
		log.Debug("canceled round, meantime round index has been changed",
			"old round", cnsDta.RoundIndex,
			"subround", sr.Name(),
			"new round", sr.Rounder().Index())
		return false
	}

	if err != nil {
		log.Debug("canceled round",
			"round", sr.Rounder().Index(),
			"subround", sr.Name(),
			"error", err.Error())

		sr.RoundCanceled = true

		return false
	}

	err = sr.SetJobDone(node, sr.Current(), true)
	if err != nil {
		log.Debug("canceled round",
			"round", sr.Rounder().Index(),
			"subround", sr.Name(),
			"error", err.Error())
		return false
	}

	return true
}

func (sr *subroundBlock) computeSubroundProcessingMetric(startTime time.Time, metric string) {
	subRoundDuration := sr.EndTime() - sr.StartTime()
	if subRoundDuration == 0 {
		//can not do division by 0
		return
	}

	percent := uint64(time.Since(startTime)) * 100 / uint64(subRoundDuration)
	sr.AppStatusHandler().SetUInt64Value(metric, percent)
}

// doBlockConsensusCheck method checks if the consensus in the subround Block is achieved
func (sr *subroundBlock) doBlockConsensusCheck() bool {
	if sr.RoundCanceled {
		return false
	}

	if sr.IsSubroundFinished(sr.Current()) {
		return true
	}

	threshold := sr.Threshold(sr.Current())
	if sr.isBlockReceived(threshold) {
		log.Debug("step 1: subround has been finished",
			"subround", sr.Name())
		sr.SetStatus(sr.Current(), spos.SsFinished)
		return true
	}

	return false
}

// isBlockReceived method checks if the block was received from the leader in the current round
func (sr *subroundBlock) isBlockReceived(threshold int) bool {
	n := 0

	for i := 0; i < len(sr.ConsensusGroup()); i++ {
		node := sr.ConsensusGroup()[i]
		isJobDone, err := sr.JobDone(node, sr.Current())
		if err != nil {
			log.Debug("isBlockReceived.JobDone",
				"node", node,
				"subround", sr.Name(),
				"error", err.Error())
			continue
		}

		if isJobDone {
			n++
		}
	}

	return n >= threshold
}

func (sr *subroundBlock) getRoundInLastCommittedBlock() int64 {
	roundInLastCommittedBlock := int64(0)
	if sr.Blockchain().GetCurrentBlockHeader() != nil {
		roundInLastCommittedBlock = int64(sr.Blockchain().GetCurrentBlockHeader().GetRound())
	}

	return roundInLastCommittedBlock
}
