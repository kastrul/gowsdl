// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"encoding/json"
	"fmt"
	"mime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// soapVersionWSDL is a service whose port type is bound by the bindings in
// %[1]s; %[2]s is the port's address element.
const soapVersionWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:soap12="http://schemas.xmlsoap.org/wsdl/soap12/"
             xmlns:tns="urn:svc"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:svc">
    <types>
        <xsd:schema targetNamespace="urn:svc">
            <xsd:element name="EchoRequest">
                <xsd:complexType><xsd:sequence><xsd:element name="text" type="xsd:string"/></xsd:sequence></xsd:complexType>
            </xsd:element>
            <xsd:element name="EchoResponse">
                <xsd:complexType><xsd:sequence><xsd:element name="text" type="xsd:string"/></xsd:sequence></xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
    <message name="EchoIn"><part name="parameters" element="tns:EchoRequest"/></message>
    <message name="EchoOut"><part name="parameters" element="tns:EchoResponse"/></message>
    <portType name="EchoPort">
        <operation name="Echo"><input message="tns:EchoIn"/><output message="tns:EchoOut"/></operation>
    </portType>
    %[1]s
    <service name="EchoService">
        <port name="EchoPort" binding="tns:EchoBinding">%[2]s</port>
    </service>
</definitions>`

// soapVersionBinding returns a binding of EchoPort named name; prefix is
// "soap" or "soap12" and style "document" or "rpc".
func soapVersionBinding(name, prefix, style, action string) string {
	return fmt.Sprintf(`<binding name="%[1]s" type="tns:EchoPort">
        <%[2]s:binding style="%[3]s" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="Echo">
            <%[2]s:operation soapAction="%[4]s"/>
            <input><%[2]s:body use="literal" namespace="urn:op"/></input>
            <output><%[2]s:body use="literal" namespace="urn:op"/></output>
        </operation>
    </binding>`, name, prefix, style, action)
}

// soapVersionDriver calls Echo against a server that records the request
// and answers with the response body %[1]q of Content-Type %[2]q, then
// prints what the server received and the response text as JSON.
const soapVersionDriver = `package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/kastrul/gowsdl/soap"

	"soapver/svc"
)

