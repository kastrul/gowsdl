// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package testing

import (
	"encoding/xml"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleEngine(t *testing.T) {
	engine := NewRuleEngine()
	assert.NotNil(t, engine)
	assert.Len(t, engine.rules, 6, "Should have 6 default rules")
}

func TestRPCOperationWrapperRule(t *testing.T) {
	rule := &RPCOperationWrapperRule{}

	tests := []struct {
		name           string
		soapXML        string
		style          string
		expectedErrors int
	}{
		{
			name: "Valid RPC operation with xmlns",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://example.com/rpc">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 0,
		},
		{
			// A prefix qualifies the wrapper as well as a default namespace
			// does (Namespaces in XML 1.0, section 6), and leaves the part
			// accessor in no namespace, as WS-I Basic Profile R2735 requires.
			name: "Valid RPC operation with tns prefix",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <tns:information xmlns:tns="http://example.com/test">
      <userId>123</userId>
    </tns:information>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 0,
		},
		{
			name: "Valid RPC operation with prefix declared on the Envelope",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:m="http://example.com/test">
  <soap:Body>
    <m:information>
      <userId>123</userId>
    </m:information>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 0,
		},
		{
			name: "RPC operation with namespace undeclared",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 1, // xmlns="" leaves the wrapper in no namespace
		},
		{
			name: "RPC operation missing namespace",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser>
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 1, // Should detect missing xmlns
		},
		{
			name: "Document style - should be ignored",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <tns:information xmlns:tns="http://example.com/test">
      <userId>123</userId>
    </tns:information>
  </soap:Body>
</soap:Envelope>`,
			style:          "document",
			expectedErrors: 0, // Should ignore document style
		},
		{
			name:           "Invalid XML",
			soapXML:        `<invalid xml`,
			style:          "rpc",
			expectedErrors: 1, // Should detect parse error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := rule.Validate(tt.soapXML, tt.style)
			assert.Len(t, violations, tt.expectedErrors, "Expected %d violations, got %d", tt.expectedErrors, len(violations))

			for _, violation := range violations {
				assert.Equal(t, rule.Name(), violation.RuleName)
				assert.NotEmpty(t, violation.Message)
				assert.NotEmpty(t, violation.Severity)
			}
		})
	}
}

func TestRPCPartAccessorRule(t *testing.T) {
	envelope := func(body string) string {
		return `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body>` + body + `</soap:Body></soap:Envelope>`
	}

	tests := []struct {
		name    string
		soapXML string
		style   string
		// paths lists the paths of the expected violations.
		paths []string
	}{
		{
			name:    "accessors of a prefixed wrapper are in no namespace",
			soapXML: envelope(`<m:getUser xmlns:m="urn:a"><userId>1</userId><flag>true</flag></m:getUser>`),
			style:   "rpc",
		},
		{
			name:    "accessors undeclare the wrapper's default namespace",
			soapXML: envelope(`<getUser xmlns="urn:a"><userId xmlns="">1</userId></getUser>`),
			style:   "rpc",
		},
		{
			name:    "qualified descendants of an accessor are allowed (R2737)",
			soapXML: envelope(`<m:getUser xmlns:m="urn:a"><parameters><m:userId>1</m:userId></parameters></m:getUser>`),
			style:   "rpc",
		},
		{
			name:    "accessors inheriting the wrapper's default namespace",
			soapXML: envelope(`<getUser xmlns="urn:a"><userId>1</userId><flag xmlns="">true</flag></getUser>`),
			style:   "rpc",
			paths:   []string{"soap:Body/getUser/userId"},
		},
		{
			name:    "prefixed accessors",
			soapXML: envelope(`<m:getUser xmlns:m="urn:a"><m:userId>1</m:userId><m:flag>true</m:flag></m:getUser>`),
			style:   "rpc",
			paths:   []string{"soap:Body/getUser/userId", "soap:Body/getUser/flag"},
		},
		{
			name:    "an accessor in another namespace",
			soapXML: envelope(`<m:getUser xmlns:m="urn:a"><p:userId xmlns:p="urn:b">1</p:userId></m:getUser>`),
			style:   "rpc",
			paths:   []string{"soap:Body/getUser/userId"},
		},
		{
			name:    "document style is not checked",
			soapXML: envelope(`<GetUser xmlns="urn:a"><userId>1</userId></GetUser>`),
			style:   "document",
		},
		{
			name:    "empty body",
			soapXML: envelope(``),
			style:   "rpc",
		},
		{
			name:    "invalid XML",
			soapXML: `<invalid xml`,
			style:   "rpc",
			paths:   []string{"soap:Envelope"},
		},
	}

	rule := &RPCPartAccessorRule{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := rule.Validate(tt.soapXML, tt.style)
			var paths []string
			for _, v := range violations {
				assert.Equal(t, rule.Name(), v.RuleName)
				assert.Equal(t, "error", v.Severity)
				assert.NotEmpty(t, v.Message)
				paths = append(paths, v.Path)
			}
			assert.Equal(t, tt.paths, paths, "violation paths")
		})
	}
}

func TestDocumentStyleRule(t *testing.T) {
	rule := &DocumentStyleRule{}

	tests := []struct {
		name           string
		soapXML        string
		style          string
		expectedErrors int
	}{
		{
			name: "Valid document style",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetUserRequest xmlns="http://example.com/simple">
      <userId>123</userId>
    </GetUserRequest>
  </soap:Body>
</soap:Envelope>`,
			style:          "document",
			expectedErrors: 0,
		},
		{
			name: "Document with operation wrapper (warning)",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUserOperation xmlns="http://example.com/simple">
      <userId>123</userId>
    </getUserOperation>
  </soap:Body>
</soap:Envelope>`,
			style:          "document",
			expectedErrors: 1, // Should warn about operation wrapper
		},
		{
			name: "RPC style - should be ignored",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUserOperation xmlns="http://example.com/rpc">
      <userId>123</userId>
    </getUserOperation>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 0, // Should ignore RPC style
		},
		{
			name:           "Invalid XML",
			soapXML:        `<invalid xml`,
			style:          "document",
			expectedErrors: 1, // Should detect parse error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := rule.Validate(tt.soapXML, tt.style)
			assert.Len(t, violations, tt.expectedErrors)

			for _, violation := range violations {
				assert.Equal(t, rule.Name(), violation.RuleName)
				assert.NotEmpty(t, violation.Message)
				assert.NotEmpty(t, violation.Severity)
			}
		})
	}
}

func TestNamespaceConsistencyRule(t *testing.T) {
	rule := &NamespaceConsistencyRule{}

	tests := []struct {
		name           string
		soapXML        string
		style          string
		expectedErrors int
	}{
		{
			name: "Valid namespaces",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <tns:getUser xmlns:tns="http://example.com/test">
      <tns:userId>123</tns:userId>
    </tns:getUser>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 0,
		},
		{
			name: "Undefined prefix",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <tns:getUser>
      <tns:userId>123</tns:userId>
    </tns:getUser>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 1, // Should detect undefined tns prefix
		},
		{
			name: "Standard prefixes are allowed",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body xml:lang="en">
    <getUser xmlns="http://example.com/test">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 0, // xml: prefix is standard
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := rule.Validate(tt.soapXML, tt.style)
			assert.Len(t, violations, tt.expectedErrors)

			for _, violation := range violations {
				assert.Equal(t, rule.Name(), violation.RuleName)
				assert.NotEmpty(t, violation.Message)
				assert.NotEmpty(t, violation.Severity)
			}
		})
	}
}

func TestSOAPEnvelopeRule(t *testing.T) {
	rule := &SOAPEnvelopeRule{}

	tests := []struct {
		name           string
		soapXML        string
		style          string
		expectedErrors int
	}{
		{
			name: "Valid SOAP envelope",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://example.com/test">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 0,
		},
		{
			name: "Missing envelope",
			soapXML: `<?xml version="1.0"?>
<root>
  <getUser>
    <userId>123</userId>
  </getUser>
</root>`,
			style:          "rpc",
			expectedErrors: 3, // Missing envelope, body, and namespace
		},
		{
			name: "Missing body",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <getUser>
    <userId>123</userId>
  </getUser>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 1, // Missing body
		},
		{
			name: "Wrong namespace",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://wrong.namespace.com/">
  <soap:Body>
    <getUser>
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 1, // Wrong SOAP namespace
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := rule.Validate(tt.soapXML, tt.style)
			assert.Len(t, violations, tt.expectedErrors)

			for _, violation := range violations {
				assert.Equal(t, rule.Name(), violation.RuleName)
				assert.NotEmpty(t, violation.Message)
				assert.NotEmpty(t, violation.Severity)
			}
		})
	}
}

func TestRequiredElementsRule(t *testing.T) {
	rule := &RequiredElementsRule{}

	tests := []struct {
		name           string
		soapXML        string
		style          string
		expectedErrors int
	}{
		{
			name: "Valid request with content",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://example.com/test">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 0,
		},
		{
			name: "Empty body",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 1, // Empty body
		},
		{
			name: "Body with only whitespace",
			soapXML: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectedErrors: 1, // Empty body (whitespace only)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := rule.Validate(tt.soapXML, tt.style)
			assert.Len(t, violations, tt.expectedErrors)

			for _, violation := range violations {
				assert.Equal(t, rule.Name(), violation.RuleName)
				assert.NotEmpty(t, violation.Message)
				assert.NotEmpty(t, violation.Severity)
			}
		})
	}
}

func TestRuleEngineIntegration(t *testing.T) {
	engine := NewRuleEngine()

	// Test with a problematic RPC request: the operation wrapper is in no
	// namespace, although WSDL 1.1 section 3.5 puts it in the soap:body
	// namespace.
	soapXML := `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <information>
      <userId>123</userId>
    </information>
  </soap:Body>
</soap:Envelope>`

	violations := engine.ValidateSOAPRequest(soapXML, "rpc")

	assert.NotEmpty(t, violations, "Should detect violations")

	var rpcViolation *RuleViolation
	for _, v := range violations {
		if v.RuleName == "RPC Operation Wrapper" {
			rpcViolation = &v
			break
		}
	}

	require.NotNil(t, rpcViolation, "Should detect RPC operation wrapper violation")
	// The test checks for either the tns: prefix issue OR missing xmlns issue
	assert.True(t,
		strings.Contains(rpcViolation.Message, "tns:") ||
			strings.Contains(rpcViolation.Message, "namespace"),
		"Should mention either tns: prefix or namespace issue")
	assert.Equal(t, "error", rpcViolation.Severity)
}

// TestRuleEngineWithKnownIssues tests the rule engine with known problematic XML
func TestRuleEngineWithKnownIssues(t *testing.T) {
	ruleEngine := NewRuleEngine()

	testCases := []struct {
		name           string
		soapXML        string
		style          string
		expectErrors   int
		expectWarnings int
	}{
		{
			name: "RPC with tns prefix (should pass)",
			soapXML: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <tns:information xmlns:tns="http://example.com/rpc">
      <userId>123</userId>
    </tns:information>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectErrors:   0, // The prefix qualifies the wrapper as a default namespace would
			expectWarnings: 0,
		},
		{
			name: "RPC with correct namespace (should pass)",
			soapXML: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <information xmlns="http://example.com/rpc">
      <userId xmlns="">123</userId>
    </information>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectErrors:   0,
			expectWarnings: 0,
		},
		{
			name: "RPC accessor inheriting the wrapper's default namespace (should fail)",
			soapXML: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <information xmlns="http://example.com/rpc">
      <userId>123</userId>
    </information>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectErrors:   1, // userId is in http://example.com/rpc, not in no namespace (R2735)
			expectWarnings: 0,
		},
		{
			name: "RPC wrapper in no namespace (should fail)",
			soapXML: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <information>
      <userId>123</userId>
    </information>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectErrors:   1,
			expectWarnings: 0,
		},
		{
			name: "Missing SOAP namespace (should fail)",
			soapXML: `<?xml version="1.0" encoding="UTF-8"?>
<Envelope>
  <Body>
    <GetUser>
      <userId>123</userId>
    </GetUser>
  </Body>
</Envelope>`,
			style:          "document",
			expectErrors:   1, // Missing SOAP namespace
			expectWarnings: 0,
		},
		{
			name: "Empty SOAP body (should fail)",
			soapXML: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
  </soap:Body>
</soap:Envelope>`,
			style:          "document",
			expectErrors:   1, // Empty body
			expectWarnings: 0,
		},
		{
			name: "RPC with tns prefix and several parts (should pass)",
			soapXML: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <tns:information xmlns:tns="https://xbeserve.bedirect.de/soap/2">
      <KID>10320</KID>
      <IDSTRING>meweew3xaj9A</IDSTRING>
      <BIPID>asdasd</BIPID>
    </tns:information>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectErrors:   0,
			expectWarnings: 0,
		},
		{
			name: "RPC with default namespace and several parts (should pass)",
			soapXML: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <information xmlns="https://xbeserve.bedirect.de/soap/2">
      <KID xmlns="">10320</KID>
      <IDSTRING xmlns="">meweew3xaj9A</IDSTRING>
      <BIPID xmlns="">asdasd</BIPID>
    </information>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectErrors:   0,
			expectWarnings: 0,
		},
		{
			name: "RPC with default namespace inherited by several parts (should fail)",
			soapXML: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <information xmlns="https://xbeserve.bedirect.de/soap/2">
      <KID>10320</KID>
      <IDSTRING>meweew3xaj9A</IDSTRING>
      <BIPID>asdasd</BIPID>
    </information>
  </soap:Body>
</soap:Envelope>`,
			style:          "rpc",
			expectErrors:   3, // one per accessor (R2735)
			expectWarnings: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			violations := ruleEngine.ValidateSOAPRequest(tc.soapXML, tc.style)

			errorCount := 0
			warningCount := 0

			for _, violation := range violations {
				switch violation.Severity {
				case "error":
					errorCount++
					t.Logf("Error [%s]: %s", violation.RuleName, violation.Message)
				case "warning":
					warningCount++
					t.Logf("Warning [%s]: %s", violation.RuleName, violation.Message)
				}
			}

			assert.Equal(t, tc.expectErrors, errorCount,
				"Expected %d errors, got %d", tc.expectErrors, errorCount)
			assert.Equal(t, tc.expectWarnings, warningCount,
				"Expected %d warnings, got %d", tc.expectWarnings, warningCount)
		})
	}
}

func TestRegressionPreventionFixtures(t *testing.T) {
	// Test that our fixtures would catch the RPC namespace issue
	runner := NewFixtureRunner()
	engine := NewRuleEngine()

	// Test the namespace edge case fixture specifically
	fixturesDir := filepath.Join("..", "..", "test_fixtures")
	err := runner.LoadFixtures(fixturesDir)
	require.NoError(t, err)

	fixtures := runner.GetFixtures()
	var namespaceFixture *FixtureTestCase

	for i, fixture := range fixtures {
		if fixture.Name == "namespace_edge_case" {
			namespaceFixture = &fixtures[i]
			break
		}
	}

	require.NotNil(t, namespaceFixture, "Should have namespace edge case fixture")
	assert.Equal(t, "rpc", namespaceFixture.Style, "Namespace fixture should be RPC style")

	// The operation wrapper must be in the soap:body namespace (WSDL 1.1
	// section 3.5); whether a prefix or a default namespace puts it there
	// does not matter.
	expectedXML := namespaceFixture.ExpectedRequest
	envelope, err := NewXMLComparator().parseSOAPEnvelopeNode(expectedXML)
	require.NoError(t, err)
	body := soapBodyChildren(envelope)
	require.NotEmpty(t, body)
	assert.Equal(t, xml.Name{Space: "http://example.com/namespace-test", Local: "information"}, body[0].Name,
		"Expected XML should put the operation wrapper in the soap:body namespace")

	// Test validation rules against both correct and incorrect formats
	correctXML := expectedXML
	incorrectXML := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <information>
      <parameters>
        <dataId>test123</dataId>
        <includeDetails>true</includeDetails>
      </parameters>
    </information>
  </soap:Body>
</soap:Envelope>`

	// Validate correct format passes
	correctViolations := engine.ValidateSOAPRequest(correctXML, "rpc")
	t.Logf("Correct XML violations: %d", len(correctViolations))
	for _, v := range correctViolations {
		t.Logf("  - %s: %s", v.RuleName, v.Message)
	}

	// Validate incorrect format fails
	incorrectViolations := engine.ValidateSOAPRequest(incorrectXML, "rpc")
	t.Logf("Incorrect XML violations: %d", len(incorrectViolations))
	for _, v := range incorrectViolations {
		t.Logf("  - %s: %s", v.RuleName, v.Message)
	}

	// The incorrect format should have more violations
	assert.Greater(t, len(incorrectViolations), len(correctViolations),
		"Incorrect namespace format should trigger more violations")

	// Check for specific RPC operation wrapper violations
	var rpcViolation *RuleViolation
	for _, v := range incorrectViolations {
		if v.RuleName == "RPC Operation Wrapper" {
			rpcViolation = &v
			break
		}
	}

	require.NotNil(t, rpcViolation, "Should detect RPC operation wrapper violation")
	assert.True(t,
		strings.Contains(rpcViolation.Message, "tns:") ||
			strings.Contains(rpcViolation.Message, "namespace"),
		"Should mention either tns: prefix or namespace issue in violation message: %s", rpcViolation.Message)
}

func TestRuleEngineAddRule(t *testing.T) {
	engine := NewRuleEngine()
	initialCount := len(engine.rules)

	// Create a custom rule
	customRule := &testCustomRule{}
	engine.AddRule(customRule)

	assert.Len(t, engine.rules, initialCount+1, "Should have added one rule")

	// Test that custom rule is called
	violations := engine.ValidateSOAPRequest("<test/>", "rpc")
	var customViolation *RuleViolation
	for _, v := range violations {
		if v.RuleName == "Test Custom Rule" {
			customViolation = &v
			break
		}
	}

	require.NotNil(t, customViolation, "Custom rule should have been executed")
	assert.Equal(t, "test violation", customViolation.Message)
}

// testCustomRule is a test rule for testing rule engine extensibility
type testCustomRule struct{}

func (r *testCustomRule) Name() string {
	return "Test Custom Rule"
}

func (r *testCustomRule) Description() string {
	return "A test rule for unit testing"
}

func (r *testCustomRule) Validate(_ string, _ string) []RuleViolation {
	return []RuleViolation{
		{
			RuleName: r.Name(),
			Severity: "info",
			Message:  "test violation",
			Path:     "/",
		},
	}
}
