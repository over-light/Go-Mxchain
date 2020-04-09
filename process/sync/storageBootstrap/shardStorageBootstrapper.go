package storageBootstrap

import (
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
)

type shardStorageBootstrapper struct {
	*storageBootstrapper
}

// NewShardStorageBootstrapper is method used to create a new storage bootstrapper
func NewShardStorageBootstrapper(arguments ArgsShardStorageBootstrapper) (*shardStorageBootstrapper, error) {
	err := checkShardStorageBootstrapperArgs(arguments)
	if err != nil {
		return nil, err
	}

	base := &storageBootstrapper{
		bootStorer:        arguments.BootStorer,
		forkDetector:      arguments.ForkDetector,
		blkExecutor:       arguments.BlockProcessor,
		blkc:              arguments.ChainHandler,
		marshalizer:       arguments.Marshalizer,
		store:             arguments.Store,
		shardCoordinator:  arguments.ShardCoordinator,
		nodesCoordinator:  arguments.NodesCoordinator,
		epochStartTrigger: arguments.EpochStartTrigger,
		blockTracker:      arguments.BlockTracker,

		uint64Converter:     arguments.Uint64Converter,
		bootstrapRoundIndex: arguments.BootstrapRoundIndex,
	}

	boot := shardStorageBootstrapper{
		storageBootstrapper: base,
	}

	base.bootstrapper = &boot
	hdrNonceHashDataUnit := dataRetriever.ShardHdrNonceHashDataUnit + dataRetriever.UnitType(boot.shardCoordinator.SelfId())
	base.headerNonceHashStore = boot.store.GetStorer(hdrNonceHashDataUnit)

	return &boot, nil
}

// LoadFromStorage will load all blocks from storage
func (ssb *shardStorageBootstrapper) LoadFromStorage() error {
	return ssb.loadBlocks()
}

// IsInterfaceNil returns true if there is no value under the interface
func (ssb *shardStorageBootstrapper) IsInterfaceNil() bool {
	return ssb == nil
}

func (ssb *shardStorageBootstrapper) getHeader(hash []byte) (data.HeaderHandler, error) {
	return process.GetShardHeaderFromStorage(hash, ssb.marshalizer, ssb.store)
}

func (ssb *shardStorageBootstrapper) applyCrossNotarizedHeaders(crossNotarizedHeaders []bootstrapStorage.BootstrapHeaderInfo) error {
	for _, crossNotarizedHeader := range crossNotarizedHeaders {
		if crossNotarizedHeader.ShardId != core.MetachainShardId {
			continue
		}

		metaBlock, err := process.GetMetaHeaderFromStorage(crossNotarizedHeader.Hash, ssb.marshalizer, ssb.store)
		if err != nil {
			return err
		}

		log.Debug("added cross notarized header in block tracker",
			"shard", core.MetachainShardId,
			"round", metaBlock.GetRound(),
			"nonce", metaBlock.GetNonce(),
			"hash", crossNotarizedHeader.Hash)

		ssb.blockTracker.AddCrossNotarizedHeader(core.MetachainShardId, metaBlock, crossNotarizedHeader.Hash)
		ssb.blockTracker.AddTrackedHeader(metaBlock, crossNotarizedHeader.Hash)
	}

	return nil
}

func (ssb *shardStorageBootstrapper) cleanupNotarizedStorage(shardHeaderHash []byte) {
	log.Debug("cleanup notarized storage")

	shardHeader, err := process.GetShardHeaderFromStorage(shardHeaderHash, ssb.marshalizer, ssb.store)
	if err != nil {
		log.Debug("shard header is not found in BlockHeaderUnit storage",
			"hash", shardHeaderHash)
		return
	}

	for _, metaBlockHash := range shardHeader.MetaBlockHashes {
		var metaBlock *block.MetaBlock
		metaBlock, err = process.GetMetaHeaderFromStorage(metaBlockHash, ssb.marshalizer, ssb.store)
		if err != nil {
			log.Debug("meta block is not found in MetaBlockUnit storage",
				"hash", metaBlockHash)
			continue
		}

		log.Debug("removing meta block from storage",
			"shardId", metaBlock.GetShardID(),
			"nonce", metaBlock.GetNonce(),
			"hash", metaBlockHash)

		nonceToByteSlice := ssb.uint64Converter.ToByteSlice(metaBlock.GetNonce())
		err = ssb.store.GetStorer(dataRetriever.MetaHdrNonceHashDataUnit).Remove(nonceToByteSlice)
		if err != nil {
			log.Debug("meta block was not removed from MetaHdrNonceHashDataUnit storage",
				"shardId", metaBlock.GetShardID(),
				"nonce", metaBlock.GetNonce(),
				"hash", metaBlockHash,
				"error", err.Error())
		}

		err = ssb.store.GetStorer(dataRetriever.MetaBlockUnit).Remove(metaBlockHash)
		if err != nil {
			log.Debug("meta block was not removed from MetaBlockUnit storage",
				"shardId", metaBlock.GetShardID(),
				"nonce", metaBlock.GetNonce(),
				"hash", metaBlockHash,
				"error", err.Error())
		}
	}
}

func (ssb *shardStorageBootstrapper) applySelfNotarizedHeaders(selfNotarizedHeadersHashes [][]byte) ([]data.HeaderHandler, error) {
	selfNotarizedHeaders := make([]data.HeaderHandler, 0, len(selfNotarizedHeadersHashes))
	for _, selfNotarizedHeaderHash := range selfNotarizedHeadersHashes {
		selfNotarizedHeader, err := ssb.getHeader(selfNotarizedHeaderHash)
		if err != nil {
			return nil, err
		}

		selfNotarizedHeaders = append(selfNotarizedHeaders, selfNotarizedHeader)

		log.Debug("added self notarized header in block tracker",
			"shard", core.MetachainShardId,
			"round", selfNotarizedHeader.GetRound(),
			"nonce", selfNotarizedHeader.GetNonce(),
			"hash", selfNotarizedHeaderHash)

		ssb.blockTracker.AddSelfNotarizedHeader(core.MetachainShardId, selfNotarizedHeader, selfNotarizedHeaderHash)
	}

	return selfNotarizedHeaders, nil
}

func (ssb *shardStorageBootstrapper) applyNumPendingMiniBlocks(_ []bootstrapStorage.PendingMiniBlocksInfo) {
}

func checkShardStorageBootstrapperArgs(args ArgsShardStorageBootstrapper) error {
	return checkBaseStorageBootrstrapperArguments(args.ArgsBaseStorageBootstrapper)
}
