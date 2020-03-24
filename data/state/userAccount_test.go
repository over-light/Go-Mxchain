package state_test

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/mock"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/stretchr/testify/assert"
)

func TestNewUserAccount_NilAddressContainerShouldErr(t *testing.T) {
	t.Parallel()

	acc, err := state.NewUserAccount(nil)

	assert.Nil(t, acc)
	assert.Equal(t, state.ErrNilAddressContainer, err)
}

func TestNewUserAccount_OkParamsShouldWork(t *testing.T) {
	t.Parallel()

	acc, err := state.NewUserAccount(&mock.AddressMock{})

	assert.Nil(t, err)
	assert.NotNil(t, acc)
}

func TestUserAccount_AddToBalanceInsufficientFundsShouldErr(t *testing.T) {
	t.Parallel()

	acc, _ := state.NewUserAccount(&mock.AddressMock{})
	value := big.NewInt(-1)

	err := acc.AddToBalance(value)
	assert.Equal(t, state.ErrInsufficientFunds, err)
}

func TestUserAccount_SubFromBalanceInsufficientFundsShouldErr(t *testing.T) {
	t.Parallel()

	acc, _ := state.NewUserAccount(&mock.AddressMock{})
	value := big.NewInt(1)

	err := acc.SubFromBalance(value)
	assert.Equal(t, state.ErrInsufficientFunds, err)
}

func TestUserAccount_GetBalance(t *testing.T) {
	t.Parallel()

	acc, _ := state.NewUserAccount(&mock.AddressMock{})
	balance := big.NewInt(100)
	subFromBalance := big.NewInt(20)

	_ = acc.AddToBalance(balance)
	assert.Equal(t, balance, acc.GetBalance())
	_ = acc.SubFromBalance(subFromBalance)
	assert.Equal(t, big.NewInt(0).Sub(balance, subFromBalance), acc.GetBalance())
}

func TestUserAccount_AddToDeveloperReward(t *testing.T) {
	t.Parallel()

	acc, _ := state.NewUserAccount(&mock.AddressMock{})
	reward := big.NewInt(10)
	acc.AddToDeveloperReward(reward)

	assert.Equal(t, reward, acc.GetDeveloperReward())
}

func TestUserAccount_ClaimDeveloperRewardsWrongAddressShouldErr(t *testing.T) {
	t.Parallel()

	acc, _ := state.NewUserAccount(&mock.AddressMock{})
	val, err := acc.ClaimDeveloperRewards([]byte("wrong address"))
	assert.Nil(t, val)
	assert.Equal(t, state.ErrOperationNotPermitted, err)
}

func TestUserAccount_ClaimDeveloperRewards(t *testing.T) {
	t.Parallel()

	acc, _ := state.NewUserAccount(&mock.AddressMock{})
	reward := big.NewInt(10)
	acc.AddToDeveloperReward(reward)

	val, err := acc.ClaimDeveloperRewards(acc.OwnerAddress)
	assert.Nil(t, err)
	assert.Equal(t, reward, val)
	assert.Equal(t, big.NewInt(0), acc.GetDeveloperReward())
}

func TestUserAccount_ChangeOwnerAddressWrongAddressShouldErr(t *testing.T) {
	t.Parallel()

	acc, _ := state.NewUserAccount(&mock.AddressMock{})
	err := acc.ChangeOwnerAddress([]byte("wrong address"), []byte{})
	assert.Equal(t, state.ErrOperationNotPermitted, err)
}

func TestUserAccount_ChangeOwnerAddressInvalidAddressShouldErr(t *testing.T) {
	t.Parallel()

	acc, _ := state.NewUserAccount(mock.NewAddressMock())
	err := acc.ChangeOwnerAddress(acc.OwnerAddress, []byte("new address"))
	assert.Equal(t, state.ErrInvalidAddressLength, err)
}

func TestUserAccount_ChangeOwnerAddress(t *testing.T) {
	t.Parallel()

	newAddress := mock.NewAddressMock()
	acc, _ := state.NewUserAccount(mock.NewAddressMock())

	err := acc.ChangeOwnerAddress(acc.OwnerAddress, newAddress.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, newAddress.Bytes(), acc.GetOwnerAddress())
}

func TestUserAccount_SetOwnerAddress(t *testing.T) {
	t.Parallel()

	newAddress := []byte("new address")
	acc, _ := state.NewUserAccount(mock.NewAddressMock())

	acc.SetOwnerAddress(newAddress)
	assert.Equal(t, newAddress, acc.GetOwnerAddress())
}

func TestUserAccount_SetAndGetNonce(t *testing.T) {
	t.Parallel()

	acc, _ := state.NewUserAccount(&mock.AddressMock{})
	nonce := uint64(5)
	acc.IncreaseNonce(nonce)

	assert.Equal(t, nonce, acc.GetNonce())
}

func TestUserAccount_SetAndGetCodeHash(t *testing.T) {
	t.Parallel()

	acc, _ := state.NewUserAccount(&mock.AddressMock{})
	codeHash := []byte("code hash")
	acc.SetCodeHash(codeHash)

	assert.Equal(t, codeHash, acc.GetCodeHash())
}

func TestUserAccount_SetAndGetRootHash(t *testing.T) {
	t.Parallel()

	acc, _ := state.NewUserAccount(&mock.AddressMock{})
	rootHash := []byte("root hash")
	acc.SetRootHash(rootHash)

	assert.Equal(t, rootHash, acc.GetRootHash())
}
