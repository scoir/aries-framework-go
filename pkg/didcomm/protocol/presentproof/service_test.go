/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package presentproof

import (
	"encoding/base64"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/aries-framework-go/pkg/didcomm/common/model"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/common/service"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/decorator"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	serviceMocks "github.com/hyperledger/aries-framework-go/pkg/internal/gomocks/didcomm/common/service"
	presentproofMocks "github.com/hyperledger/aries-framework-go/pkg/internal/gomocks/didcomm/protocol/presentproof"
	vdriMocks "github.com/hyperledger/aries-framework-go/pkg/internal/gomocks/framework/aries/api/vdri"
	storageMocks "github.com/hyperledger/aries-framework-go/pkg/internal/gomocks/storage"
	verifiableStoreMocks "github.com/hyperledger/aries-framework-go/pkg/internal/gomocks/store/verifiable"
	"github.com/hyperledger/aries-framework-go/pkg/storage"
	"github.com/hyperledger/aries-framework-go/pkg/storage/mem"
)

const (
	Alice = "Alice"
	Bob   = "Bob"
)

func TestNew(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("Success", func(t *testing.T) {
		storeProvider := storageMocks.NewMockProvider(ctrl)
		storeProvider.EXPECT().OpenStore(gomock.Any()).Return(nil, nil)

		verifiableStore := verifiableStoreMocks.NewMockStore(ctrl)

		provider := presentproofMocks.NewMockProvider(ctrl)
		provider.EXPECT().Messenger().Return(nil)
		provider.EXPECT().StorageProvider().Return(storeProvider)
		provider.EXPECT().VDRIRegistry().Return(nil)
		provider.EXPECT().VerifiableStore().Return(verifiableStore).AnyTimes()

		svc, err := New(provider)
		require.NoError(t, err)
		require.NotNil(t, svc)
	})

	t.Run("No verifiable store", func(t *testing.T) {
		storeProvider := storageMocks.NewMockProvider(ctrl)
		storeProvider.EXPECT().OpenStore(gomock.Any()).Return(nil, nil)

		provider := presentproofMocks.NewMockProvider(ctrl)
		provider.EXPECT().StorageProvider().Return(storeProvider)
		provider.EXPECT().VerifiableStore().Return(nil).AnyTimes()

		svc, err := New(provider)
		require.EqualError(t, err, "verifiable store is nil")
		require.Nil(t, svc)
	})

	t.Run("Error open store", func(t *testing.T) {
		const errMsg = "error"

		storeProvider := storageMocks.NewMockProvider(ctrl)
		storeProvider.EXPECT().OpenStore(Name).Return(nil, errors.New(errMsg))

		provider := presentproofMocks.NewMockProvider(ctrl)
		provider.EXPECT().StorageProvider().Return(storeProvider)

		svc, err := New(provider)
		require.Contains(t, fmt.Sprintf("%v", err), errMsg)
		require.Nil(t, svc)
	})
}

func TestService_ActionContinue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const errMsg = "error"

	store := storageMocks.NewMockStore(ctrl)
	storeProvider := storageMocks.NewMockProvider(ctrl)
	storeProvider.EXPECT().OpenStore(gomock.Any()).Return(store, nil).AnyTimes()

	verifiableStore := verifiableStoreMocks.NewMockStore(ctrl)

	provider := presentproofMocks.NewMockProvider(ctrl)
	provider.EXPECT().Messenger().Return(nil).AnyTimes()
	provider.EXPECT().VDRIRegistry().Return(nil).AnyTimes()
	provider.EXPECT().VerifiableStore().Return(verifiableStore).AnyTimes()
	provider.EXPECT().StorageProvider().Return(storeProvider).AnyTimes()

	t.Run("Error transitional payload (get)", func(t *testing.T) {
		store.EXPECT().Get(gomock.Any()).Return(nil, errors.New(errMsg))

		svc, err := New(provider)
		require.NoError(t, err)

		err = svc.ActionContinue("piID", nil)
		require.Contains(t, fmt.Sprintf("%v", err), "get transitional payload: store get: "+errMsg)
	})

	t.Run("Error transitional payload (delete)", func(t *testing.T) {
		store.EXPECT().Get(gomock.Any()).Return([]byte(`{}`), nil)
		store.EXPECT().Delete(gomock.Any()).Return(errors.New(errMsg))

		svc, err := New(provider)
		require.NoError(t, err)

		err = svc.ActionContinue("piID", nil)
		require.Contains(t, fmt.Sprintf("%v", err), "delete transitional payload: "+errMsg)
	})
}

