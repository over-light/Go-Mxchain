package groups

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go/api/errors"
	"github.com/ElrondNetwork/elrond-go/api/shared"
	"github.com/gin-gonic/gin"
)

const (
	getRawMetaBlockByNoncePath       = "/raw/metablock/by-nonce/:nonce"
	getRawMetaBlockByHashPath        = "/raw/metablock/by-hash/:hash"
	getRawMetaBlockByRoundPath       = "/raw/metablock/by-round/:round"
	getRawShardBlockByNoncePath      = "/raw/shardblock/by-nonce/:nonce"
	getRawShardBlockByHashPath       = "/raw/shardblock/by-hash/:hash"
	getRawShardBlockByRoundPath      = "/raw/shardblock/by-round/:round"
	getInternalMetaBlockByNoncePath  = "/json/metablock/by-nonce/:nonce"
	getInternalMetaBlockByHashPath   = "/json/metablock/by-hash/:hash"
	getInternalMetaBlockByRoundPath  = "/json/metablock/by-round/:round"
	getInternalShardBlockByNoncePath = "/json/shardblock/by-nonce/:nonce"
	getInternalShardBlockByHashPath  = "/json/shardblock/by-hash/:hash"
	getInternalShardBlockByRoundPath = "/json/shardblock/by-round/:round"
	getRawMiniBlockByHashPath        = "/raw/miniblock/by-hash/:hash"
	getInternalMiniBlockByHashPath   = "/json/miniblock/by-hash/:hash"
)

// rawBlockFacadeHandler defines the methods to be implemented by a facade for handling block requests
type rawBlockFacadeHandler interface {
	GetRawMetaBlockByHash(hash string) ([]byte, error)
	GetRawMetaBlockByNonce(nonce uint64) ([]byte, error)
	GetRawMetaBlockByRound(round uint64) ([]byte, error)
	GetRawShardBlockByHash(hash string) ([]byte, error)
	GetRawShardBlockByNonce(nonce uint64) ([]byte, error)
	GetRawShardBlockByRound(round uint64) ([]byte, error)
	GetInternalMetaBlockByHash(hash string) (*block.MetaBlock, error)
	GetInternalMetaBlockByNonce(nonce uint64) (*block.MetaBlock, error)
	GetInternalMetaBlockByRound(round uint64) (*block.MetaBlock, error)
	GetInternalShardBlockByHash(hash string) (*block.Header, error)
	GetInternalShardBlockByNonce(nonce uint64) (*block.Header, error)
	GetInternalShardBlockByRound(round uint64) (*block.Header, error)
	GetRawMiniBlockByHash(hash string) ([]byte, error)
	GetInternalMiniBlockByHash(hash string) (*block.MiniBlock, error)
	IsInterfaceNil() bool
}

type rawBlockGroup struct {
	*baseGroup
	facade    rawBlockFacadeHandler
	mutFacade sync.RWMutex
}

// NewRawBlockGroup returns a new instance of rawBlockGroup
func NewRawBlockGroup(facade rawBlockFacadeHandler) (*rawBlockGroup, error) {
	if check.IfNil(facade) {
		return nil, fmt.Errorf("%w for raw block group", errors.ErrNilFacadeHandler)
	}

	rb := &rawBlockGroup{
		facade:    facade,
		baseGroup: &baseGroup{},
	}

	endpoints := []*shared.EndpointHandlerData{
		{
			Path:    getRawMetaBlockByNoncePath,
			Method:  http.MethodGet,
			Handler: rb.getRawMetaBlockByNonce,
		},
		{
			Path:    getRawMetaBlockByHashPath,
			Method:  http.MethodGet,
			Handler: rb.getRawMetaBlockByHash,
		},
		{
			Path:    getRawMetaBlockByRoundPath,
			Method:  http.MethodGet,
			Handler: rb.getRawMetaBlockByRound,
		},
		{
			Path:    getRawShardBlockByNoncePath,
			Method:  http.MethodGet,
			Handler: rb.getRawShardBlockByNonce,
		},
		{
			Path:    getRawShardBlockByHashPath,
			Method:  http.MethodGet,
			Handler: rb.getRawShardBlockByHash,
		},
		{
			Path:    getRawShardBlockByRoundPath,
			Method:  http.MethodGet,
			Handler: rb.getRawShardBlockByRound,
		},
		{
			Path:    getInternalMetaBlockByNoncePath,
			Method:  http.MethodGet,
			Handler: rb.getInternalMetaBlockByNonce,
		},
		{
			Path:    getInternalMetaBlockByHashPath,
			Method:  http.MethodGet,
			Handler: rb.getInternalMetaBlockByHash,
		},
		{
			Path:    getInternalMetaBlockByRoundPath,
			Method:  http.MethodGet,
			Handler: rb.getInternalMetaBlockByRound,
		},
		{
			Path:    getInternalShardBlockByNoncePath,
			Method:  http.MethodGet,
			Handler: rb.getInternalShardBlockByNonce,
		},
		{
			Path:    getInternalShardBlockByHashPath,
			Method:  http.MethodGet,
			Handler: rb.getInternalShardBlockByHash,
		},
		{
			Path:    getInternalShardBlockByRoundPath,
			Method:  http.MethodGet,
			Handler: rb.getInternalShardBlockByRound,
		},
		{
			Path:    getRawMiniBlockByHashPath,
			Method:  http.MethodGet,
			Handler: rb.getRawMiniBlockByHash,
		},
		{
			Path:    getInternalMiniBlockByHashPath,
			Method:  http.MethodGet,
			Handler: rb.getInternalMiniBlockByHash,
		},
	}
	rb.endpoints = endpoints

	return rb, nil
}

