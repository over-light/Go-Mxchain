package sharding

import (
	"bytes"
	"crypto/rand"
	"reflect"
	"sort"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/stretchr/testify/assert"
)

var firstArray = []byte{0xFF, 0xFF, 0xAA, 0xAA, 0x00, 0x00}
var secondArray = []byte{0xFF, 0x00, 0xAA, 0x55, 0x00, 0xFF}
var expectedArray = []byte{0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF}

func generateRandomByteArray(size int) []byte {
	r := make([]byte, size)
	_, _ = rand.Read(r)

	return r
}

func generateValidatorList(number int) []Validator {
	v := make([]Validator, number)

	for i := 0; i < number; i++ {
		v[i] = &validator{
			pubKey: generateRandomByteArray(32),
		}
	}

	return v
}

func generateValidatorMap(
	nodesPerShard int,
	nbShards uint32,
) map[uint32][]Validator {
	validatorsMap := make(map[uint32][]Validator)

	for i := uint32(0); i < nbShards; i++ {
		validatorsMap[i] = generateValidatorList(nodesPerShard)
	}

	validatorsMap[core.MetachainShardId] = generateValidatorList(nodesPerShard)

	return validatorsMap
}

func contains(a []Validator, b []Validator) bool {
	var found bool
	for _, va := range a {
		found = false
		for _, vb := range b {
			if reflect.DeepEqual(va, vb) {
				found = true
				break
			}
		}
		if !found {
			return found
		}
	}

	return found
}

func testRemoveValidators(
	t *testing.T,
	initialValidators []Validator,
	validatorsToRemove []Validator,
	remaining []Validator,
	removed []Validator,
	maxToRemove int,
) {
	nbRemoved := maxToRemove
	if nbRemoved > len(validatorsToRemove) {
		nbRemoved = len(validatorsToRemove)
	}

	assert.Equal(t, nbRemoved, len(removed))
	assert.Equal(t, len(initialValidators)-len(remaining), nbRemoved)

	all := append(remaining, removed...)
	assert.True(t, contains(all, initialValidators))
	assert.Equal(t, len(initialValidators), len(all))
}

func testDistributeValidators(
	t *testing.T,
	initialMap map[uint32][]Validator,
	resultedMap map[uint32][]Validator,
	distributedNodes []Validator,
) {
	totalResultingValidators := make([]Validator, 0)
	totalLen := 0
	for _, valList := range resultedMap {
		totalResultingValidators = append(totalResultingValidators, valList...)
		totalLen += len(valList)
	}

	totalValidators := make([]Validator, 0)
	for _, valList := range initialMap {
		totalValidators = append(totalValidators, valList...)
	}
	assert.Equal(t, len(totalValidators)+len(distributedNodes), totalLen)

	totalValidators = append(totalValidators, distributedNodes...)
	assert.True(t, contains(totalResultingValidators, totalValidators))
}

func numberMatchingNodes(searchList []Validator, toFind []Validator) int {
	nbFound := 0
	for _, v1 := range toFind {
		for _, v2 := range searchList {
			if bytes.Equal(v1.PubKey(), v2.PubKey()) {
				nbFound++
				break
			}
		}
	}

	return nbFound
}

func testLeaving(
	t *testing.T,
	eligible map[uint32][]Validator,
	waiting map[uint32][]Validator,
	prevLeaving []Validator,
	newLeaving []Validator,
) (int, map[uint32]int) {
	nbLeavingPerShard := make(map[uint32]int)

	nbLeavingFromEligible := 0
	for i, eligibleList := range eligible {
		nbWantingToLeaveFromList := numberMatchingNodes(eligibleList, prevLeaving)
		maxAllowedToLeaveFromList := len(waiting[i])
		nbLeaving := nbWantingToLeaveFromList
		if nbLeaving > maxAllowedToLeaveFromList {
			nbLeaving = maxAllowedToLeaveFromList
		}

		nbLeavingPerShard[i] += nbLeaving
		nbLeavingFromEligible += nbLeaving
	}
	assert.Equal(t, nbLeavingFromEligible, len(prevLeaving)-len(newLeaving))

	return nbLeavingFromEligible, nbLeavingPerShard
}

