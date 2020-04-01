package coordinator

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/stretchr/testify/assert"
)

func createMockPubkeyConverter() *mock.PubkeyConverterMock {
	return mock.NewPubkeyConverterMock(32)
}

func TestNewTxTypeHandler_NilAddrConv(t *testing.T) {
	t.Parallel()

	tth, err := NewTxTypeHandler(
		nil,
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
	)

	assert.Nil(t, tth)
	assert.Equal(t, process.ErrNilPubkeyConverter, err)
}

func TestNewTxTypeHandler_NilShardCoord(t *testing.T) {
	t.Parallel()

	tth, err := NewTxTypeHandler(
		createMockPubkeyConverter(),
		nil,
		&mock.AccountsStub{},
	)

	assert.Nil(t, tth)
	assert.Equal(t, process.ErrNilShardCoordinator, err)
}

func TestNewTxTypeHandler_NilAccounts(t *testing.T) {
	t.Parallel()

	tth, err := NewTxTypeHandler(
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3),
		nil,
	)

	assert.Nil(t, tth)
	assert.Equal(t, process.ErrNilAccountsAdapter, err)
}

func TestNewTxTypeHandler_ValsOk(t *testing.T) {
	t.Parallel()

	tth, err := NewTxTypeHandler(
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
	)

	assert.NotNil(t, tth)
	assert.Nil(t, err)
	assert.False(t, tth.IsInterfaceNil())
}

func generateRandomByteSlice(size int) []byte {
	buff := make([]byte, size)
	_, _ = rand.Reader.Read(buff)

	return buff
}

func createAccounts(tx *transaction.Transaction) (state.UserAccountHandler, state.UserAccountHandler) {
	acntSrc, _ := state.NewUserAccount(mock.NewAddressMock(tx.SndAddr))
	acntSrc.Balance = acntSrc.Balance.Add(acntSrc.Balance, tx.Value)
	totalFee := big.NewInt(0)
	totalFee = totalFee.Mul(big.NewInt(int64(tx.GasLimit)), big.NewInt(int64(tx.GasPrice)))
	acntSrc.Balance.Set(acntSrc.Balance.Add(acntSrc.Balance, totalFee))

	acntDst, _ := state.NewUserAccount(mock.NewAddressMock(tx.RcvAddr))

	return acntSrc, acntDst
}

func TestTxTypeHandler_ComputeTransactionTypeNil(t *testing.T) {
	t.Parallel()

	tth, err := NewTxTypeHandler(
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
	)

	assert.NotNil(t, tth)
	assert.Nil(t, err)

	_, err = tth.ComputeTransactionType(nil)
	assert.Equal(t, process.ErrNilTransaction, err)
}

func TestTxTypeHandler_ComputeTransactionTypeNilTx(t *testing.T) {
	t.Parallel()

	tth, err := NewTxTypeHandler(
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
	)

	assert.NotNil(t, tth)
	assert.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte("DST")
	tx.Value = big.NewInt(45)

	tx = nil
	_, err = tth.ComputeTransactionType(tx)
	assert.Equal(t, process.ErrNilTransaction, err)
}

func TestTxTypeHandler_ComputeTransactionTypeErrWrongTransaction(t *testing.T) {
	t.Parallel()

	tth, err := NewTxTypeHandler(
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
	)

	assert.NotNil(t, tth)
	assert.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = nil
	tx.Value = big.NewInt(45)

	_, err = tth.ComputeTransactionType(tx)
	assert.Equal(t, process.ErrWrongTransaction, err)
}

func TestTxTypeHandler_ComputeTransactionTypeScDeployment(t *testing.T) {
	t.Parallel()

	tth, err := NewTxTypeHandler(
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
	)

	assert.NotNil(t, tth)
	assert.Nil(t, err)

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = make([]byte, createMockPubkeyConverter().Len())
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)

	txType, err := tth.ComputeTransactionType(tx)
	assert.Nil(t, err)
	assert.Equal(t, process.SCDeployment, txType)
}

func TestTxTypeHandler_ComputeTransactionTypeScInvoking(t *testing.T) {
	t.Parallel()

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255}
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)

	_, acntDst := createAccounts(tx)
	acntDst.SetCode([]byte("code"))

	tth, err := NewTxTypeHandler(
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{
			LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
				return acntDst, nil
			}},
	)

	assert.NotNil(t, tth)
	assert.Nil(t, err)

	txType, err := tth.ComputeTransactionType(tx)
	assert.Nil(t, err)
	assert.Equal(t, process.SCInvoking, txType)
}

func TestTxTypeHandler_ComputeTransactionTypeMoveBalance(t *testing.T) {
	t.Parallel()

	tx := &transaction.Transaction{}
	tx.Nonce = 0
	tx.SndAddr = []byte("SRC")
	tx.RcvAddr = generateRandomByteSlice(createMockPubkeyConverter().Len())
	tx.Data = []byte("data")
	tx.Value = big.NewInt(45)

	_, acntDst := createAccounts(tx)

	tth, err := NewTxTypeHandler(
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{
			LoadAccountCalled: func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
				return acntDst, nil
			}},
	)

	assert.NotNil(t, tth)
	assert.Nil(t, err)

	txType, err := tth.ComputeTransactionType(tx)
	assert.Nil(t, err)
	assert.Equal(t, process.MoveBalance, txType)
}

func TestTxTypeHandler_ComputeTransactionTypeRewardTx(t *testing.T) {
	t.Parallel()

	tth, err := NewTxTypeHandler(
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3),
		&mock.AccountsStub{},
	)

	assert.NotNil(t, tth)
	assert.Nil(t, err)

	tx := &rewardTx.RewardTx{RcvAddr: []byte("leader")}
	txType, err := tth.ComputeTransactionType(tx)
	assert.Equal(t, process.ErrWrongTransaction, err)
	assert.Equal(t, process.InvalidTransaction, txType)

	tx = &rewardTx.RewardTx{RcvAddr: generateRandomByteSlice(createMockPubkeyConverter().Len())}
	txType, err = tth.ComputeTransactionType(tx)
	assert.Nil(t, err)
	assert.Equal(t, process.RewardTx, txType)
}