func (rb *rawBlockGroup) getRawMetaBlockByNonce(c *gin.Context) {
	nonce, err := getQueryParamNonce(c)
	if err != nil {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrInvalidBlockNonce.Error()),
		)
		return
	}

	start := time.Now()
	rawBlock, err := rb.getFacade().GetRawMetaBlockByNonce(nonce)
	log.Debug(fmt.Sprintf("GetRawMetaBlockByNonce took %s", time.Since(start)))
	if err != nil {
		shared.RespondWith(
			c,
			http.StatusInternalServerError,
			nil,
			fmt.Sprintf("%s: %s", errors.ErrGetBlock.Error(), err.Error()),
			shared.ReturnCodeInternalError,
		)
		return
	}

	shared.RespondWith(c, http.StatusOK, gin.H{"block": rawBlock}, "", shared.ReturnCodeSuccess)
}

func (rb *rawBlockGroup) getRawMetaBlockByHash(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrValidationEmptyBlockHash.Error()),
		)
		return
	}

	start := time.Now()
	rawBlock, err := rb.getFacade().GetRawMetaBlockByHash(hash)
	log.Debug(fmt.Sprintf("GetRawMetaBlockByHash took %s", time.Since(start)))
	if err != nil {
		shared.RespondWith(
			c,
			http.StatusInternalServerError,
			nil,
			fmt.Sprintf("%s: %s", errors.ErrGetBlock.Error(), err.Error()),
			shared.ReturnCodeInternalError,
		)
		return
	}

	shared.RespondWith(c, http.StatusOK, gin.H{"block": rawBlock}, "", shared.ReturnCodeSuccess)
}

func (rb *rawBlockGroup) getRawMetaBlockByRound(c *gin.Context) {
	round, err := getQueryParamRound(c)
	if err != nil {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrInvalidBlockRound.Error()),
		)
		return
	}

	start := time.Now()
	rawBlock, err := rb.getFacade().GetRawMetaBlockByRound(round)
	log.Debug(fmt.Sprintf("GetRawMetaBlockByRound took %s", time.Since(start)))
	if err != nil {
		shared.RespondWith(
			c,
			http.StatusInternalServerError,
			nil,
			fmt.Sprintf("%s: %s", errors.ErrGetBlock.Error(), err.Error()),
			shared.ReturnCodeInternalError,
		)
		return
	}

	shared.RespondWith(c, http.StatusOK, gin.H{"block": rawBlock}, "", shared.ReturnCodeSuccess)
}

func (rb *rawBlockGroup) getRawShardBlockByNonce(c *gin.Context) {
	nonce, err := getQueryParamNonce(c)
	if err != nil {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrInvalidBlockNonce.Error()),
		)
		return
	}

	start := time.Now()
	rawBlock, err := rb.getFacade().GetRawShardBlockByNonce(nonce)
	log.Debug(fmt.Sprintf("GetRawShardBlockByNonce took %s", time.Since(start)))
	if err != nil {
		shared.RespondWith(
			c,
			http.StatusInternalServerError,
			nil,
			fmt.Sprintf("%s: %s", errors.ErrGetBlock.Error(), err.Error()),
			shared.ReturnCodeInternalError,
		)
		return
	}

	shared.RespondWith(c, http.StatusOK, gin.H{"block": rawBlock}, "", shared.ReturnCodeSuccess)
}

