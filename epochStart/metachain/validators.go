package metachain

import (
	"bytes"
	"sort"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
)

// ArgsNewValidatorInfoCreator defines the arguments structure needed to create a new validatorInfo creator
type ArgsNewValidatorInfoCreator struct {
	ShardCoordinator sharding.Coordinator
	MiniBlockStorage storage.Storer
	Hasher           hashing.Hasher
	Marshalizer      marshal.Marshalizer
}

type validatorInfoCreator struct {
	shardCoordinator sharding.Coordinator
	miniBlockStorage storage.Storer
	hasher           hashing.Hasher
	marshalizer      marshal.Marshalizer
}

// NewValidatorInfoCreator creates a new validatorInfo creator object
func NewValidatorInfoCreator(args ArgsNewValidatorInfoCreator) (*validatorInfoCreator, error) {
	if check.IfNil(args.ShardCoordinator) {
		return nil, epochStart.ErrNilShardCoordinator
	}
	if check.IfNil(args.Marshalizer) {
		return nil, epochStart.ErrNilMarshalizer
	}
	if check.IfNil(args.Hasher) {
		return nil, epochStart.ErrNilHasher
	}
	if check.IfNil(args.MiniBlockStorage) {
		return nil, epochStart.ErrNilStorage
	}

	vic := &validatorInfoCreator{
		shardCoordinator: args.ShardCoordinator,
		hasher:           args.Hasher,
		marshalizer:      args.Marshalizer,
		miniBlockStorage: args.MiniBlockStorage,
	}

	return vic, nil
}

// CreateValidatorInfoMiniBlocks creates the validatorInfo miniblocks according to the provided validatorInfo map
func (r *validatorInfoCreator) CreateValidatorInfoMiniBlocks(validatorInfo map[uint32][]*state.ValidatorInfo) (block.MiniBlockSlice, error) {
	if validatorInfo == nil {
		return nil, epochStart.ErrNilValidatorInfo
	}

	miniblocks := make([]*block.MiniBlock, 0)

	for shardId := uint32(0); shardId < r.shardCoordinator.NumberOfShards(); shardId++ {
		validators := validatorInfo[shardId]
		if len(validators) == 0 {
			continue
		}

		miniBlock, err := r.createMiniBlock(validators)
		if err != nil {
			return nil, err
		}

		miniblocks = append(miniblocks, miniBlock)
	}

	validators := validatorInfo[core.MetachainShardId]
	if len(validators) == 0 {
		return miniblocks, nil
	}

	miniBlock, err := r.createMiniBlock(validators)
	if err != nil {
		return nil, err
	}

	miniblocks = append(miniblocks, miniBlock)

	return miniblocks, nil
}

func (r *validatorInfoCreator) createMiniBlock(validators []*state.ValidatorInfo) (*block.MiniBlock, error) {
	miniBlock := &block.MiniBlock{}
	miniBlock.SenderShardID = r.shardCoordinator.SelfId()
	miniBlock.ReceiverShardID = core.AllShardId
	miniBlock.TxHashes = make([][]byte, len(validators))
	miniBlock.Type = block.PeerBlock

	validatorCopy := make([]*state.ValidatorInfo, len(validators))
	copy(validatorCopy, validators)
	sort.Slice(validatorCopy, func(a, b int) bool {
		return bytes.Compare(validatorCopy[a].PublicKey, validatorCopy[b].PublicKey) < 0
	})

	for index, validator := range validatorCopy {
		marshalizedValidator, err := r.marshalizer.Marshal(validator)
		if err != nil {
			return nil, err
		}
		miniBlock.TxHashes[index] = marshalizedValidator
	}
	return miniBlock, nil
}

// VerifyValidatorInfoMiniBlocks verifies if received validatorinfo miniblocks are correct
func (r *validatorInfoCreator) VerifyValidatorInfoMiniBlocks(
	miniblocks []*block.MiniBlock,
	validatorInfos map[uint32][]*state.ValidatorInfo,
) error {
	if len(miniblocks) == 0 {
		return epochStart.ErrNilMiniblocks
	}

	createdMiniBlocks, err := r.CreateValidatorInfoMiniBlocks(validatorInfos)
	if err != nil {
		return err
	}

	hashesToMiniBlocks := make(map[string]*block.MiniBlock)
	for _, mb := range createdMiniBlocks {
		hash, hashError := core.CalculateHash(r.marshalizer, r.hasher, mb)
		if hashError != nil {
			return hashError
		}
		hashesToMiniBlocks[string(hash)] = mb
	}

	numReceivedValidatorInfoMBs := 0
	var receivedMbHash []byte
	for _, receivedMb := range miniblocks {
		if receivedMb == nil {
			return epochStart.ErrNilMiniblock
		}

		if receivedMb.Type != block.PeerBlock {
			continue
		}

		numReceivedValidatorInfoMBs++
		receivedMbHash, err = core.CalculateHash(r.marshalizer, r.hasher, receivedMb)
		if err != nil {
			return err
		}

		_, ok := hashesToMiniBlocks[string(receivedMbHash)]

		if !ok {
			// TODO: add display debug prints of miniblocks contents
			return epochStart.ErrValidatorMiniBlockHashDoesNotMatch
		}
	}

	if len(createdMiniBlocks) != numReceivedValidatorInfoMBs {
		return epochStart.ErrValidatorInfoMiniBlocksNumDoesNotMatch
	}

	return nil
}

// SaveValidatorInfoBlocksToStorage saves created data to storage
func (r *validatorInfoCreator) SaveValidatorInfoBlocksToStorage(
	metaBlock *block.MetaBlock,
	body *block.Body,
	dataPool dataRetriever.PoolsHolder,
) {
	for _, miniBlock := range body.MiniBlocks {
		if miniBlock.Type != block.PeerBlock {
			continue
		}

		for _, mbHeader := range metaBlock.MiniBlockHeaders {
			if mbHeader.Type != block.PeerBlock {
				continue
			}

			marshaledData, err := r.marshalizer.Marshal(miniBlock)
			if err != nil {
				continue
			}

			mbHash := r.hasher.Compute(string(marshaledData))
			if !bytes.Equal(mbHeader.Hash, mbHash) {
				continue
			}

			_ = r.miniBlockStorage.Put(mbHeader.Hash, marshaledData)
			dataPool.MiniBlocks().Remove(mbHeader.Hash)
			break
		}
	}
}

// DeleteValidatorInfoBlocksFromStorage deletes data from storage
func (r *validatorInfoCreator) DeleteValidatorInfoBlocksFromStorage(metaBlock *block.MetaBlock) {
	for _, mbHeader := range metaBlock.MiniBlockHeaders {
		if mbHeader.Type == block.PeerBlock {
			_ = r.miniBlockStorage.Remove(mbHeader.Hash)
		}
	}
}

// IsInterfaceNil return true if underlying object is nil
func (r *validatorInfoCreator) IsInterfaceNil() bool {
	return r == nil
}
