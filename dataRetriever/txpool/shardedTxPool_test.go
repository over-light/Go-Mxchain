package txpool

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/stretchr/testify/require"
)

func Test_NewShardedTxPool(t *testing.T) {
	pool, err := newTxPoolToTest()

	require.Nil(t, err)
	require.NotNil(t, pool)
	require.Implements(t, (*dataRetriever.ShardedDataCacherNotifier)(nil), pool)
}

func Test_NewShardedTxPool_WhenBadConfig(t *testing.T) {
	goodArgs := ArgShardedTxPool{Config: storageUnit.CacheConfig{Size: 100, SizePerSender: 10, SizeInBytes: 409600, SizeInBytesPerSender: 40960, Shards: 16}, MinGasPrice: 200000000000, NumberOfShards: 1}

	args := goodArgs
	args.Config.SizeInBytes = 0
	pool, err := NewShardedTxPool(args)
	require.Nil(t, pool)
	require.NotNil(t, err)
	require.Errorf(t, err, dataRetriever.ErrCacheConfigInvalidSizeInBytes.Error())

	args = goodArgs
	args.Config.SizeInBytesPerSender = 0
	pool, err = NewShardedTxPool(args)
	require.Nil(t, pool)
	require.NotNil(t, err)
	require.Errorf(t, err, dataRetriever.ErrCacheConfigInvalidSizeInBytes.Error())

	args = goodArgs
	args.Config.Size = 0
	pool, err = NewShardedTxPool(args)
	require.Nil(t, pool)
	require.NotNil(t, err)
	require.Errorf(t, err, dataRetriever.ErrCacheConfigInvalidSize.Error())

	args = goodArgs
	args.Config.SizePerSender = 0
	pool, err = NewShardedTxPool(args)
	require.Nil(t, pool)
	require.NotNil(t, err)
	require.Errorf(t, err, dataRetriever.ErrCacheConfigInvalidSize.Error())

	args = goodArgs
	args.Config.Shards = 0
	pool, err = NewShardedTxPool(args)
	require.Nil(t, pool)
	require.NotNil(t, err)
	require.Errorf(t, err, dataRetriever.ErrCacheConfigInvalidShards.Error())

	args = goodArgs
	args.MinGasPrice = 0
	pool, err = NewShardedTxPool(args)
	require.Nil(t, pool)
	require.NotNil(t, err)
	require.Errorf(t, err, dataRetriever.ErrCacheConfigInvalidEconomics.Error())

	args = goodArgs
	args.NumberOfShards = 0
	pool, err = NewShardedTxPool(args)
	require.Nil(t, pool)
	require.NotNil(t, err)
	require.Errorf(t, err, dataRetriever.ErrCacheConfigInvalidSharding.Error())
}

func Test_NewShardedTxPool_ComputesCacheConfig(t *testing.T) {
	config := storageUnit.CacheConfig{SizeInBytes: 524288000, SizeInBytesPerSender: 614400, Size: 900000, SizePerSender: 1000, Shards: 1}
	args := ArgShardedTxPool{Config: config, MinGasPrice: 200000000000, NumberOfShards: 5}

	poolAsInterface, err := NewShardedTxPool(args)
	require.Nil(t, err)

	pool := poolAsInterface.(*shardedTxPool)

	require.Equal(t, true, pool.configPrototypeSourceMe.EvictionEnabled)
	require.Equal(t, uint32(291271110), pool.configPrototypeSourceMe.NumBytesThreshold)
	require.Equal(t, uint32(614400), pool.configPrototypeSourceMe.NumBytesPerSenderThreshold)
	require.Equal(t, uint32(1000), pool.configPrototypeSourceMe.CountPerSenderThreshold)
	require.Equal(t, uint32(100), pool.configPrototypeSourceMe.NumSendersToEvictInOneStep)
	require.Equal(t, uint32(200), pool.configPrototypeSourceMe.MinGasPriceNanoErd)
	require.Equal(t, uint32(500000), pool.configPrototypeSourceMe.CountThreshold)

	require.Equal(t, uint32(100000), pool.configPrototypeDestinationMe.MaxNumItems)
	require.Equal(t, uint32(58254222), pool.configPrototypeDestinationMe.MaxNumBytes)
}

func Test_ShardDataStore_Or_GetTxCache(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	pool := poolAsInterface.(*shardedTxPool)

	fooGenericCache := pool.ShardDataStore("foo")
	fooTxCache := pool.getTxCache("foo")
	require.Equal(t, fooGenericCache, fooTxCache)
}

