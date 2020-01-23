package scToProtocol

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/vm/factory"
	"github.com/ElrondNetwork/elrond-go/vm/systemSmartContracts"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/assert"
)

func createMockArgumentsNewStakingToPeer() ArgStakingToPeer {
	return ArgStakingToPeer{
		AdrConv:     &mock.AddressConverterMock{},
		Hasher:      &mock.HasherMock{},
		Marshalizer: &mock.MarshalizerStub{},
		PeerState:   &mock.AccountsStub{},
		BaseState:   &mock.AccountsStub{},
		ArgParser:   &mock.ArgumentParserMock{},
		CurrTxs:     &mock.TxForCurrentBlockStub{},
		ScQuery:     &mock.ScQueryMock{},
	}
}

func createBlockBody() block.Body {
	return block.Body{
		{
			TxHashes:        [][]byte{[]byte("hash1"), []byte("hash2")},
			ReceiverShardID: 0,
			SenderShardID:   sharding.MetachainShardId,
			Type:            block.SmartContractResultBlock,
		},
	}
}

func TestNewStakingToPeerNilAddrConverterShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.AdrConv = nil

	stp, err := NewStakingToPeer(arguments)
	assert.Nil(t, stp)
	assert.Equal(t, process.ErrNilAddressConverter, err)
}

func TestNewStakingToPeerNilHasherShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.Hasher = nil

	stp, err := NewStakingToPeer(arguments)
	assert.Nil(t, stp)
	assert.Equal(t, process.ErrNilHasher, err)
}

func TestNewStakingToPeerNilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.Marshalizer = nil

	stp, err := NewStakingToPeer(arguments)
	assert.Nil(t, stp)
	assert.Equal(t, process.ErrNilMarshalizer, err)
}

func TestNewStakingToPeerNilPeerAccountAdapterShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.PeerState = nil

	stp, err := NewStakingToPeer(arguments)
	assert.Nil(t, stp)
	assert.Equal(t, process.ErrNilPeerAccountsAdapter, err)
}

func TestNewStakingToPeerNilBaseAccountAdapterShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.BaseState = nil

	stp, err := NewStakingToPeer(arguments)
	assert.Nil(t, stp)
	assert.Equal(t, process.ErrNilAccountsAdapter, err)
}

func TestNewStakingToPeerNilArgumentParserShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = nil

	stp, err := NewStakingToPeer(arguments)
	assert.Nil(t, stp)
	assert.Equal(t, process.ErrNilArgumentParser, err)
}

func TestNewStakingToPeerNilCurrentBlockHeaderShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.CurrTxs = nil

	stp, err := NewStakingToPeer(arguments)
	assert.Nil(t, stp)
	assert.Equal(t, process.ErrNilTxForCurrentBlockHandler, err)
}

func TestNewStakingToPeerNilScDataGetterShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ScQuery = nil

	stp, err := NewStakingToPeer(arguments)
	assert.Nil(t, stp)
	assert.Equal(t, process.ErrNilSCDataGetter, err)
}

func TestNewStakingToPeer_ShouldWork(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()

	stakingToPeer, err := NewStakingToPeer(arguments)
	assert.NotNil(t, stakingToPeer)
	assert.Nil(t, err)
}

