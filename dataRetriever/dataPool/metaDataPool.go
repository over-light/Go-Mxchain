package dataPool

import (
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/storage"
)

type metaDataPool struct {
	metaBlocks           storage.Cacher
	miniBlocks           storage.Cacher
	shardHeaders         storage.Cacher
	headersNonces        dataRetriever.Uint64SyncMapCacher
	transactions         dataRetriever.ShardedDataCacherNotifier
	unsignedTransactions dataRetriever.ShardedDataCacherNotifier
	currBlockTxs         dataRetriever.TxsForCurrentBlockHandler
}

// NewMetaDataPool creates a data pools holder object
func NewMetaDataPool(
	metaBlocks storage.Cacher,
	miniBlocks storage.Cacher,
	shardHeaders storage.Cacher,
	headersNonces dataRetriever.Uint64SyncMapCacher,
	transactions dataRetriever.ShardedDataCacherNotifier,
	unsignedTransactions dataRetriever.ShardedDataCacherNotifier,
	currBlockTxs dataRetriever.TxsForCurrentBlockHandler,
) (*metaDataPool, error) {

	if metaBlocks == nil || metaBlocks.IsInterfaceNil() {
		return nil, dataRetriever.ErrNilMetaBlockPool
	}
	if miniBlocks == nil || miniBlocks.IsInterfaceNil() {
		return nil, dataRetriever.ErrNilMiniBlockHashesPool
	}
	if shardHeaders == nil || shardHeaders.IsInterfaceNil() {
		return nil, dataRetriever.ErrNilShardHeaderPool
	}
	if headersNonces == nil || headersNonces.IsInterfaceNil() {
		return nil, dataRetriever.ErrNilMetaBlockNoncesPool
	}
	if transactions == nil || transactions.IsInterfaceNil() {
		return nil, dataRetriever.ErrNilTxDataPool
	}
	if unsignedTransactions == nil || unsignedTransactions.IsInterfaceNil() {
		return nil, dataRetriever.ErrNilUnsignedTransactionPool
	}
	if currBlockTxs == nil || currBlockTxs.IsInterfaceNil() {
		return nil, dataRetriever.ErrNilCurrBlockTxs
	}

	return &metaDataPool{
		metaBlocks:           metaBlocks,
		miniBlocks:           miniBlocks,
		shardHeaders:         shardHeaders,
		headersNonces:        headersNonces,
		transactions:         transactions,
		unsignedTransactions: unsignedTransactions,
	}, nil
}

// CurrentBlockTxs returns the holder for current block transactions
func (mdp *metaDataPool) CurrentBlocksTxs() dataRetriever.TxsForCurrentBlockHandler {
	return mdp.currBlockTxs
}

// MetaBlocks returns the holder for meta blocks
func (mdp *metaDataPool) MetaBlocks() storage.Cacher {
	return mdp.metaBlocks
}

// MiniBlocks returns the holder for meta mini block hashes
func (mdp *metaDataPool) MiniBlocks() storage.Cacher {
	return mdp.miniBlocks
}

// ShardHeaders returns the holder for shard headers
func (mdp *metaDataPool) ShardHeaders() storage.Cacher {
	return mdp.shardHeaders
}

// HeadersNonces returns the holder nonce-block hash pairs. It will hold both shard headers nonce-hash pairs
// also metachain header nonce-hash pairs
func (mdp *metaDataPool) HeadersNonces() dataRetriever.Uint64SyncMapCacher {
	return mdp.headersNonces
}

// Transactions returns the holder for transactions which interact with the metachain
func (mdp *metaDataPool) Transactions() dataRetriever.ShardedDataCacherNotifier {
	return mdp.transactions
}

// UnsignedTransactions returns the holder for unsigned transactions which are generated by the metachain
func (mdp *metaDataPool) UnsignedTransactions() dataRetriever.ShardedDataCacherNotifier {
	return mdp.unsignedTransactions
}

// IsInterfaceNil returns true if there is no value under the interface
func (mdp *metaDataPool) IsInterfaceNil() bool {
	if mdp == nil {
		return true
	}
	return false
}
