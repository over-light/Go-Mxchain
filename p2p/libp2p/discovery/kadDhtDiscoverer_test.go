package discovery_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p"
	libp2p2 "github.com/ElrondNetwork/elrond-go/p2p/libp2p"
	"github.com/ElrondNetwork/elrond-go/p2p/libp2p/discovery"
	"github.com/ElrondNetwork/elrond-go/p2p/mock"
	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/assert"
)

var timeoutWaitResponses = 2 * time.Second

func createDummyHost() libp2p2.ConnectableHost {
	netw := mocknet.New(context.Background())

	h, _ := netw.GenPeer()
	return libp2p2.NewConnectableHost(h)
}

func TestNewKadDhtPeerDiscoverer_InvalidPeersRefreshIntervalShouldErr(t *testing.T) {
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: time.Second - time.Microsecond,
		RandezVous:           "randez vous",
		InitialPeersList:     []string{"peer1", "peer2"},
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}

	kdd, err := discovery.NewKadDhtPeerDiscoverer(arg)

	assert.Nil(t, kdd)
	assert.True(t, errors.Is(err, p2p.ErrInvalidValue))
}

func TestNewKadDhtPeerDiscoverer_InvalidRoutingTableRefreshIntervalShouldErr(t *testing.T) {
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: time.Second,
		RandezVous:           "randez vous",
		InitialPeersList:     []string{"peer1", "peer2"},
		BucketSize:           100,
		RoutingTableRefresh:  time.Second - time.Microsecond,
	}

	kdd, err := discovery.NewKadDhtPeerDiscoverer(arg)

	assert.Nil(t, kdd)
	assert.True(t, errors.Is(err, p2p.ErrInvalidValue))
}

func TestNewKadDhtPeerDiscoverer_ShouldSetValues(t *testing.T) {
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: 4 * time.Second,
		RandezVous:           "randez vous",
		InitialPeersList:     []string{"peer1", "peer2"},
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}

	kdd, err := discovery.NewKadDhtPeerDiscoverer(arg)

	assert.Nil(t, err)
	assert.Equal(t, arg.PeersRefreshInterval, kdd.PeersRefreshInterval())
	assert.Equal(t, arg.RandezVous, kdd.RandezVous())
	assert.Equal(t, arg.InitialPeersList, kdd.InitialPeersList())
	assert.Equal(t, arg.RoutingTableRefresh, kdd.RoutingTableRefresh())
	assert.Equal(t, arg.BucketSize, kdd.BucketSize())

	assert.False(t, kdd.IsDiscoveryPaused())
	kdd.Pause()
	assert.True(t, kdd.IsDiscoveryPaused())
	kdd.Resume()
	assert.False(t, kdd.IsDiscoveryPaused())
}

//------- Bootstrap

func TestKadDhtPeerDiscoverer_BootstrapCalledWithoutContextAppliedShouldErr(t *testing.T) {
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: time.Second,
		RandezVous:           "",
		InitialPeersList:     nil,
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}
	kdd, _ := discovery.NewKadDhtPeerDiscoverer(arg)
	err := kdd.Bootstrap()

	assert.Equal(t, p2p.ErrNilContextProvider, err)
}

func TestKadDhtPeerDiscoverer_BootstrapCalledOnceShouldWork(t *testing.T) {
	interval := time.Second

	h := createDummyHost()
	ctx, _ := libp2p.NewLibp2pContext(context.Background(), h)

	arg := discovery.ArgKadDht{
		PeersRefreshInterval: interval,
		RandezVous:           "",
		InitialPeersList:     nil,
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}

	kdd, _ := discovery.NewKadDhtPeerDiscoverer(arg)
	defer func() {
		_ = h.Close()
	}()

	_ = kdd.ApplyContext(ctx)
	err := kdd.Bootstrap()

	assert.Nil(t, err)

	if !testing.Short() {
		time.Sleep(interval * 1)
		kdd.Pause()
		time.Sleep(interval * 2)
	}
}

func TestKadDhtPeerDiscoverer_BootstrapCalledTwiceShouldErr(t *testing.T) {
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: time.Second,
		RandezVous:           "",
		InitialPeersList:     nil,
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}

	h := createDummyHost()
	ctx, _ := libp2p.NewLibp2pContext(context.Background(), h)
	kdd, _ := discovery.NewKadDhtPeerDiscoverer(arg)

	defer func() {
		_ = h.Close()
	}()

	_ = kdd.ApplyContext(ctx)
	_ = kdd.Bootstrap()
	err := kdd.Bootstrap()

	assert.Equal(t, p2p.ErrPeerDiscoveryProcessAlreadyStarted, err)
}

//------- connectToOnePeerFromInitialPeersList

func TestKadDhtPeerDiscoverer_ConnectToOnePeerFromInitialPeersListNilListShouldRetWithChanFull(t *testing.T) {
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: time.Second,
		RandezVous:           "",
		InitialPeersList:     nil,
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}

	kdd, _ := discovery.NewKadDhtPeerDiscoverer(arg)
	lctx, _ := libp2p.NewLibp2pContext(context.Background(), &mock.ConnectableHostStub{})
	_ = kdd.ApplyContext(lctx)

	chanDone := kdd.ConnectToOnePeerFromInitialPeersList(time.Second, nil)

	assert.Equal(t, 1, len(chanDone))
}