func TestService_ActionStop(t *testing.T) {
	t.Run("Error transitional payload (get)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		const errMsg = "error"

		store := storageMocks.NewMockStore(ctrl)
		store.EXPECT().Get(gomock.Any()).Return(nil, errors.New(errMsg))

		verifiableStore := verifiableStoreMocks.NewMockStore(ctrl)

		storeProvider := storageMocks.NewMockProvider(ctrl)
		storeProvider.EXPECT().OpenStore(Name).Return(store, nil).AnyTimes()

		provider := presentproofMocks.NewMockProvider(ctrl)
		provider.EXPECT().Messenger().Return(nil)
		provider.EXPECT().StorageProvider().Return(storeProvider)
		provider.EXPECT().VDRIRegistry().Return(nil).AnyTimes()
		provider.EXPECT().VerifiableStore().Return(verifiableStore).AnyTimes()

		svc, err := New(provider)
		require.NoError(t, err)

		err = svc.ActionStop("piID", nil)
		require.Contains(t, fmt.Sprintf("%v", err), "get transitional payload: store get: "+errMsg)
	})

	t.Run("Error transitional payload (delete)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		const errMsg = "error"

		store := storageMocks.NewMockStore(ctrl)
		store.EXPECT().Get(gomock.Any()).Return([]byte(`{}`), nil)
		store.EXPECT().Delete(gomock.Any()).Return(errors.New(errMsg))

		verifiableStore := verifiableStoreMocks.NewMockStore(ctrl)

		storeProvider := storageMocks.NewMockProvider(ctrl)
		storeProvider.EXPECT().OpenStore(Name).Return(store, nil).AnyTimes()

		provider := presentproofMocks.NewMockProvider(ctrl)
		provider.EXPECT().Messenger().Return(nil)
		provider.EXPECT().StorageProvider().Return(storeProvider)
		provider.EXPECT().VDRIRegistry().Return(nil).AnyTimes()
		provider.EXPECT().VerifiableStore().Return(verifiableStore).AnyTimes()

		svc, err := New(provider)
		require.NoError(t, err)

		err = svc.ActionStop("piID", nil)
		require.Contains(t, fmt.Sprintf("%v", err), "delete transitional payload: "+errMsg)
	})
}