func TestStakingToPeer_UpdateProtocolCannotGetTxShouldErr(t *testing.T) {
	t.Parallel()

	called := false
	testError := errors.New("error")
	currTx := &mock.TxForCurrentBlockStub{}
	currTx.GetTxCalled = func(txHash []byte) (handler data.TransactionHandler, e error) {
		called = true
		return nil, testError
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.CurrTxs = currTx
	stakingToPeer, _ := NewStakingToPeer(arguments)

	blockBody := createBlockBody()
	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Nil(t, err)
	assert.True(t, called)
}

func TestStakingToPeer_UpdateProtocolWrongTransactionTypeShouldErr(t *testing.T) {
	t.Parallel()

	currTx := &mock.TxForCurrentBlockStub{}
	currTx.GetTxCalled = func(txHash []byte) (handler data.TransactionHandler, e error) {
		return &transaction.Transaction{
			RcvAddr: factory.StakingSCAddress,
		}, nil
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.CurrTxs = currTx
	stakingToPeer, _ := NewStakingToPeer(arguments)

	blockBody := createBlockBody()
	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Equal(t, process.ErrWrongTypeAssertion, err)
}

func TestStakingToPeer_UpdateProtocolCannotGetStorageUpdatesShouldErr(t *testing.T) {
	t.Parallel()

	testError := errors.New("error")
	currTx := &mock.TxForCurrentBlockStub{}
	currTx.GetTxCalled = func(txHash []byte) (handler data.TransactionHandler, e error) {
		return &smartContractResult.SmartContractResult{
			RcvAddr: factory.StakingSCAddress,
		}, nil
	}

	argParser := &mock.ArgumentParserMock{}
	argParser.GetStorageUpdatesCalled = func(data string) (updates []*vmcommon.StorageUpdate, e error) {
		return nil, testError
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	stakingToPeer, _ := NewStakingToPeer(arguments)

	blockBody := createBlockBody()
	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Nil(t, err)
}

func TestStakingToPeer_UpdateProtocolWrongAccountShouldErr(t *testing.T) {
	t.Parallel()

	currTx := &mock.TxForCurrentBlockStub{}
	currTx.GetTxCalled = func(txHash []byte) (handler data.TransactionHandler, e error) {
		return &smartContractResult.SmartContractResult{
			RcvAddr: factory.StakingSCAddress,
		}, nil
	}

	argParser := &mock.ArgumentParserMock{}
	argParser.GetStorageUpdatesCalled = func(data string) (updates []*vmcommon.StorageUpdate, e error) {
		return []*vmcommon.StorageUpdate{
			{Offset: []byte("off1"), Data: []byte("data1")},
		}, nil
	}

	peerState := &mock.AccountsStub{}
	peerState.GetAccountWithJournalCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return &mock.AccountWrapMock{}, nil
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	arguments.PeerState = peerState
	stakingToPeer, _ := NewStakingToPeer(arguments)

	blockBody := createBlockBody()
	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Equal(t, process.ErrWrongTypeAssertion, err)
}

func TestStakingToPeer_UpdateProtocolRemoveAccountShouldReturnNil(t *testing.T) {
	t.Parallel()

	currTx := &mock.TxForCurrentBlockStub{}
	currTx.GetTxCalled = func(txHash []byte) (handler data.TransactionHandler, e error) {
		return &smartContractResult.SmartContractResult{
			RcvAddr: factory.StakingSCAddress,
		}, nil
	}

	argParser := &mock.ArgumentParserMock{}
	argParser.GetStorageUpdatesCalled = func(data string) (updates []*vmcommon.StorageUpdate, e error) {
		return []*vmcommon.StorageUpdate{
			{Offset: []byte("off1"), Data: []byte("data1")},
		}, nil
	}

	peerState := &mock.AccountsStub{}
	peerState.GetAccountWithJournalCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		return &state.PeerAccount{
			RewardAddress: []byte("addr"),
			BLSPublicKey:  []byte("BlsAddr"),
			Stake:         big.NewInt(100),
		}, nil
	}
	peerState.RemoveAccountCalled = func(addressContainer state.AddressContainer) error {
		return nil
	}

	marshalizer := &mock.MarshalizerStub{}
	marshalizer.MarshalCalled = func(obj interface{}) (bytes []byte, e error) {
		return []byte("mashalizedData"), nil
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	arguments.PeerState = peerState
	arguments.Marshalizer = marshalizer
	stakingToPeer, _ := NewStakingToPeer(arguments)

	blockBody := createBlockBody()
	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Nil(t, err)
}

func TestStakingToPeer_UpdateProtocolCannotSetRewardAddressShouldErr(t *testing.T) {
	t.Parallel()

	currTx := &mock.TxForCurrentBlockStub{}
	currTx.GetTxCalled = func(txHash []byte) (handler data.TransactionHandler, e error) {
		return &smartContractResult.SmartContractResult{
			RcvAddr: factory.StakingSCAddress,
		}, nil
	}

	argParser := &mock.ArgumentParserMock{}
	argParser.GetStorageUpdatesCalled = func(data string) (updates []*vmcommon.StorageUpdate, e error) {
		return []*vmcommon.StorageUpdate{
			{Offset: []byte("off1"), Data: []byte("data1")},
		}, nil
	}

	peerState := &mock.AccountsStub{}
	peerState.GetAccountWithJournalCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		peerAccount, _ := state.NewPeerAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})
		peerAccount.Stake = big.NewInt(100)
		peerAccount.RewardAddress = []byte("key")
		return peerAccount, nil
	}

	stakingData := systemSmartContracts.StakedData{
		StakeValue: big.NewInt(100),
	}
	marshalizer := &mock.MarshalizerMock{}

	scDataGetter := &mock.ScQueryMock{}
	scDataGetter.ExecuteQueryCalled = func(query *process.SCQuery) (output *vmcommon.VMOutput, e error) {
		retData, _ := json.Marshal(&stakingData)
		return &vmcommon.VMOutput{ReturnData: [][]byte{retData}}, nil
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	arguments.PeerState = peerState
	arguments.Marshalizer = marshalizer
	arguments.ScQuery = scDataGetter
	stakingToPeer, _ := NewStakingToPeer(arguments)

	blockBody := createBlockBody()
	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Equal(t, state.ErrEmptyAddress, err)
}

func TestStakingToPeer_UpdateProtocolCannotSaveAccountShouldErr(t *testing.T) {
	t.Parallel()

	testError := errors.New("error")
	address := "address"
	currTx := &mock.TxForCurrentBlockStub{}
	currTx.GetTxCalled = func(txHash []byte) (handler data.TransactionHandler, e error) {
		return &smartContractResult.SmartContractResult{
			RcvAddr: factory.StakingSCAddress,
		}, nil
	}

	argParser := &mock.ArgumentParserMock{}
	argParser.GetStorageUpdatesCalled = func(data string) (updates []*vmcommon.StorageUpdate, e error) {
		return []*vmcommon.StorageUpdate{
			{Offset: []byte("off1"), Data: []byte("data1")},
		}, nil
	}

	peerState := &mock.AccountsStub{}
	peerState.GetAccountWithJournalCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		peerAccount, _ := state.NewPeerAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{
			JournalizeCalled: func(entry state.JournalEntry) {
				return
			},
			SaveAccountCalled: func(accountHandler state.AccountHandler) error {
				return testError
			},
		})
		peerAccount.Stake = big.NewInt(0)
		peerAccount.RewardAddress = []byte(address)
		return peerAccount, nil
	}

	stakingData := systemSmartContracts.StakedData{
		StakeValue:    big.NewInt(100),
		RewardAddress: []byte(address),
	}
	marshalizer := &mock.MarshalizerMock{}

	scDataGetter := &mock.ScQueryMock{}
	scDataGetter.ExecuteQueryCalled = func(query *process.SCQuery) (output *vmcommon.VMOutput, e error) {
		retData, _ := json.Marshal(&stakingData)
		return &vmcommon.VMOutput{ReturnData: [][]byte{retData}}, nil
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	arguments.PeerState = peerState
	arguments.Marshalizer = marshalizer
	arguments.ScQuery = scDataGetter
	stakingToPeer, _ := NewStakingToPeer(arguments)

	blockBody := createBlockBody()
	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Equal(t, testError, err)
}

