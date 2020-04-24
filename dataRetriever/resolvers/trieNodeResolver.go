package resolvers

import (
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/batch"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/p2p"
)

// maxBuffToSendTrieNodes represents max buffer size to send in bytes
var maxBuffToSendTrieNodes = uint64(1 << 18) //256KB

// ArgTrieNodeResolver is the argument structure used to create new TrieNodeResolver instance
type ArgTrieNodeResolver struct {
	SenderResolver   dataRetriever.TopicResolverSender
	TrieDataGetter   dataRetriever.TrieDataGetter
	Marshalizer      marshal.Marshalizer
	AntifloodHandler dataRetriever.P2PAntifloodHandler
	Throttler        dataRetriever.ResolverThrottler
}

// TrieNodeResolver is a wrapper over Resolver that is specialized in resolving trie node requests
type TrieNodeResolver struct {
	dataRetriever.TopicResolverSender
	messageProcessor
	trieDataGetter dataRetriever.TrieDataGetter
}

// NewTrieNodeResolver creates a new trie node resolver
func NewTrieNodeResolver(arg ArgTrieNodeResolver) (*TrieNodeResolver, error) {
	if check.IfNil(arg.SenderResolver) {
		return nil, dataRetriever.ErrNilResolverSender
	}
	if check.IfNil(arg.TrieDataGetter) {
		return nil, dataRetriever.ErrNilTrieDataGetter
	}
	if check.IfNil(arg.Marshalizer) {
		return nil, dataRetriever.ErrNilMarshalizer
	}
	if check.IfNil(arg.AntifloodHandler) {
		return nil, dataRetriever.ErrNilAntifloodHandler
	}
	if check.IfNil(arg.Throttler) {
		return nil, dataRetriever.ErrNilThrottler
	}

	return &TrieNodeResolver{
		TopicResolverSender: arg.SenderResolver,
		trieDataGetter:      arg.TrieDataGetter,
		messageProcessor: messageProcessor{
			marshalizer:      arg.Marshalizer,
			antifloodHandler: arg.AntifloodHandler,
			topic:            arg.SenderResolver.RequestTopic(),
			throttler:        arg.Throttler,
		},
	}, nil
}

// ProcessReceivedMessage will be the callback func from the p2p.Messenger and will be called each time a new message was received
// (for the topic this validator was registered to, usually a request topic)
func (tnRes *TrieNodeResolver) ProcessReceivedMessage(message p2p.MessageP2P, fromConnectedPeer p2p.PeerID) error {
	err := tnRes.canProcessMessage(message, fromConnectedPeer)
	if err != nil {
		return err
	}

	tnRes.throttler.StartProcessing()
	defer tnRes.throttler.EndProcessing()

	rd, err := tnRes.parseReceivedMessage(message)
	if err != nil {
		return err
	}

	switch rd.Type {
	case dataRetriever.HashType:
		var serializedNodes [][]byte
		serializedNodes, err = tnRes.trieDataGetter.GetSerializedNodes(rd.Value, maxBuffToSendTrieNodes)
		if err != nil {
			tnRes.ResolverDebugHandler().LogFailedToResolveData(
				tnRes.topic,
				rd.Value,
				err,
			)

			return err
		}

		var buff []byte
		buff, err = tnRes.marshalizer.Marshal(&batch.Batch{Data: serializedNodes})
		if err != nil {
			return err
		}

		return tnRes.Send(buff, message.Peer())
	default:
		return dataRetriever.ErrRequestTypeNotImplemented
	}
}

// RequestDataFromHash requests trie nodes from other peers having input a trie node hash
func (tnRes *TrieNodeResolver) RequestDataFromHash(hash []byte, _ uint32) error {
	return tnRes.SendOnRequestTopic(
		&dataRetriever.RequestData{
			Type:  dataRetriever.HashType,
			Value: hash,
		},
		[][]byte{hash},
	)
}

// RequestDataFromHashArray requests trie nodes from other peers having input multiple trie node hashes
func (tnRes *TrieNodeResolver) RequestDataFromHashArray(hashes [][]byte, _ uint32) error {
	b := &batch.Batch{
		Data: hashes,
	}
	buffHashes, err := tnRes.marshalizer.Marshal(b)
	if err != nil {
		return err
	}

	return tnRes.SendOnRequestTopic(
		&dataRetriever.RequestData{
			Type:  dataRetriever.HashArrayType,
			Value: buffHashes,
		},
		hashes,
	)
}

// SetNumPeersToQuery will set the number of intra shard and cross shard number of peer to query
func (tnRes *TrieNodeResolver) SetNumPeersToQuery(intra int, cross int) {
	tnRes.TopicResolverSender.SetNumPeersToQuery(intra, cross)
}

// NumPeersToQuery will return the number of intra shard and cross shard number of peer to query
func (tnRes *TrieNodeResolver) NumPeersToQuery() (int, int) {
	return tnRes.TopicResolverSender.NumPeersToQuery()
}

// SetResolverDebugHandler will set a resolver debug handler
func (tnRes *TrieNodeResolver) SetResolverDebugHandler(handler dataRetriever.ResolverDebugHandler) error {
	return tnRes.TopicResolverSender.SetResolverDebugHandler(handler)
}

// IsInterfaceNil returns true if there is no value under the interface
func (tnRes *TrieNodeResolver) IsInterfaceNil() bool {
	return tnRes == nil
}
