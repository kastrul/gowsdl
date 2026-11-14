// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rpcElementPartWSDL is an RPC-style service whose message parts are
// declared with elements rather than types. The %[1]s verb is the schema's
// attributes, %[2]s its declarations and %[3]s the extra parts of EchoIn.
const rpcElementPartWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="urn:rpcel"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:rpcel">
    <types>
        <xsd:schema targetNamespace="urn:rpcel"%[1]s>
            %[2]s
        </xsd:schema>
    </types>
    <message name="EchoIn"><part name="parameters" element="tns:EchoRequest"/>%[3]s</message>
    <message name="EchoOut"><part name="parameters" element="tns:EchoResponse"/></message>
    <portType name="EchoPort">
        <operation name="Echo"><input message="tns:EchoIn"/><output message="tns:EchoOut"/></operation>
    </portType>
    <binding name="EchoBinding" type="tns:EchoPort">
        <soap:binding style="rpc" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="Echo">
            <soap:operation soapAction="urn:echo"/>
            <input><soap:body use="literal" namespace="urn:rpcel"/></input>
            <output><soap:body use="literal" namespace="urn:rpcel"/></output>
        </operation>
    </binding>
    <service name="EchoService">
        <port name="EchoPort" binding="tns:EchoBinding"><soap:address location="http://localhost/echo"/></port>
    </service>
</definitions>`

const rpcInlineElements = `<xsd:element name="EchoRequest">
                <xsd:complexType><xsd:sequence><xsd:element name="text" type="xsd:string"/></xsd:sequence></xsd:complexType>
            </xsd:element>
            <xsd:element name="EchoResponse">
                <xsd:complexType><xsd:sequence><xsd:element name="text" type="xsd:string"/></xsd:sequence></xsd:complexType>
            </xsd:element>`

const rpcNamedTypeElements = `<xsd:complexType name="EchoRequestType"><xsd:sequence><xsd:element name="text" type="xsd:string"/></xsd:sequence></xsd:complexType>
            <xsd:complexType name="EchoResponseType"><xsd:sequence><xsd:element name="text" type="xsd:string"/></xsd:sequence></xsd:complexType>
            <xsd:element name="EchoRequest" type="tns:EchoRequestType"/>
            <xsd:element name="EchoResponse" type="tns:EchoResponseType"/>`

// rpcElementPartDriver calls Echo against a server that records the request
// and answers with responseBody, then prints the request and the response
// text.
const rpcElementPartDriver = `package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/kastrul/gowsdl/soap"

	"rpcel/svc"
)

