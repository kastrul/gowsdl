package generator

import (
	"go/format"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTypesTemplate_SharedBlocksKeepPerSiteBehaviour pins the points on which
// the call sites of the shared ElementFields and AttributeFields blocks have
// always disagreed, so that merging the blocks cannot silently unify them.
func TestTypesTemplate_SharedBlocksKeepPerSiteBehaviour(t *testing.T) {
	out := generateFromWSDLText(t, `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:tns="http://example.com/x"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/x">
    <types>
        <xsd:schema targetNamespace="http://example.com/x">
            <xsd:complexType name="Base"><xsd:sequence><xsd:element name="baseId" type="xsd:string"/></xsd:sequence></xsd:complexType>
            <xsd:complexType name="Seq">
                <xsd:sequence>
                    <xsd:element name="typed" type="xsd:int"/>
                    <xsd:element name="untyped"/>
                    <xsd:element name="simple"><xsd:simpleType><xsd:restriction base="xsd:int"/></xsd:simpleType></xsd:element>
                    <xsd:element name="inl">
                        <xsd:complexType>
                            <xsd:sequence><xsd:element name="untypedIn"/></xsd:sequence>
                            <xsd:attribute name="Id" type="xsd:string"/>
                        </xsd:complexType>
                    </xsd:element>
                </xsd:sequence>
                <xsd:attribute name="Id" type="xsd:string"/>
            </xsd:complexType>
            <xsd:complexType name="Chooser">
                <xsd:choice><xsd:element name="c1" type="xsd:int"/><xsd:element name="cUntyped"/></xsd:choice>
            </xsd:complexType>
            <xsd:complexType name="Derived">
                <xsd:complexContent>
                    <xsd:extension base="tns:Base">
                        <xsd:sequence><xsd:element name="d1" type="xsd:string"/><xsd:element name="dUntyped"/></xsd:sequence>
                        <xsd:attribute name="Id" type="xsd:string"/>
                    </xsd:extension>
                </xsd:complexContent>
            </xsd:complexType>
        </xsd:schema>
    </types>
</definitions>`)
	formatted, err := format.Source([]byte(string(out["header"]) + string(out["types"])))
	require.NoError(t, err)
	types := string(formatted)

	seq := structBody(t, types, "Seq")
	assert.Regexp(t, `Typed int32 .xml:"typed,omitempty" json:"typed,omitempty"`, seq)
	assert.Regexp(t, `Untyped Untyped `, seq, "a sequence of a named type names the type of an untyped element after the element")
	assert.Regexp(t, `Simple int32 `, seq, "a sequence of a named type uses the base of an inline simple type")
	assert.Regexp(t, `Inl SeqInlType `, seq)
	assert.Regexp(t, `IdAttr string .xml:"Id,attr,omitempty" json:"Id,omitempty"`, seq)

	inl := structBody(t, types, "SeqInlType")
	assert.Regexp(t, `UntypedIn string `, inl, "an anonymous nested type gives an untyped element a string field")
	assert.Regexp(t, `IdAttr string .xml:"Id,attr,omitempty" json:"Id,omitempty"`, inl)

	chooser := structBody(t, types, "Chooser")
	assert.Regexp(t, `C1 int32 `, chooser)
	assert.NotContains(t, chooser, "CUntyped", "a choice of a named type drops an untyped element")

	derived := structBody(t, types, "Derived")
	assert.Regexp(t, `D1 string .xml:"d1,omitempty".`, derived, "extension elements carry no json tag")
	assert.NotContains(t, derived, "DUntyped", "an extension sequence drops an untyped element")
	assert.Regexp(t, `Id string .xml:"Id,attr,omitempty".`, derived, "extension attributes of a named type are not renamed and carry no json tag")
}
