package bootstrap

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go-core/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/epochStart/bootstrap/disabled"
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
	prefsConfig config.PreferencesConfig,
	shardCoordinator sharding.Coordinator,
	pathManagerHandler storage.PathManagerHandler,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	currentEpoch uint32,
	uint64Converter typeConverters.Uint64ByteSliceConverter,
	nodeTypeProvider core.NodeTypeProviderHandler,
) (*shardStorageHandler, error) {
	epochStartNotifier := &disabled.EpochStartNotifier{}
	storageFactory, err := factory.NewStorageServiceFactory(
		&generalConfig,
		&prefsConfig,
		shardCoordinator,
		pathManagerHandler,
		epochStartNotifier,
		nodeTypeProvider,
		currentEpoch,
		false,
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
		uint64Converter:  uint64Converter,
	}

	return &shardStorageHandler{baseStorageHandler: base}, nil
}

// CloseStorageService closes the containing storage service
func (ssh *shardStorageHandler) CloseStorageService() {
	err := ssh.storageService.CloseAll()
	if err != nil {
		log.Warn("error while closing storers", "error", err)
	}
}

// SaveDataToStorage will save the fetched data to storage so it will be used by the storage bootstrap component
func (ssh *shardStorageHandler) SaveDataToStorage(components *ComponentsNeededForBootstrap, notarizedShardHeader data.HeaderHandler, withScheduled bool) error {
	bootStorer := ssh.storageService.GetStorer(dataRetriever.BootstrapUnit)

	lastHeader, err := ssh.saveLastHeader(components.ShardHeader)
	if err != nil {
		return err
	}

	err = ssh.saveEpochStartMetaHdrs(components)
	if err != nil {
		return err
	}

	processedMiniBlocks, pendingMiniBlocks, err := ssh.getProcessedAndPendingMiniBlocksWithScheduled(components.EpochStartMetaBlock, components.Headers, notarizedShardHeader, withScheduled)
	if err != nil {
		return err
	}

	triggerConfigKey, err := ssh.saveTriggerRegistry(components)
	if err != nil {
		return err
	}

	components.NodesConfig.CurrentEpoch = components.ShardHeader.GetEpoch()
	nodesCoordinatorConfigKey, err := ssh.saveNodesCoordinatorRegistry(components.EpochStartMetaBlock, components.NodesConfig)
	if err != nil {
		return err
	}

	lastCrossNotarizedHdrs, err := ssh.saveLastCrossNotarizedHeaders(components.EpochStartMetaBlock, components.Headers, withScheduled)
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

	roundToUseAsKey := int64(components.ShardHeader.GetRound())
	roundNum := bootstrapStorage.RoundNum{Num: roundToUseAsKey}
	roundNumBytes, err := ssh.marshalizer.Marshal(&roundNum)
	if err != nil {
		return err
	}

	err = bootStorer.Put([]byte(common.HighestRoundFromBootStorage), roundNumBytes)
	if err != nil {
		return err
	}

	log.Info("saved bootstrap data to storage", "round", roundToUseAsKey)
	key := []byte(strconv.FormatInt(roundToUseAsKey, 10))
	err = bootStorer.Put(key, bootStrapDataBytes)
	if err != nil {
		return err
	}

	return nil
}

func (ssh *shardStorageHandler) saveEpochStartMetaHdrs(components *ComponentsNeededForBootstrap) error {
	err := ssh.saveMetaHdrForEpochTrigger(components.EpochStartMetaBlock)
	if err != nil {
		return err
	}

	err = ssh.saveMetaHdrForEpochTrigger(components.PreviousEpochStart)
	if err != nil {
		return err
	}

	return nil
}

func getEpochStartShardData(metaBlock data.MetaHeaderHandler, shardId uint32) (data.EpochStartShardDataHandler, error) {
	for _, epochStartShardData := range metaBlock.GetEpochStartHandler().GetLastFinalizedHeaderHandlers() {
		if epochStartShardData.GetShardID() == shardId {
			return epochStartShardData, nil
		}
	}

	return &block.EpochStartShardData{}, epochStart.ErrEpochStartDataForShardNotFound
}

