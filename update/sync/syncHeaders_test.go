package sync

import (
	"encoding/json"
	"errors"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/update"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/dataPool/headersCache"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/update/mock"
	"github.com/stretchr/testify/require"
)

func createMockHeadersSyncHandlerArgs() ArgsNewHeadersSyncHandler {
	return ArgsNewHeadersSyncHandler{
		StorageService:  &mock.ChainStorerMock{},
		Cache:           &mock.HeadersCacherStub{},
		Marshalizer:     &mock.MarshalizerFake{},
		EpochHandler:    &mock.EpochStartTriggerStub{},
		RequestHandler:  &mock.RequestHandlerStub{},
		Uint64Converter: &mock.Uint64ByteSliceConverterMock{},
	}
}

func TestHeadersSyncHandler(t *testing.T) {
	t.Parallel()

	args := createMockHeadersSyncHandlerArgs()

	headersSyncHandler, err := NewHeadersSyncHandler(args)
	require.NotNil(t, headersSyncHandler)
	require.Nil(t, err)
	require.False(t, headersSyncHandler.IsInterfaceNil())
}

func TestHeadersSyncHandler_NilStorageErr(t *testing.T) {
	t.Parallel()

	args := createMockHeadersSyncHandlerArgs()
	args.StorageService = nil

	headersSyncHandler, err := NewHeadersSyncHandler(args)
	require.Nil(t, headersSyncHandler)
	require.Equal(t, update.ErrNilStorage, err)
}

func TestHeadersSyncHandler_NilCacheErr(t *testing.T) {
	t.Parallel()

	args := createMockHeadersSyncHandlerArgs()
	args.Cache = nil

	headersSyncHandler, err := NewHeadersSyncHandler(args)
	require.Nil(t, headersSyncHandler)
	require.Equal(t, update.ErrNilCacher, err)
}

func TestHeadersSyncHandler_NilEpochHandlerErr(t *testing.T) {
	t.Parallel()

	args := createMockHeadersSyncHandlerArgs()
	args.EpochHandler = nil

	headersSyncHandler, err := NewHeadersSyncHandler(args)
	require.Nil(t, headersSyncHandler)
	require.Equal(t, dataRetriever.ErrNilEpochHandler, err)
}

func TestHeadersSyncHandler_NilMarshalizerEr(t *testing.T) {
	t.Parallel()

	args := createMockHeadersSyncHandlerArgs()
	args.Marshalizer = nil

	headersSyncHandler, err := NewHeadersSyncHandler(args)
	require.Nil(t, headersSyncHandler)
	require.Equal(t, dataRetriever.ErrNilMarshalizer, err)
}

func TestHeadersSyncHandler_NilRequestHandlerEr(t *testing.T) {
	t.Parallel()

	args := createMockHeadersSyncHandlerArgs()
	args.RequestHandler = nil

	headersSyncHandler, err := NewHeadersSyncHandler(args)
	require.Nil(t, headersSyncHandler)
	require.Equal(t, process.ErrNilRequestHandler, err)
}

func TestSyncEpochStartMetaHeader_MetaBlockInStorage(t *testing.T) {
	t.Parallel()

	meta := &block.MetaBlock{Epoch: 1,
		EpochStart: block.EpochStart{
			LastFinalizedHeaders: []block.EpochStartShardData{
				{ShardId: 0, RootHash: []byte("shardDataRootHash"),
					PendingMiniBlockHeaders: []block.ShardMiniBlockHeader{
						{Hash: []byte("hash")},
					},
				},
			},
		}}
	args := createMockHeadersSyncHandlerArgs()
	args.StorageService = &mock.ChainStorerMock{GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
		return &mock.StorerStub{
			GetCalled: func(key []byte) (bytes []byte, err error) {
				return json.Marshal(meta)
			},
		}
	}}

	headersSyncHandler, err := NewHeadersSyncHandler(args)
	require.Nil(t, err)

	err = headersSyncHandler.syncEpochStartMetaHeader(1, time.Second)
	require.Nil(t, err)

	metaBlock, err := headersSyncHandler.GetEpochStartMetaBlock()
	require.Nil(t, err)
	require.Equal(t, meta, metaBlock)
}

