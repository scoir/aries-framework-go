/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package introduce

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/google/uuid"

	"github.com/hyperledger/aries-framework-go/pkg/client/didexchange"
	"github.com/hyperledger/aries-framework-go/pkg/client/introduce"
	"github.com/hyperledger/aries-framework-go/pkg/client/outofband"
	routeClient "github.com/hyperledger/aries-framework-go/pkg/client/route"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/common/service"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/decorator"
	introduceService "github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/introduce"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/route"
	"github.com/hyperledger/aries-framework-go/test/bdd/agent"
	"github.com/hyperledger/aries-framework-go/test/bdd/pkg/context"
	bddDIDExchange "github.com/hyperledger/aries-framework-go/test/bdd/pkg/didexchange"
	"github.com/hyperledger/aries-framework-go/test/bdd/pkg/didresolver"
	bddOutOfBand "github.com/hyperledger/aries-framework-go/test/bdd/pkg/outofband"
	routeBDDSteps "github.com/hyperledger/aries-framework-go/test/bdd/pkg/route"
)

const timeout = time.Second * 2

// SDKSteps is steps for introduce using client SDK
type SDKSteps struct {
	bddContext      *context.BDDContext
	didExchangeSDKS *bddDIDExchange.SDKSteps
	outofbandSDKS   *bddOutOfBand.SDKSteps
	clients         map[string]*introduce.Client
	actions         map[string]chan service.DIDCommAction
	events          map[string]chan service.StateMsg
}

// NewIntroduceSDKSteps creates steps for introduce with SDK
func NewIntroduceSDKSteps() *SDKSteps {
	return &SDKSteps{
		clients: make(map[string]*introduce.Client),
		actions: make(map[string]chan service.DIDCommAction),
		events:  make(map[string]chan service.StateMsg),
	}
}

// SetContext is called before every scenario is run with a fresh new context
func (a *SDKSteps) SetContext(ctx *context.BDDContext) {
	a.bddContext = ctx
}

// RegisterSteps registers agent steps
func (a *SDKSteps) RegisterSteps(s *godog.Suite) {
	s.Step(`^"([^"]*)" sends introduce proposal to the "([^"]*)" and "([^"]*)"$`, a.sendProposal)
	s.Step(`^"([^"]*)" sends introduce proposal to the "([^"]*)" with "([^"]*)" out-of-band request$`,
		a.sendProposalWithInvitation)
	s.Step(`^"([^"]*)" sends introduce request to the "([^"]*)" asking about "([^"]*)"$`, a.sendRequest)
	s.Step(`^"([^"]*)" sends introduce proposal back to the "([^"]*)" and requested introduce$`, a.handleRequest)
	s.Step(`^"([^"]*)" sends introduce proposal back to the requester with public out-of-band request$`,
		a.handleRequestWithInvitation)
	s.Step(`^"([^"]*)" wants to know "([^"]*)" and sends introduce response with approve$`, a.checkAndContinue)
	s.Step(`^"([^"]*)" wants to know "([^"]*)" and sends introduce response with approve and provides an out-of-band request$`, //nolint:lll
		a.checkAndContinueWithInvitation)
	s.Step(`^"([^"]*)" wants to know "([^"]*)" and sends introduce response with approve and provides an out-of-band request with an embedded "([^"]*)"$`, //nolint:lll
		a.checkAndContinueWithInvitationAndEmbeddedRequest)
	s.Step(`^"([^"]*)" doesn't want to know "([^"]*)" and sends introduce response$`, a.checkAndStop)
	s.Step(`^"([^"]*)" stops the introduce protocol$`, a.stopProtocol)
	s.Step(`^"([^"]*)" checks the history of introduce protocol events "([^"]*)"$`, a.checkHistoryEvents)
	s.Step(`^"([^"]*)" checks the history of introduce protocol events "([^"]*)" and stop$`,
		a.checkHistoryEventsAndStop)
	s.Step(`^"([^"]*)" exchange DIDs with "([^"]*)"$`, a.createConnections)
	s.Step(`^"([^"]*)" has did exchange connection with "([^"]*)"$`, a.connectionEstablished)
	s.Step(`^"([^"]*)" confirms route registration with "([^"]*)"$`, a.confirmRouteRegistration)
}

func (a *SDKSteps) connectionEstablished(agent1, agent2 string) error {
	if err := a.didExchangeSDKS.ApproveRequest(agent1); err != nil {
		return err
	}

	if err := a.didExchangeSDKS.ApproveRequest(agent2); err != nil {
		return err
	}

	return a.didExchangeSDKS.WaitForPostEvent(agent2+","+agent1, "completed")
}

