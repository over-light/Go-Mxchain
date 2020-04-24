package checking_test

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/genesis"
	"github.com/ElrondNetwork/elrond-go/genesis/checking"
	"github.com/ElrondNetwork/elrond-go/genesis/mock"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/stretchr/testify/assert"
)

func createEmptyInitialAccount() *genesis.InitialAccount {
	return &genesis.InitialAccount{
		Address:      "",
		Supply:       big.NewInt(0),
		Balance:      big.NewInt(0),
		StakingValue: big.NewInt(0),
		Delegation: &genesis.DelegationData{
			Address: "",
			Value:   big.NewInt(0),
		},
	}
}

//------- NewNodesSetupChecker

func TestNewNodesSetupChecker_NilGenesisParserShouldErr(t *testing.T) {
	t.Parallel()

	nsc, err := checking.NewNodesSetupChecker(
		nil,
		big.NewInt(0),
		mock.NewPubkeyConverterMock(32),
	)

	assert.True(t, check.IfNil(nsc))
	assert.Equal(t, genesis.ErrNilGenesisParser, err)
}

func TestNewNodesSetupChecker_NilInitialNodePriceShouldErr(t *testing.T) {
	t.Parallel()

	nsc, err := checking.NewNodesSetupChecker(
		&mock.GenesisParserStub{},
		nil,
		mock.NewPubkeyConverterMock(32),
	)

	assert.True(t, check.IfNil(nsc))
	assert.Equal(t, genesis.ErrNilInitialNodePrice, err)
}

func TestNewNodesSetupChecker_InvalidInitialNodePriceShouldErr(t *testing.T) {
	t.Parallel()

	nsc, err := checking.NewNodesSetupChecker(
		&mock.GenesisParserStub{},
		big.NewInt(-1),
		mock.NewPubkeyConverterMock(32),
	)

	assert.True(t, check.IfNil(nsc))
	assert.True(t, errors.Is(err, genesis.ErrInvalidInitialNodePrice))
}

func TestNewNodesSetupChecker_NilValidatorPubkeyConverterShouldErr(t *testing.T) {
	t.Parallel()

	nsc, err := checking.NewNodesSetupChecker(
		&mock.GenesisParserStub{},
		big.NewInt(0),
		nil,
	)

	assert.True(t, check.IfNil(nsc))
	assert.Equal(t, genesis.ErrNilPubkeyConverter, err)
}

func TestNewNodesSetupChecker_ShouldWork(t *testing.T) {
	t.Parallel()

	nsc, err := checking.NewNodesSetupChecker(
		&mock.GenesisParserStub{},
		big.NewInt(0),
		mock.NewPubkeyConverterMock(32),
	)

	assert.False(t, check.IfNil(nsc))
	assert.Nil(t, err)
}

//------- Check

func TestNewNodeSetupChecker_CheckNotStakedShouldErr(t *testing.T) {
	t.Parallel()

	ia := createEmptyInitialAccount()
	ia.SetAddressBytes([]byte("staked address"))

	nsc, _ := checking.NewNodesSetupChecker(
		&mock.GenesisParserStub{
			InitialAccountsCalled: func() []*genesis.InitialAccount {
				return []*genesis.InitialAccount{ia}
			},
		},
		big.NewInt(0),
		mock.NewPubkeyConverterMock(32),
	)

	err := nsc.Check(
		[]sharding.GenesisNodeInfoHandler{
			&mock.GenesisNodeInfoHandlerMock{
				AssignedShardValue: 0,
				AddressBytesValue:  []byte("not-staked-address"),
				PubKeyBytesValue:   []byte("pubkey"),
			},
		},
	)

	assert.True(t, errors.Is(err, genesis.ErrNodeNotStaked))
}

func TestNewNodeSetupChecker_CheckNotEnoughStakedShouldErr(t *testing.T) {
	t.Parallel()

	nodePrice := big.NewInt(32)
	ia := createEmptyInitialAccount()
	ia.StakingValue = big.NewInt(0).Set(nodePrice)
	ia.SetAddressBytes([]byte("staked address"))

	nsc, _ := checking.NewNodesSetupChecker(
		&mock.GenesisParserStub{
			InitialAccountsCalled: func() []*genesis.InitialAccount {
				return []*genesis.InitialAccount{ia}
			},
		},
		big.NewInt(nodePrice.Int64()+1),
		mock.NewPubkeyConverterMock(32),
	)

	err := nsc.Check(
		[]sharding.GenesisNodeInfoHandler{
			&mock.GenesisNodeInfoHandlerMock{
				AssignedShardValue: 0,
				AddressBytesValue:  []byte("staked address"),
				PubKeyBytesValue:   []byte("pubkey"),
			},
		},
	)

	assert.True(t, errors.Is(err, genesis.ErrStakingValueIsNotEnough))
}

