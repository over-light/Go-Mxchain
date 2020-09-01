package factory_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/factory"
	"github.com/stretchr/testify/require"
)

// ------------ Test ManagedDataComponents --------------------
func TestManagedDataComponents_CreateWithInvalidArgs_ShouldErr(t *testing.T) {
	coreComponents := getCoreComponents()
	args := getDataArgs(coreComponents)
	args.Config.ShardHdrNonceHashStorage = config.StorageConfig{}
	dataComponentsFactory, _ := factory.NewDataComponentsFactory(args)
	managedDataComponents, err := factory.NewManagedDataComponents(dataComponentsFactory)
	require.NoError(t, err)
	err = managedDataComponents.Create()
	require.Error(t, err)
	require.Nil(t, managedDataComponents.Blockchain())
}

func TestManagedDataComponents_Create_ShouldWork(t *testing.T) {
	coreComponents := getCoreComponents()
	args := getDataArgs(coreComponents)
	dataComponentsFactory, _ := factory.NewDataComponentsFactory(args)
	managedDataComponents, err := factory.NewManagedDataComponents(dataComponentsFactory)
	require.NoError(t, err)
	require.Nil(t, managedDataComponents.Blockchain())
	require.Nil(t, managedDataComponents.StorageService())
	require.Nil(t, managedDataComponents.Datapool())

	err = managedDataComponents.Create()
	require.NoError(t, err)
	require.NotNil(t, managedDataComponents.Blockchain())
	require.NotNil(t, managedDataComponents.StorageService())
	require.NotNil(t, managedDataComponents.Datapool())
}

func TestManagedDataComponents_Close(t *testing.T) {
	coreComponents := getCoreComponents()
	args := getDataArgs(coreComponents)
	dataComponentsFactory, _ := factory.NewDataComponentsFactory(args)
	managedDataComponents, _ := factory.NewManagedDataComponents(dataComponentsFactory)
	err := managedDataComponents.Create()
	require.NoError(t, err)

	err = managedDataComponents.Close()
	require.NoError(t, err)
	require.Nil(t, managedDataComponents.Blockchain())
}
