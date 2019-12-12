package smartContractResult

import (
	"io"
	"math/big"

	"github.com/ElrondNetwork/elrond-go/data/smartContractResult/capnp"
	capn "github.com/glycerine/go-capnproto"
)

// SmartContractResult holds all the data needed for results coming from smart contract processing
type SmartContractResult struct {
	Nonce    uint64   `capid:"0" json:"nonce"`
	Value    *big.Int `capid:"1" json:"value"`
	RcvAddr  []byte   `capid:"2" json:"receiver"`
	SndAddr  []byte   `capid:"3" json:"sender"`
	Code     []byte   `capid:"4" json:"code,omitempty"`
	Data     string   `capid:"5" json:"data,omitempty"`
	TxHash   []byte   `capid:"6" json:"txHash"`
	GasLimit uint64   `capid:"7" json:"gasLimit"`
	GasPrice uint64   `capid:"8" json:"gasPrice"`
}

// Save saves the serialized data of a SmartContractResult into a stream through Capnp protocol
func (scr *SmartContractResult) Save(w io.Writer) error {
	seg := capn.NewBuffer(nil)
	SmartContractResultGoToCapn(seg, scr)
	_, err := seg.WriteTo(w)
	return err
}

// Load loads the data from the stream into a SmartContractResult object through Capnp protocol
func (scr *SmartContractResult) Load(r io.Reader) error {
	capMsg, err := capn.ReadFromStream(r, nil)
	if err != nil {
		return err
	}

	z := capnp.ReadRootSmartContractResultCapn(capMsg)
	SmartContractResultCapnToGo(z, scr)
	return nil
}

// SmartContractResultCapnToGo is a helper function to copy fields from a SmartContractResultCapn object to a SmartContractResult object
func SmartContractResultCapnToGo(src capnp.SmartContractResultCapn, dest *SmartContractResult) *SmartContractResult {
	if dest == nil {
		dest = &SmartContractResult{}
	}

	if dest.Value == nil {
		dest.Value = big.NewInt(0)
	}

	dest.Nonce = src.Nonce()
	err := dest.Value.GobDecode(src.Value())

	if err != nil {
		return nil
	}

	dest.RcvAddr = src.RcvAddr()
	dest.SndAddr = src.SndAddr()
	dest.Data = string(src.Data())
	dest.Code = src.Code()
	dest.TxHash = src.TxHash()

	return dest
}

// SmartContractResultGoToCapn is a helper function to copy fields from a SmartContractResult object to a SmartContractResultCapn object
func SmartContractResultGoToCapn(seg *capn.Segment, src *SmartContractResult) capnp.SmartContractResultCapn {
	dest := capnp.AutoNewSmartContractResultCapn(seg)

	value, _ := src.Value.GobEncode()
	dest.SetNonce(src.Nonce)
	dest.SetValue(value)
	dest.SetRcvAddr(src.RcvAddr)
	dest.SetSndAddr(src.SndAddr)
	dest.SetData([]byte(src.Data))
	dest.SetCode(src.Code)
	dest.SetTxHash(src.TxHash)

	return dest
}

// IsInterfaceNil verifies if underlying object is nil
func (scr *SmartContractResult) IsInterfaceNil() bool {
	return scr == nil
}

// GetValue returns the value of the smart contract result
func (scr *SmartContractResult) GetValue() *big.Int {
	return scr.Value
}

// GetNonce returns the nonce of the smart contract result
func (scr *SmartContractResult) GetNonce() uint64 {
	return scr.Nonce
}

// GetData returns the data of the smart contract result
func (scr *SmartContractResult) GetData() string {
	return scr.Data
}

// GetRecvAddress returns the receiver address from the smart contract result
func (scr *SmartContractResult) GetRecvAddress() []byte {
	return scr.RcvAddr
}

// GetSndAddress returns the sender address from the smart contract result
func (scr *SmartContractResult) GetSndAddress() []byte {
	return scr.SndAddr
}

// GetGasLimit returns the gas limit of the smart contract result
func (scr *SmartContractResult) GetGasLimit() uint64 {
	return scr.GasLimit
}

// GetGasPrice returns the gas price of the smart contract result
func (scr *SmartContractResult) GetGasPrice() uint64 {
	return scr.GasPrice
}

// SetValue sets the value of the smart contract result
func (scr *SmartContractResult) SetValue(value *big.Int) {
	scr.Value = value
}

// SetData sets the data of the smart contract result
func (scr *SmartContractResult) SetData(data string) {
	scr.Data = data
}

// SetRecvAddress sets the receiver address of the smart contract result
func (scr *SmartContractResult) SetRecvAddress(addr []byte) {
	scr.RcvAddr = addr
}

// SetSndAddress sets the sender address of the smart contract result
func (scr *SmartContractResult) SetSndAddress(addr []byte) {
	scr.SndAddr = addr
}

// TrimSlicePtr creates a copy of the provided slice without the excess capacity
func TrimSlicePtr(in []*SmartContractResult) []*SmartContractResult {
	if len(in) == 0 {
		return []*SmartContractResult{}
	}
	ret := make([]*SmartContractResult, len(in))
	copy(ret, in)
	return ret
}
