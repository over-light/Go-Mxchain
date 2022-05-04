package dataRetriever

import (
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/testscommon"
)

// PoolsHolderStub -
type PoolsHolderStub struct {
	HeadersCalled                func() dataRetriever.HeadersPool
	TransactionsCalled           func() dataRetriever.ShardedDataCacherNotifier
	UnsignedTransactionsCalled   func() dataRetriever.ShardedDataCacherNotifier
	RewardTransactionsCalled     func() dataRetriever.ShardedDataCacherNotifier
	MiniBlocksCalled             func() storage.Cacher
	MetaBlocksCalled             func() storage.Cacher
	CurrBlockTxsCalled           func() dataRetriever.TransactionCacher
	CurrBlockValidatorInfoCalled func() dataRetriever.ValidatorInfoCacher
	TrieNodesCalled              func() storage.Cacher
	TrieNodesChunksCalled        func() storage.Cacher
	PeerChangesBlocksCalled      func() storage.Cacher
	SmartContractsCalled         func() storage.Cacher
	ValidatorsInfoCalled         func() storage.Cacher
}

// NewPoolsHolderStub -
func NewPoolsHolderStub() *PoolsHolderStub {
	return &PoolsHolderStub{}
}

// Headers -
func (holder *PoolsHolderStub) Headers() dataRetriever.HeadersPool {
	if holder.HeadersCalled != nil {
		return holder.HeadersCalled()
	}

	return nil
}

// Transactions -
func (holder *PoolsHolderStub) Transactions() dataRetriever.ShardedDataCacherNotifier {
	if holder.TransactionsCalled != nil {
		return holder.TransactionsCalled()
	}

	return testscommon.NewShardedDataStub()
}

// UnsignedTransactions -
func (holder *PoolsHolderStub) UnsignedTransactions() dataRetriever.ShardedDataCacherNotifier {
	if holder.UnsignedTransactionsCalled != nil {
		return holder.UnsignedTransactionsCalled()
	}

	return testscommon.NewShardedDataStub()
}

// RewardTransactions -
func (holder *PoolsHolderStub) RewardTransactions() dataRetriever.ShardedDataCacherNotifier {
	if holder.RewardTransactionsCalled != nil {
		return holder.RewardTransactionsCalled()
	}

	return testscommon.NewShardedDataStub()
}

// MiniBlocks -
func (holder *PoolsHolderStub) MiniBlocks() storage.Cacher {
	if holder.MiniBlocksCalled != nil {
		return holder.MiniBlocksCalled()
	}

	return testscommon.NewCacherStub()
}

// MetaBlocks -
func (holder *PoolsHolderStub) MetaBlocks() storage.Cacher {
	if holder.MetaBlocksCalled != nil {
		return holder.MetaBlocksCalled()
	}

	return testscommon.NewCacherStub()
}

// CurrentBlockTxs -
func (holder *PoolsHolderStub) CurrentBlockTxs() dataRetriever.TransactionCacher {
	if holder.CurrBlockTxsCalled != nil {
		return holder.CurrBlockTxsCalled()
	}

	return nil
}

// CurrentBlockValidatorInfo -
func (holder *PoolsHolderStub) CurrentBlockValidatorInfo() dataRetriever.ValidatorInfoCacher {
	if holder.CurrBlockValidatorInfoCalled != nil {
		return holder.CurrBlockValidatorInfoCalled()
	}

	return nil
}

// TrieNodes -
func (holder *PoolsHolderStub) TrieNodes() storage.Cacher {
	if holder.TrieNodesCalled != nil {
		return holder.TrieNodesCalled()
	}

	return testscommon.NewCacherStub()
}

// TrieNodesChunks -
func (holder *PoolsHolderStub) TrieNodesChunks() storage.Cacher {
	if holder.TrieNodesChunksCalled != nil {
		return holder.TrieNodesChunksCalled()
	}

	return testscommon.NewCacherStub()
}

// PeerChangesBlocks -
func (holder *PoolsHolderStub) PeerChangesBlocks() storage.Cacher {
	if holder.PeerChangesBlocksCalled != nil {
		return holder.PeerChangesBlocksCalled()
	}

	return testscommon.NewCacherStub()
}

// SmartContracts -
func (holder *PoolsHolderStub) SmartContracts() storage.Cacher {
	if holder.SmartContractsCalled != nil {
		return holder.SmartContractsCalled()
	}

	return testscommon.NewCacherStub()
}

// ValidatorsInfo -
func (holder *PoolsHolderStub) ValidatorsInfo() storage.Cacher {
	if holder.ValidatorsInfoCalled != nil {
		return holder.ValidatorsInfoCalled()
	}

	return testscommon.NewCacherStub()
}

// IsInterfaceNil returns true if there is no value under the interface
func (holder *PoolsHolderStub) IsInterfaceNil() bool {
	return holder == nil
}
