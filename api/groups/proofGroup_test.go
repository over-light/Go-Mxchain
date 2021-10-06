package groups_test

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiErrors "github.com/ElrondNetwork/elrond-go/api/errors"
	"github.com/ElrondNetwork/elrond-go/api/groups"
	"github.com/ElrondNetwork/elrond-go/api/mock"
	"github.com/ElrondNetwork/elrond-go/api/shared"
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProofGroup(t *testing.T) {
	t.Parallel()

	t.Run("nil facade", func(t *testing.T) {
		hg, err := groups.NewProofGroup(nil)
		require.True(t, errors.Is(err, apiErrors.ErrNilFacadeHandler))
		require.Nil(t, hg)
	})

	t.Run("should work", func(t *testing.T) {
		hg, err := groups.NewProofGroup(&mock.FacadeStub{})
		require.NoError(t, err)
		require.NotNil(t, hg)
	})
}

func TestGetProof_GetProofError(t *testing.T) {
	t.Parallel()

	getProofErr := fmt.Errorf("GetProof error")
	facade := &mock.FacadeStub{
		GetProofCalled: func(rootHash string, address string) (*common.GetProofResponse, error) {
			return nil, getProofErr
		},
	}

	proofGroup, err := groups.NewProofGroup(facade)
	require.NoError(t, err)

	ws := startWebServer(proofGroup, "proof", getProofRoutesConfig())

	req, _ := http.NewRequest("GET", "/proof/root-hash/roothash/address/addr", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)
	response := shared.GenericAPIResponse{}
	loadResponse(resp.Body, &response)

	assert.Equal(t, shared.ReturnCodeInternalError, response.Code)
	assert.True(t, strings.Contains(response.Error, apiErrors.ErrGetProof.Error()))
}

func TestGetProof(t *testing.T) {
	t.Parallel()

	validProof := [][]byte{[]byte("valid"), []byte("proof")}
	facade := &mock.FacadeStub{
		GetProofCalled: func(rootHash string, address string) (*common.GetProofResponse, error) {
			assert.Equal(t, "roothash", rootHash)
			assert.Equal(t, "addr", address)
			return &common.GetProofResponse{
				Proof: validProof,
			}, nil
		},
	}

	proofGroup, err := groups.NewProofGroup(facade)
	require.NoError(t, err)

	ws := startWebServer(proofGroup, "proof", getProofRoutesConfig())

	req, _ := http.NewRequest("GET", "/proof/root-hash/roothash/address/addr", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)
	response := shared.GenericAPIResponse{}
	loadResponse(resp.Body, &response)

	assert.Equal(t, shared.ReturnCodeSuccess, response.Code)

	responseMap, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)

	proofs, ok := responseMap["proof"].([]interface{})
	assert.True(t, ok)

	proof1 := proofs[0].(string)
	proof2 := proofs[1].(string)

	assert.Equal(t, hex.EncodeToString([]byte("valid")), proof1)
	assert.Equal(t, hex.EncodeToString([]byte("proof")), proof2)
}

func TestGetProofDataTrie_GetProofError(t *testing.T) {
	t.Parallel()

	getProofDataTrieErr := fmt.Errorf("GetProofDataTrie error")
	facade := &mock.FacadeStub{
		GetProofDataTrieCalled: func(rootHash string, address string, key string) (*common.GetProofResponse, *common.GetProofResponse, error) {
			return nil, nil, getProofDataTrieErr
		},
	}

	proofGroup, err := groups.NewProofGroup(facade)
	require.NoError(t, err)

	ws := startWebServer(proofGroup, "proof", getProofRoutesConfig())
	req, _ := http.NewRequest("GET", "/proof/root-hash/roothash/address/addr/key/key", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)
	response := shared.GenericAPIResponse{}
	loadResponse(resp.Body, &response)

	assert.Equal(t, shared.ReturnCodeInternalError, response.Code)
	assert.True(t, strings.Contains(response.Error, apiErrors.ErrGetProof.Error()))
}

