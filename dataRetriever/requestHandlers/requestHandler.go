package requestHandlers

import (
	"fmt"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/partitioning"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/process/factory"
)

type resolverRequestHandler struct {
	resolversFinder       dataRetriever.ResolversFinder
	requestedItemsHandler dataRetriever.RequestedItemsHandler
	epoch                 uint32
	whiteList             dataRetriever.WhiteListHandler
	shardID               uint32
	maxTxsToRequest       int
	sweepTime             time.Time
	requestInterval       time.Duration
	mutSweepTime          sync.Mutex
}

var log = logger.GetOrCreate("dataretriever/requesthandlers")

// NewResolverRequestHandler creates a requestHandler interface implementation with request functions
func NewResolverRequestHandler(
	finder dataRetriever.ResolversFinder,
	requestedItemsHandler dataRetriever.RequestedItemsHandler,
	whiteList dataRetriever.WhiteListHandler,
	maxTxsToRequest int,
	shardID uint32,
	requestInterval time.Duration,
) (*resolverRequestHandler, error) {

	if check.IfNil(finder) {
		return nil, dataRetriever.ErrNilResolverFinder
	}
	if check.IfNil(requestedItemsHandler) {
		return nil, dataRetriever.ErrNilRequestedItemsHandler
	}
	if maxTxsToRequest < 1 {
		return nil, dataRetriever.ErrInvalidMaxTxRequest
	}
	if check.IfNil(whiteList) {
		return nil, dataRetriever.ErrNilWhiteListHandler
	}
	if requestInterval < time.Millisecond {
		return nil, dataRetriever.ErrRequestIntervalTooSmall
	}

	rrh := &resolverRequestHandler{
		resolversFinder:       finder,
		requestedItemsHandler: requestedItemsHandler,
		epoch:                 uint32(0), // will be updated after creation of the request handler
		shardID:               shardID,
		maxTxsToRequest:       maxTxsToRequest,
		whiteList:             whiteList,
		requestInterval:       requestInterval,
	}

	rrh.sweepTime = time.Now()

	return rrh, nil
}

// SetEpoch will update the current epoch so the request handler will make requests for this received epoch
func (rrh *resolverRequestHandler) SetEpoch(epoch uint32) {
	rrh.epoch = epoch
}

// RequestTransaction method asks for transactions from the connected peers
func (rrh *resolverRequestHandler) RequestTransaction(destShardID uint32, txHashes [][]byte) {
	rrh.requestByHashes(destShardID, txHashes, factory.TransactionTopic)
}

func (rrh *resolverRequestHandler) requestByHashes(destShardID uint32, hashes [][]byte, topic string) {
	unrequestedHashes := rrh.getUnrequestedHashes(hashes)
	if len(unrequestedHashes) == 0 {
		return
	}
	log.Debug("requesting transactions from network",
		"topic", topic,
		"shard", destShardID,
		"num txs", len(unrequestedHashes),
	)
	resolver, err := rrh.resolversFinder.CrossShardResolver(topic, destShardID)
	if err != nil {
		log.Error("requestByHashes.CrossShardResolver",
			"error", err.Error(),
			"topic", topic,
			"shard", destShardID,
		)
		return
	}

	txResolver, ok := resolver.(HashSliceResolver)
	if !ok {
		log.Warn("wrong assertion type when creating transaction resolver")
		return
	}

	go func() {
		dataSplit := &partitioning.DataSplit{}
		var sliceBatches [][][]byte
		sliceBatches, err = dataSplit.SplitDataInChunks(unrequestedHashes, rrh.maxTxsToRequest)
		if err != nil {
			log.Debug("requestByHashes.SplitDataInChunks",
				"error", err.Error(),
				"num txs", len(unrequestedHashes),
				"max txs to request", rrh.maxTxsToRequest,
			)
			return
		}

		for _, batch := range sliceBatches {
			err = txResolver.RequestDataFromHashArray(batch, rrh.epoch)
			if err != nil {
				log.Debug("requestByHashes.RequestDataFromHashArray",
					"error", err.Error(),
					"epoch", rrh.epoch,
					"batch size", len(batch),
				)
			}
		}
	}()

	rrh.whiteList.Add(hashes)
	for _, hash := range unrequestedHashes {
		rrh.addRequestedItem(hash)
	}
}

// RequestUnsignedTransactions method asks for unsigned transactions from the connected peers
func (rrh *resolverRequestHandler) RequestUnsignedTransactions(destShardID uint32, scrHashes [][]byte) {
	rrh.requestByHashes(destShardID, scrHashes, factory.UnsignedTransactionTopic)
}

