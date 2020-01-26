package txcache

import (
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/stretchr/testify/require"
)

func Test_AddTx(t *testing.T) {
	cache := NewTxCache(1)

	tx := createTx("alice", 1)

	ok, added := cache.AddTx([]byte("hash-1"), tx)
	require.True(t, ok)
	require.True(t, added)

	// Add it again (no-operation)
	ok, added = cache.AddTx([]byte("hash-1"), tx)
	require.True(t, ok)
	require.False(t, added)

	foundTx, ok := cache.GetByTxHash([]byte("hash-1"))
	require.True(t, ok)
	require.Equal(t, tx, foundTx)
}

func Test_AddNilTx_DoesNothing(t *testing.T) {
	cache := NewTxCache(1)

	txHash := []byte("hash-1")

	ok, added := cache.AddTx(txHash, nil)
	require.False(t, ok)
	require.False(t, added)

	foundTx, ok := cache.GetByTxHash(txHash)
	require.False(t, ok)
	require.Nil(t, foundTx)
}

func Test_RemoveByTxHash(t *testing.T) {
	cache := NewTxCache(16)

	cache.AddTx([]byte("hash-1"), createTx("alice", 1))
	cache.AddTx([]byte("hash-2"), createTx("alice", 2))

	err := cache.RemoveTxByHash([]byte("hash-1"))
	require.Nil(t, err)
	cache.Remove([]byte("hash-2"))

	foundTx, ok := cache.GetByTxHash([]byte("hash-1"))
	require.False(t, ok)
	require.Nil(t, foundTx)

	foundTx, ok = cache.GetByTxHash([]byte("hash-2"))
	require.False(t, ok)
	require.Nil(t, foundTx)
}

func Test_CountTx_And_Len(t *testing.T) {
	cache := NewTxCache(1)

	cache.AddTx([]byte("hash-1"), createTx("alice", 1))
	cache.AddTx([]byte("hash-2"), createTx("alice", 2))
	cache.AddTx([]byte("hash-3"), createTx("alice", 3))

	require.Equal(t, int64(3), cache.CountTx())
	require.Equal(t, 3, cache.Len())
}

func Test_GetByTxHash_And_Peek_And_Get(t *testing.T) {
	cache := NewTxCache(1)

	txHash := []byte("hash-1")
	tx := createTx("alice", 1)
	cache.AddTx(txHash, tx)

	foundTx, ok := cache.GetByTxHash(txHash)
	require.True(t, ok)
	require.Equal(t, tx, foundTx)

	foundTxPeek, okPeek := cache.Peek(txHash)
	require.True(t, okPeek)
	require.Equal(t, tx, foundTxPeek)

	foundTxGet, okGet := cache.Get(txHash)
	require.True(t, okGet)
	require.Equal(t, tx, foundTxGet)
}

func Test_RemoveByTxHash_Error_WhenMissing(t *testing.T) {
	cache := NewTxCache(16)
	err := cache.RemoveTxByHash([]byte("missing"))
	require.Equal(t, err, ErrTxNotFound)
}

func Test_RemoveByTxHash_Error_WhenMapsInconsistency(t *testing.T) {
	cache := NewTxCache(16)

	txHash := []byte("hash-1")
	tx := createTx("alice", 1)
	cache.AddTx(txHash, tx)

	// Cause an inconsistency between the two internal maps (theoretically possible in case of misbehaving eviction)
	cache.txListBySender.removeTx(tx)

	err := cache.RemoveTxByHash(txHash)
	require.Equal(t, err, ErrMapsSyncInconsistency)
}

func Test_Clear(t *testing.T) {
	cache := NewTxCache(1)

	cache.AddTx([]byte("hash-alice-1"), createTx("alice", 1))
	cache.AddTx([]byte("hash-bob-7"), createTx("bob", 7))
	cache.AddTx([]byte("hash-alice-42"), createTx("alice", 42))
	require.Equal(t, int64(3), cache.CountTx())

	cache.Clear()
	require.Equal(t, int64(0), cache.CountTx())
}

func Test_ForEachTransaction(t *testing.T) {
	cache := NewTxCache(1)

	cache.AddTx([]byte("hash-alice-1"), createTx("alice", 1))
	cache.AddTx([]byte("hash-bob-7"), createTx("bob", 7))

	counter := 0
	cache.ForEachTransaction(func(txHash []byte, value data.TransactionHandler) {
		counter++
	})
	require.Equal(t, 2, counter)
}

