# Elrond go sandbox

The go implementation for the Elrond Network testnet

# Getting started

### Prerequisites

Building the repository requires Go (version 1.12 or later)

### Installation and running

Run in  %project_folder%/cmd/seednode folder the following command to build a seednode (first node in the network
 used for bootstrapping the nodes):
 
 ```
 $ go build
 $ ./seednode
 ```
 
Run in  %project_folder%/cmd/node folder the following command to build a node:

```
$ go build
$ ./node -port 23000 -tx-sign-sk "b5671723b8c64b16b3d4f5a2db9a2e3b61426e87c945b5453279f0701a10c70f" -sk "7964df79735f1ce9dff69cc2ec7c87e499a9fcab6808ee9ab4fc65443a9ff171"
```

For the network customization please take a look in the p2p.toml
For the consensus group customization please take a look in the /cmd/node/nodesSetup.json
To run multiple nodes take a look at the end of this document.

### Running the tests
```
$ go test ./...
```

# Progress
### Done
- [x] Cryptography
  - [x] Schnorr Signature
  - [x] Belare-Neven Signature
  - [x] BLS Signature
  - [x] Modified BLS Multi-signature
- [x] Datastructures
  - [x] Transaction
  - [x] Block
  - [x] Account
  - [x] Trie
- [x] Execution
  - [x] Transaction
  - [x] Block
  - [x] State update
  - [x] Synchronization
  - [x] Shard Fork choice
- [x] Peer2Peer - libp2p
- [x] Consensus - SPoS
- [x] Sharding - fixed number
  - [x] Transaction dispatcher 
  - [x] Transaction
  - [x] State
- [x] MetaChain
  - [x] Data Structures
  - [x] Block Processor
  - [x] Interceptors/Resolvers
- [x] VM - K-Framework
  - [x] K Framework go backend
  - [x] IELE Core
  - [x] IELE Core tests
- [x] Smart Contracts on a Sharded Architecture
  - [x] Concept reviewed
- [x] Governance
  - [x] Concept reviewed
- [x] Testing 
  - [x] Unit tests
  - [x] Integration tests
  - [x] TeamCity continuous integration
  - [x] Manual testing

### In progress
- [ ] Sharding - fixed number
  - [ ] Nodes dispatcher (shuffling)
  - [ ] Network
- [ ] MetaChain Consensus
- [ ] VM - K-Framework
  - [ ] IELE Adapter
  - [ ] EVM Core
  - [ ] EVM Core tests
  - [ ] EVM Adapter
- [ ] Smart Contracts on a Sharded Architecture
  - [ ] VM integration
  - [ ] SC Deployment
  - [ ] Dependency checker + SC migration
  - [ ] Storage rent + SC backup & restore
  - [ ] Request-response fallback
- [ ] Fee structure
- [ ] Adaptive State Sharding
  - [ ] Splitting
  - [ ] Merging 
  - [ ] Redundancy
- [ ] Privacy
- [ ] DEX integration
- [ ] Interoperability
- [ ] Optimizations
  - [ ] Randomness
  - [ ] Consensus
  - [ ] Smart Contract 
- [ ] Governance
  - [ ] SC for ERD IP
  - [ ] Enforced Upgrade mechanism for voted ERD IP
- [ ] Testing
  - [ ] Automate tests with AWS 
- [ ] Bugfixing

# Private/Public Keys for testing