// RequestRewardTransactions requests for reward transactions from the connected peers
func (rrh *resolverRequestHandler) RequestRewardTransactions(destShardID uint32, rewardTxHashes [][]byte) {
	rrh.requestByHashes(destShardID, rewardTxHashes, factory.RewardsTransactionTopic)
}

// RequestMiniBlock method asks for miniblock from the connected peers
func (rrh *resolverRequestHandler) RequestMiniBlock(destShardID uint32, miniblockHash []byte) {
	if !rrh.testIfRequestIsNeeded(miniblockHash) {
		return
	}

	log.Debug("requesting miniblock from network",
		"topic", factory.MiniBlocksTopic,
		"shard", destShardID,
		"hash", miniblockHash,
	)

	resolver, err := rrh.resolversFinder.CrossShardResolver(factory.MiniBlocksTopic, destShardID)
	if err != nil {
		log.Error("RequestMiniBlock.CrossShardResolver",
			"error", err.Error(),
			"topic", factory.MiniBlocksTopic,
			"shard", destShardID,
		)
		return
	}

	err = resolver.RequestDataFromHash(miniblockHash, rrh.epoch)
	if err != nil {
		log.Debug("RequestMiniBlock.RequestDataFromHash",
			"error", err.Error(),
			"epoch", rrh.epoch,
			"hash", miniblockHash,
		)
		return
	}

	rrh.addRequestedItem(miniblockHash)
}

// RequestMiniBlocks method asks for miniblocks from the connected peers
func (rrh *resolverRequestHandler) RequestMiniBlocks(destShardID uint32, miniblocksHashes [][]byte) {
	unrequestedHashes := rrh.getUnrequestedHashes(miniblocksHashes)
	if len(unrequestedHashes) == 0 {
		return
	}
	log.Debug("requesting miniblocks from network",
		"topic", factory.MiniBlocksTopic,
		"shard", destShardID,
		"num txs", len(unrequestedHashes),
	)

	resolver, err := rrh.resolversFinder.CrossShardResolver(factory.MiniBlocksTopic, destShardID)
	if err != nil {
		log.Error("RequestMiniBlocks.CrossShardResolver",
			"error", err.Error(),
			"topic", factory.MiniBlocksTopic,
			"shard", destShardID,
		)
		return
	}

	miniBlocksResolver, ok := resolver.(dataRetriever.MiniBlocksResolver)
	if !ok {
		log.Warn("wrong assertion type when creating miniblocks resolver")
		return
	}

	err = miniBlocksResolver.RequestDataFromHashArray(unrequestedHashes, rrh.epoch)
	if err != nil {
		log.Debug("RequestMiniBlocks.RequestDataFromHashArray",
			"error", err.Error(),
			"epoch", rrh.epoch,
			"num miniblocks", len(unrequestedHashes),
		)
		return
	}

	for _, hash := range unrequestedHashes {
		rrh.addRequestedItem(hash)
	}
}

// RequestShardHeader method asks for shard header from the connected peers
func (rrh *resolverRequestHandler) RequestShardHeader(shardID uint32, hash []byte) {
	if !rrh.testIfRequestIsNeeded(hash) {
		return
	}

	log.Debug("requesting shard header from network",
		"shard", shardID,
		"hash", hash,
	)

	headerResolver, err := rrh.getShardHeaderResolver(shardID)
	if err != nil {
		log.Error("RequestShardHeader.getShardHeaderResolver",
			"error", err.Error(),
			"shard", shardID,
		)
		return
	}

	err = headerResolver.RequestDataFromHash(hash, rrh.epoch)
	if err != nil {
		log.Debug("RequestShardHeader.RequestDataFromHash",
			"error", err.Error(),
			"epoch", rrh.epoch,
			"hash", hash,
		)
		return
	}

	rrh.addRequestedItem(hash)
}

// RequestMetaHeader method asks for meta header from the connected peers
func (rrh *resolverRequestHandler) RequestMetaHeader(hash []byte) {
	if !rrh.testIfRequestIsNeeded(hash) {
		return
	}

	log.Debug("requesting meta header from network",
		"hash", hash,
	)

	resolver, err := rrh.getMetaHeaderResolver()
	if err != nil {
		log.Error("RequestMetaHeader.getMetaHeaderResolver",
			"error", err.Error(),
			"hash", hash,
		)
		return
	}

	err = resolver.RequestDataFromHash(hash, rrh.epoch)
	if err != nil {
		log.Debug("RequestMetaHeader.RequestDataFromHash",
			"error", err.Error(),
			"epoch", rrh.epoch,
			"hash", hash,
		)
		return
	}

	rrh.addRequestedItem(hash)
}

