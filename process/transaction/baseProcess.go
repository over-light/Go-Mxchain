package transaction

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/atomic"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type baseTxProcessor struct {
	accounts                       state.AccountsAdapter
	shardCoordinator               sharding.Coordinator
	pubkeyConv                     core.PubkeyConverter
	economicsFee                   process.FeeHandler
	hasher                         hashing.Hasher
	marshalizer                    marshal.Marshalizer
	scProcessor                    process.SmartContractProcessor
	flagRelayedTx                  atomic.Flag
	flagPenalizedTooMuchGas        atomic.Flag
	relayedTxEnableEpoch           uint32
	penalizedTooMuchGasEnableEpoch uint32
}

func (txProc *baseTxProcessor) getAccounts(
	adrSrc, adrDst []byte,
) (state.UserAccountHandler, state.UserAccountHandler, error) {

	var acntSrc, acntDst state.UserAccountHandler

	shardForCurrentNode := txProc.shardCoordinator.SelfId()
	shardForSrc := txProc.shardCoordinator.ComputeId(adrSrc)
	shardForDst := txProc.shardCoordinator.ComputeId(adrDst)

	srcInShard := shardForSrc == shardForCurrentNode
	dstInShard := shardForDst == shardForCurrentNode

	if srcInShard && len(adrSrc) == 0 || dstInShard && len(adrDst) == 0 {
		return nil, nil, process.ErrNilAddressContainer
	}

	if bytes.Equal(adrSrc, adrDst) {
		acntWrp, err := txProc.accounts.LoadAccount(adrSrc)
		if err != nil {
			return nil, nil, err
		}

		account, ok := acntWrp.(state.UserAccountHandler)
		if !ok {
			return nil, nil, process.ErrWrongTypeAssertion
		}

		return account, account, nil
	}

	if srcInShard {
		acntSrcWrp, err := txProc.accounts.LoadAccount(adrSrc)
		if err != nil {
			return nil, nil, err
		}

		account, ok := acntSrcWrp.(state.UserAccountHandler)
		if !ok {
			return nil, nil, process.ErrWrongTypeAssertion
		}

		acntSrc = account
	}

	if dstInShard {
		acntDstWrp, err := txProc.accounts.LoadAccount(adrDst)
		if err != nil {
			return nil, nil, err
		}

		account, ok := acntDstWrp.(state.UserAccountHandler)
		if !ok {
			return nil, nil, process.ErrWrongTypeAssertion
		}

		acntDst = account
	}

	return acntSrc, acntDst, nil
}

func (txProc *baseTxProcessor) getAccountFromAddress(adrSrc []byte) (state.UserAccountHandler, error) {
	shardForCurrentNode := txProc.shardCoordinator.SelfId()
	shardForSrc := txProc.shardCoordinator.ComputeId(adrSrc)
	if shardForCurrentNode != shardForSrc {
		return nil, nil
	}

	acnt, err := txProc.accounts.LoadAccount(adrSrc)
	if err != nil {
		return nil, err
	}

	userAcc, ok := acnt.(state.UserAccountHandler)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	return userAcc, nil
}

func (txProc *baseTxProcessor) checkTxValues(
	tx *transaction.Transaction,
	acntSnd, acntDst state.UserAccountHandler,
) error {
	err := txProc.checkUserNames(tx, acntSnd, acntDst)
	if err != nil {
		return err
	}

	if check.IfNil(acntSnd) {
		return nil
	}

	if acntSnd.GetNonce() < tx.Nonce {
		return process.ErrHigherNonceInTransaction
	}
	if acntSnd.GetNonce() > tx.Nonce {
		return process.ErrLowerNonceInTransaction
	}

	err = txProc.economicsFee.CheckValidityTxValues(tx)
	if err != nil {
		return err
	}

	stAcc, ok := acntSnd.(state.UserAccountHandler)
	if !ok {
		return process.ErrWrongTypeAssertion
	}

	var txFee *big.Int
	if txProc.flagPenalizedTooMuchGas.IsSet() {
		txFee = core.SafeMul(tx.GasLimit, tx.GasPrice)
	} else {
		txFee = txProc.economicsFee.ComputeMoveBalanceFee(tx)
	}

	if stAcc.GetBalance().Cmp(txFee) < 0 {
		return fmt.Errorf("%w, has: %s, wanted: %s",
			process.ErrInsufficientFee,
			stAcc.GetBalance().String(),
			txFee.String(),
		)
	}

	var cost *big.Int
	if txProc.flagPenalizedTooMuchGas.IsSet() {
		cost = big.NewInt(0).Add(txFee, tx.Value)
	} else {
		cost = big.NewInt(0).Add(core.SafeMul(tx.GasLimit, tx.GasPrice), tx.Value)
	}

	if stAcc.GetBalance().Cmp(cost) < 0 {
		return process.ErrInsufficientFunds
	}

	return nil
}

func (txProc *baseTxProcessor) checkUserNames(tx *transaction.Transaction, acntSnd, acntDst state.UserAccountHandler) error {
	isUserNameWrong := len(tx.SndUserName) > 0 &&
		!check.IfNil(acntSnd) && !bytes.Equal(tx.SndUserName, acntSnd.GetUserName())
	if isUserNameWrong {
		return process.ErrUserNameDoesNotMatch
	}

	isUserNameWrong = len(tx.RcvUserName) > 0 &&
		!check.IfNil(acntDst) && !bytes.Equal(tx.RcvUserName, acntDst.GetUserName())
	if isUserNameWrong {
		if check.IfNil(acntSnd) {
			return process.ErrUserNameDoesNotMatchInCrossShardTx
		}
		return process.ErrUserNameDoesNotMatch
	}

	return nil
}

func (txProc *baseTxProcessor) processIfTxErrorCrossShard(tx *transaction.Transaction, errorString string) error {
	txHash, err := core.CalculateHash(txProc.marshalizer, txProc.hasher, tx)
	if err != nil {
		return err
	}

	snapshot := txProc.accounts.JournalLen()
	err = txProc.scProcessor.ProcessIfError(nil, txHash, tx, errorString, nil, snapshot)
	if err != nil {
		return err
	}

	return nil
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (txProc *baseTxProcessor) EpochConfirmed(epoch uint32) {
	txProc.flagRelayedTx.Toggle(epoch >= txProc.relayedTxEnableEpoch)
	log.Debug("txProcessor: relayed transactions", "enabled", txProc.flagRelayedTx.IsSet())

	txProc.flagPenalizedTooMuchGas.Toggle(epoch >= txProc.penalizedTooMuchGasEnableEpoch)
	log.Debug("txProcessor: penalized too much gas", "enabled", txProc.flagPenalizedTooMuchGas.IsSet())
}
