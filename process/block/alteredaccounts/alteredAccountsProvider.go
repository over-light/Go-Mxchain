package alteredaccounts

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/indexer"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/state"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var (
	log        = logger.GetOrCreate("process/block/alteredaccounts")
	zeroBigInt = big.NewInt(0)
)

type markedAlteredAccountToken struct {
	identifier string
	properties string
	nonce      uint64
}

type markedAlteredAccount struct {
	tokens map[string]*markedAlteredAccountToken
}

// ArgsAlteredAccountsProvider holds the arguments needed for creating a new instance of alteredAccountsProvider
type ArgsAlteredAccountsProvider struct {
	ShardCoordinator       sharding.Coordinator
	AddressConverter       core.PubkeyConverter
	AccountsDB             state.AccountsAdapter
	Marshalizer            marshal.Marshalizer
	EsdtDataStorageHandler vmcommon.ESDTNFTStorageHandler
}

type alteredAccountsProvider struct {
	shardCoordinator       sharding.Coordinator
	addressConverter       core.PubkeyConverter
	accountsDB             state.AccountsAdapter
	marshalizer            marshal.Marshalizer
	tokensProc             *tokensProcessor
	esdtDataStorageHandler vmcommon.ESDTNFTStorageHandler
	mutExtractAccounts     sync.Mutex
}

// NewAlteredAccountsProvider returns a new instance of alteredAccountsProvider
func NewAlteredAccountsProvider(args ArgsAlteredAccountsProvider) (*alteredAccountsProvider, error) {
	if check.IfNil(args.ShardCoordinator) {
		return nil, errNilShardCoordinator
	}
	if check.IfNil(args.AddressConverter) {
		return nil, errNilPubKeyConverter
	}
	if check.IfNil(args.AccountsDB) {
		return nil, errNilAccountsDB
	}
	if check.IfNil(args.Marshalizer) {
		return nil, errNilMarshalizer
	}
	if check.IfNil(args.EsdtDataStorageHandler) {
		return nil, errNilESDTDataStorageHandler
	}

	return &alteredAccountsProvider{
		shardCoordinator:       args.ShardCoordinator,
		addressConverter:       args.AddressConverter,
		accountsDB:             args.AccountsDB,
		marshalizer:            args.Marshalizer,
		tokensProc:             newTokensProcessor(args.ShardCoordinator),
		esdtDataStorageHandler: args.EsdtDataStorageHandler,
	}, nil
}

// ExtractAlteredAccountsFromPool will extract and return altered accounts from the pool
func (aap *alteredAccountsProvider) ExtractAlteredAccountsFromPool(txPool *indexer.Pool) (map[string]*indexer.AlteredAccount, error) {
	aap.mutExtractAccounts.Lock()
	defer aap.mutExtractAccounts.Unlock()

	markedAccounts := make(map[string]*markedAlteredAccount)
	aap.extractAddressesWithBalanceChange(txPool, markedAccounts)
	err := aap.tokensProc.extractESDTAccounts(txPool, markedAccounts)
	if err != nil {
		return nil, err
	}

	return aap.fetchDataForMarkedAccounts(markedAccounts)
}

func (aap *alteredAccountsProvider) fetchDataForMarkedAccounts(markedAccounts map[string]*markedAlteredAccount) (map[string]*indexer.AlteredAccount, error) {
	alteredAccounts := make(map[string]*indexer.AlteredAccount)
	var err error
	for address, markedAccount := range markedAccounts {
		err = aap.processMarkedAccountData(address, markedAccount.tokens, alteredAccounts)
		if err != nil {
			return nil, err
		}
	}

	return alteredAccounts, nil
}

func (aap *alteredAccountsProvider) processMarkedAccountData(
	addressStr string,
	markedAccountTokens map[string]*markedAlteredAccountToken,
	alteredAccounts map[string]*indexer.AlteredAccount,
) error {
	addressBytes := []byte(addressStr)
	encodedAddress := aap.addressConverter.Encode(addressBytes)

	account, err := aap.accountsDB.LoadAccount(addressBytes)
	if err != nil {
		return fmt.Errorf("%w while loading account when computing altered accounts. address: %s", err, encodedAddress)
	}

	userAccount, ok := account.(state.UserAccountHandler)
	if !ok {
		return fmt.Errorf("%w when computing altered accounts. address: %s", errCannotCastToUserAccountHandler, encodedAddress)
	}

	alteredAccounts[encodedAddress] = &indexer.AlteredAccount{
		Address: encodedAddress,
		Balance: userAccount.GetBalance().String(),
		Nonce:   userAccount.GetNonce(),
	}

	for tokenKey, tokenData := range markedAccountTokens {
		err = aap.addTokensDataForMarkedAccount([]byte(tokenKey), encodedAddress, userAccount, tokenData, alteredAccounts)
		if err != nil {
			return fmt.Errorf("%w while fetching token data when computing altered accounts", err)
		}
	}

	return nil
}

