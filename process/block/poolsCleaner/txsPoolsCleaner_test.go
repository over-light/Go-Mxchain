package poolsCleaner

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/txcache"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/stretchr/testify/assert"
)

func TestNewTxsPoolsCleaner_NilAddrConverterErr(t *testing.T) {
	t.Parallel()

	txsPoolsCleaner, err := NewTxsPoolsCleaner(
		nil, &mock.PoolsHolderMock{}, &mock.RounderMock{}, mock.NewMultipleShardsCoordinatorMock(),
	)
	assert.Nil(t, txsPoolsCleaner)
	assert.Equal(t, process.ErrNilPubkeyConverter, err)
}

func TestNewTxsPoolsCleaner_NilDataPoolHolderErr(t *testing.T) {
	t.Parallel()

	txsPoolsCleaner, err := NewTxsPoolsCleaner(
		&mock.PubkeyConverterStub{}, nil, &mock.RounderMock{}, mock.NewMultipleShardsCoordinatorMock(),
	)
	assert.Nil(t, txsPoolsCleaner)
	assert.Equal(t, process.ErrNilPoolsHolder, err)
}

func TestNewTxsPoolsCleaner_NilTxsPoolErr(t *testing.T) {
	t.Parallel()

	dataPool := &mock.PoolsHolderStub{
		TransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return nil
		},
	}
	txsPoolsCleaner, err := NewTxsPoolsCleaner(
		&mock.PubkeyConverterStub{}, dataPool, &mock.RounderMock{}, mock.NewMultipleShardsCoordinatorMock(),
	)
	assert.Nil(t, txsPoolsCleaner)
	assert.Equal(t, process.ErrNilTransactionPool, err)
}

func TestNewTxsPoolsCleaner_NilRewardTxsPoolErr(t *testing.T) {
	t.Parallel()

	dataPool := &mock.PoolsHolderStub{
		TransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
		RewardTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return nil
		},
	}
	txsPoolsCleaner, err := NewTxsPoolsCleaner(
		&mock.PubkeyConverterStub{}, dataPool, &mock.RounderMock{}, mock.NewMultipleShardsCoordinatorMock(),
	)
	assert.Nil(t, txsPoolsCleaner)
	assert.Equal(t, process.ErrNilRewardTxDataPool, err)
}

func TestNewTxsPoolsCleaner_NilUnsignedTxsPoolErr(t *testing.T) {
	t.Parallel()

	dataPool := &mock.PoolsHolderStub{
		TransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
		RewardTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
		UnsignedTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return nil
		},
	}
	txsPoolsCleaner, err := NewTxsPoolsCleaner(
		&mock.PubkeyConverterStub{}, dataPool, &mock.RounderMock{}, mock.NewMultipleShardsCoordinatorMock(),
	)
	assert.Nil(t, txsPoolsCleaner)
	assert.Equal(t, process.ErrNilUnsignedTxDataPool, err)
}

func TestNewTxsPoolsCleaner_NilRounderErr(t *testing.T) {
	t.Parallel()

	dataPool := &mock.PoolsHolderStub{
		TransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
		RewardTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
		UnsignedTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
	}
	txsPoolsCleaner, err := NewTxsPoolsCleaner(
		&mock.PubkeyConverterStub{}, dataPool, nil, mock.NewMultipleShardsCoordinatorMock(),
	)
	assert.Nil(t, txsPoolsCleaner)
	assert.Equal(t, process.ErrNilRounder, err)
}

func TestNewTxsPoolsCleaner_NilShardCoordinatorErr(t *testing.T) {
	t.Parallel()

	dataPool := &mock.PoolsHolderStub{
		TransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
		RewardTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
		UnsignedTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
	}
	txsPoolsCleaner, err := NewTxsPoolsCleaner(
		&mock.PubkeyConverterStub{}, dataPool, &mock.RounderMock{}, nil,
	)
	assert.Nil(t, txsPoolsCleaner)
	assert.Equal(t, process.ErrNilShardCoordinator, err)
}

