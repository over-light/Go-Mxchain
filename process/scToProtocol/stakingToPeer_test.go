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
	"github.com/ElrondNetwork/elrond-go/vm/factory"
	"github.com/ElrondNetwork/elrond-go/vm/systemSmartContracts"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/assert"
)

func createMockArgumentsNewStakingToPeer() ArgStakingToPeer {
	return ArgStakingToPeer{
		AdrConv:      &mock.AddressConverterMock{},
		Hasher:       &mock.HasherMock{},
		Marshalizer:  &mock.MarshalizerStub{},
		PeerState:    &mock.AccountsStub{},
		BaseState:    &mock.AccountsStub{},
		ArgParser:    &mock.ArgumentParserMock{},
		CurrTxs:      &mock.TxForCurrentBlockStub{},
		ScDataGetter: &mock.ScDataGetterMock{},
	}
}

func createBlockBody() block.Body {
	return block.Body{
		{
			TxHashes:        [][]byte{[]byte("hash1"), []byte("hash2")},
			ReceiverShardID: 0,
			SenderShardID:   0,
			Type:            block.SmartContractResultBlock,
		},
	}
}

func TestNewStakingToPeerNilAddrConverterShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.AdrConv = nil

	_, err := NewStakingToPeer(arguments)
	assert.Equal(t, process.ErrNilAddressConverter, err)
}

func TestNewStakingToPeerNilHasherShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.Hasher = nil

	_, err := NewStakingToPeer(arguments)
	assert.Equal(t, process.ErrNilHasher, err)
}

func TestNewStakingToPeerNilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.Marshalizer = nil

	_, err := NewStakingToPeer(arguments)
	assert.Equal(t, process.ErrNilMarshalizer, err)
}

func TestNewStakingToPeerNilPeerAccountAdapterShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.PeerState = nil

	_, err := NewStakingToPeer(arguments)
	assert.Equal(t, process.ErrNilPeerAccountsAdapter, err)
}

func TestNewStakingToPeerNilBaseAccountAdapterShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.BaseState = nil

	_, err := NewStakingToPeer(arguments)
	assert.Equal(t, process.ErrNilAccountsAdapter, err)
}

func TestNewStakingToPeerNilArgumentParserShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = nil

	_, err := NewStakingToPeer(arguments)
	assert.Equal(t, process.ErrNilArgumentParser, err)
}

func TestNewStakingToPeerNilCurrentBlockHeaderShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.CurrTxs = nil

	_, err := NewStakingToPeer(arguments)
	assert.Equal(t, process.ErrNilTxForCurrentBlockHandler, err)
}