func testShuffledOut(
	t *testing.T,
	eligibleMap map[uint32][]Validator,
	waitingMap map[uint32][]Validator,
	newEligible map[uint32][]Validator,
	shuffledOut []Validator,
	prevleaving []Validator,
	newleaving []Validator,
) {
	nbAllLeaving, _ := testLeaving(t, eligibleMap, waitingMap, prevleaving, newleaving)
	allWaiting := getValidatorsInMap(waitingMap)
	allEligible := getValidatorsInMap(eligibleMap)
	assert.Equal(t, len(shuffledOut)+nbAllLeaving, len(allWaiting))

	allNewEligible := getValidatorsInMap(newEligible)
	assert.Equal(t, len(allEligible)-len(shuffledOut)-nbAllLeaving, len(allNewEligible))

	newNodes := append(allNewEligible, shuffledOut...)
	assert.NotEqual(t, allEligible, newNodes)
	assert.True(t, contains(newNodes, allEligible))
}

func createXorShufflerInter() *randXORShuffler {
	shuffler := NewXorValidatorsShuffler(
		100,
		100,
		0.2,
		false,
	)

	shuffler.shuffleBetweenShards = true

	return shuffler
}

func createXorShufflerIntraShards() *randXORShuffler {
	shuffler := NewXorValidatorsShuffler(
		100,
		100,
		0.2,
		false,
	)

	return shuffler
}

func getValidatorsInMap(valMap map[uint32][]Validator) []Validator {
	result := make([]Validator, 0)

	for _, valList := range valMap {
		result = append(result, valList...)
	}

	return result
}

func Test_xorBytes_SameLen(t *testing.T) {
	t.Parallel()

	result := xorBytes(firstArray, secondArray)

	assert.Equal(t, expectedArray, result)
}

func Test_xorBytes_FirstLowerLen(t *testing.T) {
	t.Parallel()

	result := xorBytes(firstArray[:len(firstArray)-1], secondArray)

	assert.Equal(t, expectedArray[:len(expectedArray)-1], result)
}

func Test_xorBytes_SecondLowerLen(t *testing.T) {
	t.Parallel()

	result := xorBytes(firstArray, secondArray[:len(secondArray)-1])

	assert.Equal(t, expectedArray[:len(expectedArray)-1], result)
}

func Test_xorBytes_FirstEmpty(t *testing.T) {
	t.Parallel()

	result := xorBytes([]byte{}, secondArray)

	assert.Equal(t, []byte{}, result)
}

func Test_xorBytes_SecondEmpty(t *testing.T) {
	result := xorBytes(firstArray, []byte{})

	assert.Equal(t, []byte{}, result)
}

func Test_xorBytes_FirstNil(t *testing.T) {
	t.Parallel()

	result := xorBytes(nil, secondArray)

	assert.Equal(t, []byte{}, result)
}

func Test_xorBytes_SecondNil(t *testing.T) {
	t.Parallel()

	result := xorBytes(firstArray, nil)

	assert.Equal(t, []byte{}, result)
}

func Test_copyValidatorMap(t *testing.T) {
	t.Parallel()

	valMap := generateValidatorMap(30, 2)
	v2 := copyValidatorMap(valMap)
	assert.Equal(t, valMap, v2)

	valMap[0] = valMap[0][1:]
	assert.NotEqual(t, valMap, v2)
}

func Test_promoteWaitingToEligibleEmptyList(t *testing.T) {
	t.Parallel()

	eligibleMap := generateValidatorMap(30, 2)
	waitingMap := generateValidatorMap(0, 2)
	eligibleMapCopy := copyValidatorMap(eligibleMap)

	for k := range eligibleMap {
		assert.Equal(t, eligibleMap[k], eligibleMapCopy[k])
		assert.Empty(t, waitingMap[k])
	}
}

func Test_promoteWaitingToEligible(t *testing.T) {
	t.Parallel()

	eligibleMap := generateValidatorMap(30, 2)
	waitingMap := generateValidatorMap(22, 2)

	eligibleMapCopy := copyValidatorMap(eligibleMap)
	waitingMapCopy := copyValidatorMap(waitingMap)

	moveNodesToMap(eligibleMap, waitingMap)

	for k := range eligibleMap {
		assert.Equal(t, eligibleMap[k], append(eligibleMapCopy[k], waitingMapCopy[k]...))
		assert.Empty(t, waitingMap[k])
	}
}

