package mock

import (
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/process/track"
)

type BlockTrackerHandlerMock struct {
	GetSelfHeadersCalled              func(headerHandler data.HeaderHandler) []*track.HeaderInfo
	ComputeNumPendingMiniBlocksCalled func(headers []data.HeaderHandler)
	ComputeLongestSelfChainCalled     func() (data.HeaderHandler, []byte, []data.HeaderHandler, [][]byte)
	SortHeadersFromNonceCalled        func(shardID uint32, nonce uint64) ([]data.HeaderHandler, [][]byte)
}

func (bthm *BlockTrackerHandlerMock) GetSelfHeaders(headerHandler data.HeaderHandler) []*track.HeaderInfo {
	if bthm.GetSelfHeadersCalled != nil {
		return bthm.GetSelfHeadersCalled(headerHandler)
	}

	return nil
}

func (bthm *BlockTrackerHandlerMock) ComputeNumPendingMiniBlocks(headers []data.HeaderHandler) {
	if bthm.ComputeNumPendingMiniBlocksCalled != nil {
		bthm.ComputeNumPendingMiniBlocksCalled(headers)
	}
}

func (bthm *BlockTrackerHandlerMock) ComputeLongestSelfChain() (data.HeaderHandler, []byte, []data.HeaderHandler, [][]byte) {
	if bthm.ComputeLongestSelfChainCalled != nil {
		return bthm.ComputeLongestSelfChainCalled()
	}

	return nil, nil, nil, nil
}

func (bthm *BlockTrackerHandlerMock) SortHeadersFromNonce(shardID uint32, nonce uint64) ([]data.HeaderHandler, [][]byte) {
	if bthm.SortHeadersFromNonceCalled != nil {
		return bthm.SortHeadersFromNonceCalled(shardID, nonce)
	}

	return nil, nil
}

func (bthm *BlockTrackerHandlerMock) IsInterfaceNil() bool {
	return bthm == nil
}
