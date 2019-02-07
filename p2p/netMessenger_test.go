package p2p_test

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go-sandbox/p2p"
	"github.com/ElrondNetwork/elrond-go-sandbox/p2p/mock"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type testNetStringCreator struct {
	Data string
}

type structNetTest1 struct {
	Nonce int
	Data  float64
}

type structNetTest2 struct {
	Nonce string
	Data  []byte
}

//------- testNetStringNewer

// New will return a new instance of string. Dummy, just to implement Cloner interface as strings are immutable
func (sc *testNetStringCreator) Create() p2p.Creator {
	return &testNetStringCreator{}
}

// ID will return the same string as ID
func (sc *testNetStringCreator) ID() string {
	return sc.Data
}

//------- structNetTest1

func (s1 *structNetTest1) Create() p2p.Creator {
	return &structNetTest1{}
}

func (s1 *structNetTest1) ID() string {
	return strconv.Itoa(s1.Nonce)
}

//------- structNetTest2

func (s2 *structNetTest2) Create() p2p.Creator {
	return &structNetTest2{}
}

func (s2 *structNetTest2) ID() string {
	return s2.Nonce
}

var testNetMessengerMaxWaitResponse = time.Duration(time.Second * 5)
var testNetMessengerWaitResponseUnreceivedMsg = time.Duration(time.Second)

var pubsubAnnounceDuration = time.Second * 2

var startingPort = 4000

func createNetMessenger(t *testing.T, port int, nConns int) (*p2p.NetMessenger, error) {
	return createNetMessengerPubSub(t, port, nConns, p2p.FloodSub)
}

func createNetMessengerPubSub(t *testing.T, port int, nConns int, strategy p2p.PubSubStrategy) (*p2p.NetMessenger, error) {
	cp, err := p2p.NewConnectParamsFromPort(port)
	assert.Nil(t, err)

	return p2p.NewNetMessenger(context.Background(), &mock.MarshalizerMock{}, &mock.HasherMock{}, cp, nConns, strategy)
}

func waitForConnectionsToBeMade(nodes []p2p.Messenger, connectGraph map[int][]int, chanDone chan bool) {
	for {
		fullyConnected := true

		//for each element in the connect graph, check that is really connected to other peers
		for k, v := range connectGraph {
			for _, peerIndex := range v {
				if nodes[k].Connectedness(nodes[peerIndex].ID()) != net.Connected {
					fullyConnected = false
					break
				}
			}
		}

		if fullyConnected {
			break
		}

		time.Sleep(time.Millisecond)
	}
	chanDone <- true
}

func waitForWaitGroup(wg *sync.WaitGroup, chanDone chan bool) {
	wg.Wait()
	chanDone <- true
}

func waitForValue(value *int32, expected int32, chanDone chan bool) {
	for {
		if atomic.LoadInt32(value) == expected {
			break
		}

		time.Sleep(time.Nanosecond)
	}

	chanDone <- true
}

func closeAllNodes(nodes []p2p.Messenger) {
	fmt.Println("### Closing nodes... ###")
	for i := 0; i < len(nodes); i++ {
		err := nodes[i].Close()
		if err != nil {
			p2p.Log.Error(err.Error())
		}
	}
}

func getConnectableAddress(addresses []string) string {
	for _, addr := range addresses {
		if strings.Contains(addr, "127.0.0.1") {
			return addr
		}
	}

	return ""
}

