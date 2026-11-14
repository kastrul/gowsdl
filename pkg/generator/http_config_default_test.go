package generator

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	gowsdlhttp "github.com/kastrul/gowsdl/pkg/http"
)

func TestGenerator_RemoteWSDL_DefaultsHTTPConfig(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir(filepath.Join("..", "..", "fixtures"))))
	defer server.Close()
	url := server.URL + "/stock.wsdl"

	var nilClientConfig *gowsdlhttp.HTTPClientConfig
	cases := map[string][]Option{
		"NoOption":               nil,
		"NilInterface":           {WithHTTPConfig(nil)},
		"NilHTTPClientConfigPtr": {WithHTTPConfig(nilClientConfig)},
	}
	for name, opts := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Contains(t, generate(t, url, opts...).types(), "TradePriceRequest")
		})
	}
}
