# SOAP Request Testing Framework

This package provides comprehensive testing for SOAP request generation from WSDL files. It validates that generated Go client code produces correct SOAP XML requests that comply with SOAP standards and match expected behavior.

## Problem Statement

The original testing approach only validated generated Go code structure but didn't test the actual SOAP XML requests sent over the wire. This led to issues like:

- ❌ Incorrect namespace handling (e.g., `<tns:operation>` instead of `<operation xmlns="...">`)
- ❌ Missing operation wrappers in RPC-style requests
- ❌ Malformed SOAP envelopes
- ❌ SOAP protocol compliance violations

## Solution

This framework provides:

1. **Fixture-driven testing** - Test cases with WSDL + expected XML + test data
2. **SOAP request capture** - Intercepts actual HTTP requests to capture XML
3. **Semantic XML comparison** - Compares XML meaning, not just strings
4. **Rule-based validation** - Validates against SOAP/WSDL compliance rules
5. **Comprehensive reporting** - Clear error messages and suggestions

## Architecture

```
test_fixtures/
├── document_literal/
│   ├── simple.wsdl              # WSDL file
│   ├── simple_request.xml       # Expected SOAP request
│   └── simple_test_data.json    # Test input data
├── rpc_literal/
│   ├── basic_rpc.wsdl
│   ├── basic_rpc_request.xml
│   └── basic_rpc_test_data.json
└── ...
```

`test_fixtures/` lives at the repository root (paths given to `LoadFixtures` are relative to the test's working directory, so tests in this package use `../../test_fixtures`). Each subdirectory holds one style (`document_literal` or `rpc_literal`); a fixture is `<name>.wsdl`, `<name>_request.xml` and `<name>_test_data.json`. `LoadFixtures` reads only those two subdirectories.

## Components

### SOAPCapture
Intercepts HTTP requests to capture actual SOAP XML:

```go
capture := NewSOAPCapture()
client := &http.Client{Transport: capture}

// Make SOAP request...
request := capture.GetLastRequest()
fmt.Println(request.Body) // Actual SOAP XML
```

### XMLComparator
Provides semantic XML comparison. Elements and attributes are compared by
namespace URI and local name, with prefixes resolved against declarations
anywhere in the document (including the SOAP Envelope); namespace
declarations themselves are not compared. So `<tns:op>` with `xmlns:tns` on
the Envelope equals `<op xmlns="...">`, while an element in no namespace
still differs from one that inherits a default namespace.

```go
comparator := NewXMLComparator()
result, err := comparator.CompareSOAPRequests(expected, actual)

if !result.Equal {
    for _, diff := range result.Differences {
        fmt.Printf("Difference: %s at %s\n", diff.Type, diff.Path)
    }
}
```

### ValidationRules
Rule-based validation against SOAP standards:

```go
ruleEngine := NewRuleEngine()
violations := ruleEngine.ValidateSOAPRequest(soapXML, "rpc")

for _, violation := range violations {
    if violation.Severity == "error" {
        fmt.Printf("Error: %s\n", violation.Message)
        fmt.Printf("Suggestion: %s\n", violation.Suggestion)
    }
}
```

### FixtureRunner
Orchestrates end-to-end testing:

```go
runner := NewFixtureRunner()
runner.LoadFixtures("test_fixtures")

results, err := runner.RunAllTests()
for _, result := range results {
    if !result.Passed {
        fmt.Printf("Test %s failed: %s\n", result.TestName, result.Error)
    }
}
```

## Built-in Validation Rules

### RPC Operation Wrapper Rule
- ✅ RPC operations must have operation name as wrapper element
- ✅ Operation element must be in correct namespace
- ❌ Catches `<tns:operation>` instead of `<operation xmlns="...">`

### RPC Part Accessor Rule
- ✅ Part accessors (the children of the RPC operation wrapper) must be in no namespace (WS-I Basic Profile R2735)
- ❌ Catches `<getUser xmlns="..."><userId>` where `userId` inherits the wrapper's namespace; `<userId xmlns="">` or a prefixed wrapper is correct
- Descendants of the accessors are not checked: their qualification depends on the part type's schema (R2737)

### Document Style Rule
- ✅ Document operations should use element-based messages
- ⚠️ Warns about operation wrappers in document style

### Namespace Consistency Rule
- ✅ All namespace prefixes must be declared
- ✅ Namespace usage must be consistent

### SOAP Envelope Rule
- ✅ Valid SOAP envelope structure
- ✅ Correct SOAP namespace declaration
- ✅ Required soap:Body element

### Required Elements Rule
- ✅ SOAP body must not be empty
- ✅ Required elements must be present (extensible with schema info)

## Usage

### Basic Test
```go
func TestMyWSDL(t *testing.T) {
    runner := NewFixtureRunner()
    err := runner.LoadFixtures("test_fixtures")
    require.NoError(t, err)
    
    results, err := runner.RunAllTests()
    require.NoError(t, err)
    
    for _, result := range results {
        assert.True(t, result.Passed, "Test %s should pass", result.TestName)
    }
}
```

### Integration Test
```go
func TestWithValidation(t *testing.T) {
    runner := NewFixtureRunner()
    ruleEngine := NewRuleEngine()
    
    // ... load and run tests ...
    
    violations := ruleEngine.ValidateSOAPRequest(result.ActualXML, result.Style)
    assert.Empty(t, violations, "Should have no rule violations")
}
```

## Benefits

1. **Early Bug Detection** - Catches SOAP protocol issues immediately
2. **Regression Prevention** - Prevents breaking changes to XML output
3. **Standards Compliance** - Validates against SOAP/WSDL standards
4. **Real-world Testing** - Tests actual HTTP requests, not just Go code
5. **Clear Diagnostics** - Provides actionable error messages and suggestions

## Example: Catching the RPC Namespace Issue

Before fix:
```xml
<soap:Body>
  <tns:information xmlns:tns="https://example.com/soap">
    <userId>123</userId>
  </tns:information>
</soap:Body>
```

After fix:
```xml
<soap:Body>
  <information xmlns="https://example.com/soap">
    <userId>123</userId>
  </information>
</soap:Body>
```

The RPC Operation Wrapper Rule would immediately catch this:
```
❌ Rule violation [RPC Operation Wrapper]: RPC operation using namespace prefix instead of default namespace
   Path: soap:Body/*[1]
   Suggestion: Use xmlns="namespace" instead of tns:operation
```

## Running Tests

```bash
# Run basic tests
go test -v ./pkg/testing

# Check the generated clients against the fixtures in test_fixtures/
go test -v -run TestGenerator_TestFixtures ./pkg/generator
```

This framework would have caught the RPC namespace issue immediately and prevented it from reaching production.