package node

import (
	"encoding/hex"

	apiBlock "github.com/ElrondNetwork/elrond-go/api/block"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/node/blockAPI"
)

// GetBlockByHash return the block for a given hash
func (n *Node) GetBlockByHash(hash string, withTxs bool) (*apiBlock.APIBlock, error) {
	decodedHash, err := hex.DecodeString(hash)
	if err != nil {
		return nil, err
	}

	apiBlockProcessor := n.createAPIBlockProcessor()
	return apiBlockProcessor.GetBlockByHash(decodedHash, withTxs)
}

// GetBlockByNonce returns the block for a given nonce
func (n *Node) GetBlockByNonce(nonce uint64, withTxs bool) (*apiBlock.APIBlock, error) {
	apiBlockProcessor := n.createAPIBlockProcessor()

	return apiBlockProcessor.GetBlockByNonce(nonce, withTxs)
}

func (n *Node) createAPIBlockProcessor() blockAPI.APIBlockHandler {
	if n.processComponents.ShardCoordinator().SelfId() != core.MetachainShardId {
		return blockAPI.NewShardApiBlockProcessor(
			&blockAPI.APIBlockProcessorArg{
				SelfShardID:              n.processComponents.ShardCoordinator().SelfId(),
				Store:                    n.dataComponents.StorageService(),
				Marshalizer:              n.coreComponents.InternalMarshalizer(),
				Uint64ByteSliceConverter: n.coreComponents.Uint64ByteSliceConverter(),
				HistoryRepo:              n.processComponents.HistoryRepository(),
				UnmarshalTx:              n.unmarshalTransaction,
			},
		)
	}

	return blockAPI.NewMetaApiBlockProcessor(
		&blockAPI.APIBlockProcessorArg{
			SelfShardID:              n.processComponents.ShardCoordinator().SelfId(),
			Store:                    n.dataComponents.StorageService(),
			Marshalizer:              n.coreComponents.InternalMarshalizer(),
			Uint64ByteSliceConverter: n.coreComponents.Uint64ByteSliceConverter(),
			HistoryRepo:              n.processComponents.HistoryRepository(),
			UnmarshalTx:              n.unmarshalTransaction,
		},
	)
}
