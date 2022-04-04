package helpers

import "errors"

var errEmptyPatternArgument = errors.New("empty pattern argument")

var errInvalidPatternArgument = errors.New("invalid pattern argument. try %x")

var errCannotDecodeEpochNumber = errors.New("cannot decode epoch number")

var errEpochCannotBeZero = errors.New("epoch divisible by cannot be 0")