func (aap *alteredAccountsProvider) addTokensDataForMarkedAccount(
	tokenKey []byte,
	encodedAddress string,
	userAccount state.UserAccountHandler,
	markedAccountToken *markedAlteredAccountToken,
	alteredAccounts map[string]*indexer.AlteredAccount,
) error {
	nonce := markedAccountToken.nonce
	tokenID := markedAccountToken.identifier

	storageKey := []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier)
	storageKey = append(storageKey, tokenKey...)

	userAccountVmCommon, ok := userAccount.(vmcommon.UserAccountHandler)
	if !ok {
		return fmt.Errorf("%w for address %s", errCannotCastToVmCommonUserAccountHandler, encodedAddress)
	}

	esdtToken, _, err := aap.esdtDataStorageHandler.GetESDTNFTTokenOnDestination(userAccountVmCommon, storageKey, nonce)
	if err != nil {
		return err
	}

	alteredAccount := alteredAccounts[encodedAddress]

	alteredAccount.Tokens = append(alteredAccount.Tokens, &indexer.AccountTokenData{
		Identifier: tokenID,
		Balance:    esdtToken.Value.String(),
		Nonce:      nonce,
		Properties: string(esdtToken.Properties),
		MetaData:   esdtToken.TokenMetaData,
	})

	alteredAccounts[encodedAddress] = alteredAccount

	return nil
}

func (aap *alteredAccountsProvider) extractAddressesWithBalanceChange(
	txPool *indexer.Pool,
	markedAlteredAccounts map[string]*markedAlteredAccount,
) {
	selfShardID := aap.shardCoordinator.SelfId()

	aap.extractAddressesFromTxsHandlers(selfShardID, txPool.Txs, markedAlteredAccounts, process.MoveBalance)
	aap.extractAddressesFromTxsHandlers(selfShardID, txPool.Scrs, markedAlteredAccounts, process.SCInvoking)
	aap.extractAddressesFromTxsHandlers(selfShardID, txPool.Rewards, markedAlteredAccounts, process.RewardTx)
	aap.extractAddressesFromTxsHandlers(selfShardID, txPool.Invalid, markedAlteredAccounts, process.InvalidTransaction)
}

func (aap *alteredAccountsProvider) extractAddressesFromTxsHandlers(
	selfShardID uint32,
	txsHandlers map[string]data.TransactionHandler,
	markedAlteredAccounts map[string]*markedAlteredAccount,
	txType process.TransactionType,
) {
	for _, txHandler := range txsHandlers {
		senderAddress := txHandler.GetSndAddr()
		receiverAddress := txHandler.GetRcvAddr()

		senderShardID := aap.shardCoordinator.ComputeId(senderAddress)
		receiverShardID := aap.shardCoordinator.ComputeId(receiverAddress)

		if senderShardID == selfShardID && len(senderAddress) > 0 {
			aap.addAddressWithBalanceChangeInMap(senderAddress, markedAlteredAccounts)
		}
		if txType != process.InvalidTransaction && receiverShardID == selfShardID && len(receiverAddress) > 0 {
			aap.addAddressWithBalanceChangeInMap(receiverAddress, markedAlteredAccounts)
		}
	}
}

func (aap *alteredAccountsProvider) addAddressWithBalanceChangeInMap(
	address []byte,
	markedAlteredAccounts map[string]*markedAlteredAccount,
) {
	isValidAddress := len(address) == aap.addressConverter.Len()
	if !isValidAddress {
		return
	}

	_, addressAlreadySelected := markedAlteredAccounts[string(address)]
	if addressAlreadySelected {
		return
	}

	markedAlteredAccounts[string(address)] = &markedAlteredAccount{}
}

// IsInterfaceNil returns true if there is no value under the interface
func (aap *alteredAccountsProvider) IsInterfaceNil() bool {
	return aap == nil
}
