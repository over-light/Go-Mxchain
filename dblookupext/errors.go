package dblookupext

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	elrondErrors "github.com/ElrondNetwork/elrond-go/errors"
)

// ErrNotFoundInStorage signals that an item was not found in storage
var ErrNotFoundInStorage = errors.New("not found in storage")

// Currently, "item not found" storage errors are untyped (thus not distinguishable from others). E.g. see "pruningStorer.go".
// As a workaround, we test the error message for a match.
func isNotFoundInStorageErr(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "not found")
}

var errCannotCastToBlockBody = errors.New("cannot cast to block body")

var errNilESDTSuppliesHandler = errors.New("nil esdt supplies handler")

func newErrCannotSaveEpochByHash(what string, hash []byte, originalErr error) error {
	return fmt.Errorf("cannot save epoch num for [%s] hash [%s]: %w", what, hex.EncodeToString(hash), originalErr)
}

func newErrCannotSaveMiniblockMetadata(hash []byte, originalErr error) error {
	return fmt.Errorf("cannot save miniblock metadata, hash [%s]: %w", hex.EncodeToString(hash), originalErr)
}

func handleErrorLogging(logInstance logger.Logger, logLevel logger.LogLevel, err error, message string, args ...interface{}) {
	if elrondErrors.IsClosingError(err) {
		logLevel = logger.LogDebug
	}

	logInstance.Log(logLevel, message, args...)
}
