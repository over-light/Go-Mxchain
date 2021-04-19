package coordinator

import (
	"bytes"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/vmcommon"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

var _ process.TxTypeHandler = (*txTypeHandler)(nil)

type txTypeHandler struct {
	pubkeyConv       core.PubkeyConverter
	shardCoordinator sharding.Coordinator
	builtInFuncNames map[string]struct{}
	argumentParser   process.CallArgumentsParser
}

// ArgNewTxTypeHandler defines the arguments needed to create a new tx type handler
type ArgNewTxTypeHandler struct {
	PubkeyConverter  core.PubkeyConverter
	ShardCoordinator sharding.Coordinator
	BuiltInFuncNames map[string]struct{}
	ArgumentParser   process.CallArgumentsParser
}

// NewTxTypeHandler creates a transaction type handler
func NewTxTypeHandler(
	args ArgNewTxTypeHandler,
) (*txTypeHandler, error) {
	if check.IfNil(args.PubkeyConverter) {
		return nil, process.ErrNilPubkeyConverter
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}
	if check.IfNil(args.ArgumentParser) {
		return nil, process.ErrNilArgumentParser
	}
	if args.BuiltInFuncNames == nil {
		return nil, process.ErrNilBuiltInFunction
	}

	tc := &txTypeHandler{
		pubkeyConv:       args.PubkeyConverter,
		shardCoordinator: args.ShardCoordinator,
		argumentParser:   args.ArgumentParser,
		builtInFuncNames: args.BuiltInFuncNames,
	}

	return tc, nil
}

// ComputeTransactionType calculates the transaction type
func (tth *txTypeHandler) ComputeTransactionType(tx data.TransactionHandler) (process.TransactionType, process.TransactionType) {
	err := tth.checkTxValidity(tx)
	if err != nil {
		return process.InvalidTransaction, process.InvalidTransaction
	}

	isEmptyAddress := tth.isDestAddressEmpty(tx)
	if isEmptyAddress {
		if len(tx.GetData()) > 0 {
			return process.SCDeployment, process.SCDeployment
		}
		return process.InvalidTransaction, process.InvalidTransaction
	}

	if len(tx.GetData()) == 0 {
		return process.MoveBalance, process.MoveBalance
	}

	funcName, args := tth.getFunctionFromArguments(tx.GetData())
	isBuiltInFunction := tth.isBuiltInFunctionCall(funcName)
	if isBuiltInFunction {
		if tth.isSCCallAfterBuiltIn(funcName, args, tx) {
			return process.BuiltInFunctionCall, process.SCInvoking
		}

		return process.BuiltInFunctionCall, process.BuiltInFunctionCall
	}

	if isAsynchronousCallBack(tx) {
		return process.SCInvoking, process.SCInvoking
	}

	if len(funcName) == 0 {
		return process.MoveBalance, process.MoveBalance
	}

	if tth.isRelayedTransactionV1(funcName) {
		return process.RelayedTx, process.RelayedTx
	}

	if tth.isRelayedTransactionV2(funcName) {
		return process.RelayedTxV2, process.RelayedTxV2
	}

	isDestInSelfShard := tth.isDestAddressInSelfShard(tx.GetRcvAddr())
	if isDestInSelfShard && core.IsSmartContractAddress(tx.GetRcvAddr()) {
		return process.SCInvoking, process.SCInvoking
	}

	if core.IsSmartContractAddress(tx.GetRcvAddr()) {
		return process.MoveBalance, process.SCInvoking
	}

	return process.MoveBalance, process.MoveBalance
}

func isAsynchronousCallBack(tx data.TransactionHandler) bool {
	scr, ok := tx.(*smartContractResult.SmartContractResult)
	if !ok {
		return false
	}

	return scr.CallType == vmcommon.AsynchronousCallBack
}

func (tth *txTypeHandler) isSCCallAfterBuiltIn(function string, args [][]byte, tx data.TransactionHandler) bool {
	if !core.IsSmartContractAddress(tx.GetRcvAddr()) {
		return false
	}
	if len(args) <= 2 {
		return false
	}
	if function != core.BuiltInFunctionESDTTransfer {
		return false
	}

	return true
}

func (tth *txTypeHandler) getFunctionFromArguments(txData []byte) (string, [][]byte) {
	if len(txData) == 0 {
		return "", nil
	}

	function, args, err := tth.argumentParser.ParseData(string(txData))
	if err != nil {
		return "", nil
	}

	return function, args
}

func (tth *txTypeHandler) isBuiltInFunctionCall(functionName string) bool {
	if len(tth.builtInFuncNames) == 0 {
		return false
	}

	_, ok := tth.builtInFuncNames[functionName]
	return ok
}

func (tth *txTypeHandler) isRelayedTransactionV1(functionName string) bool {
	return functionName == core.RelayedTransaction
}

func (tth *txTypeHandler) isRelayedTransactionV2(functionName string) bool {
	return functionName == core.RelayedTransactionV2
}

func (tth *txTypeHandler) isDestAddressEmpty(tx data.TransactionHandler) bool {
	isEmptyAddress := bytes.Equal(tx.GetRcvAddr(), make([]byte, tth.pubkeyConv.Len()))
	return isEmptyAddress
}

func (tth *txTypeHandler) isDestAddressInSelfShard(address []byte) bool {
	shardForCurrentNode := tth.shardCoordinator.SelfId()
	shardForSrc := tth.shardCoordinator.ComputeId(address)
	return shardForCurrentNode == shardForSrc
}

func (tth *txTypeHandler) checkTxValidity(tx data.TransactionHandler) error {
	if check.IfNil(tx) {
		return process.ErrNilTransaction
	}

	recvAddressIsInvalid := tth.pubkeyConv.Len() != len(tx.GetRcvAddr())
	if recvAddressIsInvalid {
		return process.ErrWrongTransaction
	}

	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (tth *txTypeHandler) IsInterfaceNil() bool {
	return tth == nil
}
