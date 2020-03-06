package shardchain

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/epochStart/mock"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/stretchr/testify/assert"
)

func createMockShardEpochStartTriggerArguments() *ArgsShardEpochStartTrigger {
	return &ArgsShardEpochStartTrigger{
		Marshalizer: &mock.MarshalizerMock{},
		Hasher:      &mock.HasherMock{},
		HeaderValidator: &mock.HeaderValidatorStub{
			IsHeaderConstructionValidCalled: func(currHdr, prevHdr data.HeaderHandler) error {
				return nil
			},
		},
		Uint64Converter: &mock.Uint64ByteSliceConverterMock{},
		DataPool: &mock.PoolsHolderStub{
			HeadersCalled: func() dataRetriever.HeadersPool {
				return &mock.HeadersCacherStub{}
			},
			MiniBlocksCalled: func() storage.Cacher {
				return &mock.CacherStub{}
			},
		},
		Storage: &mock.ChainStorerStub{
			GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
				return &mock.StorerStub{
					GetCalled: func(key []byte) (bytes []byte, err error) {
						return []byte("hash"), nil
					},
					PutCalled: func(key, data []byte) error {
						return nil
					},
				}
			},
		},
		RequestHandler:     &mock.RequestHandlerStub{},
		EpochStartNotifier: &mock.EpochStartNotifierStub{},
	}
}

func TestNewEpochStartTrigger_NilArgumentsShouldErr(t *testing.T) {
	t.Parallel()

	epochStartTrigger, err := NewEpochStartTrigger(nil)

	assert.Nil(t, epochStartTrigger)
	assert.Equal(t, epochStart.ErrNilArgsNewShardEpochStartTrigger, err)
}

func TestNewEpochStartTrigger_NilHasherShouldErr(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.Hasher = nil
	epochStartTrigger, err := NewEpochStartTrigger(args)

	assert.Nil(t, epochStartTrigger)
	assert.Equal(t, epochStart.ErrNilHasher, err)
}

func TestNewEpochStartTrigger_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.Marshalizer = nil
	epochStartTrigger, err := NewEpochStartTrigger(args)

	assert.Nil(t, epochStartTrigger)
	assert.Equal(t, epochStart.ErrNilMarshalizer, err)
}

func TestNewEpochStartTrigger_NilHeaderShouldErr(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.HeaderValidator = nil
	epochStartTrigger, err := NewEpochStartTrigger(args)

	assert.Nil(t, epochStartTrigger)
	assert.Equal(t, epochStart.ErrNilHeaderValidator, err)
}

func TestNewEpochStartTrigger_NilDataPoolShouldErr(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.DataPool = nil
	epochStartTrigger, err := NewEpochStartTrigger(args)

	assert.Nil(t, epochStartTrigger)
	assert.Equal(t, epochStart.ErrNilDataPoolsHolder, err)
}

func TestNewEpochStartTrigger_NilStorageShouldErr(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.Storage = nil
	epochStartTrigger, err := NewEpochStartTrigger(args)

	assert.Nil(t, epochStartTrigger)
	assert.Equal(t, epochStart.ErrNilStorageService, err)
}

func TestNewEpochStartTrigger_NilRequestHandlerShouldErr(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.RequestHandler = nil
	epochStartTrigger, err := NewEpochStartTrigger(args)

	assert.Nil(t, epochStartTrigger)
	assert.Equal(t, epochStart.ErrNilRequestHandler, err)
}

func TestNewEpochStartTrigger_NilUint64ConverterShouldErr(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.Uint64Converter = nil
	epochStartTrigger, err := NewEpochStartTrigger(args)

	assert.Nil(t, epochStartTrigger)
	assert.Equal(t, epochStart.ErrNilUint64Converter, err)
}

func TestNewEpochStartTrigger_NilEpochStartNotifierShouldErr(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.EpochStartNotifier = nil
	epochStartTrigger, err := NewEpochStartTrigger(args)

	assert.Nil(t, epochStartTrigger)
	assert.Equal(t, epochStart.ErrNilEpochStartNotifier, err)
}

