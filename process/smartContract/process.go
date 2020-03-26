package smartContract

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"

	"github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/mitchellh/mapstructure"
)

// claimDeveloperRewardsFunctionName is a constant which defines the name for the claim developer rewards function
const claimDeveloperRewardsFunctionName = "ClaimDeveloperRewards"

// changeOwnerAddressFunctionName is a constant which defines the name for the change owner address function
const changeOwnerAddressFunctionName = "ChangeOwnerAddress"

var log = logger.GetOrCreate("process/smartcontract")

type scProcessor struct {
	accounts         state.AccountsAdapter
	tempAccounts     process.TemporaryAccountsHandler
	adrConv          state.AddressConverter
	hasher           hashing.Hasher
	marshalizer      marshal.Marshalizer
	shardCoordinator sharding.Coordinator
	vmContainer      process.VirtualMachinesContainer
	argsParser       process.ArgumentsParser
	builtInFunctions map[string]process.BuiltinFunction

	scrForwarder  process.IntermediateTransactionHandler
	txFeeHandler  process.TransactionFeeHandler
	economicsFee  process.FeeHandler
	txTypeHandler process.TxTypeHandler
	gasHandler    process.GasHandler
	gasCost       GasCost
}

// ArgsNewSmartContractProcessor defines the arguments needed for new smart contract processor
type ArgsNewSmartContractProcessor struct {
	VmContainer   process.VirtualMachinesContainer
	ArgsParser    process.ArgumentsParser
	Hasher        hashing.Hasher
	Marshalizer   marshal.Marshalizer
	AccountsDB    state.AccountsAdapter
	TempAccounts  process.TemporaryAccountsHandler
	AdrConv       state.AddressConverter
	Coordinator   sharding.Coordinator
	ScrForwarder  process.IntermediateTransactionHandler
	TxFeeHandler  process.TransactionFeeHandler
	EconomicsFee  process.FeeHandler
	TxTypeHandler process.TxTypeHandler
	GasHandler    process.GasHandler
	GasMap        map[string]map[string]uint64
}

