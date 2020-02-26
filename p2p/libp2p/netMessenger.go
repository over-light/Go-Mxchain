package libp2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/throttler"
	"github.com/ElrondNetwork/elrond-go/logger"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/connmgr"
	libp2pCrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-pubsub"
)

const durationBetweenSends = time.Microsecond * 10

// ListenAddrWithIp4AndTcp defines the listening address with ip v.4 and TCP
const ListenAddrWithIp4AndTcp = "/ip4/0.0.0.0/tcp/"

// ListenLocalhostAddrWithIp4AndTcp defines the local host listening ip v.4 address and TCP
const ListenLocalhostAddrWithIp4AndTcp = "/ip4/127.0.0.1/tcp/"

// DirectSendID represents the protocol ID for sending and receiving direct P2P messages
const DirectSendID = protocol.ID("/directsend/1.0.0")

const refreshPeersOnTopic = time.Second * 5
const ttlPeersOnTopic = time.Second * 30
const pubsubTimeCacheDuration = 10 * time.Minute
const broadcastGoRoutines = 1000
const durationBetweenPeersPrints = time.Second * 20
const defaultThresholdMinConnectedPeers = 3

//TODO remove the header size of the message when commit d3c5ecd3a3e884206129d9f2a9a4ddfd5e7c8951 from
// https://github.com/libp2p/go-libp2p-pubsub/pull/189/commits will be part of a new release
var messageHeader = 64 * 1024 //64kB
var maxSendBuffSize = (1 << 20) - messageHeader

var log = logger.GetOrCreate("p2p/libp2p")

//TODO refactor this struct to have be a wrapper (with logic) over a glue code
type networkMessenger struct {
	ctxProvider         *Libp2pContext
	pb                  *pubsub.PubSub
	ds                  p2p.DirectSender
	connMonitor         *libp2pConnectionMonitor
	peerDiscoverer      p2p.PeerDiscoverer
	mutTopics           sync.RWMutex
	topics              map[string]p2p.MessageProcessor
	outgoingPLB         p2p.ChannelLoadBalancer
	poc                 *peersOnChannel
	goRoutinesThrottler *throttler.NumGoRoutineThrottler
}

// NewNetworkMessenger creates a libP2P messenger by opening a port on the current machine
// Should be used in production!
func NewNetworkMessenger(
	ctx context.Context,
	port int,
	p2pPrivKey libp2pCrypto.PrivKey,
	conMgr connmgr.ConnManager,
	outgoingPLB p2p.ChannelLoadBalancer,
	peerDiscoverer p2p.PeerDiscoverer,
	listenAddress string,
	targetConnCount int,
) (*networkMessenger, error) {

	if ctx == nil {
		return nil, p2p.ErrNilContext
	}
	if port < 0 {
		return nil, p2p.ErrInvalidPort
	}
	if p2pPrivKey == nil {
		return nil, p2p.ErrNilP2PprivateKey
	}
	if outgoingPLB == nil || outgoingPLB.IsInterfaceNil() {
		return nil, p2p.ErrNilChannelLoadBalancer
	}
	if peerDiscoverer == nil || peerDiscoverer.IsInterfaceNil() {
		return nil, p2p.ErrNilPeerDiscoverer
	}

	address := fmt.Sprintf(listenAddress+"%d", port)
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(address),
		libp2p.Identity(p2pPrivKey),
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.ConnectionManager(conMgr),
		libp2p.DefaultTransports,
		//we need the disable relay option in order to save the node's bandwidth as much as possible
		libp2p.DisableRelay(),
		libp2p.NATPortMap(),
	}

	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	lctx, err := NewLibp2pContext(ctx, NewConnectableHost(h))
	if err != nil {
		log.LogIfError(h.Close())
		return nil, err
	}

	p2pNode, err := createMessenger(lctx, true, outgoingPLB, peerDiscoverer, targetConnCount)
	if err != nil {
		log.LogIfError(h.Close())
		return nil, err
	}

	goRoutinesThrottler, err := throttler.NewNumGoRoutineThrottler(broadcastGoRoutines)
	if err != nil {
		log.LogIfError(h.Close())
		return nil, err
	}

	p2pNode.goRoutinesThrottler = goRoutinesThrottler

	return p2pNode, nil
}

