package libp2p

import (
	"context"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/throttler"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/p2p/loadBalancer"
	"github.com/libp2p/go-libp2p-core/connmgr"
	libp2pCrypto "github.com/libp2p/go-libp2p-core/crypto"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
)

const targetPeerCount = 100

// NewMemoryMessenger creates a new sandbox testable instance of libP2P messenger
// It should not open ports on current machine
// Should be used only in testing!
func NewMemoryMessenger(
	ctx context.Context,
	mockNet mocknet.Mocknet,
	peerDiscoverer p2p.PeerDiscoverer,
) (*networkMessenger, error) {

	if ctx == nil {
		return nil, p2p.ErrNilContext
	}
	if mockNet == nil {
		return nil, p2p.ErrNilMockNet
	}
	if check.IfNil(peerDiscoverer) {
		return nil, p2p.ErrNilPeerDiscoverer
	}

	h, err := mockNet.GenPeer()
	if err != nil {
		return nil, err
	}

	lctx, err := NewLibp2pContext(ctx, NewConnectableHost(h))
	if err != nil {
		log.LogIfError(h.Close())
		return nil, err
	}

	mes, err := createMessenger(
		lctx,
		false,
		loadBalancer.NewOutgoingChannelLoadBalancer(),
		peerDiscoverer,
		targetPeerCount,
	)
	if err != nil {
		return nil, err
	}

	goRoutinesThrottler, err := throttler.NewNumGoRoutineThrottler(broadcastGoRoutines)
	if err != nil {
		log.LogIfError(h.Close())
		return nil, err
	}

	mes.goRoutinesThrottler = goRoutinesThrottler

	return mes, err
}

// NewNetworkMessengerOnFreePort tries to create a new NetworkMessenger on a free port found in the system
// Should be used only in testing!
func NewNetworkMessengerOnFreePort(
	ctx context.Context,
	p2pPrivKey libp2pCrypto.PrivKey,
	conMgr connmgr.ConnManager,
	outgoingPLB p2p.ChannelLoadBalancer,
	peerDiscoverer p2p.PeerDiscoverer,
) (*networkMessenger, error) {
	return NewNetworkMessenger(
		ctx,
		0,
		p2pPrivKey,
		conMgr,
		outgoingPLB,
		peerDiscoverer,
		ListenLocalhostAddrWithIp4AndTcp,
		targetPeerCount,
	)
}
