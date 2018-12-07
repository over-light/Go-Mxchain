package keccak

import (
	"golang.org/x/crypto/sha3"
)

var keccakEmptyHash []byte

// Keccak is a sha3-Keccak implementation of the hasher interface.
type Keccak struct {
}

// Compute takes a string, and returns the sha3-Keccak hash of that string
func (k Keccak) Compute(s string) []byte {
	if len(s) == 0 && len(keccakEmptyHash) != 0 {
		return k.EmptyHash()
	}
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(s))
	return h.Sum(nil)
}

// EmptyHash returns the sha3-Keccak hash of the empty string
func (k Keccak) EmptyHash() []byte {
	if len(keccakEmptyHash) == 0 {
		keccakEmptyHash = k.Compute("")
	}
	return keccakEmptyHash
}

// Size returns the size, in number of bytes, of a sha3-Keccak hash
func (Keccak) Size() int {
	return sha3.NewLegacyKeccak256().Size()
}