func createMessenger(
	lctx *Libp2pContext,
	withSigning bool,
	outgoingPLB p2p.ChannelLoadBalancer,
	peerDiscoverer p2p.PeerDiscoverer,
	targetConnCount int,
) (*networkMessenger, error) {

	pb, err := createPubSub(lctx, withSigning)
	if err != nil {
		return nil, err
	}

	err = peerDiscoverer.ApplyContext(lctx)
	if err != nil {
		return nil, err
	}

	reconnecter, _ := peerDiscoverer.(p2p.Reconnecter)

	netMes := networkMessenger{
		ctxProvider:    lctx,
		pb:             pb,
		topics:         make(map[string]p2p.MessageProcessor),
		outgoingPLB:    outgoingPLB,
		peerDiscoverer: peerDiscoverer,
	}
	netMes.connMonitor, err = newLibp2pConnectionMonitor(reconnecter, defaultThresholdMinConnectedPeers, targetConnCount)
	if err != nil {
		return nil, err
	}
	lctx.connHost.Network().Notify(netMes.connMonitor)

	netMes.ds, err = NewDirectSender(lctx.Context(), lctx.Host(), netMes.directMessageHandler)
	if err != nil {
		return nil, err
	}

	netMes.poc, err = newPeersOnChannel(
		pb.ListPeers,
		refreshPeersOnTopic,
		ttlPeersOnTopic)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			time.Sleep(durationBetweenPeersPrints)

			numConnectedPeers := len(netMes.ConnectedPeers())
			numKnownPeers := len(netMes.Peers())
			log.Debug("network connection status",
				"known peers", numKnownPeers,
				"connected peers", numConnectedPeers,
			)
		}
	}()

	go func(pubsub *pubsub.PubSub, plb p2p.ChannelLoadBalancer) {
		for {
			sendableData := plb.CollectOneElementFromChannels()

			if sendableData == nil {
				continue
			}

			errPublish := pb.Publish(sendableData.Topic, sendableData.Buff)
			if errPublish != nil {
				log.Trace("error sending data", "error", errPublish)
			}

			time.Sleep(durationBetweenSends)
		}
	}(pb, netMes.outgoingPLB)

	addresses := make([]interface{}, 0)
	for i, address := range netMes.ctxProvider.Host().Addrs() {
		addresses = append(addresses, fmt.Sprintf("addr%d", i))
		addresses = append(addresses, address.String()+"/p2p/"+netMes.ID().Pretty())
	}
	log.Info("listening on addresses", addresses...)

	return &netMes, nil
}

