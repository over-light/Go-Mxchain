package groups_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
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
		hg, err := groups.NewProofGroup(&mock.Facade{})
		require.NoError(t, err)
		require.NotNil(t, hg)
	})
}

func TestGetProof_GetProofError(t *testing.T) {
	t.Parallel()

	getProofErr := fmt.Errorf("GetProof error")
	facade := &mock.Facade{
		GetProofCalled: func(rootHash string, address string) ([][]byte, error) {
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
	facade := &mock.Facade{
		GetProofCalled: func(rootHash string, address string) ([][]byte, error) {
			assert.Equal(t, "roothash", rootHash)
			assert.Equal(t, "addr", address)
			return validProof, nil
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

func TestGetProofCurrentRootHash_GetProofError(t *testing.T) {
	t.Parallel()

	getProofErr := fmt.Errorf("GetProof error")
	facade := &mock.Facade{
		GetProofCurrentRootHashCalled: func(address string) ([][]byte, []byte, error) {
			return nil, nil, getProofErr
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
	rootHash := []byte("rootHash")
	facade := &mock.Facade{
		GetProofCurrentRootHashCalled: func(address string) ([][]byte, []byte, error) {
			assert.Equal(t, "addr", address)
			return validProof, rootHash, nil
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
	assert.Equal(t, hex.EncodeToString(rootHash), rh)
}

func TestVerifyProof_BadRequestShouldErr(t *testing.T) {
	t.Parallel()

	proofGroup, err := groups.NewProofGroup(&mock.Facade{})
	require.NoError(t, err)

	ws := startWebServer(proofGroup, "proof", getProofRoutesConfig())

	req, _ := http.NewRequest("POST", "/proof/verify", bytes.NewBuffer([]byte("invalid bytes")))

	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := shared.GenericAPIResponse{}
	loadResponse(resp.Body, &response)

	assert.Equal(t, shared.ReturnCodeRequestError, response.Code)
	assert.True(t, strings.Contains(response.Error, apiErrors.ErrValidation.Error()))
}

func TestVerifyProof_VerifyProofErr(t *testing.T) {
	t.Parallel()

	verifyProfErr := fmt.Errorf("VerifyProof err")
	facade := &mock.Facade{
		VerifyProofCalled: func(rootHash string, address string, proof [][]byte) (bool, error) {
			return false, verifyProfErr
		},
	}

	varifyProofParams := groups.VerifyProofRequest{
		RootHash: "rootHash",
		Address:  "address",
		Proof:    []string{hex.EncodeToString([]byte("valid")), hex.EncodeToString([]byte("proof"))},
	}
	verifyProofBytes, _ := json.Marshal(varifyProofParams)

	proofGroup, err := groups.NewProofGroup(facade)
	require.NoError(t, err)

	ws := startWebServer(proofGroup, "proof", getProofRoutesConfig())

	req, _ := http.NewRequest("POST", "/proof/verify", bytes.NewBuffer(verifyProofBytes))

	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := shared.GenericAPIResponse{}
	loadResponse(resp.Body, &response)

	assert.Equal(t, shared.ReturnCodeInternalError, response.Code)
	assert.True(t, strings.Contains(response.Error, apiErrors.ErrVerifyProof.Error()))
}

func TestVerifyProof_VerifyProofCanNotDecodeProof(t *testing.T) {
	t.Parallel()

	facade := &mock.Facade{}

	varifyProofParams := groups.VerifyProofRequest{
		RootHash: "rootHash",
		Address:  "address",
		Proof:    []string{"invalid", "hex"},
	}
	verifyProofBytes, _ := json.Marshal(varifyProofParams)

	proofGroup, err := groups.NewProofGroup(facade)
	require.NoError(t, err)

	ws := startWebServer(proofGroup, "proof", getProofRoutesConfig())

	req, _ := http.NewRequest("POST", "/proof/verify", bytes.NewBuffer(verifyProofBytes))

	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := shared.GenericAPIResponse{}
	loadResponse(resp.Body, &response)

	assert.Equal(t, shared.ReturnCodeRequestError, response.Code)
	assert.True(t, strings.Contains(response.Error, apiErrors.ErrValidation.Error()))
}

func TestVerifyProof(t *testing.T) {
	t.Parallel()

	rootHash := "rootHash"
	address := "address"
	validProof := []string{hex.EncodeToString([]byte("valid")), hex.EncodeToString([]byte("proof"))}
	varifyProofParams := groups.VerifyProofRequest{
		RootHash: rootHash,
		Address:  address,
		Proof:    validProof,
	}
	verifyProofBytes, _ := json.Marshal(varifyProofParams)

	facade := &mock.Facade{
		VerifyProofCalled: func(rH string, addr string, proof [][]byte) (bool, error) {
			assert.Equal(t, rootHash, rH)
			assert.Equal(t, address, addr)
			for i := range proof {
				assert.Equal(t, validProof[i], hex.EncodeToString(proof[i]))
			}

			return true, nil
		},
	}

	proofGroup, err := groups.NewProofGroup(facade)
	require.NoError(t, err)

	ws := startWebServer(proofGroup, "proof", getProofRoutesConfig())

	req, _ := http.NewRequest("POST", "/proof/verify", bytes.NewBuffer(verifyProofBytes))

	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := shared.GenericAPIResponse{}
	loadResponse(resp.Body, &response)

	assert.Equal(t, shared.ReturnCodeSuccess, response.Code)

	responseMap, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)

	isValid, ok := responseMap["ok"].(bool)
	assert.True(t, ok)
	assert.True(t, isValid)
}


func getProofRoutesConfig() config.ApiRoutesConfig {
	return config.ApiRoutesConfig{
		APIPackages: map[string]config.APIPackageConfig{
			"proof": {
				Routes: []config.RouteConfig{
					{Name: "/root-hash/:roothash/address/:address", Open: true},
					{Name: "/address/:address", Open: true},
					{Name: "/verify", Open: true},
				},
			},
		},
	}
}