func Test_ShardDataStore_CreatesIfMissingWithoutConcurrencyIssues(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	pool := poolAsInterface.(*shardedTxPool)

	var wg sync.WaitGroup

	// 100 * 10 caches will be created

	for i := 1; i <= 100; i++ {
		wg.Add(1)

		go func(i int) {
			for j := 111; j <= 120; j++ {
				pool.ShardDataStore(fmt.Sprintf("%d_%d", i, j))
			}

			wg.Done()
		}(i)
	}

	wg.Wait()

	require.Equal(t, 1000, len(pool.backingMap))

	for i := 1; i <= 100; i++ {
		for j := 111; j <= 120; j++ {
			_, inMap := pool.backingMap[fmt.Sprintf("%d_%d", i, j)]
			require.True(t, inMap)
		}
	}
}

func Test_AddData(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	pool := poolAsInterface.(*shardedTxPool)
	cache := pool.getTxCache("1")

	pool.AddData([]byte("hash-x"), createTx("alice", 42), "1")
	pool.AddData([]byte("hash-y"), createTx("alice", 43), "1")
	require.Equal(t, 2, cache.Len())

	// Try to add again, duplication does not occur
	pool.AddData([]byte("hash-x"), createTx("alice", 42), "1")
	require.Equal(t, 2, cache.Len())

	_, ok := cache.GetByTxHash([]byte("hash-x"))
	require.True(t, ok)
	_, ok = cache.GetByTxHash([]byte("hash-y"))
	require.True(t, ok)
}

func Test_AddData_NoPanic_IfNotATransaction(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()

	require.NotPanics(t, func() {
		poolAsInterface.AddData([]byte("hash"), &thisIsNotATransaction{}, "1")
	})
}

func Test_AddData_CallsOnAddedHandlers(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	pool := poolAsInterface.(*shardedTxPool)

	numAdded := uint32(0)
	pool.RegisterHandler(func(key []byte, value interface{}) {
		atomic.AddUint32(&numAdded, 1)
	})

	// Second addition is ignored (txhash-based deduplication)
	pool.AddData([]byte("hash-1"), createTx("alice", 42), "1")
	pool.AddData([]byte("hash-1"), createTx("alice", 42), "1")

	waitABit()
	require.Equal(t, uint32(1), atomic.LoadUint32(&numAdded))
}

func Test_SearchFirstData(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	pool := poolAsInterface.(*shardedTxPool)

	tx := createTx("alice", 42)
	pool.AddData([]byte("hash-x"), tx, "1")

	foundTx, ok := pool.SearchFirstData([]byte("hash-x"))
	require.True(t, ok)
	require.Equal(t, tx, foundTx)
}

func Test_RemoveData(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	pool := poolAsInterface.(*shardedTxPool)

	pool.AddData([]byte("hash-x"), createTx("alice", 42), "0")
	pool.AddData([]byte("hash-y"), createTx("bob", 43), "1")

	pool.RemoveData([]byte("hash-x"), "0")
	pool.RemoveData([]byte("hash-y"), "1")
	xTx, xOk := pool.searchFirstTx([]byte("hash-x"))
	yTx, yOk := pool.searchFirstTx([]byte("hash-y"))
	require.False(t, xOk)
	require.False(t, yOk)
	require.Nil(t, xTx)
	require.Nil(t, yTx)
}

func Test_RemoveSetOfDataFromPool(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	pool := poolAsInterface.(*shardedTxPool)
	cache := pool.getTxCache("0")

	pool.AddData([]byte("hash-x"), createTx("alice", 42), "0")
	pool.AddData([]byte("hash-y"), createTx("bob", 43), "0")
	require.Equal(t, 2, cache.Len())

	pool.RemoveSetOfDataFromPool([][]byte{[]byte("hash-x"), []byte("hash-y")}, "0")
	require.Zero(t, cache.Len())
}

func Test_RemoveDataFromAllShards(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	pool := poolAsInterface.(*shardedTxPool)

	pool.AddData([]byte("hash-x"), createTx("alice", 42), "0")
	pool.AddData([]byte("hash-x"), createTx("alice", 42), "1")
	pool.RemoveDataFromAllShards([]byte("hash-x"))

	require.Zero(t, pool.getTxCache("0").Len())
	require.Zero(t, pool.getTxCache("1").Len())
}

func Test_MergeShardStores(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	pool := poolAsInterface.(*shardedTxPool)

	pool.AddData([]byte("hash-x"), createTx("alice", 42), "1_0")
	pool.AddData([]byte("hash-y"), createTx("alice", 43), "2_0")
	pool.MergeShardStores("1_0", "2_0")

	require.Equal(t, 0, pool.getTxCache("1_0").Len())
	require.Equal(t, 2, pool.getTxCache("2_0").Len())
}

func Test_Clear(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	pool := poolAsInterface.(*shardedTxPool)

	pool.AddData([]byte("hash-x"), createTx("alice", 42), "0")
	pool.AddData([]byte("hash-y"), createTx("alice", 43), "1")

	pool.Clear()
	require.Zero(t, pool.getTxCache("0").Len())
	require.Zero(t, pool.getTxCache("1").Len())
}

