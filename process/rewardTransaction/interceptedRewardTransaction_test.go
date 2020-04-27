package rewardTransaction_test

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/process/rewardTransaction"
	"github.com/stretchr/testify/assert"
)

func createMockPubkeyConverter() *mock.PubkeyConverterMock {
	return mock.NewPubkeyConverterMock(32)
}

func TestNewInterceptedRewardTransaction_NilTxBuffShouldErr(t *testing.T) {
	t.Parallel()

	irt, err := rewardTransaction.NewInterceptedRewardTransaction(
		nil,
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3))

	assert.Nil(t, irt)
	assert.Equal(t, process.ErrNilBuffer, err)
}

func TestNewInterceptedRewardTransaction_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	txBuff := []byte("tx")
	irt, err := rewardTransaction.NewInterceptedRewardTransaction(
		txBuff,
		nil,
		&mock.HasherMock{},
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3))

	assert.Nil(t, irt)
	assert.Equal(t, process.ErrNilMarshalizer, err)
}

func TestNewInterceptedRewardTransaction_NilHasherShouldErr(t *testing.T) {
	t.Parallel()

	txBuff := []byte("tx")
	irt, err := rewardTransaction.NewInterceptedRewardTransaction(
		txBuff,
		&mock.MarshalizerMock{},
		nil,
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3))

	assert.Nil(t, irt)
	assert.Equal(t, process.ErrNilHasher, err)
}

func TestNewInterceptedRewardTransaction_NilPubkeyConverterShouldErr(t *testing.T) {
	t.Parallel()

	txBuff := []byte("tx")
	irt, err := rewardTransaction.NewInterceptedRewardTransaction(
		txBuff,
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		nil,
		mock.NewMultiShardsCoordinatorMock(3))

	assert.Nil(t, irt)
	assert.Equal(t, process.ErrNilPubkeyConverter, err)
}

func TestNewInterceptedRewardTransaction_NilShardCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	txBuff := []byte("tx")
	irt, err := rewardTransaction.NewInterceptedRewardTransaction(
		txBuff,
		&mock.MarshalizerMock{},
		&mock.HasherMock{},
		createMockPubkeyConverter(),
		nil)

	assert.Nil(t, irt)
	assert.Equal(t, process.ErrNilShardCoordinator, err)
}

func TestNewInterceptedRewardTransaction_OkValsShouldWork(t *testing.T) {
	t.Parallel()

	rewTx := rewardTx.RewardTx{
		Round:   0,
		Epoch:   0,
		Value:   big.NewInt(100),
		RcvAddr: []byte("receiver"),
	}

	marshalizer := &mock.MarshalizerMock{}
	txBuff, _ := marshalizer.Marshal(&rewTx)
	irt, err := rewardTransaction.NewInterceptedRewardTransaction(
		txBuff,
		marshalizer,
		&mock.HasherMock{},
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3))

	assert.NotNil(t, irt)
	assert.Nil(t, err)
}

func TestNewInterceptedRewardTransaction_TestGetters(t *testing.T) {
	t.Parallel()

	shardId := uint32(0)
	rewTx := rewardTx.RewardTx{
		Round:   0,
		Epoch:   0,
		Value:   big.NewInt(100),
		RcvAddr: []byte("receiver"),
	}

	marshalizer := &mock.MarshalizerMock{}
	shardCoord := mock.NewMultiShardsCoordinatorMock(3)
	shardCoord.ComputeIdCalled = func(address state.AddressContainer) uint32 {
		return shardId
	}

	txBuff, _ := marshalizer.Marshal(&rewTx)
	irt, err := rewardTransaction.NewInterceptedRewardTransaction(
		txBuff,
		marshalizer,
		&mock.HasherMock{},
		createMockPubkeyConverter(),
		shardCoord)

	assert.NotNil(t, irt)
	assert.Nil(t, err)

	assert.Equal(t, shardId, irt.ReceiverShardId())
	assert.Equal(t, core.MetachainShardId, irt.SenderShardId())
	assert.Equal(t, &rewTx, irt.Transaction())
	assert.True(t, irt.IsForCurrentShard())

	txHash := irt.Hasher().Compute(string(txBuff))
	assert.Equal(t, txHash, irt.Hash())
}

func TestNewInterceptedRewardTransaction_InvalidRcvAddrShouldErr(t *testing.T) {
	t.Parallel()

	rewTx := rewardTx.RewardTx{
		Round:   0,
		Epoch:   0,
		Value:   big.NewInt(100),
		RcvAddr: nil,
	}

	marshalizer := &mock.MarshalizerMock{}
	txBuff, _ := marshalizer.Marshal(&rewTx)
	expectedErr := errors.New("expected error")
	irt, err := rewardTransaction.NewInterceptedRewardTransaction(
		txBuff,
		marshalizer,
		&mock.HasherMock{},
		&mock.PubkeyConverterStub{
			CreateAddressFromBytesCalled: func(pkBytes []byte) (container state.AddressContainer, err error) {
				return nil, expectedErr
			},
		},
		mock.NewMultiShardsCoordinatorMock(3))

	assert.Nil(t, irt)
	assert.Equal(t, process.ErrInvalidRcvAddr, err)
}

