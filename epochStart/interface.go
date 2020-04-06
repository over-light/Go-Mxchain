package epochStart

import (
	"context"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
)

// TriggerHandler defines the functionalities for an start of epoch trigger
type TriggerHandler interface {
	ForceEpochStart(round uint64) error
	IsEpochStart() bool
	Epoch() uint32
	Update(round uint64, nonce uint64)
	EpochStartRound() uint64
	EpochStartMetaHdrHash() []byte
	GetSavedStateKey() []byte
	LoadState(key []byte) error
	SetProcessed(header data.HeaderHandler, body data.BodyHandler)
	SetFinalityAttestingRound(round uint64)
	EpochFinalityAttestingRound() uint64
	RevertStateToBlock(header data.HeaderHandler) error
	SetCurrentEpochStartRound(round uint64)
	RequestEpochStartIfNeeded(interceptedHeader data.HeaderHandler)
	SetAppStatusHandler(handler core.AppStatusHandler) error
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
	RequestInterval() time.Duration
	IsInterfaceNil() bool
}

// ActionHandler defines the action taken on epoch start event
type ActionHandler interface {
	EpochStartAction(hdr data.HeaderHandler)
	EpochStartPrepare(metaHdr data.HeaderHandler, body data.BodyHandler)
	NotifyOrder() uint32
}

// RegistrationHandler provides Register and Unregister functionality for the end of epoch events
type RegistrationHandler interface {
	RegisterHandler(handler ActionHandler)
	UnregisterHandler(handler ActionHandler)
}

// Notifier defines which actions should be done for handling new epoch's events
type Notifier interface {
	NotifyAll(hdr data.HeaderHandler)
	NotifyAllPrepare(metaHdr data.HeaderHandler, body data.BodyHandler)
	IsInterfaceNil() bool
}

// ValidatorStatisticsProcessorHandler defines the actions for processing validator statistics
// needed in the epoch events
type ValidatorStatisticsProcessorHandler interface {
	Process(info data.ShardValidatorInfoHandler) error
	Commit() ([]byte, error)
	IsInterfaceNil() bool
}

// HeadersByHashSyncer defines the methods to sync all missing headers by hash
type HeadersByHashSyncer interface {
	SyncMissingHeadersByHash(shardIDs []uint32, headersHashes [][]byte, ctx context.Context) error
	GetHeaders() (map[string]data.HeaderHandler, error)
	ClearFields()
	IsInterfaceNil() bool
}

// PendingMiniBlocksSyncHandler defines the methods to sync all pending miniblocks
type PendingMiniBlocksSyncHandler interface {
	SyncPendingMiniBlocks(miniBlockHeaders []block.ShardMiniBlockHeader, ctx context.Context) error
	GetMiniBlocks() (map[string]*block.MiniBlock, error)
	ClearFields()
	IsInterfaceNil() bool
}

// AccountsDBSyncer defines the methods for the accounts db syncer
type AccountsDBSyncer interface {
	GetSyncedTries() map[string]data.Trie
	SyncAccounts(rootHash []byte) error
	IsInterfaceNil() bool
}

// StartOfEpochMetaSyncer defines the methods to synchronize epoch start meta block from the network when nothing is known
type StartOfEpochMetaSyncer interface {
	SyncEpochStartMeta(waitTime time.Duration) (*block.MetaBlock, error)
	IsInterfaceNil() bool
}
