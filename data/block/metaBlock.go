package block

import (
	"fmt"
	"io"
	"math/big"

	"github.com/ElrondNetwork/elrond-go-sandbox/data/block/capnp"
	"github.com/glycerine/go-capnproto"
)

// PeerAction type represents the possible events that a node can trigger for the metachain to notarize
type PeerAction uint8

// Constants mapping the actions that a node can take
const (
	PeerRegistrantion PeerAction = iota + 1
	PeerDeregistration
)

func (pa PeerAction) String() string {
	switch pa {
	case PeerRegistrantion:
		return "PeerRegistration"
	case PeerDeregistration:
		return "PeerDeregistration"
	default:
		return fmt.Sprintf("Unknown type (%d)", pa)
	}
}

// PeerData holds information about actions taken by a peer:
//  - a peer can register with an amount to become a validator
//  - a peer can choose to deregister and get back the deposited value
type PeerData struct {
	PublicKey []byte     `capid:"0"`
	Action    PeerAction `capid:"1"`
	TimeStamp uint64     `capid:"2"`
	Value     *big.Int   `capid:"3"`
}

// ShardData holds the block information sent by the shards to the metachain
type ShardData struct {
	ShardId         uint32 `capid:"0"`
	HeaderHash      []byte `capid:"1"`
	TxBlockBodyHash []byte `capid:"2"`
}

// MetaBlock holds the data that will be saved to the metachain each round
type MetaBlock struct {
	Nonce         uint64      `capid:"0"`
	Epoch         uint32      `capid:"1"`
	Round         uint32      `capid:"2"`
	ShardInfo     []ShardData `capid:"3"`
	PeerInfo      []PeerData  `capid:"4"`
	Signature     []byte      `capid:"5"`
	PubKeysBitmap []byte      `capid:"6"`
	PreviousHash  []byte      `capid:"7"`
	StateRootHash []byte      `capid:"8"`
}

// Save saves the serialized data of a PeerData into a stream through Capnp protocol
func (p *PeerData) Save(w io.Writer) error {
	seg := capn.NewBuffer(nil)
	PeerDataGoToCapn(seg, p)
	_, err := seg.WriteTo(w)
	return err
}

// Load loads the data from the stream into a PeerData object through Capnp protocol
func (p *PeerData) Load(r io.Reader) error {
	capMsg, err := capn.ReadFromStream(r, nil)
	if err != nil {
		return err
	}
	z := capnp.ReadRootPeerDataCapn(capMsg)
	PeerDataCapnToGo(z, p)
	return nil
}

// Save saves the serialized data of a ShardData into a stream through Capnp protocol
func (s *ShardData) Save(w io.Writer) error {
	seg := capn.NewBuffer(nil)
	ShardDataGoToCapn(seg, s)
	_, err := seg.WriteTo(w)
	return err
}

// Load loads the data from the stream into a ShardData object through Capnp protocol
func (s *ShardData) Load(r io.Reader) error {
	capMsg, err := capn.ReadFromStream(r, nil)
	if err != nil {
		return err
	}
	z := capnp.ReadRootShardDataCapn(capMsg)
	ShardDataCapnToGo(z, s)
	return nil
}

// Save saves the serialized data of a MetaBlock into a stream through Capnp protocol
func (m *MetaBlock) Save(w io.Writer) error {
	seg := capn.NewBuffer(nil)
	MetaBlockGoToCapn(seg, m)
	_, err := seg.WriteTo(w)
	return err
}

// Load loads the data from the stream into a MetaBlock object through Capnp protocol
func (m *MetaBlock) Load(r io.Reader) error {
	capMsg, err := capn.ReadFromStream(r, nil)
	if err != nil {
		return err
	}
	z := capnp.ReadRootMetaBlockCapn(capMsg)
	MetaBlockCapnToGo(z, m)
	return nil
}

// PeerDataGoToCapn is a helper function to copy fields from a Peer Data object to a PeerDataCapn object
func PeerDataGoToCapn(seg *capn.Segment, src *PeerData) capnp.PeerDataCapn {
	dest := capnp.AutoNewPeerDataCapn(seg)
	value, _ := src.Value.GobEncode()
	dest.SetPublicKey(src.PublicKey)
	dest.SetAction(uint8(src.Action))
	dest.SetTimestamp(src.TimeStamp)
	dest.SetValue(value)

	return dest
}

// PeerDataCapnToGo is a helper function to copy fields from a PeerDataCapn object to a PeerData object
func PeerDataCapnToGo(src capnp.PeerDataCapn, dest *PeerData) *PeerData {
	if dest == nil {
		dest = &PeerData{}
	}
	if dest.Value == nil {
		dest.Value = big.NewInt(0)
	}
	dest.PublicKey = src.PublicKey()
	dest.Action = PeerAction(src.Action())
	dest.TimeStamp = src.Timestamp()
	err := dest.Value.GobDecode(src.Value())
	if err != nil {
		return nil
	}
	return dest
}

// ShardDataGoToCapn is a helper function to copy fields from a ShardData object to a ShardDataCapn object
func ShardDataGoToCapn(seg *capn.Segment, src *ShardData) capnp.ShardDataCapn {
	dest := capnp.AutoNewShardDataCapn(seg)

	dest.SetShardId(src.ShardId)
	dest.SetHeaderHash(src.HeaderHash)
	dest.SetTxBlockBodyHash(src.TxBlockBodyHash)

	return dest
}

