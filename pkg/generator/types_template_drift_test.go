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
	assert.Regexp(t, `D1 string .xml:"d1,omitempty" json:"d1,omitempty".`, derived)
	assert.NotContains(t, derived, "DUntyped", "an extension sequence drops an untyped element")
	assert.Regexp(t, `IdAttr string .xml:"Id,attr,omitempty" json:"Id,omitempty".`, derived)
}

func TestGenerator_GlobalElementSimpleContentAttribute_UnderscoreNameIsExported(t *testing.T) {
	out := generateFromWSDLText(t, `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:tns="http://example.com/x"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/x">
    <types>
        <xsd:schema targetNamespace="http://example.com/x">
            <xsd:element name="Response">
                <xsd:complexType>
                    <xsd:simpleContent>
                        <xsd:extension base="xsd:string">
                            <xsd:attribute name="_under" type="xsd:string"/>
                            <xsd:attribute name="lang" type="xsd:string"/>
                        </xsd:extension>
                    </xsd:simpleContent>
                </xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
</definitions>`)
	formatted, err := format.Source([]byte(string(out["header"]) + string(out["types"])))
	require.NoError(t, err)

	response := structBody(t, string(formatted), "Response")
	assert.Regexp(t, `\sUnder string .xml:"_under,attr,omitempty" json:"_under,omitempty"`, response,
		"an unexported field would be ignored by encoding/xml")
	assert.NotRegexp(t, `\s_under string`, response)
	assert.Regexp(t, `\sLang string .xml:"lang,attr,omitempty" json:"lang,omitempty"`, response)
}

func TestGenerator_InlineSimpleType_UsesRestrictionBaseAtEverySite(t *testing.T) {
	const simple = `<xsd:simpleType><xsd:restriction base="xsd:int"/></xsd:simpleType>`
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
                    <xsd:element name="s">`+simple+`</xsd:element>
                    <xsd:element name="sMany" maxOccurs="unbounded">`+simple+`</xsd:element>
                    <xsd:element name="sList"><xsd:simpleType><xsd:list itemType="xsd:int"/></xsd:simpleType></xsd:element>
                    <xsd:element name="inl">
                        <xsd:complexType>
                            <xsd:sequence><xsd:element name="i">`+simple+`</xsd:element></xsd:sequence>
                        </xsd:complexType>
                    </xsd:element>
                    <xsd:element name="inlChoice">
                        <xsd:complexType>
                            <xsd:choice><xsd:element name="ic">`+simple+`</xsd:element></xsd:choice>
                        </xsd:complexType>
                    </xsd:element>
                    <xsd:element name="inlAll">
                        <xsd:complexType>
                            <xsd:all><xsd:element name="ia">`+simple+`</xsd:element></xsd:all>
                        </xsd:complexType>
                    </xsd:element>
                </xsd:sequence>
            </xsd:complexType>
            <xsd:complexType name="Chooser">
                <xsd:choice><xsd:element name="c">`+simple+`</xsd:element><xsd:element name="cUntyped"/></xsd:choice>
            </xsd:complexType>
            <xsd:complexType name="Everything">
                <xsd:all><xsd:element name="a">`+simple+`</xsd:element></xsd:all>
            </xsd:complexType>
            <xsd:complexType name="Derived">
                <xsd:complexContent>
                    <xsd:extension base="tns:Base">
                        <xsd:sequence>
                            <xsd:element name="d">`+simple+`</xsd:element>
                            <xsd:element name="dMany" maxOccurs="unbounded">`+simple+`</xsd:element>
                        </xsd:sequence>
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
	assert.Regexp(t, `\sS int32 .xml:"s,omitempty" json:"s,omitempty"`, seq)
	assert.Regexp(t, `\sSMany \[\]int32 .xml:"sMany,omitempty" json:"sMany,omitempty"`, seq)
	assert.Regexp(t, `\sSList string `, seq, "a simple type without a restriction base stays a string")

	assert.Regexp(t, `\sI int32 .xml:"i,omitempty" json:"i,omitempty"`, structBody(t, types, "SeqInlType"))
	assert.Regexp(t, `\sIc int32 `, structBody(t, types, "SeqInlChoiceType"))
	assert.Regexp(t, `\sIa int32 `, structBody(t, types, "SeqInlAllType"))

	chooser := structBody(t, types, "Chooser")
	assert.Regexp(t, `\sC int32 .xml:"c,omitempty" json:"c,omitempty"`, chooser)
	assert.NotContains(t, chooser, "CUntyped", "an element with no type at all is still dropped in a choice")

	assert.Regexp(t, `\sA int32 `, structBody(t, types, "Everything"))

	derived := structBody(t, types, "Derived")
	assert.Regexp(t, `\sD int32 .xml:"d,omitempty" json:"d,omitempty".`, derived)
	assert.Regexp(t, `\sDMany \[\]int32 .xml:"dMany,omitempty" json:"dMany,omitempty".`, derived)
}

