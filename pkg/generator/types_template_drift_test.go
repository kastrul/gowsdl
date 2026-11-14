package generator

import (
	"go/format"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator_UntypedElement_IsStringAtEverySite(t *testing.T) {
	out := generate(t, writeWSDL(t, `<?xml version="1.0" encoding="UTF-8"?>
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
                    <xsd:element name="untypedMany" maxOccurs="unbounded"/>
                    <xsd:element name="inl">
                        <xsd:complexType>
                            <xsd:sequence><xsd:element name="untypedIn"/><xsd:element name="untypedInMany" maxOccurs="unbounded"/></xsd:sequence>
                        </xsd:complexType>
                    </xsd:element>
                    <xsd:element name="inlChoice">
                        <xsd:complexType><xsd:choice><xsd:element name="ic"/></xsd:choice></xsd:complexType>
                    </xsd:element>
                    <xsd:element name="inlAll">
                        <xsd:complexType><xsd:all><xsd:element name="ia"/></xsd:all></xsd:complexType>
                    </xsd:element>
                </xsd:sequence>
            </xsd:complexType>
            <xsd:complexType name="Chooser">
                <xsd:choice><xsd:element name="c1" type="xsd:int"/><xsd:element name="cUntyped"/></xsd:choice>
            </xsd:complexType>
            <xsd:complexType name="Everything">
                <xsd:all><xsd:element name="aUntyped"/></xsd:all>
            </xsd:complexType>
            <xsd:complexType name="Derived">
                <xsd:complexContent>
                    <xsd:extension base="tns:Base">
                        <xsd:sequence><xsd:element name="d1" type="xsd:string"/><xsd:element name="dUntyped"/><xsd:element name="dUntypedMany" maxOccurs="unbounded"/></xsd:sequence>
                    </xsd:extension>
                </xsd:complexContent>
            </xsd:complexType>
        </xsd:schema>
    </types>
</definitions>`), WithPackage("test"))
	formatted, err := format.Source([]byte(string(out["header"]) + string(out["types"])))
	require.NoError(t, err)
	types := string(formatted)

	seq := structBody(t, types, "Seq")
	assert.Regexp(t, `\sTyped int32 .xml:"typed,omitempty" json:"typed,omitempty"`, seq)
	assert.Regexp(t, `\sUntyped string .xml:"untyped,omitempty" json:"untyped,omitempty".`, seq)
	assert.Regexp(t, `\sUntypedMany \[\]string .xml:"untypedMany,omitempty" json:"untypedMany,omitempty".`, seq)

	inl := structBody(t, types, "SeqInlType")
	assert.Regexp(t, `\sUntypedIn string .xml:"untypedIn,omitempty" json:"untypedIn,omitempty".`, inl)
	assert.Regexp(t, `\sUntypedInMany \[\]string `, inl)
	assert.Regexp(t, `\sIc string `, structBody(t, types, "SeqInlChoiceType"))
	assert.Regexp(t, `\sIa string `, structBody(t, types, "SeqInlAllType"))

	chooser := structBody(t, types, "Chooser")
	assert.Regexp(t, `\sC1 int32 `, chooser)
	assert.Regexp(t, `\sCUntyped string .xml:"cUntyped,omitempty" json:"cUntyped,omitempty".`, chooser)

	assert.Regexp(t, `\sAUntyped string `, structBody(t, types, "Everything"))

	derived := structBody(t, types, "Derived")
	assert.Regexp(t, `\sD1 string .xml:"d1,omitempty" json:"d1,omitempty".`, derived)
	assert.Regexp(t, `\sDUntyped string .xml:"dUntyped,omitempty" json:"dUntyped,omitempty".`, derived)
	assert.Regexp(t, `\sDUntypedMany \[\]string `, derived)

	assertGeneratedCodeBuilds(t, "untyped", out)
}

func TestGenerator_GlobalElementSimpleContentAttribute_UnderscoreNameIsExported(t *testing.T) {
	out := generate(t, writeWSDL(t, `<?xml version="1.0" encoding="UTF-8"?>
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
</definitions>`), WithPackage("test"))
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
	out := generate(t, writeWSDL(t, `<?xml version="1.0" encoding="UTF-8"?>
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
</definitions>`), WithPackage("test"))
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

	assert.Regexp(t, `\sA int32 `, structBody(t, types, "Everything"))

	derived := structBody(t, types, "Derived")
	assert.Regexp(t, `\sD int32 .xml:"d,omitempty" json:"d,omitempty".`, derived)
	assert.Regexp(t, `\sDMany \[\]int32 .xml:"dMany,omitempty" json:"dMany,omitempty".`, derived)
}

func TestGenerator_ExtensionAttributesOfNamedTypes_HaveJSONTags(t *testing.T) {
	out := generate(t, writeWSDL(t, `<?xml version="1.0" encoding="UTF-8"?>
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
</definitions>`), WithPackage("test"))
	formatted, err := format.Source([]byte(string(out["header"]) + string(out["types"])))
	require.NoError(t, err)
	types := string(formatted)

	assert.Regexp(t, `\sVersion int32 .xml:"version,attr,omitempty" json:"version,omitempty".`, structBody(t, types, "Derived"))
	assert.Regexp(t, `\sCurrency string .xml:"currency,attr,omitempty" json:"currency,omitempty".`, structBody(t, types, "Money"))
}

func TestGenerator_ExtensionSequenceElements_HaveJSONTags(t *testing.T) {
	out := generate(t, writeWSDL(t, `<?xml version="1.0" encoding="UTF-8"?>
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
</definitions>`), WithPackage("test"))
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
	out := generate(t, writeWSDL(t, `<?xml version="1.0" encoding="UTF-8"?>
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
</definitions>`), WithPackage("test"))
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

func TestGenerator_SimpleContentValueField_HasJSONTagAtEverySite(t *testing.T) {
	const simpleContent = `<xsd:simpleContent><xsd:extension base="xsd:decimal"><xsd:attribute name="currency" type="xsd:string"/></xsd:extension></xsd:simpleContent>`
	out := generate(t, writeWSDL(t, `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:tns="http://example.com/x"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/x">
    <types>
        <xsd:schema targetNamespace="http://example.com/x">
            <xsd:complexType name="Money">`+simpleContent+`</xsd:complexType>
            <xsd:complexType name="Holder">
                <xsd:sequence><xsd:element name="amount"><xsd:complexType>`+simpleContent+`</xsd:complexType></xsd:element></xsd:sequence>
            </xsd:complexType>
            <xsd:element name="Price"><xsd:complexType>`+simpleContent+`</xsd:complexType></xsd:element>
        </xsd:schema>
    </types>
</definitions>`), WithPackage("test"))
	formatted, err := format.Source([]byte(string(out["header"]) + string(out["types"])))
	require.NoError(t, err)
	types := string(formatted)

	for _, name := range []string{"Money", "HolderAmountType", "Price"} {
		assert.Contains(t, structBody(t, types, name), "Value float64 `xml:\",chardata\" json:\"value\"`", name)
	}
}

func TestGenerator_TargetNamespaceMethod_IsDocumentedAtEverySite(t *testing.T) {
	out := generate(t, writeWSDL(t, `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:tns="http://example.com/x"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/x">
    <types>
        <xsd:schema targetNamespace="http://example.com/x">
            <xsd:complexType name="Payload"><xsd:sequence><xsd:element name="v" type="xsd:string"/></xsd:sequence></xsd:complexType>
            <xsd:element name="Payload" type="tns:Payload"/>
            <xsd:element name="Aliased" type="tns:Payload"/>
            <xsd:element name="Anonymous">
                <xsd:complexType><xsd:sequence><xsd:element name="v" type="xsd:string"/></xsd:sequence></xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
    <message name="PayloadIn"><part name="parameters" element="tns:Payload"/></message>
    <message name="AliasedIn"><part name="parameters" element="tns:Aliased"/></message>
    <message name="AnonymousIn"><part name="parameters" element="tns:Anonymous"/></message>
</definitions>`), WithPackage("test"))
	formatted, err := format.Source([]byte(string(out["header"]) + string(out["types"])))
	require.NoError(t, err)
	types := string(formatted)

	for _, name := range []string{"Payload", "Aliased", "Anonymous"} {
		assert.Contains(t, types,
			"// TargetNamespace returns the target namespace for SOAP envelope configuration\n"+
				"func (t "+name+") TargetNamespace() string {\n\treturn \"http://example.com/x\"\n}", name)
	}
}

func TestGenerator_ExtensionSequenceElement_InlineComplexTypeWinsOverTypeAttribute(t *testing.T) {
	out := generate(t, writeWSDL(t, `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:tns="http://example.com/x"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/x">
    <types>
        <xsd:schema targetNamespace="http://example.com/x">
            <xsd:complexType name="Base"><xsd:sequence><xsd:element name="baseId" type="xsd:string"/></xsd:sequence></xsd:complexType>
            <xsd:complexType name="Plain">
                <xsd:sequence>
                    <xsd:element name="both" type="xsd:string">
                        <xsd:complexType><xsd:sequence><xsd:element name="z" type="xsd:int"/></xsd:sequence></xsd:complexType>
                    </xsd:element>
                </xsd:sequence>
            </xsd:complexType>
            <xsd:complexType name="Derived">
                <xsd:complexContent>
                    <xsd:extension base="tns:Base">
                        <xsd:sequence>
                            <xsd:element name="both" type="xsd:string">
                                <xsd:complexType><xsd:sequence><xsd:element name="z" type="xsd:int"/></xsd:sequence></xsd:complexType>
                            </xsd:element>
                        </xsd:sequence>
                    </xsd:extension>
                </xsd:complexContent>
            </xsd:complexType>
        </xsd:schema>
    </types>
</definitions>`), WithPackage("test"))
	formatted, err := format.Source([]byte(string(out["header"]) + string(out["types"])))
	require.NoError(t, err)
	types := string(formatted)

	assert.Regexp(t, `\sBoth PlainBothType .xml:"both,omitempty" json:"both,`, structBody(t, types, "Plain"))
	assert.Regexp(t, `\sBoth DerivedBothType .xml:"both,omitempty" json:"both,`, structBody(t, types, "Derived"),
		"an extension sequence must resolve an element the same way a plain sequence does")
}
