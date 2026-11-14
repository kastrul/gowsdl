// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// xmlNameWant is what a generated type must declare: its XMLName struct tag,
// or none when empty, and the namespace its TargetNamespace method returns,
// or no such method when empty.
type xmlNameWant struct {
	xmlName, targetNamespace string
}

// TestGenerator_XMLNameAndTargetNamespace checks the XMLName field and
// TargetNamespace method of every type generated for each scenario:
//
//   - a global element of an unqualified schema gets XMLName "tns:<element>"
//     when it is a request and "<element>" when it is a response, and a
//     TargetNamespace method returning the schema's target namespace;
//   - a global element of a qualified schema gets XMLName "<namespace>
//     <element>" and no TargetNamespace method;
//   - complex types, named or anonymous, get neither;
//   - no type has more than one XMLName.
//
// Types not listed in a scenario must have neither.
func TestGenerator_XMLNameAndTargetNamespace(t *testing.T) {
	tests := []struct {
		name  string
		wsdl  string
		pkg   string
		types map[string]xmlNameWant
	}{
		{
			name: "unqualified request with nested inline types",
			wsdl: unqualifiedInlineChildrenWSDL,
			pkg:  "service",
			types: map[string]xmlNameWant{
				"GOWebService_SendungsErstellung":         {xmlName: `xml:"tns:GOWebService_SendungsErstellung"`, targetNamespace: "https://wsdemo.ax4.com/ws/GeneralOvernight"},
				"GOWebService_SendungsErstellungResponse": {xmlName: `xml:"GOWebService_SendungsErstellungResponse"`, targetNamespace: "https://wsdemo.ax4.com/ws/GeneralOvernight"},
			},
		},
		{
			name: "unqualified schema, elementFormDefault omitted",
			wsdl: unqualifiedOrderWSDL,
			pkg:  "service",
			types: map[string]xmlNameWant{
				"CreateOrder":                  {xmlName: `xml:"tns:CreateOrder"`, targetNamespace: "https://example.com/service"},
				"CreateOrderResponse":          {xmlName: `xml:"CreateOrderResponse"`, targetNamespace: "https://example.com/service"},
				"CreateOrderItemsType":         {},
				"CreateOrderItemsTypeItemType": {},
			},
		},
		{
			name: "qualified schema without messages",
			wsdl: qualifiedNoMessagesWSDL,
			pkg:  "service",
			types: map[string]xmlNameWant{
				"CreateOrder": {xmlName: `xml:"https://example.com/service CreateOrder"`},
			},
		},
		{
			name: "unqualified request and response",
			wsdl: unqualifiedRequestResponseWSDL,
			pkg:  "service",
			types: map[string]xmlNameWant{
				"CreateOrder":         {xmlName: `xml:"tns:CreateOrder"`, targetNamespace: "https://example.com/service"},
				"CreateOrderResponse": {xmlName: `xml:"CreateOrderResponse"`, targetNamespace: "https://example.com/service"},
			},
		},
		{
			name: "unqualified elements of named types",
			wsdl: unqualifiedNamedTypesWSDL,
			pkg:  "test",
			types: map[string]xmlNameWant{
				"TestRequest":      {xmlName: `xml:"tns:TestRequest"`, targetNamespace: "http://example.com/test"},
				"TestResponse":     {xmlName: `xml:"TestResponse"`, targetNamespace: "http://example.com/test"},
				"TestRequestType":  {},
				"TestResponseType": {},
			},
		},
		{
			name: "qualified elements of named types",
			wsdl: qualifiedNamedTypesWSDL,
			pkg:  "test",
			types: map[string]xmlNameWant{
				"QualifiedRequest":      {xmlName: `xml:"http://example.com/qualified QualifiedRequest"`},
				"QualifiedResponse":     {xmlName: `xml:"http://example.com/qualified QualifiedResponse"`},
				"QualifiedRequestType":  {},
				"QualifiedResponseType": {},
			},
		},
		{
			name: "one-way operation and an unused complex type",
			wsdl: oneWayUnusedTypeWSDL,
			pkg:  "test",
			types: map[string]xmlNameWant{
				"UsedRequest":     {xmlName: `xml:"tns:UsedRequest"`, targetNamespace: "http://example.com/unused"},
				"UnusedType":      {},
				"UsedRequestType": {},
			},
		},
		{
			name: "lower-case element names",
			wsdl: checkV1WSDL,
			pkg:  "test",
			types: map[string]xmlNameWant{
				"CheckV1":         {xmlName: `xml:"tns:checkV1"`, targetNamespace: "http://b2b.also.com/ActWebServices.PriceAvailability"},
				"CheckV1Response": {xmlName: `xml:"checkV1Response"`, targetNamespace: "http://b2b.also.com/ActWebServices.PriceAvailability"},
				"CheckV1Type":     {},
			},
		},
		{
			name: "request and response with a SOAP binding",
			wsdl: echoWSDL,
			pkg:  "test",
			types: map[string]xmlNameWant{
				"Echo":         {xmlName: `xml:"tns:Echo"`, targetNamespace: "http://integration.test/soap"},
				"EchoResponse": {xmlName: `xml:"EchoResponse"`, targetNamespace: "http://integration.test/soap"},
			},
		},
		{
			name: "complex types not used in messages",
			wsdl: nonMessageTypesWSDL,
			pkg:  "test",
			types: map[string]xmlNameWant{
				"UsedElement":     {xmlName: `xml:"tns:UsedElement"`, targetNamespace: "http://example.com/types"},
				"UsedType":        {},
				"UnusedType":      {},
				"NestedContainer": {},
			},
		},
		{
			name: "global element and type not used in messages",
			wsdl: nonMessageElementWSDL,
			pkg:  "testpkg",
			types: map[string]xmlNameWant{
				"CheckV1":           {xmlName: `xml:"tns:checkV1"`, targetNamespace: "http://b2b.also.com/ActWebServices.PriceAvailability"},
				"CheckV1Response":   {xmlName: `xml:"checkV1Response"`, targetNamespace: "http://b2b.also.com/ActWebServices.PriceAvailability"},
				"NonMessageElement": {xmlName: `xml:"tns:nonMessageElement"`, targetNamespace: "http://b2b.also.com/ActWebServices.PriceAvailability"},
				"NonMessageType":    {},
			},
		},
		{
			name: "qualified element with an inline type",
			wsdl: qualifiedInlineWSDL,
			pkg:  "testpkg",
			types: map[string]xmlNameWant{
				"QualifiedRequest":     {xmlName: `xml:"http://example.com/qualified qualifiedRequest"`},
				"QualifiedRequestType": {},
			},
		},
		{
			name: "unqualified schema without messages",
			wsdl: noMessagesWSDL,
			pkg:  "testpkg",
			types: map[string]xmlNameWant{
				"StandaloneElement": {xmlName: `xml:"tns:standaloneElement"`, targetNamespace: "http://example.com/test"},
				"StandaloneType":    {},
			},
		},
		{
			name: "responses named unlike their requests",
			wsdl: responseElementsWSDL,
			pkg:  "service",
			types: map[string]xmlNameWant{
				"SendungsErstellung":   {xmlName: `xml:"tns:SendungsErstellung"`, targetNamespace: "https://example.com/service"},
				"SendungsRueckmeldung": {xmlName: `xml:"SendungsRueckmeldung"`, targetNamespace: "https://example.com/service"},
				"GetOrder":             {xmlName: `xml:"tns:GetOrder"`, targetNamespace: "https://example.com/service"},
				"GetOrderResponse":     {xmlName: `xml:"GetOrderResponse"`, targetNamespace: "https://example.com/service"},
			},
		},
		{
			name: "WSDL 2.0 request and response",
			wsdl: wsdl2RequestResponseWSDL,
			pkg:  "service",
			types: map[string]xmlNameWant{
				"GetCustomer":         {xmlName: `xml:"tns:GetCustomer"`, targetNamespace: "https://example.com/service"},
				"GetCustomerResponse": {xmlName: `xml:"GetCustomerResponse"`, targetNamespace: "https://example.com/service"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := generate(t, writeWSDL(t, tt.wsdl), WithPackage(tt.pkg))

			methods := make(map[string]bool)
			for _, name := range out.declNames(t) {
				methods[name] = true
			}
			for name := range tt.types {
				assert.Contains(t, out.typeNames(t), name)
			}
			for _, name := range out.typeNames(t) {
				want := tt.types[name]
				decl := strings.Join(strings.Fields(out.decl(t, name)), " ")

				assert.LessOrEqual(t, strings.Count(decl, "XMLName"), 1, "%s has more than one XMLName:\n%s", name, decl)
				if want.xmlName == "" {
					assert.NotContains(t, decl, "XMLName", "%s should have no XMLName", name)
				} else {
					assert.Contains(t, decl, "XMLName xml.Name `"+want.xmlName+"`", name)
				}

				if want.targetNamespace == "" {
					assert.False(t, methods[name+".TargetNamespace"], "%s should have no TargetNamespace method", name)
				} else if assert.True(t, methods[name+".TargetNamespace"], "%s should have a TargetNamespace method", name) {
					assert.Equal(t, "func (t "+name+") TargetNamespace() string { return \""+want.targetNamespace+"\" }",
						strings.Join(strings.Fields(out.decl(t, name+".TargetNamespace")), " "))
				}
			}
		})
	}
}

// unqualifiedInlineChildrenWSDL is from TestCompleteNamespaceHandling in namespace_complete_test.go.
const unqualifiedInlineChildrenWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="https://wsdemo.ax4.com/ws/GeneralOvernight"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="https://wsdemo.ax4.com/ws/GeneralOvernight">
    <types>
        <xsd:schema targetNamespace="https://wsdemo.ax4.com/ws/GeneralOvernight">
            <xsd:element name="GOWebService_SendungsErstellung">
                <xsd:complexType>
                    <xsd:sequence>
                        <xsd:element name="Versender" type="xsd:string" minOccurs="0"/>
                        <xsd:element name="Benutzername" type="xsd:string" minOccurs="0"/>
                        <xsd:element name="Status" type="xsd:string" minOccurs="0"/>
                        <xsd:element name="Kundenreferenz" type="xsd:string" minOccurs="0"/>
                        <xsd:element name="Ansprechpartner" minOccurs="0" maxOccurs="1">
                            <xsd:complexType>
                                <xsd:sequence>
                                    <xsd:element name="Telefon" minOccurs="1" maxOccurs="1">
                                        <xsd:complexType>
                                            <xsd:sequence>
                                                <xsd:element name="LaenderPrefix" type="xsd:string" minOccurs="1" maxOccurs="1"/>
                                                <xsd:element name="Vorwahl" type="xsd:string" minOccurs="1" maxOccurs="1"/>
                                                <xsd:element name="Telefonnummer" type="xsd:string" minOccurs="1" maxOccurs="1"/>
                                            </xsd:sequence>
                                        </xsd:complexType>
                                    </xsd:element>
                                </xsd:sequence>
                            </xsd:complexType>
                        </xsd:element>
                    </xsd:sequence>
                </xsd:complexType>
            </xsd:element>
            
            <xsd:element name="GOWebService_SendungsErstellungResponse">
                <xsd:complexType>
                    <xsd:sequence>
                        <xsd:element name="status" type="xsd:string"/>
                    </xsd:sequence>
                </xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
    
    <message name="SendungsErstellungRequest">
        <part name="parameters" element="tns:GOWebService_SendungsErstellung"/>
    </message>
    
    <message name="SendungsErstellungResponse">
        <part name="parameters" element="tns:GOWebService_SendungsErstellungResponse"/>
    </message>
    
    <portType name="GeneralOvernightService">
        <operation name="SendungsErstellung">
            <input message="tns:SendungsErstellungRequest"/>
            <output message="tns:SendungsErstellungResponse"/>
        </operation>
    </portType>
    
    <binding name="GeneralOvernightServiceBinding" type="tns:GeneralOvernightService">
        <soap:binding transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="SendungsErstellung">
            <soap:operation soapAction="SendungsErstellung"/>
            <input>
                <soap:body use="literal"/>
            </input>
            <output>
                <soap:body use="literal"/>
            </output>
        </operation>
    </binding>
    
    <service name="GeneralOvernightService">
        <port name="GeneralOvernightServicePort" binding="tns:GeneralOvernightServiceBinding">
            <soap:address location="https://example.com/service"/>
        </port>
    </service>
</definitions>`

// unqualifiedOrderWSDL is from TestElementFormDefaultUnqualified in namespace_qualification_test.go.
const unqualifiedOrderWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="https://example.com/service"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="https://example.com/service">
    <types>
        <schema xmlns="http://www.w3.org/2001/XMLSchema"
                targetNamespace="https://example.com/service">
            <!-- Note: elementFormDefault is NOT specified, defaults to "unqualified" -->
            
            <element name="CreateOrder">
                <complexType>
                    <sequence>
                        <element name="customerID" type="string"/>
                        <element name="orderDate" type="date"/>
                        <element name="items">
                            <complexType>
                                <sequence>
                                    <element name="item" maxOccurs="unbounded">
                                        <complexType>
                                            <sequence>
                                                <element name="productCode" type="string"/>
                                                <element name="quantity" type="int"/>
                                            </sequence>
                                        </complexType>
                                    </element>
                                </sequence>
                            </complexType>
                        </element>
                    </sequence>
                </complexType>
            </element>
            
            <element name="CreateOrderResponse">
                <complexType>
                    <sequence>
                        <element name="orderID" type="string"/>
                        <element name="status" type="string"/>
                    </sequence>
                </complexType>
            </element>
        </schema>
    </types>
    
    <message name="CreateOrderRequest">
        <part name="parameters" element="tns:CreateOrder"/>
    </message>
    
    <message name="CreateOrderResponse">
        <part name="parameters" element="tns:CreateOrderResponse"/>
    </message>
    
    <portType name="OrderService">
        <operation name="CreateOrder">
            <input message="tns:CreateOrderRequest"/>
            <output message="tns:CreateOrderResponse"/>
        </operation>
    </portType>
</definitions>`

// qualifiedNoMessagesWSDL is from TestElementFormDefaultQualified in namespace_qualification_test.go.
const qualifiedNoMessagesWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="https://example.com/service"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="https://example.com/service">
    <types>
        <schema xmlns="http://www.w3.org/2001/XMLSchema"
                targetNamespace="https://example.com/service"
                elementFormDefault="qualified">
            
            <element name="CreateOrder">
                <complexType>
                    <sequence>
                        <element name="customerID" type="string"/>
                        <element name="orderDate" type="date"/>
                    </sequence>
                </complexType>
            </element>
        </schema>
    </types>
</definitions>`

// unqualifiedRequestResponseWSDL is from TestSOAPNamespaceHandling in soap_namespace_test.go.
const unqualifiedRequestResponseWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="https://example.com/service"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="https://example.com/service">
    <types>
        <schema xmlns="http://www.w3.org/2001/XMLSchema"
                targetNamespace="https://example.com/service">
            
            <element name="CreateOrder">
                <complexType>
                    <sequence>
                        <element name="customerID" type="string"/>
                        <element name="orderDate" type="date"/>
                    </sequence>
                </complexType>
            </element>
            
            <element name="CreateOrderResponse">
                <complexType>
                    <sequence>
                        <element name="orderID" type="string"/>
                    </sequence>
                </complexType>
            </element>
        </schema>
    </types>
    
    <message name="CreateOrderRequest">
        <part name="parameters" element="tns:CreateOrder"/>
    </message>
    
    <message name="CreateOrderResponse">
        <part name="parameters" element="tns:CreateOrderResponse"/>
    </message>
    
    <portType name="OrderService">
        <operation name="CreateOrder">
            <input message="tns:CreateOrderRequest"/>
            <output message="tns:CreateOrderResponse"/>
        </operation>
    </portType>
    
    <binding name="OrderServiceBinding" type="tns:OrderService">
        <soap:binding transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="CreateOrder">
            <soap:operation soapAction="CreateOrder"/>
            <input>
                <soap:body use="literal"/>
            </input>
            <output>
                <soap:body use="literal"/>
            </output>
        </operation>
    </binding>
    
    <service name="OrderService">
        <port name="OrderServicePort" binding="tns:OrderServiceBinding">
            <soap:address location="http://example.com/service"/>
        </port>
    </service>
</definitions>`

// unqualifiedNamedTypesWSDL is from TestTargetNamespaceMethodGeneration in target_namespace_test.go.
const unqualifiedNamedTypesWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://example.com/test"
             targetNamespace="http://example.com/test">
    <types>
        <schema xmlns="http://www.w3.org/2001/XMLSchema"
                targetNamespace="http://example.com/test"
                elementFormDefault="unqualified">
            <element name="TestRequest" type="tns:TestRequestType"/>
            <element name="TestResponse" type="tns:TestResponseType"/>
            <complexType name="TestRequestType">
                <sequence>
                    <element name="input" type="string"/>
                </sequence>
            </complexType>
            <complexType name="TestResponseType">
                <sequence>
                    <element name="output" type="string"/>
                </sequence>
            </complexType>
        </schema>
    </types>
    <message name="TestRequestMessage">
        <part name="body" element="tns:TestRequest"/>
    </message>
    <message name="TestResponseMessage">
        <part name="body" element="tns:TestResponse"/>
    </message>
    <portType name="TestPortType">
        <operation name="Test">
            <input message="tns:TestRequestMessage"/>
            <output message="tns:TestResponseMessage"/>
        </operation>
    </portType>
</definitions>`

// qualifiedNamedTypesWSDL is from TestTargetNamespaceMethodGeneration in target_namespace_test.go.
const qualifiedNamedTypesWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://example.com/qualified"
             targetNamespace="http://example.com/qualified">
    <types>
        <schema xmlns="http://www.w3.org/2001/XMLSchema"
                targetNamespace="http://example.com/qualified"
                elementFormDefault="qualified">
            <element name="QualifiedRequest" type="tns:QualifiedRequestType"/>
            <element name="QualifiedResponse" type="tns:QualifiedResponseType"/>
            <complexType name="QualifiedRequestType">
                <sequence>
                    <element name="data" type="string"/>
                </sequence>
            </complexType>
            <complexType name="QualifiedResponseType">
                <sequence>
                    <element name="result" type="string"/>
                </sequence>
            </complexType>
        </schema>
    </types>
    <message name="QualifiedRequestMessage">
        <part name="body" element="tns:QualifiedRequest"/>
    </message>
    <message name="QualifiedResponseMessage">
        <part name="body" element="tns:QualifiedResponse"/>
    </message>
    <portType name="QualifiedPortType">
        <operation name="Process">
            <input message="tns:QualifiedRequestMessage"/>
            <output message="tns:QualifiedResponseMessage"/>
        </operation>
    </portType>
</definitions>`

// oneWayUnusedTypeWSDL is from TestTargetNamespaceMethodGeneration in target_namespace_test.go.
const oneWayUnusedTypeWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://example.com/unused"
             targetNamespace="http://example.com/unused">
    <types>
        <schema xmlns="http://www.w3.org/2001/XMLSchema"
                targetNamespace="http://example.com/unused"
                elementFormDefault="unqualified">
            <complexType name="UnusedType">
                <sequence>
                    <element name="field" type="string"/>
                </sequence>
            </complexType>
            <element name="UsedRequest" type="tns:UsedRequestType"/>
            <complexType name="UsedRequestType">
                <sequence>
                    <element name="input" type="string"/>
                </sequence>
            </complexType>
        </schema>
    </types>
    <message name="UsedRequestMessage">
        <part name="body" element="tns:UsedRequest"/>
    </message>
    <portType name="TestPortType">
        <operation name="Test">
            <input message="tns:UsedRequestMessage"/>
        </operation>
    </portType>
</definitions>`

// checkV1WSDL is from TestTargetNamespaceMethodForMessageElements in target_namespace_test.go.
const checkV1WSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://b2b.also.com/ActWebServices.PriceAvailability"
             targetNamespace="http://b2b.also.com/ActWebServices.PriceAvailability">
    <types>
        <schema xmlns="http://www.w3.org/2001/XMLSchema"
                targetNamespace="http://b2b.also.com/ActWebServices.PriceAvailability"
                elementFormDefault="unqualified">
            <element name="checkV1" type="tns:CheckV1Type"/>
            <element name="checkV1Response" type="tns:CheckV1ResponseType"/>
            <complexType name="CheckV1Type">
                <sequence>
                    <element name="PartnerCountryCode" type="string"/>
                    <element name="StockType" type="string"/>
                </sequence>
            </complexType>
            <complexType name="CheckV1ResponseType">
                <sequence>
                    <element name="ErrorMessage" type="string" minOccurs="0"/>
                </sequence>
            </complexType>
        </schema>
    </types>
    <message name="checkV1Request">
        <part name="parameters" element="tns:checkV1"/>
    </message>
    <message name="checkV1Response">
        <part name="parameters" element="tns:checkV1Response"/>
    </message>
    <portType name="PriceAvailabilityPortType">
        <operation name="checkV1">
            <input message="tns:checkV1Request"/>
            <output message="tns:checkV1Response"/>
        </operation>
    </portType>
</definitions>`

// echoWSDL is from TestTargetNamespaceIntegrationWithSOAPClient in target_namespace_test.go.
const echoWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://integration.test/soap"
             targetNamespace="http://integration.test/soap">
    <types>
        <schema xmlns="http://www.w3.org/2001/XMLSchema"
                targetNamespace="http://integration.test/soap"
                elementFormDefault="unqualified">
            <element name="Echo" type="tns:EchoType"/>
            <element name="EchoResponse" type="tns:EchoResponseType"/>
            <complexType name="EchoType">
                <sequence>
                    <element name="Message" type="string"/>
                </sequence>
            </complexType>
            <complexType name="EchoResponseType">
                <sequence>
                    <element name="Result" type="string"/>
                </sequence>
            </complexType>
        </schema>
    </types>
    <message name="EchoRequest">
        <part name="parameters" element="tns:Echo"/>
    </message>
    <message name="EchoResponse">
        <part name="parameters" element="tns:EchoResponse"/>
    </message>
    <portType name="EchoServicePortType">
        <operation name="Echo">
            <input message="tns:EchoRequest"/>
            <output message="tns:EchoResponse"/>
        </operation>
    </portType>
    <binding name="EchoServiceBinding" type="tns:EchoServicePortType">
        <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="Echo">
            <soap:operation soapAction="Echo"/>
            <input>
                <soap:body use="literal"/>
            </input>
            <output>
                <soap:body use="literal"/>
            </output>
        </operation>
    </binding>
    <service name="EchoService">
        <port name="EchoServicePort" binding="tns:EchoServiceBinding">
            <soap:address location="http://localhost:8080/echo"/>
        </port>
    </service>
</definitions>`

// nonMessageTypesWSDL is from TestTargetNamespaceNotGeneratedForNonMessageTypes in target_namespace_test.go.
const nonMessageTypesWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://example.com/types"
             targetNamespace="http://example.com/types">
    <types>
        <schema xmlns="http://www.w3.org/2001/XMLSchema"
                targetNamespace="http://example.com/types"
                elementFormDefault="unqualified">
            <!-- Type used in message -->
            <element name="UsedElement" type="tns:UsedType"/>
            <complexType name="UsedType">
                <sequence>
                    <element name="field1" type="string"/>
                </sequence>
            </complexType>
            
            <!-- Type NOT used in message -->
            <complexType name="UnusedType">
                <sequence>
                    <element name="field2" type="string"/>
                </sequence>
            </complexType>
            
            <!-- Nested type -->
            <complexType name="NestedContainer">
                <sequence>
                    <element name="nested" type="tns:UnusedType"/>
                </sequence>
            </complexType>
        </schema>
    </types>
    <message name="TestMessage">
        <part name="body" element="tns:UsedElement"/>
    </message>
    <portType name="TestPortType">
        <operation name="TestOp">
            <input message="tns:TestMessage"/>
        </operation>
    </portType>
</definitions>`

// nonMessageElementWSDL is from TestXMLNameGeneration_MessageElements in xmlname_generation_test.go.
const nonMessageElementWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/" 
	targetNamespace="http://b2b.also.com/ActWebServices.PriceAvailability" 
	xmlns:tns="http://b2b.also.com/ActWebServices.PriceAvailability" 
	xmlns:xsd="http://www.w3.org/2001/XMLSchema"
	xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/">
	
	<wsdl:types>
		<xsd:schema targetNamespace="http://b2b.also.com/ActWebServices.PriceAvailability">
			<xsd:element name="checkV1" type="tns:checkV1"/>
			<xsd:element name="checkV1Response" type="tns:checkV1Response"/>
			<xsd:element name="nonMessageElement" type="tns:nonMessageType"/>
			
			<xsd:complexType name="checkV1">
				<xsd:sequence>
					<xsd:element name="PartnerCountryCode" type="xsd:string"/>
					<xsd:element name="StockType" type="xsd:string"/>
				</xsd:sequence>
			</xsd:complexType>
			
			<xsd:complexType name="checkV1Response">
				<xsd:sequence>
					<xsd:element name="ErrorMessage" nillable="true" type="xsd:string"/>
				</xsd:sequence>
			</xsd:complexType>
			
			<xsd:complexType name="nonMessageType">
				<xsd:sequence>
					<xsd:element name="data" type="xsd:string"/>
				</xsd:sequence>
			</xsd:complexType>
		</xsd:schema>
	</wsdl:types>
	
	<wsdl:message name="checkV1Request">
		<wsdl:part name="parameters" element="tns:checkV1"/>
	</wsdl:message>
	<wsdl:message name="checkV1Response">
		<wsdl:part name="parameters" element="tns:checkV1Response"/>
	</wsdl:message>
	
	<wsdl:portType name="TestPortType">
		<wsdl:operation name="checkV1">
			<wsdl:input message="tns:checkV1Request"/>
			<wsdl:output message="tns:checkV1Response"/>
		</wsdl:operation>
	</wsdl:portType>
	
	<wsdl:binding name="TestBinding" type="tns:TestPortType">
		<soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
		<wsdl:operation name="checkV1">
			<soap:operation soapAction="test_checkV1"/>
			<wsdl:input><soap:body use="literal"/></wsdl:input>
			<wsdl:output><soap:body use="literal"/></wsdl:output>
		</wsdl:operation>
	</wsdl:binding>
	
	<wsdl:service name="TestService">
		<wsdl:port name="TestPort" binding="tns:TestBinding">
			<soap:address location="http://example.com/test"/>
		</wsdl:port>
	</wsdl:service>
</wsdl:definitions>`

// qualifiedInlineWSDL is from TestXMLNameGeneration_ElementFormQualified in xmlname_generation_test.go.
const qualifiedInlineWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/" 
	targetNamespace="http://example.com/qualified" 
	xmlns:tns="http://example.com/qualified" 
	xmlns:xsd="http://www.w3.org/2001/XMLSchema"
	xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/">
	
	<wsdl:types>
		<xsd:schema targetNamespace="http://example.com/qualified" elementFormDefault="qualified">
			<xsd:element name="qualifiedRequest" type="tns:qualifiedRequestType"/>
			
			<xsd:complexType name="qualifiedRequestType">
				<xsd:sequence>
					<xsd:element name="param" type="xsd:string"/>
				</xsd:sequence>
			</xsd:complexType>
		</xsd:schema>
	</wsdl:types>
	
	<wsdl:message name="qualifiedMessage">
		<wsdl:part name="parameters" element="tns:qualifiedRequest"/>
	</wsdl:message>
	
	<wsdl:portType name="QualifiedPortType">
		<wsdl:operation name="qualifiedOp">
			<wsdl:input message="tns:qualifiedMessage"/>
		</wsdl:operation>
	</wsdl:portType>
</wsdl:definitions>`

// noMessagesWSDL is from TestXMLNameGeneration_NoMessages in xmlname_generation_test.go.
const noMessagesWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/" 
	targetNamespace="http://example.com/test" 
	xmlns:tns="http://example.com/test" 
	xmlns:xsd="http://www.w3.org/2001/XMLSchema">
	
	<wsdl:types>
		<xsd:schema targetNamespace="http://example.com/test">
			<xsd:element name="standaloneElement" type="tns:standaloneType"/>
			
			<xsd:complexType name="standaloneType">
				<xsd:sequence>
					<xsd:element name="data" type="xsd:string"/>
				</xsd:sequence>
			</xsd:complexType>
		</xsd:schema>
	</wsdl:types>
</wsdl:definitions>`

// responseElementsWSDL is from TestResponseElementNamespaceHandling in response_element_test.go.
const responseElementsWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="https://example.com/service"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="https://example.com/service">
    <types>
        <schema xmlns="http://www.w3.org/2001/XMLSchema"
                targetNamespace="https://example.com/service">
            
            <element name="SendungsErstellung">
                <complexType>
                    <sequence>
                        <element name="Versender" type="string"/>
                        <element name="Kundenreferenz" type="string"/>
                    </sequence>
                </complexType>
            </element>
            
            <element name="SendungsRueckmeldung">
                <complexType>
                    <sequence>
                        <element name="SendungsnummerAX4" type="string"/>
                        <element name="Frachtbriefnummer" type="string"/>
                    </sequence>
                </complexType>
            </element>
            
            <element name="GetOrder">
                <complexType>
                    <sequence>
                        <element name="OrderID" type="string"/>
                    </sequence>
                </complexType>
            </element>
            
            <element name="GetOrderResponse">
                <complexType>
                    <sequence>
                        <element name="OrderID" type="string"/>
                        <element name="Status" type="string"/>
                    </sequence>
                </complexType>
            </element>
        </schema>
    </types>
    
    <message name="SendungsErstellungRequest">
        <part name="parameters" element="tns:SendungsErstellung"/>
    </message>
    
    <message name="SendungsRueckmeldungResponse">
        <part name="parameters" element="tns:SendungsRueckmeldung"/>
    </message>
    
    <message name="GetOrderRequest">
        <part name="parameters" element="tns:GetOrder"/>
    </message>
    
    <message name="GetOrderResponse">
        <part name="parameters" element="tns:GetOrderResponse"/>
    </message>
    
    <portType name="ServicePortType">
        <operation name="SendungsErstellung">
            <input message="tns:SendungsErstellungRequest"/>
            <output message="tns:SendungsRueckmeldungResponse"/>
        </operation>
        <operation name="GetOrder">
            <input message="tns:GetOrderRequest"/>
            <output message="tns:GetOrderResponse"/>
        </operation>
    </portType>
    
    <binding name="ServiceBinding" type="tns:ServicePortType">
        <soap:binding transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="SendungsErstellung">
            <soap:operation soapAction="SendungsErstellung"/>
            <input><soap:body use="literal"/></input>
            <output><soap:body use="literal"/></output>
        </operation>
        <operation name="GetOrder">
            <soap:operation soapAction="GetOrder"/>
            <input><soap:body use="literal"/></input>
            <output><soap:body use="literal"/></output>
        </operation>
    </binding>
    
    <service name="Service">
        <port name="ServicePort" binding="tns:ServiceBinding">
            <soap:address location="http://example.com/service"/>
        </port>
    </service>
</definitions>`

// wsdl2RequestResponseWSDL is from TestResponseElementNamespaceHandlingWSDL2 in response_element_test.go.
const wsdl2RequestResponseWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<description xmlns="http://www.w3.org/ns/wsdl/"
             xmlns:tns="https://example.com/service"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             xmlns:wsoap="http://www.w3.org/ns/wsdl/soap"
             targetNamespace="https://example.com/service">
    
    <types>
        <xsd:schema targetNamespace="https://example.com/service">
            <xsd:element name="GetCustomer">
                <xsd:complexType>
                    <xsd:sequence>
                        <xsd:element name="CustomerID" type="xsd:string"/>
                    </xsd:sequence>
                </xsd:complexType>
            </xsd:element>
            
            <xsd:element name="GetCustomerResponse">
                <xsd:complexType>
                    <xsd:sequence>
                        <xsd:element name="CustomerID" type="xsd:string"/>
                        <xsd:element name="CustomerName" type="xsd:string"/>
                    </xsd:sequence>
                </xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
    
    <interface name="CustomerInterface">
        <operation name="GetCustomer" pattern="http://www.w3.org/ns/wsdl/in-out">
            <input element="tns:GetCustomer"/>
            <output element="tns:GetCustomerResponse"/>
        </operation>
    </interface>
    
    <binding name="CustomerBinding" interface="tns:CustomerInterface" type="http://www.w3.org/ns/wsdl/soap">
        <operation ref="tns:GetCustomer" wsoap:soapAction="GetCustomer"/>
    </binding>
    
    <service name="CustomerService" interface="tns:CustomerInterface">
        <endpoint name="CustomerEndpoint" binding="tns:CustomerBinding" 
                  address="http://example.com/customer"/>
    </service>
</description>`
