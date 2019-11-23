package shardchain

import (
	"bytes"
	"sync"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
)

// ArgsShardEpochStartTrigger struct { defines the arguments needed for new start of epoch trigger
type ArgsShardEpochStartTrigger struct {
	Marshalizer marshal.Marshalizer
	Hasher      hashing.Hasher

	HeaderValidator epochStart.HeaderValidator
	Uint64Converter typeConverters.Uint64ByteSliceConverter

	DataPool       dataRetriever.PoolsHolder
	Storage        dataRetriever.StorageService
	RequestHandler epochStart.RequestHandler

	Epoch    uint32
	Validity uint64
	Finality uint64
}

type trigger struct {
	epoch              uint32
	currentRoundIndex  int64
	epochStartRound    uint64
	epochMetaBlockHash []byte
	isEpochStart       bool
	finality           uint64
	validity           uint64

	newEpochHdrReceived bool

	mutTrigger        sync.RWMutex
	mapHashHdr        map[string]*block.MetaBlock
	mapNonceHashes    map[uint64][]string
	mapEpochStartHdrs map[string]*block.MetaBlock

	metaHdrPool         storage.Cacher
	metaHdrNonces       dataRetriever.Uint64SyncMapCacher
	metaHdrStorage      storage.Storer
	metaNonceHdrStorage storage.Storer
	uint64Converter     typeConverters.Uint64ByteSliceConverter

	marshalizer     marshal.Marshalizer
	hasher          hashing.Hasher
	headerValidator epochStart.HeaderValidator

	requestHandler epochStart.RequestHandler
}

// NewEpochStartTrigger creates a trigger to signal start of epoch
func NewEpochStartTrigger(args *ArgsShardEpochStartTrigger) (*trigger, error) {
	if args == nil {
		return nil, epochStart.ErrNilArgsNewShardEpochStartTrigger
	}
	if check.IfNil(args.Hasher) {
		return nil, epochStart.ErrNilHasher
	}
	if check.IfNil(args.Marshalizer) {
		return nil, epochStart.ErrNilMarshalizer
	}
	if check.IfNil(args.HeaderValidator) {
		return nil, epochStart.ErrNilHeaderValidator
	}
	if check.IfNil(args.DataPool) {
		return nil, epochStart.ErrNilDataPoolsHolder
	}
	if check.IfNil(args.Storage) {
		return nil, epochStart.ErrNilStorageService
	}
	if check.IfNil(args.RequestHandler) {
		return nil, epochStart.ErrNilRequestHandler
	}
	if check.IfNil(args.DataPool.MetaBlocks()) {
		return nil, epochStart.ErrNilMetaBlocksPool
	}
	if check.IfNil(args.DataPool.HeadersNonces()) {
		return nil, epochStart.ErrNilHeaderNoncesPool
	}
	if check.IfNil(args.Uint64Converter) {
		return nil, epochStart.ErrNilUint64Converter
	}

	metaHdrStorage := args.Storage.GetStorer(dataRetriever.MetaBlockUnit)
	if check.IfNil(metaHdrStorage) {
		return nil, epochStart.ErrNilMetaHdrStorage
	}

	metaHdrNoncesStorage := args.Storage.GetStorer(dataRetriever.MetaHdrNonceHashDataUnit)
	if check.IfNil(metaHdrNoncesStorage) {
		return nil, epochStart.ErrNilMetaNonceHashStorage
	}

	newTrigger := &trigger{
		epoch:               args.Epoch,
		currentRoundIndex:   0,
		epochStartRound:     0,
		isEpochStart:        false,
		validity:            args.Validity,
		finality:            args.Finality,
		newEpochHdrReceived: false,
		mutTrigger:          sync.RWMutex{},
		mapHashHdr:          make(map[string]*block.MetaBlock),
		mapNonceHashes:      make(map[uint64][]string),
		mapEpochStartHdrs:   make(map[string]*block.MetaBlock),
		metaHdrPool:         args.DataPool.MetaBlocks(),
		metaHdrNonces:       args.DataPool.HeadersNonces(),
		metaHdrStorage:      metaHdrStorage,
		metaNonceHdrStorage: metaHdrNoncesStorage,
		uint64Converter:     args.Uint64Converter,
		marshalizer:         args.Marshalizer,
		hasher:              args.Hasher,
		headerValidator:     args.HeaderValidator,
		requestHandler:      args.RequestHandler,
		epochMetaBlockHash:  nil,
	}
	return newTrigger, nil
}

