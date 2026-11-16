package generator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/types"
)

func generateXRoadFixture(t *testing.T) generated {
	t.Helper()
	return generate(t, filepath.Join("..", "..", "fixtures", "xroad", "service.wsdl"), WithPackage("xroad"))
}

// declarationCount returns how many times the generated client declares the
// type name.
func declarationCount(t *testing.T, out generated, name string) int {
	t.Helper()

	count := 0
	for _, declared := range out.typeNames(t) {
		if declared == name {
			count++
		}
	}
	return count
}

func TestGenerator_XRoad_GeneratedCodeCompiles(t *testing.T) {
	out := generate(t, filepath.Join("..", "..", "fixtures", "xroad", "service.wsdl"),
		WithPackage("xroad"), WithServerGeneration(true))
	assertGeneratedCodeBuilds(t, "xroad", out)
}

func TestGenerator_XRoad_LowercaseTypeReferences(t *testing.T) {
	out := generateXRoadFixture(t)
	code := out.client(t)

	assert.Contains(t, code, "Kind InsuranceKindType `")
	assert.Contains(t, code, "Errors ErrorListType `")
	assert.Contains(t, code, "Request SubmitRequestType `")
	assert.Contains(t, code, "Response SubmitResponseType `")
	assert.NotRegexp(t, `\s(insuranceKindType|errorListType|submitRequestType|submitResponseType)\s`, code)
}

func TestGenerator_XRoad_NumericEnumConstants(t *testing.T) {
	out := generateXRoadFixture(t)
	code := out.client(t)

	assert.Contains(t, code, "InsuranceKindType1 InsuranceKindType = 1")
	assert.Contains(t, code, "RateType05 RateType = 0.5")
	assert.Contains(t, code, "DerivedKindType1 DerivedKindType = 1")
	assert.Contains(t, code, `GenderTypeM GenderType = "M"`)
}

func TestGenerator_XRoad_NoDuplicateDeclarations(t *testing.T) {
	out := generateXRoadFixture(t)
	code := out.client(t)

	for _, name := range []string{"Submit", "SubmitResponse", "SubmitResponseType", "SubmitType"} {
		assert.LessOrEqual(t, declarationCount(t, out, name), 1, "type %s declared more than once", name)
	}
	assert.Equal(t, 1, declarationCount(t, out, "SubmitResponseType"))
	assert.Contains(t, code, "Message string `", "complexType submitResponseType must keep its own fields")
}

func TestGenerator_XRoad_AttachmentPartUsesDocumentBody(t *testing.T) {
	out := generateXRoadFixture(t)
	code := out.client(t)

	assert.Contains(t, code, "Submit(ctx context.Context, request *Submit) (*SubmitResponse, error)")
	assert.NotContains(t, code, "Attachment []byte")
}

func TestGenerator_XRoad_ResolvesSchemaImports(t *testing.T) {
	out := generateXRoadFixture(t)
	code := out.client(t)

	assert.Equal(t, 1, declarationCount(t, out, "ClientIdentifierType"), "nested import identifiers.xsd should be loaded")
	assert.Contains(t, code, "Client Client `")
	assert.Contains(t, code, "ObjectType string `xml:\"http://example.com/xsd/identifiers objectType,attr\"")
	assert.Contains(t, code, "Lang string `xml:\"http://example.com/xsd/identifiers lang,attr,omitempty\"")
}

func TestGenerator_ResolveSchemaExternals_MissingSchema(t *testing.T) {
	wsdl := `<?xml version="1.0" encoding="UTF-8"?>
<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"
	xmlns:xsd="http://www.w3.org/2001/XMLSchema"
	targetNamespace="http://example.com/test">
	<wsdl:types>
		<xsd:schema targetNamespace="http://example.com/test">
			<xsd:import namespace="http://example.com/missing" schemaLocation="missing.xsd"/>
			<xsd:element name="ping" type="xsd:string"/>
		</xsd:schema>
	</wsdl:types>
</wsdl:definitions>`
	g, err := New(writeWSDL(t, wsdl), WithPackage("test"))
	require.NoError(t, err)

	_, err = g.Generate(context.Background())
	require.NoError(t, err, "an unreachable schemaLocation should only warn")
	assert.Len(t, g.getSchemas(), 1)
}