func (ssh *shardStorageHandler) getCrossProcessedMiniBlockHeadersDestMe(shardHeader data.ShardHeaderHandler) map[string]data.MiniBlockHeaderHandler {
	crossMbsProcessed := make(map[string]data.MiniBlockHeaderHandler)
	processedMiniBlockHeaders := shardHeader.GetMiniBlockHeaderHandlers()
	ownShardID := shardHeader.GetShardID()

	for index, mbHeader := range processedMiniBlockHeaders {
		if mbHeader.GetReceiverShardID() != ownShardID {
			continue
		}
		if mbHeader.GetSenderShardID() == ownShardID {
			continue
		}

		crossMbsProcessed[string(mbHeader.GetHash())] = processedMiniBlockHeaders[index]
	}

	return crossMbsProcessed
}

func getProcessedMiniBlocksForFinishedMeta(
	referencedMetaBlockHashes [][]byte,
	headers map[string]data.HeaderHandler,
	selfShardID uint32,
) ([]bootstrapStorage.MiniBlocksInMeta, error) {

	processedMiniBlocks := make([]bootstrapStorage.MiniBlocksInMeta, 0)

	for i := 0; i < len(referencedMetaBlockHashes)-1; i++ {
		header, ok := headers[string(referencedMetaBlockHashes[i])]
		if !ok {
			return nil, fmt.Errorf("%w in getProcessedMiniBlocksForFinishedMeta: hash: %s",
				epochStart.ErrMissingHeader,
				hex.EncodeToString(referencedMetaBlockHashes[i]))
		}

		neededMeta, ok := header.(*block.MetaBlock)
		if !ok {
			return nil, epochStart.ErrWrongTypeAssertion
		}
		if check.IfNil(neededMeta) {
			return nil, epochStart.ErrNilMetaBlock
		}

		log.Debug("getProcessedMiniBlocksForFinishedMeta", "meta block hash", referencedMetaBlockHashes[i])

		miniBlockHashes := make([][]byte, 0)
		isFullyProcessed := make([]bool, 0)
		indexOfLastTxProcessed := make([]int32, 0)

		miniBlockHeadersDestMe := getMiniBlockHeadersForDest(neededMeta, selfShardID)
		for mbHash, mbHeader := range miniBlockHeadersDestMe {
			log.Debug("getProcessedMiniBlocksForFinishedMeta", "mb hash", mbHash)

			miniBlockHashes = append(miniBlockHashes, []byte(mbHash))
			isFullyProcessed = append(isFullyProcessed, mbHeader.IsFinal())
			indexOfLastTxProcessed = append(indexOfLastTxProcessed, mbHeader.GetIndexOfLastTxProcessed())
		}

		if len(miniBlockHashes) > 0 {
			processedMiniBlocks = append(processedMiniBlocks, bootstrapStorage.MiniBlocksInMeta{
				MetaHash:               referencedMetaBlockHashes[i],
				MiniBlocksHashes:       miniBlockHashes,
				IsFullyProcessed:       isFullyProcessed,
				IndexOfLastTxProcessed: indexOfLastTxProcessed,
			})
		}
	}

	return processedMiniBlocks, nil
}

