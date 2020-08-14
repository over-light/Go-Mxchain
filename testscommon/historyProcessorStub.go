package testscommon

import (
	"github.com/ElrondNetwork/elrond-go/core/fullHistory"
	"github.com/ElrondNetwork/elrond-go/data"
)

// HistoryRepositoryStub -
type HistoryRepositoryStub struct {
	RecordBlockCalled                  func(blockHeaderHash []byte, blockHeader data.HeaderHandler, blockBody data.BodyHandler) error
	GetMiniblockMetadataByTxHashCalled func(hash []byte) (*fullHistory.MiniblockMetadataWithEpoch, error)
	GetEpochByHashCalled               func(hash []byte) (uint32, error)
	IsEnabledCalled                    func() bool
}

// RecordBlock -
func (hp *HistoryRepositoryStub) RecordBlock(blockHeaderHash []byte, blockHeader data.HeaderHandler, blockBody data.BodyHandler) error {
	if hp.RecordBlockCalled != nil {
		return hp.RecordBlockCalled(blockHeaderHash, blockHeader, blockBody)
	}
	return nil
}

// GetMiniblockMetadataByTxHash -
func (hp *HistoryRepositoryStub) GetMiniblockMetadataByTxHash(hash []byte) (*fullHistory.MiniblockMetadataWithEpoch, error) {
	if hp.GetMiniblockMetadataByTxHashCalled != nil {
		return hp.GetMiniblockMetadataByTxHashCalled(hash)
	}
	return nil, nil
}

// GetEpochByHash -
func (hp *HistoryRepositoryStub) GetEpochByHash(hash []byte) (uint32, error) {
	return hp.GetEpochByHashCalled(hash)
}

// IsEnabled -
func (hp *HistoryRepositoryStub) IsEnabled() bool {
	if hp.IsEnabledCalled != nil {
		return hp.IsEnabledCalled()
	}
	return true
}

// IsInterfaceNil -
func (hp *HistoryRepositoryStub) IsInterfaceNil() bool {
	return hp == nil
}
