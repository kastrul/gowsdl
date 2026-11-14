package soap

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func countGoroutinesIn(fn string) int {
	buf := make([]byte, 1<<20)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, 2*len(buf))
	}
	count := 0
	for _, g := range strings.Split(string(buf), "\n\n") {
		if strings.Contains(g, fn) {
			count++
		}
	}
	return count
}

func TestBatchClient_CancelledBatchDoesNotLeakGoroutines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	}))
	defer server.Close()

	batchClient := NewBatchClient[GetUserRequest, GetUserResponse](server.URL, "getUser")
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		batchClient.CallBatch(ctx, []GetUserRequest{{UserID: 1}, {UserID: 2}})
		cancel()
	}

	const batchFn = "soap.(*BatchClient[...]).CallBatch"
	assert.Eventually(t, func() bool { return countGoroutinesIn(batchFn) == 0 },
		2*time.Second, 10*time.Millisecond, "goroutines started by CallBatch are still running")
}