// ShardDataCapnToGo is a helper function to copy fields from a ShardDataCapn object to a ShardData object
func ShardDataCapnToGo(src capnp.ShardDataCapn, dest *ShardData) *ShardData {
	if dest == nil {
		dest = &ShardData{}
	}
	dest.ShardId = src.ShardId()
	dest.HeaderHash = src.HeaderHash()
	dest.TxBlockBodyHash = src.TxBlockBodyHash()

	return dest
}

// MetaBlockGoToCapn is a helper function to copy fields from a MetaBlock object to a MetaBlockCapn object
func MetaBlockGoToCapn(seg *capn.Segment, src *MetaBlock) capnp.MetaBlockCapn {
	dest := capnp.AutoNewMetaBlockCapn(seg)

	dest.SetNonce(src.Nonce)
	dest.SetEpoch(src.Epoch)
	dest.SetRound(src.Round)

	if len(src.ShardInfo) > 0 {
		typedList := capnp.NewShardDataCapnList(seg, len(src.ShardInfo))
		plist := capn.PointerList(typedList)

		for i, elem := range src.ShardInfo {
			_ = plist.Set(i, capn.Object(ShardDataGoToCapn(seg, &elem)))
		}
		dest.SetShardInfo(typedList)
	}

	if len(src.PeerInfo) > 0 {
		typedList := capnp.NewPeerDataCapnList(seg, len(src.PeerInfo))
		plist := capn.PointerList(typedList)

		for i, elem := range src.PeerInfo {
			_ = plist.Set(i, capn.Object(PeerDataGoToCapn(seg, &elem)))
		}
		dest.SetPeerInfo(typedList)
	}

	dest.SetSignature(src.Signature)
	dest.SetPubKeysBitmap(src.PubKeysBitmap)
	dest.SetPreviousHash(src.PreviousHash)
	dest.SetStateRootHash(src.StateRootHash)

	return dest
}

// MetaBlockCapnToGo is a helper function to copy fields from a MetaBlockCapn object to a MetaBlock object
func MetaBlockCapnToGo(src capnp.MetaBlockCapn, dest *MetaBlock) *MetaBlock {
	if dest == nil {
		dest = &MetaBlock{}
	}
	dest.Nonce = src.Nonce()
	dest.Epoch = src.Epoch()
	dest.Round = src.Round()

	n := src.ShardInfo().Len()
	dest.ShardInfo = make([]ShardData, n)
	for i := 0; i < n; i++ {
		dest.ShardInfo[i] = *ShardDataCapnToGo(src.ShardInfo().At(i), nil)
	}
	n = src.PeerInfo().Len()
	dest.PeerInfo = make([]PeerData, n)
	for i := 0; i < n; i++ {
		dest.PeerInfo[i] = *PeerDataCapnToGo(src.PeerInfo().At(i), nil)
	}
	dest.Signature = src.Signature()
	dest.PubKeysBitmap = src.PubKeysBitmap()
	dest.PreviousHash = src.PreviousHash()
	dest.StateRootHash = src.StateRootHash()

	return dest
}

// GetNonce return header nonce
func (h *MetaBlock) GetNonce() uint64 {
	return h.Nonce
}

// GetEpoch return header epoch
func (h *MetaBlock) GetEpoch() uint32 {
	return h.Epoch
}

// GetRound return round from header
func (h *MetaBlock) GetRound() uint32 {
	return h.Round
}

// GetRootHash returns the roothash from header
func (h *MetaBlock) GetRootHash() []byte {
	return h.StateRootHash
}

// GetPrevHash returns previous block header hash
func (h *MetaBlock) GetPrevHash() []byte {
	return h.PreviousHash
}

// GetPubKeysBitmap return signers bitmap
func (h *MetaBlock) GetPubKeysBitmap() []byte {
	return h.PubKeysBitmap
}

// GetSignature return signed data
func (h *MetaBlock) GetSignature() []byte {
	return h.Signature
}

// SetNonce sets header nonce
func (h *MetaBlock) SetNonce(n uint64) {
	h.Nonce = n
}

// SetEpoch sets header epoch
func (h *MetaBlock) SetEpoch(e uint32) {
	h.Epoch = e
}

// SetRound sets header round
func (h *MetaBlock) SetRound(r uint32) {
	h.Round = r
}

// SetRootHash sets root hash
func (h *MetaBlock) SetRootHash(rHash []byte) {
	h.StateRootHash = rHash
}

// SetPrevHash sets prev hash
func (h *MetaBlock) SetPrevHash(pvHash []byte) {
	h.PreviousHash = pvHash
}

// SetPubKeysBitmap sets publick key bitmap
func (h *MetaBlock) SetPubKeysBitmap(pkbm []byte) {
	h.PubKeysBitmap = pkbm
}

// SetSignature set header signature
func (h *MetaBlock) SetSignature(sg []byte) {
	h.Signature = sg
}

// SetTimeStamp sets header timestamp
func (h *MetaBlock) SetTimeStamp(ts uint64) {
	// TODO implement me
}

// SetCommitment sets header commitment
func (h *MetaBlock) SetCommitment(commitment []byte) {
	// TODO implement me - it is implemented in another pull request
}
