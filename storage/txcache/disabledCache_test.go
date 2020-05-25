package txcache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDisabledCache_DoesNothing(t *testing.T) {
	cache := NewDisabledCache()

	ok, added := cache.AddTx(nil)
	require.False(t, ok)
	require.False(t, added)

	tx, ok := cache.GetByTxHash([]byte{})
	require.Nil(t, tx)
	require.False(t, ok)

	selection := cache.SelectTransactions(42, 42)
	require.Equal(t, 0, len(selection))

	err := cache.RemoveTxByHash([]byte{})
	require.Nil(t, err)

	count := cache.CountTx()
	require.Equal(t, int64(0), count)

	length := cache.Len()
	require.Equal(t, 0, length)

	require.NotPanics(t, func() { cache.ForEachTransaction(func(_ []byte, _ *WrappedTransaction) {}) })

	cache.Clear()

	evicted := cache.Put(nil, nil)
	require.False(t, evicted)

	value, ok := cache.Get([]byte{})
	require.Nil(t, value)
	require.False(t, ok)

	value, ok = cache.Peek([]byte{})
	require.Nil(t, value)
	require.False(t, ok)

	has := cache.Has([]byte{})
	require.False(t, has)

	has, evicted = cache.HasOrAdd([]byte{}, nil)
	require.False(t, has)
	require.False(t, evicted)

	cache.Remove([]byte{})
	cache.RemoveOldest()

	maxSize := cache.MaxSize()
	require.Equal(t, 0, maxSize)

	require.NotPanics(t, func() { cache.RegisterHandler(func(_ []byte, _ interface{}) {}) })
	require.False(t, cache.IsInterfaceNil())
}
