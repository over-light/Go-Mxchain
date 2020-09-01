//go:generate protoc -I=proto -I=$GOPATH/src -I=$GOPATH/src/github.com/ElrondNetwork/protobuf/protobuf  --gogoslick_out=. miniblockMetadata.proto

package dblookupext

import (
	"fmt"
	"sync"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/container"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/storage"
)

var log = logger.GetOrCreate("core/dblookupext")

// HistoryRepositoryArguments is a structure that stores all components that are needed to a history processor
type HistoryRepositoryArguments struct {
	SelfShardID                 uint32
	MiniblocksMetadataStorer    storage.Storer
	MiniblockHashByTxHashStorer storage.Storer
	EpochByHashStorer           storage.Storer
	Marshalizer                 marshal.Marshalizer
	Hasher                      hashing.Hasher
}

type historyRepository struct {
	selfShardID                uint32
	miniblocksMetadataStorer   storage.Storer
	miniblockHashByTxHashIndex storage.Storer
	epochByHashIndex           *epochByHashIndex
	marshalizer                marshal.Marshalizer
	hasher                     hashing.Hasher

	// These maps temporarily hold notifications of "notarized at source or destination", to deal with unwanted concurrency effects
	// The unwanted concurrency effects could be accentuated by the fast db-replay-validate mechanism.
	pendingNotarizedAtSourceNotifications      *container.MutexMap
	pendingNotarizedAtDestinationNotifications *container.MutexMap
	pendingNotarizedAtBothNotifications        *container.MutexMap

	consumePendingNotificationsMutex sync.Mutex
}

type notarizedNotification struct {
	metaNonce uint64
	metaHash  []byte
}

// NewHistoryRepository will create a new instance of HistoryRepository
func NewHistoryRepository(arguments HistoryRepositoryArguments) (*historyRepository, error) {
	if check.IfNil(arguments.MiniblocksMetadataStorer) {
		return nil, core.ErrNilStore
	}
	if check.IfNil(arguments.MiniblockHashByTxHashStorer) {
		return nil, core.ErrNilStore
	}
	if check.IfNil(arguments.EpochByHashStorer) {
		return nil, core.ErrNilStore
	}
	if check.IfNil(arguments.Marshalizer) {
		return nil, core.ErrNilMarshalizer
	}
	if check.IfNil(arguments.Hasher) {
		return nil, core.ErrNilHasher
	}

	hashToEpochIndex := newHashToEpochIndex(arguments.EpochByHashStorer, arguments.Marshalizer)

	return &historyRepository{
		selfShardID:                           arguments.SelfShardID,
		miniblocksMetadataStorer:              arguments.MiniblocksMetadataStorer,
		marshalizer:                           arguments.Marshalizer,
		hasher:                                arguments.Hasher,
		epochByHashIndex:                      hashToEpochIndex,
		miniblockHashByTxHashIndex:            arguments.MiniblockHashByTxHashStorer,
		pendingNotarizedAtSourceNotifications: container.NewMutexMap(),
		pendingNotarizedAtDestinationNotifications: container.NewMutexMap(),
		pendingNotarizedAtBothNotifications:        container.NewMutexMap(),
	}, nil
}

// RecordBlock records a block
func (hr *historyRepository) RecordBlock(blockHeaderHash []byte, blockHeader data.HeaderHandler, blockBody data.BodyHandler) error {
	body, ok := blockBody.(*block.Body)
	if !ok {
		return errCannotCastToBlockBody
	}

	epoch := blockHeader.GetEpoch()

	err := hr.epochByHashIndex.saveEpochByHash(blockHeaderHash, epoch)
	if err != nil {
		return newErrCannotSaveEpochByHash("block header", blockHeaderHash, err)
	}

	for _, miniblock := range body.MiniBlocks {
		if miniblock.Type == block.PeerBlock {
			continue
		}

		err = hr.recordMiniblock(blockHeaderHash, blockHeader, miniblock, epoch)
		if err != nil {
			continue
		}
	}

	hr.consumePendingNotificationsWithLock()

	return nil
}

