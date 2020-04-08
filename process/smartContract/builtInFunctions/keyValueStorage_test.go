package builtInFunctions

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/require"
)

func TestSaveKeyValue_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	coa := saveKeyValueStorage{
		funcGasCost: 1,
		gasConfig: BaseOperationCost{
			StorePerByte:    1,
			ReleasePerByte:  1,
			DataCopyPerByte: 1,
			PersistPerByte:  1,
			CompilePerByte:  1,
		},
	}

	addr := []byte("addr")

	acc, _ := state.NewUserAccount(mock.NewAddressMock(addr))
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{CallerAddr: addr, GasProvided: 50},
	}

	_, _, err := coa.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, process.ErrInvalidArguments, err)

	_, _, err = coa.ProcessBuiltinFunction(nil, acc, nil)
	require.Equal(t, process.ErrNilVmInput, err)

	_, _, err = coa.ProcessBuiltinFunction(nil, nil, vmInput)
	require.Equal(t, process.ErrNilSCDestAccount, err)

	newUserName := []byte("afafafafafafafafafafafafafafafaf")
	vmInput.Arguments = [][]byte{newUserName}
	_, _, err = coa.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Nil(t, err)

	_, _, err = coa.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, process.ErrUserNameChangeIsDisabled, err)
}
