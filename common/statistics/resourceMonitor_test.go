package statistics_test

import (
	errorsGo "errors"
	"fmt"
	"testing"

	stats "github.com/ElrondNetwork/elrond-go/common/statistics"
	"github.com/ElrondNetwork/elrond-go/common/statistics/disabled"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/stretchr/testify/assert"
)

func generateMockConfig() config.Config {
	return config.Config{
		ResourceStats: config.ResourceStatsConfig{
			RefreshIntervalInSec: 1,
		},
	}
}

func TestNewResourceMonitor_NilNetStatisticsShouldErr(t *testing.T) {
	t.Parallel()

	resourceMonitor, err := stats.NewResourceMonitor(
		generateMockConfig(),
		nil)

	assert.Equal(t, stats.ErrNilNetworkStatistics, err)
	assert.Nil(t, resourceMonitor)
}

func TestNewResourceMonitor_InvalidRefreshValueShouldErr(t *testing.T) {
	t.Parallel()

	resourceMonitor, err := stats.NewResourceMonitor(
		config.Config{
			ResourceStats: config.ResourceStatsConfig{
				RefreshIntervalInSec: 0,
			},
		},
		disabled.NewDisabledNetStatistics())

	assert.True(t, errorsGo.Is(err, stats.ErrInvalidRefreshIntervalValue))
	assert.Nil(t, resourceMonitor)
}

func TestResourceMonitor_NewResourceMonitorShouldWork(t *testing.T) {
	t.Parallel()

	resourceMonitor, err := stats.NewResourceMonitor(generateMockConfig(), disabled.NewDisabledNetStatistics())

	assert.Nil(t, err)
	assert.NotNil(t, resourceMonitor)
}

func TestResourceMonitor_GenerateStatisticsShouldPass(t *testing.T) {
	t.Parallel()

	resourceMonitor, err := stats.NewResourceMonitor(generateMockConfig(), disabled.NewDisabledNetStatistics())

	assert.Nil(t, err)
	statistics := resourceMonitor.GenerateStatistics()

	assert.NotNil(t, statistics)
}

func TestResourceMonitor_SaveStatisticsShouldNotPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r != nil {
			assert.Fail(t, fmt.Sprintf("test should not have paniced: %v", r))
		}
	}()

	resourceMonitor, err := stats.NewResourceMonitor(generateMockConfig(), disabled.NewDisabledNetStatistics())

	assert.Nil(t, err)
	resourceMonitor.SaveStatistics()
}
