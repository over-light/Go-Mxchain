package factory_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/factory"
	"github.com/stretchr/testify/require"
)

// ------------ Test ManagedStateComponents --------------------
func TestManagedStateComponents_CreateWithInvalidArgs_ShouldErr(t *testing.T) {
	coreComponents := getCoreComponents()
	args := getStateArgs(coreComponents)
	stateComponentsFactory, _ := factory.NewStateComponentsFactory(args)
	managedStateComponents, err := factory.NewManagedStateComponents(stateComponentsFactory)
	require.NoError(t, err)
	_ = args.Core.SetInternalMarshalizer(nil)
	err = managedStateComponents.Create()
	require.Error(t, err)
	require.Nil(t, managedStateComponents.AccountsAdapter())
}

func TestManagedStateComponents_Create_ShouldWork(t *testing.T) {
	coreComponents := getCoreComponents()
	args := getStateArgs(coreComponents)
	stateComponentsFactory, _ := factory.NewStateComponentsFactory(args)
	managedStateComponents, err := factory.NewManagedStateComponents(stateComponentsFactory)
	require.NoError(t, err)
	require.Nil(t, managedStateComponents.AccountsAdapter())
	require.Nil(t, managedStateComponents.PeerAccounts())
	require.Nil(t, managedStateComponents.TriesContainer())
	require.Nil(t, managedStateComponents.TrieStorageManagers())

	err = managedStateComponents.Create()
	require.NoError(t, err)
	require.NotNil(t, managedStateComponents.AccountsAdapter())
	require.NotNil(t, managedStateComponents.PeerAccounts())
	require.NotNil(t, managedStateComponents.TriesContainer())
	require.NotNil(t, managedStateComponents.TrieStorageManagers())
}

func TestManagedStateComponents_Close(t *testing.T) {
	coreComponents := getCoreComponents()
	args := getStateArgs(coreComponents)
	stateComponentsFactory, _ := factory.NewStateComponentsFactory(args)
	managedStateComponents, _ := factory.NewManagedStateComponents(stateComponentsFactory)
	err := managedStateComponents.Create()
	require.NoError(t, err)

	err = managedStateComponents.Close()
	require.NoError(t, err)
	require.Nil(t, managedStateComponents.AccountsAdapter())
}
