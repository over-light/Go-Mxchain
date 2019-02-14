package multisig

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go-sandbox/crypto"
	"github.com/ElrondNetwork/elrond-go-sandbox/hashing"
	"bytes"
)

/*
belnev.go implements the multi-signature algorithm (BN-musig) presented in paper
"Multi-Signatures in the Plain Public-Key Model and a General Forking Lemma"
by Mihir Bellare and Gregory Neven. See https://cseweb.ucsd.edu/~mihir/papers/multisignatures-ccs.pdf.
This package provides the functionality for the cryptographic operations.
The message transfer functionality required for the algorithm are assumed to be
handled elsewhere. An overview of the protocol will be provided below.

The BN-musig protocol has 4 phases executed between a list of participants (public keys) L,
having a protocol leader (index = 0) and validators (index > 0). Each participant has it's
own private/public key pair (x_i, X_i), where x_i is the private key of participant
i and X_i is it's associated public key X_i = x_i * G. G is the base point on the used
curve, and * is the scalar multiplication. The protocol assumes that each participant has
the same view on the ordering of participants in the list L, so when communicating it is
not needed to pass on the list as well, but only a bitmap for the participating validators.

The phases of the protocol are as follows:

1. All participants/signers in L (including the leader) choose a random scalar 1 < r_i < n-1,
called a commitment secret, calculates the commitment R_i = r_i *G, and the commitment hash
t_i = H0(R_i), where H0 is a hashing function, different than H1 which will be used in next
rounds. Each of the members then broadcast these commitment hashes to all participants
including leader. (In a protocol where the communication is done only through the leader it needs
to be taken into account that the leader might not be honest)

2. When each signer i receives the commitment hash together with the public key of the sender
(t_j, X_j) it will send back the full R_i along with its public key, (R_i, X_i)

3. When signer i receives the full commitment from a signer j, it computes t_j = H0(R_j)
and verifies it with the previously received t_j. Locally each participant keeps track of
the sender, using a bitmap initially set to 0, by setting the corresponding bit to 1 and
storing the received commitment. If the commitment fails to validate the hash the protocol
is aborted. If commitment is not received in a bounded time delta and less than 2/3 of signers
have provided the commitment then protocol is aborted. If there are enough commitments >2/3
the leader broadcasts the bitmap and calculates the aggregated commitment R = Sum(R_i * B[i])

4. When signer i receives the bitmap, it calculates the signature share and broadcasts it to all
participating signers. R = Sum(R_i) is the aggregated commitment each signer needs to calculate
before signing. s_i = r_i + H1(<L'>||X_i||R||m) * x_i is the signature share for signer i
When signer i receives all signature shares, it can calculate the aggregated signature
s=Sum(s_i) for all s_i of the participants.

5. Verification is done by checking the equality:
	s * G = R + Sum(H1(<L'>||X_i||R||m) * X_i * B[i])
*/

type belNevSigner struct {
	message        []byte
	pubKeys        []crypto.PublicKey
	privKey        crypto.PrivateKey
	mutCommHashes  *sync.RWMutex
	commHashes     [][]byte
	commSecret     crypto.Scalar
	mutCommitments *sync.RWMutex
	commitments    []crypto.Point
	aggCommitment  crypto.Point
	mutSigShares   *sync.RWMutex
	sigShares      []crypto.Scalar
	aggSig         crypto.Scalar
	ownIndex       uint16
	suite          crypto.Suite
	hasher         hashing.Hasher
	keyGen         crypto.KeyGenerator
}

// NewBelNevMultisig creates a new Bellare Neven multi-signer
func NewBelNevMultisig(
	hasher hashing.Hasher,
	pubKeys []string,
	privKey crypto.PrivateKey,
	keyGen crypto.KeyGenerator,
	ownIndex uint16) (*belNevSigner, error) {

	if hasher == nil {
		return nil, crypto.ErrNilHasher
	}

	if privKey == nil {
		return nil, crypto.ErrNilPrivateKey
	}

	if pubKeys == nil {
		return nil, crypto.ErrNilPublicKeys
	}

	if keyGen == nil {
		return nil, crypto.ErrNilKeyGenerator
	}

	if ownIndex >= uint16(len(pubKeys)) {
		return nil, crypto.ErrIndexOutOfBounds
	}

	sizeConsensus := len(pubKeys)
	commHashes := make([][]byte, sizeConsensus)
	commitments := make([]crypto.Point, sizeConsensus)
	sigShares := make([]crypto.Scalar, sizeConsensus)
	pk, err := convertStringsToPubKeys(pubKeys, keyGen)

	if err != nil {
		return nil, err
	}

	// own index is used only for signing
	return &belNevSigner{
		pubKeys:        pk,
		privKey:        privKey,
		ownIndex:       ownIndex,
		hasher:         hasher,
		keyGen:         keyGen,
		mutCommHashes:  &sync.RWMutex{},
		commHashes:     commHashes,
		mutCommitments: &sync.RWMutex{},
		commitments:    commitments,
		mutSigShares:   &sync.RWMutex{},
		sigShares:      sigShares,
		suite:          keyGen.Suite(),
	}, nil
}

