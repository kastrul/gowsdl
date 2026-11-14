// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package soap

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errorTypesPing struct {
	XMLName struct{} `xml:"ping"`
}

type errorTypesPong struct {
	XMLName struct{} `xml:"pong"`
}

const errorTypesFault = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <soap:Fault>
      <faultcode>soap:Server</faultcode>
      <faultstring>boom</faultstring>
    </soap:Fault>
  </soap:Body>
</soap:Envelope>`

// TestClientCall_ErrorTypesAreNotWrapped pins the concrete error types that
// callers of Client.Call may type-assert on: they are returned as is, not
// wrapped, so both a direct assertion and errors.As succeed.
func TestClientCall_ErrorTypesAreNotWrapped(t *testing.T) {
	t.Run("SOAP fault", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/xml")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(errorTypesFault))
		}))
		t.Cleanup(srv.Close)

		err := NewClient(srv.URL).Call("ping", &errorTypesPing{}, &errorTypesPong{})

		fault, ok := err.(*SOAPFault) //nolint:errorlint // pins that the error is returned unwrapped
		require.True(t, ok, "direct type assertion on %T", err)
		assert.Equal(t, "boom", fault.String)
		target, ok := errors.AsType[*SOAPFault](err)
		require.True(t, ok)
		assert.Same(t, fault, target)
	})

	t.Run("HTTP error status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("missing"))
		}))
		t.Cleanup(srv.Close)

		err := NewClient(srv.URL).Call("ping", &errorTypesPing{}, &errorTypesPong{})

		httpErr, ok := err.(*HTTPError) //nolint:errorlint // pins that the error is returned unwrapped
		require.True(t, ok, "direct type assertion on %T", err)
		assert.Equal(t, http.StatusNotFound, httpErr.StatusCode)
		assert.Equal(t, "missing", string(httpErr.ResponseBody))
		target, ok := errors.AsType[*HTTPError](err)
		require.True(t, ok)
		assert.Same(t, httpErr, target)
	})

	t.Run("transport failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		addr := srv.URL
		srv.Close()

		err := NewClient(addr).Call("ping", &errorTypesPing{}, &errorTypesPong{})

		urlErr, ok := err.(*url.Error) //nolint:errorlint // pins that the error is returned unwrapped
		require.True(t, ok, "direct type assertion on %T", err)
		target, ok := errors.AsType[*url.Error](err)
		require.True(t, ok)
		assert.Same(t, urlErr, target)
	})
}