func (ssh *shardStorageHandler) getProcessedAndPendingMiniBlocksWithScheduled(
	meta data.MetaHeaderHandler,
	headers map[string]data.HeaderHandler,
	header data.HeaderHandler,
	withScheduled bool,
) ([]bootstrapStorage.MiniBlocksInMeta, []bootstrapStorage.PendingMiniBlocksInfo, error) {
	log.Debug("getProcessedAndPendingMiniBlocksWithScheduled", "withScheduled", withScheduled)
	processedMiniBlocks, pendingMiniBlocks, firstPendingMetaBlockHash, err := ssh.getProcessedAndPendingMiniBlocks(meta, headers)
	if err != nil {
		return nil, nil, err
	}

	log.Debug("getProcessedAndPendingMiniBlocksWithScheduled: initial processed and pending for scheduled")
	printProcessedAndPendingMiniBlocks(processedMiniBlocks, pendingMiniBlocks)

	if !withScheduled {
		return processedMiniBlocks, pendingMiniBlocks, nil
	}

	shardHeader, ok := header.(data.ShardHeaderHandler)
	if !ok {
		return nil, nil, epochStart.ErrWrongTypeAssertion
	}

	mapHashMiniBlockHeaders := ssh.getCrossProcessedMiniBlockHeadersDestMe(shardHeader)

	referencedMetaBlocks := shardHeader.GetMetaBlockHashes()
	if len(referencedMetaBlocks) == 0 {
		referencedMetaBlocks = append(referencedMetaBlocks, firstPendingMetaBlockHash)
	}

	processedMiniBlockForFinishedMeta, err := getProcessedMiniBlocksForFinishedMeta(referencedMetaBlocks, headers, ssh.shardCoordinator.SelfId())
	if err != nil {
		return nil, nil, err
	}

	processedMiniBlocks = append(processedMiniBlockForFinishedMeta, processedMiniBlocks...)
	processedMiniBlocks, err = updateProcessedMiniBlocksForScheduled(processedMiniBlocks, mapHashMiniBlockHeaders)
	if err != nil {
		return nil, nil, err
	}

	pendingMiniBlocks = addMiniBlocksToPending(pendingMiniBlocks, mapHashMiniBlockHeaders)
	pendingMiniBlocks, err = updatePendingMiniBlocksForScheduled(referencedMetaBlocks, pendingMiniBlocks, headers, ssh.shardCoordinator.SelfId())
	if err != nil {
		return nil, nil, err
	}

	log.Debug("getProcessedAndPendingMiniBlocksWithScheduled: updated processed and pending for scheduled")
	printProcessedAndPendingMiniBlocks(processedMiniBlocks, pendingMiniBlocks)

	return processedMiniBlocks, pendingMiniBlocks, nil
}

func getPendingMiniBlocksHashes(pendingMbsInfo []bootstrapStorage.PendingMiniBlocksInfo) [][]byte {
	pendingMbHashes := make([][]byte, 0)
	for _, pendingMbInfo := range pendingMbsInfo {
		pendingMbHashes = append(pendingMbHashes, pendingMbInfo.MiniBlocksHashes...)
	}

	return pendingMbHashes
}

func updateProcessedMiniBlocksForScheduled(
	processedMiniBlocks []bootstrapStorage.MiniBlocksInMeta,
	mapHashMiniBlockHeaders map[string]data.MiniBlockHeaderHandler,
) ([]bootstrapStorage.MiniBlocksInMeta, error) {

	remainingProcessedMiniBlocks := make([]bootstrapStorage.MiniBlocksInMeta, 0)

	for _, miniBlocksInMeta := range processedMiniBlocks {
		miniBlockHashes := make([][]byte, 0)
		isFullyProcessed := make([]bool, 0)
		indexOfLastTxProcessed := make([]int32, 0)

		for index := range miniBlocksInMeta.MiniBlocksHashes {
			mbHash := miniBlocksInMeta.MiniBlocksHashes[index]
			mbHeader, ok := mapHashMiniBlockHeaders[string(mbHash)]
			if !ok {
				miniBlockHashes = append(miniBlockHashes, mbHash)
				isFullyProcessed = append(isFullyProcessed, miniBlocksInMeta.IsFullyProcessed[index])
				indexOfLastTxProcessed = append(indexOfLastTxProcessed, miniBlocksInMeta.IndexOfLastTxProcessed[index])
				continue
			}

			indexOfFirstTxProcessed := mbHeader.GetIndexOfFirstTxProcessed()
			if indexOfFirstTxProcessed > 0 {
				miniBlockHashes = append(miniBlockHashes, mbHash)
				isFullyProcessed = append(isFullyProcessed, false)
				indexOfLastTxProcessed = append(indexOfLastTxProcessed, indexOfFirstTxProcessed-1)
			}
		}

		if len(miniBlockHashes) > 0 {
			remainingProcessedMiniBlocks = append(remainingProcessedMiniBlocks, bootstrapStorage.MiniBlocksInMeta{
				MetaHash:               miniBlocksInMeta.MetaHash,
				MiniBlocksHashes:       miniBlockHashes,
				IsFullyProcessed:       isFullyProcessed,
				IndexOfLastTxProcessed: indexOfLastTxProcessed,
			})
		}
	}

	return remainingProcessedMiniBlocks, nil
}

