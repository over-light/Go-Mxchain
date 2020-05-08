package sharding

import (
	"encoding/hex"
	"strconv"

	"github.com/ElrondNetwork/elrond-go/core"
)

func computeStartIndexAndNumAppearancesForValidator(expEligibleList []uint32, idx int64) (int64, int64) {
	val := expEligibleList[idx]
	startIdx := int64(0)
	listLen := int64(len(expEligibleList))

	for i := idx - 1; i >= 0; i-- {
		if expEligibleList[i] != val {
			startIdx = i + 1
			break
		}
	}

	endIdx := listLen - 1
	for i := idx + 1; i < listLen; i++ {
		if expEligibleList[i] != val {
			endIdx = i - 1
			break
		}
	}

	return startIdx, endIdx - startIdx + 1
}

func displayValidatorsForRandomness(validators []Validator, randomness []byte) {
	strValidators := ""

	for _, v := range validators {
		strValidators += "\n" + hex.EncodeToString(v.PubKey())
	}

	log.Trace("selectValidators", "randomness", randomness, "validators", strValidators)
}

func displayNodesConfiguration(
	eligible map[uint32][]Validator,
	waiting map[uint32][]Validator,
	leaving map[uint32][]Validator,
	actualRemaining map[uint32][]Validator,
	nbShards uint32,
) {
	for shard := uint32(0); shard <= nbShards; shard++ {
		shardID := shard
		if shardID == nbShards {
			shardID = core.MetachainShardId
		}
		for _, v := range eligible[shardID] {
			pk := v.PubKey()
			log.Debug("eligible", "pk", pk, "shardID", shardID)
		}
		for _, v := range waiting[shardID] {
			pk := v.PubKey()
			log.Debug("waiting", "pk", pk, "shardID", shardID)
		}
		for _, v := range leaving[shardID] {
			pk := v.PubKey()
			log.Debug("leaving", "pk", pk, "shardID", shardID)
		}
		for _, v := range actualRemaining[shardID] {
			pk := v.PubKey()
			log.Debug("actually remaining", "pk", pk, "shardID", shardID)
		}
	}
}

func SerializableValidatorsToValidators(nodeRegistryValidators map[string][]*SerializableValidator) (map[uint32][]Validator, error) {
	validators := make(map[uint32][]Validator)
	for shardId, shardValidators := range nodeRegistryValidators {
		newValidators, err := SerializableShardValidatorListToValidatorList(shardValidators)
		if err != nil {
			return nil, err
		}
		shardIdInt, err := strconv.ParseUint(shardId, 10, 32)
		if err != nil {
			return nil, err
		}
		validators[uint32(shardIdInt)] = newValidators
	}

	return validators, nil
}

func SerializableShardValidatorListToValidatorList(shardValidators []*SerializableValidator) ([]Validator, error) {
	newValidators := make([]Validator, len(shardValidators))
	for i, validator := range shardValidators {
		v, err := NewValidator(validator.PubKey, validator.Chances, validator.Index)
		if err != nil {
			return nil, err
		}
		newValidators[i] = v
	}
	return newValidators, nil
}
