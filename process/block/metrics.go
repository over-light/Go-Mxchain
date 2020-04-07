package block

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/indexer"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

func getMetricsFromMetaHeader(
	header *block.MetaBlock,
	marshalizer marshal.Marshalizer,
	appStatusHandler core.AppStatusHandler,
	numShardHeadersFromPool int,
	numShardHeadersProcessed uint64,
) {
	numMiniBlocksMetaBlock := uint64(0)
	headerSize := uint64(0)

	for _, shardInfo := range header.ShardInfo {
		numMiniBlocksMetaBlock += uint64(len(shardInfo.ShardMiniBlockHeaders))
	}

	marshalizedHeader, err := marshalizer.Marshal(header)
	if err == nil {
		headerSize = uint64(len(marshalizedHeader))
	}

	appStatusHandler.SetUInt64Value(core.MetricHeaderSize, headerSize)
	appStatusHandler.SetUInt64Value(core.MetricNumTxInBlock, uint64(header.TxCount))
	appStatusHandler.SetUInt64Value(core.MetricNumMiniBlocks, numMiniBlocksMetaBlock)
	appStatusHandler.SetUInt64Value(core.MetricNumShardHeadersProcessed, numShardHeadersProcessed)
	appStatusHandler.SetUInt64Value(core.MetricNumShardHeadersFromPool, uint64(numShardHeadersFromPool))
}

func getMetricsFromBlockBody(
	body *block.Body,
	marshalizer marshal.Marshalizer,
	appStatusHandler core.AppStatusHandler,
) {
	mbLen := len(body.MiniBlocks)
	miniblocksSize := uint64(0)
	totalTxCount := 0
	for i := 0; i < mbLen; i++ {
		totalTxCount += len(body.MiniBlocks[i].TxHashes)

		marshalizedBlock, err := marshalizer.Marshal(body.MiniBlocks[i])
		if err == nil {
			miniblocksSize += uint64(len(marshalizedBlock))
		}
	}
	appStatusHandler.SetUInt64Value(core.MetricNumTxInBlock, uint64(totalTxCount))
	appStatusHandler.SetUInt64Value(core.MetricNumMiniBlocks, uint64(mbLen))
	appStatusHandler.SetUInt64Value(core.MetricMiniBlocksSize, miniblocksSize)
}

func getMetricsFromHeader(
	header *block.Header,
	numTxWithDst uint64,
	marshalizer marshal.Marshalizer,
	appStatusHandler core.AppStatusHandler,
) {
	headerSize := uint64(0)
	marshalizedHeader, err := marshalizer.Marshal(header)
	if err == nil {
		headerSize = uint64(len(marshalizedHeader))
	}

	appStatusHandler.SetUInt64Value(core.MetricHeaderSize, headerSize)
	appStatusHandler.SetUInt64Value(core.MetricTxPoolLoad, numTxWithDst)
}

func saveMetricsForACommittedBlock(
	nodesCoordinator sharding.NodesCoordinator,
	appStatusHandler core.AppStatusHandler,
	currentBlockHash string,
	highestFinalBlockNonce uint64,
	metaBlock data.HeaderHandler,
	shardHeader *block.Header,
) {
	incrementCountAcceptedBlocks(nodesCoordinator, appStatusHandler, shardHeader)
	appStatusHandler.SetUInt64Value(core.MetricEpochNumber, uint64(metaBlock.GetEpoch()))
	appStatusHandler.SetStringValue(core.MetricCurrentBlockHash, currentBlockHash)
	appStatusHandler.SetUInt64Value(core.MetricHighestFinalBlockInShard, highestFinalBlockNonce)
	appStatusHandler.SetStringValue(core.MetricCrossCheckBlockHeight, fmt.Sprintf("meta %d", metaBlock.GetNonce()))
}

func incrementCountAcceptedBlocks(
	nodesCoordinator sharding.NodesCoordinator,
	appStatusHandler core.AppStatusHandler,
	header *block.Header,
) {
	consensusGroup, err := nodesCoordinator.ComputeConsensusGroup(
		header.GetPrevRandSeed(),
		header.GetRound(),
		header.GetShardID(),
		header.GetEpoch(),
	)
	if err != nil {
		return
	}

	ownPubKey := nodesCoordinator.GetOwnPublicKey()
	myIndex := 0
	found := false
	for idx, val := range consensusGroup {
		if bytes.Equal(ownPubKey, val.PubKey()) {
			myIndex = idx
			found = true
			break
		}
	}

	if !found {
		return
	}

	bitMap := header.GetPubKeysBitmap()
	indexOutOfBounds := myIndex/8 >= len(bitMap)
	if indexOutOfBounds {
		log.Trace("process blocks metrics: index out of bounds",
			"index", myIndex,
			"bitMap", bitMap,
			"bitMap length", len(bitMap))
		return
	}

	indexInBitmap := bitMap[myIndex/8]&(1<<uint8(myIndex%8)) != 0
	if indexInBitmap {
		appStatusHandler.Increment(core.MetricCountConsensusAcceptedBlocks)
	}
}

