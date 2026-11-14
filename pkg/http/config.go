// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package http

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

// Config holds configuration for the HTTP client.
//
// Deprecated: use HTTPClientConfig, which has the same fields and builds its
// client from the same transport setup. Config is kept for backward
// compatibility: unlike HTTPClientConfig.Build, its Build ignores
// RateLimitPerSecond and silently ignores a malformed ProxyURL.
type Config struct {
	Timeout            time.Duration
	MaxRetries         int
	RetryDelay         time.Duration
	RateLimitPerSecond int
	MaxResponseSize    int64
	UserAgent          string
	TLSConfig          *tls.Config
	InsecureSkipVerify bool
	ProxyURL           string
}

// Build creates an HTTP client from the configuration.
//
// Deprecated: use HTTPClientConfig.Build.
func (c *Config) Build() *http.Client {
	var proxy func(*http.Request) (*url.URL, error)
	if c.ProxyURL != "" {
		proxyURL, _ := url.Parse(c.ProxyURL)
		proxy = http.ProxyURL(proxyURL)
	}

	return &http.Client{
		Transport: newTransport(c.TLSConfig, c.InsecureSkipVerify, proxy),
		Timeout:   c.Timeout,
	}
}
