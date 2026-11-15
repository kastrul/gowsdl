// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package testing

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXMLComparatorBasic(t *testing.T) {
	comparator := NewXMLComparator()
	assert.NotNil(t, comparator)
}

func TestXMLComparatorCompare(t *testing.T) {
	comparator := NewXMLComparator()

	tests := []struct {
		name          string
		expected      string
		actual        string
		shouldMatch   bool
		expectedDiffs int
	}{
		{
			name:          "Identical XML",
			expected:      `<root><child>value</child></root>`,
			actual:        `<root><child>value</child></root>`,
			shouldMatch:   true,
			expectedDiffs: 0,
		},
		{
			name:          "Different values",
			expected:      `<root><child>value1</child></root>`,
			actual:        `<root><child>value2</child></root>`,
			shouldMatch:   false,
			expectedDiffs: 1,
		},
		{
			name:     "Whitespace differences (normalized)",
			expected: `<root><child>value</child></root>`,
			actual: `<root>
				<child>value</child>
			</root>`,
			shouldMatch:   true,
			expectedDiffs: 0,
		},
		{
			name:          "Different structure",
			expected:      `<root><child>value</child></root>`,
			actual:        `<root><other>value</other></root>`,
			shouldMatch:   false,
			expectedDiffs: 1,
		},
		{
			name:          "Missing element",
			expected:      `<root><child1>value</child1><child2>value</child2></root>`,
			actual:        `<root><child1>value</child1></root>`,
			shouldMatch:   false,
			expectedDiffs: 1,
		},
		{
			name:          "Extra element",
			expected:      `<root><child1>value</child1></root>`,
			actual:        `<root><child1>value</child1><child2>value</child2></root>`,
			shouldMatch:   false,
			expectedDiffs: 1,
		},
		{
			name: "Identical XML with namespace and attribute",
			expected: `<?xml version="1.0"?>
<root xmlns="http://example.com">
  <child attr="value">content</child>
</root>`,
			actual: `<?xml version="1.0"?>
<root xmlns="http://example.com">
  <child attr="value">content</child>
</root>`,
			shouldMatch:   true,
			expectedDiffs: 0,
		},
		{
			name:     "Different whitespace (ignored by default)",
			expected: `<root><child>content</child></root>`,
			actual: `<root>
  <child>content</child>
</root>`,
			shouldMatch:   true,
			expectedDiffs: 0,
		},
		{
			name:          "Different element names",
			expected:      `<root><child>content</child></root>`,
			actual:        `<root><other>content</other></root>`,
			shouldMatch:   false,
			expectedDiffs: 1,
		},
		{
			name:          "Different text content",
			expected:      `<root><child>expected</child></root>`,
			actual:        `<root><child>actual</child></root>`,
			shouldMatch:   false,
			expectedDiffs: 1,
		},
		{
			name:          "Missing attribute",
			expected:      `<root><child attr="value">content</child></root>`,
			actual:        `<root><child>content</child></root>`,
			shouldMatch:   false,
			expectedDiffs: 1,
		},
		{
			name:          "Extra attribute",
			expected:      `<root><child>content</child></root>`,
			actual:        `<root><child extra="value">content</child></root>`,
			shouldMatch:   false,
			expectedDiffs: 1,
		},
		{
			name:          "Different attribute values",
			expected:      `<root><child attr="expected">content</child></root>`,
			actual:        `<root><child attr="actual">content</child></root>`,
			shouldMatch:   false,
			expectedDiffs: 1,
		},
		{
			name:          "Missing child element",
			expected:      `<root><child1/><child2/></root>`,
			actual:        `<root><child1/></root>`,
			shouldMatch:   false,
			expectedDiffs: 1,
		},
		{
			name:          "Extra child element",
			expected:      `<root><child1/></root>`,
			actual:        `<root><child1/><child2/></root>`,
			shouldMatch:   false,
			expectedDiffs: 1,
		},
		{
			name:          "Different default namespace",
			expected:      `<root xmlns="http://example.com"><child>content</child></root>`,
			actual:        `<root xmlns="http://other.com"><child>content</child></root>`,
			shouldMatch:   false,
			expectedDiffs: 2, // Root and child element namespaces; the xmlns declarations are not compared
		},

		{
			name:          "Identical XML with declaration",
			expected:      `<?xml version="1.0"?><root><child>value</child></root>`,
			actual:        `<?xml version="1.0"?><root><child>value</child></root>`,
			shouldMatch:   true,
			expectedDiffs: 0,
		},
		{
			name:          "Different values with declaration",
			expected:      `<?xml version="1.0"?><root><child>value1</child></root>`,
			actual:        `<?xml version="1.0"?><root><child>value2</child></root>`,
			shouldMatch:   false,
			expectedDiffs: 1,
		},
		{
			name:     "Whitespace differences with declaration",
			expected: `<?xml version="1.0"?><root><child>value</child></root>`,
			actual: `<?xml version="1.0"?>
			<root>
				<child>value</child>
			</root>`,
			shouldMatch:   true,
			expectedDiffs: 0,
		},
		{
			name: "Identical SOAP envelope with header",
			expected: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Header>
    <auth:Authentication xmlns:auth="http://example.com/auth">
      <auth:username>user</auth:username>
      <auth:password>pass</auth:password>
    </auth:Authentication>
  </soap:Header>
  <soap:Body>
    <getUser xmlns="http://example.com/test">
      <userId>123</userId>
      <options>
        <includeDetails>true</includeDetails>
        <format>xml</format>
      </options>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			actual: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Header>
    <auth:Authentication xmlns:auth="http://example.com/auth">
      <auth:username>user</auth:username>
      <auth:password>pass</auth:password>
    </auth:Authentication>
  </soap:Header>
  <soap:Body>
    <getUser xmlns="http://example.com/test">
      <userId>123</userId>
      <options>
        <includeDetails>true</includeDetails>
        <format>xml</format>
      </options>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			shouldMatch:   true,
			expectedDiffs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := comparator.Compare(tt.expected, tt.actual)
			require.NoError(t, err)

			assert.Equal(t, tt.shouldMatch, result.Equal, "Match result should be %v", tt.shouldMatch)
			assert.Len(t, result.Differences, tt.expectedDiffs, "Expected %d differences", tt.expectedDiffs)

			if !tt.shouldMatch {
				assert.NotEmpty(t, result.Differences, "Should have differences when not matching")
				for _, diff := range result.Differences {
					assert.NotEmpty(t, diff.Type, "Difference type should be set")
					assert.NotEmpty(t, diff.Path, "Difference path should be set")
				}
			}
		})
	}
}

