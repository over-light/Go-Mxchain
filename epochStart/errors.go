package epochStart

import "errors"

// ErrNilArgsNewMetaEpochStartTrigger signals that nil arguments were provided
var ErrNilArgsNewMetaEpochStartTrigger = errors.New("nil arguments for meta start of epoch trigger")

// ErrNilEpochStartSettings signals that nil start of epoch settings has been provided
var ErrNilEpochStartSettings = errors.New("nil start of epoch settings")

// ErrInvalidSettingsForEpochStartTrigger signals that settings for start of epoch trigger are invalid
var ErrInvalidSettingsForEpochStartTrigger = errors.New("invalid start of epoch trigger settings")

// ErrNilArgsNewShardEpochStartTrigger signals that nil arguments for shard epoch trigger has been provided
var ErrNilArgsNewShardEpochStartTrigger = errors.New("nil arguments for shard start of epoch trigger")

// ErrNilEpochStartNotifier signals that nil epoch start notifier has been provided
var ErrNilEpochStartNotifier = errors.New("nil epoch start notifier")

// ErrNotEnoughRoundsBetweenEpochs signals that not enough rounds has passed since last epoch start
var ErrNotEnoughRoundsBetweenEpochs = errors.New("tried to force start of epoch before passing of enough rounds")

// ErrForceEpochStartCanBeCalledOnlyOnNewRound signals that force start of epoch was called on wrong round
var ErrForceEpochStartCanBeCalledOnlyOnNewRound = errors.New("invalid time to call force start of epoch, possible only on new round")

// ErrSavedRoundIsHigherThanInputRound signals that input round was wrong
var ErrSavedRoundIsHigherThanInputRound = errors.New("saved round is higher than input round")

// ErrSavedRoundIsHigherThanInput signals that input round was wrong
var ErrSavedRoundIsHigherThanInput = errors.New("saved round is higher than input round")

// ErrWrongTypeAssertion signals wrong type assertion
var ErrWrongTypeAssertion = errors.New("wrong type assertion")

// ErrNilMarshalizer signals that nil marshalizer has been provided
var ErrNilMarshalizer = errors.New("nil marshalizer")

// ErrNilStorage signals that nil storage has been provided
var ErrNilStorage = errors.New("nil storage")

// ErrNilHeaderHandler signals that a nil header handler has been provided
var ErrNilHeaderHandler = errors.New("nil header handler")

// ErrNilMiniblocks signals that nil argument was passed
var ErrNilMiniblocks = errors.New("nil arguments for miniblocks object")

// ErrNilMiniblock signals that nil miniblock has been provided
var ErrNilMiniblock = errors.New("nil miniblock")

// ErrMetaHdrNotFound signals that metaheader was not found
var ErrMetaHdrNotFound = errors.New("meta header not found")

// ErrNilHasher signals that nil hasher has been provided
var ErrNilHasher = errors.New("nil hasher")

// ErrNilHeaderValidator signals that nil header validator has been provided
var ErrNilHeaderValidator = errors.New("nil header validator")

// ErrNilDataPoolsHolder signals that nil data pools holder has been provided
var ErrNilDataPoolsHolder = errors.New("nil data pools holder")

// ErrNilStorageService signals that nil storage service has been provided
var ErrNilStorageService = errors.New("nil storage service")

// ErrNilRequestHandler signals that nil request handler has been provided
var ErrNilRequestHandler = errors.New("nil request handler")

// ErrNilMetaBlockStorage signals that nil metablocks storage has been provided
var ErrNilMetaBlockStorage = errors.New("nil metablocks storage")

// ErrNilMetaBlocksPool signals that nil metablock pools holder has been provided
var ErrNilMetaBlocksPool = errors.New("nil metablocks pool")

// ErrNilValidatorInfoProcessor signals that a nil validator info processor has been provided
var ErrNilValidatorInfoProcessor = errors.New("nil validator info processor")

// ErrNilUint64Converter signals that nil uint64 converter has been provided
var ErrNilUint64Converter = errors.New("nil uint64 converter")

// ErrNilTriggerStorage signals that nil meta header storage has been provided
var ErrNilTriggerStorage = errors.New("nil trigger storage")

// ErrNilMetaNonceHashStorage signals that nil meta header nonce hash storage has been provided
var ErrNilMetaNonceHashStorage = errors.New("nil meta nonce hash storage")

// ErrValidatorMiniBlockHashDoesNotMatch signals that created and received validatorInfo miniblock hash does not match
var ErrValidatorMiniBlockHashDoesNotMatch = errors.New("validatorInfo miniblock hash does not match")

// ErrRewardMiniBlockHashDoesNotMatch signals that created and received rewards miniblock hash does not match
var ErrRewardMiniBlockHashDoesNotMatch = errors.New("reward miniblock hash does not match")

// ErrNilShardCoordinator is raised when a valid shard coordinator is expected but nil used
var ErrNilShardCoordinator = errors.New("shard coordinator is nil")

// ErrNilAddressConverter signals that nil address converter was provided
var ErrNilAddressConverter = errors.New("nil address converter")

// ErrRewardMiniBlocksNumDoesNotMatch signals that number of created and received rewards miniblocks is not equal
var ErrRewardMiniBlocksNumDoesNotMatch = errors.New("number of created and received rewards miniblocks missmatch")

// ErrNilRewardsHandler signals that rewards handler is nil
var ErrNilRewardsHandler = errors.New("rewards handler is nil")

// ErrNilTotalAccumulatedFeesInEpoch signals that total accumulated fees in epoch is nil
var ErrNilTotalAccumulatedFeesInEpoch = errors.New("total accumulated fees in epoch is nil")

// ErrEndOfEpochEconomicsDataDoesNotMatch signals that end of epoch data does not match
var ErrEndOfEpochEconomicsDataDoesNotMatch = errors.New("end of epoch economics data does not match")

// ErrNilRounder signals that an operation has been attempted to or with a nil Rounder implementation
var ErrNilRounder = errors.New("nil Rounder")

// ErrNilNodesCoordinator signals that an operation has been attempted to or with a nil nodes coordinator
var ErrNilNodesCoordinator = errors.New("nil nodes coordinator")

// ErrNotEpochStartBlock signals that block is not of type epoch start
var ErrNotEpochStartBlock = errors.New("not epoch start block")

// ErrNilShardHeaderStorage signals that shard header storage is nil
var ErrNilShardHeaderStorage = errors.New("nil shard header storage")

// ErrValidatorInfoMiniBlocksNumDoesNotMatch signals that number of created and received validatorInfo miniblocks is not equal
var ErrValidatorInfoMiniBlocksNumDoesNotMatch = errors.New("number of created and received validatorInfo miniblocks missmatch")

// ErrNilValidatorInfo signals that a nil value for the validatorInfo has been provided
var ErrNilValidatorInfo = errors.New("validator info is nil")

// ErrEpochStartDataForShardNotFound signals that epoch start shard data was not found for current shard id
var ErrEpochStartDataForShardNotFound = errors.New("epoch start data for current shard not found")

// ErrNumTriesExceeded signals that number of tries has exceeded
var ErrNumTriesExceeded = errors.New("number of tries exceeded")

// ErrMissingHeader signals that searched header is missing
var ErrMissingHeader = errors.New("missing header")

// ErrSmallShardEligibleListSize signals that the eligible validators list's size is less than the consensus size
var ErrSmallShardEligibleListSize = errors.New("small shard eligible list size")

// ErrSmallMetachainEligibleListSize signals that the eligible validators list's size is less than the consensus size
var ErrSmallMetachainEligibleListSize = errors.New("small metachain eligible list size")
