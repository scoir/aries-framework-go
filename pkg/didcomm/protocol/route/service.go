/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package route

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hyperledger/aries-framework-go/pkg/common/log"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/common/model"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/common/service"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/dispatcher"
	"github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdri"
	"github.com/hyperledger/aries-framework-go/pkg/internal/logutil"
	"github.com/hyperledger/aries-framework-go/pkg/kms/legacykms"
	"github.com/hyperledger/aries-framework-go/pkg/storage"
	"github.com/hyperledger/aries-framework-go/pkg/store/connection"
)

var logger = log.New("aries-framework/route/service")

// constants for route coordination spec types
const (
	// Coordination route coordination protocol
	Coordination = "routecoordination"

	// RouteCoordinationSpec defines the route coordination spec
	CoordinationSpec = "https://didcomm.org/routecoordination/1.0/"

	// RouteRequestMsgType defines the route coordination request message type.
	RequestMsgType = CoordinationSpec + "route-request"

	// RouteGrantMsgType defines the route coordination request grant message type.
	GrantMsgType = CoordinationSpec + "route-grant"

	// KeyListUpdateMsgType defines the route coordination key list update message type.
	KeylistUpdateMsgType = CoordinationSpec + "keylist_update"

	// KeyListUpdateResponseMsgType defines the route coordination key list update message response type.
	KeylistUpdateResponseMsgType = CoordinationSpec + "keylist_update_response"
)

// constants for key list update processing
// https://github.com/hyperledger/aries-rfcs/tree/master/features/0211-route-coordination#keylist-update
const (
	// add key to the store
	add = "add"

	// remove key from the store
	remove = "remove"

	// server error while storing the key
	serverError = "server_error"

	// key save success
	success = "success"
)

const (
	// data key to store router connection ID
	routeConnIDDataKey = "route-connID"

	// data key to store router config
	routeConfigDataKey = "route-config"
)

const (
	updateTimeout = 5 * time.Second
)

// ErrConnectionNotFound connection not found error
var ErrConnectionNotFound = errors.New("connection not found")

// ErrRouterNotRegistered router not registered error
var ErrRouterNotRegistered = errors.New("router not registered")

// provider contains dependencies for the Routing protocol and is typically created by using aries.Context()
type provider interface {
	OutboundDispatcher() dispatcher.Outbound
	StorageProvider() storage.Provider
	TransientStorageProvider() storage.Provider
	RouterEndpoint() string
	LegacyKMS() legacykms.KeyManager
	VDRIRegistry() vdri.Registry
}

// Options is a container for route protocol options.
type Options struct {
	ServiceEndpoint string
	RoutingKeys     []string
}

type callback struct {
	msg      service.DIDCommMsg
	myDID    string
	theirDID string
	options  *Options
	err      error
}

// Service for Route Coordination protocol.
// https://github.com/hyperledger/aries-rfcs/tree/master/features/0211-route-coordination
type Service struct {
	service.Action
	service.Message
	routeStore               storage.Store
	connectionLookup         *connection.Lookup
	outbound                 dispatcher.Outbound
	endpoint                 string
	kms                      legacykms.KeyManager
	vdRegistry               vdri.Registry
	routeRegistrationMap     map[string]chan Grant
	routeRegistrationMapLock sync.RWMutex
	keylistUpdateMap         map[string]chan *KeylistUpdateResponse
	keylistUpdateMapLock     sync.RWMutex
	callbacks                chan *callback
	clientTimeout            time.Duration
}

// New return route coordination service.
func New(prov provider) (*Service, error) {
	store, err := prov.StorageProvider().OpenStore(Coordination)
	if err != nil {
		return nil, fmt.Errorf("open route coordination store : %w", err)
	}

	connectionLookup, err := connection.NewLookup(prov)
	if err != nil {
		return nil, err
	}

	s := &Service{
		routeStore:           store,
		outbound:             prov.OutboundDispatcher(),
		endpoint:             prov.RouterEndpoint(),
		kms:                  prov.LegacyKMS(),
		vdRegistry:           prov.VDRIRegistry(),
		connectionLookup:     connectionLookup,
		routeRegistrationMap: make(map[string]chan Grant),
		keylistUpdateMap:     make(map[string]chan *KeylistUpdateResponse),
		callbacks:            make(chan *callback),
	}

	go s.listenForCallbacks()

	return s, nil
}