// RequestShardHeaderByNonce method asks for shard header from the connected peers by nonce
func (rrh *resolverRequestHandler) RequestShardHeaderByNonce(shardID uint32, nonce uint64) {
	key := []byte(fmt.Sprintf("%d-%d", shardID, nonce))
	if !rrh.testIfRequestIsNeeded(key) {
		return
	}

	log.Debug("requesting shard header by nonce from network",
		"shard", shardID,
		"nonce", nonce,
	)

	headerResolver, err := rrh.getShardHeaderResolver(shardID)
	if err != nil {
		log.Error("RequestShardHeaderByNonce.getShardHeaderResolver",
			"error", err.Error(),
			"shard", shardID,
		)
		return
	}

	err = headerResolver.RequestDataFromNonce(nonce, rrh.epoch)
	if err != nil {
		log.Debug("RequestShardHeaderByNonce.RequestDataFromNonce",
			"error", err.Error(),
			"epoch", rrh.epoch,
			"nonce", nonce,
		)
		return
	}

	rrh.addRequestedItem(key)
}

// RequestTrieNodes method asks for trie nodes from the connected peers
func (rrh *resolverRequestHandler) RequestTrieNodes(destShardID uint32, hash []byte, topic string) {
	if !rrh.testIfRequestIsNeeded(hash) {
		return
	}

	log.Debug("requesting trie from network",
		"topic", topic,
		"shard", destShardID,
		"hash", hash,
	)

	resolver, err := rrh.resolversFinder.MetaCrossShardResolver(topic, destShardID)
	if err != nil {
		log.Error("requestByHash.Resolver",
			"error", err.Error(),
			"topic", topic,
			"shard", destShardID,
		)
		return
	}

	// epoch doesn't matter because that parameter is not used in trie's resolver
	err = resolver.RequestDataFromHash(hash, 0)
	if err != nil {
		log.Debug("requestByHash.RequestDataFromHash",
			"error", err.Error(),
			"epoch", 0,
			"hash", hash,
		)
		return
	}

	rrh.addRequestedItem(hash)
}

// RequestMetaHeaderByNonce method asks for meta header from the connected peers by nonce
func (rrh *resolverRequestHandler) RequestMetaHeaderByNonce(nonce uint64) {
	key := []byte(fmt.Sprintf("%d-%d", core.MetachainShardId, nonce))
	if !rrh.testIfRequestIsNeeded(key) {
		return
	}

	log.Debug("requesting meta header by nonce from network",
		"nonce", nonce,
	)

	headerResolver, err := rrh.getMetaHeaderResolver()
	if err != nil {
		log.Error("RequestMetaHeaderByNonce.getMetaHeaderResolver",
			"error", err.Error(),
		)
		return
	}

	err = headerResolver.RequestDataFromNonce(nonce, rrh.epoch)
	if err != nil {
		log.Debug("RequestMetaHeaderByNonce.RequestDataFromNonce",
			"error", err.Error(),
			"epoch", rrh.epoch,
			"nonce", nonce,
		)
		return
	}

	rrh.addRequestedItem(key)
}

func (rrh *resolverRequestHandler) testIfRequestIsNeeded(key []byte) bool {
	rrh.sweepIfNeeded()

	if rrh.requestedItemsHandler.Has(string(key)) {
		log.Trace("item already requested",
			"key", key)
		return false
	}

	return true
}

func (rrh *resolverRequestHandler) addRequestedItem(key []byte) {
	err := rrh.requestedItemsHandler.Add(string(key))
	if err != nil {
		log.Trace("addRequestedItem",
			"error", err.Error(),
			"key", key)
		return
	}
	rrh.whiteList.Add([][]byte{key})
}

