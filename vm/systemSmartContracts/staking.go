//go:generate protoc -I=proto -I=$GOPATH/src -I=$GOPATH/src/github.com/gogo/protobuf/protobuf  --gogoslick_out=. staking.proto
package systemSmartContracts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/big"

	"github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/vm"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var log = logger.GetOrCreate("vm/systemsmartcontracts")

const ownerKey = "owner"
const nodesConfigKey = "nodesConfig"

type stakingSC struct {
	eei                      vm.SystemEI
	minStakeValue            *big.Int
	unBondPeriod             uint64
	stakeAccessAddr          []byte
	jailAccessAddr           []byte
	numRoundsWithoutBleed    uint64
	bleedPercentagePerRound  float64
	maximumPercentageToBleed float64
	gasCost                  vm.GasCost
	minNumNodes              int64
}

// ArgsNewStakingSmartContract holds the arguments needed to create a StakingSmartContract
type ArgsNewStakingSmartContract struct {
	MinNumNodes              uint32
	MinStakeValue            *big.Int
	UnBondPeriod             uint64
	Eei                      vm.SystemEI
	StakingAccessAddr        []byte
	JailAccessAddr           []byte
	NumRoundsWithoutBleed    uint64
	BleedPercentagePerRound  float64
	MaximumPercentageToBleed float64
	GasCost                  vm.GasCost
}

// NewStakingSmartContract creates a staking smart contract
func NewStakingSmartContract(
	args ArgsNewStakingSmartContract,
) (*stakingSC, error) {
	if args.MinStakeValue == nil {
		return nil, vm.ErrNilInitialStakeValue
	}
	if args.MinStakeValue.Cmp(big.NewInt(0)) < 1 {
		return nil, vm.ErrNegativeInitialStakeValue
	}
	if check.IfNil(args.Eei) {
		return nil, vm.ErrNilSystemEnvironmentInterface
	}
	if len(args.StakingAccessAddr) < 1 {
		return nil, vm.ErrInvalidStakingAccessAddress
	}
	if len(args.JailAccessAddr) < 1 {
		return nil, vm.ErrInvalidJailAccessAddress
	}

	reg := &stakingSC{
		minStakeValue:            big.NewInt(0).Set(args.MinStakeValue),
		eei:                      args.Eei,
		unBondPeriod:             args.UnBondPeriod,
		stakeAccessAddr:          args.StakingAccessAddr,
		jailAccessAddr:           args.JailAccessAddr,
		numRoundsWithoutBleed:    args.NumRoundsWithoutBleed,
		bleedPercentagePerRound:  args.BleedPercentagePerRound,
		maximumPercentageToBleed: args.MaximumPercentageToBleed,
		gasCost:                  args.GasCost,
		minNumNodes:              int64(args.MinNumNodes),
	}
	return reg, nil
}

// Execute calls one of the functions from the staking smart contract and runs the code according to the input
func (r *stakingSC) Execute(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if CheckIfNil(args) != nil {
		return vmcommon.UserError
	}

	switch args.Function {
	case core.SCDeployInitFunctionName:
		return r.init(args)
	case "stake":
		return r.stake(args, false)
	case "register":
		return r.stake(args, true)
	case "unStake":
		return r.unStake(args)
	case "unBond":
		return r.unBond(args)
	case "slash":
		return r.slash(args)
	case "get":
		return r.get(args)
	case "isStaked":
		return r.isStaked(args)
	case "setStakeValue":
		return r.setStakeValueForCurrentEpoch(args)
	case "jail":
		return r.jail(args)
	case "unJail":
		return r.unJail(args)
	case "changeRewardAddress":
		return r.changeRewardAddress(args)
	case "changeValidatorKeys":
		return r.changeValidatorKey(args)
	}

	return vmcommon.UserError
}

func getPercentageOfValue(value *big.Int, percentage float64) *big.Int {
	x := new(big.Float).SetInt(value)
	y := big.NewFloat(percentage)

	z := new(big.Float).Mul(x, y)

	op := big.NewInt(0)
	result, _ := z.Int(op)

	return result
}

