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

func Test_AddTx_IgnoresTx_IfEvictionInProgress(t *testing.T) {
	cache := NewTxCache(1)

	cache.maybeEvictionInProgress.Set()
	ok, added := cache.AddTx([]byte("hash-1"), createTx("alice", 1))
	require.False(t, ok)
	require.False(t, added)
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
	require.Equal(t, int(3), cache.Len())
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

func Test_AddWithEviction_UniformDistributionOfTxsPerSender(t *testing.T) {
	config := EvictionConfig{
		Enabled:                         true,
		NumBytesThreshold:               math.MaxUint32,
		CountThreshold:                  100,
		NumSendersToEvictInOneStep:      1,
		ALotOfTransactionsForASender:    math.MaxUint32,
		NumTxsToEvictForASenderWithALot: 0,
	}

	// 11 * 10
	cache := NewTxCacheWithEviction(16, config)
	addManyTransactionsWithUniformDistribution(cache, 11, 10)
	require.LessOrEqual(t, cache.CountTx(), int64(100))

	config = EvictionConfig{
		Enabled:                         true,
		NumBytesThreshold:               math.MaxUint32,
		CountThreshold:                  250000,
		NumSendersToEvictInOneStep:      1,
		ALotOfTransactionsForASender:    math.MaxUint32,
		NumTxsToEvictForASenderWithALot: 0,
	}

	// 1000 * 1000
	cache = NewTxCacheWithEviction(16, config)
	addManyTransactionsWithUniformDistribution(cache, 1000, 1000)
	require.LessOrEqual(t, cache.CountTx(), int64(250000))
}

func Test_AddWithEviction_SizeAndCount(t *testing.T) {
	config := EvictionConfig{
		Enabled:                    true,
		NumBytesThreshold:          200000,
		CountThreshold:             500,
		NumSendersToEvictInOneStep: 1,
	}

	cache := NewTxCacheWithEviction(16, config)

	// Alice sends 201 transactions of 1000 bytes, with gas price 15
	for i := 0; i < 201; i++ {
		cache.AddTx([]byte(fmt.Sprintf("alice-%d", i)), createTxWithGas("alice", uint64(i), 1000, 15))
	}

	require.Equal(t, int64(201*1000), cache.VolumeInBytes())
	require.Equal(t, int64(201000), cache.VolumeInBytes())

	// Alice sends another transaction
	// This transaction will cause eviction
	cache.AddTx([]byte(fmt.Sprintf("alice-foo")), createTxWithGas("alice", uint64(201), 1000, 15))

	// Only latest Alice's transaction remains in cache
	require.Equal(t, int64(1), cache.CountTx())
	require.Equal(t, int64(1), cache.countTxBySender("alice"))
	require.Equal(t, int64(1000), cache.VolumeInBytes())

	// Bob and Carol send transactions, with different gas price
	for i := 0; i < 100; i++ {
		cache.AddTx([]byte(fmt.Sprintf("bob-%d", i)), createTxWithGas("bob", uint64(i), 1000, 30))
	}

	for i := 0; i < 100; i++ {
		cache.AddTx([]byte(fmt.Sprintf("carol-%d", i)), createTxWithGas("carol", uint64(i), 1000, 20))
	}

	// 100 from Bob, 100 from Carol, one from Alice
	require.Equal(t, int64(1), cache.countTxBySender("alice"))
	require.Equal(t, int64(100), cache.countTxBySender("bob"))
	require.Equal(t, int64(100), cache.countTxBySender("carol"))
	require.Equal(t, int64(3), cache.CountSenders())
	require.Equal(t, int64(201000), cache.VolumeInBytes())

	// Carol sends another transaction
	// This transaction will cause eviction
	cache.AddTx([]byte(fmt.Sprintf("carol-foo")), createTxWithGas("carol", uint64(100), 1000, 15))

	// All other transactions (from Alice and Carol) are still in place
	// Carol has 101 transactions (1 to 100, plus the "foo" transaction)
	require.Equal(t, int64(1), cache.countTxBySender("alice"))
	require.Equal(t, int64(0), cache.countTxBySender("bob"))
	require.Equal(t, int64(101), cache.countTxBySender("carol"))
	require.Equal(t, int64(2), cache.CountSenders())
	require.Equal(t, int64(102000), cache.VolumeInBytes())
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

func addManyTransactionsWithUniformDistribution(cache *TxCache, nSenders int, nTransactionsPerSender int) {
	for senderTag := 0; senderTag < nSenders; senderTag++ {
		sender := createFakeSenderAddress(senderTag)

		for txNonce := nTransactionsPerSender; txNonce > 0; txNonce-- {
			txHash := createFakeTxHash(sender, txNonce)
			tx := createTx(string(sender), uint64(txNonce))
			cache.AddTx([]byte(txHash), tx)
		}
	}
}

func createTx(sender string, nonce uint64) *transaction.Transaction {
	return &transaction.Transaction{
		SndAddr: []byte(sender),
		Nonce:   nonce,
	}
}

func createTxWithData(sender string, nonce uint64, dataLength uint64) *transaction.Transaction {
	payloadLength := int(dataLength) - int(estimatedSizeOfBoundedTxFields)
	if payloadLength < 0 {
		panic("createTxWithData(): invalid length for dummy tx")
	}

	return &transaction.Transaction{
		SndAddr: []byte(sender),
		Nonce:   nonce,
		Data:    make([]byte, payloadLength),
	}
}

func createTxWithGas(sender string, nonce uint64, dataLength uint64, gasLimit uint64) *transaction.Transaction {
	tx := createTxWithData(sender, nonce, dataLength)
	tx.GasPrice = 1
	tx.GasLimit = gasLimit
	return tx
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