func Test_removeValidatorFromListFirst(t *testing.T) {
	t.Skip()
	t.Parallel()

	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, len(validators))
	_ = copy(validatorsCopy, validators)

	v := removeValidatorFromList(validators, 0)
	assert.Equal(t, validatorsCopy[len(validatorsCopy)-1], v[0])
	assert.NotEqual(t, validatorsCopy[0], v[0])
	assert.Equal(t, len(validatorsCopy)-1, len(v))

	for i := 1; i < len(v); i++ {
		assert.Equal(t, validatorsCopy[i], v[i])
	}
}

func Test_removeValidatorFromListLast(t *testing.T) {
	t.Parallel()

	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, len(validators))
	_ = copy(validatorsCopy, validators)

	v := removeValidatorFromList(validators, len(validators)-1)
	assert.Equal(t, len(validatorsCopy)-1, len(v))
	assert.Equal(t, validatorsCopy[:len(validatorsCopy)-1], v)
}

func Test_removeValidatorFromListMiddle(t *testing.T) {
	t.Skip()
	t.Parallel()

	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, len(validators))
	_ = copy(validatorsCopy, validators)

	v := removeValidatorFromList(validators, len(validators)/2)
	assert.Equal(t, len(validatorsCopy)-1, len(v))
	assert.Equal(t, validatorsCopy[len(validatorsCopy)-1], v[len(validatorsCopy)/2])
}

func Test_removeValidatorFromListIndexNegativeNoAction(t *testing.T) {
	t.Parallel()

	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, len(validators))
	_ = copy(validatorsCopy, validators)

	v := removeValidatorFromList(validators, -1)
	assert.Equal(t, len(validatorsCopy), len(v))
	assert.Equal(t, validatorsCopy, v)
}

func Test_removeValidatorFromListIndexTooBigNoAction(t *testing.T) {
	t.Parallel()

	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, len(validators))
	_ = copy(validatorsCopy, validators)

	v := removeValidatorFromList(validators, len(validators))
	assert.Equal(t, len(validatorsCopy), len(v))
	assert.Equal(t, validatorsCopy, v)
}

func Test_removeValidatorsFromListRemoveFromStart(t *testing.T) {
	t.Parallel()

	validatorsToRemoveFromStart := 3
	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, len(validators))
	validatorsToRemove := make([]Validator, 0)

	_ = copy(validatorsCopy, validators)
	validatorsToRemove = append(validatorsToRemove, validators[:validatorsToRemoveFromStart]...)

	v, removed := removeValidatorsFromList(validators, validatorsToRemove, len(validatorsToRemove))
	testRemoveValidators(t, validatorsCopy, validatorsToRemove, v, removed, len(validatorsToRemove))
}

func Test_removeValidatorsFromListRemoveFromLast(t *testing.T) {
	t.Parallel()

	validatorsToRemoveFromEnd := 3
	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, len(validators))
	validatorsToRemove := make([]Validator, 0)

	_ = copy(validatorsCopy, validators)
	validatorsToRemove = append(validatorsToRemove, validators[len(validators)-validatorsToRemoveFromEnd:]...)

	v, removed := removeValidatorsFromList(validators, validatorsToRemove, len(validatorsToRemove))
	testRemoveValidators(t, validatorsCopy, validatorsToRemove, v, removed, len(validatorsToRemove))
}

func Test_removeValidatorsFromListRemoveFromFirstMaxSmaller(t *testing.T) {
	t.Parallel()

	validatorsToRemoveFromStart := 3
	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, len(validators))
	validatorsToRemove := make([]Validator, 0)
	maxToRemove := validatorsToRemoveFromStart - 1

	_ = copy(validatorsCopy, validators)
	validatorsToRemove = append(validatorsToRemove, validators[:validatorsToRemoveFromStart]...)

	v, removed := removeValidatorsFromList(validators, validatorsToRemove, maxToRemove)
	testRemoveValidators(t, validatorsCopy, validatorsToRemove, v, removed, maxToRemove)
}

