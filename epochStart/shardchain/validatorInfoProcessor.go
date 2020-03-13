package shardchain

import (
	"fmt"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/storage"
)

type miniBlockInfo struct {
	mb *block.MiniBlock
}

// ArgValidatorInfoProcessor holds all dependencies required to create a validatorInfoProcessor
type ArgValidatorInfoProcessor struct {
	MiniBlocksPool               storage.Cacher
	Marshalizer                  marshal.Marshalizer
	ValidatorStatisticsProcessor epochStart.ValidatorStatisticsProcessorHandler
	Requesthandler               process.RequestHandler
	Hasher                       hashing.Hasher
}

// ValidatorInfoProcessor implements validator info processing for miniblocks of type peerMiniblock
type ValidatorInfoProcessor struct {
	miniBlocksPool               storage.Cacher
	marshalizer                  marshal.Marshalizer
	Hasher                       hashing.Hasher
	validatorStatisticsProcessor epochStart.ValidatorStatisticsProcessorHandler
	requestHandler               epochStart.RequestHandler

	allPeerMiniblocks     map[string]*miniBlockInfo
	headerHash            []byte
	metaHeader            data.HeaderHandler
	chRcvAllMiniblocks    chan struct{}
	mutMiniBlocksForBlock sync.Mutex
	numMissing            uint32
}

// NewValidatorInfoProcessor creates a new ValidatorInfoProcessor object
func NewValidatorInfoProcessor(arguments ArgValidatorInfoProcessor) (*ValidatorInfoProcessor, error) {
	if check.IfNil(arguments.ValidatorStatisticsProcessor) {
		return nil, process.ErrNilValidatorStatistics
	}
	if check.IfNil(arguments.Marshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(arguments.MiniBlocksPool) {
		return nil, process.ErrNilMiniBlockPool
	}
	if check.IfNil(arguments.Requesthandler) {
		return nil, process.ErrNilRequestHandler
	}

	vip := &ValidatorInfoProcessor{
		miniBlocksPool:               arguments.MiniBlocksPool,
		marshalizer:                  arguments.Marshalizer,
		validatorStatisticsProcessor: arguments.ValidatorStatisticsProcessor,
		requestHandler:               arguments.Requesthandler,
		Hasher:                       arguments.Hasher,
	}

	//TODO: change the registerHandler for the miniblockPool to call
	//directly with hash and value - like func (sp *shardProcessor) receivedMetaBlock
	vip.miniBlocksPool.RegisterHandler(vip.receivedMiniBlock)

	return vip, nil
}

func (vip *ValidatorInfoProcessor) init(metaBlock *block.MetaBlock, metablockHash []byte) {
	vip.mutMiniBlocksForBlock.Lock()
	vip.metaHeader = metaBlock
	vip.allPeerMiniblocks = make(map[string]*miniBlockInfo)
	vip.headerHash = metablockHash
	vip.chRcvAllMiniblocks = make(chan struct{})
	vip.mutMiniBlocksForBlock.Unlock()
}

// ProcessMetaBlock processes an epochstart block asyncrhonous, processing the PeerMiniblocks
func (vip *ValidatorInfoProcessor) ProcessMetaBlock(metaBlock *block.MetaBlock, metablockHash []byte) error {
	vip.init(metaBlock, metablockHash)

	vip.computeMissingPeerBlocks(metaBlock)

	err := vip.retrieveMissingBlocks()
	if err != nil {
		return err
	}

	err = vip.processAllPeerMiniBlocks(metaBlock)
	if err != nil {
		return err
	}

	return nil
}

func (vip *ValidatorInfoProcessor) receivedMiniBlock(key []byte) {
	mb, ok := vip.miniBlocksPool.Get(key)
	if !ok {
		return
	}

	peerMb, ok := mb.(*block.MiniBlock)
	if !ok || peerMb.Type != block.PeerBlock {
		return
	}

	log.Trace(fmt.Sprintf("received miniblock of type %s", peerMb.Type))

	vip.mutMiniBlocksForBlock.Lock()
	mbInfo, ok := vip.allPeerMiniblocks[string(key)]
	if !ok || mbInfo.mb == nil {
		vip.mutMiniBlocksForBlock.Unlock()
		return
	}

	vip.allPeerMiniblocks[string(key)].mb = peerMb
	vip.numMissing--
	missingPending := vip.numMissing
	vip.mutMiniBlocksForBlock.Unlock()

	if missingPending == 0 {
		vip.chRcvAllMiniblocks <- struct{}{}
	}
}

func (vip *ValidatorInfoProcessor) processAllPeerMiniBlocks(metaBlock *block.MetaBlock) error {
	for _, peerMiniBlock := range metaBlock.MiniBlockHeaders {
		if peerMiniBlock.Type != block.PeerBlock {
			continue
		}

		mb := vip.allPeerMiniblocks[string(peerMiniBlock.Hash)].mb
		for _, txHash := range mb.TxHashes {
			vid := &state.ValidatorInfo{}
			err := vip.marshalizer.Unmarshal(vid, txHash)
			if err != nil {
				return err
			}

			err = vip.validatorStatisticsProcessor.Process(vid)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (vip *ValidatorInfoProcessor) computeMissingPeerBlocks(metaBlock *block.MetaBlock) {
	missingNumber := uint32(0)
	vip.mutMiniBlocksForBlock.Lock()

	for _, mb := range metaBlock.MiniBlockHeaders {
		if mb.Type != block.PeerBlock {
			continue
		}

		vip.allPeerMiniblocks[string(mb.Hash)] = &miniBlockInfo{}
		mbObjectFound, ok := vip.miniBlocksPool.Peek(mb.Hash)
		if !ok {
			missingNumber++
			continue
		}

		mbFound := mbObjectFound.(*block.MiniBlock)
		if mbFound == nil {
			missingNumber++
		}
		vip.allPeerMiniblocks[string(mb.Hash)] = &miniBlockInfo{mb: mbFound}
	}

	vip.numMissing = missingNumber
	vip.mutMiniBlocksForBlock.Unlock()
}

func (vip *ValidatorInfoProcessor) retrieveMissingBlocks() error {
	vip.mutMiniBlocksForBlock.Lock()
	missingMiniblocks := make([][]byte, 0)
	for mbHash, mbInfo := range vip.allPeerMiniblocks {
		if mbInfo.mb == nil {
			missingMiniblocks = append(missingMiniblocks, []byte(mbHash))
		}
	}
	vip.numMissing = uint32(len(missingMiniblocks))
	vip.mutMiniBlocksForBlock.Unlock()

	if len(missingMiniblocks) == 0 {
		return nil
	}

	go vip.requestHandler.RequestMiniBlocks(core.MetachainShardId, missingMiniblocks)

	select {
	case <-vip.chRcvAllMiniblocks:
		return nil
	case <-time.After(time.Second):
		return process.ErrTimeIsOut
	}
}

// IsInterfaceNil returns true if underlying object is nil
func (vip *ValidatorInfoProcessor) IsInterfaceNil() bool {
	return vip == nil
}
