package roundActivation

import (
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type roundInfo struct {
	round uint64
	shard uint32
}

type roundActivation struct {
	roundHandler     process.RoundHandler
	shardCoordinator sharding.Coordinator
	configMap        map[string]roundInfo
}

func NewRoundActivation(roundHandler process.RoundHandler, shardCoordinator sharding.Coordinator, config config.RoundConfig) (process.RoundActivationHandler, error) {
	if check.IfNil(roundHandler) {
		return nil, process.ErrNilRoundHandler
	}
	if check.IfNil(shardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}

	configMap := make(map[string]roundInfo, len(config.EnableRoundsByName))
	for _, roundConfig := range config.EnableRoundsByName {
		if _, exists := configMap[roundConfig.Name]; exists {
			return nil, process.ErrDuplicateRoundActivationName
		}

		configMap[roundConfig.Name] = roundInfo{
			round: roundConfig.Round,
			shard: roundConfig.Shard,
		}
	}

	return &roundActivation{
		roundHandler:     roundHandler,
		shardCoordinator: shardCoordinator,
		configMap:        configMap,
	}, nil
}

func (ra *roundActivation) IsEnabled(name string, round uint64) bool {
	if info, exists := ra.configMap[name]; exists {
		return info.round == round && info.shard == ra.shardCoordinator.SelfId()
	}

	return false
}

func (ra *roundActivation) IsEnabledInCurrentRound(name string) bool {
	if info, exists := ra.configMap[name]; exists {
		return info.round == uint64(ra.roundHandler.Index()) && info.shard == ra.shardCoordinator.SelfId()
	}

	return false
}

func (ra *roundActivation) IsInterfaceNil() bool {
	return ra == nil
}