func (a *SDKSteps) createConnections(introducees, introducer string) error {
	const (
		inboundHost     = "localhost"
		inboundPort     = "random"
		endpointURL     = "${SIDETREE_URL}"
		acceptDidMethod = "sidetree"
	)

	participants := introducees + "," + introducer
	agentSDK := agent.NewSDKSteps()
	agentSDK.SetContext(a.bddContext)

	err := agentSDK.CreateAgentWithHTTPDIDResolver(participants, inboundHost, inboundPort, endpointURL, acceptDidMethod)
	if err != nil {
		return err
	}

	if err := didresolver.CreateDIDDocument(a.bddContext, participants, acceptDidMethod); err != nil {
		return err
	}

	a.didExchangeSDKS = bddDIDExchange.NewDIDExchangeSDKSteps()
	a.didExchangeSDKS.SetContext(a.bddContext)

	a.outofbandSDKS = bddOutOfBand.NewOutOfBandSDKSteps()
	a.outofbandSDKS.SetContext(a.bddContext)

	if err := a.didExchangeSDKS.WaitForPublicDID(participants, 10); err != nil {
		return err
	}

	if err := a.createExternalClients(participants); err != nil {
		return err
	}

	if err := a.outofbandSDKS.CreateRequestWithDID(introducer); err != nil {
		return err
	}

	for _, introducee := range strings.Split(introducees, ",") {
		if err := a.outofbandSDKS.ReceiveRequest(introducee, introducer); err != nil {
			return err
		}

		if err := a.connectionEstablished(introducee, introducer); err != nil {
			return err
		}
	}

	return a.createIntroduceClient(participants)
}

func (a *SDKSteps) createIntroduceClient(agents string) error {
	for _, agent := range strings.Split(agents, ",") {
		if err := a.createClient(agent); err != nil {
			return err
		}
	}

	return nil
}

func (a *SDKSteps) createClient(agentID string) error {
	const stateMsgChanSize = 10

	client, err := introduce.New(a.bddContext.AgentCtx[agentID])
	if err != nil {
		return err
	}

	if a.clients[agentID] != nil {
		if err := a.clients[agentID].UnregisterActionEvent(a.actions[agentID]); err != nil {
			return err
		}

		if err := a.clients[agentID].UnregisterMsgEvent(a.events[agentID]); err != nil {
			return err
		}
	}

	a.clients[agentID] = client
	a.actions[agentID] = make(chan service.DIDCommAction, 1)
	a.events[agentID] = make(chan service.StateMsg, stateMsgChanSize)

	if err := client.RegisterMsgEvent(a.events[agentID]); err != nil {
		return err
	}

	return client.RegisterActionEvent(a.actions[agentID])
}

func (a *SDKSteps) checkHistoryEvents(agentID, events string) error {
	for _, stateID := range strings.Split(events, ",") {
		select {
		case e := <-a.events[agentID]:
			if stateID != e.StateID {
				return fmt.Errorf("history of events doesn't meet the expectation %q != %q", stateID, e.StateID)
			}
		case <-time.After(timeout):
			return fmt.Errorf("waited for %s: history of events doesn't meet the expectation", stateID)
		}
	}

	return nil
}

func (a *SDKSteps) checkHistoryEventsAndStop(agentID, events string) error {
	for _, stateID := range strings.Split(events, ",") {
		select {
		case e := <-a.events[agentID]:
			if stateID != e.StateID {
				return fmt.Errorf("history of events doesn't meet the expectation %q != %q", stateID, e.StateID)
			}
		case <-time.After(timeout):
			return fmt.Errorf("waited for %s: history of events doesn't meet the expectation", stateID)
		}
	}

	return nil
}

func (a *SDKSteps) checkAndStop(agentID, introduceeID string) error {
	select {
	case e := <-a.actions[agentID]:
		proposal := &introduceService.Proposal{}
		if err := e.Message.Decode(proposal); err != nil {
			return err
		}

		if proposal.To.Name != introduceeID {
			return fmt.Errorf("%q doesn't want to know %q but got %q", agentID, introduceeID, proposal.To.Name)
		}

		e.Stop(errors.New("stop the protocol"))
	case <-time.After(timeout):
		return fmt.Errorf("timeout checkAndStop %s", agentID)
	}

	return nil
}

