package sharding

import (
	"bytes"
	"fmt"
	"sort"
	"sync"

	"github.com/ElrondNetwork/elrond-go/core"
)

var _ NodesShuffler = (*randXORShuffler)(nil)

type shuffleNodesArg struct {
	eligible          map[uint32][]Validator
	waiting           map[uint32][]Validator
	unstakeLeaving    []Validator
	additionalLeaving []Validator
	newNodes          []Validator
	randomness        []byte
	nodesMeta         uint32
	nodesPerShard     uint32
	nbShards          uint32
	distributor       ValidatorsDistributor
}

// TODO: Decide if transaction load statistics will be used for limiting the number of shards
type randXORShuffler struct {
	// TODO: remove the references to this constant and the distributor
	// when reinitialization of node in new shard is implemented
	shuffleBetweenShards bool
	validatorDistributor ValidatorsDistributor

	adaptivity        bool
	nodesShard        uint32
	nodesMeta         uint32
	shardHysteresis   uint32
	metaHysteresis    uint32
	mutShufflerParams sync.RWMutex
}

// NewXorValidatorsShuffler creates a validator shuffler that uses a XOR between validator key and a given
// random number to do the shuffling
func NewXorValidatorsShuffler(
	nodesShard uint32,
	nodesMeta uint32,
	hysteresis float32,
	adaptivity bool,
	shuffleBetweenShards bool,
) *randXORShuffler {
	log.Debug("Shuffler created", "shuffleBetweenShards", shuffleBetweenShards)
	rxs := &randXORShuffler{shuffleBetweenShards: shuffleBetweenShards}

	rxs.UpdateParams(nodesShard, nodesMeta, hysteresis, adaptivity)

	if rxs.shuffleBetweenShards {
		rxs.validatorDistributor = &CrossShardValidatorDistributor{}
	} else {
		rxs.validatorDistributor = &IntraShardValidatorDistributor{}
	}

	return rxs
}

// UpdateParams updates the shuffler parameters
// Should be called when new params are agreed through governance
func (rxs *randXORShuffler) UpdateParams(
	nodesShard uint32,
	nodesMeta uint32,
	hysteresis float32,
	adaptivity bool,
) {
	// TODO: are there constraints we want to enforce? e.g min/max hysteresis
	shardHysteresis := uint32(float32(nodesShard) * hysteresis)
	metaHysteresis := uint32(float32(nodesMeta) * hysteresis)

	rxs.mutShufflerParams.Lock()
	rxs.shardHysteresis = shardHysteresis
	rxs.metaHysteresis = metaHysteresis
	rxs.nodesShard = nodesShard
	rxs.nodesMeta = nodesMeta
	rxs.adaptivity = adaptivity
	rxs.mutShufflerParams.Unlock()
}

// UpdateNodeLists shuffles the nodes and returns the lists with the new nodes configuration
// The function needs to ensure that:
//      1.  Old eligible nodes list will have up to shuffleOutThreshold percent nodes shuffled out from each shard
//      2.  The leaving nodes are checked against the eligible nodes and waiting nodes and removed if present from the
//          pools and leaving nodes list (if remaining nodes can still sustain the shard)
//      3.  shuffledOutNodes = oldEligibleNodes + waitingListNodes - minNbNodesPerShard (for each shard)
//      4.  Old waiting nodes list for each shard will be added to the remaining eligible nodes list
//      5.  The new nodes are equally distributed among the existing shards into waiting lists
//      6.  The shuffled out nodes are distributed among the existing shards into waiting lists.
//          We may have three situations:
//          a)  In case (shuffled out nodes + new nodes) > (nbShards * perShardHysteresis + minNodesPerShard) then
//              we need to prepare for a split event, so a higher percentage of nodes need to be directed to the shard
//              that will be split.
//          b)  In case (shuffled out nodes + new nodes) < (nbShards * perShardHysteresis) then we can immediately
//              execute the shard merge
//          c)  No change in the number of shards then nothing extra needs to be done
func (rxs *randXORShuffler) UpdateNodeLists(args ArgsUpdateNodes) (*ResUpdateNodes, error) {
	eligibleAfterReshard := copyValidatorMap(args.Eligible)
	waitingAfterReshard := copyValidatorMap(args.Waiting)

	args.AdditionalLeaving = removeDupplicates(args.UnStakeLeaving, args.AdditionalLeaving)
	totalLeavingNum := len(args.AdditionalLeaving) + len(args.UnStakeLeaving)

	newNbShards := rxs.computeNewShards(
		args.Eligible,
		args.Waiting,
		len(args.NewNodes),
		totalLeavingNum,
		args.NbShards,
	)

	rxs.mutShufflerParams.RLock()
	canSplit := rxs.adaptivity && newNbShards > args.NbShards
	canMerge := rxs.adaptivity && newNbShards < args.NbShards
	nodesPerShard := rxs.nodesShard
	nodesMeta := rxs.nodesMeta
	rxs.mutShufflerParams.RUnlock()

	if canSplit {
		eligibleAfterReshard, waitingAfterReshard = rxs.splitShards(args.Eligible, args.Waiting, newNbShards)
	}
	if canMerge {
		eligibleAfterReshard, waitingAfterReshard = rxs.mergeShards(args.Eligible, args.Waiting, newNbShards)
	}

	return shuffleNodes(shuffleNodesArg{
		eligible:          eligibleAfterReshard,
		waiting:           waitingAfterReshard,
		unstakeLeaving:    args.UnStakeLeaving,
		additionalLeaving: args.AdditionalLeaving,
		newNodes:          args.NewNodes,
		randomness:        args.Rand,
		nodesMeta:         nodesMeta,
		nodesPerShard:     nodesPerShard,
		nbShards:          args.NbShards,
		distributor:       rxs.validatorDistributor,
	})
}