func Test_GetTransactions_Dummy(t *testing.T) {
	cache := NewTxCache(16)

	cache.AddTx([]byte("hash-alice-4"), createTx("alice", 4))
	cache.AddTx([]byte("hash-alice-3"), createTx("alice", 3))
	cache.AddTx([]byte("hash-alice-2"), createTx("alice", 2))
	cache.AddTx([]byte("hash-alice-1"), createTx("alice", 1))
	cache.AddTx([]byte("hash-bob-7"), createTx("bob", 7))
	cache.AddTx([]byte("hash-bob-6"), createTx("bob", 6))
	cache.AddTx([]byte("hash-bob-5"), createTx("bob", 5))
	cache.AddTx([]byte("hash-carol-1"), createTx("carol", 1))

	sorted, _ := cache.GetTransactions(10, 2)
	require.Len(t, sorted, 8)
}

func Test_GetTransactions(t *testing.T) {
	cache := NewTxCache(16)

	// Add "nSenders" * "nTransactionsPerSender" transactions in the cache (in reversed nonce order)
	nSenders := 1000
	nTransactionsPerSender := 100
	nTotalTransactions := nSenders * nTransactionsPerSender
	nRequestedTransactions := math.MaxInt16

	for senderTag := 0; senderTag < nSenders; senderTag++ {
		sender := fmt.Sprintf("sender:%d", senderTag)

		for txNonce := nTransactionsPerSender; txNonce > 0; txNonce-- {
			txHash := fmt.Sprintf("hash:%d:%d", senderTag, txNonce)
			tx := createTx(sender, uint64(txNonce))
			cache.AddTx([]byte(txHash), tx)
		}
	}

	require.Equal(t, int64(nTotalTransactions), cache.CountTx())

	sorted, _ := cache.GetTransactions(nRequestedTransactions, 2)

	require.Len(t, sorted, core.MinInt(nRequestedTransactions, nTotalTransactions))

	// Check order
	nonces := make(map[string]uint64, nSenders)
	for _, tx := range sorted {
		nonce := tx.GetNonce()
		sender := string(tx.GetSndAddress())
		previousNonce := nonces[sender]

		require.LessOrEqual(t, previousNonce, nonce)
		nonces[sender] = nonce
	}
}

func Test_Keys(t *testing.T) {
	cache := NewTxCache(16)

	cache.AddTx([]byte("alice-x"), createTx("alice", 42))
	cache.AddTx([]byte("alice-y"), createTx("alice", 43))
	cache.AddTx([]byte("bob-x"), createTx("bob", 42))
	cache.AddTx([]byte("bob-y"), createTx("bob", 43))

	keys := cache.Keys()
	require.Equal(t, 4, len(keys))
	require.Contains(t, keys, []byte("alice-x"))
	require.Contains(t, keys, []byte("alice-y"))
	require.Contains(t, keys, []byte("bob-x"))
	require.Contains(t, keys, []byte("bob-y"))
}

func Test_AddWithEviction_UniformDistribution(t *testing.T) {
	config := EvictionConfig{
		Enabled:                         true,
		CountThreshold:                  240000,
		NumOldestSendersToEvict:         10,
		ALotOfTransactionsForASender:    1000,
		NumTxsToEvictForASenderWithALot: 250,
	}

	// 5000 * 100
	cache := NewTxCacheWithEviction(16, config)
	addManyTransactionsWithUniformDistribution(cache, 5000, 100)
	require.Equal(t, int64(240000), cache.CountTx())

	// 1000 * 1000
	cache = NewTxCacheWithEviction(16, config)
	addManyTransactionsWithUniformDistribution(cache, 1000, 1000)
	require.Equal(t, int64(240000), cache.CountTx())
}

func Test_NotImplementedFunctions(t *testing.T) {
	cache := NewTxCache(1)

	evicted := cache.Put(nil, nil)
	require.False(t, evicted)

	has := cache.Has(nil)
	require.False(t, has)

	ok, evicted := cache.HasOrAdd(nil, nil)
	require.False(t, ok)
	require.False(t, evicted)

	require.NotPanics(t, func() { cache.RemoveOldest() })
	require.NotPanics(t, func() { cache.RegisterHandler(nil) })
	require.Zero(t, cache.MaxSize())
}

func Test_IsInterfaceNil(t *testing.T) {
	cache := NewTxCache(1)
	require.False(t, check.IfNil(cache))

	makeNil := func() storage.Cacher {
		return nil
	}

	thisIsNil := makeNil()
	require.True(t, check.IfNil(thisIsNil))
}