func TestGenerator_ResolveSchemaExternals_Cycle(t *testing.T) {
	dir := t.TempDir()
	a := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:a">
	<xs:import namespace="urn:b" schemaLocation="b.xsd"/>
	<xs:complexType name="aType"><xs:sequence><xs:element name="v" type="xs:string"/></xs:sequence></xs:complexType>
</xs:schema>`
	b := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:b">
	<xs:import namespace="urn:a" schemaLocation="a.xsd"/>
	<xs:complexType name="bType"><xs:sequence><xs:element name="v" type="xs:string"/></xs:sequence></xs:complexType>
</xs:schema>`
	wsdl := `<?xml version="1.0" encoding="UTF-8"?>
<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"
	xmlns:xsd="http://www.w3.org/2001/XMLSchema"
	targetNamespace="http://example.com/test">
	<wsdl:types>
		<xsd:schema targetNamespace="http://example.com/test">
			<xsd:import namespace="urn:a" schemaLocation="a.xsd"/>
		</xsd:schema>
	</wsdl:types>
</wsdl:definitions>`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.xsd"), []byte(a), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.xsd"), []byte(b), 0o600))
	path := filepath.Join(dir, "test.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(wsdl), 0o600))

	g, err := New(path, WithPackage("test"))
	require.NoError(t, err)

	out, err := g.Generate(context.Background())
	require.NoError(t, err)
	assert.Len(t, g.getSchemas(), 3)

	code := generated(out).types()
	assert.Equal(t, 1, strings.Count(code, "type AType struct"))
	assert.Equal(t, 1, strings.Count(code, "type BType struct"))
}

func TestGenerator_EnumValue(t *testing.T) {
	g := &Generator{
		typeMapper:  types.NewTypeMapper(),
		wsdlVersion: "1.1",
		wsdl: &parser.WSDL{Types: parser.WSDLType{Schemas: []*parser.XSDSchema{{
			SimpleType: []*parser.XSDSimpleType{
				{Name: "codeType", Restriction: parser.XSDRestriction{Base: "xsd:int"}},
			},
		}}}},
	}

	tests := []struct {
		base, value, expected string
	}{
		{"xsd:string", "A", `"A"`},
		{"xsd:string", `say "hi"`, `"say \"hi\""`},
		{"xsd:integer", "42", "42"},
		{"xsd:integer", "-7", "-7"},
		{"xsd:integer", "abc", ""},
		{"xsd:unsignedInt", "-1", ""},
		{"xsd:decimal", "1.25", "1.25"},
		{"xsd:boolean", "true", "true"},
		{"tns:codeType", "5", "5"},
		{"tns:unknownType", "X", `"X"`},
	}

	for _, tt := range tests {
		t.Run(tt.base+"/"+tt.value, func(t *testing.T) {
			assert.Equal(t, tt.expected, g.enumValue(tt.base, tt.value))
		})
	}
}

func TestGenerator_ToGoType(t *testing.T) {
	g := &Generator{typeMapper: types.NewTypeMapper()}

	assert.Equal(t, "string", g.toGoType("xsd:string", false))
	assert.Equal(t, "*int64", g.toGoType("xsd:integer", true))
	assert.Equal(t, "VeadLoeteluType", g.toGoType("tns:veadLoeteluType", false))
	assert.Equal(t, "*VeadLoeteluType", g.toGoType("veadLoeteluType", true))
	assert.Equal(t, "Type_", g.toGoType("tns:type", false))
	assert.Equal(t, "", g.toGoType("", false))
}

func TestCollectInlineComplexTypes_NameCollision(t *testing.T) {
	inline := &parser.XSDComplexType{
		Sequence: []*parser.XSDElement{{Name: "response", Type: "fooResponseType"}},
	}
	nested := &parser.XSDComplexType{
		Sequence: []*parser.XSDElement{{Name: "v", Type: "xsd:string"}},
	}
	schemas := []*parser.XSDSchema{{
		ComplexTypes: []*parser.XSDComplexType{
			{Name: "fooResponseType"},
			{Name: "barItemType"},
			{Name: "bar", Sequence: []*parser.XSDElement{{Name: "item", ComplexType: nested}}},
		},
		Elements: []*parser.XSDElement{
			{Name: "fooResponse", ComplexType: inline},
			{Name: "baz", ComplexType: inline},
		},
	}}

	inlineTypes := CollectInlineComplexTypes(schemas)

	_, collides := inlineTypes["element_fooResponse"]
	assert.False(t, collides, "element-level type colliding with complexType fooResponseType must be skipped")

	baz, ok := inlineTypes["element_baz"]
	require.True(t, ok)
	assert.Equal(t, "BazType", baz.GeneratedName)

	item, ok := inlineTypes["bar_item"]
	require.True(t, ok)
	assert.Equal(t, "BarItemType2", item.GeneratedName)
}

