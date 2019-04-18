package sync

import (
	"fmt"
	"time"

	"github.com/ElrondNetwork/elrond-go-sandbox/consensus"
	"github.com/ElrondNetwork/elrond-go-sandbox/data"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/block"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/state"
	"github.com/ElrondNetwork/elrond-go-sandbox/dataRetriever"
	"github.com/ElrondNetwork/elrond-go-sandbox/hashing"
	"github.com/ElrondNetwork/elrond-go-sandbox/marshal"
	"github.com/ElrondNetwork/elrond-go-sandbox/process"
	"github.com/ElrondNetwork/elrond-go-sandbox/process/factory"
	"github.com/ElrondNetwork/elrond-go-sandbox/sharding"
	"github.com/ElrondNetwork/elrond-go-sandbox/storage"
)

// MetaBootstrap implements the bootstrap mechanism
type MetaBootstrap struct {
	*baseBootstrap

	resolversFinder dataRetriever.ResolversFinder
	hdrRes          dataRetriever.HeaderResolver
}

// NewMetaBootstrap creates a new Bootstrap object
func NewMetaBootstrap(
	poolsHolder dataRetriever.MetaPoolsHolder,
	store dataRetriever.StorageService,
	blkc data.ChainHandler,
	rounder consensus.Rounder,
	blkExecutor process.BlockProcessor,
	waitTime time.Duration,
	hasher hashing.Hasher,
	marshalizer marshal.Marshalizer,
	forkDetector process.ForkDetector,
	resolversFinder dataRetriever.ResolversFinder,
	shardCoordinator sharding.Coordinator,
	accounts state.AccountsAdapter,
) (*MetaBootstrap, error) {

	if poolsHolder == nil {
		return nil, process.ErrNilPoolsHolder
	}
	if poolsHolder.MetaBlockNonces() == nil {
		return nil, process.ErrNilHeadersNoncesDataPool
	}
	if poolsHolder.MetaChainBlocks() == nil {
		return nil, process.ErrNilMetaBlockPool
	}

	err := checkBootstrapNilParameters(
		blkc,
		rounder,
		blkExecutor,
		hasher,
		marshalizer,
		forkDetector,
		resolversFinder,
		shardCoordinator,
		accounts,
		store,
	)
	if err != nil {
		return nil, err
	}

	base := &baseBootstrap{
		blkc:             blkc,
		blkExecutor:      blkExecutor,
		store:            store,
		headers:          poolsHolder.MetaChainBlocks(),
		headersNonces:    poolsHolder.MetaBlockNonces(),
		rounder:          rounder,
		waitTime:         waitTime,
		hasher:           hasher,
		marshalizer:      marshalizer,
		forkDetector:     forkDetector,
		shardCoordinator: shardCoordinator,
		accounts:         accounts,
	}

	boot := MetaBootstrap{
		baseBootstrap: base,
	}

	//there is one header topic so it is ok to save it
	hdrResolver, err := resolversFinder.MetaChainResolver(factory.MetachainBlocksTopic)
	if err != nil {
		return nil, err
	}

	//placed in struct fields for performance reasons
	hdrRes, ok := hdrResolver.(dataRetriever.HeaderResolver)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}
	boot.hdrRes = hdrRes

	boot.chRcvHdr = make(chan bool)

	boot.setRequestedHeaderNonce(nil)
	boot.headersNonces.RegisterHandler(boot.receivedHeaderNonce)
	boot.headers.RegisterHandler(boot.receivedHeader)

	boot.chStopSync = make(chan bool)

	boot.syncStateListeners = make([]func(bool), 0)
	boot.requestedHashes = process.RequiredDataPool{}

	return &boot, nil
}

func (boot *MetaBootstrap) getHeader(hash []byte) *block.MetaBlock {
	hdr := boot.getHeaderFromPool(hash)
	if hdr != nil {
		return hdr
	}

	return boot.getHeaderFromStorage(hash)
}