func createTestNetwork(t *testing.T) ([]p2p.Messenger, error) {
	nodes := make([]p2p.Messenger, 0)

	//create 5 nodes
	for i := 0; i < 5; i++ {
		node, err := createNetMessenger(t, startingPort+i, 10)
		assert.Nil(t, err)

		nodes = append(nodes, node)

		fmt.Printf("Node %v is %s\n", i+1, getConnectableAddress(node.Addresses()))
	}

	//connect one with each other manually
	// node0 --------- node1
	//   |               |
	//   +------------ node2
	//   |               |
	//   |             node3
	//   |               |
	//   +------------ node4

	nodes[1].ConnectToAddresses(context.Background(), []string{getConnectableAddress(nodes[0].Addresses())})
	nodes[2].ConnectToAddresses(context.Background(), []string{
		getConnectableAddress(nodes[1].Addresses()),
		getConnectableAddress(nodes[0].Addresses())})
	nodes[3].ConnectToAddresses(context.Background(), []string{getConnectableAddress(nodes[2].Addresses())})
	nodes[4].ConnectToAddresses(context.Background(), []string{
		getConnectableAddress(nodes[3].Addresses()),
		getConnectableAddress(nodes[0].Addresses())})

	connectGraph := make(map[int][]int)
	connectGraph[0] = []int{1, 2, 4}
	connectGraph[1] = []int{0, 2}
	connectGraph[2] = []int{0, 1, 3}
	connectGraph[3] = []int{2, 4}
	connectGraph[4] = []int{0, 3}

	chanDone := make(chan bool, 0)
	go waitForConnectionsToBeMade(nodes, connectGraph, chanDone)
	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		return nodes, errors.New("could not make connections")
	}

	return nodes, nil
}

func TestNetMessenger_RecreationSameNodeShouldWork(t *testing.T) {
	fmt.Println()

	nodes := make([]p2p.Messenger, 0)

	defer func() {
		closeAllNodes(nodes)
	}()

	node, _ := createNetMessenger(t, startingPort, 10)
	nodes = append(nodes, node)

	node, err := createNetMessenger(t, startingPort, 10)
	assert.Nil(t, err)
	nodes = append(nodes, node)

	if nodes[0].ID().Pretty() != nodes[1].ID().Pretty() {
		t.Fatal("ID mismatch")
	}
}

func TestNetMessenger_SendToSelfShouldWork(t *testing.T) {
	nodes := make([]p2p.Messenger, 0)

	defer func() {
		closeAllNodes(nodes)
	}()

	node, err := createNetMessenger(t, startingPort, 10)
	assert.Nil(t, err)
	nodes = append(nodes, node)

	wg := sync.WaitGroup{}
	wg.Add(1)
	chanDone := make(chan bool)
	go waitForWaitGroup(&wg, chanDone)

	err = nodes[0].AddTopic(p2p.NewTopic("test topic", &testNetStringCreator{}, &mock.MarshalizerMock{}))
	nodes[0].GetTopic("test topic").AddDataReceived(func(name string, data interface{}, msgInfo *p2p.MessageInfo) {
		payload := (*data.(*testNetStringCreator)).Data

		fmt.Printf("Got message: %v\n", payload)

		if payload == "ABC" {
			wg.Done()
		}
	})
	assert.Nil(t, err)

	err = nodes[0].GetTopic("test topic").Broadcast(testNetStringCreator{Data: "ABC"})
	assert.Nil(t, err)

	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "Should have been 1 (message received to self)")
	}
}