func TestNewEpochStartTrigger_NilMetaBlockUnitShouldErr(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.Storage = &mock.ChainStorerStub{
		GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
			return nil
		},
	}
	epochStartTrigger, err := NewEpochStartTrigger(args)

	assert.Nil(t, epochStartTrigger)
	assert.Equal(t, epochStart.ErrNilMetaBlockStorage, err)
}

func TestNewEpochStartTrigger_NilMetaNonceHashStorageShouldErr(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.Storage = &mock.ChainStorerStub{
		GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
			switch unitType {
			case dataRetriever.MetaHdrNonceHashDataUnit:
				return nil
			default:
				return &mock.StorerStub{}
			}
		},
	}
	epochStartTrigger, err := NewEpochStartTrigger(args)

	assert.Nil(t, epochStartTrigger)
	assert.Equal(t, epochStart.ErrNilMetaNonceHashStorage, err)
}

func TestNewEpochStartTrigger_NilMiniblockStorageShouldErr(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.Storage = &mock.ChainStorerStub{
		GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
			switch unitType {
			case dataRetriever.MiniBlockUnit:
				return nil
			default:
				return &mock.StorerStub{}
			}
		},
	}
	epochStartTrigger, err := NewEpochStartTrigger(args)

	assert.Nil(t, epochStartTrigger)
	assert.Equal(t, epochStart.ErrNilMiniblocksStorage, err)
}

func TestNewEpochStartTrigger_NilHeadersPoolShouldErr(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.DataPool = &mock.PoolsHolderStub{
		HeadersCalled: func() dataRetriever.HeadersPool {
			return nil
		},
		MiniBlocksCalled: func() storage.Cacher {
			return &mock.CacherStub{}
		},
	}
	epochStartTrigger, err := NewEpochStartTrigger(args)

	assert.Nil(t, epochStartTrigger)
	assert.Equal(t, epochStart.ErrNilMetaBlocksPool, err)
}

func TestNewEpochStartTrigger_NilMiniblocksPoolShouldErr(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.DataPool = &mock.PoolsHolderStub{
		HeadersCalled: func() dataRetriever.HeadersPool {
			return &mock.HeadersCacherStub{}
		},
		MiniBlocksCalled: func() storage.Cacher {
			return nil
		},
	}
	epochStartTrigger, err := NewEpochStartTrigger(args)

	assert.Nil(t, epochStartTrigger)
	assert.Equal(t, epochStart.ErrNilMiniblocksPool, err)
}

func TestNewEpochStartTrigger_ShouldOk(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	epochStartTrigger, err := NewEpochStartTrigger(args)

	assert.NotNil(t, epochStartTrigger)
	assert.Nil(t, err)
}

func TestTrigger_ReceivedHeaderNotEpochStart(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.Validity = 2
	args.Finality = 2
	epochStartTrigger, _ := NewEpochStartTrigger(args)

	hash := []byte("hash")
	header := &block.MetaBlock{Nonce: 100}
	header.EpochStart.LastFinalizedHeaders = []block.EpochStartShardData{{ShardId: 0, RootHash: hash, HeaderHash: hash}}
	epochStartTrigger.ReceivedHeader(header)

	assert.False(t, epochStartTrigger.IsEpochStart())
}

func TestTrigger_ReceivedHeaderIsEpochStartTrue(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.Validity = 1
	args.Finality = 2
	epochStartTrigger, _ := NewEpochStartTrigger(args)

	oldEpHeader := &block.MetaBlock{Nonce: 99, Epoch: 0}
	prevHash, _ := core.CalculateHash(args.Marshalizer, args.Hasher, oldEpHeader)

	hash := []byte("hash")
	header := &block.MetaBlock{Nonce: 100, Epoch: 1, PrevHash: prevHash}
	header.EpochStart.LastFinalizedHeaders = []block.EpochStartShardData{{ShardId: 0, RootHash: hash, HeaderHash: hash}}
	epochStartTrigger.ReceivedHeader(header)
	epochStartTrigger.ReceivedHeader(oldEpHeader)

	prevHash, _ = core.CalculateHash(args.Marshalizer, args.Hasher, header)
	header = &block.MetaBlock{Nonce: 101, Epoch: 1, PrevHash: prevHash}
	epochStartTrigger.ReceivedHeader(header)

	prevHash, _ = core.CalculateHash(args.Marshalizer, args.Hasher, header)
	header = &block.MetaBlock{Nonce: 102, Epoch: 1, PrevHash: prevHash}
	epochStartTrigger.ReceivedHeader(header)

	assert.True(t, epochStartTrigger.IsEpochStart())
}

