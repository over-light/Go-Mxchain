package parsing

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	coreData "github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go-core/data/indexer"
	transactionData "github.com/ElrondNetwork/elrond-go-core/data/transaction"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	crypto "github.com/ElrondNetwork/elrond-go-crypto"
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/genesis"
	"github.com/ElrondNetwork/elrond-go/genesis/data"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// accountsParser hold data for initial accounts decoded data from json file
type accountsParser struct {
	initialAccounts    []*data.InitialAccount
	entireSupply       *big.Int
	minterAddressBytes []byte
	pubkeyConverter    core.PubkeyConverter
	keyGenerator       crypto.KeyGenerator
	hasher             hashing.Hasher
	marshalizer        marshal.Marshalizer
}

// NewAccountsParser creates a new decoded accounts genesis structure from json config file
func NewAccountsParser(
	genesisFilePath string,
	entireSupply *big.Int,
	minterAddress string,
	pubkeyConverter core.PubkeyConverter,
	keyGenerator crypto.KeyGenerator,
	hasher hashing.Hasher,
	marshalizer marshal.Marshalizer,
) (*accountsParser, error) {

	if entireSupply == nil {
		return nil, genesis.ErrNilEntireSupply
	}
	if big.NewInt(0).Cmp(entireSupply) >= 0 {
		return nil, genesis.ErrInvalidEntireSupply
	}
	if check.IfNil(pubkeyConverter) {
		return nil, genesis.ErrNilPubkeyConverter
	}
	if check.IfNil(keyGenerator) {
		return nil, genesis.ErrNilKeyGenerator
	}
	if check.IfNil(hasher) {
		return nil, genesis.ErrNilHasher
	}
	if check.IfNil(marshalizer) {
		return nil, genesis.ErrNilMarshalizer
	}

	initialAccounts := make([]*data.InitialAccount, 0)
	err := core.LoadJsonFile(&initialAccounts, genesisFilePath)
	if err != nil {
		return nil, err
	}

	minterAddressBytes, err := pubkeyConverter.Decode(minterAddress)
	if err != nil {
		return nil, fmt.Errorf("%w for `%s`", genesis.ErrInvalidAddress, minterAddress)
	}

	gp := &accountsParser{
		initialAccounts:    initialAccounts,
		entireSupply:       entireSupply,
		minterAddressBytes: minterAddressBytes,
		pubkeyConverter:    pubkeyConverter,
		keyGenerator:       keyGenerator,
		hasher:             hasher,
		marshalizer:        marshalizer,
	}

	err = gp.process()
	if err != nil {
		return nil, err
	}

	return gp, nil
}

func (ap *accountsParser) process() error {
	totalSupply := big.NewInt(0)
	for _, initialAccount := range ap.initialAccounts {
		err := ap.parseElement(initialAccount)
		if err != nil {
			return err
		}

		err = ap.checkInitialAccount(initialAccount)
		if err != nil {
			return err
		}

		totalSupply.Add(totalSupply, initialAccount.Supply)
	}

	err := ap.checkForDuplicates()
	if err != nil {
		return err
	}

	if totalSupply.Cmp(ap.entireSupply) != 0 {
		return fmt.Errorf("%w for entire supply provided %s, computed %s",
			genesis.ErrEntireSupplyMismatch,
			ap.entireSupply.String(),
			totalSupply.String(),
		)
	}

	return nil
}

func (ap *accountsParser) parseElement(initialAccount *data.InitialAccount) error {
	if len(initialAccount.Address) == 0 {
		return genesis.ErrEmptyAddress
	}
	addressBytes, err := ap.pubkeyConverter.Decode(initialAccount.Address)
	if err != nil {
		return fmt.Errorf("%w for `%s`", genesis.ErrInvalidAddress, initialAccount.Address)
	}

	err = ap.keyGenerator.CheckPublicKeyValid(addressBytes)
	if err != nil {
		return fmt.Errorf("%w for `%s`, error: %s",
			genesis.ErrInvalidPubKey,
			initialAccount.Address,
			err.Error(),
		)
	}

	initialAccount.SetAddressBytes(addressBytes)

	return ap.parseDelegationElement(initialAccount)
}