func (hr *historyRepository) recordMiniblock(blockHeaderHash []byte, blockHeader data.HeaderHandler, miniblock *block.MiniBlock, epoch uint32) error {
	miniblockHash, err := hr.computeMiniblockHash(miniblock)
	if err != nil {
		return err
	}

	err = hr.epochByHashIndex.saveEpochByHash(miniblockHash, epoch)
	if err != nil {
		return newErrCannotSaveEpochByHash("miniblock", miniblockHash, err)
	}

	miniblockMetadata := &MiniblockMetadata{
		Type:               int32(miniblock.Type),
		Epoch:              epoch,
		HeaderHash:         blockHeaderHash,
		MiniblockHash:      miniblockHash,
		Round:              blockHeader.GetRound(),
		HeaderNonce:        blockHeader.GetNonce(),
		SourceShardID:      miniblock.GetSenderShardID(),
		DestinationShardID: miniblock.GetReceiverShardID(),
	}

	// If we are on metachain and the miniblock is towards us, then we simulate a notarization notification at commit & record time,
	// since there will be no notification via blockTracker.Register(*)NotarizedHeadersHandler anyway.
	selfIsMeta := hr.selfShardID == core.MetachainShardId
	receiverIsMeta := miniblock.GetReceiverShardID() == core.MetachainShardId
	if selfIsMeta && receiverIsMeta {
		hr.pendingNotarizedAtBothNotifications.Set(string(miniblockHash), &notarizedNotification{
			metaNonce: blockHeader.GetNonce(),
			metaHash:  blockHeaderHash,
		})
	}

	err = hr.putMiniblockMetadata(miniblockHash, miniblockMetadata)
	if err != nil {
		return err
	}

	for _, txHash := range miniblock.TxHashes {
		err := hr.miniblockHashByTxHashIndex.Put(txHash, miniblockHash)
		if err != nil {
			log.Warn("miniblockHashByTxHashIndex.putMiniblockByTx()", "txHash", txHash, "err", err)
			continue
		}
	}

	return nil
}

func (hr *historyRepository) computeMiniblockHash(miniblock *block.MiniBlock) ([]byte, error) {
	return core.CalculateHash(hr.marshalizer, hr.hasher, miniblock)
}

// GetMiniblockMetadataByTxHash will return a history transaction for the given hash from storage
func (hr *historyRepository) GetMiniblockMetadataByTxHash(hash []byte) (*MiniblockMetadata, error) {
	miniblockHash, err := hr.miniblockHashByTxHashIndex.Get(hash)
	if err != nil {
		return nil, err
	}

	return hr.getMiniblockMetadataByMiniblockHash(miniblockHash)
}

func (hr *historyRepository) putMiniblockMetadata(hash []byte, metadata *MiniblockMetadata) error {
	metadataBytes, err := hr.marshalizer.Marshal(metadata)
	if err != nil {
		return err
	}

	err = hr.miniblocksMetadataStorer.PutInEpoch(hash, metadataBytes, metadata.Epoch)
	if err != nil {
		return newErrCannotSaveMiniblockMetadata(hash, err)
	}

	return nil
}

