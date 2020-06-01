/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dispatcher

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"

	"github.com/hyperledger/aries-framework-go/pkg/didcomm/common/model"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/common/service"
	commontransport "github.com/hyperledger/aries-framework-go/pkg/didcomm/common/transport"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/decorator"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
	"github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdri"
	"github.com/hyperledger/aries-framework-go/pkg/kms/legacykms"
)

// provider interface for outbound ctx
type provider interface {
	Packager() commontransport.Packager
	OutboundTransports() []transport.OutboundTransport
	TransportReturnRoute() string
	VDRIRegistry() vdri.Registry
	LegacyKMS() legacykms.KeyManager
}

// OutboundDispatcher dispatch msgs to destination
type OutboundDispatcher struct {
	outboundTransports   []transport.OutboundTransport
	packager             commontransport.Packager
	transportReturnRoute string
	vdRegistry           vdri.Registry
	kms                  legacykms.KeyManager
}

// NewOutbound return new dispatcher outbound instance
func NewOutbound(prov provider) *OutboundDispatcher {
	return &OutboundDispatcher{
		outboundTransports:   prov.OutboundTransports(),
		packager:             prov.Packager(),
		transportReturnRoute: prov.TransportReturnRoute(),
		vdRegistry:           prov.VDRIRegistry(),
		kms:                  prov.LegacyKMS(),
	}
}

// SendToDID sends a message from myDID to the agent who owns theirDID
func (o *OutboundDispatcher) SendToDID(msg interface{}, myDID, theirDID string) error {

	dest, err := service.GetDestination(theirDID, o.vdRegistry)
	if err != nil {
		return err
	}

	src, err := service.GetDestination(myDID, o.vdRegistry)
	if err != nil {
		return err
	}

	// We get at least one recipient key, so we can use the first one
	//  (right now, with only one key type used for sending)
	// TODO: relies on hardcoded key type
	key := src.RecipientKeys[0]

	return o.Send(msg, key, dest)
}

// Send sends the message after packing with the sender key and recipient keys.
func (o *OutboundDispatcher) Send(msg interface{}, senderVerKey string, des *service.Destination) error {
	for _, v := range o.outboundTransports {
		// check if outbound accepts routing keys, else use recipient keys
		keys := des.RecipientKeys
		if len(des.RoutingKeys) != 0 {
			keys = des.RoutingKeys
		}

		if !v.AcceptRecipient(keys) {
			if !v.Accept(des.ServiceEndpoint) {
				continue
			}
		}

		req, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed marshal to bytes: %w", err)
		}

		// update the outbound message with transport return route option [all or thread]
		req, err = o.addTransportRouteOptions(req, des)
		if err != nil {
			return fmt.Errorf("add transport route options : %w", err)
		}

		packedMsg, err := o.packager.PackMessage(
			&commontransport.Envelope{Message: req, FromVerKey: base58.Decode(senderVerKey), ToVerKeys: des.RecipientKeys})
		if err != nil {
			return fmt.Errorf("failed to pack msg: %w", err)
		}

		// set the return route option
		des.TransportReturnRoute = o.transportReturnRoute

		packedMsg, err = o.createForwardMessage(packedMsg, des)
		if err != nil {
			return fmt.Errorf("create forward msg : %w", err)
		}

		_, err = v.Send(packedMsg, des)
		if err != nil {
			return fmt.Errorf("failed to send msg using outbound transport: %w", err)
		}

		return nil
	}

	return fmt.Errorf("no outbound transport found for serviceEndpoint: %s", des.ServiceEndpoint)
}

// Forward forwards the message without packing to the destination.
func (o *OutboundDispatcher) Forward(msg interface{}, des *service.Destination) error {
	for _, v := range o.outboundTransports {
		if !v.AcceptRecipient(des.RecipientKeys) {
			if !v.Accept(des.ServiceEndpoint) {
				continue
			}
		}

		req, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed marshal to bytes: %w", err)
		}

		_, err = v.Send(req, des)
		if err != nil {
			return fmt.Errorf("failed to send msg using outbound transport: %w", err)
		}

		return nil
	}

	return fmt.Errorf("no outbound transport found for serviceEndpoint: %s", des.ServiceEndpoint)
}

func (o *OutboundDispatcher) createForwardMessage(msg []byte, des *service.Destination) ([]byte, error) {
	if len(des.RoutingKeys) == 0 {
		return msg, nil
	}

	env := &model.Envelope{}

	err := json.Unmarshal(msg, env)
	if err != nil {
		return nil, fmt.Errorf("unmarshal envelope : %w", err)
	}
	// create forward message
	forward := &model.Forward{
		Type: service.ForwardMsgType,
		ID:   uuid.New().String(),
		To:   des.RecipientKeys[0],
		Msg:  env,
	}

	// convert forward message to bytes
	req, err := json.Marshal(forward)
	if err != nil {
		return nil, fmt.Errorf("failed marshal to bytes: %w", err)
	}

	// create key set
	_, senderVerKey, err := o.kms.CreateKeySet()
	if err != nil {
		return nil, fmt.Errorf("failed CreateSigningKey: %w", err)
	}

	// pack above message using auth crypt
	// TODO https://github.com/hyperledger/aries-framework-go/issues/1112 Configurable packing
	//  algorithm(auth/anon crypt) for Forward(router) message
	packedMsg, err := o.packager.PackMessage(
		&commontransport.Envelope{Message: req, FromVerKey: base58.Decode(senderVerKey), ToVerKeys: des.RoutingKeys})
	if err != nil {
		return nil, fmt.Errorf("pack forward msg: %w", err)
	}

	return packedMsg, nil
}

func (o *OutboundDispatcher) addTransportRouteOptions(req []byte, des *service.Destination) ([]byte, error) {
	// dont add transport route options for forward messages
	if len(des.RoutingKeys) != 0 {
		return req, nil
	}

	if o.transportReturnRoute == decorator.TransportReturnRouteAll ||
		o.transportReturnRoute == decorator.TransportReturnRouteThread {
		// create the decorator with the option set in the framework
		transportDec := &decorator.Transport{ReturnRoute: &decorator.ReturnRoute{Value: o.transportReturnRoute}}

		transportDecJSON, jsonErr := json.Marshal(transportDec)
		if jsonErr != nil {
			return nil, fmt.Errorf("json marshal : %w", jsonErr)
		}

		request := string(req)
		index := strings.Index(request, "{")

		// add transport route option decorator to the original request
		req = []byte(request[:index+1] + string(transportDecJSON)[1:len(string(transportDecJSON))-1] + "," +
			request[index+1:])
	}

	return req, nil
}
