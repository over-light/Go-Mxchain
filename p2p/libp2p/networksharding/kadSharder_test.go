package networksharding

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/p2p"
	"github.com/ElrondNetwork/elrond-go/p2p/mock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

const (
	testNodesCount = 1000
)

func fakeShard0(_ p2p.PeerID) uint32 {
	return 0
}

func fakeShardBit0Byte2(id p2p.PeerID) uint32 {
	ret := sha256.Sum256([]byte(id))
	return uint32(ret[2] & 1)
}

type testKadResolver struct {
	f func(p2p.PeerID) uint32
}

func (tkr *testKadResolver) ByID(peer p2p.PeerID) uint32 {
	return tkr.f(peer)
}

func (tkr *testKadResolver) NumShards() uint32 {
	return 3
}

func (tkr *testKadResolver) IsInterfaceNil() bool {
	return tkr == nil
}

var (
	fs0  = &testKadResolver{fakeShard0}
	fs20 = &testKadResolver{fakeShardBit0Byte2}
)

func TestNewKadSharder_ZeroPrioBitsShouldErr(t *testing.T) {
	t.Parallel()

	ks, err := NewKadSharder(0, &mock.PeerShardResolverStub{})

	assert.True(t, check.IfNil(ks))
	assert.True(t, errors.Is(err, ErrBadParams))
}

func TestNewKadSharder_NilPeerShardResolverShouldErr(t *testing.T) {
	t.Parallel()

	ks, err := NewKadSharder(1, nil)

	assert.True(t, check.IfNil(ks))
	assert.True(t, errors.Is(err, p2p.ErrNilPeerShardResolver))
}

func TestNewKadSharder_ShouldWork(t *testing.T) {
	t.Parallel()

	ks, err := NewKadSharder(1, &mock.PeerShardResolverStub{})

	assert.False(t, check.IfNil(ks))
	assert.Nil(t, err)
}

func TestCutoOffBits(t *testing.T) {
	i := []byte{0xff, 0xff}[:]

	testData := []struct {
		l   uint32
		exp *big.Int
	}{
		{
			l:   1,
			exp: big.NewInt(0x7f<<8 | 0xff),
		},
		{
			l:   2,
			exp: big.NewInt(0x3f<<8 | 0xff),
		},
		{
			l:   3,
			exp: big.NewInt(0x1f<<8 | 0xff),
		},
		{
			l:   7,
			exp: big.NewInt(0x1<<8 | 0xff),
		},
		{
			l:   8,
			exp: big.NewInt(0xff),
		},

		{
			l:   9,
			exp: big.NewInt(0xff),
		},
	}

	for _, td := range testData {
		tdCopy := td
		t.Run(fmt.Sprint(tdCopy.l, "_", tdCopy.exp), func(t *testing.T) {
			k, _ := NewKadSharder(tdCopy.l, fs0)
			r := k.resetDistanceBits(i)
			assert.Equal(t, big.NewInt(0).SetBytes(r), tdCopy.exp, "Should match")
		})
	}
}

func TestKadSharderDistance(t *testing.T) {
	s, _ := NewKadSharder(8, fs0)
	checkDistance(s, t)
}

func TestKadSharderOrdering2(t *testing.T) {
	s, _ := NewKadSharder(2, fs20)
	checkOrdering(s, t)
}

func TestKadSharderOrdering2_list(t *testing.T) {
	s, _ := NewKadSharder(4, fs20)

	peerList := make([]peer.ID, testNodesCount)
	for i := 0; i < testNodesCount; i++ {
		peerList[i] = peer.ID(fmt.Sprintf("NODE %d", i))
	}
	l1, _ := s.SortList(peerList, nodeA)

	refShardID := fakeShardBit0Byte2(p2p.PeerID(nodeA))
	sameShardScore := uint64(0)
	sameShardCount := uint64(0)
	otherShardScore := uint64(0)
	retLen := uint64(len(l1))
	fmt.Printf("[ref] %s , sha %x, shard %d\n", string(nodeA), sha256.Sum256([]byte(nodeA)), refShardID)
	for i, id := range l1 {
		shardID := fakeShardBit0Byte2(p2p.PeerID(id))

		if shardID == refShardID {
			sameShardScore += retLen - uint64(i)
			sameShardCount++
		} else {
			otherShardScore += retLen - uint64(i)
		}
	}

	avgSame := sameShardScore / sameShardCount
	avgOther := otherShardScore / (retLen - sameShardCount)
	fmt.Printf("Same shard avg score %d, Other shard avg score %d\n", avgSame, avgOther)

	assert.True(t, avgSame > avgOther)
}

func TestKadSharder_SetPeerShardResolverNilShouldErr(t *testing.T) {
	t.Parallel()

	ks, _ := NewKadSharder(1, &mock.PeerShardResolverStub{})

	err := ks.SetPeerShardResolver(nil)

	assert.Equal(t, p2p.ErrNilPeerShardResolver, err)
}

func TestKadSharder_SetPeerShardResolverShouldWork(t *testing.T) {
	t.Parallel()

	ks, _ := NewKadSharder(1, &mock.PeerShardResolverStub{})
	newPeerShardResolver := &mock.PeerShardResolverStub{}
	err := ks.SetPeerShardResolver(newPeerShardResolver)

	//pointer testing
	assert.True(t, ks.resolver == newPeerShardResolver)
	assert.Nil(t, err)
}
