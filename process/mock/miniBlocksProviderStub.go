package mock

import (
	"github.com/ElrondNetwork/elrond-go/data/block"
)

// MiniBlocksProviderStub -
type MiniBlocksProviderStub struct {
	GetMiniBlocksCalled         func(hashes [][]byte) (block.MiniBlockSlice, [][]byte)
	GetMiniBlocksFromPoolCalled func(hashes [][]byte) (block.MiniBlockSlice, [][]byte)
}

// GetMiniBlocks -
func (mbps *MiniBlocksProviderStub) GetMiniBlocks(hashes [][]byte) (block.MiniBlockSlice, [][]byte) {
	if mbps.GetMiniBlocksCalled != nil {
		return mbps.GetMiniBlocksCalled(hashes)
	}
	return nil, nil
}

// GetMiniBlocksFromPool -
func (mbps *MiniBlocksProviderStub) GetMiniBlocksFromPool(hashes [][]byte) (block.MiniBlockSlice, [][]byte) {
	if mbps.GetMiniBlocksFromPoolCalled != nil {
		return mbps.GetMiniBlocksFromPoolCalled(hashes)
	}
	return nil, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (mbps *MiniBlocksProviderStub) IsInterfaceNil() bool {
	return mbps == nil
}