func TestNewNodeSetupChecker_CheckTooMuchStakedShouldErr(t *testing.T) {
	t.Parallel()

	nodePrice := big.NewInt(32)
	ia := createEmptyInitialAccount()
	ia.StakingValue = big.NewInt(0).Set(nodePrice)
	ia.SetAddressBytes([]byte("staked address"))

	nsc, _ := checking.NewNodesSetupChecker(
		&mock.GenesisParserStub{
			InitialAccountsCalled: func() []*genesis.InitialAccount {
				return []*genesis.InitialAccount{ia}
			},
		},
		big.NewInt(nodePrice.Int64()-1),
		mock.NewPubkeyConverterMock(32),
	)

	err := nsc.Check(
		[]sharding.GenesisNodeInfoHandler{
			&mock.GenesisNodeInfoHandlerMock{
				AssignedShardValue: 0,
				AddressBytesValue:  []byte("staked address"),
				PubKeyBytesValue:   []byte("pubkey"),
			},
		},
	)

	assert.True(t, errors.Is(err, genesis.ErrInvalidStakingBalance))
}

func TestNewNodeSetupChecker_CheckNotEnoughDelegatedShouldErr(t *testing.T) {
	t.Parallel()

	nodePrice := big.NewInt(32)
	ia := createEmptyInitialAccount()
	ia.Delegation.SetAddressBytes([]byte("delegated address"))
	ia.Delegation.Value = big.NewInt(0).Set(nodePrice)

	nsc, _ := checking.NewNodesSetupChecker(
		&mock.GenesisParserStub{
			InitialAccountsCalled: func() []*genesis.InitialAccount {
				return []*genesis.InitialAccount{ia}
			},
		},
		big.NewInt(nodePrice.Int64()+1),
		mock.NewPubkeyConverterMock(32),
	)

	err := nsc.Check(
		[]sharding.GenesisNodeInfoHandler{
			&mock.GenesisNodeInfoHandlerMock{
				AssignedShardValue: 0,
				AddressBytesValue:  []byte("delegated address"),
				PubKeyBytesValue:   []byte("pubkey"),
			},
		},
	)

	assert.True(t, errors.Is(err, genesis.ErrDelegationValueIsNotEnough))
}

func TestNewNodeSetupChecker_CheckTooMuchDelegatedShouldErr(t *testing.T) {
	t.Parallel()

	nodePrice := big.NewInt(32)
	ia := createEmptyInitialAccount()
	ia.Delegation.SetAddressBytes([]byte("delegated address"))
	ia.Delegation.Value = big.NewInt(0).Set(nodePrice)

	nsc, _ := checking.NewNodesSetupChecker(
		&mock.GenesisParserStub{
			InitialAccountsCalled: func() []*genesis.InitialAccount {
				return []*genesis.InitialAccount{ia}
			},
		},
		big.NewInt(nodePrice.Int64()-1),
		mock.NewPubkeyConverterMock(32),
	)

	err := nsc.Check(
		[]sharding.GenesisNodeInfoHandler{
			&mock.GenesisNodeInfoHandlerMock{
				AssignedShardValue: 0,
				AddressBytesValue:  []byte("delegated address"),
				PubKeyBytesValue:   []byte("pubkey"),
			},
		},
	)

	assert.True(t, errors.Is(err, genesis.ErrInvalidDelegationValue))
}

func TestNewNodeSetupChecker_CheckStakedAndDelegatedShouldWork(t *testing.T) {
	t.Parallel()

	nodePrice := big.NewInt(32)
	iaStaked := createEmptyInitialAccount()
	iaStaked.StakingValue = big.NewInt(0).Set(nodePrice)
	iaStaked.SetAddressBytes([]byte("staked address"))

	iaDelegated := createEmptyInitialAccount()
	iaDelegated.Delegation.Value = big.NewInt(0).Set(nodePrice)
	iaDelegated.Delegation.SetAddressBytes([]byte("delegated address"))

	nsc, _ := checking.NewNodesSetupChecker(
		&mock.GenesisParserStub{
			InitialAccountsCalled: func() []*genesis.InitialAccount {
				return []*genesis.InitialAccount{iaStaked, iaDelegated}
			},
		},
		nodePrice,
		mock.NewPubkeyConverterMock(32),
	)

	err := nsc.Check(
		[]sharding.GenesisNodeInfoHandler{
			&mock.GenesisNodeInfoHandlerMock{
				AssignedShardValue: 0,
				AddressBytesValue:  []byte("delegated address"),
				PubKeyBytesValue:   []byte("pubkey"),
			},
			&mock.GenesisNodeInfoHandlerMock{
				AssignedShardValue: 0,
				AddressBytesValue:  []byte("staked address"),
				PubKeyBytesValue:   []byte("pubkey"),
			},
		},
	)

	assert.Nil(t, err)
	//the following 2 asserts assure that the original values did not changed
	assert.Equal(t, nodePrice, iaStaked.StakingValue)
	assert.Equal(t, nodePrice, iaDelegated.Delegation.Value)
}
