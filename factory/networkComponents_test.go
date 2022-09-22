package factory_test

import (
	"errors"
	"testing"

	"github.com/ElrondNetwork/elrond-go/config"
	errErd "github.com/ElrondNetwork/elrond-go/errors"
	"github.com/ElrondNetwork/elrond-go/factory"
	"github.com/ElrondNetwork/elrond-go/factory/mock"
	"github.com/ElrondNetwork/elrond-go/p2p"
	p2pConfig "github.com/ElrondNetwork/elrond-go/p2p/config"
	statusHandlerMock "github.com/ElrondNetwork/elrond-go/testscommon/statusHandler"
	"github.com/stretchr/testify/require"
)

func TestNewNetworkComponentsFactory_NilStatusHandlerShouldErr(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	args := getNetworkArgs()
	args.StatusHandler = nil
	ncf, err := factory.NewNetworkComponentsFactory(args)
	require.Nil(t, ncf)
	require.Equal(t, errErd.ErrNilStatusHandler, err)
}

func TestNewNetworkComponentsFactory_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	args := getNetworkArgs()
	args.Marshalizer = nil
	ncf, err := factory.NewNetworkComponentsFactory(args)
	require.Nil(t, ncf)
	require.True(t, errors.Is(err, errErd.ErrNilMarshalizer))
}

func TestNewNetworkComponentsFactory_OkValsShouldWork(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	args := getNetworkArgs()
	ncf, err := factory.NewNetworkComponentsFactory(args)
	require.NoError(t, err)
	require.NotNil(t, ncf)
}

func TestNetworkComponentsFactory_CreateShouldErrDueToBadConfig(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	args := getNetworkArgs()
	args.MainConfig = config.Config{}
	args.P2pConfig = p2pConfig.P2PConfig{}

	ncf, _ := factory.NewNetworkComponentsFactory(args)

	nc, err := ncf.Create()
	require.Error(t, err)
	require.Nil(t, nc)
}

func TestNetworkComponentsFactory_CreateShouldWork(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	args := getNetworkArgs()
	ncf, _ := factory.NewNetworkComponentsFactory(args)
	ncf.SetListenAddress(p2p.ListenLocalhostAddrWithIp4AndTcp)

	nc, err := ncf.Create()
	require.NoError(t, err)
	require.NotNil(t, nc)
}

// ------------ Test NetworkComponents --------------------
func TestNetworkComponents_CloseShouldWork(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	args := getNetworkArgs()
	ncf, _ := factory.NewNetworkComponentsFactory(args)

	nc, _ := ncf.Create()

	err := nc.Close()
	require.NoError(t, err)
}

func getNetworkArgs() factory.NetworkComponentsFactoryArgs {
	p2pCfg := p2pConfig.P2PConfig{
		Node: p2pConfig.NodeConfig{
			Port: "0",
			Seed: "seed",
		},
		KadDhtPeerDiscovery: p2pConfig.KadDhtPeerDiscoveryConfig{
			Enabled:                          false,
			Type:                             "optimized",
			RefreshIntervalInSec:             10,
			ProtocolID:                       "erd/kad/1.0.0",
			InitialPeerList:                  []string{"peer0", "peer1"},
			BucketSize:                       10,
			RoutingTableRefreshIntervalInSec: 5,
		},
		Sharding: p2pConfig.ShardingConfig{
			TargetPeerCount:         10,
			MaxIntraShardValidators: 10,
			MaxCrossShardValidators: 10,
			MaxIntraShardObservers:  10,
			MaxCrossShardObservers:  10,
			MaxSeeders:              2,
			Type:                    "NilListSharder",
			AdditionalConnections: p2pConfig.AdditionalConnectionsConfig{
				MaxFullHistoryObservers: 10,
			},
		},
	}

	mainConfig := config.Config{
		PeerHonesty: config.CacheConfig{
			Type:     "LRU",
			Capacity: 5000,
			Shards:   16,
		},
		Debug: config.DebugConfig{
			Antiflood: config.AntifloodDebugConfig{
				Enabled:                    true,
				CacheSize:                  100,
				IntervalAutoPrintInSeconds: 1,
			},
		},
		PeersRatingConfig: config.PeersRatingConfig{
			TopRatedCacheCapacity: 1000,
			BadRatedCacheCapacity: 1000,
		},
		PoolsCleanersConfig: config.PoolsCleanersConfig{
			MaxRoundsToKeepUnprocessedMiniBlocks:   50,
			MaxRoundsToKeepUnprocessedTransactions: 50,
		},
	}

	appStatusHandler := statusHandlerMock.NewAppStatusHandlerMock()

	return factory.NetworkComponentsFactoryArgs{
		P2pConfig:     p2pCfg,
		MainConfig:    mainConfig,
		StatusHandler: appStatusHandler,
		Marshalizer:   &mock.MarshalizerMock{},
		RatingsConfig: config.RatingsConfig{
			General:    config.General{},
			ShardChain: config.ShardChain{},
			MetaChain:  config.MetaChain{},
			PeerHonesty: config.PeerHonestyConfig{
				DecayCoefficient:             0.9779,
				DecayUpdateIntervalInSeconds: 10,
				MaxScore:                     100,
				MinScore:                     -100,
				BadPeerThreshold:             -80,
				UnitValue:                    1.0,
			},
		},
		Syncer:                &p2p.LocalSyncTimer{},
		NodeOperationMode:     p2p.NormalOperation,
		ConnectionWatcherType: p2p.ConnectionWatcherTypePrint,
	}
}
