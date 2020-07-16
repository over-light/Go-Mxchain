package factory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/appStatusPolling"
	"github.com/ElrondNetwork/elrond-go/core/indexer"
	"github.com/ElrondNetwork/elrond-go/core/serviceContainer"
	"github.com/ElrondNetwork/elrond-go/core/statistics"
	"github.com/ElrondNetwork/elrond-go/core/statistics/machine"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

var _ ComponentHandler = (*managedStatusComponents)(nil)
var _ StatusComponentsHolder = (*managedStatusComponents)(nil)
var _ StatusComponentsHandler = (*managedStatusComponents)(nil)

type managedStatusComponents struct {
	*statusComponents
	statusComponentsFactory *statusComponentsFactory
	cancelFunc              func()
	mutStatusComponents     sync.RWMutex
}

// NewManagedStatusComponents returns a new instance of managedStatusComponents
func NewManagedStatusComponents(args StatusComponentsFactoryArgs) (*managedStatusComponents, error) {
	scf, err := NewStatusComponentsFactory(args)
	if err != nil {
		return nil, err
	}

	return &managedStatusComponents{
		statusComponents:        nil,
		statusComponentsFactory: scf,
		cancelFunc:              nil,
		mutStatusComponents:     sync.RWMutex{},
	}, nil
}

// Create will create the status components
func (m *managedStatusComponents) Create() error {
	components, err := m.statusComponentsFactory.Create()
	if err != nil {
		return err
	}

	m.mutStatusComponents.Lock()
	m.statusComponents = components
	m.mutStatusComponents.Unlock()

	return nil
}

// SetForkDetector sets the fork detector
func (m *managedStatusComponents) SetForkDetector(forkDetector process.ForkDetector) {
	m.mutStatusComponents.Lock()
	m.statusComponentsFactory.forkDetector = forkDetector
	m.mutStatusComponents.Unlock()
}

// StartPolling starts polling for the updated status
func (m *managedStatusComponents) StartPolling() error {
	var ctx context.Context
	m.mutStatusComponents.Lock()
	ctx, m.cancelFunc = context.WithCancel(context.Background())
	m.mutStatusComponents.Unlock()

	err := m.startStatusPolling(ctx)
	if err != nil {
		return err
	}

	err = m.startMachineStatisticsPolling(ctx)
	if err != nil {
		return err
	}

	return nil
}

// Close will close all the underlying components
func (m *managedStatusComponents) Close() error {
	m.mutStatusComponents.Lock()
	defer m.mutStatusComponents.Unlock()

	if m.statusComponents != nil {
		err := m.statusComponents.Close()
		if err != nil {
			return err
		}
		m.statusComponents = nil
	}

	return nil
}

// TpsBenchmark returns the tps benchmark handler
func (m *managedStatusComponents) TpsBenchmark() statistics.TPSBenchmark {
	m.mutStatusComponents.RLock()
	defer m.mutStatusComponents.RUnlock()

	if m.statusComponents == nil {
		return nil
	}

	return m.statusComponents.tpsBenchmark
}

// ElasticIndexer returns the elastic indexer handler
func (m *managedStatusComponents) ElasticIndexer() indexer.Indexer {
	m.mutStatusComponents.RLock()
	defer m.mutStatusComponents.RUnlock()

	if m.statusComponents == nil {
		return nil
	}

	return m.statusComponents.elasticIndexer
}

// SoftwareVersionChecker returns the software version checker handler
func (m *managedStatusComponents) SoftwareVersionChecker() statistics.SoftwareVersionChecker {
	m.mutStatusComponents.RLock()
	defer m.mutStatusComponents.RUnlock()

	if m.statusComponents == nil {
		return nil
	}

	return m.statusComponents.softwareVersion
}

// StatusHandler returns the status handler
func (m *managedStatusComponents) StatusHandler() core.AppStatusHandler {
	m.mutStatusComponents.RLock()
	defer m.mutStatusComponents.RUnlock()

	if m.statusComponents == nil {
		return nil
	}

	return m.statusComponents.statusHandler
}

// ServiceContainer returns a ServiceContainer instance for the assigned shard
func (m *managedStatusComponents) ServiceContainer() (serviceContainer.Core, error) {
	var err error
	shardCoordinator := m.statusComponentsFactory.shardCoordinator

	var coreServiceContainer serviceContainer.Core

	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		coreServiceContainer, err = serviceContainer.NewServiceContainer(
			serviceContainer.WithIndexer(m.statusComponents.elasticIndexer),
		)
		if err != nil {
			return nil, err
		}
		return coreServiceContainer, nil
	}

	if shardCoordinator.SelfId() == core.MetachainShardId {
		var indexerToUse indexer.Indexer
		indexerToUse = indexer.NewNilIndexer()
		if m.statusComponents.elasticIndexer != nil {
			indexerToUse = m.statusComponents.elasticIndexer
		}
		coreServiceContainer, err = serviceContainer.NewServiceContainer(
			serviceContainer.WithIndexer(indexerToUse),
			serviceContainer.WithTPSBenchmark(m.statusComponents.tpsBenchmark))
		if err != nil {
			return nil, err
		}
		return coreServiceContainer, nil
	}
	return nil, errors.New("could not init core service container")
}