func TestGetProofDataTrie(t *testing.T) {
	t.Parallel()

	mainTrieProof := [][]byte{[]byte("main"), []byte("proof")}
	dataTrieProof := [][]byte{[]byte("data"), []byte("proof")}
	facade := &mock.FacadeStub{
		GetProofDataTrieCalled: func(rootHash string, address string, key string) (*common.GetProofResponse, *common.GetProofResponse, error) {
			assert.Equal(t, "roothash", rootHash)
			assert.Equal(t, "addr", address)
			return &common.GetProofResponse{Proof: mainTrieProof},
				&common.GetProofResponse{Proof: dataTrieProof},
				nil
		},
	}

	proofGroup, err := groups.NewProofGroup(facade)
	require.NoError(t, err)

	ws := startWebServer(proofGroup, "proof", getProofRoutesConfig())
	req, _ := http.NewRequest("GET", "/proof/root-hash/roothash/address/addr/key/key", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)
	response := shared.GenericAPIResponse{}
	loadResponse(resp.Body, &response)

	assert.Equal(t, shared.ReturnCodeSuccess, response.Code)

	responseMap, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)

	proofsResponseMap, ok := responseMap["proofs"].(map[string]interface{})
	assert.True(t, ok)

	proofs, ok := proofsResponseMap["mainProof"].([]interface{})
	assert.True(t, ok)

	proof1 := proofs[0].(string)
	proof2 := proofs[1].(string)

	assert.Equal(t, hex.EncodeToString([]byte("main")), proof1)
	assert.Equal(t, hex.EncodeToString([]byte("proof")), proof2)

	proofs, ok = proofsResponseMap["dataTrieProof"].([]interface{})
	assert.True(t, ok)

	proof1 = proofs[0].(string)
	proof2 = proofs[1].(string)

	assert.Equal(t, hex.EncodeToString([]byte("data")), proof1)
	assert.Equal(t, hex.EncodeToString([]byte("proof")), proof2)
}

func TestGetProofCurrentRootHash_GetProofError(t *testing.T) {
	t.Parallel()

	getProofErr := fmt.Errorf("GetProof error")
	facade := &mock.FacadeStub{
		GetProofCurrentRootHashCalled: func(address string) (*common.GetProofResponse, error) {
			return nil, getProofErr
		},
	}

	proofGroup, err := groups.NewProofGroup(facade)
	require.NoError(t, err)

	ws := startWebServer(proofGroup, "proof", getProofRoutesConfig())

	req, _ := http.NewRequest("GET", "/proof/address/addr", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)
	response := shared.GenericAPIResponse{}
	loadResponse(resp.Body, &response)

	assert.Equal(t, shared.ReturnCodeInternalError, response.Code)
	assert.True(t, strings.Contains(response.Error, apiErrors.ErrGetProof.Error()))
}

func TestGetProofCurrentRootHash(t *testing.T) {
	t.Parallel()

	validProof := [][]byte{[]byte("valid"), []byte("proof")}
	rootHash := "rootHash"
	facade := &mock.FacadeStub{
		GetProofCurrentRootHashCalled: func(address string) (*common.GetProofResponse, error) {
			assert.Equal(t, "addr", address)
			return &common.GetProofResponse{
				Proof:    validProof,
				RootHash: rootHash,
			}, nil
		},
	}

	proofGroup, err := groups.NewProofGroup(facade)
	require.NoError(t, err)

	ws := startWebServer(proofGroup, "proof", getProofRoutesConfig())

	req, _ := http.NewRequest("GET", "/proof/address/addr", nil)
	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)
	response := shared.GenericAPIResponse{}
	loadResponse(resp.Body, &response)

	assert.Equal(t, shared.ReturnCodeSuccess, response.Code)

	responseMap, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)

	proofs, ok := responseMap["proof"].([]interface{})
	assert.True(t, ok)

	proof1 := proofs[0].(string)
	proof2 := proofs[1].(string)

	assert.Equal(t, hex.EncodeToString([]byte("valid")), proof1)
	assert.Equal(t, hex.EncodeToString([]byte("proof")), proof2)

	rh, ok := responseMap["rootHash"].(string)
	assert.True(t, ok)
	assert.Equal(t, rootHash, rh)
}

func getProofRoutesConfig() config.ApiRoutesConfig {
	return config.ApiRoutesConfig{
		APIPackages: map[string]config.APIPackageConfig{
			"proof": {
				Routes: []config.RouteConfig{
					{Name: "/root-hash/:roothash/address/:address", Open: true},
					{Name: "/root-hash/:roothash/address/:address/key/:key", Open: true},
					{Name: "/address/:address", Open: true},
				},
			},
		},
	}
}