func (rrh *resolverRequestHandler) getShardHeaderResolver(shardID uint32) (dataRetriever.HeaderResolver, error) {
	isMetachainNode := rrh.shardID == core.MetachainShardId
	shardIdMissmatch := rrh.shardID != shardID
	requestOnMetachain := shardID == core.MetachainShardId
	isRequestInvalid := (!isMetachainNode && shardIdMissmatch) || requestOnMetachain
	if isRequestInvalid {
		return nil, dataRetriever.ErrBadRequest
	}

	//requests should be done on the topic shardBlocks_0_META so that is why we need to figure out
	//the cross shard id
	crossShardID := core.MetachainShardId
	if isMetachainNode {
		crossShardID = shardID
	}

	resolver, err := rrh.resolversFinder.CrossShardResolver(factory.ShardBlocksTopic, crossShardID)
	if err != nil {
		err = fmt.Errorf("%w, topic: %s, current shard ID: %d, cross shard ID: %d",
			err, factory.ShardBlocksTopic, rrh.shardID, crossShardID)

		log.Warn("available resolvers in container",
			"resolvers", rrh.resolversFinder.ResolverKeys(),
		)
		return nil, err
	}

	headerResolver, ok := resolver.(dataRetriever.HeaderResolver)
	if !ok {
		err = fmt.Errorf("%w, topic: %s, current shard ID: %d, cross shard ID: %d, expected HeaderResolver",
			dataRetriever.ErrWrongTypeInContainer, factory.ShardBlocksTopic, rrh.shardID, crossShardID)
		return nil, err
	}

	return headerResolver, nil
}

func (rrh *resolverRequestHandler) getMetaHeaderResolver() (dataRetriever.HeaderResolver, error) {
	resolver, err := rrh.resolversFinder.MetaChainResolver(factory.MetachainBlocksTopic)
	if err != nil {
		err = fmt.Errorf("%w, topic: %s, current shard ID: %d",
			err, factory.MetachainBlocksTopic, rrh.shardID)
		return nil, err
	}

	headerResolver, ok := resolver.(dataRetriever.HeaderResolver)
	if !ok {
		err = fmt.Errorf("%w, topic: %s, current shard ID: %d, expected HeaderResolver",
			dataRetriever.ErrWrongTypeInContainer, factory.ShardBlocksTopic, rrh.shardID)
		return nil, err
	}

	return headerResolver, nil
}

// RequestStartOfEpochMetaBlock method asks for the start of epoch metablock from the connected peers
func (rrh *resolverRequestHandler) RequestStartOfEpochMetaBlock(epoch uint32) {
	epochStartIdentifier := core.EpochStartIdentifier(epoch)
	if !rrh.testIfRequestIsNeeded([]byte(epochStartIdentifier)) {
		return
	}

	baseTopic := factory.MetachainBlocksTopic
	log.Debug("requesting header by epoch",
		"topic", baseTopic,
		"epoch", epoch,
		"hash", epochStartIdentifier,
	)

	resolver, err := rrh.resolversFinder.MetaChainResolver(baseTopic)
	if err != nil {
		log.Error("RequestStartOfEpochMetaBlock.MetaChainResolver",
			"error", err.Error(),
			"topic", baseTopic,
		)
		return
	}

	headerResolver, ok := resolver.(dataRetriever.HeaderResolver)
	if !ok {
		log.Warn("wrong assertion type when creating header resolver")
		return
	}

	err = headerResolver.RequestDataFromEpoch([]byte(epochStartIdentifier))
	if err != nil {
		log.Debug("RequestStartOfEpochMetaBlock.RequestDataFromEpoch",
			"error", err.Error(),
			"epochStartIdentifier", epochStartIdentifier,
		)
		return
	}

	rrh.addRequestedItem([]byte(epochStartIdentifier))
}

// RequestInterval returns the request interval between sending the same request
func (rrh *resolverRequestHandler) RequestInterval() time.Duration {
	return rrh.requestInterval
}

// IsInterfaceNil returns true if there is no value under the interface
func (rrh *resolverRequestHandler) IsInterfaceNil() bool {
	return rrh == nil
}

func (rrh *resolverRequestHandler) getUnrequestedHashes(hashes [][]byte) [][]byte {
	unrequestedHashes := make([][]byte, 0)

	rrh.sweepIfNeeded()

	for _, hash := range hashes {
		if !rrh.requestedItemsHandler.Has(string(hash)) {
			unrequestedHashes = append(unrequestedHashes, hash)
		}
	}

	return unrequestedHashes
}

func (rrh *resolverRequestHandler) sweepIfNeeded() {
	rrh.mutSweepTime.Lock()
	defer rrh.mutSweepTime.Unlock()

	if time.Since(rrh.sweepTime) <= rrh.requestInterval {
		return
	}

	rrh.sweepTime = time.Now()
	rrh.requestedItemsHandler.Sweep()
}