func main() {
	var got struct {
		ContentType, SOAPAction, Envelope, Wrapper, Response string
		HasSOAPAction                                         bool
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.ContentType = r.Header.Get("Content-Type")
		got.SOAPAction = r.Header.Get("SOAPAction")
		_, got.HasSOAPAction = r.Header["Soapaction"]
		var env struct {
			XMLName xml.Name
			Body    struct {
				Content struct{ XMLName xml.Name } ` + "`xml:\",any\"`" + `
			} ` + "`xml:\"Body\"`" + `
		}
		data, _ := io.ReadAll(r.Body)
		_ = xml.Unmarshal(data, &env)
		got.Envelope = env.XMLName.Space
		got.Wrapper = "{" + env.Body.Content.XMLName.Space + "}" + env.Body.Content.XMLName.Local
		w.Header().Set("Content-Type", %[2]q)
		_, _ = io.WriteString(w, %[1]q)
	}))
	defer server.Close()

	resp, err := svc.NewEchoPort(soap.NewClient(server.URL)).Echo(context.Background(), &svc.EchoRequest{Text: "ping"})
	if err != nil {
		fmt.Fprintln(os.Stderr, "echo error:", err)
		os.Exit(1)
	}
	got.Response = resp.Text
	_ = json.NewEncoder(os.Stdout).Encode(got)
}
`

func TestGenerator_SOAPVersion(t *testing.T) {
	const (
		soap11 = "http://schemas.xmlsoap.org/soap/envelope/"
		soap12 = "http://www.w3.org/2003/05/soap-envelope"
	)
	soap11Address := `<soap:address location="http://localhost/echo"/>`
	soap12Address := `<soap12:address location="http://localhost/echo"/>`
	documentResponse := func(env string) string {
		return `<e:Envelope xmlns:e="` + env + `"><e:Body><EchoResponse xmlns="urn:svc"><text>pong</text></EchoResponse></e:Body></e:Envelope>`
	}
	rpcResponse := func(env string) string {
		return `<e:Envelope xmlns:e="` + env + `"><e:Body><EchoResponse xmlns="urn:op"><parameters xmlns=""><EchoResponse><text>pong</text></EchoResponse></parameters></EchoResponse></e:Body></e:Envelope>`
	}

	tests := []struct {
		name, bindings, address, response, responseType string
		wantEnvelope, wantWrapper                       string
		// wantSOAP12 says the request is SOAP 1.2: the action is in the
		// Content-Type rather than a SOAPAction header.
		wantSOAP12 bool
	}{
		{
			name:         "SOAP 1.2 RPC binding",
			bindings:     soapVersionBinding("EchoBinding", "soap12", "rpc", "urn:echo12"),
			address:      soap12Address,
			response:     rpcResponse(soap12),
			responseType: "application/soap+xml; charset=utf-8",
			wantEnvelope: soap12,
			wantWrapper:  "{urn:op}Echo",
			wantSOAP12:   true,
		},
		{
			name:         "SOAP 1.2 document binding",
			bindings:     soapVersionBinding("EchoBinding", "soap12", "document", "urn:echo12"),
			address:      soap12Address,
			response:     documentResponse(soap12),
			responseType: "application/soap+xml; charset=utf-8",
			wantEnvelope: soap12,
			wantWrapper:  "{urn:svc}EchoRequest",
			wantSOAP12:   true,
		},
		{
			name: "SOAP 1.1 and 1.2 bindings use SOAP 1.1",
			bindings: soapVersionBinding("EchoBinding", "soap", "rpc", "urn:echo11") +
				soapVersionBinding("EchoBinding12", "soap12", "rpc", "urn:echo12"),
			address:      soap11Address,
			response:     rpcResponse(soap11),
			responseType: "text/xml; charset=utf-8",
			wantEnvelope: soap11,
			wantWrapper:  "{urn:op}Echo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := generate(t, writeWSDL(t, fmt.Sprintf(soapVersionWSDL, tt.bindings, tt.address)), WithPackage("svc"), WithServerGeneration(true))
			assertGeneratedCodeBuilds(t, "soapver", out)

			driver := fmt.Sprintf(soapVersionDriver, tt.response, tt.responseType)
			dir := goModule(t, "soapver", map[string]string{"svc/client.go": out.client(t), "main.go": driver})
			output := runGo(t, dir, "run", ".")

			var got struct {
				ContentType, SOAPAction, Envelope, Wrapper, Response string
				HasSOAPAction                                        bool
			}
			require.NoError(t, json.Unmarshal([]byte(output), &got), output)
			assert.Equal(t, tt.wantEnvelope, got.Envelope)
			assert.Equal(t, tt.wantWrapper, got.Wrapper)
			assert.Equal(t, "pong", got.Response)

			mediaType, params, err := mime.ParseMediaType(got.ContentType)
			require.NoError(t, err)
			if tt.wantSOAP12 {
				assert.Equal(t, "application/soap+xml", mediaType)
				assert.Equal(t, "urn:echo12", params["action"])
				assert.False(t, got.HasSOAPAction)
			} else {
				assert.Equal(t, "text/xml", mediaType)
				assert.Equal(t, "urn:echo11", got.SOAPAction)
			}
		})
	}
}

func TestGenerator_SOAPVersion_Generics(t *testing.T) {
	for name, opt := range map[string]Option{"use generics": WithGenerics(), "generics only": WithGenericsOnly()} {
		t.Run(name, func(t *testing.T) {
			wsdl := fmt.Sprintf(soapVersionWSDL, soapVersionBinding("EchoBinding", "soap12", "document", "urn:echo12"),
				`<soap12:address location="http://localhost/echo"/>`)
			out := generate(t, writeWSDL(t, wsdl), WithPackage("svc"), opt)

			generic := "echoPort.Echo"
			if name == "use generics" {
				generic = "echoPort.EchoGeneric"
				assert.Contains(t, out.decl(t, "echoPort.Echo"), "ctx = soap.ContextWithSOAP12(ctx)")
			}
			assert.Contains(t, out.decl(t, generic), "ctx = soap.ContextWithSOAP12(ctx)")
			assert.Contains(t, out.decl(t, generic), `CallResult[EchoResponse](ctx, "urn:echo12", request)`)
		})
	}
}
