package p2p

import (
	"errors"
)

// ErrNilContext signals that a nil context was provided
var ErrNilContext = errors.New("nil context")

// ErrInvalidPort signals that an invalid port was provided
var ErrInvalidPort = errors.New("invalid port provided")

// ErrNilP2PprivateKey signals that a nil P2P private key has been provided
var ErrNilP2PprivateKey = errors.New("nil P2P private key")

// ErrNilMockNet signals that a nil mocknet was provided. Should occur only in testing!!!
var ErrNilMockNet = errors.New("nil mocknet provided")

// ErrNilTopic signals that a nil topic has been provided
var ErrNilTopic = errors.New("nil topic")

// ErrTopicAlreadyExists signals that a topic already exists
var ErrTopicAlreadyExists = errors.New("topic already exists")

// ErrTopicValidatorOperationNotSupported signals that an unsupported validator operation occurred
var ErrTopicValidatorOperationNotSupported = errors.New("topic validator operation is not supported")

// ErrNilChannelLoadBalancer signals that a nil data throttler object has been provided
var ErrNilChannelLoadBalancer = errors.New("nil channel load balancer object")

// ErrChannelAlreadyExists signals that the channel is already defined (and used)
var ErrChannelAlreadyExists = errors.New("channel already exists")

// ErrChannelDoesNotExist signals that a requested channel does not exist
var ErrChannelDoesNotExist = errors.New("channel does not exist")

// ErrChannelCanNotBeDeleted signals that a channel can not be deleted (might be the default channel)
var ErrChannelCanNotBeDeleted = errors.New("channel can not be deleted")

// ErrNilMessage signals that a nil message has been received
var ErrNilMessage = errors.New("nil message")

// ErrEmptyTopicList signals that a message with empty topic ids has been received
var ErrEmptyTopicList = errors.New("empty topicIDs")

// ErrAlreadySeenMessage signals that the message has already been seen
var ErrAlreadySeenMessage = errors.New("already seen this message")

// ErrNilDirectSendMessageHandler signals that the message handler for new message has not been wired
var ErrNilDirectSendMessageHandler = errors.New("nil direct sender message handler")

// ErrPeerNotDirectlyConnected signals that the peer is not directly connected to self
var ErrPeerNotDirectlyConnected = errors.New("peer is not directly connected")

// ErrNilHost signals that a nil host has been provided
var ErrNilHost = errors.New("nil host")

// ErrNilValidator signals that a validator hasn't been set for the required topic
var ErrNilValidator = errors.New("no validator has been set for this topic")

// ErrPeerDiscoveryProcessAlreadyStarted signals that a peer discovery is already turned on
var ErrPeerDiscoveryProcessAlreadyStarted = errors.New("peer discovery is already turned on")

// ErrNilContextProvider signals that a nil context applier has been provided
var ErrNilContextProvider = errors.New("nil context provider")

// ErrWrongContextApplier signals that a wrong type of context applier has been provided
var ErrWrongContextApplier = errors.New("wrong type of ContextApplier")

// ErrNilPeerDiscoverer signals that a nil peer dicoverer has been provided
var ErrNilPeerDiscoverer = errors.New("nil peer discoverer")

// ErrNegativeOrZeroPeersRefreshInterval signals that a negative or zero peers refresh interval has been provided
var ErrNegativeOrZeroPeersRefreshInterval = errors.New("negative or zero peers refresh interval")

// ErrMessageTooLarge signals that the message provided is too large
var ErrMessageTooLarge = errors.New("buffer too large")

// ErrNilFetchPeersOnTopicHandler signals that a nil handler was provided
var ErrNilFetchPeersOnTopicHandler = errors.New("nil fetch peers on topic handler")

// ErrInvalidDurationProvided signals that an invalid time.Duration has been provided
var ErrInvalidDurationProvided = errors.New("invalid time.Duration provided")

// ErrTooManyGoroutines is raised when the number of goroutines has exceeded a threshold
var ErrTooManyGoroutines = errors.New(" number of goroutines exceeded")

// ErrInvalidValue signals that an invalid value has been provided
var ErrInvalidValue = errors.New("invalid value")

// ErrNilSharder signals that the provided sharder is nil
var ErrNilSharder = errors.New("nil sharder")

// ErrNilPeerShardResolver signals that the peer shard resolver provided is nil
var ErrNilPeerShardResolver = errors.New("nil PeerShardResolver")

// ErrWatchdogAlreadyStarted signals that a peer discovery watchdog is already started
var ErrWatchdogAlreadyStarted = errors.New("peer discovery watchdog is already started")

// ErrWatchdogNotStarted signals that a peer discovery watchdog is not started
var ErrWatchdogNotStarted = errors.New("peer discovery watchdog is not started")

// ErrContextDone signals that an operation was canceled by context done
var ErrContextDone = errors.New("context done")

// ErrTimeout signals a timeout
var ErrTimeout = errors.New("timeout")

// ErrNilNetworkShardingCollector signals that the network sharding collector provided is nil
var ErrNilNetworkShardingCollector = errors.New("nil network sharding collector")

// ErrNilSignerVerifier signals that the signer-verifier instance provided is nil
var ErrNilSignerVerifier = errors.New("nil signer-verifier")

// ErrNilMarshalizer signals that an operation has been attempted to or with a nil marshalizer implementation
var ErrNilMarshalizer = errors.New("nil marshalizer")