func TestStakingToPeer_UpdateProtocolCannotSaveAccountNonceShouldErr(t *testing.T) {
	t.Parallel()

	testError := errors.New("error")
	address := "address"
	currTx := &mock.TxForCurrentBlockStub{}
	currTx.GetTxCalled = func(txHash []byte) (handler data.TransactionHandler, e error) {
		return &smartContractResult.SmartContractResult{
			RcvAddr: factory.StakingSCAddress,
		}, nil
	}

	argParser := &mock.ArgumentParserMock{}
	argParser.GetStorageUpdatesCalled = func(data string) (updates []*vmcommon.StorageUpdate, e error) {
		return []*vmcommon.StorageUpdate{
			{Offset: []byte("off1"), Data: []byte("data1")},
		}, nil
	}

	peerState := &mock.AccountsStub{}
	peerState.GetAccountWithJournalCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		peerAccount, _ := state.NewPeerAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{
			JournalizeCalled: func(entry state.JournalEntry) {
				return
			},
			SaveAccountCalled: func(accountHandler state.AccountHandler) error {
				return testError
			},
		})
		peerAccount.Stake = big.NewInt(100)
		peerAccount.BLSPublicKey = []byte(address)
		peerAccount.Nonce = 1
		return peerAccount, nil
	}

	stakingData := systemSmartContracts.StakedData{
		StakeValue:    big.NewInt(100),
		RewardAddress: []byte(address),
	}
	marshalizer := &mock.MarshalizerMock{}

	scDataGetter := &mock.ScQueryMock{}
	scDataGetter.ExecuteQueryCalled = func(query *process.SCQuery) (output *vmcommon.VMOutput, e error) {
		retData, _ := json.Marshal(&stakingData)
		return &vmcommon.VMOutput{ReturnData: [][]byte{retData}}, nil
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	arguments.PeerState = peerState
	arguments.Marshalizer = marshalizer
	arguments.ScQuery = scDataGetter
	stakingToPeer, _ := NewStakingToPeer(arguments)

	blockBody := createBlockBody()
	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Equal(t, testError, err)
}