func TestNewStakingToPeerNilScDataGetterShouldErr(t *testing.T) {
	t.Parallel()

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ScDataGetter = nil

	_, err := NewStakingToPeer(arguments)
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

	testError := errors.New("error")
	currTx := &mock.TxForCurrentBlockStub{}
	currTx.GetTxCalled = func(txHash []byte) (handler data.TransactionHandler, e error) {
		return nil, testError
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.CurrTxs = currTx
	blockBody := createBlockBody()
	stakingToPeer, _ := NewStakingToPeer(arguments)

	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Equal(t, testError, err)
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
	blockBody := createBlockBody()
	stakingToPeer, _ := NewStakingToPeer(arguments)

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
	blockBody := createBlockBody()
	stakingToPeer, _ := NewStakingToPeer(arguments)

	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Equal(t, testError, err)
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
	blockBody := createBlockBody()
	stakingToPeer, _ := NewStakingToPeer(arguments)

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
			Address:      []byte("addr"),
			BLSPublicKey: []byte("BlsAddr"),
			Stake:        big.NewInt(100),
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
	blockBody := createBlockBody()
	stakingToPeer, _ := NewStakingToPeer(arguments)

	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Nil(t, err)
}

func TestStakingToPeer_UpdateProtocolCannotSetBLSPublicKeyShouldErr(t *testing.T) {
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
		peerAccount.BLSPublicKey = []byte("key")
		return peerAccount, nil
	}

	stakingData := systemSmartContracts.StakingData{
		StakeValue: big.NewInt(100),
	}
	marshalizer := &mock.MarshalizerStub{}
	marshalizer.MarshalCalled = func(obj interface{}) (bytes []byte, e error) {
		return json.Marshal(obj)
	}
	marshalizer.UnmarshalCalled = func(obj interface{}, buff []byte) error {
		return json.Unmarshal(buff, obj)
	}

	scDataGetter := &mock.ScDataGetterMock{}
	scDataGetter.GetCalled = func(scAddress []byte, funcName string, args ...[]byte) (bytes []byte, e error) {
		return json.Marshal(&stakingData)
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	arguments.PeerState = peerState
	arguments.Marshalizer = marshalizer
	arguments.ScDataGetter = scDataGetter
	blockBody := createBlockBody()
	stakingToPeer, _ := NewStakingToPeer(arguments)

	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Equal(t, state.ErrNilBLSPublicKey, err)
}

func TestStakingToPeer_UpdateProtocolCannotSaveAccountShouldErr(t *testing.T) {
	t.Parallel()

	testError := errors.New("error")
	blsPublicKey := "blsPublicKey"
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
		peerAccount.BLSPublicKey = []byte(blsPublicKey)
		return peerAccount, nil
	}

	stakingData := systemSmartContracts.StakingData{
		StakeValue: big.NewInt(100),
		BlsPubKey:  []byte(blsPublicKey),
	}
	marshalizer := &mock.MarshalizerStub{}
	marshalizer.MarshalCalled = func(obj interface{}) (bytes []byte, e error) {
		return json.Marshal(obj)
	}
	marshalizer.UnmarshalCalled = func(obj interface{}, buff []byte) error {
		return json.Unmarshal(buff, obj)
	}

	scDataGetter := &mock.ScDataGetterMock{}
	scDataGetter.GetCalled = func(scAddress []byte, funcName string, args ...[]byte) (bytes []byte, e error) {
		return json.Marshal(&stakingData)
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	arguments.PeerState = peerState
	arguments.Marshalizer = marshalizer
	arguments.ScDataGetter = scDataGetter
	blockBody := createBlockBody()
	stakingToPeer, _ := NewStakingToPeer(arguments)

	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Equal(t, testError, err)
}

func TestStakingToPeer_UpdateProtocolCannotSaveAccountNonceShouldErr(t *testing.T) {
	t.Parallel()

	testError := errors.New("error")
	blsPublicKey := "blsPublicKey"
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
		peerAccount.BLSPublicKey = []byte(blsPublicKey)
		peerAccount.Nonce = 1
		return peerAccount, nil
	}

	stakingData := systemSmartContracts.StakingData{
		StakeValue: big.NewInt(100),
		BlsPubKey:  []byte(blsPublicKey),
	}
	marshalizer := &mock.MarshalizerStub{}
	marshalizer.MarshalCalled = func(obj interface{}) (bytes []byte, e error) {
		return json.Marshal(obj)
	}
	marshalizer.UnmarshalCalled = func(obj interface{}, buff []byte) error {
		return json.Unmarshal(buff, obj)
	}

	scDataGetter := &mock.ScDataGetterMock{}
	scDataGetter.GetCalled = func(scAddress []byte, funcName string, args ...[]byte) (bytes []byte, e error) {
		return json.Marshal(&stakingData)
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	arguments.PeerState = peerState
	arguments.Marshalizer = marshalizer
	arguments.ScDataGetter = scDataGetter
	blockBody := createBlockBody()
	stakingToPeer, _ := NewStakingToPeer(arguments)

	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Equal(t, testError, err)
}

func TestStakingToPeer_UpdateProtocol(t *testing.T) {
	t.Parallel()

	blsPublicKey := "blsPublicKey"
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
		peerAccount.BLSPublicKey = []byte(blsPublicKey)
		peerAccount.Nonce = 1
		return peerAccount, nil
	}

	stakingData := systemSmartContracts.StakingData{
		StakeValue: big.NewInt(100),
		BlsPubKey:  []byte(blsPublicKey),
	}
	marshalizer := &mock.MarshalizerStub{}
	marshalizer.MarshalCalled = func(obj interface{}) (bytes []byte, e error) {
		return json.Marshal(obj)
	}
	marshalizer.UnmarshalCalled = func(obj interface{}, buff []byte) error {
		return json.Unmarshal(buff, obj)
	}

	scDataGetter := &mock.ScDataGetterMock{}
	scDataGetter.GetCalled = func(scAddress []byte, funcName string, args ...[]byte) (bytes []byte, e error) {
		return json.Marshal(&stakingData)
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	arguments.PeerState = peerState
	arguments.Marshalizer = marshalizer
	arguments.ScDataGetter = scDataGetter
	blockBody := createBlockBody()
	stakingToPeer, _ := NewStakingToPeer(arguments)

	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Nil(t, err)
}

func TestStakingToPeer_UpdateProtocolCannotSaveUnStakedNonceShouldErr(t *testing.T) {
	t.Parallel()

	testError := errors.New("error")
	blsPublicKey := "blsPublicKey"
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
		peerAccount.BLSPublicKey = []byte(blsPublicKey)
		peerAccount.UnStakedNonce = 1
		return peerAccount, nil
	}

	stakingData := systemSmartContracts.StakingData{
		StakeValue: big.NewInt(100),
		BlsPubKey:  []byte(blsPublicKey),
	}
	marshalizer := &mock.MarshalizerStub{}
	marshalizer.MarshalCalled = func(obj interface{}) (bytes []byte, e error) {
		return json.Marshal(obj)
	}
	marshalizer.UnmarshalCalled = func(obj interface{}, buff []byte) error {
		return json.Unmarshal(buff, obj)
	}

	scDataGetter := &mock.ScDataGetterMock{}
	scDataGetter.GetCalled = func(scAddress []byte, funcName string, args ...[]byte) (bytes []byte, e error) {
		return json.Marshal(&stakingData)
	}

	arguments := createMockArgumentsNewStakingToPeer()
	arguments.ArgParser = argParser
	arguments.CurrTxs = currTx
	arguments.PeerState = peerState
	arguments.Marshalizer = marshalizer
	arguments.ScDataGetter = scDataGetter
	blockBody := createBlockBody()
	stakingToPeer, _ := NewStakingToPeer(arguments)

	err := stakingToPeer.UpdateProtocol(blockBody, 0)
	assert.Equal(t, testError, err)
}
