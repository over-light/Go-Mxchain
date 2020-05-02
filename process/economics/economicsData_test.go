package economics_test

import (
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/stretchr/testify/assert"
)

func createDummyEconomicsConfig() *config.EconomicsConfig {
	return &config.EconomicsConfig{
		GlobalSettings: config.GlobalSettings{
			TotalSupply:      "2000000000000000000000",
			MinimumInflation: 0,
			MaximumInflation: 0.05,
		},
		RewardsSettings: config.RewardsSettings{
			LeaderPercentage:    0.1,
			CommunityPercentage: 0.1,
			CommunityAddress:    "erd1932eft30w753xyvme8d49qejgkjc09n5e49w4mwdjtm0neld797su0dlxp",
		},
		FeeSettings: config.FeeSettings{
			MaxGasLimitPerBlock:  "100000",
			MinGasPrice:          "18446744073709551615",
			MinGasLimit:          "500",
			GasPerDataByte:       "1",
			DataLimitForBaseCalc: "100000000",
		},
		ValidatorSettings: config.ValidatorSettings{
			GenesisNodePrice:         "500000000",
			UnBondPeriod:             "100000",
			TotalSupply:              "200000000000",
			MinStepValue:             "100000",
			AuctionEnableNonce:       "100000",
			StakeEnableNonce:         "100000",
			NumRoundsWithoutBleed:    "1000",
			MaximumPercentageToBleed: "0.5",
			BleedPercentagePerRound:  "0.00001",
			UnJailValue:              "1000",
		},
	}
}

func TestNewEconomicsData_InvalidMaxGasLimitPerBlockShouldErr(t *testing.T) {
	t.Parallel()

	economicsConfig := createDummyEconomicsConfig()
	badGasLimitPerBlock := []string{
		"-1",
		"-100000000000000000000",
		"badValue",
		"",
		"#########",
		"11112S",
		"1111O0000",
		"10ERD",
		"10000000000000000000000000000000000000000000000000000000000000",
	}

	for _, gasLimitPerBlock := range badGasLimitPerBlock {
		economicsConfig.FeeSettings.MaxGasLimitPerBlock = gasLimitPerBlock
		_, err := economics.NewEconomicsData(economicsConfig)
		assert.Equal(t, process.ErrInvalidMaxGasLimitPerBlock, err)
	}

}

func TestNewEconomicsData_InvalidMinGasPriceShouldErr(t *testing.T) {
	t.Parallel()

	economicsConfig := createDummyEconomicsConfig()
	badGasPrice := []string{
		"-1",
		"-100000000000000000000",
		"badValue",
		"",
		"#########",
		"11112S",
		"1111O0000",
		"10ERD",
		"10000000000000000000000000000000000000000000000000000000000000",
	}

	for _, gasPrice := range badGasPrice {
		economicsConfig.FeeSettings.MinGasPrice = gasPrice
		_, err := economics.NewEconomicsData(economicsConfig)
		assert.Equal(t, process.ErrInvalidMinimumGasPrice, err)
	}

}

func TestNewEconomicsData_InvalidMinGasLimitShouldErr(t *testing.T) {
	t.Parallel()

	economicsConfig := createDummyEconomicsConfig()
	bagMinGasLimit := []string{
		"-1",
		"-100000000000000000000",
		"badValue",
		"",
		"#########",
		"11112S",
		"1111O0000",
		"10ERD",
		"10000000000000000000000000000000000000000000000000000000000000",
	}

	for _, minGasLimit := range bagMinGasLimit {
		economicsConfig.FeeSettings.MinGasLimit = minGasLimit
		_, err := economics.NewEconomicsData(economicsConfig)
		assert.Equal(t, process.ErrInvalidMinimumGasLimitForTx, err)
	}

}

func TestNewEconomicsData_InvalidLeaderPercentageShouldErr(t *testing.T) {
	t.Parallel()

	economicsConfig := createDummyEconomicsConfig()
	economicsConfig.RewardsSettings.LeaderPercentage = -0.1

	_, err := economics.NewEconomicsData(economicsConfig)
	assert.Equal(t, process.ErrInvalidRewardsPercentages, err)

}

func TestNewEconomicsData_ShouldWork(t *testing.T) {
	t.Parallel()

	economicsConfig := createDummyEconomicsConfig()
	economicsData, _ := economics.NewEconomicsData(economicsConfig)
	assert.NotNil(t, economicsData)
}

func TestEconomicsData_LeaderPercentage(t *testing.T) {
	t.Parallel()

	leaderPercentage := 0.40
	economicsConfig := createDummyEconomicsConfig()
	economicsConfig.RewardsSettings.LeaderPercentage = leaderPercentage
	economicsData, _ := economics.NewEconomicsData(economicsConfig)

	value := economicsData.LeaderPercentage()
	assert.Equal(t, leaderPercentage, value)
}

func TestEconomicsData_ComputeFeeNoTxData(t *testing.T) {
	t.Parallel()

	gasPrice := uint64(500)
	minGasLimit := uint64(12)
	economicsConfig := createDummyEconomicsConfig()
	economicsConfig.FeeSettings.MinGasLimit = strconv.FormatUint(minGasLimit, 10)
	economicsData, _ := economics.NewEconomicsData(economicsConfig)
	tx := &transaction.Transaction{
		GasPrice: gasPrice,
		GasLimit: minGasLimit,
	}

	cost := economicsData.ComputeFee(tx)

	expectedCost := big.NewInt(0).SetUint64(gasPrice)
	expectedCost.Mul(expectedCost, big.NewInt(0).SetUint64(minGasLimit))
	assert.Equal(t, expectedCost, cost)
}

