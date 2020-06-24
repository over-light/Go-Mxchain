package bls

import (
	"time"

	"github.com/ElrondNetwork/elrond-go/consensus/spos"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/indexer"
	"github.com/ElrondNetwork/elrond-go/data/endProcess"
	"github.com/ElrondNetwork/elrond-go/statusHandler"
)

// factory defines the data needed by this factory to create all the subrounds and give them their specific
// functionality
type factory struct {
	consensusCore  spos.ConsensusCoreHandler
	consensusState *spos.ConsensusState
	worker         spos.WorkerHandler

	appStatusHandler core.AppStatusHandler
	indexer          indexer.Indexer
	chainID          []byte
	currentPid       core.PeerID

	alarmScheduler      core.TimersScheduler
	chanStopNodeProcess chan endProcess.ArgEndProcess
}

// NewSubroundsFactory creates a new consensusState object
func NewSubroundsFactory(
	consensusDataContainer spos.ConsensusCoreHandler,
	consensusState *spos.ConsensusState,
	worker spos.WorkerHandler,
	chainID []byte,
	currentPid core.PeerID,
	alarmScheduler core.TimersScheduler,
	chanStopNodeProcess chan endProcess.ArgEndProcess,
) (*factory, error) {
	err := checkNewFactoryParams(
		consensusDataContainer,
		consensusState,
		worker,
		chainID,
		alarmScheduler,
		chanStopNodeProcess,
	)
	if err != nil {
		return nil, err
	}

	fct := factory{
		consensusCore:    consensusDataContainer,
		consensusState:   consensusState,
		worker:           worker,
		appStatusHandler: statusHandler.NewNilStatusHandler(),
		chainID:          chainID,
		currentPid:       currentPid,
		alarmScheduler:      alarmScheduler,
		chanStopNodeProcess: chanStopNodeProcess,
	}

	return &fct, nil
}

func checkNewFactoryParams(
	container spos.ConsensusCoreHandler,
	state *spos.ConsensusState,
	worker spos.WorkerHandler,
	chainID []byte,
	alarmScheduler core.TimersScheduler,
	endProcess chan endProcess.ArgEndProcess,
) error {
	err := spos.ValidateConsensusCore(container)
	if err != nil {
		return err
	}
	if state == nil {
		return spos.ErrNilConsensusState
	}
	if worker == nil || worker.IsInterfaceNil() {
		return spos.ErrNilWorker
	}
	if len(chainID) == 0 {
		return spos.ErrInvalidChainID
	}
	if check.IfNil(alarmScheduler) {
		return spos.ErrNilAlarmScheduler
	}
	if endProcess == nil {
		return spos.ErrNilEndProcessChan
	}

	return nil
}

// SetAppStatusHandler method will update the value of the factory's appStatusHandler
func (fct *factory) SetAppStatusHandler(ash core.AppStatusHandler) error {
	if check.IfNil(ash) {
		return spos.ErrNilAppStatusHandler
	}
	fct.appStatusHandler = ash

	return fct.worker.SetAppStatusHandler(ash)
}

// SetIndexer method will update the value of the factory's indexer
func (fct *factory) SetIndexer(indexer indexer.Indexer) {
	fct.indexer = indexer
}

// GenerateSubrounds will generate the subrounds used in BLS Cns
func (fct *factory) GenerateSubrounds() error {
	fct.initConsensusThreshold()
	fct.consensusCore.Chronology().RemoveAllSubrounds()
	fct.worker.RemoveAllReceivedMessagesCalls()

	err := fct.generateStartRoundSubround()
	if err != nil {
		return err
	}

	err = fct.generateBlockSubround()
	if err != nil {
		return err
	}

	err = fct.generateSignatureSubround()
	if err != nil {
		return err
	}

	err = fct.generateEndRoundSubround()
	if err != nil {
		return err
	}

	return nil
}

func (fct *factory) getTimeDuration() time.Duration {
	return fct.consensusCore.Rounder().TimeDuration()
}

func (fct *factory) generateStartRoundSubround() error {
	subround, err := spos.NewSubround(
		-1,
		SrStartRound,
		SrBlock,
		int64(float64(fct.getTimeDuration())*srStartStartTime),
		int64(float64(fct.getTimeDuration())*srStartEndTime),
		getSubroundName(SrStartRound),
		fct.consensusState,
		fct.worker.GetConsensusStateChangedChannel(),
		fct.worker.ExecuteStoredMessages,
		fct.consensusCore,
		fct.chainID,
		fct.currentPid,
	)
	if err != nil {
		return err
	}

	err = subround.SetAppStatusHandler(fct.appStatusHandler)
	if err != nil {
		return err
	}

	subroundStartRound, err := NewSubroundStartRound(
		subround,
		fct.worker.Extend,
		processingThresholdPercent,
		fct.worker.ExecuteStoredMessages,
		fct.alarmScheduler,
		fct.chanStopNodeProcess,
	)
	if err != nil {
		return err
	}

	subroundStartRound.SetIndexer(fct.indexer)

	fct.consensusCore.Chronology().AddSubround(subroundStartRound)

	return nil
}