// NewSmartContractProcessor create a smart contract processor creates and interprets VM data
func NewSmartContractProcessor(args ArgsNewSmartContractProcessor) (*scProcessor, error) {

	if check.IfNil(args.VmContainer) {
		return nil, process.ErrNoVM
	}
	if check.IfNil(args.ArgsParser) {
		return nil, process.ErrNilArgumentParser
	}
	if check.IfNil(args.Hasher) {
		return nil, process.ErrNilHasher
	}
	if check.IfNil(args.Marshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(args.AccountsDB) {
		return nil, process.ErrNilAccountsAdapter
	}
	if check.IfNil(args.TempAccounts) {
		return nil, process.ErrNilTemporaryAccountsHandler
	}
	if check.IfNil(args.AdrConv) {
		return nil, process.ErrNilAddressConverter
	}
	if check.IfNil(args.Coordinator) {
		return nil, process.ErrNilShardCoordinator
	}
	if check.IfNil(args.ScrForwarder) {
		return nil, process.ErrNilIntermediateTransactionHandler
	}
	if check.IfNil(args.TxFeeHandler) {
		return nil, process.ErrNilUnsignedTxHandler
	}
	if check.IfNil(args.EconomicsFee) {
		return nil, process.ErrNilEconomicsFeeHandler
	}
	if check.IfNil(args.TxTypeHandler) {
		return nil, process.ErrNilTxTypeHandler
	}
	if check.IfNil(args.GasHandler) {
		return nil, process.ErrNilGasHandler
	}

	sc := &scProcessor{
		vmContainer:      args.VmContainer,
		argsParser:       args.ArgsParser,
		hasher:           args.Hasher,
		marshalizer:      args.Marshalizer,
		accounts:         args.AccountsDB,
		tempAccounts:     args.TempAccounts,
		adrConv:          args.AdrConv,
		shardCoordinator: args.Coordinator,
		scrForwarder:     args.ScrForwarder,
		txFeeHandler:     args.TxFeeHandler,
		economicsFee:     args.EconomicsFee,
		txTypeHandler:    args.TxTypeHandler,
		gasHandler:       args.GasHandler,
	}

	err := sc.createGasConfig(args.GasMap)
	if err != nil {
		return nil, err
	}

	err = sc.createBuiltInFunctions()
	if err != nil {
		return nil, err
	}

	return sc, nil
}

func (sc *scProcessor) createGasConfig(gasMap map[string]map[string]uint64) error {
	baseOps := &BaseOperationCost{}
	err := mapstructure.Decode(gasMap[core.BaseOperationCost], baseOps)
	if err != nil {
		return err
	}

	err = check.ForZeroUintFields(*baseOps)
	if err != nil {
		return err
	}

	builtInOps := &BuiltInCost{}
	err = mapstructure.Decode(gasMap[core.BuiltInCost], builtInOps)
	if err != nil {
		return err
	}

	err = check.ForZeroUintFields(*builtInOps)
	if err != nil {
		return err
	}

	sc.gasCost = GasCost{
		BaseOperationCost: *baseOps,
		BuiltInCost:       *builtInOps,
	}

	return nil
}

func (sc *scProcessor) createBuiltInFunctions() error {
	sc.builtInFunctions = make(map[string]process.BuiltinFunction)

	sc.builtInFunctions[claimDeveloperRewardsFunctionName] = &claimDeveloperRewards{gasCost: sc.gasCost.BuiltInCost.ClaimDeveloperRewards}
	sc.builtInFunctions[changeOwnerAddressFunctionName] = &changeOwnerAddress{gasCost: sc.gasCost.BuiltInCost.ClaimDeveloperRewards}

	return nil
}

func (sc *scProcessor) checkTxValidity(tx data.TransactionHandler) error {
	if check.IfNil(tx) {
		return process.ErrNilTransaction
	}

	recvAddressIsInvalid := sc.adrConv.AddressLen() != len(tx.GetRcvAddr())
	if recvAddressIsInvalid {
		return process.ErrWrongTransaction
	}

	return nil
}

func (sc *scProcessor) isDestAddressEmpty(tx data.TransactionHandler) bool {
	isEmptyAddress := bytes.Equal(tx.GetRcvAddr(), make([]byte, sc.adrConv.AddressLen()))
	return isEmptyAddress
}

// ExecuteSmartContractTransaction processes the transaction, call the VM and processes the SC call output
func (sc *scProcessor) ExecuteSmartContractTransaction(
	tx data.TransactionHandler,
	acntSnd, acntDst state.UserAccountHandler,
) error {
	defer sc.tempAccounts.CleanTempAccounts()

	if check.IfNil(tx) {
		return process.ErrNilTransaction
	}
	if check.IfNil(acntDst) {
		return process.ErrNilSCDestAccount
	}

	err := sc.processSCPayment(tx, acntSnd)
	if err != nil {
		log.Debug("process sc payment error", "error", err.Error())
		return err
	}

	var txHash []byte
	txHash, err = core.CalculateHash(sc.marshalizer, sc.hasher, tx)
	if err != nil {
		log.Debug("CalculateHash error", "error", err)
		return err
	}

	defer func() {
		if err != nil {
			errNotCritical := sc.ProcessIfError(acntSnd, txHash, tx, err.Error())
			if errNotCritical != nil {
				log.Debug("error while processing error in smart contract processor")
			}
		}
	}()

	err = sc.prepareSmartContractCall(tx, acntSnd)
	if err != nil {
		log.Debug("prepare smart contract call error", "error", err.Error())
		return nil
	}

	var vmInput *vmcommon.ContractCallInput
	vmInput, err = sc.createVMCallInput(tx)
	if err != nil {
		log.Debug("create vm call input error", "error", err.Error())
		return nil
	}

	var executed bool
	executed, err = sc.resolveBuiltInFunctions(txHash, tx, acntSnd, acntDst, vmInput)
	if err != nil {
		log.Debug("processed built in functions error", "error", err.Error())
		return nil
	}
	if executed {
		return nil
	}

	var vm vmcommon.VMExecutionHandler
	vm, err = findVMByTransaction(sc.vmContainer, tx)
	if err != nil {
		log.Debug("get vm from address error", "error", err.Error())
		return nil
	}

	var vmOutput *vmcommon.VMOutput
	vmOutput, err = vm.RunSmartContractCall(vmInput)
	if err != nil {
		log.Debug("run smart contract call error", "error", err.Error())
		return nil
	}

	err = sc.saveAccounts(acntSnd, acntDst)
	if err != nil {
		return err
	}

	var consumedFee *big.Int
	var results []data.TransactionHandler
	results, consumedFee, err = sc.processVMOutput(vmOutput, txHash, tx, acntSnd, vmInput.CallType)
	if err != nil {
		log.Trace("process vm output error", "error", err.Error())
		return nil
	}

	err = sc.scrForwarder.AddIntermediateTransactions(results)
	if err != nil {
		log.Debug("AddIntermediateTransactions error", "error", err.Error())
		return nil
	}

	newDeveloperReward := core.GetPercentageOfValue(consumedFee, sc.economicsFee.DeveloperPercentage())
	feeForValidators := big.NewInt(0).Sub(consumedFee, newDeveloperReward)

	acntDst, err = sc.reloadLocalAccount(acntDst)
	if err != nil {
		log.Debug("reloadLocalAccount error", "error", err.Error())
		return nil
	}

	acntDst.AddToDeveloperReward(newDeveloperReward)
	sc.txFeeHandler.ProcessTransactionFee(feeForValidators, txHash)

	err = sc.accounts.SaveAccount(acntDst)
	if err != nil {
		log.Debug("error saving account")
	}

	return nil
}

func (sc *scProcessor) saveAccounts(acntSnd, acntDst state.AccountHandler) error {
	if !check.IfNil(acntSnd) {
		err := sc.accounts.SaveAccount(acntSnd)
		if err != nil {
			return err
		}
	}

	if !check.IfNil(acntDst) {
		err := sc.accounts.SaveAccount(acntDst)
		if err != nil {
			return err
		}
	}

	return nil
}

func (sc *scProcessor) resolveBuiltInFunctions(
	txHash []byte,
	tx data.TransactionHandler,
	acntSnd, acntDst state.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (bool, error) {

	builtIn, ok := sc.builtInFunctions[vmInput.Function]
	if !ok {
		return false, nil
	}
	if check.IfNil(builtIn) {
		return true, process.ErrNilBuiltInFunction
	}

	valueToSend, err := builtIn.ProcessBuiltinFunction(tx, acntSnd, acntDst, vmInput)
	if err != nil {
		return true, err
	}

	gasConsumed := builtIn.GasUsed()
	if tx.GetGasLimit() < gasConsumed {
		return true, process.ErrNotEnoughGas
	}

	gasRemaining := tx.GetGasLimit() - gasConsumed
	scrRefund, consumedFee, err := sc.createSCRForSender(
		big.NewInt(0),
		gasRemaining,
		vmcommon.Ok,
		make([][]byte, 0),
		tx,
		txHash,
		acntSnd,
		vmcommon.DirectCall,
	)
	if err != nil {
		return true, err
	}

	scrRefund.Value.Add(scrRefund.Value, valueToSend)
	err = sc.scrForwarder.AddIntermediateTransactions([]data.TransactionHandler{scrRefund})
	if err != nil {
		log.Debug("AddIntermediateTransactions error", "error", err.Error())
		return true, err
	}

	sc.gasHandler.SetGasRefunded(gasRemaining, txHash)
	sc.txFeeHandler.ProcessTransactionFee(consumedFee, txHash)

	return true, sc.saveAccounts(acntSnd, acntDst)
}

// ProcessIfError creates a smart contract result, consumed the gas and returns the value to the user
func (sc *scProcessor) ProcessIfError(
	acntSnd state.UserAccountHandler,
	txHash []byte,
	tx data.TransactionHandler,
	returnCode string,
) error {
	consumedFee := big.NewInt(0).Mul(big.NewInt(0).SetUint64(tx.GetGasLimit()), big.NewInt(0).SetUint64(tx.GetGasPrice()))
	scrIfError, err := sc.createSCRsWhenError(txHash, tx, returnCode)
	if err != nil {
		return err
	}

	if !check.IfNil(acntSnd) {
		err = acntSnd.AddToBalance(tx.GetValue())
		if err != nil {
			return err
		}

		err = sc.accounts.SaveAccount(acntSnd)
		if err != nil {
			log.Debug("error saving account")
		}
	} else {
		moveBalanceCost := sc.economicsFee.ComputeFee(tx)
		consumedFee.Sub(consumedFee, moveBalanceCost)
	}

	err = sc.scrForwarder.AddIntermediateTransactions(scrIfError)
	if err != nil {
		return err
	}

	sc.txFeeHandler.ProcessTransactionFee(consumedFee, txHash)

	return nil
}

func (sc *scProcessor) prepareSmartContractCall(tx data.TransactionHandler, acntSnd state.UserAccountHandler) error {
	nonce := tx.GetNonce()
	if acntSnd != nil && !acntSnd.IsInterfaceNil() {
		nonce = acntSnd.GetNonce()
	}

	txValue := big.NewInt(0).Set(tx.GetValue())
	sc.tempAccounts.AddTempAccount(tx.GetSndAddr(), txValue, nonce)

	return nil
}

// DeploySmartContract processes the transaction, than deploy the smart contract into VM, final code is saved in account
func (sc *scProcessor) DeploySmartContract(
	tx data.TransactionHandler,
	acntSnd state.UserAccountHandler,
) error {
	defer sc.tempAccounts.CleanTempAccounts()

	err := sc.checkTxValidity(tx)
	if err != nil {
		log.Debug("Transaction invalid", "error", err.Error())
		return err
	}

	isEmptyAddress := sc.isDestAddressEmpty(tx)
	if !isEmptyAddress {
		log.Debug("Transaction wrong", "error", process.ErrWrongTransaction.Error())
		return process.ErrWrongTransaction
	}

	txHash, err := core.CalculateHash(sc.marshalizer, sc.hasher, tx)
	if err != nil {
		log.Debug("CalculateHash error", "error", err)
		return err
	}

	err = sc.processSCPayment(tx, acntSnd)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			errNotCritical := sc.ProcessIfError(acntSnd, txHash, tx, err.Error())
			if errNotCritical != nil {
				log.Debug("error while processing error in smart contract processor")
			}
		}
	}()

	err = sc.prepareSmartContractCall(tx, acntSnd)
	if err != nil {
		log.Debug("Transaction error", "error", err.Error())
		return nil
	}

	vmInput, vmType, _, err := sc.createVMDeployInput(tx)
	if err != nil {
		log.Debug("Transaction error", "error", err.Error())
		return nil
	}

	vm, err := sc.vmContainer.Get(vmType)
	if err != nil {
		log.Debug("VM error", "error", err.Error())
		return nil
	}

	vmOutput, err := vm.RunSmartContractCreate(vmInput)
	if err != nil {
		log.Debug("VM error", "error", err.Error())
		return nil
	}

	err = sc.accounts.SaveAccount(acntSnd)
	if err != nil {
		log.Debug("Save account error", "error", err.Error())
		return nil
	}

	results, consumedFee, err := sc.processVMOutput(vmOutput, txHash, tx, acntSnd, vmInput.CallType)
	if err != nil {
		log.Trace("Processing error", "error", err.Error())
		return nil
	}

	err = sc.scrForwarder.AddIntermediateTransactions(results)
	if err != nil {
		log.Debug("AddIntermediate Transaction error", "error", err.Error())
		return nil
	}

	sc.txFeeHandler.ProcessTransactionFee(consumedFee, txHash)

	log.Debug("SmartContract deployed")
	return nil
}

