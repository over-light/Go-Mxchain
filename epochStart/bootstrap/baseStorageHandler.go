package bootstrap

import (
	"encoding/json"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// baseStorageHandler handles the storage functions for saving bootstrap data
type baseStorageHandler struct {
	storageService   dataRetriever.StorageService
	shardCoordinator sharding.Coordinator
	marshalizer      marshal.Marshalizer
	hasher           hashing.Hasher
	currentEpoch     uint32
	uint64Converter  typeConverters.Uint64ByteSliceConverter
}

func (bsh *baseStorageHandler) groupMiniBlocksByShard(miniBlocks map[string]*block.MiniBlock) ([]bootstrapStorage.PendingMiniBlocksInfo, error) {
	pendingMBsMap := make(map[uint32][][]byte)
	for hash, miniBlock := range miniBlocks {
		senderShId := miniBlock.SenderShardID
		pendingMBsMap[senderShId] = append(pendingMBsMap[senderShId], []byte(hash))
	}

	sliceToRet := make([]bootstrapStorage.PendingMiniBlocksInfo, 0)
	for shardID, hashes := range pendingMBsMap {
		sliceToRet = append(sliceToRet, bootstrapStorage.PendingMiniBlocksInfo{
			ShardID:          shardID,
			MiniBlocksHashes: hashes,
		})
	}

	return sliceToRet, nil
}

func (bsh *baseStorageHandler) saveNodesCoordinatorRegistry(
	metaBlock *block.MetaBlock,
	nodesConfig *sharding.NodesCoordinatorRegistry,
) ([]byte, error) {
	key := append([]byte(core.NodesCoordinatorRegistryKeyPrefix), metaBlock.RandSeed...)

	// TODO: replace hardcoded json - although it is hardcoded in nodesCoordinator as well.
	registryBytes, err := json.Marshal(nodesConfig)
	if err != nil {
		return nil, err
	}

	err = bsh.storageService.GetStorer(dataRetriever.BootstrapUnit).Put(key, registryBytes)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func (bsh *baseStorageHandler) commitTries(components *ComponentsNeededForBootstrap) error {
	for _, trie := range components.UserAccountTries {
		err := trie.Commit()
		if err != nil {
			return err
		}
	}

	for _, trie := range components.PeerAccountTries {
		err := trie.Commit()
		if err != nil {
			return err
		}
	}

	return nil
}

func (bsh *baseStorageHandler) saveMetaHdrToStorage(metaBlock *block.MetaBlock) ([]byte, error) {
	headerBytes, err := bsh.marshalizer.Marshal(metaBlock)
	if err != nil {
		return nil, err
	}

	headerHash := bsh.hasher.Compute(string(headerBytes))

	metaHdrStorage := bsh.storageService.GetStorer(dataRetriever.MetaBlockUnit)
	err = metaHdrStorage.Put(headerHash, headerBytes)
	if err != nil {
		return nil, err
	}

	nonceToByteSlice := bsh.uint64Converter.ToByteSlice(metaBlock.GetNonce())
	metaHdrNonceStorage := bsh.storageService.GetStorer(dataRetriever.MetaHdrNonceHashDataUnit)
	err = metaHdrNonceStorage.Put(nonceToByteSlice, headerHash)
	if err != nil {
		return nil, err
	}

	return headerHash, nil
}

func (bsh *baseStorageHandler) saveShardHdrToStorage(hdr data.HeaderHandler) ([]byte, error) {
	headerBytes, err := bsh.marshalizer.Marshal(hdr)
	if err != nil {
		return nil, err
	}

	headerHash := bsh.hasher.Compute(string(headerBytes))

	shardHdrStorage := bsh.storageService.GetStorer(dataRetriever.BlockHeaderUnit)
	err = shardHdrStorage.Put(headerHash, headerBytes)
	if err != nil {
		return nil, err
	}

	nonceToByteSlice := bsh.uint64Converter.ToByteSlice(hdr.GetNonce())
	shardHdrNonceStorage := bsh.storageService.GetStorer(dataRetriever.ShardHdrNonceHashDataUnit + dataRetriever.UnitType(hdr.GetShardID()))
	err = shardHdrNonceStorage.Put(nonceToByteSlice, headerHash)
	if err != nil {
		return nil, err
	}

	return headerHash, nil
}

func (bsh *baseStorageHandler) saveMetaHdrForEpochTrigger(metaBlock *block.MetaBlock) error {
	lastHeaderBytes, err := bsh.marshalizer.Marshal(metaBlock)
	if err != nil {
		return err
	}

	epochStartIdentifier := core.EpochStartIdentifier(metaBlock.Epoch)
	metaHdrStorage := bsh.storageService.GetStorer(dataRetriever.MetaBlockUnit)
	err = metaHdrStorage.Put([]byte(epochStartIdentifier), lastHeaderBytes)
	if err != nil {
		return err
	}

	triggerStorage := bsh.storageService.GetStorer(dataRetriever.BootstrapUnit)
	err = triggerStorage.Put([]byte(epochStartIdentifier), lastHeaderBytes)
	if err != nil {
		return err
	}

	return nil
}
