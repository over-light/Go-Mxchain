package factory

import "errors"

// ErrNilEconomicsData signals that a nil economics data handler has been provided
var ErrNilEconomicsData = errors.New("nil economics data provided")

// ErrNilGenesisConfiguration signals that a nil genesis configuration has been provided
var ErrNilGenesisConfiguration = errors.New("nil genesis configuration provided")

// ErrNilCoreComponents signals that nil core components have been provided
var ErrNilCoreComponents = errors.New("nil core components provided")

// ErrNilTriesComponents signals that nil tries components have been provided
var ErrNilTriesComponents = errors.New("nil tries components provided")

// ErrNilShardCoordinator signals that nil core components have been provided
var ErrNilShardCoordinator = errors.New("nil shard coordinator provided")

// ErrNilPathManager signals that a nil path manager has been provided
var ErrNilPathManager = errors.New("nil path manager provided")

// ErrNilPath signals that a nil/empty path was provided
var ErrNilPath = errors.New("nil path provided")

// ErrNilMarshalizer signals that a nil marshalizer has been provided
var ErrNilMarshalizer = errors.New("nil marshalizer provided")

// ErrNilHasher signals that a nil hasher has been provided
var ErrNilHasher = errors.New("nil hasher provided")

// ErrNilEpochStartNotifier signals that a nil epoch start notifier has been provided
var ErrNilEpochStartNotifier = errors.New("nil epoch start notifier provided")

// ErrHasherCreation signals that the hasher cannot be created based on provided data
var ErrHasherCreation = errors.New("error creating hasher")

// ErrMarshalizerCreation signals that the marshalizer cannot be created based on provided data
var ErrMarshalizerCreation = errors.New("error creating marshalizer")

// ErrPubKeyConverterCreation signals that the public key converter cannot be created based on provided data
var ErrPubKeyConverterCreation = errors.New("error creating public key converter")

// ErrAccountsAdapterCreation signals that the accounts adapter cannot be created based on provided data
var ErrAccountsAdapterCreation = errors.New("error creating accounts adapter")

// ErrInitialBalancesCreation signals that the initial balances cannot be created based on provided data
var ErrInitialBalancesCreation = errors.New("error creating initial balances")

// ErrPublicKeyMismatch signals that the read public key mismatch the one read
var ErrPublicKeyMismatch = errors.New("public key mismatch between the computed and the one read from the file")

// ErrBlockchainCreation signals that the blockchain cannot be created
var ErrBlockchainCreation = errors.New("can not create blockchain")

// ErrDataStoreCreation signals that the data store cannot be created
var ErrDataStoreCreation = errors.New("can not create data store")

// ErrDataPoolCreation signals that the data pool cannot be created
var ErrDataPoolCreation = errors.New("can not create data pool")

// ErrInvalidConsensusConfig signals that an invalid consensus type is specified in the configuration file
var ErrInvalidConsensusConfig = errors.New("invalid consensus type provided in config file")

// ErrMultiSigHasherMissmatch signals that an invalid multisig hasher was provided
var ErrMultiSigHasherMissmatch = errors.New("wrong multisig hasher provided for bls consensus type")

// ErrMissingMultiHasherConfig signals that the multihasher type isn't specified in the configuration file
var ErrMissingMultiHasherConfig = errors.New("no multisig hasher provided in config file")

// ErrNilStatusHandler signals that a nil status handler has been provided
var ErrNilStatusHandler = errors.New("nil status handler provided")

// ErrWrongTypeAssertion signals that a wrong type assertion occurred
var ErrWrongTypeAssertion = errors.New("wrong type assertion")

// ErrNilAccountsParser signals that a nil accounts parser has been provided
var ErrNilAccountsParser = errors.New("nil accounts parser")

// ErrNilSmartContractParser signals that a nil smart contract parser has been provided
var ErrNilSmartContractParser = errors.New("nil smart contract parser")

// ErrNilNodesConfig signals that a nil nodes config has been provided
var ErrNilNodesConfig = errors.New("nil nodes config")

// ErrNilGasSchedule signals that a nil gas schedule has been provided
var ErrNilGasSchedule = errors.New("nil gas schedule")

// ErrNilRounder signals that a nil rounder has been provided
var ErrNilRounder = errors.New("nil rounder")

// ErrNilNodesCoordinator signals that nil nodes coordinator has been provided
var ErrNilNodesCoordinator = errors.New("nil nodes coordinator")

// ErrNilDataComponents signals that nil data components have been provided
var ErrNilDataComponents = errors.New("nil data components")

// ErrNilCoreComponentsHolder signals that a nil core components holder has been provided
var ErrNilCoreComponentsHolder = errors.New("nil core components holder")

// ErrNilCryptoComponentsHolder signals that a nil crypto components holder has been provided
var ErrNilCryptoComponentsHolder = errors.New("nil crypto components holder")

// ErrNilStateComponents signals that nil state components have been provided
var ErrNilStateComponents = errors.New("nil state components")

// ErrNilNetworkComponentsHolder signals that a nil network components holder has been provided
var ErrNilNetworkComponentsHolder = errors.New("nil network components holder")

// ErrNilCoreServiceContainer signals that a nil core service container has been provided
var ErrNilCoreServiceContainer = errors.New("nil core service container")

// ErrNilRequestedItemHandler signals that a nil requested item handler has been provided
var ErrNilRequestedItemHandler = errors.New("nil requested item handler")

// ErrNilWhiteListHandler signals that a nil white list handler has been provided
var ErrNilWhiteListHandler = errors.New("nil white list handler")

// ErrNilWhiteListVerifiedTxs signals that a nil white list verifies txs has been provided
var ErrNilWhiteListVerifiedTxs = errors.New("nil white list verified txs")

// ErrNilEpochStartConfig signals that a nil epoch start configuration has been provided
var ErrNilEpochStartConfig = errors.New("nil epoch start configuration")

// ErrNilRater signals that a nil rater has been provided
var ErrNilRater = errors.New("nil rater")

// ErrNilRatingData signals that a nil rating data has been provided
var ErrNilRatingData = errors.New("nil rating data")

// ErrNilPubKeyConverter signals that a nil public key converter has been provided
var ErrNilPubKeyConverter = errors.New("nil public key converter")

// ErrNilSystemSCConfig signals that a nil system smart contract configuration has been provided
var ErrNilSystemSCConfig = errors.New("nil system smart contract configuration")

// ErrNilTxLogsConfiguration signals that a nil transaction logs processor has been provided
var ErrNilTxLogsConfiguration = errors.New("nil transaction logs processor")

// ErrInvalidRoundDuration signals that an invalid round duration has been provided
var ErrInvalidRoundDuration = errors.New("invalid round duration provided")

// ErrNilElasticOptions signals that nil elastic options have been provided
var ErrNilElasticOptions = errors.New("nil elastic options")