// taking money from sender, as VM might not have access to him because of state sharding
func (sc *scProcessor) processSCPayment(tx data.TransactionHandler, acntSnd state.UserAccountHandler) error {
	if check.IfNil(acntSnd) {
		// transaction was already processed at sender shard
		return nil
	}

	acntSnd.IncreaseNonce(1)
	err := sc.economicsFee.CheckValidityTxValues(tx)
	if err != nil {
		return err
	}

	cost := big.NewInt(0)
	cost = cost.Mul(big.NewInt(0).SetUint64(tx.GetGasPrice()), big.NewInt(0).SetUint64(tx.GetGasLimit()))
	cost = cost.Add(cost, tx.GetValue())

	if cost.Cmp(big.NewInt(0)) == 0 {
		return nil
	}

	err = acntSnd.SubFromBalance(cost)
	if err != nil {
		return err
	}

	return nil
}

func (sc *scProcessor) processVMOutput(
	vmOutput *vmcommon.VMOutput,
	txHash []byte,
	tx data.TransactionHandler,
	acntSnd state.UserAccountHandler,
	callType vmcommon.CallType,
) ([]data.TransactionHandler, *big.Int, error) {
	if vmOutput == nil {
		return nil, nil, process.ErrNilVMOutput
	}
	if check.IfNil(tx) {
		return nil, nil, process.ErrNilTransaction
	}

	if vmOutput.ReturnCode != vmcommon.Ok {
		log.Trace("smart contract processing returned with error",
			"hash", txHash,
			"return code", vmOutput.ReturnCode.String(),
			"return message", vmOutput.ReturnMessage,
		)

		return nil, nil, fmt.Errorf(vmOutput.ReturnCode.String())
	}

	outPutAccounts := sortVMOutputInsideData(vmOutput)

	scrTxs, err := sc.processSCOutputAccounts(outPutAccounts, tx, txHash)
	if err != nil {
		return nil, nil, err
	}

	acntSnd, err = sc.reloadLocalAccount(acntSnd)
	if err != nil {
		return nil, nil, err
	}

	totalGasConsumed := tx.GetGasLimit() - vmOutput.GasRemaining
	log.Trace("total gas consumed", "value", totalGasConsumed, "hash", txHash)

	if vmOutput.GasRefund.Cmp(big.NewInt(0)) > 0 {
		log.Trace("total gas refunded", "value", vmOutput.GasRefund.String(), "hash", txHash)
	}

	scrRefund, consumedFee, err := sc.createSCRForSender(
		vmOutput.GasRefund,
		vmOutput.GasRemaining,
		vmOutput.ReturnCode,
		vmOutput.ReturnData,
		tx,
		txHash,
		acntSnd,
		callType,
	)
	if err != nil {
		return nil, nil, err
	}

	scrTxs = append(scrTxs, scrRefund)

	if !check.IfNil(acntSnd) {
		err = sc.accounts.SaveAccount(acntSnd)
		if err != nil {
			return nil, nil, err
		}
	}

	err = sc.deleteAccounts(vmOutput.DeletedAccounts)
	if err != nil {
		return nil, nil, err
	}

	err = sc.processTouchedAccounts(vmOutput.TouchedAccounts)
	if err != nil {
		return nil, nil, err
	}

	sc.gasHandler.SetGasRefunded(vmOutput.GasRemaining, txHash)

	return scrTxs, consumedFee, nil
}