func TestNetMessenger_NodesPingPongOn2TopicsShouldWork(t *testing.T) {
	fmt.Println()

	nodes := make([]p2p.Messenger, 0)

	defer func() {
		closeAllNodes(nodes)
	}()

	node, err := createNetMessenger(t, startingPort, 10)
	assert.Nil(t, err)
	nodes = append(nodes, node)

	node, err = createNetMessenger(t, startingPort+1, 10)
	assert.Nil(t, err)
	nodes = append(nodes, node)

	connectGraph := make(map[int][]int)
	connectGraph[0] = []int{1}
	connectGraph[1] = []int{0}

	nodes[0].ConnectToAddresses(context.Background(), []string{getConnectableAddress(nodes[1].Addresses())})

	wg := sync.WaitGroup{}
	chanDone := make(chan bool)
	go waitForConnectionsToBeMade(nodes, connectGraph, chanDone)
	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "Could not make a connection between the 2 peers")
		return
	}

	fmt.Printf("Node 1 is %s\n", nodes[0].Addresses()[0])
	fmt.Printf("Node 2 is %s\n", nodes[1].Addresses()[0])

	fmt.Printf("Node 1 has the addresses: %v\n", nodes[0].Addresses())
	fmt.Printf("Node 2 has the addresses: %v\n", nodes[1].Addresses())

	//create 2 topics on each node
	err = nodes[0].AddTopic(p2p.NewTopic("ping", &testNetStringCreator{}, &mock.MarshalizerMock{}))
	assert.Nil(t, err)
	err = nodes[0].AddTopic(p2p.NewTopic("pong", &testNetStringCreator{}, &mock.MarshalizerMock{}))
	assert.Nil(t, err)

	err = nodes[1].AddTopic(p2p.NewTopic("ping", &testNetStringCreator{}, &mock.MarshalizerMock{}))
	assert.Nil(t, err)
	err = nodes[1].AddTopic(p2p.NewTopic("pong", &testNetStringCreator{}, &mock.MarshalizerMock{}))
	assert.Nil(t, err)

	time.Sleep(pubsubAnnounceDuration)

	wg.Add(2)
	go waitForWaitGroup(&wg, chanDone)

	//assign some event handlers on topics
	nodes[0].GetTopic("ping").AddDataReceived(func(name string, data interface{}, msgInfo *p2p.MessageInfo) {
		payload := (*data.(*testNetStringCreator)).Data

		if payload == "ping string" {
			fmt.Println("Ping received, sending pong...")
			err = nodes[0].GetTopic("pong").Broadcast(testNetStringCreator{"pong string"})
			assert.Nil(t, err)
		}
	})

	nodes[0].GetTopic("pong").AddDataReceived(func(name string, data interface{}, msgInfo *p2p.MessageInfo) {
		payload := (*data.(*testNetStringCreator)).Data

		fmt.Printf("node1 received: %v\n", payload)

		if payload == "pong string" {
			fmt.Println("Pong received!")
			wg.Done()
		}
	})

	//for node2 topic ping we do not need an event handler in this test
	nodes[1].GetTopic("pong").AddDataReceived(func(name string, data interface{}, msgInfo *p2p.MessageInfo) {
		payload := (*data.(*testNetStringCreator)).Data

		fmt.Printf("node2 received: %v\n", payload)

		if payload == "pong string" {
			fmt.Println("Pong received!")
			wg.Done()
		}
	})

	err = nodes[1].GetTopic("ping").Broadcast(testNetStringCreator{"ping string"})
	assert.Nil(t, err)

	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "Should have been 2 (pong from node1: self and node2: received from node1)")
	}
}

func TestNetMessenger_SimpleBroadcast5nodesInlineShouldWork(t *testing.T) {
	fmt.Println()

	nodes := make([]p2p.Messenger, 0)

	defer func() {
		closeAllNodes(nodes)
	}()

	//create 5 nodes
	for i := 0; i < 5; i++ {
		node, err := createNetMessenger(t, startingPort+i, 10)
		assert.Nil(t, err)

		nodes = append(nodes, node)

		fmt.Printf("Node %v is %s\n", i+1, node.Addresses()[0])
	}

	//connect one with each other daisy-chain
	for i := 1; i < 5; i++ {
		node := nodes[i]
		node.ConnectToAddresses(context.Background(), []string{getConnectableAddress(nodes[i-1].Addresses())})
	}

	connectGraph := make(map[int][]int)
	connectGraph[0] = []int{1}
	connectGraph[1] = []int{0, 2}
	connectGraph[2] = []int{1, 3}
	connectGraph[3] = []int{2, 4}
	connectGraph[4] = []int{3}

	chanDone := make(chan bool)
	go waitForConnectionsToBeMade(nodes, connectGraph, chanDone)
	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "Could not make connections")
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(5)
	go waitForWaitGroup(&wg, chanDone)

	//print connected and create topics
	for i := 0; i < 5; i++ {
		node := nodes[i]
		node.PrintConnected()

		err := node.AddTopic(p2p.NewTopic("test", &testNetStringCreator{}, &mock.MarshalizerMock{}))
		node.GetTopic("test").AddDataReceived(
			func(name string, data interface{}, msgInfo *p2p.MessageInfo) {
				fmt.Printf("%v received from %v: %v\n", node.ID(), msgInfo.Peer, data.(*testNetStringCreator).Data)
				wg.Done()
			})
		assert.Nil(t, err)
	}

	fmt.Println()
	fmt.Println()

	time.Sleep(pubsubAnnounceDuration)

	fmt.Println("Broadcasting...")
	err := nodes[0].GetTopic("test").Broadcast(testNetStringCreator{Data: "Foo"})
	assert.Nil(t, err)

	select {
	case <-chanDone:
		fmt.Println("Got all messages!")
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "not all messages were received")
	}
}