func TestXMLComparatorWithIgnoreOptions(t *testing.T) {
	tests := []struct {
		name             string
		ignoreWhitespace bool
		ignoreOrder      bool
		ignoreNamespaces bool
		expected         string
		actual           string
		shouldEqual      bool
		description      string
	}{
		{
			name:             "Ignore namespaces",
			ignoreNamespaces: true,
			expected:         `<root xmlns="http://example.com"><child>content</child></root>`,
			actual:           `<root xmlns="http://other.com"><child>content</child></root>`,
			shouldEqual:      true,
			description:      "Should ignore namespace differences when configured",
		},
		{
			name:        "Ignore element order",
			ignoreOrder: true,
			expected:    `<root><child1/><child2/></root>`,
			actual:      `<root><child2/><child1/></root>`,
			shouldEqual: true,
			description: "Should ignore element order when configured",
		},
		{
			name:             "Don't ignore whitespace",
			ignoreWhitespace: false,
			expected:         `<root><child>content</child></root>`,
			actual: `<root>
  <child>content</child>
</root>`,
			shouldEqual: false,
			description: "Should detect whitespace differences when not ignored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comparator := &XMLComparator{
				IgnoreWhitespace: tt.ignoreWhitespace,
				IgnoreOrder:      tt.ignoreOrder,
				IgnoreNamespaces: tt.ignoreNamespaces,
			}

			result, err := comparator.Compare(tt.expected, tt.actual)
			require.NoError(t, err, "Comparison should not error")

			assert.Equal(t, tt.shouldEqual, result.Equal, tt.description)
		})
	}
}