func TestTrigger_ReceivedHeaderIsEpochStartTrueWithPeerMiniblocks(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()

	hash := []byte("hash")

	peerMiniblock := &block.MiniBlock{
		TxHashes:        [][]byte{},
		ReceiverShardID: core.AllShardId,
		SenderShardID:   core.MetachainShardId,
	}

	peerMiniBlockHash, _ := args.Marshalizer.Marshal(peerMiniblock)

	miniBlockHeader := block.MiniBlockHeader{
		Hash: peerMiniBlockHash, Type: block.PeerBlock, SenderShardID: core.MetachainShardId, ReceiverShardID: core.AllShardId, TxCount: 1}

	previousHeader99 := &block.MetaBlock{Nonce: 99, Epoch: 0}
	previousHeaderHash, _ := core.CalculateHash(args.Marshalizer, args.Hasher, previousHeader99)

	epochStartHeader := &block.MetaBlock{Nonce: 100, Epoch: 1, PrevHash: previousHeaderHash}
	epochStartHeader.EpochStart.LastFinalizedHeaders = []block.EpochStartShardData{{ShardId: 0, RootHash: hash, HeaderHash: hash}}
	epochStartHeader.MiniBlockHeaders = []block.MiniBlockHeader{miniBlockHeader}
	epochStartHeaderHash, _ := core.CalculateHash(args.Marshalizer, args.Hasher, epochStartHeader)

	newHeader101 := &block.MetaBlock{Nonce: 101, Epoch: 1, PrevHash: epochStartHeaderHash}
	newHeaderHash101, _ := core.CalculateHash(args.Marshalizer, args.Hasher, newHeader101)

	newHeader102 := &block.MetaBlock{Nonce: 102, Epoch: 1, PrevHash: newHeaderHash101}
	newHeaderHash102, _ := core.CalculateHash(args.Marshalizer, args.Hasher, newHeader102)

	hashesToHeaders := make(map[string]data.HeaderHandler)
	hashesToHeaders[string(previousHeaderHash)] = previousHeader99
	hashesToHeaders[string(epochStartHeaderHash)] = epochStartHeader
	hashesToHeaders[string(newHeaderHash101)] = newHeader101
	hashesToHeaders[string(newHeaderHash102)] = newHeader102

	noncesToHeader := make(map[string][]byte)
	noncesToHeader[fmt.Sprint(previousHeader99.Nonce)] = previousHeaderHash
	noncesToHeader[fmt.Sprint(epochStartHeader.Nonce)] = epochStartHeaderHash
	noncesToHeader[fmt.Sprint(newHeader101.Nonce)] = newHeaderHash101
	noncesToHeader[fmt.Sprint(newHeader102.Nonce)] = newHeaderHash102

	args.DataPool = &mock.PoolsHolderStub{
		HeadersCalled: func() dataRetriever.HeadersPool {
			return &mock.HeadersCacherStub{
				GetHeaderByHashCalled: func(hash []byte) (handler data.HeaderHandler, err error) {
					header, ok := hashesToHeaders[string(hash)]
					if !ok {
						return nil, nil
					}
					return header, nil
				},
				GetHeaderByNonceAndShardIdCalled: func(hdrNonce uint64, shardId uint32) (handlers []data.HeaderHandler, i [][]byte, err error) {
					return nil, nil, nil
				},
			}
		},
		MiniBlocksCalled: func() storage.Cacher {
			return &mock.CacherStub{
				GetCalled: func(key []byte) (value interface{}, ok bool) {
					if bytes.Compare(key, peerMiniBlockHash) == 0 {
						return peerMiniblock, true
					}
					return nil, false
				},
			}
		},
	}
	args.Uint64Converter = &mock.Uint64ByteSliceConverterMock{
		ToByteSliceCalled: func(u uint64) []byte {
			return []byte(fmt.Sprint(u))
		},
	}
	args.Storage = &mock.ChainStorerStub{
		GetStorerCalled: func(unitType dataRetriever.UnitType) storage.Storer {
			return &mock.StorerStub{
				GetCalled: func(key []byte) (bytes []byte, err error) {
					value := string(key)
					return noncesToHeader[value], nil
				},
				PutCalled: func(key, data []byte) error {
					return nil
				},
			}
		},
	}

	args.Validity = 1
	args.Finality = 2

	epochStartTrigger, _ := NewEpochStartTrigger(args)

	epochStartTrigger.ReceivedHeader(previousHeader99)
	assert.False(t, epochStartTrigger.IsEpochStart())
	epochStartTrigger.ReceivedHeader(epochStartHeader)
	assert.False(t, epochStartTrigger.IsEpochStart())
	epochStartTrigger.ReceivedHeader(newHeader101)
	assert.False(t, epochStartTrigger.IsEpochStart())
	epochStartTrigger.ReceivedHeader(newHeader102)
	assert.True(t, epochStartTrigger.IsEpochStart())
}