func TestNetMessenger_SimpleBroadcast5nodesBetterConnectedShouldWork(t *testing.T) {
	var nodes []p2p.Messenger

	defer func() {
		closeAllNodes(nodes)
	}()

	nodes, err := createTestNetwork(t)
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}

	chanDone := make(chan bool, 0)

	wg := sync.WaitGroup{}
	wg.Add(5)
	go waitForWaitGroup(&wg, chanDone)

	//print connected and create topics
	for i := 0; i < 5; i++ {
		node := nodes[i]
		node.PrintConnected()

		err := node.AddTopic(p2p.NewTopic("test", &testNetStringCreator{}, &mock.MarshalizerMock{}))
		node.GetTopic("test").AddDataReceived(
			func(name string, data interface{}, msgInfo *p2p.MessageInfo) {
				fmt.Printf("%v received from %v: %v\n", node.ID(), msgInfo.Peer, data.(*testNetStringCreator).Data)
				wg.Done()
			})
		assert.Nil(t, err)
	}

	fmt.Println()
	fmt.Println()

	time.Sleep(pubsubAnnounceDuration)

	fmt.Println("Broadcasting...")
	err = nodes[0].GetTopic("test").Broadcast(testNetStringCreator{Data: "Foo"})
	assert.Nil(t, err)

	select {
	case <-chanDone:
		fmt.Println("Got all messages!")
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "not all messages were received")
	}
}

func TestNetMessenger_SendingNilShouldErr(t *testing.T) {
	nodes := make([]p2p.Messenger, 0)

	defer func() {
		closeAllNodes(nodes)
	}()

	node, err := createNetMessenger(t, startingPort, 10)
	assert.Nil(t, err)
	nodes = append(nodes, node)

	err = node.AddTopic(p2p.NewTopic("test", &testNetStringCreator{}, &mock.MarshalizerMock{}))
	assert.Nil(t, err)
	err = node.GetTopic("test").Broadcast(nil)
	assert.NotNil(t, err)
}

func TestNetMessenger_CreateNodeWithNilMarshalizerShouldErr(t *testing.T) {
	cp, err := p2p.NewConnectParamsFromPort(startingPort)
	assert.Nil(t, err)

	_, err = p2p.NewNetMessenger(context.Background(), nil, &mock.HasherMock{}, cp, 10, p2p.FloodSub)
	assert.NotNil(t, err)
}

func TestNetMessenger_CreateNodeWithNilHasherShouldErr(t *testing.T) {
	cp, err := p2p.NewConnectParamsFromPort(startingPort)
	assert.Nil(t, err)

	_, err = p2p.NewNetMessenger(context.Background(), &mock.MarshalizerMock{}, nil, cp, 10, p2p.FloodSub)
	assert.NotNil(t, err)
}

func TestNetMessenger_BadObjectToUnmarshalShouldFilteredOut(t *testing.T) {
	//stress test to check if the node is able to cope
	//with unmarshaling a bad object
	//both structs have the same fields but incompatible types

	//node1 registers topic 'test' with struct1
	//node2 registers topic 'test' with struct2

	nodes := make([]p2p.Messenger, 0)

	defer func() {
		closeAllNodes(nodes)
	}()

	node, err := createNetMessenger(t, startingPort, 10)
	assert.Nil(t, err)
	nodes = append(nodes, node)

	node, err = createNetMessenger(t, startingPort+1, 10)
	assert.Nil(t, err)
	nodes = append(nodes, node)

	//connect nodes
	nodes[0].ConnectToAddresses(context.Background(), []string{getConnectableAddress(nodes[1].Addresses())})

	connectGraph := make(map[int][]int)
	connectGraph[0] = []int{1}
	connectGraph[1] = []int{0}

	chanDone := make(chan bool)
	go waitForConnectionsToBeMade(nodes, connectGraph, chanDone)
	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "Could not make a connection between the 2 peers")
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go waitForWaitGroup(&wg, chanDone)

	//create topics for each node
	err = nodes[0].AddTopic(p2p.NewTopic("test", &structNetTest1{}, &mock.MarshalizerMock{}))
	assert.Nil(t, err)
	err = nodes[1].AddTopic(p2p.NewTopic("test", &structNetTest2{}, &mock.MarshalizerMock{}))
	assert.Nil(t, err)

	time.Sleep(pubsubAnnounceDuration)

	//node 1 sends, node 2 receives
	nodes[1].GetTopic("test").AddDataReceived(func(name string, data interface{}, msgInfo *p2p.MessageInfo) {
		fmt.Printf("received: %v", data)
		wg.Done()
	})

	err = nodes[0].GetTopic("test").Broadcast(&structNetTest1{Nonce: 4, Data: 4.5})
	assert.Nil(t, err)

	select {
	case <-chanDone:
		assert.Fail(t, "Should have not received the message")
	case <-time.After(testNetMessengerWaitResponseUnreceivedMsg):
	}
}