func TestXMLComparatorCompareSOAPRequests(t *testing.T) {
	comparator := NewXMLComparator()

	tests := []struct {
		name          string
		expected      string
		actual        string
		shouldEqual   bool
		expectedDiffs int
	}{
		{
			name: "Identical SOAP requests",
			expected: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://example.com/service">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			actual: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://example.com/service">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			shouldEqual:   true,
			expectedDiffs: 0,
		},
		{
			name: "Different operation names",
			expected: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://example.com/service">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			actual: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUserInfo xmlns="http://example.com/service">
      <userId>123</userId>
    </getUserInfo>
  </soap:Body>
</soap:Envelope>`,
			shouldEqual:   false,
			expectedDiffs: 1,
		},
		{
			name: "Different parameter values",
			expected: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://example.com/service">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			actual: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://example.com/service">
      <userId>456</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			shouldEqual:   false,
			expectedDiffs: 1,
		},
		{
			name: "RPC vs Document style namespace handling",
			expected: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <information xmlns="http://example.com/service">
      <userId>123</userId>
    </information>
  </soap:Body>
</soap:Envelope>`,
			actual: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <tns:information xmlns:tns="http://example.com/service">
      <userId>123</userId>
    </tns:information>
  </soap:Body>
</soap:Envelope>`,
			shouldEqual:   false,
			expectedDiffs: 1, // The wrappers are the same element; userId inherits the default namespace in one and is in none in the other
		},
		{
			name: "Identical SOAP requests (test namespace)",
			expected: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://example.com/test">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			actual: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://example.com/test">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			shouldEqual:   true,
			expectedDiffs: 0,
		},
		{
			name: "Different operation names (test namespace)",
			expected: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://example.com/test">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			actual: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getCustomer xmlns="http://example.com/test">
      <userId>123</userId>
    </getCustomer>
  </soap:Body>
</soap:Envelope>`,
			shouldEqual:   false,
			expectedDiffs: 1,
		},
		{
			name: "Different parameter values (test namespace)",
			expected: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://example.com/test">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			actual: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://example.com/test">
      <userId>456</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			shouldEqual:   false,
			expectedDiffs: 1,
		},
		{
			name: "Different namespaces (test namespace)",
			expected: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://example.com/test">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			actual: `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUser xmlns="http://different.example.com/test">
      <userId>123</userId>
    </getUser>
  </soap:Body>
</soap:Envelope>`,
			shouldEqual:   false,
			expectedDiffs: 2, // getUser and userId namespaces
		},
		{
			name: "RPC wrapper with tns prefix vs default namespace, several parts",
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <information xmlns="https://xbeserve.bedirect.de/soap/2">
      <KID>10320</KID>
      <IDSTRING>meweew3xaj9A</IDSTRING>
      <BIPID>asdasd</BIPID>
    </information>
  </soap:Body>
</soap:Envelope>`,
			actual: `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <tns:information xmlns:tns="https://xbeserve.bedirect.de/soap/2">
      <KID>10320</KID>
      <IDSTRING>meweew3xaj9A</IDSTRING>
      <BIPID>asdasd</BIPID>
    </tns:information>
  </soap:Body>
</soap:Envelope>`,
			shouldEqual:   false,
			expectedDiffs: 3, // KID, IDSTRING and BIPID inherit the default namespace in one and are in none in the other
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := comparator.CompareSOAPRequests(tt.expected, tt.actual)
			require.NoError(t, err, "SOAP comparison should not error")

			assert.Equal(t, tt.shouldEqual, result.Equal)
			assert.Len(t, result.Differences, tt.expectedDiffs, "Expected %d differences, got %d", tt.expectedDiffs, len(result.Differences))

			// Log differences for debugging
			if len(result.Differences) > 0 {
				t.Logf("SOAP Differences found:")
				for i, diff := range result.Differences {
					t.Logf("  %d. Type: %s, Path: %s, Expected: %q, Actual: %q, Description: %s",
						i+1, diff.Type, diff.Path, diff.Expected, diff.Actual, diff.Description)
				}
			}
		})
	}
}

