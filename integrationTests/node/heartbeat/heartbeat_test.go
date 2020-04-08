package heartbeat

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/crypto/signing"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/mcl"
	mclsig "github.com/ElrondNetwork/elrond-go/crypto/signing/mcl/singlesig"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/node/heartbeat"
	"github.com/ElrondNetwork/elrond-go/node/mock"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/stretchr/testify/assert"
)

var stepDelay = time.Second
var log = logger.GetOrCreate("integrationtests/node")

// TestHeartbeatMonitorWillUpdateAnInactivePeer test what happen if a peer out of 2 stops being responsive on heartbeat status
// The active monitor should change it's active flag to false when a new heartbeat message has arrived.
func TestHeartbeatMonitorWillUpdateAnInactivePeer(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()
	advertiserAddr := integrationTests.GetConnectableAddress(advertiser)
	maxUnresposiveTime := time.Second * 10

	monitor := createMonitor(maxUnresposiveTime)
	nodes, senders, pks := prepareNodes(advertiserAddr, monitor, 3, "nodeName")

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Close()
		}
	}()

	fmt.Println("Delaying for node bootstrap and topic announcement...")
	time.Sleep(time.Second * 5)

	fmt.Println("Sending first messages from both public keys...")
	err := senders[0].SendHeartbeat()
	log.LogIfError(err)

	err = senders[1].SendHeartbeat()
	log.LogIfError(err)

	time.Sleep(stepDelay)

	fmt.Println("Checking both public keys are active...")
	checkReceivedMessages(t, monitor, pks, []int{0, 1})

	fmt.Println("Waiting for max unresponsive time...")
	time.Sleep(maxUnresposiveTime)

	fmt.Println("Only first pk will send another message...")
	_ = senders[0].SendHeartbeat()

	time.Sleep(stepDelay)

	fmt.Println("Checking only first pk is active...")
	checkReceivedMessages(t, monitor, pks, []int{0})
}

func TestHeartbeatMonitorWillNotUpdateTooLongHeartbeatMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	advertiser := integrationTests.CreateMessengerWithKadDht(context.Background(), "")
	_ = advertiser.Bootstrap()
	advertiserAddr := integrationTests.GetConnectableAddress(advertiser)
	maxUnresposiveTime := time.Second * 10

	length := 129
	buff := make([]byte, length)

	for i := 0; i < length; i++ {
		buff[i] = byte(97)
	}
	bigNodeName := string(buff)

	monitor := createMonitor(maxUnresposiveTime)
	nodes, senders, pks := prepareNodes(advertiserAddr, monitor, 3, bigNodeName)

	defer func() {
		_ = advertiser.Close()
		for _, n := range nodes {
			_ = n.Close()
		}
	}()

	fmt.Println("Delaying for node bootstrap and topic announcement...")
	time.Sleep(time.Second * 5)

	fmt.Println("Sending first messages with long name...")
	_ = senders[1].SendHeartbeat()

	time.Sleep(stepDelay)

	secondPK := pks[1]

	pkHeartBeats := monitor.GetHeartbeats()

	assert.True(t, isPkActive(pkHeartBeats, secondPK))
	expectedLen := 128
	assert.True(t, isMessageCorrectLen(pkHeartBeats, secondPK, expectedLen))
}

func prepareNodes(
	advertiserAddr string,
	monitor *heartbeat.Monitor,
	interactingNodes int,
	defaultNodeName string,
) ([]p2p.Messenger, []*heartbeat.Sender, []crypto.PublicKey) {

	senderIdxs := []int{0, 1}
	nodes := make([]p2p.Messenger, interactingNodes)
	topicHeartbeat := "topic"
	senders := make([]*heartbeat.Sender, 0)
	pks := make([]crypto.PublicKey, 0)

	for i := 0; i < interactingNodes; i++ {
		nodes[i] = integrationTests.CreateMessengerWithKadDht(context.Background(), advertiserAddr)
		_ = nodes[i].CreateTopic(topicHeartbeat, true)

		isSender := integrationTests.IsIntInSlice(i, senderIdxs)
		if isSender {
			sender, pk := createSenderWithName(nodes[i], topicHeartbeat, defaultNodeName)
			senders = append(senders, sender)
			pks = append(pks, pk)
		} else {
			_ = nodes[i].RegisterMessageProcessor(topicHeartbeat, monitor)
		}

		_ = nodes[i].Bootstrap()
	}

	return nodes, senders, pks
}