func TestNewTxsPoolsCleaner_ShouldWork(t *testing.T) {
	t.Parallel()

	dataPool := &mock.PoolsHolderStub{
		TransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
		RewardTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
		UnsignedTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
	}

	txsPoolsCleaner, err := NewTxsPoolsCleaner(
		&mock.PubkeyConverterStub{}, dataPool, &mock.RounderMock{}, mock.NewMultipleShardsCoordinatorMock(),
	)
	assert.Nil(t, err)
	assert.NotNil(t, txsPoolsCleaner)
}

func TestGetShardFromAddress(t *testing.T) {
	t.Parallel()

	addrLen := 64
	addrConverter := &mock.PubkeyConverterStub{
		LenCalled: func() int {
			return addrLen
		},
	}
	expectedShard := uint32(2)
	txsPoolsCleaner, _ := NewTxsPoolsCleaner(
		addrConverter,
		&mock.PoolsHolderStub{},
		&mock.RounderMock{},
		&mock.CoordinatorStub{
			ComputeIdCalled: func(address []byte) uint32 {
				return expectedShard
			},
		},
	)

	emptyAddr := make([]byte, addrLen)
	result, err := txsPoolsCleaner.getShardFromAddress(emptyAddr)
	assert.Nil(t, err)
	assert.Equal(t, uint32(0), result)

	result, err = txsPoolsCleaner.getShardFromAddress([]byte("123"))
	assert.Nil(t, err)
	assert.Equal(t, expectedShard, result)
}

func TestReceivedBlockTx_ShouldBeAddedInMapTxsRounds(t *testing.T) {
	t.Parallel()

	txsPoolsCleaner, _ := NewTxsPoolsCleaner(
		&mock.PubkeyConverterStub{},
		&mock.PoolsHolderStub{
			TransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
				return &mock.ShardedDataStub{
					ShardDataStoreCalled: func(cacheId string) (c storage.Cacher) {
						return testscommon.NewCacherMock()
					},
				}
			},
		},
		&mock.RounderMock{},
		&mock.CoordinatorStub{},
	)

	txWrap := &txcache.WrappedTransaction{
		Tx:            &transaction.Transaction{},
		SenderShardID: 2,
	}
	txBlockKey := []byte("key")
	txsPoolsCleaner.receivedBlockTx(txBlockKey, txWrap)
	assert.NotNil(t, txsPoolsCleaner.mapTxsRounds[string(txBlockKey)])
}

func TestReceivedRewardTx_ShouldBeAddedInMapTxsRounds(t *testing.T) {
	t.Parallel()

	txsPoolsCleaner, _ := NewTxsPoolsCleaner(
		&mock.PubkeyConverterStub{},
		&mock.PoolsHolderStub{
			RewardTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
				return &mock.ShardedDataStub{
					ShardDataStoreCalled: func(cacheId string) (c storage.Cacher) {
						return testscommon.NewCacherMock()
					},
				}
			},
		},
		&mock.RounderMock{},
		&mock.CoordinatorStub{},
	)

	txKey := []byte("key")
	txsPoolsCleaner.receivedRewardTx(txKey, nil)
	assert.NotNil(t, txsPoolsCleaner.mapTxsRounds[string(txKey)])
}

func TestReceivedUnsignedTx_ShouldBeAddedInMapTxsRounds(t *testing.T) {
	t.Parallel()

	sndAddr := []byte("sndAddr")
	txsPoolsCleaner, _ := NewTxsPoolsCleaner(
		&mock.PubkeyConverterStub{},
		&mock.PoolsHolderStub{
			UnsignedTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
				return &mock.ShardedDataStub{
					ShardDataStoreCalled: func(cacheId string) (c storage.Cacher) {
						return testscommon.NewCacherMock()
					},
				}
			},
		},
		&mock.RounderMock{},
		&mock.CoordinatorStub{
			ComputeIdCalled: func(address []byte) uint32 {
				return 2
			},
		},
	)

	txKey := []byte("key")
	tx := &transaction.Transaction{
		SndAddr: sndAddr,
	}
	txsPoolsCleaner.receivedUnsignedTx(txKey, tx)
	assert.NotNil(t, txsPoolsCleaner.mapTxsRounds[string(txKey)])
}

