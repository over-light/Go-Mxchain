package genesis

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/update"
)

var _ update.ExportHandler = (*stateExport)(nil)

// ArgsNewStateExporter defines the arguments needed to create new state exporter
type ArgsNewStateExporter struct {
	ShardCoordinator         sharding.Coordinator
	StateSyncer              update.StateSyncer
	Marshalizer              marshal.Marshalizer
	Hasher                   hashing.Hasher
	HardforkStorer           update.HardforkStorer
	ExportFolder             string
	AddressPubKeyConverter   core.PubkeyConverter
	ValidatorPubKeyConverter core.PubkeyConverter
	GenesisNodesSetupHandler update.GenesisNodesSetupHandler
}

type stateExport struct {
	stateSyncer              update.StateSyncer
	shardCoordinator         sharding.Coordinator
	marshalizer              marshal.Marshalizer
	hasher                   hashing.Hasher
	hardforkStorer           update.HardforkStorer
	exportFolder             string
	addressPubKeyConverter   core.PubkeyConverter
	validatorPubKeyConverter core.PubkeyConverter
	genesisNodesSetupHandler update.GenesisNodesSetupHandler
}

var log = logger.GetOrCreate("update/genesis")

// NewStateExporter exports all the data at a specific moment to a hardfork storer
func NewStateExporter(args ArgsNewStateExporter) (*stateExport, error) {
	if check.IfNil(args.ShardCoordinator) {
		return nil, data.ErrNilShardCoordinator
	}
	if check.IfNil(args.StateSyncer) {
		return nil, update.ErrNilStateSyncer
	}
	if check.IfNil(args.Marshalizer) {
		return nil, data.ErrNilMarshalizer
	}
	if check.IfNil(args.Hasher) {
		return nil, update.ErrNilHasher
	}
	if check.IfNil(args.HardforkStorer) {
		return nil, update.ErrNilHardforkStorer
	}
	if len(args.ExportFolder) == 0 {
		return nil, update.ErrEmptyExportFolderPath
	}
	if check.IfNil(args.AddressPubKeyConverter) {
		return nil, fmt.Errorf("%w for address", update.ErrNilPubKeyConverter)
	}
	if check.IfNil(args.ValidatorPubKeyConverter) {
		return nil, fmt.Errorf("%w for validators", update.ErrNilPubKeyConverter)
	}
	if check.IfNil(args.GenesisNodesSetupHandler) {
		return nil, update.ErrNilGenesisNodesSetupHandler
	}

	se := &stateExport{
		stateSyncer:              args.StateSyncer,
		shardCoordinator:         args.ShardCoordinator,
		marshalizer:              args.Marshalizer,
		hasher:                   args.Hasher,
		hardforkStorer:           args.HardforkStorer,
		exportFolder:             args.ExportFolder,
		addressPubKeyConverter:   args.AddressPubKeyConverter,
		validatorPubKeyConverter: args.ValidatorPubKeyConverter,
		genesisNodesSetupHandler: args.GenesisNodesSetupHandler,
	}

	return se, nil
}

// ExportAll syncs and exports all the data from every shard for a certain epoch start block
func (se *stateExport) ExportAll(epoch uint32) error {
	err := se.stateSyncer.SyncAllState(epoch)
	if err != nil {
		return err
	}

	defer func() {
		errClose := se.hardforkStorer.Close()
		log.LogIfError(errClose)
	}()

	err = se.exportMeta()
	if err != nil {
		return err
	}

	err = se.exportAllTries()
	if err != nil {
		return err
	}

	err = se.exportAllMiniBlocks()
	if err != nil {
		return err
	}

	err = se.exportAllTransactions()
	if err != nil {
		return err
	}

	return nil
}

func (se *stateExport) exportAllTransactions() error {
	toExportTransactions, err := se.stateSyncer.GetAllTransactions()
	if err != nil {
		return err
	}

	log.Debug("Exported transactions", "len", len(toExportTransactions))
	for key, tx := range toExportTransactions {
		errExport := se.exportTx(key, tx)
		if errExport != nil {
			return errExport
		}
	}

	return se.hardforkStorer.FinishedIdentifier(TransactionsIdentifier)
}

func (se *stateExport) exportAllMiniBlocks() error {
	toExportMBs, err := se.stateSyncer.GetAllMiniBlocks()
	if err != nil {
		return err
	}

	log.Debug("Exported miniBlocks", "len", len(toExportMBs))
	for key, mb := range toExportMBs {
		errExport := se.exportMBs(key, mb)
		if errExport != nil {
			return errExport
		}
	}

	return se.hardforkStorer.FinishedIdentifier(MiniBlocksIdentifier)
}

func (se *stateExport) exportAllTries() error {
	toExportTries, err := se.stateSyncer.GetAllTries()
	if err != nil {
		return err
	}

	for key, trie := range toExportTries {
		err = se.exportTrie(key, trie)
		if err != nil {
			return err
		}
	}

	return nil
}

func (se *stateExport) exportMeta() error {
	metaBlock, err := se.stateSyncer.GetEpochStartMetaBlock()
	if err != nil {
		return err
	}

	jsonData, err := json.Marshal(metaBlock)
	if err != nil {
		return err
	}

	metaHash := se.hasher.Compute(string(jsonData))
	versionKey := CreateVersionKey(metaBlock, metaHash)

	err = se.hardforkStorer.Write(MetaBlockIdentifier, []byte(versionKey), jsonData)
	if err != nil {
		return err
	}

	err = se.hardforkStorer.FinishedIdentifier(MetaBlockIdentifier)
	if err != nil {
		return err
	}

	log.Debug("Exported metaBlock", "rootHash", metaBlock.RootHash)

	return nil
}

