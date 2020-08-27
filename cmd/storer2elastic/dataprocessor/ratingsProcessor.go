package dataprocessor

import (
	"fmt"
	"path/filepath"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/indexer"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	factory2 "github.com/ElrondNetwork/elrond-go/data/state/factory"
	"github.com/ElrondNetwork/elrond-go/data/trie/factory"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage/pathmanager"
)

// RatingProcessorArgs holds the arguments needed for creating a new ratingsProcessor
type RatingProcessorArgs struct {
	ValidatorPubKeyConverter core.PubkeyConverter
	GenesisNodesConfig       sharding.GenesisNodesSetupHandler
	ShardCoordinator         sharding.Coordinator
	DbPathWithChainID        string
	GeneralConfig            config.Config
	Marshalizer              marshal.Marshalizer
	Hasher                   hashing.Hasher
	ElasticIndexer           indexer.Indexer
}

type ratingsProcessor struct {
	validatorPubKeyConverter core.PubkeyConverter
	shardCoordinator         sharding.Coordinator
	generalConfig            config.Config
	dbPathWithChainID        string
	marshalizer              marshal.Marshalizer
	hasher                   hashing.Hasher
	elasticIndexer           indexer.Indexer
	peerAdapter              state.AccountsAdapter
	genesisNodesConfig       sharding.GenesisNodesSetupHandler
}

// NewRatingsProcessor will return a new instance of ratingsProcessor
func NewRatingsProcessor(args RatingProcessorArgs) (*ratingsProcessor, error) {
	if check.IfNil(args.ValidatorPubKeyConverter) {
		return nil, ErrNilPubKeyConverter
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, ErrNilShardCoordinator
	}
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.Hasher) {
		return nil, ErrNilHasher
	}
	if check.IfNil(args.ElasticIndexer) {
		return nil, ErrNilElasticIndexer
	}
	if check.IfNil(args.GenesisNodesConfig) {
		return nil, ErrNilGenesisNodesSetup
	}

	rp := &ratingsProcessor{
		validatorPubKeyConverter: args.ValidatorPubKeyConverter,
		shardCoordinator:         args.ShardCoordinator,
		generalConfig:            args.GeneralConfig,
		marshalizer:              args.Marshalizer,
		hasher:                   args.Hasher,
		elasticIndexer:           args.ElasticIndexer,
		dbPathWithChainID:        args.DbPathWithChainID,
		genesisNodesConfig:       args.GenesisNodesConfig,
	}

	err := rp.createPeerAdapter()
	if err != nil {
		return nil, err
	}

	return rp, nil
}

// IndexRatingsForEpochStartMetaBlock will index the ratings for an epoch start meta block
func (rp *ratingsProcessor) IndexRatingsForEpochStartMetaBlock(metaBlock *block.MetaBlock) error {
	if metaBlock.GetNonce() == 0 {
		rp.indexRating(0, rp.getGenesisRating())
		return nil
	}

	rootHash := metaBlock.ValidatorStatsRootHash
	leaves, err := rp.peerAdapter.GetAllLeaves(rootHash)
	if err != nil {
		return err
	}

	validatorsRatingData, err := rp.getValidatorsRatingFromLeaves(leaves)
	if err != nil {
		return err
	}

	rp.indexRating(metaBlock.GetEpoch(), validatorsRatingData)

	return nil
}

func (rp *ratingsProcessor) indexRating(epoch uint32, validatorsRatingData map[uint32][]indexer.ValidatorRatingInfo) {
	for shardID, validators := range validatorsRatingData {
		index := fmt.Sprintf("%d_%d", shardID, epoch)
		rp.elasticIndexer.SaveValidatorsRating(index, validators)
		log.Info("indexed validators rating", "shard ID", shardID, "num peers", len(validators))
	}
}