func (rb *rawBlockGroup) getRawShardBlockByHash(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrValidationEmptyBlockHash.Error()),
		)
		return
	}

	start := time.Now()
	rawBlock, err := rb.getFacade().GetRawShardBlockByHash(hash)
	log.Debug(fmt.Sprintf("GetRawShardBlockByHash took %s", time.Since(start)))
	if err != nil {
		shared.RespondWith(
			c,
			http.StatusInternalServerError,
			nil,
			fmt.Sprintf("%s: %s", errors.ErrGetBlock.Error(), err.Error()),
			shared.ReturnCodeInternalError,
		)
		return
	}

	shared.RespondWith(c, http.StatusOK, gin.H{"block": rawBlock}, "", shared.ReturnCodeSuccess)
}

func (rb *rawBlockGroup) getRawShardBlockByRound(c *gin.Context) {
	round, err := getQueryParamRound(c)
	if err != nil {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrInvalidBlockRound.Error()),
		)
		return
	}

	start := time.Now()
	rawBlock, err := rb.getFacade().GetRawShardBlockByRound(round)
	log.Debug(fmt.Sprintf("GetRawShardBlockByRound took %s", time.Since(start)))
	if err != nil {
		shared.RespondWith(
			c,
			http.StatusInternalServerError,
			nil,
			fmt.Sprintf("%s: %s", errors.ErrGetBlock.Error(), err.Error()),
			shared.ReturnCodeInternalError,
		)
		return
	}

	shared.RespondWith(c, http.StatusOK, gin.H{"block": rawBlock}, "", shared.ReturnCodeSuccess)
}

func (rb *rawBlockGroup) getInternalMetaBlockByNonce(c *gin.Context) {
	nonce, err := getQueryParamNonce(c)
	if err != nil {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrInvalidBlockNonce.Error()),
		)
		return
	}

	start := time.Now()
	block, err := rb.getFacade().GetInternalMetaBlockByNonce(nonce)
	log.Debug(fmt.Sprintf("GetInternalMetaBlockByNonce took %s", time.Since(start)))
	if err != nil {
		shared.RespondWith(
			c,
			http.StatusInternalServerError,
			nil,
			fmt.Sprintf("%s: %s", errors.ErrGetBlock.Error(), err.Error()),
			shared.ReturnCodeInternalError,
		)
		return
	}

	shared.RespondWith(c, http.StatusOK, gin.H{"block": block}, "", shared.ReturnCodeSuccess)
}

func (rb *rawBlockGroup) getInternalMetaBlockByHash(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrValidationEmptyBlockHash.Error()),
		)
		return
	}

	start := time.Now()
	block, err := rb.getFacade().GetInternalMetaBlockByHash(hash)
	log.Debug(fmt.Sprintf("GetInternalMetaBlockByHash took %s", time.Since(start)))
	if err != nil {
		shared.RespondWith(
			c,
			http.StatusInternalServerError,
			nil,
			fmt.Sprintf("%s: %s", errors.ErrGetBlock.Error(), err.Error()),
			shared.ReturnCodeInternalError,
		)
		return
	}

	shared.RespondWith(c, http.StatusOK, gin.H{"block": block}, "", shared.ReturnCodeSuccess)
}

func (rb *rawBlockGroup) getInternalMetaBlockByRound(c *gin.Context) {
	round, err := getQueryParamRound(c)
	if err != nil {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrInvalidBlockRound.Error()),
		)
		return
	}

	start := time.Now()
	block, err := rb.getFacade().GetInternalMetaBlockByRound(round)
	log.Debug(fmt.Sprintf("GetInternalMetaBlockByRound took %s", time.Since(start)))
	if err != nil {
		shared.RespondWith(
			c,
			http.StatusInternalServerError,
			nil,
			fmt.Sprintf("%s: %s", errors.ErrGetBlock.Error(), err.Error()),
			shared.ReturnCodeInternalError,
		)
		return
	}

	shared.RespondWith(c, http.StatusOK, gin.H{"block": block}, "", shared.ReturnCodeSuccess)
}

