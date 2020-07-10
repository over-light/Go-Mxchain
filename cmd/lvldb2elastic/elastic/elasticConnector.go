package elastic

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go/cmd/lvldb2elastic/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/indexer"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
)

// ConnectorFactoryArgs holds the data needed for creating a new elastic search connector factory
type ConnectorFactoryArgs struct {
	ElasticConfig            config.ElasticSearchConfig
	Marshalizer              marshal.Marshalizer
	Hasher                   hashing.Hasher
	ValidatorPubKeyConverter core.PubkeyConverter
	AddressPubKeyConverter   core.PubkeyConverter
}

type elasticSearchConnectorFactory struct {
	elasticConfig            config.ElasticSearchConfig
	marshalizer              marshal.Marshalizer
	hasher                   hashing.Hasher
	validatorPubKeyConverter core.PubkeyConverter
	addressPubKeyConverter   core.PubkeyConverter
}

// NewConnectorFactory will return a new instance of a elasticSearchConnectorFactory
func NewConnectorFactory(args ConnectorFactoryArgs) (*elasticSearchConnectorFactory, error) {
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.Hasher) {
		return nil, ErrNilHasher
	}
	if check.IfNil(args.AddressPubKeyConverter) {
		return nil, fmt.Errorf("%w for addresses", ErrNilPubKeyConverter)
	}
	if check.IfNil(args.ValidatorPubKeyConverter) {
		return nil, fmt.Errorf("%w for public keys", ErrNilPubKeyConverter)
	}

	return &elasticSearchConnectorFactory{
		elasticConfig:            args.ElasticConfig,
		marshalizer:              args.Marshalizer,
		hasher:                   args.Hasher,
		validatorPubKeyConverter: args.ValidatorPubKeyConverter,
		addressPubKeyConverter:   args.AddressPubKeyConverter,
	}, nil
}

// Create will create and return a new indexer database handler
func (escf *elasticSearchConnectorFactory) Create() (indexer.DatabaseHandler, error) {
	databaseHandlerArgs := indexer.ElasticSearchDatabaseArgs{
		Url:                      escf.elasticConfig.URL,
		UserName:                 escf.elasticConfig.Username,
		Password:                 escf.elasticConfig.Password,
		Marshalizer:              escf.marshalizer,
		Hasher:                   escf.hasher,
		AddressPubkeyConverter:   escf.addressPubKeyConverter,
		ValidatorPubkeyConverter: escf.validatorPubKeyConverter,
	}

	return indexer.NewElasticSearchDatabase(databaseHandlerArgs)
}

// IsInterfaceNil returns true if there is no value under the interface
func (escf *elasticSearchConnectorFactory) IsInterfaceNil() bool {
	return escf == nil
}
