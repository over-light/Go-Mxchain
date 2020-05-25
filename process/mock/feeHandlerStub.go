package mock

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/process"
)

// FeeHandlerStub -
type FeeHandlerStub struct {
	SetMaxGasLimitPerBlockCalled func(maxGasLimitPerBlock uint64)
	SetMinGasPriceCalled         func(minGasPrice uint64)
	SetMinGasLimitCalled         func(minGasLimit uint64)
	MaxGasLimitPerBlockCalled    func() uint64
	ComputeGasLimitCalled        func(tx process.TransactionWithFeeHandler) uint64
	ComputeFeeCalled             func(tx process.TransactionWithFeeHandler) *big.Int
	CheckValidityTxValuesCalled  func(tx process.TransactionWithFeeHandler) error
	DeveloperPercentageCalled    func() float64
	MinGasPriceCalled            func() uint64
}

// MinGasPrice -
func (fhs *FeeHandlerStub) MinGasPrice() uint64 {
	if fhs.MinGasPriceCalled != nil {
		return fhs.MinGasPriceCalled()
	}
	return 0
}

// DeveloperPercentage -
func (fhs *FeeHandlerStub) DeveloperPercentage() float64 {
	return fhs.DeveloperPercentageCalled()
}

// SetMaxGasLimitPerBlock -
func (fhs *FeeHandlerStub) SetMaxGasLimitPerBlock(maxGasLimitPerBlock uint64) {
	fhs.SetMaxGasLimitPerBlockCalled(maxGasLimitPerBlock)
}

// SetMinGasPrice -
func (fhs *FeeHandlerStub) SetMinGasPrice(minGasPrice uint64) {
	fhs.SetMinGasPriceCalled(minGasPrice)
}

// SetMinGasLimit -
func (fhs *FeeHandlerStub) SetMinGasLimit(minGasLimit uint64) {
	fhs.SetMinGasLimitCalled(minGasLimit)
}

// MaxGasLimitPerBlock -
func (fhs *FeeHandlerStub) MaxGasLimitPerBlock(uint32) uint64 {
	return fhs.MaxGasLimitPerBlockCalled()
}

// ComputeGasLimit -
func (fhs *FeeHandlerStub) ComputeGasLimit(tx process.TransactionWithFeeHandler) uint64 {
	if fhs.ComputeGasLimitCalled != nil {
		return fhs.ComputeGasLimitCalled(tx)
	}
	return 0
}

// ComputeFee -
func (fhs *FeeHandlerStub) ComputeFee(tx process.TransactionWithFeeHandler) *big.Int {
	if fhs.ComputeFeeCalled != nil {
		return fhs.ComputeFeeCalled(tx)
	}
	return big.NewInt(0)
}

// CheckValidityTxValues -
func (fhs *FeeHandlerStub) CheckValidityTxValues(tx process.TransactionWithFeeHandler) error {
	if fhs.CheckValidityTxValuesCalled != nil {
		return fhs.CheckValidityTxValuesCalled(tx)
	}
	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (fhs *FeeHandlerStub) IsInterfaceNil() bool {
	return fhs == nil
}
