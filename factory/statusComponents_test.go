package factory_test

import (
	"fmt"
	"testing"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/indexer"
	"github.com/ElrondNetwork/elrond-go/factory"
	"github.com/ElrondNetwork/elrond-go/factory/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatusComponentsFactory_NilCoreComponentsShouldErr(t *testing.T) {
	t.Parallel()

	args, _ := getStatusComponentsFactoryArgsAndProcessComponents()
	args.CoreComponents = nil
	scf, err := factory.NewStatusComponentsFactory(args)
	assert.True(t, check.IfNil(scf))
	assert.Equal(t, factory.ErrNilCoreComponentsHolder, err)
}

func TestNewStatusComponentsFactory_NilNodesCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	args, _ := getStatusComponentsFactoryArgsAndProcessComponents()
	args.NodesCoordinator = nil
	scf, err := factory.NewStatusComponentsFactory(args)
	assert.True(t, check.IfNil(scf))
	assert.Equal(t, factory.ErrNilNodesCoordinator, err)
}

func TestNewStatusComponentsFactory_NilEpochStartNotifierShouldErr(t *testing.T) {
	t.Parallel()

	args, _ := getStatusComponentsFactoryArgsAndProcessComponents()
	args.EpochStartNotifier = nil
	scf, err := factory.NewStatusComponentsFactory(args)
	assert.True(t, check.IfNil(scf))
	assert.Equal(t, factory.ErrNilEpochStartNotifier, err)
}

func TestNewStatusComponentsFactory_NilStatusHandlerErr(t *testing.T) {
	t.Parallel()

	args, _ := getStatusComponentsFactoryArgsAndProcessComponents()
	args.StatusUtils = nil
	scf, err := factory.NewStatusComponentsFactory(args)
	assert.True(t, check.IfNil(scf))
	assert.Equal(t, factory.ErrNilStatusHandlersUtils, err)
}

func TestNewStatusComponentsFactory_NilNetworkComponentsShouldErr(t *testing.T) {
	t.Parallel()

	args, _ := getStatusComponentsFactoryArgsAndProcessComponents()
	args.NetworkComponents = nil
	scf, err := factory.NewStatusComponentsFactory(args)
	assert.True(t, check.IfNil(scf))
	assert.Equal(t, factory.ErrNilNetworkComponentsHolder, err)
}

func TestNewStatusComponentsFactory_NilShardCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	args, _ := getStatusComponentsFactoryArgsAndProcessComponents()
	args.ShardCoordinator = nil
	scf, err := factory.NewStatusComponentsFactory(args)
	assert.True(t, check.IfNil(scf))
	assert.Equal(t, factory.ErrNilShardCoordinator, err)
}

func TestNewStatusComponentsFactory_InvalidRoundDurationShouldErr(t *testing.T) {
	t.Parallel()

	args, _ := getStatusComponentsFactoryArgsAndProcessComponents()
	args.RoundDurationSec = 0
	scf, err := factory.NewStatusComponentsFactory(args)
	assert.True(t, check.IfNil(scf))
	assert.Equal(t, factory.ErrInvalidRoundDuration, err)
}

func TestNewStatusComponentsFactory_ShouldWork(t *testing.T) {
	t.Parallel()

	args, _ := getStatusComponentsFactoryArgsAndProcessComponents()
	scf, err := factory.NewStatusComponentsFactory(args)
	require.NoError(t, err)
	require.False(t, check.IfNil(scf))
}

func TestStatusComponentsFactory_Create(t *testing.T) {
	t.Parallel()

	args, _ := getStatusComponentsFactoryArgsAndProcessComponents()
	scf, err := factory.NewStatusComponentsFactory(args)
	require.Nil(t, err)

	res, err := scf.Create()
	require.NoError(t, err)
	require.NotNil(t, res)
}

