/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package controller

import (
	"fmt"

	"github.com/hyperledger/aries-framework-go/pkg/controller/command"
	didexchangecmd "github.com/hyperledger/aries-framework-go/pkg/controller/command/didexchange"
	issuecredentialcmd "github.com/hyperledger/aries-framework-go/pkg/controller/command/issuecredential"
	"github.com/hyperledger/aries-framework-go/pkg/controller/command/kms"
	routercmd "github.com/hyperledger/aries-framework-go/pkg/controller/command/mediator"
	messagingcmd "github.com/hyperledger/aries-framework-go/pkg/controller/command/messaging"
	presentproofcmd "github.com/hyperledger/aries-framework-go/pkg/controller/command/presentproof"
	vdricmd "github.com/hyperledger/aries-framework-go/pkg/controller/command/vdri"
	"github.com/hyperledger/aries-framework-go/pkg/controller/command/verifiable"
	"github.com/hyperledger/aries-framework-go/pkg/controller/rest"
	didexchangerest "github.com/hyperledger/aries-framework-go/pkg/controller/rest/didexchange"
	issuecredentialrest "github.com/hyperledger/aries-framework-go/pkg/controller/rest/issuecredential"
	kmsrest "github.com/hyperledger/aries-framework-go/pkg/controller/rest/kms"
	"github.com/hyperledger/aries-framework-go/pkg/controller/rest/mediator"
	messagingrest "github.com/hyperledger/aries-framework-go/pkg/controller/rest/messaging"
	presentproofrest "github.com/hyperledger/aries-framework-go/pkg/controller/rest/presentproof"
	vdrirest "github.com/hyperledger/aries-framework-go/pkg/controller/rest/vdri"
	verifiablerest "github.com/hyperledger/aries-framework-go/pkg/controller/rest/verifiable"
	"github.com/hyperledger/aries-framework-go/pkg/controller/webnotifier"
	"github.com/hyperledger/aries-framework-go/pkg/framework/context"
)

type allOpts struct {
	webhookURLs  []string
	defaultLabel string
	autoAccept   bool
	msgHandler   command.MessageHandler
	notifier     command.Notifier
}

const wsPath = "/ws"

// Opt represents a controller option.
type Opt func(opts *allOpts)

// WithWebhookURLs is an option for setting up a webhook dispatcher which will notify clients of events
func WithWebhookURLs(webhookURLs ...string) Opt {
	return func(opts *allOpts) {
		opts.webhookURLs = webhookURLs
	}
}

// WithNotifier is an option for setting up a notifier which will notify clients of events
func WithNotifier(notifier command.Notifier) Opt {
	return func(opts *allOpts) {
		opts.notifier = notifier
	}
}

// WithDefaultLabel is an option allowing for the defaultLabel to be set.
func WithDefaultLabel(defaultLabel string) Opt {
	return func(opts *allOpts) {
		opts.defaultLabel = defaultLabel
	}
}

// WithAutoAccept is an option allowing for the auto accept to be set.
func WithAutoAccept(autoAccept bool) Opt {
	return func(opts *allOpts) {
		opts.autoAccept = autoAccept
	}
}

// WithMessageHandler is an option allowing for the message handler to be set.
func WithMessageHandler(handler command.MessageHandler) Opt {
	return func(opts *allOpts) {
		opts.msgHandler = handler
	}
}