func (r *stakingSC) getConfig() *StakingNodesConfig {
	config := &StakingNodesConfig{
		MinNumNodes: r.minNumNodes,
	}
	configData := r.eei.GetStorage([]byte(nodesConfigKey))
	if len(configData) == 0 {
		return config
	}

	err := json.Unmarshal(configData, config)
	if err != nil {
		log.Warn("unmarshal error on getConfig function, returning baseConfig",
			"error", err.Error(),
		)
		return &StakingNodesConfig{
			MinNumNodes: r.minNumNodes,
		}
	}

	return config
}

func (r *stakingSC) setConfig(config *StakingNodesConfig) {
	configData, err := json.Marshal(config)
	if err != nil {
		log.Warn("marshal error on setConfig function",
			"error", err.Error(),
		)
		return
	}

	r.eei.SetStorage([]byte(nodesConfigKey), configData)
}

func (r *stakingSC) addToStakedNodes() {
	config := r.getConfig()
	config.StakedNodes++
	r.setConfig(config)
}

func (r *stakingSC) removeFromStakedNodes() {
	config := r.getConfig()
	if config.StakedNodes > 0 {
		config.StakedNodes--
	}
	r.setConfig(config)
}

func (r *stakingSC) addToJailedNodes() {
	config := r.getConfig()
	config.JailedNodes++
	r.setConfig(config)
}

func (r *stakingSC) removeFromJailedNodes() {
	config := r.getConfig()
	if config.JailedNodes > 0 {
		config.JailedNodes--
	}
	r.setConfig(config)
}

func (r *stakingSC) numSpareNodes() int64 {
	config := r.getConfig()
	return config.StakedNodes - config.JailedNodes - config.MinNumNodes
}

func (r *stakingSC) canUnStake() bool {
	return r.numSpareNodes() > 0
}

func (r *stakingSC) canUnBond() bool {
	return r.numSpareNodes() >= 0
}

func (r *stakingSC) calculateStakeAfterBleed(startRound uint64, endRound uint64, stake *big.Int) *big.Int {
	if startRound > endRound {
		return stake
	}
	if endRound-startRound < r.numRoundsWithoutBleed {
		return stake
	}

	totalRoundsToBleed := endRound - startRound - r.numRoundsWithoutBleed
	totalPercentageToBleed := float64(totalRoundsToBleed) * r.bleedPercentagePerRound
	totalPercentageToBleed = math.Min(totalPercentageToBleed, r.maximumPercentageToBleed)

	bleedValue := getPercentageOfValue(stake, totalPercentageToBleed)
	stakeAfterBleed := big.NewInt(0).Sub(stake, bleedValue)

	if stakeAfterBleed.Cmp(big.NewInt(0)) < 0 {
		stakeAfterBleed = big.NewInt(0)
	}

	return stakeAfterBleed
}

func (r *stakingSC) changeValidatorKey(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if !bytes.Equal(args.CallerAddr, r.stakeAccessAddr) {
		log.Debug("stake function not allowed to be called by", "address", args.CallerAddr)
		return vmcommon.UserError
	}
	if len(args.Arguments) < 2 {
		return vmcommon.UserError
	}

	oldKey := args.Arguments[0]
	newKey := args.Arguments[1]
	if len(oldKey) != len(newKey) {
		return vmcommon.UserError
	}

	stakedData, err := r.getOrCreateRegisteredData(oldKey)
	if err != nil {
		return vmcommon.UserError
	}
	if len(stakedData.RewardAddress) == 0 {
		// if not registered this is not an error
		return vmcommon.Ok
	}

	r.eei.SetStorage(oldKey, nil)
	err = r.saveStakingData(newKey, stakedData)
	if err != nil {
		return vmcommon.UserError
	}

	return vmcommon.Ok
}

func (r *stakingSC) changeRewardAddress(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if !bytes.Equal(args.CallerAddr, r.stakeAccessAddr) {
		log.Debug("stake function not allowed to be called by", "address", args.CallerAddr)
		return vmcommon.UserError
	}
	if len(args.Arguments) < 2 {
		return vmcommon.UserError
	}

	newRewardAddress := args.Arguments[0]
	if len(newRewardAddress) != len(args.CallerAddr) {
		return vmcommon.UserError
	}

	for i := 1; i < len(args.Arguments); i++ {
		blsKey := args.Arguments[i]
		stakedData, err := r.getOrCreateRegisteredData(blsKey)
		if err != nil {
			return vmcommon.UserError
		}
		if len(stakedData.RewardAddress) == 0 {
			continue
		}

		stakedData.RewardAddress = newRewardAddress
		err = r.saveStakingData(blsKey, stakedData)
		if err != nil {
			return vmcommon.UserError
		}
	}

	return vmcommon.Ok
}

