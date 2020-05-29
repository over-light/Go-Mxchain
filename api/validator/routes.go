package validator

import (
	"net/http"

	"github.com/ElrondNetwork/elrond-go/api/errors"
	"github.com/ElrondNetwork/elrond-go/api/wrapper"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/gin-gonic/gin"
)

// ValidatorsStatisticsApiHandler interface defines methods that can be used from `elrondFacade` context variable
type ValidatorsStatisticsApiHandler interface {
	ValidatorStatisticsApi() (map[string]*state.ValidatorApiResponse, error)
	IsInterfaceNil() bool
}

// Routes defines validators' related routes
func Routes(router *wrapper.RouterWrapper) {
	router.RegisterHandler(http.MethodGet, "/statistics", Statistics)
}

// Statistics will return the validation statistics for all validators
func Statistics(c *gin.Context) {
	ef, ok := c.MustGet("elrondFacade").(ValidatorsStatisticsApiHandler)
	if !ok {
		c.JSON(
			http.StatusInternalServerError,
			core.GenericAPIResponse{
				Data:  nil,
				Error: errors.ErrInvalidAppContext.Error(),
				Code:  string(core.ReturnCodeInternalError),
			},
		)
		return
	}

	valStats, err := ef.ValidatorStatisticsApi()
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			core.GenericAPIResponse{
				Data:  nil,
				Error: err.Error(),
				Code:  string(core.ReturnCodeRequestErrror),
			},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		core.GenericAPIResponse{
			Data:  gin.H{"statistics": valStats},
			Error: "",
			Code:  string(core.ReturnCodeSuccess),
		},
	)
}
