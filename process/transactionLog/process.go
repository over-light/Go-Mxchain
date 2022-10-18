package transactionLog

import (
	"encoding/hex"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/transaction"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/storageunit"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var _ process.TransactionLogProcessor = (*txLogProcessor)(nil)

var log = logger.GetOrCreate("process/transactionLog")

// ArgTxLogProcessor defines the arguments needed for transaction logs processor
type ArgTxLogProcessor struct {
	Storer               storage.Storer
	Marshalizer          marshal.Marshalizer
	SaveInStorageEnabled bool
}

type txLogProcessor struct {
	logs        []*data.LogData
	logsIndices map[string]int
	mut         sync.RWMutex
	storer      storage.Storer
	marshalizer marshal.Marshalizer
}

// NewTxLogProcessor creates a transaction log processor capable of parsing logs from the VM
//  and saving them into the injected storage
func NewTxLogProcessor(args ArgTxLogProcessor) (*txLogProcessor, error) {
	storer := args.Storer
	if check.IfNil(storer) && args.SaveInStorageEnabled {
		return nil, process.ErrNilStore
	}

	if !args.SaveInStorageEnabled {
		storer = storageunit.NewNilStorer()
	}

	if check.IfNil(args.Marshalizer) {
		return nil, process.ErrNilMarshalizer
	}

	return &txLogProcessor{
		storer:      storer,
		marshalizer: args.Marshalizer,
		logs:        make([]*data.LogData, 0),
		logsIndices: make(map[string]int),
		mut:         sync.RWMutex{},
	}, nil
}

// GetLog retrieves a log generated by a transaction
func (tlp *txLogProcessor) GetLog(txHash []byte) (data.LogHandler, error) {
	tlp.mut.RLock()
	defer tlp.mut.RUnlock()

	index, ok := tlp.logsIndices[string(txHash)]
	if ok {
		return tlp.logs[index], nil
	}

	txLogBuff, err := tlp.storer.Get(txHash)
	if err != nil {
		return nil, process.ErrLogNotFound
	}

	var txLog transaction.Log
	err = tlp.marshalizer.Unmarshal(&txLog, txLogBuff)
	if err != nil {
		return nil, err
	}

	return &txLog, nil
}

// GetLogFromCache retrieves a log generated by a transaction from ram or from storage
// TODO remove this function and change the interface no longer needed
func (tlp *txLogProcessor) GetLogFromCache(txHash []byte) (*data.LogData, bool) {
	tlp.mut.RLock()
	defer tlp.mut.RUnlock()

	index, ok := tlp.logsIndices[string(txHash)]
	if ok {
		return tlp.logs[index], true
	}

	txLog, err := tlp.GetLog(txHash)
	if err != nil {
		return nil, false
	}

	return &data.LogData{
		LogHandler: txLog,
		TxHash:     string(txHash),
	}, true
}

// GetAllCurrentLogs will return all generated logs for the current block
func (tlp *txLogProcessor) GetAllCurrentLogs() []*data.LogData {
	tlp.mut.RLock()
	logsSlice := make([]*data.LogData, 0, len(tlp.logs))
	logsSlice = append(logsSlice, tlp.logs...)
	tlp.mut.RUnlock()

	return logsSlice
}

// EnableLogToBeSavedInCache will set a flag with true and txLogProcessor will start saving transactions logs also in RAM
func (tlp *txLogProcessor) EnableLogToBeSavedInCache() {
}

// Clean will remove all transaction logs from RAM memory
func (tlp *txLogProcessor) Clean() {
	tlp.mut.Lock()
	tlp.logsIndices = make(map[string]int)
	tlp.logs = make([]*data.LogData, 0)
	tlp.mut.Unlock()
}

// SaveLog takes the VM logs and saves them into the correct format in storage
func (tlp *txLogProcessor) SaveLog(txHash []byte, tx data.TransactionHandler, logEntries []*vmcommon.LogEntry) error {
	if len(txHash) == 0 {
		return process.ErrNilTxHash
	}

	if check.IfNil(tx) {
		return process.ErrNilTransaction
	}

	if len(logEntries) == 0 {
		log.Trace("txLogProcessor.SaveLog()",
			"txHash", hex.EncodeToString(txHash),
			"message", "no logEntries provided",
		)

		return nil
	}

	txLog := &transaction.Log{
		Address: getLogAddressByTx(tx),
	}

	for _, logEntry := range logEntries {
		txLog.Events = append(txLog.Events, &transaction.Event{
			Identifier: logEntry.Identifier,
			Address:    logEntry.Address,
			Topics:     logEntry.Topics,
			Data:       logEntry.Data,
		})
	}

	tlp.saveLogToCache(txHash, txLog)

	buff, err := tlp.marshalizer.Marshal(txLog)
	if err != nil {
		return err
	}

	return tlp.storer.Put(txHash, buff)
}

func (tlp *txLogProcessor) saveLogToCache(txHash []byte, log *transaction.Log) {
	tlp.mut.Lock()
	tlp.logs = append(tlp.logs, &data.LogData{
		TxHash:     string(txHash),
		LogHandler: log,
	})
	tlp.logsIndices[string(txHash)] = len(tlp.logs) - 1
	tlp.mut.Unlock()

}

// For SC deployment transactions, we use the sender address
func getLogAddressByTx(tx data.TransactionHandler) []byte {
	if core.IsEmptyAddress(tx.GetRcvAddr()) {
		return tx.GetSndAddr()
	}

	return tx.GetRcvAddr()
}

// IsInterfaceNil returns true if there is no value under the interface
func (tlp *txLogProcessor) IsInterfaceNil() bool {
	return tlp == nil
}