func (r *stakingSC) unJail(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if !bytes.Equal(args.CallerAddr, r.stakeAccessAddr) {
		log.Debug("stake function not allowed to be called by", "address", args.CallerAddr)
		return vmcommon.UserError
	}

	for _, argument := range args.Arguments {
		stakedData, err := r.getOrCreateRegisteredData(argument)
		if err != nil {
			return vmcommon.UserError
		}
		if len(stakedData.RewardAddress) == 0 {
			return vmcommon.UserError
		}

		if stakedData.UnJailedNonce <= stakedData.JailedNonce {
			r.removeFromJailedNodes()
		}

		stakedData.StakeValue = r.calculateStakeAfterBleed(
			stakedData.JailedRound,
			r.eei.BlockChainHook().CurrentRound(),
			stakedData.StakeValue,
		)
		stakedData.JailedRound = math.MaxUint64
		stakedData.UnJailedNonce = r.eei.BlockChainHook().CurrentNonce()

		err = r.saveStakingData(argument, stakedData)
		if err != nil {
			return vmcommon.UserError
		}
	}

	return vmcommon.Ok
}

func (r *stakingSC) getOrCreateRegisteredData(key []byte) (*StakedData, error) {
	registrationData := StakedData{
		RegisterNonce: 0,
		Staked:        false,
		UnStakedNonce: 0,
		UnStakedEpoch: 0,
		RewardAddress: nil,
		StakeValue:    big.NewInt(0),
		JailedRound:   math.MaxUint64,
		UnJailedNonce: 0,
		JailedNonce:   0,
	}

	data := r.eei.GetStorage(key)
	if len(data) > 0 {
		err := json.Unmarshal(data, &registrationData)
		if err != nil {
			log.Debug("unmarshal error on staking SC stake function",
				"error", err.Error(),
			)
			return nil, err
		}
	}

	return &registrationData, nil
}

func (r *stakingSC) saveStakingData(key []byte, stakedData *StakedData) error {
	data, err := json.Marshal(*stakedData)
	if err != nil {
		log.Debug("marshal error on staking SC stake function ",
			"error", err.Error(),
		)
		return err
	}

	r.eei.SetStorage(key, data)
	return nil
}

func (r *stakingSC) jail(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if !bytes.Equal(args.CallerAddr, r.jailAccessAddr) {
		return vmcommon.UserError
	}

	for _, argument := range args.Arguments {
		stakedData, err := r.getOrCreateRegisteredData(argument)
		if err != nil {
			return vmcommon.UserError
		}
		if len(stakedData.RewardAddress) == 0 {
			return vmcommon.UserError
		}

		if stakedData.UnJailedNonce <= stakedData.JailedNonce {
			r.addToJailedNodes()
		}

		stakedData.JailedRound = r.eei.BlockChainHook().CurrentRound()
		stakedData.JailedNonce = r.eei.BlockChainHook().CurrentNonce()
		err = r.saveStakingData(argument, stakedData)
		if err != nil {
			return vmcommon.UserError
		}
	}

	return vmcommon.Ok
}

func (r *stakingSC) get(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if len(args.Arguments) < 1 {
		return vmcommon.UserError
	}

	value := r.eei.GetStorage(args.Arguments[0])
	r.eei.Finish(value)

	return vmcommon.Ok
}

func (r *stakingSC) init(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	ownerAddress := r.eei.GetStorage([]byte(ownerKey))
	if ownerAddress != nil {
		log.Debug("smart contract was already initialized")
		return vmcommon.UserError
	}

	r.eei.SetStorage([]byte(ownerKey), args.CallerAddr)
	r.eei.SetStorage(args.CallerAddr, big.NewInt(0).Bytes())

	epoch := r.eei.BlockChainHook().CurrentEpoch()
	epochData := fmt.Sprintf("epoch_%d", epoch)

	r.eei.SetStorage([]byte(epochData), r.minStakeValue.Bytes())

	config := &StakingNodesConfig{MinNumNodes: r.minNumNodes}
	r.setConfig(config)

	return vmcommon.Ok
}

