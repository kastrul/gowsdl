// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"go/ast"
	"go/parser"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const underscoreWSDLTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://example.com/vb"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/vb">
    <types>
        <xsd:schema targetNamespace="http://example.com/vb" elementFormDefault="qualified">
            <xsd:element name="getName">
                <xsd:complexType>
                    <xsd:sequence>
                        %s
                    </xsd:sequence>
                    <xsd:attribute name="_attr" type="xsd:string"/>
                </xsd:complexType>
            </xsd:element>
            <xsd:element name="getNameResponse">
                <xsd:complexType>
                    <xsd:sequence>
                        <xsd:element name="returnval" type="xsd:string"/>
                    </xsd:sequence>
                </xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
    <message name="getNameRequest">
        <part name="parameters" element="tns:getName"/>
    </message>
    <message name="getNameResponseMsg">
        <part name="parameters" element="tns:getNameResponse"/>
    </message>
    <portType name="VBPortType">
        <operation name="getName">
            <input message="tns:getNameRequest"/>
            <output message="tns:getNameResponseMsg"/>
        </operation>
    </portType>
    <binding name="VBBinding" type="tns:VBPortType">
        <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="getName">
            <soap:operation soapAction=""/>
            <input><soap:body use="literal"/></input>
            <output><soap:body use="literal"/></output>
        </operation>
    </binding>
    <service name="VBService">
        <port name="VBPort" binding="tns:VBBinding">
            <soap:address location="http://localhost:18083/"/>
        </port>
    </service>
</definitions>`

func generateUnderscoreWSDL(t *testing.T, sequence string) generated {
	t.Helper()
	return generate(t, writeWSDL(t, strings.Replace(underscoreWSDLTemplate, "%s", sequence, 1)), WithPackage("vb"))
}

func TestGenerator_UnderscoreElement_ExportedField(t *testing.T) {
	out := generateUnderscoreWSDL(t, `<xsd:element name="_this" type="xsd:string"/>`)
	code := out.client(t)

	assert.Regexp(t, regexp.MustCompile(`\bThis\s+string\s+`+"`"+`xml:"_this,omitempty" json:"_this,omitempty"`+"`"), code)
	assert.NotRegexp(t, regexp.MustCompile(`(?m)^\s*_this\s`), code)
	assert.Regexp(t, regexp.MustCompile(`\bAttr\s+string\s+`+"`"+`xml:"_attr,attr,omitempty"`), code)
	assert.NotRegexp(t, regexp.MustCompile(`(?m)^\s*_attr\s`), code)
}

func TestGenerator_UnderscoreElement_NameCollision(t *testing.T) {
	out := generateUnderscoreWSDL(t, `<xsd:element name="this" type="xsd:string"/>
                        <xsd:element name="_this" type="xsd:string"/>`)
	code := out.client(t)

	assert.Regexp(t, regexp.MustCompile(`\bThis\s+string\s+`+"`"+`xml:"this,omitempty"`), code)
	assert.Regexp(t, regexp.MustCompile(`\bThis_\s+string\s+`+"`"+`xml:"_this,omitempty"`), code)
}

func TestGenerator_UnderscoreElement_MarshalsXML(t *testing.T) {
	out := generateUnderscoreWSDL(t, `<xsd:element name="this" type="xsd:string"/>
                        <xsd:element name="_this" type="xsd:string"/>`)

	const mainGo = `package main

import (
	"encoding/xml"
	"fmt"

	"vbtest/vb"
)

func main() {
	data, err := xml.Marshal(vb.GetName{This: "plain", This_: "ref-1", Attr: "a"})
	if err != nil {
		panic(err)
	}
	fmt.Print(string(data))
}
`
	dir := goModule(t, "vbtest", map[string]string{"vb/vb.go": out.client(t), "main.go": mainGo})
	runGo(t, dir, "vet", "./...")
	output := runGo(t, dir, "run", ".")

	assert.Contains(t, output, "<_this>ref-1</_this>")
	assert.Contains(t, output, "<this>plain</this>")
	assert.Contains(t, output, `_attr="a"`)
}

