package address_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go/api/address"
	errors2 "github.com/ElrondNetwork/elrond-go/api/errors"
	"github.com/ElrondNetwork/elrond-go/api/middleware"
	"github.com/ElrondNetwork/elrond-go/api/mock"
	"github.com/ElrondNetwork/elrond-go/api/wrapper"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type AccountResponse struct {
	Account struct {
		Address  string `json:"address"`
		Nonce    uint64 `json:"nonce"`
		Balance  string `json:"balance"`
		Code     string `json:"code"`
		CodeHash []byte `json:"codeHash"`
		RootHash []byte `json:"rootHash"`
	} `json:"account"`
}

func TestAddressRoute_EmptyTrailReturns404(t *testing.T) {
	t.Parallel()
	facade := mock.Facade{}
	ws := startNodeServer(&facade)

	req, _ := http.NewRequest("GET", "/address", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func getValueForKey(dataFromResponse interface{}, key string) string {
	dataMap, ok := dataFromResponse.(map[string]interface{})
	if !ok {
		return ""
	}

	if valueI, ok := dataMap[key]; ok {
		return fmt.Sprintf("%v", valueI)
	}
	return ""
}

func TestGetBalance_WithCorrectAddressShouldNotReturnError(t *testing.T) {
	t.Parallel()
	amount := big.NewInt(10)
	addr := "testAddress"
	facade := mock.Facade{
		BalanceHandler: func(s string) (i *big.Int, e error) {
			return amount, nil
		},
	}

	ws := startNodeServer(&facade)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/address/%s/balance", addr), nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := core.GenericAPIResponse{}
	loadResponse(resp.Body, &response)
	assert.Equal(t, http.StatusOK, resp.Code)

	balanceStr := getValueForKey(response.Data, "balance")
	balanceResponse, ok := big.NewInt(0).SetString(balanceStr, 10)
	assert.True(t, ok)
	assert.Equal(t, amount, balanceResponse)
	assert.Equal(t, "", response.Error)
}

func TestGetBalance_WithWrongAddressShouldError(t *testing.T) {
	t.Parallel()
	otherAddress := "otherAddress"
	facade := mock.Facade{
		BalanceHandler: func(s string) (i *big.Int, e error) {
			return big.NewInt(0), nil
		},
	}

	ws := startNodeServer(&facade)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/address/%s/balance", otherAddress), nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := core.GenericAPIResponse{}
	loadResponse(resp.Body, &response)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "", response.Error)
}

func TestGetBalance_NodeGetBalanceReturnsError(t *testing.T) {
	t.Parallel()
	addr := "addr"
	balanceError := errors.New("error")
	facade := mock.Facade{
		BalanceHandler: func(s string) (i *big.Int, e error) {
			return nil, balanceError
		},
	}

	ws := startNodeServer(&facade)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/address/%s/balance", addr), nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := core.GenericAPIResponse{}
	loadResponse(resp.Body, &response)
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, fmt.Sprintf("%s: %s", errors2.ErrGetBalance.Error(), balanceError.Error()), response.Error)
}

func TestGetBalance_WithEmptyAddressShoudReturnError(t *testing.T) {
	t.Parallel()
	facade := mock.Facade{
		BalanceHandler: func(s string) (i *big.Int, e error) {
			return big.NewInt(0), errors.New("address was empty")
		},
	}

	emptyAddress := ""
	ws := startNodeServer(&facade)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/address/%s/balance", emptyAddress), nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := core.GenericAPIResponse{}
	loadResponse(resp.Body, &response)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.NotEmpty(t, response)
	assert.True(t, strings.Contains(response.Error,
		fmt.Sprintf("%s: %s", errors2.ErrGetBalance.Error(), errors2.ErrEmptyAddress.Error()),
	))
}