func saveMetachainCommitBlockMetrics(
	appStatusHandler core.AppStatusHandler,
	header *block.MetaBlock,
	headerHash []byte,
	nodesCoordinator sharding.NodesCoordinator,

) {
	appStatusHandler.SetStringValue(core.MetricCurrentBlockHash, logger.DisplayByteSlice(headerHash))
	appStatusHandler.SetUInt64Value(core.MetricEpochNumber, uint64(header.Epoch))

	// TODO: remove if epoch start block needs to be validated by the new epoch nodes
	epoch := header.GetEpoch()
	if header.IsStartOfEpochBlock() && epoch > 0 {
		epoch = epoch - 1
	}

	pubKeys, err := nodesCoordinator.GetConsensusValidatorsPublicKeys(
		header.PrevRandSeed,
		header.Round,
		core.MetachainShardId,
		epoch,
	)
	if err != nil {
		log.Debug("cannot get validators public keys", "error", err.Error())
	}

	countNotarizedHeaders(pubKeys, nodesCoordinator.GetOwnPublicKey(), appStatusHandler, len(header.ShardInfo))
}

func countNotarizedHeaders(
	publicKeys []string,
	ownPublicKey []byte,
	appStatusHandler core.AppStatusHandler,
	numBlockHeaders int,
) {
	isInConsensus := false

	for _, publicKey := range publicKeys {
		if bytes.Equal([]byte(publicKey), ownPublicKey) {
			isInConsensus = true
			break
		}
	}

	if !isInConsensus || numBlockHeaders == 0 {
		return
	}

	appStatusHandler.AddUint64(core.MetricCountConsensusAcceptedBlocks, uint64(numBlockHeaders))
}

func indexRoundInfo(
	indexerHandler indexer.Indexer,
	nodesCoordinator sharding.NodesCoordinator,
	shardId uint32,
	header data.HeaderHandler,
	lastHeader data.HeaderHandler,
	signersIndexes []uint64,
) {
	roundInfo := indexer.RoundInfo{
		Index:            header.GetRound(),
		SignersIndexes:   signersIndexes,
		BlockWasProposed: true,
		ShardId:          shardId,
		Timestamp:        time.Duration(header.GetTimeStamp()),
	}

	go indexerHandler.SaveRoundInfo(roundInfo)

	if check.IfNil(lastHeader) {
		return
	}

	lastBlockRound := lastHeader.GetRound()
	currentBlockRound := header.GetRound()
	roundDuration := calculateRoundDuration(lastHeader.GetTimeStamp(), header.GetTimeStamp(), lastBlockRound, currentBlockRound)
	for i := lastBlockRound + 1; i < currentBlockRound; i++ {
		publicKeys, err := nodesCoordinator.GetConsensusValidatorsPublicKeys(lastHeader.GetRandSeed(), i, shardId, lastHeader.GetEpoch())
		if err != nil {
			continue
		}
		signersIndexes, err = nodesCoordinator.GetValidatorsIndexes(publicKeys, lastHeader.GetEpoch())
		if err != nil {
			log.Error(err.Error(), "round", i)
			continue
		}

		roundInfo = indexer.RoundInfo{
			Index:            i,
			SignersIndexes:   signersIndexes,
			BlockWasProposed: false,
			ShardId:          shardId,
			Timestamp:        time.Duration(header.GetTimeStamp() - ((currentBlockRound - i) * roundDuration)),
		}

		go indexerHandler.SaveRoundInfo(roundInfo)
	}
}

func indexValidatorsRating(
	indexerHandler indexer.Indexer,
	valStatProc process.ValidatorStatisticsProcessor,
	metaBlock data.HeaderHandler,
) {
	// TODO use validatorInfoProvider  to get information about rating
	latestHash, err := valStatProc.RootHash()
	if err != nil {
		return
	}

	validators, err := valStatProc.GetValidatorInfoForRootHash(latestHash)
	if err != nil {
		return
	}

	for shardID, validatorInfosInShard := range validators {
		validatorsInfos := make([]indexer.ValidatorRatingInfo, 0)
		for _, validatorInfo := range validatorInfosInShard {
			validatorsInfos = append(validatorsInfos, indexer.ValidatorRatingInfo{
				PublicKey: hex.EncodeToString(validatorInfo.PublicKey),
				Rating:    float32(validatorInfo.Rating) * 100 / 1000000,
			})

		}

		indexID := fmt.Sprintf("%d_%d", shardID, metaBlock.GetEpoch())
		indexerHandler.SaveValidatorsRating(indexID, validatorsInfos)
	}
}

func calculateRoundDuration(
	lastBlockTimestamp uint64,
	currentBlockTimestamp uint64,
	lastBlockRound uint64,
	currentBlockRound uint64,
) uint64 {
	if lastBlockTimestamp >= currentBlockTimestamp {
		log.Debug("last block timestamp is greater or equals than current block timestamp")
		return 0
	}
	if lastBlockRound >= currentBlockRound {
		log.Debug("last block round is greater or equals than current block round")
		return 0
	}

	diffTimeStamp := currentBlockTimestamp - lastBlockTimestamp
	diffRounds := currentBlockRound - lastBlockRound

	return diffTimeStamp / diffRounds
}
