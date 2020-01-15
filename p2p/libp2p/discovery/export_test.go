package discovery

import (
	"time"

	"github.com/ElrondNetwork/elrond-go/p2p/libp2p"
)

func (kdd *KadDhtDiscoverer) PeersRefreshInterval() time.Duration {
	return kdd.peersRefreshInterval
}

func (kdd *KadDhtDiscoverer) InitialPeersList() []string {
	return kdd.initialPeersList
}

func (kdd *KadDhtDiscoverer) RandezVous() string {
	return kdd.randezVous
}

func (kdd *KadDhtDiscoverer) RoutingTableRefresh() time.Duration {
	return kdd.routingTableRefresh
}

func (kdd *KadDhtDiscoverer) BucketSize() int {
	return kdd.bucketSize
}

func (kdd *KadDhtDiscoverer) ContextProvider() *libp2p.Libp2pContext {
	return kdd.contextProvider
}

func (kdd *KadDhtDiscoverer) ConnectToOnePeerFromInitialPeersList(
	durationBetweenAttempts time.Duration,
	initialPeersList []string) <-chan struct{} {

	return kdd.connectToOnePeerFromInitialPeersList(durationBetweenAttempts, initialPeersList)
}
