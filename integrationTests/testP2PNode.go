package integrationTests

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/crypto/signing"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/kyber"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/kyber/singlesig"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/display"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/sharding/networksharding"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
)

// ShardTopic is the topic string generator for sharded topics
// Will generate topics in the following pattern: shard_0, shard_0_1, shard_0_META, shard_1 and so on
const ShardTopic = "shard"

// GlobalTopic is a global testing that all nodes will bind an interceptor
const GlobalTopic = "global"

// TestP2PNode represents a container type of class used in integration tests
// with all its fields exported, used mainly on P2P tests
type TestP2PNode struct {
	Messenger              p2p.Messenger
	NodesCoordinator       sharding.NodesCoordinator
	ShardCoordinator       sharding.Coordinator
	NetworkShardingUpdater NetworkShardingUpdater
	Node                   *node.Node
	NodeKeys               TestKeyPair
	SingleSigner           crypto.SingleSigner
	KeyGen                 crypto.KeyGenerator
	Storage                dataRetriever.StorageService
	Interceptor            *CountInterceptor
}

// NewTestP2PNode returns a new NewTestP2PNode instance
func NewTestP2PNode(
	maxShards uint32,
	nodeShardId uint32,
	p2pConfig config.P2PConfig,
	coordinator sharding.NodesCoordinator,
	keys TestKeyPair,
) *TestP2PNode {

	tP2pNode := &TestP2PNode{
		NodesCoordinator: coordinator,
		NodeKeys:         keys,
		Interceptor:      NewCountInterceptor(),
	}

	shardCoordinator, err := sharding.NewMultiShardCoordinator(maxShards, nodeShardId)
	if err != nil {
		fmt.Printf("Error creating shard coordinator: %s\n", err.Error())
	}

	tP2pNode.ShardCoordinator = shardCoordinator

	pidPk, _ := storageUnit.NewCache(storageUnit.LRUCache, 1000, 0)
	pkShardId, _ := storageUnit.NewCache(storageUnit.LRUCache, 1000, 0)
	pidShardId, _ := storageUnit.NewCache(storageUnit.LRUCache, 1000, 0)
	tP2pNode.NetworkShardingUpdater, err = networksharding.NewPeerShardMapper(
		pidPk,
		pkShardId,
		pidShardId,
		coordinator,
		&mock.EpochStartTriggerStub{},
	)
	if err != nil {
		fmt.Printf("Error creating NewPeerShardMapper: %s\n", err.Error())
	}

	tP2pNode.Messenger = CreateMessengerFromConfig(context.Background(), p2pConfig)
	localId := tP2pNode.Messenger.ID()
	tP2pNode.NetworkShardingUpdater.UpdatePeerIdShardId(localId, shardCoordinator.SelfId())

	err = tP2pNode.Messenger.SetPeerShardResolver(tP2pNode.NetworkShardingUpdater)
	if err != nil {
		fmt.Printf("Error setting messenger.SetPeerShardResolver: %s\n", err.Error())
	}

	tP2pNode.initStorage()
	tP2pNode.initCrypto()
	tP2pNode.initNode()

	return tP2pNode
}

func (tP2pNode *TestP2PNode) initStorage() {
	if tP2pNode.ShardCoordinator.SelfId() == core.MetachainShardId {
		tP2pNode.Storage = CreateMetaStore(tP2pNode.ShardCoordinator)
	} else {
		tP2pNode.Storage = CreateShardStore(tP2pNode.ShardCoordinator.NumberOfShards())
	}
}

func (tP2pNode *TestP2PNode) initCrypto() {
	tP2pNode.SingleSigner = &singlesig.BlsSingleSigner{}
	suite := kyber.NewSuitePairingBn256()
	tP2pNode.KeyGen = signing.NewKeyGenerator(suite)
}

