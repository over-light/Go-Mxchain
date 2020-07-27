package peer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"sort"
	"sync"

	"github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

var log = logger.GetOrCreate("process/peer")

var _ process.ValidatorStatisticsProcessor = (*validatorStatistics)(nil)

type validatorActionType uint8

const (
	unknownAction             validatorActionType = 0
	leaderSuccess             validatorActionType = 1
	leaderFail                validatorActionType = 2
	validatorSuccess          validatorActionType = 3
	validatorIgnoredSignature validatorActionType = 4
)

// ArgValidatorStatisticsProcessor holds all dependencies for the validatorStatistics
type ArgValidatorStatisticsProcessor struct {
	Marshalizer         marshal.Marshalizer
	NodesCoordinator    sharding.NodesCoordinator
	ShardCoordinator    sharding.Coordinator
	DataPool            DataPool
	StorageService      dataRetriever.StorageService
	PubkeyConv          core.PubkeyConverter
	PeerAdapter         state.AccountsAdapter
	Rater               sharding.PeerAccountListAndRatingHandler
	RewardsHandler      process.RewardsHandler
	MaxComputableRounds uint64
	NodesSetup          sharding.GenesisNodesSetupHandler
	GenesisNonce        uint64
	RatingEnableEpoch   uint32
}

type validatorStatistics struct {
	marshalizer            marshal.Marshalizer
	dataPool               DataPool
	storageService         dataRetriever.StorageService
	nodesCoordinator       sharding.NodesCoordinator
	shardCoordinator       sharding.Coordinator
	pubkeyConv             core.PubkeyConverter
	peerAdapter            state.AccountsAdapter
	rater                  sharding.PeerAccountListAndRatingHandler
	rewardsHandler         process.RewardsHandler
	maxComputableRounds    uint64
	missedBlocksCounters   validatorRoundCounters
	mutValidatorStatistics sync.RWMutex
	genesisNonce           uint64
	ratingEnableEpoch      uint32
	lastFinalizedRootHash  []byte
}

