package factory

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/crypto"
	"github.com/ElrondNetwork/elrond-go/crypto/signing"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/ed25519"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/ed25519/singlesig"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/mcl"
	mclmultisig "github.com/ElrondNetwork/elrond-go/crypto/signing/mcl/multisig"
	mclsig "github.com/ElrondNetwork/elrond-go/crypto/signing/mcl/singlesig"
	"github.com/ElrondNetwork/elrond-go/crypto/signing/multisig"
	"github.com/ElrondNetwork/elrond-go/data/state"
	stateFactory "github.com/ElrondNetwork/elrond-go/data/state/factory"
	"github.com/ElrondNetwork/elrond-go/epochStart/genesis"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/hashing/blake2b"
	"github.com/ElrondNetwork/elrond-go/hashing/sha256"
	"github.com/ElrondNetwork/elrond-go/vm"
	systemVM "github.com/ElrondNetwork/elrond-go/vm/process"
)

// CryptoComponentsFactoryArgs holds the arguments needed for creating crypto components
type CryptoComponentsFactoryArgs struct {
	ValidatorKeyPemFileName              string
	SkIndex                              int
	Config                               config.Config
	CoreComponentsHandler                CoreComponentsHandler
	ActivateBLSPubKeyMessageVerification bool
}

type cryptoComponentsFactory struct {
	pubKeyConverter                      state.PubkeyConverter
	suite                                crypto.Suite
	validatorKeyPemFileName              string
	skIndex                              int
	config                               config.Config
	coreComponentsHandler                CoreComponentsHandler
	activateBLSPubKeyMessageVerification bool
}

// CryptoParams holds the node public/private key data
type CryptoParams struct {
	PublicKey       crypto.PublicKey
	PrivateKey      crypto.PrivateKey
	PublicKeyString string
	PublicKeyBytes  []byte
	PrivateKeyKey   []byte
}

// CryptoComponents struct holds the crypto components
type CryptoComponents struct {
	TxSingleSigner      crypto.SingleSigner
	SingleSigner        crypto.SingleSigner
	MultiSigner         crypto.MultiSigner
	BlockSignKeyGen     crypto.KeyGenerator
	TxSignKeyGen        crypto.KeyGenerator
	MessageSignVerifier vm.MessageSignVerifier
	CryptoParams
}

// NewCryptoComponentsFactory returns a new crypto components factory
func NewCryptoComponentsFactory(args CryptoComponentsFactoryArgs) (*cryptoComponentsFactory, error) {
	if check.IfNil(args.CoreComponentsHandler) {
		return nil, ErrNilCoreComponents
	}

	pubKeyConverter, err := stateFactory.NewPubkeyConverter(args.Config.ValidatorPubkeyConverter)
	if err != nil {
		return nil, err
	}

	suite, err := getSuite(&args.Config)
	if err != nil {
		return nil, err
	}

	return &cryptoComponentsFactory{
		pubKeyConverter:                      pubKeyConverter,
		suite:                                suite,
		validatorKeyPemFileName:              args.ValidatorKeyPemFileName,
		skIndex:                              args.SkIndex,
		config:                               args.Config,
		coreComponentsHandler:                args.CoreComponentsHandler,
		activateBLSPubKeyMessageVerification: args.ActivateBLSPubKeyMessageVerification,
	}, nil
}

// Create will create and return crypto components
func (ccf *cryptoComponentsFactory) Create() (*CryptoComponents, error) {
	blockSignKeyGen := signing.NewKeyGenerator(ccf.suite)
	cp, err := ccf.createCryptoParams(blockSignKeyGen)
	if err != nil {
		return nil, err
	}

	txSignKeyGen := signing.NewKeyGenerator(ed25519.NewEd25519())
	txSingleSigner := &singlesig.Ed25519Signer{}
	singleSigner, err := ccf.createSingleSigner()
	if err != nil {
		return nil, err
	}

	multisigHasher, err := ccf.getMultisigHasherFromConfig()
	if err != nil {
		return nil, err
	}

	multiSigner, err := ccf.createMultiSigner(multisigHasher, cp, blockSignKeyGen)
	if err != nil {
		return nil, err
	}

	var messageSignVerifier vm.MessageSignVerifier
	if ccf.activateBLSPubKeyMessageVerification {
		messageSignVerifier, err = systemVM.NewMessageSigVerifier(blockSignKeyGen, singleSigner)
		if err != nil {
			return nil, err
		}
	} else {
		messageSignVerifier = &genesis.NilMessageSignVerifier{}
	}

	return &CryptoComponents{
		TxSingleSigner:      txSingleSigner,
		SingleSigner:        singleSigner,
		MultiSigner:         multiSigner,
		BlockSignKeyGen:     blockSignKeyGen,
		TxSignKeyGen:        txSignKeyGen,
		MessageSignVerifier: messageSignVerifier,
		CryptoParams:        *cp,
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
	cp *CryptoParams,
	blSignKeyGen crypto.KeyGenerator,
) (crypto.MultiSigner, error) {
	switch ccf.config.Consensus.Type {
	case consensus.BlsConsensusType:
		blsSigner := &mclmultisig.BlsMultiSigner{Hasher: hasher}
		return multisig.NewBLSMultisig(blsSigner, []string{cp.PublicKeyString}, cp.PrivateKey, blSignKeyGen, uint16(0))
	default:
		return nil, ErrMissingConsensusConfig
	}
}

func getSuite(config *config.Config) (crypto.Suite, error) {
	switch config.Consensus.Type {
	case consensus.BlsConsensusType:
		return mcl.NewSuiteBLS12(), nil
	default:
		return nil, errors.New("no consensus provided in config file")
	}
}

func (ccf *cryptoComponentsFactory) createCryptoParams(
	keygen crypto.KeyGenerator,
) (*CryptoParams, error) {

	cryptoParams := &CryptoParams{}
	sk, readPk, err := ccf.getSkPk()
	if err != nil {
		return nil, err
	}

	cryptoParams.PrivateKey, err = keygen.PrivateKeyFromByteArray(sk)
	if err != nil {
		return nil, err
	}

	cryptoParams.PublicKey = cryptoParams.PrivateKey.GeneratePublic()
	if len(readPk) > 0 {

		cryptoParams.PublicKeyBytes, err = cryptoParams.PublicKey.ToByteArray()
		if err != nil {
			return nil, err
		}

		if !bytes.Equal(cryptoParams.PublicKeyBytes, readPk) {
			return nil, ErrPublicKeyMismatch
		}
	}

	cryptoParams.PublicKeyString = ccf.pubKeyConverter.Encode(cryptoParams.PublicKeyBytes)

	return cryptoParams, nil
}

func (ccf *cryptoComponentsFactory) getSkPk() ([]byte, []byte, error) {
	encodedSk, pkString, err := core.LoadSkPkFromPemFile(ccf.validatorKeyPemFileName, ccf.skIndex)
	if err != nil {
		return nil, nil, err
	}

	skBytes, err := hex.DecodeString(string(encodedSk))
	if err != nil {
		return nil, nil, fmt.Errorf("%w for encoded secret key", err)
	}

	pkBytes, err := ccf.pubKeyConverter.Decode(pkString)
	if err != nil {
		return nil, nil, fmt.Errorf("%w for encoded public key %s", err, pkString)
	}

	return skBytes, pkBytes, nil
}
