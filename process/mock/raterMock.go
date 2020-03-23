package mock

import (
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// RaterMock -
type RaterMock struct {
	StartRating           uint32
	MinRating             uint32
	MaxRating             uint32
	Chance                uint32
	IncreaseProposer      int32
	DecreaseProposer      int32
	IncreaseValidator     int32
	DecreaseValidator     int32
	MetaIncreaseProposer  int32
	MetaDecreaseProposer  int32
	MetaIncreaseValidator int32
	MetaDecreaseValidator int32

	GetRatingCalled                func(string) uint32
	GetStartRatingCalled           func() uint32
	ComputeIncreaseProposerCalled  func(shardId uint32, rating uint32) uint32
	ComputeDecreaseProposerCalled  func(shardId uint32, rating uint32) uint32
	ComputeIncreaseValidatorCalled func(shardId uint32, rating uint32) uint32
	ComputeDecreaseValidatorCalled func(shardId uint32, rating uint32) uint32
	GetChancesCalled               func(val uint32) uint32
	RatingReader                   sharding.RatingReader
}

// GetNewMockRater -
func GetNewMockRater() *RaterMock {
	raterMock := &RaterMock{}
	raterMock.GetRatingCalled = func(s string) uint32 {
		return raterMock.StartRating
	}
	raterMock.GetStartRatingCalled = func() uint32 {
		return raterMock.StartRating
	}
	raterMock.ComputeIncreaseProposerCalled = func(shardId uint32, rating uint32) uint32 {
		var ratingStep int32
		if shardId == core.MetachainShardId {
			ratingStep = int32(raterMock.MetaIncreaseProposer)
		} else {
			ratingStep = int32(raterMock.IncreaseProposer)
		}
		return raterMock.computeRating(rating, ratingStep)
	}
	raterMock.ComputeDecreaseProposerCalled = func(shardId uint32, rating uint32) uint32 {
		var ratingStep int32
		if shardId == core.MetachainShardId {
			ratingStep = raterMock.MetaDecreaseProposer
		} else {
			ratingStep = raterMock.DecreaseProposer
		}
		return raterMock.computeRating(rating, ratingStep)
	}
	raterMock.ComputeIncreaseValidatorCalled = func(shardId uint32, rating uint32) uint32 {
		var ratingStep int32
		if shardId == core.MetachainShardId {
			ratingStep = raterMock.MetaIncreaseValidator
		} else {
			ratingStep = raterMock.IncreaseValidator
		}
		return raterMock.computeRating(rating, ratingStep)
	}
	raterMock.ComputeDecreaseValidatorCalled = func(shardId uint32, rating uint32) uint32 {
		var ratingStep int32
		if shardId == core.MetachainShardId {
			ratingStep = raterMock.MetaDecreaseValidator
		} else {
			ratingStep = raterMock.DecreaseValidator
		}
		return raterMock.computeRating(rating, ratingStep)
	}
	raterMock.GetChancesCalled = func(val uint32) uint32 {
		return raterMock.Chance
	}
	raterMock.GetChancesCalled = func(val uint32) uint32 {
		return raterMock.Chance
	}
	return raterMock
}

func (rm *RaterMock) computeRating(rating uint32, ratingStep int32) uint32 {
	newVal := int64(rating) + int64(ratingStep)
	if newVal < int64(rm.MinRating) {
		return rm.MinRating
	}
	if newVal > int64(rm.MaxRating) {
		return rm.MaxRating
	}

	return uint32(newVal)
}

// GetRating -
func (rm *RaterMock) GetRating(pk string) uint32 {
	return rm.GetRatingCalled(pk)
}

// GetStartRating -
func (rm *RaterMock) GetStartRating() uint32 {
	return rm.GetStartRatingCalled()
}

// ComputeIncreaseProposer -
func (rm *RaterMock) ComputeIncreaseProposer(shardId uint32, currentRating uint32) uint32 {
	return rm.ComputeIncreaseProposerCalled(shardId, currentRating)
}

// ComputeDecreaseProposer -
func (rm *RaterMock) ComputeDecreaseProposer(shardId uint32, currentRating uint32) uint32 {
	return rm.ComputeDecreaseProposerCalled(shardId, currentRating)
}

// ComputeIncreaseValidator -
func (rm *RaterMock) ComputeIncreaseValidator(shardId uint32, currentRating uint32) uint32 {
	return rm.ComputeIncreaseValidatorCalled(shardId, currentRating)
}

// ComputeDecreaseValidator -
func (rm *RaterMock) ComputeDecreaseValidator(shardId uint32, currentRating uint32) uint32 {
	return rm.ComputeDecreaseValidatorCalled(shardId, currentRating)
}

// SetRatingReader -
func (rm *RaterMock) SetRatingReader(reader sharding.RatingReader) {
	rm.RatingReader = reader
}

// GetChance -
func (rm *RaterMock) GetChance(rating uint32) uint32 {
	return rm.GetChancesCalled(rating)
}

// IsInterfaceNil -
func (rm *RaterMock) IsInterfaceNil() bool {
	return rm == nil
}
