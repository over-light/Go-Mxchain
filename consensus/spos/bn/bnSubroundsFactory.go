package bn

import (
	"time"

	"github.com/ElrondNetwork/elrond-go/consensus/spos"
	"github.com/ElrondNetwork/elrond-go/consensus/spos/commonSubround"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/indexer"
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
}

// NewSubroundsFactory creates a new factory for BN subrounds
func NewSubroundsFactory(
	consensusDataContainer spos.ConsensusCoreHandler,
	consensusState *spos.ConsensusState,
	worker spos.WorkerHandler,
	chainID []byte,
) (*factory, error) {

	err := checkNewFactoryParams(
		consensusDataContainer,
		consensusState,
		worker,
		chainID,
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
	}

	return &fct, nil
}

func checkNewFactoryParams(
	container spos.ConsensusCoreHandler,
	state *spos.ConsensusState,
	worker spos.WorkerHandler,
	chainID []byte,
) error {
	err := spos.ValidateConsensusCore(container)
	if err != nil {
		return err
	}
	if state == nil {
		return spos.ErrNilConsensusState
	}
	if check.IfNil(worker) {
		return spos.ErrNilWorker
	}
	if len(chainID) == 0 {
		return spos.ErrInvalidChainID
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

// GenerateSubrounds will generate the subrounds used in Belare & Naveen Cns
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

	err = fct.generateCommitmentHashSubround()
	if err != nil {
		return err
	}

	err = fct.generateBitmapSubround()
	if err != nil {
		return err
	}

	err = fct.generateCommitmentSubround()
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
	)
	if err != nil {
		return err
	}

	err = subround.SetAppStatusHandler(fct.appStatusHandler)
	if err != nil {
		return err
	}

	subroundStartRound, err := commonSubround.NewSubroundStartRound(
		subround,
		fct.worker.Extend,
		processingThresholdPercent,
		getSubroundName,
		fct.worker.ExecuteStoredMessages,
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
		SrCommitmentHash,
		int64(float64(fct.getTimeDuration())*srBlockStartTime),
		int64(float64(fct.getTimeDuration())*srBlockEndTime),
		getSubroundName(SrBlock),
		fct.consensusState,
		fct.worker.GetConsensusStateChangedChannel(),
		fct.worker.ExecuteStoredMessages,
		fct.consensusCore,
		fct.chainID,
	)
	if err != nil {
		return err
	}

	err = subround.SetAppStatusHandler(fct.appStatusHandler)
	if err != nil {
		return err
	}

	subroundBlock, err := commonSubround.NewSubroundBlock(
		subround,
		fct.worker.Extend,
		int(MtBlockBody),
		int(MtBlockHeader),
		processingThresholdPercent,
		getSubroundName,
	)
	if err != nil {
		return err
	}

	fct.worker.AddReceivedMessageCall(MtBlockBody, subroundBlock.ReceivedBlockBody)
	fct.worker.AddReceivedMessageCall(MtBlockHeader, subroundBlock.ReceivedBlockHeader)
	fct.consensusCore.Chronology().AddSubround(subroundBlock)

	return nil
}

func (fct *factory) generateCommitmentHashSubround() error {
	subround, err := spos.NewSubround(
		SrBlock,
		SrCommitmentHash,
		SrBitmap,
		int64(float64(fct.getTimeDuration())*srCommitmentHashStartTime),
		int64(float64(fct.getTimeDuration())*srCommitmentHashEndTime),
		getSubroundName(SrCommitmentHash),
		fct.consensusState,
		fct.worker.GetConsensusStateChangedChannel(),
		fct.worker.ExecuteStoredMessages,
		fct.consensusCore,
		fct.chainID,
	)
	if err != nil {
		return err
	}

	subroundCommitmentHashObject, err := NewSubroundCommitmentHash(
		subround,
		fct.worker.Extend,
	)
	if err != nil {
		return err
	}

	fct.worker.AddReceivedMessageCall(MtCommitmentHash, subroundCommitmentHashObject.receivedCommitmentHash)
	fct.consensusCore.Chronology().AddSubround(subroundCommitmentHashObject)

	return nil
}

func (fct *factory) generateBitmapSubround() error {
	subround, err := spos.NewSubround(
		SrCommitmentHash,
		SrBitmap,
		SrCommitment,
		int64(float64(fct.getTimeDuration())*srBitmapStartTime),
		int64(float64(fct.getTimeDuration())*srBitmapEndTime),
		getSubroundName(SrBitmap),
		fct.consensusState,
		fct.worker.GetConsensusStateChangedChannel(),
		fct.worker.ExecuteStoredMessages,
		fct.consensusCore,
		fct.chainID,
	)
	if err != nil {
		return err
	}

	subroundBitmapObject, err := NewSubroundBitmap(
		subround,
		fct.worker.Extend,
	)
	if err != nil {
		return err
	}

	fct.worker.AddReceivedMessageCall(MtBitmap, subroundBitmapObject.receivedBitmap)
	fct.consensusCore.Chronology().AddSubround(subroundBitmapObject)

	return nil
}

func (fct *factory) generateCommitmentSubround() error {
	subroundObject, err := spos.NewSubround(
		SrBitmap,
		SrCommitment,
		SrSignature,
		int64(float64(fct.getTimeDuration())*srCommitmentStartTime),
		int64(float64(fct.getTimeDuration())*srCommitmentEndTime),
		getSubroundName(SrCommitment),
		fct.consensusState,
		fct.worker.GetConsensusStateChangedChannel(),
		fct.worker.ExecuteStoredMessages,
		fct.consensusCore,
		fct.chainID,
	)
	if err != nil {
		return err
	}

	subroundCommitmentObject, err := NewSubroundCommitment(
		subroundObject,
		fct.worker.Extend,
	)
	if err != nil {
		return err
	}

	fct.worker.AddReceivedMessageCall(MtCommitment, subroundCommitmentObject.receivedCommitment)
	fct.consensusCore.Chronology().AddSubround(subroundCommitmentObject)

	return nil
}

func (fct *factory) generateSignatureSubround() error {
	subround, err := spos.NewSubround(
		SrCommitment,
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
	)
	if err != nil {
		return err
	}

	subroundEndRoundObject, err := NewSubroundEndRound(
		subround,
		fct.worker.Extend,
	)
	if err != nil {
		return err
	}

	err = subroundEndRoundObject.SetAppStatusHandler(fct.appStatusHandler)
	if err != nil {
		return err
	}

	fct.consensusCore.Chronology().AddSubround(subroundEndRoundObject)

	return nil
}

func (fct *factory) initConsensusThreshold() {
	pbftThreshold := fct.consensusState.ConsensusGroupSize()*2/3 + 1
	fct.consensusState.SetThreshold(SrBlock, 1)
	fct.consensusState.SetThreshold(SrCommitmentHash, pbftThreshold)
	fct.consensusState.SetThreshold(SrBitmap, pbftThreshold)
	fct.consensusState.SetThreshold(SrCommitment, pbftThreshold)
	fct.consensusState.SetThreshold(SrSignature, pbftThreshold)
}

// IsInterfaceNil returns true if there is no value under the interface
func (fct *factory) IsInterfaceNil() bool {
	return fct == nil
}

func debugError(message string, err error) {
	log.Debug(message, "type", "spos/bn", "error", err.Error())
}
