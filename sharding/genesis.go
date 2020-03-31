package sharding

import (
	"encoding/hex"
	"math/big"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/state"
)

var log = logger.GetOrCreate("sharding")

// InitialBalance holds data from json and decoded data from genesis process
type InitialBalance struct {
	PubKey  string `json:"pubkey"`
	Balance string `json:"balance"`
	pubKey  []byte
	balance *big.Int
}

// Genesis hold data for decoded data from json file
type Genesis struct {
	InitialBalances []*InitialBalance `json:"initialBalances"`
}

// NewGenesisConfig creates a new decoded genesis structure from json config file
func NewGenesisConfig(genesisFilePath string) (*Genesis, error) {
	genesis := &Genesis{}

	err := core.LoadJsonFile(genesis, genesisFilePath)
	if err != nil {
		return nil, err
	}

	err = genesis.processConfig()
	if err != nil {
		return nil, err
	}

	return genesis, nil
}

func (g *Genesis) processConfig() error {
	var err error
	var ok bool

	for i := 0; i < len(g.InitialBalances); i++ {
		g.InitialBalances[i].pubKey, err = hex.DecodeString(g.InitialBalances[i].PubKey)

		// decoder treats empty string as correct, it is not allowed to have empty string as public key
		if g.InitialBalances[i].PubKey == "" || err != nil {
			g.InitialBalances[i].pubKey = nil
			return ErrCouldNotParsePubKey
		}

		g.InitialBalances[i].balance, ok = new(big.Int).SetString(g.InitialBalances[i].Balance, 10)
		if !ok {
			log.Debug("error decoding balance for public key - setting to 0",
				"balance", g.InitialBalances[i].Balance,
				"pubkey", g.InitialBalances[i].PubKey)
			g.InitialBalances[i].balance = big.NewInt(0)
		}
	}

	return nil
}

// InitialNodesBalances - gets the initial balances of the nodes
func (g *Genesis) InitialNodesBalances(shardCoordinator Coordinator, adrConv state.AddressConverter) (map[string]*big.Int, error) {
	if shardCoordinator == nil || shardCoordinator.IsInterfaceNil() {
		return nil, ErrNilShardCoordinator
	}
	if adrConv == nil || adrConv.IsInterfaceNil() {
		return nil, ErrNilAddressConverter
	}

	var balances = make(map[string]*big.Int)
	for _, in := range g.InitialBalances {
		address, err := adrConv.CreateAddressFromPublicKeyBytes(in.pubKey)
		if err != nil {
			return nil, err
		}
		addressShard := shardCoordinator.ComputeId(address)
		if addressShard == shardCoordinator.SelfId() {
			balances[string(in.pubKey)] = in.balance
		}
	}

	return balances, nil
}
