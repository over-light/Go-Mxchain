package dataprocessor

import "errors"

// ErrNilElasticIndexer signals that a nil elastic indexer has been provided
var ErrNilElasticIndexer = errors.New("nil elastic indexer")

// ErrNilDatabaseReader signals that a nil databse reader has been provided
var ErrNilDatabaseReader = errors.New("nil database reader")

// ErrNilShardCoordinator signals that a nil shard coordinator has been provided
var ErrNilShardCoordinator = errors.New("nil shard coordinator")

// ErrNilMarshalizer signals that a nil marshalizer has been provided
var ErrNilMarshalizer = errors.New("nil marshalizer")

// ErrNilHasher signals that a nil hasher has been provided
var ErrNilHasher = errors.New("nil hasher")

// ErrNilUint64ByteSliceConverter signals that a nil uint64 byte slice converter has been provided
var ErrNilUint64ByteSliceConverter = errors.New("nil uint64 byte slice converter")

// ErrNilGenesisNodesSetup signals that a nil genesis nodes setup handler has been provided
var ErrNilGenesisNodesSetup = errors.New("nil genesis nodes setup")

// ErrWrongTypeAssertion signals that an interface is not of a desired type
var ErrWrongTypeAssertion = errors.New("wrong type assertion")

// ErrTimeIsOut signals that time is out when indexing data to elastic
var ErrTimeIsOut = errors.New("time is out when indexing")

// ErrNoMetachainDatabase signals that no metachain database hasn't been found
var ErrNoMetachainDatabase = errors.New("no metachain database - cannot index")

// ErrDatabaseInfoNotFound signals that a database information hasn't been found
var ErrDatabaseInfoNotFound = errors.New("database info not found")

// ErrNilHeaderMarshalizer signals that a nil header marshalizer has been provided
var ErrNilHeaderMarshalizer = errors.New("nil header marshalizer")

// ErrRangeIsOver signals that the range cannot be continued as the handler returned false
var ErrRangeIsOver = errors.New("range is over")

// ErrNilDataReplayer signals that a nil data replayer has been provided
var ErrNilDataReplayer = errors.New("nil data replayer")

// ErrNilPubKeyConverter signals that a nil public key converter has been provided
var ErrNilPubKeyConverter = errors.New("nil public key converter")