// IsInterfaceNil returns true if there is no value under the interface
func (m *managedStatusComponents) IsInterfaceNil() bool {
	return m == nil
}

func (m *managedStatusComponents) startStatusPolling(ctx context.Context) error {
	// TODO: inject the context to the AppStatusPolling
	appStatusPollingHandler, err := appStatusPolling.NewAppStatusPolling(
		m.statusComponentsFactory.coreComponents.StatusHandler(),
		time.Duration(m.statusComponentsFactory.config.GeneralSettings.StatusPollingIntervalSec)*time.Second,
	)
	if err != nil {
		return errors.New("cannot init AppStatusPolling")
	}

	err = registerPollConnectedPeers(appStatusPollingHandler, m.statusComponentsFactory.networkComponents)
	if err != nil {
		return err
	}

	err = registerPollProbableHighestNonce(appStatusPollingHandler, m.statusComponentsFactory.forkDetector)
	if err != nil {
		return err
	}

	err = registerShardsInformation(appStatusPollingHandler, m.statusComponentsFactory.shardCoordinator)
	if err != nil {
		return err
	}

	appStatusPollingHandler.Poll(ctx)

	return nil
}

func registerPollConnectedPeers(
	appStatusPollingHandler *appStatusPolling.AppStatusPolling,
	networkComponents NetworkComponentsHolder,
) error {

	p2pMetricsHandlerFunc := func(appStatusHandler core.AppStatusHandler) {
		computeNumConnectedPeers(appStatusHandler, networkComponents)
		computeConnectedPeers(appStatusHandler, networkComponents)
	}

	err := appStatusPollingHandler.RegisterPollingFunc(p2pMetricsHandlerFunc)
	if err != nil {
		return errors.New("cannot register handler func for num of connected peers")
	}

	return nil
}

func registerShardsInformation(
	appStatusPollingHandler *appStatusPolling.AppStatusPolling,
	coordinator sharding.Coordinator,
) error {

	computeShardsInfo := func(appStatusHandler core.AppStatusHandler) {
		shardId := uint64(coordinator.SelfId())
		numOfShards := uint64(coordinator.NumberOfShards())

		appStatusHandler.SetUInt64Value(core.MetricShardId, shardId)
		appStatusHandler.SetUInt64Value(core.MetricNumShardsWithoutMetacahin, numOfShards)
	}

	err := appStatusPollingHandler.RegisterPollingFunc(computeShardsInfo)
	if err != nil {
		return fmt.Errorf("%w, cannot register handler func for shards information", err)
	}

	return nil
}

func computeNumConnectedPeers(
	appStatusHandler core.AppStatusHandler,
	networkComponents NetworkComponentsHolder,
) {
	numOfConnectedPeers := uint64(len(networkComponents.NetworkMessenger().ConnectedAddresses()))
	appStatusHandler.SetUInt64Value(core.MetricNumConnectedPeers, numOfConnectedPeers)
}

func computeConnectedPeers(
	appStatusHandler core.AppStatusHandler,
	networkComponents NetworkComponentsHolder,
) {
	peersInfo := networkComponents.NetworkMessenger().GetConnectedPeersInfo()

	peerClassification := fmt.Sprintf("intraVal:%d,crossVal:%d,intraObs:%d,crossObs:%d,unknown:%d,",
		len(peersInfo.IntraShardValidators),
		len(peersInfo.CrossShardValidators),
		len(peersInfo.IntraShardObservers),
		len(peersInfo.CrossShardObservers),
		len(peersInfo.UnknownPeers),
	)
	appStatusHandler.SetStringValue(core.MetricNumConnectedPeersClassification, peerClassification)
	appStatusHandler.SetStringValue(core.MetricP2PNumConnectedPeersClassification, peerClassification)

	setP2pConnectedPeersMetrics(appStatusHandler, peersInfo)
	setCurrentP2pNodeAddresses(appStatusHandler, networkComponents)
}

