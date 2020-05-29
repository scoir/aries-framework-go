/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package verifiable

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/hyperledger/aries-framework-go/pkg/controller/command/verifiable"
	"github.com/hyperledger/aries-framework-go/pkg/controller/internal/cmdutil"
	"github.com/hyperledger/aries-framework-go/pkg/controller/rest"
	ariescrypto "github.com/hyperledger/aries-framework-go/pkg/crypto"
	"github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdri"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/pkg/storage"
)

const (
	// roots
	verifiableOperationID      = "/verifiable"
	verifiableCredentialPath   = verifiableOperationID + "/credential"
	verifiablePresentationPath = verifiableOperationID + "/presentation"

	// credential paths
	validateCredentialPath  = verifiableCredentialPath + "/validate"
	saveCredentialPath      = verifiableCredentialPath
	getCredentialPath       = verifiableCredentialPath + "/{id}"
	getCredentialByNamePath = verifiableCredentialPath + "/name" + "/{name}"
	getCredentialsPath      = verifiableOperationID + "/credentials"

	// presentation paths
	generatePresentationPath     = verifiablePresentationPath + "/generate"
	generatePresentationByIDPath = verifiablePresentationPath + "/generatebyid"
	savePresentationPath         = verifiablePresentationPath
	getPresentationPath          = verifiablePresentationPath + "/{id}"
	getPresentationsPath         = verifiableOperationID + "/presentations"
)

// provider contains dependencies for the verifiable command and is typically created by using aries.Context().
type provider interface {
	StorageProvider() storage.Provider
	VDRIRegistry() vdri.Registry
	KMS() kms.KeyManager
	Crypto() ariescrypto.Crypto
}

// Operation contains basic common operations provided by controller REST API
type Operation struct {
	handlers []rest.Handler
	command  *verifiable.Command
}

// New returns new common operations rest client instance
func New(p provider) (*Operation, error) {
	cmd, err := verifiable.New(p)
	if err != nil {
		return nil, fmt.Errorf("verfiable new: %w", err)
	}

	o := &Operation{command: cmd}
	o.registerHandler()

	return o, nil
}

// GetRESTHandlers get all controller API handler available for this service
func (o *Operation) GetRESTHandlers() []rest.Handler {
	return o.handlers
}

// registerHandler register handlers to be exposed from this protocol service as REST API endpoints
func (o *Operation) registerHandler() {
	o.handlers = []rest.Handler{
		cmdutil.NewHTTPHandler(validateCredentialPath, http.MethodPost, o.ValidateCredential),
		cmdutil.NewHTTPHandler(saveCredentialPath, http.MethodPost, o.SaveCredential),
		cmdutil.NewHTTPHandler(getCredentialPath, http.MethodGet, o.GetCredential),
		cmdutil.NewHTTPHandler(getCredentialByNamePath, http.MethodGet, o.GetCredentialByName),
		cmdutil.NewHTTPHandler(getCredentialsPath, http.MethodGet, o.GetCredentials),
		cmdutil.NewHTTPHandler(generatePresentationPath, http.MethodPost, o.GeneratePresentation),
		cmdutil.NewHTTPHandler(generatePresentationByIDPath, http.MethodPost, o.GeneratePresentationByID),
		cmdutil.NewHTTPHandler(savePresentationPath, http.MethodPost, o.SavePresentation),
		cmdutil.NewHTTPHandler(getPresentationPath, http.MethodGet, o.GetPresentation),
		cmdutil.NewHTTPHandler(getPresentationsPath, http.MethodGet, o.GetPresentations),
	}
}

// ValidateCredential swagger:route POST /verifiable/credential/validate verifiable validateCredentialReq
//
// Validates the verifiable credential.
//
// Responses:
//    default: genericError
//        200: emptyRes
func (o *Operation) ValidateCredential(rw http.ResponseWriter, req *http.Request) {
	rest.Execute(o.command.ValidateCredential, rw, req.Body)
}

