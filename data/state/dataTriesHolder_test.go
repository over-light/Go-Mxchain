package state_test

import (
	"strconv"
	"sync"
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/mock"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/stretchr/testify/assert"
)

func TestNewDataTriesHolder(t *testing.T) {
	t.Parallel()

	dth := state.NewDataTriesHolder()
	assert.NotNil(t, dth)
}

func TestDataTriesHolder_PutAndGet(t *testing.T) {
	t.Parallel()

	tr1 := &mock.TrieStub{}

	dth := state.NewDataTriesHolder()
	dth.Put([]byte("trie1"), tr1)
	tr := dth.Get([]byte("trie1"))

	assert.True(t, tr == tr1)
}

func TestDataTriesHolder_GetAll(t *testing.T) {
	t.Parallel()

	tr1 := &mock.TrieStub{}
	tr2 := &mock.TrieStub{}
	tr3 := &mock.TrieStub{}

	dth := state.NewDataTriesHolder()
	dth.Put([]byte("trie1"), tr1)
	dth.Put([]byte("trie2"), tr2)
	dth.Put([]byte("trie3"), tr3)
	tries := dth.GetAll()

	assert.Equal(t, 3, len(tries))
}

func TestDataTriesHolder_Reset(t *testing.T) {
	t.Parallel()

	tr1 := &mock.TrieStub{}

	dth := state.NewDataTriesHolder()
	dth.Put([]byte("trie1"), tr1)
	dth.Reset()

	tr := dth.Get([]byte("trie1"))
	assert.Nil(t, tr)
}

func TestDataTriesHolder_Concurrency(t *testing.T) {
	t.Parallel()

	dth := state.NewDataTriesHolder()
	numTries := 1000

	wg := sync.WaitGroup{}
	wg.Add(numTries)

	for i := 0; i < numTries; i++ {
		go func(key int) {
			dth.Put([]byte(strconv.Itoa(key)), &mock.TrieStub{})
			wg.Done()
		}(i)
	}

	wg.Wait()

	tries := dth.GetAll()
	assert.Equal(t, numTries, len(tries))
}
