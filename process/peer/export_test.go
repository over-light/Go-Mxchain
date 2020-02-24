package peer

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/data/block"
)

// CheckForMissedBlocks -
func (vs *validatorStatistics) CheckForMissedBlocks(
	currentHeaderRound uint64,
	previousHeaderRound uint64,
	prevRandSeed []byte,
	shardId uint32,
) error {
	return vs.checkForMissedBlocks(currentHeaderRound, previousHeaderRound, prevRandSeed, shardId)
}

// SaveInitialState -
func (vs *validatorStatistics) SaveInitialState(stakeValue *big.Int, initialRating uint32) error {
	return vs.saveInitialState(stakeValue, initialRating)
}

// GetMatchingPrevShardData -
func (vs *validatorStatistics) GetMatchingPrevShardData(currentShardData block.ShardData, shardInfo []block.ShardData) *block.ShardData {
	return vs.getMatchingPrevShardData(currentShardData, shardInfo)
}

// GetLeaderDecreaseCount -
func (vs *validatorStatistics) GetLeaderDecreaseCount(key []byte) uint32 {
	vs.mutMissedBlocksCounters.RLock()
	defer vs.mutMissedBlocksCounters.RUnlock()

	return vs.missedBlocksCounters.get(key).leaderDecreaseCount
}