func (tP2pNode *TestP2PNode) initNode() {
	var err error

	tP2pNode.Node, err = node.NewNode(
		node.WithMessenger(tP2pNode.Messenger),
		node.WithInternalMarshalizer(TestMarshalizer, 100),
		node.WithHasher(TestHasher),
		node.WithKeyGen(tP2pNode.KeyGen),
		node.WithShardCoordinator(tP2pNode.ShardCoordinator),
		node.WithNodesCoordinator(tP2pNode.NodesCoordinator),
		node.WithSingleSigner(tP2pNode.SingleSigner),
		node.WithPrivKey(tP2pNode.NodeKeys.Sk),
		node.WithPubKey(tP2pNode.NodeKeys.Pk),
		node.WithNetworkShardingCollector(tP2pNode.NetworkShardingUpdater),
		node.WithDataStore(tP2pNode.Storage),
	)
	if err != nil {
		fmt.Printf("Error creating node: %s\n", err.Error())
	}

	hbConfig := config.HeartbeatConfig{
		Enabled:                             true,
		MinTimeToWaitBetweenBroadcastsInSec: 4,
		MaxTimeToWaitBetweenBroadcastsInSec: 6,
		DurationInSecToConsiderUnresponsive: 60,
	}
	err = tP2pNode.Node.StartHeartbeat(hbConfig, "test", "")
	if err != nil {
		fmt.Printf("Error starting heartbeat: %s\n", err.Error())
	}
}

// CreateTestInterceptors creates test interceptors that count the number of received messages
func (tP2pNode *TestP2PNode) CreateTestInterceptors() {
	tP2pNode.RegisterTopicValidator(GlobalTopic, tP2pNode.Interceptor)

	metaIdentifier := ShardTopic + tP2pNode.ShardCoordinator.CommunicationIdentifier(core.MetachainShardId)
	tP2pNode.RegisterTopicValidator(metaIdentifier, tP2pNode.Interceptor)

	for i := uint32(0); i < tP2pNode.ShardCoordinator.NumberOfShards(); i++ {
		identifier := ShardTopic + tP2pNode.ShardCoordinator.CommunicationIdentifier(i)
		tP2pNode.RegisterTopicValidator(identifier, tP2pNode.Interceptor)
	}
}

// CountGlobalMessages returns the messages count on the global topic
func (tP2pNode *TestP2PNode) CountGlobalMessages() int {
	return tP2pNode.Interceptor.MessageCount(GlobalTopic)
}

// CountIntraShardMessages returns the messages count on the intra-shard topic
func (tP2pNode *TestP2PNode) CountIntraShardMessages() int {
	identifier := ShardTopic + tP2pNode.ShardCoordinator.CommunicationIdentifier(tP2pNode.ShardCoordinator.SelfId())
	return tP2pNode.Interceptor.MessageCount(identifier)
}

// CountCrossShardMessages returns the messages count on the cross-shard topics
func (tP2pNode *TestP2PNode) CountCrossShardMessages() int {
	messages := 0

	if tP2pNode.ShardCoordinator.SelfId() != core.MetachainShardId {
		metaIdentifier := ShardTopic + tP2pNode.ShardCoordinator.CommunicationIdentifier(core.MetachainShardId)
		messages += tP2pNode.Interceptor.MessageCount(metaIdentifier)
	}

	for i := uint32(0); i < tP2pNode.ShardCoordinator.NumberOfShards(); i++ {
		if i == tP2pNode.ShardCoordinator.SelfId() {
			continue
		}

		metaIdentifier := ShardTopic + tP2pNode.ShardCoordinator.CommunicationIdentifier(i)
		messages += tP2pNode.Interceptor.MessageCount(metaIdentifier)
	}

	return messages
}

// RegisterTopicValidator registers a message processor instance on the provided topic
func (tP2pNode *TestP2PNode) RegisterTopicValidator(topic string, processor p2p.MessageProcessor) {
	err := tP2pNode.Messenger.CreateTopic(topic, true)
	if err != nil {
		fmt.Printf("error while creating topic %s: %s\n", topic, err.Error())
		return
	}

	err = tP2pNode.Messenger.RegisterMessageProcessor(topic, processor)
	if err != nil {
		fmt.Printf("error while registering topic validator %s: %s\n", topic, err.Error())
		return
	}
}

