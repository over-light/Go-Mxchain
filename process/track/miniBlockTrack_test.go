package track_test

import (
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/process/track"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewMiniBlockTrack_NilDataPoolHolderErr(t *testing.T) {
	t.Parallel()

	miniBlockTrack, err := track.NewMiniBlockTrack(nil, mock.NewMultipleShardsCoordinatorMock())
	assert.Nil(t, miniBlockTrack)
	assert.Equal(t, process.ErrNilPoolsHolder, err)
}

func TestNewMiniBlockTrack_NilTxsPoolErr(t *testing.T) {
	t.Parallel()

	dataPool := &mock.PoolsHolderStub{
		TransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return nil
		},
	}
	miniBlockTrack, err := track.NewMiniBlockTrack(dataPool, mock.NewMultipleShardsCoordinatorMock())
	assert.Nil(t, miniBlockTrack)
	assert.Equal(t, process.ErrNilTransactionPool, err)
}

func TestNewMiniBlockTrack_NilRewardTxsPoolErr(t *testing.T) {
	t.Parallel()

	dataPool := &mock.PoolsHolderStub{
		TransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return &mock.ShardedDataStub{}
		},
		RewardTransactionsCalled: func() dataRetriever.ShardedDataCacherNotifier {
			return nil
		},
	}
	miniBlockTrack, err := track.NewMiniBlockTrack(dataPool, mock.NewMultipleShardsCoordinatorMock())
	assert.Nil(t, miniBlockTrack)
	assert.Equal(t, process.ErrNilRewardTxDataPool, err)
}

func TestNewMiniBlockTrack_NilUnsignedTxsPoolErr(t *testing.T) {
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
	miniBlockTrack, err := track.NewMiniBlockTrack(dataPool, mock.NewMultipleShardsCoordinatorMock())
	assert.Nil(t, miniBlockTrack)
	assert.Equal(t, process.ErrNilUnsignedTxDataPool, err)
}

func TestNewMiniBlockTrack_NilMiniBlockPoolShouldErr(t *testing.T) {
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
		MiniBlocksCalled: func() storage.Cacher {
			return nil
		},
	}
	miniBlockTrack, err := track.NewMiniBlockTrack(dataPool, mock.NewMultipleShardsCoordinatorMock())
	assert.Nil(t, miniBlockTrack)
	assert.Equal(t, process.ErrNilMiniBlockPool, err)
}

func TestNewMiniBlockTrack_NilShardCoordinatorErr(t *testing.T) {
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
		MiniBlocksCalled: func() storage.Cacher {
			return &mock.CacherStub{}
		},
	}
	miniBlockTrack, err := track.NewMiniBlockTrack(dataPool, nil)
	assert.Nil(t, miniBlockTrack)
	assert.Equal(t, process.ErrNilShardCoordinator, err)
}

func TestNewMiniBlockTrack_ShouldWork(t *testing.T) {
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
		MiniBlocksCalled: func() storage.Cacher {
			return &mock.CacherStub{}
		},
	}

	miniBlockTrack, err := track.NewMiniBlockTrack(dataPool, mock.NewMultipleShardsCoordinatorMock())
	assert.Nil(t, err)
	assert.NotNil(t, miniBlockTrack)
}

// TODO: Unit tests for receivedMiniBlock and getTransactionPool methods should be added
