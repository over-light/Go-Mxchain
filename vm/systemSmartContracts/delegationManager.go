//go:generate protoc -I=proto -I=$GOPATH/src -I=$GOPATH/src/github.com/ElrondNetwork/protobuf/protobuf  --gogoslick_out=. delegation.proto
package systemSmartContracts

import (
	"bytes"
	"fmt"
	"math"
	"math/big"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/atomic"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/vm"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

const delegationManagmentKey = "delegationManagement"
const delegationContractsList = "delegationContracts"

type delegationManager struct {
	eei                      vm.SystemEI
	delegationMgrSCAddress   []byte
	stakingSCAddr            []byte
	auctionSCAddr            []byte
	gasCost                  vm.GasCost
	marshalizer              marshal.Marshalizer
	delegationMgrEnabled     atomic.Flag
	enableDelegationMgrEpoch uint32
	baseIssuingCost          *big.Int
	minCreationDeposit       *big.Int
}

// ArgsNewDelegationManager defines the arguments to create the delegation manager system smart contract
type ArgsNewDelegationManager struct {
	DelegationMgrSCConfig  config.DelegationManagerSystemSCConfig
	Eei                    vm.SystemEI
	DelegationMgrSCAddress []byte
	StakingSCAddress       []byte
	AuctionSCAddress       []byte
	GasCost                vm.GasCost
	Marshalizer            marshal.Marshalizer
	EpochNotifier          vm.EpochNotifier
}

// NewDelegationManagerSystemSC creates a new delegation manager system SC
func NewDelegationManagerSystemSC(args ArgsNewDelegationManager) (*delegationManager, error) {
	if check.IfNil(args.Eei) {
		return nil, vm.ErrNilSystemEnvironmentInterface
	}
	if len(args.StakingSCAddress) < 1 {
		return nil, fmt.Errorf("%w for staking sc address", vm.ErrInvalidAddress)
	}
	if len(args.AuctionSCAddress) < 1 {
		return nil, fmt.Errorf("%w for auction sc address", vm.ErrInvalidAddress)
	}
	if len(args.DelegationMgrSCAddress) < 1 {
		return nil, fmt.Errorf("%w for delegation sc address", vm.ErrInvalidAddress)
	}
	if check.IfNil(args.Marshalizer) {
		return nil, vm.ErrNilMarshalizer
	}
	if check.IfNil(args.EpochNotifier) {
		return nil, vm.ErrNilEpochNotifier
	}

	baseIssuingCost, okConvert := big.NewInt(0).SetString(args.DelegationMgrSCConfig.BaseIssuingCost, conversionBase)
	if !okConvert || baseIssuingCost.Cmp(zero) < 0 {
		return nil, vm.ErrInvalidBaseIssuingCost
	}

	minCreationDeposit, okConvert := big.NewInt(0).SetString(args.DelegationMgrSCConfig.MinCreationDeposit, conversionBase)
	if !okConvert || baseIssuingCost.Cmp(zero) < 0 {
		return nil, vm.ErrInvalidBaseIssuingCost
	}

	d := &delegationManager{
		eei:                      args.Eei,
		stakingSCAddr:            args.StakingSCAddress,
		auctionSCAddr:            args.AuctionSCAddress,
		delegationMgrSCAddress:   args.DelegationMgrSCAddress,
		gasCost:                  args.GasCost,
		marshalizer:              args.Marshalizer,
		delegationMgrEnabled:     atomic.Flag{},
		enableDelegationMgrEpoch: args.DelegationMgrSCConfig.EnabledEpoch,
		baseIssuingCost:          baseIssuingCost,
		minCreationDeposit:       minCreationDeposit,
	}

	args.EpochNotifier.RegisterNotifyHandler(d)

	return d, nil
}

// Execute  calls one of the functions from the delegation manager contract and runs the code according to the input
func (d *delegationManager) Execute(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if CheckIfNil(args) != nil {
		return vmcommon.UserError
	}

	if !d.delegationMgrEnabled.IsSet() {
		d.eei.AddReturnMessage("delegation manager contract is not enabled")
		return vmcommon.UserError
	}

	switch args.Function {
	case core.SCDeployInitFunctionName:
		return d.init(args)
	case "createNewDelegationContract":
		return d.createNewDelegationContract(args)
	case "getAllContractAddresses":
		return d.getAllContractAddresses(args)
	case "changeBaseIssuingCost":
		return d.changeBaseIssuingCost(args)
	case "changeMinDeposit":
		return d.changeMinDeposit(args)
	}

	d.eei.AddReturnMessage("invalid function to call")
	return vmcommon.UserError
}

func (d *delegationManager) init(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if args.CallValue.Cmp(zero) != 0 {
		d.eei.AddReturnMessage("callValue must be 0")
		return vmcommon.UserError
	}

	managementData := &DelegationManagement{
		NumberOfContract: 0,
		LastAddress:      vm.FirstDelegationSCAddress,
		MinServiceFee:    0,
		MaxServiceFee:    math.MaxUint64,
		BaseIssueingCost: d.baseIssuingCost,
		MinDeposit:       d.minCreationDeposit,
	}
	err := d.saveDelegationManagementData(managementData)
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.UserError
	}

	delegationList := &DelegationContractList{Addresses: make([][]byte, 0)}
	err = d.saveDelegationContractList(delegationList)
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.UserError
	}

	return vmcommon.Ok
}

