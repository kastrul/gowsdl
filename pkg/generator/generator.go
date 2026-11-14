// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"context"
	"log"
	"strings"
	"sync"

	"github.com/kastrul/gowsdl/pkg/http"
	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/types"
)

// Generator handles WSDL code generation
type Generator struct {
	loc             *http.Location
	pkg             string
	wsdl            *parser.WSDL
	wsdl2           *parser.WSDL2
	wsdlVersion     string // "1.1" or "2.0"
	httpConfig      http.HTTPConfig
	useGenerics     bool
	genericsOnly    bool
	generateServer  bool
	operations      map[string]bool
	keepUnusedTypes bool
	typeMapper      *types.TypeMapper
	refNamespaces   map[*parser.XSDElement]string
}

// Option defines a function that configures Generator
type Option func(*Generator) error

// WithPackage sets the package name
func WithPackage(pkg string) Option {
	return func(g *Generator) error {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			pkg = "myservice"
		}
		g.pkg = pkg
		return nil
	}
}

// WithHTTPConfig sets the HTTP configuration. A nil config, including a nil
// *http.HTTPClientConfig, selects http.DefaultHTTPClientConfig.
func WithHTTPConfig(config http.HTTPConfig) Option {
	return func(g *Generator) error {
		if c, ok := config.(*http.HTTPClientConfig); config == nil || (ok && c == nil) {
			config = http.DefaultHTTPClientConfig()
		}
		g.httpConfig = config
		return nil
	}
}

// WithGenerics enables generic code generation
func WithGenerics() Option {
	return func(g *Generator) error {
		g.useGenerics = true
		return nil
	}
}

// WithGenericsOnly enables generic code generation and replaces the standard
// methods of WSDL 1.1 document-style operations with generic ones that keep the
// plain operation name and return soap.Result. Operations that cannot use the
// generic client keep their standard methods.
func WithGenericsOnly() Option {
	return func(g *Generator) error {
		g.useGenerics = true
		g.genericsOnly = true
		return nil
	}
}

// WithExportAllTypes is accepted for API compatibility. It currently has no
// effect on generated output.
func WithExportAllTypes(_ bool) Option {
	return func(_ *Generator) error {
		return nil
	}
}

// WithServerGeneration controls whether to generate server code
func WithServerGeneration(generate bool) Option {
	return func(g *Generator) error {
		g.generateServer = generate
		return nil
	}
}

// New creates a new Generator instance
func New(file string, opts ...Option) (*Generator, error) {
	loc, err := http.ParseLocation(file)
	if err != nil {
		return nil, err
	}

	g := &Generator{
		loc:        loc,
		pkg:        "myservice",
		httpConfig: http.DefaultHTTPClientConfig(),
		typeMapper: types.NewTypeMapper(),
	}

	for _, opt := range opts {
		if err := opt(g); err != nil {
			return nil, err
		}
	}

	return g, nil
}

// Generate initiates the code generation process
func (g *Generator) Generate(ctx context.Context) (map[string][]byte, error) {
	if err := g.unmarshal(ctx); err != nil {
		return nil, err
	}

	if err := g.validateOperationFilter(); err != nil {
		return nil, err
	}
	if g.wsdl != nil {
		httpOnly := g.httpOnlyPortTypes()
		var skipped []string
		for _, pt := range g.wsdl.PortTypes {
			if httpOnly[pt.Name] {
				skipped = append(skipped, pt.Name)
			}
		}
		if len(skipped) > 0 {
			log.Printf("[INFO] skipping port type(s) %s: bound only via HTTP GET/POST, not SOAP", strings.Join(skipped, ", "))
		}
	}
	if len(g.operations) > 0 && !g.keepUnusedTypes && g.generateServer {
		log.Printf("[WARN] type pruning is disabled because server code needs the types of every operation")
	}

	// Process WSDL nodes
	schemas := g.getSchemas()

	for _, schema := range schemas {
		parser.NewTraverser(schema, schemas).Traverse()
	}

	return g.generateCode(ctx)
}

func (g *Generator) generateCode(ctx context.Context) (map[string][]byte, error) {
	gocode := make(map[string][]byte)

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Calculate number of goroutines based on generateServer flag. The
	// header is generated afterwards because its imports depend on the
	// types and operations code.
	numGoroutines := 2 // types, operations
	if g.generateServer {
		numGoroutines = 5 // add server, server_header, server_wsdl
	}
	errChan := make(chan error, numGoroutines)

	// Generate all components in parallel
	wg.Add(numGoroutines)

	go func() {
		defer wg.Done()
		if data, err := g.genTypes(); err != nil {
			errChan <- err
		} else {
			mu.Lock()
			gocode["types"] = data
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		if data, err := g.genOperations(); err != nil {
			errChan <- err
		} else {
			mu.Lock()
			gocode["operations"] = data
			mu.Unlock()
		}
	}()

	// Only generate server files if generateServer is enabled
	if g.generateServer {
		go func() {
			defer wg.Done()
			if data, err := g.genServer(); err != nil {
				errChan <- err
			} else {
				mu.Lock()
				gocode["server"] = data
				mu.Unlock()
			}
		}()

		go func() {
			defer wg.Done()
			if data, err := g.genServerHeader(); err != nil {
				errChan <- err
			} else {
				mu.Lock()
				gocode["server_header"] = data
				mu.Unlock()
			}
		}()

		go func() {
			defer wg.Done()
			if data, err := g.genServerWSDL(ctx); err != nil {
				errChan <- err
			} else {
				mu.Lock()
				gocode["server_wsdl"] = data
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		return nil, err
	}

	header, err := g.genHeader(gocode["types"], gocode["operations"])
	if err != nil {
		return nil, err
	}
	gocode["header"] = header

	return gocode, nil
}