func (se *stateExport) exportTrie(key string, trie data.Trie) error {
	identifier := TrieIdentifier + atSep + key

	accType, shId, err := GetTrieTypeAndShId(identifier)
	if err != nil {
		return err
	}

	leaves, err := trie.GetAllLeaves()
	if err != nil {
		return err
	}

	// no need to export validator account
	// TODO: export validatorAccount trie to nodeSetup.json with all ratings
	if accType == ValidatorAccount {
		validatorData, err := getValidatorDataFromLeaves(leaves, se.shardCoordinator, se.marshalizer)
		if err != nil {
			return err
		}

		return se.exportNodesSetupJson(validatorData)
	}

	if shId > se.shardCoordinator.NumberOfShards() && shId != core.MetachainShardId {
		return sharding.ErrInvalidShardId
	}

	rootHashKey := CreateRootHashKey(key)
	rootHash, err := trie.Root()
	if err != nil {
		return err
	}

	err = se.hardforkStorer.Write(identifier, []byte(rootHashKey), rootHash)
	if err != nil {
		return err
	}

	if accType == DataTrie {
		return se.exportDataTries(leaves, accType, shId, identifier)
	}

	return se.exportAccountLeafs(leaves, accType, shId, identifier)
}

func (se *stateExport) exportDataTries(leafs map[string][]byte, accType Type, shId uint32, identifier string) error {
	for address, buff := range leafs {
		keyToExport := CreateAccountKey(accType, shId, address)

		err := se.hardforkStorer.Write(identifier, []byte(keyToExport), buff)
		if err != nil {
			return err
		}
	}

	err := se.hardforkStorer.FinishedIdentifier(identifier)
	if err != nil {
		return err
	}

	return nil
}

func (se *stateExport) exportAccountLeafs(leafs map[string][]byte, accType Type, shId uint32, identifier string) error {
	for address, buff := range leafs {
		keyToExport := CreateAccountKey(accType, shId, address)

		err := se.hardforkStorer.Write(identifier, []byte(keyToExport), buff)
		if err != nil {
			return err
		}
	}

	err := se.hardforkStorer.FinishedIdentifier(identifier)
	if err != nil {
		return err
	}

	return nil
}

func (se *stateExport) marshallLeafToJson(
	accType Type,
	address, buff []byte,
) ([]byte, error) {
	account, err := NewEmptyAccount(accType, address)
	if err != nil {
		log.Warn("error creating new account account", "address", address, "error", err)
		return nil, err
	}

	err = se.marshalizer.Unmarshal(account, buff)
	if err != nil {
		log.Trace("error unmarshaling account this is maybe a code error",
			"key", hex.EncodeToString(address),
			"error", err,
		)

		return buff, nil
	}

	jsonData, err := json.Marshal(account)
	if err != nil {
		log.Warn("error marshaling account", "address", address, "error", err)
		return nil, err
	}

	return jsonData, nil
}

func (se *stateExport) exportMBs(key string, mb *block.MiniBlock) error {
	marshaledData, err := json.Marshal(mb)
	if err != nil {
		return err
	}

	keyToSave := CreateMiniBlockKey(key)

	err = se.hardforkStorer.Write(MiniBlocksIdentifier, []byte(keyToSave), marshaledData)
	if err != nil {
		return err
	}

	return nil
}

func (se *stateExport) exportTx(key string, tx data.TransactionHandler) error {
	marshaledData, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	keyToSave := CreateTransactionKey(key, tx)

	err = se.hardforkStorer.Write(TransactionsIdentifier, []byte(keyToSave), marshaledData)
	if err != nil {
		return err
	}

	return nil
}

func (se *stateExport) exportNodesSetupJson(validators map[uint32][]*state.ValidatorInfo) error {
	acceptedListsForExport := []core.PeerType{core.EligibleList, core.WaitingList, core.JailedList}
	initialNodes := make([]*sharding.InitialNode, 0)

	for _, validatorsInShard := range validators {
		for _, validator := range validatorsInShard {
			if shouldExportValidator(validator, acceptedListsForExport) {
				initialNodes = append(initialNodes, &sharding.InitialNode{
					PubKey:        se.validatorPubKeyConverter.Encode(validator.GetPublicKey()),
					Address:       se.addressPubKeyConverter.Encode(validator.GetRewardAddress()),
					InitialRating: validator.GetRating(),
				})
			}
		}
	}

	genesisNodesSetupHandler := se.genesisNodesSetupHandler
	nodesSetup := &sharding.NodesSetup{
		StartTime:                   genesisNodesSetupHandler.GetStartTime(),
		RoundDuration:               genesisNodesSetupHandler.GetRoundDuration(),
		ConsensusGroupSize:          genesisNodesSetupHandler.GetShardConsensusGroupSize(),
		MinNodesPerShard:            genesisNodesSetupHandler.MinNumberOfShardNodes(),
		ChainID:                     genesisNodesSetupHandler.GetChainId(),
		MinTransactionVersion:       genesisNodesSetupHandler.GetMinTransactionVersion(),
		MetaChainConsensusGroupSize: genesisNodesSetupHandler.GetMetaConsensusGroupSize(),
		MetaChainMinNodes:           genesisNodesSetupHandler.MinNumberOfMetaNodes(),
		Hysteresis:                  genesisNodesSetupHandler.GetHysteresis(),
		Adaptivity:                  genesisNodesSetupHandler.GetAdaptivity(),
		InitialNodes:                initialNodes,
	}

	nodesSetupBytes, err := json.MarshalIndent(nodesSetup, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(se.exportFolder, "nodesSetup.json"), nodesSetupBytes, 0755)
}

// IsInterfaceNil returns true if underlying object is nil
func (se *stateExport) IsInterfaceNil() bool {
	return se == nil
}