func (boot *MetaBootstrap) getHeaderFromPool(hash []byte) *block.MetaBlock {
	hdr, ok := boot.headers.Peek(hash)
	if !ok {
		log.Debug(fmt.Sprintf("header with hash %v not found in headers cache\n", hash))
		return nil
	}

	header, ok := hdr.(*block.MetaBlock)
	if !ok {
		log.Debug(fmt.Sprintf("data with hash %v is not metablock\n", hash))
		return nil
	}

	return header
}

func (boot *MetaBootstrap) getHeaderFromStorage(hash []byte) *block.MetaBlock {
	headerStore := boot.store.GetStorer(dataRetriever.MetaBlockUnit)
	if headerStore == nil {
		log.Error(process.ErrNilHeadersStorage.Error())
		return nil
	}

	buffHeader, _ := headerStore.Get(hash)
	header := &block.MetaBlock{}
	err := boot.marshalizer.Unmarshal(header, buffHeader)
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	return header
}

func (boot *MetaBootstrap) receivedHeader(headerHash []byte) {
	header := boot.getHeader(headerHash)
	boot.processReceivedHeader(header, headerHash)
}

// StartSync method will start SyncBlocks as a go routine
func (boot *MetaBootstrap) StartSync() {
	go boot.syncBlocks()
}

// StopSync method will stop SyncBlocks
func (boot *MetaBootstrap) StopSync() {
	boot.chStopSync <- true
}

// syncBlocks method calls repeatedly synchronization method SyncBlock
func (boot *MetaBootstrap) syncBlocks() {
	for {
		time.Sleep(sleepTime)
		select {
		case <-boot.chStopSync:
			return
		default:
			err := boot.SyncBlock()

			if err != nil {
				log.Info(err.Error())
			}
		}
	}
}

// SyncBlock method actually does the synchronization. It requests the next block header from the pool
// and if it is not found there it will be requested from the network. After the header is received,
// it requests the block body in the same way(pool and than, if it is not found in the pool, from network).
// If either header and body are received the ProcessAndCommit method will be called. This method will execute
// the block and its transactions. Finally if everything works, the block will be committed in the blockchain,
// and all this mechanism will be reiterated for the next block.
func (boot *MetaBootstrap) SyncBlock() error {
	if !boot.ShouldSync() {
		return nil
	}

	if boot.isForkDetected {
		log.Info(fmt.Sprintf("fork detected at nonce %d\n", boot.forkNonce))
		return boot.forkChoice()
	}

	boot.setRequestedHeaderNonce(nil)

	nonce := boot.getNonceForNextBlock()

	hdr, err := boot.getHeaderRequestingIfMissing(nonce)
	if err != nil {
		if err == process.ErrMissingHeader {
			if boot.shouldCreateEmptyBlock(nonce) {
				log.Info(err.Error())
				err = boot.createAndBroadcastEmptyBlock()
			}
		}

		return err
	}

	haveTime := func() time.Duration {
		return boot.rounder.TimeDuration()
	}

	err = boot.blkExecutor.ProcessBlock(boot.blkc, hdr, nil, haveTime)
	if err != nil {
		isForkDetected := err == process.ErrInvalidBlockHash || err == process.ErrRootStateMissmatch
		if isForkDetected {
			log.Info(err.Error())
			boot.removeHeaderFromPools(hdr)
			err = boot.forkChoice()
		}

		return err
	}

	err = boot.blkExecutor.CommitBlock(boot.blkc, hdr, nil)
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("block with nonce %d was synced successfully\n", hdr.Nonce))
	return nil
}

func (boot *MetaBootstrap) createAndBroadcastEmptyBlock() error {
	txBlockBody, header, err := boot.CreateAndCommitEmptyBlock(boot.shardCoordinator.SelfId())

	if err == nil {
		log.Info(fmt.Sprintf("body and header with root hash %s and nonce %d were created and commited through the recovery mechanism\n",
			toB64(header.GetRootHash()),
			header.GetNonce()))

		err = boot.broadcastEmptyBlock(txBlockBody.(block.Body), header.(*block.MetaBlock))
	}

	return err
}