func convertStringsToPubKeys(pubKeys []string, kg crypto.KeyGenerator) ([]crypto.PublicKey, error) {
	var pk []crypto.PublicKey

	//convert pubKeys
	for _, pubKeyStr := range pubKeys {
		if pubKeyStr == "" {
			return nil, crypto.ErrEmptyPubKeyString
		}

		pubKey, err := kg.PublicKeyFromByteArray([]byte(pubKeyStr))
		if err != nil {
			return nil, crypto.ErrInvalidPublicKeyString
		}

		pk = append(pk, pubKey)
	}
	return pk, nil
}

// Reset resets the multiSigner and initializes corresponding fields with the given params
func (bn *belNevSigner) Reset(pubKeys []string, index uint16) error {
	if pubKeys == nil {
		return crypto.ErrNilPublicKeys
	}

	pk, err := convertStringsToPubKeys(pubKeys, bn.keyGen)

	if err != nil {
		return err
	}

	sizeConsensus := len(pubKeys)
	bn.message = nil
	bn.ownIndex = index
	bn.pubKeys = pk
	bn.commSecret = nil
	bn.aggCommitment = nil
	bn.aggSig = nil

	bn.mutCommHashes.Lock()
	bn.commHashes = make([][]byte, sizeConsensus)
	bn.mutCommHashes.Unlock()

	bn.mutCommitments.Lock()
	bn.commitments = make([]crypto.Point, sizeConsensus)
	bn.mutCommitments.Unlock()

	bn.mutSigShares.Lock()
	bn.sigShares = make([]crypto.Scalar, sizeConsensus)
	bn.mutSigShares.Unlock()

	return nil
}

// SetMessage sets the message to be multi-signed upon
func (bn *belNevSigner) SetMessage(msg []byte) error {
	if msg == nil {
		return crypto.ErrNilMessage
	}

	if len(msg) == 0 {
		return crypto.ErrInvalidParam
	}

	bn.message = msg

	return nil
}

// AddCommitmentHash sets a commitment Hash
func (bn *belNevSigner) AddCommitmentHash(index uint16, commHash []byte) error {
	if commHash == nil {
		return crypto.ErrNilCommitmentHash
	}

	bn.mutCommHashes.Lock()
	if int(index) >= len(bn.commHashes) {
		bn.mutCommHashes.Unlock()
		return crypto.ErrIndexOutOfBounds
	}

	bn.commHashes[index] = commHash
	bn.mutCommHashes.Unlock()
	return nil
}

// CommitmentHash returns the commitment hash from the list on the specified position
func (bn *belNevSigner) CommitmentHash(index uint16) ([]byte, error) {
	bn.mutCommHashes.RLock()
	defer bn.mutCommHashes.RUnlock()

	if int(index) >= len(bn.commHashes) {
		return nil, crypto.ErrIndexOutOfBounds
	}

	if bn.commHashes[index] == nil {
		return nil, crypto.ErrNilElement
	}

	return bn.commHashes[index], nil
}

// CreateCommitment creates a secret commitment and the corresponding public commitment point
func (bn *belNevSigner) CreateCommitment() (commSecret []byte, commitment []byte) {
	rand := bn.suite.RandomStream()
	sk, _ := bn.suite.CreateScalar().Pick(rand)
	pk := bn.suite.CreatePoint().Base()
	pk, _ = pk.Mul(sk)

	commSecret, _ = sk.MarshalBinary()
	commitment, _ = pk.MarshalBinary()

	return commSecret, commitment
}

