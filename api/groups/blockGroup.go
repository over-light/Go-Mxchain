package groups

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/api"
	"github.com/ElrondNetwork/elrond-go/api/errors"
	"github.com/ElrondNetwork/elrond-go/api/shared"
	"github.com/gin-gonic/gin"
)

const (
	getBlockByNoncePath = "/by-nonce/:nonce"
	getBlockByHashPath  = "/by-hash/:hash"
	getBlockByRoundPath = "/by-round/:round"
)

// blockFacadeHandler defines the methods to be implemented by a facade for handling block requests
type blockFacadeHandler interface {
	GetBlockByHash(hash string, options api.BlockQueryOptions) (*api.Block, error)
	GetBlockByNonce(nonce uint64, options api.BlockQueryOptions) (*api.Block, error)
	GetBlockByRound(round uint64, options api.BlockQueryOptions) (*api.Block, error)
	IsInterfaceNil() bool
}

type blockGroup struct {
	*baseGroup
	facade    blockFacadeHandler
	mutFacade sync.RWMutex
}

// NewBlockGroup returns a new instance of blockGroup
func NewBlockGroup(facade blockFacadeHandler) (*blockGroup, error) {
	if check.IfNil(facade) {
		return nil, fmt.Errorf("%w for block group", errors.ErrNilFacadeHandler)
	}

	bg := &blockGroup{
		facade:    facade,
		baseGroup: &baseGroup{},
	}

	endpoints := []*shared.EndpointHandlerData{
		{
			Path:    getBlockByNoncePath,
			Method:  http.MethodGet,
			Handler: bg.getBlockByNonce,
		},
		{
			Path:    getBlockByHashPath,
			Method:  http.MethodGet,
			Handler: bg.getBlockByHash,
		},
		{
			Path:    getBlockByRoundPath,
			Method:  http.MethodGet,
			Handler: bg.getBlockByRound,
		},
	}
	bg.endpoints = endpoints

	return bg, nil
}

func (bg *blockGroup) getBlockByNonce(c *gin.Context) {
	nonce, err := getQueryParamNonce(c)
	if err != nil {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrInvalidBlockNonce.Error()),
		)
		return
	}

	options, err := parseBlockQueryOptions(c)
	if err != nil {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrInvalidQueryParameter.Error()),
		)
		return
	}

	start := time.Now()
	block, err := bg.getFacade().GetBlockByNonce(nonce, options)
	log.Debug("API call: GetBlockByNonce", "duration", time.Since(start))
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

func (bg *blockGroup) getBlockByHash(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrValidationEmptyBlockHash.Error()),
		)
		return
	}

	options, err := parseBlockQueryOptions(c)
	if err != nil {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrInvalidBlockNonce.Error()),
		)
		return
	}

	start := time.Now()
	block, err := bg.getFacade().GetBlockByHash(hash, options)
	log.Debug("API call: GetBlockByHash", "duration", time.Since(start))
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

func (bg *blockGroup) getBlockByRound(c *gin.Context) {
	round, err := getQueryParamRound(c)
	if err != nil {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrInvalidBlockRound.Error()),
		)
		return
	}

	options, err := parseBlockQueryOptions(c)
	if err != nil {
		shared.RespondWithValidationError(
			c, fmt.Sprintf("%s: %s", errors.ErrValidation.Error(), errors.ErrInvalidQueryParameter.Error()),
		)
		return
	}

	start := time.Now()
	block, err := bg.getFacade().GetBlockByRound(round, options)
	log.Debug("API call: GetBlockByRound", "duration", time.Since(start))
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

func parseBlockQueryOptions(c *gin.Context) (api.BlockQueryOptions, error) {
	withTxs, err := parseBoolUrlParam(c, "withTxs")
	if err != nil {
		return api.BlockQueryOptions{}, err
	}

	withLogs, err := parseBoolUrlParam(c, "withLogs")
	if err != nil {
		return api.BlockQueryOptions{}, err
	}

	options := api.BlockQueryOptions{WithTransactions: withTxs, WithLogs: withLogs}
	return options, nil
}

func parseBoolUrlParam(c *gin.Context, name string) (bool, error) {
	param := c.Request.URL.Query().Get(name)
	if param == "" {
		return false, nil
	}

	return strconv.ParseBool(param)
}

func getQueryParamNonce(c *gin.Context) (uint64, error) {
	nonceStr := c.Param("nonce")
	return strconv.ParseUint(nonceStr, 10, 64)
}

func getQueryParamRound(c *gin.Context) (uint64, error) {
	roundStr := c.Param("round")
	return strconv.ParseUint(roundStr, 10, 64)
}

func (bg *blockGroup) getFacade() blockFacadeHandler {
	bg.mutFacade.RLock()
	defer bg.mutFacade.RUnlock()

	return bg.facade
}

// UpdateFacade will update the facade
func (bg *blockGroup) UpdateFacade(newFacade interface{}) error {
	if newFacade == nil {
		return errors.ErrNilFacadeHandler
	}
	castFacade, ok := newFacade.(blockFacadeHandler)
	if !ok {
		return errors.ErrFacadeWrongTypeAssertion
	}

	bg.mutFacade.Lock()
	bg.facade = castFacade
	bg.mutFacade.Unlock()

	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (bg *blockGroup) IsInterfaceNil() bool {
	return bg == nil
}