func Test_removeValidatorsFromListRemoveFromFirstMaxGreater(t *testing.T) {
	t.Parallel()

	validatorsToRemoveFromStart := 3
	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, len(validators))
	validatorsToRemove := make([]Validator, 0)
	maxToRemove := validatorsToRemoveFromStart + 1

	_ = copy(validatorsCopy, validators)
	validatorsToRemove = append(validatorsToRemove, validators[:validatorsToRemoveFromStart]...)

	v, removed := removeValidatorsFromList(validators, validatorsToRemove, maxToRemove)
	testRemoveValidators(t, validatorsCopy, validatorsToRemove, v, removed, maxToRemove)
}

func Test_removeValidatorsFromListRemoveFromLastMaxSmaller(t *testing.T) {
	t.Parallel()

	validatorsToRemoveFromEnd := 3
	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, len(validators))
	validatorsToRemove := make([]Validator, 0)
	maxToRemove := validatorsToRemoveFromEnd - 1

	_ = copy(validatorsCopy, validators)
	validatorsToRemove = append(validatorsToRemove, validators[len(validators)-validatorsToRemoveFromEnd:]...)
	assert.Equal(t, validatorsToRemoveFromEnd, len(validatorsToRemove))

	v, removed := removeValidatorsFromList(validators, validatorsToRemove, maxToRemove)
	testRemoveValidators(t, validatorsCopy, validatorsToRemove, v, removed, maxToRemove)
}

func Test_removeValidatorsFromListRemoveFromLastMaxGreater(t *testing.T) {
	t.Parallel()

	validatorsToRemoveFromEnd := 3
	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, len(validators))
	validatorsToRemove := make([]Validator, 0)
	maxToRemove := validatorsToRemoveFromEnd + 1

	_ = copy(validatorsCopy, validators)
	validatorsToRemove = append(validatorsToRemove, validators[len(validators)-validatorsToRemoveFromEnd:]...)
	assert.Equal(t, validatorsToRemoveFromEnd, len(validatorsToRemove))

	v, removed := removeValidatorsFromList(validators, validatorsToRemove, maxToRemove)
	testRemoveValidators(t, validatorsCopy, validatorsToRemove, v, removed, maxToRemove)
}

func Test_removeValidatorsFromListRandomValidatorsMaxSmaller(t *testing.T) {
	t.Parallel()

	nbValidatotrsToRemove := 10
	maxToRemove := nbValidatotrsToRemove - 3
	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, len(validators))
	validatorsToRemove := make([]Validator, 0)

	_ = copy(validatorsCopy, validators)

	sort.Slice(validators, func(i, j int) bool {
		return bytes.Compare(validators[i].PubKey(), validators[j].PubKey()) < 0
	})

	validatorsToRemove = append(validatorsToRemove, validators[:nbValidatotrsToRemove]...)

	v, removed := removeValidatorsFromList(validators, validatorsToRemove, maxToRemove)
	testRemoveValidators(t, validatorsCopy, validatorsToRemove, v, removed, maxToRemove)
}

func Test_removeValidatorsFromListRandomValidatorsMaxGreater(t *testing.T) {
	t.Parallel()

	nbValidatotrsToRemove := 10
	maxToRemove := nbValidatotrsToRemove + 3
	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, len(validators))
	validatorsToRemove := make([]Validator, 0)

	_ = copy(validatorsCopy, validators)

	sort.Slice(validators, func(i, j int) bool {
		return bytes.Compare(validators[i].PubKey(), validators[j].PubKey()) < 0
	})

	validatorsToRemove = append(validatorsToRemove, validators[:nbValidatotrsToRemove]...)

	v, removed := removeValidatorsFromList(validators, validatorsToRemove, maxToRemove)
	testRemoveValidators(t, validatorsCopy, validatorsToRemove, v, removed, maxToRemove)
}

func Test_shuffleList(t *testing.T) {
	t.Parallel()

	randomness := generateRandomByteArray(32)
	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, 0)
	validatorsCopy = append(validatorsCopy, validators...)

	shuffled := shuffleList(validators, randomness)
	assert.Equal(t, len(validatorsCopy), len(shuffled))
	assert.NotEqual(t, validatorsCopy, shuffled)
	assert.True(t, contains(shuffled, validatorsCopy))
}

func Test_shuffleListParameterNotChanged(t *testing.T) {
	t.Parallel()

	randomness := generateRandomByteArray(32)
	validators := generateValidatorList(30)
	validatorsCopy := make([]Validator, len(validators))
	_ = copy(validatorsCopy, validators)

	_ = shuffleList(validators, randomness)
	assert.Equal(t, validatorsCopy, validators)
}

