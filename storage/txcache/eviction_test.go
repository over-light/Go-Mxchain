package txcache

import (
	"math"
	"sync"
	"testing"

	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/stretchr/testify/require"
)

func TestEviction_EvictSendersWhileTooManyTxs(t *testing.T) {
	config := ConfigSourceMe{
		Name:                          "untitled",
		NumChunks:                     16,
		CountThreshold:                100,
		CountPerSenderThreshold:       math.MaxUint32,
		NumSendersToPreemptivelyEvict: 20,
		NumBytesThreshold:             maxNumBytesUpperBound,
		NumBytesPerSenderThreshold:    maxNumBytesPerSenderUpperBound,
		MinGasPriceNanoErd:            100,
	}

	cache, err := NewTxCache(config)
	require.Nil(t, err)
	require.NotNil(t, cache)

	// 200 senders, each with 1 transaction
	for index := 0; index < 200; index++ {
		sender := string(createFakeSenderAddress(index))
		cache.AddTx(createTx([]byte{byte(index)}, sender, uint64(1)))
	}

	require.Equal(t, int64(200), cache.txListBySender.counter.Get())
	require.Equal(t, int64(200), cache.txByHash.counter.Get())

	cache.makeSnapshotOfSenders()
	steps, nTxs, nSenders := cache.evictSendersInLoop()

	require.Equal(t, uint32(5), steps)
	require.Equal(t, uint32(100), nTxs)
	require.Equal(t, uint32(100), nSenders)
	require.Equal(t, int64(100), cache.txListBySender.counter.Get())
	require.Equal(t, int64(100), cache.txByHash.counter.Get())
}

func TestEviction_EvictSendersWhileTooManyBytes(t *testing.T) {
	numBytesPerTx := uint32(1000)

	config := ConfigSourceMe{
		Name:                          "untitled",
		NumChunks:                     16,
		CountThreshold:                math.MaxUint32,
		CountPerSenderThreshold:       math.MaxUint32,
		NumBytesThreshold:             numBytesPerTx * 100,
		NumBytesPerSenderThreshold:    maxNumBytesPerSenderUpperBound,
		NumSendersToPreemptivelyEvict: 20,
		MinGasPriceNanoErd:            100,
	}

	cache, err := NewTxCache(config)
	require.Nil(t, err)
	require.NotNil(t, cache)

	// 200 senders, each with 1 transaction
	for index := 0; index < 200; index++ {
		sender := string(createFakeSenderAddress(index))
		cache.AddTx(createTxWithParams([]byte{byte(index)}, sender, uint64(1), uint64(numBytesPerTx), 10000, 100*oneBillion))
	}

	require.Equal(t, int64(200), cache.txListBySender.counter.Get())
	require.Equal(t, int64(200), cache.txByHash.counter.Get())

	cache.makeSnapshotOfSenders()
	steps, nTxs, nSenders := cache.evictSendersInLoop()

	require.Equal(t, uint32(5), steps)
	require.Equal(t, uint32(100), nTxs)
	require.Equal(t, uint32(100), nSenders)
	require.Equal(t, int64(100), cache.txListBySender.counter.Get())
	require.Equal(t, int64(100), cache.txByHash.counter.Get())
}

