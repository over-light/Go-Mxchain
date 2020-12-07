package external_test

import (
	"fmt"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/vmcommon"
	"github.com/ElrondNetwork/elrond-go/node/external"
	"github.com/ElrondNetwork/elrond-go/node/mock"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/stretchr/testify/assert"
)

func TestNewNodeApiResolver_NilSCQueryServiceShouldErr(t *testing.T) {
	t.Parallel()

	args := external.ApiResolverArgs{
		ScQueryService: nil,
		StatusMetrics:  &mock.StatusMetricsStub{},
		TxCostHandler:  &mock.TransactionCostEstimatorMock{},
	}
	nar, err := external.NewNodeApiResolver(args)

	assert.Nil(t, nar)
	assert.Equal(t, external.ErrNilSCQueryService, err)
}

func TestNewNodeApiResolver_NilStatusMetricsShouldErr(t *testing.T) {
	t.Parallel()

	args := external.ApiResolverArgs{
		ScQueryService: &mock.SCQueryServiceStub{},
		StatusMetrics:  nil,
		TxCostHandler:  &mock.TransactionCostEstimatorMock{},
	}
	nar, err := external.NewNodeApiResolver(args)

	assert.Nil(t, nar)
	assert.Equal(t, external.ErrNilStatusMetrics, err)
}

func TestNewNodeApiResolver_NilTransactionCostEstimator(t *testing.T) {
	t.Parallel()

	args := external.ApiResolverArgs{
		ScQueryService: &mock.SCQueryServiceStub{},
		StatusMetrics:  &mock.StatusMetricsStub{},
		TxCostHandler:  nil,
	}
	nar, err := external.NewNodeApiResolver(args)

	assert.Nil(t, nar)
	assert.Equal(t, external.ErrNilTransactionCostHandler, err)
}

func TestNewNodeApiResolver_ShouldWork(t *testing.T) {
	t.Parallel()

	args := external.ApiResolverArgs{
		ScQueryService: &mock.SCQueryServiceStub{},
		StatusMetrics:  &mock.StatusMetricsStub{},
		TxCostHandler:  &mock.TransactionCostEstimatorMock{},
	}
	nar, err := external.NewNodeApiResolver(args)

	assert.Nil(t, err)
	assert.False(t, check.IfNil(nar))
}

func TestNodeApiResolver_Close_ShouldWork(t *testing.T) {
	t.Parallel()

	calledClose := false
	args := external.ApiResolverArgs{
		ScQueryService: &mock.SCQueryServiceStub{},
		StatusMetrics:  &mock.StatusMetricsStub{},
		TxCostHandler:  &mock.TransactionCostEstimatorMock{},
		VmFactory:      &mock.VmMachinesContainerFactoryMock{},
		VmContainer: &mock.VMContainerMock{
			CloseCalled: func() error {
				calledClose = true
				return nil
			},
		},
	}

	nar, _ := external.NewNodeApiResolver(args)

	err := nar.Close()
	assert.Nil(t, err)
	assert.True(t, calledClose)
}

func TestNodeApiResolver_Close_OnErrorShouldError(t *testing.T) {
	t.Parallel()

	args := external.ApiResolverArgs{
		ScQueryService: &mock.SCQueryServiceStub{},
		StatusMetrics:  &mock.StatusMetricsStub{},
		TxCostHandler:  &mock.TransactionCostEstimatorMock{},
		VmFactory:      &mock.VmMachinesContainerFactoryMock{},
		VmContainer:    &mock.VMContainerMock{
			CloseCalled: func() error {
				return fmt.Errorf("error")
			},
		},
	}
	nar, _ := external.NewNodeApiResolver(args)

	err := nar.Close()
	assert.NotNil(t, err)
}

