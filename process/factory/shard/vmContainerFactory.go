package shard

import (
	arwen "github.com/ElrondNetwork/arwen-wasm-vm/arwen/host"
	ipcCommon "github.com/ElrondNetwork/arwen-wasm-vm/ipc/common"
	ipcLogger "github.com/ElrondNetwork/arwen-wasm-vm/ipc/logger"
	ipcMarshaling "github.com/ElrondNetwork/arwen-wasm-vm/ipc/marshaling"
	ipcNodePart "github.com/ElrondNetwork/arwen-wasm-vm/ipc/nodepart"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/logger"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/process/factory/containers"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/hooks"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var logVMContainerFactory = logger.GetOrCreate("vmContainerFactory")

type vmContainerFactory struct {
	config             config.VirtualMachineConfig
	blockChainHookImpl *hooks.BlockChainHookImpl
	cryptoHook         vmcommon.CryptoHook
	blockGasLimit      uint64
	gasSchedule        map[string]map[string]uint64
}

// NewVMContainerFactory is responsible for creating a new virtual machine factory object
func NewVMContainerFactory(
	config config.VirtualMachineConfig,
	blockGasLimit uint64,
	gasSchedule map[string]map[string]uint64,
	argBlockChainHook hooks.ArgBlockChainHook,
) (*vmContainerFactory, error) {
	if gasSchedule == nil {
		return nil, process.ErrNilGasSchedule
	}

	blockChainHookImpl, err := hooks.NewBlockChainHookImpl(argBlockChainHook)
	if err != nil {
		return nil, err
	}
	cryptoHook := hooks.NewVMCryptoHook()

	return &vmContainerFactory{
		config:             config,
		blockChainHookImpl: blockChainHookImpl,
		cryptoHook:         cryptoHook,
		blockGasLimit:      blockGasLimit,
		gasSchedule:        gasSchedule,
	}, nil
}

// Create sets up all the needed virtual machine returning a container of all the VMs
func (vmf *vmContainerFactory) Create() (process.VirtualMachinesContainer, error) {
	container := containers.NewVirtualMachinesContainer()

	currVm, err := vmf.createArwenVM()
	if err != nil {
		return nil, err
	}

	err = container.Add(factory.ArwenVirtualMachine, currVm)
	if err != nil {
		return nil, err
	}

	return container, nil
}

func (vmf *vmContainerFactory) createArwenVM() (vmcommon.VMExecutionHandler, error) {
	if vmf.config.OutOfProcessEnabled {
		return vmf.createOutOfProcessArwenVM()
	}

	return vmf.createInProcessArwenVM()
}

func (vmf *vmContainerFactory) createOutOfProcessArwenVM() (vmcommon.VMExecutionHandler, error) {
	logVMContainerFactory.Info("createOutOfProcessArwenVM")
	outOfProcessConfig := vmf.config.OutOfProcessConfig

	arwenVM, err := ipcNodePart.NewArwenDriver(
		logVMContainerFactory,
		vmf.blockChainHookImpl,
		ipcCommon.ArwenArguments{
			VMHostArguments: ipcCommon.VMHostArguments{
				VMType:        factory.ArwenVirtualMachine,
				BlockGasLimit: vmf.blockGasLimit,
				GasSchedule:   vmf.gasSchedule,
			},
			LogLevel:            ipcLogger.LogLevel(outOfProcessConfig.LogLevel),
			LogsMarshalizer:     ipcMarshaling.MarshalizerKind(outOfProcessConfig.LogsMarshalizer),
			MessagesMarshalizer: ipcMarshaling.MarshalizerKind(outOfProcessConfig.MessagesMarshalizer),
		},
		ipcNodePart.Config{MaxLoopTime: outOfProcessConfig.MaxLoopTime},
	)
	return arwenVM, err
}

func (vmf *vmContainerFactory) createInProcessArwenVM() (vmcommon.VMExecutionHandler, error) {
	logVMContainerFactory.Info("createInProcessArwenVM")
	return arwen.NewArwenVM(vmf.blockChainHookImpl, vmf.cryptoHook, factory.ArwenVirtualMachine, vmf.blockGasLimit, vmf.gasSchedule)
}

// BlockChainHookImpl returns the created blockChainHookImpl
func (vmf *vmContainerFactory) BlockChainHookImpl() process.BlockChainHookHandler {
	return vmf.blockChainHookImpl
}

// IsInterfaceNil returns true if there is no value under the interface
func (vmf *vmContainerFactory) IsInterfaceNil() bool {
	return vmf == nil
}