// nolint: gocyclo
func TestService_HandleInbound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const errMsg = "error"

	store := storageMocks.NewMockStore(ctrl)
	verifiableStore := verifiableStoreMocks.NewMockStore(ctrl)

	storeProvider := storageMocks.NewMockProvider(ctrl)
	storeProvider.EXPECT().OpenStore(Name).Return(store, nil).AnyTimes()

	messenger := serviceMocks.NewMockMessenger(ctrl)

	registry := vdriMocks.NewMockRegistry(ctrl)

	provider := presentproofMocks.NewMockProvider(ctrl)
	provider.EXPECT().Messenger().Return(messenger).AnyTimes()
	provider.EXPECT().StorageProvider().Return(storeProvider).AnyTimes()
	provider.EXPECT().VDRIRegistry().Return(registry).AnyTimes()
	provider.EXPECT().VerifiableStore().Return(verifiableStore).AnyTimes()

	t.Run("No clients", func(t *testing.T) {
		svc, err := New(provider)
		require.NoError(t, err)

		_, err = svc.HandleInbound(randomInboundMessage(""), "", "")
		require.Contains(t, fmt.Sprintf("%v", err), "no clients")
	})

	t.Run("DB error", func(t *testing.T) {
		store.EXPECT().Get(gomock.Any()).Return(nil, errors.New(errMsg))

		svc, err := New(provider)
		require.NoError(t, err)

		require.NoError(t, svc.RegisterActionEvent(make(chan<- service.DIDCommAction)))

		msg := service.NewDIDCommMsgMap(struct {
			ID     string           `json:"@id"`
			Thread decorator.Thread `json:"~thread"`
		}{ID: "ID", Thread: decorator.Thread{PID: "PID"}})

		require.NoError(t, msg.SetID(uuid.New().String()))
		_, err = svc.HandleInbound(msg, "", "")
		require.Contains(t, fmt.Sprintf("%v", err), "doHandle: getCurrentStateNameAndPIID: currentStateName: "+errMsg)
	})

	t.Run("DB error (saveTransitionalPayload)", func(t *testing.T) {
		store.EXPECT().Get(gomock.Any()).Return(nil, storage.ErrDataNotFound)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Return(errors.New(errMsg))

		svc, err := New(provider)
		require.NoError(t, err)

		require.NoError(t, svc.RegisterActionEvent(make(chan<- service.DIDCommAction)))

		_, err = svc.HandleInbound(randomInboundMessage(RequestPresentationMsgType), "", "")
		require.Contains(t, fmt.Sprintf("%v", err), "save transitional payload: "+errMsg)
	})

	t.Run("Unrecognized msgType", func(t *testing.T) {
		store.EXPECT().Get(gomock.Any()).Return(nil, storage.ErrDataNotFound)

		svc, err := New(provider)
		require.NoError(t, err)

		require.NoError(t, svc.RegisterActionEvent(make(chan<- service.DIDCommAction)))

		msg := service.NewDIDCommMsgMap(struct{}{})

		require.NoError(t, msg.SetID(uuid.New().String()))
		_, err = svc.HandleInbound(msg, "", "")
		require.Contains(t, fmt.Sprintf("%v", err), "doHandle: nextState: unrecognized msgType: ")
	})

	t.Run("Invalid state transition", func(t *testing.T) {
		svc, err := New(provider)
		require.NoError(t, err)

		ch := make(chan service.DIDCommAction, 1)
		require.NoError(t, svc.RegisterActionEvent(ch))

		chState := make(chan service.StateMsg, 2)
		require.NoError(t, svc.RegisterMsgEvent(chState))

		_, err = svc.HandleInbound(service.NewDIDCommMsgMap(model.ProblemReport{
			Type: ProblemReportMsgType,
		}), "", "")
		require.Contains(t, fmt.Sprintf("%v", err), "doHandle: invalid state transition")
	})

	t.Run("Receive Request Presentation (Stop)", func(t *testing.T) {
		var done = make(chan struct{})

		messenger.EXPECT().
			ReplyToNested(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(_ string, msg service.DIDCommMsgMap, myDID, theirDID string) error {
				r := &model.ProblemReport{}
				require.NoError(t, msg.Decode(r))
				require.Equal(t, codeRejectedError, r.Description.Code)
				require.Equal(t, ProblemReportMsgType, r.Type)

				return nil
			})

		store.EXPECT().Get(gomock.Any()).Return(nil, storage.ErrDataNotFound)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			require.Equal(t, "abandoning", string(name))

			return nil
		})
		store.EXPECT().Delete(gomock.Any()).Return(errors.New(errMsg))
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			defer close(done)

			require.Equal(t, "done", string(name))

			return nil
		})

		svc, err := New(provider)
		require.NoError(t, err)

		ch := make(chan service.DIDCommAction, 1)
		require.NoError(t, svc.RegisterActionEvent(ch))

		_, err = svc.HandleInbound(randomInboundMessage(RequestPresentationMsgType), Alice, Bob)
		require.NoError(t, err)

		(<-ch).Stop(nil)

		select {
		case <-done:
			return
		case <-time.After(time.Second):
			t.Error("timeout")
		}
	})

	t.Run("Receive Request Presentation (continue with presentation)", func(t *testing.T) {
		var done = make(chan struct{})

		messenger.EXPECT().ReplyTo(gomock.Any(), gomock.Any()).
			Do(func(_ string, msg service.DIDCommMsgMap) error {
				defer close(done)

				r := &Presentation{}
				require.NoError(t, msg.Decode(r))
				require.Equal(t, PresentationMsgType, r.Type)

				return nil
			})

		store.EXPECT().Get(gomock.Any()).Return(nil, storage.ErrDataNotFound)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			require.Equal(t, "request-received", string(name))

			return nil
		})
		store.EXPECT().Delete(gomock.Any()).Return(nil)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			require.Equal(t, "presentation-sent", string(name))

			return nil
		})

		svc, err := New(provider)
		require.NoError(t, err)

		ch := make(chan service.DIDCommAction, 1)
		require.NoError(t, svc.RegisterActionEvent(ch))

		_, err = svc.HandleInbound(randomInboundMessage(RequestPresentationMsgType), Alice, Bob)
		require.NoError(t, err)

		(<-ch).Continue(WithPresentation(&Presentation{}))

		select {
		case <-done:
			return
		case <-time.After(time.Second):
			t.Error("timeout")
		}
	})

	t.Run("Receive Request Presentation (continue with presentation) async", func(t *testing.T) {
		var done = make(chan struct{})

		newProvider := presentproofMocks.NewMockProvider(ctrl)
		newProvider.EXPECT().Messenger().Return(messenger)
		newProvider.EXPECT().StorageProvider().Return(mem.NewProvider())
		newProvider.EXPECT().VDRIRegistry().Return(nil)
		newProvider.EXPECT().VerifiableStore().Return(verifiableStore)

		messenger.EXPECT().ReplyTo(gomock.Any(), gomock.Any()).
			Do(func(_ string, msg service.DIDCommMsgMap) error {
				defer close(done)

				r := &Presentation{}
				require.NoError(t, msg.Decode(r))
				require.Equal(t, PresentationMsgType, r.Type)

				return nil
			})

		svc, err := New(newProvider)
		require.NoError(t, err)

		ch := make(chan service.DIDCommAction, 1)
		require.NoError(t, svc.RegisterActionEvent(ch))

		_, err = svc.HandleInbound(randomInboundMessage(RequestPresentationMsgType), Alice, Bob)
		require.NoError(t, err)

		actions, err := svc.Actions()
		require.NoError(t, err)
		for _, action := range actions {
			require.NoError(t, svc.ActionContinue(action.PIID, WithPresentation(&Presentation{})))
		}

		select {
		case <-done:
			return
		case <-time.After(time.Second):
			t.Error("timeout")
		}
	})

	t.Run("Receive Request Presentation (Stop) async", func(t *testing.T) {
		var done = make(chan struct{})

		newProvider := presentproofMocks.NewMockProvider(ctrl)
		newProvider.EXPECT().Messenger().Return(messenger)
		newProvider.EXPECT().StorageProvider().Return(mem.NewProvider())
		newProvider.EXPECT().VDRIRegistry().Return(nil)
		newProvider.EXPECT().VerifiableStore().Return(verifiableStore)

		messenger.EXPECT().
			ReplyToNested(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(_ string, msg service.DIDCommMsgMap, myDID, theirDID string) error {
				defer close(done)

				r := &model.ProblemReport{}
				require.NoError(t, msg.Decode(r))
				require.Equal(t, codeRejectedError, r.Description.Code)
				require.Equal(t, ProblemReportMsgType, r.Type)

				return nil
			})

		svc, err := New(newProvider)
		require.NoError(t, err)

		ch := make(chan service.DIDCommAction, 1)
		require.NoError(t, svc.RegisterActionEvent(ch))

		_, err = svc.HandleInbound(randomInboundMessage(RequestPresentationMsgType), Alice, Bob)
		require.NoError(t, err)

		actions, err := svc.Actions()
		require.NoError(t, err)
		for _, action := range actions {
			require.NoError(t, svc.ActionStop(action.PIID, nil))
		}

		select {
		case <-done:
			return
		case <-time.After(time.Second):
			t.Error("timeout")
		}
	})

	t.Run("Receive Request Presentation (continue with proposal)", func(t *testing.T) {
		var done = make(chan struct{})

		messenger.EXPECT().ReplyTo(gomock.Any(), gomock.Any()).
			Do(func(_ string, msg service.DIDCommMsgMap) error {
				defer close(done)

				r := &ProposePresentation{}
				require.NoError(t, msg.Decode(r))
				require.Equal(t, ProposePresentationMsgType, r.Type)

				return nil
			})

		store.EXPECT().Get(gomock.Any()).Return(nil, storage.ErrDataNotFound)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			require.Equal(t, "request-received", string(name))

			return nil
		})
		store.EXPECT().Delete(gomock.Any()).Return(nil)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			require.Equal(t, "proposal-sent", string(name))

			return nil
		})

		svc, err := New(provider)
		require.NoError(t, err)

		ch := make(chan service.DIDCommAction, 1)
		require.NoError(t, svc.RegisterActionEvent(ch))

		_, err = svc.HandleInbound(randomInboundMessage(RequestPresentationMsgType), Alice, Bob)
		require.NoError(t, err)

		(<-ch).Continue(WithProposePresentation(&ProposePresentation{}))

		select {
		case <-done:
			return
		case <-time.After(time.Second):
			t.Error("timeout")
		}
	})

	t.Run("Receive Propose Presentation (continue)", func(t *testing.T) {
		var done = make(chan struct{})

		messenger.EXPECT().ReplyTo(gomock.Any(), gomock.Any()).
			Do(func(_ string, msg service.DIDCommMsgMap) error {
				defer close(done)

				r := &RequestPresentation{}
				require.NoError(t, msg.Decode(r))
				require.Equal(t, RequestPresentationMsgType, r.Type)

				return nil
			})

		store.EXPECT().Get(gomock.Any()).Return(nil, storage.ErrDataNotFound)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().Delete(gomock.Any()).Return(nil)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			require.Equal(t, "proposal-received", string(name))

			return nil
		})

		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			require.Equal(t, "request-sent", string(name))

			return nil
		})

		svc, err := New(provider)
		require.NoError(t, err)

		ch := make(chan service.DIDCommAction, 1)
		require.NoError(t, svc.RegisterActionEvent(ch))

		_, err = svc.HandleInbound(randomInboundMessage(ProposePresentationMsgType), Alice, Bob)
		require.NoError(t, err)

		(<-ch).Continue(WithRequestPresentation(&RequestPresentation{}))

		select {
		case <-done:
			return
		case <-time.After(time.Second):
			t.Error("timeout")
		}
	})

	t.Run("Receive Propose Presentation (continue without request)", func(t *testing.T) {
		var done = make(chan struct{})

		messenger.EXPECT().
			ReplyToNested(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(_ string, msg service.DIDCommMsgMap, myDID, theirDID string) error {
				r := &model.ProblemReport{}
				require.NoError(t, msg.Decode(r))
				require.Equal(t, codeInternalError, r.Description.Code)
				require.Equal(t, ProblemReportMsgType, r.Type)

				return nil
			})

		store.EXPECT().Get(gomock.Any()).Return(nil, storage.ErrDataNotFound)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().Delete(gomock.Any()).Return(nil)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			require.Equal(t, "proposal-received", string(name))

			return nil
		})

		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			require.Equal(t, "abandoning", string(name))

			return nil
		})

		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			defer close(done)

			require.Equal(t, "done", string(name))

			return nil
		})

		svc, err := New(provider)
		require.NoError(t, err)

		ch := make(chan service.DIDCommAction, 1)
		require.NoError(t, svc.RegisterActionEvent(ch))

		_, err = svc.HandleInbound(randomInboundMessage(ProposePresentationMsgType), Alice, Bob)
		require.NoError(t, err)

		(<-ch).Continue(nil)

		select {
		case <-done:
			return
		case <-time.After(time.Second):
			t.Error("timeout")
		}
	})

	t.Run("Receive Presentation (continue)", func(t *testing.T) {
		var done = make(chan struct{})

		messenger.EXPECT().ReplyTo(gomock.Any(), gomock.Any()).
			Do(func(_ string, msg service.DIDCommMsgMap) error {
				r := &model.Ack{}
				require.NoError(t, msg.Decode(r))
				require.Equal(t, AckMsgType, r.Type)

				return nil
			})

		registry.EXPECT().Resolve("did:example:ebfeb1f712ebc6f1c276e12ec21").Return(&did.Doc{
			PublicKey: []did.PublicKey{{
				ID:    "key-1",
				Value: []byte{61, 133, 23, 17, 77, 132, 169, 196, 47, 203, 19, 71, 145, 144, 92, 145, 131, 101, 36, 251, 89, 216, 117, 140, 132, 226, 78, 187, 59, 58, 200, 255}, //nolint:lll
			}},
		}, nil)

		verifiableStore.EXPECT().SavePresentation(gomock.Any(), gomock.Any()).Return(nil)

		store.EXPECT().Get(gomock.Any()).Return([]byte("request-sent"), nil)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().Delete(gomock.Any()).Return(nil)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			require.Equal(t, "presentation-received", string(name))

			return nil
		})

		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			defer close(done)

			require.Equal(t, "done", string(name))

			return nil
		})

		svc, err := New(provider)
		require.NoError(t, err)

		ch := make(chan service.DIDCommAction, 1)
		require.NoError(t, svc.RegisterActionEvent(ch))

		vpJWS := "eyJhbGciOiJFZERTQSIsImtpZCI6ImtleS0xIiwidHlwIjoiSldUIn0.eyJpc3MiOiJkaWQ6ZXhhbXBsZTplYmZlYjFmNzEyZWJjNmYxYzI3NmUxMmVjMjEiLCJqdGkiOiJ1cm46dXVpZDozOTc4MzQ0Zi04NTk2LTRjM2EtYTk3OC04ZmNhYmEzOTAzYzUiLCJ2cCI6eyJAY29udGV4dCI6WyJodHRwczovL3d3dy53My5vcmcvMjAxOC9jcmVkZW50aWFscy92MSIsImh0dHBzOi8vd3d3LnczLm9yZy8yMDE4L2NyZWRlbnRpYWxzL2V4YW1wbGVzL3YxIl0sInR5cGUiOlsiVmVyaWZpYWJsZVByZXNlbnRhdGlvbiIsIlVuaXZlcnNpdHlEZWdyZWVDcmVkZW50aWFsIl0sInZlcmlmaWFibGVDcmVkZW50aWFsIjpbeyJAY29udGV4dCI6WyJodHRwczovL3d3dy53My5vcmcvMjAxOC9jcmVkZW50aWFscy92MSIsImh0dHBzOi8vd3d3LnczLm9yZy8yMDE4L2NyZWRlbnRpYWxzL2V4YW1wbGVzL3YxIl0sImNyZWRlbnRpYWxTY2hlbWEiOltdLCJjcmVkZW50aWFsU3ViamVjdCI6eyJkZWdyZWUiOnsidHlwZSI6IkJhY2hlbG9yRGVncmVlIiwidW5pdmVyc2l0eSI6Ik1JVCJ9LCJpZCI6ImRpZDpleGFtcGxlOmViZmViMWY3MTJlYmM2ZjFjMjc2ZTEyZWMyMSIsIm5hbWUiOiJKYXlkZW4gRG9lIiwic3BvdXNlIjoiZGlkOmV4YW1wbGU6YzI3NmUxMmVjMjFlYmZlYjFmNzEyZWJjNmYxIn0sImV4cGlyYXRpb25EYXRlIjoiMjAyMC0wMS0wMVQxOToyMzoyNFoiLCJpZCI6Imh0dHA6Ly9leGFtcGxlLmVkdS9jcmVkZW50aWFscy8xODcyIiwiaXNzdWFuY2VEYXRlIjoiMjAxMC0wMS0wMVQxOToyMzoyNFoiLCJpc3N1ZXIiOnsiaWQiOiJkaWQ6ZXhhbXBsZTo3NmUxMmVjNzEyZWJjNmYxYzIyMWViZmViMWYiLCJuYW1lIjoiRXhhbXBsZSBVbml2ZXJzaXR5In0sInJlZmVyZW5jZU51bWJlciI6OC4zMjk0ODQ3ZSswNywidHlwZSI6WyJWZXJpZmlhYmxlQ3JlZGVudGlhbCIsIlVuaXZlcnNpdHlEZWdyZWVDcmVkZW50aWFsIl19XX19.RlO_1B-7qhQNwo2mmOFUWSa8A6hwaJrtq3q7yJDkKq4k6B-EJ-oyLNM6H_g2_nko2Yg9Im1CiROFm6nK12U_AQ" //nolint:lll

		msg := service.NewDIDCommMsgMap(Presentation{
			Type: PresentationMsgType,
			Presentations: []decorator.Attachment{{
				Data: decorator.AttachmentData{
					Base64: base64.StdEncoding.EncodeToString([]byte(vpJWS)),
				},
			}},
		})
		require.NoError(t, msg.SetID(uuid.New().String()))
		msg["~thread"] = decorator.Thread{ID: uuid.New().String()}

		_, err = svc.HandleInbound(msg, Alice, Bob)
		require.NoError(t, err)

		(<-ch).Continue(nil)

		select {
		case <-done:
			return
		case <-time.After(time.Second * 10):
			t.Error("timeout")
		}
	})

	t.Run("Receive Ack", func(t *testing.T) {
		var done = make(chan struct{})

		store.EXPECT().Get(gomock.Any()).Return([]byte("presentation-sent"), nil)
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			defer close(done)

			require.Equal(t, "done", string(name))

			return nil
		})

		svc, err := New(provider)
		require.NoError(t, err)

		ch := make(chan service.DIDCommAction, 1)
		require.NoError(t, svc.RegisterActionEvent(ch))

		_, err = svc.HandleInbound(randomInboundMessage(AckMsgType), Alice, Bob)
		require.NoError(t, err)

		select {
		case <-done:
			return
		case <-time.After(time.Second):
			t.Error("timeout")
		}
	})

	t.Run("Send Request Presentation", func(t *testing.T) {
		var done = make(chan struct{})

		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			require.Equal(t, "request-sent", string(name))

			return nil
		})

		svc, err := New(provider)
		require.NoError(t, err)

		msg := service.NewDIDCommMsgMap(RequestPresentation{
			Type: RequestPresentationMsgType,
		})

		messenger.EXPECT().Send(msg, Alice, Bob).
			Do(func(msg service.DIDCommMsgMap, myDID, theirDID string) error {
				defer close(done)

				return nil
			})

		_, err = svc.HandleInbound(msg, Alice, Bob)
		require.NoError(t, err)

		select {
		case <-done:
			return
		case <-time.After(time.Second):
			t.Error("timeout")
		}
	})

	t.Run("Send Request Presentation with error", func(t *testing.T) {
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)

		svc, err := New(provider)
		require.NoError(t, err)

		msg := service.NewDIDCommMsgMap(RequestPresentation{
			Type: RequestPresentationMsgType,
		})

		messenger.EXPECT().Send(msg, Alice, Bob).Return(errors.New(errMsg))

		_, err = svc.HandleInbound(msg, Alice, Bob)
		require.Contains(t, fmt.Sprintf("%v", err), "action request-sent: "+errMsg)
	})

	t.Run("Send Proposal", func(t *testing.T) {
		var done = make(chan struct{})

		store.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(_ string, name []byte) error {
			require.Equal(t, "proposal-sent", string(name))

			return nil
		})

		svc, err := New(provider)
		require.NoError(t, err)

		msg := service.NewDIDCommMsgMap(ProposePresentation{
			Type: ProposePresentationMsgType,
		})

		messenger.EXPECT().Send(msg, Alice, Bob).
			Do(func(msg service.DIDCommMsgMap, myDID, theirDID string) error {
				defer close(done)

				return nil
			})

		_, err = svc.HandleInbound(msg, Alice, Bob)
		require.NoError(t, err)

		select {
		case <-done:
			return
		case <-time.After(time.Second):
			t.Error("timeout")
		}
	})

	t.Run("Send Proposal with error", func(t *testing.T) {
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)

		svc, err := New(provider)
		require.NoError(t, err)

		msg := service.NewDIDCommMsgMap(ProposePresentation{
			Type: ProposePresentationMsgType,
		})

		messenger.EXPECT().Send(msg, Alice, Bob).Return(errors.New(errMsg))

		_, err = svc.HandleInbound(msg, Alice, Bob)
		require.Contains(t, fmt.Sprintf("%v", err), "action proposal-sent: "+errMsg)
	})
}

