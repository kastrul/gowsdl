// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSOAPNamespaceHandling tests that generated code produces correct SOAP envelopes
func TestSOAPNamespaceHandling(t *testing.T) {
	// WSDL with unqualified schema (default)
	wsdl := `<?xml version="1.0" encoding="UTF-8"?>
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

	file, err := os.CreateTemp("", "test-soap-namespace-*.wsdl")
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

	// Check that XMLName includes namespace
	typesStr := string(types)
	assert.Contains(t, typesStr, "CreateOrder")
	assert.Contains(t, typesStr, "XMLName")
	assert.Contains(t, typesStr, "https://example.com/service")
}