func updatePendingMiniBlocksForScheduled(
	referencedMetaBlockHashes [][]byte,
	pendingMiniBlocks []bootstrapStorage.PendingMiniBlocksInfo,
	headers map[string]data.HeaderHandler,
	selfShardID uint32,
) ([]bootstrapStorage.PendingMiniBlocksInfo, error) {
	remainingPendingMiniBlocks := make([]bootstrapStorage.PendingMiniBlocksInfo, 0)
	for index, metaBlockHash := range referencedMetaBlockHashes {
		if index == 0 {
			//TODO: There could be situations when even first meta block referenced in one shard block was started and finalized there
			//and the pending mini blocks could be removed at all with the code below, as the roll back will go before this meta block
			//Anyway, even if they will remain as pending here, this is not critical yet, as they count only for isShardStuck analysis
			continue
		}
		mbHashes, err := getProcessedMiniBlockHashesForMetaBlockHash(selfShardID, metaBlockHash, headers)
		if err != nil {
			return nil, err
		}

		if len(mbHashes) > 0 {
			for i := range pendingMiniBlocks {
				pendingMiniBlocks[i].MiniBlocksHashes = removeHashes(pendingMiniBlocks[i].MiniBlocksHashes, mbHashes)
			}
		}
	}

	for index := range pendingMiniBlocks {
		if len(pendingMiniBlocks[index].MiniBlocksHashes) > 0 {
			remainingPendingMiniBlocks = append(remainingPendingMiniBlocks, pendingMiniBlocks[index])
		}
	}

	return remainingPendingMiniBlocks, nil
}

func getProcessedMiniBlockHashesForMetaBlockHash(
	selfShardID uint32,
	metaBlockHash []byte,
	headers map[string]data.HeaderHandler,
) ([][]byte, error) {
	noPendingMbs := make(map[string]struct{})
	metaHeaderHandler, ok := headers[string(metaBlockHash)]
	if !ok {
		return nil, fmt.Errorf("%w in getProcessedMiniBlockHashesForMetaBlockHash: hash: %s",
			epochStart.ErrMissingHeader,
			hex.EncodeToString(metaBlockHash))
	}
	neededMeta, ok := metaHeaderHandler.(*block.MetaBlock)
	if !ok {
		return nil, epochStart.ErrWrongTypeAssertion
	}
	mbHeaders := getProcessedMiniBlockHeaders(neededMeta, selfShardID, noPendingMbs)
	mbHashes := make([][]byte, 0)
	for mbHash := range mbHeaders {
		mbHashes = append(mbHashes, []byte(mbHash))
	}

	return mbHashes, nil
}

func removeHashes(hashes [][]byte, hashesToRemove [][]byte) [][]byte {
	resultedHashes := hashes
	for _, hashToRemove := range hashesToRemove {
		resultedHashes = removeHash(resultedHashes, hashToRemove)
	}
	return resultedHashes
}

func removeHash(hashes [][]byte, hashToRemove []byte) [][]byte {
	result := make([][]byte, 0)
	for i, hash := range hashes {
		if bytes.Equal(hash, hashToRemove) {
			result = append(result, hashes[:i]...)
			result = append(result, hashes[i+1:]...)
			return result
		}
	}

	return append(result, hashes...)
}

