package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/process"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/require"
)

func TestClaimDeveloperRewards_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	cdr := claimDeveloperRewards{}

	sender := []byte("sender")
	acc, _ := state.NewUserAccount([]byte("addr12"))

	vmOutput, err := cdr.ProcessBuiltinFunction(nil, acc, nil)
	require.Nil(t, vmOutput)
	require.Equal(t, process.ErrNilVmInput, err)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  sender,
			GasProvided: 100,
		},
	}
	vmOutput, err = cdr.ProcessBuiltinFunction(nil, nil, vmInput)
	require.Nil(t, err)

	vmOutput, err = cdr.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Nil(t, vmOutput)
	require.Equal(t, state.ErrOperationNotPermitted, err)

	acc.OwnerAddress = sender
	value := big.NewInt(100)
	acc.AddToDeveloperReward(value)
	vmOutput, err = cdr.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Nil(t, err)
	require.Equal(t, value, vmOutput.OutputAccounts[string(vmInput.CallerAddr)].BalanceDelta)
}
