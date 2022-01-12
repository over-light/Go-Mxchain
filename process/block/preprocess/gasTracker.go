package preprocess

import (
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type gasTracker struct {
	shardCoordinator sharding.Coordinator
	economicsFee     process.FeeHandler
	gasHandler       process.GasHandler
}

func (gt *gasTracker) computeGasConsumed(
	senderShardId uint32,
	receiverShardId uint32,
	tx data.TransactionHandler,
	txHash []byte,
	gasInfo *gasConsumedInfo,
) (uint64, error) {
	gasConsumedByTxInSenderShard, gasConsumedByTxInReceiverShard, err := gt.computeGasConsumedByTx(
		senderShardId,
		receiverShardId,
		tx,
		txHash)
	if err != nil {
		return 0, err
	}

	gasConsumedByTxInSelfShard := uint64(0)
	if gt.shardCoordinator.SelfId() == senderShardId {
		gasConsumedByTxInSelfShard = gasConsumedByTxInSenderShard

		if gasConsumedByTxInReceiverShard > gt.economicsFee.MaxGasLimitPerTx() {
			return 0, process.ErrMaxGasLimitPerOneTxInReceiverShardIsReached
		}

		if gasInfo.gasConsumedByMiniBlockInReceiverShard+gasConsumedByTxInReceiverShard > gt.economicsFee.MaxGasLimitPerBlockForSafeCrossShard() {
			return 0, process.ErrMaxGasLimitPerMiniBlockInReceiverShardIsReached
		}
	} else {
		gasConsumedByTxInSelfShard = gasConsumedByTxInReceiverShard
	}

	if gasInfo.totalGasConsumedInSelfShard+gasConsumedByTxInSelfShard > gt.economicsFee.MaxGasLimitPerBlock(gt.shardCoordinator.SelfId()) {
		return 0, process.ErrMaxGasLimitPerBlockInSelfShardIsReached
	}

	gasInfo.gasConsumedByMiniBlocksInSenderShard += gasConsumedByTxInSenderShard
	gasInfo.gasConsumedByMiniBlockInReceiverShard += gasConsumedByTxInReceiverShard
	gasInfo.totalGasConsumedInSelfShard += gasConsumedByTxInSelfShard

	return gasConsumedByTxInSelfShard, nil
}

func (gt *gasTracker) computeGasConsumedByTx(
	senderShardId uint32,
	receiverShardId uint32,
	tx data.TransactionHandler,
	txHash []byte,
) (uint64, uint64, error) {

	txGasLimitInSenderShard, txGasLimitInReceiverShard, err := gt.gasHandler.ComputeGasConsumedByTx(
		senderShardId,
		receiverShardId,
		tx)
	if err != nil {
		return 0, 0, err
	}

	if core.IsSmartContractAddress(tx.GetRcvAddr()) {
		txGasRefunded := gt.gasHandler.GasRefunded(txHash)
		txGasPenalized := gt.gasHandler.GasPenalized(txHash)
		txGasToBeSubtracted := txGasRefunded + txGasPenalized
		if txGasLimitInReceiverShard < txGasToBeSubtracted {
			return 0, 0, process.ErrInsufficientGasLimitInTx
		}

		if senderShardId == receiverShardId {
			txGasLimitInSenderShard -= txGasToBeSubtracted
			txGasLimitInReceiverShard -= txGasToBeSubtracted
		}
	}

	return txGasLimitInSenderShard, txGasLimitInReceiverShard, nil
}