func setP2pConnectedPeersMetrics(appStatusHandler core.AppStatusHandler, info *p2p.ConnectedPeersInfo) {
	appStatusHandler.SetStringValue(core.MetricP2PUnknownPeers, sliceToString(info.UnknownPeers))
	appStatusHandler.SetStringValue(core.MetricP2PIntraShardValidators, sliceToString(info.IntraShardValidators))
	appStatusHandler.SetStringValue(core.MetricP2PIntraShardObservers, sliceToString(info.IntraShardObservers))
	appStatusHandler.SetStringValue(core.MetricP2PCrossShardValidators, sliceToString(info.CrossShardValidators))
	appStatusHandler.SetStringValue(core.MetricP2PCrossShardObservers, sliceToString(info.CrossShardObservers))
}

func sliceToString(input []string) string {
	output := ""
	for _, str := range input {
		output += str + ","
	}

	return output
}

func setCurrentP2pNodeAddresses(
	appStatusHandler core.AppStatusHandler,
	networkComponents NetworkComponentsHolder,
) {
	appStatusHandler.SetStringValue(core.MetricP2PPeerInfo, sliceToString(networkComponents.NetworkMessenger().Addresses()))
}

func registerPollProbableHighestNonce(
	appStatusPollingHandler *appStatusPolling.AppStatusPolling,
	forkDetector process.ForkDetector,
) error {

	probableHighestNonceHandlerFunc := func(appStatusHandler core.AppStatusHandler) {
		probableHigherNonce := forkDetector.ProbableHighestNonce()
		appStatusHandler.SetUInt64Value(core.MetricProbableHighestNonce, probableHigherNonce)
	}

	err := appStatusPollingHandler.RegisterPollingFunc(probableHighestNonceHandlerFunc)
	if err != nil {
		return errors.New("cannot register handler func for forkdetector's probable higher nonce")
	}

	return nil
}

func (m *managedStatusComponents) startMachineStatisticsPolling(ctx context.Context) error {
	appStatusPollingHandler, err := appStatusPolling.NewAppStatusPolling(m.statusComponentsFactory.coreComponents.StatusHandler(), time.Second)
	if err != nil {
		return errors.New("cannot init AppStatusPolling")
	}

	err = registerCpuStatistics(appStatusPollingHandler, ctx)
	if err != nil {
		return err
	}

	err = registerMemStatistics(appStatusPollingHandler, ctx)
	if err != nil {
		return err
	}

	err = registerNetStatistics(appStatusPollingHandler, ctx)
	if err != nil {
		return err
	}

	appStatusPollingHandler.Poll(ctx)

	return nil
}

func registerMemStatistics(appStatusPollingHandler *appStatusPolling.AppStatusPolling, _ context.Context) error {
	return appStatusPollingHandler.RegisterPollingFunc(func(appStatusHandler core.AppStatusHandler) {
		mem := machine.AcquireMemStatistics()

		appStatusHandler.SetUInt64Value(core.MetricMemLoadPercent, mem.PercentUsed)
		appStatusHandler.SetUInt64Value(core.MetricMemTotal, mem.Total)
		appStatusHandler.SetUInt64Value(core.MetricMemUsedGolang, mem.UsedByGolang)
		appStatusHandler.SetUInt64Value(core.MetricMemUsedSystem, mem.UsedBySystem)
		appStatusHandler.SetUInt64Value(core.MetricMemHeapInUse, mem.HeapInUse)
		appStatusHandler.SetUInt64Value(core.MetricMemStackInUse, mem.StackInUse)
	})
}

func registerNetStatistics(appStatusPollingHandler *appStatusPolling.AppStatusPolling, ctx context.Context) error {
	netStats := &machine.NetStatistics{}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			netStats.ComputeStatistics()
		}
	}()

	return appStatusPollingHandler.RegisterPollingFunc(func(appStatusHandler core.AppStatusHandler) {
		appStatusHandler.SetUInt64Value(core.MetricNetworkRecvBps, netStats.BpsRecv())
		appStatusHandler.SetUInt64Value(core.MetricNetworkRecvBpsPeak, netStats.BpsRecvPeak())
		appStatusHandler.SetUInt64Value(core.MetricNetworkRecvPercent, netStats.PercentRecv())

		appStatusHandler.SetUInt64Value(core.MetricNetworkSentBps, netStats.BpsSent())
		appStatusHandler.SetUInt64Value(core.MetricNetworkSentBpsPeak, netStats.BpsSentPeak())
		appStatusHandler.SetUInt64Value(core.MetricNetworkSentPercent, netStats.PercentSent())
	})
}

func registerCpuStatistics(appStatusPollingHandler *appStatusPolling.AppStatusPolling, ctx context.Context) error {
	cpuStats, err := machine.NewCpuStatistics()
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			cpuStats.ComputeStatistics()
		}
	}()

	return appStatusPollingHandler.RegisterPollingFunc(func(appStatusHandler core.AppStatusHandler) {
		appStatusHandler.SetUInt64Value(core.MetricCpuLoadPercent, cpuStats.CpuPercentUsage())
	})
}
