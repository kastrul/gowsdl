# WSDL to Go

[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/hooklift/gowsdl?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![GoDoc](https://godoc.org/github.com/kastrul/gowsdl?status.svg)](https://godoc.org/github.com/kastrul/gowsdl)
[![Build Status](https://github.com/kastrul/gowsdl/workflows/Go/badge.svg)](https://github.com/kastrul/gowsdl/actions)
[![codecov](https://codecov.io/gh/kastrul/gowsdl/branch/main/graph/badge.svg)](https://codecov.io/gh/kastrul/gowsdl)
[![Go Report Card](https://goreportcard.com/badge/github.com/kastrul/gowsdl)](https://goreportcard.com/report/github.com/kastrul/gowsdl)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

Generates idiomatic Go code from WSDL files with support for WSDL 1.1, WSDL 2.0, and modern Go features including generics.

### Requirements

* Go 1.27 or higher

### Install

* [Download release](https://github.com/kastrul/gowsdl/releases)
* Download and build locally
    * Go 1.27+: `go install github.com/kastrul/gowsdl/cmd/gowsdl@latest`
* Install from Homebrew: `brew install gowsdl`

### Goals
* Generate idiomatic Go code as much as possible
* Support only Document/Literal wrapped services, which are [WS-I](http://ws-i.org/) compliant
* Support:
	* WSDL 1.1 and WSDL 2.0
	* XML Schema 1.0
	* SOAP 1.1, and SOAP 1.2 for WSDL 1.1 port types bound only with SOAP 1.2 (without MTOM or MIME attachments)
* Resolve external XML Schemas
* Support external and local WSDL
* Full namespace handling and preservation
* XSD element reference resolution

### Caveats
* Please keep in mind that the generated code is just a reflection of what the WSDL is like. If your WSDL has duplicated type definitions, your Go code is going to have the same and may not compile.

### Usage
```
Usage: gowsdl [options] myservice.wsdl
  -d string
    	Directory under which package directory will be created (default "./")
  -generate-server
    	Generate server implementation code
  -generics-only
    	Generate only generic methods (returning soap.Result) for WSDL 1.1 document-style operations; implies -use-generics
  -http-proxy string
    	HTTP proxy URL
  -http-rate-limit int
    	Rate limit for HTTP requests per second (0 = no limit)
  -http-retries int
    	Number of retries for failed requests
  -http-retry-delay duration
    	Delay between retries (default 1s)
  -http-timeout duration
    	Timeout for HTTP requests (default 30s)
  -http-user-agent string
    	User-Agent header for requests (default "gowsdl/1.0")
  -i	Skips TLS Verification
  -make-public
    	Make the generated types public/exported (default true)
  -o string
    	File where the generated code will be saved (default "myservice.go")
  -operations string
    	Comma-separated list of operation names to generate client methods for (default: all)
  -p string
    	Package under which code will be generated (default "myservice")
  -prune-types
    	With -operations, generate only the types used by the selected operations (default true)
  -tls-ca-cert string
    	Path to CA certificate file
  -tls-client-cert string
    	Path to client certificate file (requires -tls-client-key)
  -tls-client-key string
    	Path to client key file (requires -tls-client-cert)
  -tls-min-version string
    	Minimum TLS version (1.0, 1.1, 1.2, 1.3) (default "1.2")
  -use-generics
    	Generate code using Go generics (requires Go 1.27+)
  -v	Shows gowsdl version
```

### Generating Only Selected Operations

Large WSDLs often expose far more operations than you need. Use `-operations` to generate client methods only for the operations you list:

```bash
gowsdl -operations "GetUser,UpdateUser" -p myservice -o myservice.go myservice.wsdl
```

* Names match the WSDL `operation` name exactly (case-sensitive).
* Port types / interfaces with none of the listed operations are left out of the client.
* Only the types those operations use are generated: request, response, fault and SOAP header types, plus everything they reference (nested elements, base types, attributes, enums, lists and unions). Pass `-prune-types=false` to keep every schema type.
* Server code (`-generate-server`) is not filtered. Because it covers every operation, type pruning is turned off when it is enabled.
* An unknown name fails generation and lists the available operations.

From Go, use the `WithOperations` option:

```go
g, err := gowsdl.NewGoWSDL("myservice.wsdl",
    gowsdl.WithPackage("myservice"),
    gowsdl.WithOperations("GetUser", "UpdateUser"),
    // gowsdl.WithTypePruning(false), // keep every schema type
)
```

### Go Generics Support

gowsdl can generate code that uses generics for type-safe SOAP operations. The generated code calls generic methods, so it needs Go 1.27 or newer. This feature provides:

* An extra `<Operation>Generic` method per operation, backed by `soap.Client.CallResult`
* A `soap.Result[T]` return value that separates successful responses from SOAP faults and transport errors
* Backward compatibility: the standard methods are generated unchanged, and nothing generic is generated without the flag

To enable generic code generation, use the `-use-generics` flag:

```bash
gowsdl -use-generics -p myservice -o myservice.go myservice.wsdl
```

Generic methods are generated for WSDL 1.1 document-style operations. RPC-style port types and WSDL 2.0 files get the standard methods only.

#### Example Usage

With generics enabled, you get an additional method for each operation. It shares the `*soap.Client` passed to the service constructor, so authentication, TLS, headers and timeouts apply to both:

```go
service := NewUserServicePortType(soap.NewClient(url, soap.WithBasicAuth("user", "pass")))

// Standard method (always generated)
response, err := service.GetUser(ctx, &GetUserRequest{UserId: 123})
if err != nil {
    // Handle error
}

// Generic method (only with -use-generics flag)
result, err := service.GetUserGeneric(ctx, &GetUserRequest{UserId: 123})
if err != nil {
    // Only returned for a nil request (soap.ErrNilRequest)
}

if result.IsSuccess() {
    user, _ := result.Unwrap()
    // Use user
} else if result.TransportError {
    // Network/transport error (connection refused, timeout, HTTP status, etc.).
    // result.Err is the original error, so errors.Is and errors.As work on it.
    if errors.Is(result.Err, context.DeadlineExceeded) {
        // Retry later
    }
    var httpErr *soap.HTTPError
    if errors.As(result.Err, &httpErr) {
        fmt.Printf("HTTP status: %d\n", httpErr.StatusCode)
    }
} else {
    // Business logic error from SOAP service
    fmt.Printf("SOAP Fault: %s\n", result.Fault.String)
}
```

The generic method provides better type safety and explicit fault handling, making it easier to distinguish between transport errors and business logic errors (SOAP faults). The error returned by `result.Unwrap()` also wraps the original transport error, so `errors.Is(err, context.DeadlineExceeded)` works on it too.

#### Generic-only Mode

Use `-generics-only` (or `WithGenericsOnly()` when using gowsdl as a library) to generate only the generic methods. They keep the plain operation name and replace the standard methods:

```bash
gowsdl -generics-only -p myservice -o myservice.go myservice.wsdl
```

```go
type UserServicePortType interface {
    GetUser(ctx context.Context, request *GetUserRequest) (soap.Result[GetUserResponse], error)
}
```

Operations that cannot use the generic client keep their standard methods: RPC-style port types, WSDL 2.0 files, and operations without a request or response type.

### xsd:anyType and xs:any Fields

Elements declared as `xsd:anyType` are generated as `soap.AnyType`. A received element keeps its content as self-contained XML, so you can decode it once you know its type:

```go
// Receiving
var item Item
if err := resp.Value.Decode(&item); err != nil {
    // Handle error
}
// resp.Value.Type() returns its resolved xsi:type, e.g. {Space: "urn:x", Local: "Item"},
// resp.Value.Attrs its attributes and resp.Value.InnerXML its content as XML.

// Sending: Value is encoded as the element's content
req.Value = soap.AnyType{Value: Item{ID: 7}}
```

An unset `soap.AnyType` field is left out of the request. `xsd:anySimpleType` is generated as `string`.

A type with an `xs:any` wildcard gets an `Items []soap.AnyElement` field holding every element the schema doesn't name. Each item works like `soap.AnyType`, and names itself when sent:

```go
for _, item := range resp.Items {
    if item.XMLName.Space == "urn:example:ext" && item.XMLName.Local == "flag" {
        var flag Flag
        _ = item.Decode(&flag)
    }
}

// Sending: named by XMLName, or else by Value's own XMLName or type name
req.Items = []soap.AnyElement{{Value: Flag{On: true}}}
```

### New Features

#### WSDL 2.0 Support

gowsdl now fully supports both WSDL 1.1 and WSDL 2.0 documents with complete code generation capabilities. Features include:

* **Automatic version detection** - WSDL version is detected from the root element namespace
* **Complete parsing support** for WSDL 2.0 structures:
  - Interface definitions (replacing portType from WSDL 1.1)
  - Endpoint definitions (replacing port from WSDL 1.1)
  - Modern message exchange patterns
  - Enhanced fault handling with input/output fault references
* **Full code generation** - Operations, types, and server code are generated using WSDL 2.0-specific templates
* **Backward compatibility** - All existing WSDL 1.1 functionality remains unchanged

Example usage with WSDL 2.0 files:
```bash
# Works seamlessly with both WSDL 1.1 and 2.0
gowsdl -p myservice -o myservice.go https://example.com/service.wsdl
```

#### Enhanced Namespace Handling

Namespaces are now properly preserved and managed throughout the code generation process:

* XML namespace prefixes are maintained in generated code
* Namespace collisions are automatically resolved
* Generated XML tags include proper namespace information
* Support for qualified and unqualified element/attribute forms

#### XSD Element Reference Resolution

The code generator now properly resolves XSD element references:

* Global element references are fully resolved
* Cross-schema element references are supported
* Properties from referenced elements are properly inherited
* Complex type references within elements are handled correctly

#### Improved Local Schema Resolution

When working with local WSDL files that reference external schemas:

* Relative schema paths are correctly resolved
* Local schema imports work seamlessly
* Recursive schema resolution is supported with cycle detection
* Mixed local and remote schema references are handled

### Implementation Status

This implementation addresses all the features requested in [issue #10](https://github.com/kastrul/gowsdl/issues/10), including the original TODOs:

- ✅ **Support for generating namespaces** - Complete namespace management system
- ✅ **Resolve XSD element references** - Full element reference resolution
- ✅ **Local schema resolution** - Enhanced local and remote schema handling
- ✅ **WSDL 2.0 Support** - Complete parsing and code generation
- ✅ **Better error messages** - Contextual errors with suggestions

All features include comprehensive test coverage and maintain backward compatibility with existing WSDL 1.1 code.

### Project Structure

The project has been modularized for better maintainability and follows Go standard project layout:

```
gowsdl/
├── cmd/gowsdl/        # Command-line interface
│   └── main.go        # CLI entry point with extensive flag support
├── pkg/               # Core library packages
│   ├── core/          # Core utilities and error handling
│   ├── generator/     # Code generation logic
│   │   └── templates/ # Go templates for code generation
│   ├── http/          # HTTP client with security features
│   ├── parser/        # WSDL/XSD parsers (1.1 and 2.0)
│   ├── testing/       # Fixture runner, request capture and XML comparison for testing generated clients
│   ├── types/         # Type mapping and conversion
│   └── utils/         # General utility functions
├── soap/              # SOAP runtime imported by generated code (client, envelope, faults, MTOM/MMA, XSD types)
├── gowsdl_facade.go   # Public gowsdl package: facade over pkg/generator (NewGoWSDL and options)
├── compat.go          # Aliases of pkg/ types and functions kept for older callers
├── fixtures/          # Test WSDL/XSD files
├── test_fixtures/     # WSDL, expected request and test data triples (document_literal/, rpc_literal/)
├── example/           # Usage examples
└── docs/              # Documentation
```

### Recent Improvements

#### Security Enhancements
- **Path traversal prevention** in file operations
- **Input validation** for package names and file paths
- **Configurable TLS settings** with custom CA certificates
- **Client certificate support** for mutual TLS
- **Secure defaults** for HTTPS connections

#### Modern Go Features
- **Context support** for cancellation and timeouts
- **Wrapped errors** with contextual information
- **Custom error types** for better error handling
- **Go 1.27+ compatibility** with latest Go idioms

#### HTTP Client Configuration
- Configurable timeouts and retries
- Rate limiting support
- HTTP/HTTPS proxy support
- Custom transport configuration
- Connection pooling

#### Testing and Quality
- **80%+ code coverage** target
- Unit tests throughout the codebase
- Integration tests for end-to-end scenarios
- Benchmark tests for performance monitoring
- GitHub Actions CI/CD pipeline

### Building and Testing

```bash
# Run all tests
go test -v -race ./...

# Run tests with coverage report
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -html=coverage.out -o coverage.html

# Run only the fast tests (skips those that build and run generated code with the go toolchain)
go test -short ./...

# Also run the tests that download from the public internet
GOWSDL_NETWORK_TESTS=1 go test -v ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run linter (install the version pinned in .github/workflows/go.yml: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0)
golangci-lint run

# Update dependencies
go get -u ./...
go mod tidy
```

### Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please ensure:
- All tests pass (`go test -v -race ./...`)
- Code coverage remains above 80% (`go test -coverprofile=coverage.out ./...`)
- Linter passes (`golangci-lint run`)
- New features include appropriate tests

### License

This project is licensed under the Mozilla Public License 2.0 - see the [LICENSE](LICENSE) file for details.