// SetCommitmentSecret sets the committment secret
func (bn *belNevSigner) SetCommitmentSecret(commSecret []byte) error {
	if commSecret == nil {
		return crypto.ErrNilCommitmentSecret
	}

	commSecretScalar := bn.suite.CreateScalar()
	err := commSecretScalar.UnmarshalBinary(commSecret)
	if err != nil {
		return err
	}

	secret, err := commSecretScalar.MarshalBinary()
	if err != nil {
		return err
	}

	if !bytes.Equal(secret, commSecret) {
		return crypto.ErrInvalidParam
	}

	bn.commSecret = commSecretScalar

	return nil
}

// CommitmentSecret returns the set commitment secret
func (bn *belNevSigner) CommitmentSecret() ([]byte, error) {
	if bn.commSecret == nil {
		return nil, crypto.ErrNilCommitmentSecret
	}

	commSecret, err := bn.commSecret.MarshalBinary()

	return commSecret, err
}

// AddCommitment adds a commitment to the list on the specified position
func (bn *belNevSigner) AddCommitment(index uint16, commitment []byte) error {
	if commitment == nil {
		return crypto.ErrNilCommitment
	}

	commPoint := bn.suite.CreatePoint()
	err := commPoint.UnmarshalBinary(commitment)

	if err != nil {
		return err
	}

	bn.mutCommitments.Lock()
	if int(index) >= len(bn.commitments) {
		bn.mutCommitments.Unlock()
		return crypto.ErrIndexOutOfBounds
	}

	bn.commitments[index] = commPoint
	bn.mutCommitments.Unlock()
	return nil
}

// Commitment returns the commitment from the list with the specified position
func (bn *belNevSigner) Commitment(index uint16) ([]byte, error) {
	bn.mutCommitments.RLock()
	defer bn.mutCommitments.RUnlock()

	if int(index) >= len(bn.commitments) {
		return nil, crypto.ErrIndexOutOfBounds
	}

	if bn.commitments[index] == nil {
		return nil, crypto.ErrNilElement
	}

	commArray, err := bn.commitments[index].MarshalBinary()

	if err != nil {
		return nil, err
	}

	return commArray, nil
}

// AggregateCommitments aggregates the list of commitments
func (bn *belNevSigner) AggregateCommitments(bitmap []byte) ([]byte, error) {
	var err error

	if bitmap == nil {
		return nil, crypto.ErrNilBitmap
	}

	maxFlags := len(bitmap) * 8
	flagsMismatch := maxFlags < len(bn.pubKeys)
	if flagsMismatch {
		return nil, crypto.ErrBitmapMismatch
	}

	aggComm := bn.suite.CreatePoint().Null()
	bn.mutCommitments.RLock()
	defer bn.mutCommitments.RUnlock()

	for i := range bn.commitments {
		err := bn.isValidIndex(uint16(i), bitmap)
		if err != nil {
			continue
		}

		aggComm, err = aggComm.Add(bn.commitments[i])
		if err != nil {
			return nil, err
		}
	}

	aggCommBytes, err := aggComm.MarshalBinary()
	if err != nil {
		return nil, err
	}

	bn.aggCommitment = aggComm

	return aggCommBytes, nil
}

// SetAggCommitment sets the aggregated commitment for the marked signers in bitmap
func (bn *belNevSigner) SetAggCommitment(aggCommitment []byte) error {
	if aggCommitment == nil {
		return crypto.ErrNilAggregatedCommitment
	}

	aggCommPoint := bn.suite.CreatePoint()
	err := aggCommPoint.UnmarshalBinary(aggCommitment)

	if err != nil {
		return err
	}

	bn.aggCommitment = aggCommPoint

	return nil
}

// AggCommitment returns the set/computed aggregated commitment or error if not set
func (bn *belNevSigner) AggCommitment() ([]byte, error) {
	if bn.aggCommitment == nil {
		return nil, crypto.ErrNilAggregatedCommitment
	}

	return bn.aggCommitment.MarshalBinary()
}