// SaveCredential swagger:route POST /verifiable/credential verifiable saveCredentialReq
//
// Saves the verifiable credential.
//
// Responses:
//    default: genericError
//        200: emptyRes
func (o *Operation) SaveCredential(rw http.ResponseWriter, req *http.Request) {
	rest.Execute(o.command.SaveCredential, rw, req.Body)
}

// SavePresentation swagger:route POST /verifiable/presentation verifiable savePresentationReq
//
// Saves the verifiable presentation.
//
// Responses:
//    default: genericError
//        200: emptyRes
func (o *Operation) SavePresentation(rw http.ResponseWriter, req *http.Request) {
	rest.Execute(o.command.SavePresentation, rw, req.Body)
}

// GetCredential swagger:route GET /verifiable/credential/{id} verifiable getCredentialReq
//
// Retrieves the verifiable credential.
//
// Responses:
//    default: genericError
//        200: credentialRes
func (o *Operation) GetCredential(rw http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]

	decodedID, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		rest.SendHTTPStatusError(rw, http.StatusBadRequest, verifiable.InvalidRequestErrorCode, err)
		return
	}

	request := fmt.Sprintf(`{"id":"%s"}`, string(decodedID))

	rest.Execute(o.command.GetCredential, rw, bytes.NewBufferString(request))
}

// GetPresentation swagger:route GET /verifiable/presentation/{id} verifiable getPresentationReq
//
// Retrieves the verifiable presentation.
//
// Responses:
//    default: genericError
//        200: presentationRes
func (o *Operation) GetPresentation(rw http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]

	decodedID, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		rest.SendHTTPStatusError(rw, http.StatusBadRequest, verifiable.InvalidRequestErrorCode, err)
		return
	}

	request := fmt.Sprintf(`{"id":"%s"}`, string(decodedID))

	rest.Execute(o.command.GetPresentation, rw, bytes.NewBufferString(request))
}

// GetCredentialByName swagger:route GET /verifiable/credential/name/{name} verifiable getCredentialByNameReq
//
// Retrieves the verifiable credential by name.
//
// Responses:
//    default: genericError
//        200: credentialRecord
func (o *Operation) GetCredentialByName(rw http.ResponseWriter, req *http.Request) {
	name := mux.Vars(req)["name"]

	request := fmt.Sprintf(`{"name":"%s"}`, name)

	rest.Execute(o.command.GetCredentialByName, rw, bytes.NewBufferString(request))
}

// GetCredentials swagger:route GET /verifiable/credentials verifiable getCredentials
//
// Retrieves the verifiable credentials.
//
// Responses:
//    default: genericError
//        200: credentialRecordResult
func (o *Operation) GetCredentials(rw http.ResponseWriter, req *http.Request) {
	rest.Execute(o.command.GetCredentials, rw, req.Body)
}

// GetPresentations swagger:route GET /verifiable/presentations verifiable
//
// Retrieves the verifiable credentials.
//
// Responses:
//    default: genericError
//        200: presentationRecordResult
func (o *Operation) GetPresentations(rw http.ResponseWriter, req *http.Request) {
	rest.Execute(o.command.GetPresentations, rw, req.Body)
}

// GeneratePresentation swagger:route POST /verifiable/presentation/generate verifiable generatePresentationReq
//
// Generates the verifiable presentation from a verifiable credential.
//
// Responses:
//    default: genericError
//        200: presentationRes
func (o *Operation) GeneratePresentation(rw http.ResponseWriter, req *http.Request) {
	rest.Execute(o.command.GeneratePresentation, rw, req.Body)
}

// GeneratePresentationByID swagger:route POST /verifiable/presentation/generatebyid
// verifiable PresentationRequestByID model
//
// Generates the verifiable presentation from a stored verifiable credential.
//
// Responses:
//    default: genericError
//        200: presentationRes
func (o *Operation) GeneratePresentationByID(rw http.ResponseWriter, req *http.Request) {
	rest.Execute(o.command.GeneratePresentationByID, rw, req.Body)
}
