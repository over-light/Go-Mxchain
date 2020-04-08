package facade

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/node/external"
	"github.com/ElrondNetwork/elrond-go/node/heartbeat"
	"github.com/ElrondNetwork/elrond-go/process"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

//NodeWrapper contains all functions that a node should contain.
type NodeWrapper interface {
	// Start will create a new messenger and and set up the Node state as running
	Start() error

	// P2PBootstrap starts the peer discovery process and peer connection filtering
	P2PBootstrap() error

	//IsRunning returns if the underlying node is running
	IsRunning() bool

	// StartConsensus will start the consesus service for the current node
	StartConsensus() error

	//GetBalance returns the balance for a specific address
	GetBalance(address string) (*big.Int, error)

	//CreateTransaction will return a transaction from all needed fields
	CreateTransaction(nonce uint64, value string, receiverHex string, senderHex string, gasPrice uint64,
		gasLimit uint64, data []byte, signatureHex string) (*transaction.Transaction, []byte, error)

	//ValidateTransaction will validate a transaction
	ValidateTransaction(tx *transaction.Transaction) error

	//SendBulkTransactions will send a bulk of transactions on the 'send transactions pipe' channel
	SendBulkTransactions(txs []*transaction.Transaction) (uint64, error)

	//GetTransaction gets the transaction
	GetTransaction(hash string) (*transaction.Transaction, error)

	// GetAccount returns an accountResponse containing information
	//  about the account corelated with provided address
	GetAccount(address string) (state.UserAccountHandler, error)

	// GetHeartbeats returns the heartbeat status for each public key defined in genesis.json
	GetHeartbeats() []heartbeat.PubKeyHeartbeat

	// IsInterfaceNil returns true if there is no value under the interface
	IsInterfaceNil() bool

	// ValidatorStatisticsApi return the statistics for all the validators
	ValidatorStatisticsApi() (map[string]*state.ValidatorApiResponse, error)

	EncodeAddressPubkey(pk []byte) (string, error)
	DecodeAddressPubkey(pk string) ([]byte, error)
}

// ApiResolver defines a structure capable of resolving REST API requests
type ApiResolver interface {
	ExecuteSCQuery(query *process.SCQuery) (*vmcommon.VMOutput, error)
	ComputeTransactionGasLimit(tx *transaction.Transaction) (uint64, error)
	StatusMetrics() external.StatusMetricsHandler
	IsInterfaceNil() bool
}
