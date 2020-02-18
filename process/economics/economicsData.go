package economics

import (
	"math/big"
	"strconv"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/process"
)

// EconomicsData will store information about economics
type EconomicsData struct {
	leaderPercentage     float64
	maxGasLimitPerBlock  uint64
	gasPerDataByte       uint64
	dataLimitForBaseCalc uint64
	minGasPrice          uint64
	minGasLimit          uint64
	communityAddress     string
	burnAddress          string
	stakeValue           *big.Int
	unBoundPeriod        uint64
	ratingsData          *RatingsData
	developerPercentage  float64
}

// NewEconomicsData will create and object with information about economics parameters
func NewEconomicsData(economics *config.ConfigEconomics) (*EconomicsData, error) {
	data, err := convertValues(economics)
	if err != nil {
		return nil, err
	}

	err = checkValues(economics)
	if err != nil {
		return nil, err
	}

	rd, err := NewRatingsData(economics.RatingSettings)
	if err != nil {
		return nil, err
	}

	if data.maxGasLimitPerBlock < data.minGasLimit {
		return nil, process.ErrInvalidMaxGasLimitPerBlock
	}

	return &EconomicsData{
		leaderPercentage:     economics.RewardsSettings.LeaderPercentage,
		maxGasLimitPerBlock:  data.maxGasLimitPerBlock,
		minGasPrice:          data.minGasPrice,
		minGasLimit:          data.minGasLimit,
		communityAddress:     economics.EconomicsAddresses.CommunityAddress,
		burnAddress:          economics.EconomicsAddresses.BurnAddress,
		stakeValue:           data.stakeValue,
		unBoundPeriod:        data.unBoundPeriod,
		gasPerDataByte:       data.gasPerDataByte,
		dataLimitForBaseCalc: data.dataLimitForBaseCalc,
		ratingsData:          rd,
		developerPercentage:  economics.RewardsSettings.DeveloperPercentage,
	}, nil
}

func convertValues(economics *config.ConfigEconomics) (*EconomicsData, error) {
	conversionBase := 10
	bitConversionSize := 64

	minGasPrice, err := strconv.ParseUint(economics.FeeSettings.MinGasPrice, conversionBase, bitConversionSize)
	if err != nil {
		return nil, process.ErrInvalidMinimumGasPrice
	}

	minGasLimit, err := strconv.ParseUint(economics.FeeSettings.MinGasLimit, conversionBase, bitConversionSize)
	if err != nil {
		return nil, process.ErrInvalidMinimumGasLimitForTx
	}

	stakeValue := new(big.Int)
	stakeValue, ok := stakeValue.SetString(economics.ValidatorSettings.StakeValue, conversionBase)
	if !ok {
		return nil, process.ErrInvalidRewardsValue
	}

	unBoundPeriod, err := strconv.ParseUint(economics.ValidatorSettings.UnBoundPeriod, conversionBase, bitConversionSize)
	if err != nil {
		return nil, process.ErrInvalidUnboundPeriod
	}

	maxGasLimitPerBlock, err := strconv.ParseUint(economics.FeeSettings.MaxGasLimitPerBlock, conversionBase, bitConversionSize)
	if err != nil {
		return nil, process.ErrInvalidMaxGasLimitPerBlock
	}

	gasPerDataByte, err := strconv.ParseUint(economics.FeeSettings.GasPerDataByte, conversionBase, bitConversionSize)
	if err != nil {
		return nil, process.ErrInvalidGasPerDataByte
	}

	dataLimitForBaseCalc, err := strconv.ParseUint(economics.FeeSettings.DataLimitForBaseCalc, conversionBase, bitConversionSize)
	if err != nil {
		return nil, process.ErrInvalidGasPerDataByte
	}

	return &EconomicsData{
		minGasPrice:          minGasPrice,
		minGasLimit:          minGasLimit,
		stakeValue:           stakeValue,
		unBoundPeriod:        unBoundPeriod,
		maxGasLimitPerBlock:  maxGasLimitPerBlock,
		gasPerDataByte:       gasPerDataByte,
		dataLimitForBaseCalc: dataLimitForBaseCalc,
	}, nil
}

func checkValues(economics *config.ConfigEconomics) error {
	if isPercentageInvalid(economics.RewardsSettings.LeaderPercentage) ||
		isPercentageInvalid(economics.RewardsSettings.DeveloperPercentage){
		return process.ErrInvalidRewardsPercentages
	}

	return nil
}

func isPercentageInvalid(percentage float64) bool {
	isLessThanZero := percentage < 0.0
	isGreaterThanOne := percentage > 1.0
	if isLessThanZero || isGreaterThanOne {
		return true
	}
	return false
}

// LeaderPercentage will return leader reward percentage
func (ed *EconomicsData) LeaderPercentage() float64 {
	return ed.leaderPercentage
}

// MinGasPrice will return min gas price
func (ed *EconomicsData) MinGasPrice() uint64 {
	return ed.minGasPrice
}

// ComputeFee computes the provided transaction's fee
func (ed *EconomicsData) ComputeFee(tx process.TransactionWithFeeHandler) *big.Int {
	gasPrice := big.NewInt(0).SetUint64(tx.GetGasPrice())
	gasLimit := big.NewInt(0).SetUint64(ed.ComputeGasLimit(tx))

	return gasPrice.Mul(gasPrice, gasLimit)
}

// CheckValidityTxValues checks if the provided transaction is economically correct
func (ed *EconomicsData) CheckValidityTxValues(tx process.TransactionWithFeeHandler) error {
	if ed.minGasPrice > tx.GetGasPrice() {
		return process.ErrInsufficientGasPriceInTx
	}

	requiredGasLimit := ed.ComputeGasLimit(tx)
	if requiredGasLimit > tx.GetGasLimit() {
		return process.ErrInsufficientGasLimitInTx
	}

	if requiredGasLimit > ed.maxGasLimitPerBlock {
		return process.ErrHigherGasLimitRequiredInTx
	}

	return nil
}

// MaxGasLimitPerBlock will return maximum gas limit allowed per block
func (ed *EconomicsData) MaxGasLimitPerBlock() uint64 {
	return ed.maxGasLimitPerBlock
}

// DeveloperPercentage will return the developer percentage value
func (ed *EconomicsData) DeveloperPercentage() float64 {
	return ed.developerPercentage
}

// ComputeGasLimit returns the gas limit need by the provided transaction in order to be executed
func (ed *EconomicsData) ComputeGasLimit(tx process.TransactionWithFeeHandler) uint64 {
	gasLimit := ed.minGasLimit

	dataLen := uint64(len(tx.GetData()))
	gasLimit += dataLen * ed.gasPerDataByte

	return gasLimit
}

// CommunityAddress will return community address
func (ed *EconomicsData) CommunityAddress() string {
	return ed.communityAddress
}

// BurnAddress will return burn address
func (ed *EconomicsData) BurnAddress() string {
	return ed.burnAddress
}

// StakeValue will return the minimum stake value
func (ed *EconomicsData) StakeValue() *big.Int {
	return ed.stakeValue
}

// UnBoundPeriod will return the unbound period
func (ed *EconomicsData) UnBoundPeriod() uint64 {
	return ed.unBoundPeriod
}

// IsInterfaceNil returns true if there is no value under the interface
func (ed *EconomicsData) IsInterfaceNil() bool {
	return ed == nil
}

// RatingsData will return the ratingsDataObject
func (ed *EconomicsData) RatingsData() *RatingsData {
	return ed.ratingsData
}