func TestCleanTxsPoolsIfNeeded_CannotFindTxInPoolShouldBeRemovedFromMap(t *testing.T) {
	t.Parallel()

	sndAddr := []byte("sndAddr")
	txsPoolsCleaner, _ := NewTxsPoolsCleaner(
		&mock.PubkeyConverterStub{},
		&mock.PoolsHolderStub{
			UnsignedTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
				return &mock.ShardedDataStub{
					ShardDataStoreCalled: func(cacheId string) (c storage.Cacher) {
						return testscommon.NewCacherMock()
					},
				}
			},
		},
		&mock.RounderMock{},
		&mock.CoordinatorStub{
			ComputeIdCalled: func(address []byte) uint32 {
				return 2
			},
		},
	)

	txKey := []byte("key")
	tx := &transaction.Transaction{
		SndAddr: sndAddr,
	}
	txsPoolsCleaner.receivedUnsignedTx(txKey, tx)

	numTxsInMap := txsPoolsCleaner.cleanTxsPoolsIfNeeded()
	assert.Equal(t, 0, numTxsInMap)
}

func TestCleanTxsPoolsIfNeeded_RoundDiffTooSmallShouldNotBeRemoved(t *testing.T) {
	t.Parallel()

	sndAddr := []byte("sndAddr")
	txsPoolsCleaner, _ := NewTxsPoolsCleaner(
		&mock.PubkeyConverterStub{},
		&mock.PoolsHolderStub{
			UnsignedTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
				return &mock.ShardedDataStub{
					ShardDataStoreCalled: func(cacheId string) (c storage.Cacher) {
						return &mock.CacherStub{
							GetCalled: func(key []byte) (value interface{}, ok bool) {
								return nil, true
							},
						}
					},
				}
			},
		},
		&mock.RounderMock{},
		&mock.CoordinatorStub{
			ComputeIdCalled: func(address []byte) uint32 {
				return 2
			},
		},
	)

	txKey := []byte("key")
	tx := &transaction.Transaction{
		SndAddr: sndAddr,
	}
	txsPoolsCleaner.receivedUnsignedTx(txKey, tx)

	numTxsInMap := txsPoolsCleaner.cleanTxsPoolsIfNeeded()
	assert.Equal(t, 1, numTxsInMap)
}

func TestCleanTxsPoolsIfNeeded_RoundDiffTooBigShouldBeRemoved(t *testing.T) {
	t.Parallel()

	rounder := &mock.RoundStub{IndexCalled: func() int64 {
		return 0
	}}
	called := false
	sndAddr := []byte("sndAddr")
	txsPoolsCleaner, _ := NewTxsPoolsCleaner(
		&mock.PubkeyConverterStub{},
		&mock.PoolsHolderStub{
			UnsignedTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
				return &mock.ShardedDataStub{
					ShardDataStoreCalled: func(cacheId string) (c storage.Cacher) {
						return &mock.CacherStub{
							GetCalled: func(key []byte) (value interface{}, ok bool) {
								return nil, true
							},
							RemoveCalled: func(key []byte) {
								called = true
							},
						}
					},
				}
			},
		},
		rounder,
		&mock.CoordinatorStub{
			ComputeIdCalled: func(address []byte) uint32 {
				return 2
			},
		},
	)

	txKey := []byte("key")
	tx := &transaction.Transaction{
		SndAddr: sndAddr,
	}
	txsPoolsCleaner.receivedUnsignedTx(txKey, tx)

	rounder.IndexCalled = func() int64 {
		return process.MaxRoundsToKeepUnprocessedTransactions + 1
	}
	numTxsInMap := txsPoolsCleaner.cleanTxsPoolsIfNeeded()
	assert.Equal(t, 0, numTxsInMap)
	assert.Nil(t, txsPoolsCleaner.mapTxsRounds[string(txKey)])
	assert.True(t, called)
}
