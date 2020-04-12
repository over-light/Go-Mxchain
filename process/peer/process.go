package peer

import (
	"bytes"
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

type validatorActionType uint8

const (
	unknownAction    validatorActionType = 0
	leaderSuccess    validatorActionType = 1
	leaderFail       validatorActionType = 2
	validatorSuccess validatorActionType = 3
	validatorFail    validatorActionType = 4
)

// ArgValidatorStatisticsProcessor holds all dependencies for the validatorStatistics
type ArgValidatorStatisticsProcessor struct {
	StakeValue          *big.Int
	Marshalizer         marshal.Marshalizer
	NodesCoordinator    sharding.NodesCoordinator
	ShardCoordinator    sharding.Coordinator
	DataPool            DataPool
	StorageService      dataRetriever.StorageService
	AdrConv             state.AddressConverter
	PeerAdapter         state.AccountsAdapter
	Rater               sharding.PeerAccountListAndRatingHandler
	RewardsHandler      process.RewardsHandler
	MaxComputableRounds uint64
	StartEpoch          uint32
	NodesSetup          sharding.GenesisNodesSetupHandler
}

type validatorStatistics struct {
	marshalizer             marshal.Marshalizer
	dataPool                DataPool
	storageService          dataRetriever.StorageService
	nodesCoordinator        sharding.NodesCoordinator
	shardCoordinator        sharding.Coordinator
	adrConv                 state.AddressConverter
	peerAdapter             state.AccountsAdapter
	rater                   sharding.PeerAccountListAndRatingHandler
	rewardsHandler          process.RewardsHandler
	maxComputableRounds     uint64
	missedBlocksCounters    validatorRoundCounters
	mutMissedBlocksCounters sync.RWMutex
}

// NewValidatorStatisticsProcessor instantiates a new validatorStatistics structure responsible of keeping account of
//  each validator actions in the consensus process
func NewValidatorStatisticsProcessor(arguments ArgValidatorStatisticsProcessor) (*validatorStatistics, error) {
	if check.IfNil(arguments.PeerAdapter) {
		return nil, process.ErrNilPeerAccountsAdapter
	}
	if check.IfNil(arguments.AdrConv) {
		return nil, process.ErrNilAddressConverter
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
	if arguments.StakeValue == nil {
		return nil, process.ErrNilEconomicsData
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
		adrConv:              arguments.AdrConv,
		nodesCoordinator:     arguments.NodesCoordinator,
		shardCoordinator:     arguments.ShardCoordinator,
		dataPool:             arguments.DataPool,
		storageService:       arguments.StorageService,
		marshalizer:          arguments.Marshalizer,
		missedBlocksCounters: make(validatorRoundCounters),
		rater:                arguments.Rater,
		rewardsHandler:       arguments.RewardsHandler,
		maxComputableRounds:  arguments.MaxComputableRounds,
	}

	rater := arguments.Rater
	err := vs.saveInitialState(arguments.NodesSetup, arguments.StakeValue, rater.GetStartRating())
	if err != nil {
		return nil, err
	}

	return vs, nil
}

// saveNodesCoordinatorUpdates is called at the first block after start in epoch to update the state trie according to
// the shuffling and changes done by the nodesCoordinator at the end of the epoch
func (vs *validatorStatistics) saveNodesCoordinatorUpdates(epoch uint32) error {
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

	nodesList, err := vs.nodesCoordinator.GetAllLeavingValidatorsPublicKeys(epoch)
	if err != nil {
		return err
	}

	err = vs.saveUpdatesForList(nodesList, 0, core.LeavingList)
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
	stakeValue *big.Int,
	startRating uint32,
) error {
	eligibleNodesInfo, waitingNodesInfo := nodesConfig.InitialNodesInfo()
	err := vs.saveInitialValueForMap(eligibleNodesInfo, stakeValue, startRating, core.EligibleList)
	if err != nil {
		return err
	}

	err = vs.saveInitialValueForMap(waitingNodesInfo, stakeValue, startRating, core.WaitingList)
	if err != nil {
		return err
	}

	hash, err := vs.peerAdapter.Commit()
	if err != nil {
		return err
	}

	log.Trace("committed peer adapter", "root hash", core.ToHex(hash))

	return nil
}

func (vs *validatorStatistics) saveInitialValueForMap(
	nodesInfo map[uint32][]sharding.GenesisNodeInfoHandler,
	stakeValue *big.Int,
	startRating uint32,
	peerType core.PeerType,
) error {
	if len(nodesInfo) == 0 {
		return nil
	}

	// TODO: check why range over map does not give the same validator statistics roothash
	for shardID := uint32(0); shardID < vs.shardCoordinator.NumberOfShards(); shardID++ {
		nodeInfoList := nodesInfo[shardID]
		for index, nodeInfo := range nodeInfoList {
			err := vs.initializeNode(nodeInfo, stakeValue, startRating, shardID, peerType, uint32(index))
			if err != nil {
				return err
			}
		}
	}

	shardID := core.MetachainShardId
	nodeInfoList := nodesInfo[shardID]
	for index, nodeInfo := range nodeInfoList {
		err := vs.initializeNode(nodeInfo, stakeValue, startRating, shardID, peerType, uint32(index))
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdatePeerState takes a header, updates the peer state for all of the
//  consensus members and returns the new root hash
func (vs *validatorStatistics) UpdatePeerState(header data.HeaderHandler, cache map[string]data.HeaderHandler) ([]byte, error) {
	if header.GetNonce() == 0 {
		return vs.peerAdapter.RootHash()
	}

	vs.mutMissedBlocksCounters.Lock()
	vs.missedBlocksCounters.reset()
	vs.mutMissedBlocksCounters.Unlock()

	// TODO: remove if start of epoch block needs to be validated by the new epoch nodes
	epoch := header.GetEpoch()
	if header.IsStartOfEpochBlock() && epoch > 0 {
		epoch = epoch - 1
	}

	previousHeader, ok := cache[string(header.GetPrevHash())]
	if !ok {
		log.Warn("UpdatePeerState could not get meta header from cache", "error", process.ErrMissingHeader.Error(), "hash", header.GetPrevHash(), "round", header.GetRound(), "nonce", header.GetNonce())
		return nil, process.ErrMissingHeader
	}

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

	if header.GetNonce() == 1 {
		return vs.peerAdapter.RootHash()
	}
	log.Trace("Increasing", "round", previousHeader.GetRound(), "prevRandSeed", previousHeader.GetPrevRandSeed())
	consensusGroup, err := vs.nodesCoordinator.ComputeConsensusGroup(previousHeader.GetPrevRandSeed(), previousHeader.GetRound(), previousHeader.GetShardID(), epoch)
	if err != nil {
		return nil, err
	}
	leaderPK := core.GetTrimmedPk(core.ToHex(consensusGroup[0].PubKey()))
	log.Trace("Increasing for leader", "leader", leaderPK, "round", previousHeader.GetRound())
	err = vs.updateValidatorInfo(consensusGroup, previousHeader.GetPubKeysBitmap(), previousHeader.GetAccumulatedFees(), previousHeader.GetShardID())
	if err != nil {
		return nil, err
	}

	rootHash, err := vs.peerAdapter.Commit()
	vs.displayRatings(header.GetEpoch())
	if err != nil {
		return nil, err
	}

	log.Trace("after updating validator stats", "rootHash", rootHash, "round", header.GetRound(), "selfId", vs.shardCoordinator.SelfId())

	return rootHash, nil
}

func (vs *validatorStatistics) displayRatings(epoch uint32) {
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

		currentShardId := peerAccount.GetCurrentShardId()
		validatorInfoData := vs.peerAccountToValidatorInfo(peerAccount)
		validators[currentShardId] = append(validators[currentShardId], validatorInfoData)
	}

	return validators, nil
}

func (vs *validatorStatistics) peerAccountToValidatorInfo(peerAccount state.PeerAccountHandler) *state.ValidatorInfo {
	return &state.ValidatorInfo{
		PublicKey:                  peerAccount.GetBLSPublicKey(),
		ShardId:                    peerAccount.GetCurrentShardId(),
		List:                       peerAccount.GetList(),
		Index:                      peerAccount.GetIndex(),
		TempRating:                 peerAccount.GetTempRating(),
		Rating:                     peerAccount.GetRating(),
		RewardAddress:              peerAccount.GetRewardAddress(),
		LeaderSuccess:              peerAccount.GetLeaderSuccessRate().NumSuccess,
		LeaderFailure:              peerAccount.GetLeaderSuccessRate().NumFailure,
		ValidatorSuccess:           peerAccount.GetValidatorSuccessRate().NumSuccess,
		ValidatorFailure:           peerAccount.GetValidatorSuccessRate().NumFailure,
		TotalLeaderSuccess:         peerAccount.GetTotalLeaderSuccessRate().NumSuccess,
		TotalLeaderFailure:         peerAccount.GetTotalLeaderSuccessRate().NumFailure,
		TotalValidatorSuccess:      peerAccount.GetTotalValidatorSuccessRate().NumSuccess,
		TotalValidatorFailure:      peerAccount.GetTotalValidatorSuccessRate().NumFailure,
		NumSelectedInSuccessBlocks: peerAccount.GetNumSelectedInSuccessBlocks(),
		AccumulatedFees:            big.NewInt(0).Set(peerAccount.GetAccumulatedFees()),
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
func (vs *validatorStatistics) ProcessRatingsEndOfEpoch(validatorInfos map[uint32][]*state.ValidatorInfo) error {
	if len(validatorInfos) == 0 {
		return process.ErrNilValidatorInfos
	}

	signedThreshold := vs.rater.GetSignedBlocksThreshold()
	for shardId, validators := range validatorInfos {
		for _, validator := range validators {
			err := vs.verifySignaturesBelowSignedThreshold(validator, signedThreshold, shardId)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (vs *validatorStatistics) verifySignaturesBelowSignedThreshold(validator *state.ValidatorInfo, signedThreshold float32, shardId uint32) error {
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
			addrContainer, err := vs.adrConv.CreateAddressFromPublicKeyBytes(validator.GetPublicKey())
			if err != nil {
				return err
			}
			account, err := vs.peerAdapter.LoadAccount(addrContainer)
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

func (vs *validatorStatistics) computeDecrease(previousHeaderRound uint64, currentHeaderRound uint64, prevRandSeed []byte, shardID uint32, epoch uint32) error {
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
		leaderPK := core.GetTrimmedPk(core.ToHex(consensusGroup[0].PubKey()))
		log.Trace("Decreasing for leader", "leader", leaderPK, "round", i)
		swInner.Stop("GetPeerAccount")
		if err != nil {
			return err
		}

		vs.mutMissedBlocksCounters.Lock()
		vs.missedBlocksCounters.decreaseLeader(consensusGroup[0].PubKey())
		vs.mutMissedBlocksCounters.Unlock()

		swInner.Start("ComputeDecreaseProposer")
		newRating := vs.rater.ComputeDecreaseProposer(shardID, leaderPeerAcc.GetTempRating(), leaderPeerAcc.GetConsecutiveProposerMisses())
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
		err = vs.decreaseForConsensusValidators(consensusGroup, shardID)
		swInner.Stop("ComputeDecreaseAllValidators")
		if err != nil {
			return err
		}
		sw.Add(swInner)
	}
	return nil
}

func (vs *validatorStatistics) decreaseForConsensusValidators(consensusGroup []sharding.Validator, shardId uint32) error {
	vs.mutMissedBlocksCounters.Lock()
	defer vs.mutMissedBlocksCounters.Unlock()

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

func (vs *validatorStatistics) updateShardDataPeerState(header data.HeaderHandler, cacheMap map[string]data.HeaderHandler) error {
	metaHeader, ok := header.(*block.MetaBlock)
	if !ok {
		return process.ErrInvalidMetaHeader
	}

	// TODO: remove if start of epoch block needs to be validated by the new epoch nodes
	epoch := header.GetEpoch()
	if header.IsStartOfEpochBlock() && epoch > 0 {
		epoch = epoch - 1
	}

	for _, h := range metaHeader.ShardInfo {
		shardConsensus, shardInfoErr := vs.nodesCoordinator.ComputeConsensusGroup(h.PrevRandSeed, h.Round, h.ShardID, epoch)
		if shardInfoErr != nil {
			return shardInfoErr
		}

		shardInfoErr = vs.updateValidatorInfo(shardConsensus, h.PubKeysBitmap, h.AccumulatedFees, h.ShardID)
		if shardInfoErr != nil {
			return shardInfoErr
		}

		if h.Nonce == 1 {
			continue
		}

		prevShardData, shardInfoErr := vs.searchInMap(h.PrevHash, cacheMap)
		if shardInfoErr != nil {
			return shardInfoErr
		}

		shardInfoErr = vs.checkForMissedBlocks(
			h.Round,
			prevShardData.Round,
			prevShardData.PrevRandSeed,
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
	stakeValue *big.Int,
	startRating uint32,
	shardID uint32,
	peerType core.PeerType,
	index uint32,
) error {
	peerAccount, err := vs.GetPeerAccount(node.PubKey())
	if err != nil {
		return err
	}

	return vs.savePeerAccountData(peerAccount, node, stakeValue, startRating, shardID, peerType, index)
}

func (vs *validatorStatistics) savePeerAccountData(
	peerAccount state.PeerAccountHandler,
	node sharding.GenesisNodeInfoHandler,
	stakeValue *big.Int,
	startRating uint32,
	shardID uint32,
	peerType core.PeerType,
	index uint32,
) error {
	err := peerAccount.SetRewardAddress(node.Address())
	if err != nil {
		return err
	}

	err = peerAccount.SetBLSPublicKey(node.PubKey())
	if err != nil {
		return err
	}

	err = peerAccount.SetStake(stakeValue)
	if err != nil {
		return err
	}

	peerAccount.SetCurrentShardId(shardID)
	peerAccount.SetRating(startRating)
	peerAccount.SetTempRating(startRating)
	peerAccount.SetListAndIndex(shardID, string(peerType), index)

	return vs.peerAdapter.SaveAccount(peerAccount)
}

func (vs *validatorStatistics) updateValidatorInfo(validatorList []sharding.Validator, signingBitmap []byte, accumulatedFees *big.Int, shardId uint32) error {
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

		var newRating uint32
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
		case validatorFail:
			peerAcc.DecreaseValidatorSuccessRate(1)
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
	addressContainer, err := vs.adrConv.CreateAddressFromPublicKeyBytes(address)
	if err != nil {
		return nil, err
	}

	account, err := vs.peerAdapter.LoadAccount(addressContainer)
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
	vs.mutMissedBlocksCounters.Lock()
	defer func() {
		vs.missedBlocksCounters.reset()
		vs.mutMissedBlocksCounters.Unlock()
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
		return validatorFail
	}

	return unknownAction
}

// IsInterfaceNil returns true if there is no value under the interface
func (vs *validatorStatistics) IsInterfaceNil() bool {
	return vs == nil
}

func (vs *validatorStatistics) getRating(s string) uint32 {
	peer, err := vs.GetPeerAccount([]byte(s))
	if err != nil {
		log.Debug("error getting peer account", "error", err, "key", s)
		return vs.rater.GetStartRating()
	}

	return peer.GetRating()
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
		"val fail", acc.GetValidatorSuccessRate().NumFailure,
		"val success", acc.GetValidatorSuccessRate().NumSuccess,
		"temp rating", acc.GetTempRating(),
		"rating", acc.GetRating(),
	)
}

func (vs *validatorStatistics) decreaseAll(shardID uint32, missedRounds uint64, epoch uint32) error {

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
