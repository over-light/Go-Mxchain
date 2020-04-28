package factory

import "errors"

// ErrNilConfiguration signals that a nil configuration has been provided
var ErrNilConfiguration = errors.New("nil configuration provided")

// ErrNilEconomicsData signals that a nil economics data handler has been provided
var ErrNilEconomicsData = errors.New("nil economics data provided")

// ErrNilGenesisConfiguration signals that a nil genesis configuration has been provided
var ErrNilGenesisConfiguration = errors.New("nil genesis configuration provided")

// ErrNilCoreComponents signals that nil core components have been provided
var ErrNilCoreComponents = errors.New("nil core components provided")

// ErrNilShardCoordinator signals that nil core components have been provided
var ErrNilShardCoordinator = errors.New("nil shard coordinator provided")

// ErrNilPathManager signals that a nil path manager has been provided
var ErrNilPathManager = errors.New("nil path manager provided")

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

// ErrNilPubKeyConverter signals that a nil public key converter has been provided
var ErrNilPubKeyConverter = errors.New("nil public key converter provided")

// ErrNilSuite signals that a nil suite has been provided
var ErrNilSuite = errors.New("nil suite provided")

// ErrPublicKeyMismatch signals that the read public key mismatch the one read
var ErrPublicKeyMismatch = errors.New("public key mismatch between the computed and the one read from the file")

// ErrBlockchainCreation signals that the blockchain cannot be created
var ErrBlockchainCreation = errors.New("can not create blockchain")

// ErrDataStoreCreation signals that the data store cannot be created
var ErrDataStoreCreation = errors.New("can not create data store")

// ErrDataPoolCreation signals that the data pool cannot be created
var ErrDataPoolCreation = errors.New("can not create data pool")