func TestDedupeStructFields(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no collision leaves source unchanged",
			input:    "type A struct {\n\tThis string `xml:\"this\"`\n\tThat string `xml:\"_that\"`\n}\n",
			expected: "type A struct {\n\tThis string `xml:\"this\"`\n\tThat string `xml:\"_that\"`\n}\n",
		},
		{
			name:     "derived field renamed even when it comes first",
			input:    "type A struct {\n\tThis string `xml:\"_this\"`\n\tThis string `xml:\"this\"`\n}\n",
			expected: "type A struct {\n\tThis_ string `xml:\"_this\"`\n\tThis string `xml:\"this\"`\n}\n",
		},
		{
			name:     "three-way collision gets distinct suffixes",
			input:    "type A struct {\n\tX string `xml:\"x\"`\n\tX string `xml:\"_x\"`\n\tX string `xml:\"__x\"`\n}\n",
			expected: "type A struct {\n\tX string `xml:\"x\"`\n\tX_ string `xml:\"_x\"`\n\tX_2 string `xml:\"__x\"`\n}\n",
		},
		{
			name:     "embedded field is never renamed",
			input:    "type A struct {\n\tBase\n\tBase string `xml:\"_base\"`\n}\n",
			expected: "type A struct {\n\tBase\n\tBase_ string `xml:\"_base\"`\n}\n",
		},
		{
			name:     "embedded pointer field is never renamed",
			input:    "type A struct {\n\t*Base\n\tBase string `xml:\"_base\"`\n}\n",
			expected: "type A struct {\n\t*Base\n\tBase_ string `xml:\"_base\"`\n}\n",
		},
		{
			name:     "embedded package-qualified field is never renamed",
			input:    "type A struct {\n\tsoap.Header\n\tHeader string `xml:\"_header\"`\n}\n",
			expected: "type A struct {\n\tsoap.Header\n\tHeader_ string `xml:\"_header\"`\n}\n",
		},
		{
			name:     "embedded generic field is never renamed",
			input:    "type A struct {\n\tList[int]\n\tList string `xml:\"_list\"`\n}\n",
			expected: "type A struct {\n\tList[int]\n\tList_ string `xml:\"_list\"`\n}\n",
		},
		{
			name:     "embedded multi-parameter generic field is never renamed",
			input:    "type A struct {\n\t*Pair[string, int]\n\tPair string `xml:\"_pair\"`\n}\n",
			expected: "type A struct {\n\t*Pair[string, int]\n\tPair_ string `xml:\"_pair\"`\n}\n",
		},
		{
			name:     "collisions are scoped per struct",
			input:    "type A struct {\n\tX string `xml:\"x\"`\n}\ntype B struct {\n\tX string `xml:\"_x\"`\n}\n",
			expected: "type A struct {\n\tX string `xml:\"x\"`\n}\ntype B struct {\n\tX string `xml:\"_x\"`\n}\n",
		},
		{
			name:     "unparseable source is returned unchanged",
			input:    "type A struct {",
			expected: "type A struct {",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(dedupeStructFields([]byte(tt.input))))
		})
	}
}

func TestEmbeddedFieldIdent(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"Base", "Base"},
		{"*Base", "Base"},
		{"soap.Header", "Header"},
		{"*soap.Header", "Header"},
		{"List[int]", "List"},
		{"Pair[string, int]", "Pair"},
		{"*pkg.Pair[string, int]", "Pair"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.expr)
			require.NoError(t, err)
			id := embeddedFieldIdent(expr)
			require.NotNil(t, id)
			assert.Equal(t, tt.expected, id.Name)
		})
	}
}

func TestEmbeddedFieldIdent_UnsupportedExpr(t *testing.T) {
	for _, src := range []string{"[]int", "map[string]int", "func()"} {
		expr, err := parser.ParseExpr(src)
		require.NoError(t, err)
		assert.Nil(t, embeddedFieldIdent(expr), src)
	}
	assert.Nil(t, embeddedFieldIdent(&ast.BadExpr{}))
}
