package economics

import (
	"math/big"
	"strconv"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/process"
)

var _ process.RewardsHandler = (*EconomicsData)(nil)
var _ process.ValidatorSettingsHandler = (*EconomicsData)(nil)
var _ process.FeeHandler = (*EconomicsData)(nil)

// EconomicsData will store information about economics
type EconomicsData struct {
	leaderPercentage         float64
	maxGasLimitPerBlock      uint64
	gasPerDataByte           uint64
	dataLimitForBaseCalc     uint64
	minGasPrice              uint64
	minGasLimit              uint64
	genesisNodePrice         *big.Int
	unBondPeriod             uint64
	developerPercentage      float64
	genesisTotalSupply       *big.Int
	minInflation             float64
	maxInflation             float64
	minStep                  *big.Int
	unJailPrice              *big.Int
	numNodes                 uint32
	auctionEnableNonce       uint64
	stakeEnableNonce         uint64
	numRoundsWithoutBleed    uint64
	bleedPercentagePerRound  float64
	maximumPercentageToBleed float64
}

// NewEconomicsData will create and object with information about economics parameters
func NewEconomicsData(economics *config.EconomicsConfig) (*EconomicsData, error) {
	data, err := convertValues(economics)
	if err != nil {
		return nil, err
	}

	err = checkValues(economics)
	if err != nil {
		return nil, err
	}

	if data.maxGasLimitPerBlock < data.minGasLimit {
		return nil, process.ErrInvalidMaxGasLimitPerBlock
	}

	return &EconomicsData{
		leaderPercentage:         economics.RewardsSettings.LeaderPercentage,
		maxGasLimitPerBlock:      data.maxGasLimitPerBlock,
		minGasPrice:              data.minGasPrice,
		minGasLimit:              data.minGasLimit,
		genesisNodePrice:         data.genesisNodePrice,
		unBondPeriod:             data.unBondPeriod,
		gasPerDataByte:           data.gasPerDataByte,
		dataLimitForBaseCalc:     data.dataLimitForBaseCalc,
		developerPercentage:      economics.RewardsSettings.DeveloperPercentage,
		minInflation:             economics.GlobalSettings.MinimumInflation,
		maxInflation:             economics.GlobalSettings.MaximumInflation,
		genesisTotalSupply:       data.genesisTotalSupply,
		minStep:                  data.minStep,
		numNodes:                 data.numNodes,
		auctionEnableNonce:       data.auctionEnableNonce,
		stakeEnableNonce:         data.stakeEnableNonce,
		numRoundsWithoutBleed:    data.numRoundsWithoutBleed,
		bleedPercentagePerRound:  data.bleedPercentagePerRound,
		maximumPercentageToBleed: data.maximumPercentageToBleed,
		unJailPrice:              data.unJailPrice,
	}, nil
}