// This seems to be the worst case in terms of eviction complexity
// Eviction is triggered often and little eviction (only 10 senders) is done
func Benchmark_AddWithEviction_UniformDistribution_250000x1_WithConfig_NumOldestSendersToEvict_10(b *testing.B) {
	config := EvictionConfig{
		Enabled:                         true,
		CountThreshold:                  240000,
		NumOldestSendersToEvict:         10,
		ALotOfTransactionsForASender:    1000,
		NumTxsToEvictForASenderWithALot: 250,
	}

	cache := NewTxCacheWithEviction(16, config)
	addManyTransactionsWithUniformDistribution(cache, 250000, 1)
	require.Equal(b, int64(240000), cache.CountTx())
}

func Benchmark_AddWithEviction_UniformDistribution_250000x1_WithConfig_NumOldestSendersToEvict_100(b *testing.B) {
	config := EvictionConfig{
		Enabled:                         true,
		CountThreshold:                  240000,
		NumOldestSendersToEvict:         100,
		ALotOfTransactionsForASender:    1000,
		NumTxsToEvictForASenderWithALot: 250,
	}

	cache := NewTxCacheWithEviction(16, config)
	addManyTransactionsWithUniformDistribution(cache, 250000, 1)
	require.Equal(b, int64(240000), cache.CountTx())
}

func Benchmark_AddWithEviction_UniformDistribution_250000x1_WithConfig_NumOldestSendersToEvict_1000(b *testing.B) {
	config := EvictionConfig{
		Enabled:                         true,
		CountThreshold:                  240000,
		NumOldestSendersToEvict:         1000,
		ALotOfTransactionsForASender:    1000,
		NumTxsToEvictForASenderWithALot: 250,
	}

	cache := NewTxCacheWithEviction(16, config)
	addManyTransactionsWithUniformDistribution(cache, 250000, 1)
	require.Equal(b, int64(240000), cache.CountTx())
}

func Benchmark_AddWithEviction_UniformDistribution_10x25000(b *testing.B) {
	config := EvictionConfig{
		Enabled:                         true,
		CountThreshold:                  240000,
		NumOldestSendersToEvict:         1000,
		ALotOfTransactionsForASender:    1000,
		NumTxsToEvictForASenderWithALot: 250,
	}

	cache := NewTxCacheWithEviction(16, config)
	addManyTransactionsWithUniformDistribution(cache, 10, 25000)
	require.Equal(b, int64(240000), cache.CountTx())
}

func Benchmark_AddWithEviction_UniformDistribution_1x250000(b *testing.B) {
	config := EvictionConfig{
		Enabled:                         true,
		CountThreshold:                  240000,
		NumOldestSendersToEvict:         1000,
		ALotOfTransactionsForASender:    1000,
		NumTxsToEvictForASenderWithALot: 250,
	}

	cache := NewTxCacheWithEviction(16, config)
	addManyTransactionsWithUniformDistribution(cache, 1, 250000)
	require.Equal(b, int64(240000), cache.CountTx())
}

func addManyTransactionsWithUniformDistribution(cache *TxCache, nSenders int, nTransactionsPerSender int) {
	for senderTag := 0; senderTag < nSenders; senderTag++ {
		sender := createFakeSenderAddress(senderTag)

		for txNonce := nTransactionsPerSender; txNonce > 0; txNonce-- {
			txHash := createFakeTxHash(sender, txNonce)
			tx := createTx(string(sender), uint64(txNonce))
			cache.AddTx(txHash, tx)
		}
	}
}

func createTx(sender string, nonce uint64) data.TransactionHandler {
	return &transaction.Transaction{
		SndAddr: []byte(sender),
		Nonce:   nonce,
	}
}

func createFakeSenderAddress(senderTag int) []byte {
	bytes := make([]byte, 32)
	binary.LittleEndian.PutUint64(bytes, uint64(senderTag))
	binary.LittleEndian.PutUint64(bytes[24:], uint64(senderTag))
	return bytes
}

func createFakeTxHash(fakeSenderAddress []byte, nonce int) []byte {
	bytes := make([]byte, 32)
	copy(bytes, fakeSenderAddress)
	binary.LittleEndian.PutUint64(bytes[8:], uint64(nonce))
	binary.LittleEndian.PutUint64(bytes[16:], uint64(nonce))
	return bytes
}