func removeDupplicates(unstake []Validator, additionalLeaving []Validator) []Validator {
	additionalCopy := make([]Validator, 0, len(additionalLeaving))
	additionalCopy = append(additionalCopy, additionalLeaving...)

	for _, unstakeValidator := range unstake {
		for i := len(additionalCopy) - 1; i >= 0; i-- {
			if bytes.Equal(unstakeValidator.PubKey(), additionalCopy[i].PubKey()) {
				additionalCopy = removeValidatorFromList(additionalCopy, i)
			}
		}
	}

	return additionalCopy
}

func removeNodesFromMap(
	existingNodes map[uint32][]Validator,
	leavingNodes []Validator,
	numToRemove map[uint32]int,
) (map[uint32][]Validator, []Validator) {
	sortedShardIds := sortKeys(existingNodes)
	numRemoved := 0

	for _, shardId := range sortedShardIds {
		numToRemoveOnShard := numToRemove[shardId]
		leavingNodes, numRemoved = removeNodesFromShard(existingNodes, leavingNodes, shardId, numToRemoveOnShard)
		numToRemove[shardId] -= numRemoved
	}

	return existingNodes, leavingNodes
}

func removeNodesFromShard(existingNodes map[uint32][]Validator, leavingNodes []Validator, shard uint32, nbToRemove int) ([]Validator, int) {
	vList := existingNodes[shard]
	if len(leavingNodes) < nbToRemove {
		nbToRemove = len(leavingNodes)
	}

	vList, removedNodes := removeValidatorsFromList(existingNodes[shard], leavingNodes, nbToRemove)
	leavingNodes, _ = removeValidatorsFromList(leavingNodes, removedNodes, len(removedNodes))
	existingNodes[shard] = vList
	return leavingNodes, len(removedNodes)
}

// IsInterfaceNil verifies if the underlying object is nil
func (rxs *randXORShuffler) IsInterfaceNil() bool {
	return rxs == nil
}

