// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kastrul/gowsdl/pkg/parser"
)

var multiSchemaFixture = filepath.Join("..", "..", "fixtures", "multi_schema_namespaces.wsdl")

func TestResolveDeclarationNamespaces(t *testing.T) {
	headerEl := &parser.XSDElement{Name: "client", Type: "hdr:ClientType"}
	inlineCT := &parser.XSDComplexType{}
	inlineEl := &parser.XSDElement{Name: "requestHash", ComplexType: inlineCT}
	headerCT := &parser.XSDComplexType{Name: "ClientType"}
	mainEl := &parser.XSDElement{Name: "GetStatus"}
	mainCT := &parser.XSDComplexType{Name: "Ping"}
	noNSEl := &parser.XSDElement{Name: "loose"}
	noNSCT := &parser.XSDComplexType{Name: "LooseType"}

	schemas := []*parser.XSDSchema{
		{TargetNamespace: "urn:header", Elements: []*parser.XSDElement{headerEl, inlineEl}, ComplexTypes: []*parser.XSDComplexType{headerCT}},
		{TargetNamespace: "urn:main", Elements: []*parser.XSDElement{mainEl}, ComplexTypes: []*parser.XSDComplexType{mainCT}},
		{Elements: []*parser.XSDElement{noNSEl}, ComplexTypes: []*parser.XSDComplexType{noNSCT}},
	}

	d := resolveDeclarationNamespaces(schemas, "urn:fallback")

	assert.Equal(t, "urn:header", d.element(headerEl))
	assert.Equal(t, "urn:header", d.element(inlineEl))
	assert.Equal(t, "urn:header", d.complexType(inlineCT), "inline complex type of a global element takes the element's namespace")
	assert.Equal(t, "urn:header", d.complexType(headerCT))
	assert.Equal(t, "urn:main", d.element(mainEl))
	assert.Equal(t, "urn:main", d.complexType(mainCT))

	t.Run("schema without targetNamespace uses fallback", func(t *testing.T) {
		assert.Equal(t, "urn:fallback", d.element(noNSEl))
		assert.Equal(t, "urn:fallback", d.complexType(noNSCT))
	})

	t.Run("unknown declaration uses fallback", func(t *testing.T) {
		assert.Equal(t, "urn:fallback", d.element(&parser.XSDElement{Name: "other"}))
		assert.Equal(t, "urn:fallback", d.complexType(&parser.XSDComplexType{Name: "Other"}))
	})
}

func TestGenerator_MultiSchema_XMLNameUsesDeclaringSchemaNamespace(t *testing.T) {
	code := generate(t, multiSchemaFixture, WithPackage("ms")).client(t)

	tests := []struct {
		structName string
		xmlName    string
	}{
		{"Client", `xml:"urn:ms:header client"`},
		{"RequestHash", `xml:"urn:ms:header requestHash"`},
		{"RepresentedParty", `xml:"urn:ms:repr representedParty"`},
		{"GetStatus", `xml:"urn:ms:main GetStatus"`},
		{"GetStatusResponse", `xml:"urn:ms:main GetStatusResponse"`},
		{"Ping", `xml:"urn:ms:main Ping"`},
	}

	for _, tt := range tests {
		t.Run(tt.structName, func(t *testing.T) {
			assert.Contains(t, structBody(t, code, tt.structName), "XMLName xml.Name `"+tt.xmlName+"`")
		})
	}
}

func TestGenerator_ElementRefs_SchemaOrderDoesNotChangeXMLName(t *testing.T) {
	src, err := os.ReadFile(elementRefsFixture)
	require.NoError(t, err)

	text := string(src)
	start := strings.Index(text, `<xsd:schema targetNamespace="urn:refs:other"`)
	require.GreaterOrEqual(t, start, 0)
	end := start + strings.Index(text[start:], "</xsd:schema>") + len("</xsd:schema>")
	otherSchema := text[start:end]
	text = text[:start] + text[end:]

	mainStart := strings.Index(text, `<xsd:schema targetNamespace="urn:refs:main"`)
	require.GreaterOrEqual(t, mainStart, 0)
	text = text[:mainStart] + otherSchema + "\n" + text[mainStart:]

	code := generate(t, writeWSDL(t, text), WithPackage("refs")).client(t)

	assert.Contains(t, structBody(t, code, "GetParty"), "XMLName xml.Name `xml:\"urn:refs:main GetParty\"`")
	assert.Contains(t, structBody(t, code, "GetPartyResponse"), "XMLName xml.Name `xml:\"urn:refs:main GetPartyResponse\"`")
	assert.NotContains(t, code, "urn:refs:other GetParty")
}

const unqualifiedMultiSchemaWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             xmlns:tns="urn:uq:main"
             xmlns:hdr="urn:uq:header"
             targetNamespace="urn:uq:main">
    <types>
        <xsd:schema targetNamespace="urn:uq:header">
            <xsd:complexType name="ClientType">
                <xsd:sequence><xsd:element name="code" type="xsd:string"/></xsd:sequence>
            </xsd:complexType>
            <xsd:element name="client" type="hdr:ClientType"/>
            <xsd:element name="trace">
                <xsd:complexType>
                    <xsd:sequence><xsd:element name="id" type="xsd:string"/></xsd:sequence>
                </xsd:complexType>
            </xsd:element>
        </xsd:schema>
        <xsd:schema targetNamespace="urn:uq:main">
            <xsd:complexType name="Echo">
                <xsd:sequence><xsd:element name="text" type="xsd:string"/></xsd:sequence>
            </xsd:complexType>
            <xsd:element name="Echo" type="tns:Echo"/>
            <xsd:element name="EchoResponse">
                <xsd:complexType>
                    <xsd:sequence><xsd:element name="text" type="xsd:string"/></xsd:sequence>
                </xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
    <message name="EchoRequest">
        <part name="parameters" element="tns:Echo"/>
        <part name="client" element="hdr:client"/>
        <part name="trace" element="hdr:trace"/>
    </message>
    <message name="EchoResponseMsg">
        <part name="parameters" element="tns:EchoResponse"/>
    </message>
    <portType name="EchoPort">
        <operation name="Echo">
            <input message="tns:EchoRequest"/>
            <output message="tns:EchoResponseMsg"/>
        </operation>
    </portType>
    <binding name="EchoBinding" type="tns:EchoPort">
        <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="Echo">
            <soap:operation soapAction="urn:uq:main/Echo"/>
            <input><soap:body parts="parameters" use="literal"/></input>
            <output><soap:body use="literal"/></output>
        </operation>
    </binding>
    <service name="EchoService">
        <port name="EchoPort" binding="tns:EchoBinding">
            <soap:address location="http://localhost/echo"/>
        </port>
    </service>
</definitions>`

func TestGenerator_MultiSchema_UnqualifiedTargetNamespaceMethod(t *testing.T) {
	code := generate(t, writeWSDL(t, unqualifiedMultiSchemaWSDL), WithPackage("uq")).client(t)

	t.Run("XMLName keeps the unqualified tns prefix form", func(t *testing.T) {
		assert.Contains(t, structBody(t, code, "Client"), "XMLName xml.Name `xml:\"tns:client\"`")
		assert.Contains(t, structBody(t, code, "Trace"), "XMLName xml.Name `xml:\"tns:trace\"`")
		assert.Contains(t, structBody(t, code, "Echo"), "XMLName xml.Name `xml:\"tns:Echo\"`")
	})

	t.Run("response element keeps local name only", func(t *testing.T) {
		assert.Contains(t, structBody(t, code, "EchoResponse"), "XMLName xml.Name `xml:\"EchoResponse\"`")
	})

	t.Run("TargetNamespace returns the declaring schema namespace", func(t *testing.T) {
		normalized := strings.Join(strings.Fields(code), " ")
		assert.Contains(t, normalized, `func (t Client) TargetNamespace() string { return "urn:uq:header" }`)
		assert.Contains(t, normalized, `func (t Trace) TargetNamespace() string { return "urn:uq:header" }`)
		assert.Contains(t, normalized, `func (t Echo) TargetNamespace() string { return "urn:uq:main" }`)
	})
}

