package sync

import (
	"context"
	"sync"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/update"
)

var _ update.StateSyncer = (*syncState)(nil)

var log = logger.GetOrCreate("update/genesis")

type syncState struct {
	syncingEpoch uint32

	headers      update.HeaderSyncHandler
	tries        update.EpochStartTriesSyncHandler
	miniBlocks   update.EpochStartPendingMiniBlocksSyncHandler
	transactions update.PendingTransactionsSyncHandler
}

// ArgsNewSyncState defines the arguments for the new sync state
type ArgsNewSyncState struct {
	Headers      update.HeaderSyncHandler
	Tries        update.EpochStartTriesSyncHandler
	MiniBlocks   update.EpochStartPendingMiniBlocksSyncHandler
	Transactions update.PendingTransactionsSyncHandler
}

// NewSyncState creates a complete syncer which saves the state of the blockchain with pending values as well
func NewSyncState(args ArgsNewSyncState) (*syncState, error) {
	if check.IfNil(args.Headers) {
		return nil, update.ErrNilHeaderSyncHandler
	}
	if check.IfNil(args.Tries) {
		return nil, update.ErrNilTrieSyncers
	}
	if check.IfNil(args.MiniBlocks) {
		return nil, update.ErrNilMiniBlocksSyncHandler
	}
	if check.IfNil(args.Transactions) {
		return nil, update.ErrNilTransactionsSyncHandler
	}

	ss := &syncState{
		tries:        args.Tries,
		miniBlocks:   args.MiniBlocks,
		transactions: args.Transactions,
		headers:      args.Headers,
		syncingEpoch: 0,
	}

	return ss, nil
}

// SyncAllState gets an epoch number and will sync the complete data for that epoch start metablock
func (ss *syncState) SyncAllState(epoch uint32) error {

	ss.syncingEpoch = epoch
	err := ss.headers.SyncUnFinishedMetaHeaders(epoch)
	if err != nil {
		return err
	}

	meta, err := ss.headers.GetEpochStartMetaBlock()
	if err != nil {
		return err
	}

	unFinished, err := ss.headers.GetUnfinishedMetaBlocks()
	if err != nil {
		return err
	}

	ss.syncingEpoch = meta.GetEpoch()

	wg := sync.WaitGroup{}
	wg.Add(2)

	var errFound error
	mutErr := sync.Mutex{}

	go func() {
		err := ss.tries.SyncTriesFrom(meta, time.Hour)
		if err != nil {
			mutErr.Lock()
			errFound = err
			mutErr.Unlock()
		}
		wg.Done()
	}()

	go func() {
		defer wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
		err := ss.miniBlocks.SyncPendingMiniBlocksFromMeta(meta, unFinished, ctx)
		cancel()
		if err != nil {
			mutErr.Lock()
			errFound = err
			mutErr.Unlock()
			return
		}

		syncedMiniBlocks, err := ss.miniBlocks.GetMiniBlocks()
		if err != nil {
			mutErr.Lock()
			errFound = err
			mutErr.Unlock()
			return
		}

		ctx, cancel = context.WithTimeout(context.Background(), time.Hour)
		err = ss.transactions.SyncPendingTransactionsFor(syncedMiniBlocks, ss.syncingEpoch, ctx)
		cancel()
		if err != nil {
			mutErr.Lock()
			errFound = err
			mutErr.Unlock()
			return
		}
	}()

	// TODO: might think of a way to stop waiting at a signal
	wg.Wait()

	if errFound != nil {
		return errFound
	}

	return nil
}

// GetEpochStartMetaBlock returns the synced metablock
func (ss *syncState) GetEpochStartMetaBlock() (*block.MetaBlock, error) {
	return ss.headers.GetEpochStartMetaBlock()
}

// GetUnfinishedMetaBlocks returns the synced unfinished metablocks
func (ss *syncState) GetUnfinishedMetaBlocks() (map[string]*block.MetaBlock, error) {
	return ss.headers.GetUnfinishedMetaBlocks()
}

// GetAllTries returns the synced tries
func (ss *syncState) GetAllTries() (map[string]data.Trie, error) {
	return ss.tries.GetTries()
}

// GetAllTransactions returns the synced transactions
func (ss *syncState) GetAllTransactions() (map[string]data.TransactionHandler, error) {
	return ss.transactions.GetTransactions()
}

// GetAllMiniBlocks returns the synced miniblocks
func (ss *syncState) GetAllMiniBlocks() (map[string]*block.MiniBlock, error) {
	return ss.miniBlocks.GetMiniBlocks()
}

// IsInterfaceNil returns if underlying objects in nil
func (ss *syncState) IsInterfaceNil() bool {
	return ss == nil
}
