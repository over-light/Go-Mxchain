package monitor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/heartbeat"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

const topicSeparator = "_"

// ArgsCrossShardPeerTopicNotifier represents the arguments for the cross shard peer topic notifier
type ArgsCrossShardPeerTopicNotifier struct {
	ShardCoordinator sharding.Coordinator
	PeerShardMapper  heartbeat.PeerShardMapper
}

type crossShardPeerTopicNotifier struct {
	shardCoordinator sharding.Coordinator
	peerShardMapper  heartbeat.PeerShardMapper
}

// NewCrossShardPeerTopicNotifier create a new cross shard peer topic notifier instance
func NewCrossShardPeerTopicNotifier(args ArgsCrossShardPeerTopicNotifier) (*crossShardPeerTopicNotifier, error) {
	err := checkArgsCrossShardPeerTopicNotifier(args)
	if err != nil {
		return nil, err
	}

	notifier := &crossShardPeerTopicNotifier{
		shardCoordinator: args.ShardCoordinator,
		peerShardMapper:  args.PeerShardMapper,
	}

	return notifier, nil
}

func checkArgsCrossShardPeerTopicNotifier(args ArgsCrossShardPeerTopicNotifier) error {
	if check.IfNil(args.PeerShardMapper) {
		return heartbeat.ErrNilPeerShardMapper
	}
	if check.IfNil(args.ShardCoordinator) {
		return heartbeat.ErrNilShardCoordinator
	}

	return nil
}

// NewPeerFound is called whenever a new peer was found
func (notifier *crossShardPeerTopicNotifier) NewPeerFound(pid core.PeerID, topic string) {
	splt := strings.Split(topic, topicSeparator)
	if len(splt) != 3 {
		// not a cross shard peer or the topic is global
		return
	}

	shardID1, err := notifier.getShardID(splt[1])
	if err != nil {
		return
	}

	shardID2, err := notifier.getShardID(splt[2])
	if err != nil {
		return
	}
	if shardID1 == shardID2 {
		return
	}
	if shardID1 == notifier.shardCoordinator.SelfId() {
		log.Trace("crossShardPeerTopicNotifier.NewPeerFound found a cross shard peer",
			"topic", topic,
			"pid", pid.Pretty(),
			"shard", shardID2)
		notifier.peerShardMapper.PutPeerIdShardId(pid, shardID2)
	}
	if shardID2 == notifier.shardCoordinator.SelfId() {
		log.Trace("crossShardPeerTopicNotifier.NewPeerFound found a cross shard peer",
			"topic", topic,
			"pid", pid.Pretty(),
			"shard", shardID1)
		notifier.peerShardMapper.PutPeerIdShardId(pid, shardID1)
	}
}

func (notifier *crossShardPeerTopicNotifier) getShardID(data string) (uint32, error) {
	if data == common.MetachainTopicIdentifier {
		return common.MetachainShardId, nil
	}
	val, err := strconv.Atoi(data)
	if err != nil {
		return 0, err
	}
	if uint32(val) >= notifier.shardCoordinator.NumberOfShards() || val < 0 {
		return 0, fmt.Errorf("negative value in crossShardPeerTopicNotifier.getShardID %d", val)
	}

	return uint32(val), nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (notifier *crossShardPeerTopicNotifier) IsInterfaceNil() bool {
	return notifier == nil
}
