package resolvers

import (
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/logger"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/storage"
)

var log = logger.GetOrCreate("dataretriever/resolvers")

// HeaderResolver is a wrapper over Resolver that is specialized in resolving headers requests
type HeaderResolver struct {
	dataRetriever.TopicResolverSender
	headers              dataRetriever.HeadersPool
	hdrStorage           storage.Storer
	hdrNoncesStorage     storage.Storer
	marshalizer          marshal.Marshalizer
	nonceConverter       typeConverters.Uint64ByteSliceConverter
	epochHandler         dataRetriever.EpochHandler
	epochProviderByNonce dataRetriever.EpochProviderByNonce
}

// NewHeaderResolver creates a new header resolver
func NewHeaderResolver(
	senderResolver dataRetriever.TopicResolverSender,
	headers dataRetriever.HeadersPool,
	hdrStorage storage.Storer,
	headersNoncesStorage storage.Storer,
	marshalizer marshal.Marshalizer,
	nonceConverter typeConverters.Uint64ByteSliceConverter,
) (*HeaderResolver, error) {

	if check.IfNil(senderResolver) {
		return nil, dataRetriever.ErrNilResolverSender
	}
	if check.IfNil(headers) {
		return nil, dataRetriever.ErrNilHeadersDataPool
	}
	if check.IfNil(hdrStorage) {
		return nil, dataRetriever.ErrNilHeadersStorage
	}
	if check.IfNil(headersNoncesStorage) {
		return nil, dataRetriever.ErrNilHeadersNoncesStorage
	}
	if check.IfNil(marshalizer) {
		return nil, dataRetriever.ErrNilMarshalizer
	}
	if check.IfNil(nonceConverter) {
		return nil, dataRetriever.ErrNilUint64ByteSliceConverter
	}

	epochHandler := &nilEpochHandler{}
	hdrResolver := &HeaderResolver{
		TopicResolverSender:  senderResolver,
		headers:              headers,
		hdrStorage:           hdrStorage,
		hdrNoncesStorage:     headersNoncesStorage,
		marshalizer:          marshalizer,
		nonceConverter:       nonceConverter,
		epochHandler:         epochHandler,
		epochProviderByNonce: NewSimpleEpochProviderByNonce(epochHandler),
	}

	return hdrResolver, nil
}

// SetEpochHandler sets the epoch handler for this component
func (hdrRes *HeaderResolver) SetEpochHandler(epochHandler dataRetriever.EpochHandler) error {
	if check.IfNil(epochHandler) {
		return dataRetriever.ErrNilEpochHandler
	}

	hdrRes.epochHandler = epochHandler
	return nil
}

// ProcessReceivedMessage will be the callback func from the p2p.Messenger and will be called each time a new message was received
// (for the topic this validator was registered to, usually a request topic)
func (hdrRes *HeaderResolver) ProcessReceivedMessage(message p2p.MessageP2P, _ func(buffToSend []byte)) error {
	rd, err := hdrRes.parseReceivedMessage(message)
	if err != nil {
		return err
	}
	var buff []byte

	switch rd.Type {
	case dataRetriever.HashType:
		buff, err = hdrRes.resolveHeaderFromHash(rd)
	case dataRetriever.NonceType:
		buff, err = hdrRes.resolveHeaderFromNonce(rd)
	case dataRetriever.EpochType:
		buff, err = hdrRes.resolveHeaderFromEpoch(rd.Value)
	default:
		return dataRetriever.ErrResolveTypeUnknown
	}
	if err != nil {
		return err
	}
	if buff == nil {
		log.Trace("missing data",
			"data", rd)
		return nil
	}

	return hdrRes.Send(buff, message.Peer())
}

