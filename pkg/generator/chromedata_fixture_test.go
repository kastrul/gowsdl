// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
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

func generateFixture(t *testing.T, fixture string, opts ...Option) map[string][]byte {
	t.Helper()

	g, err := New(filepath.Join("..", "..", "fixtures", fixture), append([]Option{WithPackage("chrome")}, opts...)...)
	require.NoError(t, err)

	out, err := g.Generate(context.Background())
	require.NoError(t, err)
	return out
}

func TestGenerator_ChromeData_GeneratedCodeCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go build of generated code in short mode")
	}

	out := generateFixture(t, "chromedata.wsdl", WithServerGeneration(true))

	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)

	dir := t.TempDir()
	client := string(out["header"]) + string(out["types"]) + string(out["operations"])
	server := string(out["server_header"]) + string(out["server_wsdl"]) + string(out["server"])
	goMod := "module chrome\n\ngo 1.21\n\nrequire github.com/kastrul/gowsdl v0.0.0\n\nreplace github.com/kastrul/gowsdl => " + root + "\n"
	goSum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "client.go"), []byte(client), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "server.go"), []byte(server), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.sum"), goSum, 0o600))

	cmd := exec.Command("go", "build", "-mod=mod", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err, "generated code should compile:\n%s", output)
}

func TestGenerator_ChromeData_NestedChoiceAndSequenceFields(t *testing.T) {
	out := generateFixture(t, "chromedata.wsdl")
	src := string(out["header"]) + string(out["types"])

	assert.ElementsMatch(t,
		[]string{"CategoryId", "Definition", "StyleId", "Installed"},
		structFieldNames(t, src, "GenericEquipment"))
	assert.ElementsMatch(t,
		[]string{"TitleId", "Definition", "Range_", "Value"},
		structFieldNames(t, src, "TechnicalSpecification"))
	assert.ElementsMatch(t,
		[]string{"StyleId", "Value", "Condition"},
		structFieldNames(t, src, "TechnicalSpecificationValueType"))
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

	g, err := New(path, WithPackage("svc"), WithExportAllTypes(true))
	require.NoError(t, err)
	out, err := g.Generate(context.Background())
	require.NoError(t, err)
	src := string(out["header"]) + string(out["types"])

	assert.ElementsMatch(t, []string{"Amount", "Card", "Iban"}, structFieldNames(t, src, "Payment"))
	assert.ElementsMatch(t, []string{"Code", "Label"}, structFieldNames(t, src, "HolderDetailType"))
}
