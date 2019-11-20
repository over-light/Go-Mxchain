package sharding

import (
	"fmt"
	"github.com/ElrondNetwork/elrond-go/core"
)

type indexHashedNodesCoordinatorWithRater struct {
	indexHashedNodesCoordinator
	RatingReader
}

// NewIndexHashedNodesCoordinator creates a new index hashed group selector
func NewIndexHashedNodesCoordinatorWithRater(arguments ArgNodesCoordinator, rater RatingReader) (*indexHashedNodesCoordinatorWithRater,
	error) {
	indexNodesCoordinator, err := NewIndexHashedNodesCoordinator(arguments)

	if err != nil {
		return nil, err
	}

	if rater == nil {
		return nil, ErrNilRater
	}

	return &indexHashedNodesCoordinatorWithRater{
		indexHashedNodesCoordinator: *indexNodesCoordinator,
		RatingReader:                rater,
	}, nil
}

func (ihgs *indexHashedNodesCoordinatorWithRater) ComputeValidatorsGroup(
	randomness []byte,
	round uint64,
	shardId uint32,
) (validatorsGroup []Validator, err error) {
	if randomness == nil {
		return nil, ErrNilRandomness
	}

	if shardId >= ihgs.nbShards && shardId != MetachainShardId {
		return nil, ErrInvalidShardId
	}

	if ihgs == nil {
		return nil, ErrNilRandomness
	}

	tempList := make([]Validator, 0)
	consensusSize := ihgs.consensusGroupSize(shardId)
	randomness = []byte(fmt.Sprintf("%d-%s", round, core.ToB64(randomness)))

	// TODO: pre-compute eligible list and update only on rating change.
	expandedList := ihgs.expandEligibleList(shardId)
	lenExpandedList := len(expandedList)

	for startIdx := 0; startIdx < consensusSize; startIdx++ {
		proposedIndex := ihgs.computeListIndex(startIdx, lenExpandedList, string(randomness))
		checkedIndex := ihgs.checkIndex(proposedIndex, expandedList, tempList)
		tempList = append(tempList, expandedList[checkedIndex])
	}

	return tempList, nil
}

func (ihgs *indexHashedNodesCoordinatorWithRater) expandEligibleList(shardId uint32) []Validator {
	validatorList := make([]Validator, 0)

	for _, validator := range ihgs.nodesMap[shardId] {
		pk := validator.PubKey()
		rating := ihgs.GetRating(string(pk))
		for i := uint32(0); i < rating; i++ {
			validatorList = append(validatorList, validator)
		}
	}

	return validatorList
}
