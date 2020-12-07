package factory

import (
	"bytes"
	"fmt"
	"os"
	"runtime/pprof"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
)

var log = logger.GetOrCreate("integrationtests")

// PrintStack -
func PrintStack() {
	buffer := new(bytes.Buffer)
	err := pprof.Lookup("goroutine").WriteTo(buffer, 2)
	if err != nil {
		log.Debug("could not dump goroutines")
	}

	log.Debug(fmt.Sprintf("\n%s", buffer.String()))
}

// CleanupWorkingDir -
func CleanupWorkingDir() {
	workingDir := WorkingDir
	if _, err := os.Stat(workingDir); !os.IsNotExist(err) {
		err = os.RemoveAll(workingDir)
		if err != nil {
			log.Debug("CleanupWorkingDir", "error", err.Error())
		}
	}
}

// CreateDefaultConfig -
func CreateDefaultConfig() *config.Configs {
	generalConfig, _ := core.LoadMainConfig(ConfigPath)
	ratingsConfig, _ := core.LoadRatingsConfig(RatingsPath)
	economicsConfig, _ := core.LoadEconomicsConfig(EconomicsPath)
	prefsConfig, _ := core.LoadPreferencesConfig(PrefsPath)
	p2pConfig, _ := core.LoadP2PConfig(P2pPath)
	externalConfig, _ := core.LoadExternalConfig(ExternalPath)
	systemSCConfig, _ := core.LoadSystemSmartContractsConfig(SystemSCConfigPath)

	p2pConfig.KadDhtPeerDiscovery.Enabled = false
	prefsConfig.Preferences.DestinationShardAsObserver = "0"

	configs := &config.Configs{}
	configs.GeneralConfig = generalConfig
	configs.RatingsConfig = ratingsConfig
	configs.EconomicsConfig = economicsConfig
	configs.SystemSCConfig = systemSCConfig
	configs.PreferencesConfig = prefsConfig
	configs.P2pConfig = p2pConfig
	configs.ExternalConfig = externalConfig
	configs.FlagsConfig = &config.ContextFlagsConfig{
		WorkingDir:                        "workingDir",
		UseLogView:                        true,
		ValidatorKeyPemFileName:           ValidatorKeyPemPath,
		GasScheduleConfigurationDirectory: GasSchedule,
		Version:                           Version,
		GenesisFileName:                   GenesisPath,
		SmartContractsFileName:            GenesisSmartContracts,
		NodesFileName:                     NodesSetupPath,
	}
	configs.ImportDbConfig = &config.ImportDbConfig{}

	configs.ConfigurationGasScheduleDirectoryName = GasSchedule
	configs.ConfigurationSystemSCFilename = SystemSCConfigPath
	configs.ConfigurationExternalFileName = ExternalPath
	configs.ConfigurationFileName = ConfigPath
	configs.ConfigurationEconomicsFileName = EconomicsPath
	configs.ConfigurationRatingsFileName = RatingsPath
	configs.ConfigurationPreferencesFileName = PrefsPath
	configs.P2pConfigurationFileName = P2pPath

	return configs
}
