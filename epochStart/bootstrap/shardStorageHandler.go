package bootstrap

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/epochStart/bootstrap/disabled"
	"github.com/ElrondNetwork/elrond-go/epochStart/shardchain"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/factory"
)

type shardStorageHandler struct {
	*baseStorageHandler
}

// NewShardStorageHandler will return a new instance of shardStorageHandler
func NewShardStorageHandler(
	generalConfig config.Config,
	shardCoordinator sharding.Coordinator,
	pathManagerHandler storage.PathManagerHandler,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	currentEpoch uint32,
) (*shardStorageHandler, error) {
	epochStartNotifier := &disabled.EpochStartNotifier{}
	storageFactory, err := factory.NewStorageServiceFactory(
		&generalConfig,
		shardCoordinator,
		pathManagerHandler,
		epochStartNotifier,
		currentEpoch,
	)
	if err != nil {
		return nil, err
	}

	storageService, err := storageFactory.CreateForShard()
	if err != nil {
		return nil, err
	}

	base := &baseStorageHandler{
		storageService:   storageService,
		shardCoordinator: shardCoordinator,
		marshalizer:      marshalizer,
		hasher:           hasher,
		currentEpoch:     currentEpoch,
	}

	return &shardStorageHandler{baseStorageHandler: base}, nil
}

// SaveDataToStorage will save the fetched data to storage so it will be used by the storage bootstrap component
func (ssh *shardStorageHandler) SaveDataToStorage(components *ComponentsNeededForBootstrap) error {
	defer func() {
		err := ssh.storageService.CloseAll()
		if err != nil {
			log.Warn("error while closing storers", "error", err)
		}
	}()

	bootStorer := ssh.storageService.GetStorer(dataRetriever.BootstrapUnit)

	lastHeader, err := ssh.saveLastHeader(components.ShardHeader)
	if err != nil {
		return err
	}

	processedMiniBlocks, err := ssh.getProcessedMiniBlocks(components.PendingMiniBlocks, components.EpochStartMetaBlock, components.Headers)
	if err != nil {
		return err
	}

	pendingMiniBlocks, err := ssh.groupMiniBlocksByShard(components.PendingMiniBlocks)
	if err != nil {
		return err
	}

	triggerConfigKey, err := ssh.saveTriggerRegistry(components)
	if err != nil {
		return err
	}

	nodesCoordinatorConfigKey, err := ssh.saveNodesCoordinatorRegistry(components.EpochStartMetaBlock, components.NodesConfig)
	if err != nil {
		return err
	}

	lastCrossNotarizedHdrs, err := ssh.getLastCrossNotarizedHeaders(components.EpochStartMetaBlock, components.Headers)
	if err != nil {
		return err
	}

	bootStrapData := bootstrapStorage.BootstrapData{
		LastHeader:                 lastHeader,
		LastCrossNotarizedHeaders:  lastCrossNotarizedHdrs,
		LastSelfNotarizedHeaders:   []bootstrapStorage.BootstrapHeaderInfo{lastHeader},
		ProcessedMiniBlocks:        processedMiniBlocks,
		PendingMiniBlocks:          pendingMiniBlocks,
		NodesCoordinatorConfigKey:  nodesCoordinatorConfigKey,
		EpochStartTriggerConfigKey: triggerConfigKey,
		HighestFinalBlockNonce:     lastHeader.Nonce,
		LastRound:                  0,
	}
	bootStrapDataBytes, err := ssh.marshalizer.Marshal(&bootStrapData)
	if err != nil {
		return err
	}

	roundToUseAsKey := int64(components.ShardHeader.Round)
	roundNum := bootstrapStorage.RoundNum{Num: roundToUseAsKey}
	roundNumBytes, err := ssh.marshalizer.Marshal(&roundNum)
	if err != nil {
		return err
	}

	err = bootStorer.Put([]byte(core.HighestRoundFromBootStorage), roundNumBytes)
	if err != nil {
		return err
	}

	log.Info("saved bootstrap data to storage")
	key := []byte(strconv.FormatInt(roundToUseAsKey, 10))
	err = bootStorer.Put(key, bootStrapDataBytes)
	if err != nil {
		return err
	}

	err = ssh.commitTries(components)
	if err != nil {
		return err
	}

	return nil
}

func getEpochStartShardData(metaBlock *block.MetaBlock, shardId uint32) (block.EpochStartShardData, error) {
	for _, epochStartShardData := range metaBlock.EpochStart.LastFinalizedHeaders {
		if epochStartShardData.ShardID == shardId {
			return epochStartShardData, nil
		}
	}

	return block.EpochStartShardData{}, epochStart.ErrEpochStartDataForShardNotFound
}

func (ssh *shardStorageHandler) getProcessedMiniBlocks(
	pendingMiniBlocks map[string]*block.MiniBlock,
	meta *block.MetaBlock,
	headers map[string]data.HeaderHandler,
) ([]bootstrapStorage.MiniBlocksInMeta, error) {
	shardData, err := getEpochStartShardData(meta, ssh.shardCoordinator.SelfId())
	if err != nil {
		return nil, err
	}

	neededMeta, ok := headers[string(shardData.FirstPendingMetaBlock)].(*block.MetaBlock)
	if !ok {
		return nil, epochStart.ErrMissingHeader
	}

	if check.IfNil(neededMeta) {
		return nil, epochStart.ErrMissingHeader
	}

	processedMbHashes := make([][]byte, 0)
	miniBlocksDstMe := getAllMiniBlocksWithDst(neededMeta, ssh.shardCoordinator.SelfId())
	for hash, mb := range miniBlocksDstMe {
		if _, ok := pendingMiniBlocks[hash]; ok {
			continue
		}

		processedMbHashes = append(processedMbHashes, mb.Hash)
	}

	processedMiniBlocks := make([]bootstrapStorage.MiniBlocksInMeta, 0)
	processedMiniBlocks = append(processedMiniBlocks, bootstrapStorage.MiniBlocksInMeta{
		MetaHash:         shardData.FirstPendingMetaBlock,
		MiniBlocksHashes: processedMbHashes,
	})

	return processedMiniBlocks, nil
}

