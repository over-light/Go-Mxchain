package genesis

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/update"
)

// MetaBlockFileName is the constant which defines the export/import filename for metaBlock
const MetaBlockFileName = "metaBlock"

// TransactionsFileName is the constant which defines the export/import filename for transactions
const TransactionsFileName = "transactions"

// MiniBlocksFileName is the constant which defines the export/import filename for miniBlocks
const MiniBlocksFileName = "miniBlocks"

// TrieFileName is the constant which defines the export/import filename for tries
const TrieFileName = "trie"

// Type identifies the type of the export / import
type Type uint8

const (
	// Unknown is an export/import type which is not known by the system
	Unknown Type = iota
	// Transaction is the export/import type for pending transactions
	Transaction
	// SmartContractResult is the export/import type for pending smart contract results
	SmartContractResult
	// RewardTransaction is the export/import type for pending reward transaction
	RewardTransaction
	// MiniBlock is the export/import type for pending miniBlock
	MiniBlock
	// Header is the export/import type for pending headers
	Header
	// MetaHeader is the export/import type for pending meta headers
	MetaHeader
	// RootHash is the export/import type for byte array which has to be treated as rootHash
	RootHash
	// UserAccount is the export/import type for an account of type user account
	UserAccount
	// PeerAccount is the export/import type for peer account
	ValidatorAccount
	// DataTrie identifies the data trie kept under a specific account
	DataTrie
)

// SupportedAccountTypes is the list to describe the possible account types in the accounts DB
var SupportedAccountTypes = []Type{UserAccount, ValidatorAccount, DataTrie}

// atSep is a separator used for export and import to decipher needed types
const atSep = "@"

// NewObject creates an object according to the given type
func NewObject(objType Type) (interface{}, error) {
	switch objType {
	case Transaction:
		return &transaction.Transaction{}, nil
	case SmartContractResult:
		return &smartContractResult.SmartContractResult{}, nil
	case RewardTransaction:
		return &rewardTx.RewardTx{}, nil
	case MiniBlock:
		return &block.MiniBlock{}, nil
	case Header:
		return &block.Header{}, nil
	case MetaHeader:
		return &block.MetaBlock{}, nil
	case RootHash:
		return make([]byte, 0), nil
	}
	return nil, update.ErrUnknownType
}

// NewEmptyAccount returns a new account according to the given type
func NewEmptyAccount(accType Type) (state.AccountHandler, error) {
	switch accType {
	case UserAccount:
		return state.NewEmptyUserAccount(), nil
	case ValidatorAccount:
		return state.NewEmptyPeerAccount(), nil
	case DataTrie:
		return nil, nil
	}
	return nil, update.ErrUnknownType
}

// GetTrieTypeAndShId returns the type and shard Id for a given account according to the saved key
func GetTrieTypeAndShId(key string) (Type, uint32, error) {
	splitString := strings.Split(key, atSep)
	if len(splitString) < 3 {
		return UserAccount, 0, update.ErrUnknownType
	}

	accTypeInt64, err := strconv.ParseInt(splitString[1], 10, 0)
	if err != nil {
		return UserAccount, 0, err
	}
	accType := getAccountType(int(accTypeInt64))

	shId, err := strconv.ParseInt(splitString[1], 10, 0)
	if err != nil {
		return UserAccount, 0, err
	}
	return accType, uint32(shId), nil
}

func getTransactionKeyTypeAndHash(splitString []string) (Type, []byte, error) {
	if len(splitString) < 2 {
		return Unknown, nil, update.ErrUnknownType
	}

	switch splitString[0] {
	case "nrm":
		return Transaction, []byte(splitString[1]), nil
	case "scr":
		return SmartContractResult, []byte(splitString[1]), nil
	case "rwd":
		return RewardTransaction, []byte(splitString[1]), nil
	}

	return Unknown, nil, update.ErrUnknownType
}

func getAccountType(intType int) Type {
	accType := UserAccount
	for currType := range SupportedAccountTypes {
		if currType == intType {
			accType = Type(currType)
			break
		}
	}
	return accType
}

func getTrieTypeAndHash(splitString []string) (Type, []byte, error) {
	if len(splitString) < 3 {
		return Unknown, nil, update.ErrUnknownType
	}

	accTypeInt64, err := strconv.ParseInt(splitString[1], 10, 0)
	if err != nil {
		return Unknown, nil, err
	}
	accType := getAccountType(int(accTypeInt64))

	return accType, []byte(splitString[2]), nil
}

// GetKeyTypeAndHash returns the type of the key by splitting it up and deciphering it
func GetKeyTypeAndHash(key string) (Type, []byte, error) {
	splitString := strings.Split(key, atSep)

	if len(splitString) < 2 {
		return Unknown, nil, update.ErrUnknownType
	}

	switch splitString[0] {
	case "meta":
		return getHeaderTypeAndHash(splitString)
	case "mb":
		return getMbTypeAndHash(splitString)
	case "tx":
		return getTransactionKeyTypeAndHash(splitString[1:])
	case "tr":
		return getTrieTypeAndHash(splitString[1:])
	case "rt":
		return RootHash, []byte(key), nil
	}

	return Unknown, nil, update.ErrUnknownType
}

func getHeaderTypeAndHash(splitString []string) (Type, []byte, error) {
	if len(splitString) < 3 {
		return Unknown, nil, update.ErrUnknownType
	}

	hash, err := hex.DecodeString(splitString[2])
	if err != nil {
		return Unknown, nil, err
	}

	return MetaHeader, hash, nil
}

func getMbTypeAndHash(splitString []string) (Type, []byte, error) {
	hash, err := hex.DecodeString(splitString[1])
	if err != nil {
		return Unknown, nil, err
	}

	return MiniBlock, hash, nil
}

// CreateVersionKey creates a version key from the given metaBlock
func CreateVersionKey(meta *block.MetaBlock, hash []byte) string {
	return "meta" + atSep + string(meta.ChainID) + atSep + hex.EncodeToString(hash)
}

// CreateAccountKey creates a key for an account according to its type, shard ID and address
func CreateAccountKey(accType Type, shId uint32, address string) string {
	key := CreateTrieIdentifier(shId, accType)
	return key + atSep + address
}

// CreateRootHashKey creates a key of type roothash for a given trie identifier
func CreateRootHashKey(trieIdentifier string) string {
	return "rt" + atSep + trieIdentifier
}

// CreateTrieIdentifier creates a trie identifier according to trie type and shard id
func CreateTrieIdentifier(shID uint32, accountType Type) string {
	return fmt.Sprint("tr", atSep, shID, atSep, accountType)
}

// AddRootHashToIdentifier adds the roothash to the current identifier
func AddRootHashToIdentifier(identifier string, hash string) string {
	return identifier + atSep + hash
}

// CreateMiniBlockKey returns a miniblock key
func CreateMiniBlockKey(key string) string {
	return "mb" + atSep + hex.EncodeToString([]byte(key))
}

// CreateTransactionKey create a transaction key according to its type
func CreateTransactionKey(key string, tx data.TransactionHandler) string {
	switch tx.(type) {
	case *transaction.Transaction:
		return "tx" + atSep + "nrm" + atSep + key
	case *smartContractResult.SmartContractResult:
		return "tx" + atSep + "scr" + atSep + key
	case *rewardTx.RewardTx:
		return "tx" + atSep + "rwd" + atSep + key
	default:
		return "tx" + atSep + "ukw" + key
	}
}
