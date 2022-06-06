package state

import (
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

// >> BEGIN notes for review
// "accountsDBApi" could not be used because it only allows its creator (caller of constructor) to control the provisioning of the "targetRootHash".
// Being constrained by the AccountsAdapter interface, it does not allow the caller of its methods to specify the "targetRootHash".
// > Design constraint: we have to return the nonce & hash associated with the rootHash (on GET account?onFinalBlock=true), as well.
// > Design constraint: the one that provides the "targetRootHash" must also access in a consistent manner (e.g. critical section) the block nonce & block hash associated with the rootHash,
// so that they are paired (in a consistent manner) with the loaded account data.
//
// Having "accountsDBApi" to use blockchain.GetFinalBlockInfo().rootHash to load the account data, then having the "nodeFacade" to return the block nonce and block hash (also from the chainHandler)
// would have possibly resulted in occasional inconsistencies. Instead, we'll make the "nodeFacade" responsible to call blockchain.GetFinalBlockInfo(), hold the results,
// call accountsRepository with the returned rootHash etc.
// << END notes for review

// Question for review: perhaps rename to "accountsByRootHashRepository"?
type accountsRepository struct {
	innerAccountsAdapter AccountsAdapter
	trieController       *accountsDBApiTrieController
}

// NewAccountsRepository creates a new accountsRepository
func NewAccountsRepository(innerAccountsAdapter AccountsAdapter) (*accountsRepository, error) {
	if check.IfNil(innerAccountsAdapter) {
		return nil, ErrNilAccountsAdapter
	}

	return &accountsRepository{
		innerAccountsAdapter: innerAccountsAdapter,
		trieController:       newAccountsDBApiTrieController(innerAccountsAdapter),
	}, nil
}

// GetExistingAccount will call the inner accountsAdapter method after trying to recreate the trie
func (repository *accountsRepository) GetExistingAccount(address []byte, rootHash []byte) (vmcommon.AccountHandler, error) {
	err := repository.trieController.recreateTrieIfNecessary(rootHash)
	if err != nil {
		return nil, err
	}

	return repository.innerAccountsAdapter.GetExistingAccount(address)
}

// GetCode will call the inner accountsAdapter method after trying to recreate the trie
func (repository *accountsRepository) GetCode(codeHash []byte, rootHash []byte) []byte {
	err := repository.trieController.recreateTrieIfNecessary(rootHash)
	if err != nil {
		return nil
	}

	return repository.innerAccountsAdapter.GetCode(codeHash)
}

// Close will handle the closing of the underlying components
func (repository *accountsRepository) Close() error {
	return repository.innerAccountsAdapter.Close()
}

// IsInterfaceNil returns true if there is no value under the interface
func (repository *accountsRepository) IsInterfaceNil() bool {
	return repository == nil
}
