package presenter

import (
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/stretchr/testify/assert"
)

func TestPresenterStatusHandler_GetNonce(t *testing.T) {
	t.Parallel()

	nonce := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(core.MetricNonce, nonce)
	result := presenterStatusHandler.GetNonce()

	assert.Equal(t, nonce, result)
}

func TestPresenterStatusHandler_GetIsSyncing(t *testing.T) {
	t.Parallel()

	isSyncing := uint64(1)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(core.MetricIsSyncing, isSyncing)
	result := presenterStatusHandler.GetIsSyncing()

	assert.Equal(t, isSyncing, result)
}

func TestPresenterStatusHandler_GetTxPoolLoad(t *testing.T) {
	t.Parallel()

	txPoolLoad := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(core.MetricTxPoolLoad, txPoolLoad)
	result := presenterStatusHandler.GetTxPoolLoad()

	assert.Equal(t, txPoolLoad, result)
}

func TestPresenterStatusHandler_GetProbableHighestNonce(t *testing.T) {
	t.Parallel()

	probableHighestNonce := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(core.MetricProbableHighestNonce, probableHighestNonce)
	result := presenterStatusHandler.GetProbableHighestNonce()

	assert.Equal(t, probableHighestNonce, result)
}

func TestPresenterStatusHandler_GetSynchronizedRound(t *testing.T) {
	t.Parallel()

	synchronizedRound := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(core.MetricSynchronizedRound, synchronizedRound)
	result := presenterStatusHandler.GetSynchronizedRound()

	assert.Equal(t, synchronizedRound, result)
}

func TestPresenterStatusHandler_GetRoundTime(t *testing.T) {
	t.Parallel()

	roundTime := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(core.MetricRoundTime, roundTime)
	result := presenterStatusHandler.GetRoundTime()

	assert.Equal(t, roundTime, result)
}

func TestPresenterStatusHandler_GetLiveValidatorNodes(t *testing.T) {
	t.Parallel()

	numLiveValidatorNodes := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(core.MetricLiveValidatorNodes, numLiveValidatorNodes)
	result := presenterStatusHandler.GetLiveValidatorNodes()

	assert.Equal(t, numLiveValidatorNodes, result)
}

func TestPresenterStatusHandler_GetConnectedNodes(t *testing.T) {
	t.Parallel()

	numConnectedNodes := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(core.MetricConnectedNodes, numConnectedNodes)
	result := presenterStatusHandler.GetConnectedNodes()

	assert.Equal(t, numConnectedNodes, result)
}

func TestPresenterStatusHandler_GetNumConnectedPeers(t *testing.T) {
	t.Parallel()

	numConnectedPeers := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(core.MetricNumConnectedPeers, numConnectedPeers)
	result := presenterStatusHandler.GetNumConnectedPeers()

	assert.Equal(t, numConnectedPeers, result)
}

func TestPresenterStatusHandler_GetCurrentRound(t *testing.T) {
	t.Parallel()

	currentRound := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(core.MetricCurrentRound, currentRound)
	result := presenterStatusHandler.GetCurrentRound()

	assert.Equal(t, currentRound, result)
}

func TestPresenterStatusHandler_GetSynchronizationEstimation(t *testing.T) {
	t.Parallel()

	currentBlockNonce := uint64(10)
	probableHighestNonce := uint64(200)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.startTime = time.Now()
	presenterStatusHandler.startBlock = 0

	time.Sleep(time.Second)
	presenterStatusHandler.SetUInt64Value(core.MetricNonce, currentBlockNonce)
	presenterStatusHandler.SetUInt64Value(core.MetricProbableHighestNonce, probableHighestNonce)
	synchronizationEstimation := presenterStatusHandler.GetSynchronizationEstimation()

	// Node needs to synchronize 190 blocks and synchronization speed is 10 blocks/s
	// Synchronization estimation will be equals with (200-10)/10 seconds
	numBlocksThatNeedToBeSynchronized := int(probableHighestNonce - currentBlockNonce)
	blocksPerSecond := int(currentBlockNonce - presenterStatusHandler.startBlock)
	expectedTimeEstimation := secondsToHuman(numBlocksThatNeedToBeSynchronized / blocksPerSecond)

	assert.Equal(t, expectedTimeEstimation, synchronizationEstimation)
}

func TestPresenterStatusHandler_GetSynchronizationSpeed(t *testing.T) {
	t.Parallel()

	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(core.MetricNonce, 40)
	syncSpeed := presenterStatusHandler.GetSynchronizationSpeed()
	presenterStatusHandler.SetUInt64Value(core.MetricNonce, 50)
	syncSpeed = presenterStatusHandler.GetSynchronizationSpeed()

	assert.Equal(t, uint64(10), syncSpeed)
}

func TestPresenterStatusHandler_GetNumTxProcessed(t *testing.T) {
	t.Parallel()

	numTxProcessed := uint64(1000)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(core.MetricNumTxProcessed, numTxProcessed)
	result := presenterStatusHandler.GetNumTxProcessed()

	assert.Equal(t, numTxProcessed, result)
}

func TestPresenterStatusHandler_PrepareForCalculationSynchronizationTime(t *testing.T) {
	t.Parallel()

	blockNonce := uint64(40)
	presenterStatusHandler := NewPresenterStatusHandler()
	presenterStatusHandler.SetUInt64Value(core.MetricIsSyncing, 0)
	presenterStatusHandler.SetUInt64Value(core.MetricNonce, 1)
	presenterStatusHandler.PrepareForCalculationSynchronizationTime()
	time.Sleep(time.Second)
	presenterStatusHandler.SetUInt64Value(core.MetricNonce, blockNonce)
	presenterStatusHandler.SetUInt64Value(core.MetricIsSyncing, 1)
	time.Sleep(time.Second)

	presenterStatusHandler.mutEstimationTime.Lock()
	startBlockNonce := presenterStatusHandler.startBlock
	presenterStatusHandler.mutEstimationTime.Unlock()

	assert.Equal(t, blockNonce, startBlockNonce)
}