func printProcessedAndPendingMiniBlocks(processedMiniBlocks []bootstrapStorage.MiniBlocksInMeta, pendingMiniBlocks []bootstrapStorage.PendingMiniBlocksInfo) {
	for _, miniBlocksInMeta := range processedMiniBlocks {
		log.Debug("processed meta block", "hash", miniBlocksInMeta.MetaHash)
		for index, mbHash := range miniBlocksInMeta.MiniBlocksHashes {
			isFullyProcessed := true
			if miniBlocksInMeta.IsFullyProcessed != nil && index < len(miniBlocksInMeta.IsFullyProcessed) {
				isFullyProcessed = miniBlocksInMeta.IsFullyProcessed[index]
			}

			indexOfLastTxProcessed := int32(math.MaxInt32 - 1)
			if miniBlocksInMeta.IndexOfLastTxProcessed != nil && index < len(miniBlocksInMeta.IndexOfLastTxProcessed) {
				indexOfLastTxProcessed = miniBlocksInMeta.IndexOfLastTxProcessed[index]
			}

			log.Debug("processedMiniBlock", "hash", mbHash,
				"index of last tx processed", indexOfLastTxProcessed,
				"is fully processed", isFullyProcessed)
		}
	}

	for _, pendingMbsInShard := range pendingMiniBlocks {
		log.Debug("shard", "shardID", pendingMbsInShard.ShardID)
		for _, mbHash := range pendingMbsInShard.MiniBlocksHashes {
			log.Debug("pendingMiniBlock", "hash", mbHash)
		}
	}
}

func addMiniBlockToPendingList(
	mbHeader data.MiniBlockHeaderHandler,
	pendingMiniBlocks []bootstrapStorage.PendingMiniBlocksInfo,
) []bootstrapStorage.PendingMiniBlocksInfo {
	for i := range pendingMiniBlocks {
		if pendingMiniBlocks[i].ShardID == mbHeader.GetReceiverShardID() {
			for _, mbHash := range pendingMiniBlocks[i].MiniBlocksHashes {
				if bytes.Equal(mbHash, mbHeader.GetHash()) {
					return pendingMiniBlocks
				}
			}

			pendingMiniBlocks[i].MiniBlocksHashes = append(pendingMiniBlocks[i].MiniBlocksHashes, mbHeader.GetHash())
			return pendingMiniBlocks
		}
	}

	pendingMbInfo := bootstrapStorage.PendingMiniBlocksInfo{
		ShardID:          mbHeader.GetReceiverShardID(),
		MiniBlocksHashes: [][]byte{mbHeader.GetHash()},
	}

	pendingMiniBlocks = append(pendingMiniBlocks, pendingMbInfo)

	return pendingMiniBlocks
}

func addMiniBlocksToPending(
	pendingMiniBlocks []bootstrapStorage.PendingMiniBlocksInfo,
	mapHashMiniBlockHeaders map[string]data.MiniBlockHeaderHandler,
) []bootstrapStorage.PendingMiniBlocksInfo {
	for _, miniBlockHeader := range mapHashMiniBlockHeaders {
		pendingMiniBlocks = addMiniBlockToPendingList(miniBlockHeader, pendingMiniBlocks)
	}

	return pendingMiniBlocks
}

