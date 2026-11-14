// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// oneWayWSDL11 has a one-way operation, Notify, next to a request-response
// operation, Echo; style is the SOAP binding style.
const oneWayWSDL11 = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="urn:oneway"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:oneway">
    <types>
        <xsd:schema targetNamespace="urn:oneway">
            <xsd:element name="NotifyRequest">
                <xsd:complexType><xsd:sequence><xsd:element name="event" type="xsd:string"/></xsd:sequence></xsd:complexType>
            </xsd:element>
            <xsd:element name="EchoRequest">
                <xsd:complexType><xsd:sequence><xsd:element name="text" type="xsd:string"/></xsd:sequence></xsd:complexType>
            </xsd:element>
            <xsd:element name="EchoResponse">
                <xsd:complexType><xsd:sequence><xsd:element name="text" type="xsd:string"/></xsd:sequence></xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
    <message name="NotifyIn"><part name="parameters" element="tns:NotifyRequest"/></message>
    <message name="EchoIn"><part name="parameters" element="tns:EchoRequest"/></message>
    <message name="EchoOut"><part name="parameters" element="tns:EchoResponse"/></message>
    <portType name="EventPort">
        <operation name="Notify"><input message="tns:NotifyIn"/></operation>
        <operation name="Echo"><input message="tns:EchoIn"/><output message="tns:EchoOut"/></operation>
    </portType>
    <binding name="EventBinding" type="tns:EventPort">
        <soap:binding style="%s" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="Notify">
            <soap:operation soapAction="urn:notify"/>
            <input><soap:body use="literal" namespace="urn:oneway"/></input>
        </operation>
        <operation name="Echo">
            <soap:operation soapAction="urn:echo"/>
            <input><soap:body use="literal" namespace="urn:oneway"/></input>
            <output><soap:body use="literal" namespace="urn:oneway"/></output>
        </operation>
    </binding>
    <service name="EventService">
        <port name="EventPort" binding="tns:EventBinding"><soap:address location="http://localhost/events"/></port>
    </service>
</definitions>`

// oneWayRPCWSDL11 is oneWayWSDL11 in RPC style, whose message parts have
// types rather than elements.
const oneWayRPCWSDL11 = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="urn:oneway"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:oneway">
    <message name="NotifyIn"><part name="event" type="xsd:string"/></message>
    <message name="EchoIn"><part name="text" type="xsd:string"/></message>
    <message name="EchoOut"><part name="text" type="xsd:string"/></message>
    <portType name="EventPort">
        <operation name="Notify"><input message="tns:NotifyIn"/></operation>
        <operation name="Echo"><input message="tns:EchoIn"/><output message="tns:EchoOut"/></operation>
    </portType>
    <binding name="EventBinding" type="tns:EventPort">
        <soap:binding style="rpc" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="Notify">
            <soap:operation soapAction="urn:notify"/>
            <input><soap:body use="literal" namespace="urn:oneway"/></input>
        </operation>
        <operation name="Echo">
            <soap:operation soapAction="urn:echo"/>
            <input><soap:body use="literal" namespace="urn:oneway"/></input>
            <output><soap:body use="literal" namespace="urn:oneway"/></output>
        </operation>
    </binding>
    <service name="EventService">
        <port name="EventPort" binding="tns:EventBinding"><soap:address location="http://localhost/events"/></port>
    </service>
</definitions>`

// oneWayWSDL20 has an in-only operation, Notify, next to an in-out
// operation, Echo.
const oneWayWSDL20 = `<?xml version="1.0" encoding="UTF-8"?>
<description xmlns="http://www.w3.org/ns/wsdl"
             xmlns:tns="urn:oneway"
             xmlns:wsoap="http://www.w3.org/ns/wsdl/soap"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:oneway">
    <types>
        <xsd:schema targetNamespace="urn:oneway">
            <xsd:element name="NotifyRequest">
                <xsd:complexType><xsd:sequence><xsd:element name="event" type="xsd:string"/></xsd:sequence></xsd:complexType>
            </xsd:element>
            <xsd:element name="EchoRequest">
                <xsd:complexType><xsd:sequence><xsd:element name="text" type="xsd:string"/></xsd:sequence></xsd:complexType>
            </xsd:element>
            <xsd:element name="EchoResponse">
                <xsd:complexType><xsd:sequence><xsd:element name="text" type="xsd:string"/></xsd:sequence></xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
    <interface name="EventPort">
        <operation name="Notify" pattern="http://www.w3.org/ns/wsdl/in-only">
            <input element="tns:NotifyRequest"/>
        </operation>
        <operation name="Echo" pattern="http://www.w3.org/ns/wsdl/in-out">
            <input element="tns:EchoRequest"/>
            <output element="tns:EchoResponse"/>
        </operation>
    </interface>
    <binding name="EventBinding" interface="tns:EventPort" type="http://www.w3.org/ns/wsdl/soap">
        <operation ref="tns:Notify" wsoap:action="urn:notify"/>
        <operation ref="tns:Echo" wsoap:action="urn:echo"/>
    </binding>
    <service name="EventService" interface="tns:EventPort">
        <endpoint name="EventEndpoint" binding="tns:EventBinding" address="http://localhost/events"/>
    </service>
</description>`