func TestNewInterceptedRewardTransaction_NonceShouldBeZero(t *testing.T) {
	t.Parallel()

	rewTx := rewardTx.RewardTx{
		Round:   0,
		Epoch:   0,
		Value:   big.NewInt(100),
		RcvAddr: nil,
	}

	marshalizer := &mock.MarshalizerMock{}
	txBuff, _ := marshalizer.Marshal(&rewTx)
	irt, _ := rewardTransaction.NewInterceptedRewardTransaction(
		txBuff,
		marshalizer,
		&mock.HasherMock{},
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3))

	nonce := irt.Nonce()
	assert.Equal(t, uint64(0), nonce)
}

func TestNewInterceptedRewardTransaction_Fee(t *testing.T) {
	t.Parallel()

	rewTx := rewardTx.RewardTx{
		Round:   0,
		Epoch:   0,
		Value:   big.NewInt(100),
		RcvAddr: nil,
	}

	marshalizer := &mock.MarshalizerMock{}
	txBuff, _ := marshalizer.Marshal(&rewTx)
	irt, _ := rewardTransaction.NewInterceptedRewardTransaction(
		txBuff,
		marshalizer,
		&mock.HasherMock{},
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3))

	assert.Equal(t, big.NewInt(0), irt.Fee())
}

func TestNewInterceptedRewardTransaction_SenderAddress(t *testing.T) {
	t.Parallel()

	value := big.NewInt(100)
	rewTx := rewardTx.RewardTx{
		Round:   0,
		Epoch:   0,
		Value:   value,
		RcvAddr: nil,
	}

	marshalizer := &mock.MarshalizerMock{}
	txBuff, _ := marshalizer.Marshal(&rewTx)
	irt, _ := rewardTransaction.NewInterceptedRewardTransaction(
		txBuff,
		marshalizer,
		&mock.HasherMock{},
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3))

	senderAddr := irt.SenderAddress()
	assert.Nil(t, senderAddr)
}

func TestNewInterceptedRewardTransaction_CheckValidityNilRcvAddrShouldErr(t *testing.T) {
	t.Parallel()

	value := big.NewInt(100)
	rewTx := rewardTx.RewardTx{
		Round:   0,
		Epoch:   0,
		Value:   value,
		RcvAddr: nil,
	}

	marshalizer := &mock.MarshalizerMock{}
	txBuff, _ := marshalizer.Marshal(&rewTx)
	irt, _ := rewardTransaction.NewInterceptedRewardTransaction(
		txBuff,
		marshalizer,
		&mock.HasherMock{},
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3))

	err := irt.CheckValidity()
	assert.Equal(t, process.ErrNilRcvAddr, err)
}

func TestNewInterceptedRewardTransaction_CheckValidityNilValueShouldErr(t *testing.T) {
	t.Parallel()

	rewTx := rewardTx.RewardTx{
		Round:   0,
		Epoch:   0,
		Value:   nil,
		RcvAddr: []byte("addr"),
	}

	marshalizer := &mock.MarshalizerMock{}
	txBuff, _ := marshalizer.Marshal(&rewTx)
	irt, _ := rewardTransaction.NewInterceptedRewardTransaction(
		txBuff,
		marshalizer,
		&mock.HasherMock{},
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3))

	err := irt.CheckValidity()
	assert.Equal(t, process.ErrNilValue, err)
}

func TestNewInterceptedRewardTransaction_CheckValidityNegativeValueShouldErr(t *testing.T) {
	t.Parallel()

	value := big.NewInt(-100)
	rewTx := rewardTx.RewardTx{
		Round:   0,
		Epoch:   0,
		Value:   value,
		RcvAddr: []byte("addr"),
	}

	marshalizer := &mock.MarshalizerMock{}
	txBuff, _ := marshalizer.Marshal(&rewTx)
	irt, _ := rewardTransaction.NewInterceptedRewardTransaction(
		txBuff,
		marshalizer,
		&mock.HasherMock{},
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3))

	err := irt.CheckValidity()
	assert.Equal(t, process.ErrNegativeValue, err)
}

func TestNewInterceptedRewardTransaction_CheckValidityShouldWork(t *testing.T) {
	t.Parallel()

	value := big.NewInt(100)
	rewTx := rewardTx.RewardTx{
		Round:   0,
		Epoch:   0,
		Value:   value,
		RcvAddr: []byte("addr"),
	}

	marshalizer := &mock.MarshalizerMock{}
	txBuff, _ := marshalizer.Marshal(&rewTx)
	irt, _ := rewardTransaction.NewInterceptedRewardTransaction(
		txBuff,
		marshalizer,
		&mock.HasherMock{},
		createMockPubkeyConverter(),
		mock.NewMultiShardsCoordinatorMock(3))

	err := irt.CheckValidity()
	assert.Nil(t, err)
}