func TestNetMessenger_BroadcastOnInexistentTopicShouldFilteredOut(t *testing.T) {
	//stress test to check if the node is able to cope
	//with receiving on an inexistent topic

	nodes := make([]p2p.Messenger, 0)

	defer func() {
		closeAllNodes(nodes)
	}()

	node, err := createNetMessenger(t, startingPort, 10)
	assert.Nil(t, err)
	nodes = append(nodes, node)

	node, err = createNetMessenger(t, startingPort+1, 10)
	assert.Nil(t, err)
	nodes = append(nodes, node)

	//connect nodes
	nodes[0].ConnectToAddresses(context.Background(), []string{getConnectableAddress(nodes[1].Addresses())})

	connectGraph := make(map[int][]int)
	connectGraph[0] = []int{1}
	connectGraph[1] = []int{0}

	chanDone := make(chan bool)
	go waitForConnectionsToBeMade(nodes, connectGraph, chanDone)
	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "Could not make a connection between the 2 peers")
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go waitForWaitGroup(&wg, chanDone)

	//create topics for each node
	err = nodes[0].AddTopic(p2p.NewTopic("test1", &testNetStringCreator{}, &mock.MarshalizerMock{}))
	assert.Nil(t, err)
	err = nodes[1].AddTopic(p2p.NewTopic("test2", &testNetStringCreator{}, &mock.MarshalizerMock{}))
	assert.Nil(t, err)

	time.Sleep(pubsubAnnounceDuration)

	//node 1 sends, node 2 receives
	nodes[1].GetTopic("test2").AddDataReceived(func(name string, data interface{}, msgInfo *p2p.MessageInfo) {
		fmt.Printf("received: %v", data)
		wg.Done()
	})

	err = nodes[0].GetTopic("test1").Broadcast(testNetStringCreator{"Foo"})
	assert.Nil(t, err)

	select {
	case <-chanDone:
		assert.Fail(t, "Should have not received the message")
	case <-time.After(testNetMessengerWaitResponseUnreceivedMsg):
	}
}

func TestNetMessenger_BroadcastWithValidatorsShouldWork(t *testing.T) {
	var nodes []p2p.Messenger

	defer func() {
		closeAllNodes(nodes)
	}()

	nodes, err := createTestNetwork(t)
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}

	chanDone := make(chan bool, 0)

	wg := sync.WaitGroup{}

	recv := func(name string, data interface{}, msgInfo *p2p.MessageInfo) {
		fmt.Printf("%v got from %v the message: %v\n", msgInfo.CurrentPeer, msgInfo.Peer, data)
		wg.Done()
	}

	//print connected and create topics
	for i := 0; i < 5; i++ {
		node := nodes[i]
		node.PrintConnected()

		err := node.AddTopic(p2p.NewTopic("test", &testNetStringCreator{}, &mock.MarshalizerMock{}))
		assert.Nil(t, err)
		node.GetTopic("test").AddDataReceived(recv)
	}

	time.Sleep(pubsubAnnounceDuration)

	// dummy validator that prevents propagation of "AAA" message
	v := func(ctx context.Context, mes *pubsub.Message) bool {
		obj := &testNetStringCreator{}

		marsh := mock.MarshalizerMock{}
		err := marsh.Unmarshal(obj, mes.GetData())
		assert.Nil(t, err)

		return obj.Data != "AAA"
	}

	//node 2 has validator in place
	err = nodes[2].GetTopic("test").RegisterValidator(v)
	assert.Nil(t, err)

	fmt.Println()
	fmt.Println()

	//send AAA, wait, check that 4 peers got the message
	fmt.Println("Broadcasting AAA...")
	wg.Add(4)
	go waitForWaitGroup(&wg, chanDone)
	err = nodes[0].GetTopic("test").Broadcast(testNetStringCreator{Data: "AAA"})
	assert.Nil(t, err)

	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "not all 4 peers got AAA message")
		return
	}

	//send BBB, wait, check that all peers got the message
	fmt.Println("Broadcasting BBB...")
	wg.Add(5)
	go waitForWaitGroup(&wg, chanDone)

	err = nodes[0].GetTopic("test").Broadcast(testNetStringCreator{Data: "BBB"})
	assert.Nil(t, err)

	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "not all 5 peers got BBB message")
		return
	}

	//add the validator on node 4
	err = nodes[4].GetTopic("test").RegisterValidator(v)
	assert.Nil(t, err)

	fmt.Println("Waiting for cooldown period (timecache should empty map)")
	time.Sleep(p2p.DurTimeCache + time.Millisecond*100)

	//send AAA, wait, check that 2 peers got the message
	fmt.Println("Resending AAA...")
	wg.Add(2)
	go waitForWaitGroup(&wg, chanDone)

	err = nodes[0].GetTopic("test").Broadcast(testNetStringCreator{Data: "AAA"})
	assert.Nil(t, err)

	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "not all 2 peers got AAA message")
	}
}

