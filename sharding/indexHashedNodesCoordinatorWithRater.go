package sharding

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go/core/check"
)

type indexHashedNodesCoordinatorWithRater struct {
	*indexHashedNodesCoordinator
	RatingReader
}

// NewIndexHashedNodesCoordinator creates a new index hashed group selector
func NewIndexHashedNodesCoordinatorWithRater(
	indexNodesCoordinator *indexHashedNodesCoordinator,
	rater RatingReader,
) (*indexHashedNodesCoordinatorWithRater, error) {
	if check.IfNil(indexNodesCoordinator) {
		return nil, ErrNilNodesCoordinator
	}
	if check.IfNil(rater) {
		return nil, ErrNilRater
	}

	ihncr := &indexHashedNodesCoordinatorWithRater{
		indexHashedNodesCoordinator: indexNodesCoordinator,
		RatingReader:                rater,
	}

	indexNodesCoordinator.doExpandEligibleList = ihncr.expandEligibleList

	return ihncr, nil
}

func (ihgs *indexHashedNodesCoordinatorWithRater) expandEligibleList(validators []Validator, mut *sync.RWMutex) []Validator {
	mut.RLock()
	defer mut.RUnlock()

	validatorList := make([]Validator, 0)

	for _, validator := range validators {
		pk := validator.Address()
		rating := ihgs.GetRating(string(pk))
		for i := uint32(0); i < rating; i++ {
			validatorList = append(validatorList, validator)
		}
	}

	return validatorList
}

//IsInterfaceNil verifies that the underlying value is nil
func (ihgs *indexHashedNodesCoordinatorWithRater) IsInterfaceNil() bool {
	return ihgs == nil
}
