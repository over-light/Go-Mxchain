package factory

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/crypto/signing"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/ed25519"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/ed25519/singlesig"
	mclmultisig "github.com/ElrondNetwork/elrond-go/crypto/signing/mcl/multisig"
	mclsig "github.com/ElrondNetwork/elrond-go/crypto/signing/mcl/singlesig"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/multisig"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/hashing/blake2b"
	"github.com/ElrondNetwork/elrond-go/hashing/sha256"
	"github.com/ElrondNetwork/elrond-go/sharding"
	systemVM "github.com/ElrondNetwork/elrond-go/vm/process"
)

// CryptoComponentsFactoryArgs holds the arguments needed for creating crypto components
type CryptoComponentsFactoryArgs struct {
	Config           *config.Config
	NodesConfig      NodesSetupHandler
	ShardCoordinator sharding.Coordinator
	KeyGen           crypto.KeyGenerator
	PrivKey          crypto.PrivateKey
}

type cryptoComponentsFactory struct {
	config           *config.Config
	nodesConfig      NodesSetupHandler
	shardCoordinator sharding.Coordinator
	keyGen           crypto.KeyGenerator
	privKey          crypto.PrivateKey
}

// NewCryptoComponentsFactory returns a new crypto components factory
func NewCryptoComponentsFactory(args CryptoComponentsFactoryArgs) (*cryptoComponentsFactory, error) {
	if args.Config == nil {
		return nil, ErrNilConfiguration
	}
	if args.NodesConfig == nil {
		return nil, ErrNilNodesConfig
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, ErrNilShardCoordinator
	}
	if check.IfNil(args.KeyGen) {
		return nil, ErrNilKeyGen
	}
	if check.IfNil(args.PrivKey) {
		return nil, ErrNilPrivateKey
	}

	return &cryptoComponentsFactory{
		config:           args.Config,
		nodesConfig:      args.NodesConfig,
		shardCoordinator: args.ShardCoordinator,
		keyGen:           args.KeyGen,
		privKey:          args.PrivKey,
	}, nil
}

// Create will create and return crypto components
func (ccf *cryptoComponentsFactory) Create() (*CryptoComponents, error) {
	initialPubKeys := ccf.nodesConfig.InitialNodesPubKeys()
	txSingleSigner := &singlesig.Ed25519Signer{}
	singleSigner, err := ccf.createSingleSigner()
	if err != nil {
		return nil, err
	}

	multisigHasher, err := ccf.getMultisigHasherFromConfig()
	if err != nil {
		return nil, err
	}

	currentShardNodesPubKeys, err := ccf.nodesConfig.InitialEligibleNodesPubKeysForShard(ccf.shardCoordinator.SelfId())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrMultiSigCreation, err.Error())
	}

	multiSigner, err := ccf.createMultiSigner(multisigHasher, currentShardNodesPubKeys)
	if err != nil {
		return nil, err
	}

	txSignKeyGen := signing.NewKeyGenerator(ed25519.NewEd25519())

	messageSignVerifier, err := systemVM.NewMessageSigVerifier(ccf.keyGen, singleSigner)
	if err != nil {
		return nil, err
	}

	return &CryptoComponents{
		TxSingleSigner:      txSingleSigner,
		SingleSigner:        singleSigner,
		MultiSigner:         multiSigner,
		BlockSignKeyGen:     ccf.keyGen,
		TxSignKeyGen:        txSignKeyGen,
		InitialPubKeys:      initialPubKeys,
		MessageSignVerifier: messageSignVerifier,
	}, nil
}

func (ccf *cryptoComponentsFactory) createSingleSigner() (crypto.SingleSigner, error) {
	switch ccf.config.Consensus.Type {
	case consensus.BlsConsensusType:
		return &mclsig.BlsSingleSigner{}, nil
	default:
		return nil, ErrMissingConsensusConfig
	}
}

func (ccf *cryptoComponentsFactory) getMultisigHasherFromConfig() (hashing.Hasher, error) {
	if ccf.config.Consensus.Type == consensus.BlsConsensusType && ccf.config.MultisigHasher.Type != "blake2b" {
		return nil, ErrMultiSigHasherMissmatch
	}

	switch ccf.config.MultisigHasher.Type {
	case "sha256":
		return sha256.Sha256{}, nil
	case "blake2b":
		if ccf.config.Consensus.Type == consensus.BlsConsensusType {
			return &blake2b.Blake2b{HashSize: multisig.BlsHashSize}, nil
		}
		return &blake2b.Blake2b{}, nil
	}

	return nil, ErrMissingMultiHasherConfig
}

func (ccf *cryptoComponentsFactory) createMultiSigner(
	hasher hashing.Hasher,
	pubKeys []string,
) (crypto.MultiSigner, error) {

	//TODO: the instantiation of BLS multi signer can be done with own public key instead of all public keys
	// e.g []string{ownPubKey}.
	// The parameter pubKeys for multi-signer is relevant when we want to create a multisig and in the multisig bitmap
	// we care about the order of the initial public keys that signed, but we never use the entire set of initial
	// public keys in their initial order.

	switch ccf.config.Consensus.Type {
	case consensus.BlsConsensusType:
		blsSigner := &mclmultisig.BlsMultiSigner{Hasher: hasher}
		return multisig.NewBLSMultisig(blsSigner, pubKeys, ccf.privKey, ccf.keyGen, uint16(0))
	default:
		return nil, ErrMissingConsensusConfig
	}
}
