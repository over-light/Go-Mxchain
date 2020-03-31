package bootstrap

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/epochStart/bootstrap/disabled"
	"github.com/ElrondNetwork/elrond-go/epochStart/metachain"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/factory"
)

type metaStorageHandler struct {
	*baseStorageHandler
}

// NewMetaStorageHandler will return a new instance of metaStorageHandler
func NewMetaStorageHandler(
	generalConfig config.Config,
	shardCoordinator sharding.Coordinator,
	pathManagerHandler storage.PathManagerHandler,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	currentEpoch uint32,
) (*metaStorageHandler, error) {
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

	storageService, err := storageFactory.CreateForMeta()
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

	return &metaStorageHandler{baseStorageHandler: base}, nil
}

// SaveDataToStorage will save the fetched data to storage so it will be used by the storage bootstrap component
func (msh *metaStorageHandler) SaveDataToStorage(components *ComponentsNeededForBootstrap) error {
	defer func() {
		err := msh.storageService.CloseAll()
		if err != nil {
			log.Debug("error while closing storers", "error", err)
		}
	}()

	bootStorer := msh.storageService.GetStorer(dataRetriever.BootstrapUnit)

	lastHeader, err := msh.saveLastHeader(components.EpochStartMetaBlock)
	if err != nil {
		return err
	}

	miniBlocks, err := msh.groupMiniBlocksByShard(components.PendingMiniBlocks)
	if err != nil {
		return err
	}

	triggerConfigKey, err := msh.saveTriggerRegistry(components)
	if err != nil {
		return err
	}

	nodesCoordinatorConfigKey, err := msh.saveNodesCoordinatorRegistry(components.EpochStartMetaBlock, components.NodesConfig)
	if err != nil {
		return err
	}

	lastCrossNotarizedHeader := msh.getLastCrossNotarizedHeaders(components.EpochStartMetaBlock)

	bootStrapData := bootstrapStorage.BootstrapData{
		LastHeader:                 lastHeader,
		LastCrossNotarizedHeaders:  lastCrossNotarizedHeader,
		LastSelfNotarizedHeaders:   []bootstrapStorage.BootstrapHeaderInfo{lastHeader},
		ProcessedMiniBlocks:        nil,
		PendingMiniBlocks:          miniBlocks,
		NodesCoordinatorConfigKey:  nodesCoordinatorConfigKey,
		EpochStartTriggerConfigKey: triggerConfigKey,
		HighestFinalBlockNonce:     lastHeader.Nonce,
		LastRound:                  int64(components.EpochStartMetaBlock.Round),
	}
	bootStrapDataBytes, err := msh.marshalizer.Marshal(&bootStrapData)
	if err != nil {
		return err
	}

	roundToUseAsKey := int64(components.EpochStartMetaBlock.Round + 2)
	roundNum := bootstrapStorage.RoundNum{Num: roundToUseAsKey}
	roundNumBytes, err := msh.marshalizer.Marshal(&roundNum)
	if err != nil {
		return err
	}

	err = bootStorer.Put([]byte(core.HighestRoundFromBootStorage), roundNumBytes)
	if err != nil {
		return err
	}
	key := []byte(strconv.FormatInt(roundToUseAsKey, 10))
	err = bootStorer.Put(key, bootStrapDataBytes)
	if err != nil {
		return err
	}

	err = msh.commitTries(components)
	if err != nil {
		return err
	}

	log.Info("saved bootstrap data to storage")
	return nil
}

func (msh *metaStorageHandler) getLastCrossNotarizedHeaders(meta *block.MetaBlock) []bootstrapStorage.BootstrapHeaderInfo {
	crossNotarizedHdrs := make([]bootstrapStorage.BootstrapHeaderInfo, 0)
	for _, epochStartShardData := range meta.EpochStart.LastFinalizedHeaders {
		crossNotarizedHdrs = append(crossNotarizedHdrs, bootstrapStorage.BootstrapHeaderInfo{
			ShardId: epochStartShardData.ShardID,
			Nonce:   epochStartShardData.Nonce,
			Hash:    epochStartShardData.HeaderHash,
		})
	}

	return crossNotarizedHdrs
}

func (msh *metaStorageHandler) saveLastHeader(metaBlock *block.MetaBlock) (bootstrapStorage.BootstrapHeaderInfo, error) {
	lastHeaderHash, err := core.CalculateHash(msh.marshalizer, msh.hasher, metaBlock)
	if err != nil {
		return bootstrapStorage.BootstrapHeaderInfo{}, err
	}

	lastHeaderBytes, err := msh.marshalizer.Marshal(metaBlock)
	if err != nil {
		return bootstrapStorage.BootstrapHeaderInfo{}, err
	}

	err = msh.storageService.GetStorer(dataRetriever.MetaBlockUnit).Put(lastHeaderHash, lastHeaderBytes)
	if err != nil {
		return bootstrapStorage.BootstrapHeaderInfo{}, err
	}

	bootstrapHdrInfo := bootstrapStorage.BootstrapHeaderInfo{
		ShardId: core.MetachainShardId,
		Epoch:   metaBlock.Epoch,
		Nonce:   metaBlock.Nonce,
		Hash:    lastHeaderHash,
	}

	return bootstrapHdrInfo, nil
}

func (msh *metaStorageHandler) saveTriggerRegistry(components *ComponentsNeededForBootstrap) ([]byte, error) {
	metaBlock := components.EpochStartMetaBlock
	hash, err := core.CalculateHash(msh.marshalizer, msh.hasher, metaBlock)
	if err != nil {
		return nil, err
	}

	triggerReg := metachain.TriggerRegistry{
		Epoch:                       metaBlock.Epoch,
		CurrentRound:                metaBlock.Round,
		EpochFinalityAttestingRound: metaBlock.Round,
		CurrEpochStartRound:         metaBlock.Round,
		PrevEpochStartRound:         components.PreviousEpochStartRound,
		EpochStartMetaHash:          hash,
		EpochStartMeta:              metaBlock,
	}

	trigInternalKey := append([]byte(core.TriggerRegistryKeyPrefix), []byte(fmt.Sprint(metaBlock.Round))...)

	triggerRegBytes, err := json.Marshal(&triggerReg)
	if err != nil {
		return nil, err
	}

	errPut := msh.storageService.GetStorer(dataRetriever.BootstrapUnit).Put(trigInternalKey, triggerRegBytes)
	if errPut != nil {
		return nil, errPut
	}

	return []byte(core.TriggerRegistryKeyPrefix), nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (msh *metaStorageHandler) IsInterfaceNil() bool {
	return msh == nil
}