func TestNodeApiResolver_GetDataValueShouldCall(t *testing.T) {
	t.Parallel()

	wasCalled := false
	args := external.ApiResolverArgs{
		ScQueryService: &mock.SCQueryServiceStub{
			ExecuteQueryCalled: func(query *process.SCQuery) (vmOutput *vmcommon.VMOutput, e error) {
				wasCalled = true
				return &vmcommon.VMOutput{}, nil
			},
		},
		StatusMetrics:  &mock.StatusMetricsStub{},
		TxCostHandler:  &mock.TransactionCostEstimatorMock{},
	}

	nar, _ := external.NewNodeApiResolver(args)

	_, _ = nar.ExecuteSCQuery(&process.SCQuery{
		ScAddress: []byte{0},
		FuncName:  "",
	})

	assert.True(t, wasCalled)
}

func TestNodeApiResolver_StatusMetricsMapWithoutP2PShouldBeCalled(t *testing.T) {
	t.Parallel()

	wasCalled := false
	args := external.ApiResolverArgs{
		ScQueryService: &mock.SCQueryServiceStub{},
		StatusMetrics:  &mock.StatusMetricsStub{
			StatusMetricsMapWithoutP2PCalled: func() map[string]interface{} {
				wasCalled = true
				return nil
			},
		},
		TxCostHandler:  &mock.TransactionCostEstimatorMock{},
	}

	nar, _ := external.NewNodeApiResolver(args)
	_ = nar.StatusMetrics().StatusMetricsMapWithoutP2P()

	assert.True(t, wasCalled)
}

func TestNodeApiResolver_StatusP2pMetricsMapShouldBeCalled(t *testing.T) {
	t.Parallel()

	wasCalled := false
	args := external.ApiResolverArgs{
		ScQueryService: &mock.SCQueryServiceStub{},
		StatusMetrics:  &mock.StatusMetricsStub{
			StatusP2pMetricsMapCalled: func() map[string]interface{} {
				wasCalled = true
				return nil
			},
		},
		TxCostHandler:  &mock.TransactionCostEstimatorMock{},
	}

	nar, _ := external.NewNodeApiResolver(args)
	_ = nar.StatusMetrics().StatusP2pMetricsMap()

	assert.True(t, wasCalled)
}

func TestNodeApiResolver_StatusMetricsMapWhitoutP2PShouldBeCalled(t *testing.T) {
	t.Parallel()

	wasCalled := false
	args := external.ApiResolverArgs{
		ScQueryService: &mock.SCQueryServiceStub{},
		StatusMetrics:  &mock.StatusMetricsStub{
			StatusMetricsMapWithoutP2PCalled: func() map[string]interface{} {
				wasCalled = true
				return nil
			},
		},
		TxCostHandler:  &mock.TransactionCostEstimatorMock{},
	}
	nar, _ := external.NewNodeApiResolver(args)
	_ = nar.StatusMetrics().StatusMetricsMapWithoutP2P()

	assert.True(t, wasCalled)
}

func TestNodeApiResolver_StatusP2PMetricsMapShouldBeCalled(t *testing.T) {
	t.Parallel()

	wasCalled := false
	args := external.ApiResolverArgs{
		ScQueryService: &mock.SCQueryServiceStub{},
		StatusMetrics:  &mock.StatusMetricsStub{
			StatusP2pMetricsMapCalled: func() map[string]interface{} {
				wasCalled = true
				return nil
			},
		},
		TxCostHandler:  &mock.TransactionCostEstimatorMock{},
	}
	nar, _ := external.NewNodeApiResolver(args)
	_ = nar.StatusMetrics().StatusP2pMetricsMap()

	assert.True(t, wasCalled)
}

func TestNodeApiResolver_NetworkMetricsMapShouldBeCalled(t *testing.T) {
	t.Parallel()

	wasCalled := false
	args := external.ApiResolverArgs{
		ScQueryService: &mock.SCQueryServiceStub{},
		StatusMetrics:  &mock.StatusMetricsStub{
			NetworkMetricsCalled: func() map[string]interface{} {
				wasCalled = true
				return nil
			},
		},
		TxCostHandler:  &mock.TransactionCostEstimatorMock{},
	}
	nar, _ := external.NewNodeApiResolver(args)
	_ = nar.StatusMetrics().NetworkMetrics()

	assert.True(t, wasCalled)
}