// getHeaderFromPoolHavingNonce method returns the block header from a given nonce
func (boot *MetaBootstrap) getHeaderFromPoolHavingNonce(nonce uint64) *block.MetaBlock {
	hash, _ := boot.headersNonces.Get(nonce)
	if hash == nil {
		log.Debug(fmt.Sprintf("nonce %d not found in headers-nonces cache\n", nonce))
		return nil
	}

	hdr, ok := boot.headers.Peek(hash)
	if !ok {
		log.Debug(fmt.Sprintf("header with hash %v not found in headers cache\n", hash))
		return nil
	}

	header, ok := hdr.(*block.MetaBlock)
	if !ok {
		log.Debug(fmt.Sprintf("header with hash %v not found in headers cache\n", hash))
		return nil
	}

	return header
}

// requestHeader method requests a block header from network when it is not found in the pool
func (boot *MetaBootstrap) requestHeader(nonce uint64) {
	boot.setRequestedHeaderNonce(&nonce)
	err := boot.hdrRes.RequestDataFromNonce(nonce)

	log.Info(fmt.Sprintf("requested header with nonce %d from network\n", nonce))

	if err != nil {
		log.Error(err.Error())
	}
}

// getHeaderWithNonce method gets the header with given nonce from pool, if it exist there,
// and if not it will be requested from network
func (boot *MetaBootstrap) getHeaderRequestingIfMissing(nonce uint64) (*block.MetaBlock, error) {
	hdr := boot.getHeaderFromPoolHavingNonce(nonce)

	if hdr == nil {
		emptyChannel(boot.chRcvHdr)
		boot.requestHeader(nonce)
		boot.waitForHeaderNonce()
		hdr = boot.getHeaderFromPoolHavingNonce(nonce)
		if hdr == nil {
			return nil, process.ErrMissingHeader
		}
	}

	return hdr, nil
}

// forkChoice decides if rollback must be called
func (boot *MetaBootstrap) forkChoice() error {
	log.Info("starting fork choice\n")
	isForkResolved := false
	for !isForkResolved {
		header, err := boot.getCurrentHeader()
		if err != nil {
			return err
		}

		msg := fmt.Sprintf("roll back to header with nonce %d and hash %s",
			header.GetNonce()-1, toB64(header.GetPrevHash()))

		isSigned := isSigned(header)
		if isSigned {
			msg = fmt.Sprintf("%s from a signed block, as the highest final block nonce is %d",
				msg,
				boot.forkDetector.GetHighestFinalBlockNonce())
			canRevertBlock := header.GetNonce() > boot.forkDetector.GetHighestFinalBlockNonce()
			if !canRevertBlock {
				return &ErrSignedBlock{CurrentNonce: header.GetNonce()}
			}
		}

		log.Info(msg + "\n")

		err = boot.rollback(header)
		if err != nil {
			return err
		}

		if header.GetNonce() <= boot.forkNonce {
			isForkResolved = true
		}
	}

	log.Info("ending fork choice\n")
	return nil
}

