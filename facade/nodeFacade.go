package facade

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/api"
	"github.com/ElrondNetwork/elrond-go/api/middleware"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/statistics"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/node/external"
	"github.com/ElrondNetwork/elrond-go/node/heartbeat"
	"github.com/ElrondNetwork/elrond-go/ntp"
	"github.com/ElrondNetwork/elrond-go/process"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

// DefaultRestInterface is the default interface the rest API will start on if not specified
const DefaultRestInterface = "localhost:8080"

// DefaultRestPortOff is the default value that should be passed if it is desired
//  to start the node without a REST endpoint available
const DefaultRestPortOff = "off"

var log = logger.GetOrCreate("facade")

type resetHandler interface {
	Reset()
	IsInterfaceNil() bool
}

// ArgNodeFacade represents the argument for the nodeFacade
type ArgNodeFacade struct {
	Node                   NodeHandler
	ApiResolver            ApiResolver
	RestAPIServerDebugMode bool
	WsAntifloodConfig      config.WebServerAntifloodConfig
	FacadeConfig           config.FacadeConfig
}

// nodeFacade represents a facade for grouping the functionality for the node
type nodeFacade struct {
	node                   NodeHandler
	apiResolver            ApiResolver
	syncer                 ntp.SyncTimer
	tpsBenchmark           *statistics.TpsBenchmark
	config                 config.FacadeConfig
	restAPIServerDebugMode bool
	wsAntifloodConfig      config.WebServerAntifloodConfig
}

// NewNodeFacade creates a new Facade with a NodeWrapper
func NewNodeFacade(arg ArgNodeFacade) (*nodeFacade, error) {
	if check.IfNil(arg.Node) {
		return nil, ErrNilNode
	}
	if check.IfNil(arg.ApiResolver) {
		return nil, ErrNilApiResolver
	}
	if arg.WsAntifloodConfig.SimultaneousRequests == 0 {
		return nil, fmt.Errorf("%w, SimultaneousRequests should not be 0", ErrInvalidValue)
	}
	if arg.WsAntifloodConfig.SameSourceRequests == 0 {
		return nil, fmt.Errorf("%w, SameSourceRequests should not be 0", ErrInvalidValue)
	}
	if arg.WsAntifloodConfig.SameSourceResetIntervalInSec == 0 {
		return nil, fmt.Errorf("%w, SameSourceResetIntervalInSec should not be 0", ErrInvalidValue)
	}

	return &nodeFacade{
		node:                   arg.Node,
		apiResolver:            arg.ApiResolver,
		restAPIServerDebugMode: arg.RestAPIServerDebugMode,
		wsAntifloodConfig:      arg.WsAntifloodConfig,
		config:                 arg.FacadeConfig,
	}, nil
}

// SetSyncer sets the current syncer
func (nf *nodeFacade) SetSyncer(syncer ntp.SyncTimer) {
	nf.syncer = syncer
}

// SetTpsBenchmark sets the tps benchmark handler
func (nf *nodeFacade) SetTpsBenchmark(tpsBenchmark *statistics.TpsBenchmark) {
	nf.tpsBenchmark = tpsBenchmark
}

// TpsBenchmark returns the tps benchmark handler
func (nf *nodeFacade) TpsBenchmark() *statistics.TpsBenchmark {
	return nf.tpsBenchmark
}

// StartNode starts the underlying node
func (nf *nodeFacade) StartNode() error {
	nf.node.Start()
	return nf.node.StartConsensus()
}

// StartBackgroundServices starts all background services needed for the correct functionality of the node
func (nf *nodeFacade) StartBackgroundServices() {
	go nf.startRest()
}

// IsNodeRunning gets if the underlying node is running
func (nf *nodeFacade) IsNodeRunning() bool {
	return nf.node.IsRunning()
}

// RestAPIServerDebugMode return true is debug mode for Rest API is enabled
func (nf *nodeFacade) RestAPIServerDebugMode() bool {
	return nf.restAPIServerDebugMode
}

// RestApiInterface returns the interface on which the rest API should start on, based on the config file provided.
// The API will start on the DefaultRestInterface value unless a correct value is passed or
//  the value is explicitly set to off, in which case it will not start at all
func (nf *nodeFacade) RestApiInterface() string {
	if nf.config.RestApiInterface == "" {
		return DefaultRestInterface
	}

	return nf.config.RestApiInterface
}