func TestKadDhtPeerDiscoverer_ConnectToOnePeerFromInitialPeersListEmptyListShouldRetWithChanFull(t *testing.T) {
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: time.Second,
		RandezVous:           "",
		InitialPeersList:     nil,
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}

	kdd, _ := discovery.NewKadDhtPeerDiscoverer(arg)
	lctx, _ := libp2p.NewLibp2pContext(context.Background(), &mock.ConnectableHostStub{})
	_ = kdd.ApplyContext(lctx)

	chanDone := kdd.ConnectToOnePeerFromInitialPeersList(time.Second, make([]string, 0))

	assert.Equal(t, 1, len(chanDone))
}

func TestKadDhtPeerDiscoverer_ConnectToOnePeerFromInitialPeersOnePeerShouldTryToConnect(t *testing.T) {
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: time.Second,
		RandezVous:           "",
		InitialPeersList:     nil,
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}
	peerID := "peer"
	wasConnectCalled := int32(0)
	uhs := &mock.ConnectableHostStub{
		ConnectToPeerCalled: func(ctx context.Context, address string) error {
			if peerID == address {
				atomic.AddInt32(&wasConnectCalled, 1)
			}

			return nil
		},
	}
	kdd, _ := discovery.NewKadDhtPeerDiscoverer(arg)
	lctx, _ := libp2p.NewLibp2pContext(context.Background(), uhs)
	_ = kdd.ApplyContext(lctx)

	chanDone := kdd.ConnectToOnePeerFromInitialPeersList(time.Second, []string{peerID})

	select {
	case <-chanDone:
		assert.Equal(t, int32(1), atomic.LoadInt32(&wasConnectCalled))
	case <-time.After(timeoutWaitResponses):
		assert.Fail(t, "timeout")
	}
}

func TestKadDhtPeerDiscoverer_ConnectToOnePeerFromInitialPeersOnePeerShouldTryToConnectContinously(t *testing.T) {
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: time.Second,
		RandezVous:           "",
		InitialPeersList:     nil,
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}
	peerID := "peer"
	wasConnectCalled := int32(0)
	errDidNotConnect := errors.New("did not connect")
	noOfTimesToRefuseConnection := 5
	uhs := &mock.ConnectableHostStub{
		ConnectToPeerCalled: func(ctx context.Context, address string) error {
			if peerID != address {
				assert.Fail(t, "should have tried to connect to the same ID")
			}

			atomic.AddInt32(&wasConnectCalled, 1)

			if atomic.LoadInt32(&wasConnectCalled) < int32(noOfTimesToRefuseConnection) {
				return errDidNotConnect
			}

			return nil
		},
	}
	kdd, _ := discovery.NewKadDhtPeerDiscoverer(arg)
	lctx, _ := libp2p.NewLibp2pContext(context.Background(), uhs)
	_ = kdd.ApplyContext(lctx)

	chanDone := kdd.ConnectToOnePeerFromInitialPeersList(time.Millisecond*10, []string{peerID})

	select {
	case <-chanDone:
		assert.Equal(t, int32(noOfTimesToRefuseConnection), atomic.LoadInt32(&wasConnectCalled))
	case <-time.After(timeoutWaitResponses):
		assert.Fail(t, "timeout")
	}
}

func TestKadDhtPeerDiscoverer_ConnectToOnePeerFromInitialPeersTwoPeersShouldAlternate(t *testing.T) {
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: time.Second,
		RandezVous:           "",
		InitialPeersList:     nil,
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}
	peerID1 := "peer1"
	peerID2 := "peer2"
	wasConnectCalled := int32(0)
	errDidNotConnect := errors.New("did not connect")
	noOfTimesToRefuseConnection := 5
	uhs := &mock.ConnectableHostStub{
		ConnectToPeerCalled: func(ctx context.Context, address string) error {
			connCalled := atomic.LoadInt32(&wasConnectCalled)

			atomic.AddInt32(&wasConnectCalled, 1)

			if connCalled >= int32(noOfTimesToRefuseConnection) {
				return nil
			}

			connCalled = connCalled % 2
			if connCalled == 0 {
				if peerID1 != address {
					assert.Fail(t, "should have tried to connect to "+peerID1)
				}
			}

			if connCalled == 1 {
				if peerID2 != address {
					assert.Fail(t, "should have tried to connect to "+peerID2)
				}
			}

			return errDidNotConnect
		},
	}

	kdd, _ := discovery.NewKadDhtPeerDiscoverer(arg)
	lctx, _ := libp2p.NewLibp2pContext(context.Background(), uhs)
	_ = kdd.ApplyContext(lctx)

	chanDone := kdd.ConnectToOnePeerFromInitialPeersList(time.Millisecond*10, []string{peerID1, peerID2})

	select {
	case <-chanDone:
	case <-time.After(timeoutWaitResponses):
		assert.Fail(t, "timeout")
	}
}

//------- ApplyContext