func TestXMLComparatorComplexStructures(t *testing.T) {
	comparator := NewXMLComparator()

	expected := `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUserDetails xmlns="http://example.com/service">
      <userRequest>
        <userId type="integer">123</userId>
        <includeAddress>true</includeAddress>
        <includeHistory>false</includeHistory>
        <options>
          <format>json</format>
          <locale>en-US</locale>
        </options>
      </userRequest>
    </getUserDetails>
  </soap:Body>
</soap:Envelope>`

	actual := `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <getUserDetails xmlns="http://example.com/service">
      <userRequest>
        <userId type="integer">456</userId>
        <includeAddress>true</includeAddress>
        <includeHistory>false</includeHistory>
        <options>
          <format>xml</format>
          <locale>en-US</locale>
        </options>
      </userRequest>
    </getUserDetails>
  </soap:Body>
</soap:Envelope>`

	result, err := comparator.CompareSOAPRequests(expected, actual)
	require.NoError(t, err)

	assert.False(t, result.Equal, "Should detect differences in complex structure")
	assert.Equal(t, 2, len(result.Differences), "Should find 2 differences (userId and format)")

	// Verify specific differences
	var userIDDiff, formatDiff *Difference
	for i := range result.Differences {
		diff := &result.Differences[i] // Take address of slice element, not loop variable
		if strings.Contains(diff.Path, "userId") {
			userIDDiff = diff
		} else if strings.Contains(diff.Path, "format") {
			formatDiff = diff
		}
	}

	require.NotNil(t, userIDDiff, "Should detect userId difference")
	assert.Equal(t, "123", userIDDiff.Expected)
	assert.Equal(t, "456", userIDDiff.Actual)

	require.NotNil(t, formatDiff, "Should detect format difference")
	assert.Equal(t, "json", formatDiff.Expected)
	assert.Equal(t, "xml", formatDiff.Actual)
}

func TestXMLComparatorInvalidXML(t *testing.T) {
	comparator := NewXMLComparator()

	tests := []struct {
		name     string
		expected string
		actual   string
	}{
		{
			name:     "Invalid expected XML",
			expected: "<invalid xml>",
			actual:   "<root>value</root>",
		},
		{
			name:     "Invalid actual XML",
			expected: "<root>value</root>",
			actual:   "<invalid xml>",
		},
		{
			name:     "Both invalid",
			expected: "<invalid xml>",
			actual:   "<also invalid>",
		},
		{
			name:     "Empty expected",
			expected: "",
			actual:   "<root>value</root>",
		},
		{
			name:     "Empty actual",
			expected: "<root>value</root>",
			actual:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := comparator.Compare(tt.expected, tt.actual)

			// Should handle invalid XML gracefully
			if err != nil {
				assert.Contains(t, err.Error(), "parse", "Error should mention parsing issue")
			} else {
				// If no error, should at least detect they're different
				assert.False(t, result.Equal, "Invalid XML should not match")
			}
		})
	}
}

func TestXMLComparatorErrorHandling(t *testing.T) {
	comparator := NewXMLComparator()

	tests := []struct {
		name        string
		expected    string
		actual      string
		expectError bool
		description string
	}{
		{
			name:        "Invalid expected XML",
			expected:    "<invalid xml>",
			actual:      "<root>valid</root>",
			expectError: true,
			description: "Should error on invalid expected XML",
		},
		{
			name:        "Invalid actual XML",
			expected:    "<root>valid</root>",
			actual:      "<invalid xml>",
			expectError: true,
			description: "Should error on invalid actual XML",
		},
		{
			name:        "Empty XML strings",
			expected:    "",
			actual:      "",
			expectError: true,
			description: "Should error on empty XML strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := comparator.Compare(tt.expected, tt.actual)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, result, "Result should not be nil on success")
			}
		})
	}
}

func TestXMLComparatorAttributeOrder(t *testing.T) {
	comparator := NewXMLComparator()

	// Attributes in different order should still match
	expected := `<root attr1="value1" attr2="value2">content</root>`
	actual := `<root attr2="value2" attr1="value1">content</root>`

	result, err := comparator.Compare(expected, actual)
	require.NoError(t, err)

	// TODO: Implement semantic XML comparison that ignores attribute order
	// For now, the simple implementation treats these as different
	assert.False(t, result.Equal, "Current implementation does not support attribute order normalization")
	assert.NotEmpty(t, result.Differences, "Should have differences due to attribute order")
}

func TestXMLComparatorNamespaceHandling(t *testing.T) {
	comparator := NewXMLComparator()

	tests := []struct {
		name        string
		expected    string
		actual      string
		shouldMatch bool
	}{
		{
			name: "Same namespace, different prefixes",
			expected: `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
				<soap:Body><test:element xmlns:test="http://example.com/test">value</test:element></soap:Body>
			</soap:Envelope>`,
			actual: `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
				<s:Body><t:element xmlns:t="http://example.com/test">value</t:element></s:Body>
			</s:Envelope>`,
			shouldMatch: true,
		},
		{
			name:        "Different namespaces",
			expected:    `<root xmlns="http://example.com/ns1">value</root>`,
			actual:      `<root xmlns="http://example.com/ns2">value</root>`,
			shouldMatch: false,
		},
		{
			name:        "Namespace vs no namespace",
			expected:    `<root>value</root>`,
			actual:      `<root xmlns="http://example.com/ns">value</root>`,
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := comparator.Compare(tt.expected, tt.actual)
			require.NoError(t, err)

			assert.Equal(t, tt.shouldMatch, result.Equal, "Namespace handling should match expected result")
		})
	}
}

