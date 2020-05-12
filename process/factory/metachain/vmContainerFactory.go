package metachain

import (
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/factory/containers"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/hooks"
	"github.com/ElrondNetwork/elrond-go/vm"
	systemVMFactory "github.com/ElrondNetwork/elrond-go/vm/factory"
	systemVMProcess "github.com/ElrondNetwork/elrond-go/vm/process"
	"github.com/ElrondNetwork/elrond-go/vm/systemSmartContracts"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var _ process.VirtualMachinesContainerFactory = (*vmContainerFactory)(nil)

type vmContainerFactory struct {
	validatorAccountsDB state.AccountsAdapter
	blockChainHookImpl  *hooks.BlockChainHookImpl
	cryptoHook          vmcommon.CryptoHook
	systemContracts     vm.SystemSCContainer
	economics           *economics.EconomicsData
	messageSigVerifier  vm.MessageSignVerifier
	nodesConfigProvider vm.NodesConfigProvider
	gasSchedule         map[string]map[string]uint64
	hasher              hashing.Hasher
	marshalizer         marshal.Marshalizer
	systemSCConfig      *config.SystemSmartContractsConfig
}

// NewVMContainerFactory is responsible for creating a new virtual machine factory object
func NewVMContainerFactory(
	argBlockChainHook hooks.ArgBlockChainHook,
	economics *economics.EconomicsData,
	messageSignVerifier vm.MessageSignVerifier,
	gasSchedule map[string]map[string]uint64,
	nodesConfigProvider vm.NodesConfigProvider,
	hasher hashing.Hasher,
	marshalizer marshal.Marshalizer,
	systemSCConfig *config.SystemSmartContractsConfig,
	validatorAccountsDB state.AccountsAdapter,
) (*vmContainerFactory, error) {
	if economics == nil {
		return nil, process.ErrNilEconomicsData
	}
	if check.IfNil(messageSignVerifier) {
		return nil, process.ErrNilKeyGen
	}
	if check.IfNil(nodesConfigProvider) {
		return nil, process.ErrNilNodesConfigProvider
	}
	if check.IfNil(hasher) {
		return nil, process.ErrNilHasher
	}
	if check.IfNil(marshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if systemSCConfig == nil {
		return nil, process.ErrNilSystemSCConfig
	}
	if check.IfNil(validatorAccountsDB) {
		return nil, vm.ErrNilValidatorAccountsDB
	}

	blockChainHookImpl, err := hooks.NewBlockChainHookImpl(argBlockChainHook)
	if err != nil {
		return nil, err
	}
	cryptoHook := hooks.NewVMCryptoHook()

	return &vmContainerFactory{
		blockChainHookImpl:  blockChainHookImpl,
		cryptoHook:          cryptoHook,
		economics:           economics,
		messageSigVerifier:  messageSignVerifier,
		gasSchedule:         gasSchedule,
		nodesConfigProvider: nodesConfigProvider,
		hasher:              hasher,
		marshalizer:         marshalizer,
		systemSCConfig:      systemSCConfig,
	}, nil
}

// Create sets up all the needed virtual machine returning a container of all the VMs
func (vmf *vmContainerFactory) Create() (process.VirtualMachinesContainer, error) {
	container := containers.NewVirtualMachinesContainer()

	currVm, err := vmf.createSystemVM()
	if err != nil {
		return nil, err
	}

	err = container.Add(factory.SystemVirtualMachine, currVm)
	if err != nil {
		return nil, err
	}

	return container, nil
}

func (vmf *vmContainerFactory) createSystemVM() (vmcommon.VMExecutionHandler, error) {
	atArgumentParser := vmcommon.NewAtArgumentParser()

	systemEI, err := systemSmartContracts.NewVMContext(
		vmf.blockChainHookImpl,
		vmf.cryptoHook,
		atArgumentParser,
		vmf.validatorAccountsDB,
	)
	if err != nil {
		return nil, err
	}

	argsNewSystemScFactory := systemVMFactory.ArgsNewSystemSCFactory{
		SystemEI:            systemEI,
		ValidatorSettings:   vmf.economics,
		SigVerifier:         vmf.messageSigVerifier,
		GasMap:              vmf.gasSchedule,
		NodesConfigProvider: vmf.nodesConfigProvider,
		Hasher:              vmf.hasher,
		Marshalizer:         vmf.marshalizer,
		SystemSCConfig:      vmf.systemSCConfig,
	}
	scFactory, err := systemVMFactory.NewSystemSCFactory(argsNewSystemScFactory)
	if err != nil {
		return nil, err
	}

	vmf.systemContracts, err = scFactory.Create()
	if err != nil {
		return nil, err
	}

	err = systemEI.SetSystemSCContainer(vmf.systemContracts)
	if err != nil {
		return nil, err
	}

	systemVM, err := systemVMProcess.NewSystemVM(systemEI, vmf.systemContracts, factory.SystemVirtualMachine)
	if err != nil {
		return nil, err
	}

	return systemVM, nil
}

// BlockChainHookImpl returns the created blockChainHookImpl
func (vmf *vmContainerFactory) BlockChainHookImpl() process.BlockChainHookHandler {
	return vmf.blockChainHookImpl
}

// SystemSmartContractContainer return the created system smart contracts
func (vmf *vmContainerFactory) SystemSmartContractContainer() vm.SystemSCContainer {
	return vmf.systemContracts
}

// IsInterfaceNil returns true if there is no value under the interface
func (vmf *vmContainerFactory) IsInterfaceNil() bool {
	return vmf == nil
}