func sortVMOutputInsideData(vmOutput *vmcommon.VMOutput) []*vmcommon.OutputAccount {
	sort.Slice(vmOutput.DeletedAccounts, func(i, j int) bool {
		return bytes.Compare(vmOutput.DeletedAccounts[i], vmOutput.DeletedAccounts[j]) < 0
	})
	sort.Slice(vmOutput.TouchedAccounts, func(i, j int) bool {
		return bytes.Compare(vmOutput.TouchedAccounts[i], vmOutput.TouchedAccounts[j]) < 0
	})

	outPutAccounts := make([]*vmcommon.OutputAccount, len(vmOutput.OutputAccounts))
	i := 0
	for _, outAcc := range vmOutput.OutputAccounts {
		outPutAccounts[i] = outAcc
		i++
	}

	sort.Slice(outPutAccounts, func(i, j int) bool {
		return bytes.Compare(outPutAccounts[i].Address, outPutAccounts[j].Address) < 0
	})

	return outPutAccounts
}

func getSortedStorageUpdates(account *vmcommon.OutputAccount) []*vmcommon.StorageUpdate {
	storageUpdates := make([]*vmcommon.StorageUpdate, len(account.StorageUpdates))
	i := 0
	for _, update := range account.StorageUpdates {
		storageUpdates[i] = update
		i++
	}

	sort.Slice(storageUpdates, func(i, j int) bool {
		return bytes.Compare(storageUpdates[i].Offset, storageUpdates[j].Offset) < 0
	})

	return storageUpdates
}