func TestEviction_DoEvictionDoneInPassTwo_BecauseOfCount(t *testing.T) {
	config := ConfigSourceMe{
		Name:                          "untitled",
		NumChunks:                     16,
		NumBytesThreshold:             maxNumBytesUpperBound,
		NumBytesPerSenderThreshold:    maxNumBytesPerSenderUpperBound,
		CountThreshold:                2,
		CountPerSenderThreshold:       math.MaxUint32,
		NumSendersToPreemptivelyEvict: 2,
		MinGasPriceNanoErd:            100,
	}

	cache, err := NewTxCache(config)
	require.Nil(t, err)
	require.NotNil(t, cache)

	cache.AddTx(createTxWithParams([]byte("hash-alice"), "alice", uint64(1), 1000, 100000, 100*oneBillion))
	cache.AddTx(createTxWithParams([]byte("hash-bob"), "bob", uint64(1), 1000, 100000, 100*oneBillion))
	cache.AddTx(createTxWithParams([]byte("hash-carol"), "carol", uint64(1), 1000, 100000, 700*oneBillion))

	cache.doEviction()
	require.Equal(t, uint32(2), cache.evictionJournal.passOneNumTxs)
	require.Equal(t, uint32(2), cache.evictionJournal.passOneNumSenders)
	require.Equal(t, uint32(1), cache.evictionJournal.passOneNumSteps)

	// Alice and Bob evicted. Carol still there.
	_, ok := cache.GetByTxHash([]byte("hash-carol"))
	require.True(t, ok)
	require.Equal(t, uint64(1), cache.CountSenders())
	require.Equal(t, uint64(1), cache.CountTx())
}

func TestEviction_DoEvictionDoneInPassTwo_BecauseOfSize(t *testing.T) {
	config := ConfigSourceMe{
		Name:                          "untitled",
		NumChunks:                     16,
		CountThreshold:                math.MaxUint32,
		CountPerSenderThreshold:       math.MaxUint32,
		NumBytesThreshold:             1000,
		NumBytesPerSenderThreshold:    maxNumBytesPerSenderUpperBound,
		NumSendersToPreemptivelyEvict: 2,
		MinGasPriceNanoErd:            100,
	}

	cache, err := NewTxCache(config)
	require.Nil(t, err)
	require.NotNil(t, cache)

	cache.AddTx(createTxWithParams([]byte("hash-alice"), "alice", uint64(1), 800, 100000, 100*oneBillion))
	cache.AddTx(createTxWithParams([]byte("hash-bob"), "bob", uint64(1), 500, 100000, 100*oneBillion))
	cache.AddTx(createTxWithParams([]byte("hash-carol"), "carol", uint64(1), 200, 100000, 700*oneBillion))

	require.Equal(t, uint32(19), cache.getScoreOfSender("alice"))
	require.Equal(t, uint32(23), cache.getScoreOfSender("bob"))
	require.Equal(t, uint32(100), cache.getScoreOfSender("carol"))

	cache.doEviction()
	require.Equal(t, uint32(2), cache.evictionJournal.passOneNumTxs)
	require.Equal(t, uint32(2), cache.evictionJournal.passOneNumSenders)
	require.Equal(t, uint32(1), cache.evictionJournal.passOneNumSteps)

	// Alice and Bob evicted (lower score). Carol still there.
	_, ok := cache.GetByTxHash([]byte("hash-carol"))
	require.True(t, ok)
	require.Equal(t, uint64(1), cache.CountSenders())
	require.Equal(t, uint64(1), cache.CountTx())
}

func TestEviction_doEvictionDoesNothingWhenAlreadyInProgress(t *testing.T) {
	config := ConfigSourceMe{
		Name:                          "untitled",
		NumChunks:                     1,
		CountThreshold:                0,
		NumSendersToPreemptivelyEvict: 1,
		NumBytesPerSenderThreshold:    maxNumBytesPerSenderUpperBound,
		CountPerSenderThreshold:       math.MaxUint32,
		MinGasPriceNanoErd:            100,
	}

	cache, err := NewTxCache(config)
	require.Nil(t, err)
	require.NotNil(t, cache)

	cache.AddTx(createTx([]byte("hash-alice"), "alice", uint64(1)))

	cache.isEvictionInProgress.Set()
	cache.doEviction()

	require.False(t, cache.evictionJournal.evictionPerformed)
}

