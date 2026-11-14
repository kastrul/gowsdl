package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kastrul/gowsdl/pkg/parser"
)

func inlineElement(name string, children ...*parser.XSDElement) *parser.XSDElement {
	return &parser.XSDElement{Name: name, ComplexType: &parser.XSDComplexType{Sequence: children}}
}

func TestInlineTypes_NameOf_ResolvesByTypeNotByNames(t *testing.T) {
	deep := inlineElement("range", &parser.XSDElement{Name: "from", Type: "xsd:int"})
	filter := inlineElement("filter", deep)
	request := inlineElement("Request", filter)
	nested := inlineElement("item")
	schemas := []*parser.XSDSchema{{
		Elements:     []*parser.XSDElement{request},
		ComplexTypes: []*parser.XSDComplexType{{Name: "Order", Sequence: []*parser.XSDElement{nested}}},
	}}

	collected := collectInlineTypes(schemas)

	for el, want := range map[*parser.XSDElement]string{
		request: "RequestType",
		filter:  "RequestFilterType",
		deep:    "RequestFilterTypeRangeType",
		nested:  "OrderItemType",
	} {
		got, err := collected.nameOf(el)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}

	copied := *deep
	got, err := collected.nameOf(&copied)
	require.NoError(t, err)
	assert.Equal(t, "RequestFilterTypeRangeType", got, "a copy of the element still has the same anonymous type")
}

func TestInlineTypes_NameOf_AgreesWithExportedLookups(t *testing.T) {
	filter := inlineElement("filter")
	request := inlineElement("Request", filter)
	schemas := []*parser.XSDSchema{{Elements: []*parser.XSDElement{request}}}

	collected := collectInlineTypes(schemas)
	exported := CollectInlineComplexTypes(schemas)

	byName, ok := FindInlineType(exported, "Request", "filter")
	require.True(t, ok)
	byType, err := collected.nameOf(filter)
	require.NoError(t, err)
	assert.Equal(t, byName, byType)

	require.Contains(t, exported, GetInlineTypeKey("", "Request"))
	require.Contains(t, exported, GetInlineTypeKey("Request", "filter"))
	assert.Same(t, filter.ComplexType, exported[GetInlineTypeKey("Request", "filter")].ComplexType)
}

func TestInlineTypes_NameOf_ReportsMissesInsteadOfInventingNames(t *testing.T) {
	collected := collectInlineTypes(nil)

	_, err := collected.nameOf(inlineElement("stray"))
	assert.ErrorContains(t, err, `no name was assigned to the inline complex type of element "stray"`)

	_, err = collected.nameOf(&parser.XSDElement{Name: "plain", Type: "xsd:string"})
	assert.Error(t, err)

	_, err = collected.nameOf(nil)
	assert.Error(t, err)
}

func TestGenerator_InlineTypeInExtensionWithoutBase_IsDeclared(t *testing.T) {
	out := generate(t, writeWSDL(t, `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:tns="http://example.com/x"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/x">
    <types>
        <xsd:schema targetNamespace="http://example.com/x">
            <xsd:complexType name="Weird">
                <xsd:complexContent>
                    <xsd:extension>
                        <xsd:sequence>
                            <xsd:element name="inner">
                                <xsd:complexType>
                                    <xsd:sequence><xsd:element name="v" type="xsd:string"/></xsd:sequence>
                                </xsd:complexType>
                            </xsd:element>
                        </xsd:sequence>
                    </xsd:extension>
                </xsd:complexContent>
            </xsd:complexType>
        </xsd:schema>
    </types>
</definitions>`), WithPackage("test"))

	types := string(out["types"])

	assert.Regexp(t, `Inner\s+WeirdInnerType\s`, types)
	assert.Contains(t, types, "type WeirdInnerType struct", "every inline type a field refers to must be declared")
}