func (sc *scProcessor) createSCRsWhenError(
	txHash []byte,
	tx data.TransactionHandler,
	returnCode string,
) ([]data.TransactionHandler, error) {
	rcvAddress := tx.GetSndAddr()

	callType := determineCallType(tx)
	if callType == vmcommon.AsynchronousCallBack {
		rcvAddress = tx.GetRcvAddr()
	}

	scr := &smartContractResult.SmartContractResult{
		Nonce:   tx.GetNonce(),
		Value:   tx.GetValue(),
		RcvAddr: rcvAddress,
		SndAddr: tx.GetRcvAddr(),
		Code:    nil,
		Data:    []byte("@" + hex.EncodeToString([]byte(returnCode)) + "@" + hex.EncodeToString(txHash)),
		TxHash:  txHash,
	}

	resultedScrs := []data.TransactionHandler{scr}

	return resultedScrs, nil
}

// reloadLocalAccount will reload from current account state the sender account
// this requirement is needed because in the case of refunding the exact account that was previously
// modified in saveSCOutputToCurrentState, the modifications done there should be visible here
func (sc *scProcessor) reloadLocalAccount(acntSnd state.UserAccountHandler) (state.UserAccountHandler, error) {
	if check.IfNil(acntSnd) {
		return acntSnd, nil
	}

	isAccountFromCurrentShard := acntSnd.AddressContainer() != nil
	if !isAccountFromCurrentShard {
		return acntSnd, nil
	}

	return sc.getAccountFromAddress(acntSnd.AddressContainer().Bytes())
}

