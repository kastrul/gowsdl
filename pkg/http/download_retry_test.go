// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadFile_RetrySucceedsAfterTransportError(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
