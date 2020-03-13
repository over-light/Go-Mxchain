package node

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/stretchr/testify/assert"
)

func TestNode_GetAccountAccountDoesNotExistsShouldRetEmpty(t *testing.T) {
	t.Parallel()

	accDB, _, _ := integrationTests.CreateAccountsDB(0)

	n, _ := node.NewNode(
		node.WithAccountsAdapter(accDB),
		node.WithAddressConverter(integrationTests.TestAddressConverter),
	)

	recovAccnt, err := n.GetAccount(integrationTests.CreateRandomHexString(64))

	assert.Nil(t, err)
	assert.Equal(t, uint64(0), recovAccnt.GetNonce())
	assert.Equal(t, big.NewInt(0), recovAccnt.GetBalance())
	assert.Nil(t, recovAccnt.GetCodeHash())
	assert.Nil(t, recovAccnt.GetRootHash())
}

func TestNode_GetAccountAccountExistsShouldReturn(t *testing.T) {
	t.Parallel()

	accDB, _, _ := integrationTests.CreateAccountsDB(0)

	addressHex := integrationTests.CreateRandomHexString(64)
	addressBytes, _ := hex.DecodeString(addressHex)
	address, _ := integrationTests.TestAddressConverter.CreateAddressFromPublicKeyBytes(addressBytes)

	nonce := uint64(2233)
	account, _ := accDB.LoadAccount(address)
	_ = account.IncreaseNonce(nonce)
	_ = accDB.SaveAccount(account)
	_, _ = accDB.Commit()

	n, _ := node.NewNode(
		node.WithAccountsAdapter(accDB),
		node.WithAddressConverter(integrationTests.TestAddressConverter),
	)

	recovAccnt, err := n.GetAccount(addressHex)

	assert.Nil(t, err)
	assert.Equal(t, nonce, recovAccnt.GetNonce())
}
