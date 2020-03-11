package state_test

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/mock"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/stretchr/testify/assert"
)

func TestPeerAccount_MarshalUnmarshal_ShouldWork(t *testing.T) {
	t.Parallel()

	addr := &mock.AddressMock{}
	addrTr := &mock.AccountTrackerStub{}
	acnt, _ := state.NewPeerAccount(addr, addrTr)

	marshalizer := mock.MarshalizerMock{}
	buff, _ := marshalizer.Marshal(acnt)

	acntRecovered, _ := state.NewPeerAccount(addr, addrTr)
	_ = marshalizer.Unmarshal(acntRecovered, buff)

	assert.Equal(t, acnt, acntRecovered)
}

func TestPeerAccount_NewAccountNilAddress(t *testing.T) {
	t.Parallel()

	acc, err := state.NewPeerAccount(nil, &mock.AccountTrackerStub{})

	assert.Nil(t, acc)
	assert.Equal(t, err, state.ErrNilAddressContainer)
}

func TestPeerAccount_NewPeerAccountNilAaccountTracker(t *testing.T) {
	t.Parallel()

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, nil)

	assert.Nil(t, acc)
	assert.Equal(t, err, state.ErrNilAccountTracker)
}

func TestPeerAccount_NewPeerAccountOk(t *testing.T) {
	t.Parallel()

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})

	assert.Nil(t, err)
	assert.False(t, check.IfNil(acc))
}

func TestPeerAccount_AddressContainer(t *testing.T) {
	t.Parallel()

	addr := &mock.AddressMock{}
	acc, err := state.NewPeerAccount(addr, &mock.AccountTrackerStub{})

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, addr, acc.AddressContainer())
}

func TestPeerAccount_GetCode(t *testing.T) {
	t.Parallel()

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})
	assert.Nil(t, err)

	code := []byte("code")
	acc.SetCode(code)

	assert.NotNil(t, acc)
	assert.Equal(t, code, acc.GetCode())
}

func TestPeerAccount_GetCodeHash(t *testing.T) {
	t.Parallel()

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})
	assert.Nil(t, err)

	code := []byte("code")
	acc.CodeHash = code

	assert.NotNil(t, acc)
	assert.Equal(t, code, acc.GetCodeHash())
}

func TestPeerAccount_SetCodeHash(t *testing.T) {
	t.Parallel()

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})
	assert.Nil(t, err)

	code := []byte("code")
	acc.SetCodeHash(code)

	assert.NotNil(t, acc)
	assert.Equal(t, code, acc.GetCodeHash())
}

func TestPeerAccount_GetRootHash(t *testing.T) {
	t.Parallel()

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})
	assert.Nil(t, err)

	root := []byte("root")
	acc.RootHash = root

	assert.NotNil(t, acc)
	assert.Equal(t, root, acc.GetRootHash())
}

func TestPeerAccount_SetRootHash(t *testing.T) {
	t.Parallel()

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})
	assert.Nil(t, err)

	root := []byte("root")
	acc.SetRootHash(root)

	assert.NotNil(t, acc)
	assert.Equal(t, root, acc.GetRootHash())
}

func TestPeerAccount_DataTrie(t *testing.T) {
	t.Parallel()

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})
	assert.Nil(t, err)

	trie := &mock.TrieStub{}
	acc.SetDataTrie(trie)

	assert.NotNil(t, acc)
	assert.Equal(t, trie, acc.DataTrie())
}

