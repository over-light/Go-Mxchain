package factory

import "github.com/libp2p/go-libp2p-core/network"

// ConnectionMonitor defines the behavior of a connection monitor
type ConnectionMonitor interface {
	network.Notifiee
	SetSharder(sharder interface{}) error
	IsConnectedToTheNetwork(netw network.Network) bool
	SetThresholdMinConnectedPeers(thresholdMinConnectedPeers int, netw network.Network)
	ThresholdMinConnectedPeers() int
}
