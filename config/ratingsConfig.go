package config

// RatingsConfig will hold the configuration data needed for the ratings
type RatingsConfig struct {
	General    General
	ShardChain ShardChain
	MetaChain  MetaChain
}

// General will hold ratings settings both for metachain and shardChain
type General struct {
	StartRating      uint32
	MaxRating        uint32
	MinRating        uint32
	SelectionChances []*SelectionChance
}

// ShardChain will hold RatingSteps for the Shard
type ShardChain struct {
	RatingSteps
}

// MetaChain will hold RatingSteps for the Meta
type MetaChain struct {
	RatingSteps
}

//RatingValue will hold different rating options with increase and decrease steps
type RatingValue struct {
	Name  string
	Value int32
}

//RatingValue will hold different rating options with increase and decresea steps
type SelectionChance struct {
	MaxThreshold  uint32
	ChancePercent uint32
}

// RatingSteps holds the necessary increases and decreases of the rating steps
type RatingSteps struct {
	ProposerIncreaseRatingStep  int32
	ProposerDecreaseRatingStep  int32
	ValidatorIncreaseRatingStep int32
	ValidatorDecreaseRatingStep int32
}