// IsEpochStart returns true if conditions are fulfilled for start of epoch
func (t *trigger) IsEpochStart() bool {
	t.mutTrigger.RLock()
	defer t.mutTrigger.RUnlock()

	return t.isEpochStart
}

// Epoch returns the current epoch number
func (t *trigger) Epoch() uint32 {
	t.mutTrigger.RLock()
	defer t.mutTrigger.RUnlock()

	return t.epoch
}

// EpochStartRound returns the start round of the current epoch
func (t *trigger) EpochStartRound() uint64 {
	t.mutTrigger.RLock()
	defer t.mutTrigger.RUnlock()

	return t.epochStartRound
}

// ForceEpochStart sets the conditions for start of epoch to true in case of edge cases
func (t *trigger) ForceEpochStart(round uint64) error {
	t.mutTrigger.Lock()
	defer t.mutTrigger.Unlock()

	return nil
}

// ReceivedHeader saves the header into pool to verify if end-of-epoch conditions are fulfilled
func (t *trigger) ReceivedHeader(header data.HeaderHandler) {
	t.mutTrigger.Lock()
	defer t.mutTrigger.Unlock()

	if t.isEpochStart == true {
		return
	}

	metaHdr, ok := header.(*block.MetaBlock)
	if !ok {
		return
	}

	if t.newEpochHdrReceived == false && !metaHdr.IsStartOfEpochBlock() {
		return
	}

	isMetaStartOfEpochForCurrentEpoch := metaHdr.Epoch == t.epoch && metaHdr.IsStartOfEpochBlock()
	if isMetaStartOfEpochForCurrentEpoch {
		return
	}

	hdrHash, err := core.CalculateHash(t.marshalizer, t.hasher, metaHdr)
	if err != nil {
		return
	}

	if _, ok := t.mapHashHdr[string(hdrHash)]; ok {
		return
	}
	if _, ok := t.mapEpochStartHdrs[string(hdrHash)]; ok {
		return
	}

	t.updateTriggerFromMeta(metaHdr, hdrHash)
}

// call only if mutex is locked before
func (t *trigger) updateTriggerFromMeta(metaHdr *block.MetaBlock, hdrHash []byte) {
	if metaHdr.IsStartOfEpochBlock() {
		t.newEpochHdrReceived = true
		t.mapEpochStartHdrs[string(hdrHash)] = metaHdr
	} else {
		t.mapHashHdr[string(hdrHash)] = metaHdr
		t.mapNonceHashes[metaHdr.Nonce] = append(t.mapNonceHashes[metaHdr.Nonce], string(hdrHash))
	}

	for hash, meta := range t.mapEpochStartHdrs {
		canActivateEpochStart := t.checkIfTriggerCanBeActivated(hash, meta)
		if canActivateEpochStart && t.epoch != meta.Epoch {
			t.epoch = meta.Epoch
			t.isEpochStart = true
			t.epochStartRound = meta.Round
			t.epochMetaBlockHash = []byte(hash)
			break
		}
	}
}

// call only if mutex is locked before
func (t *trigger) isMetaBlockValid(hash string, metaHdr *block.MetaBlock) bool {
	currHdr := metaHdr
	for i := metaHdr.Nonce - 1; i >= metaHdr.Nonce-t.validity; i-- {
		neededHdr, err := t.getHeaderWithNonceAndHash(i, currHdr.PrevHash)
		if err != nil {
			return false
		}

		err = t.headerValidator.IsHeaderConstructionValid(currHdr, neededHdr)
		if err != nil {
			return false
		}
	}

	return true
}

