package spos

import "github.com/ElrondNetwork/elrond-go/core/check"

// ValidateConsensusCore checks for nil all the container objects
func ValidateConsensusCore(container ConsensusCoreHandler) error {
	if container == nil || container.IsInterfaceNil() {
		return ErrNilConsensusCore
	}
	if container.Blockchain() == nil || container.Blockchain().IsInterfaceNil() {
		return ErrNilBlockChain
	}
	if container.BlockProcessor() == nil || container.BlockProcessor().IsInterfaceNil() {
		return ErrNilBlockProcessor
	}
	if container.BootStrapper() == nil || container.BootStrapper().IsInterfaceNil() {
		return ErrNilBootstrapper
	}
	if container.BroadcastMessenger() == nil || container.BroadcastMessenger().IsInterfaceNil() {
		return ErrNilBroadcastMessenger
	}
	if container.Chronology() == nil || container.Chronology().IsInterfaceNil() {
		return ErrNilChronologyHandler
	}
	if container.Hasher() == nil || container.Hasher().IsInterfaceNil() {
		return ErrNilHasher
	}
	if container.Marshalizer() == nil || container.Marshalizer().IsInterfaceNil() {
		return ErrNilMarshalizer
	}
	if container.MultiSigner() == nil || container.MultiSigner().IsInterfaceNil() {
		return ErrNilMultiSigner
	}
	if container.Rounder() == nil || container.Rounder().IsInterfaceNil() {
		return ErrNilRounder
	}
	if container.ShardCoordinator() == nil || container.ShardCoordinator().IsInterfaceNil() {
		return ErrNilShardCoordinator
	}
	if container.SyncTimer() == nil || container.SyncTimer().IsInterfaceNil() {
		return ErrNilSyncTimer
	}
	if container.NodesCoordinator() == nil || container.NodesCoordinator().IsInterfaceNil() {
		return ErrNilNodesCoordinator
	}
	if container.PrivateKey() == nil || container.PrivateKey().IsInterfaceNil() {
		return ErrNilBlsPrivateKey
	}
	if container.SingleSigner() == nil || container.SingleSigner().IsInterfaceNil() {
		return ErrNilBlsSingleSigner
	}
	if check.IfNil(container.GetAntiFloodPreventer()) {
		return ErrNilAntifloodHandler
	}

	return nil
}