func Test_shuffleListConsistentShuffling(t *testing.T) {
	t.Parallel()

	randomness := generateRandomByteArray(32)
	validators := generateValidatorList(30)

	nbTrials := 10
	shuffled := shuffleList(validators, randomness)
	for i := 0; i < nbTrials; i++ {
		shuffled2 := shuffleList(validators, randomness)
		assert.Equal(t, shuffled, shuffled2)
	}
}

func Test_distributeValidatorsEqualNumber(t *testing.T) {
	t.Parallel()

	randomness := generateRandomByteArray(32)
	nodesPerShard := 30
	newNodesPerShard := 10
	validatorsMap := generateValidatorMap(nodesPerShard, 2)
	validatorsCopy := copyValidatorMap(validatorsMap)

	nbLists := len(validatorsMap)
	validatorsToDistribute := generateValidatorList(nbLists * newNodesPerShard)
	distributeValidators(validatorsToDistribute, validatorsMap, randomness, uint32(newNodesPerShard+1))
	testDistributeValidators(t, validatorsCopy, validatorsMap, validatorsToDistribute)
}

func Test_distributeValidatorsEqualNumberConsistent(t *testing.T) {
	t.Parallel()

	randomness := generateRandomByteArray(32)
	nodesPerShard := 30
	newNodesPerShard := 10
	validatorsMap := generateValidatorMap(nodesPerShard, 2)
	validatorsCopy := copyValidatorMap(validatorsMap)

	nbLists := len(validatorsMap)
	validatorsToDistribute := generateValidatorList(nbLists * newNodesPerShard)
	distributeValidators(validatorsToDistribute, validatorsMap, randomness, uint32(newNodesPerShard+1))
	testDistributeValidators(t, validatorsCopy, validatorsMap, validatorsToDistribute)

	distributeValidators(validatorsToDistribute, validatorsCopy, randomness, uint32(newNodesPerShard+1))
	for i := range validatorsCopy {
		assert.Equal(t, validatorsMap[i], validatorsCopy[i])
	}
}

func Test_distributeValidatorsUnequalNumber(t *testing.T) {
	t.Parallel()

	randomness := generateRandomByteArray(32)
	nodesPerShard := 30
	nbShards := uint32(2)
	validatorsMap := generateValidatorMap(nodesPerShard, nbShards)
	validatorsCopy := copyValidatorMap(validatorsMap)

	nbLists := len(validatorsMap)
	maxNewNodesPerShard := 10
	newNodes := nbLists*maxNewNodesPerShard - 1
	validatorsToDistribute := generateValidatorList(nbLists*newNodes - 1)
	distributeValidators(validatorsToDistribute, validatorsMap, randomness, nbShards+1)
	testDistributeValidators(t, validatorsCopy, validatorsMap, validatorsToDistribute)
}

func Test_distributeValidatorsUnequalNumberConsistent(t *testing.T) {
	t.Parallel()

	randomness := generateRandomByteArray(32)
	nodesPerShard := 30
	nbShards := uint32(2)
	validatorsMap := generateValidatorMap(nodesPerShard, nbShards)
	validatorsCopy := copyValidatorMap(validatorsMap)

	nbLists := len(validatorsMap)
	maxNewNodesPerShard := 10
	newNodes := nbLists*maxNewNodesPerShard - 1
	validatorsToDistribute := generateValidatorList(nbLists*newNodes - 1)
	distributeValidators(validatorsToDistribute, validatorsMap, randomness, nbShards+1)
	testDistributeValidators(t, validatorsCopy, validatorsMap, validatorsToDistribute)

	distributeValidators(validatorsToDistribute, validatorsCopy, randomness, nbShards+1)
	for i := range validatorsCopy {
		assert.Equal(t, validatorsMap[i], validatorsCopy[i])
	}
}

func Test_shuffleOutNodesNoLeaving(t *testing.T) {
	t.Parallel()

	randomness := generateRandomByteArray(32)
	eligibleNodesPerShard := 100
	waitingNodesPerShard := 40
	nbShards := uint32(2)
	var leaving []Validator

	eligibleMap := generateValidatorMap(eligibleNodesPerShard, nbShards)
	waitingMap := generateValidatorMap(waitingNodesPerShard, nbShards)

	shuffledOut, newEligible, newLeaving := shuffleOutNodes(eligibleMap, waitingMap, leaving, randomness)
	testShuffledOut(t, eligibleMap, waitingMap, newEligible, shuffledOut, leaving, newLeaving)
}