func (r *stakingSC) setStakeValueForCurrentEpoch(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if !bytes.Equal(args.CallerAddr, r.stakeAccessAddr) {
		log.Debug("stake function not allowed to be called by", "address", args.CallerAddr)
		return vmcommon.UserError
	}

	if len(args.Arguments) < 1 {
		log.Debug("nil arguments to call setStakeValueForCurrentEpoch")
		return vmcommon.UserError
	}

	epoch := r.eei.BlockChainHook().CurrentEpoch()
	epochData := fmt.Sprintf("epoch_%d", epoch)

	inputStakeValue := big.NewInt(0).SetBytes(args.Arguments[0])
	if inputStakeValue.Cmp(r.minStakeValue) < 0 {
		inputStakeValue.Set(r.minStakeValue)
	}

	r.eei.SetStorage([]byte(epochData), inputStakeValue.Bytes())

	return vmcommon.Ok
}

func (r *stakingSC) getStakeValueForCurrentEpoch() *big.Int {
	stakeValue := big.NewInt(0)

	epoch := r.eei.BlockChainHook().CurrentEpoch()
	epochData := fmt.Sprintf("epoch_%d", epoch)

	stakeValueBytes := r.eei.GetStorage([]byte(epochData))
	stakeValue.SetBytes(stakeValueBytes)

	if stakeValue.Cmp(r.minStakeValue) < 0 {
		stakeValue.Set(r.minStakeValue)
	}

	return stakeValue
}

func (r *stakingSC) stake(args *vmcommon.ContractCallInput, onlyRegister bool) vmcommon.ReturnCode {
	if !bytes.Equal(args.CallerAddr, r.stakeAccessAddr) {
		log.Debug("stake function not allowed to be called by", "address", args.CallerAddr)
		return vmcommon.UserError
	}
	if len(args.Arguments) < 2 {
		log.Debug("not enough arguments, needed BLS key and reward address")
		return vmcommon.UserError
	}

	stakeValue := r.getStakeValueForCurrentEpoch()
	registrationData, err := r.getOrCreateRegisteredData(args.Arguments[0])
	if err != nil {
		return vmcommon.UserError
	}

	if registrationData.StakeValue.Cmp(stakeValue) < 0 {
		registrationData.StakeValue.Set(stakeValue)
	}

	if !onlyRegister {
		if !registrationData.Staked {
			r.addToStakedNodes()
		}
		registrationData.Staked = true
	}

	registrationData.RegisterNonce = r.eei.BlockChainHook().CurrentNonce()
	registrationData.RewardAddress = args.Arguments[1]

	err = r.saveStakingData(args.Arguments[0], registrationData)
	if err != nil {
		return vmcommon.UserError
	}

	return vmcommon.Ok
}

func (r *stakingSC) unStake(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if !bytes.Equal(args.CallerAddr, r.stakeAccessAddr) {
		log.Debug("unStake function not allowed to be called by", "address", args.CallerAddr)
		return vmcommon.UserError
	}
	if len(args.Arguments) < 2 {
		log.Debug("not enough arguments, needed BLS key and reward address")
		return vmcommon.UserError
	}

	registrationData, err := r.getOrCreateRegisteredData(args.Arguments[0])
	if err != nil {
		return vmcommon.UserError
	}
	if len(registrationData.RewardAddress) == 0 {
		return vmcommon.UserError
	}

	if !bytes.Equal(args.Arguments[1], registrationData.RewardAddress) {
		log.Debug("unStake possible only from staker",
			"caller", args.CallerAddr,
			"staker", registrationData.RewardAddress,
		)
		return vmcommon.UserError
	}

	if !registrationData.Staked {
		log.Error("unStake is not possible for address with is already unStaked")
		return vmcommon.UserError
	}
	if registrationData.JailedRound != math.MaxUint64 {
		log.Error("unStake is not possible for jailed nodes")
		return vmcommon.UserError
	}
	if !r.canUnStake() {
		log.Error("unStake is not possible as too many left")
		return vmcommon.UserError
	}

	r.removeFromStakedNodes()
	registrationData.Staked = false
	registrationData.UnStakedEpoch = r.eei.BlockChainHook().CurrentEpoch()
	registrationData.UnStakedNonce = r.eei.BlockChainHook().CurrentNonce()

	err = r.saveStakingData(args.Arguments[0], registrationData)
	if err != nil {
		return vmcommon.UserError
	}

	return vmcommon.Ok
}