func shuffleNodes(arg shuffleNodesArg) (*ResUpdateNodes, error) {
	allLeaving := append(arg.unstakeLeaving, arg.additionalLeaving...)

	waitingCopy := copyValidatorMap(arg.waiting)
	eligibleCopy := copyValidatorMap(arg.eligible)

	createListsForAllShards(waitingCopy, arg.nbShards)

	numToRemove, err := computeNumToRemove(arg)
	if err != nil {
		return nil, err
	}

	remainingUnstakeLeaving, _ := removeLeavingNodesNotExistingInEligibleOrWaiting(arg.unstakeLeaving, waitingCopy, eligibleCopy)
	remainingAdditionalLeaving, _ := removeLeavingNodesNotExistingInEligibleOrWaiting(arg.additionalLeaving, waitingCopy, eligibleCopy)

	newEligible, newWaiting, stillRemainingUnstakeLeaving := removeLeavingNodesFromValidatorMaps(eligibleCopy, waitingCopy, numToRemove, remainingUnstakeLeaving)
	newEligible, newWaiting, stillRemainingAdditionalLeaving := removeLeavingNodesFromValidatorMaps(newEligible, newWaiting, numToRemove, remainingAdditionalLeaving)

	stillRemainingInLeaving := append(stillRemainingUnstakeLeaving, stillRemainingAdditionalLeaving...)

	shuffledOutMap, newEligible := shuffleOutNodes(newEligible, numToRemove, arg.randomness)

	err = moveNodesToMap(newEligible, newWaiting)
	if err != nil {
		log.Warn("moveNodesToMap failed", "error", err)
	}
	err = distributeValidators(newWaiting, arg.newNodes, arg.randomness)
	if err != nil {
		log.Warn("distributeValidators newNodes failed", "error", err)
	}

	err = arg.distributor.DistributeValidators(newWaiting, shuffledOutMap, arg.randomness)
	if err != nil {
		log.Warn("distributeValidators shuffledOut failed", "error", err)
	}

	actualLeaving, _ := removeValidatorsFromList(allLeaving, stillRemainingInLeaving, len(stillRemainingInLeaving))

	return &ResUpdateNodes{
		Eligible:       newEligible,
		Waiting:        newWaiting,
		Leaving:        actualLeaving,
		StillRemaining: stillRemainingInLeaving,
	}, nil
}

func createListsForAllShards(shardMap map[uint32][]Validator, shards uint32) {
	for shardId := uint32(0); shardId < shards; shardId++ {
		if shardMap[shardId] == nil {
			shardMap[shardId] = make([]Validator, 0)
		}
	}

	if shardMap[core.MetachainShardId] == nil {
		shardMap[core.MetachainShardId] = make([]Validator, 0)
	}
}

func computeNumToRemove(arg shuffleNodesArg) (map[uint32]int, error) {
	numToRemove := make(map[uint32]int)
	if arg.nbShards == 0 {
		return numToRemove, nil
	}

	for shardId := uint32(0); shardId < arg.nbShards; shardId++ {
		maxToRemove, err := computeNumToRemovePerShard(
			len(arg.eligible[shardId]),
			len(arg.waiting[shardId]),
			int(arg.nodesPerShard))
		if err != nil {
			return nil, fmt.Errorf("%w shard=%v", err, shardId)
		}
		numToRemove[shardId] = maxToRemove
	}

	maxToRemove, err := computeNumToRemovePerShard(
		len(arg.eligible[core.MetachainShardId]),
		len(arg.waiting[core.MetachainShardId]),
		int(arg.nodesMeta))
	if err != nil {
		return nil, fmt.Errorf("%w shard=%v", err, core.MetachainShardId)
	}
	numToRemove[core.MetachainShardId] = maxToRemove

	return numToRemove, nil
}

func computeNumToRemovePerShard(numEligible int, numWaiting int, nodesPerShard int) (int, error) {
	notEnoughValidatorsInShard := numEligible+numWaiting < nodesPerShard
	if notEnoughValidatorsInShard {
		return 0, ErrSmallShardEligibleListSize
	}
	return numEligible + numWaiting - nodesPerShard, nil
}

func removeLeavingNodesNotExistingInEligibleOrWaiting(
	leavingValidators []Validator,
	waiting map[uint32][]Validator,
	eligible map[uint32][]Validator,
) ([]Validator, []Validator) {
	notFoundValidators := make([]Validator, 0)

	for _, v := range leavingValidators {
		found, _ := searchInMap(waiting, v.PubKey())
		if found {
			continue
		}
		found, _ = searchInMap(eligible, v.PubKey())
		if !found {
			log.Debug("Leaving validator not found in waiting or eligible", "pk", v.PubKey())
			notFoundValidators = append(notFoundValidators, v)
		}
	}

	return removeValidatorsFromList(leavingValidators, notFoundValidators, len(notFoundValidators))
}

