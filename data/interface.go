package data

import (
	"context"
	"math/big"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
)

// TriePruningIdentifier is the type for trie pruning identifiers
type TriePruningIdentifier byte

const (
	// OldRoot is appended to the key when oldHashes are added to the evictionWaitingList
	OldRoot TriePruningIdentifier = 0
	// NewRoot is appended to the key when newHashes are added to the evictionWaitingList
	NewRoot TriePruningIdentifier = 1
)

// ModifiedHashes is used to memorize all old hashes and new hashes from when a trie is committed
type ModifiedHashes map[string]struct{}

// Clone is used to create a clone of the map
func (mh ModifiedHashes) Clone() ModifiedHashes {
	newMap := make(ModifiedHashes)
	for key := range mh {
		newMap[key] = struct{}{}
	}

	return newMap
}

// HeaderHandler defines getters and setters for header data holder
type HeaderHandler interface {
	GetShardID() uint32
	GetNonce() uint64
	GetEpoch() uint32
	GetRound() uint64
	GetRootHash() []byte
	GetValidatorStatsRootHash() []byte
	GetPrevHash() []byte
	GetPrevRandSeed() []byte
	GetRandSeed() []byte
	GetPubKeysBitmap() []byte
	GetSignature() []byte
	GetLeaderSignature() []byte
	GetChainID() []byte
	GetSoftwareVersion() []byte
	GetTimeStamp() uint64
	GetTxCount() uint32
	GetReceiptsHash() []byte
	GetAccumulatedFees() *big.Int
	GetDeveloperFees() *big.Int
	GetEpochStartMetaHash() []byte
	GetReserved() []byte

	SetAccumulatedFees(value *big.Int)
	SetDeveloperFees(value *big.Int)
	SetShardID(shId uint32)
	SetNonce(n uint64)
	SetEpoch(e uint32)
	SetRound(r uint64)
	SetTimeStamp(ts uint64)
	SetRootHash(rHash []byte)
	SetValidatorStatsRootHash(rHash []byte)
	SetPrevHash(pvHash []byte)
	SetPrevRandSeed(pvRandSeed []byte)
	SetRandSeed(randSeed []byte)
	SetPubKeysBitmap(pkbm []byte)
	SetSignature(sg []byte)
	SetLeaderSignature(sg []byte)
	SetChainID(chainID []byte)
	SetSoftwareVersion(version []byte)
	SetTxCount(txCount uint32)

	IsStartOfEpochBlock() bool
	GetMiniBlockHeadersWithDst(destId uint32) map[string]uint32
	GetOrderedCrossMiniblocksWithDst(destId uint32) []*MiniBlockInfo
	GetMiniBlockHeadersHashes() [][]byte

	IsInterfaceNil() bool
	Clone() HeaderHandler
}

// BodyHandler interface for a block body
type BodyHandler interface {
	Clone() BodyHandler
	// IntegrityAndValidity checks the integrity and validity of the block
	IntegrityAndValidity() error
	// IsInterfaceNil returns true if there is no value under the interface
	IsInterfaceNil() bool
}

// ChainHandler is the interface defining the functionality a blockchain should implement
type ChainHandler interface {
	GetGenesisHeader() HeaderHandler
	SetGenesisHeader(gb HeaderHandler) error
	GetGenesisHeaderHash() []byte
	SetGenesisHeaderHash(hash []byte)
	GetCurrentBlockHeader() HeaderHandler
	SetCurrentBlockHeader(bh HeaderHandler) error
	GetCurrentBlockHeaderHash() []byte
	SetCurrentBlockHeaderHash(hash []byte)
	IsInterfaceNil() bool
	CreateNewHeader() HeaderHandler
}

// TransactionHandler defines the type of executable transaction
type TransactionHandler interface {
	IsInterfaceNil() bool

	GetValue() *big.Int
	GetNonce() uint64
	GetData() []byte
	GetRcvAddr() []byte
	GetRcvUserName() []byte
	GetSndAddr() []byte
	GetGasLimit() uint64
	GetGasPrice() uint64

	SetValue(*big.Int)
	SetData([]byte)
	SetRcvAddr([]byte)
	SetSndAddr([]byte)
	Size() int

	CheckIntegrity() error
}

// LogHandler defines the type for a log resulted from executing a transaction or smart contract call
type LogHandler interface {
	// GetAddress returns the address of the sc that was originally called by the user
	GetAddress() []byte
	// GetLogEvents returns the events from a transaction log entry
	GetLogEvents() []EventHandler

	IsInterfaceNil() bool
}

// EventHandler defines the type for an event resulted from a smart contract call contained in a log
type EventHandler interface {
	// GetAddress returns the address of the contract that generated this event
	//  - in sc calling another sc situation this will differ from the
	//    LogHandler's GetAddress, whereas in the single sc situation
	//    they will be the same
	GetAddress() []byte
	// GetIdentifier returns identifier of the event, that together with the ABI can
	//   be used to understand the type of the event by other applications
	GetIdentifier() []byte
	// GetTopics returns the data that can be indexed so that it would be searchable
	//  by other applications
	GetTopics() [][]byte
	// GetData returns the rest of the event data, which will not be indexed, so storing
	//  information here should be cheaper
	GetData() []byte

	IsInterfaceNil() bool
}

