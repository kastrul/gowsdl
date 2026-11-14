// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"context"
	"encoding/json"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kastrul/gowsdl/pkg/parser"
)

var multiNamespaceFixture = filepath.Join("..", "..", "fixtures", "multi_namespace.wsdl")

func generateTypes(t *testing.T, wsdlPath string) (map[string][]byte, string) {
	t.Helper()

	g, err := New(wsdlPath, WithPackage("mn"))
	require.NoError(t, err)

	out, err := g.Generate(context.Background())
	require.NoError(t, err)

	formatted, err := format.Source([]byte(string(out["header"]) + string(out["types"])))
	require.NoError(t, err)
	return out, string(formatted)
}

func TestGenerator_XMLNameUsesDeclaringSchemaNamespace(t *testing.T) {
	_, code := generateTypes(t, multiNamespaceFixture)

	tests := []struct {
		structName string
		xmlName    string
	}{
		{"Client", "urn:ns:hdr client"},
		{"Represented", "urn:ns:hdr represented"},
		{"GetThing", "urn:ns:main GetThing"},
		{"GetThingResponse", "urn:ns:main GetThingResponse"},
	}

	for _, tt := range tests {
		t.Run(tt.structName, func(t *testing.T) {
			assert.Contains(t, structBody(t, code, tt.structName), "XMLName xml.Name `xml:\""+tt.xmlName+"\"`")
		})
	}
}

func TestGenerator_TargetNamespaceMethodUsesDeclaringSchemaNamespace(t *testing.T) {
	data, err := os.ReadFile(multiNamespaceFixture)
	require.NoError(t, err)

	unqualified := strings.NewReplacer(
		` elementFormDefault="qualified"`, "",
		`<xsd:complexType name="GetThingResponseType">`, `<xsd:complexType name="GetThingResponse">`,
		`type="tns:GetThingResponseType"`, `type="tns:GetThingResponse"`,
	).Replace(string(data))
	path := filepath.Join(t.TempDir(), "unqualified.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(unqualified), 0o600))

	_, code := generateTypes(t, path)

	tests := []struct {
		typeName  string
		namespace string
	}{
		{"Client", "urn:ns:hdr"},
		{"Represented", "urn:ns:hdr"},
		{"GetThing", "urn:ns:main"},
		{"GetThingResponse", "urn:ns:main"},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			assert.Contains(t, code, "func (t "+tt.typeName+") TargetNamespace() string {\n\treturn \""+tt.namespace+"\"\n}")
		})
	}
}

func TestResolveDeclarationNamespaces(t *testing.T) {
	inline := &parser.XSDComplexType{}
	globalA := &parser.XSDElement{Name: "a", ComplexType: inline}
	typeA := &parser.XSDComplexType{Name: "TA"}
	globalB := &parser.XSDElement{Name: "b"}
	noNS := &parser.XSDElement{Name: "c"}

	schemas := []*parser.XSDSchema{
		{TargetNamespace: "urn:a", Elements: []*parser.XSDElement{globalA}, ComplexTypes: []*parser.XSDComplexType{typeA}},
		{TargetNamespace: "urn:b", Elements: []*parser.XSDElement{globalB}},
		{Elements: []*parser.XSDElement{noNS}},
	}

	got := resolveDeclarationNamespaces(schemas)
	assert.Equal(t, map[any]string{
		globalA: "urn:a",
		inline:  "urn:a",
		typeA:   "urn:a",
		globalB: "urn:b",
	}, got)

	g := &Generator{declNamespaces: got}
	assert.Equal(t, "urn:b", g.namespaceOf(globalB))
	assert.Equal(t, "urn:a", g.namespaceOf(typeA))
	assert.Equal(t, "", g.namespaceOf(noNS), "falls back to the first schema's namespace, which is empty without a WSDL")
}

const multiNamespaceRoundTripProgram = `package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"

	"mn"
)

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	client, err := xml.Marshal(mn.Client{ClientType: mn.ClientType{MemberCode: "70000000"}})
	must(err)
	request, err := xml.Marshal(mn.GetThing{Id: "42"})
	must(err)

	var gotClient mn.Client
	clientErr := xml.Unmarshal([]byte(` + "`" + `<h:client xmlns:h="urn:ns:hdr"><h:memberCode>70000000</h:memberCode></h:client>` + "`" + `), &gotClient)

	var gotRepresented mn.Represented
	representedErr := xml.Unmarshal([]byte(` + "`" + `<h:represented xmlns:h="urn:ns:hdr"><h:partyCode>COM</h:partyCode></h:represented>` + "`" + `), &gotRepresented)

	var gotResponse mn.GetThingResponse
	responseErr := xml.Unmarshal([]byte(` + "`" + `<m:GetThingResponse xmlns:m="urn:ns:main"><m:result>ok</m:result></m:GetThingResponse>` + "`" + `), &gotResponse)

	errString := func(err error) string {
		if err == nil {
			return ""
		}
		return err.Error()
	}

	must(json.NewEncoder(os.Stdout).Encode(map[string]any{
		"client":         string(client),
		"request":        string(request),
		"clientErr":      errString(clientErr),
		"memberCode":     gotClient.MemberCode,
		"representedErr": errString(representedErr),
		"partyCode":      gotRepresented.PartyCode,
		"responseErr":    errString(responseErr),
		"result":         gotResponse.Result,
	}))
}
`

func TestGenerator_XMLNameNamespaceRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go run of generated code in short mode")
	}

	g, err := New(multiNamespaceFixture, WithPackage("mn"))
	require.NoError(t, err)
	out, err := g.Generate(context.Background())
	require.NoError(t, err)

	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	goSum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	require.NoError(t, err)

	dir := t.TempDir()
	goMod := "module mn\n\ngo 1.21\n\nrequire github.com/kastrul/gowsdl v0.0.0\n\nreplace github.com/kastrul/gowsdl => " + root + "\n"
	client := string(out["header"]) + string(out["types"]) + string(out["operations"])
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.sum"), goSum, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mn.go"), []byte(client), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "roundtrip"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", "roundtrip", "main.go"), []byte(multiNamespaceRoundTripProgram), 0o600))

	cmd := exec.Command("go", "run", "-mod=mod", "./cmd/roundtrip")
	cmd.Dir = dir
	cmd.Stderr = &strings.Builder{}
	stdout, err := cmd.Output()
	require.NoError(t, err, "running generated code failed:\n%s", cmd.Stderr)

	var result map[string]string
	require.NoError(t, json.Unmarshal(stdout, &result), "output: %s", stdout)

	assert.Contains(t, result["client"], `<client xmlns="urn:ns:hdr">`)
	assert.Contains(t, result["request"], `<GetThing xmlns="urn:ns:main">`)

	assert.Empty(t, result["clientErr"])
	assert.Equal(t, "70000000", result["memberCode"])
	assert.Empty(t, result["representedErr"])
	assert.Equal(t, "COM", result["partyCode"])
	assert.Empty(t, result["responseErr"])
	assert.Equal(t, "ok", result["result"])
}