func getStatusComponentsFactoryArgsAndProcessComponents() (factory.StatusComponentsFactoryArgs, factory.ProcessComponentsHolder) {
	coreArgs := getCoreArgs()
	coreComponents := getCoreComponents()
	networkComponents := getNetworkComponents()
	dataComponents := getDataComponents(coreComponents)
	cryptoComponents := getCryptoComponents(coreComponents)
	stateComponents := getStateComponents(coreComponents)
	processComponents := getProcessComponents(
		coreComponents,
		networkComponents,
		dataComponents,
		cryptoComponents,
		stateComponents,
	)

	return factory.StatusComponentsFactoryArgs{
		Config:             coreArgs.Config,
		ExternalConfig:     config.ExternalConfig{},
		RoundDurationSec:   4,
		ElasticOptions:     &indexer.Options{},
		ShardCoordinator:   mock.NewMultiShardsCoordinatorMock(2),
		NodesCoordinator:   processComponents.NodesCoordinator(),
		EpochStartNotifier: processComponents.EpochStartNotifier(),
		CoreComponents:     coreComponents,
		DataComponents:     dataComponents,
		NetworkComponents:  networkComponents,
		StatusUtils:        &mock.StatusHandlersUtilsMock{},
	}, processComponents
}

func getNetworkComponents() factory.NetworkComponentsHolder {
	networkArgs := getNetworkArgs()
	networkComponents, _ := factory.NewManagedNetworkComponents(networkArgs)
	_ = networkComponents.Create()
	return networkComponents
}

func getDataComponents(coreComponents factory.CoreComponentsHolder) factory.DataComponentsHolder {
	dataArgs := getDataArgs(coreComponents)
	dataComponents, _ := factory.NewManagedDataComponents(factory.DataComponentsHandlerArgs(dataArgs))
	_ = dataComponents.Create()
	return dataComponents
}

func getCryptoComponents(coreComponents factory.CoreComponentsHolder) factory.CryptoComponentsHolder {
	cryptoArgs := getCryptoArgs(coreComponents)
	cryptoComponents, err := factory.NewManagedCryptoComponents(factory.CryptoComponentsHandlerArgs(cryptoArgs))
	if err != nil {
		fmt.Println("getCryptoComponents NewManagedCryptoComponents", "error", err.Error())
		return nil
	}

	ccf := cryptoComponents.GetFactory()
	keyLoader := dummyLoadSkPkFromPemFile([]byte(dummySk), dummyPk, nil)
	ccf.SetKeyLoader(keyLoader)

	err = cryptoComponents.Create()
	if err != nil {
		fmt.Println("getCryptoComponents Create", "error", err.Error())
		return nil
	}
	return cryptoComponents
}

func getStateComponents(coreComponents factory.CoreComponentsHolder) factory.StateComponentsHolder {
	stateArgs := getStateArgs(coreComponents)
	stateComponents, err := factory.NewManagedStateComponents(stateArgs)
	if err != nil {
		fmt.Println("getStateComponents NewManagedStateComponents", "error", err.Error())
		return nil
	}
	err = stateComponents.Create()
	if err != nil {
		fmt.Println("getStateComponents Create", "error", err.Error())
	}
	return stateComponents
}

func getProcessComponents(
	coreComponents factory.CoreComponentsHolder,
	networkComponents factory.NetworkComponentsHolder,
	dataComponents factory.DataComponentsHolder,
	cryptoComponents factory.CryptoComponentsHolder,
	stateComponents factory.StateComponentsHolder,
) factory.ProcessComponentsHolder {
	processArgs := getProcessArgs(
		getCoreArgs(),
		coreComponents,
		dataComponents,
		cryptoComponents,
		stateComponents,
		networkComponents,
	)
	processComponents, err := factory.NewManagedProcessComponents(processArgs)
	if err != nil {
		fmt.Println("getProcessComponents NewManagedProcessComponents", "error", err.Error())
		return nil
	}
	err = processComponents.Create()
	if err != nil {
		fmt.Println("getProcessComponents Create", "error", err.Error())
	}
	return processComponents
}
