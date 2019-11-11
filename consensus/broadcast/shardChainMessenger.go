package broadcast

import (
	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/consensus/spos"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/partitioning"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type shardChainMessenger struct {
	*commonMessenger
	marshalizer      marshal.Marshalizer
	messenger        consensus.P2PMessenger
	shardCoordinator sharding.Coordinator
}

// NewShardChainMessenger creates a new shardChainMessenger object
func NewShardChainMessenger(
	marshalizer marshal.Marshalizer,
	messenger consensus.P2PMessenger,
	privateKey crypto.PrivateKey,
	shardCoordinator sharding.Coordinator,
	singleSigner crypto.SingleSigner,
) (*shardChainMessenger, error) {

	err := checkShardChainNilParameters(marshalizer, messenger, shardCoordinator, privateKey, singleSigner)
	if err != nil {
		return nil, err
	}

	cm := &commonMessenger{
		marshalizer:      marshalizer,
		messenger:        messenger,
		privateKey:       privateKey,
		shardCoordinator: shardCoordinator,
		singleSigner:     singleSigner,
	}

	scm := &shardChainMessenger{
		commonMessenger:  cm,
		marshalizer:      marshalizer,
		messenger:        messenger,
		shardCoordinator: shardCoordinator,
	}

	return scm, nil
}

func checkShardChainNilParameters(
	marshalizer marshal.Marshalizer,
	messenger consensus.P2PMessenger,
	shardCoordinator sharding.Coordinator,
	privateKey crypto.PrivateKey,
	singleSigner crypto.SingleSigner,
) error {
	if marshalizer == nil || marshalizer.IsInterfaceNil() {
		return spos.ErrNilMarshalizer
	}
	if messenger == nil || messenger.IsInterfaceNil() {
		return spos.ErrNilMessenger
	}
	if shardCoordinator == nil || shardCoordinator.IsInterfaceNil() {
		return spos.ErrNilShardCoordinator
	}
	if privateKey == nil || privateKey.IsInterfaceNil() {
		return spos.ErrNilPrivateKey
	}
	if singleSigner == nil || singleSigner.IsInterfaceNil() {
		return spos.ErrNilSingleSigner
	}

	return nil
}

// BroadcastBlock will send on in-shard headers topic and on in-shard miniblocks topic the header and block body
func (scm *shardChainMessenger) BroadcastBlock(blockBody data.BodyHandler, header data.HeaderHandler) error {
	if blockBody == nil || blockBody.IsInterfaceNil() {
		return spos.ErrNilBody
	}

	err := blockBody.IntegrityAndValidity()
	if err != nil {
		return err
	}

	if header == nil || header.IsInterfaceNil() {
		return spos.ErrNilHeader
	}

	msgHeader, err := scm.marshalizer.Marshal(header)
	if err != nil {
		return err
	}

	msgBlockBody, err := scm.marshalizer.Marshal(blockBody)
	if err != nil {
		return err
	}

	selfIdentifier := scm.shardCoordinator.CommunicationIdentifier(scm.shardCoordinator.SelfId())

	go scm.messenger.Broadcast(factory.HeadersTopic+selfIdentifier, msgHeader)
	go scm.messenger.Broadcast(factory.MiniBlocksTopic+selfIdentifier, msgBlockBody)

	return nil
}

// BroadcastShardHeader will send on shard headers for metachain topic the header
func (scm *shardChainMessenger) BroadcastShardHeader(header data.HeaderHandler) error {
	if header == nil || header.IsInterfaceNil() {
		return spos.ErrNilHeader
	}

	msgHeader, err := scm.marshalizer.Marshal(header)
	if err != nil {
		return err
	}

	shardHeaderForMetachainTopic := factory.ShardHeadersForMetachainTopic +
		scm.shardCoordinator.CommunicationIdentifier(sharding.MetachainShardId)

	go scm.messenger.Broadcast(shardHeaderForMetachainTopic, msgHeader)

	return nil
}

// BroadcastMiniBlocks will send on miniblocks topic the cross-shard miniblocks
func (scm *shardChainMessenger) BroadcastMiniBlocks(miniBlocks map[uint32][]byte) error {
	mbs := 0
	for k, v := range miniBlocks {
		mbs++
		miniBlocksTopic := factory.MiniBlocksTopic +
			scm.shardCoordinator.CommunicationIdentifier(k)

		go scm.messenger.Broadcast(miniBlocksTopic, v)
	}

	if mbs > 0 {
		log.Debug("miniblocks sent", "number", mbs)
	}

	return nil
}

// BroadcastTransactions will send on transaction topic the transactions
func (scm *shardChainMessenger) BroadcastTransactions(transactions map[string][][]byte) error {
	dataPacker, err := partitioning.NewSimpleDataPacker(scm.marshalizer)
	if err != nil {
		return err
	}

	txs := 0
	for topic, v := range transactions {
		txs += len(v)
		// forward txs to the destination shards in packets
		packets, err := dataPacker.PackDataInChunks(v, core.MaxBulkTransactionSize)
		if err != nil {
			return err
		}

		for _, buff := range packets {
			go scm.messenger.Broadcast(topic, buff)
		}
	}

	if txs > 0 {
		log.Debug("transactions sent", "number", txs)
	}

	return nil
}

// BroadcastHeader will send on in-shard headers topic the header
func (scm *shardChainMessenger) BroadcastHeader(header data.HeaderHandler) error {
	if header == nil || header.IsInterfaceNil() {
		return spos.ErrNilHeader
	}

	msgHeader, err := scm.marshalizer.Marshal(header)
	if err != nil {
		return err
	}

	selfIdentifier := scm.shardCoordinator.CommunicationIdentifier(scm.shardCoordinator.SelfId())

	go scm.messenger.Broadcast(factory.HeadersTopic+selfIdentifier, msgHeader)

	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (scm *shardChainMessenger) IsInterfaceNil() bool {
	if scm == nil {
		return true
	}
	return false
}
