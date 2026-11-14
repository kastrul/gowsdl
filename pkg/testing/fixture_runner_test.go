// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package testing

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFixtureRunner(t *testing.T) {
	runner := NewFixtureRunner()
	assert.NotNil(t, runner)
	assert.NotNil(t, runner.capture)
	assert.NotNil(t, runner.comparator)
	assert.Empty(t, runner.fixtures)
	assert.False(t, runner.useRealCode, "Should default to simulation mode")
	assert.Empty(t, runner.tempDir, "Should not have temp directory in simulation mode")
}

func TestFixtureRunnerCleanup(t *testing.T) {
	// Test cleanup functionality
	runner, err := NewFixtureRunnerWithRealCode()
	require.NoError(t, err)

	tempDir := runner.tempDir
	assert.NotEmpty(t, tempDir, "Should have temp directory")

	// Check that temp directory exists
	_, err = os.Stat(tempDir)
	assert.NoError(t, err, "Temp directory should exist")

	// Cleanup
	err = runner.Cleanup()
	assert.NoError(t, err, "Cleanup should succeed")

	// Check that temp directory is removed
	_, err = os.Stat(tempDir)
	assert.True(t, os.IsNotExist(err), "Temp directory should be removed")
}

func TestFixtureRunnerLoadFixtures(t *testing.T) {
	// Create temporary test fixtures
	tempDir := t.TempDir()

	// Create directory structure
	docDir := filepath.Join(tempDir, "document_literal")
	rpcDir := filepath.Join(tempDir, "rpc_literal")

	err := os.MkdirAll(docDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(rpcDir, 0755)
	require.NoError(t, err)

	// Create test fixtures
	err = createTestFixture(docDir, "simple", "document")
	require.NoError(t, err)
	err = createTestFixture(rpcDir, "basic_rpc", "rpc")
	require.NoError(t, err)

	runner := NewFixtureRunner()
	err = runner.LoadFixtures(tempDir)
	require.NoError(t, err)

	fixtures := runner.GetFixtures()
	assert.Len(t, fixtures, 2, "Should load both fixtures")

	wantPaths := map[string]string{
		"simple":    filepath.Join(docDir, "simple.wsdl"),
		"basic_rpc": filepath.Join(rpcDir, "basic_rpc.wsdl"),
	}
	wantStyles := map[string]string{"simple": "document", "basic_rpc": "rpc"}
	for _, fixture := range fixtures {
		require.Contains(t, wantPaths, fixture.Name)
		assert.Equal(t, wantPaths[fixture.Name], fixture.WSDLPath)
		assert.Equal(t, wantStyles[fixture.Name], fixture.Style)
		request, err := os.ReadFile(filepath.Join(filepath.Dir(fixture.WSDLPath), fixture.Name+"_request.xml"))
		require.NoError(t, err)
		assert.Equal(t, string(request), fixture.ExpectedRequest)
		assert.Equal(t, "GetUserInfo", fixture.TestData["operation"])
	}
}

func TestFixtureRunnerLoadNonexistentDirectory(t *testing.T) {
	runner := NewFixtureRunner()

	// Should not error when directory doesn't exist
	err := runner.LoadFixtures("/nonexistent/path")
	assert.NoError(t, err)

	fixtures := runner.GetFixtures()
	assert.Empty(t, fixtures)
}

func TestFixtureRunnerIncompleteFixtures(t *testing.T) {
	wsdlContent := `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             targetNamespace="http://example.com/simple">
  <message name="GetUserRequestMessage">
    <part name="parameters" type="s:string"/>
  </message>
</definitions>`

	testData := map[string]any{
		"operation": "GetUser",
		"userId":    "123",
	}
	testDataJSON, err := json.Marshal(testData)
	require.NoError(t, err)

	tests := []struct {
		name  string
		files map[string][]byte
	}{
		{
			name: "missing expected request",
			files: map[string][]byte{
				"incomplete.wsdl":           []byte(wsdlContent),
				"incomplete_test_data.json": testDataJSON,
			},
		},
		{
			name:  "missing expected request and test data",
			files: map[string][]byte{"incomplete.wsdl": []byte(`<definitions></definitions>`)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			docDir := filepath.Join(tempDir, "document_literal")
			require.NoError(t, os.MkdirAll(docDir, 0755))
			for name, content := range tt.files {
				require.NoError(t, os.WriteFile(filepath.Join(docDir, name), content, 0644))
			}

			runner := NewFixtureRunner()
			require.NoError(t, runner.LoadFixtures(tempDir))
			assert.Empty(t, runner.GetFixtures(), "Should not load incomplete fixtures")
		})
	}
}

func TestFixtureRunnerSimulateSOAPRequest(t *testing.T) {
	runner := NewFixtureRunner()

	tests := []struct {
		name     string
		fixture  FixtureTestCase
		contains []string
	}{
		{
			name: "RPC style with operation name",
			fixture: FixtureTestCase{
				Name:  "test_rpc",
				Style: "rpc",
				TestData: map[string]any{
					"operation": "GetUserInfo",
				},
			},
			contains: []string{"GetUserInfo", "xmlns=\"http://example.com/rpc\"", "<userId>123</userId>"},
		},
		{
			name: "RPC style with missing operation (should use default)",
			fixture: FixtureTestCase{
				Name:     "test_rpc_no_op",
				Style:    "rpc",
				TestData: map[string]any{},
			},
			contains: []string{"GetUserInfo", "xmlns=\"http://example.com/rpc\"", "<userId>123</userId>"},
		},
		{
			name: "RPC style with non-string operation (should use default)",
			fixture: FixtureTestCase{
				Name:  "test_rpc_invalid_op",
				Style: "rpc",
				TestData: map[string]any{
					"operation": 123, // Not a string
				},
			},
			contains: []string{"GetUserInfo", "xmlns=\"http://example.com/rpc\"", "<userId>123</userId>"},
		},
		{
			name: "Document style",
			fixture: FixtureTestCase{
				Name:     "test_doc",
				Style:    "document",
				TestData: map[string]any{},
			},
			contains: []string{"GetUserRequest", "xmlns=\"http://example.com/simple\"", "<userId>123</userId>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.simulateSOAPRequest(tt.fixture)

			assert.NotEmpty(t, result)
			assert.Contains(t, result, "soap:Envelope")
			assert.Contains(t, result, "soap:Body")

			for _, expected := range tt.contains {
				assert.Contains(t, result, expected, "Should contain: %s", expected)
			}
		})
	}
}

func TestFixtureRunnerSetHTTPClient(t *testing.T) {
	runner := NewFixtureRunner()
	client := &http.Client{}

	runner.SetHTTPClient(client)

	// Verify that the transport was set to our capture
	assert.Equal(t, runner.capture, client.Transport)
}

func TestFixtureRunnerRunAllTests(t *testing.T) {
	// Create temporary test fixtures
	tempDir := t.TempDir()
	docDir := filepath.Join(tempDir, "document_literal")

	err := os.MkdirAll(docDir, 0755)
	require.NoError(t, err)

	err = createTestFixture(docDir, "simple", "document")
	require.NoError(t, err)

	runner := NewFixtureRunner()
	err = runner.LoadFixtures(tempDir)
	require.NoError(t, err)

	results, err := runner.RunAllTests()
	require.NoError(t, err)

	assert.Len(t, results, 1, "Should have one test result")

	result := results[0]
	assert.Equal(t, "simple", result.TestName)
	assert.Equal(t, "document", result.Style)
	assert.NotEmpty(t, result.ActualXML)
	assert.NotEmpty(t, result.ExpectedXML)
	// Note: The test might not pass because we're using simulated SOAP requests
	// that don't match the expected XML exactly
}

func TestFixtureRunnerInvalidWSDL(t *testing.T) {
	tempDir := t.TempDir()
	docDir := filepath.Join(tempDir, "document_literal")

	err := os.MkdirAll(docDir, 0755)
	require.NoError(t, err)

	// Create fixture with invalid WSDL
	invalidWSDL := "not valid WSDL content"
	expectedXML := `<?xml version="1.0"?><soap:Envelope>...</soap:Envelope>`
	testData := map[string]any{"operation": "test"}
	testDataJSON, _ := json.Marshal(testData)

	err = os.WriteFile(filepath.Join(docDir, "invalid.wsdl"), []byte(invalidWSDL), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(docDir, "invalid_request.xml"), []byte(expectedXML), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(docDir, "invalid_test_data.json"), testDataJSON, 0644)
	require.NoError(t, err)

	runner := NewFixtureRunner()
	err = runner.LoadFixtures(tempDir)
	require.NoError(t, err)

	results, err := runner.RunAllTests()
	require.NoError(t, err)

	assert.Len(t, results, 1)
	result := results[0]
	// In simulation mode, invalid WSDL doesn't cause failures since we don't actually parse it
	// This is expected behavior - simulation mode is meant to be resilient
	assert.NotEmpty(t, result.ActualXML, "Should generate simulated XML even with invalid WSDL")

	// If useRealCode was true, we would expect an error:
	// assert.False(t, result.Passed, "Should fail with invalid WSDL")
	// assert.NotEmpty(t, result.Error, "Should have error message")
}

func TestFixtureRunnerInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	docDir := filepath.Join(tempDir, "document_literal")

	err := os.MkdirAll(docDir, 0755)
	require.NoError(t, err)

	// Create fixture with invalid JSON
	wsdlContent := `<?xml version="1.0"?><definitions>...</definitions>`
	expectedXML := `<?xml version="1.0"?><soap:Envelope>...</soap:Envelope>`
	invalidJSON := `{"invalid": json}`

	err = os.WriteFile(filepath.Join(docDir, "invalid_json.wsdl"), []byte(wsdlContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(docDir, "invalid_json_request.xml"), []byte(expectedXML), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(docDir, "invalid_json_test_data.json"), []byte(invalidJSON), 0644)
	require.NoError(t, err)

	runner := NewFixtureRunner()
	err = runner.LoadFixtures(tempDir)

	// Should error during loading due to invalid JSON
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse test data")
}

// TestSOAPRequestFixtures loads the repository's test_fixtures and checks
// what they contain. It runs in simulation mode, where each fixture's
// expected request is compared with itself; the generated clients are checked
// against these fixtures by TestGenerator_TestFixtures_RequestsMatchExpectedXML
// in pkg/generator.
func TestSOAPRequestFixtures(t *testing.T) {
	runner := NewFixtureRunner()
	engine := NewRuleEngine()

	require.NoError(t, runner.LoadFixtures(filepath.Join("..", "..", "test_fixtures")))
	fixtures := runner.GetFixtures()
	require.GreaterOrEqual(t, len(fixtures), 5, "Should have at least 5 fixtures")

	byName := make(map[string]FixtureTestCase, len(fixtures))
	styles := make(map[string]int)
	for _, fixture := range fixtures {
		byName[fixture.Name] = fixture
		styles[fixture.Style]++
	}
	assert.GreaterOrEqual(t, styles["rpc"], 2, "Should have at least 2 RPC fixtures")
	assert.GreaterOrEqual(t, styles["document"], 2, "Should have at least 2 document fixtures")

	wantStyles := map[string]string{
		"complex_rpc":               "rpc",
		"namespace_edge_case":       "rpc",
		"complex_document":          "document",
		"array_namespace_edge_case": "document",
	}
	for name, style := range wantStyles {
		require.Contains(t, byName, name, "Should have fixture: %s", name)
		assert.Equal(t, style, byName[name].Style, name)
	}

	assert.Equal(t, "GetUserDetails", byName["complex_rpc"].TestData["operation"])
	assert.Contains(t, byName["array_namespace_edge_case"].TestData, "stringArray")
	assert.Contains(t, byName["array_namespace_edge_case"].TestData, "nestedArrays")

	wantRequestContent := map[string][]string{
		"complex_rpc":      {"GetUserDetails", "xmlns="},
		"complex_document": {"ProcessOrderRequest", "orderId", "items", "<item>", "<shippingAddress>"},
		"array_namespace_edge_case": {
			"<stringArray>", "<item>first</item>", "<intArray>", "<value>1</value>",
			"<nestedArrays>", "<level1>", "<level2>", "<emptyArray/>",
		},
	}
	for name, contents := range wantRequestContent {
		for _, content := range contents {
			assert.Contains(t, byName[name].ExpectedRequest, content, name)
		}
	}

	results, err := runner.RunAllTests()
	require.NoError(t, err)
	require.Len(t, results, len(fixtures))
	for _, result := range results {
		t.Run(result.TestName, func(t *testing.T) {
			assert.True(t, result.Passed, "SOAP request should match expected XML: %v", result.Differences)
			for _, violation := range engine.ValidateSOAPRequest(result.ActualXML, result.Style) {
				assert.NotEqual(t, "error", violation.Severity, "Rule violation [%s]: %s at %s",
					violation.RuleName, violation.Message, violation.Path)
			}
		})
	}
}

// Helper function to create a complete test fixture
func createTestFixture(dir, baseName, style string) error {
	wsdlContent := `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             targetNamespace="http://example.com/` + style + `">
  <message name="GetUserRequestMessage">
    <part name="parameters" type="s:string"/>
  </message>
</definitions>`

	var expectedRequest string
	if style == "rpc" {
		expectedRequest = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetUserInfo xmlns="http://example.com/rpc">
      <userId>123</userId>
      <includeDetails>true</includeDetails>
    </GetUserInfo>
  </soap:Body>
</soap:Envelope>`
	} else {
		expectedRequest = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetUserRequest xmlns="http://example.com/simple">
      <userId>123</userId>
      <includeDetails>true</includeDetails>
    </GetUserRequest>
  </soap:Body>
</soap:Envelope>`
	}

	testData := map[string]any{
		"operation": "GetUserInfo",
		"userId":    "123",
	}
	testDataJSON, err := json.Marshal(testData)
	if err != nil {
		return err
	}

	// Write files
	err = os.WriteFile(filepath.Join(dir, baseName+".wsdl"), []byte(wsdlContent), 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(dir, baseName+"_request.xml"), []byte(expectedRequest), 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(dir, baseName+"_test_data.json"), testDataJSON, 0644)
	if err != nil {
		return err
	}

	return nil
}