func (sc *scProcessor) createSmartContractResult(
	outAcc *vmcommon.OutputAccount,
	tx data.TransactionHandler,
	txHash []byte,
	storageUpdates []*vmcommon.StorageUpdate,
) *smartContractResult.SmartContractResult {
	result := &smartContractResult.SmartContractResult{}

	result.Value = outAcc.BalanceDelta
	result.Nonce = outAcc.Nonce
	result.RcvAddr = outAcc.Address
	result.SndAddr = tx.GetRcvAddr()
	result.Code = outAcc.Code
	result.Data = append(outAcc.Data, sc.argsParser.CreateDataFromStorageUpdate(storageUpdates)...)
	result.GasLimit = outAcc.GasLimit
	result.GasPrice = tx.GetGasPrice()
	result.TxHash = txHash

	return result
}

// createSCRForSender(vmOutput, tx, txHash, acntSnd)
// give back the user the unused gas money
func (sc *scProcessor) createSCRForSender(
	gasRefund *big.Int,
	gasRemaining uint64,
	returnCode vmcommon.ReturnCode,
	returnData [][]byte,
	tx data.TransactionHandler,
	txHash []byte,
	acntSnd state.UserAccountHandler,
	callType vmcommon.CallType,
) (*smartContractResult.SmartContractResult, *big.Int, error) {
	storageFreeRefund := big.NewInt(0).Mul(gasRefund, big.NewInt(0).SetUint64(sc.economicsFee.MinGasPrice()))

	consumedFee := big.NewInt(0)
	consumedFee.Mul(big.NewInt(0).SetUint64(tx.GetGasPrice()), big.NewInt(0).SetUint64(tx.GetGasLimit()))

	refundErd := big.NewInt(0)
	refundErd.Mul(big.NewInt(0).SetUint64(gasRemaining), big.NewInt(0).SetUint64(tx.GetGasPrice()))
	consumedFee.Sub(consumedFee, refundErd)

	rcvAddress := tx.GetSndAddr()
	if callType == vmcommon.AsynchronousCallBack {
		rcvAddress = tx.GetRcvAddr()
	}

	scTx := &smartContractResult.SmartContractResult{}
	scTx.Value = big.NewInt(0).Add(refundErd, storageFreeRefund)
	scTx.RcvAddr = rcvAddress
	scTx.SndAddr = tx.GetRcvAddr()
	scTx.Nonce = tx.GetNonce() + 1
	scTx.TxHash = txHash
	scTx.GasLimit = gasRemaining
	scTx.GasPrice = tx.GetGasPrice()

	scTx.Data = []byte("@" + hex.EncodeToString([]byte(returnCode.String())))
	for _, retData := range returnData {
		scTx.Data = append(scTx.Data, []byte("@"+hex.EncodeToString(retData))...)
	}

	if callType == vmcommon.AsynchronousCall {
		scTx.CallType = vmcommon.AsynchronousCallBack
	}

	if check.IfNil(acntSnd) {
		// cross shard move balance fee was already consumed at sender shard
		moveBalanceCost := sc.economicsFee.ComputeFee(tx)
		consumedFee.Sub(consumedFee, moveBalanceCost)
		return scTx, consumedFee, nil
	}

	err := acntSnd.AddToBalance(scTx.Value)
	if err != nil {
		return nil, nil, err
	}

	return scTx, consumedFee, nil
}

