package factory

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/p2p/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewP2PAntiFloodAndBlackList_NilStatusHandlerShouldErr(t *testing.T) {
	t.Parallel()

	cfg := config.Config{}
	components, err := NewP2PAntiFloodAndBlackList(cfg, nil)
	assert.Nil(t, components)
	assert.Equal(t, p2p.ErrNilStatusHandler, err)
}

func TestNewP2PAntiFloodAndBlackList_ShouldWorkAndReturnDisabledImplementations(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Antiflood: config.AntifloodConfig{
			Enabled: false,
		},
	}
	ash := &mock.AppStatusHandlerMock{}
	components, err := NewP2PAntiFloodAndBlackList(cfg, ash)
	assert.NotNil(t, components)
	assert.Nil(t, err)

	_, ok1 := components.AntiFloodHandler.(*disabledAntiFlood)
	_, ok2 := components.BlacklistHandler.(*disabledBlacklistHandler)
	assert.True(t, ok1)
	assert.True(t, ok2)
}

func TestNewP2PAntiFloodAndBlackList_ShouldWorkAndReturnOkImplementations(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Antiflood: config.AntifloodConfig{
			Enabled: true,
			Cache: config.CacheConfig{
				Type:   "LRU",
				Size:   10,
				Shards: 2,
			},
			PeerMaxInput: config.AntifloodLimitsConfig{
				MessagesPerSecond:  10,
				TotalSizePerSecond: 10,
			},
			NetworkMaxInput: config.AntifloodLimitsConfig{
				MessagesPerSecond:  10,
				TotalSizePerSecond: 10,
			},
			Topic: config.TopicAntifloodConfig{
				DefaultMaxMessagesPerSec: 10,
			},
			BlackList: config.BlackListConfig{
				ThresholdNumMessagesPerSecond: 10,
				ThresholdSizePerSecond:        10,
				NumFloodingRounds:             10,
				PeerBanDurationInSeconds:      10,
			},
		},
	}

	ash := &mock.AppStatusHandlerMock{}
	components, err := NewP2PAntiFloodAndBlackList(cfg, ash)
	assert.Nil(t, err)
	assert.NotNil(t, components.AntiFloodHandler)
	assert.NotNil(t, components.BlacklistHandler)
}
