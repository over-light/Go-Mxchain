package state

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
)

// PeerAccountsDB will save and synchronize data from peer processor, plus will synchronize with nodesCoordinator
type PeerAccountsDB struct {
	*AccountsDB
}

// NewPeerAccountsDB creates a new account manager
func NewPeerAccountsDB(
	trie data.Trie,
	hasher hashing.Hasher,
	marshalizer marshal.Marshalizer,
	accountFactory AccountFactory,
) (*PeerAccountsDB, error) {
	if trie == nil {
		return nil, ErrNilTrie
	}
	if hasher == nil {
		return nil, ErrNilHasher
	}
	if marshalizer == nil {
		return nil, ErrNilMarshalizer
	}
	if accountFactory == nil {
		return nil, ErrNilAccountFactory
	}

	return &PeerAccountsDB{
		&AccountsDB{
			mainTrie:       trie,
			hasher:         hasher,
			marshalizer:    marshalizer,
			accountFactory: accountFactory,
			entries:        make([]JournalEntry, 0),
			mutEntries:     sync.RWMutex{},
		},
	}, nil
}
