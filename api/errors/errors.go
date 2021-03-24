package errors

import (
	"errors"
)

// ErrNilAppContext signals that no context was passed to the routing system
var ErrNilAppContext = errors.New("nil app context")

// ErrInvalidAppContext signals an invalid context passed to the routing system
var ErrInvalidAppContext = errors.New("invalid app context")

// ErrInvalidJSONRequest signals an error in json request formatting
var ErrInvalidJSONRequest = errors.New("invalid json request")

// ErrCouldNotGetAccount signals that a requested account could not be retrieved
var ErrCouldNotGetAccount = errors.New("could not get requested account")

// ErrGetBalance signals an error in getting the balance for an account
var ErrGetBalance = errors.New("get balance error")

// ErrGetUsername signals an error in getting the username for an account
var ErrGetUsername = errors.New("get username error")

// ErrGetValueForKey signals an error in getting the value of a key for an account
var ErrGetValueForKey = errors.New("get value for key error")

// ErrGetKeyValuePairs signals an error in getting the key-value pairs of a key for an account
var ErrGetKeyValuePairs = errors.New("get key-value pairs error")

// ErrGetESDTTokens signals an error in getting esdt tokens for a given address
var ErrGetESDTTokens = errors.New("get esdt tokens for account error")

// ErrGetESDTBalance signals an error in getting esdt balance for given address
var ErrGetESDTBalance = errors.New("get esdt balance for account error")

// ErrEmptyAddress signals an empty address was provided
var ErrEmptyAddress = errors.New("address is empty")

// ErrEmptyKey signals an empty key was provided
var ErrEmptyKey = errors.New("key is empty")

// ErrValidation signals an error in validation
var ErrValidation = errors.New("validation error")

// ErrTxGenerationFailed signals an error generating a transaction
var ErrTxGenerationFailed = errors.New("transaction generation failed")

// ErrValidationEmptyTxHash signals an empty tx hash was provided
var ErrValidationEmptyTxHash = errors.New("TxHash is empty")

// ErrInvalidBlockNonce signals an invalid block nonce was provided
var ErrInvalidBlockNonce = errors.New("invalid block nonce")

// ErrInvalidQueryParameter signals and invalid query parameter was provided
var ErrInvalidQueryParameter = errors.New("invalid query parameter")

// ErrValidationEmptyBlockHash signals an empty block hash was provided
var ErrValidationEmptyBlockHash = errors.New("block hash is empty")

// ErrGetTransaction signals an error happening when trying to fetch a transaction
var ErrGetTransaction = errors.New("getting transaction failed")

// ErrGetBlock signals an error happening when trying to fetch a block
var ErrGetBlock = errors.New("getting block failed")

// ErrQueryError signals a general query error
var ErrQueryError = errors.New("query error")

// ErrGetPidInfo signals that an error occurred while getting peer ID info
var ErrGetPidInfo = errors.New("error getting peer id info")

// ErrTooManyRequests signals that too many requests were simultaneously received
var ErrTooManyRequests = errors.New("too many requests")

// ErrValidationEmptyRootHash signals an empty root hash was provided
var ErrValidationEmptyRootHash = errors.New("rootHash is empty")

// ErrValidationEmptyAddress signals an empty address was provided
var ErrValidationEmptyAddress = errors.New("address is empty")

// ErrGetProof signals an error happening when trying to compute a Merkle proof
var ErrGetProof = errors.New("getting proof failed")

// ErrVerifyProof signals an error happening when trying to verify a Merkle proof
var ErrVerifyProof = errors.New("verifying proof failed")
