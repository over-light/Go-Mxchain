package trie

import (
	"errors"
)

// ErrInvalidNode is raised when we reach an invalid node
var ErrInvalidNode = errors.New("invalid node")

// ErrNilHasher is raised when the NewTrie() function is called, but a hasher isn't provided
var ErrNilHasher = errors.New("no hasher provided")

// ErrNilMarshalizer is raised when the NewTrie() function is called, but a marshalizer isn't provided
var ErrNilMarshalizer = errors.New("no marshalizer provided")

// ErrNilDatabase is raised when a database operation is called, but no database is provided
var ErrNilDatabase = errors.New("no database provided")

// ErrInvalidEncoding is raised when the encoded information cannot be decoded
var ErrInvalidEncoding = errors.New("cannot decode this invalid encoding")

// ErrValueTooShort is raised when we try to remove something from a value, and the value is too short
var ErrValueTooShort = errors.New("cannot remove bytes from value because value is too short")

// ErrChildPosOutOfRange is raised when the position of a child in a branch node is less than 0 or greater than 16
var ErrChildPosOutOfRange = errors.New("the position of the child is out of range")

// ErrNodeNotFound is raised when we try to get a node that is not present in the trie
var ErrNodeNotFound = errors.New("the node is not present in the trie")

// ErrEmptyNode is raised when we reach an empty node (a node with no children or no value)
var ErrEmptyNode = errors.New("the node is empty")

// ErrNilNode is raised when we reach a nil node
var ErrNilNode = errors.New("the node is nil")

// ErrInvalidLength signals that length of the array is invalid
var ErrInvalidLength = errors.New("invalid array length")

// ErrWrongTypeAssertion signals that wrong type was provided
var ErrWrongTypeAssertion = errors.New("wrong type assertion")

// ErrNilTrie is raised when the trie is nil
var ErrNilTrie = errors.New("the trie is nil")

// ErrNilRequestHandler is raised when the given request handler is nil
var ErrNilRequestHandler = errors.New("the request handler is nil")

// ErrInvalidHash is raised when the given hash is invalid
var ErrInvalidHash = errors.New("the received hash is invalid")

// ErrTimeIsOut signals that time is out
var ErrTimeIsOut = errors.New("time is out")

// ErrHashNotFound signals that the given hash was not found in db or snapshots
var ErrHashNotFound = errors.New("hash not found")

// ErrNilTrieStorage is raised when a nil trie storage is provided
var ErrNilTrieStorage = errors.New("nil trie storage provided")

// ErrNilEvictionWaitingList is raised when a nil eviction waiting list is provided
var ErrNilEvictionWaitingList = errors.New("nil eviction waiting list provided")

// ErrNilPathManager signals that a nil path manager has been provided
var ErrNilPathManager = errors.New("nil path manager")

// ErrInvalidTrieTopic signals that invalid trie topic has been provided
var ErrInvalidTrieTopic = errors.New("invalid trie topic")
