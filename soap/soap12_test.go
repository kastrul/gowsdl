// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package soap

import (
	"context"
	"encoding/xml"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const soap12Response = `<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope"><env:Body>` +
	`<PingResponse xmlns="http://example.com/service.xsd"><PingResult><Message>pong</Message></PingResult></PingResponse>` +
	`</env:Body></env:Envelope>`

const soap12FaultResponse = `<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope"><env:Body><env:Fault>` +
	`<env:Code><env:Value>env:Sender</env:Value><env:Subcode><env:Value>m:BadInput</env:Value></env:Subcode></env:Code>` +
	`<env:Reason><env:Text xml:lang="en">bad input</env:Text></env:Reason>` +
	`<env:Role>urn:role</env:Role>` +
	`<env:Detail><Detail>detail message</Detail><Num>7.7</Num></env:Detail>` +
	`</env:Fault></env:Body></env:Envelope>`

// soap12Request records what a server received.
type soap12Request struct {
	contentType, soapAction string
	envelope                xml.Name
	hasSOAPAction           bool
}

func soap12Server(t *testing.T, status int, contentType, body string) (*httptest.Server, *soap12Request) {
	t.Helper()
	got := &soap12Request{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.contentType = r.Header.Get("Content-Type")
		_, got.hasSOAPAction = r.Header["Soapaction"]
		got.soapAction = r.Header.Get("SOAPAction")
		var env struct{ XMLName xml.Name }
		data, _ := io.ReadAll(r.Body)
		_ = xml.Unmarshal(data, &env)
		got.envelope = env.XMLName
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return server, got
}

// assertSOAP12ContentType checks a SOAP 1.2 request Content-Type: the
// application/soap+xml media type in UTF-8 with the given action
// parameter, or none when action is empty (RFC 3902).
func assertSOAP12ContentType(t *testing.T, contentType, action string) {
	t.Helper()
	mediaType, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err)
	assert.Equal(t, "application/soap+xml", mediaType)
	assert.Equal(t, "utf-8", params["charset"])
	gotAction, ok := params["action"]
	assert.Equal(t, action != "", ok, "action parameter present")
	assert.Equal(t, action, gotAction)
}

func TestClient_SOAP12_Request(t *testing.T) {
	tests := []struct {
		name   string
		opts   []Option
		ctx    func(context.Context) context.Context
		action string
		// wantAction is the action parameter of the Content-Type, absent
		// when empty.
		wantAction string
	}{
		{
			name:       "WithSOAP12 option",
			opts:       []Option{WithSOAP12()},
			action:     "urn:ping",
			wantAction: "urn:ping",
		},
		{
			name:       "context",
			ctx:        ContextWithSOAP12,
			action:     "urn:ping",
			wantAction: "urn:ping",
		},
		{
			name: "no action",
			opts: []Option{WithSOAP12()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, got := soap12Server(t, http.StatusOK, "application/soap+xml; charset=utf-8", soap12Response)
			ctx := context.Background()
			if tt.ctx != nil {
				ctx = tt.ctx(ctx)
			}

			var resp PingResponse
			err := NewClient(server.URL, tt.opts...).CallContext(ctx, tt.action, &Ping{Request: &PingRequest{Message: "hi"}}, &resp)
			require.NoError(t, err)

			assertSOAP12ContentType(t, got.contentType, tt.wantAction)
			assert.False(t, got.hasSOAPAction, "SOAP 1.2 carries the action in the Content-Type, not a SOAPAction header")
			assert.Equal(t, xml.Name{Space: XMLNsSoap12Env, Local: "Envelope"}, got.envelope)
			require.NotNil(t, resp.PingResult)
			assert.Equal(t, "pong", resp.PingResult.Message)
		})
	}
}

func TestClient_SOAP11_RequestUnchanged(t *testing.T) {
	server, got := soap12Server(t, http.StatusOK, "text/xml; charset=utf-8",
		`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><PingResponse xmlns="http://example.com/service.xsd"/></soap:Body></soap:Envelope>`)

	require.NoError(t, NewClient(server.URL).CallContext(context.Background(), "urn:ping", &Ping{}, &PingResponse{}))
	assert.Equal(t, `text/xml; charset="utf-8"`, got.contentType)
	assert.Equal(t, "urn:ping", got.soapAction)
	assert.Equal(t, xml.Name{Space: XmlNsSoapEnv, Local: "Envelope"}, got.envelope)
}

func TestClient_SOAP12_Fault(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server, _ := soap12Server(t, status, "application/soap+xml; charset=utf-8", soap12FaultResponse)

			detail := &SimpleNode{}
			err := NewClient(server.URL, WithSOAP12()).CallContextWithFaultDetail(context.Background(), "urn:ping", &Ping{}, &PingResponse{}, detail)

			var fault *SOAPFault
			require.ErrorAs(t, err, &fault)
			assert.Equal(t, "env:Sender", fault.Code)
			assert.Equal(t, "bad input", fault.String)
			assert.Equal(t, "urn:role", fault.Actor)
			assert.Equal(t, "detail message", detail.Detail)
			assert.InDelta(t, 7.7, detail.Num, 0.001)
		})
	}
}