func (nf *nodeFacade) startRest() {
	log.Trace("starting REST api server")

	switch nf.RestApiInterface() {
	case DefaultRestPortOff:
		log.Debug("web server is off")
	default:
		log.Debug("creating web server limiters")
		limiters, err := nf.createMiddlewareLimiters()
		if err != nil {
			log.Error("error creating web server limiters",
				"error", err.Error(),
			)
			log.Error("web server is off")
			return
		}

		log.Debug("starting web server",
			"SimultaneousRequests", nf.wsAntifloodConfig.SimultaneousRequests,
			"SameSourceRequests", nf.wsAntifloodConfig.SameSourceRequests,
			"SameSourceResetIntervalInSec", nf.wsAntifloodConfig.SameSourceResetIntervalInSec,
		)

		err = api.Start(nf, limiters...)
		if err != nil {
			log.Error("could not start webserver",
				"error", err.Error(),
			)
		}
	}
}

func (nf *nodeFacade) createMiddlewareLimiters() ([]api.MiddlewareProcessor, error) {
	sourceLimiter, err := middleware.NewSourceThrottler(nf.wsAntifloodConfig.SameSourceRequests)
	if err != nil {
		return nil, err
	}
	go nf.sourceLimiterReset(sourceLimiter)

	globalLimiter, err := middleware.NewGlobalThrottler(nf.wsAntifloodConfig.SimultaneousRequests)
	if err != nil {
		return nil, err
	}

	return []api.MiddlewareProcessor{sourceLimiter, globalLimiter}, nil
}

func (nf *nodeFacade) sourceLimiterReset(reset resetHandler) {
	for {
		time.Sleep(time.Second * time.Duration(nf.wsAntifloodConfig.SameSourceResetIntervalInSec))

		log.Trace("calling reset on WS source limiter")
		reset.Reset()
	}
}

// GetBalance gets the current balance for a specified address
func (nf *nodeFacade) GetBalance(address string) (*big.Int, error) {
	return nf.node.GetBalance(address)
}

// CreateTransaction creates a transaction from all needed fields
func (nf *nodeFacade) CreateTransaction(
	nonce uint64,
	value string,
	receiverHex string,
	senderHex string,
	gasPrice uint64,
	gasLimit uint64,
	txData []byte,
	signatureHex string,
) (*transaction.Transaction, []byte, error) {

	return nf.node.CreateTransaction(nonce, value, receiverHex, senderHex, gasPrice, gasLimit, txData, signatureHex)
}

// ValidateTransaction will validate a transaction
func (nf *nodeFacade) ValidateTransaction(tx *transaction.Transaction) error {
	return nf.node.ValidateTransaction(tx)
}

// ValidatorStatisticsApi will return the statistics for all validators
func (nf *nodeFacade) ValidatorStatisticsApi() (map[string]*state.ValidatorApiResponse, error) {
	return nf.node.ValidatorStatisticsApi()
}

// SendBulkTransactions will send a bulk of transactions on the topic channel
func (nf *nodeFacade) SendBulkTransactions(txs []*transaction.Transaction) (uint64, error) {
	return nf.node.SendBulkTransactions(txs)
}

// GetTransaction gets the transaction with a specified hash
func (nf *nodeFacade) GetTransaction(hash string) (*transaction.Transaction, error) {
	return nf.node.GetTransaction(hash)
}

// ComputeTransactionGasLimit will estimate how many gas a transaction will consume
func (nf *nodeFacade) ComputeTransactionGasLimit(tx *transaction.Transaction) (uint64, error) {
	return nf.apiResolver.ComputeTransactionGasLimit(tx)
}

// GetAccount returns an accountResponse containing information
// about the account correlated with provided address
func (nf *nodeFacade) GetAccount(address string) (state.UserAccountHandler, error) {
	return nf.node.GetAccount(address)
}

// GetHeartbeats returns the heartbeat status for each public key from initial list or later joined to the network
func (nf *nodeFacade) GetHeartbeats() ([]heartbeat.PubKeyHeartbeat, error) {
	hbStatus := nf.node.GetHeartbeats()
	if hbStatus == nil {
		return nil, ErrHeartbeatsNotActive
	}

	return hbStatus, nil
}

// StatusMetrics will return the node's status metrics
func (nf *nodeFacade) StatusMetrics() external.StatusMetricsHandler {
	return nf.apiResolver.StatusMetrics()
}

// ExecuteSCQuery retrieves data from existing SC trie
func (nf *nodeFacade) ExecuteSCQuery(query *process.SCQuery) (*vmcommon.VMOutput, error) {
	return nf.apiResolver.ExecuteSCQuery(query)
}

// PprofEnabled returns if profiling mode should be active or not on the application
func (nf *nodeFacade) PprofEnabled() bool {
	return nf.config.PprofEnabled
}

// TriggerHardfork will trigger a hardfork event
func (nf *nodeFacade) TriggerHardfork() error {
	return nf.node.DirectTrigger()
}

// IsSelfTrigger returns true if the self public key is the same with the registered public key
func (nf *nodeFacade) IsSelfTrigger() bool {
	return nf.node.IsSelfTrigger()
}

// IsInterfaceNil returns true if there is no value under the interface
func (nf *nodeFacade) IsInterfaceNil() bool {
	return nf == nil
}