func (hdrRes *HeaderResolver) resolveHeaderFromNonce(rd *dataRetriever.RequestData) ([]byte, error) {
	// key is now an encoded nonce (uint64)
	nonce, err := hdrRes.nonceConverter.ToUint64(rd.Value)
	if err != nil {
		return nil, dataRetriever.ErrInvalidNonceByteSlice
	}

	epoch := rd.Epoch

	// TODO : uncomment this when epoch provider by nonce is complete
	//epoch, err = hdrRes.epochProviderByNonce.EpochForNonce(nonce)
	//if err != nil {
	//	return nil, err
	//}
	//
	//hash, err := hdrRes.hdrNoncesStorage.GetFromEpoch(rd.Value, epoch)
	hash, err := hdrRes.hdrNoncesStorage.SearchFirst(rd.Value)
	if err != nil {
		log.Trace("hdrNoncesStorage.Get from calculated epoch", "error", err.Error())
		// Search the nonce-key pair in data pool
		var hdrBytes []byte
		hdrBytes, err = hdrRes.searchInCache(nonce)
		if err != nil {
			return nil, err
		}

		return hdrBytes, nil
	}

	newRd := &dataRetriever.RequestData{
		Type:  rd.Type,
		Value: hash,
		Epoch: epoch,
	}

	return hdrRes.resolveHeaderFromHash(newRd)
}

func (hdrRes *HeaderResolver) searchInCache(nonce uint64) ([]byte, error) {
	headers, _, err := hdrRes.headers.GetHeadersByNonceAndShardId(nonce, hdrRes.TargetShardID())
	if err != nil {
		return nil, err
	}

	// TODO maybe we can return a slice of headers
	hdr := headers[len(headers)-1]
	buff, err := hdrRes.marshalizer.Marshal(hdr)
	if err != nil {
		return nil, err
	}

	return buff, nil
}

// resolveHeaderFromHash resolves a header using its key (header hash)
func (hdrRes *HeaderResolver) resolveHeaderFromHash(rd *dataRetriever.RequestData) ([]byte, error) {
	value, err := hdrRes.headers.GetHeaderByHash(rd.Value)
	if err != nil {
		return hdrRes.hdrStorage.SearchFirst(rd.Value)

		// TODO : uncomment this when epoch provider by nonce is complete

		//  return hdrRes.hdrStorage.GetFromEpoch(rd.Value, rd.Epoch)
	}

	buff, err := hdrRes.marshalizer.Marshal(value)
	if err != nil {
		return nil, err
	}

	return buff, nil
}

// resolveHeaderFromEpoch resolves a header using its key based on epoch
func (hdrRes *HeaderResolver) resolveHeaderFromEpoch(key []byte) ([]byte, error) {
	actualKey := key

	isUnknownEpoch, err := core.IsUnknownEpochIdentifier(key)
	if err != nil {
		return nil, err
	}

	if isUnknownEpoch {
		actualKey = []byte(core.EpochStartIdentifier(hdrRes.epochHandler.Epoch()))
	}

	return hdrRes.hdrStorage.Get(actualKey)
}

// parseReceivedMessage will transform the received p2p.Message in a RequestData object.
func (hdrRes *HeaderResolver) parseReceivedMessage(message p2p.MessageP2P) (*dataRetriever.RequestData, error) {
	rd := &dataRetriever.RequestData{}
	err := rd.Unmarshal(hdrRes.marshalizer, message)
	if err != nil {
		return nil, err
	}
	if rd.Value == nil {
		return nil, dataRetriever.ErrNilValue
	}

	return rd, nil
}

// RequestDataFromHash requests a header from other peers having input the hdr hash
func (hdrRes *HeaderResolver) RequestDataFromHash(hash []byte, epoch uint32) error {
	return hdrRes.SendOnRequestTopic(&dataRetriever.RequestData{
		Type:  dataRetriever.HashType,
		Value: hash,
		Epoch: epoch,
	})
}

// RequestDataFromNonce requests a header from other peers having input the hdr nonce
func (hdrRes *HeaderResolver) RequestDataFromNonce(nonce uint64, epoch uint32) error {
	return hdrRes.SendOnRequestTopic(&dataRetriever.RequestData{
		Type:  dataRetriever.NonceType,
		Value: hdrRes.nonceConverter.ToByteSlice(nonce),
		Epoch: epoch,
	})
}

// RequestDataFromEpoch requests a header from other peers having input the epoch
func (hdrRes *HeaderResolver) RequestDataFromEpoch(identifier []byte) error {
	return hdrRes.SendOnRequestTopic(&dataRetriever.RequestData{
		Type:  dataRetriever.EpochType,
		Value: identifier,
	})
}

// IsInterfaceNil returns true if there is no value under the interface
func (hdrRes *HeaderResolver) IsInterfaceNil() bool {
	return hdrRes == nil
}
