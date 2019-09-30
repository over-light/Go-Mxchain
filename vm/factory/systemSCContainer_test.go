package factory

import (
	"github.com/ElrondNetwork/elrond-go/vm"
	"testing"

	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/vm/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewSystemSCContainer_ShouldWork(t *testing.T) {
	t.Parallel()

	c := NewSystemSCContainer()

	assert.NotNil(t, c)
}

//------- Add

func TestSystemSCContainer_AddAlreadyExistingShouldErr(t *testing.T) {
	t.Parallel()

	c := NewSystemSCContainer()

	_ = c.Add([]byte("0001"), &mock.SystemSCStub{})
	err := c.Add([]byte("0001"), &mock.SystemSCStub{})

	assert.Equal(t, process.ErrContainerKeyAlreadyExists, err)
}

func TestSystemSCContainer_AddNilShouldErr(t *testing.T) {
	t.Parallel()

	c := NewSystemSCContainer()

	err := c.Add([]byte("0001"), nil)

	assert.Equal(t, process.ErrNilContainerElement, err)
}

func TestSystemSCContainer_AddShouldWork(t *testing.T) {
	t.Parallel()

	c := NewSystemSCContainer()

	err := c.Add([]byte("0001"), &mock.SystemSCStub{})

	assert.Nil(t, err)
	assert.Equal(t, 1, c.Len())
}

//------- AddMultiple

func TestSystemSCContainer_AddMultipleAlreadyExistingShouldErr(t *testing.T) {
	t.Parallel()

	c := NewSystemSCContainer()

	keys := [][]byte{[]byte("0001"), []byte("0001")}
	scs := []vm.SystemSmartContract{&mock.SystemSCStub{}, &mock.SystemSCStub{}}

	err := c.AddMultiple(keys, scs)

	assert.Equal(t, process.ErrContainerKeyAlreadyExists, err)
}

func TestSystemSCContainer_AddMultipleLenMismatchShouldErr(t *testing.T) {
	t.Parallel()

	c := NewSystemSCContainer()

	keys := [][]byte{[]byte("0001")}
	scs := []vm.SystemSmartContract{&mock.SystemSCStub{}, &mock.SystemSCStub{}}

	err := c.AddMultiple(keys, scs)

	assert.Equal(t, process.ErrLenMismatch, err)
}

func TestSystemSCContainer_AddMultipleShouldWork(t *testing.T) {
	t.Parallel()

	c := NewSystemSCContainer()

	keys := [][]byte{[]byte("0001"), []byte("0002")}
	scs := []vm.SystemSmartContract{&mock.SystemSCStub{}, &mock.SystemSCStub{}}

	err := c.AddMultiple(keys, scs)

	assert.Nil(t, err)
	assert.Equal(t, 2, c.Len())
}

//------- Get

func TestSystemSCContainer_GetNotFoundShouldErr(t *testing.T) {
	t.Parallel()

	c := NewSystemSCContainer()

	key := []byte("0001")
	keyNotFound := []byte("0002")
	val := &mock.SystemSCStub{}

	_ = c.Add(key, val)
	valRecovered, err := c.Get(keyNotFound)

	assert.Nil(t, valRecovered)
	assert.Equal(t, process.ErrInvalidContainerKey, err)
}

func TestSystemSCContainer_GetShouldWork(t *testing.T) {
	t.Parallel()

	c := NewSystemSCContainer()

	key := []byte("0001")
	val := &mock.SystemSCStub{}

	_ = c.Add(key, val)
	valRecovered, err := c.Get(key)

	assert.True(t, val == valRecovered)
	assert.Nil(t, err)
}

//------- Replace

func TestSystemSCContainer_ReplaceNilValueShouldErrAndNotModify(t *testing.T) {
	t.Parallel()

	c := NewSystemSCContainer()

	key := []byte("0001")
	val := &mock.SystemSCStub{}

	_ = c.Add(key, val)
	err := c.Replace(key, nil)

	valRecovered, _ := c.Get(key)

	assert.Equal(t, process.ErrNilContainerElement, err)
	assert.Equal(t, val, valRecovered)
}

func TestSystemSCContainer_ReplaceShouldWork(t *testing.T) {
	t.Parallel()

	c := NewSystemSCContainer()

	key := []byte("0001")
	val := &mock.SystemSCStub{}
	val2 := &mock.SystemSCStub{}

	_ = c.Add(key, val)
	err := c.Replace(key, val2)

	valRecovered, _ := c.Get(key)

	assert.True(t, val2 == valRecovered)
	assert.Nil(t, err)
}

//------- Remove

func TestSystemSCContainer_RemoveShouldWork(t *testing.T) {
	t.Parallel()

	c := NewSystemSCContainer()

	key := []byte("0001")
	val := &mock.SystemSCStub{}

	_ = c.Add(key, val)
	c.Remove(key)

	valRecovered, err := c.Get(key)

	assert.Nil(t, valRecovered)
	assert.Equal(t, process.ErrInvalidContainerKey, err)
}

//------- Len

func TestSystemSCContainer_LenShouldWork(t *testing.T) {
	t.Parallel()

	c := NewSystemSCContainer()

	_ = c.Add([]byte("0001"), &mock.SystemSCStub{})
	assert.Equal(t, 1, c.Len())

	keys := c.Keys()
	assert.Equal(t, 1, len(keys))
	assert.Equal(t, []byte("0001"), keys[0])

	_ = c.Add([]byte("0002"), &mock.SystemSCStub{})
	assert.Equal(t, 2, c.Len())

	keys = c.Keys()
	assert.Equal(t, 2, len(keys))
	assert.Contains(t, keys, []byte("0001"))
	assert.Contains(t, keys, []byte("0002"))

	c.Remove([]byte("0002"))
	assert.Equal(t, 1, c.Len())

	keys = c.Keys()
	assert.Equal(t, 1, len(keys))
	assert.Equal(t, []byte("0001"), keys[0])
}
