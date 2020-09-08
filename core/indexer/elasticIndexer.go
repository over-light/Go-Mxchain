package indexer

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/statistics"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/elastic/go-elasticsearch/v7/esapi"
)

type elasticIndexer struct {
	*txDatabaseProcessor

	elasticClient databaseClientHandler
	parser        *dataParser
}

// NewElasticIndexer creates an elasticsearch es and handles saving
func NewElasticIndexer(arguments ElasticIndexerArgs) (ElasticIndexer, error) {
	ei := &elasticIndexer{
		elasticClient: arguments.DBClient,
		parser: &dataParser{
			hasher:      arguments.Hasher,
			marshalizer: arguments.Marshalizer,
		},
	}

	ei.txDatabaseProcessor = newTxDatabaseProcessor(
		arguments.Hasher,
		arguments.Marshalizer,
		arguments.AddressPubkeyConverter,
		arguments.ValidatorPubkeyConverter,
	)

	if arguments.Options.UseKibana {
		err := ei.initWithKibana(arguments.IndexTemplates, arguments.IndexPolicies)
		if err != nil {
			return nil, err
		}
	} else {
		err := ei.initNoKibana(arguments.IndexTemplates)
		if err != nil {
			return nil, err
		}
	}

	return ei, nil
}

func (ei *elasticIndexer) initWithKibana(indexTemplates, indexPolicies map[string]*bytes.Buffer) error {
	err := ei.createOpenDistroTemplates(indexTemplates)
	if err != nil {
		return err
	}

	err = ei.createIndexPolicies(indexPolicies)
	if err != nil {
		return err
	}

	err = ei.createIndexTemplates(indexTemplates)
	if err != nil {
		return err
	}

	err = ei.createIndexes()
	if err != nil {
		return err
	}

	err = ei.setInitialAliases()
	if err != nil {
		return err
	}

	return nil
}

func (ei *elasticIndexer) initNoKibana(indexTemplates map[string]*bytes.Buffer) error {
	err := ei.createOpenDistroTemplates(indexTemplates)
	if err != nil {
		return err
	}

	return ei.createIndexes()
}