func (rb *rawBlockGroup) getInternalShardBlockByNonce(c *gin.Context) {
	nonce, err := getQueryParamNonce(c)
	if err != nil {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrInvalidBlockNonce.Error()),
		)
		return
	}

	start := time.Now()
	block, err := rb.getFacade().GetInternalShardBlockByNonce(nonce)
	log.Debug(fmt.Sprintf("GetInternalShardBlockByNonce took %s", time.Since(start)))
	if err != nil {
		shared.RespondWith(
			c,
			http.StatusInternalServerError,
			nil,
			fmt.Sprintf("%s: %s", errors.ErrGetBlock.Error(), err.Error()),
			shared.ReturnCodeInternalError,
		)
		return
	}

	shared.RespondWith(c, http.StatusOK, gin.H{"block": block}, "", shared.ReturnCodeSuccess)
}

func (rb *rawBlockGroup) getInternalShardBlockByHash(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrValidationEmptyBlockHash.Error()),
		)
		return
	}

	start := time.Now()
	block, err := rb.getFacade().GetInternalShardBlockByHash(hash)
	log.Debug(fmt.Sprintf("GetInternalShardBlockByHash took %s", time.Since(start)))
	if err != nil {
		shared.RespondWith(
			c,
			http.StatusInternalServerError,
			nil,
			fmt.Sprintf("%s: %s", errors.ErrGetBlock.Error(), err.Error()),
			shared.ReturnCodeInternalError,
		)
		return
	}

	shared.RespondWith(c, http.StatusOK, gin.H{"block": block}, "", shared.ReturnCodeSuccess)
}

func (rb *rawBlockGroup) getInternalShardBlockByRound(c *gin.Context) {
	round, err := getQueryParamRound(c)
	if err != nil {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrInvalidBlockRound.Error()),
		)
		return
	}

	start := time.Now()
	block, err := rb.getFacade().GetInternalShardBlockByRound(round)
	log.Debug(fmt.Sprintf("GetInternalShardBlockByRound took %s", time.Since(start)))
	if err != nil {
		shared.RespondWith(
			c,
			http.StatusInternalServerError,
			nil,
			fmt.Sprintf("%s: %s", errors.ErrGetBlock.Error(), err.Error()),
			shared.ReturnCodeInternalError,
		)
		return
	}

	shared.RespondWith(c, http.StatusOK, gin.H{"block": block}, "", shared.ReturnCodeSuccess)
}

func (rb *rawBlockGroup) getRawMiniBlockByHash(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrValidationEmptyBlockHash.Error()),
		)
		return
	}

	start := time.Now()
	miniBlock, err := rb.getFacade().GetRawMiniBlockByHash(hash)
	log.Debug(fmt.Sprintf("GetRawMiniBlockByHash took %s", time.Since(start)))
	if err != nil {
		shared.RespondWith(
			c,
			http.StatusInternalServerError,
			nil,
			fmt.Sprintf("%s: %s", errors.ErrGetBlock.Error(), err.Error()),
			shared.ReturnCodeInternalError,
		)
		return
	}

	shared.RespondWith(c, http.StatusOK, gin.H{"block": miniBlock}, "", shared.ReturnCodeSuccess)
}

func (rb *rawBlockGroup) getInternalMiniBlockByHash(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrValidationEmptyBlockHash.Error()),
		)
		return
	}

	start := time.Now()
	miniBlock, err := rb.getFacade().GetInternalMiniBlockByHash(hash)
	log.Debug(fmt.Sprintf("GetInternalMiniBlockByHash took %s", time.Since(start)))
	if err != nil {
		shared.RespondWith(
			c,
			http.StatusInternalServerError,
			nil,
			fmt.Sprintf("%s: %s", errors.ErrGetBlock.Error(), err.Error()),
			shared.ReturnCodeInternalError,
		)
		return
	}

	shared.RespondWith(c, http.StatusOK, gin.H{"block": miniBlock}, "", shared.ReturnCodeSuccess)
}

func (rb *rawBlockGroup) getFacade() rawBlockFacadeHandler {
	rb.mutFacade.RLock()
	defer rb.mutFacade.RUnlock()

	return rb.facade
}

// UpdateFacade will update the facade
func (rb *rawBlockGroup) UpdateFacade(newFacade interface{}) error {
	if newFacade == nil {
		return errors.ErrNilFacadeHandler
	}
	castFacade, ok := newFacade.(rawBlockFacadeHandler)
	if !ok {
		return errors.ErrFacadeWrongTypeAssertion
	}

	rb.mutFacade.Lock()
	rb.facade = castFacade
	rb.mutFacade.Unlock()

	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (rb *rawBlockGroup) IsInterfaceNil() bool {
	return rb == nil
}