func Test_ClearShardStore(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	pool := poolAsInterface.(*shardedTxPool)

	pool.AddData([]byte("hash-x"), createTx("alice", 42), "1")
	pool.AddData([]byte("hash-y"), createTx("alice", 43), "1")
	pool.AddData([]byte("hash-z"), createTx("alice", 15), "5")

	pool.ClearShardStore("1")
	require.Equal(t, 0, pool.getTxCache("1").Len())
	require.Equal(t, 1, pool.getTxCache("5").Len())
}

func Test_RegisterHandler(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	pool := poolAsInterface.(*shardedTxPool)

	pool.RegisterHandler(func(key []byte, value interface{}) {})
	require.Equal(t, 1, len(pool.onAddCallbacks))

	pool.RegisterHandler(nil)
	require.Equal(t, 1, len(pool.onAddCallbacks))
}

func Test_GetCounts(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	pool := poolAsInterface.(*shardedTxPool)

	require.Equal(t, int64(0), pool.GetCounts().GetTotal())
	pool.AddData([]byte("hash-x"), createTx("alice", 42), "1")
	pool.AddData([]byte("hash-y"), createTx("alice", 43), "1")
	pool.AddData([]byte("hash-z"), createTx("bob", 15), "3")
	require.Equal(t, int64(3), pool.GetCounts().GetTotal())
	pool.RemoveDataFromAllShards([]byte("hash-x"))
	require.Equal(t, int64(2), pool.GetCounts().GetTotal())
	pool.Clear()
	require.Equal(t, int64(0), pool.GetCounts().GetTotal())
}

func Test_IsInterfaceNil(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	require.False(t, check.IfNil(poolAsInterface))

	makeNil := func() dataRetriever.ShardedDataCacherNotifier {
		return nil
	}

	thisIsNil := makeNil()
	require.True(t, check.IfNil(thisIsNil))
}

func Test_NotImplementedFunctions(t *testing.T) {
	poolAsInterface, _ := newTxPoolToTest()
	pool := poolAsInterface.(*shardedTxPool)

	require.NotPanics(t, func() { pool.CreateShardStore("foo") })
}

func Test_routeToCacheUnions(t *testing.T) {
	config := storageUnit.CacheConfig{Size: 100, SizePerSender: 10, SizeInBytes: 409600, SizeInBytesPerSender: 40960, Shards: 16}
	args := ArgShardedTxPool{Config: config, MinGasPrice: 200000000000, NumberOfShards: 4, SelfShardID: 42}
	poolAsInterface, _ := NewShardedTxPool(args)
	pool := poolAsInterface.(*shardedTxPool)

	require.Equal(t, "42", pool.routeToCacheUnions("42"))
	require.Equal(t, "42", pool.routeToCacheUnions("42_0"))
	require.Equal(t, "42", pool.routeToCacheUnions("42_1"))
	require.Equal(t, "42", pool.routeToCacheUnions("42_2"))
	require.Equal(t, "42", pool.routeToCacheUnions("42_42"))
	require.Equal(t, "2_5", pool.routeToCacheUnions("2_5"))
	require.Equal(t, "foobar", pool.routeToCacheUnions("foobar"))
}

// TODO: Add another test to check the whole allocated space (size, count)
// func Test_getCacheConfig(t *testing.T) {
// 	config := storageUnit.CacheConfig{Size: 150, SizePerSender: 1, SizeInBytes: 61440, SizeInBytesPerSender: 40960, Shards: 16}
// 	args := ArgShardedTxPool{Config: config, MinGasPrice: 200000000000, NumberOfShards: 8, SelfShardID: 4}
// 	poolAsInterface, _ := NewShardedTxPool(args)
// 	pool := poolAsInterface.(*shardedTxPool)

// 	numBytesAccumulator := uint32(0)
// 	countAccumulator := uint32(0)

// 	for i := 0; i < 8; i++ {
// 		cacheConfig := pool.getCacheConfig(fmt.Sprint(i))
// 		numBytesAccumulator += cacheConfig.NumBytesThreshold
// 		countAccumulator += cacheConfig.CountThreshold
// 	}

// 	// Cache configurations are complementary, they use the whole allocated space (size, count)
// 	require.Equal(t, 61440, int(numBytesAccumulator))
// 	require.Equal(t, 150, int(countAccumulator))
// }

func createTx(sender string, nonce uint64) data.TransactionHandler {
	return &transaction.Transaction{
		SndAddr: []byte(sender),
		Nonce:   nonce,
	}
}

func waitABit() {
	time.Sleep(10 * time.Millisecond)
}

type thisIsNotATransaction struct {
}

func newTxPoolToTest() (dataRetriever.ShardedDataCacherNotifier, error) {
	config := storageUnit.CacheConfig{Size: 100, SizePerSender: 10, SizeInBytes: 409600, SizeInBytesPerSender: 40960, Shards: 16}
	args := ArgShardedTxPool{Config: config, MinGasPrice: 200000000000, NumberOfShards: 4}
	return NewShardedTxPool(args)
}
