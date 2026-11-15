package generator

import (
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
