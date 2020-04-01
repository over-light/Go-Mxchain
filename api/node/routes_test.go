package node_test

import (
	"encoding/json"
	errs "errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/api/errors"
	"github.com/ElrondNetwork/elrond-go/api/mock"
	"github.com/ElrondNetwork/elrond-go/api/node"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/statistics"
	"github.com/ElrondNetwork/elrond-go/node/external"
	"github.com/ElrondNetwork/elrond-go/node/heartbeat"
	"github.com/ElrondNetwork/elrond-go/statusHandler"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type GeneralResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

type StatusResponse struct {
	GeneralResponse
	Running bool `json:"running"`
}

type StatisticsResponse struct {
	GeneralResponse
	Statistics struct {
		LiveTPS               float32 `json:"liveTPS"`
		PeakTPS               float32 `json:"peakTPS"`
		NrOfShards            uint32  `json:"nrOfShards"`
		BlockNumber           uint64  `json:"blockNumber"`
		RoundTime             uint32  `json:"roundTime"`
		AverageBlockTxCount   float32 `json:"averageBlockTxCount"`
		LastBlockTxCount      uint32  `json:"lastBlockTxCount"`
		TotalProcessedTxCount uint32  `json:"totalProcessedTxCount"`
	} `json:"statistics"`
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestStartNode_FailsWithoutFacade(t *testing.T) {
	t.Parallel()
	ws := startNodeServer(nil)
	defer func() {
		r := recover()
		assert.Nil(t, r, "Not providing elrondFacade context should panic")
	}()
	req, _ := http.NewRequest("GET", "/node/start", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)
}

//------- Heartbeatstatus

func TestHeartbeatStatus_FailsWithoutFacade(t *testing.T) {
	t.Parallel()

	ws := startNodeServer(nil)
	defer func() {
		r := recover()

		assert.NotNil(t, r, "Not providing elrondFacade context should panic")
	}()
	req, _ := http.NewRequest("GET", "/node/heartbeatstatus", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)
}

func TestHeartbeatstatus_FailsWithWrongFacadeTypeConversion(t *testing.T) {
	t.Parallel()

	facade := mock.Facade{}
	facade.Running = true
	ws := startNodeServerWrongFacade()
	req, _ := http.NewRequest("GET", "/node/heartbeatstatus", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	statusRsp := StatusResponse{}
	loadResponse(resp.Body, &statusRsp)

	assert.Equal(t, resp.Code, http.StatusInternalServerError)
	assert.Equal(t, statusRsp.Error, errors.ErrInvalidAppContext.Error())
}

func TestHeartbeatstatus_FromFacadeErrors(t *testing.T) {
	t.Parallel()

	errExpected := errs.New("expected error")
	facade := mock.Facade{
		GetHeartbeatsHandler: func() ([]heartbeat.PubKeyHeartbeat, error) {
			return nil, errExpected
		},
	}
	ws := startNodeServer(&facade)
	req, _ := http.NewRequest("GET", "/node/heartbeatstatus", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	statusRsp := StatusResponse{}
	loadResponse(resp.Body, &statusRsp)

	assert.Equal(t, resp.Code, http.StatusInternalServerError)
	assert.Equal(t, errExpected.Error(), statusRsp.Error)
}

func TestHeartbeatstatus(t *testing.T) {
	t.Parallel()

	hbStatus := []heartbeat.PubKeyHeartbeat{
		{
			HexPublicKey:    "pk1",
			TimeStamp:       time.Now(),
			MaxInactiveTime: heartbeat.Duration{Duration: 0},
			IsActive:        true,
			ReceivedShardID: uint32(0),
		},
	}
	facade := mock.Facade{
		GetHeartbeatsHandler: func() (heartbeats []heartbeat.PubKeyHeartbeat, e error) {
			return hbStatus, nil
		},
	}
	ws := startNodeServer(&facade)
	req, _ := http.NewRequest("GET", "/node/heartbeatstatus", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	statusRsp := StatusResponse{}
	loadResponseAsString(resp.Body, &statusRsp)

	assert.Equal(t, resp.Code, http.StatusOK)
	assert.NotEqual(t, "", statusRsp.Message)
}

func TestStatistics_FailsWithoutFacade(t *testing.T) {
	t.Parallel()
	ws := startNodeServer(nil)
	defer func() {
		r := recover()
		assert.NotNil(t, r, "Not providing elrondFacade context should panic")
	}()
	req, _ := http.NewRequest("GET", "/node/statistics", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)
}

func TestStatistics_FailsWithWrongFacadeTypeConversion(t *testing.T) {
	t.Parallel()
	ws := startNodeServerWrongFacade()
	req, _ := http.NewRequest("GET", "/node/statistics", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	statisticsRsp := StatisticsResponse{}
	loadResponse(resp.Body, &statisticsRsp)
	assert.Equal(t, resp.Code, http.StatusInternalServerError)
	assert.Equal(t, statisticsRsp.Error, errors.ErrInvalidAppContext.Error())
}

func TestStatistics_ReturnsSuccessfully(t *testing.T) {
	nrOfShards := uint32(10)
	roundTime := uint64(4)
	benchmark, _ := statistics.NewTPSBenchmark(nrOfShards, roundTime)

	facade := mock.Facade{}
	facade.TpsBenchmarkHandler = func() *statistics.TpsBenchmark {
		return benchmark
	}

	ws := startNodeServer(&facade)
	req, _ := http.NewRequest("GET", "/node/statistics", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	statisticsRsp := StatisticsResponse{}
	loadResponse(resp.Body, &statisticsRsp)
	assert.Equal(t, resp.Code, http.StatusOK)
	assert.Equal(t, statisticsRsp.Statistics.NrOfShards, nrOfShards)
}

func TestStatusMetrics_ShouldDisplayNonP2pMetrics(t *testing.T) {
	statusMetricsProvider := statusHandler.NewStatusMetrics()
	key := "test-details-key"
	value := "test-details-value"
	statusMetricsProvider.SetStringValue(key, value)

	p2pKey := "a_p2p_specific_key"
	statusMetricsProvider.SetStringValue(p2pKey, "p2p value")

	facade := mock.Facade{}
	facade.StatusMetricsHandler = func() external.StatusMetricsHandler {
		return statusMetricsProvider
	}

	ws := startNodeServer(&facade)
	req, _ := http.NewRequest("GET", "/node/status", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	respBytes, _ := ioutil.ReadAll(resp.Body)
	respStr := string(respBytes)
	assert.Equal(t, resp.Code, http.StatusOK)

	keyAndValueFoundInResponse := strings.Contains(respStr, key) && strings.Contains(respStr, value)
	assert.True(t, keyAndValueFoundInResponse)
	assert.False(t, strings.Contains(respStr, p2pKey))
}

func TestP2PStatusMetrics_ShouldDisplayNonP2pMetrics(t *testing.T) {
	statusMetricsProvider := statusHandler.NewStatusMetrics()
	key := "test-details-key"
	value := "test-details-value"
	statusMetricsProvider.SetStringValue(key, value)

	p2pKey := "a_p2p_specific_key"
	p2pValue := "p2p value"
	statusMetricsProvider.SetStringValue(p2pKey, p2pValue)

	facade := mock.Facade{}
	facade.StatusMetricsHandler = func() external.StatusMetricsHandler {
		return statusMetricsProvider
	}

	ws := startNodeServer(&facade)
	req, _ := http.NewRequest("GET", "/node/p2pstatus", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	respBytes, _ := ioutil.ReadAll(resp.Body)
	respStr := string(respBytes)
	assert.Equal(t, resp.Code, http.StatusOK)

	keyAndValueFoundInResponse := strings.Contains(respStr, p2pKey) && strings.Contains(respStr, p2pValue)
	assert.True(t, keyAndValueFoundInResponse)

	assert.False(t, strings.Contains(respStr, key))
}

func TestEpochMetrics_ShouldWork(t *testing.T) {
	statusMetricsProvider := statusHandler.NewStatusMetrics()
	key := core.MetricEpochNumber
	value := uint64(37)
	statusMetricsProvider.SetUInt64Value(key, value)

	facade := mock.Facade{}
	facade.StatusMetricsHandler = func() external.StatusMetricsHandler {
		return statusMetricsProvider
	}

	ws := startNodeServer(&facade)
	req, _ := http.NewRequest("GET", "/node/epoch", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	respBytes, _ := ioutil.ReadAll(resp.Body)
	respStr := string(respBytes)
	assert.Equal(t, resp.Code, http.StatusOK)

	keyAndValueFoundInResponse := strings.Contains(respStr, key) && strings.Contains(respStr, fmt.Sprintf("%d", value))
	assert.True(t, keyAndValueFoundInResponse)
}

func loadResponse(rsp io.Reader, destination interface{}) {
	jsonParser := json.NewDecoder(rsp)
	err := jsonParser.Decode(destination)
	if err != nil {
		logError(err)
	}
}

func loadResponseAsString(rsp io.Reader, response *StatusResponse) {
	buff, err := ioutil.ReadAll(rsp)
	if err != nil {
		logError(err)
		return
	}

	response.Message = string(buff)
}

func logError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func startNodeServer(handler node.FacadeHandler) *gin.Engine {
	server := startNodeServerWithFacade(handler)
	return server
}

func startNodeServerWrongFacade() *gin.Engine {
	return startNodeServerWithFacade(mock.WrongFacade{})
}

func startNodeServerWithFacade(facade interface{}) *gin.Engine {
	ws := gin.New()
	ws.Use(cors.Default())
	if facade != nil {
		ws.Use(func(c *gin.Context) {
			c.Set("elrondFacade", facade)
		})
	}

	nodeRoutes := ws.Group("/node")
	node.Routes(nodeRoutes)
	return ws
}
