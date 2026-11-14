// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gowsdl

import (
	"context"
	"errors"
	"strings"

	"github.com/kastrul/gowsdl/pkg/generator"
	"github.com/kastrul/gowsdl/pkg/utils"
)

// GoWSDL is the main facade for WSDL code generation
type GoWSDL struct {
	generator *generator.Generator
}

// Option defines a function that configures GoWSDL
type Option func(*GoWSDL) error

// WithHTTPConfig sets the HTTP client configuration
func WithHTTPConfig(config *HTTPClientConfig) Option {
	return func(g *GoWSDL) error {
		return generator.WithHTTPConfig(config)(g.generator)
	}
}

// WithGenerics enables Go generics code generation
func WithGenerics() Option {
	return func(g *GoWSDL) error {
		return generator.WithGenerics()(g.generator)
	}
}

// WithGenericsOnly enables Go generics code generation and replaces the standard
// methods of WSDL 1.1 document-style operations with generic ones returning soap.Result
func WithGenericsOnly() Option {
	return func(g *GoWSDL) error {
		return generator.WithGenericsOnly()(g.generator)
	}
}

// WithExportAllTypes enables exporting all types (making them public)
func WithExportAllTypes(export bool) Option {
	return func(g *GoWSDL) error {
		return generator.WithExportAllTypes(export)(g.generator)
	}
}

// WithPackage sets the package name for generated code
func WithPackage(pkg string) Option {
	return func(g *GoWSDL) error {
		return generator.WithPackage(pkg)(g.generator)
	}
}

// WithServerGeneration enables server code generation
func WithServerGeneration(generate bool) Option {
	return func(g *GoWSDL) error {
		return generator.WithServerGeneration(generate)(g.generator)
	}
}

// WithOperations restricts client code generation to the named WSDL operations.
// An empty list generates all operations.
func WithOperations(names ...string) Option {
	return func(g *GoWSDL) error {
		return generator.WithOperations(names...)(g.generator)
	}
}

// WithTypePruning controls whether, when WithOperations is used, only the
// types needed by the selected operations are generated. Enabled by default.
func WithTypePruning(enabled bool) Option {
	return func(g *GoWSDL) error {
		return generator.WithTypePruning(enabled)(g.generator)
	}
}

// ParseOperationList splits a comma-separated list of operation names,
// trimming whitespace and dropping empty entries.
func ParseOperationList(list string) []string {
	return generator.ParseOperationList(list)
}

// NewGoWSDL creates a new GoWSDL instance with the provided options
func NewGoWSDL(file string, opts ...Option) (*GoWSDL, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return nil, errors.New("WSDL file is required to generate Go proxy")
	}

	// Convert options to generator options
	genOpts := make([]generator.Option, 0, len(opts))

	// Create generator
	gen, err := generator.New(file, genOpts...)
	if err != nil {
		return nil, err
	}

	g := &GoWSDL{
		generator: gen,
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(g); err != nil {
			return nil, err
		}
	}

	return g, nil
}

// NewGoWSDLWithConfig creates a GoWSDL with server generation enabled. It is
// kept for backward compatibility; new code should call NewGoWSDL with options.
func NewGoWSDLWithConfig(file, pkg string, httpConfig *HTTPClientConfig, exportAllTypes bool) (*GoWSDL, error) {
	opts := []Option{
		WithPackage(pkg),
		WithHTTPConfig(httpConfig),
		WithExportAllTypes(exportAllTypes),
		WithServerGeneration(true), // Default to true for backward compatibility
	}
	return NewGoWSDL(file, opts...)
}

// NewGoWSDLWithOptions creates a GoWSDL with server generation enabled and,
// if useGenerics is true, generic client operations. It is kept for backward
// compatibility; new code should call NewGoWSDL with options.
func NewGoWSDLWithOptions(file, pkg string, httpConfig *HTTPClientConfig, exportAllTypes bool, useGenerics bool) (*GoWSDL, error) {
	opts := []Option{
		WithPackage(pkg),
		WithHTTPConfig(httpConfig),
		WithExportAllTypes(exportAllTypes),
		WithServerGeneration(true), // Default to true for backward compatibility
	}
	if useGenerics {
		opts = append(opts, WithGenerics())
	}
	return NewGoWSDL(file, opts...)
}

// StartWithContext initiates the code generation process with context support
func (g *GoWSDL) StartWithContext(ctx context.Context) (map[string][]byte, error) {
	return g.generator.Generate(ctx)
}

// Start initiates the code generation process
func (g *GoWSDL) Start() (map[string][]byte, error) {
	return g.StartWithContext(context.Background())
}

// MakePublic is exported for backward compatibility. It converts the first
// letter of a string to uppercase.
var (
	MakePublic = utils.MakePublic
)