// Creates the challenge for the specific index H1(<L'>||X_i||R||m)
func (bn *belNevSigner) computeChallenge(index uint16, bitmap []byte) (crypto.Scalar, error) {
	sizeConsensus := uint16(len(bn.commitments))

	if index >= sizeConsensus {
		return nil, crypto.ErrIndexOutOfBounds
	}

	if bn.commitments[index] == nil {
		return nil, crypto.ErrNilCommitment
	}

	if bn.message == nil {
		return nil, crypto.ErrNilMessage
	}

	if bitmap == nil {
		return nil, crypto.ErrNilBitmap
	}

	concatenated := make([]byte, 0)

	for i := range bn.pubKeys {
		err := bn.isValidIndex(uint16(i), bitmap)

		if err != nil {
			continue
		}

		pubKey, _ := bn.pubKeys[i].Point().MarshalBinary()

		concatenated = append(concatenated[:], pubKey[:]...)
	}

	// Concatenate pubKeys to form <L'>
	pubKey, err := bn.pubKeys[index].Point().MarshalBinary()
	if err != nil {
		return nil, err
	}

	if bn.aggCommitment == nil {
		return nil, crypto.ErrNilAggregatedCommitment
	}

	aggCommBytes, _ := bn.aggCommitment.MarshalBinary()

	// <L'> || X_i
	concatenated = append(concatenated[:], pubKey[:]...)
	// <L'> || X_i || R
	concatenated = append(concatenated[:], aggCommBytes[:]...)
	// <L'> || X_i || R || m
	concatenated = append(concatenated[:], bn.message[:]...)
	// H(<L'> || X_i || R || m)
	challenge := bn.hasher.Compute(string(concatenated))

	challengeScalar := bn.suite.CreateScalar()
	challengeScalar, err = challengeScalar.SetBytes(challenge)

	if err != nil {
		return nil, err
	}

	return challengeScalar, nil
}

// CreateSignatureShare creates a partial signature s_i = r_i + H(<L'> || X_i || R || m)*x_i
func (bn *belNevSigner) CreateSignatureShare(bitmap []byte) ([]byte, error) {
	if bitmap == nil {
		return nil, crypto.ErrNilBitmap
	}

	maxFlags := len(bitmap) * 8
	flagsMismatch := maxFlags < len(bn.pubKeys)
	if flagsMismatch {
		return nil, crypto.ErrBitmapMismatch
	}

	challengeScalar, err := bn.computeChallenge(bn.ownIndex, bitmap)
	if err != nil {
		return nil, err
	}

	privKeyScalar := bn.privKey.Scalar()
	// H(<L'> || X_i || R || m)*x_i
	sigShareScalar, err := challengeScalar.Mul(privKeyScalar)

	if err != nil {
		return nil, err
	}

	if bn.commSecret == nil {
		return nil, crypto.ErrNilCommitmentSecret
	}

	// s_i = r_i + H(<L'> || X_i || R || m)*x_i
	sigShareScalar, err = sigShareScalar.Add(bn.commSecret)
	if err != nil {
		return nil, err
	}

	sigShare, err := sigShareScalar.MarshalBinary()
	if err != nil {
		return nil, err
	}

	bn.mutSigShares.Lock()
	bn.sigShares[bn.ownIndex] = sigShareScalar
	bn.mutSigShares.Unlock()

	return sigShare, nil
}

func (bn *belNevSigner) isValidIndex(index uint16, bitmap []byte) error {
	indexOutOfBounds := index >= uint16(len(bn.pubKeys))
	if indexOutOfBounds {
		return crypto.ErrIndexOutOfBounds
	}

	indexNotInBitmap := bitmap[index/8]&(1<<uint8(index%8)) == 0
	if indexNotInBitmap {
		return crypto.ErrIndexNotSelected
	}

	return nil
}

// VerifySignatureShare verifies the partial signature of the signer with specified position
// s_i * G = R_i + H1(<L'>||X_i||R||m)*X_i
func (bn *belNevSigner) VerifySignatureShare(index uint16, sig []byte, bitmap []byte) error {
	if sig == nil {
		return crypto.ErrNilSignature
	}

	if bitmap == nil {
		return crypto.ErrNilBitmap
	}

	err := bn.isValidIndex(index, bitmap)
	if err != nil {
		return err
	}

	sigScalar := bn.suite.CreateScalar()
	_ = sigScalar.UnmarshalBinary(sig)

	// s_i * G
	basePoint := bn.suite.CreatePoint().Base()
	left, _ := basePoint.Mul(sigScalar)

	challengeScalar, err := bn.computeChallenge(index, bitmap)
	if err != nil {
		return err
	}

	pubKey := bn.pubKeys[index].Point()
	// H1(<L'>||X_i||R||m)*X_i
	right, _ := pubKey.Mul(challengeScalar)

	// R_i + H1(<L'>||X_i||R||m)*X_i
	bn.mutCommitments.RLock()
	right, err = right.Add(bn.commitments[index])
	bn.mutCommitments.RUnlock()
	if err != nil {
		return err
	}

	eq, err := right.Equal(left)
	if err != nil {
		return err
	}

	if !eq {
		return crypto.ErrSigNotValid
	}

	return nil
}