func (rp *ratingsProcessor) createPeerAdapter() error {
	pathTemplateForPruningStorer := filepath.Join(
		rp.dbPathWithChainID,
		fmt.Sprintf("%s_%s", "Epoch", core.PathEpochPlaceholder),
		fmt.Sprintf("%s_%s", "Shard", core.PathShardPlaceholder),
		core.PathIdentifierPlaceholder)

	pathTemplateForStaticStorer := filepath.Join(
		rp.dbPathWithChainID,
		"Static",
		fmt.Sprintf("%s_%s", "Shard", core.PathShardPlaceholder),
		core.PathIdentifierPlaceholder)
	pathManager, err := pathmanager.NewPathManager(pathTemplateForPruningStorer, pathTemplateForStaticStorer)
	if err != nil {
		return err
	}
	trieFactoryArgs := factory.TrieFactoryArgs{
		EvictionWaitingListCfg:   rp.generalConfig.EvictionWaitingList,
		SnapshotDbCfg:            rp.generalConfig.TrieSnapshotDB,
		Marshalizer:              rp.marshalizer,
		Hasher:                   rp.hasher,
		PathManager:              pathManager,
		TrieStorageManagerConfig: rp.generalConfig.TrieStorageManagerConfig,
	}
	trieFactory, err := factory.NewTrieFactory(trieFactoryArgs)
	if err != nil {
		return err
	}

	_, peerAccountsTrie, err := trieFactory.Create(
		rp.generalConfig.PeerAccountsTrieStorage,
		core.GetShardIDString(core.MetachainShardId),
		rp.generalConfig.StateTriesConfig.PeerStatePruningEnabled,
		rp.generalConfig.StateTriesConfig.MaxPeerTrieLevelInMemory,
	)
	if err != nil {
		return err
	}

	peerAdapter, err := state.NewPeerAccountsDB(
		peerAccountsTrie,
		rp.hasher,
		rp.marshalizer,
		factory2.NewPeerAccountCreator(),
	)
	if err != nil {
		return err
	}

	rp.peerAdapter = peerAdapter
	return nil
}

func (rp *ratingsProcessor) getValidatorsRatingFromLeaves(leaves map[string][]byte) (map[uint32][]indexer.ValidatorRatingInfo, error) {
	validatorsRatingInfo := make(map[uint32][]indexer.ValidatorRatingInfo)
	for _, pa := range leaves {
		peerAccount, err := unmarshalPeer(pa, rp.marshalizer)
		if err != nil {
			continue
		}

		validatorsRatingInfo[peerAccount.GetShardId()] = append(validatorsRatingInfo[peerAccount.GetShardId()],
			indexer.ValidatorRatingInfo{
				PublicKey: rp.validatorPubKeyConverter.Encode(peerAccount.GetBLSPublicKey()),
				Rating:    float32(peerAccount.GetRating()) * 100 / 10000000,
			})
	}

	return validatorsRatingInfo, nil
}

func (rp *ratingsProcessor) getGenesisRating() map[uint32][]indexer.ValidatorRatingInfo {
	mapToRet := make(map[uint32][]indexer.ValidatorRatingInfo)

	eligible, waiting := rp.genesisNodesConfig.InitialNodesInfo()
	for shardId, nodesInShard := range eligible {
		for _, node := range nodesInShard {
			mapToRet[shardId] = append(mapToRet[shardId], indexer.ValidatorRatingInfo{
				PublicKey: rp.validatorPubKeyConverter.Encode(node.PubKeyBytes()),
				Rating:    float32(node.GetInitialRating()),
			})
		}
	}
	for shardId, nodesInShard := range waiting {
		for _, node := range nodesInShard {
			mapToRet[shardId] = append(mapToRet[shardId], indexer.ValidatorRatingInfo{
				PublicKey: rp.validatorPubKeyConverter.Encode(node.PubKeyBytes()),
				Rating:    float32(node.GetInitialRating()) * 100 / 10000000,
			})
		}
	}

	return mapToRet
}

func unmarshalPeer(pa []byte, marshalizer marshal.Marshalizer) (state.PeerAccountHandler, error) {
	peerAccount := state.NewEmptyPeerAccount()
	err := marshalizer.Unmarshal(peerAccount, pa)
	if err != nil {
		log.Error("cannot unmarshal peer account", "error", err)
		return nil, err
	}

	return peerAccount, nil
}