func TestNetMessenger_BroadcastToGossipSubShouldWork(t *testing.T) {
	var nodes []p2p.Messenger

	defer func() {
		closeAllNodes(nodes)
	}()

	nodes, err := createTestNetwork(t)
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}

	chanDone := make(chan bool, 0)

	wg := sync.WaitGroup{}
	doWaitGroup := false
	counter := int32(0)

	recv1 := func(name string, data interface{}, msgInfo *p2p.MessageInfo) {
		if doWaitGroup {
			wg.Done()
		}

		atomic.AddInt32(&counter, 1)
	}

	//print connected and create topics
	for i := 0; i < 5; i++ {
		node := nodes[i]
		node.PrintConnected()

		err := node.AddTopic(p2p.NewTopic("test", &testNetStringCreator{}, &mock.MarshalizerMock{}))
		assert.Nil(t, err)
		node.GetTopic("test").AddDataReceived(recv1)
	}

	time.Sleep(pubsubAnnounceDuration)

	//send a piggyback message, wait 1 sec
	fmt.Println("Broadcasting piggyback message...")
	err = nodes[0].GetTopic("test").Broadcast(testNetStringCreator{Data: "piggyback"})
	assert.Nil(t, err)
	time.Sleep(time.Second)
	fmt.Printf("%d peers got the message!\n", atomic.LoadInt32(&counter))

	atomic.StoreInt32(&counter, 0)

	fmt.Println("Broadcasting AAA...")
	doWaitGroup = true
	wg.Add(5)
	go waitForWaitGroup(&wg, chanDone)
	err = nodes[0].GetTopic("test").Broadcast(testNetStringCreator{Data: "AAA"})
	assert.Nil(t, err)

	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "not all 5 peers got AAA message")
	}
}

func TestNetMessenger_BroadcastToUnknownSubShouldErr(t *testing.T) {
	fmt.Println()

	_, err := createNetMessengerPubSub(t, startingPort, 10, 500)
	assert.NotNil(t, err)
}

