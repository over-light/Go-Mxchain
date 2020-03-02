package mock

import (
	"time"

	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process/block/processedMb"
)

// BlockProcessorMock mocks the implementation for a blockProcessor
type BlockProcessorMock struct {
	NrCommitBlockCalled                     uint32
	Marshalizer                             marshal.Marshalizer
	ProcessBlockCalled                      func(header data.HeaderHandler, body data.BodyHandler, haveTime func() time.Duration) error
	CommitBlockCalled                       func(header data.HeaderHandler, body data.BodyHandler) error
	RevertAccountStateCalled                func()
	CreateBlockCalled                       func(initialHdrData data.HeaderHandler, haveTime func() bool) (data.HeaderHandler, data.BodyHandler, error)
	RestoreBlockIntoPoolsCalled             func(header data.HeaderHandler, body data.BodyHandler) error
	MarshalizedDataToBroadcastCalled        func(header data.HeaderHandler, body data.BodyHandler) (map[uint32][]byte, map[string][][]byte, error)
	AddLastNotarizedHdrCalled               func(shardId uint32, processedHdr data.HeaderHandler)
	CreateNewHeaderCalled                   func() data.HeaderHandler
	UpdateEpochStartTriggerRoundCalled      func(round uint64)
	PruneStateOnRollbackCalled              func(currHeader data.HeaderHandler, prevHeader data.HeaderHandler)
	RestoreLastNotarizedHrdsToGenesisCalled func()
	RevertStateToBlockCalled                func(header data.HeaderHandler) error
}

// RestoreLastNotarizedHrdsToGenesis -
func (bpm *BlockProcessorMock) RestoreLastNotarizedHrdsToGenesis() {
	if bpm.RestoreLastNotarizedHrdsToGenesisCalled != nil {
		bpm.RestoreLastNotarizedHrdsToGenesisCalled()
	}
}

// ProcessBlock mocks pocessing a block
func (bpm *BlockProcessorMock) ProcessBlock(header data.HeaderHandler, body data.BodyHandler, haveTime func() time.Duration) error {
	return bpm.ProcessBlockCalled(header, body, haveTime)
}

// ApplyProcessedMiniBlocks -
func (bpm *BlockProcessorMock) ApplyProcessedMiniBlocks(_ *processedMb.ProcessedMiniBlockTracker) {
}

// CommitBlock mocks the commit of a block
func (bpm *BlockProcessorMock) CommitBlock(header data.HeaderHandler, body data.BodyHandler) error {
	return bpm.CommitBlockCalled(header, body)
}

// RevertAccountState mocks revert of the accounts state
func (bpm *BlockProcessorMock) RevertAccountState() {
	bpm.RevertAccountStateCalled()
}

// CreateNewHeader -
func (bpm *BlockProcessorMock) CreateNewHeader() data.HeaderHandler {
	return bpm.CreateNewHeaderCalled()
}

// UpdateEpochStartTriggerRound -
func (bpm *BlockProcessorMock) UpdateEpochStartTriggerRound(round uint64) {
	if bpm.UpdateEpochStartTriggerRoundCalled != nil {
		bpm.UpdateEpochStartTriggerRoundCalled(round)
	}
}

// CreateBlock -
func (bpm *BlockProcessorMock) CreateBlock(initialHdrData data.HeaderHandler, haveTime func() bool) (data.HeaderHandler, data.BodyHandler, error) {
	return bpm.CreateBlockCalled(initialHdrData, haveTime)
}

// RestoreBlockIntoPools -
func (bpm *BlockProcessorMock) RestoreBlockIntoPools(header data.HeaderHandler, body data.BodyHandler) error {
	return bpm.RestoreBlockIntoPoolsCalled(header, body)
}

// RevertStateToBlock recreates the state tries to the root hashes indicated by the provided header
func (bpm *BlockProcessorMock) RevertStateToBlock(header data.HeaderHandler) error {
	if bpm.RevertStateToBlockCalled != nil {
		return bpm.RevertStateToBlockCalled(header)
	}
	return nil
}

// PruneStateOnRollback recreates the state tries to the root hashes indicated by the provided header
func (bpm *BlockProcessorMock) PruneStateOnRollback(currHeader data.HeaderHandler, prevHeader data.HeaderHandler) {
	if bpm.PruneStateOnRollbackCalled != nil {
		bpm.PruneStateOnRollbackCalled(currHeader, prevHeader)
	}
}

// SetNumProcessedObj -
func (bpm *BlockProcessorMock) SetNumProcessedObj(_ uint64) {
}

// MarshalizedDataToBroadcast -
func (bpm *BlockProcessorMock) MarshalizedDataToBroadcast(header data.HeaderHandler, body data.BodyHandler) (map[uint32][]byte, map[string][][]byte, error) {
	return bpm.MarshalizedDataToBroadcastCalled(header, body)
}

// AddLastNotarizedHdr -
func (bpm *BlockProcessorMock) AddLastNotarizedHdr(shardId uint32, processedHdr data.HeaderHandler) {
	bpm.AddLastNotarizedHdrCalled(shardId, processedHdr)
}

// IsInterfaceNil returns true if there is no value under the interface
func (bpm *BlockProcessorMock) IsInterfaceNil() bool {
	return bpm == nil
}