```
# Start n instances in a group of <consensusGroupSize> (nodesSetup.json) from a total of <initialNodes>
# (nodesSetup.json) validators, with round duration of <roundDuration> (nodesSetup.json) miliseconds

# Private keys for validators (BLS):

#  1: 7964df79735f1ce9dff69cc2ec7c87e499a9fcab6808ee9ab4fc65443a9ff171
#  2: 1a5bd59b78e27dd165a0bdea7edd82af8b7de16ff102fad5ed1a17cafdd4bc1f
#  3: 6747e8fdfe77ea315e79358dec6c524db9e26ea3b334292253a275af1251a594
#  4: 19ac4f93739b20a7b33e0864df66096945959424d147886e277dbc527f35fd67
#  5: 14fdb1e100dad84db9d1a93494f704496478ad8a7b2f6ec818a2135dcaa4a0b9
#  6: 8b1166af09d8ebb9476cd47ed55cf73eea05fd93001399d20fd37198e6450be8
#  7: 26339affab7b031b1762f4642dc59733e57ae3c9f0f5320e6a68dc9e39786228
#  8: 070d9af1bd9614cd5e838edf833e1227a8a3908b7a8194670a94d2c8bbfc50fa
#  9: 5ff8ab7b3d64f394a0cd24dcf29d518a44067be9c6342dbc7ac1846804d4337e
# 10: 37fb9521d4488dc51f18e6bd85f499d4155427dfd9577c08a169859a9a6d73df
# 11: 25774ac581f953fee0f699bbb88604ac02919ab410ef2c1e9a9aa23909a2e60b
# 12: 256e6676f34809bd26b33f94b63001d911a04922834ad421e21a3fdb34609ffa
# 13: 782da6f7a741fb0e1ada1d8e7d179e4bcdb6fdd8c807b5d14d24fbe11db7b8d0
# 14: 53852c6c8e4c015bd2e7e25df6d3e4abf6f603532fd4764c964c676d81ba9ea7
# 15: 2a0e9ce582e0fa86dd7f9661a2ce1fece751ad8a64ee55788b0b24c289677d57
# 16: 59c3dc82794e55b7aeff586fb8c3c779f72f7d9764145fa68c45f4c42232fc75
# 17: 5ff05efdc3fef6b10af8fa380c3819c949cea89ac862449303d5eb284381d7e5
# 18: 7b0a0fbba05637dd55c227751c1cc022bd4eaf8900f86800d478a295588db5f6
# 19: 551e6dc6f3aca609811c5be778f1f0e0b8cb8494036c2c9c74b48ec9473dfeb8
# 20: 4cdbe9bad3f53156179eba436cac3b9a7a1a33f7f1eda6bdd15372a67ddce3c2
# 21: 65d93daf71cb8d1c7458d662c07410cfa0385e106217f246c7035b4f9cc75400

# Public keys for validators (BLS):

#  1: 600eb1aeecf1264a15f1639cef05d3a26820720b74de93ccaea128fb55e1dbd312baa4c6c28393f36ec634d48bee642edca6b35e5088b1362bf4709b6b65bf8e42a45a21956eacbcfcfbb885524b4cf94e4a7356429535ac75b10dc9eea184c08b2a1d96f2de1f2fecb483e1843966c76473db67362ea47a0c1c728f769dd4d6
#  2: 0d76fb0af4e10da8a833438435abd797f05d72df42bd52f1126a2e4af22955c3148b512eb404ee5ec11a48feefdaa55ee2148cdf4b1b352fa5c061defc2e119117d18da1c0f730f0e72131076b84671fe6bfe6cd7a1a4fcd6919201509816d965000b8d69d77cbf3a90be77c2a27bac202dca07a1b7257b63a7bb66f79333b90
#  3: 1e392bdf0c41d59e9dd59b22b75518aa209f938ae48788cd341d8b9b68cc91518667334148bf35ed205c75764e768ba2ff2349bdf451abf3d9daca3bfd797ec7425ccae5afe414b6af15f358f60b4caf4e8f128fa162ce72265dae923e98f5de36666be84619ebedbe9cb913201013843aa1c8c1f04eb5c635de6f54c64698c0
#  4: 385ce983c15f2fc8368b6d31f3f9fa9e0f350cc9d87a51d428226c5e530487ae5fbe2a943abdb6f9f1dc0759b307b8ffa954eb781731a8a5f6fcb656fb9ec68219e1101e37a6794209ed99019ef5e00aaef64e727780daddd3865f0f26a1734669d69fbdb679a07ce6a83a6d9a5bead06386e644f00223ba958e6f2efd91c4ea
#  5: 60900538f372a3eb375efbee09dbab54ded3c98334466595c215a00c1714deed1f8ad472f5adf59f747902f58e24eab06043694e40330d6017288d00a2d150f862513e0d55ac96b2385188ec36c4b676dbd4dea652a38df8c2b389b6bef2f4cd593fce7a6d6f0f88333f6d345e355f3aa3e887f3b0a549990d91747ea302d047
#  6: 44163b060ded1fbcfacde3614ca54c77c554e76f628fb18bf1d21ec57daa5e546607748b4f1c6d2b64849b26d49c5b9918d470d5dd841f5f8daa0496f283a3d120c68f46f9b93dcfb362a557cfd15d780234dc5ad02b4e241d84c0e941463b30112566cf2d66679606aab91c62f1f3db2d382f9b9f97ad7570a283b7b7be1a7a
#  7: 5403bc8dedcdc967b488e0893a3ec5e8b818f3b395f55926f6bfc81992e1443e259b181cf55ba24f4fd5ff451778b8e7d4e6dc80b6759d1405dc236d03686bcc55c93d038566648047e51a00c1d2729a8ca3274dc6206d88be2d145b16b94bcd638f5345e20b73a05dd082e3f6242abda69e46b4d80f34ff28a7821f55a11eca
#  8: 8a05ca3c771abb152b4e0371977477b93c177357a3cecc930aab7cb97245147d88fd3cfa3186736c34172414a58d40018e2b616be70390edc6dfa3b05d2f84e0695434f259b62877d1b395df256e8ab7baba85cf1bf1605bb4190a42574f018238cdf51469588242241b5e2c423e8c3c1704b1c96fb57821516a1bb3c06e3e6a
#  9: 0ddf722b8a293957feb2bb5f46932522b2aa0b22ba970ed45676edb12c495e6321a87b251b281cba79da55ef8d77348b2a1743d49889ff8edb5d4f1395423fd923450e0692849a4e192ad571914aed67d8ab083f57d85f8c8a5b1d1d065111ec1816fef4fb2833764ed62810815215723e0eb0693f17b8fe9551103b3e6b3fd4
# 10: 0718ef3d13530fa4788926366bacbc7ed9301e40bea3eb95a015916b480d96ec43bf545e367f6a87e24f4937947b7bab2020b7dc122074e49b7ee57d31a488127b3595b15a49448c8bf4b1b7e2b22c85991e4277bcf3f802279a8da2e300dd363c44105cf31c1f936564adff01529e16ef0d0f57e66bc3ea8077b02692163b59
# 11: 8b0b394b66f6698879ae1ae334905715ca8ce5c6aad390d126883d43f37c9a7d0ddb8fad3a58d821da9aac914f0fd0b8d97545d3ca4dd495e15f1f58b77ae38e7166c6952f492e1853ad2ecd064713ec94d8abe80894dfbc952ee198598019c61c112a8682978f10d905cc7b88edfdfc7a51c83e48ad7d0a7704326a7aef1913
# 12: 24b18a8b07dfd9da1ac5d4800b0e7f214fe95deef858b59572bd8b8fde6c20e41bab016ee3021c2b61652925689163900ffa8a0744bec6ad1b322f97d3721586134a3a97aeaa51b5a5bc047090b384da0140308bc760ebaa52c1585ce90152153c2a65a911c01687ec0a3ccfc57c0e242a4bbe054735c73323a06ed0bddfb386
# 13: 5565fd7d3c2f92881fb3a34c13790a547996cd4dc76c1d03e4c23ca359e948041957f3d86a925a92b67397fff4c5e0246f131f0eed38b36c78eb6b576323e00540c7147c8364142c285a9c3bde89279893206affcb952790304b1ef58dd5a26721a089b7863c647e4abee567baba2a5babdc1858a3cca1c87a7966c916bd1997
# 14: 20a7accc3129607604f251b567ad63b778a3681e1fd52ffd65f70688f996661a8e5d6a478f53a718b79c4cd4fe95b22670992886fb4f442520971d080ec49a0818a9dfff8709e4ea0dcdb4610972b7311b76488d8ff95580f84b049081af4ca26f88b3ba74d25367e2fe76de2af60e0697daa586b1e3742e81994fab1ee8a2ac
# 15: 1c55eebb1b91d7d64b15c9e352d95b24ade6ccb6ee22f2f5c2a4a02c04cbca2423ce0239dcbcc2436178236e25fcc2934ef33a98d93631169240dda8ff90cee2656ce03f04eeaa028c255393a7d6f6d00718a1041c17fdd782f3188c9cac6d521a2398e8bdf93e6c34961486dd79be65cf053d83f8480731e431e1067a3b6c9b
# 16: 63c326868c95cd798bb0b75f282ff34375884e9397d336883b2c4f97b0df93427c0d6b4a974d4b05ad503099774a27507c12e330213071774fbb9d1e345835ca768f855e027ca19ea4c980b8520274a5d50c2b5630ff64d8c4c7f56c05c0b3ab0763ad8ae118994183947e32a903a3ba81f008cda764be292b5eec33062fa2bf
# 17: 280d39fdd86039d92792d81db0ee32f4709e3492050a71840572d7e4d6b17e15409505f1623ec5459e04c6940a1f6a18a6f7cd3ff9028dd835fccf1047bba60722f492260e1cd68d325ef85bf48b363ecf4014f7d2e312d474e3441434a0fcd620a5d39915ef191cf5801858e3b4f512cd8998562fc2680a011b8857fa6226a8
# 18: 7864dead30e99d9954a828299559ab0f1e9ce29436a1cd040cb77e093b4249bb8b00d817a4f19be3c57d790af1b1789ff36ee75527ff52c7d3af3483cc5ada0416924cf2c4534e68ce4cc66512817b258e687845864e3be875d138eb877f06c63d12e169ddc65aeba056bc340d8e9ad1d0c82a9d67078bdd9b12f1027265d555
# 19: 8325d3cb472e89c76737ee3e2d92dcd87b0482f9a7ab011f984ad65da0b2319d349d504c34c669197f5b42f975fc24123cc0615bbfdab87cda47ab15a432cbda8abf1a074c8a21fc4b3098ec176b21e3116dcf2771cd3d137bab25774e1abc5e348a39022a4b2fd650565c8641069cde13321b5fdb3ae45e37116fe6a0231ee9
# 20: 464c96aa19fd1a47dfc30d5caeb7d15efd6cd28e04f622a30ef2157acc31c2ce3767d6d7715725ab8c2f9b22d6d8abe71632a85fbd8a655abec64f614b6e745b6e6a0a9c59486de4b56cb9e7c75f8fdd0355cf9a89b6bee0def588763c04ad3e7dfbcb6f6cd16a5c0044cd7c7a5cb9bf8fd2ac76f6e2a92a25c13f403ad9e0cf
# 21: 36f331cef8a0bbbcc9525798594453b60a98c8c983ea475688541ce598ef64123d4d34245a0231ad66b7bdfcb8061d5918c6466607cd7c3fd31620305bb5d5b95381216f1b359a3d1a5a19bab81b0c9f4b1cca07766b54c93bd8abd4fb43ab5b0a04129927de6efd1339eac0787e47c3cd85732d145cd645245b780937046f28
      
# Private keys for signing txs:

#  1: b5671723b8c64b16b3d4f5a2db9a2e3b61426e87c945b5453279f0701a10c70f
#  2: 8c7e9c79206f2bf7425050dc14d9b220596cee91c09bcbdd1579297572b63109
#  3: 52964f3887b72ea0bd385ee3b60d2321034256d17cb4eec8333d4a4ce1692b08
#  4: 2c5a2b1d724c21be3aebf46dfa1db841b8b58d063066c19e983ff03b3f955f08
#  5: 6532ccdb32e95ca3f4e5fc5b43c41dda610a469f7f18c76c278b9b559779300a
#  6: 772f371cafb44da6ade4af11c5799bd1c25bbdfb17335f4fc102a81b2d66cc04
#  7: 12ffff943b39b21f1c5f1455e06a2ab60d442ff9cb65451334551a0e84049409
#  8: a7160d033389e99198331a4c9e3c7417722ecc29246f42049335e972e4df5b0f
#  9: 9cf7b345fdf3c6d2de2d6b28cc0019c02966ef88774069d530b636f760292c00
# 10: f236b2f60ad8864ea89fd69bf74ec65f64bd82c2f310b81a8492ba93e8b6c402
# 11: 0f04b269d382944c5c246264816567c9a33b2a9bf78f075d9a17b13e7b925603
# 12: 8cf6e6aeb878ef01399e413bc7dd788a69221a37c29021bd3851f2f5fe67f203
# 13: c7f48a69e4b2159fe209bdb4608410516f28186ad498ca78b16d8b2bebfb1f0f
# 14: 7579d506ff015e5e720b2e75e784c13a4662f48b6e2038af6e902b1157239101
# 15: b7877c28e394ab4c89d80e8b2818ef1346ee8c0fdd6566a6d27088ad097e4f05
# 16: 055ae06aad2c7f8d50ecd4bd7c4145cb19636b0b0126ffa4ee1326afb3876000
# 17: c47b89db3e3ad067863af5a7b7f9e9dec0e47516e87d5d6d744e0af581a79404
# 18: 843c4bea60b629fae50a0334ba9c7284f886b90502b740c8f95ab13a36a08c0e
# 19: 92561fd546014adcd13ff7776829f1c8c0886e83eb04fb723fc3636da8f2960b
# 20: 22a3922963cc1fe57a59178f021282223a8742fb4476f7a5c5b4c2c2aa2d4f0f
# 21: 02c9d56e503857832c07d78b0d75aabb8e6c109e9cec641b8681afaee2c9a701

# Public keys for signing txs:

#  1: 5126b6505a73e59a994caa8f556f8c335d4399229de42102bb4814ca261c7419
#  2: 8e0b815be8026a6732eea132e113913c12e2f5b19f25e86a8403afffbaf02088
#  3: e6ec171959063bd0d61f95a52de73d6a16e649ba5fa8b12663b092b48cc99434
#  4: 20ccf92c80065a0f9ce1f1b8e3dee1e0b9774f4eebf2af7e8fa3ac503923360d
#  5: 9a3b8e67f42aef9544e0888ea9daee77af90292c86336203d224691d55306d08
#  6: 0740bccedc28084ab811065cb618fec4ee623384b4b3d5466190d11ff6d77007
#  7: 0ccba0f98829ea9f337035a1f7b13cbd8e9ffb94f2c538e2cafb34ca7f2bcd24
#  8: d9e9596c28a3945253d46bc1b9418963c0672a26a0b40ee7372cb9ec34d1ee07
#  9: 86fbd8606e73b7a4f45a51b443270f3050aff571a29b9804d2444c081560d1dd
# 10: 2084f2493e68443a5b156ec42a8cd9072c47aa453df4acd20524792a4fd9f474
# 11: f91d24256d918144aaacfa641cd113af05d56cfb7a5b8ba5885ebd8edd43fe1e
# 12: e8d4bcfe91c3c7788d8ab3704b192229900ec3fe3f1eb6f841c440e223d401a0
# 13: 4bf7ee0e17a0b76d3837494d3950113d3e77db055b2c07c9cb443f529d73c8e3
# 14: 20f12f7bdd4ab65321eb58ce8f90eec733e3e9a4cc9d6d5d7e57d2e86c6c2c76
# 15: 34cf226f4d62a22e4993a1a2835f05a4bb2fb48304e16f2dc18f99b39c496f7d
# 16: b9f0fc3e1baa49c027205946af7d6c79b749481e5ab766356db3b878c0929558
# 17: 6670b048a3f9d93fdacb4d60ff7c2f3bd7440d5175ca8b9d2475a444cd7a129b
# 18: d82b3f4490ccb2ffbba5695c1b7c345a5709584737a263999c77cc1a09136de1
# 19: 29ba49f47e2b86b143418db31c696791215236925802ea1f219780e360a8209e
# 20: 199866d09b8385023c25f261460d4d20ae0d5bc72ddf1fa5c1b32768167a8fb0
# 21: 0098f7634d7327139848a0f6ad926051596e5a0f692adfb671ab02092b77181d

# Ex1:
numOfNodes=10
for i in $(seq 1 $numOfNodes)
do
index=$(( $i - 1 ))
offset=23000
port=$(( $offset + $index ))
gnome-terminal -- ./node -port $port -tx-sign-sk-index $index -sk-index $index -num-of-nodes $numOfNodes
done

# Ex2:
gnome-terminal -- ./node -port 23000 -tx-sign-sk "b5671723b8c64b16b3d4f5a2db9a2e3b61426e87c945b5453279f0701a10c70f" -sk "7964df79735f1ce9dff69cc2ec7c87e499a9fcab6808ee9ab4fc65443a9ff171"
gnome-terminal -- ./node -port 23001 -tx-sign-sk "8c7e9c79206f2bf7425050dc14d9b220596cee91c09bcbdd1579297572b63109" -sk "1a5bd59b78e27dd165a0bdea7edd82af8b7de16ff102fad5ed1a17cafdd4bc1f"
gnome-terminal -- ./node -port 23002 -tx-sign-sk "52964f3887b72ea0bd385ee3b60d2321034256d17cb4eec8333d4a4ce1692b08" -sk "6747e8fdfe77ea315e79358dec6c524db9e26ea3b334292253a275af1251a594"
```