func TestXMLComparatorDifferenceTypes(t *testing.T) {
	comparator := NewXMLComparator()

	tests := []struct {
		name         string
		expected     string
		actual       string
		expectedType string
	}{
		{
			name:         "Value difference",
			expected:     `<root>value1</root>`,
			actual:       `<root>value2</root>`,
			expectedType: "different", // Current implementation uses "different" for all types
		},
		{
			name:         "Element name difference",
			expected:     `<root><child1>value</child1></root>`,
			actual:       `<root><child2>value</child2></root>`,
			expectedType: "different", // Current implementation uses "different" for all types
		},
		{
			name:         "Attribute difference",
			expected:     `<root attr="value1">content</root>`,
			actual:       `<root attr="value2">content</root>`,
			expectedType: "different", // Current implementation uses "different" for all types
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := comparator.Compare(tt.expected, tt.actual)
			require.NoError(t, err)

			assert.False(t, result.Equal, "Should not match")
			assert.NotEmpty(t, result.Differences, "Should have differences")

			// Check that appropriate difference type is detected
			found := false
			for _, diff := range result.Differences {
				if diff.Type == tt.expectedType {
					found = true
					break
				}
			}
			assert.True(t, found, "Should detect difference type: %s", tt.expectedType)
		})
	}
}

// TestExtractOperationName tests operation name extraction
func TestExtractOperationName(t *testing.T) {
	tests := []struct {
		name        string
		soapXML     string
		expectedOp  string
		shouldError bool
	}{
		{
			name: "RPC style operation",
			soapXML: `<?xml version="1.0"?>
			<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
				<soap:Body>
					<getUser xmlns="http://example.com/rpc">
						<userId>123</userId>
					</getUser>
				</soap:Body>
			</soap:Envelope>`,
			expectedOp:  "getUser",
			shouldError: false,
		},
		{
			name: "Document style operation",
			soapXML: `<?xml version="1.0"?>
			<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
				<soap:Body>
					<GetUserRequest xmlns="http://example.com/simple">
						<userId>123</userId>
					</GetUserRequest>
				</soap:Body>
			</soap:Envelope>`,
			expectedOp:  "GetUserRequest",
			shouldError: false,
		},
		{
			name: "Operation with namespace prefix",
			soapXML: `<?xml version="1.0"?>
			<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
				<soap:Body>
					<tns:information xmlns:tns="http://example.com/test">
						<userId>123</userId>
					</tns:information>
				</soap:Body>
			</soap:Envelope>`,
			expectedOp:  "information",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, err := ExtractOperationName(tt.soapXML)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOp, op)
			}
		})
	}
}