func (ei *elasticIndexer) createIndexPolicies(indexPolicies map[string]*bytes.Buffer) error {
	txp := getTemplateByName(txPolicy, indexPolicies)
	if txp != nil {
		err := ei.elasticClient.CheckAndCreatePolicy(txPolicy, txp)
		if err != nil {
			return err
		}
	}

	blockp := getTemplateByName(blockPolicy, indexPolicies)
	if blockp != nil {
		err := ei.elasticClient.CheckAndCreatePolicy(blockPolicy, blockp)
		if err != nil {
			return err
		}
	}

	roundsp := getTemplateByName(roundPolicy, indexPolicies)
	if blockp != nil {
		err := ei.elasticClient.CheckAndCreatePolicy(roundPolicy, roundsp)
		if err != nil {
			return err
		}
	}

	validatorsp := getTemplateByName(validatorsPolicy, indexPolicies)
	if blockp != nil {
		err := ei.elasticClient.CheckAndCreatePolicy(validatorsPolicy, validatorsp)
		if err != nil {
			return err
		}
	}

	ratingp := getTemplateByName(ratingPolicy, indexPolicies)
	if blockp != nil {
		err := ei.elasticClient.CheckAndCreatePolicy(ratingPolicy, ratingp)
		if err != nil {
			return err
		}
	}

	tpsp := getTemplateByName(tpsPolicy, indexPolicies)
	if blockp != nil {
		err := ei.elasticClient.CheckAndCreatePolicy(tpsPolicy, tpsp)
		if err != nil {
			return err
		}
	}

	miniblocksp := getTemplateByName(miniblocksPolicy, indexPolicies)
	if blockp != nil {
		err := ei.elasticClient.CheckAndCreatePolicy(miniblocksPolicy, miniblocksp)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ei *elasticIndexer) createOpenDistroTemplates(indexTemplates map[string]*bytes.Buffer) error {
	opendistroTemplate := getTemplateByName("opendistro", indexTemplates)
	if opendistroTemplate != nil {
		err := ei.elasticClient.CheckAndCreateTemplate("opendistro", opendistroTemplate)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ei *elasticIndexer) createIndexTemplates(indexTemplates map[string]*bytes.Buffer) error {
	txTemplate := getTemplateByName(txIndex, indexTemplates)
	if txTemplate != nil {
		err := ei.elasticClient.CheckAndCreateTemplate(txIndex, txTemplate)
		if err != nil {
			return err
		}
	}

	blocksTemplate := getTemplateByName(blockIndex, indexTemplates)
	if blocksTemplate != nil {
		err := ei.elasticClient.CheckAndCreateTemplate(blockIndex, blocksTemplate)
		if err != nil {
			return err
		}
	}

	miniblocksTemplate := getTemplateByName(miniblocksIndex, indexTemplates)
	if miniblocksTemplate != nil {
		err := ei.elasticClient.CheckAndCreateTemplate(miniblocksIndex, miniblocksTemplate)
		if err != nil {
			return err
		}
	}

	tpsTemplate := getTemplateByName(tpsIndex, indexTemplates)
	if tpsTemplate != nil {
		err := ei.elasticClient.CheckAndCreateTemplate(tpsIndex, tpsTemplate)
		if err != nil {
			return err
		}
	}

	ratingTemplate := getTemplateByName(ratingIndex, indexTemplates)
	if ratingTemplate != nil {
		err := ei.elasticClient.CheckAndCreateTemplate(ratingIndex, ratingTemplate)
		if err != nil {
			return err
		}
	}

	roundsTemplate := getTemplateByName(roundIndex, indexTemplates)
	if ratingTemplate != nil {
		err := ei.elasticClient.CheckAndCreateTemplate(roundIndex, roundsTemplate)
		if err != nil {
			return err
		}
	}

	validatorsTemplate := getTemplateByName(validatorsIndex, indexTemplates)
	if ratingTemplate != nil {
		err := ei.elasticClient.CheckAndCreateTemplate(validatorsIndex, validatorsTemplate)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ei *elasticIndexer) createIndexes() error {
	firstTxIndexName := fmt.Sprintf("%s-000001", txIndex)
	err := ei.elasticClient.CheckAndCreateIndex(firstTxIndexName)
	if err != nil {
		return err
	}

	firstBlocksIndexName := fmt.Sprintf("%s-000001", blockIndex)
	err = ei.elasticClient.CheckAndCreateIndex(firstBlocksIndexName)
	if err != nil {
		return err
	}

	firstMiniBlocksIndexName := fmt.Sprintf("%s-000001", miniblocksIndex)
	err = ei.elasticClient.CheckAndCreateIndex(firstMiniBlocksIndexName)
	if err != nil {
		return err
	}

	firstTpsIndexName := fmt.Sprintf("%s-000001", tpsIndex)
	err = ei.elasticClient.CheckAndCreateIndex(firstTpsIndexName)
	if err != nil {
		return err
	}

	firstRatingIndexName := fmt.Sprintf("%s-000001", ratingIndex)
	err = ei.elasticClient.CheckAndCreateIndex(firstRatingIndexName)
	if err != nil {
		return err
	}

	firstRoundsIndexName := fmt.Sprintf("%s-000001", roundIndex)
	err = ei.elasticClient.CheckAndCreateIndex(firstRoundsIndexName)
	if err != nil {
		return err
	}

	firstValidatorsIndexName := fmt.Sprintf("%s-000001", validatorsIndex)
	err = ei.elasticClient.CheckAndCreateIndex(firstValidatorsIndexName)
	if err != nil {
		return err
	}

	return nil
}

func (ei *elasticIndexer) setInitialAliases() error {
	firstTxIndexName := fmt.Sprintf("%s-000001", txIndex)
	err := ei.elasticClient.CheckAndCreateAlias(txIndex, firstTxIndexName)
	if err != nil {
		return err
	}

	firstBlocksIndexName := fmt.Sprintf("%s-000001", blockIndex)
	err = ei.elasticClient.CheckAndCreateAlias(blockIndex, firstBlocksIndexName)
	if err != nil {
		return err
	}

	firstMiniBlocksIndexName := fmt.Sprintf("%s-000001", miniblocksIndex)
	err = ei.elasticClient.CheckAndCreateAlias(miniblocksIndex, firstMiniBlocksIndexName)
	if err != nil {
		return err
	}

	firstTpsIndexName := fmt.Sprintf("%s-000001", tpsIndex)
	err = ei.elasticClient.CheckAndCreateAlias(tpsIndex, firstTpsIndexName)
	if err != nil {
		return err
	}

	firstRatingIndexName := fmt.Sprintf("%s-000001", ratingIndex)
	err = ei.elasticClient.CheckAndCreateAlias(ratingIndex, firstRatingIndexName)
	if err != nil {
		return err
	}

	firstRoundsIndexName := fmt.Sprintf("%s-000001", roundIndex)
	err = ei.elasticClient.CheckAndCreateAlias(roundIndex, firstRoundsIndexName)
	if err != nil {
		return err
	}

	firstValidatorsIndexName := fmt.Sprintf("%s-000001", validatorsIndex)
	err = ei.elasticClient.CheckAndCreateAlias(validatorsIndex, firstValidatorsIndexName)
	if err != nil {
		return err
	}

	return nil
}

func (ei *elasticIndexer) foundedObjMap(hashes []string, index string) (map[string]bool, error) {
	if len(hashes) == 0 {
		return make(map[string]bool), nil
	}

	response, err := ei.elasticClient.DoMultiGet(getDocumentsByIDsQuery(hashes), index)
	if err != nil {
		return nil, err
	}

	return getDecodedResponseMultiGet(response), nil
}

func getTemplateByName(templateName string, templateList map[string]*bytes.Buffer) *bytes.Buffer {
	if template, ok := templateList[templateName]; ok {
		return template
	}

	log.Debug("elasticIndexer.getTemplateByName", "could not find template", templateName)
	return nil
}

// SaveHeader will prepare and save information about a header in elasticsearch server
func (ei *elasticIndexer) SaveHeader(
	header data.HeaderHandler,
	signersIndexes []uint64,
	body *block.Body,
	notarizedHeadersHashes []string,
	txsSize int,
) error {
	var buff bytes.Buffer

	serializedBlock, headerHash := ei.parser.getSerializedElasticBlockAndHeaderHash(header, signersIndexes, body, notarizedHeadersHashes, txsSize)

	buff.Grow(len(serializedBlock))
	_, err := buff.Write(serializedBlock)
	if err != nil {
		return err
	}

	req := &esapi.IndexRequest{
		Index:      blockIndex,
		DocumentID: hex.EncodeToString(headerHash),
		Body:       bytes.NewReader(buff.Bytes()),
		Refresh:    "true",
	}

	return ei.elasticClient.DoRequest(req)
}

// RemoveHeader will remove a block from elasticsearch server
func (ei *elasticIndexer) RemoveHeader(header data.HeaderHandler) error {
	headerHash, err := core.CalculateHash(ei.marshalizer, ei.hasher, header)
	if err != nil {
		return err
	}

	return ei.elasticClient.DoBulkRemove(blockIndex, []string{hex.EncodeToString(headerHash)})
}

// RemoveMiniblocks will remove all miniblocks that are in header from elasticsearch server
func (ei *elasticIndexer) RemoveMiniblocks(header data.HeaderHandler) error {
	miniblocksHashes := header.GetMiniBlockHeadersHashes()
	if len(miniblocksHashes) == 0 {
		return nil
	}

	encodedMiniblocksHashes := make([]string, 0)
	for _, mbHash := range miniblocksHashes {
		encodedMiniblocksHashes = append(encodedMiniblocksHashes, hex.EncodeToString(mbHash))
	}

	return ei.elasticClient.DoBulkRemove(miniblocksIndex, encodedMiniblocksHashes)
}

// SetTxLogsProcessor will set tx logs processor
func (ei *elasticIndexer) SetTxLogsProcessor(txLogsProc process.TransactionLogProcessorDatabase) {
	ei.txLogsProcessor = txLogsProc
}

// SaveMiniblocks will prepare and save information about miniblocks in elasticsearch server
func (ei *elasticIndexer) SaveMiniblocks(header data.HeaderHandler, body *block.Body) (map[string]bool, error) {
	miniblocks := ei.parser.getMiniblocks(header, body)
	if miniblocks == nil {
		log.Warn("indexer: could not index miniblocks")
		return make(map[string]bool), nil
	}
	if len(miniblocks) == 0 {
		return make(map[string]bool), nil
	}

	buff, mbHashDb := serializeBulkMiniBlocks(header.GetShardID(), miniblocks, ei.foundedObjMap)
	return mbHashDb, ei.elasticClient.DoBulkRequest(&buff, miniblocksIndex)
}

// SaveTransactions will prepare and save information about a transactions in elasticsearch server
func (ei *elasticIndexer) SaveTransactions(
	body *block.Body,
	header data.HeaderHandler,
	txPool map[string]data.TransactionHandler,
	selfShardID uint32,
	mbsInDb map[string]bool,
) error {
	txs := ei.prepareTransactionsForDatabase(body, header, txPool, selfShardID)
	buffSlice := serializeTransactions(txs, selfShardID, ei.foundedObjMap, mbsInDb)

	for idx := range buffSlice {
		err := ei.elasticClient.DoBulkRequest(&buffSlice[idx], txIndex)
		if err != nil {
			log.Warn("indexer indexing bulk of transactions",
				"error", err.Error())
			return err
		}
	}

	return nil
}

// SaveShardStatistics will prepare and save information about a shard statistics in elasticsearch server
func (ei *elasticIndexer) SaveShardStatistics(tpsBenchmark statistics.TPSBenchmark) error {
	buff := prepareGeneralInfo(tpsBenchmark)

	for _, shardInfo := range tpsBenchmark.ShardStatistics() {
		serializedShardInfo, serializedMetaInfo := serializeShardInfo(shardInfo)
		if serializedShardInfo == nil {
			continue
		}

		buff.Grow(len(serializedMetaInfo) + len(serializedShardInfo))
		_, err := buff.Write(serializedMetaInfo)
		if err != nil {
			log.Warn("elastic search: update TPS write meta", "error", err.Error())
		}
		_, err = buff.Write(serializedShardInfo)
		if err != nil {
			log.Warn("elastic search: update TPS write serialized data", "error", err.Error())
		}
	}

	return ei.elasticClient.DoBulkRequest(&buff, tpsIndex)
}

// SaveValidatorsRating will save validators rating
func (ei *elasticIndexer) SaveValidatorsRating(index string, validatorsRatingInfo []ValidatorRatingInfo) error {
	var buff bytes.Buffer

	infosRating := ValidatorsRatingInfo{ValidatorsInfos: validatorsRatingInfo}

	marshalizedInfoRating, err := json.Marshal(&infosRating)
	if err != nil {
		log.Debug("indexer: marshal", "error", "could not marshal validators rating")
		return err
	}

	buff.Grow(len(marshalizedInfoRating))
	_, err = buff.Write(marshalizedInfoRating)
	if err != nil {
		log.Warn("elastic search: save validators rating, write", "error", err.Error())
	}

	req := &esapi.IndexRequest{
		Index:      ratingIndex,
		DocumentID: index,
		Body:       bytes.NewReader(buff.Bytes()),
		Refresh:    "true",
	}

	return ei.elasticClient.DoRequest(req)
}

// SaveShardValidatorsPubKeys will prepare and save information about a shard validators public keys in elasticsearch server
func (ei *elasticIndexer) SaveShardValidatorsPubKeys(shardID, epoch uint32, shardValidatorsPubKeys [][]byte) error {
	var buff bytes.Buffer

	shardValPubKeys := ValidatorsPublicKeys{
		PublicKeys: make([]string, 0, len(shardValidatorsPubKeys)),
	}
	for _, validatorPk := range shardValidatorsPubKeys {
		strValidatorPk := ei.validatorPubkeyConverter.Encode(validatorPk)
		shardValPubKeys.PublicKeys = append(shardValPubKeys.PublicKeys, strValidatorPk)
	}

	marshalizedValidatorPubKeys, err := json.Marshal(shardValPubKeys)
	if err != nil {
		log.Debug("indexer: marshal", "error", "could not marshal validators public keys")
		return err
	}

	buff.Grow(len(marshalizedValidatorPubKeys))
	_, err = buff.Write(marshalizedValidatorPubKeys)
	if err != nil {
		log.Warn("elastic search: save shard validators pub keys, write", "error", err.Error())
	}

	req := &esapi.IndexRequest{
		Index:      validatorsIndex,
		DocumentID: fmt.Sprintf("%d_%d", shardID, epoch),
		Body:       bytes.NewReader(buff.Bytes()),
		Refresh:    "true",
	}

	return ei.elasticClient.DoRequest(req)
}

// SaveRoundsInfos will prepare and save information about a slice of rounds in elasticsearch server
func (ei *elasticIndexer) SaveRoundsInfos(infos []RoundInfo) error {
	var buff bytes.Buffer

	for _, info := range infos {
		serializedRoundInfo, meta := serializeRoundInfo(info)

		buff.Grow(len(meta) + len(serializedRoundInfo))
		_, err := buff.Write(meta)
		if err != nil {
			log.Warn("indexer: cannot write meta", "error", err.Error())
		}

		_, err = buff.Write(serializedRoundInfo)
		if err != nil {
			log.Warn("indexer: cannot write serialized round info", "error", err.Error())
		}
	}

	return ei.elasticClient.DoBulkRequest(&buff, roundIndex)
}