func TestClient_SOAP12_FaultWithoutDetail(t *testing.T) {
	server, _ := soap12Server(t, http.StatusInternalServerError, "application/soap+xml",
		`<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope"><env:Body><env:Fault>`+
			`<env:Code><env:Value>env:Receiver</env:Value></env:Code><env:Reason><env:Text xml:lang="en">down</env:Text></env:Reason>`+
			`</env:Fault></env:Body></env:Envelope>`)

	err := NewClient(server.URL, WithSOAP12()).CallContext(context.Background(), "", &Ping{}, &PingResponse{})
	var fault *SOAPFault
	require.ErrorAs(t, err, &fault)
	assert.Equal(t, "env:Receiver", fault.Code)
	assert.Equal(t, "down", fault.Error())
}

func TestClient_SOAP12_HTTPErrorWithoutEnvelope(t *testing.T) {
	server, _ := soap12Server(t, http.StatusBadRequest, "text/plain", "bad request")

	err := NewClient(server.URL, WithSOAP12()).CallContext(context.Background(), "", &Ping{}, &PingResponse{})
	var httpErr *HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusBadRequest, httpErr.StatusCode)
	assert.Equal(t, "bad request", string(httpErr.ResponseBody))
}

func TestClient_SOAP12_OneWay(t *testing.T) {
	server, got := soap12Server(t, http.StatusAccepted, "", "")

	require.NoError(t, NewClient(server.URL, WithSOAP12()).CallContext(context.Background(), "urn:notify", &Ping{}, nil))
	assertSOAP12ContentType(t, got.contentType, "urn:notify")
}

func TestClient_SOAP12_AttachmentsUnsupported(t *testing.T) {
	for name, opt := range map[string]Option{"MTOM": WithMTOM(), "MIME multipart": WithMIMEMultipartAttachments()} {
		t.Run(name, func(t *testing.T) {
			server, _ := soap12Server(t, http.StatusOK, "", "")
			err := NewClient(server.URL, WithSOAP12(), opt).CallContext(context.Background(), "", &Ping{}, &PingResponse{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "SOAP 1.2")
		})
	}
}

func TestSOAPEnvelopeResponse_RejectsOtherNamespaces(t *testing.T) {
	env := SOAPEnvelopeResponse{Body: SOAPBodyResponse{Body: &PingResponse{}}}
	err := xml.Unmarshal([]byte(`<Envelope xmlns="urn:other"><Body/></Envelope>`), &env)
	require.Error(t, err)
}

func TestClient_SOAP12_BadRequestWithoutFault(t *testing.T) {
	server, _ := soap12Server(t, http.StatusBadRequest, "application/soap+xml", soap12Response)

	err := NewClient(server.URL, WithSOAP12()).CallContext(context.Background(), "", &Ping{}, &PingResponse{})
	var httpErr *HTTPError
	require.ErrorAs(t, err, &httpErr, "a 400 reply that holds no fault is still an error")
	assert.Equal(t, http.StatusBadRequest, httpErr.StatusCode)
}
