package spos

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/consensus/mock"
	"github.com/stretchr/testify/assert"
)

func initConsensusDataContainer() *ConsensusCore {
	blockChain := &mock.BlockChainMock{}
	blockProcessorMock := mock.InitBlockProcessorMock()
	blocksTrackerMock := &mock.BlocksTrackerMock{}
	bootstrapperMock := &mock.BootstrapperMock{}
	broadcastMessengerMock := &mock.BroadcastMessengerMock{}
	chronologyHandlerMock := mock.InitChronologyHandlerMock()
	blsPrivateKeyMock := &mock.PrivateKeyMock{}
	blsSingleSignerMock := &mock.SingleSignerMock{}
	multiSignerMock := mock.NewMultiSigner()
	hasherMock := mock.HasherMock{}
	marshalizerMock := mock.MarshalizerMock{}
	rounderMock := &mock.RounderMock{}
	shardCoordinatorMock := mock.ShardCoordinatorMock{}
	syncTimerMock := &mock.SyncTimerMock{}
	validatorGroupSelector := &mock.NodesCoordinatorMock{}

	return &ConsensusCore{
		blockChain:         blockChain,
		blockProcessor:     blockProcessorMock,
		blocksTracker:      blocksTrackerMock,
		bootstrapper:       bootstrapperMock,
		broadcastMessenger: broadcastMessengerMock,
		chronologyHandler:  chronologyHandlerMock,
		hasher:             hasherMock,
		marshalizer:        marshalizerMock,
		blsPrivateKey:      blsPrivateKeyMock,
		blsSingleSigner:    blsSingleSignerMock,
		multiSigner:        multiSignerMock,
		rounder:            rounderMock,
		shardCoordinator:   shardCoordinatorMock,
		syncTimer:          syncTimerMock,
		nodesCoordinator:   validatorGroupSelector,
	}
}

func TestConsensusContainerValidator_ValidateNilBlockchainShouldFail(t *testing.T) {
	t.Parallel()

	container := initConsensusDataContainer()
	container.blockChain = nil

	err := ValidateConsensusCore(container)

	assert.Equal(t, ErrNilBlockChain, err)
}

func TestConsensusContainerValidator_ValidateNilProcessorShouldFail(t *testing.T) {
	t.Parallel()

	container := initConsensusDataContainer()
	container.blockProcessor = nil

	err := ValidateConsensusCore(container)

	assert.Equal(t, ErrNilBlockProcessor, err)
}

func TestConsensusContainerValidator_ValidateNilBootstrapperShouldFail(t *testing.T) {
	t.Parallel()

	container := initConsensusDataContainer()
	container.bootstrapper = nil

	err := ValidateConsensusCore(container)

	assert.Equal(t, ErrNilBootstrapper, err)
}

func TestConsensusContainerValidator_ValidateNilChronologyShouldFail(t *testing.T) {
	t.Parallel()

	container := initConsensusDataContainer()
	container.chronologyHandler = nil

	err := ValidateConsensusCore(container)

	assert.Equal(t, ErrNilChronologyHandler, err)
}

func TestConsensusContainerValidator_ValidateNilHasherShouldFail(t *testing.T) {
	t.Parallel()

	container := initConsensusDataContainer()
	container.hasher = nil

	err := ValidateConsensusCore(container)

	assert.Equal(t, ErrNilHasher, err)
}

func TestConsensusContainerValidator_ValidateNilMarshalizerShouldFail(t *testing.T) {
	t.Parallel()

	container := initConsensusDataContainer()
	container.marshalizer = nil

	err := ValidateConsensusCore(container)

	assert.Equal(t, ErrNilMarshalizer, err)
}

func TestConsensusContainerValidator_ValidateNilMultiSignerShouldFail(t *testing.T) {
	t.Parallel()

	container := initConsensusDataContainer()
	container.multiSigner = nil

	err := ValidateConsensusCore(container)

	assert.Equal(t, ErrNilMultiSigner, err)
}

func TestConsensusContainerValidator_ValidateNilRounderShouldFail(t *testing.T) {
	t.Parallel()

	container := initConsensusDataContainer()
	container.rounder = nil

	err := ValidateConsensusCore(container)

	assert.Equal(t, ErrNilRounder, err)
}

func TestConsensusContainerValidator_ValidateNilShardCoordinatorShouldFail(t *testing.T) {
	t.Parallel()

	container := initConsensusDataContainer()
	container.shardCoordinator = nil

	err := ValidateConsensusCore(container)

	assert.Equal(t, ErrNilShardCoordinator, err)
}

func TestConsensusContainerValidator_ValidateNilSyncTimerShouldFail(t *testing.T) {
	t.Parallel()

	container := initConsensusDataContainer()
	container.syncTimer = nil

	err := ValidateConsensusCore(container)

	assert.Equal(t, ErrNilSyncTimer, err)
}

func TestConsensusContainerValidator_ValidateNilValidatorGroupSelectorShouldFail(t *testing.T) {
	t.Parallel()

	container := initConsensusDataContainer()
	container.nodesCoordinator = nil

	err := ValidateConsensusCore(container)

	assert.Equal(t, ErrNilValidatorGroupSelector, err)
}