func (boot *MetaBootstrap) rollback(header *block.MetaBlock) error {
	if header.GetNonce() == 0 {
		return process.ErrRollbackFromGenesis
	}
	headerStore := boot.store.GetStorer(dataRetriever.MetaBlockUnit)
	if headerStore == nil {
		return process.ErrNilHeadersStorage
	}

	var err error
	var newHeader *block.MetaBlock
	var newHeaderHash []byte
	var newRootHash []byte

	if header.GetNonce() > 1 {
		newHeader, err = boot.getPrevHeader(headerStore, header)
		if err != nil {
			return err
		}

		newHeaderHash = header.GetPrevHash()
		newRootHash = newHeader.GetRootHash()
	} else { // rollback to genesis block
		newRootHash = boot.blkc.GetGenesisHeader().GetRootHash()
	}

	err = boot.blkc.SetCurrentBlockHeader(newHeader)
	if err != nil {
		return err
	}

	boot.blkc.SetCurrentBlockHeaderHash(newHeaderHash)

	err = boot.accounts.RecreateTrie(newRootHash)
	if err != nil {
		return err
	}

	boot.cleanCachesOnRollback(header, headerStore)
	errNotCritical := boot.blkExecutor.RestoreBlockIntoPools(boot.blkc, nil)
	if errNotCritical != nil {
		log.Info(errNotCritical.Error())
	}

	return nil
}

func (boot *MetaBootstrap) getPrevHeader(headerStore storage.Storer, header *block.MetaBlock) (*block.MetaBlock, error) {
	prevHash := header.GetPrevHash()
	buffHeader, _ := headerStore.Get(prevHash)
	newHeader := &block.MetaBlock{}

	err := boot.marshalizer.Unmarshal(newHeader, buffHeader)
	if err != nil {
		return nil, err
	}

	return newHeader, nil
}

func (boot *MetaBootstrap) createHeader() (data.HeaderHandler, error) {
	var prevHeaderHash []byte
	hdr := &block.MetaBlock{}

	if boot.blkc.GetCurrentBlockHeader() == nil {
		hdr.Nonce = 1
		hdr.Round = 0
		prevHeaderHash = boot.blkc.GetGenesisHeaderHash()
		hdr.PrevRandSeed = boot.blkc.GetGenesisHeader().GetSignature()
	} else {
		hdr.Nonce = boot.blkc.GetCurrentBlockHeader().GetNonce() + 1
		hdr.Round = boot.blkc.GetCurrentBlockHeader().GetRound() + 1
		prevHeaderHash = boot.blkc.GetCurrentBlockHeaderHash()
		hdr.PrevRandSeed = boot.blkc.GetCurrentBlockHeader().GetSignature()
	}

	hdr.RandSeed = []byte{0}
	hdr.TimeStamp = uint64(boot.getTimeStampForRound(hdr.Round).Unix())
	hdr.SetPrevHash(prevHeaderHash)
	hdr.SetRootHash(boot.accounts.RootHash())
	hdr.PubKeysBitmap = make([]byte, 0)

	return hdr, nil
}

// CreateAndCommitEmptyBlock creates and commits an empty block
func (boot *MetaBootstrap) CreateAndCommitEmptyBlock(shardForCurrentNode uint32) (data.BodyHandler, data.HeaderHandler, error) {
	log.Info(fmt.Sprintf("creating and commiting an empty block\n"))

	blk := make(block.Body, 0)

	hdr, err := boot.createHeader()
	if err != nil {
		return nil, nil, err
	}

	// TODO: decide the signature for the empty block
	headerStr, err := boot.marshalizer.Marshal(hdr)
	if err != nil {
		return nil, nil, err
	}
	hdrHash := boot.hasher.Compute(string(headerStr))
	hdr.SetSignature(hdrHash)

	// Commit the block (commits also the account state)
	err = boot.blkExecutor.CommitBlock(boot.blkc, hdr, blk)
	if err != nil {
		return nil, nil, err
	}

	msg := fmt.Sprintf("Added empty block with nonce  %d  in blockchain", hdr.GetNonce())
	log.Info(log.Headline(msg, "", "*"))

	return blk, hdr, nil
}

func (boot *MetaBootstrap) getCurrentHeader() (*block.MetaBlock, error) {
	blockHeader := boot.blkc.GetCurrentBlockHeader()
	if blockHeader == nil {
		return nil, process.ErrNilBlockHeader
	}

	header, ok := blockHeader.(*block.MetaBlock)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	return header, nil
}
