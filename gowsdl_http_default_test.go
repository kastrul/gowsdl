// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gowsdl

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoWSDL_RemoteWSDL_WithoutHTTPConfig(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("fixtures")))
	defer server.Close()
	url := server.URL + "/stock.wsdl"

	constructors := map[string]func() (*GoWSDL, error){
		"NewGoWSDL":                func() (*GoWSDL, error) { return NewGoWSDL(url) },
		"NewGoWSDL_NilHTTPConfig":  func() (*GoWSDL, error) { return NewGoWSDL(url, WithHTTPConfig(nil)) },
		"NewGoWSDLWithConfig_Nil":  func() (*GoWSDL, error) { return NewGoWSDLWithConfig(url, "stock", nil, true) },
		"NewGoWSDLWithOptions_Nil": func() (*GoWSDL, error) { return NewGoWSDLWithOptions(url, "stock", nil, true, false) },
	}
	for name, newGoWSDL := range constructors {
		t.Run(name, func(t *testing.T) {
			g, err := newGoWSDL()
			require.NoError(t, err)

			out, err := g.Start()
			require.NoError(t, err)
			assert.Contains(t, string(out["types"]), "TradePriceRequest")
		})
	}
}
