package mock

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/node/heartbeat"
)

// NodeMock -
type NodeMock struct {
	AddressHandler             func() (string, error)
	StartHandler               func() error
	StopHandler                func() error
	P2PBootstrapHandler        func() error
	IsRunningHandler           func() bool
	ConnectToAddressesHandler  func([]string) error
	StartConsensusHandler      func() error
	GetBalanceHandler          func(address string) (*big.Int, error)
	GenerateTransactionHandler func(sender string, receiver string, amount string, code string) (*transaction.Transaction, error)
	CreateTransactionHandler   func(nonce uint64, value string, receiverHex string, senderHex string, gasPrice uint64,
		gasLimit uint64, data []byte, signatureHex string) (*transaction.Transaction, error)
	GetTransactionHandler                          func(hash string) (*transaction.Transaction, error)
	SendTransactionHandler                         func(nonce uint64, sender string, receiver string, amount string, txData []byte, signature []byte) (string, error)
	SendBulkTransactionsHandler                    func(txs []*transaction.Transaction) (uint64, error)
	GetAccountHandler                              func(address string) (*state.Account, error)
	GetCurrentPublicKeyHandler                     func() string
	GenerateAndSendBulkTransactionsHandler         func(destination string, value *big.Int, nrTransactions uint64) error
	GenerateAndSendBulkTransactionsOneByOneHandler func(destination string, value *big.Int, nrTransactions uint64) error
	GetHeartbeatsHandler                           func() []heartbeat.PubKeyHeartbeat
	ValidatorStatisticsApiCalled                   func() (map[string]*state.ValidatorApiResponse, error)
}

// Address -
func (nm *NodeMock) Address() (string, error) {
	return nm.AddressHandler()
}

// Start -
func (nm *NodeMock) Start() error {
	return nm.StartHandler()
}

// P2PBootstrap -
func (nm *NodeMock) P2PBootstrap() error {
	return nm.P2PBootstrapHandler()
}

// IsRunning -
func (nm *NodeMock) IsRunning() bool {
	return nm.IsRunningHandler()
}

// ConnectToAddresses -
func (nm *NodeMock) ConnectToAddresses(addresses []string) error {
	return nm.ConnectToAddressesHandler(addresses)
}

// StartConsensus -
func (nm *NodeMock) StartConsensus() error {
	return nm.StartConsensusHandler()
}

// GetBalance -
func (nm *NodeMock) GetBalance(address string) (*big.Int, error) {
	return nm.GetBalanceHandler(address)
}

// GenerateTransaction -
func (nm *NodeMock) GenerateTransaction(sender string, receiver string, amount string, code string) (*transaction.Transaction, error) {
	return nm.GenerateTransactionHandler(sender, receiver, amount, code)
}

// CreateTransaction -
func (nm *NodeMock) CreateTransaction(nonce uint64, value string, receiverHex string, senderHex string, gasPrice uint64,
	gasLimit uint64, data []byte, signatureHex string) (*transaction.Transaction, error) {

	return nm.CreateTransactionHandler(nonce, value, receiverHex, senderHex, gasPrice, gasLimit, data, signatureHex)
}

// GetTransaction -
func (nm *NodeMock) GetTransaction(hash string) (*transaction.Transaction, error) {
	return nm.GetTransactionHandler(hash)
}

// SendTransaction -
func (nm *NodeMock) SendTransaction(nonce uint64, sender string, receiver string, value string, gasPrice uint64, gasLimit uint64, transactionData []byte, signature []byte) (string, error) {
	return nm.SendTransactionHandler(nonce, sender, receiver, value, transactionData, signature)
}

// SendBulkTransactions -
func (nm *NodeMock) SendBulkTransactions(txs []*transaction.Transaction) (uint64, error) {
	return nm.SendBulkTransactionsHandler(txs)
}

// GetCurrentPublicKey -
func (nm *NodeMock) GetCurrentPublicKey() string {
	return nm.GetCurrentPublicKeyHandler()
}

// GenerateAndSendBulkTransactions -
func (nm *NodeMock) GenerateAndSendBulkTransactions(receiverHex string, value *big.Int, noOfTx uint64) error {
	return nm.GenerateAndSendBulkTransactionsHandler(receiverHex, value, noOfTx)
}

// GenerateAndSendBulkTransactionsOneByOne -
func (nm *NodeMock) GenerateAndSendBulkTransactionsOneByOne(receiverHex string, value *big.Int, noOfTx uint64) error {
	return nm.GenerateAndSendBulkTransactionsOneByOneHandler(receiverHex, value, noOfTx)
}

// GetAccount -
func (nm *NodeMock) GetAccount(address string) (*state.Account, error) {
	return nm.GetAccountHandler(address)
}

// GetHeartbeats -
func (nm *NodeMock) GetHeartbeats() []heartbeat.PubKeyHeartbeat {
	return nm.GetHeartbeatsHandler()
}

// ValidatorStatisticsApi -
func (nm *NodeMock) ValidatorStatisticsApi() (map[string]*state.ValidatorApiResponse, error) {
	return nm.ValidatorStatisticsApiCalled()
}

// IsInterfaceNil returns true if there is no value under the interface
func (nm *NodeMock) IsInterfaceNil() bool {
	if nm == nil {
		return true
	}
	return false
}