func main() {
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		received = string(data)
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = io.WriteString(w, %q)
	}))
	defer server.Close()

	resp, err := svc.NewEchoPort(soap.NewClient(server.URL)).Echo(context.Background(), %s)
	if err != nil {
		fmt.Fprintln(os.Stderr, "echo error:", err)
		os.Exit(1)
	}
	fmt.Println(received)
	fmt.Println("RESPONSE " + %s)
}
`

func TestGenerator_RPCElementParts(t *testing.T) {
	const (
		soapNS = "{http://schemas.xmlsoap.org/soap/envelope/}"
		tns    = "{urn:rpcel}"
	)
	response := func(element string) string {
		return `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body>` +
			`<EchoResponse xmlns="urn:rpcel"><parameters>` + element + `</parameters></EchoResponse>` +
			`</soap:Body></soap:Envelope>`
	}

	tests := []struct {
		name         string
		schemaAttrs  string
		declarations string
		extraParts   string
		request      string
		responseText string
		response     string
		// wantRequest is the request envelope's elements, by qualified name.
		// Part accessors are in no namespace (WS-I Basic Profile R2735) and
		// local elements follow their schema's elementFormDefault.
		wantRequest []string
	}{
		{
			name:         "unqualified inline elements",
			declarations: rpcInlineElements,
			request:      `&svc.EchoRequest{Text: "ping"}`,
			responseText: "resp.Text",
			response:     response(`<EchoResponse><text>pong</text></EchoResponse>`),
			wantRequest: []string{soapNS + "Envelope", soapNS + "Body", tns + "Echo",
				"{}parameters", tns + "EchoRequest", "{}text"},
		},
		{
			name:         "qualified inline elements",
			schemaAttrs:  ` elementFormDefault="qualified"`,
			declarations: rpcInlineElements,
			request:      `&svc.EchoRequest{Text: "ping"}`,
			responseText: "resp.Text",
			response:     response(`<EchoResponse xmlns="urn:rpcel"><text>pong</text></EchoResponse>`),
			wantRequest: []string{soapNS + "Envelope", soapNS + "Body", tns + "Echo",
				"{}parameters", tns + "EchoRequest", tns + "text"},
		},
		{
			name:         "elements of named types",
			declarations: rpcNamedTypeElements,
			request:      `&svc.EchoRequest{EchoRequestType: svc.EchoRequestType{Text: "ping"}}`,
			responseText: "resp.Text",
			response:     response(`<EchoResponse><text>pong</text></EchoResponse>`),
			wantRequest: []string{soapNS + "Envelope", soapNS + "Body", tns + "Echo",
				"{}parameters", tns + "EchoRequest", "{}text"},
		},
		{
			name:         "message with two element parts",
			declarations: rpcInlineElements + `<xsd:element name="Trace"><xsd:complexType><xsd:sequence><xsd:element name="id" type="xsd:string"/></xsd:sequence></xsd:complexType></xsd:element>`,
			extraParts:   `<part name="trace" element="tns:Trace"/>`,
			request:      `&svc.EchoIn{Parameters: svc.EchoRequest{Text: "ping"}, Trace: svc.Trace{Id: "t1"}}`,
			responseText: "resp.Text",
			response:     response(`<EchoResponse><text>pong</text></EchoResponse>`),
			wantRequest: []string{soapNS + "Envelope", soapNS + "Body", tns + "Echo",
				"{}parameters", tns + "EchoRequest", "{}text",
				"{}trace", tns + "Trace", "{}id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wsdl := fmt.Sprintf(rpcElementPartWSDL, tt.schemaAttrs, tt.declarations, tt.extraParts)
			out := generate(t, writeWSDL(t, wsdl), WithPackage("svc"), WithServerGeneration(true))
			assertGeneratedCodeBuilds(t, "rpcel", out)

			driver := fmt.Sprintf(rpcElementPartDriver, tt.response, tt.request, tt.responseText)
			dir := goModule(t, "rpcel", map[string]string{"svc/client.go": out.client(t), "main.go": driver})
			output := runGo(t, dir, "run", ".")

			request, resp, ok := strings.Cut(strings.TrimSpace(output), "\nRESPONSE ")
			require.True(t, ok, "driver output:\n%s", output)
			assert.Equal(t, tt.wantRequest, xmlNodes(t, request), "request:\n%s", request)
			assert.Equal(t, "pong", resp)
		})
	}
}

// rpcTypePartWSDL is an RPC-style service whose message parts are declared
// with types. The %[1]s verb is the schema's attributes.
const rpcTypePartWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="urn:rpcel"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:rpcel">
    <types>
        <xsd:schema targetNamespace="urn:rpcel"%[1]s>
            <xsd:complexType name="Item"><xsd:sequence><xsd:element name="text" type="xsd:string"/></xsd:sequence></xsd:complexType>
        </xsd:schema>
    </types>
    <message name="EchoIn"><part name="item" type="tns:Item"/><part name="count" type="xsd:int"/></message>
    <message name="EchoOut"><part name="item" type="tns:Item"/></message>
    <portType name="EchoPort">
        <operation name="Echo"><input message="tns:EchoIn"/><output message="tns:EchoOut"/></operation>
    </portType>
    <binding name="EchoBinding" type="tns:EchoPort">
        <soap:binding style="rpc" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="Echo">
            <soap:operation soapAction="urn:echo"/>
            <input><soap:body use="literal" namespace="urn:rpcel"/></input>
            <output><soap:body use="literal" namespace="urn:rpcel"/></output>
        </operation>
    </binding>
    <service name="EchoService">
        <port name="EchoPort" binding="tns:EchoBinding"><soap:address location="http://localhost/echo"/></port>
    </service>
</definitions>`

func TestGenerator_RPCTypeParts(t *testing.T) {
	const (
		soapNS = "{http://schemas.xmlsoap.org/soap/envelope/}"
		tns    = "{urn:rpcel}"
	)

	tests := []struct {
		name        string
		schemaAttrs string
		response    string
		// wantRequest is the request envelope's elements, by qualified name.
		// Part accessors are in no namespace (WS-I Basic Profile R2735) and
		// the fields of a complex type follow its schema's elementFormDefault.
		wantRequest []string
	}{
		{
			name:     "unqualified schema",
			response: `<item><text>pong</text></item>`,
			wantRequest: []string{soapNS + "Envelope", soapNS + "Body", tns + "Echo",
				"{}item", "{}text", "{}count"},
		},
		{
			name:        "qualified schema",
			schemaAttrs: ` elementFormDefault="qualified"`,
			response:    `<item><text xmlns="urn:rpcel">pong</text></item>`,
			wantRequest: []string{soapNS + "Envelope", soapNS + "Body", tns + "Echo",
				"{}item", tns + "text", "{}count"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := generate(t, writeWSDL(t, fmt.Sprintf(rpcTypePartWSDL, tt.schemaAttrs)), WithPackage("svc"), WithServerGeneration(true))
			assertGeneratedCodeBuilds(t, "rpcel", out)

			response := `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body>` +
				`<EchoResponse xmlns="urn:rpcel">` + tt.response + `</EchoResponse></soap:Body></soap:Envelope>`
			driver := fmt.Sprintf(rpcElementPartDriver, response, `&svc.EchoIn{Item: svc.Item{Text: "ping"}, Count: 3}`, "resp.Item.Text")
			dir := goModule(t, "rpcel", map[string]string{"svc/client.go": out.client(t), "main.go": driver})
			output := runGo(t, dir, "run", ".")

			request, resp, ok := strings.Cut(strings.TrimSpace(output), "\nRESPONSE ")
			require.True(t, ok, "driver output:\n%s", output)
			assert.Equal(t, tt.wantRequest, xmlNodes(t, request), "request:\n%s", request)
			assert.Contains(t, request, ">ping<")
			assert.Contains(t, request, ">3<")
			assert.Equal(t, "pong", resp)
		})
	}
}