func TestResolveAttributeRefs(t *testing.T) {
	ref := &parser.XSDAttribute{Ref: "id:objectType"}
	unknown := &parser.XSDAttribute{Ref: "xml:lang"}
	local := &parser.XSDAttribute{Name: "code", Type: "xsd:int"}
	inlineRef := &parser.XSDAttribute{Ref: "kind"}

	schemas := []*parser.XSDSchema{{
		Attributes: []*parser.XSDAttribute{
			{Name: "objectType", Type: "id:XRoadObjectType"},
			{Name: "kind", SimpleType: &parser.XSDSimpleType{Restriction: parser.XSDRestriction{Base: "xsd:integer"}}},
		},
		ComplexTypes: []*parser.XSDComplexType{
			{Name: "a", Attributes: []*parser.XSDAttribute{ref, local}},
			{Name: "b", SimpleContent: parser.XSDSimpleContent{Extension: parser.XSDExtension{
				Base:       "xsd:string",
				Attributes: []*parser.XSDAttribute{unknown},
			}}},
		},
		Elements: []*parser.XSDElement{
			{Name: "c", ComplexType: &parser.XSDComplexType{Attributes: []*parser.XSDAttribute{inlineRef}}},
		},
	}}

	ResolveAttributeRefs(schemas)

	assert.Equal(t, "objectType", ref.Name)
	assert.Equal(t, "id:XRoadObjectType", ref.Type)
	assert.Equal(t, "lang", unknown.Name)
	assert.Equal(t, "xsd:string", unknown.Type)
	assert.Equal(t, "code", local.Name)
	assert.Equal(t, "xsd:int", local.Type)
	assert.Equal(t, "kind", inlineRef.Name)
	assert.Equal(t, "xsd:integer", inlineRef.Type)
}

func TestGenerator_XRoad_UnqualifiedBodyBesideQualifiedImports(t *testing.T) {
	out := generateXRoadFixture(t)
	code := out.client(t)

	assert.Contains(t, structBody(t, code, "Submit"), "XMLName xml.Name `xml:\"tns:submit\"`")
	assert.Contains(t, code, "func (t Submit) TargetNamespace() string")
	assert.Contains(t, structBody(t, code, "Client"), "XMLName xml.Name `xml:\"http://example.com/xsd/header.xsd client\"`")
	assert.NotContains(t, code, "func (t Client) TargetNamespace() string")
}

func TestGenerator_XRoad_HeaderMessagePartsAreSiblings(t *testing.T) {
	out := generateXRoadFixture(t)
	code := out.client(t)

	body := structBody(t, code, "RequestHeader")
	assert.Contains(t, body, "Client Client `xml:\"http://example.com/xsd/header.xsd client,omitempty\"")
	assert.Contains(t, body, "UserId UserId `xml:\"http://example.com/xsd/header.xsd userId,omitempty\"")
	assert.Contains(t, code, "func (h RequestHeader) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {\n\treturn soap.MarshalParts(e, h)\n}")
}

func TestGenerator_IsHeaderMessage(t *testing.T) {
	header := &parser.WSDLMessage{Name: "hdr"}
	body := &parser.WSDLMessage{Name: "req"}
	both := &parser.WSDLMessage{Name: "both"}

	op := &parser.WSDLOperation{Name: "Op"}
	op.Input.Message = "tns:req"
	op.Output.Message = "tns:both"

	bop := &parser.WSDLOperation{Name: "Op"}
	bop.Input.SOAPHeader = []*parser.WSDLSOAPHeader{{Message: "tns:hdr"}, {Message: "tns:both"}}

	g := &Generator{wsdl: &parser.WSDL{
		PortTypes: []*parser.WSDLPortType{{Name: "PT", Operations: []*parser.WSDLOperation{op}}},
		Binding:   []*parser.WSDLBinding{{Name: "B", Type: "tns:PT", Operations: []*parser.WSDLOperation{bop}}},
	}}

	assert.True(t, g.isHeaderMessage(header))
	assert.False(t, g.isHeaderMessage(body), "a message bound only as a body is not a header message")
	assert.False(t, g.isHeaderMessage(both), "a message that is also an operation output keeps its wrapper")
	assert.False(t, (&Generator{}).isHeaderMessage(header))
}