// GetRESTHandlers returns all REST handlers provided by controller.
func GetRESTHandlers(ctx *context.Provider, opts ...Opt) ([]rest.Handler, error) { // nolint: funlen,gocyclo
	restAPIOpts := &allOpts{}
	// Apply options
	for _, opt := range opts {
		opt(restAPIOpts)
	}

	notifier := restAPIOpts.notifier
	if notifier == nil {
		notifier = webnotifier.New(wsPath, restAPIOpts.webhookURLs)
	}

	// DID Exchange REST operation
	exchangeOp, err := didexchangerest.New(ctx, notifier, restAPIOpts.defaultLabel,
		restAPIOpts.autoAccept)
	if err != nil {
		return nil, err
	}

	// VDRI REST operation
	vdriOp, err := vdrirest.New(ctx)
	if err != nil {
		return nil, err
	}

	// messaging REST operation
	messagingOp, err := messagingrest.New(ctx, restAPIOpts.msgHandler, notifier)
	if err != nil {
		return nil, err
	}

	// route REST operation
	routeOp, err := mediator.New(ctx, restAPIOpts.autoAccept)
	if err != nil {
		return nil, err
	}

	// verifiable command operation
	verifiablecmd, err := verifiablerest.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create verifiable rest command : %w", err)
	}

	// issuecredential REST operation
	issuecredentialOp, err := issuecredentialrest.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create issue-credential rest command : %w", err)
	}

	// presentproof REST operation
	presentproofOp, err := presentproofrest.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create present-proof rest command : %w", err)
	}

	// kms command operation
	kmscmd := kmsrest.New(ctx)

	// creat handlers from all operations
	var allHandlers []rest.Handler
	allHandlers = append(allHandlers, exchangeOp.GetRESTHandlers()...)
	allHandlers = append(allHandlers, vdriOp.GetRESTHandlers()...)
	allHandlers = append(allHandlers, messagingOp.GetRESTHandlers()...)
	allHandlers = append(allHandlers, routeOp.GetRESTHandlers()...)
	allHandlers = append(allHandlers, verifiablecmd.GetRESTHandlers()...)
	allHandlers = append(allHandlers, issuecredentialOp.GetRESTHandlers()...)
	allHandlers = append(allHandlers, presentproofOp.GetRESTHandlers()...)
	allHandlers = append(allHandlers, kmscmd.GetRESTHandlers()...)

	nhp, ok := notifier.(handlerProvider)
	if ok {
		allHandlers = append(allHandlers, nhp.GetRESTHandlers()...)
	}

	return allHandlers, nil
}

type handlerProvider interface {
	GetRESTHandlers() []rest.Handler
}

// GetCommandHandlers returns all command handlers provided by controller.
func GetCommandHandlers(ctx *context.Provider, opts ...Opt) ([]command.Handler, error) { // nolint: funlen
	cmdOpts := &allOpts{}
	// Apply options
	for _, opt := range opts {
		opt(cmdOpts)
	}

	notifier := cmdOpts.notifier
	if notifier == nil {
		notifier = webnotifier.New(wsPath, cmdOpts.webhookURLs)
	}

	// did exchange command operation
	didexcmd, err := didexchangecmd.New(ctx, notifier, cmdOpts.defaultLabel,
		cmdOpts.autoAccept)
	if err != nil {
		return nil, fmt.Errorf("failed initialized didexchange command: %w", err)
	}

	// VDRI command operation
	vcmd, err := vdricmd.New(ctx)
	if err != nil {
		return nil, err
	}

	// messaging command operation
	msgcmd, err := messagingcmd.New(ctx, cmdOpts.msgHandler, notifier)
	if err != nil {
		return nil, err
	}

	// route command operation
	routecmd, err := routercmd.New(ctx, cmdOpts.autoAccept)
	if err != nil {
		return nil, err
	}

	// verifiable command operation
	verifiablecmd, err := verifiable.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create verifiable command : %w", err)
	}

	// issuecredential command operation
	issuecredential, err := issuecredentialcmd.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create issue-credential command : %w", err)
	}

	// presentproof command operation
	presentproof, err := presentproofcmd.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create present-proof command : %w", err)
	}

	// kms command operation
	kmscmd := kms.New(ctx)

	var allHandlers []command.Handler
	allHandlers = append(allHandlers, didexcmd.GetHandlers()...)
	allHandlers = append(allHandlers, vcmd.GetHandlers()...)
	allHandlers = append(allHandlers, msgcmd.GetHandlers()...)
	allHandlers = append(allHandlers, routecmd.GetHandlers()...)
	allHandlers = append(allHandlers, verifiablecmd.GetHandlers()...)
	allHandlers = append(allHandlers, kmscmd.GetHandlers()...)
	allHandlers = append(allHandlers, issuecredential.GetHandlers()...)
	allHandlers = append(allHandlers, presentproof.GetHandlers()...)

	return allHandlers, nil
}