// save account changes in state from vmOutput - protected by VM - every output can be treated as is.
func (sc *scProcessor) processSCOutputAccounts(
	outputAccounts []*vmcommon.OutputAccount,
	tx data.TransactionHandler,
	txHash []byte,
) ([]data.TransactionHandler, error) {
	scResults := make([]data.TransactionHandler, 0, len(outputAccounts))

	sumOfAllDiff := big.NewInt(0)
	sumOfAllDiff.Sub(sumOfAllDiff, tx.GetValue())

	zero := big.NewInt(0)
	for i := 0; i < len(outputAccounts); i++ {
		outAcc := outputAccounts[i]
		acc, err := sc.getAccountFromAddress(outAcc.Address)
		if err != nil {
			return nil, err
		}

		storageUpdates := getSortedStorageUpdates(outAcc)
		scTx := sc.createSmartContractResult(outAcc, tx, txHash, storageUpdates)
		scResults = append(scResults, scTx)

		if check.IfNil(acc) {
			if outAcc.BalanceDelta != nil {
				sumOfAllDiff.Add(sumOfAllDiff, outAcc.BalanceDelta)
			}
			continue
		}

		for j := 0; j < len(storageUpdates); j++ {
			storeUpdate := storageUpdates[j]
			acc.DataTrieTracker().SaveKeyValue(storeUpdate.Offset, storeUpdate.Data)

			log.Trace("storeUpdate", "acc", outAcc.Address, "key", storeUpdate.Offset, "data", storeUpdate.Data)
		}

		err = sc.updateSmartContractCode(acc, outAcc, tx)
		if err != nil {
			return nil, err
		}

		// change nonce only if there is a change
		if outAcc.Nonce != acc.GetNonce() && outAcc.Nonce != 0 {
			if outAcc.Nonce < acc.GetNonce() {
				return nil, process.ErrWrongNonceInVMOutput
			}

			nonceDifference := outAcc.Nonce - acc.GetNonce()
			acc.IncreaseNonce(nonceDifference)
		}

		// if no change then continue
		if outAcc.BalanceDelta == nil || outAcc.BalanceDelta.Cmp(zero) == 0 {
			err = sc.accounts.SaveAccount(acc)
			if err != nil {
				return nil, err
			}

			continue
		}

		sumOfAllDiff = sumOfAllDiff.Add(sumOfAllDiff, outAcc.BalanceDelta)

		err = acc.AddToBalance(outAcc.BalanceDelta)
		if err != nil {
			return nil, err
		}

		err = sc.accounts.SaveAccount(acc)
		if err != nil {
			return nil, err
		}
	}

	if sumOfAllDiff.Cmp(zero) != 0 {
		return nil, process.ErrOverallBalanceChangeFromSC
	}

	return scResults, nil
}

