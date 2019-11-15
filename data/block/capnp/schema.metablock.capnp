@0xddffc3d7f7f36183;
using Go = import "/go.capnp";
using MiniBlockHeaderCapn = import "./schema.capnp".MiniBlockHeaderCapn;
$Go.package("capnp");
$Go.import("_");

struct PeerDataCapn {
    address   @0: Data;
    publicKey @1: Data;
    action    @2: UInt8;
    timestamp @3: UInt64;
    value     @4: Data;
}

struct ShardMiniBlockHeaderCapn {
   hash            @0: Data;
   receiverShardId @1: UInt32;
   senderShardId   @2: UInt32;
   txCount         @3: UInt32;
}

struct ShardDataCapn {
    shardId               @0: UInt32;
    headerHash            @1: Data;
    shardMiniBlockHeaders @2: List(ShardMiniBlockHeaderCapn);
    prevRandSeed          @3:  Data;
    pubKeysBitmap         @4: Data;
    signature             @5: Data;
    txCount               @6: UInt32;
    round                 @7: UInt64;
    prevHash              @8: Data;
    nonce                 @9: UInt64;
}

struct MetaBlockCapn {
    nonce                  @0:  UInt64;
    epoch                  @1:  UInt32;
    round                  @2:  UInt64;
    timeStamp              @3:  UInt64;
    shardInfo              @4:  List(ShardDataCapn);
    peerInfo               @5:  List(PeerDataCapn);
    signature              @6:  Data;
    pubKeysBitmap          @7:  Data;
    prevHash               @8:  Data;
    prevRandSeed           @9:  Data;
    randSeed               @10: Data;
    rootHash               @11: Data;
    validatorStatsRootHash @12: Data;
    txCount                @13: UInt32;
    miniBlockHeaders       @14: List(MiniBlockHeaderCapn);
}

##compile with:

##
##
##   capnpc  -I$GOPATH/src/github.com/glycerine/go-capnproto -ogo $GOPATH/src/github.com/ElrondNetwork/elrond-go/data/block/capnp/schema.metablock.capnp