func TestEviction_evictSendersInLoop_CoverLoopBreak_WhenSmallBatch(t *testing.T) {
	config := ConfigSourceMe{
		Name:                          "untitled",
		NumChunks:                     1,
		CountThreshold:                0,
		NumSendersToPreemptivelyEvict: 42,
		NumBytesPerSenderThreshold:    maxNumBytesPerSenderUpperBound,
		CountPerSenderThreshold:       math.MaxUint32,
		MinGasPriceNanoErd:            100,
	}

	cache, err := NewTxCache(config)
	require.Nil(t, err)
	require.NotNil(t, cache)

	cache.AddTx(createTx([]byte("hash-alice"), "alice", uint64(1)))

	cache.makeSnapshotOfSenders()

	steps, nTxs, nSenders := cache.evictSendersInLoop()
	require.Equal(t, uint32(0), steps)
	require.Equal(t, uint32(1), nTxs)
	require.Equal(t, uint32(1), nSenders)
}

func TestEviction_evictSendersWhile_ShouldContinueBreak(t *testing.T) {
	config := ConfigSourceMe{
		Name:                          "untitled",
		NumChunks:                     1,
		CountThreshold:                0,
		NumSendersToPreemptivelyEvict: 1,
		NumBytesPerSenderThreshold:    maxNumBytesPerSenderUpperBound,
		CountPerSenderThreshold:       math.MaxUint32,
		MinGasPriceNanoErd:            100,
	}

	cache, err := NewTxCache(config)
	require.Nil(t, err)
	require.NotNil(t, cache)

	cache.AddTx(createTx([]byte("hash-alice"), "alice", uint64(1)))
	cache.AddTx(createTx([]byte("hash-bob"), "bob", uint64(1)))

	cache.makeSnapshotOfSenders()

	steps, nTxs, nSenders := cache.evictSendersWhile(func() bool {
		return false
	})

	require.Equal(t, uint32(0), steps)
	require.Equal(t, uint32(0), nTxs)
	require.Equal(t, uint32(0), nSenders)
}

// This seems to be the most reasonable "bad-enough" (not worst) scenario to benchmark:
// 25000 senders with 10 transactions each, with default "NumSendersToPreemptivelyEvict".
// ~1 second on average laptop.
func Test_AddWithEviction_UniformDistribution_25000x10(t *testing.T) {
	config := ConfigSourceMe{
		Name:                          "untitled",
		NumChunks:                     16,
		EvictionEnabled:               true,
		NumBytesThreshold:             1000000000,
		CountThreshold:                240000,
		NumSendersToPreemptivelyEvict: dataRetriever.TxPoolNumSendersToPreemptivelyEvict,
		NumBytesPerSenderThreshold:    maxNumBytesPerSenderUpperBound,
		CountPerSenderThreshold:       math.MaxUint32,
		MinGasPriceNanoErd:            100,
	}

	numSenders := 25000
	numTxsPerSender := 10

	cache, err := NewTxCache(config)
	require.Nil(t, err)
	require.NotNil(t, cache)

	addManyTransactionsWithUniformDistribution(cache, numSenders, numTxsPerSender)

	// Sometimes (due to map iteration non-determinism), more eviction happens - one more step of 100 senders.
	require.LessOrEqual(t, uint32(cache.CountTx()), config.CountThreshold)
	require.GreaterOrEqual(t, uint32(cache.CountTx()), config.CountThreshold-config.NumSendersToPreemptivelyEvict*uint32(numTxsPerSender))
}

func Test_EvictSendersAndTheirTxs_Concurrently(t *testing.T) {
	cache := newUnconstrainedCacheToTest()
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(3)

		go func() {
			cache.AddTx(createTx([]byte("alice-x"), "alice", 42))
			cache.AddTx(createTx([]byte("alice-y"), "alice", 43))
			cache.AddTx(createTx([]byte("bob-x"), "bob", 42))
			cache.AddTx(createTx([]byte("bob-y"), "bob", 43))
			cache.Remove([]byte("alice-x"))
			cache.Remove([]byte("bob-x"))
			wg.Done()
		}()

		go func() {
			snapshot := cache.txListBySender.getSnapshotAscending()
			cache.evictSendersAndTheirTxs(snapshot)
			wg.Done()
		}()

		go func() {
			snapshot := cache.txListBySender.getSnapshotAscending()
			cache.evictSendersAndTheirTxs(snapshot)
			wg.Done()
		}()
	}

	wg.Wait()
}