func removeLeavingNodesFromValidatorMaps(
	eligible map[uint32][]Validator,
	waiting map[uint32][]Validator,
	numToRemove map[uint32]int,
	leaving []Validator,
) (map[uint32][]Validator, map[uint32][]Validator, []Validator) {

	stillRemainingInLeaving := make([]Validator, len(leaving))
	copy(stillRemainingInLeaving, leaving)

	waiting, stillRemainingInLeaving = removeNodesFromMap(waiting, stillRemainingInLeaving, numToRemove)
	eligible, stillRemainingInLeaving = removeNodesFromMap(eligible, stillRemainingInLeaving, numToRemove)
	return eligible, waiting, stillRemainingInLeaving
}

// computeNewShards determines the new number of shards based on the number of nodes in the network
func (rxs *randXORShuffler) computeNewShards(
	eligible map[uint32][]Validator,
	waiting map[uint32][]Validator,
	numNewNodes int,
	numLeavingNodes int,
	nbShards uint32,
) uint32 {

	nbEligible := 0
	nbWaiting := 0
	for shard := range eligible {
		nbEligible += len(eligible[shard])
		nbWaiting += len(waiting[shard])
	}

	nodesNewEpoch := uint32(nbEligible + nbWaiting + numNewNodes - numLeavingNodes)

	rxs.mutShufflerParams.RLock()
	maxNodesMeta := rxs.nodesMeta + rxs.metaHysteresis
	maxNodesShard := rxs.nodesShard + rxs.shardHysteresis
	nodesForSplit := (nbShards+1)*maxNodesShard + maxNodesMeta
	nodesForMerge := nbShards*rxs.nodesShard + rxs.nodesMeta
	rxs.mutShufflerParams.RUnlock()

	nbShardsNew := nbShards
	if nodesNewEpoch > nodesForSplit {
		nbNodesWithoutMaxMeta := nodesNewEpoch - maxNodesMeta
		nbShardsNew = nbNodesWithoutMaxMeta / maxNodesShard

		return nbShardsNew
	}

	if nodesNewEpoch < nodesForMerge {
		return nbShardsNew - 1
	}

	return nbShardsNew
}

// shuffleOutNodes shuffles the list of eligible validators in each shard and returns the map of shuffled out
// validators
func shuffleOutNodes(
	eligible map[uint32][]Validator,
	numToShuffle map[uint32]int,
	randomness []byte,
) (map[uint32][]Validator, map[uint32][]Validator) {
	shuffledOutMap := make(map[uint32][]Validator)
	newEligible := make(map[uint32][]Validator)
	var shardShuffledOut []Validator

	sortedShardIds := sortKeys(eligible)
	for _, shardId := range sortedShardIds {
		validators := eligible[shardId]
		shardShuffledOut, validators = shuffleOutShard(validators, numToShuffle[shardId], randomness)
		shuffledOutMap[shardId] = shardShuffledOut
		newEligible[shardId], _ = removeValidatorsFromList(validators, shardShuffledOut, len(shardShuffledOut))
	}

	return shuffledOutMap, newEligible
}

// shuffleOutShard selects the validators to be shuffled out from a shard
func shuffleOutShard(
	validators []Validator,
	validatorsToSelect int,
	randomness []byte,
) ([]Validator, []Validator) {
	if len(validators) < validatorsToSelect {
		validatorsToSelect = len(validators)
	}

	shardShuffledEligible := shuffleList(validators, randomness)
	shardShuffledOut := shardShuffledEligible[:validatorsToSelect]
	remainingEligible := shardShuffledEligible[validatorsToSelect:]

	return shardShuffledOut, remainingEligible
}

// shuffleList returns a shuffled list of validators.
// The shuffling is done based by xor-ing the randomness with the
// public keys of validators and sorting the validators depending on
// the xor result.
func shuffleList(validators []Validator, randomness []byte) []Validator {
	keys := make([]string, len(validators))
	mapValidators := make(map[string]Validator)

	for i, v := range validators {
		keys[i] = string(xorBytes(v.PubKey(), randomness))
		mapValidators[keys[i]] = v
	}

	sort.Strings(keys)

	result := make([]Validator, len(validators))
	for i := 0; i < len(validators); i++ {
		result[i] = mapValidators[keys[i]]
	}

	return result
}

