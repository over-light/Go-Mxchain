package txpool

import (
	"strconv"
	"sync"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core/counting"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/txcache"
)

var _ counting.Countable = (*shardedTxPool)(nil)
var _ dataRetriever.ShardedDataCacherNotifier = (*shardedTxPool)(nil)

var log = logger.GetOrCreate("txpool")

// shardedTxPool holds transaction caches organised by destination shard
type shardedTxPool struct {
	mutexBackingMap                  sync.RWMutex
	backingMap                       map[string]*txPoolShard
	mutexAddCallbacks                sync.RWMutex
	onAddCallbacks                   []func(key []byte, value interface{})
	cacheConfigPrototype             txcache.CacheConfig
	cacheConfigPrototypeForSelfShard txcache.CacheConfig
	selfShardID                      uint32
}

type txPoolShard struct {
	CacheID string
	Cache   txCache
}

// NewShardedTxPool creates a new sharded tx pool
// Implements "dataRetriever.TxPool"
func NewShardedTxPool(args ArgShardedTxPool) (dataRetriever.ShardedDataCacherNotifier, error) {
	log.Info("NewShardedTxPool", "args", args)

	err := args.verify()
	if err != nil {
		return nil, err
	}

	const oneBillion = 1000000 * 1000
	numCaches := 2*args.NumberOfShards - 1

	cacheConfigPrototype := txcache.CacheConfig{
		NumChunksHint:              args.Config.Shards,
		EvictionEnabled:            true,
		NumBytesThreshold:          args.Config.SizeInBytes / numCaches,
		NumBytesPerSenderThreshold: args.Config.SizeInBytesPerSender,
		CountThreshold:             args.Config.Size / numCaches,
		CountPerSenderThreshold:    args.Config.SizePerSender,
		NumSendersToEvictInOneStep: dataRetriever.TxPoolNumSendersToEvictInOneStep,
		MinGasPriceNanoErd:         uint32(args.MinGasPrice / oneBillion),
	}

	cacheConfigPrototypeForSelfShard := cacheConfigPrototype
	cacheConfigPrototypeForSelfShard.CountThreshold *= args.NumberOfShards
	cacheConfigPrototypeForSelfShard.NumBytesThreshold *= args.NumberOfShards

	shardedTxPoolObject := &shardedTxPool{
		mutexBackingMap:                  sync.RWMutex{},
		backingMap:                       make(map[string]*txPoolShard),
		mutexAddCallbacks:                sync.RWMutex{},
		onAddCallbacks:                   make([]func(key []byte, value interface{}), 0),
		cacheConfigPrototype:             cacheConfigPrototype,
		cacheConfigPrototypeForSelfShard: cacheConfigPrototypeForSelfShard,
		selfShardID:                      args.SelfShardID,
	}

	return shardedTxPoolObject, nil
}

// ShardDataStore returns the requested cache, as the generic Cacher interface
func (txPool *shardedTxPool) ShardDataStore(cacheID string) storage.Cacher {
	cache := txPool.getTxCache(cacheID)
	return cache
}

// getTxCache returns the requested cache
func (txPool *shardedTxPool) getTxCache(cacheID string) txCache {
	shard := txPool.getOrCreateShard(cacheID)
	return shard.Cache
}

// TODO: Perhaps create all caches in constructor?
func (txPool *shardedTxPool) getOrCreateShard(cacheID string) *txPoolShard {
	cacheID = txPool.routeToCacheUnions(cacheID)

	txPool.mutexBackingMap.RLock()
	shard, ok := txPool.backingMap[cacheID]
	txPool.mutexBackingMap.RUnlock()

	if ok {
		return shard
	}

	shard = txPool.createShard(cacheID)
	return shard
}

func (txPool *shardedTxPool) createShard(cacheID string) *txPoolShard {
	txPool.mutexBackingMap.Lock()
	defer txPool.mutexBackingMap.Unlock()

	shard, ok := txPool.backingMap[cacheID]
	if !ok {
		cache := txPool.createTxCache(cacheID)
		shard = &txPoolShard{
			CacheID: cacheID,
			Cache:   cache,
		}

		txPool.backingMap[cacheID] = shard
	}

	return shard
}

func (txPool *shardedTxPool) createTxCache(cacheID string) txCache {
	cacheConfig := txPool.getCacheConfig(cacheID)
	cache, err := txcache.NewTxCache(cacheConfig)
	if err != nil {
		log.Error("shardedTxPool.createTxCache()", "err", err)
		return txcache.NewDisabledCache()
	}

	return cache
}

func (txPool *shardedTxPool) getCacheConfig(cacheID string) txcache.CacheConfig {
	var cacheConfig txcache.CacheConfig

	isForSelfShard := process.IsShardCacherIdentifierIntraShard(cacheID, txPool.selfShardID)
	if isForSelfShard {
		cacheConfig = txPool.cacheConfigPrototypeForSelfShard
	} else {
		cacheConfig = txPool.cacheConfigPrototype
	}

	cacheConfig.Name = cacheID
	return cacheConfig
}

// AddData adds the transaction to the cache
func (txPool *shardedTxPool) AddData(key []byte, value interface{}, cacheID string) {
	valueAsTransaction, ok := value.(data.TransactionHandler)
	if !ok {
		return
	}

	sourceShardID, destinationShardID, err := process.ParseShardCacherIdentifier(cacheID)
	if err != nil {
		log.Error("shardedTxPool.AddData()", "err", err)
		return
	}

	wrapper := &txcache.WrappedTransaction{
		Tx:              valueAsTransaction,
		TxHash:          key,
		SenderShardID:   sourceShardID,
		ReceiverShardID: destinationShardID,
	}

	txPool.addTx(wrapper, cacheID)
}

