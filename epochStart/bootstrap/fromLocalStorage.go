package bootstrap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
)

func (e *epochStartBootstrap) initializeFromLocalStorage() {
	latestData, errNotCritical := e.latestStorageDataProvider.Get()
	if errNotCritical != nil {
		e.baseData.storageExists = false
		log.Debug("no epoch db found in storage", "error", errNotCritical.Error())
	} else {
		e.baseData.storageExists = true
		e.baseData.lastEpoch = latestData.Epoch
		e.baseData.shardId = latestData.ShardID
		e.baseData.lastRound = latestData.LastRound
		e.baseData.epochStartRound = latestData.EpochStartRound
		log.Debug("got last data from storage",
			"epoch", e.baseData.lastEpoch,
			"last round", e.baseData.lastRound,
			"last shard ID", e.baseData.shardId,
			"epoch start Round", e.baseData.epochStartRound)
	}
}

func (e *epochStartBootstrap) prepareEpochFromStorage() (Parameters, error) {
	storer, err := e.storageOpenerHandler.GetMostRecentBootstrapStorageUnit()
	defer func() {
		if check.IfNil(storer) {
			return
		}

		errClose := storer.Close()
		log.LogIfError(errClose)
	}()

	if err != nil {
		return Parameters{}, err
	}

	_, e.nodesConfig, err = e.getLastBootstrapData(storer)
	if err != nil {
		return Parameters{}, err
	}

	pubKey, err := e.publicKey.ToByteArray()
	if err != nil {
		return Parameters{}, err
	}

	e.epochStartMeta, err = e.getEpochStartMetaFromStorage(storer)
	if err != nil {
		return Parameters{}, err
	}
	e.baseData.numberOfShards = uint32(len(e.epochStartMeta.EpochStart.LastFinalizedHeaders))

	newShardId, isShuffledOut := e.checkIfShuffledOut(pubKey, e.nodesConfig)
	if !isShuffledOut {
		parameters := Parameters{
			Epoch:       e.baseData.lastEpoch,
			SelfShardId: e.baseData.shardId,
			NumOfShards: e.baseData.numberOfShards,
		}
		return parameters, nil
	}

	log.Debug("prepareEpochFromStorage for shuffled out", "initial shard id", e.baseData.shardId, "new shard id", newShardId)
	e.baseData.shardId = newShardId
	err = e.createSyncers()
	if err != nil {
		return Parameters{}, err
	}

	e.syncedHeaders, err = e.syncHeadersFrom(e.epochStartMeta)
	if err != nil {
		return Parameters{}, err
	}

	prevEpochStartMetaHash := e.epochStartMeta.EpochStart.Economics.PrevEpochStartHash
	prevEpochStartMeta, ok := e.syncedHeaders[string(prevEpochStartMetaHash)].(*block.MetaBlock)
	if !ok {
		return Parameters{}, epochStart.ErrWrongTypeAssertion
	}
	e.prevEpochStartMeta = prevEpochStartMeta

	e.shardCoordinator, err = sharding.NewMultiShardCoordinator(e.baseData.numberOfShards, e.baseData.shardId)
	if err != nil {
		return Parameters{}, err
	}

	if e.shardCoordinator.SelfId() != e.genesisShardCoordinator.SelfId() {
		err = e.createTriesForNewShardId(e.shardCoordinator.SelfId())
		if err != nil {
			return Parameters{}, err
		}
	}

	err = e.messenger.CreateTopic(core.ConsensusTopic+e.shardCoordinator.CommunicationIdentifier(e.shardCoordinator.SelfId()), true)
	if err != nil {
		return Parameters{}, err
	}

	if e.shardCoordinator.SelfId() == core.MetachainShardId {
		err = e.requestAndProcessForMeta()
		if err != nil {
			return Parameters{}, err
		}
	} else {
		err = e.requestAndProcessForShard()
		if err != nil {
			return Parameters{}, err
		}
	}

	parameters := Parameters{
		Epoch:       e.baseData.lastEpoch,
		SelfShardId: e.shardCoordinator.SelfId(),
		NumOfShards: e.shardCoordinator.NumberOfShards(),
		NodesConfig: e.nodesConfig,
	}
	return parameters, nil
}

func (e *epochStartBootstrap) checkIfShuffledOut(
	pubKey []byte,
	nodesConfig *sharding.NodesCoordinatorRegistry,
) (uint32, bool) {
	epochIDasString := fmt.Sprint(e.baseData.lastEpoch)
	epochConfig := nodesConfig.EpochsConfig[epochIDasString]

	newShardId, isWaitingForShard := checkIfPubkeyIsInMap(pubKey, epochConfig.WaitingValidators)
	if isWaitingForShard {
		isShuffledOut := newShardId != e.baseData.shardId
		return newShardId, isShuffledOut
	}

	newShardId, isEligibleForShard := checkIfPubkeyIsInMap(pubKey, epochConfig.EligibleValidators)
	if isEligibleForShard {
		isShuffledOut := newShardId != e.baseData.shardId
		return newShardId, isShuffledOut
	}

	return e.baseData.shardId, false
}

func checkIfPubkeyIsInMap(
	pubKey []byte,
	allShardList map[string][]*sharding.SerializableValidator,
) (uint32, bool) {
	for shardIdStr, validatorList := range allShardList {
		isValidatorInList := checkIfValidatorIsInList(pubKey, validatorList)
		if isValidatorInList {
			shardId, err := strconv.ParseInt(shardIdStr, 10, 64)
			if err != nil {
				log.Error("checkIfIsValidatorForEpoch parsing string to int error should not happen", "err", err)
				return 0, false
			}

			return uint32(shardId), true
		}
	}
	return 0, false
}

func checkIfValidatorIsInList(
	pubKey []byte,
	validatorList []*sharding.SerializableValidator,
) bool {
	for _, validator := range validatorList {
		if bytes.Equal(pubKey, validator.PubKey) {
			return true
		}
	}
	return false
}

func (e *epochStartBootstrap) getLastBootstrapData(storer storage.Storer) (*bootstrapStorage.BootstrapData, *sharding.NodesCoordinatorRegistry, error) {
	bootStorer, err := bootstrapStorage.NewBootstrapStorer(e.marshalizer, storer)
	if err != nil {
		return nil, nil, err
	}

	highestRound := bootStorer.GetHighestRound()
	bootstrapData, err := bootStorer.Get(highestRound)
	if err != nil {
		return nil, nil, err
	}

	ncInternalkey := append([]byte(core.NodesCoordinatorRegistryKeyPrefix), bootstrapData.NodesCoordinatorConfigKey...)
	data, err := storer.SearchFirst(ncInternalkey)
	if err != nil {
		log.Debug("getLastBootstrapData", "key", ncInternalkey, "error", err)
		return nil, nil, err
	}

	config := &sharding.NodesCoordinatorRegistry{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, nil, err
	}

	return &bootstrapData, config, nil
}

func (e *epochStartBootstrap) getEpochStartMetaFromStorage(storer storage.Storer) (*block.MetaBlock, error) {
	epochIdentifier := core.EpochStartIdentifier(e.baseData.lastEpoch)
	data, err := storer.SearchFirst([]byte(epochIdentifier))
	if err != nil {
		log.Debug("getEpochStartMetaFromStorage", "key", epochIdentifier, "error", err)
		return nil, err
	}

	metaBlock := &block.MetaBlock{}
	err = e.marshalizer.Unmarshal(metaBlock, data)
	if err != nil {
		return nil, err
	}

	return metaBlock, nil
}
