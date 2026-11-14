// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseElementNamespaceHandling(t *testing.T) {
	// Test WSDL with properly defined operations and messages
	wsdl := `<?xml version="1.0" encoding="UTF-8"?>
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

	file, err := os.CreateTemp("", "test-response-element-*.wsdl")
	require.NoError(t, err)
	defer func() { _ = os.Remove(file.Name()) }()
	defer func() { _ = file.Close() }()

	_, err = file.WriteString(wsdl)
	require.NoError(t, err)

	g, err := New(file.Name(), WithPackage("service"))
	require.NoError(t, err)

	files, err := g.Generate(context.Background())
	require.NoError(t, err)

	types, ok := files["types"]
	require.True(t, ok, "types file should be generated")

	typesStr := string(types)
	t.Logf("Generated types:\n%s", typesStr)

	// Response elements should use local name only (no namespace prefix)
	assert.True(t, strings.Contains(typesStr, `XMLName xml.Name `+"`"+`xml:"SendungsRueckmeldung"`+"`"),
		"SendungsRueckmeldung should use local name only")
	assert.True(t, strings.Contains(typesStr, `XMLName xml.Name `+"`"+`xml:"GetOrderResponse"`+"`"),
		"GetOrderResponse should use local name only")

	// Request elements should still use namespace prefix
	assert.True(t, strings.Contains(typesStr, `XMLName xml.Name `+"`"+`xml:"tns:SendungsErstellung"`+"`"),
		"SendungsErstellung should use tns: prefix")
	assert.True(t, strings.Contains(typesStr, `XMLName xml.Name `+"`"+`xml:"tns:GetOrder"`+"`"),
		"GetOrder should use tns: prefix")
}

// TestResponseElementNamespaceHandlingWSDL2 tests that WSDL 2.0 response elements are handled correctly
func TestResponseElementNamespaceHandlingWSDL2(t *testing.T) {
	// Test WSDL 2.0 with properly defined interfaces and operations
	wsdl := `<?xml version="1.0" encoding="UTF-8"?>
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

	file, err := os.CreateTemp("", "test-wsdl2-response-*.wsdl")
	require.NoError(t, err)
	defer func() { _ = os.Remove(file.Name()) }()
	defer func() { _ = file.Close() }()

	_, err = file.WriteString(wsdl)
	require.NoError(t, err)

	g, err := New(file.Name(), WithPackage("service"))
	require.NoError(t, err)

	files, err := g.Generate(context.Background())
	require.NoError(t, err)

	types, ok := files["types"]
	require.True(t, ok, "types file should be generated")

	typesStr := string(types)
	t.Logf("Generated types:\n%s", typesStr)

	// Response elements should use local name only (no namespace prefix)
	assert.True(t, strings.Contains(typesStr, `XMLName xml.Name `+"`"+`xml:"GetCustomerResponse"`+"`"),
		"GetCustomerResponse should use local name only")

	// Request elements should still use namespace prefix
	assert.True(t, strings.Contains(typesStr, `XMLName xml.Name `+"`"+`xml:"tns:GetCustomer"`+"`"),
		"GetCustomer should use tns: prefix")
}
