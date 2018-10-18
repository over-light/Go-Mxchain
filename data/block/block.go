package block

import (
	"io"

	"math/rand"

	"github.com/ElrondNetwork/elrond-go-sandbox/data"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/block/capnproto1"
	"github.com/glycerine/go-capnproto"
)

type MiniBlock struct {
	TxHashes    [][]byte
	DestShardID uint32
}

type Block struct {
	MiniBlocks []MiniBlock
}

type Header struct {
	Nonce    []byte
	PrevHash []byte
	// temporary keep list of public keys of signers in header
	// to be removed later
	PubKeys    [][]byte
	ShardId    uint32
	TimeStamp  []byte
	Round      uint32
	BlockHash  []byte
	Signature  []byte
	Commitment []byte
}

func (s *Block) Save(w io.Writer) error {
	seg := capn.NewBuffer(nil)
	BlockGoToCapn(seg, s)
	_, err := seg.WriteTo(w)
	return err
}

func (s *Block) Load(r io.Reader) error {
	capMsg, err := capn.ReadFromStream(r, nil)
	if err != nil {
		//panic(fmt.Errorf("capn.ReadFromStream error: %s", err))
		return err
	}
	z := capnproto1.ReadRootBlockCapn(capMsg)
	BlockCapnToGo(z, s)
	return nil
}

func BlockCapnToGo(src capnproto1.BlockCapn, dest *Block) *Block {
	if dest == nil {
		dest = &Block{}
	}

	var n int

	// MiniBlocks
	n = src.MiniBlocks().Len()
	dest.MiniBlocks = make([]MiniBlock, n)
	for i := 0; i < n; i++ {
		dest.MiniBlocks[i] = *MiniBlockCapnToGo(src.MiniBlocks().At(i), nil)
	}

	return dest
}

func BlockGoToCapn(seg *capn.Segment, src *Block) capnproto1.BlockCapn {
	dest := capnproto1.AutoNewBlockCapn(seg)

	// MiniBlocks -> MiniBlockCapn (go slice to capn list)
	if len(src.MiniBlocks) > 0 {
		typedList := capnproto1.NewMiniBlockCapnList(seg, len(src.MiniBlocks))
		plist := capn.PointerList(typedList)
		i := 0
		for _, ele := range src.MiniBlocks {
			plist.Set(i, capn.Object(MiniBlockGoToCapn(seg, &ele)))
			i++
		}
		dest.SetMiniBlocks(typedList)
	}

	return dest
}

func (s *Header) Save(w io.Writer) error {
	seg := capn.NewBuffer(nil)
	HeaderGoToCapn(seg, s)
	_, err := seg.WriteTo(w)
	return err
}

func (s *Header) Load(r io.Reader) error {
	capMsg, err := capn.ReadFromStream(r, nil)
	if err != nil {
		//panic(fmt.Errorf("capn.ReadFromStream error: %s", err))
		return err
	}
	z := capnproto1.ReadRootHeaderCapn(capMsg)
	HeaderCapnToGo(z, s)
	return nil
}

func HeaderCapnToGo(src capnproto1.HeaderCapn, dest *Header) *Header {
	if dest == nil {
		dest = &Header{}
	}

	var n int

	// Nonce
	dest.Nonce = src.Nonce()

	// PrevHash
	dest.PrevHash = src.PrevHash()

	// PubKeys
	n = src.PubKeys().Len()
	dest.PubKeys = make([][]byte, n)
	for i := 0; i < n; i++ {
		dest.PubKeys[i] = src.PubKeys().At(i)
	}

	// ShardId
	dest.ShardId = src.ShardId()

	// TimeStamp
	dest.TimeStamp = src.TimeStamp()

	// Round
	dest.Round = src.Round()

	// BlockHash
	dest.BlockHash = src.BlockHash()

	// Signature
	dest.Signature = src.Signature()

	// Commitment
	dest.Commitment = src.Commitment()

	return dest
}

