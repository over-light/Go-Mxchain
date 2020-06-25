package sposFactory

import (
	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/broadcast"
	"github.com/ElrondNetwork/elrond-go/consensus/spos"
	"github.com/ElrondNetwork/elrond-go/consensus/spos/bls"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/indexer"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// GetSubroundsFactory returns a subrounds factory depending of the given parameter
func GetSubroundsFactory(
	consensusDataContainer spos.ConsensusCoreHandler,
	consensusState *spos.ConsensusState,
	worker spos.WorkerHandler,
	consensusType string,
	appStatusHandler core.AppStatusHandler,
	indexer indexer.Indexer,
	chainID []byte,
	currentPid core.PeerID,
) (spos.SubroundsFactory, error) {
	switch consensusType {
	case blsConsensusType:
		subRoundFactoryBls, err := bls.NewSubroundsFactory(
			consensusDataContainer,
			consensusState,
			worker,
			chainID,
			currentPid,
		)
		if err != nil {
			return nil, err
		}

		err = subRoundFactoryBls.SetAppStatusHandler(appStatusHandler)
		if err != nil {
			return nil, err
		}

		subRoundFactoryBls.SetIndexer(indexer)

		return subRoundFactoryBls, nil
	default:
		return nil, ErrInvalidConsensusType
	}
}

// GetConsensusCoreFactory returns a consensus service depending of the given parameter
func GetConsensusCoreFactory(consensusType string) (spos.ConsensusService, error) {
	switch consensusType {
	case blsConsensusType:
		return bls.NewConsensusService()
	default:
		return nil, ErrInvalidConsensusType
	}
}

// GetBroadcastMessenger returns a consensus service depending of the given parameter
func GetBroadcastMessenger(
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	messenger consensus.P2PMessenger,
	shardCoordinator sharding.Coordinator,
	privateKey crypto.PrivateKey,
	singleSigner crypto.SingleSigner,
	headersSubscriber consensus.HeadersPoolSubscriber,
	interceptorsContainer process.InterceptorsContainer,
) (consensus.BroadcastMessenger, error) {

	commonMessengerArgs := broadcast.CommonMessengerArgs{
		Marshalizer:                marshalizer,
		Hasher:                     hasher,
		Messenger:                  messenger,
		PrivateKey:                 privateKey,
		ShardCoordinator:           shardCoordinator,
		SingleSigner:               singleSigner,
		HeadersSubscriber:          headersSubscriber,
		MaxDelayCacheSize:          maxDelayCacheSize,
		MaxValidatorDelayCacheSize: maxDelayCacheSize,
		InterceptorsContainer:      interceptorsContainer,
	}

	if shardCoordinator.SelfId() < shardCoordinator.NumberOfShards() {
		shardMessengerArgs := broadcast.ShardChainMessengerArgs{
			CommonMessengerArgs: commonMessengerArgs,
		}

		return broadcast.NewShardChainMessenger(shardMessengerArgs)
	}

	if shardCoordinator.SelfId() == core.MetachainShardId {
		metaMessengerArgs := broadcast.MetaChainMessengerArgs{
			CommonMessengerArgs: commonMessengerArgs,
		}

		return broadcast.NewMetaChainMessenger(metaMessengerArgs)
	}

	return nil, ErrInvalidShardId
}