func (t *trigger) isMetaBlockFinal(hash string, metaHdr *block.MetaBlock) bool {
	nextBlocksVerified := uint64(0)
	currHdr := metaHdr
	for nonce := metaHdr.Nonce + 1; nonce <= metaHdr.Nonce+t.finality; nonce++ {
		currHash, err := core.CalculateHash(t.marshalizer, t.hasher, currHdr)
		if err != nil {
			continue
		}

		neededHdr, err := t.getHeaderWithNonceAndPrevHash(nonce, currHash)
		if err != nil {
			continue
		}

		err = t.headerValidator.IsHeaderConstructionValid(neededHdr, currHdr)
		if err != nil {
			continue
		}

		currHdr = neededHdr
		nextBlocksVerified += 1
	}

	if nextBlocksVerified < t.finality {
		for nonce := currHdr.Nonce + 1; nonce <= currHdr.Nonce+t.finality; nonce++ {
			go t.requestHandler.RequestHeaderByNonce(sharding.MetachainShardId, nonce)
		}
		return false
	}

	return true
}

// call only if mutex is locked before
func (t *trigger) checkIfTriggerCanBeActivated(hash string, metaHdr *block.MetaBlock) bool {
	isMetaHdrValid := t.isMetaBlockValid(hash, metaHdr)
	if !isMetaHdrValid {
		return isMetaHdrValid
	}

	isMetaHdrFinal := t.isMetaBlockFinal(hash, metaHdr)
	return isMetaHdrFinal
}

// call only if mutex is locked before
func (t *trigger) getHeaderWithNonceAndHashFromMaps(nonce uint64, neededHash []byte) *block.MetaBlock {
	metaHdrHashesWithNonce := t.mapNonceHashes[nonce]
	for _, hash := range metaHdrHashesWithNonce {
		if !bytes.Equal(neededHash, []byte(hash)) {
			continue
		}

		neededHdr := t.mapHashHdr[hash]
		if neededHdr != nil {
			return neededHdr
		}
	}

	return nil
}

// call only if mutex is locked before
func (t *trigger) getHeaderWithHashFromPool(neededHash []byte) *block.MetaBlock {
	peekedData, _ := t.metaHdrPool.Peek(neededHash)
	neededHdr, ok := peekedData.(*block.MetaBlock)
	if ok {
		t.mapHashHdr[string(neededHash)] = neededHdr
		t.mapNonceHashes[neededHdr.Nonce] = append(t.mapNonceHashes[neededHdr.Nonce], string(neededHash))
		return neededHdr
	}

	return nil
}

// call only if mutex is locked before
func (t *trigger) getHeaderWithHashFromStorage(neededHash []byte) *block.MetaBlock {
	storageData, err := t.metaHdrStorage.Get(neededHash)
	if err == nil {
		var neededHdr block.MetaBlock
		err = t.marshalizer.Unmarshal(&neededHdr, storageData)
		if err == nil {
			t.mapHashHdr[string(neededHash)] = &neededHdr
			t.mapNonceHashes[neededHdr.Nonce] = append(t.mapNonceHashes[neededHdr.Nonce], string(neededHash))
			return &neededHdr
		}
	}

	return nil
}

// call only if mutex is locked before
func (t *trigger) getHeaderWithNonceAndHash(nonce uint64, neededHash []byte) (*block.MetaBlock, error) {
	metaHdr := t.getHeaderWithNonceAndHashFromMaps(nonce, neededHash)
	if metaHdr != nil {
		return metaHdr, nil
	}

	metaHdr = t.getHeaderWithHashFromPool(neededHash)
	if metaHdr != nil {
		return metaHdr, nil
	}

	metaHdr = t.getHeaderWithHashFromStorage(neededHash)
	if metaHdr != nil {
		return metaHdr, nil
	}

	go t.requestHandler.RequestHeader(sharding.MetachainShardId, neededHash)

	return nil, epochStart.ErrMetaHdrNotFound
}