func TestXMLComparator_CompareSOAPRequests_NamespaceAware(t *testing.T) {
	const (
		soap11 = `http://schemas.xmlsoap.org/soap/envelope/`
		soap12 = `http://www.w3.org/2003/05/soap-envelope`
	)
	envelope := func(envelopeAttrs, body string) string {
		return `<soap:Envelope xmlns:soap="` + soap11 + `"` + envelopeAttrs + `><soap:Body>` + body + `</soap:Body></soap:Envelope>`
	}

	tests := []struct {
		name     string
		expected string
		actual   string
		// diffs lists the expected differences as "type path: expected | actual".
		diffs []string
	}{
		{
			name:     "prefix declared on the Envelope and default namespace on the element are the same",
			expected: envelope(``, `<Op xmlns="urn:a"><userId xmlns="">1</userId></Op>`),
			actual:   envelope(` xmlns:tns="urn:a"`, `<tns:Op><userId>1</userId></tns:Op>`),
		},
		{
			name:     "different prefixes for the same namespace are the same",
			expected: envelope(``, `<m:Op xmlns:m="urn:a"><m:userId>1</m:userId></m:Op>`),
			actual:   envelope(` xmlns:tns="urn:a"`, `<tns:Op><tns:userId>1</tns:userId></tns:Op>`),
		},
		{
			name:     "an inherited default namespace and a prefix are the same",
			expected: envelope(``, `<Op xmlns="urn:a"><userId>1</userId></Op>`),
			actual:   envelope(``, `<m:Op xmlns:m="urn:a"><m:userId>1</m:userId></m:Op>`),
		},
		{
			name:     "namespaced attributes compare by namespace, not prefix",
			expected: envelope(``, `<Op xmlns="urn:a" xmlns:x="urn:x" x:flag="1"/>`),
			actual:   envelope(` xmlns:y="urn:x"`, `<Op xmlns="urn:a" y:flag="1"/>`),
		},
		{
			name:     "unused namespace declarations are not compared",
			expected: envelope(``, `<Op xmlns="urn:a"/>`),
			actual:   envelope(` xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:unused="urn:u"`, `<Op xmlns="urn:a" xmlns:other="urn:o"/>`),
		},
		{
			name:     "different SOAP envelope prefixes are the same",
			expected: envelope(``, `<Op xmlns="urn:a"/>`),
			actual:   `<s:Envelope xmlns:s="` + soap11 + `"><s:Body><Op xmlns="urn:a"/></s:Body></s:Envelope>`,
		},
		{
			name:     "an element in no namespace differs from one inheriting the default namespace",
			expected: envelope(``, `<Op xmlns="urn:a"><userId>1</userId></Op>`),
			actual:   envelope(``, `<m:Op xmlns:m="urn:a"><userId>1</userId></m:Op>`),
			diffs:    []string{"different soap:Body/Op/userId: {urn:a}userId | userId"},
		},
		{
			name:     "xmlns=\"\" takes an element out of the default namespace",
			expected: envelope(``, `<Op xmlns="urn:a"><userId>1</userId></Op>`),
			actual:   envelope(``, `<Op xmlns="urn:a"><userId xmlns="">1</userId></Op>`),
			diffs:    []string{"different soap:Body/Op/userId: {urn:a}userId | userId"},
		},
		{
			name:     "an unqualified element differs from a qualified one",
			expected: envelope(``, `<Op>1</Op>`),
			actual:   envelope(` xmlns:tns="urn:a"`, `<tns:Op>1</tns:Op>`),
			diffs:    []string{"different soap:Body/Op: Op | {urn:a}Op"},
		},
		{
			name:     "an unprefixed attribute is in no namespace",
			expected: envelope(``, `<Op xmlns="urn:a" flag="1"/>`),
			actual:   envelope(``, `<Op xmlns="urn:a" xmlns:x="urn:a" x:flag="1"/>`),
			diffs: []string{
				// The attribute order check also sees a different name in the same position.
				"different soap:Body/Op/@order: attribute order preserved | attribute order changed",
				"missing soap:Body/Op/@flag: 1 | ",
				"extra soap:Body/Op/@{urn:a}flag:  | 1",
			},
		},
		{
			name:     "SOAP 1.1 and SOAP 1.2 envelopes differ",
			expected: envelope(``, `<Op xmlns="urn:a"/>`),
			actual:   `<env:Envelope xmlns:env="` + soap12 + `"><env:Body><Op xmlns="urn:a"/></env:Body></env:Envelope>`,
			diffs:    []string{"different soap:Envelope: {" + soap11 + "}Envelope | {" + soap12 + "}Envelope"},
		},
		{
			name:     "every Body child is compared",
			expected: envelope(``, `<A xmlns="urn:a"/>`),
			actual:   envelope(``, `<A xmlns="urn:a"/><B xmlns="urn:a"/>`),
			diffs:    []string{"extra soap:Body/B:  | {urn:a}B"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewXMLComparator().CompareSOAPRequests(tt.expected, tt.actual)
			require.NoError(t, err)

			got := make([]string, 0, len(result.Differences))
			for _, d := range result.Differences {
				got = append(got, d.Type+" "+d.Path+": "+d.Expected+" | "+d.Actual)
			}
			assert.ElementsMatch(t, tt.diffs, got)
			assert.Equal(t, len(tt.diffs) == 0, result.Equal)
		})
	}
}

func TestXMLComparator_CompareSOAPRequests_RejectsNonEnvelope(t *testing.T) {
	_, err := NewXMLComparator().CompareSOAPRequests(`<Op/>`, `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"/>`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Envelope")
}