func TestNetMessenger_RequestResolveTestCfg1ShouldWork(t *testing.T) {
	var nodes []p2p.Messenger

	defer func() {
		closeAllNodes(nodes)
	}()

	nodes, err := createTestNetwork(t)
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}

	chanDone := make(chan bool, 0)

	recv := func(name string, data interface{}, msgInfo *p2p.MessageInfo) {
		if data.(*testNetStringCreator).Data == "Real object1" {
			chanDone <- true
		}

		fmt.Printf("Received: %v\n", data.(*testNetStringCreator).Data)
	}

	//print connected and create topics
	for i := 0; i < 5; i++ {
		node := nodes[i]
		node.PrintConnected()

		err := node.AddTopic(p2p.NewTopic("test", &testNetStringCreator{}, &mock.MarshalizerMock{}))
		assert.Nil(t, err)
	}

	time.Sleep(pubsubAnnounceDuration)

	//to simplify, only node 0 should have a recv event handler
	nodes[0].GetTopic("test").AddDataReceived(recv)

	//setup a resolver func for node 3
	nodes[3].GetTopic("test").ResolveRequest = func(hash []byte) []byte {
		if bytes.Equal(hash, []byte("A000")) {
			marshalizer := &mock.MarshalizerMock{}
			buff, _ := marshalizer.Marshal(&testNetStringCreator{Data: "Real object1"})
			return buff
		}

		return nil
	}

	//node0 requests an unavailable data
	err = nodes[0].GetTopic("test").SendRequest([]byte("B000"))
	assert.Nil(t, err)
	fmt.Println("Sent request B000")
	select {
	case <-chanDone:
		assert.Fail(t, "Should have not sent object")
	case <-time.After(testNetMessengerWaitResponseUnreceivedMsg):
	}

	//node0 requests an available data on node 3
	err = nodes[0].GetTopic("test").SendRequest([]byte("A000"))
	assert.Nil(t, err)
	fmt.Println("Sent request A000")

	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "Should have sent object")
		return
	}
}

func TestNetMessenger_RequestResolveTestCfg2ShouldWork(t *testing.T) {
	var nodes []p2p.Messenger

	defer func() {
		closeAllNodes(nodes)
	}()

	nodes, err := createTestNetwork(t)
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}

	chanDone := make(chan bool, 0)

	recv := func(name string, data interface{}, msgInfo *p2p.MessageInfo) {
		if data.(*testNetStringCreator).Data == "Real object1" {
			chanDone <- true
		}

		fmt.Printf("Received: %v from %v\n", data.(*testNetStringCreator).Data, msgInfo.Peer)
	}

	//print connected and create topics
	for i := 0; i < 5; i++ {
		node := nodes[i]
		node.PrintConnected()

		err := node.AddTopic(p2p.NewTopic("test", &testNetStringCreator{}, &mock.MarshalizerMock{}))
		assert.Nil(t, err)
	}

	time.Sleep(pubsubAnnounceDuration)

	//to simplify, only node 1 should have a recv event handler
	nodes[1].GetTopic("test").AddDataReceived(recv)

	//resolver func for node 0 and 2
	resolverOK := func(hash []byte) []byte {
		if bytes.Equal(hash, []byte("A000")) {
			marshalizer := &mock.MarshalizerMock{}
			buff, _ := marshalizer.Marshal(&testNetStringCreator{Data: "Real object1"})
			return buff
		}

		return nil
	}

	//resolver func for other nodes
	resolverNOK := func(hash []byte) []byte {
		panic("Should have not reached this point")

		return nil
	}

	nodes[0].GetTopic("test").ResolveRequest = resolverOK
	nodes[2].GetTopic("test").ResolveRequest = resolverOK

	nodes[3].GetTopic("test").ResolveRequest = resolverNOK
	nodes[4].GetTopic("test").ResolveRequest = resolverNOK

	//node1 requests an available data
	err = nodes[1].GetTopic("test").SendRequest([]byte("A000"))
	assert.Nil(t, err)
	fmt.Println("Sent request A000")

	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "Should have sent object")
	}

}

