package http

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Build_MalformedProxyURLIsIgnored(t *testing.T) {
	transport := builtTransport(t, (&Config{ProxyURL: "http://bad host:1"}).Build())

	assert.Empty(t, proxyFor(t, transport, "http://example.com"))
}

func TestHTTPClientConfig_Build_MalformedProxyURLPanics(t *testing.T) {
	config := &HTTPClientConfig{ProxyURL: "http://bad host:1"}

	assert.Panics(t, func() { config.Build() })
}

func TestConfig_Build_IgnoresRateLimit(t *testing.T) {
	client := (&Config{RateLimitPerSecond: 5}).Build()

	_, ok := client.Transport.(*http.Transport)
	assert.True(t, ok, "Config.Build must not wrap the transport in a rate limiter")
}

func TestHTTPClientConfig_Build_AppliesRateLimit(t *testing.T) {
	client := (&HTTPClientConfig{RateLimitPerSecond: 5}).Build()

	_, ok := client.Transport.(*http.Transport)
	assert.False(t, ok, "HTTPClientConfig.Build must wrap the transport in a rate limiter")
}

func TestBuild_BothConfigsShareTransportSetup(t *testing.T) {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	fromConfig := builtTransport(t, (&Config{TLSConfig: tlsConfig, InsecureSkipVerify: true}).Build())
	require.True(t, tlsConfig.InsecureSkipVerify)

	fromHTTPClientConfig := builtTransport(t, (&HTTPClientConfig{TLSConfig: tlsConfig}).Build())

	assert.Same(t, fromConfig.TLSClientConfig, fromHTTPClientConfig.TLSClientConfig)
	assert.Equal(t, fromConfig.MaxIdleConns, fromHTTPClientConfig.MaxIdleConns)
	assert.Equal(t, fromConfig.IdleConnTimeout, fromHTTPClientConfig.IdleConnTimeout)
	assert.Equal(t, fromConfig.TLSHandshakeTimeout, fromHTTPClientConfig.TLSHandshakeTimeout)
	assert.Equal(t, fromConfig.ExpectContinueTimeout, fromHTTPClientConfig.ExpectContinueTimeout)
}
