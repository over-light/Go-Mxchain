package mock

import (
	"errors"

	"github.com/ElrondNetwork/elrond-go/data"
)

var errNotImplemented = errors.New("not implemented")

type TrieStub struct {
	GetCalled          func(key []byte) ([]byte, error)
	UpdateCalled       func(key, value []byte) error
	DeleteCalled       func(key []byte) error
	RootCalled         func() ([]byte, error)
	ProveCalled        func(key []byte) ([][]byte, error)
	VerifyProofCalled  func(proofs [][]byte, key []byte) (bool, error)
	CommitCalled       func() error
	RecreateCalled     func(root []byte) (data.Trie, error)
	DeepCloneCalled    func() (data.Trie, error)
	GetAllLeavesCalled func() (map[string][]byte, error)
}

func (ts *TrieStub) ClosePersister() error {
	return nil
}

func (ts *TrieStub) Get(key []byte) ([]byte, error) {
	if ts.GetCalled != nil {
		return ts.GetCalled(key)
	}

	return nil, errNotImplemented
}

func (ts *TrieStub) Update(key, value []byte) error {
	if ts.UpdateCalled != nil {
		return ts.UpdateCalled(key, value)
	}

	return errNotImplemented
}

func (ts *TrieStub) Delete(key []byte) error {
	if ts.DeleteCalled != nil {
		return ts.DeleteCalled(key)
	}

	return errNotImplemented
}

func (ts *TrieStub) Root() ([]byte, error) {
	if ts.RootCalled != nil {
		return ts.RootCalled()
	}

	return nil, errNotImplemented
}

func (ts *TrieStub) Prove(key []byte) ([][]byte, error) {
	if ts.ProveCalled != nil {
		return ts.ProveCalled(key)
	}

	return nil, errNotImplemented
}

func (ts *TrieStub) VerifyProof(proofs [][]byte, key []byte) (bool, error) {
	if ts.VerifyProofCalled != nil {
		return ts.VerifyProofCalled(proofs, key)
	}

	return false, errNotImplemented
}

func (ts *TrieStub) Commit() error {
	if ts != nil {
		return ts.CommitCalled()
	}

	return errNotImplemented
}

func (ts *TrieStub) Recreate(root []byte) (data.Trie, error) {
	if ts.RecreateCalled != nil {
		return ts.RecreateCalled(root)
	}

	return nil, errNotImplemented
}

func (ts *TrieStub) String() string {
	return "stub trie"
}

func (ts *TrieStub) DeepClone() (data.Trie, error) {
	return ts.DeepCloneCalled()
}

func (ts *TrieStub) GetAllLeaves() (map[string][]byte, error) {
	if ts.GetAllLeavesCalled != nil {
		return ts.GetAllLeavesCalled()
	}

	return nil, errNotImplemented
}

// IsInterfaceNil returns true if there is no value under the interface
func (ts *TrieStub) IsInterfaceNil() bool {
	if ts == nil {
		return true
	}
	return false
}