func (a *SDKSteps) handleRequest(agentID, introducee string) error {
	select {
	case e := <-a.actions[agentID]:
		request := &introduceService.Request{}
		if err := e.Message.Decode(request); err != nil {
			return err
		}

		conn, err := a.getConnection(agentID, request.PleaseIntroduceTo.Name)
		if err != nil {
			return err
		}

		recipient := &introduceService.Recipient{
			To:       &introduceService.To{Name: introducee},
			MyDID:    conn.MyDID,
			TheirDID: conn.TheirDID,
		}

		to := &introduceService.To{Name: request.PleaseIntroduceTo.Name}

		e.Continue(introduce.WithRecipients(to, recipient))
	case <-time.After(timeout):
		return fmt.Errorf("timeout checkAndContinue %s", agentID)
	}

	return nil
}

func (a *SDKSteps) handleRequestWithInvitation(agentID string) error {
	select {
	case e := <-a.actions[agentID]:
		request := &introduceService.Request{}
		if err := e.Message.Decode(request); err != nil {
			return err
		}

		introduceTo := request.PleaseIntroduceTo.Name

		req, err := a.newOOBRequest(introduceTo)
		if err != nil {
			return err
		}

		to := &introduceService.To{Name: req.Label}

		e.Continue(introduce.WithPublicOOBRequest(req, to))
	case <-time.After(timeout):
		return fmt.Errorf("timeout checkAndContinue %s", agentID)
	}

	return nil
}

func (a *SDKSteps) checkAndContinue(agentID, introduceeID string) error {
	select {
	case e := <-a.actions[agentID]:
		proposal := &introduceService.Proposal{}
		if err := e.Message.Decode(proposal); err != nil {
			return err
		}

		if proposal.To.Name != introduceeID {
			return fmt.Errorf("%q wants to know %q but got %q", agentID, introduceeID, proposal.To.Name)
		}

		e.Continue(nil)

		go a.outofbandSDKS.ApproveRequest(agentID)
	case <-time.After(timeout):
		return fmt.Errorf("timeout checkAndContinue %s", agentID)
	}

	return nil
}

func (a *SDKSteps) checkAndContinueWithInvitation(agentID, introduceeID string) error {
	select {
	case e := <-a.actions[agentID]:
		proposal := &introduceService.Proposal{}
		if err := e.Message.Decode(proposal); err != nil {
			return err
		}

		if proposal.To.Name != introduceeID {
			return fmt.Errorf("%q wants to know %q but got %q", agentID, introduceeID, proposal.To.Name)
		}

		req, err := a.newOOBRequest(agentID)
		if err != nil {
			return err
		}

		e.Continue(introduce.WithOOBRequest(req))

		go a.outofbandSDKS.ApproveRequest(agentID)
	case <-time.After(timeout):
		return fmt.Errorf("timeout checkAndContinue %s", agentID)
	}

	return nil
}

func (a *SDKSteps) checkAndContinueWithInvitationAndEmbeddedRequest(agentID, introduceeID, request string) error {
	select {
	case e := <-a.actions[agentID]:
		proposal := &introduceService.Proposal{}
		if err := e.Message.Decode(proposal); err != nil {
			return err
		}

		if proposal.To.Name != introduceeID {
			return fmt.Errorf("%q wants to know %q but got %q", agentID, introduceeID, proposal.To.Name)
		}

		req, err := a.newOOBRequest(agentID, request)
		if err != nil {
			return err
		}

		e.Continue(introduce.WithOOBRequest(req))

		go a.outofbandSDKS.ApproveRequest(agentID)
	case <-time.After(timeout):
		return fmt.Errorf("timeout checkAndContinue %s", agentID)
	}

	return nil
}

func (a *SDKSteps) stopProtocol(agentID string) error {
	select {
	case e := <-a.actions[agentID]:
		e.Stop(errors.New("stop the protocol"))
	case <-time.After(timeout):
		return fmt.Errorf("timeout stopProtocol %s", agentID)
	}

	return nil
}

func (a *SDKSteps) getConnection(agent1, agent2 string) (*didexchange.Connection, error) {
	connections, err := a.bddContext.DIDExchangeClients[agent1].QueryConnections(&didexchange.QueryConnectionsParams{})
	if err != nil {
		return nil, err
	}

	for i := range connections {
		if connections[i].TheirLabel == agent2 {
			return connections[i], nil
		}
	}

	return nil, errors.New("no connection between agents")
}