func Test_shuffleOutNodesWithLeaving(t *testing.T) {
	t.Parallel()

	randomness := generateRandomByteArray(32)
	eligibleNodesPerShard := 100
	waitingNodesPerShard := 40
	nbShards := uint32(2)
	leaving := make([]Validator, 0)

	eligibleMap := generateValidatorMap(eligibleNodesPerShard, nbShards)
	waitingMap := generateValidatorMap(waitingNodesPerShard, nbShards)
	for _, valList := range eligibleMap {
		leaving = append(leaving, valList[:len(valList)/5]...)
	}

	shuffledOut, newEligible, newLeaving := shuffleOutNodes(eligibleMap, waitingMap, leaving, randomness)
	testShuffledOut(t, eligibleMap, waitingMap, newEligible, shuffledOut, leaving, newLeaving)
}

func Test_shuffleOutNodesWithLeavingMoreThanWaiting(t *testing.T) {
	t.Parallel()

	randomness := generateRandomByteArray(32)
	eligibleNodesPerShard := 100
	waitingNodesPerShard := 40
	nbShards := uint32(2)
	leaving := make([]Validator, 0)

	eligibleMap := generateValidatorMap(eligibleNodesPerShard, nbShards)
	waitingMap := generateValidatorMap(waitingNodesPerShard, nbShards)
	for _, valList := range eligibleMap {
		leaving = append(leaving, valList[:len(valList)/2+1]...)
	}

	shuffledOut, newEligible, newLeaving := shuffleOutNodes(eligibleMap, waitingMap, leaving, randomness)
	testShuffledOut(t, eligibleMap, waitingMap, newEligible, shuffledOut, leaving, newLeaving)
}

func TestNewXorValidatorsShuffler(t *testing.T) {
	t.Parallel()

	shuffler := NewXorValidatorsShuffler(
		100,
		100,
		0.2,
		false,
	)

	assert.NotNil(t, shuffler)
}

func TestRandXORShuffler_computeNewShardsNotChanging(t *testing.T) {
	t.Parallel()

	currentNbShards := uint32(3)
	shuffler := createXorShufflerInter()
	eligible := generateValidatorMap(int(shuffler.nodesShard), currentNbShards)
	nbShards := currentNbShards + 1 // account for meta
	maxNodesNoSplit := (nbShards + 1) * (shuffler.nodesShard + shuffler.shardHysteresis)
	nbWaitingPerShard := int(maxNodesNoSplit/nbShards - shuffler.nodesShard)
	waiting := generateValidatorMap(nbWaitingPerShard, currentNbShards)
	newNodes := generateValidatorList(0)
	leaving := generateValidatorList(0)

	newNbShards := shuffler.computeNewShards(eligible, waiting, newNodes, leaving, currentNbShards)
	assert.Equal(t, currentNbShards, newNbShards)
}

func TestRandXORShuffler_computeNewShardsWithSplit(t *testing.T) {
	t.Parallel()

	currentNbShards := uint32(3)
	shuffler := createXorShufflerInter()
	eligible := generateValidatorMap(int(shuffler.nodesShard), currentNbShards)
	nbShards := currentNbShards + 1 // account for meta
	maxNodesNoSplit := (nbShards + 1) * (shuffler.nodesShard + shuffler.shardHysteresis)
	nbWaitingPerShard := int(maxNodesNoSplit/nbShards-shuffler.nodesShard) + 1
	waiting := generateValidatorMap(nbWaitingPerShard, currentNbShards)
	newNodes := generateValidatorList(0)
	leaving := generateValidatorList(0)

	newNbShards := shuffler.computeNewShards(eligible, waiting, newNodes, leaving, currentNbShards)
	assert.Equal(t, currentNbShards+1, newNbShards)
}