func Test_stateFromName(t *testing.T) {
	require.Equal(t, stateFromName(stateNameStart), &start{})
	require.Equal(t, stateFromName(stateNameAbandoning), &abandoning{})
	require.Equal(t, stateFromName(stateNameDone), &done{})
	require.Equal(t, stateFromName(stateNameRequestSent), &requestSent{})
	require.Equal(t, stateFromName(stateNamePresentationReceived), &presentationReceived{})
	require.Equal(t, stateFromName(stateNameProposalReceived), &proposalReceived{})
	require.Equal(t, stateFromName(stateNameRequestReceived), &requestReceived{})
	require.Equal(t, stateFromName(stateNamePresentationSent), &presentationSent{})
	require.Equal(t, stateFromName(stateNameProposalSent), &proposalSent{})
	require.Equal(t, stateFromName("unknown"), &noOp{})
}

func TestService_Name(t *testing.T) {
	require.Equal(t, (*Service).Name(nil), Name)
}

func TestService_Accept(t *testing.T) {
	require.True(t, (*Service).Accept(nil, ProposePresentationMsgType))
	require.True(t, (*Service).Accept(nil, RequestPresentationMsgType))
	require.True(t, (*Service).Accept(nil, PresentationMsgType))
	require.True(t, (*Service).Accept(nil, AckMsgType))
	require.True(t, (*Service).Accept(nil, ProblemReportMsgType))
	require.False(t, (*Service).Accept(nil, "unknown"))
}