func (a *SDKSteps) sendProposal(introducer, introducee1, introducee2 string) error {
	conn1, err := a.getConnection(introducer, introducee1)
	if err != nil {
		return err
	}

	conn2, err := a.getConnection(introducer, introducee2)
	if err != nil {
		return err
	}

	return a.clients[introducer].SendProposal(&introduceService.Recipient{
		To:       &introduceService.To{Name: conn2.TheirLabel},
		MyDID:    conn1.MyDID,
		TheirDID: conn1.TheirDID,
	}, &introduceService.Recipient{
		To:       &introduceService.To{Name: conn1.TheirLabel},
		MyDID:    conn2.MyDID,
		TheirDID: conn2.TheirDID,
	})
}

func (a *SDKSteps) sendProposalWithInvitation(introducer, introducee1, introducee2 string) error {
	conn1, err := a.getConnection(introducer, introducee1)
	if err != nil {
		return err
	}

	req, err := a.newOOBRequest(introducee2)
	if err != nil {
		return err
	}

	return a.clients[introducer].SendProposalWithOOBRequest(req, &introduceService.Recipient{
		To:       &introduceService.To{Name: introducee2},
		MyDID:    conn1.MyDID,
		TheirDID: conn1.TheirDID,
	})
}

func (a *SDKSteps) sendRequest(introducee1, introducer, introducee2 string) error {
	conn1, err := a.getConnection(introducee1, introducer)
	if err != nil {
		return err
	}

	to := &introduceService.PleaseIntroduceTo{To: introduceService.To{Name: introducee2}}

	return a.clients[introducee1].SendRequest(to, conn1.MyDID, conn1.TheirDID)
}

func (a *SDKSteps) newOOBRequest(agentID string, requests ...interface{}) (*outofband.Request, error) {
	client, err := outofband.New(a.bddContext.AgentCtx[agentID])
	if err != nil {
		return nil, err
	}

	// default
	attachments := []*decorator.Attachment{{
		ID:          uuid.New().String(),
		Description: "test",
		Data: decorator.AttachmentData{
			JSON: map[string]interface{}{},
		},
	}}

	if len(requests) > 0 {
		// override
		attachments = []*decorator.Attachment{}

		for _, r := range requests {
			if r != "route-request" {
				return nil, fmt.Errorf("unsupported request type: %s", r)
			}

			bytes, er := json.Marshal(&route.Request{
				ID:   uuid.New().String(),
				Type: route.RequestMsgType,
			})
			if er != nil {
				return nil, er
			}

			attachments = append(attachments, &decorator.Attachment{
				ID:          uuid.New().String(),
				Description: "test",
				Data: decorator.AttachmentData{
					Base64: base64.StdEncoding.EncodeToString(bytes),
				},
			})
		}
	}

	r, err := client.CreateRequest(
		attachments,
		outofband.WithLabel(agentID),
	)
	if err != nil {
		return nil, err
	}

	return r, nil
}

//  creates clients for other protocols (eg. out-of-band, did-exchange)
func (a *SDKSteps) createExternalClients(participants string) error {
	if err := a.didExchangeSDKS.CreateDIDExchangeClient(participants); err != nil {
		return err
	}

	if err := a.outofbandSDKS.CreateClients(participants); err != nil {
		return err
	}

	if err := a.didExchangeSDKS.RegisterPostMsgEvent(participants, "completed"); err != nil {
		return err
	}

	return nil
}

func (a *SDKSteps) confirmRouteRegistration(agentID, router string) error {
	routeSteps := routeBDDSteps.NewRouteSDKSteps()
	routeSteps.SetContext(a.bddContext)

	go routeSteps.ApproveRequest(router, service.Empty{})

	expected, err := a.getConnection(agentID, router)
	if err != nil {
		return err
	}

	client, err := routeClient.New(a.bddContext.AgentCtx[agentID])
	if err != nil {
		return err
	}

	var result string

	err = errors.New("dummy")
	deadline := time.Now().Add(timeout)

	// TODO add protocol state msg event capability to routing service
	//  https://github.com/hyperledger/aries-framework-go/issues/1718
	for err != nil && time.Now().Before(deadline) {
		result, err = client.GetConnection()
		if err != nil {
			time.Sleep(250 * time.Millisecond) //nolint:gomnd
		}
	}

	if err != nil {
		return err
	}

	if expected.ConnectionID != result {
		return fmt.Errorf(
			"mismatch: %s has connectionID=%s with router %s but its routing ID is %s",
			agentID, expected.ConnectionID, router, result)
	}

	return nil
}
