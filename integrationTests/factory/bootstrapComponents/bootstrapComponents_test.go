package bootstrapComponents

import (
	"runtime"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/integrationTests/factory"
	"github.com/stretchr/testify/require"
)

// ------------ Test CryptoComponents --------------------
func TestBootstrapComponents_Create_Close_ShouldWork(t *testing.T) {
	nrBefore := runtime.NumGoroutine()

	generalConfig, _ := core.LoadMainConfig(factory.ConfigPath)
	ratingsConfig, _ := core.LoadRatingsConfig(factory.RatingsPath)
	economicsConfig, _ := core.LoadEconomicsConfig(factory.EconomicsPath)
	prefsConfig, _ := core.LoadPreferencesConfig(factory.PrefsPath)
	p2pConfig, _ := core.LoadP2PConfig(factory.P2pPath)
	systemSCConfig, _ := core.LoadSystemSmartContractsConfig(factory.SystemSCConfigPath)

	coreComponents, err := factory.CreateCoreComponents(*generalConfig, *ratingsConfig, *economicsConfig)
	require.Nil(t, err)
	require.NotNil(t, coreComponents)

	cryptoComponents, err := factory.CreateCryptoComponents(*generalConfig, *systemSCConfig, coreComponents)
	require.Nil(t, err)
	require.NotNil(t, cryptoComponents)

	networkComponents, err := factory.CreateNetworkComponents(*generalConfig, *p2pConfig, *ratingsConfig, coreComponents)
	require.Nil(t, err)
	require.NotNil(t, networkComponents)

	time.Sleep(2 * time.Second)

	bootstrapComponents, err := factory.CreateBootstrapComponents(
		*generalConfig,
		prefsConfig.Preferences,
		coreComponents,
		cryptoComponents,
		networkComponents)
	require.Nil(t, err)
	require.NotNil(t, bootstrapComponents)

	time.Sleep(2 * time.Second)
	err = bootstrapComponents.Close()
	require.Nil(t, err)

	err = networkComponents.Close()
	require.Nil(t, err)

	err = cryptoComponents.Close()
	require.Nil(t, err)

	err = coreComponents.Close()
	require.Nil(t, err)

	time.Sleep(5 * time.Second)

	nrAfter := runtime.NumGoroutine()

	if nrBefore != nrAfter {
		factory.PrintStack()
	}

	require.Equal(t, nrBefore, nrAfter)

	factory.CleanupWorkingDir()
}
