package p2pQuota_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/statusHandler"
	"github.com/ElrondNetwork/elrond-go/statusHandler/mock"
	"github.com/ElrondNetwork/elrond-go/statusHandler/p2pQuota"
	"github.com/stretchr/testify/assert"
)

func TestNewP2PQuotaProcessor_NilStatusHandlerShouldErr(t *testing.T) {
	t.Parallel()

	pqp, err := p2pQuota.NewP2PQuotaProcessor(nil, "identifier")
	assert.True(t, check.IfNil(pqp))
	assert.Equal(t, statusHandler.ErrNilAppStatusHandler, err)
}

func TestNewP2PQuotaProcessor_ShouldWork(t *testing.T) {
	t.Parallel()

	pqp, err := p2pQuota.NewP2PQuotaProcessor(&mock.AppStatusHandlerStub{}, "identifier")
	assert.False(t, check.IfNil(pqp))
	assert.Nil(t, err)
}

//------- AddQuota

func TestP2PQuotaProcessor_AddQuotaShouldWork(t *testing.T) {
	t.Parallel()

	pqp, _ := p2pQuota.NewP2PQuotaProcessor(&mock.AppStatusHandlerStub{}, "identifier")
	nonExistingIdentifier := "non existing identifier"
	identifier := "identifier"
	numReceived := uint32(1)
	sizeReceived := uint64(2)
	numProcessed := uint32(3)
	sizeProcessed := uint64(4)

	pqp.AddQuota(identifier, numReceived, sizeReceived, numProcessed, sizeProcessed)

	nonExistentQuota := pqp.GetQuota(nonExistingIdentifier)
	assert.Nil(t, nonExistentQuota)

	quota := pqp.GetQuota(identifier)
	assert.Equal(t, numReceived, quota.NumReceived())
	assert.Equal(t, sizeReceived, quota.SizeReceived())
	assert.Equal(t, numProcessed, quota.NumProcessed())
	assert.Equal(t, sizeProcessed, quota.SizeProcessed())
}

//------- ResetStatistics

func TestP2PQuotaProcessor_ResetStatisticsShouldEmptyStatsAndCallSetOnAllMetrics(t *testing.T) {
	t.Parallel()

	identifier := "identifier"
	numReceived := uint64(1)
	sizeReceived := uint64(2)
	numProcessed := uint64(3)
	sizeProcessed := uint64(4)

	status := mock.NewAppStatusHandlerMock()
	quotaIdentifier := "identifier"
	pqp, _ := p2pQuota.NewP2PQuotaProcessor(status, quotaIdentifier)
	pqp.AddQuota(identifier, uint32(numReceived), sizeReceived, uint32(numProcessed), sizeProcessed)

	pqp.ResetStatistics()

	assert.Nil(t, pqp.GetQuota(identifier))

	numReceivers := uint64(1)
	checkPeerMetrics(t, status, numReceived, sizeReceived, numProcessed, sizeProcessed, quotaIdentifier)
	checkPeakPeerMetrics(t, status, numReceived, sizeReceived, numProcessed, sizeProcessed, quotaIdentifier)
	checkNumReceivers(t, status, numReceivers, numReceivers, quotaIdentifier)
}

func TestP2PQuotaProcessor_ResetStatisticsShouldSetPeerStatisticsTops(t *testing.T) {
	t.Parallel()

	identifier1 := "identifier"
	numReceived1 := uint64(10)
	sizeReceived1 := uint64(20)
	numProcessed1 := uint64(30)
	sizeProcessed1 := uint64(40)

	identifier2 := "identifier"
	numReceived2 := uint64(1)
	sizeReceived2 := uint64(2)
	numProcessed2 := uint64(3)
	sizeProcessed2 := uint64(4)

	status := mock.NewAppStatusHandlerMock()
	quotaIdentifier := "identifier"
	pqp, _ := p2pQuota.NewP2PQuotaProcessor(status, quotaIdentifier)
	pqp.AddQuota(identifier1, uint32(numReceived1), sizeReceived1, uint32(numProcessed1), sizeProcessed1)
	pqp.ResetStatistics()
	pqp.AddQuota(identifier2, uint32(numReceived2), sizeReceived2, uint32(numProcessed2), sizeProcessed2)

	pqp.ResetStatistics()

	numReceivers := uint64(1)
	checkPeerMetrics(t, status, numReceived2, sizeReceived2, numProcessed2, sizeProcessed2, quotaIdentifier)
	checkPeakPeerMetrics(t, status, numReceived1, sizeReceived1, numProcessed1, sizeProcessed1, quotaIdentifier)
	checkNumReceivers(t, status, numReceivers, numReceivers, quotaIdentifier)
}

func checkPeerMetrics(
	t *testing.T,
	status *mock.AppStatusHandlerMock,
	numReceived uint64,
	sizeReceived uint64,
	numProcessed uint64,
	sizeProcessed uint64,
	identifier string,
) {

	value := status.GetUint64(core.MetricP2PPeerNumReceivedMessages + "_" + identifier)
	assert.Equal(t, value, numReceived)

	value = status.GetUint64(core.MetricP2PPeerSizeReceivedMessages + "_" + identifier)
	assert.Equal(t, value, sizeReceived)

	value = status.GetUint64(core.MetricP2PPeerNumProcessedMessages + "_" + identifier)
	assert.Equal(t, value, numProcessed)

	value = status.GetUint64(core.MetricP2PPeerSizeProcessedMessages + "_" + identifier)
	assert.Equal(t, value, sizeProcessed)
}

func checkPeakPeerMetrics(
	t *testing.T,
	status *mock.AppStatusHandlerMock,
	numReceived uint64,
	sizeReceived uint64,
	numProcessed uint64,
	sizeProcessed uint64,
	identifier string,
) {

	value := status.GetUint64(core.MetricP2PPeakPeerNumReceivedMessages + "_" + identifier)
	assert.Equal(t, value, numReceived)

	value = status.GetUint64(core.MetricP2PPeakPeerSizeReceivedMessages + "_" + identifier)
	assert.Equal(t, value, sizeReceived)

	value = status.GetUint64(core.MetricP2PPeakPeerNumProcessedMessages + "_" + identifier)
	assert.Equal(t, value, numProcessed)

	value = status.GetUint64(core.MetricP2PPeakPeerSizeProcessedMessages + "_" + identifier)
	assert.Equal(t, value, sizeProcessed)
}

func checkNumReceivers(
	t *testing.T,
	status *mock.AppStatusHandlerMock,
	numReceiverPeers uint64,
	topNumReceiverPeers uint64,
	identifier string,
) {
	value := status.GetUint64(core.MetricP2PNumReceiverPeers + "_" + identifier)
	assert.Equal(t, value, numReceiverPeers)

	value = status.GetUint64(core.MetricP2PPeakNumReceiverPeers + "_" + identifier)
	assert.Equal(t, value, topNumReceiverPeers)
}
