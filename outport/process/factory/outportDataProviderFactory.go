package factory

import (
	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-go/outport"
	"github.com/multiversx/mx-chain-go/outport/process"
	"github.com/multiversx/mx-chain-go/outport/process/alteredaccounts"
	"github.com/multiversx/mx-chain-go/outport/process/disabled"
	"github.com/multiversx/mx-chain-go/outport/process/transactionsfee"
	processTxs "github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/sharding"
	"github.com/multiversx/mx-chain-go/sharding/nodesCoordinator"
	"github.com/multiversx/mx-chain-go/state"
	"github.com/multiversx/mx-chain-go/storage"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

// ArgOutportDataProviderFactory holds the arguments needed for creating a new instance of outport.DataProviderOutport
type ArgOutportDataProviderFactory struct {
	IsImportDBMode         bool
	HasDrivers             bool
	AddressConverter       core.PubkeyConverter
	AccountsDB             state.AccountsAdapter
	Marshaller             marshal.Marshalizer
	EsdtDataStorageHandler vmcommon.ESDTNFTStorageHandler
	TransactionsStorer     storage.Storer
	ShardCoordinator       sharding.Coordinator
	TxCoordinator          processTxs.TransactionCoordinator
	NodesCoordinator       nodesCoordinator.NodesCoordinator
	GasConsumedProvider    process.GasConsumedProvider
	EconomicsData          process.EconomicsDataHandler
}

// CreateOutportDataProvider will create a new instance of outport.DataProviderOutport
func CreateOutportDataProvider(arg ArgOutportDataProviderFactory) (outport.DataProviderOutport, error) {
	if !arg.HasDrivers {
		return disabled.NewDisabledOutportDataProvider(), nil
	}

	err := checkArgOutportDataProviderFactory(arg)
	if err != nil {
		return nil, err
	}

	alteredAccountsProvider, err := alteredaccounts.NewAlteredAccountsProvider(alteredaccounts.ArgsAlteredAccountsProvider{
		ShardCoordinator:       arg.ShardCoordinator,
		AddressConverter:       arg.AddressConverter,
		AccountsDB:             arg.AccountsDB,
		EsdtDataStorageHandler: arg.EsdtDataStorageHandler,
	})
	if err != nil {
		return nil, err
	}

	transactionsFeeProc, err := transactionsfee.NewTransactionsFeeProcessor(transactionsfee.ArgTransactionsFeeProcessor{
		Marshaller:         arg.Marshaller,
		TransactionsStorer: arg.TransactionsStorer,
		ShardCoordinator:   arg.ShardCoordinator,
		TxFeeCalculator:    arg.EconomicsData,
	})
	if err != nil {
		return nil, err
	}

	return process.NewOutportDataProvider(process.ArgOutportDataProvider{
		IsImportDBMode:           arg.IsImportDBMode,
		ShardCoordinator:         arg.ShardCoordinator,
		AlteredAccountsProvider:  alteredAccountsProvider,
		TransactionsFeeProcessor: transactionsFeeProc,
		TxCoordinator:            arg.TxCoordinator,
		NodesCoordinator:         arg.NodesCoordinator,
		GasConsumedProvider:      arg.GasConsumedProvider,
		EconomicsData:            arg.EconomicsData,
	})
}