// CreateNodesWithTestP2PNodes returns a map with nodes per shard each using a real nodes coordinator
// and TestP2PNodes
func CreateNodesWithTestP2PNodes(
	nodesPerShard int,
	nbMetaNodes int,
	nbShards int,
	shardConsensusGroupSize int,
	metaConsensusGroupSize int,
	p2pConfig config.P2PConfig,
) map[uint32][]*TestP2PNode {

	cp := CreateCryptoParams(nodesPerShard, nbMetaNodes, uint32(nbShards))
	pubKeys := PubKeysMapFromKeysMap(cp.Keys)
	validatorsMap := GenValidatorsFromPubKeys(pubKeys, uint32(nbShards))
	nodesMap := make(map[uint32][]*TestP2PNode)
	cacherCfg := storageUnit.CacheConfig{Size: 10000, Type: storageUnit.LRUCache, Shards: 1}
	cache, _ := storageUnit.NewCache(cacherCfg.Type, cacherCfg.Size, cacherCfg.Shards)
	for shardId, validatorList := range validatorsMap {
		argumentsNodesCoordinator := sharding.ArgNodesCoordinator{
			ShardConsensusGroupSize: shardConsensusGroupSize,
			MetaConsensusGroupSize:  metaConsensusGroupSize,
			Hasher:                  TestHasher,
			ShardId:                 shardId,
			NbShards:                uint32(nbShards),
			EligibleNodes:           validatorsMap,
			SelfPublicKey:           []byte(strconv.Itoa(int(shardId))),
			ConsensusGroupCache:     cache,
			Shuffler:                &mock.NodeShufflerMock{},
			BootStorer:              CreateMemUnit(),
			WaitingNodes:            make(map[uint32][]sharding.Validator),
			Epoch:                   0,
			EpochStartSubscriber:    &mock.EpochStartNotifierStub{},
		}
		nodesCoordinator, err := sharding.NewIndexHashedNodesCoordinator(argumentsNodesCoordinator)

		if err != nil {
			fmt.Println("Error creating node coordinator " + err.Error())
		}

		nodesList := make([]*TestP2PNode, len(validatorList))
		for i := range validatorList {
			kp := cp.Keys[shardId][i]
			nodesList[i] = NewTestP2PNode(
				uint32(nbShards),
				shardId,
				p2pConfig,
				nodesCoordinator,
				*kp,
			)
		}
		nodesMap[shardId] = nodesList
	}

	return nodesMap
}

// MakeDisplayTableForP2PNodes will output a string containing counters for received messages for all provided test nodes
func MakeDisplayTableForP2PNodes(nodes map[uint32][]*TestP2PNode) string {
	header := []string{"pk", "pid", "shard ID", "messages global", "messages intra", "messages cross", "conns Total/Intra/Cross/Unk"}
	dataLines := make([]*display.LineData, 0)

	for shardId, nodesList := range nodes {
		for _, n := range nodesList {
			buffPk, _ := n.NodeKeys.Pk.ToByteArray()

			peerInfo := n.Messenger.GetConnectedPeersInfo()

			pid := n.Messenger.ID().Pretty()
			lineData := display.NewLineData(
				false,
				[]string{
					core.GetTrimmedPk(hex.EncodeToString(buffPk)),
					pid[len(pid)-6:],
					fmt.Sprintf("%d", shardId),
					fmt.Sprintf("%d", n.CountGlobalMessages()),
					fmt.Sprintf("%d", n.CountIntraShardMessages()),
					fmt.Sprintf("%d", n.CountCrossShardMessages()),
					fmt.Sprintf("%d/%d/%d/%d",
						len(n.Messenger.ConnectedPeers()),
						len(peerInfo.IntraShardPeers),
						len(peerInfo.CrossShardPeers),
						len(peerInfo.UnknownPeers),
					),
				},
			)

			dataLines = append(dataLines, lineData)
		}
	}
	table, _ := display.CreateTableString(header, dataLines)

	return table
}