func (ssh *shardStorageHandler) getLastCrossNotarizedHeaders(meta *block.MetaBlock, headers map[string]data.HeaderHandler) ([]bootstrapStorage.BootstrapHeaderInfo, error) {
	shardData, err := getEpochStartShardData(meta, ssh.shardCoordinator.SelfId())
	if err != nil {
		return nil, err
	}

	neededMeta, ok := headers[string(shardData.LastFinishedMetaBlock)]
	if !ok {
		return nil, epochStart.ErrMissingHeader
	}

	crossNotarizedHdrs := make([]bootstrapStorage.BootstrapHeaderInfo, 0)
	crossNotarizedHdrs = append(crossNotarizedHdrs, bootstrapStorage.BootstrapHeaderInfo{
		ShardId: core.MetachainShardId,
		Nonce:   neededMeta.GetNonce(),
		Hash:    shardData.LastFinishedMetaBlock,
	})

	neededMeta, ok = headers[string(shardData.FirstPendingMetaBlock)]
	if !ok {
		return nil, epochStart.ErrMissingHeader
	}

	crossNotarizedHdrs = append(crossNotarizedHdrs, bootstrapStorage.BootstrapHeaderInfo{
		ShardId: core.MetachainShardId,
		Nonce:   neededMeta.GetNonce(),
		Hash:    shardData.FirstPendingMetaBlock,
	})

	return crossNotarizedHdrs, nil
}

func (ssh *shardStorageHandler) saveLastHeader(shardHeader *block.Header) (bootstrapStorage.BootstrapHeaderInfo, error) {
	lastHeaderHash, err := core.CalculateHash(ssh.marshalizer, ssh.hasher, shardHeader)
	if err != nil {
		return bootstrapStorage.BootstrapHeaderInfo{}, err
	}

	lastHeaderBytes, err := ssh.marshalizer.Marshal(shardHeader)
	if err != nil {
		return bootstrapStorage.BootstrapHeaderInfo{}, err
	}

	err = ssh.storageService.GetStorer(dataRetriever.BlockHeaderUnit).Put(lastHeaderHash, lastHeaderBytes)
	if err != nil {
		return bootstrapStorage.BootstrapHeaderInfo{}, err
	}

	bootstrapHdrInfo := bootstrapStorage.BootstrapHeaderInfo{
		ShardId: core.MetachainShardId,
		Epoch:   shardHeader.Epoch,
		Nonce:   shardHeader.Nonce,
		Hash:    lastHeaderHash,
	}

	return bootstrapHdrInfo, nil
}

func (ssh *shardStorageHandler) saveTriggerRegistry(components *ComponentsNeededForBootstrap) ([]byte, error) {
	shardHeader := components.ShardHeader

	metaBlock := components.EpochStartMetaBlock
	metaBlockHash, err := core.CalculateHash(ssh.marshalizer, ssh.hasher, metaBlock)
	if err != nil {
		return nil, err
	}

	triggerReg := shardchain.TriggerRegistry{
		Epoch:                       shardHeader.Epoch,
		CurrentRoundIndex:           int64(shardHeader.Round),
		EpochStartRound:             shardHeader.Round,
		EpochMetaBlockHash:          metaBlockHash,
		IsEpochStart:                false,
		NewEpochHeaderReceived:      false,
		EpochFinalityAttestingRound: 0,
	}

	trigInternalKey := append([]byte(core.TriggerRegistryKeyPrefix), []byte(fmt.Sprint(shardHeader.Round))...)

	triggerRegBytes, err := json.Marshal(&triggerReg)
	if err != nil {
		return nil, err
	}

	errPut := ssh.storageService.GetStorer(dataRetriever.BootstrapUnit).Put(trigInternalKey, triggerRegBytes)
	if errPut != nil {
		return nil, errPut
	}

	return trigInternalKey, nil
}

func getAllMiniBlocksWithDst(metaBlock *block.MetaBlock, destId uint32) map[string]block.ShardMiniBlockHeader {
	hashDst := make(map[string]block.ShardMiniBlockHeader)
	for i := 0; i < len(metaBlock.ShardInfo); i++ {
		if metaBlock.ShardInfo[i].ShardID == destId {
			continue
		}

		for _, val := range metaBlock.ShardInfo[i].ShardMiniBlockHeaders {
			if val.ReceiverShardID == destId && val.SenderShardID != destId {
				hashDst[string(val.Hash)] = val
			}
		}
	}

	for _, val := range metaBlock.MiniBlockHeaders {
		isCrossShardDestMe := val.ReceiverShardID == destId && val.SenderShardID != destId
		if isCrossShardDestMe {
			shardMiniBlockHdr := block.ShardMiniBlockHeader{
				Hash:            val.Hash,
				ReceiverShardID: val.ReceiverShardID,
				SenderShardID:   val.SenderShardID,
				TxCount:         val.TxCount,
			}
			hashDst[string(val.Hash)] = shardMiniBlockHdr
		}
	}

	return hashDst
}

// IsInterfaceNil returns true if there is no value under the interface
func (ssh *shardStorageHandler) IsInterfaceNil() bool {
	return ssh == nil
}
