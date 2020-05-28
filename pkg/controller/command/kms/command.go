/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kms

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/hyperledger/aries-framework-go/pkg/common/log"
	"github.com/hyperledger/aries-framework-go/pkg/controller/command"
	"github.com/hyperledger/aries-framework-go/pkg/controller/internal/cmdutil"
	"github.com/hyperledger/aries-framework-go/pkg/doc/jose"
	"github.com/hyperledger/aries-framework-go/pkg/internal/logutil"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"github.com/hyperledger/aries-framework-go/pkg/kms/legacykms"
)

var logger = log.New("aries-framework/command/kms")

// Error codes
const (
	// InvalidRequestErrorCode is typically a code for invalid requests
	InvalidRequestErrorCode = command.Code(iota + command.KMS)
	// CreateKeySetError is for failures while creating key set
	CreateKeySetError
	// ImportKeyError is for failures while importing key
	ImportKeyError
)

const (
	// command name
	commandName = "kms"

	// command name
	legacyKMSCommandName = "legacykms"

	// command methods
	createKeySetCommandMethod = "CreateKeySet"
	importKeyCommandMethod    = "ImportKey"

	// error messages
	errEmptyKeyType = "key type is mandatory"
	errEmptyKeyID   = "key id is mandatory"
)

// provider contains dependencies for the kms command and is typically created by using aries.Context().
type provider interface {
	KMS() kms.KeyManager
	LegacyKMS() legacykms.KeyManager
}

// Command contains command operations provided by verifiable credential controller.
type Command struct {
	ctx               provider
	exportPubKeyBytes func(id string) ([]byte, error) // needed for unit test
	importKey         func(privKey interface{}, kt kms.KeyType,
		opts ...kms.PrivateKeyOpts) (string, interface{}, error) // needed for unit test
}

// New returns new kms command instance.
func New(p provider) *Command {
	return &Command{
		ctx: p,
		exportPubKeyBytes: func(id string) ([]byte, error) {
			return p.KMS().ExportPubKeyBytes(id)
		},
		importKey: func(privKey interface{}, kt kms.KeyType,
			opts ...kms.PrivateKeyOpts) (string, interface{}, error) {
			return p.KMS().ImportPrivateKey(privKey, kt, opts...)
		},
	}
}

// GetHandlers returns list of all commands supported by this controller command.
func (o *Command) GetHandlers() []command.Handler {
	return []command.Handler{
		cmdutil.NewCommandHandler(commandName, createKeySetCommandMethod, o.CreateKeySet),
		cmdutil.NewCommandHandler(commandName, importKeyCommandMethod, o.ImportKey),
		cmdutil.NewCommandHandler(legacyKMSCommandName, createKeySetCommandMethod, o.CreateKeySetLegacyKMS),
	}
}

// CreateKeySetLegacyKMS create a new public/private encryption and signature key pairs set.
// TODO Remove it after switching packer to use new kms https://github.com/hyperledger/aries-framework-go/issues/1828
func (o *Command) CreateKeySetLegacyKMS(rw io.Writer, req io.Reader) command.Error {
	_, signaturePublicKey, err := o.ctx.LegacyKMS().CreateKeySet()
	if err != nil {
		logutil.LogError(logger, legacyKMSCommandName, createKeySetCommandMethod, err.Error())
		return command.NewExecuteError(CreateKeySetError, err)
	}

	command.WriteNillableResponse(rw, &CreateKeySetResponse{
		PublicKey: signaturePublicKey,
	}, logger)

	logutil.LogDebug(logger, legacyKMSCommandName, createKeySetCommandMethod, "success")

	return nil
}

// CreateKeySet create a new public/private encryption and signature key pairs set.
func (o *Command) CreateKeySet(rw io.Writer, req io.Reader) command.Error {
	var request CreateKeySetRequest

	err := json.NewDecoder(req).Decode(&request)
	if err != nil {
		logutil.LogInfo(logger, commandName, createKeySetCommandMethod, err.Error())
		return command.NewValidationError(InvalidRequestErrorCode, fmt.Errorf("failed request decode : %w", err))
	}

	if request.KeyType == "" {
		logutil.LogDebug(logger, commandName, createKeySetCommandMethod, errEmptyKeyType)
		return command.NewValidationError(InvalidRequestErrorCode, fmt.Errorf(errEmptyKeyType))
	}

	keyID, _, err := o.ctx.KMS().Create(kms.KeyType(request.KeyType))
	if err != nil {
		logutil.LogError(logger, commandName, createKeySetCommandMethod, err.Error())
		return command.NewExecuteError(CreateKeySetError, err)
	}

	pubKeyBytes, err := o.exportPubKeyBytes(keyID)
	if err != nil {
		logutil.LogError(logger, commandName, createKeySetCommandMethod, err.Error())
		return command.NewExecuteError(CreateKeySetError, err)
	}

	command.WriteNillableResponse(rw, &CreateKeySetResponse{
		KeyID:     keyID,
		PublicKey: base64.RawURLEncoding.EncodeToString(pubKeyBytes),
	}, logger)

	logutil.LogDebug(logger, commandName, createKeySetCommandMethod, "success")

	return nil
}

// ImportKey import key
func (o *Command) ImportKey(rw io.Writer, req io.Reader) command.Error {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(req)

	if err != nil {
		logutil.LogInfo(logger, commandName, importKeyCommandMethod, err.Error())
		return command.NewValidationError(InvalidRequestErrorCode, fmt.Errorf("failed request decode : %w", err))
	}

	var jwk jose.JWK
	if errUnmarshal := jwk.UnmarshalJSON(buf.Bytes()); errUnmarshal != nil {
		logutil.LogInfo(logger, commandName, importKeyCommandMethod, errUnmarshal.Error())
		return command.NewValidationError(InvalidRequestErrorCode, fmt.Errorf("failed request decode : %w", err))
	}

	if jwk.KeyID == "" {
		logutil.LogDebug(logger, commandName, importKeyCommandMethod, errEmptyKeyID)
		return command.NewValidationError(InvalidRequestErrorCode, fmt.Errorf(errEmptyKeyID))
	}

	var kType kms.KeyType

	switch jwk.Crv {
	case "Ed25519":
		kType = kms.ED25519Type
	case "P-256":
		kType = kms.ECDSAP256TypeIEEEP1363
	default:
		return command.NewValidationError(InvalidRequestErrorCode,
			fmt.Errorf("import key type not supported %s", jwk.Crv))
	}

	_, _, err = o.importKey(jwk.Key, kType, kms.WithKeyID(jwk.KeyID))
	if err != nil {
		logutil.LogError(logger, commandName, importKeyCommandMethod, err.Error())
		return command.NewExecuteError(ImportKeyError, err)
	}

	command.WriteNillableResponse(rw, nil, logger)

	logutil.LogDebug(logger, commandName, importKeyCommandMethod, "success")

	return nil
}