// NewValidatorStatisticsProcessor instantiates a new validatorStatistics structure responsible of keeping account of
//  each validator actions in the consensus process
func NewValidatorStatisticsProcessor(arguments ArgValidatorStatisticsProcessor) (*validatorStatistics, error) {
	if check.IfNil(arguments.PeerAdapter) {
		return nil, process.ErrNilPeerAccountsAdapter
	}
	if check.IfNil(arguments.PubkeyConv) {
		return nil, process.ErrNilPubkeyConverter
	}
	if check.IfNil(arguments.DataPool) {
		return nil, process.ErrNilDataPoolHolder
	}
	if check.IfNil(arguments.StorageService) {
		return nil, process.ErrNilStorage
	}
	if check.IfNil(arguments.NodesCoordinator) {
		return nil, process.ErrNilNodesCoordinator
	}
	if check.IfNil(arguments.ShardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}
	if check.IfNil(arguments.Marshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if arguments.MaxComputableRounds == 0 {
		return nil, process.ErrZeroMaxComputableRounds
	}
	if check.IfNil(arguments.Rater) {
		return nil, process.ErrNilRater
	}
	if check.IfNil(arguments.RewardsHandler) {
		return nil, process.ErrNilRewardsHandler
	}
	if check.IfNil(arguments.NodesSetup) {
		return nil, process.ErrNilNodesSetup
	}

	vs := &validatorStatistics{
		peerAdapter:          arguments.PeerAdapter,
		pubkeyConv:           arguments.PubkeyConv,
		nodesCoordinator:     arguments.NodesCoordinator,
		shardCoordinator:     arguments.ShardCoordinator,
		dataPool:             arguments.DataPool,
		storageService:       arguments.StorageService,
		marshalizer:          arguments.Marshalizer,
		missedBlocksCounters: make(validatorRoundCounters),
		rater:                arguments.Rater,
		rewardsHandler:       arguments.RewardsHandler,
		maxComputableRounds:  arguments.MaxComputableRounds,
		genesisNonce:         arguments.GenesisNonce,
		ratingEnableEpoch:    arguments.RatingEnableEpoch,
	}

	rater := arguments.Rater
	err := vs.saveInitialState(arguments.NodesSetup, rater.GetStartRating())
	if err != nil {
		return nil, err
	}

	return vs, nil
}

// saveNodesCoordinatorUpdates is called at the first block after start in epoch to update the state trie according to
// the shuffling and changes done by the nodesCoordinator at the end of the epoch
func (vs *validatorStatistics) saveNodesCoordinatorUpdates(epoch uint32) error {
	log.Debug("save nodes coordinator updates ", "epoch", epoch)

	nodesMap, err := vs.nodesCoordinator.GetAllEligibleValidatorsPublicKeys(epoch)
	if err != nil {
		return err
	}

	err = vs.saveUpdatesForNodesMap(nodesMap, core.EligibleList)
	if err != nil {
		return err
	}

	nodesMap, err = vs.nodesCoordinator.GetAllWaitingValidatorsPublicKeys(epoch)
	if err != nil {
		return err
	}

	err = vs.saveUpdatesForNodesMap(nodesMap, core.WaitingList)
	if err != nil {
		return err
	}

	nodesMap, err = vs.nodesCoordinator.GetAllLeavingValidatorsPublicKeys(epoch)
	if err != nil {
		return err
	}

	err = vs.saveUpdatesForNodesMap(nodesMap, core.InactiveList)
	if err != nil {
		return err
	}

	return nil
}

func (vs *validatorStatistics) saveUpdatesForNodesMap(
	nodesMap map[uint32][][]byte,
	peerType core.PeerType,
) error {
	for shardID := uint32(0); shardID < vs.shardCoordinator.NumberOfShards(); shardID++ {
		err := vs.saveUpdatesForList(nodesMap[shardID], shardID, peerType)
		if err != nil {
			return err
		}
	}

	err := vs.saveUpdatesForList(nodesMap[core.MetachainShardId], core.MetachainShardId, peerType)
	if err != nil {
		return err
	}

	return nil
}

func (vs *validatorStatistics) saveUpdatesForList(
	pks [][]byte,
	shardID uint32,
	peerType core.PeerType,
) error {
	for index, pubKey := range pks {
		peerAcc, err := vs.GetPeerAccount(pubKey)
		if err != nil {
			log.Debug("error getting peer account", "error", err, "key", pubKey)
			return err
		}

		peerAcc.SetListAndIndex(shardID, string(peerType), uint32(index))

		err = vs.peerAdapter.SaveAccount(peerAcc)
		if err != nil {
			return err
		}
	}

	return nil
}

// saveInitialState takes an initial peer list, validates it and sets up the initial state for each of the peers
func (vs *validatorStatistics) saveInitialState(
	nodesConfig sharding.GenesisNodesSetupHandler,
	startRating uint32,
) error {
	eligibleNodesInfo, waitingNodesInfo := nodesConfig.InitialNodesInfo()
	err := vs.saveInitialValueForMap(eligibleNodesInfo, startRating, core.EligibleList)
	if err != nil {
		return err
	}

	err = vs.saveInitialValueForMap(waitingNodesInfo, startRating, core.WaitingList)
	if err != nil {
		return err
	}

	hash, err := vs.peerAdapter.Commit()
	if err != nil {
		return err
	}

	log.Trace("committed peer adapter", "root hash", hex.EncodeToString(hash))

	return nil
}

func (vs *validatorStatistics) saveInitialValueForMap(
	nodesInfo map[uint32][]sharding.GenesisNodeInfoHandler,
	startRating uint32,
	peerType core.PeerType,
) error {
	if len(nodesInfo) == 0 {
		return nil
	}

	for shardID := uint32(0); shardID < vs.shardCoordinator.NumberOfShards(); shardID++ {
		nodeInfoList := nodesInfo[shardID]
		for index, nodeInfo := range nodeInfoList {
			err := vs.initializeNode(nodeInfo, startRating, shardID, peerType, uint32(index))
			if err != nil {
				return err
			}
		}
	}

	shardID := core.MetachainShardId
	nodeInfoList := nodesInfo[shardID]
	for index, nodeInfo := range nodeInfoList {
		err := vs.initializeNode(nodeInfo, startRating, shardID, peerType, uint32(index))
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdatePeerState takes a header, updates the peer state for all of the
// consensus members and returns the new root hash
func (vs *validatorStatistics) UpdatePeerState(header data.HeaderHandler, cache map[string]data.HeaderHandler) ([]byte, error) {
	if header.GetNonce() == vs.genesisNonce {
		return vs.peerAdapter.RootHash()
	}

	vs.mutValidatorStatistics.Lock()
	vs.missedBlocksCounters.reset()
	vs.mutValidatorStatistics.Unlock()

	previousHeader, ok := cache[string(header.GetPrevHash())]
	if !ok {
		return nil, fmt.Errorf("%w - updatePeerState get header from cache - hash: %s, round: %v, nonce: %v",
			process.ErrMissingHeader,
			hex.EncodeToString(header.GetPrevHash()),
			header.GetRound(),
			header.GetNonce())
	}

	epoch := computeEpoch(header)

	var err error
	if previousHeader.IsStartOfEpochBlock() {
		err = vs.saveNodesCoordinatorUpdates(previousHeader.GetEpoch())
		if err != nil {
			log.Warn("could not update info from nodesCoordinator")
			return nil, err
		}
	}

	err = vs.checkForMissedBlocks(
		header.GetRound(),
		previousHeader.GetRound(),
		previousHeader.GetRandSeed(),
		previousHeader.GetShardID(),
		epoch,
	)
	if err != nil {
		return nil, err
	}

	err = vs.updateShardDataPeerState(header, cache)
	if err != nil {
		return nil, err
	}

	err = vs.updateMissedBlocksCounters()
	if err != nil {
		return nil, err
	}

	if header.GetNonce() == vs.genesisNonce+1 {
		return vs.peerAdapter.RootHash()
	}
	log.Trace("Increasing", "round", previousHeader.GetRound(), "prevRandSeed", previousHeader.GetPrevRandSeed())

	consensusGroupEpoch := computeEpoch(previousHeader)
	consensusGroup, err := vs.nodesCoordinator.ComputeConsensusGroup(
		previousHeader.GetPrevRandSeed(),
		previousHeader.GetRound(),
		previousHeader.GetShardID(),
		consensusGroupEpoch)
	if err != nil {
		return nil, err
	}
	leaderPK := core.GetTrimmedPk(vs.pubkeyConv.Encode(consensusGroup[0].PubKey()))
	log.Trace("Increasing for leader", "leader", leaderPK, "round", previousHeader.GetRound())
	err = vs.updateValidatorInfoOnSuccessfulBlock(
		consensusGroup,
		previousHeader.GetPubKeysBitmap(),
		big.NewInt(0).Sub(previousHeader.GetAccumulatedFees(), previousHeader.GetDeveloperFees()),
		previousHeader.GetShardID())
	if err != nil {
		return nil, err
	}

	rootHash, err := vs.peerAdapter.RootHash()
	if err != nil {
		return nil, err
	}

	log.Trace("after updating validator stats", "rootHash", rootHash, "round", header.GetRound(), "selfId", vs.shardCoordinator.SelfId())

	return rootHash, nil
}

func computeEpoch(header data.HeaderHandler) uint32 {
	// TODO: change if start of epoch block needs to be validated by the new epoch nodes
	// previous block was proposed by the consensus group of the previous epoch
	epoch := header.GetEpoch()
	if header.IsStartOfEpochBlock() && epoch > 0 {
		epoch = epoch - 1
	}

	return epoch
}

func (vs *validatorStatistics) DisplayRatings(epoch uint32) {
	validatorPKs, err := vs.nodesCoordinator.GetAllEligibleValidatorsPublicKeys(epoch)
	if err != nil {
		log.Warn("could not get ValidatorPublicKeys", "epoch", epoch)
		return
	}
	log.Trace("started printing tempRatings")
	for shardID, list := range validatorPKs {
		for _, pk := range list {
			log.Trace("tempRating", "PK", pk, "tempRating", vs.getTempRating(string(pk)), "ShardID", shardID)
		}
	}
	log.Trace("finished printing tempRatings")
}

// Commit commits the validator statistics trie and returns the root hash
func (vs *validatorStatistics) Commit() ([]byte, error) {
	return vs.peerAdapter.Commit()
}

// RootHash returns the root hash of the validator statistics trie
func (vs *validatorStatistics) RootHash() ([]byte, error) {
	return vs.peerAdapter.RootHash()
}

func (vs *validatorStatistics) getValidatorDataFromLeaves(
	leaves map[string][]byte,
) (map[uint32][]*state.ValidatorInfo, error) {

	validators := make(map[uint32][]*state.ValidatorInfo, vs.shardCoordinator.NumberOfShards()+1)
	for i := uint32(0); i < vs.shardCoordinator.NumberOfShards(); i++ {
		validators[i] = make([]*state.ValidatorInfo, 0)
	}
	validators[core.MetachainShardId] = make([]*state.ValidatorInfo, 0)

	sliceLeaves := vs.convertMapToSortedSlice(leaves)

	sort.Slice(sliceLeaves, func(i, j int) bool {
		return bytes.Compare(sliceLeaves[i], sliceLeaves[j]) < 0
	})

	for _, pa := range sliceLeaves {
		peerAccount, err := vs.unmarshalPeer(pa)
		if err != nil {
			return nil, err
		}

		currentShardId := peerAccount.GetShardId()
		validatorInfoData := vs.peerAccountToValidatorInfo(peerAccount)
		validators[currentShardId] = append(validators[currentShardId], validatorInfoData)
	}

	return validators, nil
}

func getActualList(peerAccount state.PeerAccountHandler) string {
	savedList := peerAccount.GetList()
	if peerAccount.GetUnStakedEpoch() == core.DefaultUnstakedEpoch {
		if savedList == string(core.InactiveList) {
			return string(core.JailedList)
		}
		return savedList
	}
	if savedList == string(core.InactiveList) {
		return savedList
	}

	return string(core.LeavingList)
}

func (vs *validatorStatistics) peerAccountToValidatorInfo(peerAccount state.PeerAccountHandler) *state.ValidatorInfo {
	chance := vs.rater.GetChance(peerAccount.GetRating())
	startRatingChance := vs.rater.GetChance(vs.rater.GetStartRating())
	ratingModifier := float32(chance) / float32(startRatingChance)

	return &state.ValidatorInfo{
		PublicKey:                       peerAccount.GetBLSPublicKey(),
		ShardId:                         peerAccount.GetShardId(),
		List:                            getActualList(peerAccount),
		Index:                           peerAccount.GetIndexInList(),
		TempRating:                      peerAccount.GetTempRating(),
		Rating:                          peerAccount.GetRating(),
		RatingModifier:                  ratingModifier,
		RewardAddress:                   peerAccount.GetRewardAddress(),
		LeaderSuccess:                   peerAccount.GetLeaderSuccessRate().NumSuccess,
		LeaderFailure:                   peerAccount.GetLeaderSuccessRate().NumFailure,
		ValidatorSuccess:                peerAccount.GetValidatorSuccessRate().NumSuccess,
		ValidatorFailure:                peerAccount.GetValidatorSuccessRate().NumFailure,
		ValidatorIgnoredSignatures:      peerAccount.GetValidatorIgnoredSignaturesRate(),
		TotalLeaderSuccess:              peerAccount.GetTotalLeaderSuccessRate().NumSuccess,
		TotalLeaderFailure:              peerAccount.GetTotalLeaderSuccessRate().NumFailure,
		TotalValidatorSuccess:           peerAccount.GetTotalValidatorSuccessRate().NumSuccess,
		TotalValidatorFailure:           peerAccount.GetTotalValidatorSuccessRate().NumFailure,
		TotalValidatorIgnoredSignatures: peerAccount.GetTotalValidatorIgnoredSignaturesRate(),
		NumSelectedInSuccessBlocks:      peerAccount.GetNumSelectedInSuccessBlocks(),
		AccumulatedFees:                 big.NewInt(0).Set(peerAccount.GetAccumulatedFees()),
	}
}

func (vs *validatorStatistics) unmarshalPeer(pa []byte) (state.PeerAccountHandler, error) {
	peerAccount := state.NewEmptyPeerAccount()
	err := vs.marshalizer.Unmarshal(peerAccount, pa)
	if err != nil {
		return nil, err
	}
	return peerAccount, nil
}

func (vs *validatorStatistics) convertMapToSortedSlice(leaves map[string][]byte) [][]byte {
	newLeaves := make([][]byte, len(leaves))
	i := 0
	for _, pa := range leaves {
		newLeaves[i] = pa
		i++
	}

	return newLeaves
}

// GetValidatorInfoForRootHash returns all the peer accounts from the trie with the given rootHash
func (vs *validatorStatistics) GetValidatorInfoForRootHash(rootHash []byte) (map[uint32][]*state.ValidatorInfo, error) {
	sw := core.NewStopWatch()
	sw.Start("GetValidatorInfoForRootHash")
	defer func() {
		sw.Stop("GetValidatorInfoForRootHash")
		log.Debug("GetValidatorInfoForRootHash", sw.GetMeasurements()...)
	}()

	allLeaves, err := vs.peerAdapter.GetAllLeaves(rootHash)
	if err != nil {
		return nil, err
	}

	vInfos, err := vs.getValidatorDataFromLeaves(allLeaves)
	if err != nil {
		return nil, err
	}

	return vInfos, err
}

// ProcessRatingsEndOfEpoch makes end of epoch process on the rating
func (vs *validatorStatistics) ProcessRatingsEndOfEpoch(
	validatorInfos map[uint32][]*state.ValidatorInfo,
	epoch uint32,
) error {
	if len(validatorInfos) == 0 {
		return process.ErrNilValidatorInfos
	}

	if epoch > 0 {
		epoch = epoch - 1
	}

	signedThreshold := vs.rater.GetSignedBlocksThreshold()
	for shardId, validators := range validatorInfos {
		for _, validator := range validators {
			if validator.List != string(core.EligibleList) {
				continue
			}

			err := vs.verifySignaturesBelowSignedThreshold(validator, signedThreshold, shardId, epoch)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (vs *validatorStatistics) verifySignaturesBelowSignedThreshold(
	validator *state.ValidatorInfo,
	signedThreshold float32,
	shardId uint32,
	epoch uint32,
) error {

	if epoch < vs.ratingEnableEpoch {
		return nil
	}

	validatorAppereances := core.MaxUint32(1, validator.ValidatorSuccess+validator.ValidatorFailure)
	computedThreshold := float32(validator.ValidatorSuccess) / float32(validatorAppereances)

	if computedThreshold <= signedThreshold {
		newTempRating := vs.rater.RevertIncreaseValidator(shardId, validator.TempRating, validator.ValidatorFailure)
		pa, err := vs.GetPeerAccount(validator.PublicKey)
		if err != nil {
			return err
		}

		pa.SetTempRating(newTempRating)

		err = vs.peerAdapter.SaveAccount(pa)
		if err != nil {
			return err
		}

		log.Debug("below signed blocks threshold",
			"pk", validator.PublicKey,
			"signed %", computedThreshold,
			"validatorSuccess", validator.ValidatorSuccess,
			"validatorFailure", validator.ValidatorFailure,
			"validatorIgnored", validator.ValidatorIgnoredSignatures,
			"new tempRating", newTempRating,
			"old tempRating", validator.TempRating,
		)

		validator.TempRating = newTempRating
	}

	return nil
}

// ResetValidatorStatisticsAtNewEpoch resets the validator info at the start of a new epoch
func (vs *validatorStatistics) ResetValidatorStatisticsAtNewEpoch(vInfos map[uint32][]*state.ValidatorInfo) error {
	sw := core.NewStopWatch()
	sw.Start("ResetValidatorStatisticsAtNewEpoch")
	defer func() {
		sw.Stop("ResetValidatorStatisticsAtNewEpoch")
		log.Debug("ResetValidatorStatisticsAtNewEpoch", sw.GetMeasurements()...)
	}()

	for _, validators := range vInfos {
		for _, validator := range validators {
			account, err := vs.peerAdapter.LoadAccount(validator.GetPublicKey())
			if err != nil {
				return err
			}

			peerAccount, ok := account.(state.PeerAccountHandler)
			if !ok {
				return process.ErrWrongTypeAssertion
			}
			peerAccount.ResetAtNewEpoch()

			err = vs.peerAdapter.SaveAccount(peerAccount)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (vs *validatorStatistics) checkForMissedBlocks(
	currentHeaderRound,
	previousHeaderRound uint64,
	prevRandSeed []byte,
	shardID uint32,
	epoch uint32,
) error {
	missedRounds := currentHeaderRound - previousHeaderRound
	if missedRounds <= 1 {
		return nil
	}

	tooManyComputations := missedRounds > vs.maxComputableRounds
	if !tooManyComputations {
		return vs.computeDecrease(previousHeaderRound, currentHeaderRound, prevRandSeed, shardID, epoch)
	}

	return vs.decreaseAll(shardID, missedRounds-1, epoch)
}

func (vs *validatorStatistics) computeDecrease(
	previousHeaderRound uint64,
	currentHeaderRound uint64,
	prevRandSeed []byte,
	shardID uint32,
	epoch uint32,
) error {
	if epoch < vs.ratingEnableEpoch {
		return nil
	}

	sw := core.NewStopWatch()
	sw.Start("checkForMissedBlocks")
	defer func() {
		sw.Stop("checkForMissedBlocks")
		log.Trace("measurements checkForMissedBlocks", sw.GetMeasurements()...)
	}()

	for i := previousHeaderRound + 1; i < currentHeaderRound; i++ {
		swInner := core.NewStopWatch()

		swInner.Start("ComputeValidatorsGroup")
		log.Trace("Decreasing", "round", i, "prevRandSeed", prevRandSeed, "shardId", shardID)
		consensusGroup, err := vs.nodesCoordinator.ComputeConsensusGroup(prevRandSeed, i, shardID, epoch)
		swInner.Stop("ComputeValidatorsGroup")
		if err != nil {
			return err
		}

		swInner.Start("GetPeerAccount")
		leaderPeerAcc, err := vs.GetPeerAccount(consensusGroup[0].PubKey())
		leaderPK := core.GetTrimmedPk(vs.pubkeyConv.Encode(consensusGroup[0].PubKey()))
		log.Trace("Decreasing for leader", "leader", leaderPK, "round", i)
		swInner.Stop("GetPeerAccount")
		if err != nil {
			return err
		}

		vs.mutValidatorStatistics.Lock()
		vs.missedBlocksCounters.decreaseLeader(consensusGroup[0].PubKey())
		vs.mutValidatorStatistics.Unlock()

		swInner.Start("ComputeDecreaseProposer")
		newRating := vs.rater.ComputeDecreaseProposer(
			shardID,
			leaderPeerAcc.GetTempRating(),
			leaderPeerAcc.GetConsecutiveProposerMisses())
		swInner.Stop("ComputeDecreaseProposer")

		swInner.Start("SetConsecutiveProposerMisses")
		leaderPeerAcc.SetConsecutiveProposerMisses(leaderPeerAcc.GetConsecutiveProposerMisses() + 1)
		swInner.Stop("SetConsecutiveProposerMisses")

		swInner.Start("SetTempRating")
		leaderPeerAcc.SetTempRating(newRating)
		err = vs.peerAdapter.SaveAccount(leaderPeerAcc)
		swInner.Stop("SetTempRating")
		if err != nil {
			return err
		}

		swInner.Start("ComputeDecreaseAllValidators")
		err = vs.decreaseForConsensusValidators(consensusGroup, shardID, epoch)
		swInner.Stop("ComputeDecreaseAllValidators")
		if err != nil {
			return err
		}
		sw.Add(swInner)
	}
	return nil
}

func (vs *validatorStatistics) decreaseForConsensusValidators(
	consensusGroup []sharding.Validator,
	shardId uint32,
	epoch uint32,
) error {
	if epoch < vs.ratingEnableEpoch {
		return nil
	}

	vs.mutValidatorStatistics.Lock()
	defer vs.mutValidatorStatistics.Unlock()

	for j := 1; j < len(consensusGroup); j++ {
		validatorPeerAccount, verr := vs.GetPeerAccount(consensusGroup[j].PubKey())
		if verr != nil {
			return verr
		}
		vs.missedBlocksCounters.decreaseValidator(consensusGroup[j].PubKey())

		newRating := vs.rater.ComputeDecreaseValidator(shardId, validatorPeerAccount.GetTempRating())
		validatorPeerAccount.SetTempRating(newRating)

		err := vs.peerAdapter.SaveAccount(validatorPeerAccount)
		if err != nil {
			return err
		}
	}

	return nil
}

// RevertPeerState takes the current and previous headers and undos the peer state
//  for all of the consensus members
func (vs *validatorStatistics) RevertPeerState(header data.HeaderHandler) error {
	return vs.peerAdapter.RecreateTrie(header.GetValidatorStatsRootHash())
}

func (vs *validatorStatistics) updateShardDataPeerState(
	header data.HeaderHandler,
	cacheMap map[string]data.HeaderHandler,
) error {
	metaHeader, ok := header.(*block.MetaBlock)
	if !ok {
		return process.ErrInvalidMetaHeader
	}

	for _, h := range metaHeader.ShardInfo {
		if h.Nonce == vs.genesisNonce {
			continue
		}

		currentHeader, ok := cacheMap[string(h.HeaderHash)]
		if !ok {
			return fmt.Errorf("%w - updateShardDataPeerState header from cache - hash: %s, round: %v, nonce: %v",
				process.ErrMissingHeader,
				hex.EncodeToString(h.HeaderHash),
				h.GetRound(),
				h.GetNonce())
		}

		epoch := computeEpoch(currentHeader)

		shardConsensus, shardInfoErr := vs.nodesCoordinator.ComputeConsensusGroup(h.PrevRandSeed, h.Round, h.ShardID, epoch)
		if shardInfoErr != nil {
			return shardInfoErr
		}

		shardInfoErr = vs.updateValidatorInfoOnSuccessfulBlock(
			shardConsensus,
			h.PubKeysBitmap,
			big.NewInt(0).Sub(h.AccumulatedFees, h.DeveloperFees),
			h.ShardID,
		)
		if shardInfoErr != nil {
			return shardInfoErr
		}

		if h.Nonce == vs.genesisNonce+1 {
			continue
		}

		prevShardData, shardInfoErr := vs.searchInMap(h.PrevHash, cacheMap)
		if shardInfoErr != nil {
			return shardInfoErr
		}

		shardInfoErr = vs.checkForMissedBlocks(
			h.Round,
			prevShardData.Round,
			prevShardData.RandSeed,
			h.ShardID,
			epoch,
		)
		if shardInfoErr != nil {
			return shardInfoErr
		}
	}

	return nil
}

func (vs *validatorStatistics) searchInMap(hash []byte, cacheMap map[string]data.HeaderHandler) (*block.Header, error) {
	blkHandler := cacheMap[string(hash)]
	if check.IfNil(blkHandler) {
		return nil, fmt.Errorf("%w : searchInMap hash = %s",
			process.ErrMissingHeader, logger.DisplayByteSlice(hash))
	}

	blk, ok := blkHandler.(*block.Header)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	return blk, nil
}

func (vs *validatorStatistics) initializeNode(
	node sharding.GenesisNodeInfoHandler,
	startRating uint32,
	shardID uint32,
	peerType core.PeerType,
	index uint32,
) error {
	peerAccount, err := vs.GetPeerAccount(node.PubKeyBytes())
	if err != nil {
		return err
	}

	return vs.savePeerAccountData(peerAccount, node, startRating, shardID, peerType, index)
}

func (vs *validatorStatistics) savePeerAccountData(
	peerAccount state.PeerAccountHandler,
	node sharding.GenesisNodeInfoHandler,
	startRating uint32,
	shardID uint32,
	peerType core.PeerType,
	index uint32,
) error {
	err := peerAccount.SetRewardAddress(node.AddressBytes())
	if err != nil {
		return err
	}

	err = peerAccount.SetBLSPublicKey(node.PubKeyBytes())
	if err != nil {
		return err
	}

	peerAccount.SetRating(startRating)
	peerAccount.SetTempRating(startRating)
	peerAccount.SetListAndIndex(shardID, string(peerType), index)

	return vs.peerAdapter.SaveAccount(peerAccount)
}

func (vs *validatorStatistics) updateValidatorInfoOnSuccessfulBlock(
	validatorList []sharding.Validator,
	signingBitmap []byte,
	accumulatedFees *big.Int,
	shardId uint32,
) error {

	if len(signingBitmap) == 0 {
		return process.ErrNilPubKeysBitmap
	}
	lenValidators := len(validatorList)
	for i := 0; i < lenValidators; i++ {
		peerAcc, err := vs.GetPeerAccount(validatorList[i].PubKey())
		if err != nil {
			return err
		}

		peerAcc.IncreaseNumSelectedInSuccessBlocks()

		newRating := peerAcc.GetRating()
		isLeader := i == 0
		validatorSigned := (signingBitmap[i/8] & (1 << (uint16(i) % 8))) != 0
		actionType := vs.computeValidatorActionType(isLeader, validatorSigned)

		switch actionType {
		case leaderSuccess:
			peerAcc.IncreaseLeaderSuccessRate(1)
			peerAcc.SetConsecutiveProposerMisses(0)
			newRating = vs.rater.ComputeIncreaseProposer(shardId, peerAcc.GetTempRating())
			leaderAccumulatedFees := core.GetPercentageOfValue(accumulatedFees, vs.rewardsHandler.LeaderPercentage())
			peerAcc.AddToAccumulatedFees(leaderAccumulatedFees)
		case validatorSuccess:
			peerAcc.IncreaseValidatorSuccessRate(1)
			newRating = vs.rater.ComputeIncreaseValidator(shardId, peerAcc.GetTempRating())
		case validatorIgnoredSignature:
			peerAcc.IncreaseValidatorIgnoredSignaturesRate(1)
			newRating = vs.rater.ComputeIncreaseValidator(shardId, peerAcc.GetTempRating())
		}

		peerAcc.SetTempRating(newRating)

		err = vs.peerAdapter.SaveAccount(peerAcc)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetPeerAccount will return a PeerAccountHandler for a given address
func (vs *validatorStatistics) GetPeerAccount(address []byte) (state.PeerAccountHandler, error) {
	account, err := vs.peerAdapter.LoadAccount(address)
	if err != nil {
		return nil, err
	}

	peerAccount, ok := account.(state.PeerAccountHandler)
	if !ok {
		return nil, process.ErrInvalidPeerAccount
	}

	return peerAccount, nil
}

func (vs *validatorStatistics) getMatchingPrevShardData(currentShardData block.ShardData, shardInfo []block.ShardData) *block.ShardData {
	for _, prevShardData := range shardInfo {
		if currentShardData.ShardID != prevShardData.ShardID {
			continue
		}
		if currentShardData.Nonce == prevShardData.Nonce+1 {
			return &prevShardData
		}
	}

	return nil
}

func (vs *validatorStatistics) updateMissedBlocksCounters() error {
	vs.mutValidatorStatistics.Lock()
	defer func() {
		vs.missedBlocksCounters.reset()
		vs.mutValidatorStatistics.Unlock()
	}()

	for pubKey, roundCounters := range vs.missedBlocksCounters {
		peerAccount, err := vs.GetPeerAccount([]byte(pubKey))
		if err != nil {
			return err
		}

		if roundCounters.leaderDecreaseCount > 0 {
			peerAccount.DecreaseLeaderSuccessRate(roundCounters.leaderDecreaseCount)
		}

		if roundCounters.validatorDecreaseCount > 0 {
			peerAccount.DecreaseValidatorSuccessRate(roundCounters.validatorDecreaseCount)
		}

		err = vs.peerAdapter.SaveAccount(peerAccount)
		if err != nil {
			return err
		}
	}

	return nil
}

func (vs *validatorStatistics) computeValidatorActionType(isLeader, validatorSigned bool) validatorActionType {
	if isLeader && validatorSigned {
		return leaderSuccess
	}
	if isLeader && !validatorSigned {
		return leaderFail
	}
	if !isLeader && validatorSigned {
		return validatorSuccess
	}
	if !isLeader && !validatorSigned {
		return validatorIgnoredSignature
	}

	return unknownAction
}

// IsInterfaceNil returns true if there is no value under the interface
func (vs *validatorStatistics) IsInterfaceNil() bool {
	return vs == nil
}

func (vs *validatorStatistics) getTempRating(s string) uint32 {
	peer, err := vs.GetPeerAccount([]byte(s))

	if err != nil {
		log.Debug("Error getting peer account", "error", err)
		return vs.rater.GetStartRating()
	}

	return peer.GetTempRating()
}

func (vs *validatorStatistics) display(validatorKey string) {
	peerAcc, err := vs.GetPeerAccount([]byte(validatorKey))

	if err != nil {
		log.Trace("display peer acc", "error", err)
		return
	}

	acc, ok := peerAcc.(state.PeerAccountHandler)

	if !ok {
		log.Trace("display", "error", "not a peeracc")
		return
	}

	log.Trace("validator statistics",
		"pk", acc.GetBLSPublicKey(),
		"leader fail", acc.GetLeaderSuccessRate().NumFailure,
		"leader success", acc.GetLeaderSuccessRate().NumSuccess,
		"val success", acc.GetValidatorSuccessRate().NumSuccess,
		"val ignored sigs", acc.GetValidatorIgnoredSignaturesRate(),
		"val fail", acc.GetValidatorSuccessRate().NumFailure,
		"temp rating", acc.GetTempRating(),
		"rating", acc.GetRating(),
	)
}

func (vs *validatorStatistics) decreaseAll(
	shardID uint32,
	missedRounds uint64,
	epoch uint32,
) error {
	if epoch < vs.ratingEnableEpoch {
		return nil
	}

	log.Trace("ValidatorStatistics decreasing all", "shardID", shardID, "missedRounds", missedRounds)
	consensusGroupSize := vs.nodesCoordinator.ConsensusGroupSize(shardID)
	validators, err := vs.nodesCoordinator.GetAllEligibleValidatorsPublicKeys(epoch)
	if err != nil {
		return err
	}

	shardValidators := validators[shardID]
	validatorsCount := len(shardValidators)
	percentageRoundMissedFromTotalValidators := float64(missedRounds) / float64(validatorsCount)
	leaderAppearances := uint32(percentageRoundMissedFromTotalValidators + 1 - math.SmallestNonzeroFloat64)
	consensusGroupAppearances := uint32(float64(consensusGroupSize)*percentageRoundMissedFromTotalValidators +
		1 - math.SmallestNonzeroFloat64)
	ratingDifference := uint32(0)

	for i, validator := range shardValidators {
		validatorPeerAccount, err := vs.GetPeerAccount(validator)
		if err != nil {
			return err
		}
		validatorPeerAccount.DecreaseLeaderSuccessRate(leaderAppearances)
		validatorPeerAccount.DecreaseValidatorSuccessRate(consensusGroupAppearances)

		currentTempRating := validatorPeerAccount.GetTempRating()
		for ct := uint32(0); ct < leaderAppearances; ct++ {
			currentTempRating = vs.rater.ComputeDecreaseProposer(shardID, currentTempRating, 0)
		}

		for ct := uint32(0); ct < consensusGroupAppearances; ct++ {
			currentTempRating = vs.rater.ComputeDecreaseValidator(shardID, currentTempRating)
		}

		if i == 0 {
			ratingDifference = validatorPeerAccount.GetTempRating() - currentTempRating
		}

		validatorPeerAccount.SetTempRating(currentTempRating)
		err = vs.peerAdapter.SaveAccount(validatorPeerAccount)
		if err != nil {
			return err
		}

		vs.display(string(validator))
	}

	log.Trace(fmt.Sprintf("Decrease leader: %v, decrease validator: %v, ratingDifference: %v", leaderAppearances, consensusGroupAppearances, ratingDifference))

	return nil
}

// Process - processes a validatorInfo and updates fields
func (vs *validatorStatistics) Process(svi data.ShardValidatorInfoHandler) error {
	log.Trace("ValidatorInfoData", "pk", svi.GetPublicKey(), "tempRating", svi.GetTempRating())

	pa, err := vs.GetPeerAccount(svi.GetPublicKey())
	if err != nil {
		return err
	}

	pa.SetRating(svi.GetTempRating())
	return vs.peerAdapter.SaveAccount(pa)
}

// SetLastFinalizedRootHash - sets the last finalized root hash needed for correct validatorStatistics computations
func (vs *validatorStatistics) SetLastFinalizedRootHash(lastFinalizedRootHash []byte) {
	if len(lastFinalizedRootHash) == 0 {
		return
	}

	vs.mutValidatorStatistics.Lock()
	vs.lastFinalizedRootHash = lastFinalizedRootHash
	vs.mutValidatorStatistics.Unlock()
}

// LastFinalizedRootHash returns the root hash of the validator statistics trie that was last finalized
func (vs *validatorStatistics) LastFinalizedRootHash() []byte {
	vs.mutValidatorStatistics.RLock()
	defer vs.mutValidatorStatistics.RUnlock()
	return vs.lastFinalizedRootHash
}