func (ap *accountsParser) parseDelegationElement(initialAccount *data.InitialAccount) error {
	delegationData := initialAccount.Delegation

	if big.NewInt(0).Cmp(delegationData.Value) == 0 {
		return nil
	}

	if len(delegationData.Address) == 0 {
		return fmt.Errorf("%w for address '%s'",
			genesis.ErrEmptyDelegationAddress, initialAccount.Address)
	}
	addressBytes, err := ap.pubkeyConverter.Decode(delegationData.Address)
	if err != nil {
		return fmt.Errorf("%w for `%s`, address %s, error %s",
			genesis.ErrInvalidDelegationAddress,
			delegationData.Address,
			initialAccount.Address,
			err.Error(),
		)
	}

	delegationData.SetAddressBytes(addressBytes)

	return nil
}

func (ap *accountsParser) checkInitialAccount(initialAccount *data.InitialAccount) error {
	isSmartContract := core.IsSmartContractAddress(initialAccount.AddressBytes())
	if isSmartContract {
		return fmt.Errorf("%w for address %s",
			genesis.ErrAddressIsSmartContract,
			initialAccount.Address,
		)
	}

	if big.NewInt(0).Cmp(initialAccount.Supply) >= 0 {
		return fmt.Errorf("%w for '%s', address %s",
			genesis.ErrInvalidSupply,
			initialAccount.Supply,
			initialAccount.Address,
		)
	}

	if big.NewInt(0).Cmp(initialAccount.Balance) > 0 {
		return fmt.Errorf("%w for '%s', address %s",
			genesis.ErrInvalidBalance,
			initialAccount.Balance,
			initialAccount.Address,
		)
	}

	if big.NewInt(0).Cmp(initialAccount.StakingValue) > 0 {
		return fmt.Errorf("%w for '%s', address %s",
			genesis.ErrInvalidStakingBalance,
			initialAccount.Balance,
			initialAccount.Address,
		)
	}

	if big.NewInt(0).Cmp(initialAccount.Delegation.Value) > 0 {
		return fmt.Errorf("%w for '%s', address %s",
			genesis.ErrInvalidDelegationValue,
			initialAccount.Delegation.Value,
			initialAccount.Address,
		)
	}

	sum := big.NewInt(0)
	sum.Add(sum, initialAccount.Balance)
	sum.Add(sum, initialAccount.StakingValue)
	sum.Add(sum, initialAccount.Delegation.Value)

	isSupplyCorrect := big.NewInt(0).Cmp(initialAccount.Supply) < 0 && initialAccount.Supply.Cmp(sum) == 0
	if !isSupplyCorrect {
		return fmt.Errorf("%w for address %s, provided %s, computed %s",
			genesis.ErrSupplyMismatch,
			initialAccount.Address,
			initialAccount.Supply.String(),
			sum.String(),
		)
	}

	return nil
}

func (ap *accountsParser) checkForDuplicates() error {
	for idx1 := 0; idx1 < len(ap.initialAccounts); idx1++ {
		ia1 := ap.initialAccounts[idx1]
		for idx2 := idx1 + 1; idx2 < len(ap.initialAccounts); idx2++ {
			ia2 := ap.initialAccounts[idx2]
			if ia1.Address == ia2.Address {
				return fmt.Errorf("%w found for '%s'",
					genesis.ErrDuplicateAddress,
					ia1.Address,
				)
			}
		}
	}

	return nil
}

// InitialAccounts return the initial accounts contained by this parser
func (ap *accountsParser) InitialAccounts() []genesis.InitialAccountHandler {
	accounts := make([]genesis.InitialAccountHandler, len(ap.initialAccounts))

	for idx, ia := range ap.initialAccounts {
		accounts[idx] = ia
	}

	return accounts
}