// oneWayDriver calls Notify against a server that answers 202 Accepted with
// no body, as WS-I Basic Profile R2750 has one-way receivers do, and Echo
// against one that answers with a response, and prints what it saw.
const oneWayDriver = `package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/kastrul/gowsdl/soap"

	"oneway/svc"
)

func main() {
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		received = string(data)
		if r.Header.Get("SOAPAction") == "\"urn:notify\"" || r.Header.Get("SOAPAction") == "urn:notify" {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = io.WriteString(w, ` + "`" + `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><EchoResponse xmlns="urn:oneway"><text>pong</text></EchoResponse></soap:Body></soap:Envelope>` + "`" + `)
	}))
	defer server.Close()

	port := svc.NewEventPort(soap.NewClient(server.URL))
	if err := port.Notify(context.Background(), &svc.NotifyRequest{Event: "started"}); err != nil {
		fmt.Println("notify error:", err)
		os.Exit(1)
	}
	fmt.Println("notify sent:", received)

	resp, err := port.Echo(context.Background(), &svc.EchoRequest{Text: "ping"})
	if err != nil {
		fmt.Println("echo error:", err)
		os.Exit(1)
	}
	fmt.Println("echo:", resp.Text)
}
`

func TestGenerator_OneWayOperation(t *testing.T) {
	tests := []struct {
		name string
		wsdl string
		opts []Option
		// notifyMethod makes notifyCall; WSDL 2.0 clients delegate Notify
		// to NotifyContext.
		notifyMethod, notifyCall string
	}{
		{
			name:       "WSDL 1.1 document",
			wsdl:       strings.Replace(oneWayWSDL11, "%s", "document", 1),
			notifyCall: `return service.client.CallContext(ctx, "urn:notify", request, nil)`,
		},
		{
			name:       "WSDL 1.1 document with generics",
			wsdl:       strings.Replace(oneWayWSDL11, "%s", "document", 1),
			opts:       []Option{WithGenerics()},
			notifyCall: `return service.client.CallContext(ctx, "urn:notify", request, nil)`,
		},
		{
			name:       "WSDL 1.1 document with generics only",
			wsdl:       strings.Replace(oneWayWSDL11, "%s", "document", 1),
			opts:       []Option{WithGenericsOnly()},
			notifyCall: `return service.client.CallContext(ctx, "urn:notify", request, nil)`,
		},
		{
			name:       "WSDL 1.1 RPC",
			wsdl:       oneWayRPCWSDL11,
			notifyCall: `return service.client.CallContext(ctx, "urn:notify", α, nil)`,
		},
		{
			name:         "WSDL 2.0 in-only",
			wsdl:         oneWayWSDL20,
			notifyMethod: "NotifyContext",
			notifyCall:   `return service.client.CallContext(ctx, "urn:notify", request, nil)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := generate(t, writeWSDL(t, tt.wsdl), append([]Option{WithPackage("svc")}, tt.opts...)...)
			ops := strings.Join(strings.Fields(out.decl(t, "EventPort")), " ")

			assert.Regexp(t, `Notify\(ctx context\.Context, request \*\w+\) error`, ops)
			assert.NotContains(t, ops, "NotifyGeneric", "a one-way operation has no response for a generic result")
			assert.NotContains(t, out.operations(), "Result[]")

			method := tt.notifyMethod
			if method == "" {
				method = "Notify"
			}
			notify := strings.Join(strings.Fields(out.decl(t, "eventPort."+method)), " ")
			assert.Regexp(t, `^func \(service \*eventPort\) `+method+`\(ctx context\.Context, request \*\w+\) error \{`, notify)
			assert.Contains(t, notify, tt.notifyCall)
			if tt.notifyMethod != "" {
				assert.Contains(t, ops, tt.notifyMethod+"(ctx context.Context, request *NotifyRequest) error")
			}
		})
	}
}

func TestGenerator_OneWayOperation_Runs(t *testing.T) {
	for name, wsdl := range map[string]string{
		"WSDL 1.1 document": strings.Replace(oneWayWSDL11, "%s", "document", 1),
		"WSDL 2.0 in-only":  oneWayWSDL20,
	} {
		t.Run(name, func(t *testing.T) {
			out := generate(t, writeWSDL(t, wsdl), WithPackage("svc"))
			dir := goModule(t, "oneway", map[string]string{"svc/client.go": out.client(t), "main.go": oneWayDriver})
			output := runGo(t, dir, "run", ".")

			assert.Contains(t, output, "notify sent:")
			assert.Contains(t, output, "<event>started</event>")
			assert.Contains(t, output, "echo: pong")
		})
	}
}

func TestGenerator_OneWayOperation_ClientAndServerBuild(t *testing.T) {
	for name, wsdl := range map[string]string{
		"WSDL 1.1 document": strings.Replace(oneWayWSDL11, "%s", "document", 1),
		"WSDL 1.1 RPC":      oneWayRPCWSDL11,
		"WSDL 2.0 in-only":  oneWayWSDL20,
	} {
		t.Run(name, func(t *testing.T) {
			assertGeneratedCodeBuilds(t, "oneway", generate(t, writeWSDL(t, wsdl), WithPackage("svc"), WithServerGeneration(true)))
		})
	}
}
