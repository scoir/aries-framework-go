/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/aries-framework-go/pkg/controller/command"
	"github.com/hyperledger/aries-framework-go/pkg/controller/command/kms"
	"github.com/hyperledger/aries-framework-go/pkg/controller/rest"
	mockprovider "github.com/hyperledger/aries-framework-go/pkg/internal/mock/provider"
	mockkms "github.com/hyperledger/aries-framework-go/pkg/mock/kms"
	mocklegacykms "github.com/hyperledger/aries-framework-go/pkg/mock/kms/legacykms"
)

func TestNew(t *testing.T) {
	t.Run("test new command - success", func(t *testing.T) {
		cmd := New(&mockprovider.Provider{
			KMSValue: &mockkms.KeyManager{},
		})
		require.NotNil(t, cmd)
		require.Equal(t, 3, len(cmd.GetRESTHandlers()))
	})
}

func TestCreateKeySetLegacyKMS(t *testing.T) {
	t.Run("test create key set - success", func(t *testing.T) {
		cmd := New(&mockprovider.Provider{
			LegacyKMSValue: &mocklegacykms.CloseableKMS{CreateEncryptionKeyValue: "encryptionKey",
				CreateSigningKeyValue: "signingKey"},
		})
		require.NotNil(t, cmd)

		handler := lookupHandler(t, cmd, createKeySetLegacyKMSPath)
		err := getSuccessResponseFromHandler(handler, createKeySetLegacyKMSPath)
		require.NoError(t, err)
	})

	t.Run("test create key set - error", func(t *testing.T) {
		cmd := New(&mockprovider.Provider{
			LegacyKMSValue: &mocklegacykms.CloseableKMS{CreateKeyErr: fmt.Errorf("error create key set")},
		})
		require.NotNil(t, cmd)

		handler := lookupHandler(t, cmd, createKeySetLegacyKMSPath)
		buf, code, err := sendRequestToHandler(handler, nil, createKeySetLegacyKMSPath)
		require.NoError(t, err)
		require.NotEmpty(t, buf)

		require.Equal(t, http.StatusInternalServerError, code)
		verifyError(t, kms.CreateKeySetError, "error create key set", buf.Bytes())
	})
}

func TestCreateKeySet(t *testing.T) {
	t.Run("test create key set - success", func(t *testing.T) {
		cmd := New(&mockprovider.Provider{
			KMSValue: &mockkms.KeyManager{},
		})
		cmd.command = &mockKMSCommand{}

		handler := lookupHandler(t, cmd, createKeySetPath)
		err := getSuccessResponseFromHandler(handler, createKeySetPath)
		require.NoError(t, err)
	})

	t.Run("test create key set - error", func(t *testing.T) {
		cmd := New(&mockprovider.Provider{
			KMSValue: &mockkms.KeyManager{CreateKeyErr: fmt.Errorf("error create key set")},
		})
		require.NotNil(t, cmd)

		handler := lookupHandler(t, cmd, createKeySetPath)

		req := createKeySetReq{CreateKeySetRequest: kms.CreateKeySetRequest{
			KeyType: "ED25519",
		}}
		reqBytes, err := json.Marshal(req)
		require.NoError(t, err)

		buf, code, err := sendRequestToHandler(handler, bytes.NewBuffer(reqBytes), createKeySetPath)
		require.NoError(t, err)
		require.NotEmpty(t, buf)

		require.Equal(t, http.StatusInternalServerError, code)
		verifyError(t, kms.CreateKeySetError, "error create key set", buf.Bytes())
	})
}

func TestImportKey(t *testing.T) {
	t.Run("test import key - success", func(t *testing.T) {
		cmd := New(&mockprovider.Provider{})
		cmd.command = &mockKMSCommand{}

		handler := lookupHandler(t, cmd, importKeyPath)
		err := getSuccessResponseFromHandler(handler, importKeyPath)
		require.NoError(t, err)
	})

	t.Run("test import key - error", func(t *testing.T) {
		cmd := New(&mockprovider.Provider{})
		require.NotNil(t, cmd)

		cmd.command = &mockKMSCommand{importKeyError: command.NewExecuteError(kms.ImportKeyError,
			fmt.Errorf("failed to import key"))}

		handler := lookupHandler(t, cmd, importKeyPath)

		req := importKeyReq{JSONWebKey: kms.JSONWebKey{Kid: "k1"}}
		reqBytes, err := json.Marshal(req)
		require.NoError(t, err)

		buf, code, err := sendRequestToHandler(handler, bytes.NewBuffer(reqBytes), importKeyPath)
		require.NoError(t, err)
		require.NotEmpty(t, buf)

		require.Equal(t, http.StatusInternalServerError, code)
		verifyError(t, kms.ImportKeyError, "failed to import key", buf.Bytes())
	})
}

func lookupHandler(t *testing.T, op *Operation, path string) rest.Handler {
	handlers := op.GetRESTHandlers()
	require.NotEmpty(t, handlers)

	for _, h := range handlers {
		if h.Path() == path && h.Method() == http.MethodPost {
			return h
		}
	}

	require.Fail(t, "unable to find handler")

	return nil
}

// getSuccessResponseFromHandler reads response from given http handle func.
// expects http status OK.
func getSuccessResponseFromHandler(handler rest.Handler,
	path string) error {
	_, status, err := sendRequestToHandler(handler, nil, path)
	if status != http.StatusOK {
		return fmt.Errorf("unexpected status code: got %v, want %v",
			status, http.StatusOK)
	}

	return err
}

// sendRequestToHandler reads response from given http handle func.
func sendRequestToHandler(handler rest.Handler, requestBody io.Reader, path string) (*bytes.Buffer, int, error) {
	// prepare request
	req, err := http.NewRequest(handler.Method(), path, requestBody)
	if err != nil {
		return nil, 0, err
	}

	// prepare router
	router := mux.NewRouter()

	router.HandleFunc(handler.Path(), handler.Handle()).Methods(handler.Method())

	// create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	// serve http on given response and request
	router.ServeHTTP(rr, req)

	return rr.Body, rr.Code, nil
}

func verifyError(t *testing.T, expectedCode command.Code, expectedMsg string, data []byte) {
	// Parser generic error response
	errResponse := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{}
	err := json.Unmarshal(data, &errResponse)
	require.NoError(t, err)

	// verify response
	require.EqualValues(t, expectedCode, errResponse.Code)
	require.NotEmpty(t, errResponse.Message)

	if expectedMsg != "" {
		require.Contains(t, errResponse.Message, expectedMsg)
	}
}

type mockKMSCommand struct {
	importKeyError command.Error
}

func (m *mockKMSCommand) CreateKeySet(rw io.Writer, req io.Reader) command.Error {
	return nil
}

func (m *mockKMSCommand) CreateKeySetLegacyKMS(rw io.Writer, req io.Reader) command.Error {
	return nil
}

func (m *mockKMSCommand) ImportKey(rw io.Writer, req io.Reader) command.Error {
	return m.importKeyError
}
