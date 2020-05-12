package update

import (
	"context"
	"time"

	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/process"
)

// StateSyncer interface defines the methods needed to sync and get all states
type StateSyncer interface {
	GetEpochStartMetaBlock() (*block.MetaBlock, error)
	GetUnfinishedMetaBlocks() (map[string]*block.MetaBlock, error)
	SyncAllState(epoch uint32) error
	GetAllTries() (map[string]data.Trie, error)
	GetAllTransactions() (map[string]data.TransactionHandler, error)
	GetAllMiniBlocks() (map[string]*block.MiniBlock, error)
	IsInterfaceNil() bool
}

// TrieSyncer synchronizes the trie, asking on the network for the missing nodes
type TrieSyncer interface {
	StartSyncing(rootHash []byte, ctx context.Context) error
	Trie() data.Trie
	IsInterfaceNil() bool
}

// TrieSyncContainer keep a list of TrieSyncer
type TrieSyncContainer interface {
	Get(key string) (TrieSyncer, error)
	Add(key string, val TrieSyncer) error
	AddMultiple(keys []string, interceptors []TrieSyncer) error
	Replace(key string, val TrieSyncer) error
	Remove(key string)
	Len() int
	IsInterfaceNil() bool
}

// EpochStartVerifier defines the functionality needed by sync all state from epochTrigger
type EpochStartVerifier interface {
	IsEpochStart() bool
	Epoch() uint32
	EpochStartMetaHdrHash() []byte
	IsInterfaceNil() bool
}

// HistoryStorer provides storage services in a two layered storage construct, where the first layer is
// represented by a cache and second layer by a persitent storage (DB-like)
type HistoryStorer interface {
	Put(key, data []byte) error
	Get(key []byte) ([]byte, error)
	Has(key []byte) error
	Remove(key []byte) error
	ClearCache()
	DestroyUnit() error
	GetFromEpoch(key []byte, epoch uint32) ([]byte, error)
	HasInEpoch(key []byte, epoch uint32) error

	IsInterfaceNil() bool
}

// MultiFileWriter writes several files in a buffered manner
type MultiFileWriter interface {
	NewFile(name string) error
	Write(fileName string, key string, value []byte) error
	Finish()
	IsInterfaceNil() bool
}

// MultiFileReader reads data from several files in a buffered way
type MultiFileReader interface {
	GetFileNames() []string
	ReadNextItem(fileName string) (string, []byte, error)
	Finish()
	IsInterfaceNil() bool
}

// RequestHandler defines the methods through which request to data can be made
type RequestHandler interface {
	RequestTransaction(shardId uint32, txHashes [][]byte)
	RequestUnsignedTransactions(destShardID uint32, scrHashes [][]byte)
	RequestRewardTransactions(destShardID uint32, txHashes [][]byte)
	RequestMiniBlock(shardId uint32, miniblockHash []byte)
	RequestStartOfEpochMetaBlock(epoch uint32)
	RequestShardHeader(shardId uint32, hash []byte)
	RequestMetaHeader(hash []byte)
	RequestMetaHeaderByNonce(nonce uint64)
	RequestShardHeaderByNonce(shardId uint32, nonce uint64)
	RequestTrieNodes(destShardID uint32, hashes [][]byte, topic string)
	RequestInterval() time.Duration
	SetNumPeersToQuery(key string, intra int, cross int) error
	GetNumPeersToQuery(key string) (int, int, error)
	IsInterfaceNil() bool
}

// ExportHandler defines the methods to export the current state of the blockchain
type ExportHandler interface {
	ExportAll(epoch uint32) error
	IsInterfaceNil() bool
}

// ImportHandler defines the methods to import the full state of the blockchain
type ImportHandler interface {
	ImportAll() error
	GetValidatorAccountsDB() state.AccountsAdapter
	GetMiniBlocks() map[string]*block.MiniBlock
	GetHardForkMetaBlock() *block.MetaBlock
	GetTransactions() map[string]data.TransactionHandler
	GetAccountsDBForShard(shardID uint32) state.AccountsAdapter
	IsInterfaceNil() bool
}