func TestStakingToPeer_UpdateProtocol(t *testing.T) {
	t.Parallel()

	address := "address"
	currTx := &mock.TxForCurrentBlockStub{}
	currTx.GetTxCalled = func(txHash []byte) (handler data.TransactionHandler, e error) {
		return &smartContractResult.SmartContractResult{
			RcvAddr: factory.StakingSCAddress,
		}, nil
	}

	argParser := &mock.ArgumentParserMock{}
	argParser.GetStorageUpdatesCalled = func(data string) (updates []*vmcommon.StorageUpdate, e error) {
		return []*vmcommon.StorageUpdate{
			{Offset: []byte("off1"), Data: []byte("data1")},
		}, nil
	}

	peerState := &mock.AccountsStub{}
	peerState.GetAccountWithJournalCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		peerAccount, _ := state.NewPeerAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{
			JournalizeCalled: func(entry state.JournalEntry) {
				return
			},
			SaveAccountCalled: func(accountHandler state.AccountHandler) error {

				return nil
			},
		})
		peerAccount.Stake = big.NewInt(100)
		peerAccount.BLSPublicKey = []byte(address)
		peerAccount.Nonce = 1
		return peerAccount, nil
	}

	stakingData := systemSmartContracts.StakedData{
		StakeValue:    big.NewInt(100),
		RewardAddress: []byte(address),
	}
	marshalizer := &mock.MarshalizerMock{}

	scDataGetter := &mock.ScQueryMock{}
	scDataGetter.ExecuteQueryCalled = func(query *process.SCQuery) (output *vmcommon.VMOutput, e error) {
		retData, _ := json.Marshal(&stakingData)
		return &vmcommon.VMOutput{ReturnData: [][]byte{retData}}, nil
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	arguments.PeerState = peerState
	arguments.Marshalizer = marshalizer
	arguments.ScQuery = scDataGetter
	stakingToPeer, _ := NewStakingToPeer(arguments)

	blockBody := createBlockBody()
	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Nil(t, err)
}

func TestStakingToPeer_UpdateProtocolCannotSaveUnStakedNonceShouldErr(t *testing.T) {
	t.Parallel()

	testError := errors.New("error")
	address := "address"
	currTx := &mock.TxForCurrentBlockStub{}
	currTx.GetTxCalled = func(txHash []byte) (handler data.TransactionHandler, e error) {
		return &smartContractResult.SmartContractResult{
			RcvAddr: factory.StakingSCAddress,
		}, nil
	}

	argParser := &mock.ArgumentParserMock{}
	argParser.GetStorageUpdatesCalled = func(data string) (updates []*vmcommon.StorageUpdate, e error) {
		return []*vmcommon.StorageUpdate{
			{Offset: []byte("off1"), Data: []byte("data1")},
		}, nil
	}

	peerState := &mock.AccountsStub{}
	peerState.GetAccountWithJournalCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		peerAccount, _ := state.NewPeerAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{
			JournalizeCalled: func(entry state.JournalEntry) {
				return
			},
			SaveAccountCalled: func(accountHandler state.AccountHandler) error {
				return testError
			},
		})
		peerAccount.Stake = big.NewInt(100)
		peerAccount.BLSPublicKey = []byte(address)
		peerAccount.UnStakedNonce = 1
		return peerAccount, nil
	}

	stakingData := systemSmartContracts.StakedData{
		StakeValue:    big.NewInt(100),
		RewardAddress: []byte(address),
	}
	marshalizer := &mock.MarshalizerMock{}

	scDataGetter := &mock.ScQueryMock{}
	scDataGetter.ExecuteQueryCalled = func(query *process.SCQuery) (output *vmcommon.VMOutput, e error) {
		retData, _ := json.Marshal(&stakingData)
		return &vmcommon.VMOutput{ReturnData: [][]byte{retData}}, nil
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	arguments.PeerState = peerState
	arguments.Marshalizer = marshalizer
	arguments.ScQuery = scDataGetter
	stakingToPeer, _ := NewStakingToPeer(arguments)

	blockBody := createBlockBody()
	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Equal(t, testError, err)
}

