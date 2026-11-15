package generator

import (
	"go/format"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator_UntypedElement_IsAnyTypeAtEverySite(t *testing.T) {
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
	assert.Regexp(t, `\sUntyped soap\.AnyType .xml:"untyped,omitempty" json:"untyped,omitzero".`, seq)
	assert.Regexp(t, `\sUntypedMany \[\]soap\.AnyType .xml:"untypedMany,omitempty" json:"untypedMany,omitempty".`, seq)

	inl := structBody(t, types, "SeqInlType")
	assert.Regexp(t, `\sUntypedIn soap\.AnyType .xml:"untypedIn,omitempty" json:"untypedIn,omitzero".`, inl)
	assert.Regexp(t, `\sUntypedInMany \[\]soap\.AnyType `, inl)
	assert.Regexp(t, `\sIc soap\.AnyType `, structBody(t, types, "SeqInlChoiceType"))
	assert.Regexp(t, `\sIa soap\.AnyType `, structBody(t, types, "SeqInlAllType"))

	chooser := structBody(t, types, "Chooser")
	assert.Regexp(t, `\sC1 int32 `, chooser)
	assert.Regexp(t, `\sCUntyped soap\.AnyType .xml:"cUntyped,omitempty" json:"cUntyped,omitzero".`, chooser)

	assert.Regexp(t, `\sAUntyped soap\.AnyType `, structBody(t, types, "Everything"))

	derived := structBody(t, types, "Derived")
	assert.Regexp(t, `\sD1 string .xml:"d1,omitempty" json:"d1,omitempty".`, derived)
	assert.Regexp(t, `\sDUntyped soap\.AnyType .xml:"dUntyped,omitempty" json:"dUntyped,omitzero".`, derived)
	assert.Regexp(t, `\sDUntypedMany \[\]soap\.AnyType `, derived)

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

const underscoreAttributeRoundTrip = `package test

import (
	"encoding/xml"
	"testing"
)

func TestUnderscoreAttributeRoundTrip(t *testing.T) {
	var r Response
	if err := xml.Unmarshal([]byte(` + "`" + `<Response xmlns="http://example.com/x" _under="u" lang="et">text</Response>` + "`" + `), &r); err != nil {
		t.Fatal(err)
	}
	if r.Under != "u" || r.Lang != "et" || r.Value != "text" {
		t.Fatalf("decoded %+v", r)
	}
	out, err := xml.Marshal(Response{Value: "v", Under: "sent"})
	if err != nil {
		t.Fatal(err)
	}
	const want = ` + "`" + `<Response xmlns="http://example.com/x" _under="sent">v</Response>` + "`" + `
	if string(out) != want {
		t.Fatalf("marshalled %s, want %s", out, want)
	}
}
`

func TestGenerator_GlobalElementSimpleContentAttribute_UnderscoreRoundTrips(t *testing.T) {
	out := generate(t, writeWSDL(t, `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:tns="http://example.com/x"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/x">
    <types>
        <xsd:schema targetNamespace="http://example.com/x" elementFormDefault="qualified">
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
	runGeneratedModuleTests(t, "example.com/underscore", out, map[string]string{"underscore_test.go": underscoreAttributeRoundTrip})
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

const untypedElementWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:tns="http://example.com/x"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/x">
    <types>
        <xsd:schema targetNamespace="http://example.com/x">
            <xsd:complexType name="Base"><xsd:sequence><xsd:element name="baseId" type="xsd:string"/></xsd:sequence></xsd:complexType>
            <xsd:complexType name="Seq">
                <xsd:sequence>
                    <xsd:element name="one"%s/>
                    <xsd:element name="many" maxOccurs="unbounded"%s/>
                    <xsd:element name="nil" nillable="true"%s/>
                    <xsd:element name="opt" minOccurs="0"%s/>
                    <xsd:element name="inl">
                        <xsd:complexType><xsd:sequence><xsd:element name="in"%s/></xsd:sequence></xsd:complexType>
                    </xsd:element>
                </xsd:sequence>
            </xsd:complexType>
            <xsd:complexType name="Chooser"><xsd:choice><xsd:element name="c"%s/></xsd:choice></xsd:complexType>
            <xsd:complexType name="Everything"><xsd:all><xsd:element name="a"%s/></xsd:all></xsd:complexType>
            <xsd:complexType name="Derived">
                <xsd:complexContent>
                    <xsd:extension base="tns:Base"><xsd:sequence><xsd:element name="d"%s/></xsd:sequence></xsd:extension>
                </xsd:complexContent>
            </xsd:complexType>
            <xsd:element name="Wrapper">
                <xsd:complexType><xsd:sequence><xsd:element name="w"%s/></xsd:sequence></xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
</definitions>`

func TestGenerator_UntypedElement_GeneratesLikeExplicitAnyType(t *testing.T) {
	untyped := generate(t, writeWSDL(t, strings.ReplaceAll(untypedElementWSDL, "%s", "")), WithPackage("test"))
	explicit := generate(t, writeWSDL(t, strings.ReplaceAll(untypedElementWSDL, "%s", ` type="xsd:anyType"`)), WithPackage("test"))

	for _, name := range []string{"Seq", "SeqInlType", "Chooser", "Everything", "Derived", "Wrapper"} {
		assert.Equal(t, explicit.decl(t, name), untyped.decl(t, name), name)
	}
	assert.Equal(t, explicit.client(t), untyped.client(t))
}

const untypedElementRoundTrip = `package test

import (
	"encoding/json"
	"encoding/xml"
	"testing"

	"github.com/kastrul/gowsdl/soap"
)

func TestUntypedElementRoundTrip(t *testing.T) {
	var s Seq
	in := "<Seq><one>first</one><many><k>1</k></many><many>2</many><inl><in>nested</in></inl></Seq>"
	if err := xml.Unmarshal([]byte(in), &s); err != nil {
		t.Fatal(err)
	}
	var one string
	if err := s.One.Decode(&one); err != nil || one != "first" {
		t.Fatalf("one = %q, %v", one, err)
	}
	var k struct {
		K int ` + "`" + `xml:"k"` + "`" + `
	}
	if len(s.Many) != 2 || s.Many[0].Decode(&k) != nil || k.K != 1 || s.Many[1].InnerXML != "2" {
		t.Fatalf("many = %+v", s.Many)
	}
	if s.Inl.In.InnerXML != "nested" {
		t.Fatalf("inl.in = %+v", s.Inl.In)
	}
	if !s.Opt.IsZero() {
		t.Fatalf("opt should be unset: %+v", s.Opt)
	}

	out, err := xml.Marshal(Seq{One: s.One, Many: s.Many, Opt: soap.AnyType{Value: "sent"}})
	if err != nil {
		t.Fatal(err)
	}
	const want = "<Seq><one>first</one><many><k xmlns=\"\">1</k></many><many>2</many><opt>sent</opt><inl></inl></Seq>"
	if string(out) != want {
		t.Fatalf("marshalled\n%s\nwant\n%s", out, want)
	}

	var c Chooser
	if err := xml.Unmarshal([]byte("<Chooser><c>picked</c></Chooser>"), &c); err != nil || c.C.InnerXML != "picked" {
		t.Fatalf("choice = %+v, %v", c, err)
	}
	var d Derived
	if err := xml.Unmarshal([]byte("<Derived><baseId>b</baseId><d>ext</d></Derived>"), &d); err != nil || d.D.InnerXML != "ext" {
		t.Fatalf("extension = %+v, %v", d, err)
	}

	js, err := json.Marshal(Chooser{})
	if err != nil || string(js) != "{}" {
		t.Fatalf("json = %s, %v", js, err)
	}
}
`

func TestGenerator_UntypedElement_RoundTripsThroughEncodingXML(t *testing.T) {
	out := generate(t, writeWSDL(t, strings.ReplaceAll(untypedElementWSDL, "%s", "")), WithPackage("test"))
	runGeneratedModuleTests(t, "example.com/untyped", out, map[string]string{
		"untyped_test.go": untypedElementRoundTrip,
	})
}
