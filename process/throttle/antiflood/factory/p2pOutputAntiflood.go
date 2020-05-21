package factory

import (
	"math"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/throttle/antiflood"
	"github.com/ElrondNetwork/elrond-go/process/throttle/antiflood/floodPreventers"
	storageFactory "github.com/ElrondNetwork/elrond-go/storage/factory"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
)

// NewP2POutputAntiFlood will return an instance of an output antiflood component based on the config
func NewP2POutputAntiFlood(mainConfig config.Config) (process.P2PAntifloodHandler, process.FloodPreventer, error) {
	if mainConfig.Antiflood.Enabled {
		return initP2POutputAntiFlood(mainConfig)
	}

	return &disabledAntiFlood{}, &disabledFloodPreventer{}, nil
}

func initP2POutputAntiFlood(mainConfig config.Config) (process.P2PAntifloodHandler, process.FloodPreventer, error) {
	cacheConfig := storageFactory.GetCacherFromConfig(mainConfig.Antiflood.Cache)
	antifloodCache, err := storageUnit.NewCache(cacheConfig.Type, cacheConfig.Size, cacheConfig.Shards)
	if err != nil {
		return nil, nil, err
	}

	peerMaxMessagesPerSecond := mainConfig.Antiflood.PeerMaxOutput.MessagesPerSecond
	peerMaxTotalSizePerSecond := mainConfig.Antiflood.PeerMaxOutput.TotalSizePerSecond
	floodPreventer, err := floodPreventers.NewQuotaFloodPreventer(
		antifloodCache,
		make([]floodPreventers.QuotaStatusHandler, 0),
		peerMaxMessagesPerSecond,
		peerMaxTotalSizePerSecond,
		math.MaxUint32,
		math.MaxUint64,
	)
	if err != nil {
		return nil, nil, err
	}

	topicFloodPreventer := floodPreventers.NewNilTopicFloodPreventer()

	p2pAntiFlood, err := antiflood.NewP2PAntiflood(floodPreventer, topicFloodPreventer)
	return p2pAntiFlood, floodPreventer, nil
}
