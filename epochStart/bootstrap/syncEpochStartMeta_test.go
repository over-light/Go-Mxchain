package bootstrap

import (
	"errors"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/epochStart/mock"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/process/economics"
	"github.com/stretchr/testify/require"
)

func TestNewEpochStartMetaSyncer_ShouldWork(t *testing.T) {
	t.Parallel()

	args := getEpochStartSyncerArgs()
	ess, err := NewEpochStartMetaSyncer(args)
	require.NoError(t, err)
	require.False(t, check.IfNil(ess))
}

func TestEpochStartMetaSyncer_SyncEpochStartMeta_ErrRegisterMessengerProcessorShouldErr(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("expected error")

	args := getEpochStartSyncerArgs()
	messenger := &mock.MessengerStub{
		RegisterMessageProcessorCalled: func(_ string, _ p2p.MessageProcessor) error {
			return expectedErr
		},
	}
	args.Messenger = messenger
	ess, _ := NewEpochStartMetaSyncer(args)

	mb, err := ess.SyncEpochStartMeta(1 * time.Second)
	require.Equal(t, expectedErr, err)
	require.Nil(t, mb)
}

func TestEpochStartMetaSyncer_SyncEpochStartMeta_ProcessorReturnsErrorShouldErr(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("expected error")

	args := getEpochStartSyncerArgs()
	messenger := &mock.MessengerStub{
		ConnectedPeersCalled: func() []p2p.PeerID {
			return []p2p.PeerID{"peer_0", "peer_1", "peer_2", "peer_3", "peer_4", "peer_5"}
		},
	}
	args.Messenger = messenger
	ess, _ := NewEpochStartMetaSyncer(args)

	mbIntercProc := &mock.MetaBlockInterceptorProcessorStub{
		GetEpochStartMetaBlockCalled: func() (*block.MetaBlock, error) {
			return nil, expectedErr
		},
	}
	ess.SetEpochStartMetaBlockInterceptorProcessor(mbIntercProc)

	mb, err := ess.SyncEpochStartMeta(1 * time.Second)
	require.Equal(t, expectedErr, err)
	require.Nil(t, mb)
}

func TestEpochStartMetaSyncer_SyncEpochStartMeta_ShouldWork(t *testing.T) {
	t.Parallel()

	expectedMb := &block.MetaBlock{Nonce: 37}

	args := getEpochStartSyncerArgs()
	messenger := &mock.MessengerStub{
		ConnectedPeersCalled: func() []p2p.PeerID {
			return []p2p.PeerID{"peer_0", "peer_1", "peer_2", "peer_3", "peer_4", "peer_5"}
		},
	}
	args.Messenger = messenger
	ess, _ := NewEpochStartMetaSyncer(args)

	mbIntercProc := &mock.MetaBlockInterceptorProcessorStub{
		GetEpochStartMetaBlockCalled: func() (*block.MetaBlock, error) {
			return expectedMb, nil
		},
	}
	ess.SetEpochStartMetaBlockInterceptorProcessor(mbIntercProc)

	mb, err := ess.SyncEpochStartMeta(1 * time.Second)
	require.NoError(t, err)
	require.Equal(t, expectedMb, mb)
}

func getEpochStartSyncerArgs() ArgsNewEpochStartMetaSyncer {
	return ArgsNewEpochStartMetaSyncer{
		RequestHandler:    &mock.RequestHandlerStub{},
		Messenger:         &mock.MessengerStub{},
		Marshalizer:       &mock.MarshalizerMock{},
		TxSignMarshalizer: &mock.MarshalizerMock{},
		ShardCoordinator:  mock.NewMultiShardsCoordinatorMock(2),
		KeyGen:            &mock.KeyGenMock{},
		BlockKeyGen:       &mock.KeyGenMock{},
		Hasher:            &mock.HasherMock{},
		Signer:            &mock.SignerStub{},
		BlockSigner:       &mock.SignerStub{},
		ChainID:           []byte("chain-ID"),
		EconomicsData:     &economics.EconomicsData{},
		WhitelistHandler:  &mock.WhiteListHandlerStub{},
		AddressPubkeyConv: mock.NewPubkeyConverterMock(32),
		NonceConverter:    &mock.Uint64ByteSliceConverterMock{},
	}

}
