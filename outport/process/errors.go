package process

import "errors"

// ErrNilGasConsumedProvider signals that a nil gas consumed provider has been provided
var ErrNilGasConsumedProvider = errors.New("nil gas consumed provider")

// ErrNilNodesCoordinator is raised when a valid validator group selector is expected but nil used
var ErrNilNodesCoordinator = errors.New("validator group selector is nil")

// ErrNilTransactionCoordinator signals that transaction coordinator is nil
var ErrNilTransactionCoordinator = errors.New("transaction coordinator is nil")

// errNilHeaderHandler signal that provided header handler is nil
var errNilHeaderHandler = errors.New("nil header handler")

// errNilBodyHandler signal that provided body handler is nil
var errNilBodyHandler = errors.New("nil body handler")