func (d *delegationManager) createNewDelegationContract(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	err := d.eei.UseGas(d.gasCost.MetaChainSystemSCsCost.DelegationMgrOps)
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.OutOfGas
	}
	if d.callerAlreadyDeployed(args.CallerAddr) {
		d.eei.AddReturnMessage("caller already deployed a delegation sc")
		return vmcommon.UserError
	}

	delegationManagement, err := d.getDelegationManagementData()
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.UserError
	}

	minValue := big.NewInt(0).Add(delegationManagement.MinDeposit, delegationManagement.BaseIssueingCost)
	if args.CallValue.Cmp(minValue) < 0 {
		d.eei.AddReturnMessage("not enough call value")
		return vmcommon.UserError
	}

	delegationList, err := d.getDelegationContractList()
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.UserError
	}

	depositValue := big.NewInt(0).Sub(args.CallValue, delegationManagement.BaseIssueingCost)
	newAddress := createNewAddress(delegationManagement.LastAddress)

	returnCode, err := d.eei.DeploySystemSC(vm.FirstDelegationSCAddress, newAddress, args.CallerAddr, depositValue, args.Arguments)
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.UserError
	}
	if returnCode != vmcommon.Ok {
		return returnCode
	}

	delegationManagement.NumberOfContract += 1
	delegationManagement.LastAddress = newAddress
	delegationList.Addresses = append(delegationList.Addresses, newAddress)

	d.eei.SetStorage(args.CallerAddr, newAddress)
	err = d.saveDelegationManagementData(delegationManagement)
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.UserError
	}

	err = d.saveDelegationContractList(delegationList)
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.UserError
	}

	return vmcommon.Ok
}
func (d *delegationManager) checkConfigChangeInput(args *vmcommon.ContractCallInput) error {
	if args.CallValue.Cmp(zero) != 0 {
		return vm.ErrCallValueMustBeZero
	}
	if len(args.Arguments) != 1 {
		return vm.ErrInvalidNumOfArguments
	}
	if !bytes.Equal(args.CallerAddr, d.delegationMgrSCAddress) {
		return vm.ErrInvalidCaller
	}
	return nil
}

func (d *delegationManager) changeBaseIssuingCost(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	err := d.checkConfigChangeInput(args)
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.UserError
	}

	delegationManagment, err := d.getDelegationManagementData()
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.UserError
	}

	baseIssuingCost := big.NewInt(0).SetBytes(args.Arguments[0])
	if baseIssuingCost.Cmp(zero) < 0 {
		d.eei.AddReturnMessage("invalid base issuing cost")
		return vmcommon.UserError
	}
	delegationManagment.BaseIssueingCost = baseIssuingCost
	err = d.saveDelegationManagementData(delegationManagment)
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.UserError
	}

	return vmcommon.UserError
}