// call only if mutex is locked before
func (t *trigger) getHeaderWithNonceAndPrevHashFromMaps(nonce uint64, prevHash []byte) *block.MetaBlock {
	metaHdrHashesWithNonce := t.mapNonceHashes[nonce]
	for _, hash := range metaHdrHashesWithNonce {
		hdrWithNonce := t.mapHashHdr[hash]
		if hdrWithNonce != nil && bytes.Equal(hdrWithNonce.PrevHash, prevHash) {
			return hdrWithNonce
		}
	}
	return nil
}

// call only if mutex is locked before
func (t *trigger) getHeaderWithNonceAndPrevHashFromCache(nonce uint64, prevHash []byte) *block.MetaBlock {
	shIdMap, ok := t.metaHdrNonces.Get(nonce)
	if ok {
		hdrHash, ok := shIdMap.Load(sharding.MetachainShardId)
		if ok {
			dataHdr, _ := t.metaHdrPool.Peek(hdrHash)
			hdrWithNonce, ok := dataHdr.(*block.MetaBlock)
			if ok && bytes.Equal(hdrWithNonce.PrevHash, prevHash) {
				t.mapHashHdr[string(hdrHash)] = hdrWithNonce
				t.mapNonceHashes[hdrWithNonce.Nonce] = append(t.mapNonceHashes[hdrWithNonce.Nonce], string(hdrHash))
				return hdrWithNonce
			}
		}
	}
	return nil
}

// call only if mutex is locked before
func (t *trigger) getHeaderWithNonceAndPrevHash(nonce uint64, prevHash []byte) (*block.MetaBlock, error) {
	metaHdr := t.getHeaderWithNonceAndPrevHashFromMaps(nonce, prevHash)
	if metaHdr != nil {
		return metaHdr, nil
	}

	metaHdr = t.getHeaderWithNonceAndPrevHashFromCache(nonce, prevHash)
	if metaHdr != nil {
		return metaHdr, nil
	}

	nonceToByteSlice := t.uint64Converter.ToByteSlice(nonce)
	dataHdr, err := t.metaNonceHdrStorage.Get(nonceToByteSlice)
	if err != nil || len(dataHdr) == 0 {
		go t.requestHandler.RequestHeaderByNonce(sharding.MetachainShardId, nonce)
		return nil, err
	}

	var neededHash []byte
	err = t.marshalizer.Unmarshal(neededHash, dataHdr)
	if err != nil {
		return nil, err
	}

	return t.getHeaderWithNonceAndHash(nonce, neededHash)
}

// Update updates the end-of-epoch trigger
func (t *trigger) Update(round uint64) {
}

// Processed sets start of epoch to false and cleans underlying structure
func (t *trigger) Processed() {
	t.mutTrigger.Lock()
	defer t.mutTrigger.Unlock()

	t.isEpochStart = false
	t.newEpochHdrReceived = false

	t.mapHashHdr = make(map[string]*block.MetaBlock)
	t.mapNonceHashes = make(map[uint64][]string)
	t.mapEpochStartHdrs = make(map[string]*block.MetaBlock)
}

// Revert sets the start of epoch back to true
func (t *trigger) Revert() {
	t.mutTrigger.Lock()
	defer t.mutTrigger.Unlock()

	t.isEpochStart = true
	t.newEpochHdrReceived = true
}

// EpochStartMetaHdrHash returns the announcing meta header hash which created the new epoch
func (t *trigger) EpochStartMetaHdrHash() []byte {
	t.mutTrigger.RLock()
	defer t.mutTrigger.RUnlock()

	return t.epochMetaBlockHash
}

// IsInterfaceNil returns true if underlying object is nil
func (t *trigger) IsInterfaceNil() bool {
	return t == nil
}