func (s *Service) listenForCallbacks() {
	for c := range s.callbacks {
		logger.Debugf("handling user callback for msgID=%s err=%s", c.msg.ID(), c.err)

		if c.err != nil {
			go s.handleUserRejection(c)
			continue
		}

		switch c.msg.Type() {
		case RequestMsgType:
			err := s.handleInboundRequest(c)
			if err != nil {
				logger.Errorf("failed to handle inbound request: %+v : %w", c.msg, err)
			}
		default:
			logger.Warnf("ignoring unsupported message type %s", c.msg.Type())
		}
	}
}

func (s *Service) handleUserRejection(c *callback) {
	logger.Infof("user aborted response action for msgID=%s", c.msg.ID())
}

func triggersActionEvent(msgType string) bool {
	return msgType == RequestMsgType
}

func (s *Service) sendActionEvent(msg service.DIDCommMsg, myDID, theirDID string) error {
	events := s.ActionEvent()
	if events == nil {
		return fmt.Errorf("no clients registered to handle action events for %s protocol", Coordination)
	}

	logger.Debugf("dispatching action event for msg=%+v myDID=%s theirDID=%s", msg, myDID, theirDID)

	go func() {
		c := &callback{
			msg:      msg,
			myDID:    myDID,
			theirDID: theirDID,
		}

		events <- service.DIDCommAction{
			ProtocolName: Coordination,
			Message:      msg,
			Continue: func(args interface{}) {
				opts, ok := args.(Options)
				if !ok {
					opts = Options{}
				}

				c.options = &opts

				s.callbacks <- c
			},
			Stop: func(err error) {
				c.err = err

				s.callbacks <- c
			},
		}
	}()

	return nil
}

// HandleInbound handles inbound route coordination messages.
func (s *Service) HandleInbound(msg service.DIDCommMsg, myDID, theirDID string) (string, error) {
	logger.Debugf("input: msg=%+v myDID=%s theirDID=%s", msg, myDID, theirDID)

	if triggersActionEvent(msg.Type()) {
		return msg.ID(), s.sendActionEvent(msg, myDID, theirDID)
	}

	// perform action on inbound message asynchronously
	go func() {
		var err error

		switch msg.Type() {
		case GrantMsgType:
			err = s.handleGrant(msg)
		case KeylistUpdateMsgType:
			err = s.handleKeylistUpdate(msg, myDID, theirDID)
		case KeylistUpdateResponseMsgType:
			err = s.handleKeylistUpdateResponse(msg)
		case service.ForwardMsgType:
			err = s.handleForward(msg)
		}

		connectionID, connErr := s.connectionLookup.GetConnectionIDByDIDs(myDID, theirDID)
		if connErr != nil {
			logutil.LogError(logger, Coordination, "connectionID lookup using DIDs", connErr.Error())
		}

		if err != nil {
			logutil.LogError(logger, Coordination, "processMessage", err.Error(),
				logutil.CreateKeyValueString("msgType", msg.Type()),
				logutil.CreateKeyValueString("msgID", msg.ID()),
				logutil.CreateKeyValueString("connectionID", connectionID))
		} else {
			logutil.LogDebug(logger, Coordination, "processMessage", "success",
				logutil.CreateKeyValueString("msgType", msg.Type()),
				logutil.CreateKeyValueString("msgID", msg.ID()),
				logutil.CreateKeyValueString("connectionID", connectionID))
		}
	}()

	return msg.ID(), nil
}

// HandleOutbound handles outbound route coordination messages.
func (s *Service) HandleOutbound(msg service.DIDCommMsg, myDID, theirDID string) error {
	if !s.Accept(msg.Type()) {
		return fmt.Errorf("unsupported message type %s", msg.Type())
	}

	switch msg.Type() {
	case RequestMsgType:
		return s.handleOutboundRequest(msg, myDID, theirDID)
	default:
		return fmt.Errorf("invalid or unsupported outbound message type %s", msg.Type())
	}
}

// Accept checks whether the service can handle the message type.
func (s *Service) Accept(msgType string) bool {
	switch msgType {
	case RequestMsgType, GrantMsgType, KeylistUpdateMsgType, KeylistUpdateResponseMsgType, service.ForwardMsgType:
		return true
	}

	return false
}

// Name of the service
func (s *Service) Name() string {
	return Coordination
}

func (s *Service) handleInboundRequest(c *callback) error {
	// unmarshal the payload
	request := &Request{}

	err := c.msg.Decode(request)
	if err != nil {
		return fmt.Errorf("route request message unmarshal : %w", err)
	}

	grant, err := outboundGrant(
		c.msg.ID(),
		c.options,
		s.endpoint,
		func() (string, error) {
			_, key, er := s.kms.CreateKeySet()
			return key, er
		},
	)
	if err != nil {
		return fmt.Errorf("failed to handle inbound request : %w", err)
	}

	return s.outbound.SendToDID(grant, c.myDID, c.theirDID)
}

