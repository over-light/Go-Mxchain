package blockAPI

import (
	"encoding/hex"

	apiBlock "github.com/ElrondNetwork/elrond-go/api/block"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
)

type shardAPIBlockProcessor struct {
	*baseAPIBockProcessor
}

// NewShardApiBlockProcessor will create a new instance of shard api block processor
func NewShardApiBlockProcessor(arg *APIBlockProcessorArg) *shardAPIBlockProcessor {
	isFullHistoryNode := arg.HistoryRepo.IsEnabled()
	return &shardAPIBlockProcessor{
		baseAPIBockProcessor: &baseAPIBockProcessor{
			isFullHistoryNode:        isFullHistoryNode,
			selfShardID:              arg.SelfShardID,
			store:                    arg.Store,
			marshalizer:              arg.Marshalizer,
			uint64ByteSliceConverter: arg.Uint64ByteSliceConverter,
			historyRepo:              arg.HistoryRepo,
			unmarshalTx:              arg.UnmarshalTx,
		},
	}
}

// GetBlockByNonce will return a shard APIBlock by nonce
func (sbp *shardAPIBlockProcessor) GetBlockByNonce(nonce uint64, withTxs bool) (*apiBlock.APIBlock, error) {
	storerUnit := dataRetriever.ShardHdrNonceHashDataUnit + dataRetriever.UnitType(sbp.selfShardID)

	nonceToByteSlice := sbp.uint64ByteSliceConverter.ToByteSlice(nonce)
	headerHash, err := sbp.store.Get(storerUnit, nonceToByteSlice)
	if err != nil {
		return nil, err
	}

	blockBytes, err := sbp.getFromStorer(dataRetriever.BlockHeaderUnit, headerHash)
	if err != nil {
		return nil, err
	}

	return sbp.convertShardBlockBytesToAPIBlock(headerHash, blockBytes, withTxs)
}

// GetBlockByHash will return a shard APIBlock by hash
func (sbp *shardAPIBlockProcessor) GetBlockByHash(hash []byte, withTxs bool) (*apiBlock.APIBlock, error) {
	blockBytes, err := sbp.getFromStorer(dataRetriever.BlockHeaderUnit, hash)
	if err != nil {
		return nil, err
	}

	return sbp.convertShardBlockBytesToAPIBlock(hash, blockBytes, withTxs)
}

func (sbp *shardAPIBlockProcessor) convertShardBlockBytesToAPIBlock(hash []byte, blockBytes []byte, withTxs bool) (*apiBlock.APIBlock, error) {
	blockHeader := &block.Header{}
	err := sbp.marshalizer.Unmarshal(blockHeader, blockBytes)
	if err != nil {
		return nil, err
	}

	headerEpoch := blockHeader.Epoch

	numOfTxs := uint32(0)
	miniblocks := make([]*apiBlock.APIMiniBlock, 0)
	for _, mb := range blockHeader.MiniBlockHeaders {
		if mb.Type == block.PeerBlock {
			continue
		}

		numOfTxs += mb.TxCount

		miniblockAPI := &apiBlock.APIMiniBlock{
			Hash:               hex.EncodeToString(mb.Hash),
			Type:               mb.Type.String(),
			SourceShardID:      mb.SenderShardID,
			DestinationShardID: mb.ReceiverShardID,
		}
		if withTxs {
			miniblockAPI.Transactions = sbp.getTxsByMb(&mb, headerEpoch)
		}

		miniblocks = append(miniblocks, miniblockAPI)
	}

	return &apiBlock.APIBlock{
		Nonce:         blockHeader.Nonce,
		Round:         blockHeader.Round,
		Epoch:         blockHeader.Epoch,
		ShardID:       blockHeader.ShardID,
		Hash:          hex.EncodeToString(hash),
		PrevBlockHash: hex.EncodeToString(blockHeader.PrevHash),
		NumTxs:        numOfTxs,
		MiniBlocks:    miniblocks,
	}, nil
}