func TestSyncEpochStartMetaHeader_MissingHeaderTimeout(t *testing.T) {
	t.Parallel()

	localErr := errors.New("not found")
	args := createMockHeadersSyncHandlerArgs()
	args.StorageService = &mock.ChainStorerMock{GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
		return &mock.StorerStub{
			GetCalled: func(key []byte) (bytes []byte, err error) {
				return nil, localErr
			},
			GetFromEpochCalled: func(key []byte, epoch uint32) (bytes []byte, err error) {
				return nil, localErr
			},
		}
	}}

	headersSyncHandler, err := NewHeadersSyncHandler(args)
	require.Nil(t, err)

	err = headersSyncHandler.syncEpochStartMetaHeader(1, time.Second)
	require.Equal(t, process.ErrTimeIsOut, err)
}

func TestSyncEpochStartMetaHeader_ReceiveWrongHeaderTimeout(t *testing.T) {
	t.Parallel()

	localErr := errors.New("not found")
	metaHash := []byte("metaHash")
	meta := &block.MetaBlock{Epoch: 1}
	args := createMockHeadersSyncHandlerArgs()
	args.Cache, _ = headersCache.NewHeadersPool(config.HeadersPoolConfig{
		MaxHeadersPerShard:            1000,
		NumElementsToRemoveOnEviction: 1,
	})
	args.EpochHandler = &mock.EpochStartTriggerStub{IsEpochStartCalled: func() bool {
		return true
	}}

	args.StorageService = &mock.ChainStorerMock{GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
		return &mock.StorerStub{
			GetCalled: func(key []byte) (bytes []byte, err error) {
				return nil, localErr
			},
			GetFromEpochCalled: func(key []byte, epoch uint32) (bytes []byte, err error) {
				return nil, localErr
			},
		}
	}}

	headersSyncHandler, err := NewHeadersSyncHandler(args)
	require.Nil(t, err)

	go func() {
		time.Sleep(100 * time.Millisecond)
		headersSyncHandler.metaBlockPool.AddHeader(metaHash, meta)
	}()

	err = headersSyncHandler.syncEpochStartMetaHeader(1, time.Second)
	require.Equal(t, process.ErrTimeIsOut, err)
}

func TestSyncEpochStartMetaHeader_ReceiveHeaderOk(t *testing.T) {
	t.Parallel()

	localErr := errors.New("not found")
	metaHash := []byte("epochStartBlock_0")
	meta := &block.MetaBlock{Epoch: 1,
		EpochStart: block.EpochStart{
			LastFinalizedHeaders: []block.EpochStartShardData{
				{ShardId: 0, RootHash: []byte("shardDataRootHash"),
					PendingMiniBlockHeaders: []block.ShardMiniBlockHeader{
						{Hash: []byte("hash")},
					},
				},
			},
		}}
	args := createMockHeadersSyncHandlerArgs()
	args.Cache, _ = headersCache.NewHeadersPool(config.HeadersPoolConfig{
		MaxHeadersPerShard:            1000,
		NumElementsToRemoveOnEviction: 1,
	})

	args.EpochHandler = &mock.EpochStartTriggerStub{
		IsEpochStartCalled: func() bool {
			return true
		},
		EpochStartMetaHdrHashCalled: func() []byte {
			return metaHash
		},
	}

	args.StorageService = &mock.ChainStorerMock{GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
		return &mock.StorerStub{
			GetCalled: func(key []byte) (bytes []byte, err error) {
				return nil, localErr
			},
			GetFromEpochCalled: func(key []byte, epoch uint32) (bytes []byte, err error) {
				return nil, localErr
			},
		}
	}}

	headersSyncHandler, err := NewHeadersSyncHandler(args)
	require.Nil(t, err)

	go func() {
		time.Sleep(100 * time.Millisecond)
		headersSyncHandler.metaBlockPool.AddHeader(metaHash, meta)
	}()

	err = headersSyncHandler.syncEpochStartMetaHeader(1, 2*time.Second)
	require.Nil(t, err)

	metaBlockSync, err := headersSyncHandler.GetEpochStartMetaBlock()
	require.Nil(t, err)
	require.Equal(t, meta, metaBlockSync)

}