func TestStakingToPeer_UpdateProtocolPeerChangesVerifyPeerChanges(t *testing.T) {
	t.Parallel()

	address := "address"
	currTx := &mock.TxForCurrentBlockStub{}
	currTx.GetTxCalled = func(txHash []byte) (handler data.TransactionHandler, e error) {
		return &smartContractResult.SmartContractResult{
			RcvAddr: factory.StakingSCAddress,
		}, nil
	}

	argParser := &mock.ArgumentParserMock{}
	argParser.GetStorageUpdatesCalled = func(data string) (updates []*vmcommon.StorageUpdate, e error) {
		return []*vmcommon.StorageUpdate{
			{Offset: []byte("off1"), Data: []byte("data1")},
		}, nil
	}

	peerState := &mock.AccountsStub{}
	peerState.GetAccountWithJournalCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		peerAccount, _ := state.NewPeerAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{
			JournalizeCalled: func(entry state.JournalEntry) {
				return
			},
			SaveAccountCalled: func(accountHandler state.AccountHandler) error {
				return nil
			},
		})
		peerAccount.Stake = big.NewInt(100)
		peerAccount.BLSPublicKey = []byte("")
		peerAccount.UnStakedNonce = 1
		return peerAccount, nil
	}

	stakeValue := big.NewInt(100)
	stakingData := systemSmartContracts.StakedData{
		StakeValue:    stakeValue,
		RewardAddress: []byte(address),
	}
	marshalizer := &mock.MarshalizerMock{}

	scDataGetter := &mock.ScQueryMock{}
	scDataGetter.ExecuteQueryCalled = func(query *process.SCQuery) (output *vmcommon.VMOutput, e error) {
		retData, _ := json.Marshal(&stakingData)
		return &vmcommon.VMOutput{ReturnData: [][]byte{retData}}, nil
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	arguments.PeerState = peerState
	arguments.Marshalizer = marshalizer
	arguments.ScQuery = scDataGetter
	stakingToPeer, _ := NewStakingToPeer(arguments)

	blockBody := createBlockBody()
	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Nil(t, err)

	peersData := stakingToPeer.PeerChanges()
	assert.Equal(t, 1, len(peersData))
	assert.Equal(t, stakeValue, peersData[0].ValueChange)

	err = stakingToPeer.VerifyPeerChanges(peersData)
	assert.Nil(t, err)
}

func TestStakingToPeer_VerifyPeerChangesShouldErr(t *testing.T) {
	t.Parallel()

	address := "address"
	currTx := &mock.TxForCurrentBlockStub{}
	currTx.GetTxCalled = func(txHash []byte) (handler data.TransactionHandler, e error) {
		return &smartContractResult.SmartContractResult{
			RcvAddr: factory.StakingSCAddress,
		}, nil
	}

	argParser := &mock.ArgumentParserMock{}
	argParser.GetStorageUpdatesCalled = func(data string) (updates []*vmcommon.StorageUpdate, e error) {
		return []*vmcommon.StorageUpdate{
			{Offset: []byte("off1"), Data: []byte("data1")},
		}, nil
	}

	peerState := &mock.AccountsStub{}
	peerState.GetAccountWithJournalCalled = func(addressContainer state.AddressContainer) (handler state.AccountHandler, e error) {
		peerAccount, _ := state.NewPeerAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{
			JournalizeCalled: func(entry state.JournalEntry) {
				return
			},
			SaveAccountCalled: func(accountHandler state.AccountHandler) error {
				return nil
			},
		})
		peerAccount.Stake = big.NewInt(100)
		peerAccount.BLSPublicKey = []byte("")
		peerAccount.UnStakedNonce = 1
		return peerAccount, nil
	}

	stakeValue := big.NewInt(100)
	stakingData := systemSmartContracts.StakedData{
		StakeValue:    stakeValue,
		RewardAddress: []byte(address),
	}
	marshalizer := &mock.MarshalizerMock{}

	scDataGetter := &mock.ScQueryMock{}
	scDataGetter.ExecuteQueryCalled = func(query *process.SCQuery) (output *vmcommon.VMOutput, e error) {
		retData, _ := json.Marshal(&stakingData)
		return &vmcommon.VMOutput{ReturnData: [][]byte{retData}}, nil
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	arguments.PeerState = peerState
	arguments.Marshalizer = marshalizer
	arguments.ScQuery = scDataGetter
	stakingToPeer, _ := NewStakingToPeer(arguments)

	blockBody := createBlockBody()
	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Nil(t, err)

	peersData := []block.PeerData{{}}
	err = stakingToPeer.VerifyPeerChanges(peersData)
	assert.Equal(t, process.ErrPeerChangesHashDoesNotMatch, err)
}
