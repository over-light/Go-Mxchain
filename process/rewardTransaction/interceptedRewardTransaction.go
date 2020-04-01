package rewardTransaction

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// InterceptedRewardTransaction holds and manages a transaction based struct with extended functionality
type InterceptedRewardTransaction struct {
	rTx               *rewardTx.RewardTx
	marshalizer       marshal.Marshalizer
	hasher            hashing.Hasher
	pubkeyConv        state.PubkeyConverter
	coordinator       sharding.Coordinator
	hash              []byte
	rcvShard          uint32
	sndShard          uint32
	isForCurrentShard bool
}

// NewInterceptedRewardTransaction returns a new instance of InterceptedRewardTransaction
func NewInterceptedRewardTransaction(
	rewardTxBuff []byte,
	marshalizer marshal.Marshalizer,
	hasher hashing.Hasher,
	pubkeyConv state.PubkeyConverter,
	coordinator sharding.Coordinator,
) (*InterceptedRewardTransaction, error) {
	if rewardTxBuff == nil {
		return nil, process.ErrNilBuffer
	}
	if check.IfNil(marshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(hasher) {
		return nil, process.ErrNilHasher
	}
	if check.IfNil(pubkeyConv) {
		return nil, process.ErrNilPubkeyConverter
	}
	if check.IfNil(coordinator) {
		return nil, process.ErrNilShardCoordinator
	}

	rTx := &rewardTx.RewardTx{}
	err := marshalizer.Unmarshal(rTx, rewardTxBuff)
	if err != nil {
		return nil, err
	}

	inRewardTx := &InterceptedRewardTransaction{
		rTx:         rTx,
		marshalizer: marshalizer,
		hasher:      hasher,
		pubkeyConv:  pubkeyConv,
		coordinator: coordinator,
	}

	err = inRewardTx.processFields(rewardTxBuff)
	if err != nil {
		return nil, err
	}

	return inRewardTx, nil
}

func (inRTx *InterceptedRewardTransaction) processFields(rewardTxBuff []byte) error {
	inRTx.hash = inRTx.hasher.Compute(string(rewardTxBuff))

	rcvAddr, err := inRTx.pubkeyConv.CreateAddressFromBytes(inRTx.rTx.RcvAddr)
	if err != nil {
		return process.ErrInvalidRcvAddr
	}

	inRTx.rcvShard = inRTx.coordinator.ComputeId(rcvAddr)
	inRTx.sndShard = core.MetachainShardId

	isForCurrentShardRecv := inRTx.rcvShard == inRTx.coordinator.SelfId()
	isForCurrentShardSender := inRTx.sndShard == inRTx.coordinator.SelfId()
	inRTx.isForCurrentShard = isForCurrentShardRecv || isForCurrentShardSender

	return nil
}

// integrity checks for not nil fields and negative value
func (inRTx *InterceptedRewardTransaction) integrity() error {
	if len(inRTx.rTx.RcvAddr) == 0 {
		return process.ErrNilRcvAddr
	}

	if inRTx.rTx.Value == nil {
		return process.ErrNilValue
	}

	if inRTx.rTx.Value.Cmp(big.NewInt(0)) < 0 {
		return process.ErrNegativeValue
	}

	return nil
}

// Nonce returns the transaction nonce
func (inRTx *InterceptedRewardTransaction) Nonce() uint64 {
	return inRTx.rTx.GetNonce()
}

// Fee represents the reward transaction fee. It is always 0
func (inRTx *InterceptedRewardTransaction) Fee() *big.Int {
	return big.NewInt(0)
}

// SenderAddress returns the transaction sender address
func (inRTx *InterceptedRewardTransaction) SenderAddress() state.AddressContainer {
	return nil
}

// ReceiverShardId returns the receiver shard
func (inRTx *InterceptedRewardTransaction) ReceiverShardId() uint32 {
	return inRTx.rcvShard
}

// SenderShardId returns the sender shard
func (inRTx *InterceptedRewardTransaction) SenderShardId() uint32 {
	return inRTx.sndShard
}

// Transaction returns the reward transaction pointer that actually holds the data
func (inRTx *InterceptedRewardTransaction) Transaction() data.TransactionHandler {
	return inRTx.rTx
}

// Hash gets the hash of this transaction
func (inRTx *InterceptedRewardTransaction) Hash() []byte {
	return inRTx.hash
}

// CheckValidity checks if the received transaction is valid (not nil fields, valid sig and so on)
func (inRTx *InterceptedRewardTransaction) CheckValidity() error {
	err := inRTx.integrity()
	if err != nil {
		return err
	}

	return nil
}

// IsForCurrentShard returns true if this transaction is meant to be processed by the node from this shard
func (inRTx *InterceptedRewardTransaction) IsForCurrentShard() bool {
	return inRTx.isForCurrentShard
}

// Type returns the type of this intercepted data
func (inRTx *InterceptedRewardTransaction) Type() string {
	return "intercepted reward tx"
}

// IsInterfaceNil returns true if there is no value under the interface
func (inRTx *InterceptedRewardTransaction) IsInterfaceNil() bool {
	return inRTx == nil
}