func TestGetBalance_FailsWithWrongFacadeTypeConversion(t *testing.T) {
	t.Parallel()

	ws := startNodeServerWrongFacade()
	req, _ := http.NewRequest("GET", "/address/empty/balance", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := core.GenericAPIResponse{}
	loadResponse(resp.Body, &response)
	assert.Equal(t, resp.Code, http.StatusInternalServerError)
	assert.Equal(t, response.Error, errors2.ErrInvalidAppContext.Error())
}

func TestGetAccount_FailsWithWrongFacadeTypeConversion(t *testing.T) {
	t.Parallel()

	ws := startNodeServerWrongFacade()
	req, _ := http.NewRequest("GET", "/address/empty", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := core.GenericAPIResponse{}
	loadResponse(resp.Body, &response)
	assert.Equal(t, resp.Code, http.StatusInternalServerError)
	assert.Equal(t, response.Error, errors2.ErrInvalidAppContext.Error())
}

func TestGetAccount_FailWhenFacadeGetAccountFails(t *testing.T) {
	t.Parallel()
	returnedError := "i am an error"
	facade := mock.Facade{
		GetAccountHandler: func(address string) (state.UserAccountHandler, error) {
			return nil, errors.New(returnedError)
		},
	}
	ws := startNodeServer(&facade)

	req, _ := http.NewRequest("GET", "/address/test", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := core.GenericAPIResponse{}
	loadResponse(resp.Body, &response)
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Empty(t, response.Data)
	assert.NotEmpty(t, response.Error)
	assert.True(t, strings.Contains(response.Error, fmt.Sprintf("%s: %s", errors2.ErrCouldNotGetAccount.Error(), returnedError)))
}

func TestGetAccount_ReturnsSuccessfully(t *testing.T) {
	t.Parallel()
	facade := mock.Facade{
		GetAccountHandler: func(address string) (state.UserAccountHandler, error) {
			acc, _ := state.NewUserAccount([]byte("1234"))
			_ = acc.AddToBalance(big.NewInt(100))
			acc.IncreaseNonce(1)

			return acc, nil
		},
	}
	ws := startNodeServer(&facade)

	reqAddress := "test"
	req, _ := http.NewRequest("GET", fmt.Sprintf("/address/%s", reqAddress), nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := core.GenericAPIResponse{}
	loadResponse(resp.Body, &response)
	mapResponse := response.Data.(map[string]interface{})
	accountResponse := AccountResponse{}

	mapResponseBytes, _ := json.Marshal(&mapResponse)
	_ = json.Unmarshal(mapResponseBytes, &accountResponse)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, accountResponse.Account.Address, reqAddress)
	assert.Equal(t, accountResponse.Account.Nonce, uint64(1))
	assert.Equal(t, accountResponse.Account.Balance, "100")
	assert.Empty(t, response.Error)
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

func startNodeServer(handler address.FacadeHandler) *gin.Engine {
	ws := gin.New()
	ws.Use(cors.Default())
	addressRoutes := ws.Group("/address")
	if handler != nil {
		addressRoutes.Use(middleware.WithElrondFacade(handler))
	}
	addressRoute, _ := wrapper.NewRouterWrapper("address", addressRoutes, getRoutesConfig())
	address.Routes(addressRoute)
	return ws
}

func startNodeServerWrongFacade() *gin.Engine {
	ws := gin.New()
	ws.Use(cors.Default())
	ws.Use(func(c *gin.Context) {
		c.Set("elrondFacade", mock.WrongFacade{})
	})
	ginAddressRoute := ws.Group("/address")
	addressRoute, _ := wrapper.NewRouterWrapper("address", ginAddressRoute, getRoutesConfig())
	address.Routes(addressRoute)
	return ws
}

func getRoutesConfig() config.ApiRoutesConfig {
	return config.ApiRoutesConfig{
		APIPackages: map[string]config.APIPackageConfig{
			"address": {
				[]config.RouteConfig{
					{Name: "/:address", Open: true},
					{Name: "/:address/balance", Open: true},
				},
			},
		},
	}
}
