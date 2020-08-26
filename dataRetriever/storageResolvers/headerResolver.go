package storageResolvers

import (
	"sync"

	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/resolvers/epochproviders"
	"github.com/ElrondNetwork/elrond-go/storage"
)

var log = logger.GetOrCreate("dataretriever/storageresolvers")

// ArgHeaderResolver is the argument structure used to create new HeaderResolver instance
type ArgHeaderResolver struct {
	Messenger                dataRetriever.MessageHandler
	ResponseTopicName        string
	NonceConverter           typeConverters.Uint64ByteSliceConverter
	HdrStorage               storage.Storer
	HeadersNoncesStorage     storage.Storer
	ManualEpochStartNotifier dataRetriever.ManualEpochStartNotifier
}

type headerResolver struct {
	*storageResolver
	nonceConverter           typeConverters.Uint64ByteSliceConverter
	mutEpochHandler          sync.RWMutex
	epochHandler             dataRetriever.EpochHandler
	hdrStorage               storage.Storer
	hdrNoncesStorage         storage.Storer
	manualEpochStartNotifier dataRetriever.ManualEpochStartNotifier
	currentEpoch             uint32
}

// NewHeaderResolver creates a new storage header resolver
func NewHeaderResolver(arg ArgHeaderResolver) (*headerResolver, error) {
	if check.IfNil(arg.Messenger) {
		return nil, dataRetriever.ErrNilMessenger
	}
	if check.IfNil(arg.HdrStorage) {
		return nil, dataRetriever.ErrNilHeadersStorage
	}
	if check.IfNil(arg.HeadersNoncesStorage) {
		return nil, dataRetriever.ErrNilHeadersNoncesStorage
	}
	if check.IfNil(arg.NonceConverter) {
		return nil, dataRetriever.ErrNilUint64ByteSliceConverter
	}
	if check.IfNil(arg.ManualEpochStartNotifier) {
		return nil, dataRetriever.ErrNilManualEpochStartNotifier
	}

	epochHandler := epochproviders.NewNilEpochHandler()
	return &headerResolver{
		storageResolver: &storageResolver{
			messenger:         arg.Messenger,
			responseTopicName: arg.ResponseTopicName,
		},
		hdrStorage:               arg.HdrStorage,
		hdrNoncesStorage:         arg.HeadersNoncesStorage,
		nonceConverter:           arg.NonceConverter,
		epochHandler:             epochHandler,
		manualEpochStartNotifier: arg.ManualEpochStartNotifier,
	}, nil
}

// RequestDataFromHash searches the hash in provided storage and then will send to self the message
func (hdrRes *headerResolver) RequestDataFromHash(hash []byte, _ uint32) error {
	hdrRes.mutEpochHandler.RLock()
	metaEpoch := hdrRes.epochHandler.MetaEpoch()
	hdrRes.mutEpochHandler.RUnlock()

	shouldNotifyEpochChange := hdrRes.currentEpoch <= metaEpoch
	if shouldNotifyEpochChange {
		hdrRes.currentEpoch = metaEpoch + 1
		hdrRes.manualEpochStartNotifier.NewEpoch(hdrRes.currentEpoch)
	}

	buff, err := hdrRes.hdrStorage.SearchFirst(hash)
	if err != nil {
		return err
	}

	return hdrRes.sendToSelf(buff)
}

// RequestDataFromNonce requests a header by its nonce
func (hdrRes *headerResolver) RequestDataFromNonce(nonce uint64, epoch uint32) error {
	nonceKey := hdrRes.nonceConverter.ToByteSlice(nonce)
	hash, err := hdrRes.hdrNoncesStorage.SearchFirst(nonceKey)
	if err != nil {
		return err
	}

	return hdrRes.RequestDataFromHash(hash, epoch)
}

// RequestDataFromEpoch requests the epoch start block
func (hdrRes *headerResolver) RequestDataFromEpoch(identifier []byte) error {
	return hdrRes.RequestDataFromHash(identifier, 0)
}

// SetEpochHandler sets the epoch handler
func (hdrRes *headerResolver) SetEpochHandler(epochHandler dataRetriever.EpochHandler) error {
	if check.IfNil(epochHandler) {
		return dataRetriever.ErrNilEpochHandler
	}

	hdrRes.mutEpochHandler.Lock()
	hdrRes.epochHandler = epochHandler
	hdrRes.mutEpochHandler.Unlock()

	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (hdrRes *headerResolver) IsInterfaceNil() bool {
	return hdrRes == nil
}