// addTx adds the transaction to the cache
func (txPool *shardedTxPool) addTx(tx *txcache.WrappedTransaction, cacheID string) {
	shard := txPool.getOrCreateShard(cacheID)
	cache := shard.Cache
	_, added := cache.AddTx(tx)
	if added {
		txPool.onAdded(tx.TxHash, tx)
	}
}

func (txPool *shardedTxPool) onAdded(key []byte, value interface{}) {
	txPool.mutexAddCallbacks.RLock()
	defer txPool.mutexAddCallbacks.RUnlock()

	for _, handler := range txPool.onAddCallbacks {
		go handler(key, value)
	}
}

// SearchFirstData searches the transaction against all shard data store, retrieving the first found
func (txPool *shardedTxPool) SearchFirstData(key []byte) (interface{}, bool) {
	tx, ok := txPool.searchFirstTx(key)
	return tx, ok
}

// searchFirstTx searches the transaction against all shard data store, retrieving the first found
func (txPool *shardedTxPool) searchFirstTx(txHash []byte) (tx data.TransactionHandler, ok bool) {
	txPool.mutexBackingMap.RLock()
	defer txPool.mutexBackingMap.RUnlock()

	var txFromCache *txcache.WrappedTransaction
	var hashExists bool
	for _, shard := range txPool.backingMap {
		txFromCache, hashExists = shard.Cache.GetByTxHash(txHash)
		if hashExists {
			return txFromCache.Tx, true
		}
	}

	return nil, false
}

// RemoveData removes the transaction from the pool
func (txPool *shardedTxPool) RemoveData(key []byte, cacheID string) {
	txPool.removeTx(key, cacheID)
}

// removeTx removes the transaction from the pool
func (txPool *shardedTxPool) removeTx(txHash []byte, cacheID string) {
	shard := txPool.getOrCreateShard(cacheID)
	_ = shard.Cache.RemoveTxByHash(txHash)
}

// RemoveSetOfDataFromPool removes a bunch of transactions from the pool
func (txPool *shardedTxPool) RemoveSetOfDataFromPool(keys [][]byte, cacheID string) {
	txPool.removeTxBulk(keys, cacheID)
}

// removeTxBulk removes a bunch of transactions from the pool
func (txPool *shardedTxPool) removeTxBulk(txHashes [][]byte, cacheID string) {
	for _, key := range txHashes {
		txPool.removeTx(key, cacheID)
	}
}

// RemoveDataFromAllShards removes the transaction from the pool (it searches in all shards)
func (txPool *shardedTxPool) RemoveDataFromAllShards(key []byte) {
	txPool.removeTxFromAllShards(key)
}

// removeTxFromAllShards removes the transaction from the pool (it searches in all shards)
func (txPool *shardedTxPool) removeTxFromAllShards(txHash []byte) {
	txPool.mutexBackingMap.RLock()
	defer txPool.mutexBackingMap.RUnlock()

	for _, shard := range txPool.backingMap {
		cache := shard.Cache
		_ = cache.RemoveTxByHash(txHash)
	}
}

// MergeShardStores merges two shards of the pool
func (txPool *shardedTxPool) MergeShardStores(sourceCacheID, destCacheID string) {
	sourceCacheID = txPool.routeToCacheUnions(sourceCacheID)
	destCacheID = txPool.routeToCacheUnions(destCacheID)

	sourceShard := txPool.getOrCreateShard(sourceCacheID)
	sourceCache := sourceShard.Cache

	sourceCache.ForEachTransaction(func(txHash []byte, tx *txcache.WrappedTransaction) {
		txPool.addTx(tx, destCacheID)
	})

	txPool.mutexBackingMap.Lock()
	delete(txPool.backingMap, sourceCacheID)
	txPool.mutexBackingMap.Unlock()
}

// Clear clears everything in the pool
func (txPool *shardedTxPool) Clear() {
	txPool.mutexBackingMap.Lock()
	txPool.backingMap = make(map[string]*txPoolShard)
	txPool.mutexBackingMap.Unlock()
}

// ClearShardStore clears a specific cache
func (txPool *shardedTxPool) ClearShardStore(cacheID string) {
	shard := txPool.getOrCreateShard(cacheID)
	shard.Cache.Clear()
}

// CreateShardStore is not implemented for this pool, since shard creations is managed internally
func (txPool *shardedTxPool) CreateShardStore(_ string) {
}

// RegisterHandler registers a new handler to be called when a new transaction is added
func (txPool *shardedTxPool) RegisterHandler(handler func(key []byte, value interface{})) {
	if handler == nil {
		log.Error("attempt to register a nil handler")
		return
	}

	txPool.mutexAddCallbacks.Lock()
	txPool.onAddCallbacks = append(txPool.onAddCallbacks, handler)
	txPool.mutexAddCallbacks.Unlock()
}

// GetCounts returns the total number of transactions in the pool
func (txPool *shardedTxPool) GetCounts() counting.Counts {
	txPool.mutexBackingMap.RLock()
	defer txPool.mutexBackingMap.RUnlock()

	counts := counting.NewShardedCounts()

	for cacheID, shard := range txPool.backingMap {
		cache := shard.Cache
		counts.PutCounts(cacheID, int64(cache.CountTx()))
	}

	return counts
}

// IsInterfaceNil returns true if there is no value under the interface
func (txPool *shardedTxPool) IsInterfaceNil() bool {
	return txPool == nil
}

func (txPool *shardedTxPool) routeToCacheUnions(cacheID string) string {
	sourceShardID, _, err := process.ParseShardCacherIdentifier(cacheID)
	if err != nil {
		return cacheID
	}

	if sourceShardID == txPool.selfShardID {
		return strconv.Itoa(int(sourceShardID))
	}

	return cacheID
}
