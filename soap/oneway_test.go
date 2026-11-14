// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package soap

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_CallContext_OneWay(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{
			name:   "202 Accepted with no body and no Content-Type",
			status: http.StatusAccepted,
		},
		{
			name:   "200 OK with no body and no Content-Type",
			status: http.StatusOK,
		},
		{
			name:   "empty SOAP body",
			status: http.StatusOK,
			body:   `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body/></soap:Envelope>`,
		},
		{
			name:   "SOAP body with an element is ignored",
			status: http.StatusOK,
			body:   `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><Ack xmlns="urn:x">ok</Ack></soap:Body></soap:Envelope>`,
		},
		{
			name:    "SOAP fault is still reported",
			status:  http.StatusInternalServerError,
			body:    `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><soap:Fault><faultcode>soap:Server</faultcode><faultstring>rejected</faultstring></soap:Fault></soap:Body></soap:Envelope>`,
			wantErr: "rejected",
		},
		{
			name:    "500 with no body is still an error",
			status:  http.StatusInternalServerError,
			wantErr: "500",
		},
		{
			name:    "HTTP error is still reported",
			status:  http.StatusBadRequest,
			body:    "bad request",
			wantErr: "400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var received string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data, _ := io.ReadAll(r.Body)
				received = string(data)
				if tt.body != "" {
					w.Header().Set("Content-Type", "text/xml; charset=utf-8")
				}
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()

			err := NewClient(server.URL).CallContext(context.Background(), "urn:notify", &Ping{Request: &PingRequest{Message: "hi"}}, nil)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Contains(t, received, "<Message>hi</Message>", "the request should still be sent")
		})
	}
}
