package integrationTests

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ElrondNetwork/elrond-go/crypto"
	ed25519SingleSig "github.com/ElrondNetwork/elrond-go/crypto/signing/ed25519/singlesig"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// TestWalletAccount creates and account with balance and crypto necessary to sign transactions
type TestWalletAccount struct {
	SingleSigner      crypto.SingleSigner
	BlockSingleSigner crypto.SingleSigner
	SkTxSign          crypto.PrivateKey
	PkTxSign          crypto.PublicKey
	PkTxSignBytes     []byte
	KeygenTxSign      crypto.KeyGenerator
	KeygenBlockSign   crypto.KeyGenerator

	Address state.AddressContainer
	Nonce   uint64
	Balance *big.Int
}

// CreateTestWalletAccount creates an wallet account in a selected shard
func CreateTestWalletAccount(coordinator sharding.Coordinator, shardId uint32) *TestWalletAccount {
	testWalletAccount := &TestWalletAccount{}
	testWalletAccount.initCrypto(coordinator, shardId)
	return testWalletAccount
}

// CreateTestWalletAccountWithKeygenAndSingleSigner creates a wallet account in a selected shard
func CreateTestWalletAccountWithKeygenAndSingleSigner(
	coordinator sharding.Coordinator,
	shardId uint32,
	blockSingleSigner crypto.SingleSigner,
	keyGenBlockSign crypto.KeyGenerator,
) *TestWalletAccount {
	twa := CreateTestWalletAccount(coordinator, shardId)
	twa.KeygenBlockSign = keyGenBlockSign
	twa.BlockSingleSigner = blockSingleSigner

	return twa
}

// initCrypto initializes the crypto for the account
func (twa *TestWalletAccount) initCrypto(coordinator sharding.Coordinator, shardId uint32) {
	twa.SingleSigner = &ed25519SingleSig.Ed25519Signer{}
	twa.BlockSingleSigner = &mock.SignerMock{
		VerifyStub: func(public crypto.PublicKey, msg []byte, sig []byte) error {
			return nil
		},
	}
	sk, pk, keyGen := GenerateSkAndPkInShard(coordinator, shardId)

	pkBuff, _ := pk.ToByteArray()
	fmt.Printf("Found pk: %s in shard %d\n", hex.EncodeToString(pkBuff), shardId)

	twa.SkTxSign = sk
	twa.PkTxSign = pk
	twa.PkTxSignBytes, _ = pk.ToByteArray()
	twa.KeygenTxSign = keyGen
	twa.KeygenBlockSign = &mock.KeyGenMock{}
	twa.Address, _ = TestAddressPubkeyConverter.CreateAddressFromBytes(twa.PkTxSignBytes)
}

// LoadTxSignSkBytes alters the already generated sk/pk pair
func (twa *TestWalletAccount) LoadTxSignSkBytes(skBytes []byte) {
	newSk, _ := twa.KeygenTxSign.PrivateKeyFromByteArray(skBytes)
	newPk := newSk.GeneratePublic()

	twa.SkTxSign = newSk
	twa.PkTxSign = newPk
	twa.PkTxSignBytes, _ = newPk.ToByteArray()
	twa.Address, _ = TestAddressPubkeyConverter.CreateAddressFromBytes(twa.PkTxSignBytes)
}
