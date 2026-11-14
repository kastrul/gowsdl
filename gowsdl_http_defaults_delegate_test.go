package gowsdl

import (
	"testing"

	"github.com/stretchr/testify/assert"

	gowsdlhttp "github.com/kastrul/gowsdl/pkg/http"
)

func TestDefaultHTTPClientConfig_DelegatesToPkgHTTP(t *testing.T) {
	assert.Equal(t, gowsdlhttp.DefaultHTTPClientConfig(), DefaultHTTPClientConfig())
}