func TestEconomicsData_ComputeFeeWithTxData(t *testing.T) {
	t.Parallel()

	gasPrice := uint64(500)
	minGasLimit := uint64(12)
	txData := "text to be notarized"
	economicsConfig := createDummyEconomicsConfig()
	economicsConfig.FeeSettings.MinGasLimit = strconv.FormatUint(minGasLimit, 10)
	economicsData, _ := economics.NewEconomicsData(economicsConfig)
	tx := &transaction.Transaction{
		GasPrice: gasPrice,
		GasLimit: minGasLimit,
		Data:     []byte(txData),
	}

	cost := economicsData.ComputeFee(tx)

	expectedGasLimit := big.NewInt(0).SetUint64(minGasLimit)
	expectedGasLimit.Add(expectedGasLimit, big.NewInt(int64(len(txData))))
	expectedCost := big.NewInt(0).SetUint64(gasPrice)
	expectedCost.Mul(expectedCost, expectedGasLimit)
	assert.Equal(t, expectedCost, cost)
}

func TestEconomicsData_TxWithLowerGasPriceShouldErr(t *testing.T) {
	t.Parallel()

	minGasPrice := uint64(500)
	minGasLimit := uint64(12)
	economicsConfig := createDummyEconomicsConfig()
	economicsConfig.FeeSettings.MinGasPrice = fmt.Sprintf("%d", minGasPrice)
	economicsConfig.FeeSettings.MinGasLimit = fmt.Sprintf("%d", minGasLimit)
	economicsData, _ := economics.NewEconomicsData(economicsConfig)
	tx := &transaction.Transaction{
		GasPrice: minGasPrice - 1,
		GasLimit: minGasLimit,
	}

	err := economicsData.CheckValidityTxValues(tx)

	assert.Equal(t, process.ErrInsufficientGasPriceInTx, err)
}

func TestEconomicsData_TxWithLowerGasLimitShouldErr(t *testing.T) {
	t.Parallel()

	minGasPrice := uint64(500)
	minGasLimit := uint64(12)
	economicsConfig := createDummyEconomicsConfig()
	economicsConfig.FeeSettings.MinGasPrice = fmt.Sprintf("%d", minGasPrice)
	economicsConfig.FeeSettings.MinGasLimit = fmt.Sprintf("%d", minGasLimit)
	economicsData, _ := economics.NewEconomicsData(economicsConfig)
	tx := &transaction.Transaction{
		GasPrice: minGasPrice,
		GasLimit: minGasLimit - 1,
	}

	err := economicsData.CheckValidityTxValues(tx)

	assert.Equal(t, process.ErrInsufficientGasLimitInTx, err)
}

func TestEconomicsData_TxWithHigherGasLimitShouldErr(t *testing.T) {
	t.Parallel()

	minGasPrice := uint64(500)
	minGasLimit := uint64(12)
	maxGasLimitPerBlock := minGasLimit
	economicsConfig := createDummyEconomicsConfig()
	economicsConfig.FeeSettings.MaxGasLimitPerBlock = fmt.Sprintf("%d", maxGasLimitPerBlock)
	economicsConfig.FeeSettings.MinGasPrice = fmt.Sprintf("%d", minGasPrice)
	economicsConfig.FeeSettings.MinGasLimit = fmt.Sprintf("%d", minGasLimit)
	economicsData, _ := economics.NewEconomicsData(economicsConfig)
	tx := &transaction.Transaction{
		GasPrice: minGasPrice,
		GasLimit: minGasLimit + 1,
		Data:     []byte("1"),
	}

	err := economicsData.CheckValidityTxValues(tx)

	assert.Equal(t, process.ErrHigherGasLimitRequiredInTx, err)
}

func TestEconomicsData_TxWithWithEqualGasPriceLimitShouldWork(t *testing.T) {
	t.Parallel()

	minGasPrice := uint64(500)
	minGasLimit := uint64(12)
	maxGasLimitPerBlock := minGasLimit
	economicsConfig := createDummyEconomicsConfig()
	economicsConfig.FeeSettings.MaxGasLimitPerBlock = fmt.Sprintf("%d", maxGasLimitPerBlock)
	economicsConfig.FeeSettings.MinGasPrice = fmt.Sprintf("%d", minGasPrice)
	economicsConfig.FeeSettings.MinGasLimit = fmt.Sprintf("%d", minGasLimit)
	economicsData, _ := economics.NewEconomicsData(economicsConfig)
	tx := &transaction.Transaction{
		GasPrice: minGasPrice,
		GasLimit: minGasLimit,
	}

	err := economicsData.CheckValidityTxValues(tx)

	assert.Nil(t, err)
}

func TestEconomicsData_TxWithWithMoreGasPriceLimitShouldWork(t *testing.T) {
	t.Parallel()

	minGasPrice := uint64(500)
	minGasLimit := uint64(12)
	maxGasLimitPerBlock := minGasLimit + 1
	economicsConfig := createDummyEconomicsConfig()
	economicsConfig.FeeSettings.MaxGasLimitPerBlock = fmt.Sprintf("%d", maxGasLimitPerBlock)
	economicsConfig.FeeSettings.MinGasPrice = fmt.Sprintf("%d", minGasPrice)
	economicsConfig.FeeSettings.MinGasLimit = fmt.Sprintf("%d", minGasLimit)
	economicsData, _ := economics.NewEconomicsData(economicsConfig)
	tx := &transaction.Transaction{
		GasPrice: minGasPrice + 1,
		GasLimit: minGasLimit + 1,
	}

	err := economicsData.CheckValidityTxValues(tx)

	assert.Nil(t, err)
}
