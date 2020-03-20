package rating

import (
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/process"
)

// RatingsData will store information about ratingsComputation specific for a shard or metachain
type RatingStep struct {
	proposerIncreaseRatingStep  int32
	proposerDecreaseRatingStep  int32
	validatorIncreaseRatingStep int32
	validatorDecreaseRatingStep int32
}

// NewRatingsData creates a new RatingsData instance
func NewRatingStepData(steps config.RatingSteps) process.RatingsStepHandler {
	return &RatingStep{
		proposerIncreaseRatingStep:  steps.ProposerIncreaseRatingStep,
		proposerDecreaseRatingStep:  steps.ProposerDecreaseRatingStep,
		validatorIncreaseRatingStep: steps.ValidatorIncreaseRatingStep,
		validatorDecreaseRatingStep: steps.ValidatorDecreaseRatingStep,
	}
}

// ProposerIncreaseRatingStep will return the rating step increase for validator
func (rd *RatingStep) ProposerIncreaseRatingStep() int32 {
	return rd.proposerIncreaseRatingStep
}

// ProposerDecreaseRatingStep will return the rating step decrease for proposer
func (rd *RatingStep) ProposerDecreaseRatingStep() int32 {
	return rd.proposerDecreaseRatingStep
}

// ValidatorIncreaseRatingStep will return the rating step increase for validator
func (rd *RatingStep) ValidatorIncreaseRatingStep() int32 {
	return rd.validatorIncreaseRatingStep
}

// ValidatorDecreaseRatingStep will return the rating step decrease for validator
func (rd *RatingStep) ValidatorDecreaseRatingStep() int32 {
	return rd.validatorDecreaseRatingStep
}
