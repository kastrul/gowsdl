// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package http

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const proxyEnvHelperGuard = "GOWSDL_PROXY_ENV_HELPER"

var proxyEnvVars = []string{
	"HTTP_PROXY", "http_proxy",
	"HTTPS_PROXY", "https_proxy",
	"NO_PROXY", "no_proxy",
	"REQUEST_METHOD",
}

func builtTransport(t *testing.T, client *http.Client) *http.Transport {
	t.Helper()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.Transport)
	}
	return transport
}

func proxyFor(t *testing.T, transport *http.Transport, rawURL string) string {
	t.Helper()
	if transport.Proxy == nil {
		t.Fatal("expected transport.Proxy to be set")
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("proxy func returned error: %v", err)
	}
	if proxyURL == nil {
		return ""
	}
	return proxyURL.Host
}

func TestHTTPClientConfig_Build_DefaultsToEnvironmentProxy(t *testing.T) {
	config := DefaultHTTPClientConfig()
	transport := builtTransport(t, config.Build())

	assertUsesEnvironmentProxy(t, transport)
}

func TestConfig_Build_DefaultsToEnvironmentProxy(t *testing.T) {
	config := &Config{}
	transport := builtTransport(t, config.Build())

	assertUsesEnvironmentProxy(t, transport)
}

func TestHTTPClientConfig_Build_ExplicitProxyOverridesEnvironment(t *testing.T) {
	config := DefaultHTTPClientConfig()
	config.ProxyURL = "http://explicit.example:8080"
	transport := builtTransport(t, config.Build())

	if got := proxyFor(t, transport, "https://www.w3.org/2001/XMLSchema.xsd"); got != "explicit.example:8080" {
		t.Errorf("expected explicit proxy explicit.example:8080, got %q", got)
	}
}

func TestConfig_Build_ExplicitProxyOverridesEnvironment(t *testing.T) {
	config := &Config{ProxyURL: "http://explicit.example:8080"}
	transport := builtTransport(t, config.Build())

	if got := proxyFor(t, transport, "https://www.w3.org/2001/XMLSchema.xsd"); got != "explicit.example:8080" {
		t.Errorf("expected explicit proxy explicit.example:8080, got %q", got)
	}
}

// TestHTTPClientConfig_Build_HonorsProxyEnvVars runs the env check in a child
// process because http.ProxyFromEnvironment caches the environment on first use.
func TestHTTPClientConfig_Build_HonorsProxyEnvVars(t *testing.T) {
	env := make([]string, 0, len(os.Environ())+4)
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if !containsString(proxyEnvVars, name) {
			env = append(env, kv)
		}
	}
	env = append(env,
		proxyEnvHelperGuard+"=1",
		"HTTPS_PROXY=http://envproxy.example:3128",
		"HTTP_PROXY=http://envproxy.example:3129",
		"NO_PROXY=bypass.example",
	)

	cmd := exec.Command(os.Args[0], "-test.run=^TestProxyEnvHelper$", "-test.v")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("proxy env helper failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "PROXY_ENV_HELPER_OK") {
		t.Fatalf("proxy env helper did not run checks:\n%s", out)
	}
}

func TestProxyEnvHelper(t *testing.T) {
	if os.Getenv(proxyEnvHelperGuard) != "1" {
		t.Skip("helper process for TestHTTPClientConfig_Build_HonorsProxyEnvVars")
	}

	builds := map[string]*http.Client{
		"HTTPClientConfig": DefaultHTTPClientConfig().Build(),
		"Config":           (&Config{}).Build(),
	}
	for name, client := range builds {
		transport := builtTransport(t, client)

		cases := []struct {
			url  string
			want string
		}{
			{"https://www.w3.org/2001/XMLSchema.xsd", "envproxy.example:3128"},
			{"http://www.w3.org/2001/XMLSchema.xsd", "envproxy.example:3129"},
			{"https://bypass.example/service.wsdl", ""},
		}
		for _, tc := range cases {
			if got := proxyFor(t, transport, tc.url); got != tc.want {
				t.Errorf("%s: proxy for %s = %q, want %q", name, tc.url, got, tc.want)
			}
		}
	}

	if !t.Failed() {
		fmt.Println("PROXY_ENV_HELPER_OK")
	}
}

func assertUsesEnvironmentProxy(t *testing.T, transport *http.Transport) {
	t.Helper()
	if transport.Proxy == nil {
		t.Fatal("expected transport.Proxy to default to http.ProxyFromEnvironment, got nil")
	}
	got := reflect.ValueOf(transport.Proxy).Pointer()
	want := reflect.ValueOf(http.ProxyFromEnvironment).Pointer()
	if got != want {
		t.Error("expected transport.Proxy to be http.ProxyFromEnvironment")
	}
}

func containsString(list []string, s string) bool {
	return slices.Contains(list, s)
}