func (sc *scProcessor) updateSmartContractCode(
	account state.UserAccountHandler,
	outAcc *vmcommon.OutputAccount,
	tx data.TransactionHandler,
) error {
	if len(outAcc.Code) == 0 {
		return nil
	}

	isDeployment := len(account.GetCode()) == 0
	if isDeployment {
		account.SetOwnerAddress(tx.GetSndAddr())
		account.SetCode(outAcc.Code)

		log.Trace("created SC address", "address", hex.EncodeToString(outAcc.Address))
		return nil
	}

	// TODO: implement upgradable flag for smart contracts
	isUpgradeEnabled := bytes.Equal(account.GetOwnerAddress(), tx.GetSndAddr())
	if isUpgradeEnabled {
		account.SetCode(outAcc.Code)

		log.Trace("created SC address", "address", hex.EncodeToString(outAcc.Address))
		return nil
	}

	// TODO: change to return some error when IELE is updated. Currently IELE sends the code in output account even for normal SC RUN
	return nil
}

// delete accounts - only suicide by current SC or another SC called by current SC - protected by VM
func (sc *scProcessor) deleteAccounts(deletedAccounts [][]byte) error {
	for _, value := range deletedAccounts {
		acc, err := sc.getAccountFromAddress(value)
		if err != nil {
			return err
		}

		if acc == nil || acc.IsInterfaceNil() {
			//TODO: sharded Smart Contract processing
			continue
		}

		err = sc.accounts.RemoveAccount(acc.AddressContainer())
		if err != nil {
			return err
		}
	}
	return nil
}

func (sc *scProcessor) processTouchedAccounts(_ [][]byte) error {
	//TODO: implement
	return nil
}

func (sc *scProcessor) getAccountFromAddress(address []byte) (state.UserAccountHandler, error) {
	adrSrc, err := sc.adrConv.CreateAddressFromPublicKeyBytes(address)
	if err != nil {
		return nil, err
	}

	shardForCurrentNode := sc.shardCoordinator.SelfId()
	shardForSrc := sc.shardCoordinator.ComputeId(adrSrc)
	if shardForCurrentNode != shardForSrc {
		return nil, nil
	}

	acnt, err := sc.accounts.LoadAccount(adrSrc)
	if err != nil {
		return nil, err
	}

	stAcc, ok := acnt.(state.UserAccountHandler)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	return stAcc, nil
}

// ProcessSmartContractResult updates the account state from the smart contract result
func (sc *scProcessor) ProcessSmartContractResult(scr *smartContractResult.SmartContractResult) error {
	if scr == nil {
		return process.ErrNilSmartContractResult
	}

	var err error
	txHash, err := core.CalculateHash(sc.marshalizer, sc.hasher, scr)
	if err != nil {
		log.Debug("CalculateHash error", "error", err)
		return err
	}

	defer func() {
		if err != nil {
			errNotCritical := sc.ProcessIfError(nil, txHash, scr, err.Error())
			if errNotCritical != nil {
				log.Debug("error while processing error in smart contract processor")
			}
		}
	}()

	dstAcc, err := sc.getAccountFromAddress(scr.RcvAddr)
	if err != nil {
		return nil
	}
	if check.IfNil(dstAcc) {
		err = process.ErrNilSCDestAccount
		return nil
	}

	process.DisplayProcessTxDetails("ProcessSmartContractResult: receiver account details", dstAcc, scr)

	txType, err := sc.txTypeHandler.ComputeTransactionType(scr)
	if err != nil {
		return nil
	}

	switch txType {
	case process.MoveBalance:
		err = sc.processSimpleSCR(scr, dstAcc)
		return nil
	case process.SCDeployment:
		err = process.ErrSCDeployFromSCRIsNotPermitted
		return nil
	case process.SCInvoking:
		err = sc.ExecuteSmartContractTransaction(scr, nil, dstAcc)
		return nil
	}

	err = process.ErrWrongTransaction
	return nil
}

func (sc *scProcessor) processSimpleSCR(
	scr *smartContractResult.SmartContractResult,
	dstAcc state.UserAccountHandler,
) error {
	err := sc.updateSmartContractCode(dstAcc, &vmcommon.OutputAccount{Code: scr.Code, Address: scr.RcvAddr}, scr)
	if err != nil {
		return err
	}

	if scr.Value != nil {
		err = dstAcc.AddToBalance(scr.Value)
		if err != nil {
			return err
		}

		err = sc.accounts.SaveAccount(dstAcc)
		if err != nil {
			return err
		}
	}

	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (sc *scProcessor) IsInterfaceNil() bool {
	return sc == nil
}
