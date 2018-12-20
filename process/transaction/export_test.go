package transaction

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go-sandbox/data/state"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/transaction"
)

func (txProc *txProcessor) GetAddresses(tx *transaction.Transaction) (adrSrc, adrDest state.AddressContainer, err error) {
	return txProc.getAddresses(tx)
}

func (txProc *txProcessor) GetAccounts(adrSrc, adrDest state.AddressContainer) (acntSrc, acntDest state.JournalizedAccountWrapper, err error) {
	return txProc.getAccounts(adrSrc, adrDest)
}

func (txProc *txProcessor) CallSCHandler(tx *transaction.Transaction) error {
	return txProc.callSCHandler(tx)
}

func (txProc *txProcessor) CheckTxValues(acntSrc state.JournalizedAccountWrapper, value *big.Int, nonce uint64) error {
	return txProc.checkTxValues(acntSrc, value, nonce)
}

func (txProc *txProcessor) MoveBalances(acntSrc, acntDest state.JournalizedAccountWrapper, value *big.Int) error {
	return txProc.moveBalances(acntSrc, acntDest, value)
}

func (txProc *txProcessor) IncreaseNonceAcntSrc(acntSrc state.JournalizedAccountWrapper) error {
	return txProc.increaseNonceAcntSrc(acntSrc)
}