func outboundGrant(
	msgID string, opts *Options,
	defaultEndpoint string, defaultKey func() (string, error)) (*Grant, error) {
	grant := &Grant{
		ID:          msgID,
		Type:        GrantMsgType,
		Endpoint:    opts.ServiceEndpoint,
		RoutingKeys: opts.RoutingKeys,
	}

	if grant.Endpoint == "" {
		grant.Endpoint = defaultEndpoint
	}

	if len(grant.RoutingKeys) == 0 {
		keys, err := defaultKey()
		if err != nil {
			return nil, fmt.Errorf("failed to create keys : %w", err)
		}

		grant.RoutingKeys = []string{keys}
	}

	return grant, nil
}

func (s *Service) handleGrant(msg service.DIDCommMsg) error {
	// unmarshal the payload
	grantMsg := &Grant{}

	err := msg.Decode(grantMsg)
	if err != nil {
		return fmt.Errorf("route grant message unmarshal : %w", err)
	}

	// check if there are any channels registered for the message ID
	grantCh := s.getRouteRegistrationCh(grantMsg.ID)

	if grantCh == nil {
		logger.Warnf("no channels awaiting grant with msgID=%s", grantMsg.ID)
		return nil
	}

	grantCh <- *grantMsg

	return nil
}

func (s *Service) handleKeylistUpdate(msg service.DIDCommMsg, myDID, theirDID string) error {
	// unmarshal the payload
	keyUpdate := &KeylistUpdate{}

	err := msg.Decode(keyUpdate)
	if err != nil {
		return fmt.Errorf("route key list update message unmarshal : %w", err)
	}

	var updates []UpdateResponse

	// update the db
	for _, v := range keyUpdate.Updates {
		if v.Action == add {
			val := theirDID
			result := success

			err = s.routeStore.Put(dataKey(v.RecipientKey), []byte(val))
			if err != nil {
				logger.Errorf("failed to add the route key to store : %s", err)

				result = serverError
			}

			// construct the response doc
			updates = append(updates, UpdateResponse{
				RecipientKey: v.RecipientKey,
				Action:       v.Action,
				Result:       result,
			})
		} else if v.Action == remove {
			// TODO remove from the store

			// construct the response doc
			updates = append(updates, UpdateResponse{
				RecipientKey: v.RecipientKey,
				Action:       v.Action,
				Result:       serverError,
			})
		}
	}

	// send the key update response
	updateResponse := &KeylistUpdateResponse{
		Type:    KeylistUpdateResponseMsgType,
		ID:      msg.ID(),
		Updated: updates,
	}

	return s.outbound.SendToDID(updateResponse, myDID, theirDID)
}

func (s *Service) handleKeylistUpdateResponse(msg service.DIDCommMsg) error {
	// unmarshal the payload
	respMsg := &KeylistUpdateResponse{}

	err := msg.Decode(respMsg)
	if err != nil {
		return fmt.Errorf("route keylist update response message unmarshal : %w", err)
	}

	// check if there are any channels registered for the message ID
	keylistUpdateCh := s.getKeyUpdateResponseCh(respMsg.ID)

	if keylistUpdateCh != nil {
		// invoke the channel for the incoming message
		keylistUpdateCh <- respMsg
	}

	return nil
}

func (s *Service) handleForward(msg service.DIDCommMsg) error {
	// unmarshal the payload
	forward := &model.Forward{}

	err := msg.Decode(forward)
	if err != nil {
		return fmt.Errorf("forward message unmarshal : %w", err)
	}

	// TODO Open question - https://github.com/hyperledger/aries-framework-go/issues/965 Mismatch between Route
	//  Coordination and Forward RFC. For now assume, the TO field contains the recipient key.
	theirDID, err := s.routeStore.Get(dataKey(forward.To))
	if err != nil {
		return fmt.Errorf("route key fetch : %w", err)
	}

	dest, err := service.GetDestination(string(theirDID), s.vdRegistry)
	if err != nil {
		return fmt.Errorf("get destination : %w", err)
	}

	aEvent := s.ActionEvent()
	if aEvent != nil {
		// trigger action event
		aEvent <- service.DIDCommAction{
			ProtocolName: Coordination,
			Message:      msg,
			Continue: func(args interface{}) {
				err := s.outbound.Forward(forward.Msg, dest)
				if err != nil {
					logger.Errorf("Router forward failed in continuation for")
				}
			},
			Stop: func(err error) {
			},
			Properties: &forwardEvent{
				theirDID: string(theirDID),
			},
		}
		return nil
	}

	return s.outbound.Forward(forward.Msg, dest)
}

