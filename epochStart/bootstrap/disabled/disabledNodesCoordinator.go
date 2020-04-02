package disabled

import (
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// nodesCoordinator -
type nodesCoordinator struct {
}

// NewNodesCoordinator returns a new instance of nodesCoordinator
func NewNodesCoordinator() *nodesCoordinator {
	return &nodesCoordinator{}
}

// GetAllLeavingValidatorsPublicKeys -
func (n *nodesCoordinator) GetAllLeavingValidatorsPublicKeys(_ uint32) ([][]byte, error) {
	return nil, nil
}

// ValidatorsWeights -
func (n *nodesCoordinator) ValidatorsWeights(validators []sharding.Validator) ([]uint32, error) {
	return make([]uint32, len(validators)), nil
}

// ComputeLeaving -
func (n *nodesCoordinator) ComputeLeaving(_ []sharding.Validator) []sharding.Validator {
	return nil
}

// GetValidatorsIndexes -
func (n *nodesCoordinator) GetValidatorsIndexes(_ []string, _ uint32) ([]uint64, error) {
	return nil, nil
}

// GetAllEligibleValidatorsPublicKeys -
func (n *nodesCoordinator) GetAllEligibleValidatorsPublicKeys(_ uint32) (map[uint32][][]byte, error) {
	return nil, nil
}

// GetAllWaitingValidatorsPublicKeys -
func (n *nodesCoordinator) GetAllWaitingValidatorsPublicKeys(_ uint32) (map[uint32][][]byte, error) {
	return nil, nil
}

// GetConsensusValidatorsPublicKeys -
func (n *nodesCoordinator) GetConsensusValidatorsPublicKeys(_ []byte, _ uint64, _ uint32, _ uint32) ([]string, error) {
	return nil, nil
}

// GetOwnPublicKey -
func (n *nodesCoordinator) GetOwnPublicKey() []byte {
	return nil
}

// ComputeConsensusGroup -
func (n *nodesCoordinator) ComputeConsensusGroup(_ []byte, _ uint64, _ uint32, _ uint32) (validatorsGroup []sharding.Validator, err error) {
	return nil, nil
}

// GetValidatorWithPublicKey -
func (n *nodesCoordinator) GetValidatorWithPublicKey(_ []byte, _ uint32) (validator sharding.Validator, shardId uint32, err error) {
	return nil, 0, nil
}

// LoadState -
func (n *nodesCoordinator) LoadState(_ []byte) error {
	return nil
}

// GetSavedStateKey -
func (n *nodesCoordinator) GetSavedStateKey() []byte {
	return nil
}

// ShardIdForEpoch -
func (n *nodesCoordinator) ShardIdForEpoch(_ uint32) (uint32, error) {
	return 0, nil
}

// GetConsensusWhitelistedNodes -
func (n *nodesCoordinator) GetConsensusWhitelistedNodes(_ uint32) (map[string]struct{}, error) {
	return nil, nil
}

// ConsensusGroupSize -
func (n *nodesCoordinator) ConsensusGroupSize(uint32) int {
	return 0
}

// GetNumTotalEligible -
func (n *nodesCoordinator) GetNumTotalEligible() uint64 {
	return 0
}

// IsInterfaceNil -
func (n *nodesCoordinator) IsInterfaceNil() bool {
	return n == nil
}