func TestKadDhtPeerDiscoverer_ApplyContextNilProviderShouldErr(t *testing.T) {
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: time.Second,
		RandezVous:           "",
		InitialPeersList:     nil,
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}
	kdd, _ := discovery.NewKadDhtPeerDiscoverer(arg)

	err := kdd.ApplyContext(nil)

	assert.Equal(t, p2p.ErrNilContextProvider, err)
}

func TestKadDhtPeerDiscoverer_ApplyContextWrongProviderShouldErr(t *testing.T) {
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: time.Second,
		RandezVous:           "",
		InitialPeersList:     nil,
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}
	kdd, _ := discovery.NewKadDhtPeerDiscoverer(arg)

	err := kdd.ApplyContext(&mock.ContextProviderMock{})

	assert.Equal(t, p2p.ErrWrongContextProvider, err)
}

func TestKadDhtPeerDiscoverer_ApplyContextShouldWork(t *testing.T) {
	ctx, _ := libp2p.NewLibp2pContext(context.Background(), &mock.ConnectableHostStub{})
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: time.Second,
		RandezVous:           "",
		InitialPeersList:     nil,
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}
	kdd, _ := discovery.NewKadDhtPeerDiscoverer(arg)

	err := kdd.ApplyContext(ctx)

	assert.Nil(t, err)
	assert.True(t, ctx == kdd.ContextProvider())
}

//----------- Watchdog

func TestKadDhtPeerDiscoverer_Watchdog(t *testing.T) {
	ctx, _ := libp2p.NewLibp2pContext(context.Background(), &mock.ConnectableHostStub{})
	interval := time.Second
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: interval,
		RandezVous:           "",
		InitialPeersList:     nil,
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}
	kdd, _ := discovery.NewKadDhtPeerDiscoverer(arg)

	// starting with no context fails (no panic)
	err := kdd.StartWatchdog(interval)
	assert.NotNil(t, err)

	err = kdd.StopWatchdog()
	assert.NotNil(t, err)

	err = kdd.KickWatchdog()
	assert.NotNil(t, err)

	err = kdd.ApplyContext(ctx)
	assert.Nil(t, err)

	err = kdd.StartWatchdog(interval / 2)
	assert.NotNil(t, err)

	s1 := kdd.StartWatchdog(interval)
	s2 := kdd.StartWatchdog(interval)

	assert.Nil(t, s1)
	assert.Equal(t, s2, p2p.ErrWatchdogAlreadyStarted)

	if !testing.Short() {
		//kick
		_ = kdd.KickWatchdog()
		kdd.Pause()
		assert.True(t, kdd.IsDiscoveryPaused())
		time.Sleep(interval / 2)
		assert.True(t, kdd.IsDiscoveryPaused())
		time.Sleep(interval)
		assert.False(t, kdd.IsDiscoveryPaused())
	}

	_ = kdd.StopWatchdog()
}

//---------- Protocols
func TestKadDhtPeerDiscoverer_Protocols(t *testing.T) {
	streams := make(map[protocol.ID]network.StreamHandler)
	notifeesCnt := 0
	net := &mock.NetworkStub{
		ConnectednessCalled: func(p peer.ID) network.Connectedness {
			t.Logf("Conn to %s", p.Pretty())
			return network.CannotConnect
		},

		NotifyCalled:     func(nn network.Notifiee) { notifeesCnt++ },
		StopNotifyCalled: func(nn network.Notifiee) { notifeesCnt-- },
	}

	host := &mock.ConnectableHostStub{
		IDCalled: func() peer.ID {
			return "local peer"
		},
		PeerstoreCalled: func() peerstore.Peerstore {
			return nil
		},
		NetworkCalled: func() network.Network {
			return net
		},
		SetStreamHandlerCalled: func(proto protocol.ID, sh network.StreamHandler) {
			t.Logf("Set strem hndl %v", proto)
			streams[proto] = sh
		},
		RemoveStreamHandlerCalled: func(proto protocol.ID) {
			t.Logf("Remove stream %v", proto)
			streams[proto] = nil
		},
		ConnManagerCalled: func() connmgr.ConnManager {
			return nil
		},
	}

	ctx, _ := libp2p.NewLibp2pContext(context.Background(), host)
	interval := time.Second
	arg := discovery.ArgKadDht{
		PeersRefreshInterval: interval,
		RandezVous:           "r1",
		InitialPeersList:     nil,
		BucketSize:           100,
		RoutingTableRefresh:  5 * time.Second,
	}
	kdd, _ := discovery.NewKadDhtPeerDiscoverer(arg)

	err := kdd.ApplyContext(ctx)
	assert.Nil(t, err)

	err = kdd.Bootstrap()
	assert.Nil(t, err)

	assert.Equal(t, notifeesCnt, 1)
	err = kdd.UpdateRandezVous("r2")
	assert.Nil(t, err)
	assert.Equal(t, notifeesCnt, 1)
	err = kdd.StopDHT()
	assert.Nil(t, err)

	assert.Equal(t, notifeesCnt, 0)
	for p, cb := range streams {
		assert.Nil(t, cb, p, "should have no callback")
	}
}
