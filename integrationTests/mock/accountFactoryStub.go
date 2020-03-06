package mock

import "github.com/ElrondNetwork/elrond-go/data/state"

// AccountsFactoryStub -
type AccountsFactoryStub struct {
	CreateAccountCalled func(address state.AddressContainer, tracker state.AccountTracker) (state.AccountHandler, error)
}

// CreateAccount -
func (afs *AccountsFactoryStub) CreateAccount(address state.AddressContainer, tracker state.AccountTracker) (state.AccountHandler, error) {
	return afs.CreateAccountCalled(address, tracker)
}

// IsInterfaceNil returns true if there is no value under the interface
func (afs *AccountsFactoryStub) IsInterfaceNil() bool {
	return afs == nil
}
