package mock

import "math/big"

// BlockChainHookStub -
type BlockChainHookStub struct {
	AccountExtistsCalled    func(address []byte) (bool, error)
	NewAddressCalled        func(creatorAddress []byte, creatorNonce uint64, vmType []byte) ([]byte, error)
	GetBalanceCalled        func(address []byte) (*big.Int, error)
	GetNonceCalled          func(address []byte) (uint64, error)
	GetStorageDataCalled    func(accountsAddress []byte, index []byte) ([]byte, error)
	IsCodeEmptyCalled       func(address []byte) (bool, error)
	GetCodeCalled           func(address []byte) ([]byte, error)
	GetBlockHashCalled      func(nonce uint64) ([]byte, error)
	LastNonceCalled         func() uint64
	LastRoundCalled         func() uint64
	LastTimeStampCalled     func() uint64
	LastRandomSeedCalled    func() []byte
	LastEpochCalled         func() uint32
	GetStateRootHashCalled  func() []byte
	CurrentNonceCalled      func() uint64
	CurrentRoundCalled      func() uint64
	CurrentTimeStampCalled  func() uint64
	CurrentRandomSeedCalled func() []byte
	CurrentEpochCalled      func() uint32
}

// AccountExists -
func (b *BlockChainHookStub) AccountExists(address []byte) (bool, error) {
	if b.AccountExtistsCalled != nil {
		return b.AccountExtistsCalled(address)
	}
	return false, nil
}

// NewAddress -
func (b *BlockChainHookStub) NewAddress(creatorAddress []byte, creatorNonce uint64, vmType []byte) ([]byte, error) {
	if b.NewAddressCalled != nil {
		return b.NewAddressCalled(creatorAddress, creatorNonce, vmType)
	}
	return []byte("newAddress"), nil
}

// GetBalance -
func (b *BlockChainHookStub) GetBalance(address []byte) (*big.Int, error) {
	if b.GetBalanceCalled != nil {
		return b.GetBalanceCalled(address)
	}
	return big.NewInt(0), nil
}

// GetNonce -
func (b *BlockChainHookStub) GetNonce(address []byte) (uint64, error) {
	if b.GetNonceCalled != nil {
		return b.GetNonceCalled(address)
	}
	return 0, nil
}

// GetStorageData -
func (b *BlockChainHookStub) GetStorageData(accountAddress []byte, index []byte) ([]byte, error) {
	if b.GetStorageDataCalled != nil {
		return b.GetStorageDataCalled(accountAddress, index)
	}
	return nil, nil
}

// IsCodeEmpty -
func (b *BlockChainHookStub) IsCodeEmpty(address []byte) (bool, error) {
	if b.IsCodeEmptyCalled != nil {
		return b.IsCodeEmptyCalled(address)
	}
	return true, nil
}

// GetCode -
func (b *BlockChainHookStub) GetCode(address []byte) ([]byte, error) {
	if b.GetCodeCalled != nil {
		return b.GetCodeCalled(address)
	}
	return nil, nil
}

// GetBlockhash -
func (b *BlockChainHookStub) GetBlockhash(nonce uint64) ([]byte, error) {
	if b.GetBlockHashCalled != nil {
		return b.GetBlockHashCalled(nonce)
	}
	return []byte("roothash"), nil
}

// LastNonce -
func (b *BlockChainHookStub) LastNonce() uint64 {
	if b.LastNonceCalled != nil {
		return b.LastNonceCalled()
	}
	return 0
}

// LastRound -
func (b *BlockChainHookStub) LastRound() uint64 {
	if b.LastRoundCalled != nil {
		return b.LastRoundCalled()
	}
	return 0
}

// LastTimeStamp -
func (b *BlockChainHookStub) LastTimeStamp() uint64 {
	if b.LastTimeStampCalled != nil {
		return b.LastTimeStampCalled()
	}
	return 0
}

// LastRandomSeed -
func (b *BlockChainHookStub) LastRandomSeed() []byte {
	if b.LastRandomSeedCalled != nil {
		return b.LastRandomSeedCalled()
	}
	return []byte("seed")
}

// LastEpoch -
func (b *BlockChainHookStub) LastEpoch() uint32 {
	if b.LastEpochCalled != nil {
		return b.LastEpochCalled()
	}
	return 0
}

// GetStateRootHash -
func (b *BlockChainHookStub) GetStateRootHash() []byte {
	if b.GetStateRootHashCalled != nil {
		return b.GetStateRootHashCalled()
	}
	return []byte("roothash")
}

// CurrentNonce -
func (b *BlockChainHookStub) CurrentNonce() uint64 {
	if b.CurrentNonceCalled != nil {
		return b.CurrentNonceCalled()
	}
	return 0
}

// CurrentRound -
func (b *BlockChainHookStub) CurrentRound() uint64 {
	if b.CurrentRoundCalled != nil {
		return b.CurrentRoundCalled()
	}
	return 0
}

// CurrentTimeStamp -
func (b *BlockChainHookStub) CurrentTimeStamp() uint64 {
	if b.CurrentTimeStampCalled != nil {
		return b.CurrentTimeStampCalled()
	}
	return 0
}

// CurrentRandomSeed -
func (b *BlockChainHookStub) CurrentRandomSeed() []byte {
	if b.CurrentRandomSeedCalled != nil {
		return b.CurrentRandomSeedCalled()
	}
	return []byte("seedseed")
}

// CurrentEpoch -
func (b *BlockChainHookStub) CurrentEpoch() uint32 {
	if b.CurrentEpochCalled != nil {
		return b.CurrentEpochCalled()
	}
	return 0
}