const multiSchemaRoundTripProgram = `package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"

	"ms"
)

const incomingClient = ` + "`" + `<h:client xmlns:h="urn:ms:header"><h:memberClass>COM</h:memberClass><h:memberCode>12345</h:memberCode></h:client>` + "`" + `

const incomingParty = ` + "`" + `<representedParty xmlns="urn:ms:repr"><partyClass>COM</partyClass><partyCode>70000000</partyCode></representedParty>` + "`" + `

const incomingHash = ` + "`" + `<requestHash xmlns="urn:ms:header" algorithmId="SHA-512">abc==</requestHash>` + "`" + `

const wrongNamespaceClient = ` + "`" + `<client xmlns="urn:ms:main"><memberClass>COM</memberClass></client>` + "`" + `

func roundTrip(v, back any) (string, error) {
	data, err := xml.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), xml.Unmarshal(data, back)
}

func main() {
	result := map[string]any{}
	fail := func(err error) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client := ms.Client{ClientType: ms.ClientType{MemberClass: "COM", MemberCode: "12345"}}
	var clientBack ms.Client
	marshaled, err := roundTrip(client, &clientBack)
	if err != nil {
		fail(err)
	}
	result["clientXML"] = marshaled
	result["clientBack"] = clientBack.ClientType

	party := ms.RepresentedParty{RepresentedPartyType: ms.RepresentedPartyType{PartyClass: "COM", PartyCode: "70000000"}}
	var partyBack ms.RepresentedParty
	if marshaled, err = roundTrip(party, &partyBack); err != nil {
		fail(err)
	}
	result["partyXML"] = marshaled
	result["partyBack"] = partyBack.RepresentedPartyType

	status := ms.GetStatus{PersonCode: "38001010000"}
	var statusBack ms.GetStatus
	if marshaled, err = roundTrip(status, &statusBack); err != nil {
		fail(err)
	}
	result["statusXML"] = marshaled
	result["statusBack"] = statusBack.PersonCode

	ping := ms.Ping{Token: "t-1"}
	var pingBack ms.Ping
	if marshaled, err = roundTrip(ping, &pingBack); err != nil {
		fail(err)
	}
	result["pingXML"] = marshaled
	result["pingBack"] = pingBack.Token

	hash := ms.RequestHash{Value: "abc==", AlgorithmId: "SHA-512"}
	var hashBack ms.RequestHash
	if marshaled, err = roundTrip(hash, &hashBack); err != nil {
		fail(err)
	}
	result["hashXML"] = marshaled
	result["hashBack"] = map[string]string{"value": hashBack.Value, "algorithmId": hashBack.AlgorithmId}

	var incomingH ms.RequestHash
	if err := xml.Unmarshal([]byte(incomingHash), &incomingH); err != nil {
		fail(err)
	}
	result["incomingHash"] = map[string]string{"value": incomingH.Value, "algorithmId": incomingH.AlgorithmId}

	var incoming ms.Client
	if err := xml.Unmarshal([]byte(incomingClient), &incoming); err != nil {
		fail(err)
	}
	result["incomingClient"] = incoming.ClientType

	var incomingRep ms.RepresentedParty
	if err := xml.Unmarshal([]byte(incomingParty), &incomingRep); err != nil {
		fail(err)
	}
	result["incomingParty"] = incomingRep.RepresentedPartyType

	var wrong ms.Client
	result["wrongNamespaceErr"] = fmt.Sprint(xml.Unmarshal([]byte(wrongNamespaceClient), &wrong))

	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fail(err)
	}
}
`

func TestGenerator_MultiSchema_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go run of generated code in short mode")
	}

	out := generate(t, multiSchemaFixture, WithPackage("ms"))

	dir := goModule(t, "ms", map[string]string{
		"ms.go":                 out.client(t),
		"cmd/roundtrip/main.go": multiSchemaRoundTripProgram,
	})
	stdout := []byte(runGo(t, dir, "run", "./cmd/roundtrip"))

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout, &result), "output: %s", stdout)

	t.Run("marshal emits the declaring schema namespace", func(t *testing.T) {
		assert.Contains(t, result["clientXML"], `<client xmlns="urn:ms:header">`)
		assert.Contains(t, result["partyXML"], `<representedParty xmlns="urn:ms:repr">`)
		assert.Contains(t, result["statusXML"], `<GetStatus xmlns="urn:ms:main">`)
		assert.Contains(t, result["pingXML"], `<Ping xmlns="urn:ms:main">`)
		assert.Contains(t, result["hashXML"], `<requestHash xmlns="urn:ms:header" algorithmId="SHA-512">abc==</requestHash>`)
	})

	t.Run("marshal then unmarshal preserves values", func(t *testing.T) {
		assert.Equal(t, map[string]any{"memberClass": "COM", "memberCode": "12345"}, result["clientBack"])
		assert.Equal(t, map[string]any{"partyClass": "COM", "partyCode": "70000000"}, result["partyBack"])
		assert.Equal(t, "38001010000", result["statusBack"])
		assert.Equal(t, "t-1", result["pingBack"])
		assert.Equal(t, map[string]any{"value": "abc==", "algorithmId": "SHA-512"}, result["hashBack"])
	})

	t.Run("unmarshal accepts elements in the declaring namespace", func(t *testing.T) {
		assert.Equal(t, map[string]any{"memberClass": "COM", "memberCode": "12345"}, result["incomingClient"])
		assert.Equal(t, map[string]any{"partyClass": "COM", "partyCode": "70000000"}, result["incomingParty"])
		assert.Equal(t, map[string]any{"value": "abc==", "algorithmId": "SHA-512"}, result["incomingHash"])
	})

	t.Run("unmarshal rejects elements in another namespace", func(t *testing.T) {
		assert.Contains(t, result["wrongNamespaceErr"], "expected element <client> in name space urn:ms:header")
	})
}

func TestGenerator_GlobalElementInlineSimpleContent(t *testing.T) {
	code := generate(t, multiSchemaFixture, WithPackage("ms")).client(t)

	body := structBody(t, code, "RequestHash")
	assert.Contains(t, body, "Value string `xml:\",chardata\" json:\"value\"`")
	assert.Contains(t, body, "AlgorithmId string `xml:\"algorithmId,attr,omitempty\" json:\"algorithmId,omitempty\"`")
}
