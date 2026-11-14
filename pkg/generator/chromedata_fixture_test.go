// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// structFieldNames parses Go source and returns the field names of the named struct type.
func structFieldNames(t *testing.T, src, typeName string) []string {
	t.Helper()

	f, err := parser.ParseFile(token.NewFileSet(), "generated.go", src, 0)
	require.NoError(t, err)

	obj := f.Scope.Lookup(typeName)
	require.NotNil(t, obj, "type %s not found", typeName)
	spec, ok := obj.Decl.(*ast.TypeSpec)
	require.True(t, ok, "%s is not a type declaration", typeName)
	st, ok := spec.Type.(*ast.StructType)
	require.True(t, ok, "%s is not a struct", typeName)

	var names []string
	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}
	return names
}

func TestGenerator_ChromeData_NestedChoiceAndSequenceFields(t *testing.T) {
	code, _ := generateCode(t, chromeDataWSDL(), WithPackage("chromedata"))

	assert.ElementsMatch(t,
		[]string{"CategoryId", "Definition", "StyleId", "Installed"},
		structFieldNames(t, code, "GenericEquipment"))
	assert.ElementsMatch(t,
		[]string{"TitleId", "Definition", "Range_", "Value"},
		structFieldNames(t, code, "TechnicalSpecification"))
	assert.ElementsMatch(t,
		[]string{"StyleId", "Value", "Condition"},
		structFieldNames(t, code, "TechnicalSpecificationValueType"))
}

func TestGenerator_SequenceChoiceFieldsRendered(t *testing.T) {
	wsdl := `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:tns="urn:test"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:test">
    <types>
        <xsd:schema targetNamespace="urn:test">
            <xsd:complexType name="Payment">
                <xsd:sequence>
                    <xsd:element name="amount" type="xsd:decimal"/>
                    <xsd:choice>
                        <xsd:element name="card" type="xsd:string"/>
                        <xsd:element name="iban" type="xsd:string"/>
                    </xsd:choice>
                </xsd:sequence>
            </xsd:complexType>
            <xsd:element name="holder">
                <xsd:complexType>
                    <xsd:sequence>
                        <xsd:element name="detail">
                            <xsd:complexType>
                                <xsd:sequence>
                                    <xsd:choice>
                                        <xsd:element name="code" type="xsd:int"/>
                                        <xsd:element name="label" type="xsd:string"/>
                                    </xsd:choice>
                                </xsd:sequence>
                            </xsd:complexType>
                        </xsd:element>
                    </xsd:sequence>
                </xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
</definitions>`

	path := filepath.Join(t.TempDir(), "service.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(wsdl), 0o600))

	code, _ := generateCode(t, path, WithPackage("svc"), WithExportAllTypes(true))

	assert.ElementsMatch(t, []string{"Amount", "Card", "Iban"}, structFieldNames(t, code, "Payment"))
	assert.ElementsMatch(t, []string{"Code", "Label"}, structFieldNames(t, code, "HolderDetailType"))
}