func (fct *factory) generateBlockSubround() error {
	subround, err := spos.NewSubround(
		SrStartRound,
		SrBlock,
		SrSignature,
		int64(float64(fct.getTimeDuration())*srBlockStartTime),
		int64(float64(fct.getTimeDuration())*srBlockEndTime),
		getSubroundName(SrBlock),
		fct.consensusState,
		fct.worker.GetConsensusStateChangedChannel(),
		fct.worker.ExecuteStoredMessages,
		fct.consensusCore,
		fct.chainID,
		fct.currentPid,
	)
	if err != nil {
		return err
	}

	err = subround.SetAppStatusHandler(fct.appStatusHandler)
	if err != nil {
		return err
	}

	subroundBlock, err := NewSubroundBlock(
		subround,
		fct.worker.Extend,
		processingThresholdPercent,
	)
	if err != nil {
		return err
	}

	fct.worker.AddReceivedMessageCall(MtBlockBodyAndHeader, subroundBlock.receivedBlockBodyAndHeader)
	fct.worker.AddReceivedMessageCall(MtBlockBody, subroundBlock.receivedBlockBody)
	fct.worker.AddReceivedMessageCall(MtBlockHeader, subroundBlock.receivedBlockHeader)
	fct.consensusCore.Chronology().AddSubround(subroundBlock)

	return nil
}

func (fct *factory) generateSignatureSubround() error {
	subround, err := spos.NewSubround(
		SrBlock,
		SrSignature,
		SrEndRound,
		int64(float64(fct.getTimeDuration())*srSignatureStartTime),
		int64(float64(fct.getTimeDuration())*srSignatureEndTime),
		getSubroundName(SrSignature),
		fct.consensusState,
		fct.worker.GetConsensusStateChangedChannel(),
		fct.worker.ExecuteStoredMessages,
		fct.consensusCore,
		fct.chainID,
		fct.currentPid,
	)
	if err != nil {
		return err
	}

	subroundSignatureObject, err := NewSubroundSignature(
		subround,
		fct.worker.Extend,
	)
	if err != nil {
		return err
	}

	err = subroundSignatureObject.SetAppStatusHandler(fct.appStatusHandler)
	if err != nil {
		return err
	}

	fct.worker.AddReceivedMessageCall(MtSignature, subroundSignatureObject.receivedSignature)
	fct.consensusCore.Chronology().AddSubround(subroundSignatureObject)

	return nil
}

func (fct *factory) generateEndRoundSubround() error {
	subround, err := spos.NewSubround(
		SrSignature,
		SrEndRound,
		-1,
		int64(float64(fct.getTimeDuration())*srEndStartTime),
		int64(float64(fct.getTimeDuration())*srEndEndTime),
		getSubroundName(SrEndRound),
		fct.consensusState,
		fct.worker.GetConsensusStateChangedChannel(),
		fct.worker.ExecuteStoredMessages,
		fct.consensusCore,
		fct.chainID,
		fct.currentPid,
	)
	if err != nil {
		return err
	}

	subroundEndRoundObject, err := NewSubroundEndRound(
		subround,
		fct.worker.Extend,
		spos.MaxThresholdPercent,
		fct.worker.DisplayStatistics,
	)
	if err != nil {
		return err
	}

	err = subroundEndRoundObject.SetAppStatusHandler(fct.appStatusHandler)
	if err != nil {
		return err
	}

	fct.worker.AddReceivedMessageCall(MtBlockHeaderFinalInfo, subroundEndRoundObject.receivedBlockHeaderFinalInfo)
	fct.worker.AddReceivedHeaderHandler(subroundEndRoundObject.receivedHeader)
	fct.consensusCore.Chronology().AddSubround(subroundEndRoundObject)

	return nil
}

func (fct *factory) initConsensusThreshold() {
	pbftThreshold := fct.consensusState.ConsensusGroupSize()*2/3 + 1
	fct.consensusState.SetThreshold(SrBlock, 1)
	fct.consensusState.SetThreshold(SrSignature, pbftThreshold)
}

// IsInterfaceNil returns true if there is no value under the interface
func (fct *factory) IsInterfaceNil() bool {
	return fct == nil
}