func TestGenerator_ExtensionAttributesOfNamedTypes_HaveJSONTags(t *testing.T) {
	out := generateFromWSDLText(t, `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:tns="http://example.com/x"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/x">
    <types>
        <xsd:schema targetNamespace="http://example.com/x">
            <xsd:complexType name="Base"><xsd:sequence><xsd:element name="baseId" type="xsd:string"/></xsd:sequence></xsd:complexType>
            <xsd:complexType name="Derived">
                <xsd:complexContent>
                    <xsd:extension base="tns:Base">
                        <xsd:attribute name="version" type="xsd:int"/>
                    </xsd:extension>
                </xsd:complexContent>
            </xsd:complexType>
            <xsd:complexType name="Money">
                <xsd:simpleContent>
                    <xsd:extension base="xsd:decimal">
                        <xsd:attribute name="currency" type="xsd:string"/>
                    </xsd:extension>
                </xsd:simpleContent>
            </xsd:complexType>
        </xsd:schema>
    </types>
</definitions>`)
	formatted, err := format.Source([]byte(string(out["header"]) + string(out["types"])))
	require.NoError(t, err)
	types := string(formatted)

	assert.Regexp(t, `\sVersion int32 .xml:"version,attr,omitempty" json:"version,omitempty".`, structBody(t, types, "Derived"))
	assert.Regexp(t, `\sCurrency string .xml:"currency,attr,omitempty" json:"currency,omitempty".`, structBody(t, types, "Money"))
}

func TestGenerator_ExtensionSequenceElements_HaveJSONTags(t *testing.T) {
	out := generateFromWSDLText(t, `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:tns="http://example.com/x"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/x">
    <types>
        <xsd:schema targetNamespace="http://example.com/x">
            <xsd:element name="Shared" type="xsd:string"/>
            <xsd:complexType name="Base"><xsd:sequence><xsd:element name="baseId" type="xsd:string"/></xsd:sequence></xsd:complexType>
            <xsd:complexType name="Derived">
                <xsd:complexContent>
                    <xsd:extension base="tns:Base">
                        <xsd:sequence>
                            <xsd:element name="typed" type="xsd:string"/>
                            <xsd:element name="many" type="xsd:int" maxOccurs="unbounded"/>
                            <xsd:element ref="tns:Shared"/>
                            <xsd:element name="opt" minOccurs="0">
                                <xsd:complexType><xsd:sequence><xsd:element name="z" type="xsd:string"/></xsd:sequence></xsd:complexType>
                            </xsd:element>
                            <xsd:element name="list" maxOccurs="unbounded">
                                <xsd:complexType><xsd:sequence><xsd:element name="z" type="xsd:string"/></xsd:sequence></xsd:complexType>
                            </xsd:element>
                        </xsd:sequence>
                    </xsd:extension>
                </xsd:complexContent>
            </xsd:complexType>
        </xsd:schema>
    </types>
</definitions>`)
	formatted, err := format.Source([]byte(string(out["header"]) + string(out["types"])))
	require.NoError(t, err)

	derived := structBody(t, string(formatted), "Derived")
	assert.Regexp(t, `\sTyped string .xml:"typed,omitempty" json:"typed,omitempty".`, derived)
	assert.Regexp(t, `\sMany \[\]int32 .xml:"many,omitempty" json:"many,omitempty".`, derived)
	assert.Regexp(t, `\sShared Shared .xml:"Shared,omitempty" json:"Shared,omitempty".`, derived)
	assert.Regexp(t, `\sOpt \*DerivedOptType .xml:"opt,omitempty" json:"opt,omitempty".`, derived)
	assert.Regexp(t, `\sList \[\]DerivedListType .xml:"list,omitempty" json:"list,omitempty".`, derived)
}

func TestGenerator_IDAttribute_IsRenamedAtEverySite(t *testing.T) {
	const idAttrs = `<xsd:attribute name="Id" type="xsd:string"/><xsd:attribute name="lang" type="xsd:string"/>`
	out := generateFromWSDLText(t, `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:tns="http://example.com/x"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/x">
    <types>
        <xsd:schema targetNamespace="http://example.com/x">
            <xsd:complexType name="Base"><xsd:sequence><xsd:element name="Id" type="xsd:string"/></xsd:sequence></xsd:complexType>
            <xsd:complexType name="Direct"><xsd:sequence><xsd:element name="v" type="xsd:string"/></xsd:sequence>`+idAttrs+`</xsd:complexType>
            <xsd:complexType name="Derived">
                <xsd:complexContent><xsd:extension base="tns:Base">`+idAttrs+`</xsd:extension></xsd:complexContent>
            </xsd:complexType>
            <xsd:complexType name="Money">
                <xsd:simpleContent><xsd:extension base="xsd:decimal">`+idAttrs+`</xsd:extension></xsd:simpleContent>
            </xsd:complexType>
            <xsd:complexType name="Holder">
                <xsd:sequence>
                    <xsd:element name="sc">
                        <xsd:complexType><xsd:simpleContent><xsd:extension base="xsd:string">`+idAttrs+`</xsd:extension></xsd:simpleContent></xsd:complexType>
                    </xsd:element>
                    <xsd:element name="ext">
                        <xsd:complexType><xsd:complexContent><xsd:extension base="tns:Base">`+idAttrs+`</xsd:extension></xsd:complexContent></xsd:complexType>
                    </xsd:element>
                </xsd:sequence>
            </xsd:complexType>
            <xsd:element name="Response">
                <xsd:complexType><xsd:simpleContent><xsd:extension base="xsd:string">`+idAttrs+`</xsd:extension></xsd:simpleContent></xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
</definitions>`)
	formatted, err := format.Source([]byte(string(out["header"]) + string(out["types"])))
	require.NoError(t, err)
	types := string(formatted)

	for _, name := range []string{"Direct", "Derived", "Money", "HolderScType", "HolderExtType", "Response"} {
		body := structBody(t, types, name)
		assert.Regexp(t, `\sIdAttr string .xml:"Id,attr,omitempty" json:"Id,omitempty".`, body, name)
		assert.NotRegexp(t, `\sId string .xml:"Id,attr`, body, name)
		assert.Regexp(t, `\sLang string .xml:"lang,attr,omitempty"`, body, name)
	}
}