func (hr *historyRepository) getMiniblockMetadataByMiniblockHash(hash []byte) (*MiniblockMetadata, error) {
	epoch, err := hr.epochByHashIndex.getEpochByHash(hash)
	if err != nil {
		return nil, err
	}

	metadataBytes, err := hr.miniblocksMetadataStorer.GetFromEpoch(hash, epoch)
	if err != nil {
		return nil, err
	}

	metadata := &MiniblockMetadata{}
	err = hr.marshalizer.Unmarshal(metadata, metadataBytes)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// GetEpochByHash will return epoch for a given hash
// This works for Blocks, Miniblocks
// It doesn't work for transactions (not needed, there we have a static storer for "miniblockHashByTxHashIndex" as well)!
func (hr *historyRepository) GetEpochByHash(hash []byte) (uint32, error) {
	return hr.epochByHashIndex.getEpochByHash(hash)
}

// RegisterToBlockTracker registers the history repository to blockTracker events
func (hr *historyRepository) RegisterToBlockTracker(blockTracker BlockTracker) {
	if check.IfNil(blockTracker) {
		log.Error("RegisterToBlockTracker(): blockTracker is nil")
		return
	}

	blockTracker.RegisterCrossNotarizedHeadersHandler(hr.onNotarizedBlocks)
	blockTracker.RegisterSelfNotarizedHeadersHandler(hr.onNotarizedBlocks)
}

func (hr *historyRepository) onNotarizedBlocks(shardID uint32, headers []data.HeaderHandler, headersHashes [][]byte) {
	log.Trace("onNotarizedBlocks()", "shardID", shardID, "len(headers)", len(headers))

	if shardID != core.MetachainShardId {
		return
	}

	for i, headerHandler := range headers {
		headerHash := headersHashes[i]

		metaBlock, isMetaBlock := headerHandler.(*block.MetaBlock)
		if isMetaBlock {
			for _, shardData := range metaBlock.ShardInfo {
				hr.onNotarizedInMetaBlock(metaBlock.GetNonce(), headerHash, &shardData)
			}
			continue
		}

		header, isHeader := headerHandler.(*block.Header)
		if isHeader {
			hr.onNotarizedInRegularBlockOfMeta(header.GetNonce(), headerHash, header)
			continue
		}

		log.Error("onNotarizedBlocks(): unexpected type of header", "type", fmt.Sprintf("%T", headerHandler))
	}
}

func (hr *historyRepository) onNotarizedInMetaBlock(metaBlockNonce uint64, metaBlockHash []byte, shardData *block.ShardData) {
	for _, miniblockHeader := range shardData.GetShardMiniBlockHeaders() {
		hr.onNotarizedMiniblock(metaBlockNonce, metaBlockHash, shardData.GetShardID(), miniblockHeader)
	}
}

func (hr *historyRepository) onNotarizedInRegularBlockOfMeta(blockOfMetaNonce uint64, blockOfMetaHash []byte, blockHeader *block.Header) {
	for _, miniblockHeader := range blockHeader.GetMiniBlockHeaders() {
		hr.onNotarizedMiniblock(blockOfMetaNonce, blockOfMetaHash, blockHeader.GetShardID(), miniblockHeader)
	}
}

func (hr *historyRepository) onNotarizedMiniblock(metaBlockNonce uint64, metaBlockHash []byte, shardOfContainingBlock uint32, miniblockHeader block.MiniBlockHeader) {
	miniblockHash := miniblockHeader.Hash
	isIntra := miniblockHeader.SenderShardID == miniblockHeader.ReceiverShardID
	notarizedAtSource := miniblockHeader.SenderShardID == shardOfContainingBlock
	isToMeta := miniblockHeader.ReceiverShardID == core.MetachainShardId

	iDontCare := miniblockHeader.SenderShardID != hr.selfShardID && miniblockHeader.ReceiverShardID != hr.selfShardID
	if iDontCare {
		return
	}

	log.Trace("onNotarizedMiniblock()",
		"metaBlockNonce", metaBlockNonce,
		"metaBlockHash", metaBlockHash,
		"shardOfContainingBlock", shardOfContainingBlock,
		"miniblock", miniblockHash,
		"direction", fmt.Sprintf("[%d -> %d]", miniblockHeader.SenderShardID, miniblockHeader.ReceiverShardID),
	)

	if isIntra || isToMeta {
		hr.pendingNotarizedAtBothNotifications.Set(string(miniblockHash), &notarizedNotification{
			metaNonce: metaBlockNonce,
			metaHash:  metaBlockHash,
		})
	} else {
		// Is cross-shard miniblock
		if notarizedAtSource {
			hr.pendingNotarizedAtSourceNotifications.Set(string(miniblockHash), &notarizedNotification{
				metaNonce: metaBlockNonce,
				metaHash:  metaBlockHash,
			})
		} else {
			hr.pendingNotarizedAtDestinationNotifications.Set(string(miniblockHash), &notarizedNotification{
				metaNonce: metaBlockNonce,
				metaHash:  metaBlockHash,
			})
		}
	}

	hr.consumePendingNotificationsWithLock()
}

func (hr *historyRepository) consumePendingNotificationsWithLock() {
	hr.consumePendingNotificationsMutex.Lock()
	defer hr.consumePendingNotificationsMutex.Unlock()

	log.Debug("consumePendingNotificationsWithLock() begin",
		"len(source)", hr.pendingNotarizedAtSourceNotifications.Len(),
		"len(destination)", hr.pendingNotarizedAtDestinationNotifications.Len(),
		"len(both)", hr.pendingNotarizedAtBothNotifications.Len(),
	)

	hr.consumePendingNotificationsNoLock(hr.pendingNotarizedAtSourceNotifications, func(metadata *MiniblockMetadata, notification *notarizedNotification) {
		metadata.NotarizedAtSourceInMetaNonce = notification.metaNonce
		metadata.NotarizedAtSourceInMetaHash = notification.metaHash
	})

	hr.consumePendingNotificationsNoLock(hr.pendingNotarizedAtDestinationNotifications, func(metadata *MiniblockMetadata, notification *notarizedNotification) {
		metadata.NotarizedAtDestinationInMetaNonce = notification.metaNonce
		metadata.NotarizedAtDestinationInMetaHash = notification.metaHash
	})

	hr.consumePendingNotificationsNoLock(hr.pendingNotarizedAtBothNotifications, func(metadata *MiniblockMetadata, notification *notarizedNotification) {
		metadata.NotarizedAtSourceInMetaNonce = notification.metaNonce
		metadata.NotarizedAtSourceInMetaHash = notification.metaHash
		metadata.NotarizedAtDestinationInMetaNonce = notification.metaNonce
		metadata.NotarizedAtDestinationInMetaHash = notification.metaHash
	})

	log.Debug("consumePendingNotificationsWithLock() end",
		"len(source)", hr.pendingNotarizedAtSourceNotifications.Len(),
		"len(destination)", hr.pendingNotarizedAtDestinationNotifications.Len(),
		"len(both)", hr.pendingNotarizedAtBothNotifications.Len(),
	)
}

func (hr *historyRepository) consumePendingNotificationsNoLock(pendingMap *container.MutexMap, patchMetadataFunc func(*MiniblockMetadata, *notarizedNotification)) {
	for _, key := range pendingMap.Keys() {
		notification, ok := pendingMap.Get(key)
		if !ok {
			continue
		}

		keyTyped, ok := key.(string)
		if !ok {
			log.Error("consumePendingNotificationsNoLock(): bad key", "key", key)
			continue
		}

		notificationTyped, ok := notification.(*notarizedNotification)
		if !ok {
			log.Error("consumePendingNotificationsNoLock(): bad value", "value", fmt.Sprintf("%T", notification))
			continue
		}

		miniblockHash := []byte(keyTyped)
		metadata, err := hr.getMiniblockMetadataByMiniblockHash(miniblockHash)
		if err != nil {
			// Maybe not yet committed / saved in storer
			continue
		}

		patchMetadataFunc(metadata, notificationTyped)
		err = hr.putMiniblockMetadata(miniblockHash, metadata)
		if err != nil {
			log.Error("consumePendingNotificationsNoLock(): cannot put miniblock metadata", "miniblockHash", miniblockHash, "err", err)
			continue
		}

		pendingMap.Remove(key)
	}
}

// IsEnabled will always returns true
func (hr *historyRepository) IsEnabled() bool {
	return true
}

// IsInterfaceNil returns true if there is no value under the interface
func (hr *historyRepository) IsInterfaceNil() bool {
	return hr == nil
}