// NumNodesDTO represents the DTO structure that will hold the number of nodes split by category and other
// trie structure relevant data such as maximum number of trie levels including the roothash node and all leaves
type NumNodesDTO struct {
	Leaves     int
	Extensions int
	Branches   int
	MaxLevel   int
}

//Trie is an interface for Merkle Trees implementations
type Trie interface {
	Get(key []byte) ([]byte, error)
	Update(key, value []byte) error
	Delete(key []byte) error
	RootHash() ([]byte, error)
	Commit() error
	Recreate(root []byte) (Trie, error)
	String() string
	GetObsoleteHashes() [][]byte
	GetDirtyHashes() (ModifiedHashes, error)
	GetOldRoot() []byte
	GetSerializedNodes([]byte, uint64) ([][]byte, uint64, error)
	GetSerializedNode([]byte) ([]byte, error)
	GetNumNodes() NumNodesDTO
	GetAllLeavesOnChannel(rootHash []byte) (chan core.KeyValueHolder, error)
	GetAllHashes() ([][]byte, error)
	GetProof(key []byte) ([][]byte, error)
	VerifyProof(key []byte, proof [][]byte) (bool, error)
	GetStorageManager() StorageManager
	Close() error
	IsInterfaceNil() bool
}

// DBWriteCacher is used to cache changes made to the trie, and only write to the database when it's needed
type DBWriteCacher interface {
	Put(key, val []byte) error
	Get(key []byte) ([]byte, error)
	Remove(key []byte) error
	Close() error
	IsInterfaceNil() bool
}

// TrieSyncer synchronizes the trie, asking on the network for the missing nodes
type TrieSyncer interface {
	StartSyncing(rootHash []byte, ctx context.Context) error
	IsInterfaceNil() bool
}

// StorageManager manages all trie storage operations
type StorageManager interface {
	Database() DBWriteCacher
	TakeSnapshot([]byte, bool, chan core.KeyValueHolder)
	SetCheckpoint([]byte, chan core.KeyValueHolder)
	GetSnapshotThatContainsHash(rootHash []byte) SnapshotDbHandler
	IsPruningEnabled() bool
	IsPruningBlocked() bool
	EnterPruningBufferingMode()
	ExitPruningBufferingMode()
	GetSnapshotDbBatchDelay() int
	AddDirtyCheckpointHashes([]byte, ModifiedHashes) bool
	Remove(hash []byte) error
	Close() error
	IsInterfaceNil() bool
}

// TrieCreateArgs holds arguments for calling the Create method on the TrieFactory
type TrieCreateArgs struct {
	TrieStorageConfig  config.StorageConfig
	ShardID            string
	PruningEnabled     bool
	CheckpointsEnabled bool
	MaxTrieLevelInMem  uint
}

// TrieFactory creates new tries
type TrieFactory interface {
	Create(TrieCreateArgs) (StorageManager, Trie, error)
	IsInterfaceNil() bool
}

// ValidatorInfoHandler is used to store multiple validatorInfo properties
type ValidatorInfoHandler interface {
	GetPublicKey() []byte
	GetShardId() uint32
	GetList() string
	GetIndex() uint32
	GetTempRating() uint32
	GetRating() uint32
	String() string
	IsInterfaceNil() bool
}

// ShardValidatorInfoHandler is used to store multiple validatorInfo properties required in shards
type ShardValidatorInfoHandler interface {
	GetPublicKey() []byte
	GetTempRating() uint32
	String() string
	IsInterfaceNil() bool
}

// GoRoutineThrottler can monitor the number of the currently running go routines
type GoRoutineThrottler interface {
	CanProcess() bool
	StartProcessing()
	EndProcessing()
	IsInterfaceNil() bool
}

// SnapshotDbHandler is used to keep track of how many references a snapshot db has
type SnapshotDbHandler interface {
	DBWriteCacher
	IsInUse() bool
	DecreaseNumReferences()
	IncreaseNumReferences()
	MarkForRemoval()
	MarkForDisconnection()
	SetPath(string)
}

// MiniBlockInfo holds information about a cross miniblock referenced in a received block
type MiniBlockInfo struct {
	Hash          []byte
	SenderShardID uint32
	Round         uint64
}

// SyncStatisticsHandler defines the methods for a component able to store the sync statistics for a trie
type SyncStatisticsHandler interface {
	Reset()
	AddNumReceived(value int)
	AddNumLarge(value int)
	SetNumMissing(rootHash []byte, value int)
	NumReceived() int
	NumLarge() int
	NumMissing() int
	IsInterfaceNil() bool
}

// CheckpointHashesHolder is used to hold the hashes that need to be committed in the future state checkpoint
type CheckpointHashesHolder interface {
	Put(rootHash []byte, hashes ModifiedHashes) bool
	RemoveCommitted(lastCommittedRootHash []byte)
	Remove(hash []byte)
	ShouldCommit(hash []byte) bool
	IsInterfaceNil() bool
}