func removeValidatorsFromList(
	validatorList []Validator,
	validatorsToRemove []Validator,
	maxToRemove int,
) ([]Validator, []Validator) {
	resultedList := make([]Validator, 0)
	resultedList = append(resultedList, validatorList...)
	removed := make([]Validator, 0)

	for _, valToRemove := range validatorsToRemove {
		if len(removed) == maxToRemove {
			break
		}

		for i := len(resultedList) - 1; i >= 0; i-- {
			val := resultedList[i]
			if bytes.Equal(val.PubKey(), valToRemove.PubKey()) {
				resultedList = removeValidatorFromList(resultedList, i)

				removed = append(removed, val)
				break
			}
		}
	}

	return resultedList, removed
}

// removeValidatorFromList replaces the element at given index with the last element in the slice and returns a slice
// with a decremented length.The order in the list is important as long as it is kept the same for all validators,
// so not critical to maintain the original order inside the list, as that would be slower.
//
// Attention: The slice given as parameter will have its element on position index swapped with the last element
func removeValidatorFromList(validatorList []Validator, index int) []Validator {
	indexNotOK := index > len(validatorList)-1 || index < 0
	if indexNotOK {
		return validatorList
	}

	validatorList[index] = validatorList[len(validatorList)-1]
	return validatorList[:len(validatorList)-1]
}

func removeValidatorFromListKeepOrder(validatorList []Validator, index int) []Validator {
	indexNotOK := index > len(validatorList)-1 || index < 0
	if indexNotOK {
		return validatorList
	}

	return append(validatorList[:index], validatorList[index+1:]...)
}

// xorBytes XORs two byte arrays up to the shortest length of the two, and returns the resulted XORed bytes.
func xorBytes(a []byte, b []byte) []byte {
	lenA := len(a)
	lenB := len(b)
	minLen := lenA

	if lenB < minLen {
		minLen = lenB
	}

	result := make([]byte, minLen)
	for i := 0; i < minLen; i++ {
		result[i] = a[i] ^ b[i]
	}

	return result
}

// splitShards prepares for the shards split, or if already prepared does the split returning the resulting
// shards configuration for eligible and waiting lists
func (rxs *randXORShuffler) splitShards(
	eligible map[uint32][]Validator,
	waiting map[uint32][]Validator,
	_ uint32,
) (map[uint32][]Validator, map[uint32][]Validator) {
	log.Error(ErrNotImplemented.Error())

	// TODO: do the split
	return copyValidatorMap(eligible), copyValidatorMap(waiting)
}

// mergeShards merges the required shards, returning the resulting shards configuration for eligible and waiting lists
func (rxs *randXORShuffler) mergeShards(
	eligible map[uint32][]Validator,
	waiting map[uint32][]Validator,
	_ uint32,
) (map[uint32][]Validator, map[uint32][]Validator) {
	log.Error(ErrNotImplemented.Error())

	// TODO: do the merge
	return copyValidatorMap(eligible), copyValidatorMap(waiting)
}

// copyValidatorMap creates a copy for the Validators map, creating copies for each of the lists for each shard
func copyValidatorMap(validators map[uint32][]Validator) map[uint32][]Validator {
	result := make(map[uint32][]Validator)

	for k, v := range validators {
		elems := make([]Validator, 0)
		result[k] = append(elems, v...)
	}

	return result
}

// moveNodesToMap moves the validators in the waiting list to corresponding eligible list
func moveNodesToMap(destination map[uint32][]Validator, source map[uint32][]Validator) error {
	if destination == nil {
		return ErrNilOrEmptyDestinationForDistribute
	}

	for k, v := range source {
		destination[k] = append(destination[k], v...)
		source[k] = make([]Validator, 0)
	}
	return nil
}

// distributeNewNodes distributes a list of validators to the given validators map
func distributeValidators(destLists map[uint32][]Validator, validators []Validator, randomness []byte) error {
	if len(destLists) == 0 {
		return ErrNilOrEmptyDestinationForDistribute
	}

	// if there was a split or a merge, eligible map should already have a different nb of keys (shards)
	shuffledValidators := shuffleList(validators, randomness)
	var shardId uint32

	sortedShardIds := sortKeys(destLists)
	destLength := uint32(len(sortedShardIds))

	for i, v := range shuffledValidators {
		shardId = sortedShardIds[uint32(i)%destLength]
		destLists[shardId] = append(destLists[shardId], v)
	}

	return nil
}

func sortKeys(nodes map[uint32][]Validator) []uint32 {
	keys := make([]uint32, 0, len(nodes))
	for k := range nodes {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	return keys
}
