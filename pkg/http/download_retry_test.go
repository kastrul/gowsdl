// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadFile_RetrySucceedsAfterTransportError(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			conn, _, err := w.(http.Hijacker).Hijack()
			require.NoError(t, err)
			_ = conn.Close()
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	config := DefaultHTTPClientConfig().WithRetries(2, time.Millisecond)
	data, err := DownloadFile(context.Background(), server.URL, config)
	require.NoError(t, err)
	assert.Equal(t, "ok", string(data))
	assert.Equal(t, int32(2), attempts.Load())
}

func TestDownloadFile_RetryDelayHonoursContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	config := DefaultHTTPClientConfig().WithRetries(3, 10*time.Second)
	start := time.Now()
	_, err := DownloadFile(ctx, server.URL, config)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 2*time.Second)
}

type closeTrackingBody struct {
	io.Reader
	closed atomic.Bool
}

func (b *closeTrackingBody) Close() error {
	b.closed.Store(true)
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type stubHTTPConfig struct{ client *http.Client }

func (c stubHTTPConfig) Build() *http.Client          { return c.client }
func (c stubHTTPConfig) GetUserAgent() string         { return "" }
func (c stubHTTPConfig) GetMaxRetries() int           { return 0 }
func (c stubHTTPConfig) GetRetryDelay() time.Duration { return 0 }
func (c stubHTTPConfig) GetMaxResponseSize() int64    { return 1 << 20 }

func TestDownloadFile_ClosesBodyWhenContextCancelledAfterResponse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	body := &closeTrackingBody{Reader: strings.NewReader("ok")}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		cancel()
		return &http.Response{StatusCode: http.StatusOK, Body: body, Header: http.Header{}, Request: r}, nil
	})}

	_, err := DownloadFile(ctx, "http://example.invalid/x.wsdl", stubHTTPConfig{client: client})
	require.ErrorIs(t, err, context.Canceled)
	assert.True(t, body.closed.Load(), "response body must be closed")
}