func checkReceivedMessages(t *testing.T, monitor *heartbeat.Monitor, pks []crypto.PublicKey, activeIdxs []int) {
	pkHeartBeats := monitor.GetHeartbeats()

	extraPkInMonitor := 1
	assert.Equal(t, len(pks), len(pkHeartBeats)-extraPkInMonitor)

	for idx, pk := range pks {
		pkShouldBeActive := integrationTests.IsIntInSlice(idx, activeIdxs)
		assert.Equal(t, pkShouldBeActive, isPkActive(pkHeartBeats, pk))
		assert.True(t, isMessageReceived(pkHeartBeats, pk))
	}
}

func isMessageReceived(heartbeats []heartbeat.PubKeyHeartbeat, pk crypto.PublicKey) bool {
	pkBytes, _ := pk.ToByteArray()

	for _, hb := range heartbeats {
		isPkMatching := hb.PublicKey == integrationTests.TestValidatorPubkeyConverter.Encode(pkBytes)
		if isPkMatching {
			return true
		}
	}

	return false
}

func isPkActive(heartbeats []heartbeat.PubKeyHeartbeat, pk crypto.PublicKey) bool {
	pkBytes, _ := pk.ToByteArray()

	for _, hb := range heartbeats {
		isPkMatchingAndActve := hb.PublicKey == integrationTests.TestValidatorPubkeyConverter.Encode(pkBytes) && hb.IsActive
		if isPkMatchingAndActve {
			return true
		}
	}

	return false
}

func isMessageCorrectLen(heartbeats []heartbeat.PubKeyHeartbeat, pk crypto.PublicKey, expectedLen int) bool {
	pkBytes, _ := pk.ToByteArray()

	for _, hb := range heartbeats {
		isPkMatching := hb.PublicKey == integrationTests.TestValidatorPubkeyConverter.Encode(pkBytes)
		if isPkMatching {
			return len(hb.NodeDisplayName) == expectedLen
		}
	}

	return false
}

func createSenderWithName(messenger p2p.Messenger, topic string, nodeName string) (*heartbeat.Sender, crypto.PublicKey) {
	suite := mcl.NewSuiteBLS12()
	signer := &mclsig.BlsSingleSigner{}
	keyGen := signing.NewKeyGenerator(suite)
	sk, pk := keyGen.GeneratePair()
	version := "v01"
	sender, _ := heartbeat.NewSender(
		messenger,
		signer,
		sk,
		integrationTests.TestMarshalizer,
		topic,
		&sharding.OneShardCoordinator{},
		&mock.PeerTypeProviderStub{},
		&mock.AppStatusHandlerStub{},
		version,
		nodeName,
	)
	return sender, pk
}

func createMonitor(maxDurationPeerUnresponsive time.Duration) *heartbeat.Monitor {
	suite := mcl.NewSuiteBLS12()
	singlesigner := &mclsig.BlsSingleSigner{}
	keyGen := signing.NewKeyGenerator(suite)
	marshalizer := &marshal.GogoProtoMarshalizer{}

	mp, _ := heartbeat.NewMessageProcessor(
		singlesigner,
		keyGen,
		marshalizer,
		&mock.NetworkShardingCollectorStub{
			UpdatePeerIdPublicKeyCalled: func(pid p2p.PeerID, pk []byte) {},
			UpdatePeerIdShardIdCalled:   func(pid p2p.PeerID, shardId uint32) {},
		})

	monitor, _ := heartbeat.NewMonitor(
		integrationTests.TestMarshalizer,
		maxDurationPeerUnresponsive,
		map[uint32][]string{0: {""}},
		time.Now(),
		mp,
		&mock.HeartbeatStorerStub{
			UpdateGenesisTimeCalled: func(genesisTime time.Time) error {
				return nil
			},
			LoadHbmiDTOCalled: func(pubKey string) (*heartbeat.HeartbeatDTO, error) {
				return nil, errors.New("not found")
			},
			LoadKeysCalled: func() ([][]byte, error) {
				return nil, nil
			},
			SavePubkeyDataCalled: func(pubkey []byte, heartbeat *heartbeat.HeartbeatDTO) error {
				return nil
			},
			SaveKeysCalled: func(peersSlice [][]byte) error {
				return nil
			},
		},
		&mock.PeerTypeProviderStub{},
		&heartbeat.RealTimer{},
		&mock.P2PAntifloodHandlerStub{
			CanProcessMessageCalled: func(message p2p.MessageP2P, fromConnectedPeer p2p.PeerID) error {
				return nil
			},
		},
		integrationTests.TestValidatorPubkeyConverter,
	)

	return monitor
}
