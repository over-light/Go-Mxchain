package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/stretchr/testify/require"
)

func TestClaimDeveloperRewards_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	cdr := claimDeveloperRewards{}

	sender := []byte("sender")
	acc, _ := state.NewUserAccount(mock.NewAddressMock([]byte("addr12")))

	reward, err := cdr.ProcessBuiltinFunction(nil, acc, nil)
	require.Nil(t, reward)
	require.Equal(t, process.ErrNilTransaction, err)

	reward, err = cdr.ProcessBuiltinFunction(nil, nil, nil)
	require.Nil(t, reward)
	require.Equal(t, process.ErrNilSCDestAccount, err)

	reward, err = cdr.ProcessBuiltinFunction(nil, acc, nil)
	require.Nil(t, reward)
	require.Equal(t, state.ErrOperationNotPermitted, err)

	acc.OwnerAddress = sender
	value := big.NewInt(100)
	acc.AddToDeveloperReward(value)
	reward, err = cdr.ProcessBuiltinFunction(nil, acc, nil)
	require.Nil(t, err)
	require.Equal(t, value, reward)

}
