package generator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComplexTypeWithInlineSequence(t *testing.T) {
	wsdlFile := "../../fixtures/complex_sequence_test.wsdl"

	gen, err := New(wsdlFile, WithPackage("testpkg"))
	require.NoError(t, err)

	code, err := gen.Generate(context.TODO())
	require.NoError(t, err)

	// Get the generated code - concatenate all files
	var codeStr string
	for filename, content := range code {
		t.Logf("Generated file: %s", filename)
		codeStr += string(content) + "\n"
	}

	// Check that Person type has Address field with named type
	assert.Contains(t, codeStr, "type Person struct {")
	assert.Contains(t, codeStr, "Address PersonAddressType")

	// Check that PersonAddressType has the expected fields, not just Value
	assert.Contains(t, codeStr, "type PersonAddressType struct {")
	assert.Contains(t, codeStr, "Street string")
	assert.Contains(t, codeStr, "City string")
	assert.Contains(t, codeStr, "ZipCode string")

	// Check Contact field too
	assert.Contains(t, codeStr, "Contact PersonContactType")
	assert.Contains(t, codeStr, "type PersonContactType struct {")
	assert.Contains(t, codeStr, "Phone string")
	assert.Contains(t, codeStr, "Email string")

	// Should NOT contain Value string for these complex types
	assert.NotContains(t, codeStr, "Address struct {\n\t\tValue string")
	assert.NotContains(t, codeStr, "Contact struct {\n\t\tValue string")
}

func TestNestedComplexTypes(t *testing.T) {
	// Test deeply nested complex types as identified by Gemini's review
	wsdlFile := "../../fixtures/nested_complex_type_test.wsdl"

	gen, err := New(wsdlFile, WithPackage("testpkg"))
	require.NoError(t, err)

	code, err := gen.Generate(context.TODO())
	require.NoError(t, err)

	// Get the generated code
	var codeStr string
	for filename, content := range code {
		t.Logf("Generated file: %s", filename)
		codeStr += string(content) + "\n"
	}

	// Check that Company type exists
	assert.Contains(t, codeStr, "type Company struct {")

	// Check that Department is now a named type
	assert.Contains(t, codeStr, "Department CompanyDepartmentType")
	assert.Contains(t, codeStr, "type CompanyDepartmentType struct {")
	assert.Contains(t, codeStr, "DeptName string")

	// Check that Manager is a named type within Department
	assert.Contains(t, codeStr, "Manager CompanyDepartmentTypeManagerType")
	assert.Contains(t, codeStr, "type CompanyDepartmentTypeManagerType struct {")
	assert.Contains(t, codeStr, "FirstName string")
	assert.Contains(t, codeStr, "LastName string")
	assert.Contains(t, codeStr, "Email string")

	// Check that Employees is an array of named types
	assert.Contains(t, codeStr, "Employees []CompanyDepartmentTypeEmployeesType")
	assert.Contains(t, codeStr, "type CompanyDepartmentTypeEmployeesType struct {")
	assert.Contains(t, codeStr, "EmployeeId int32")
	assert.Contains(t, codeStr, "Role string")

	// Ensure nested complex types don't fallback to string
	assert.NotContains(t, codeStr, "Manager string")
	assert.NotContains(t, codeStr, "Employees string")
}
