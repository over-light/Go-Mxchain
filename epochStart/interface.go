package epochStart

import (
	"time"

	"github.com/ElrondNetwork/elrond-go/data"
)

// TriggerHandler defines the functionalities for an start of epoch trigger
type TriggerHandler interface {
	ForceEpochStart(round uint64) error
	IsEpochStart() bool
	Epoch() uint32
	Update(round uint64)
	EpochStartRound() uint64
	EpochStartMetaHdrHash() []byte
	GetSavedStateKey() []byte
	LoadState(key []byte) error
	SetProcessed(header data.HeaderHandler)
	SetFinalityAttestingRound(round uint64)
	EpochFinalityAttestingRound() uint64
	RevertStateToBlock(header data.HeaderHandler) error
	SetCurrentEpochStartRound(round uint64)
	RequestEpochStartIfNeeded(interceptedHeader data.HeaderHandler)
	IsInterfaceNil() bool
}

// Rounder defines the actions which should be handled by a round implementation
type Rounder interface {
	// Index returns the current round
	Index() int64
	// TimeStamp returns the time stamp of the round
	TimeStamp() time.Time
	IsInterfaceNil() bool
}

// HeaderValidator defines the actions needed to validate a header
type HeaderValidator interface {
	IsHeaderConstructionValid(currHdr, prevHdr data.HeaderHandler) error
	IsInterfaceNil() bool
}

// RequestHandler defines the methods through which request to data can be made
type RequestHandler interface {
	RequestShardHeader(shardId uint32, hash []byte)
	RequestMetaHeader(hash []byte)
	RequestMetaHeaderByNonce(nonce uint64)
	RequestShardHeaderByNonce(shardId uint32, nonce uint64)
	RequestStartOfEpochMetaBlock(epoch uint32)
	RequestMiniBlocks(destShardID uint32, miniblocksHashes [][]byte)
	IsInterfaceNil() bool
}

// EpochStartHandler defines the action taken on epoch start event
type EpochStartHandler interface {
	EpochStartAction(hdr data.HeaderHandler)
	EpochStartPrepare(hdr data.HeaderHandler)
	NotifyOrder() uint32
}

// EpochStartSubscriber provides Register and Unregister functionality for the end of epoch events
type EpochStartSubscriber interface {
	RegisterHandler(handler EpochStartHandler)
	UnregisterHandler(handler EpochStartHandler)
}

// EpochStartNotifier defines which actions should be done for handling new epoch's events
type EpochStartNotifier interface {
	NotifyAll(hdr data.HeaderHandler)
	NotifyAllPrepare(hdr data.HeaderHandler)
	IsInterfaceNil() bool
}

// ValidatorStatisticsProcessorHandler defines the actions for processing validator statistics
// needed in the epoch events
type ValidatorStatisticsProcessorHandler interface {
	Process(info data.ShardValidatorInfoHandler) error
	Commit() ([]byte, error)
	IsInterfaceNil() bool
}