func TestRandXORShuffler_computeNewShardsWithMerge(t *testing.T) {
	t.Parallel()

	currentNbShards := uint32(3)
	shuffler := createXorShufflerInter()
	eligible := generateValidatorMap(int(shuffler.nodesShard), currentNbShards)
	nbWaitingPerShard := 0
	waiting := generateValidatorMap(nbWaitingPerShard, currentNbShards)
	newNodes := generateValidatorList(0)
	leaving := generateValidatorList(1)

	newNbShards := shuffler.computeNewShards(eligible, waiting, newNodes, leaving, currentNbShards)
	assert.Equal(t, currentNbShards-1, newNbShards)
}

func TestRandXORShuffler_UpdateParams(t *testing.T) {
	t.Parallel()

	shuffler := createXorShufflerInter()
	shuffler2 := &randXORShuffler{
		nodesShard:           200,
		nodesMeta:            200,
		shardHysteresis:      0,
		metaHysteresis:       0,
		adaptivity:           true,
		shuffleBetweenShards: true,
	}

	shuffler.UpdateParams(
		shuffler2.nodesShard,
		shuffler2.nodesMeta,
		0,
		shuffler2.adaptivity,
	)

	assert.Equal(t, shuffler2, shuffler)
}

func TestRandXORShuffler_UpdateNodeListsNoReSharding(t *testing.T) {
	t.Parallel()

	shuffler := createXorShufflerInter()

	eligiblePerShard := int(shuffler.nodesShard)
	waitingPerShard := 30
	nbShards := uint32(3)
	randomness := generateRandomByteArray(32)

	leavingNodes := make([]Validator, 0)
	newNodes := make([]Validator, 0)

	eligibleMap := generateValidatorMap(eligiblePerShard, nbShards)
	waitingMap := generateValidatorMap(waitingPerShard, nbShards)

	args := ArgsUpdateNodes{
		Eligible: eligibleMap,
		Waiting:  waitingMap,
		NewNodes: newNodes,
		Leaving:  leavingNodes,
		Rand:     randomness,
	}

	eligible, waiting, _ := shuffler.UpdateNodeLists(args)

	allPrevEligible := getValidatorsInMap(eligibleMap)
	allNewEligible := getValidatorsInMap(eligible)
	allPrevWaiting := getValidatorsInMap(waitingMap)
	allNewWaiting := getValidatorsInMap(waiting)

	assert.Equal(t, len(allPrevEligible)+len(allPrevWaiting), len(allNewEligible)+len(allNewWaiting))
}

func TestRandXORShuffler_UpdateNodeListsWithLeavingRemovesFromEligible(t *testing.T) {
	t.Parallel()

	eligiblePerShard := 10
	eligibleMeta := 10

	shuffler := NewXorValidatorsShuffler(
		uint32(eligiblePerShard),
		uint32(eligibleMeta),
		0.2,
		false,
	)
	shuffler.shuffleBetweenShards = false

	waitingPerShard := 2
	nbShards := 0

	args := createShufflerArgs(eligiblePerShard, waitingPerShard, uint32(nbShards))

	args.Leaving = []Validator{
		args.Eligible[core.MetachainShardId][0],
		args.Eligible[core.MetachainShardId][1],
	}

	eligible, waiting, leaving := shuffler.UpdateNodeLists(args)

	allNewEligible := getValidatorsInMap(eligible)
	allNewWaiting := getValidatorsInMap(waiting)

	allNewEligibleMap := listToMap(allNewEligible)

	for _, leavingValidator := range leaving {
		assert.Nil(t, allNewEligibleMap[string(leavingValidator.PubKey())])
	}

	previousNumberOfNodes := (eligiblePerShard + waitingPerShard) * (nbShards + 1)
	currentNumberOfNodes := len(allNewEligible) + len(allNewWaiting) + len(leaving)
	assert.Equal(t, previousNumberOfNodes, currentNumberOfNodes)

}

