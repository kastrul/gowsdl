// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gowsdl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithOptions(t *testing.T) {
	tests := []struct {
		name   string
		option Option
	}{
		{
			name: "WithHTTPConfig",
			option: WithHTTPConfig(&HTTPClientConfig{
				Timeout: 30,
			}),
		},
		{
			name:   "WithGenerics",
			option: WithGenerics(),
		},
		{
			name:   "WithGenericsOnly",
			option: WithGenericsOnly(),
		},
		{
			name:   "WithExportAllTypes",
			option: WithExportAllTypes(true),
		},
		{
			name:   "WithPackage",
			option: WithPackage("testpkg"),
		},
		{
			name:   "WithServerGeneration",
			option: WithServerGeneration(true),
		},
		{
			name:   "WithOperations",
			option: WithOperations("TestOperation"),
		},
		{
			name:   "WithTypePruning",
			option: WithTypePruning(false),
		},
	}

	// Create a temporary WSDL file for testing
	tempFile := minimalServiceWSDL

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gowsdl, err := NewGoWSDL(tempFile, tt.option)
			require.NoError(t, err)
			assert.NotNil(t, gowsdl)
			assert.NotNil(t, gowsdl.generator)
		})
	}
}

func TestNewGoWSDL(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		opts    []Option
		wantErr bool
	}{
		{
			name: "valid_wsdl_file",
			file: minimalServiceWSDL,
			opts: []Option{WithPackage("test")},
		},
		{
			name:    "empty_file_path",
			file:    "",
			wantErr: true,
		},
		{
			name:    "whitespace_file_path",
			file:    "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gowsdl, err := NewGoWSDL(tt.file, tt.opts...)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, gowsdl)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, gowsdl)
				assert.NotNil(t, gowsdl.generator)
			}
		})
	}
}

func TestNewGoWSDLWithConfig(t *testing.T) {
	tempFile := minimalServiceWSDL

	httpConfig := &HTTPClientConfig{
		Timeout: 30,
	}

	gowsdl, err := NewGoWSDLWithConfig(tempFile, "testpkg", httpConfig, true)
	require.NoError(t, err)
	assert.NotNil(t, gowsdl)
	assert.NotNil(t, gowsdl.generator)
}

func TestNewGoWSDLWithOptions(t *testing.T) {
	tempFile := minimalServiceWSDL

	httpConfig := &HTTPClientConfig{
		Timeout: 30,
	}

	tests := []struct {
		name        string
		useGenerics bool
	}{
		{
			name:        "with_generics",
			useGenerics: true,
		},
		{
			name:        "without_generics",
			useGenerics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gowsdl, err := NewGoWSDLWithOptions(tempFile, "testpkg", httpConfig, true, tt.useGenerics)
			require.NoError(t, err)
			assert.NotNil(t, gowsdl)
			assert.NotNil(t, gowsdl.generator)
		})
	}
}

func TestGoWSDLStart(t *testing.T) {
	tempFile := minimalServiceWSDL

	gowsdl, err := NewGoWSDL(tempFile, WithPackage("test"))
	require.NoError(t, err)

	code, err := gowsdl.Start()
	require.NoError(t, err)
	assert.NotNil(t, code)
	assert.Contains(t, code, "header")
	assert.Contains(t, code, "types")
	assert.Contains(t, code, "operations")
}

func TestGoWSDLStartWithContext(t *testing.T) {
	tempFile := minimalServiceWSDL

	gowsdl, err := NewGoWSDL(tempFile, WithPackage("test"))
	require.NoError(t, err)

	ctx := context.Background()
	code, err := gowsdl.StartWithContext(ctx)
	require.NoError(t, err)
	assert.NotNil(t, code)
	assert.Contains(t, code, "header")
	assert.Contains(t, code, "types")
	assert.Contains(t, code, "operations")
}

func TestGoWSDLWithServerGeneration(t *testing.T) {
	tempFile := minimalServiceWSDL

	gowsdl, err := NewGoWSDL(tempFile, WithPackage("test"), WithServerGeneration(true))
	require.NoError(t, err)

	code, err := gowsdl.Start()
	require.NoError(t, err)
	assert.NotNil(t, code)
	assert.Contains(t, code, "header")
	assert.Contains(t, code, "types")
	assert.Contains(t, code, "operations")
	assert.Contains(t, code, "server")
	assert.Contains(t, code, "server_header")
	assert.Contains(t, code, "server_wsdl")
}

func TestParseOperationList(t *testing.T) {
	assert.Equal(t, []string{"A", "B"}, ParseOperationList(" A, ,B "))
	assert.Nil(t, ParseOperationList(""))
}

// minimalServiceWSDL is a one-operation document/literal WSDL shared with the
// pkg/generator tests.
const minimalServiceWSDL = "pkg/generator/testdata/minimal_service.wsdl"
