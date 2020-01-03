package mock

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/node/heartbeat"
)

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
		gasLimit uint64, data string, signatureHex string) (*transaction.Transaction, error)
	GetTransactionHandler                          func(hash string) (*transaction.Transaction, error)
	SendTransactionHandler                         func(nonce uint64, sender string, receiver string, amount string, code string, signature []byte) (string, error)
	SendBulkTransactionsHandler                    func(txs []*transaction.Transaction) (uint64, error)
	GetAccountHandler                              func(address string) (*state.Account, error)
	GetCurrentPublicKeyHandler                     func() string
	GenerateAndSendBulkTransactionsHandler         func(destination string, value *big.Int, nrTransactions uint64) error
	GenerateAndSendBulkTransactionsOneByOneHandler func(destination string, value *big.Int, nrTransactions uint64) error
	GetHeartbeatsHandler                           func() []heartbeat.PubKeyHeartbeat
	ValidatorStatisticsApiCalled                   func() (map[string]*state.ValidatorApiResponse, error)
}

func (nm *NodeMock) Address() (string, error) {
	return nm.AddressHandler()
}

func (nm *NodeMock) Start() error {
	return nm.StartHandler()
}

func (nm *NodeMock) P2PBootstrap() error {
	return nm.P2PBootstrapHandler()
}

func (nm *NodeMock) IsRunning() bool {
	return nm.IsRunningHandler()
}

func (nm *NodeMock) ConnectToAddresses(addresses []string) error {
	return nm.ConnectToAddressesHandler(addresses)
}

func (nm *NodeMock) StartConsensus() error {
	return nm.StartConsensusHandler()
}

func (nm *NodeMock) GetBalance(address string) (*big.Int, error) {
	return nm.GetBalanceHandler(address)
}

func (nm *NodeMock) GenerateTransaction(sender string, receiver string, amount string, code string) (*transaction.Transaction, error) {
	return nm.GenerateTransactionHandler(sender, receiver, amount, code)
}

func (nm *NodeMock) CreateTransaction(nonce uint64, value string, receiverHex string, senderHex string, gasPrice uint64,
	gasLimit uint64, data string, signatureHex string) (*transaction.Transaction, error) {

	return nm.CreateTransactionHandler(nonce, value, receiverHex, senderHex, gasPrice, gasLimit, data, signatureHex)
}

func (nm *NodeMock) GetTransaction(hash string) (*transaction.Transaction, error) {
	return nm.GetTransactionHandler(hash)
}

func (nm *NodeMock) SendTransaction(
	nonce uint64,
	sender string,
	receiver string,
	value string,
	_ uint64,
	_ uint64,
	transactionData string,
	signature []byte,
) (string, error) {

	return nm.SendTransactionHandler(nonce, sender, receiver, value, transactionData, signature)
}

func (nm *NodeMock) SendBulkTransactions(txs []*transaction.Transaction) (uint64, error) {
	return nm.SendBulkTransactionsHandler(txs)
}

func (nm *NodeMock) GetCurrentPublicKey() string {
	return nm.GetCurrentPublicKeyHandler()
}

func (nm *NodeMock) GenerateAndSendBulkTransactions(receiverHex string, value *big.Int, noOfTx uint64) error {
	return nm.GenerateAndSendBulkTransactionsHandler(receiverHex, value, noOfTx)
}

func (nm *NodeMock) GenerateAndSendBulkTransactionsOneByOne(receiverHex string, value *big.Int, noOfTx uint64) error {
	return nm.GenerateAndSendBulkTransactionsOneByOneHandler(receiverHex, value, noOfTx)
}

func (nm *NodeMock) GetAccount(address string) (*state.Account, error) {
	return nm.GetAccountHandler(address)
}

func (nm *NodeMock) GetHeartbeats() []heartbeat.PubKeyHeartbeat {
	return nm.GetHeartbeatsHandler()
}

func (nm *NodeMock) ValidatorStatisticsApi() (map[string]*state.ValidatorApiResponse, error) {
	return nm.ValidatorStatisticsApiCalled()
}

// IsInterfaceNil returns true if there is no value under the interface
func (nm *NodeMock) IsInterfaceNil() bool {
	return nm == nil
}
