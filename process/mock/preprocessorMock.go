package mock

import (
	"time"

	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/storage"
)

type PreProcessorMock struct {
	CreateBlockStartedCalled              func()
	IsDataPreparedCalled                  func(requestedTxs int, haveTime func() time.Duration) error
	RemoveTxBlockFromPoolsCalled          func(body block.Body, miniBlockPool storage.Cacher) error
	RestoreTxBlockIntoPoolsCalled         func(body block.Body, miniBlockHashes map[int][]byte, miniBlockPool storage.Cacher) (int, error)
	SaveTxBlockToStorageCalled            func(body block.Body) error
	ProcessBlockTransactionsCalled        func(body block.Body, round uint32, haveTime func() time.Duration) error
	RequestBlockTransactionsCalled        func(body block.Body) int
	CreateMarshalizedDataCalled           func(txHashes [][]byte) ([][]byte, error)
	RequestTransactionsForMiniBlockCalled func(mb block.MiniBlock) int
	ProcessMiniBlockCalled                func(miniBlock *block.MiniBlock, haveTime func() bool, round uint32) error
	CreateAndProcessMiniBlockCalled       func(sndShardId, dstShardId uint32, spaceRemained int, haveTime func() bool, round uint32) (*block.MiniBlock, error)
	GetAllCurrentUsedTxsCalled            func() map[string]data.TransactionHandler
}

func (ppm *PreProcessorMock) CreateBlockStarted() {
	if ppm.CreateBlockStartedCalled == nil {
		return
	}
	ppm.CreateBlockStartedCalled()
}

func (ppm *PreProcessorMock) IsDataPrepared(requestedTxs int, haveTime func() time.Duration) error {
	if ppm.IsDataPreparedCalled == nil {
		return nil
	}
	return ppm.IsDataPreparedCalled(requestedTxs, haveTime)
}

func (ppm *PreProcessorMock) RemoveTxBlockFromPools(body block.Body, miniBlockPool storage.Cacher) error {
	if ppm.RemoveTxBlockFromPoolsCalled == nil {
		return nil
	}
	return ppm.RemoveTxBlockFromPoolsCalled(body, miniBlockPool)
}

func (ppm *PreProcessorMock) RestoreTxBlockIntoPools(body block.Body, miniBlockHashes map[int][]byte, miniBlockPool storage.Cacher) (int, error) {
	if ppm.RestoreTxBlockIntoPoolsCalled == nil {
		return 0, nil
	}
	return ppm.RestoreTxBlockIntoPoolsCalled(body, miniBlockHashes, miniBlockPool)
}

func (ppm *PreProcessorMock) SaveTxBlockToStorage(body block.Body) error {
	if ppm.SaveTxBlockToStorageCalled == nil {
		return nil
	}
	return ppm.SaveTxBlockToStorageCalled(body)
}

func (ppm *PreProcessorMock) ProcessBlockTransactions(body block.Body, round uint32, haveTime func() time.Duration) error {
	if ppm.ProcessBlockTransactionsCalled == nil {
		return nil
	}
	return ppm.ProcessBlockTransactionsCalled(body, round, haveTime)
}

func (ppm *PreProcessorMock) RequestBlockTransactions(body block.Body) int {
	if ppm.RequestBlockTransactionsCalled == nil {
		return 0
	}
	return ppm.RequestBlockTransactionsCalled(body)
}

func (ppm *PreProcessorMock) CreateMarshalizedData(txHashes [][]byte) ([][]byte, error) {
	if ppm.CreateMarshalizedDataCalled == nil {
		return nil, nil
	}
	return ppm.CreateMarshalizedDataCalled(txHashes)
}

func (ppm *PreProcessorMock) RequestTransactionsForMiniBlock(mb block.MiniBlock) int {
	if ppm.RequestTransactionsForMiniBlockCalled == nil {
		return 0
	}
	return ppm.RequestTransactionsForMiniBlockCalled(mb)
}

func (ppm *PreProcessorMock) ProcessMiniBlock(miniBlock *block.MiniBlock, haveTime func() bool, round uint32) error {
	if ppm.ProcessMiniBlockCalled == nil {
		return nil
	}
	return ppm.ProcessMiniBlockCalled(miniBlock, haveTime, round)
}

func (ppm *PreProcessorMock) CreateAndProcessMiniBlock(sndShardId, dstShardId uint32, spaceRemained int, haveTime func() bool, round uint32) (*block.MiniBlock, error) {
	if ppm.CreateAndProcessMiniBlockCalled == nil {
		return nil, nil
	}
	return ppm.CreateAndProcessMiniBlockCalled(sndShardId, dstShardId, spaceRemained, haveTime, round)
}

func (ppm *PreProcessorMock) GetAllCurrentUsedTxs() map[string]data.TransactionHandler {
	if ppm.GetAllCurrentUsedTxsCalled == nil {
		return nil
	}
	return ppm.GetAllCurrentUsedTxsCalled()
}
