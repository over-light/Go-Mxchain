package network_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go/api/errors"
	"github.com/ElrondNetwork/elrond-go/api/middleware"
	"github.com/ElrondNetwork/elrond-go/api/mock"
	"github.com/ElrondNetwork/elrond-go/api/network"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/node/external"
	"github.com/ElrondNetwork/elrond-go/statusHandler"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNetworkMetrics_ShouldWork(t *testing.T) {
	statusMetricsProvider := statusHandler.NewStatusMetrics()
	key := core.MetricEpochNumber
	value := uint64(37)
	statusMetricsProvider.SetUInt64Value(key, value)

	facade := mock.Facade{}
	facade.StatusMetricsHandler = func() external.StatusMetricsHandler {
		return statusMetricsProvider
	}

	ws := startNodeServer(&facade)
	req, _ := http.NewRequest("GET", "/network", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	respBytes, _ := ioutil.ReadAll(resp.Body)
	respStr := string(respBytes)
	assert.Equal(t, resp.Code, http.StatusOK)

	keyAndValueFoundInResponse := strings.Contains(respStr, key) && strings.Contains(respStr, fmt.Sprintf("%d", value))
	assert.True(t, keyAndValueFoundInResponse)
}

func TestNetwork_FailsWithWrongFacadeTypeConversion(t *testing.T) {
	t.Parallel()

	ws := startNodeServerWrongFacade()
	req, _ := http.NewRequest("GET", "/network", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	statusRsp := GeneralResponse{}
	loadResponse(resp.Body, &statusRsp)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, statusRsp.Error, errors.ErrInvalidAppContext.Error())
}

func loadResponse(rsp io.Reader, destination interface{}) {
	jsonParser := json.NewDecoder(rsp)
	err := jsonParser.Decode(destination)
	logError(err)
}

func logError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func startNodeServer(handler network.FacadeHandler) *gin.Engine {
	ws := gin.New()
	ws.Use(cors.Default())
	networkRoutes := ws.Group("/network")
	if handler != nil {
		networkRoutes.Use(middleware.WithElrondFacade(handler))
	}
	network.Routes(networkRoutes)
	return ws
}

func startNodeServerWrongFacade() *gin.Engine {
	ws := gin.New()
	ws.Use(cors.Default())
	ws.Use(func(c *gin.Context) {
		c.Set("elrondFacade", mock.WrongFacade{})
	})
	networkRoute := ws.Group("/network")
	network.Routes(networkRoute)
	return ws
}

type GeneralResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}
