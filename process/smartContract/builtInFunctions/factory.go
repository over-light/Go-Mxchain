package builtInFunctions

import (
	"fmt"
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/state"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	vmcommonBuiltInFunctions "github.com/ElrondNetwork/elrond-vm-common/builtInFunctions"
)

var log = logger.GetOrCreate("process/smartcontract/builtInFunctions")

// ArgsCreateBuiltInFunctionContainer defines the argument structure to create new built in function container
type ArgsCreateBuiltInFunctionContainer struct {
	GasSchedule                              core.GasScheduleNotifier
	MapDNSAddresses                          map[string]struct{}
	EnableUserNameChange                     bool
	Marshalizer                              marshal.Marshalizer
	Accounts                                 state.AccountsAdapter
	ShardCoordinator                         sharding.Coordinator
	EpochNotifier                            vmcommon.EpochNotifier
	ESDTMultiTransferEnableEpoch             uint32
	ESDTTransferRoleEnableEpoch              uint32
	GlobalMintBurnDisableEpoch               uint32
	ESDTTransferMetaEnableEpoch              uint32
	OptimizeNFTStoreEnableEpoch              uint32
	CheckCorrectTokenIDEnableEpoch           uint32
	CheckFunctionArgumentEnableEpoch         uint32
	ESDTMetadataContinuousCleanupEnableEpoch uint32
	FixOldTokenLiquidityEnableEpoch          uint32
	MaxNumNodesInTransferRole                uint32
	AutomaticCrawlerAddresses                [][]byte
}

// CreateBuiltInFunctionsFactory creates a container that will hold all the available built in functions
func CreateBuiltInFunctionsFactory(args ArgsCreateBuiltInFunctionContainer) (vmcommon.BuiltInFunctionFactory, error) {
	if check.IfNil(args.GasSchedule) {
		return nil, process.ErrNilGasSchedule
	}
	if check.IfNil(args.Marshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(args.Accounts) {
		return nil, process.ErrNilAccountsAdapter
	}
	if args.MapDNSAddresses == nil {
		return nil, process.ErrNilDnsAddresses
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, process.ErrNilShardCoordinator
	}
	if check.IfNil(args.EpochNotifier) {
		return nil, process.ErrNilEpochNotifier
	}

	vmcommonAccounts, ok := args.Accounts.(vmcommon.AccountsAdapter)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	crawlerAllowedAddress, err := GetAllowedAddress(
		args.ShardCoordinator,
		args.AutomaticCrawlerAddresses)
	if err != nil {
		return nil, err
	}

	log.Debug("createBuiltInFunctionsFactory",
		"shardId", args.ShardCoordinator.SelfId(),
		"crawlerAllowedAddress", crawlerAllowedAddress,
	)

	modifiedArgs := vmcommonBuiltInFunctions.ArgsCreateBuiltInFunctionContainer{
		GasMap:                              args.GasSchedule.LatestGasSchedule(),
		MapDNSAddresses:                     args.MapDNSAddresses,
		EnableUserNameChange:                args.EnableUserNameChange,
		Marshalizer:                         args.Marshalizer,
		Accounts:                            vmcommonAccounts,
		ShardCoordinator:                    args.ShardCoordinator,
		EpochNotifier:                       args.EpochNotifier,
		ESDTNFTImprovementV1ActivationEpoch: args.ESDTMultiTransferEnableEpoch,
		ESDTTransferToMetaEnableEpoch:       args.ESDTTransferMetaEnableEpoch,
		ESDTTransferRoleEnableEpoch:         args.ESDTTransferRoleEnableEpoch,
		GlobalMintBurnDisableEpoch:          args.GlobalMintBurnDisableEpoch,
		SaveNFTToSystemAccountEnableEpoch:   args.OptimizeNFTStoreEnableEpoch,
		CheckCorrectTokenIDEnableEpoch:      args.CheckCorrectTokenIDEnableEpoch,
		CheckFunctionArgumentEnableEpoch:    args.CheckFunctionArgumentEnableEpoch,
		SendESDTMetadataAlwaysEnableEpoch:   args.ESDTMetadataContinuousCleanupEnableEpoch,
		MaxNumOfAddressesForTransferRole:    args.MaxNumNodesInTransferRole,
		FixAsyncCallbackCheckEnableEpoch:    args.ESDTMetadataContinuousCleanupEnableEpoch,
		FixOldTokenLiquidityEnableEpoch:     args.FixOldTokenLiquidityEnableEpoch,
		ConfigAddress:                       crawlerAllowedAddress,
	}

	bContainerFactory, err := vmcommonBuiltInFunctions.NewBuiltInFunctionsCreator(modifiedArgs)
	if err != nil {
		return nil, err
	}

	err = bContainerFactory.CreateBuiltInFunctionContainer()
	if err != nil {
		return nil, err
	}

	args.GasSchedule.RegisterNotifyHandler(bContainerFactory)

	return bContainerFactory, nil
}

func GetAllowedAddress(coordinator sharding.Coordinator, addresses [][]byte) ([]byte, error) {
	if check.IfNil(coordinator) {
		return nil, process.ErrNilShardCoordinator
	}

	if len(addresses) == 0 {
		return nil, fmt.Errorf("%w for shard %d, provided count is %d", process.ErrNilCrawlerAllowedAddress, coordinator.SelfId(), len(addresses))
	}

	if coordinator.SelfId() == core.MetachainShardId {
		return core.SystemAccountAddress, nil
	}

	for _, address := range addresses {
		allowedAddressShardId := coordinator.ComputeId(address)
		if allowedAddressShardId == coordinator.SelfId() {
			return address, nil
		}
	}

	return nil, fmt.Errorf("%w for shard %d, provided count is %d", process.ErrNilCrawlerAllowedAddress, coordinator.SelfId(), len(addresses))
}