func (d *delegationManager) changeMinDeposit(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	err := d.checkConfigChangeInput(args)
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.UserError
	}

	delegationManagment, err := d.getDelegationManagementData()
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.UserError
	}

	minDeposit := big.NewInt(0).SetBytes(args.Arguments[0])
	if minDeposit.Cmp(zero) < 0 {
		d.eei.AddReturnMessage("invalid base issuing cost")
		return vmcommon.UserError
	}
	delegationManagment.MinDeposit = minDeposit
	err = d.saveDelegationManagementData(delegationManagment)
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.UserError
	}

	return vmcommon.UserError
}

func (d *delegationManager) getAllContractAddresses(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if !bytes.Equal(args.CallerAddr, d.delegationMgrSCAddress) {
		d.eei.AddReturnMessage(vm.ErrInvalidCaller.Error())
		return vmcommon.UserError
	}

	contractList, err := d.getDelegationContractList()
	if err != nil {
		d.eei.AddReturnMessage(err.Error())
		return vmcommon.UserError
	}

	for _, address := range contractList.Addresses {
		d.eei.Finish(address)
	}

	return vmcommon.UserError
}

func createNewAddress(lastAddress []byte) []byte {
	newAddress := make([]byte, 0, len(lastAddress))
	copy(newAddress, lastAddress)

	for i := len(newAddress) - 1; i > 0; i++ {
		if newAddress[i] < 255 {
			newAddress[i]++
			break
		}
	}

	return newAddress
}

func (d *delegationManager) callerAlreadyDeployed(address []byte) bool {
	return len(d.eei.GetStorage(address)) > 0
}

func (d *delegationManager) getDelegationManagementData() (*DelegationManagement, error) {
	marshalledData := d.eei.GetStorage([]byte(delegationManagmentKey))
	if len(marshalledData) == 0 {
		return nil, fmt.Errorf("%w getDelegationManagementData", vm.ErrDataNotFoundUnderKey)
	}

	managementData := &DelegationManagement{}
	err := d.marshalizer.Unmarshal(managementData, marshalledData)
	if err != nil {
		return nil, err
	}
	return managementData, nil
}

func (d *delegationManager) saveDelegationManagementData(managementData *DelegationManagement) error {
	marshalledData, err := d.marshalizer.Marshal(managementData)
	if err != nil {
		return err
	}

	d.eei.SetStorage([]byte(delegationManagmentKey), marshalledData)
	return nil
}

func (d *delegationManager) getDelegationContractList() (*DelegationContractList, error) {
	marshalledData := d.eei.GetStorage([]byte(delegationContractsList))
	if len(marshalledData) == 0 {
		return nil, fmt.Errorf("%w getDelegationContractList", vm.ErrDataNotFoundUnderKey)
	}

	contractList := &DelegationContractList{}
	err := d.marshalizer.Unmarshal(contractList, marshalledData)
	if err != nil {
		return nil, err
	}
	return contractList, nil
}

func (d *delegationManager) saveDelegationContractList(list *DelegationContractList) error {
	marshalledData, err := d.marshalizer.Marshal(list)
	if err != nil {
		return err
	}

	d.eei.SetStorage([]byte(delegationContractsList), marshalledData)
	return nil
}

// EpochConfirmed  is called whenever a new epoch is confirmed
func (d *delegationManager) EpochConfirmed(epoch uint32) {
	d.delegationMgrEnabled.Toggle(epoch >= d.enableDelegationMgrEpoch)
	log.Debug("delegationManager", "enabled", d.delegationMgrEnabled.IsSet())
}

// IsContractEnabled returns true if contract can be used
func (d *delegationManager) IsContractEnabled() bool {
	return d.delegationMgrEnabled.IsSet()
}

// IsInterfaceNil returns true if underlying object is nil
func (d *delegationManager) IsInterfaceNil() bool {
	return d == nil
}
