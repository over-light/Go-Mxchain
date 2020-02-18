package process

import (
	"encoding/hex"
	"math"
	"sort"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/logger"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

var log = logger.GetOrCreate("process")

// EmptyChannel empties the given channel
func EmptyChannel(ch chan bool) int {
	readsCnt := 0
	for {
		select {
		case <-ch:
			readsCnt++
		default:
			return readsCnt
		}
	}
}

// GetShardHeader gets the header, which is associated with the given hash, from pool or storage
func GetShardHeader(
	hash []byte,
	headersCacher dataRetriever.HeadersPool,
	marshalizer marshal.Marshalizer,
	storageService dataRetriever.StorageService,
) (*block.Header, error) {

	err := checkGetHeaderParamsForNil(headersCacher, marshalizer, storageService)
	if err != nil {
		return nil, err
	}

	hdr, err := GetShardHeaderFromPool(hash, headersCacher)
	if err != nil {
		hdr, err = GetShardHeaderFromStorage(hash, marshalizer, storageService)
		if err != nil {
			return nil, err
		}
	}

	return hdr, nil
}

// GetMetaHeader gets the header, which is associated with the given hash, from pool or storage
func GetMetaHeader(
	hash []byte,
	headersCacher dataRetriever.HeadersPool,
	marshalizer marshal.Marshalizer,
	storageService dataRetriever.StorageService,
) (*block.MetaBlock, error) {

	err := checkGetHeaderParamsForNil(headersCacher, marshalizer, storageService)
	if err != nil {
		return nil, err
	}

	hdr, err := GetMetaHeaderFromPool(hash, headersCacher)
	if err != nil {
		hdr, err = GetMetaHeaderFromStorage(hash, marshalizer, storageService)
		if err != nil {
			return nil, err
		}
	}

	return hdr, nil
}

// GetShardHeaderFromPool gets the header, which is associated with the given hash, from pool
func GetShardHeaderFromPool(
	hash []byte,
	headersCacher dataRetriever.HeadersPool,
) (*block.Header, error) {

	obj, err := getHeaderFromPool(hash, headersCacher)
	if err != nil {
		return nil, err
	}

	hdr, ok := obj.(*block.Header)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return hdr, nil
}

// GetMetaHeaderFromPool gets the header, which is associated with the given hash, from pool
func GetMetaHeaderFromPool(
	hash []byte,
	headersCacher dataRetriever.HeadersPool,
) (*block.MetaBlock, error) {

	obj, err := getHeaderFromPool(hash, headersCacher)
	if err != nil {
		return nil, err
	}

	hdr, ok := obj.(*block.MetaBlock)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return hdr, nil
}

// GetShardHeaderFromStorage gets the header, which is associated with the given hash, from storage
func GetShardHeaderFromStorage(
	hash []byte,
	marshalizer marshal.Marshalizer,
	storageService dataRetriever.StorageService,
) (*block.Header, error) {

	buffHdr, err := GetMarshalizedHeaderFromStorage(dataRetriever.BlockHeaderUnit, hash, marshalizer, storageService)
	if err != nil {
		return nil, err
	}

	hdr := &block.Header{}
	err = marshalizer.Unmarshal(hdr, buffHdr)
	if err != nil {
		return nil, ErrUnmarshalWithoutSuccess
	}

	return hdr, nil
}

// GetMetaHeaderFromStorage gets the header, which is associated with the given hash, from storage
func GetMetaHeaderFromStorage(
	hash []byte,
	marshalizer marshal.Marshalizer,
	storageService dataRetriever.StorageService,
) (*block.MetaBlock, error) {

	buffHdr, err := GetMarshalizedHeaderFromStorage(dataRetriever.MetaBlockUnit, hash, marshalizer, storageService)
	if err != nil {
		return nil, err
	}

	hdr := &block.MetaBlock{}
	err = marshalizer.Unmarshal(hdr, buffHdr)
	if err != nil {
		return nil, ErrUnmarshalWithoutSuccess
	}

	return hdr, nil
}

// GetMarshalizedHeaderFromStorage gets the marshalized header, which is associated with the given hash, from storage
func GetMarshalizedHeaderFromStorage(
	blockUnit dataRetriever.UnitType,
	hash []byte,
	marshalizer marshal.Marshalizer,
	storageService dataRetriever.StorageService,
) ([]byte, error) {

	if marshalizer == nil || marshalizer.IsInterfaceNil() {
		return nil, ErrNilMarshalizer
	}
	if storageService == nil || storageService.IsInterfaceNil() {
		return nil, ErrNilStorage
	}

	hdrStore := storageService.GetStorer(blockUnit)
	if hdrStore == nil || hdrStore.IsInterfaceNil() {
		return nil, ErrNilHeadersStorage
	}

	buffHdr, err := hdrStore.Get(hash)
	if err != nil {
		return nil, ErrMissingHeader
	}

	return buffHdr, nil
}

// GetShardHeaderWithNonce method returns a shard block header with a given nonce and shardId
func GetShardHeaderWithNonce(
	nonce uint64,
	shardId uint32,
	headersCacher dataRetriever.HeadersPool,
	marshalizer marshal.Marshalizer,
	storageService dataRetriever.StorageService,
	uint64Converter typeConverters.Uint64ByteSliceConverter,
) (*block.Header, []byte, error) {

	err := checkGetHeaderWithNonceParamsForNil(headersCacher, marshalizer, storageService, uint64Converter)
	if err != nil {
		return nil, nil, err
	}

	hdr, hash, err := GetShardHeaderFromPoolWithNonce(nonce, shardId, headersCacher)
	if err != nil {
		hdr, hash, err = GetShardHeaderFromStorageWithNonce(nonce, shardId, storageService, uint64Converter, marshalizer)
		if err != nil {
			return nil, nil, err
		}
	}

	return hdr, hash, nil
}

// GetMetaHeaderWithNonce method returns a meta block header with a given nonce
func GetMetaHeaderWithNonce(
	nonce uint64,
	headersCacher dataRetriever.HeadersPool,
	marshalizer marshal.Marshalizer,
	storageService dataRetriever.StorageService,
	uint64Converter typeConverters.Uint64ByteSliceConverter,
) (*block.MetaBlock, []byte, error) {

	err := checkGetHeaderWithNonceParamsForNil(headersCacher, marshalizer, storageService, uint64Converter)
	if err != nil {
		return nil, nil, err
	}

	hdr, hash, err := GetMetaHeaderFromPoolWithNonce(nonce, headersCacher)
	if err != nil {
		hdr, hash, err = GetMetaHeaderFromStorageWithNonce(nonce, storageService, uint64Converter, marshalizer)
		if err != nil {
			return nil, nil, err
		}
	}

	return hdr, hash, nil
}

// GetShardHeaderFromPoolWithNonce method returns a shard block header from pool with a given nonce and shardId
func GetShardHeaderFromPoolWithNonce(
	nonce uint64,
	shardId uint32,
	headersCacher dataRetriever.HeadersPool,
) (*block.Header, []byte, error) {

	obj, hash, err := getHeaderFromPoolWithNonce(nonce, shardId, headersCacher)
	if err != nil {
		return nil, nil, err
	}

	hdr, ok := obj.(*block.Header)
	if !ok {
		return nil, nil, ErrWrongTypeAssertion
	}

	return hdr, hash, nil
}

// GetMetaHeaderFromPoolWithNonce method returns a meta block header from pool with a given nonce
func GetMetaHeaderFromPoolWithNonce(
	nonce uint64,
	headersCacher dataRetriever.HeadersPool,
) (*block.MetaBlock, []byte, error) {

	obj, hash, err := getHeaderFromPoolWithNonce(nonce, sharding.MetachainShardId, headersCacher)
	if err != nil {
		return nil, nil, err
	}

	hdr, ok := obj.(*block.MetaBlock)
	if !ok {
		return nil, nil, ErrWrongTypeAssertion
	}

	return hdr, hash, nil
}

// GetHeaderFromStorageWithNonce method returns a block header from storage with a given nonce and shardId
func GetHeaderFromStorageWithNonce(
	nonce uint64,
	shardId uint32,
	storageService dataRetriever.StorageService,
	uint64Converter typeConverters.Uint64ByteSliceConverter,
	marshalizer marshal.Marshalizer,
) (data.HeaderHandler, []byte, error) {

	if shardId == sharding.MetachainShardId {
		return GetMetaHeaderFromStorageWithNonce(nonce, storageService, uint64Converter, marshalizer)
	}
	return GetShardHeaderFromStorageWithNonce(nonce, shardId, storageService, uint64Converter, marshalizer)
}

// GetShardHeaderFromStorageWithNonce method returns a shard block header from storage with a given nonce and shardId
func GetShardHeaderFromStorageWithNonce(
	nonce uint64,
	shardId uint32,
	storageService dataRetriever.StorageService,
	uint64Converter typeConverters.Uint64ByteSliceConverter,
	marshalizer marshal.Marshalizer,
) (*block.Header, []byte, error) {

	hash, err := getHeaderHashFromStorageWithNonce(
		nonce,
		storageService,
		uint64Converter,
		marshalizer,
		dataRetriever.ShardHdrNonceHashDataUnit+dataRetriever.UnitType(shardId))
	if err != nil {
		return nil, nil, err
	}

	hdr, err := GetShardHeaderFromStorage(hash, marshalizer, storageService)
	if err != nil {
		return nil, nil, err
	}

	return hdr, hash, nil
}

// GetMetaHeaderFromStorageWithNonce method returns a meta block header from storage with a given nonce
func GetMetaHeaderFromStorageWithNonce(
	nonce uint64,
	storageService dataRetriever.StorageService,
	uint64Converter typeConverters.Uint64ByteSliceConverter,
	marshalizer marshal.Marshalizer,
) (*block.MetaBlock, []byte, error) {

	hash, err := getHeaderHashFromStorageWithNonce(
		nonce,
		storageService,
		uint64Converter,
		marshalizer,
		dataRetriever.MetaHdrNonceHashDataUnit)
	if err != nil {
		return nil, nil, err
	}

	hdr, err := GetMetaHeaderFromStorage(hash, marshalizer, storageService)
	if err != nil {
		return nil, nil, err
	}

	return hdr, hash, nil
}

// GetTransactionHandler gets the transaction with a given sender/receiver shardId and txHash
func GetTransactionHandler(
	senderShardID uint32,
	destShardID uint32,
	txHash []byte,
	shardedDataCacherNotifier dataRetriever.ShardedDataCacherNotifier,
	storageService dataRetriever.StorageService,
	marshalizer marshal.Marshalizer,
	searchFirst bool,
) (data.TransactionHandler, error) {

	err := checkGetTransactionParamsForNil(shardedDataCacherNotifier, storageService, marshalizer)
	if err != nil {
		return nil, err
	}

	tx, err := GetTransactionHandlerFromPool(senderShardID, destShardID, txHash, shardedDataCacherNotifier, searchFirst)
	if err != nil {
		tx, err = GetTransactionHandlerFromStorage(txHash, storageService, marshalizer)
		if err != nil {
			return nil, err
		}
	}

	return tx, nil
}

// GetTransactionHandlerFromPool gets the transaction from pool with a given sender/receiver shardId and txHash
func GetTransactionHandlerFromPool(
	senderShardID uint32,
	destShardID uint32,
	txHash []byte,
	shardedDataCacherNotifier dataRetriever.ShardedDataCacherNotifier,
	searchFirst bool,
) (data.TransactionHandler, error) {

	if shardedDataCacherNotifier == nil {
		return nil, ErrNilShardedDataCacherNotifier
	}

	var val interface{}
	ok := false
	if searchFirst {
		val, ok = shardedDataCacherNotifier.SearchFirstData(txHash)
		if !ok {
			return nil, ErrTxNotFound
		}
	} else {
		strCache := ShardCacherIdentifier(senderShardID, destShardID)
		txStore := shardedDataCacherNotifier.ShardDataStore(strCache)
		if txStore == nil {
			return nil, ErrNilStorage
		}

		val, ok = txStore.Peek(txHash)
	}

	if !ok {
		return nil, ErrTxNotFound
	}

	tx, ok := val.(data.TransactionHandler)
	if !ok {
		return nil, ErrInvalidTxInPool
	}

	return tx, nil
}

// GetTransactionHandlerFromStorage gets the transaction from storage with a given sender/receiver shardId and txHash
func GetTransactionHandlerFromStorage(
	txHash []byte,
	storageService dataRetriever.StorageService,
	marshalizer marshal.Marshalizer,
) (data.TransactionHandler, error) {

	if storageService == nil || storageService.IsInterfaceNil() {
		return nil, ErrNilStorage
	}
	if marshalizer == nil || marshalizer.IsInterfaceNil() {
		return nil, ErrNilMarshalizer
	}

	txBuff, err := storageService.Get(dataRetriever.TransactionUnit, txHash)
	if err != nil {
		return nil, err
	}

	tx := transaction.Transaction{}
	err = marshalizer.Unmarshal(&tx, txBuff)
	if err != nil {
		return nil, err
	}

	return &tx, nil
}

func checkGetHeaderParamsForNil(
	cacher dataRetriever.HeadersPool,
	marshalizer marshal.Marshalizer,
	storageService dataRetriever.StorageService,
) error {

	if cacher == nil || cacher.IsInterfaceNil() {
		return ErrNilCacher
	}
	if marshalizer == nil || marshalizer.IsInterfaceNil() {
		return ErrNilMarshalizer
	}
	if storageService == nil || storageService.IsInterfaceNil() {
		return ErrNilStorage
	}

	return nil
}

func checkGetHeaderWithNonceParamsForNil(
	headersCacher dataRetriever.HeadersPool,
	marshalizer marshal.Marshalizer,
	storageService dataRetriever.StorageService,
	uint64Converter typeConverters.Uint64ByteSliceConverter,
) error {

	err := checkGetHeaderParamsForNil(headersCacher, marshalizer, storageService)
	if err != nil {
		return err
	}
	if check.IfNil(uint64Converter) {
		return ErrNilUint64Converter
	}

	return nil
}

func checkGetTransactionParamsForNil(
	shardedDataCacherNotifier dataRetriever.ShardedDataCacherNotifier,
	storageService dataRetriever.StorageService,
	marshalizer marshal.Marshalizer,
) error {

	if shardedDataCacherNotifier == nil || shardedDataCacherNotifier.IsInterfaceNil() {
		return ErrNilShardedDataCacherNotifier
	}
	if storageService == nil || storageService.IsInterfaceNil() {
		return ErrNilStorage
	}
	if marshalizer == nil || marshalizer.IsInterfaceNil() {
		return ErrNilMarshalizer
	}

	return nil
}

func getHeaderFromPool(
	hash []byte,
	headersCacher dataRetriever.HeadersPool,
) (interface{}, error) {

	if check.IfNil(headersCacher) {
		return nil, ErrNilCacher
	}

	obj, err := headersCacher.GetHeaderByHash(hash)
	if err != nil {
		return nil, ErrMissingHeader
	}

	return obj, nil
}

func getHeaderFromPoolWithNonce(
	nonce uint64,
	shardId uint32,
	headersCacher dataRetriever.HeadersPool,
) (interface{}, []byte, error) {

	if check.IfNil(headersCacher) {
		return nil, nil, ErrNilCacher
	}

	headers, hashes, err := headersCacher.GetHeadersByNonceAndShardId(nonce, shardId)
	if err != nil {
		return nil, nil, ErrMissingHeader
	}

	//TODO what should we do when we get from pool more than one header with same nonce and shardId
	return headers[len(headers)-1], hashes[len(hashes)-1], nil
}

func getHeaderHashFromStorageWithNonce(
	nonce uint64,
	storageService dataRetriever.StorageService,
	uint64Converter typeConverters.Uint64ByteSliceConverter,
	marshalizer marshal.Marshalizer,
	blockUnit dataRetriever.UnitType,
) ([]byte, error) {

	if storageService == nil || storageService.IsInterfaceNil() {
		return nil, ErrNilStorage
	}
	if uint64Converter == nil || uint64Converter.IsInterfaceNil() {
		return nil, ErrNilUint64Converter
	}
	if marshalizer == nil || marshalizer.IsInterfaceNil() {
		return nil, ErrNilMarshalizer
	}

	headerStore := storageService.GetStorer(blockUnit)
	if headerStore == nil {
		return nil, ErrNilHeadersStorage
	}

	nonceToByteSlice := uint64Converter.ToByteSlice(nonce)
	hash, err := headerStore.Get(nonceToByteSlice)
	if err != nil {
		return nil, ErrMissingHashForHeaderNonce
	}

	return hash, nil
}

// SortHeadersByNonce will sort a given list of headers by nonce
func SortHeadersByNonce(headers []data.HeaderHandler) {
	if len(headers) > 1 {
		sort.Slice(headers, func(i, j int) bool {
			return headers[i].GetNonce() < headers[j].GetNonce()
		})
	}
}

// IsInProperRound checks if the given round index satisfies the round modulus trigger
func IsInProperRound(index int64) bool {
	return index%RoundModulusTrigger == 0
}

// AddHeaderToBlackList adds a hash to black list handler. Logs if the operation did not succeed
func AddHeaderToBlackList(blackListHandler BlackListHandler, hash []byte) {
	blackListHandler.Sweep()
	err := blackListHandler.Add(string(hash))
	if err != nil {
		log.Trace("blackListHandler.Add", "error", err.Error())
	}

	log.Debug("header has been added to blacklist",
		"hash", hash)
}

// ForkInfo hold the data related to a detected fork
type ForkInfo struct {
	IsDetected bool
	Nonce      uint64
	Round      uint64
	Hash       []byte
}

// NewForkInfo creates a new ForkInfo object
func NewForkInfo() *ForkInfo {
	return &ForkInfo{IsDetected: false, Nonce: math.MaxUint64, Round: math.MaxUint64, Hash: nil}
}

// DisplayProcessTxDetails displays information related to the tx which should be executed
func DisplayProcessTxDetails(
	message string,
	accountHandler state.AccountHandler,
	txHandler data.TransactionHandler,
) {
	if !check.IfNil(accountHandler) {
		account, ok := accountHandler.(*state.Account)
		if ok {
			log.Trace(message,
				"nonce", account.Nonce,
				"balance", account.Balance,
			)
		}
	}

	if !check.IfNil(txHandler) {
		log.Trace("executing transaction",
			"nonce", txHandler.GetNonce(),
			"value", txHandler.GetValue(),
			"gas limit", txHandler.GetGasLimit(),
			"gas price", txHandler.GetGasPrice(),
			"data", hex.EncodeToString(txHandler.GetData()),
			"sender", hex.EncodeToString(txHandler.GetSndAddress()),
			"receiver", hex.EncodeToString(txHandler.GetRecvAddress()))
	}
}