func TestNetMessenger_RequestResolveTestSelfShouldWork(t *testing.T) {
	var nodes []p2p.Messenger

	defer func() {
		closeAllNodes(nodes)
	}()

	nodes, err := createTestNetwork(t)
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}

	chanDone := make(chan bool, 0)

	recv := func(name string, data interface{}, msgInfo *p2p.MessageInfo) {
		if data.(*testNetStringCreator).Data == "Real object1" {
			chanDone <- true
		}

		fmt.Printf("Received: %v from %v\n", data.(*testNetStringCreator).Data, msgInfo.Peer)
	}

	//print connected and create topics
	for i := 0; i < 5; i++ {
		node := nodes[i]
		node.PrintConnected()

		err := node.AddTopic(p2p.NewTopic("test", &testNetStringCreator{}, &mock.MarshalizerMock{}))
		assert.Nil(t, err)
	}

	time.Sleep(pubsubAnnounceDuration)

	//to simplify, only node 1 should have a recv event handler
	nodes[1].GetTopic("test").AddDataReceived(recv)

	//resolver func for node 1
	resolverOK := func(hash []byte) []byte {
		if bytes.Equal(hash, []byte("A000")) {
			marshalizer := &mock.MarshalizerMock{}
			buff, _ := marshalizer.Marshal(&testNetStringCreator{Data: "Real object1"})
			return buff
		}

		return nil
	}

	//resolver func for other nodes
	resolverNOK := func(hash []byte) []byte {
		panic("Should have not reached this point")

		return nil
	}

	nodes[1].GetTopic("test").ResolveRequest = resolverOK

	nodes[0].GetTopic("test").ResolveRequest = resolverNOK
	nodes[2].GetTopic("test").ResolveRequest = resolverNOK
	nodes[3].GetTopic("test").ResolveRequest = resolverNOK
	nodes[4].GetTopic("test").ResolveRequest = resolverNOK

	//node1 requests an available data
	err = nodes[1].GetTopic("test").SendRequest([]byte("A000"))
	assert.Nil(t, err)
	fmt.Println("Sent request A000")

	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "Should have self-sent object")
	}

}

func TestNetMessenger_RequestResolveResendingShouldWork(t *testing.T) {
	var nodes []p2p.Messenger

	defer func() {
		closeAllNodes(nodes)
	}()

	nodes, err := createTestNetwork(t)
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}

	chanDone := make(chan bool, 0)

	counter := int32(0)

	recv := func(name string, data interface{}, msgInfo *p2p.MessageInfo) {
		atomic.AddInt32(&counter, 1)

		fmt.Printf("Received: %v from %v\n", data.(*testNetStringCreator).Data, msgInfo.Peer)
	}

	//print connected and create topics
	for i := 0; i < 5; i++ {
		node := nodes[i]
		node.PrintConnected()

		err := node.AddTopic(p2p.NewTopic("test", &testNetStringCreator{}, &mock.MarshalizerMock{}))
		assert.Nil(t, err)
	}

	time.Sleep(pubsubAnnounceDuration)

	//to simplify, only node 1 should have a recv event handler
	nodes[1].GetTopic("test").AddDataReceived(recv)

	//resolver func for node 0 and 2
	resolverOK := func(hash []byte) []byte {
		if bytes.Equal(hash, []byte("A000")) {
			marshalizer := &mock.MarshalizerMock{}
			buff, _ := marshalizer.Marshal(&testNetStringCreator{Data: "Real object0"})
			return buff
		}

		return nil
	}

	//resolver func for other nodes
	resolverNOK := func(hash []byte) []byte {
		panic("Should have not reached this point")

		return nil
	}

	nodes[0].GetTopic("test").ResolveRequest = resolverOK
	nodes[2].GetTopic("test").ResolveRequest = resolverOK

	nodes[3].GetTopic("test").ResolveRequest = resolverNOK
	nodes[4].GetTopic("test").ResolveRequest = resolverNOK

	//node1 requests an available data
	go waitForValue(&counter, 1, chanDone)
	err = nodes[1].GetTopic("test").SendRequest([]byte("A000"))
	assert.Nil(t, err)
	fmt.Println("Sent request A000")
	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "Should have received 1 object")
		return
	}

	//resending request. This should be filtered out
	atomic.StoreInt32(&counter, 0)
	go waitForValue(&counter, 1, chanDone)
	err = nodes[1].GetTopic("test").SendRequest([]byte("A000"))
	assert.Nil(t, err)
	fmt.Println("Re-sent request A000")
	select {
	case <-chanDone:
		assert.Fail(t, "Should have not received")
		return
	case <-time.After(testNetMessengerWaitResponseUnreceivedMsg):
	}

	fmt.Println("delaying as to clear timecache buffer")
	time.Sleep(p2p.DurTimeCache + time.Millisecond*100)

	//resending
	atomic.StoreInt32(&counter, 0)
	go waitForValue(&counter, 1, chanDone)
	err = nodes[1].GetTopic("test").SendRequest([]byte("A000"))
	assert.Nil(t, err)
	fmt.Println("Re-sent request A000")
	select {
	case <-chanDone:
	case <-time.After(testNetMessengerMaxWaitResponse):
		assert.Fail(t, "Should have received 2 objects")
		return
	}

}
