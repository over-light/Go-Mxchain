package process

import (
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/vm"
)

type genesisProcessors struct {
	txCoordinator process.TransactionCoordinator
	systemSCs     vm.SystemSCContainer
	txProcessor   process.TransactionProcessor
	scProcessor   process.SmartContractProcessor
	scrProcessor  process.SmartContractResultProcessor
	rwdProcessor  process.RewardTransactionProcessor
}
