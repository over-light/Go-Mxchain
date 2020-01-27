package receipt_test

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/data/receipt"
	"github.com/stretchr/testify/assert"
)

func TestReceipt_SettersAndGetters(t *testing.T) {
	t.Parallel()

	r := receipt.Receipt{}

	data := []byte("data")
	value := big.NewInt(37)
	addr := []byte("addr")

	r.SetData(data)
	r.SetValue(value)
	r.SetRecvAddress(addr)
	r.SetSndAddress(addr)

	assert.Equal(t, data, r.GetData())
	assert.Equal(t, value, r.GetValue())
	assert.Equal(t, addr, r.GetRecvAddress())
	assert.Equal(t, addr, r.GetSndAddress())
	assert.Equal(t, uint64(0), r.GetNonce())
	assert.Equal(t, uint64(0), r.GetGasPrice())
	assert.Equal(t, uint64(0), r.GetGasLimit())
}
