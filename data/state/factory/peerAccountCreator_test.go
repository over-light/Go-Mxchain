package factory_test

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/mock"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/state/factory"
	"github.com/stretchr/testify/assert"
)

func TestPeerAccountCreator_CreateAccountNilAddress(t *testing.T) {
	t.Parallel()

	accF, err := factory.NewAccountFactoryCreator(state.ValidatorAccount)
	assert.Nil(t, err)

	_, ok := accF.(*factory.PeerAccountCreator)
	assert.Equal(t, true, ok)

	acc, err := accF.CreateAccount(nil, &mock.AccountTrackerStub{})

	assert.Nil(t, acc)
	assert.Equal(t, err, state.ErrNilAddressContainer)
}

func TestPeerAccountCreator_CreateAccountNilAccountTraccer(t *testing.T) {
	t.Parallel()

	accF, err := factory.NewAccountFactoryCreator(state.ValidatorAccount)
	assert.Nil(t, err)

	_, ok := accF.(*factory.PeerAccountCreator)
	assert.Equal(t, true, ok)

	acc, err := accF.CreateAccount(&mock.AddressMock{}, nil)

	assert.Nil(t, acc)
	assert.Equal(t, err, state.ErrNilAccountTracker)
}

func TestPeerAccountCreator_CreateAccountOk(t *testing.T) {
	t.Parallel()

	accF, err := factory.NewAccountFactoryCreator(state.ValidatorAccount)
	assert.Nil(t, err)
	assert.False(t, check.IfNil(accF))

	_, ok := accF.(*factory.PeerAccountCreator)
	assert.Equal(t, true, ok)

	acc, err := accF.CreateAccount(&mock.AddressMock{}, &mock.AccountTrackerStub{})

	assert.NotNil(t, acc)
	assert.Nil(t, err)
}