func (r *stakingSC) unBond(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if !bytes.Equal(args.CallerAddr, r.stakeAccessAddr) {
		log.Debug("unStake function not allowed to be called by", "address", args.CallerAddr)
		return vmcommon.UserError
	}
	if len(args.Arguments) < 1 {
		log.Debug("not enough arguments, needed BLS key and reward address")
		return vmcommon.UserError
	}

	registrationData, err := r.getOrCreateRegisteredData(args.Arguments[0])
	if err != nil {
		return vmcommon.UserError
	}
	if len(registrationData.RewardAddress) == 0 {
		return vmcommon.UserError
	}

	if registrationData.Staked || registrationData.UnStakedNonce <= registrationData.RegisterNonce {
		log.Debug("unBond is not possible for address which is staked or is not in unBond period")
		return vmcommon.UserError
	}

	currentNonce := r.eei.BlockChainHook().CurrentNonce()
	if currentNonce-registrationData.UnStakedNonce < r.unBondPeriod {
		log.Debug("unBond is not possible for address because unBond period did not pass")
		return vmcommon.UserError
	}
	if registrationData.JailedRound != math.MaxUint64 {
		log.Error("unBond is not possible for jailed nodes")
		return vmcommon.UserError
	}
	if !r.canUnBond() || r.eei.IsValidator(args.Arguments[0]) {
		log.Error("unBond is not possible as not enough left")
		return vmcommon.UserError
	}

	r.eei.SetStorage(args.Arguments[0], nil)
	r.eei.Finish(registrationData.StakeValue.Bytes())
	r.eei.Finish(big.NewInt(0).SetUint64(uint64(registrationData.UnStakedEpoch)).Bytes())

	return vmcommon.Ok
}

func (r *stakingSC) slash(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	ownerAddress := r.eei.GetStorage([]byte(ownerKey))
	if !bytes.Equal(ownerAddress, args.CallerAddr) {
		log.Debug("slash function called by not the owners address")
		return vmcommon.UserError
	}

	if len(args.Arguments) != 2 {
		log.Debug("slash function called by wrong number of arguments")
		return vmcommon.UserError
	}

	registrationData, err := r.getOrCreateRegisteredData(args.Arguments[0])
	if err != nil {
		return vmcommon.UserError
	}
	if len(registrationData.RewardAddress) == 0 {
		return vmcommon.UserError
	}
	if !registrationData.Staked {
		log.Debug("cannot slash already unstaked or user not staked")
		return vmcommon.UserError
	}

	if registrationData.UnJailedNonce >= registrationData.JailedNonce {
		r.addToJailedNodes()
	}

	stakedValue := big.NewInt(0).Set(registrationData.StakeValue)
	slashValue := big.NewInt(0).SetBytes(args.Arguments[1])
	registrationData.StakeValue = registrationData.StakeValue.Sub(stakedValue, slashValue)
	registrationData.JailedRound = r.eei.BlockChainHook().CurrentRound()
	registrationData.JailedNonce = r.eei.BlockChainHook().CurrentNonce()

	err = r.saveStakingData(args.Arguments[0], registrationData)
	if err != nil {
		return vmcommon.UserError
	}

	return vmcommon.Ok
}

func (r *stakingSC) isStaked(args *vmcommon.ContractCallInput) vmcommon.ReturnCode {
	if len(args.Arguments) < 1 {
		return vmcommon.UserError
	}

	registrationData, err := r.getOrCreateRegisteredData(args.Arguments[0])
	if err != nil {
		return vmcommon.UserError
	}
	if len(registrationData.RewardAddress) == 0 {
		return vmcommon.UserError
	}

	if registrationData.Staked {
		log.Debug("account already staked, re-staking is invalid")
		return vmcommon.Ok
	}

	return vmcommon.UserError
}

// IsInterfaceNil verifies if the underlying object is nil or not
func (r *stakingSC) IsInterfaceNil() bool {
	return r == nil
}