func createPubSub(ctxProvider *Libp2pContext, withSigning bool) (*pubsub.PubSub, error) {
	optsPS := []pubsub.Option{
		pubsub.WithMessageSigning(withSigning),
	}

	pubsub.TimeCacheDuration = pubsubTimeCacheDuration

	ps, err := pubsub.NewGossipSub(ctxProvider.Context(), ctxProvider.Host(), optsPS...)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

// Close closes the host, connections and streams
func (netMes *networkMessenger) Close() error {
	return netMes.ctxProvider.Host().Close()
}

// ID returns the messenger's ID
func (netMes *networkMessenger) ID() p2p.PeerID {
	h := netMes.ctxProvider.Host()

	return p2p.PeerID(h.ID())
}

// Peers returns the list of all known peers ID (including self)
func (netMes *networkMessenger) Peers() []p2p.PeerID {
	h := netMes.ctxProvider.Host()
	peers := make([]p2p.PeerID, 0)

	for _, p := range h.Peerstore().Peers() {
		peers = append(peers, p2p.PeerID(p))
	}
	return peers
}

// Addresses returns all addresses found in peerstore
func (netMes *networkMessenger) Addresses() []string {
	h := netMes.ctxProvider.Host()
	addrs := make([]string, 0)

	for _, address := range h.Addrs() {
		addrs = append(addrs, address.String()+"/p2p/"+netMes.ID().Pretty())
	}

	return addrs
}

// ConnectToPeer tries to open a new connection to a peer
func (netMes *networkMessenger) ConnectToPeer(address string) error {
	h := netMes.ctxProvider.Host()
	ctx := netMes.ctxProvider.ctx

	return h.ConnectToPeer(ctx, address)
}

// TrimConnections will trigger a manual sweep onto current connection set reducing the
// number of connections if needed
func (netMes *networkMessenger) TrimConnections() {
	h := netMes.ctxProvider.Host()
	ctx := netMes.ctxProvider.Context()

	h.ConnManager().TrimOpenConns(ctx)
}

// Bootstrap will start the peer discovery mechanism
func (netMes *networkMessenger) Bootstrap() error {
	return netMes.peerDiscoverer.Bootstrap()
}

// IsConnected returns true if current node is connected to provided peer
func (netMes *networkMessenger) IsConnected(peerID p2p.PeerID) bool {
	h := netMes.ctxProvider.Host()

	connectedness := h.Network().Connectedness(peer.ID(peerID))

	return connectedness == network.Connected
}

// ConnectedPeers returns the current connected peers list
func (netMes *networkMessenger) ConnectedPeers() []p2p.PeerID {
	h := netMes.ctxProvider.Host()

	connectedPeers := make(map[p2p.PeerID]struct{})

	for _, conn := range h.Network().Conns() {
		p := p2p.PeerID(conn.RemotePeer())

		if netMes.IsConnected(p) {
			connectedPeers[p] = struct{}{}
		}
	}

	peerList := make([]p2p.PeerID, len(connectedPeers))

	index := 0
	for k := range connectedPeers {
		peerList[index] = k
		index++
	}

	return peerList
}

// ConnectedAddresses returns all connected peer's addresses
func (netMes *networkMessenger) ConnectedAddresses() []string {
	h := netMes.ctxProvider.Host()
	conns := make([]string, 0)

	for _, c := range h.Network().Conns() {
		conns = append(conns, c.RemoteMultiaddr().String()+"/p2p/"+c.RemotePeer().Pretty())
	}
	return conns
}

// PeerAddress returns the peer's address or empty string if the peer is unknown
func (netMes *networkMessenger) PeerAddress(pid p2p.PeerID) string {
	h := netMes.ctxProvider.Host()

	//check if the peer is connected to return it's connected address
	for _, c := range h.Network().Conns() {
		if string(c.RemotePeer()) == string(pid.Bytes()) {
			return c.RemoteMultiaddr().String()
		}
	}

	//check in peerstore (maybe it is known but not connected)
	addresses := h.Peerstore().Addrs(peer.ID(pid.Bytes()))
	if len(addresses) == 0 {
		return ""
	}

	//return the first address from multi address slice
	return addresses[0].String()
}

// ConnectedPeersOnTopic returns the connected peers on a provided topic
func (netMes *networkMessenger) ConnectedPeersOnTopic(topic string) []p2p.PeerID {
	return netMes.poc.ConnectedPeersOnChannel(topic)
}

// CreateTopic opens a new topic using pubsub infrastructure
func (netMes *networkMessenger) CreateTopic(name string, createChannelForTopic bool) error {
	ctx := netMes.ctxProvider.Context()

	netMes.mutTopics.Lock()
	_, found := netMes.topics[name]
	if found {
		netMes.mutTopics.Unlock()
		return p2p.ErrTopicAlreadyExists
	}

	//TODO investigate if calling Subscribe on the pubsub impl does exactly the same thing as Topic.Subscribe
	// after calling pubsub.Join
	netMes.topics[name] = nil
	subscrRequest, err := netMes.pb.Subscribe(name)
	if err != nil {
		netMes.mutTopics.Unlock()
		return err
	}
	netMes.mutTopics.Unlock()

	if createChannelForTopic {
		err = netMes.outgoingPLB.AddChannel(name)
	}

	//just a dummy func to consume messages received by the newly created topic
	go func() {
		for {
			_, _ = subscrRequest.Next(ctx)
		}
	}()

	return err
}

// HasTopic returns true if the topic has been created
func (netMes *networkMessenger) HasTopic(name string) bool {
	netMes.mutTopics.RLock()
	_, found := netMes.topics[name]
	netMes.mutTopics.RUnlock()

	return found
}

// HasTopicValidator returns true if the topic has a validator set
func (netMes *networkMessenger) HasTopicValidator(name string) bool {
	netMes.mutTopics.RLock()
	validator := netMes.topics[name]
	netMes.mutTopics.RUnlock()

	return validator != nil
}

// OutgoingChannelLoadBalancer returns the channel load balancer object used by the messenger to send data
func (netMes *networkMessenger) OutgoingChannelLoadBalancer() p2p.ChannelLoadBalancer {
	return netMes.outgoingPLB
}

// BroadcastOnChannelBlocking tries to send a byte buffer onto a topic using provided channel
// It is a blocking method. It needs to be launched on a go routine
func (netMes *networkMessenger) BroadcastOnChannelBlocking(channel string, topic string, buff []byte) error {
	if len(buff) > maxSendBuffSize {
		return p2p.ErrMessageTooLarge
	}

	if !netMes.goRoutinesThrottler.CanProcess() {
		return p2p.ErrTooManyGoroutines
	}

	netMes.goRoutinesThrottler.StartProcessing()

	sendable := &p2p.SendableData{
		Buff:  buff,
		Topic: topic,
	}
	netMes.outgoingPLB.GetChannelOrDefault(channel) <- sendable
	netMes.goRoutinesThrottler.EndProcessing()
	return nil
}

// BroadcastOnChannel tries to send a byte buffer onto a topic using provided channel
func (netMes *networkMessenger) BroadcastOnChannel(channel string, topic string, buff []byte) {
	go func() {
		err := netMes.BroadcastOnChannelBlocking(channel, topic, buff)
		if err != nil {
			//TODO remove msg print, keep WARN log level
			log.Warn("p2p broadcast", "error", err.Error(), "msg", fmt.Sprintf("%s", buff))
		}
	}()
}

// Broadcast tries to send a byte buffer onto a topic using the topic name as channel
func (netMes *networkMessenger) Broadcast(topic string, buff []byte) {
	netMes.BroadcastOnChannel(topic, topic, buff)
}

// RegisterMessageProcessor registers a message process on a topic
func (netMes *networkMessenger) RegisterMessageProcessor(topic string, handler p2p.MessageProcessor) error {
	if check.IfNil(handler) {
		return p2p.ErrNilValidator
	}

	netMes.mutTopics.Lock()
	defer netMes.mutTopics.Unlock()
	validator, found := netMes.topics[topic]
	if !found {
		return p2p.ErrNilTopic
	}
	if validator != nil {
		return p2p.ErrTopicValidatorOperationNotSupported
	}

	broadcastHandler := func(buffToSend []byte) {
		netMes.Broadcast(topic, buffToSend)
	}

	err := netMes.pb.RegisterTopicValidator(topic, func(ctx context.Context, pid peer.ID, message *pubsub.Message) bool {
		wrappedMsg, err := NewMessage(message)
		if err != nil {
			log.Trace("p2p validator - new message", "error", err.Error(), "topics", message.TopicIDs)
			return false
		}
		err = handler.ProcessReceivedMessage(wrappedMsg, broadcastHandler)
		if err != nil {
			log.Trace("p2p validator",
				"error", err.Error(),
				"topics", message.TopicIDs,
				"pid", p2p.MessageOriginatorPid(wrappedMsg),
				"seq no", p2p.MessageOriginatorSeq(wrappedMsg),
			)

			return false
		}

		return true
	})
	if err != nil {
		return err
	}

	netMes.topics[topic] = handler
	return nil
}

// UnregisterMessageProcessor registers a message processes on a topic
func (netMes *networkMessenger) UnregisterMessageProcessor(topic string) error {
	netMes.mutTopics.Lock()
	defer netMes.mutTopics.Unlock()
	validator, found := netMes.topics[topic]

	if !found {
		return p2p.ErrNilTopic
	}

	if validator == nil {
		return p2p.ErrTopicValidatorOperationNotSupported
	}

	err := netMes.pb.UnregisterTopicValidator(topic)
	if err != nil {
		return err
	}

	netMes.topics[topic] = nil
	return nil
}

// SendToConnectedPeer sends a direct message to a connected peer
func (netMes *networkMessenger) SendToConnectedPeer(topic string, buff []byte, peerID p2p.PeerID) error {
	return netMes.ds.Send(topic, buff, peerID)
}

func (netMes *networkMessenger) directMessageHandler(message p2p.MessageP2P) error {
	var processor p2p.MessageProcessor

	netMes.mutTopics.RLock()
	processor = netMes.topics[message.TopicIDs()[0]]
	netMes.mutTopics.RUnlock()

	if processor == nil {
		return p2p.ErrNilValidator
	}

	go func(msg p2p.MessageP2P) {
		err := processor.ProcessReceivedMessage(msg, nil)
		if err != nil {
			log.Trace("p2p validator",
				"error", err.Error(),
				"topics", msg.TopicIDs(),
				"pid", p2p.MessageOriginatorPid(msg),
				"seq no", p2p.MessageOriginatorSeq(msg),
			)
		}
	}(message)

	return nil
}

// IsConnectedToTheNetwork returns true if the current node is connected to the network
func (netMes *networkMessenger) IsConnectedToTheNetwork() bool {
	netw := netMes.ctxProvider.connHost.Network()
	return netMes.connMonitor.isConnectedToTheNetwork(netw)
}

// SetThresholdMinConnectedPeers sets the minimum connected peers before triggering a new reconnection
func (netMes *networkMessenger) SetThresholdMinConnectedPeers(minConnectedPeers int) error {
	if minConnectedPeers < 0 {
		return p2p.ErrInvalidValue
	}

	netw := netMes.ctxProvider.connHost.Network()
	netMes.connMonitor.thresholdMinConnectedPeers = minConnectedPeers
	netMes.connMonitor.doReconnectionIfNeeded(netw)

	return nil
}

// ThresholdMinConnectedPeers returns the minimum connected peers before triggering a new reconnection
func (netMes *networkMessenger) ThresholdMinConnectedPeers() int {
	return netMes.connMonitor.thresholdMinConnectedPeers
}

// IsInterfaceNil returns true if there is no value under the interface
func (netMes *networkMessenger) IsInterfaceNil() bool {
	return netMes == nil
}
