# Error Handling in gowsdl

## Overview

`gowsdl` uses Go 1.13+ error wrapping patterns to provide detailed context about errors. This allows consumers to understand what went wrong and where, and to handle specific error types programmatically.

## Error Types

### WSDLError

Represents errors related to WSDL processing.

```go
type WSDLError struct {
    Op   string // operation that failed
    Path string // file path or URL
    Err  error  // underlying error
}
```

`gowsdl.WSDLError` is an alias of `types.WSDLError`. Operations (`Op` values) the code uses:
- `"fetch"` - Failed to download or read the WSDL file
- `"detect_version"` - Failed to detect the WSDL version
- `"parse"` - Failed to parse the WSDL XML
- `"download"` - Failed to download a file over HTTP (`pkg/http`)

### SchemaError

Represents errors related to XSD schema processing.

```go
type SchemaError struct {
    Op     string // operation that failed
    Schema string // schema file or URL
    Err    error  // underlying error
}
```

`gowsdl.SchemaError` is an alias of `types.SchemaError`. The type is defined and the CLI checks for it, but nothing in the library currently returns one: the generator logs schema import failures (`[WARN] failed to fetch schema ...`) and carries on instead of returning them.

`core.ContextualError` (package `pkg/core`) is likewise defined but not constructed anywhere.

### ValidationError (CLI only)

Represents input validation errors in the command-line tool.

```go
type ValidationError struct {
    Field string // field that failed validation
    Value string // invalid value
    Err   error  // underlying error
}
```

`ValidationError` is defined in `cmd/gowsdl/main.go`, not in the library. The validations it reports:
- Empty output path (`"path cannot be empty"`)
- Empty package or file names (`"cannot be empty"`)
- Unsafe characters in package or file names (`"contains unsafe characters"`)
- Path traversal sequences (`..` or `./`) in package or file names (`"contains path traversal sequences"`)

## Error Handling Patterns

### Checking for Specific Error Types

Use `errors.As` to check for specific error types:

```go
import "github.com/kastrul/gowsdl"

g, err := gowsdl.NewGoWSDL(wsdlPath, gowsdl.WithPackage("myservice"))
if err != nil {
    var wsdlErr *gowsdl.WSDLError
    if errors.As(err, &wsdlErr) {
        log.Printf("WSDL error in operation %s for %s: %v", 
            wsdlErr.Op, wsdlErr.Path, wsdlErr.Err)
    }
    return err
}
```

### Checking for Underlying Errors

Use `errors.Is` to check for specific underlying errors:

```go
gocode, err := g.StartWithContext(ctx)
if err != nil {
    if errors.Is(err, io.EOF) {
        // Handle unexpected EOF
    }
    return err
}
```

### Error Context

All errors include context about what operation failed and what resource was being processed:

```
wsdl parse "http://example.com/service.wsdl": XML syntax error on line 42
wsdl fetch "http://example.com/service.wsdl": connection timeout
validation failed for package "../pkg": contains path traversal sequences
```

## Best Practices

1. **Always check errors**: Don't ignore returned errors
2. **Use error types for handling**: Use `errors.As` when you need to handle specific error conditions
3. **Log full error context**: The error messages include valuable debugging information
4. **Wrap errors in your code**: When building on top of gowsdl, wrap errors with additional context:
   ```go
   if err != nil {
       return fmt.Errorf("generating client for service %s: %w", serviceName, err)
   }
   ```

## Migration from Older Versions

If you're upgrading from an older version of gowsdl:

1. Replace direct error comparisons:
   ```go
   // Old
   if err == io.EOF {
   
   // New
   if errors.Is(err, io.EOF) {
   ```

2. Use error type assertions for better error handling:
   ```go
   // Old
   if err != nil {
       log.Fatal(err)
   }
   
   // New
   if err != nil {
       var wsdlErr *gowsdl.WSDLError
       if errors.As(err, &wsdlErr) {
           // Handle WSDL-specific error
       }
       return err
   }
   ```

## Goroutine Error Handling

The generator renders the types, operations and (optionally) server parts concurrently. If any part fails, `Generate` / `StartWithContext` returns no code and the error of the first failing part, in part order (types, operations, then the server parts). Errors from the other parts are not combined into it.