func convertValues(economics *config.EconomicsConfig) (*EconomicsData, error) {
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

	genesisNodePrice := new(big.Int)
	genesisNodePrice, ok := genesisNodePrice.SetString(economics.ValidatorSettings.GenesisNodePrice, conversionBase)
	if !ok {
		return nil, process.ErrInvalidRewardsValue
	}

	unBondPeriod, err := strconv.ParseUint(economics.ValidatorSettings.UnBondPeriod, conversionBase, bitConversionSize)
	if err != nil {
		return nil, process.ErrInvalidUnBondPeriod
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

	genesisTotalSupply := new(big.Int)
	genesisTotalSupply, ok = genesisTotalSupply.SetString(economics.GlobalSettings.TotalSupply, conversionBase)
	if !ok {
		return nil, process.ErrInvalidGenesisTotalSupply
	}

	minStepValue := new(big.Int)
	minStepValue, ok = minStepValue.SetString(economics.ValidatorSettings.MinStepValue, conversionBase)
	if !ok {
		return nil, process.ErrInvalidMinStepValue
	}

	auctionEnableNonce, err := strconv.ParseUint(economics.ValidatorSettings.AuctionEnableNonce, conversionBase, bitConversionSize)
	if err != nil {
		return nil, process.ErrInvalidAuctionEnableNonce
	}

	stakeEnableNonce, err := strconv.ParseUint(economics.ValidatorSettings.StakeEnableNonce, conversionBase, bitConversionSize)
	if err != nil {
		return nil, process.ErrInvalidStakingEnableNonce
	}

	numRoundsWithoutBleed, err := strconv.ParseUint(economics.ValidatorSettings.NumRoundsWithoutBleed, conversionBase, bitConversionSize)
	if err != nil {
		return nil, process.ErrInvalidUnBondPeriod
	}

	maximumPercentageToBleed, err := strconv.ParseFloat(economics.ValidatorSettings.MaximumPercentageToBleed, bitConversionSize)
	if err != nil {
		return nil, process.ErrInvalidUnBondPeriod
	}

	bleedPercentagePerRound, err := strconv.ParseFloat(economics.ValidatorSettings.BleedPercentagePerRound, bitConversionSize)
	if err != nil {
		return nil, process.ErrInvalidUnBondPeriod
	}

	unJailPrice := new(big.Int)
	unJailPrice, ok = unJailPrice.SetString(economics.ValidatorSettings.UnJailValue, conversionBase)
	if !ok {
		return nil, process.ErrInvalidUnJailPrice
	}

	return &EconomicsData{
		minGasPrice:              minGasPrice,
		minGasLimit:              minGasLimit,
		genesisNodePrice:         genesisNodePrice,
		unBondPeriod:             unBondPeriod,
		maxGasLimitPerBlock:      maxGasLimitPerBlock,
		gasPerDataByte:           gasPerDataByte,
		dataLimitForBaseCalc:     dataLimitForBaseCalc,
		genesisTotalSupply:       genesisTotalSupply,
		minStep:                  minStepValue,
		numNodes:                 economics.ValidatorSettings.NumNodes,
		auctionEnableNonce:       auctionEnableNonce,
		stakeEnableNonce:         stakeEnableNonce,
		numRoundsWithoutBleed:    numRoundsWithoutBleed,
		bleedPercentagePerRound:  bleedPercentagePerRound,
		maximumPercentageToBleed: maximumPercentageToBleed,
		unJailPrice:              unJailPrice,
	}, nil
}

func checkValues(economics *config.EconomicsConfig) error {
	if isPercentageInvalid(economics.RewardsSettings.LeaderPercentage) ||
		isPercentageInvalid(economics.RewardsSettings.DeveloperPercentage) ||
		isPercentageInvalid(economics.GlobalSettings.MaximumInflation) ||
		isPercentageInvalid(economics.GlobalSettings.MaximumInflation) {
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

// MinInflationRate will return the minimum inflation rate
func (ed *EconomicsData) MinInflationRate() float64 {
	return ed.minInflation
}

// MaxInflationRate will return the maximum inflation rate
func (ed *EconomicsData) MaxInflationRate() float64 {
	return ed.maxInflation
}

// GenesisTotalSupply will return the genesis total supply
func (ed *EconomicsData) GenesisTotalSupply() *big.Int {
	return ed.genesisTotalSupply
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

// GenesisNodePrice will return the minimum stake value
func (ed *EconomicsData) GenesisNodePrice() *big.Int {
	return ed.genesisNodePrice
}

// UnBondPeriod will return the unbond period
func (ed *EconomicsData) UnBondPeriod() uint64 {
	return ed.unBondPeriod
}

// NumRoundsWithoutBleed will return the numRoundsWithoutBleed period
func (ed *EconomicsData) NumRoundsWithoutBleed() uint64 {
	return ed.numRoundsWithoutBleed
}

// BleedPercentagePerRound will return the bleedPercentagePerRound
func (ed *EconomicsData) BleedPercentagePerRound() float64 {
	return ed.bleedPercentagePerRound
}

// MaximumPercentageToBleed will return the maximumPercentageToBleed
func (ed *EconomicsData) MaximumPercentageToBleed() float64 {
	return ed.maximumPercentageToBleed
}

// MinStepValue returns the step value which is considered in the node price determination
func (ed *EconomicsData) MinStepValue() *big.Int {
	return ed.minStep
}

// UnJailValue returns the unjail value which is considered the price to bail out of jail
func (ed *EconomicsData) UnJailValue() *big.Int {
	return ed.unJailPrice
}

// TotalSupply returns the total supply of the protocol
func (ed *EconomicsData) TotalSupply() *big.Int {
	return ed.genesisTotalSupply
}

// NumNodes returns the total node number for current setting
func (ed *EconomicsData) NumNodes() uint32 {
	return ed.numNodes
}

// AuctionEnableNonce returns the nonce from which the auction process is enabled
func (ed *EconomicsData) AuctionEnableNonce() uint64 {
	return ed.auctionEnableNonce
}

// StakeEnableNonce returns the nonce from which the staking/unstaking function is enabled
func (ed *EconomicsData) StakeEnableNonce() uint64 {
	return ed.stakeEnableNonce
}

// IsInterfaceNil returns true if there is no value under the interface
func (ed *EconomicsData) IsInterfaceNil() bool {
	return ed == nil
}
