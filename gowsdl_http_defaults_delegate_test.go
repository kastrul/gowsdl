package gowsdl

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pkghttp "github.com/kastrul/gowsdl/pkg/http"
)

func TestDefaultHTTPClientConfig_DelegatesToPkgHTTP(t *testing.T) {
	assert.Equal(t, pkghttp.DefaultHTTPClientConfig(), DefaultHTTPClientConfig())
}
