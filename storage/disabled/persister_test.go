package disabled

import (
	"fmt"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	storageRepo "github.com/ElrondNetwork/elrond-go-storage"
	"github.com/stretchr/testify/assert"
)

func TestPersister_MethodsDoNotPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r != nil {
			assert.Fail(t, fmt.Sprintf("should have not panicked: %v", r))
		}
	}()

	p := NewPersister()
	assert.False(t, check.IfNil(p))
	assert.Nil(t, p.Put(nil, nil))
	assert.Equal(t, storageRepo.ErrKeyNotFound, p.Has(nil))
	assert.Nil(t, p.Close())
	assert.Nil(t, p.Remove(nil))
	assert.Nil(t, p.Destroy())
	assert.Nil(t, p.DestroyClosed())
	p.RangeKeys(nil)

	val, err := p.Get(nil)
	assert.Nil(t, val)
	assert.Equal(t, storageRepo.ErrKeyNotFound, err)
}
