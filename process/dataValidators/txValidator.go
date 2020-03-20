package dataValidators

import (
	"encoding/hex"
	"fmt"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// txValidator represents a tx handler validator that doesn't check the validity of provided txHandler
type txValidator struct {
	accounts             state.AccountsAdapter
	shardCoordinator     sharding.Coordinator
	maxNonceDeltaAllowed int
}

// NewTxValidator creates a new nil tx handler validator instance
func NewTxValidator(
	accounts state.AccountsAdapter,
	shardCoordinator sharding.Coordinator,
	maxNonceDeltaAllowed int,
) (*txValidator, error) {

	if check.IfNil(accounts) {
		return nil, process.ErrNilAccountsAdapter
	}
	if check.IfNil(shardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}

	return &txValidator{
		accounts:             accounts,
		shardCoordinator:     shardCoordinator,
		maxNonceDeltaAllowed: maxNonceDeltaAllowed,
	}, nil
}

// CheckTxValidity will filter transactions that needs to be added in pools
func (txv *txValidator) CheckTxValidity(interceptedTx process.TxValidatorHandler) error {
	// TODO: Refactor, extract methods.

	shardID := txv.shardCoordinator.SelfId()
	txShardID := interceptedTx.SenderShardId()
	senderIsInAnotherShard := shardID != txShardID
	if senderIsInAnotherShard {
		return nil
	}

	senderAddress := interceptedTx.SenderAddress()
	accountHandler, err := txv.accounts.GetExistingAccount(senderAddress)
	if err != nil {
		return fmt.Errorf("%w for address %s and shard %d",
			process.ErrAddressNotInThisShard,
			hex.EncodeToString(senderAddress.Bytes()),
			shardID,
		)
	}

	accountNonce := accountHandler.GetNonce()
	txNonce := interceptedTx.Nonce()
	lowerNonceInTx := txNonce < accountNonce
	veryHighNonceInTx := txNonce > accountNonce+uint64(txv.maxNonceDeltaAllowed)
	isTxRejected := lowerNonceInTx || veryHighNonceInTx
	if isTxRejected {
		return fmt.Errorf("%w lowerNonceInTx: %v, veryHighNonceInTx: %v",
			process.ErrWrongTransaction,
			lowerNonceInTx,
			veryHighNonceInTx,
		)
	}

	account, ok := accountHandler.(state.UserAccountHandler)
	if !ok {
		return fmt.Errorf("%w, account is not of type *state.Account, address: %s",
			process.ErrWrongTypeAssertion,
			hex.EncodeToString(senderAddress.Bytes()),
		)
	}

	accountBalance := account.GetBalance()
	txFee := interceptedTx.Fee()
	if accountBalance.Cmp(txFee) < 0 {
		return fmt.Errorf("%w, for address: %s, wanted %v, have %v",
			process.ErrInsufficientFunds,
			hex.EncodeToString(senderAddress.Bytes()),
			txFee,
			accountBalance,
		)
	}

	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (txv *txValidator) IsInterfaceNil() bool {
	return txv == nil
}
