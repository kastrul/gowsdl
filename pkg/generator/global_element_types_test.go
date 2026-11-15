package generator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const globalExtensionWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:tns="http://example.com/x"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/x">
    <types>
        <xsd:schema targetNamespace="http://example.com/x" elementFormDefault="qualified">
            <xsd:complexType name="Base"><xsd:sequence><xsd:element name="b" type="xsd:string"/></xsd:sequence></xsd:complexType>
            <xsd:complexType name="Named">
                <xsd:complexContent>
                    <xsd:extension base="tns:Base">
                        <xsd:sequence><xsd:element name="e" type="xsd:string"/></xsd:sequence>
                        <xsd:attribute name="at" type="xsd:string"/>
                    </xsd:extension>
                </xsd:complexContent>
            </xsd:complexType>
            <xsd:element name="Anon">
                <xsd:complexType>
                    <xsd:complexContent>
                        <xsd:extension base="tns:Base">
                            <xsd:sequence><xsd:element name="e" type="xsd:string"/></xsd:sequence>
                            <xsd:attribute name="at" type="xsd:string"/>
                        </xsd:extension>
                    </xsd:complexContent>
                </xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
</definitions>`

func TestGenerator_GlobalElementAnonymousExtension_KeepsExtensionFields(t *testing.T) {
	out := generate(t, writeWSDL(t, globalExtensionWSDL), WithPackage("test"))

	assert.Equal(t, `type Anon struct {
	XMLName	xml.Name	`+"`"+`xml:"http://example.com/x Anon"`+"`"+`

	Base

	E	string	`+"`"+`xml:"e,omitempty" json:"e,omitempty"`+"`"+`

	At	string	`+"`"+`xml:"at,attr,omitempty" json:"at,omitempty"`+"`"+`
}`, out.decl(t, "Anon"))
}

const globalExtensionRoundTrip = `package test

import (
	"encoding/xml"
	"testing"
)

func TestAnonExtensionRoundTrip(t *testing.T) {
	const in = ` + "`" + `<Anon xmlns="http://example.com/x" at="a"><b>base</b><e>ext</e></Anon>` + "`" + `
	var a Anon
	if err := xml.Unmarshal([]byte(in), &a); err != nil {
		t.Fatal(err)
	}
	if a.B != "base" || a.E != "ext" || a.At != "a" {
		t.Fatalf("decoded %+v", a)
	}
	out, err := xml.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != in {
		t.Fatalf("marshalled %s, want %s", out, in)
	}
}
`

func TestGenerator_GlobalElementAnonymousExtension_RoundTripsThroughEncodingXML(t *testing.T) {
	out := generate(t, writeWSDL(t, globalExtensionWSDL), WithPackage("test"))
	runGeneratedModuleTests(t, "example.com/anonext", out, map[string]string{"anon_test.go": globalExtensionRoundTrip})
}

const untypedGlobalElementWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:tns="http://example.com/x"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/x">
    <types>
        <xsd:schema targetNamespace="http://example.com/x" elementFormDefault="qualified">
            <xsd:element name="untypedGlobal"%s/>
            <xsd:complexType name="UsesRef">
                <xsd:sequence>
                    <xsd:element ref="tns:untypedGlobal"/>
                    <xsd:element name="other" type="xsd:string"/>
                </xsd:sequence>
            </xsd:complexType>
        </xsd:schema>
    </types>
</definitions>`

func TestGenerator_UntypedGlobalElement_GeneratesLikeExplicitAnyType(t *testing.T) {
	untyped := generate(t, writeWSDL(t, strings.Replace(untypedGlobalElementWSDL, "%s", "", 1)), WithPackage("test"))
	explicit := generate(t, writeWSDL(t, strings.Replace(untypedGlobalElementWSDL, "%s", ` type="xsd:anyType"`, 1)), WithPackage("test"))

	assert.Equal(t, "type UntypedGlobal soap.AnyType", untyped.decl(t, "UntypedGlobal"))
	assert.Equal(t, explicit.client(t), untyped.client(t))
}

const untypedGlobalRoundTrip = `package test

import (
	"encoding/xml"
	"testing"

	"github.com/kastrul/gowsdl/soap"
)

func TestUntypedGlobalRoundTrip(t *testing.T) {
	const in = ` + "`" + `<UsesRef xmlns="http://example.com/x"><untypedGlobal><k>1</k></untypedGlobal><other>o</other></UsesRef>` + "`" + `
	var u UsesRef
	if err := xml.Unmarshal([]byte(in), &u); err != nil {
		t.Fatal(err)
	}
	var k struct {
		K int ` + "`" + `xml:"k"` + "`" + `
	}
	if err := u.UntypedGlobal.Decode(&k); err != nil || k.K != 1 || u.Other != "o" {
		t.Fatalf("decoded %+v (k %+v, %v)", u, k, err)
	}
	out, err := xml.Marshal(UsesRef{UntypedGlobal: UntypedGlobal(soap.AnyType{Value: "v"}), Other: "o"})
	if err != nil {
		t.Fatal(err)
	}
	const want = ` + "`" + `<UsesRef><untypedGlobal>v</untypedGlobal><other>o</other></UsesRef>` + "`" + `
	if string(out) != want {
		t.Fatalf("marshalled %s, want %s", out, want)
	}
}
`

func TestGenerator_UntypedGlobalElement_RoundTripsThroughEncodingXML(t *testing.T) {
	out := generate(t, writeWSDL(t, strings.Replace(untypedGlobalElementWSDL, "%s", "", 1)), WithPackage("test"))
	runGeneratedModuleTests(t, "example.com/untypedglobal", out, map[string]string{"untyped_global_test.go": untypedGlobalRoundTrip})
}