func TestService_canTriggerActionEvents(t *testing.T) {
	require.True(t, canTriggerActionEvents(service.NewDIDCommMsgMap(ProposePresentation{
		Type: ProposePresentationMsgType,
	})))

	require.True(t, canTriggerActionEvents(service.NewDIDCommMsgMap(RequestPresentation{
		Type: RequestPresentationMsgType,
	})))

	require.True(t, canTriggerActionEvents(service.NewDIDCommMsgMap(Presentation{
		Type: PresentationMsgType,
	})))

	require.False(t, canTriggerActionEvents(service.NewDIDCommMsgMap(struct{}{})))
}

func Test_getTransitionalPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := storageMocks.NewMockStore(ctrl)
	verifiableStore := verifiableStoreMocks.NewMockStore(ctrl)

	storeProvider := storageMocks.NewMockProvider(ctrl)
	storeProvider.EXPECT().OpenStore(Name).Return(store, nil).AnyTimes()

	provider := presentproofMocks.NewMockProvider(ctrl)
	provider.EXPECT().Messenger().Return(nil).AnyTimes()
	provider.EXPECT().StorageProvider().Return(storeProvider).AnyTimes()
	provider.EXPECT().VDRIRegistry().Return(nil).AnyTimes()
	provider.EXPECT().VerifiableStore().Return(verifiableStore).AnyTimes()

	store.EXPECT().Get(fmt.Sprintf(transitionalPayloadKey, "ID")).Return([]byte(`[]`), nil)

	svc, err := New(provider)
	require.NoError(t, err)

	res, err := svc.getTransitionalPayload("ID")
	require.Nil(t, res)
	require.Contains(t, fmt.Sprintf("%v", err), "unmarshal transitional payload")
}