func (ssh *shardStorageHandler) getProcessedAndPendingMiniBlocks(
	meta data.MetaHeaderHandler,
	headers map[string]data.HeaderHandler,
) ([]bootstrapStorage.MiniBlocksInMeta, []bootstrapStorage.PendingMiniBlocksInfo, []byte, error) {

	epochShardData, err := getEpochStartShardData(meta, ssh.shardCoordinator.SelfId())
	if err != nil {
		return nil, nil, nil, err
	}

	header, ok := headers[string(epochShardData.GetFirstPendingMetaBlock())]
	if !ok {
		return nil, nil, nil, fmt.Errorf("%w in getProcessedAndPendingMiniBlocks: hash: %s",
			epochStart.ErrMissingHeader,
			hex.EncodeToString(epochShardData.GetFirstPendingMetaBlock()))
	}

	neededMeta, ok := header.(*block.MetaBlock)
	if !ok {
		return nil, nil, nil, epochStart.ErrWrongTypeAssertion
	}
	if check.IfNil(neededMeta) {
		return nil, nil, nil, epochStart.ErrNilMetaBlock
	}

	miniBlockHashes := make([][]byte, 0)
	isFullyProcessed := make([]bool, 0)
	indexOfLastTxProcessed := make([]int32, 0)
	pendingMiniBlocksMap := make(map[string]struct{})
	pendingMiniBlocksPerShardMap := make(map[uint32][][]byte)

	for _, mbHeader := range epochShardData.GetPendingMiniBlockHeaderHandlers() {
		log.Debug("shardStorageHandler.getProcessedAndPendingMiniBlocks: epochShardData.GetPendingMiniBlockHeaderHandlers",
			"mb hash", mbHeader.GetHash(),
			"len(reserved)", len(mbHeader.GetReserved()),
			"index of first tx processed", mbHeader.GetIndexOfFirstTxProcessed(),
			"index of last tx processed", mbHeader.GetIndexOfLastTxProcessed(),
		)

		receiverShardID := mbHeader.GetReceiverShardID()
		pendingMiniBlocksPerShardMap[receiverShardID] = append(pendingMiniBlocksPerShardMap[receiverShardID], mbHeader.GetHash())
		pendingMiniBlocksMap[string(mbHeader.GetHash())] = struct{}{}

		if mbHeader.GetIndexOfLastTxProcessed() > -1 {
			miniBlockHashes = append(miniBlockHashes, mbHeader.GetHash())
			isFullyProcessed = append(isFullyProcessed, false)
			indexOfLastTxProcessed = append(indexOfLastTxProcessed, mbHeader.GetIndexOfLastTxProcessed())
		}
	}

	miniBlockHeaders := getProcessedMiniBlockHeaders(neededMeta, ssh.shardCoordinator.SelfId(), pendingMiniBlocksMap)
	for mbHash, mbHeader := range miniBlockHeaders {
		log.Debug("shardStorageHandler.getProcessedAndPendingMiniBlocks: miniBlockHeaders",
			"mb hash", mbHeader.GetHash(),
			"len(reserved)", len(mbHeader.GetReserved()),
			"index of first tx processed", mbHeader.GetIndexOfFirstTxProcessed(),
			"index of last tx processed", mbHeader.GetIndexOfLastTxProcessed(),
		)

		miniBlockHashes = append(miniBlockHashes, []byte(mbHash))
		isFullyProcessed = append(isFullyProcessed, mbHeader.IsFinal())
		indexOfLastTxProcessed = append(indexOfLastTxProcessed, mbHeader.GetIndexOfLastTxProcessed())
	}

	processedMiniBlocks := make([]bootstrapStorage.MiniBlocksInMeta, 0)
	if len(miniBlockHashes) > 0 {
		processedMiniBlocks = append(processedMiniBlocks, bootstrapStorage.MiniBlocksInMeta{
			MetaHash:               epochShardData.GetFirstPendingMetaBlock(),
			MiniBlocksHashes:       miniBlockHashes,
			IsFullyProcessed:       isFullyProcessed,
			IndexOfLastTxProcessed: indexOfLastTxProcessed,
		})
	}

	pendingMiniBlocks := make([]bootstrapStorage.PendingMiniBlocksInfo, 0)
	for receiverShardID, mbHashes := range pendingMiniBlocksPerShardMap {
		pendingMiniBlocks = append(pendingMiniBlocks, bootstrapStorage.PendingMiniBlocksInfo{
			ShardID:          receiverShardID,
			MiniBlocksHashes: mbHashes,
		})
	}

	return processedMiniBlocks, pendingMiniBlocks, epochShardData.GetFirstPendingMetaBlock(), nil
}