func TestRandXORShuffler_UpdateNodeListsWithLeavingRemovesFromWaiting(t *testing.T) {
	t.Parallel()

	eligiblePerShard := 10
	eligibleMeta := 10

	shuffler := NewXorValidatorsShuffler(
		uint32(eligiblePerShard),
		uint32(eligibleMeta),
		0.2,
		false,
	)
	shuffler.shuffleBetweenShards = false

	waitingPerShard := 2
	nbShards := 0

	args := createShufflerArgs(eligiblePerShard, waitingPerShard, uint32(nbShards))

	args.Leaving = []Validator{
		args.Waiting[core.MetachainShardId][0],
		args.Waiting[core.MetachainShardId][1],
	}

	eligible, waiting, leaving := shuffler.UpdateNodeLists(args)

	allNewEligible := getValidatorsInMap(eligible)
	allNewWaiting := getValidatorsInMap(waiting)

	allNewWaitingMap := listToMap(allNewWaiting)

	for _, leavingValidator := range leaving {
		assert.Nil(t, allNewWaitingMap[string(leavingValidator.PubKey())])
	}

	previousNumberOfNodes := (eligiblePerShard + waitingPerShard) * (nbShards + 1)
	currentNumberOfNodes := len(allNewEligible) + len(allNewWaiting) + len(leaving)
	assert.Equal(t, previousNumberOfNodes, currentNumberOfNodes)
}

func TestRandXORShuffler_UpdateNodeListsWithNonExistentLeavingDoesNotRemove(t *testing.T) {
	t.Parallel()

	shuffler := NewXorValidatorsShuffler(
		10,
		10,
		0.2,
		false,
	)
	shuffler.shuffleBetweenShards = false

	eligiblePerShard := int(shuffler.nodesShard)
	waitingPerShard := 2
	nbShards := uint32(0)

	args := createShufflerArgs(eligiblePerShard, waitingPerShard, nbShards)

	args.Leaving = []Validator{
		&validator{
			pubKey: generateRandomByteArray(32),
		},
		&validator{
			pubKey: generateRandomByteArray(32),
		},
	}

	eligible, waiting, leaving := shuffler.UpdateNodeLists(args)

	allNewEligible := getValidatorsInMap(eligible)
	allNewEligibleMap := listToMap(allNewEligible)
	allNewWaiting := getValidatorsInMap(waiting)
	allNewWaitingMap := listToMap(allNewWaiting)

	for _, leavingValidator := range leaving {
		assert.Nil(t, allNewEligibleMap[string(leavingValidator.PubKey())])
	}
	for _, leavingValidator := range leaving {
		assert.Nil(t, allNewWaitingMap[string(leavingValidator.PubKey())])
	}

	for i := range leaving {
		assert.Equal(t, args.Leaving[i], leaving[i])
	}
}

func TestRandXORShuffler_UpdateNodeListsNoReShardingIntraShardShuffling(t *testing.T) {
	t.Parallel()

	shuffler := createXorShufflerIntraShards()

	eligiblePerShard := int(shuffler.nodesShard)
	waitingPerShard := 30
	nbShards := uint32(3)
	randomness := generateRandomByteArray(32)

	leavingNodes := make([]Validator, 0)
	newNodes := make([]Validator, 0)

	eligibleMap := generateValidatorMap(eligiblePerShard, nbShards)
	waitingMap := generateValidatorMap(waitingPerShard, nbShards)

	args := ArgsUpdateNodes{
		Eligible: eligibleMap,
		Waiting:  waitingMap,
		NewNodes: newNodes,
		Leaving:  leavingNodes,
		Rand:     randomness,
	}

	eligible, waiting, _ := shuffler.UpdateNodeLists(args)

	allPrevEligible := getValidatorsInMap(eligibleMap)
	allNewEligible := getValidatorsInMap(eligible)
	allPrevWaiting := getValidatorsInMap(waitingMap)
	allNewWaiting := getValidatorsInMap(waiting)

	assert.Equal(t, len(allPrevEligible)+len(allPrevWaiting), len(allNewEligible)+len(allNewWaiting))
}

func listToMap(validators []Validator) map[string]Validator {
	validatorMap := make(map[string]Validator)

	for _, v := range validators {
		validatorMap[string(v.PubKey())] = v
	}

	return validatorMap
}

func createShufflerArgs(eligiblePerShard int, waitingPerShard int, nbShards uint32) ArgsUpdateNodes {
	randomness := generateRandomByteArray(32)

	leavingNodes := make([]Validator, 0)
	newNodes := make([]Validator, 0)

	eligibleMap := generateValidatorMap(eligiblePerShard, nbShards)
	waitingMap := generateValidatorMap(waitingPerShard, nbShards)

	args := ArgsUpdateNodes{
		Eligible: eligibleMap,
		Waiting:  waitingMap,
		NewNodes: newNodes,
		Leaving:  leavingNodes,
		Rand:     randomness,
	}
	return args
}
