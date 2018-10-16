package p2p

//import (
//	"context"
//	"crypto"
//	"encoding/base64"
//	"fmt"
//	"sync"
//	"time"
//
//	"github.com/ElrondNetwork/elrond-go-sandbox/marshal"
//	"github.com/ElrondNetwork/elrond-go-sandbox/p2p/mock"
//	"github.com/btcsuite/btcutil/base58"
//	"github.com/libp2p/go-libp2p-net"
//	"github.com/libp2p/go-libp2p-peer"
//	"github.com/multiformats/go-multiaddr"
//	"github.com/pkg/errors"
//)
//
//var mutGloballyRegPeers *sync.Mutex
//
//// GloballyRegisteredPeers is the main map used for in memory communication
//var GloballyRegisteredPeers map[peer.ID]*MemoryMessenger
//
//func init() {
//	mutGloballyRegPeers = &sync.Mutex{}
//	GloballyRegisteredPeers = make(map[peer.ID]*MemoryMessenger)
//}
//
//// MemoryMessenger is a fake memory Messenger used for testing
//type MemoryMessenger struct {
//	peerID peer.ID
//
//	mutConnectedPeers sync.Mutex
//	connectedPeers    map[peer.ID]*MemoryMessenger
//
//	marsh marshal.Marshalizer
//	rt    *RoutingTable
//	queue *MessageQueue
//
//	maxAllowedPeers int
//
//	onMsgRecv func(caller Messenger, peerID string, m *Message)
//
//	mutClosed sync.RWMutex
//	closed    bool
//}
//
//// NewMemoryMessenger instantiate a new memory Messenger
//func NewMemoryMessenger(m marshal.Marshalizer, pid peer.ID, maxAllowedPeers int) (*MemoryMessenger, error) {
//	if m == nil {
//		return nil, errors.New("Marshalizer is nil! Can't create messenger!")
//	}
//
//	mm := MemoryMessenger{peerID: pid, marsh: m, maxAllowedPeers: maxAllowedPeers, closed: false}
//
//	mm.mutConnectedPeers = sync.Mutex{}
//	mm.mutConnectedPeers.Lock()
//	mm.connectedPeers = make(map[peer.ID]*MemoryMessenger)
//	mm.mutConnectedPeers.Unlock()
//
//	mm.rt = NewRoutingTable(mm.peerID)
//	mm.queue = NewMessageQueue(50000)
//
//	mm.mutClosed = sync.RWMutex{}
//
//	mutGloballyRegPeers.Lock()
//	GloballyRegisteredPeers[pid] = &mm
//	mutGloballyRegPeers.Unlock()
//
//	return &mm, nil
//}
//
//// Closes a MemoryMessenger. Receiving and sending data is no longer possible
//func (mm *MemoryMessenger) Close() error {
//	mm.mutClosed.Lock()
//	mm.closed = true
//	mm.mutClosed.Unlock()
//	return nil
//}
//
//// ID returns the current id
//func (mm *MemoryMessenger) ID() peer.ID {
//	return mm.peerID
//}
//
//// Peers returns the connected peers list
//func (mm *MemoryMessenger) Peers() []peer.ID {
//	peers := make([]peer.ID, 0)
//
//	mm.mutConnectedPeers.Lock()
//	for k := range mm.connectedPeers {
//		peers = append(peers, k)
//	}
//	mm.mutConnectedPeers.Unlock()
//
//	return peers
//}
//
//// Conns return the connections made by this memory messenger
//func (mm *MemoryMessenger) Conns() []net.Conn {
//	conns := make([]net.Conn, 0)
//
//	mm.mutConnectedPeers.Lock()
//	for k := range mm.connectedPeers {
//		c := &mock.ConnMock{LocalP: mm.peerID, RemoteP: k}
//		conns = append(conns, c)
//	}
//	mm.mutConnectedPeers.Unlock()
//
//	return conns
//}
//
//// Marshalizer returns the used marshalizer object
//func (mm *MemoryMessenger) Marshalizer() marshal.Marshalizer {
//	return mm.marsh
//}
//
//// RouteTable will return the RoutingTable object
//func (mm *MemoryMessenger) RouteTable() *RoutingTable {
//	return mm.rt
//}
//
//// Addrs will return all addresses bind to current messenger
//func (mm *MemoryMessenger) Addrs() []string {
//	return []string{string(mm.peerID.Pretty())}
//}
//
//// OnRecvMsg returns the function used as callback whenever a new message arrives
//func (mm *MemoryMessenger) OnRecvMsg() func(caller Messenger, peerID string, m *Message) {
//	return mm.onMsgRecv
//}
//
//// SetOnRecvMsg sets the function used as callback whenever a new message arrives
//func (mm *MemoryMessenger) SetOnRecvMsg(f func(caller Messenger, peerID string, m *Message)) {
//	mm.onMsgRecv = f
//}
//
//// ConnectToAddresses is used to explicitly connect to a well known set of addresses
//func (mm *MemoryMessenger) ConnectToAddresses(ctx context.Context, addresses []string) {
//	for i := 0; i < len(addresses); i++ {
//		addr := peer.ID(base58.Decode(addresses[i]))
//
//		mutGloballyRegPeers.Lock()
//		val, ok := GloballyRegisteredPeers[addr]
//		mutGloballyRegPeers.Unlock()
//
//		if !ok {
//			fmt.Printf("Bootstrapping the peer '%v' failed! [not found]\n", addresses[i])
//			continue
//		}
//
//		if mm.peerID == addr {
//			//won't add self
//			continue
//		}
//
//		mm.mutConnectedPeers.Lock()
//		//connect this to other peer
//		mm.connectedPeers[addr] = val
//		//connect other the other peer to this
//		val.connectedPeers[mm.peerID] = mm
//		mm.mutConnectedPeers.Unlock()
//	}
//}
//
//func (mm *MemoryMessenger) gotNewMessage(sender *MemoryMessenger, buff []byte) {
//	mm.mutClosed.RLock()
//	if mm.closed {
//		mm.mutClosed.RUnlock()
//		return
//	}
//	mm.mutClosed.RUnlock()
//
//	if len(buff) == 0 {
//		return
//	}
//
//	if (len(buff) == 1) && (buff[0] == byte('\n')) {
//		return
//	}
//
//	if buff[len(buff)-1] == byte('\n') {
//		buff = buff[0 : len(buff)-1]
//	}
//
//	m, err := CreateFromByteArray(mm.marsh, buff)
//
//	if err != nil {
//		return
//	}
//
//	err = m.VerifyAndSetSigned()
//	if err != nil {
//		return
//	}
//
//	sha3 := crypto.SHA3_256.New()
//	b64 := base64.StdEncoding
//	hash := b64.EncodeToString(sha3.Sum([]byte(m.Payload)))
//
//	if mm.queue.ContainsAndAdd(hash) {
//		return
//	}
//
//	if mm.onMsgRecv != nil {
//		go mm.onMsgRecv(mm, sender.peerID.Pretty(), m)
//	}
//}
//
//func (mm *MemoryMessenger) sendDirectRAW(peerID string, buff []byte) error {
//	mm.mutClosed.RLock()
//	if mm.closed {
//		mm.mutClosed.RUnlock()
//		return &NodeError{PeerRecv: peerID, PeerSend: mm.ID().Pretty(), Err: "Attempt to write on a closed messenger!\n"}
//	}
//	mm.mutClosed.RUnlock()
//
//	if peerID == mm.ID().Pretty() {
//		//send to self allowed
//		mm.gotNewMessage(mm, buff)
//		return nil
//	}
//
//	mm.mutConnectedPeers.Lock()
//	val, ok := mm.connectedPeers[peer.ID(base58.Decode(peerID))]
//	mm.mutConnectedPeers.Unlock()
//
//	if (!ok) || (val == nil) {
//		return &NodeError{PeerRecv: peerID, PeerSend: mm.ID().Pretty(), Err: fmt.Sprintf("Can not send to %v. Not connected?\n", peerID)}
//	}
//
//	val.gotNewMessage(mm, buff)
//
//	return nil
//}
//
//// SendDirectBuff allows to send a slice of bytes directly to a peer. It assumes that the connection has been done already
//func (mm *MemoryMessenger) SendDirectBuff(peerID string, buff []byte) error {
//	m := NewMessage(mm.ID().Pretty(), buff, mm.marsh)
//
//	buff, err := m.ToByteArray()
//	if err != nil {
//		return &NodeError{PeerRecv: peerID, PeerSend: mm.ID().Pretty(), Err: err.Error()}
//	}
//
//	return mm.sendDirectRAW(peerID, buff)
//}
//
//// SendDirectString allows to send a string to a peer. It assumes that the connection has been done already
//func (mm *MemoryMessenger) SendDirectString(peerID string, message string) error {
//	return mm.SendDirectBuff(peerID, []byte(message))
//}
//
//// SendDirectMessage allows to send a message directly to a peer. It assumes that the connection has been done already
//func (mm *MemoryMessenger) SendDirectMessage(peerID string, m *Message) error {
//	if m == nil {
//		return &NodeError{PeerRecv: peerID, PeerSend: mm.ID().Pretty(), Err: fmt.Sprintf("Can not send NIL message!\n")}
//	}
//
//	buff, err := m.ToByteArray()
//	if err != nil {
//		return &NodeError{PeerRecv: peerID, PeerSend: mm.ID().Pretty(), Err: err.Error()}
//	}
//
//	return mm.sendDirectRAW(peerID, buff)
//}
//
//func (mm *MemoryMessenger) broadcastRAW(buff []byte, excs []string) error {
//	mm.mutClosed.RLock()
//	if mm.closed {
//		mm.mutClosed.RUnlock()
//		return &NodeError{PeerRecv: "", PeerSend: mm.ID().Pretty(), Err: "Attempt to write on a closed messenger!\n"}
//	}
//	mm.mutClosed.RUnlock()
//
//	var errFound = &NodeError{}
//
//	peers := mm.Peers()
//
//	for i := 0; i < len(peers); i++ {
//		peerID := peer.ID(peers[i]).Pretty()
//
//		if peerID == mm.ID().Pretty() {
//			continue
//		}
//
//		found := false
//		for j := 0; j < len(excs); j++ {
//			if peerID == excs[j] {
//				found = true
//				break
//			}
//		}
//
//		if found {
//			continue
//		}
//
//		err := mm.sendDirectRAW(peerID, buff)
//
//		if err != nil {
//			errNode, _ := err.(*NodeError)
//			errFound.NestedErrors = append(errFound.NestedErrors, *errNode)
//		}
//	}
//
//	if len(peers) == 0 {
//		return &NodeError{PeerRecv: "", PeerSend: mm.ID().Pretty(), Err: "Attempt to send to no one!\n"}
//	}
//
//	if len(errFound.NestedErrors) == 0 {
//		return nil
//	}
//
//	if len(errFound.NestedErrors) == 1 {
//		return &errFound.NestedErrors[0]
//	}
//
//	errFound.Err = "Multiple errors found!"
//	return errFound
//}
//
//// BroadcastBuff allows to send a slice of bytes directly to a all connected peers
//func (mm *MemoryMessenger) BroadcastBuff(buff []byte, excs []string) error {
//	m := NewMessage(mm.peerID.Pretty(), buff, mm.marsh)
//
//	buff, err := m.ToByteArray()
//	if err != nil {
//		return &NodeError{PeerRecv: "", PeerSend: mm.peerID.Pretty(), Err: err.Error()}
//	}
//
//	return mm.broadcastRAW(buff, excs)
//}
//
//// BroadcastString allows to send a string directly to a all connected peers
//func (mm *MemoryMessenger) BroadcastString(message string, excs []string) error {
//	return mm.BroadcastBuff([]byte(message), excs)
//}
//
//// BroadcastMessage allows to send a message directly to a all connected peers
//func (mm *MemoryMessenger) BroadcastMessage(m *Message, excs []string) error {
//	if m == nil {
//		return &NodeError{PeerRecv: "", PeerSend: mm.ID().Pretty(), Err: fmt.Sprintf("Can not broadcast NIL message!\n")}
//	}
//
//	buff, err := m.ToByteArray()
//	if err != nil {
//		return &NodeError{PeerRecv: "", PeerSend: mm.ID().Pretty(), Err: err.Error()}
//	}
//
//	return mm.broadcastRAW(buff, excs)
//}
//
//// StreamHandler is a dummy function. Will panic
//func (mm *MemoryMessenger) StreamHandler(stream net.Stream) {
//	panic("implement me")
//}
//
//// Bootstrap will try to connect to as many peers as possible
//func (mm *MemoryMessenger) Bootstrap(ctx context.Context) {
//	go mm.doBootstrap()
//}
//
//func (mm *MemoryMessenger) doBootstrap() {
//	for {
//		mm.mutClosed.RLock()
//		if mm.closed {
//			mm.mutClosed.RUnlock()
//			return
//		}
//		mm.mutClosed.RUnlock()
//
//		temp := make(map[peer.ID]*MemoryMessenger, 0)
//
//		mutGloballyRegPeers.Lock()
//		for k, v := range GloballyRegisteredPeers {
//			if !mm.rt.Has(k) {
//				mm.rt.Update(k)
//
//				temp[k] = v
//			}
//		}
//		mutGloballyRegPeers.Unlock()
//
//		mm.mutConnectedPeers.Lock()
//		for k, v := range temp {
//			mm.connectedPeers[k] = v
//		}
//		mm.mutConnectedPeers.Unlock()
//
//		time.Sleep(time.Second)
//	}
//
//}
//
//// PrintConnected displays the connected peers
//func (mm *MemoryMessenger) PrintConnected() {
//	conns := mm.Conns()
//
//	fmt.Printf("Node %s is connected to: \n", mm.ID().Pretty())
//
//	for i := 0; i < len(conns); i++ {
//		fmt.Printf("\t- %s with distance %d\n", conns[i].RemotePeer().Pretty(),
//			ComputeDistanceAD(mm.ID(), conns[i].RemotePeer()))
//	}
//}
//
//// AddAddr adds a new address to peer store
//func (mm *MemoryMessenger) AddAddr(p peer.ID, addr multiaddr.Multiaddr, ttl time.Duration) {
//	mutGloballyRegPeers.Lock()
//	val, ok := GloballyRegisteredPeers[p]
//	mutGloballyRegPeers.Unlock()
//
//	if !ok {
//		val = nil
//	}
//
//	mm.mutConnectedPeers.Lock()
//	mm.connectedPeers[p] = val
//	mm.mutConnectedPeers.Unlock()
//}
//
//// Connectedness tests for a connection between self and another peer
//func (mm *MemoryMessenger) Connectedness(pid peer.ID) net.Connectedness {
//	mm.mutConnectedPeers.Lock()
//	_, ok := mm.connectedPeers[pid]
//	mm.mutConnectedPeers.Unlock()
//
//	if ok {
//		return net.Connected
//	} else {
//		return net.NotConnected
//	}
//
//}