// Register registers the agent with the router on the other end of the connection identified by
// connectionID. This method blocks until a response is received from the router or it times out.
// The agent is registered with the router and retrieves the router endpoint and routing keys.
// This function throws an error if the agent is already registered against a router.
// TODO https://github.com/hyperledger/aries-framework-go/issues/1076 Register agent with
//  multiple routers
func (s *Service) Register(connectionID string) error {
	// check if router is already registered
	routerConnID, err := s.getRouterConnectionID()
	if err != nil && !errors.Is(err, storage.ErrDataNotFound) {
		return fmt.Errorf("fetch router connection id : %w", err)
	}

	if routerConnID != "" {
		return errors.New("router is already registered")
	}

	// get the connection record for the ID to fetch DID information
	conn, err := s.getConnection(connectionID)
	if err != nil {
		return err
	}

	// generate message ID
	msgID := uuid.New().String()

	// register chan for callback processing
	grantCh := make(chan Grant)
	s.setRouteRegistrationCh(msgID, grantCh)

	// create request message
	req := &Request{
		ID:   msgID,
		Type: RequestMsgType,
	}

	// send message to the router
	if err := s.outbound.SendToDID(req, conn.MyDID, conn.TheirDID); err != nil {
		return fmt.Errorf("send route request: %w", err)
	}

	// callback processing (to make this function look like a sync function)
	select {
	case grantResp := <-grantCh:
		conf := &config{
			RouterEndpoint: grantResp.Endpoint,
			RoutingKeys:    grantResp.RoutingKeys,
		}

		if err := s.saveRouterConfig(conf); err != nil {
			return fmt.Errorf("save route config : %w", err)
		}
	// TODO https://github.com/hyperledger/aries-framework-go/issues/1134 configure this timeout at decorator level
	case <-time.After(updateTimeout):
		return errors.New(fmt.Sprintf("timeout waiting for grant from the router %s", updateTimeout))
	}

	// remove the channel once its been processed
	s.setRouteRegistrationCh(msgID, nil)

	// save the connectionID of the router
	return s.saveRouterConnectionID(connectionID)
}

// Unregister unregisters the agent with the router.
func (s *Service) Unregister() error {
	// check if router is already registered
	_, err := s.getRouterConnectionID()
	if err != nil && !errors.Is(err, storage.ErrDataNotFound) {
		return fmt.Errorf("fetch router connection id : %w", err)
	} else if errors.Is(err, storage.ErrDataNotFound) {
		return ErrRouterNotRegistered
	}

	// TODO Remove all the recKeys from the router
	//  https://github.com/hyperledger/aries-rfcs/tree/master/features/0211-route-coordination#keylist-update-response

	// reset the connectionID
	return s.saveRouterConnectionID("")
}

// GetConnection returns the connectionID of the router.
func (s *Service) GetConnection() (string, error) {
	routerConnID, err := s.getRouterConnectionID()
	if err != nil && !errors.Is(err, storage.ErrDataNotFound) {
		return "", fmt.Errorf("fetch router connection id : %w", err)
	} else if errors.Is(err, storage.ErrDataNotFound) || routerConnID == "" {
		return "", ErrRouterNotRegistered
	}

	return routerConnID, nil
}

// AddKey adds a recKey of the agent to the registered router. This method blocks until a response is
// received from the router or it times out.
// TODO https://github.com/hyperledger/aries-framework-go/issues/1076 Support for multiple routers
// TODO https://github.com/hyperledger/aries-framework-go/issues/1105 Support to Add multiple
//  recKeys to the Router
func (s *Service) AddKey(recKey string) error {
	// check if router is already registered
	routerConnID, err := s.getRouterConnectionID()
	if err != nil && !errors.Is(err, storage.ErrDataNotFound) {
		return fmt.Errorf("fetch router connection id : %w", err)
	}

	if routerConnID == "" {
		return ErrRouterNotRegistered
	}

	// get the connection record for the ID to fetch DID information
	conn, err := s.getConnection(routerConnID)
	if err != nil {
		return err
	}

	// generate message ID
	msgID := uuid.New().String()

	// register chan for callback processing
	keyUpdateCh := make(chan *KeylistUpdateResponse)
	s.setKeyUpdateResponseCh(msgID, keyUpdateCh)

	keyUpdate := &KeylistUpdate{
		ID:   msgID,
		Type: KeylistUpdateMsgType,
		Updates: []Update{
			{
				RecipientKey: recKey,
				Action:       add,
			},
		},
	}

	if err := s.outbound.SendToDID(keyUpdate, conn.MyDID, conn.TheirDID); err != nil {
		return fmt.Errorf("send route request: %w", err)
	}

	select {
	case keyUpdateResp := <-keyUpdateCh:
		if err := processKeylistUpdateResp(recKey, keyUpdateResp); err != nil {
			return err
		}
	// TODO https://github.com/hyperledger/aries-framework-go/issues/1134 configure this timeout at decorator level
	case <-time.After(updateTimeout):
		return errors.New("timeout waiting for keylist update response from the router")
	}

	// remove the channel once its been processed
	s.setKeyUpdateResponseCh(msgID, nil)

	return nil
}

