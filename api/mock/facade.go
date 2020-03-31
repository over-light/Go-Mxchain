package mock

import (
	"errors"
	"math/big"

	"github.com/ElrondNetwork/elrond-go/core/statistics"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/node/external"
	"github.com/ElrondNetwork/elrond-go/node/heartbeat"
	"github.com/ElrondNetwork/elrond-go/process"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

// Facade is the mock implementation of a node router handler
type Facade struct {
	Running                           bool
	ShouldErrorStart                  bool
	ShouldErrorStop                   bool
	TpsBenchmarkHandler               func() *statistics.TpsBenchmark
	GetHeartbeatsHandler              func() ([]heartbeat.PubKeyHeartbeat, error)
	BalanceHandler                    func(string) (*big.Int, error)
	GetAccountHandler                 func(address string) (state.UserAccountHandler, error)
	GenerateTransactionHandler        func(sender string, receiver string, value *big.Int, code string) (*transaction.Transaction, error)
	GetTransactionHandler             func(hash string) (*transaction.Transaction, error)
	CreateTransactionHandler          func(nonce uint64, value string, receiverHex string, senderHex string, gasPrice uint64, gasLimit uint64, data []byte, signatureHex string) (*transaction.Transaction, []byte, error)
	ValidateTransactionHandler        func(tx *transaction.Transaction) error
	SendBulkTransactionsHandler       func(txs []*transaction.Transaction) (uint64, error)
	ExecuteSCQueryHandler             func(query *process.SCQuery) (*vmcommon.VMOutput, error)
	StatusMetricsHandler              func() external.StatusMetricsHandler
	ValidatorStatisticsHandler        func() (map[string]*state.ValidatorApiResponse, error)
	ComputeTransactionGasLimitHandler func(tx *transaction.Transaction) (uint64, error)
}

// RestApiInterface -
func (f *Facade) RestApiInterface() string {
	return "localhost:8080"
}

// RestAPIServerDebugMode -
func (f *Facade) RestAPIServerDebugMode() bool {
	return false
}

// PprofEnabled -
func (f *Facade) PprofEnabled() bool {
	return false
}

// IsNodeRunning is the mock implementation of a handler's IsNodeRunning method
func (f *Facade) IsNodeRunning() bool {
	return f.Running
}

// StartNode is the mock implementation of a handler's StartNode method
func (f *Facade) StartNode() error {
	if f.ShouldErrorStart {
		return errors.New("error")
	}
	return nil
}

// TpsBenchmark is the mock implementation for retreiving the TpsBenchmark
func (f *Facade) TpsBenchmark() *statistics.TpsBenchmark {
	if f.TpsBenchmarkHandler != nil {
		return f.TpsBenchmarkHandler()
	}
	return nil
}

// StopNode is the mock implementation of a handler's StopNode method
func (f *Facade) StopNode() error {
	if f.ShouldErrorStop {
		return errors.New("error")
	}
	f.Running = false
	return nil
}

// GetHeartbeats returns the slice of heartbeat info
func (f *Facade) GetHeartbeats() ([]heartbeat.PubKeyHeartbeat, error) {
	return f.GetHeartbeatsHandler()
}

// GetBalance is the mock implementation of a handler's GetBalance method
func (f *Facade) GetBalance(address string) (*big.Int, error) {
	return f.BalanceHandler(address)
}

// GetAccount is the mock implementation of a handler's GetAccount method
func (f *Facade) GetAccount(address string) (state.UserAccountHandler, error) {
	return f.GetAccountHandler(address)
}

// GenerateTransaction is the mock implementation of a handler's GenerateTransaction method
func (f *Facade) GenerateTransaction(sender string, receiver string, value *big.Int,
	code string) (*transaction.Transaction, error) {
	return f.GenerateTransactionHandler(sender, receiver, value, code)
}

// CreateTransaction is  mock implementation of a handler's CreateTransaction method
func (f *Facade) CreateTransaction(
	nonce uint64,
	value string,
	receiverHex string,
	senderHex string,
	gasPrice uint64,
	gasLimit uint64,
	data []byte,
	signatureHex string,
) (*transaction.Transaction, []byte, error) {
	return f.CreateTransactionHandler(nonce, value, receiverHex, senderHex, gasPrice, gasLimit, data, signatureHex)
}

// GetTransaction is the mock implementation of a handler's GetTransaction method
func (f *Facade) GetTransaction(hash string) (*transaction.Transaction, error) {
	return f.GetTransactionHandler(hash)
}

// SendBulkTransactions is the mock implementation of a handler's SendBulkTransactions method
func (f *Facade) SendBulkTransactions(txs []*transaction.Transaction) (uint64, error) {
	return f.SendBulkTransactionsHandler(txs)
}

//ValidateTransaction --
func (f *Facade) ValidateTransaction(tx *transaction.Transaction) error {
	return f.ValidateTransactionHandler(tx)
}

// ValidatorStatisticsApi is the mock implementation of a handler's ValidatorStatisticsApi method
func (f *Facade) ValidatorStatisticsApi() (map[string]*state.ValidatorApiResponse, error) {
	return f.ValidatorStatisticsHandler()
}

// ExecuteSCQuery is a mock implementation.
func (f *Facade) ExecuteSCQuery(query *process.SCQuery) (*vmcommon.VMOutput, error) {
	return f.ExecuteSCQueryHandler(query)
}

// StatusMetrics is the mock implementation for the StatusMetrics
func (f *Facade) StatusMetrics() external.StatusMetricsHandler {
	return f.StatusMetricsHandler()
}

// ComputeTransactionGasLimit --
func (f *Facade) ComputeTransactionGasLimit(tx *transaction.Transaction) (uint64, error) {
	return f.ComputeTransactionGasLimitHandler(tx)
}

// IsInterfaceNil returns true if there is no value under the interface
func (f *Facade) IsInterfaceNil() bool {
	return f == nil
}

// WrongFacade is a struct that can be used as a wrong implementation of the node router handler
type WrongFacade struct {
}