func getProcessedMiniBlockHeaders(metaBlock *block.MetaBlock, destShardID uint32, pendingMBsMap map[string]struct{}) map[string]block.MiniBlockHeader {
	processedMiniBlockHeaders := make(map[string]block.MiniBlockHeader)
	miniBlockHeadersDestMe := getMiniBlockHeadersForDest(metaBlock, destShardID)
	for hash, mbh := range miniBlockHeadersDestMe {
		if _, hashExists := pendingMBsMap[hash]; hashExists {
			continue
		}

		processedMiniBlockHeaders[hash] = mbh
	}
	return processedMiniBlockHeaders
}

func (ssh *shardStorageHandler) saveLastCrossNotarizedHeaders(
	meta data.MetaHeaderHandler,
	headers map[string]data.HeaderHandler,
	withScheduled bool,
) ([]bootstrapStorage.BootstrapHeaderInfo, error) {
	log.Debug("saveLastCrossNotarizedHeaders", "withScheduled", withScheduled)

	shardData, err := getEpochStartShardData(meta, ssh.shardCoordinator.SelfId())
	if err != nil {
		return nil, err
	}

	lastCrossMetaHdrHash := shardData.GetLastFinishedMetaBlock()
	if len(shardData.GetPendingMiniBlockHeaderHandlers()) == 0 {
		log.Debug("saveLastCrossNotarizedHeaders changing lastCrossMetaHdrHash", "initial hash", lastCrossMetaHdrHash, "final hash", shardData.GetFirstPendingMetaBlock())
		lastCrossMetaHdrHash = shardData.GetFirstPendingMetaBlock()
	}

	if withScheduled {
		log.Debug("saveLastCrossNotarizedHeaders", "lastCrossMetaHdrHash before update", lastCrossMetaHdrHash)
		lastCrossMetaHdrHash, err = updateLastCrossMetaHdrHashIfNeeded(headers, shardData, lastCrossMetaHdrHash)
		if err != nil {
			return nil, err
		}
		log.Debug("saveLastCrossNotarizedHeaders", "lastCrossMetaHdrHash after update", lastCrossMetaHdrHash)
	}

	neededHdr, ok := headers[string(lastCrossMetaHdrHash)]
	if !ok {
		return nil, fmt.Errorf("%w in saveLastCrossNotarizedHeaders: hash: %s",
			epochStart.ErrMissingHeader,
			hex.EncodeToString(lastCrossMetaHdrHash))
	}

	neededMeta, ok := neededHdr.(*block.MetaBlock)
	if !ok {
		return nil, epochStart.ErrWrongTypeAssertion
	}

	_, err = ssh.saveMetaHdrToStorage(neededMeta)
	if err != nil {
		return nil, err
	}

	crossNotarizedHdrs := make([]bootstrapStorage.BootstrapHeaderInfo, 0)
	crossNotarizedHdrs = append(crossNotarizedHdrs, bootstrapStorage.BootstrapHeaderInfo{
		ShardId: core.MetachainShardId,
		Nonce:   neededMeta.GetNonce(),
		Hash:    lastCrossMetaHdrHash,
	})

	return crossNotarizedHdrs, nil
}

func updateLastCrossMetaHdrHashIfNeeded(
	headers map[string]data.HeaderHandler,
	shardData data.EpochStartShardDataHandler,
	lastCrossMetaHdrHash []byte,
) ([]byte, error) {
	_, metaBlockHashes, err := getShardHeaderAndMetaHashes(headers, shardData.GetHeaderHash())
	if err != nil {
		return nil, err
	}
	if len(metaBlockHashes) == 0 {
		return lastCrossMetaHdrHash, nil
	}

	metaHdr, found := headers[string(metaBlockHashes[0])]
	if !found {
		return nil, fmt.Errorf("%w in updateLastCrossMetaHdrHashIfNeeded: hash: %s",
			epochStart.ErrMissingHeader,
			hex.EncodeToString(metaBlockHashes[0]))
	}

	lastCrossMetaHdrHash = metaHdr.GetPrevHash()

	return lastCrossMetaHdrHash, nil
}