func TestTrigger_Epoch(t *testing.T) {
	t.Parallel()

	epoch := uint32(1)
	args := createMockShardEpochStartTriggerArguments()
	args.Epoch = epoch
	epochStartTrigger, _ := NewEpochStartTrigger(args)

	currentEpoch := epochStartTrigger.Epoch()
	assert.Equal(t, epoch, currentEpoch)
}

func TestTrigger_ProcessedAndRevert(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	args.Validity = 0
	args.Finality = 0
	args.EpochStartNotifier = &mock.EpochStartNotifierStub{NotifyAllCalled: func(hdr data.HeaderHandler) {}}
	et, _ := NewEpochStartTrigger(args)

	hash := []byte("hash")
	epochStartRound := uint64(100)
	header := &block.MetaBlock{Nonce: 100, Round: epochStartRound, Epoch: 1}
	header.EpochStart.LastFinalizedHeaders = []block.EpochStartShardData{{ShardId: 0, RootHash: hash, HeaderHash: hash}}
	et.ReceivedHeader(header)
	header = &block.MetaBlock{Nonce: 101, Round: epochStartRound + 1, Epoch: 1}
	et.ReceivedHeader(header)

	assert.True(t, et.IsEpochStart())
	assert.Equal(t, epochStartRound, et.EpochStartRound())

	et.SetProcessed(&block.Header{EpochStartMetaHash: []byte("metahash")})
	assert.False(t, et.isEpochStart)
	assert.False(t, et.newEpochHdrReceived)

	et.Revert(epochStartRound)
	assert.True(t, et.isEpochStart)
	assert.True(t, et.newEpochHdrReceived)
}

func TestTrigger_RequestEpochStartIfNeeded(t *testing.T) {
	t.Parallel()

	args := createMockShardEpochStartTriggerArguments()
	called := false
	args.RequestHandler = &mock.RequestHandlerStub{RequestStartOfEpochMetaBlockCalled: func(_ uint32) {
		called = true
	}}
	et, _ := NewEpochStartTrigger(args)
	et.epoch = 2

	hash := []byte("hash")
	et.RequestEpochStartIfNeeded(&block.Header{Epoch: 10})
	assert.False(t, called)

	et.RequestEpochStartIfNeeded(&block.MetaBlock{Epoch: 3,
		EpochStart: block.EpochStart{LastFinalizedHeaders: []block.EpochStartShardData{{ShardId: 0, RootHash: hash, HeaderHash: hash}}}})
	assert.False(t, called)

	et.RequestEpochStartIfNeeded(&block.MetaBlock{Epoch: 2})
	assert.False(t, called)

	et.mapEpochStartHdrs[string(hash)] = &block.MetaBlock{Epoch: 3}
	et.RequestEpochStartIfNeeded(&block.MetaBlock{Epoch: 3})
	assert.False(t, called)

	et.RequestEpochStartIfNeeded(&block.MetaBlock{Epoch: 4})
	assert.True(t, called)
}
