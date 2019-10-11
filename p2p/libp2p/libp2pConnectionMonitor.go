package libp2p

import (
	"math"
	"time"

	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-kbucket"
	"github.com/multiformats/go-multiaddr"
)

// DurationBetweenReconnectAttempts is used as to not call reconnecter.ReconnectToNetwork() to often
// when there are a lot of peers disconnecting and reconnection to initial nodes succeed
var DurationBetweenReconnectAttempts = time.Second * 5

type libp2pConnectionMonitor struct {
	chDoReconnect              chan struct{}
	reconnecter                p2p.Reconnecter
	thresholdMinConnectedPeers int
	targetConnCount            int
}

func newLibp2pConnectionMonitor(reconnecter p2p.Reconnecter, thresholdMinConnectedPeers int, targetConnCount int) (*libp2pConnectionMonitor, error) {
	if thresholdMinConnectedPeers < 0 {
		return nil, p2p.ErrInvalidValue
	}

	cm := &libp2pConnectionMonitor{
		reconnecter:                reconnecter,
		chDoReconnect:              make(chan struct{}, 0),
		thresholdMinConnectedPeers: thresholdMinConnectedPeers,
		targetConnCount:            targetConnCount,
	}

	if reconnecter != nil {
		go cm.doReconnection()
	}

	return cm, nil
}

// Listen is called when network starts listening on an addr
func (lcm *libp2pConnectionMonitor) Listen(network.Network, multiaddr.Multiaddr) {}

// ListenClose is called when network stops listening on an addr
func (lcm *libp2pConnectionMonitor) ListenClose(network.Network, multiaddr.Multiaddr) {}

// ThresholdDiscoveryPause if the number of connected peers is over this value, the dht discovery is stopped
func (lcm *libp2pConnectionMonitor) ThresholdDiscoveryPause() int {
	if lcm.targetConnCount > 0 {
		return lcm.targetConnCount
	}
	return math.MaxInt32
}

// ThresholdDiscoveryResume if the number of connected peers drop under this value, the dht discovery is restarted
func (lcm *libp2pConnectionMonitor) ThresholdDiscoveryResume() int {
	if lcm.targetConnCount > 0 {
		return lcm.targetConnCount * 4 / 5
	}
	return 0
}

// ThresholdRandomTrim if the number of connected peers is over this value, we start cutting of connections at random
func (lcm *libp2pConnectionMonitor) ThresholdRandomTrim() int {
	if lcm.targetConnCount > 0 {
		return lcm.targetConnCount * 6 / 5
	}
	return math.MaxInt32
}

// Request a reconnet to initial list
func (lcm *libp2pConnectionMonitor) doReconn() {
	select {
	case lcm.chDoReconnect <- struct{}{}:
	default:
	}
}

// Connected is called when a connection opened
func (lcm *libp2pConnectionMonitor) Connected(netw network.Network, conn network.Conn) {
	if len(netw.Conns()) > lcm.ThresholdDiscoveryPause() {
		lcm.reconnecter.Pause()
	}
	if len(netw.Conns()) > lcm.ThresholdRandomTrim() {
		sorted := kbucket.SortClosestPeers(netw.Peers(), kbucket.ConvertPeerID(netw.LocalPeer()))
		for i := lcm.ThresholdDiscoveryPause(); i < len(sorted); i++ {
			netw.ClosePeer(sorted[i])
		}
		lcm.doReconn()
	}
}

// Disconnected is called when a connection closed
func (lcm *libp2pConnectionMonitor) Disconnected(netw network.Network, conn network.Conn) {
	lcm.doReconnectionIfNeeded(netw)

	if len(netw.Conns()) < lcm.ThresholdDiscoveryResume() && lcm.reconnecter != nil  {
		lcm.reconnecter.Resume()
		lcm.doReconn()
	}
}

func (lcm *libp2pConnectionMonitor) doReconnectionIfNeeded(netw network.Network) {
	if !lcm.isConnectedToTheNetwork(netw) {
		lcm.doReconn()
	}
}

// OpenedStream is called when a stream opened
func (lcm *libp2pConnectionMonitor) OpenedStream(network.Network, network.Stream) {}

// ClosedStream is called when a stream closed
func (lcm *libp2pConnectionMonitor) ClosedStream(network.Network, network.Stream) {}

func (lcm *libp2pConnectionMonitor) doReconnection() {
	for {
		select {
		case <-lcm.chDoReconnect:
			<-lcm.reconnecter.ReconnectToNetwork()
		}

		time.Sleep(DurationBetweenReconnectAttempts)
	}
}

func (lcm *libp2pConnectionMonitor) isConnectedToTheNetwork(netw network.Network) bool {
	return len(netw.Conns()) >= lcm.thresholdMinConnectedPeers
}