func Test_nextState(t *testing.T) {
	next, err := nextState(service.NewDIDCommMsgMap(RequestPresentation{
		Type: RequestPresentationMsgType,
	}))
	require.NoError(t, err)
	require.Equal(t, next, &requestSent{})

	next, err = nextState(randomInboundMessage(RequestPresentationMsgType))
	require.NoError(t, err)
	require.Equal(t, next, &requestReceived{})

	next, err = nextState(service.NewDIDCommMsgMap(ProposePresentation{
		Type: ProposePresentationMsgType,
	}))
	require.NoError(t, err)
	require.Equal(t, next, &proposalSent{})

	next, err = nextState(randomInboundMessage(ProposePresentationMsgType))
	require.NoError(t, err)
	require.Equal(t, next, &proposalReceived{})

	next, err = nextState(randomInboundMessage(PresentationMsgType))
	require.NoError(t, err)
	require.Equal(t, next, &presentationReceived{})

	next, err = nextState(randomInboundMessage(AckMsgType))
	require.NoError(t, err)
	require.Equal(t, next, &done{})

	next, err = nextState(randomInboundMessage(ProblemReportMsgType))
	require.NoError(t, err)
	require.Equal(t, next, &abandoning{})

	next, err = nextState(service.NewDIDCommMsgMap(struct{}{}))
	require.Error(t, err)
	require.Nil(t, next)
}