func TestPeerAccount_SetNonceWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	nonce := uint64(0)
	err = acc.SetNonceWithJournal(nonce)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, nonce, acc.Nonce)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_SetCodeHashWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	codeHash := []byte("codehash")
	err = acc.SetCodeHashWithJournal(codeHash)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, codeHash, acc.CodeHash)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_SetAddressWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	address := []byte("address")
	err = acc.SetRewardAddressWithJournal(address)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, address, acc.RewardAddress)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_SetSchnorrPublicKeyWithJournalWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	pubKey := []byte("pubkey")
	err = acc.SetSchnorrPublicKeyWithJournal(pubKey)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, pubKey, acc.SchnorrPublicKey)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_SetBLSPublicKeyWithJournalWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	pubKey := []byte("pubkey")
	err = acc.SetBLSPublicKeyWithJournal(pubKey)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, pubKey, acc.BLSPublicKey)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_SetStakeWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	stake := big.NewInt(250000)
	err = acc.SetStakeWithJournal(stake)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, stake.Uint64(), acc.Stake.Uint64())
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_SetCurrentShardIdWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	shId := uint32(10)
	err = acc.SetCurrentShardIdWithJournal(shId)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, shId, acc.CurrentShardId)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_SetNextShardIdWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	shId := uint32(10)
	err = acc.SetNextShardIdWithJournal(shId)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, shId, acc.NextShardId)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_SetNodeInWaitingListWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	err = acc.SetNodeInWaitingListWithJournal(true)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, true, acc.NodeInWaitingList)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_SetRatingWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	rating := uint32(10)
	err = acc.SetRatingWithJournal(rating)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, rating, acc.Rating)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_SetJailTimeWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	jailTime := state.TimePeriod{
		StartTime: state.TimeStamp{Epoch: 12, Round: 12},
		EndTime:   state.TimeStamp{Epoch: 13, Round: 13},
	}
	err = acc.SetJailTimeWithJournal(jailTime)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, jailTime, acc.JailTime)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_IncreaseLeaderSuccessRateWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	acc.LeaderSuccessRate = state.SignRate{NrSuccess: 10, NrFailure: 10}
	err = acc.IncreaseLeaderSuccessRateWithJournal(1)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, uint32(11), acc.LeaderSuccessRate.NrSuccess)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_IncreaseValidatorSuccessRateWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	acc.ValidatorSuccessRate = state.SignRate{NrSuccess: 10, NrFailure: 10}
	err = acc.IncreaseValidatorSuccessRateWithJournal(1)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, uint32(11), acc.ValidatorSuccessRate.NrSuccess)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_DecreaseLeaderSuccessRateWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	acc.LeaderSuccessRate = state.SignRate{NrSuccess: 10, NrFailure: 10}
	err = acc.DecreaseLeaderSuccessRateWithJournal(1)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, uint32(11), acc.LeaderSuccessRate.NrFailure)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_DecreaseValidatorSuccessRateWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := 0
	saveAccountCalled := 0
	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled++
			return nil
		},
	}

	acc, err := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	assert.Nil(t, err)

	acc.ValidatorSuccessRate = state.SignRate{NrSuccess: 10, NrFailure: 10}
	err = acc.DecreaseValidatorSuccessRateWithJournal(1)

	assert.NotNil(t, acc)
	assert.Nil(t, err)
	assert.Equal(t, uint32(11), acc.ValidatorSuccessRate.NrFailure)
	assert.Equal(t, 1, journalizeCalled)
	assert.Equal(t, 1, saveAccountCalled)
}

func TestPeerAccount_IncreaseNumSelectedInSuccessBlocks(t *testing.T) {
	t.Parallel()

	journalizeCalled := false
	saveAccountCalled := false

	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled = true
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled = true
			return nil
		},
	}

	acc, _ := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	err := acc.IncreaseNumSelectedInSuccessBlocks()

	assert.Nil(t, err)
	assert.Equal(t, uint32(1), acc.NumSelectedInSuccessBlocks)
	assert.True(t, journalizeCalled)
	assert.True(t, saveAccountCalled)
}

func TestPeerAccount_IncreaseNumSelectedInSuccessBlocksWithRevert(t *testing.T) {
	t.Parallel()

	journalizeCalled := false
	saveAccountCalled := false

	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled = true
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled = true
			return nil
		},
	}

	acc, _ := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	err := acc.IncreaseNumSelectedInSuccessBlocks()

	assert.Nil(t, err)
	assert.Equal(t, uint32(1), acc.NumSelectedInSuccessBlocks)
	assert.True(t, journalizeCalled)
	assert.True(t, saveAccountCalled)
}

func TestPeerAccount_AddToAccumulatedFees(t *testing.T) {
	t.Parallel()

	journalizeCalled := false
	saveAccountCalled := false
	accFee := big.NewInt(37)

	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled = true
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled = true
			return nil
		},
	}

	acc, _ := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	err := acc.AddToAccumulatedFees(accFee)

	assert.Nil(t, err)
	assert.Equal(t, accFee.Int64(), acc.AccumulatedFees.Int64())
	assert.True(t, journalizeCalled)
	assert.True(t, saveAccountCalled)
}

func TestPeerAccount_SetTempRatingWithJournal(t *testing.T) {
	t.Parallel()

	journalizeCalled := false
	saveAccountCalled := false
	tempRating := uint32(100)

	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			journalizeCalled = true
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			saveAccountCalled = true
			return nil
		},
	}

	acc, _ := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	err := acc.SetTempRatingWithJournal(tempRating)

	assert.Nil(t, err)
	assert.Equal(t, tempRating, acc.TempRating)
	assert.True(t, journalizeCalled)
	assert.True(t, saveAccountCalled)
}

func TestPeerAccount_ResetAtNewEpoch(t *testing.T) {
	numTimesJournalizeWasCalled := 0
	numTimesSaveAccountWasCalled := 0

	tracker := &mock.AccountTrackerStub{
		JournalizeCalled: func(entry state.JournalEntry) {
			numTimesJournalizeWasCalled++
		},
		SaveAccountCalled: func(accountHandler state.AccountHandler) error {
			numTimesSaveAccountWasCalled++
			return nil
		},
	}

	acc, _ := state.NewPeerAccount(&mock.AddressMock{}, tracker)
	err := acc.ResetAtNewEpoch()

	assert.Nil(t, err)
	assert.Equal(t, 5, numTimesJournalizeWasCalled)
	assert.Equal(t, 5, numTimesSaveAccountWasCalled)
}
