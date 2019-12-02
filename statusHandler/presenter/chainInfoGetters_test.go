package presenter

import (
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core/constants"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/stretchr/testify/assert"
)

func TestPresenterStatusHandler_GetNonce(t *testing.T) {
	t.Parallel()

	nonce := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(constants.MetricNonce, nonce)
	result := presenterStatusHandler.GetNonce()

	assert.Equal(t, nonce, result)
}

func TestPresenterStatusHandler_GetIsSyncing(t *testing.T) {
	t.Parallel()

	isSyncing := uint64(1)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(constants.MetricIsSyncing, isSyncing)
	result := presenterStatusHandler.GetIsSyncing()

	assert.Equal(t, isSyncing, result)
}

func TestPresenterStatusHandler_GetTxPoolLoad(t *testing.T) {
	t.Parallel()

	txPoolLoad := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(constants.MetricTxPoolLoad, txPoolLoad)
	result := presenterStatusHandler.GetTxPoolLoad()

	assert.Equal(t, txPoolLoad, result)
}

func TestPresenterStatusHandler_GetProbableHighestNonce(t *testing.T) {
	t.Parallel()

	probableHighestNonce := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(constants.MetricProbableHighestNonce, probableHighestNonce)
	result := presenterStatusHandler.GetProbableHighestNonce()

	assert.Equal(t, probableHighestNonce, result)
}

func TestPresenterStatusHandler_GetSynchronizedRound(t *testing.T) {
	t.Parallel()

	synchronizedRound := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(constants.MetricSynchronizedRound, synchronizedRound)
	result := presenterStatusHandler.GetSynchronizedRound()

	assert.Equal(t, synchronizedRound, result)
}

func TestPresenterStatusHandler_GetRoundTime(t *testing.T) {
	t.Parallel()

	roundTime := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(constants.MetricRoundTime, roundTime)
	result := presenterStatusHandler.GetRoundTime()

	assert.Equal(t, roundTime, result)
}

func TestPresenterStatusHandler_GetLiveValidatorNodes(t *testing.T) {
	t.Parallel()

	numLiveValidatorNodes := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(constants.MetricLiveValidatorNodes, numLiveValidatorNodes)
	result := presenterStatusHandler.GetLiveValidatorNodes()

	assert.Equal(t, numLiveValidatorNodes, result)
}

func TestPresenterStatusHandler_GetConnectedNodes(t *testing.T) {
	t.Parallel()

	numConnectedNodes := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(constants.MetricConnectedNodes, numConnectedNodes)
	result := presenterStatusHandler.GetConnectedNodes()

	assert.Equal(t, numConnectedNodes, result)
}

func TestPresenterStatusHandler_GetNumConnectedPeers(t *testing.T) {
	t.Parallel()

	numConnectedPeers := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(constants.MetricNumConnectedPeers, numConnectedPeers)
	result := presenterStatusHandler.GetNumConnectedPeers()

	assert.Equal(t, numConnectedPeers, result)
}

func TestPresenterStatusHandler_GetCurrentRound(t *testing.T) {
	t.Parallel()

	currentRound := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(constants.MetricCurrentRound, currentRound)
	result := presenterStatusHandler.GetCurrentRound()

	assert.Equal(t, currentRound, result)
}

func TestPresenterStatusHandler_CalculateTimeToSynchronize(t *testing.T) {
	t.Parallel()

	currentBlockNonce := uint64(10)
	probableHighestNonce := uint64(200)
	synchronizationSpeed := uint64(10)
	presenterStatusHandler := NewPresenterStatusHandler()

	time.Sleep(time.Second)
	presenterStatusHandler.SetUInt64Value(constants.MetricNonce, currentBlockNonce)
	presenterStatusHandler.SetUInt64Value(constants.MetricProbableHighestNonce, probableHighestNonce)
	presenterStatusHandler.synchronizationSpeedHistory = append(presenterStatusHandler.synchronizationSpeedHistory, synchronizationSpeed)
	synchronizationEstimation := presenterStatusHandler.CalculateTimeToSynchronize()

	// Node needs to synchronize 190 blocks and synchronization speed is 10 blocks/s
	// Synchronization estimation will be equals with ((200-10)/10) seconds
	numBlocksThatNeedToBeSynchronized := probableHighestNonce - currentBlockNonce
	synchronizationEstimationExpected := numBlocksThatNeedToBeSynchronized / synchronizationSpeed

	assert.Equal(t, core.SecondsToHourMinSec(int(synchronizationEstimationExpected)), synchronizationEstimation)
}

func TestPresenterStatusHandler_CalculateSynchronizationSpeed(t *testing.T) {
	t.Parallel()

	initialNonce := uint64(10)
	currentNonce := uint64(20)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(constants.MetricNonce, initialNonce)
	syncSpeed := presenterStatusHandler.CalculateSynchronizationSpeed()
	presenterStatusHandler.SetUInt64Value(constants.MetricNonce, currentNonce)
	syncSpeed = presenterStatusHandler.CalculateSynchronizationSpeed()

	expectedSpeed := currentNonce - initialNonce
	assert.Equal(t, expectedSpeed, syncSpeed)
}

func TestPresenterStatusHandler_GetNumTxProcessed(t *testing.T) {
	t.Parallel()

	numTxProcessed := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(constants.MetricNumProcessedTxs, numTxProcessed)
	result := presenterStatusHandler.GetNumTxProcessed()

	assert.Equal(t, numTxProcessed, result)
}

func TestPresenterStatusHandler_GetNumShardHeadersInPool(t *testing.T) {
	t.Parallel()

	numShardHeadersInPool := uint64(100)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(constants.MetricNumShardHeadersFromPool, numShardHeadersInPool)
	result := presenterStatusHandler.GetNumShardHeadersInPool()

	assert.Equal(t, numShardHeadersInPool, result)
}

func TestNewPresenterStatusHandler_GetNumShardHeadersProcessed(t *testing.T) {
	t.Parallel()

	numShardHeadersProcessed := uint64(100)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(constants.MetricNumShardHeadersProcessed, numShardHeadersProcessed)
	result := presenterStatusHandler.GetNumShardHeadersProcessed()

	assert.Equal(t, numShardHeadersProcessed, result)
}