// AddSignatureShare adds the partial signature of the signer with specified position
func (bn *belNevSigner) AddSignatureShare(index uint16, sig []byte) error {
	if sig == nil {
		return crypto.ErrNilSignature
	}

	sigScalar := bn.suite.CreateScalar()
	err := sigScalar.UnmarshalBinary(sig)

	if err != nil {
		return err
	}

	bn.mutSigShares.Lock()
	if int(index) >= len(bn.sigShares) {
		bn.mutSigShares.Unlock()
		return crypto.ErrIndexOutOfBounds
	}

	bn.sigShares[index] = sigScalar
	bn.mutSigShares.Unlock()

	return nil
}

// SignatureShare returns the partial signature set for given index
func (bn *belNevSigner) SignatureShare(index uint16) ([]byte, error) {
	bn.mutSigShares.RLock()
	defer bn.mutSigShares.RUnlock()

	if int(index) >= len(bn.sigShares) {
		return nil, crypto.ErrIndexOutOfBounds
	}

	if bn.sigShares[index] == nil {
		return nil, crypto.ErrNilElement
	}

	sigShareBytes, err := bn.sigShares[index].MarshalBinary()

	if err != nil {
		return nil, err
	}

	return sigShareBytes, nil
}

// AggregateSigs aggregates all collected partial signatures
func (bn *belNevSigner) AggregateSigs(bitmap []byte) ([]byte, error) {
	var err error

	if bitmap == nil {
		return nil, crypto.ErrNilBitmap
	}

	maxFlags := len(bitmap) * 8
	flagsMismatch := maxFlags < len(bn.pubKeys)
	if flagsMismatch {
		return nil, crypto.ErrBitmapMismatch
	}

	aggSig := bn.suite.CreateScalar().Zero()
	bn.mutSigShares.RLock()
	defer bn.mutSigShares.RUnlock()

	for i := range bn.sigShares {
		err := bn.isValidIndex(uint16(i), bitmap)
		if err != nil {
			continue
		}

		aggSig, err = aggSig.Add(bn.sigShares[i])
		if err != nil {
			return nil, err
		}
	}

	aggSigBytes, err := aggSig.MarshalBinary()
	if err != nil {
		return nil, err
	}

	bn.aggSig = aggSig

	return aggSigBytes, nil
}

// SetAggregatedSig sets the aggregated signature
func (bn *belNevSigner) SetAggregatedSig(aggSig []byte) error {
	if aggSig == nil {
		return crypto.ErrNilSignature
	}

	aggSigPoint := bn.suite.CreateScalar()
	err := aggSigPoint.UnmarshalBinary(aggSig)

	if err != nil {
		return err
	}

	bn.aggSig = aggSigPoint

	return nil
}

// Verify verifies the aggregated signature by checking equality
// s * G = R + Sum(H1(<L'>||X_i||R||m)*X_i*B[i])
func (bn *belNevSigner) Verify(bitmap []byte) error {
	if bitmap == nil {
		return crypto.ErrNilBitmap
	}

	right := bn.suite.CreatePoint().Null()
	var err error

	for i := range bn.pubKeys {
		err := bn.isValidIndex(uint16(i), bitmap)
		if err != nil {
			return err
		}

		challengeScalar, err := bn.computeChallenge(uint16(i), bitmap)
		if err != nil {
			return err
		}

		pubKey := bn.pubKeys[i].Point()
		// H1(<L'>||X_i||R||m)*X_i
		part, err := pubKey.Mul(challengeScalar)
		if err != nil {
			return err
		}

		right, err = right.Add(part)
		if err != nil {
			return err
		}
	}

	// R + Sum(H1(<L'>||X_i||R||m)*X_i)
	right, err = right.Add(bn.aggCommitment)
	if err != nil {
		return err
	}

	// s * G
	left := bn.suite.CreatePoint().Base()
	left, err = left.Mul(bn.aggSig)
	if err != nil {
		return err
	}

	// s * G = R + Sum(H1(<L'>||X_i||R||m)*X_i)
	eq, err := right.Equal(left)
	if err != nil {
		return err
	}

	if !eq {
		return crypto.ErrSigNotValid
	}

	return nil
}