// InitialAccountsSplitOnAddressesShards gets the initial accounts of the nodes split on the addresses's shards
func (ap *accountsParser) InitialAccountsSplitOnAddressesShards(
	shardCoordinator sharding.Coordinator,
) (map[uint32][]genesis.InitialAccountHandler, error) {

	if check.IfNil(shardCoordinator) {
		return nil, genesis.ErrNilShardCoordinator
	}

	var addresses = make(map[uint32][]genesis.InitialAccountHandler)
	for _, ia := range ap.initialAccounts {
		shardID := shardCoordinator.ComputeId(ia.AddressBytes())

		addresses[shardID] = append(addresses[shardID], ia)
	}

	return addresses, nil
}

// GetTotalStakedForDelegationAddress returns the total staked value for a provided delegation address
func (ap *accountsParser) GetTotalStakedForDelegationAddress(delegationAddress string) *big.Int {
	sum := big.NewInt(0)

	for _, in := range ap.initialAccounts {
		if in.Delegation.Address == delegationAddress {
			sum.Add(sum, in.Delegation.Value)
		}
	}

	return sum
}

// GetInitialAccountsForDelegated returns the initial accounts that are set to the provided delegated address
func (ap *accountsParser) GetInitialAccountsForDelegated(addressBytes []byte) []genesis.InitialAccountHandler {
	list := make([]genesis.InitialAccountHandler, 0)
	for _, ia := range ap.initialAccounts {
		if bytes.Equal(ia.Delegation.AddressBytes(), addressBytes) {
			list = append(list, ia)
		}
	}

	return list
}

// IsInterfaceNil returns if underlying object is true
func (ap *accountsParser) IsInterfaceNil() bool {
	return ap == nil
}

func (ap *accountsParser) generateIntraShardMiniBlocks(txsHashesPerShard map[uint32][][]byte) []*block.MiniBlock {
	miniBlocks := make([]*block.MiniBlock, 0)

	for shardId, txsHashes := range txsHashesPerShard {
		miniBlock := &block.MiniBlock{
			TxHashes:        txsHashes,
			ReceiverShardID: shardId,
			SenderShardID:   shardId,
			Type:            block.TxBlock,
		}

		miniBlocks = append(miniBlocks, miniBlock)
	}

	return miniBlocks
}

func (ap *accountsParser) getMintTransaction(ia *data.InitialAccount, nonce uint64, signature []byte) *transactionData.Transaction {
	tx := &transactionData.Transaction{
		Nonce:     nonce,
		SndAddr:   ap.minterAddressBytes,
		Value:     ia.GetSupply(),
		RcvAddr:   ia.AddressBytes(),
		GasPrice:  0,
		GasLimit:  0,
		Signature: signature,
	}

	return tx
}

// GenerateInitialTransactions will generate initial transactions pool and the in shard miniblocks for the generated transactions
func (ap *accountsParser) GenerateInitialTransactions(shardCoordinator sharding.Coordinator) ([]*block.MiniBlock, map[uint32]*indexer.Pool, error) {
	if check.IfNil(shardCoordinator) {
		return nil, nil, genesis.ErrNilShardCoordinator
	}

	txsHashesPerShard := make(map[uint32][][]byte)
	txsPoolPerShard := make(map[uint32]*indexer.Pool)

	mintTxSignatureBytes := []byte(common.GenesisTxSignatureString)

	for i := uint32(0); i < shardCoordinator.NumberOfShards(); i++ {
		txsPoolPerShard[i] = &indexer.Pool{
			Txs: make(map[string]coreData.TransactionHandler),
		}
	}

	var nonce uint64 = 0
	for _, ia := range ap.initialAccounts {
		shardID := shardCoordinator.ComputeId(ia.AddressBytes())

		tx := ap.getMintTransaction(ia, nonce, mintTxSignatureBytes)

		nonce++

		txHash, err := core.CalculateHash(ap.marshalizer, ap.hasher, tx)
		if err != nil {
			return nil, nil, err
		}

		txsPoolPerShard[shardID].Txs[string(txHash)] = tx
		txsHashesPerShard[shardID] = append(txsHashesPerShard[shardID], txHash)
	}

	miniBlocks := ap.generateIntraShardMiniBlocks(txsHashesPerShard)

	return miniBlocks, txsPoolPerShard, nil
}