func HeaderGoToCapn(seg *capn.Segment, src *Header) capnproto1.HeaderCapn {
	dest := capnproto1.AutoNewHeaderCapn(seg)
	dest.SetNonce(src.Nonce)
	dest.SetPrevHash(src.PrevHash)
	myList3 := seg.NewDataList(len(src.PubKeys))
	for i := range src.PubKeys {
		myList3.Set(i, src.PubKeys[i])
	}

	dest.SetPubKeys(myList3)
	dest.SetShardId(src.ShardId)
	dest.SetTimeStamp(src.TimeStamp)
	dest.SetRound(src.Round)
	dest.SetBlockHash(src.BlockHash)
	dest.SetSignature(src.Signature)
	dest.SetCommitment(src.Commitment)

	return dest
}

func (s *MiniBlock) Save(w io.Writer) error {
	seg := capn.NewBuffer(nil)
	MiniBlockGoToCapn(seg, s)
	_, err := seg.WriteTo(w)
	return err
}

func (s *MiniBlock) Load(r io.Reader) error {
	capMsg, err := capn.ReadFromStream(r, nil)
	if err != nil {
		//panic(fmt.Errorf("capn.ReadFromStream error: %s", err))
		return err
	}
	z := capnproto1.ReadRootMiniBlockCapn(capMsg)
	MiniBlockCapnToGo(z, s)
	return nil
}

func MiniBlockCapnToGo(src capnproto1.MiniBlockCapn, dest *MiniBlock) *MiniBlock {
	if dest == nil {
		dest = &MiniBlock{}
	}

	var n int

	// TxHashes
	n = src.TxHashes().Len()
	dest.TxHashes = make([][]byte, n)
	for i := 0; i < n; i++ {
		dest.TxHashes[i] = src.TxHashes().At(i)
	}

	dest.DestShardID = src.DestShardID()

	return dest
}

func MiniBlockGoToCapn(seg *capn.Segment, src *MiniBlock) capnproto1.MiniBlockCapn {
	dest := capnproto1.AutoNewMiniBlockCapn(seg)

	mylist1 := seg.NewDataList(len(src.TxHashes))

	for i := range src.TxHashes {
		mylist1.Set(i, src.TxHashes[i])
	}

	dest.SetTxHashes(mylist1)
	dest.SetDestShardID(src.DestShardID)

	return dest
}

// GenerateDummyArray is used for tests to generate an array of blocks with dummy data
func (blk *Block) GenerateDummyArray() []data.CapnpHelper {
	blocks := make([]data.CapnpHelper, 0, 100)
	for i := 0; i < 100; i++ {
		lenMini := rand.Intn(20) + 1
		miniblocks := make([]MiniBlock, 0, lenMini)
		for j := 0; j < lenMini; j++ {
			lenTxHashes := rand.Intn(20) + 1
			txHashes := make([][]byte, 0, lenTxHashes)
			for k := 0; k < lenTxHashes; k++ {
				txHashes = append(txHashes, []byte(data.RandomStr(32)))
			}
			miniblock := MiniBlock{
				DestShardID: uint32(rand.Intn(20)),
				TxHashes:    txHashes,
			}

			miniblocks = append(miniblocks, miniblock)
		}
		bl := Block{MiniBlocks: miniblocks}
		blocks = append(blocks, &bl)
	}

	return blocks
}

// GenerateDummyArray is used for tests to generate an array of headers with dummy data
func (h *Header) GenerateDummyArray() []data.CapnpHelper {
	headers := make([]data.CapnpHelper, 0, 1000)

	pkList := make([][]byte, 0, 21)

	for i := 0; i < 21; i++ {
		pkList = append(pkList, []byte(data.RandomStr(32)))
	}

	for i := 0; i < 1000; i++ {
		headers = append(headers, &Header{
			Nonce:      []byte(data.RandomStr(4)),
			PrevHash:   []byte(data.RandomStr(32)),
			ShardId:    uint32(rand.Intn(20)),
			TimeStamp:  []byte(data.RandomStr(20)),
			Round:      uint32(rand.Intn(20000)),
			BlockHash:  []byte(data.RandomStr(32)),
			Signature:  []byte(data.RandomStr(32)),
			Commitment: []byte(data.RandomStr(32)),
			PubKeys:    pkList,
		})
	}

	return headers
}