// Config fetches the router config - endpoint and routingKeys.
func (s *Service) Config() (*Config, error) {
	// check if router is already registered
	_, err := s.getRouterConnectionID()
	if err != nil && !errors.Is(err, storage.ErrDataNotFound) {
		return nil, fmt.Errorf("fetch router connection id : %w", err)
	} else if errors.Is(err, storage.ErrDataNotFound) {
		return nil, ErrRouterNotRegistered
	}

	return s.getRouterConfig()
}

func processKeylistUpdateResp(recKey string, keyUpdateResp *KeylistUpdateResponse) error {
	for _, result := range keyUpdateResp.Updated {
		if result.RecipientKey == recKey && result.Action == add && result.Result != success {
			return errors.New("failed to update the recipient key with the router")
		}
	}

	return nil
}

func (s *Service) getRouteRegistrationCh(msgID string) chan Grant {
	s.routeRegistrationMapLock.RLock()
	defer s.routeRegistrationMapLock.RUnlock()

	return s.routeRegistrationMap[msgID]
}

func (s *Service) setRouteRegistrationCh(msgID string, grantCh chan Grant) {
	s.routeRegistrationMapLock.Lock()
	defer s.routeRegistrationMapLock.Unlock()

	if grantCh == nil {
		delete(s.routeRegistrationMap, msgID)
	} else {
		s.routeRegistrationMap[msgID] = grantCh
	}
}

func (s *Service) getKeyUpdateResponseCh(msgID string) chan *KeylistUpdateResponse {
	s.keylistUpdateMapLock.RLock()
	defer s.keylistUpdateMapLock.RUnlock()

	return s.keylistUpdateMap[msgID]
}

func (s *Service) setKeyUpdateResponseCh(msgID string, keyUpdateCh chan *KeylistUpdateResponse) {
	s.keylistUpdateMapLock.Lock()
	defer s.keylistUpdateMapLock.Unlock()

	if keyUpdateCh == nil {
		delete(s.keylistUpdateMap, msgID)
	} else {
		s.keylistUpdateMap[msgID] = keyUpdateCh
	}
}

func (s *Service) getRouterConnectionID() (string, error) {
	id, err := s.routeStore.Get(routeConnIDDataKey)
	if err != nil {
		return "", err
	}

	return string(id), nil
}

func (s *Service) saveRouterConnectionID(id string) error {
	return s.routeStore.Put(routeConnIDDataKey, []byte(id))
}

type config struct {
	RouterEndpoint string
	RoutingKeys    []string
}

func (s *Service) getRouterConfig() (*Config, error) {
	val, err := s.routeStore.Get(routeConfigDataKey)
	if err != nil {
		return nil, fmt.Errorf("get router config data : %w", err)
	}

	conf := &config{}

	err = json.Unmarshal(val, conf)
	if err != nil {
		return nil, fmt.Errorf("unmarshal router config data : %w", err)
	}

	return NewConfig(conf.RouterEndpoint, conf.RoutingKeys), nil
}

func (s *Service) saveRouterConfig(conf *config) error {
	bytes, err := json.Marshal(conf)
	if err != nil {
		return fmt.Errorf("store router config data : %w", err)
	}

	return s.routeStore.Put(routeConfigDataKey, bytes)
}

func (s *Service) getConnection(routerConnID string) (*connection.Record, error) {
	conn, err := s.connectionLookup.GetConnectionRecord(routerConnID)
	if err != nil {
		if errors.Is(err, storage.ErrDataNotFound) {
			return nil, ErrConnectionNotFound
		}

		return nil, fmt.Errorf("fetch connection record from store : %w", err)
	}

	return conn, nil
}

func (s *Service) handleOutboundRequest(_ service.DIDCommMsg, myDID, theirDID string) error {
	connID, err := s.connectionLookup.GetConnectionIDByDIDs(myDID, theirDID)
	if err != nil {
		return fmt.Errorf("failed to lookup connection record for myDID=%s theirDID=%s : %w", myDID, theirDID, err)
	}

	return s.Register(connID)
}

func dataKey(id string) string {
	return "route-" + id
}