// HardForkBlockProcessor defines the methods to process after hardfork
type HardForkBlockProcessor interface {
	CreateNewBlock(chainID string, round uint64, nonce uint64, epoch uint32) (data.HeaderHandler, data.BodyHandler, error)
	IsInterfaceNil() bool
}

// PendingTransactionProcessor defines the methods to process a transaction destination me
type PendingTransactionProcessor interface {
	ProcessTransactionsDstMe(mapTxs map[string]data.TransactionHandler) (block.MiniBlockSlice, error)
	RootHash() ([]byte, error)
	IsInterfaceNil() bool
}

// HeaderSyncHandler defines the methods to sync and get the epoch start metablock
type HeaderSyncHandler interface {
	SyncUnFinishedMetaHeaders(epoch uint32) error
	GetEpochStartMetaBlock() (*block.MetaBlock, error)
	GetUnfinishedMetaBlocks() (map[string]*block.MetaBlock, error)
	IsInterfaceNil() bool
}

// EpochStartTriesSyncHandler defines the methods to sync all tries from a given epoch start metablock
type EpochStartTriesSyncHandler interface {
	SyncTriesFrom(meta *block.MetaBlock, waitTime time.Duration) error
	GetTries() (map[string]data.Trie, error)
	IsInterfaceNil() bool
}

// EpochStartPendingMiniBlocksSyncHandler defines the methods to sync all pending miniblocks
type EpochStartPendingMiniBlocksSyncHandler interface {
	SyncPendingMiniBlocksFromMeta(epochStart *block.MetaBlock, unFinished map[string]*block.MetaBlock, ctx context.Context) error
	GetMiniBlocks() (map[string]*block.MiniBlock, error)
	IsInterfaceNil() bool
}

// PendingTransactionsSyncHandler defines the methods to sync all transactions from a set of miniblocks
type PendingTransactionsSyncHandler interface {
	SyncPendingTransactionsFor(miniBlocks map[string]*block.MiniBlock, epoch uint32, ctx context.Context) error
	GetTransactions() (map[string]data.TransactionHandler, error)
	IsInterfaceNil() bool
}

// MissingHeadersByHashSyncer defines the methods to sync all missing headers by hash
type MissingHeadersByHashSyncer interface {
	SyncMissingHeadersByHash(shardIDs []uint32, headersHashes [][]byte, ctx context.Context) error
	GetHeaders() (map[string]data.HeaderHandler, error)
	ClearFields()
	IsInterfaceNil() bool
}

// DataWriter defines the methods to write data
type DataWriter interface {
	WriteString(s string) (int, error)
	Flush() error
}

// DataReader defines the methods to read data
type DataReader interface {
	Text() string
	Scan() bool
	Err() error
}

// WhiteListHandler is the interface needed to add whitelisted data
type WhiteListHandler interface {
	Remove(keys [][]byte)
	Add(keys [][]byte)
	IsWhiteListed(interceptedData process.InterceptedData) bool
	IsInterfaceNil() bool
}

// AccountsDBSyncer defines the methods for the accounts db syncer
type AccountsDBSyncer interface {
	GetSyncedTries() map[string]data.Trie
	SyncAccounts(rootHash []byte) error
	IsInterfaceNil() bool
}

// AccountsDBSyncContainer keep a list of TrieSyncer
type AccountsDBSyncContainer interface {
	Get(key string) (AccountsDBSyncer, error)
	Add(key string, val AccountsDBSyncer) error
	AddMultiple(keys []string, values []AccountsDBSyncer) error
	Replace(key string, val AccountsDBSyncer) error
	Remove(key string)
	Len() int
	IsInterfaceNil() bool
}

// SigVerifier is used to verify the signature on a provided message
type SigVerifier interface {
	Verify(message []byte, sig []byte, pk []byte) error
	IsInterfaceNil() bool
}
