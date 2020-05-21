package mock

import (
	"encoding/hex"
	"math/big"

	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/debug"
	"github.com/ElrondNetwork/elrond-go/heartbeat/data"
)

// NodeStub -
type NodeStub struct {
	AddressHandler             func() (string, error)
	ConnectToAddressesHandler  func([]string) error
	StartConsensusHandler      func() error
	GetBalanceHandler          func(address string) (*big.Int, error)
	GenerateTransactionHandler func(sender string, receiver string, amount string, code string) (*transaction.Transaction, error)
	CreateTransactionHandler   func(nonce uint64, value string, receiverHex string, senderHex string, gasPrice uint64,
		gasLimit uint64, data string, signatureHex string) (*transaction.Transaction, []byte, error)
	ValidateTransactionHandler                     func(tx *transaction.Transaction) error
	GetTransactionHandler                          func(hash string) (*transaction.Transaction, error)
	SendBulkTransactionsHandler                    func(txs []*transaction.Transaction) (uint64, error)
	GetAccountHandler                              func(address string) (state.UserAccountHandler, error)
	GetCurrentPublicKeyHandler                     func() string
	GenerateAndSendBulkTransactionsHandler         func(destination string, value *big.Int, nrTransactions uint64) error
	GenerateAndSendBulkTransactionsOneByOneHandler func(destination string, value *big.Int, nrTransactions uint64) error
	GetHeartbeatsHandler                           func() []data.PubKeyHeartbeat
	ValidatorStatisticsApiCalled                   func() (map[string]*state.ValidatorApiResponse, error)
	DirectTriggerCalled                            func() error
	IsSelfTriggerCalled                            func() bool
	GetQueryHandlerCalled                          func(name string) (debug.QueryHandler, error)
	GetTransactionStatusCalled                     func(hash string) (string, error)
	GetValueForKeyCalled                           func(address string, key string) (string, error)
}

// GetValueForKey -
func (ns *NodeStub) GetValueForKey(address string, key string) (string, error) {
	if ns.GetValueForKeyCalled != nil {
		return ns.GetValueForKeyCalled(address, key)
	}

	return "", nil
}

// GetTransactionStatus -
func (ns *NodeStub) GetTransactionStatus(hash string) (string, error) {
	if ns.GetTransactionStatusCalled != nil {
		return ns.GetTransactionStatusCalled(hash)
	}

	return "unknown", nil
}

// EncodeAddressPubkey -
func (ns *NodeStub) EncodeAddressPubkey(pk []byte) (string, error) {
	return hex.EncodeToString(pk), nil
}

// DecodeAddressPubkey -
func (ns *NodeStub) DecodeAddressPubkey(pk string) ([]byte, error) {
	return hex.DecodeString(pk)
}

// StartConsensus -
func (ns *NodeStub) StartConsensus() error {
	return ns.StartConsensusHandler()
}

// GetBalance -
func (ns *NodeStub) GetBalance(address string) (*big.Int, error) {
	return ns.GetBalanceHandler(address)
}

// CreateTransaction -
func (ns *NodeStub) CreateTransaction(nonce uint64, value string, receiverHex string, senderHex string, gasPrice uint64,
	gasLimit uint64, data string, signatureHex string) (*transaction.Transaction, []byte, error) {

	return ns.CreateTransactionHandler(nonce, value, receiverHex, senderHex, gasPrice, gasLimit, data, signatureHex)
}

//ValidateTransaction --
func (ns *NodeStub) ValidateTransaction(tx *transaction.Transaction) error {
	return ns.ValidateTransactionHandler(tx)
}

// GetTransaction -
func (ns *NodeStub) GetTransaction(hash string) (*transaction.Transaction, error) {
	return ns.GetTransactionHandler(hash)
}

// SendBulkTransactions -
func (ns *NodeStub) SendBulkTransactions(txs []*transaction.Transaction) (uint64, error) {
	return ns.SendBulkTransactionsHandler(txs)
}

// GetAccount -
func (ns *NodeStub) GetAccount(address string) (state.UserAccountHandler, error) {
	return ns.GetAccountHandler(address)
}

// GetHeartbeats -
func (ns *NodeStub) GetHeartbeats() []data.PubKeyHeartbeat {
	return ns.GetHeartbeatsHandler()
}

// ValidatorStatisticsApi -
func (ns *NodeStub) ValidatorStatisticsApi() (map[string]*state.ValidatorApiResponse, error) {
	return ns.ValidatorStatisticsApiCalled()
}

// DirectTrigger -
func (ns *NodeStub) DirectTrigger() error {
	return ns.DirectTriggerCalled()
}

// IsSelfTrigger -
func (ns *NodeStub) IsSelfTrigger() bool {
	return ns.IsSelfTriggerCalled()
}

// GetQueryHandler -
func (ns *NodeStub) GetQueryHandler(name string) (debug.QueryHandler, error) {
	if ns.GetQueryHandlerCalled != nil {
		return ns.GetQueryHandlerCalled(name)
	}

	return nil, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (ns *NodeStub) IsInterfaceNil() bool {
	return ns == nil
}