func getShardHeaderAndMetaHashes(headers map[string]data.HeaderHandler, headerHash []byte) (data.ShardHeaderHandler, [][]byte, error) {
	header, ok := headers[string(headerHash)]
	if !ok {
		return nil, nil, fmt.Errorf("%w in getShardHeaderAndMetaHashes: hash: %s",
			epochStart.ErrMissingHeader,
			hex.EncodeToString(headerHash))
	}

	shardHeader, ok := header.(data.ShardHeaderHandler)
	if !ok {
		return nil, nil, epochStart.ErrWrongTypeAssertion
	}

	metaBlockHashes := shardHeader.GetMetaBlockHashes()

	return shardHeader, metaBlockHashes, nil
}

func (ssh *shardStorageHandler) saveLastHeader(shardHeader data.HeaderHandler) (bootstrapStorage.BootstrapHeaderInfo, error) {
	lastHeaderHash, err := ssh.saveShardHdrToStorage(shardHeader)
	if err != nil {
		return bootstrapStorage.BootstrapHeaderInfo{}, err
	}

	bootstrapHdrInfo := bootstrapStorage.BootstrapHeaderInfo{
		ShardId: shardHeader.GetShardID(),
		Epoch:   shardHeader.GetEpoch(),
		Nonce:   shardHeader.GetNonce(),
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

	triggerReg := block.ShardTriggerRegistry{
		Epoch:                       shardHeader.GetEpoch(),
		MetaEpoch:                   metaBlock.GetEpoch(),
		CurrentRoundIndex:           int64(shardHeader.GetRound()),
		EpochStartRound:             shardHeader.GetRound(),
		EpochMetaBlockHash:          metaBlockHash,
		IsEpochStart:                true,
		NewEpochHeaderReceived:      true,
		EpochFinalityAttestingRound: 0,
		EpochStartShardHeader:       &block.Header{},
	}

	bootstrapKey := []byte(fmt.Sprint(shardHeader.GetRound()))
	trigInternalKey := append([]byte(common.TriggerRegistryKeyPrefix), bootstrapKey...)

	triggerRegBytes, err := ssh.marshalizer.Marshal(&triggerReg)
	if err != nil {
		return nil, err
	}

	errPut := ssh.storageService.GetStorer(dataRetriever.BootstrapUnit).Put(trigInternalKey, triggerRegBytes)
	if errPut != nil {
		return nil, errPut
	}

	return bootstrapKey, nil
}

func getMiniBlockHeadersForDest(metaBlock *block.MetaBlock, destId uint32) map[string]block.MiniBlockHeader {
	hashDst := make(map[string]block.MiniBlockHeader)
	for i := 0; i < len(metaBlock.ShardInfo); i++ {
		if metaBlock.ShardInfo[i].ShardID == destId {
			continue
		}

		for _, val := range metaBlock.ShardInfo[i].ShardMiniBlockHeaders {
			isCrossShardDestMe := val.ReceiverShardID == destId && val.SenderShardID != destId
			if !isCrossShardDestMe {
				continue
			}

			hashDst[string(val.Hash)] = val
		}
	}

	for _, val := range metaBlock.MiniBlockHeaders {
		isCrossShardDestMe := (val.ReceiverShardID == destId || val.ReceiverShardID == core.AllShardId) && val.SenderShardID != destId
		if !isCrossShardDestMe {
			continue
		}
		hashDst[string(val.Hash)] = val
	}

	return hashDst
}

// IsInterfaceNil returns true if there is no value under the interface
func (ssh *shardStorageHandler) IsInterfaceNil() bool {
	return ssh == nil
}
