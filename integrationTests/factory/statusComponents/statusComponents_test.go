package statusComponents

import (
	"runtime"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/endProcess"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	mainFactory "github.com/ElrondNetwork/elrond-go/factory"
	"github.com/ElrondNetwork/elrond-go/integrationTests/factory"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/stretchr/testify/require"
)

// ------------ Test StatusComponents --------------------
func TestStatusComponents_Create_Close_ShouldWork(t *testing.T) {
	defer factory.CleanupWorkingDir()
	time.Sleep(time.Second)

	nrBefore := runtime.NumGoroutine()
	factory.PrintStack()

	configs := factory.CreateDefaultConfig()
	chanStopNodeProcess := make(chan endProcess.ArgEndProcess)
	managedCoreComponents, err := node.CreateManagedCoreComponents(configs, chanStopNodeProcess)
	require.Nil(t, err)
	managedCryptoComponents, err := node.CreateManagedCryptoComponents(configs, managedCoreComponents)
	require.Nil(t, err)
	managedNetworkComponents, err := node.CreateManagedNetworkComponents(configs, managedCoreComponents)
	require.Nil(t, err)
	managedBootstrapComponents, err := node.CreateManagedBootstrapComponents(configs, managedCoreComponents, managedCryptoComponents, managedNetworkComponents)
	require.Nil(t, err)
	managedDataComponents, err := node.CreateManagedDataComponents(configs, managedCoreComponents, managedBootstrapComponents)
	require.Nil(t, err)
	managedStateComponents, err := node.CreateManagedStateComponents(configs, managedCoreComponents, managedBootstrapComponents)
	require.Nil(t, err)
	nodesShufflerOut, err := mainFactory.CreateNodesShuffleOut(managedCoreComponents.GenesisNodesSetup(), configs.GeneralConfig.EpochStartConfig, managedCoreComponents.ChanStopNodeProcess())
	require.Nil(t, err)
	nodesCoordinator, err := mainFactory.CreateNodesCoordinator(
		nodesShufflerOut,
		managedCoreComponents.GenesisNodesSetup(),
		configs.PreferencesConfig.Preferences,
		managedCoreComponents.EpochStartNotifierWithConfirm(),
		managedCryptoComponents.PublicKey(),
		managedCoreComponents.InternalMarshalizer(),
		managedCoreComponents.Hasher(),
		managedCoreComponents.Rater(),
		managedDataComponents.StorageService().GetStorer(dataRetriever.BootstrapUnit),
		managedCoreComponents.NodesShuffler(),
		managedBootstrapComponents.ShardCoordinator().SelfId(),
		managedBootstrapComponents.EpochBootstrapParams(),
		managedBootstrapComponents.EpochBootstrapParams().Epoch(),
	)
	require.Nil(t, err)
	managedStatusComponents, err := node.CreateManagedStatusComponents(
		configs,
		managedCoreComponents,
		managedNetworkComponents,
		managedBootstrapComponents,
		managedDataComponents,
		managedStateComponents,
		nodesCoordinator,
		"",
		false,
	)
	require.Nil(t, err)
	require.NotNil(t, managedStatusComponents)

	gasSchedule, err := core.LoadGasScheduleConfig(configs.FlagsConfig.GasScheduleConfigurationFileName)
	require.Nil(t, err)
	managedProcessComponents, err := node.CreateManagedProcessComponents(configs, managedCoreComponents, managedCryptoComponents, managedNetworkComponents, managedBootstrapComponents, managedStateComponents, managedDataComponents, managedStatusComponents, gasSchedule, nodesCoordinator)
	require.Nil(t, err)
	time.Sleep(2 * time.Second)

	managedStatusComponents.SetForkDetector(managedProcessComponents.ForkDetector())
	err = managedStatusComponents.StartPolling()
	require.Nil(t, err)

	time.Sleep(5 * time.Second)

	err = managedStatusComponents.Close()
	require.Nil(t, err)
	err = managedProcessComponents.Close()
	require.Nil(t, err)
	err = managedStateComponents.Close()
	require.Nil(t, err)
	err = managedDataComponents.Close()
	require.Nil(t, err)
	err = managedBootstrapComponents.Close()
	require.Nil(t, err)
	err = managedNetworkComponents.Close()
	require.Nil(t, err)
	err = managedCryptoComponents.Close()
	require.Nil(t, err)
	err = managedCoreComponents.Close()
	require.Nil(t, err)

	time.Sleep(5 * time.Second)

	nrAfter := runtime.NumGoroutine()
	if nrBefore != nrAfter {
		factory.PrintStack()
	}

	require.Equal(t, nrBefore, nrAfter)
}
