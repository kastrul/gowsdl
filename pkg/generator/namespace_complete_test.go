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

// TestCompleteNamespaceHandling tests the complete flow with a WSDL similar to the user's
func TestCompleteNamespaceHandling(t *testing.T) {
	// WSDL similar to user's case - without elementFormDefault (defaults to unqualified)
	wsdl := `<?xml version="1.0" encoding="UTF-8"?>
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

	file, err := os.CreateTemp("", "test-namespace-complete-*.wsdl")
	require.NoError(t, err)
	defer func() { _ = os.Remove(file.Name()) }()
	defer func() { _ = file.Close() }()

	_, err = file.WriteString(wsdl)
	require.NoError(t, err)

	// Generate code
	g, err := New(file.Name(), WithPackage("service"))
	require.NoError(t, err)

	files, err := g.Generate(context.Background())
	require.NoError(t, err)

	types, ok := files["types"]
	require.True(t, ok, "types file should be generated")

	typesStr := string(types)

	// Check that XMLName uses prefix
	assert.Contains(t, typesStr, `xml:"tns:GOWebService_SendungsErstellung"`)

	// Check that TargetNamespace method is generated
	assert.Contains(t, typesStr, "func (t GOWebService_SendungsErstellung) TargetNamespace() string")
	assert.Contains(t, typesStr, `return "https://wsdemo.ax4.com/ws/GeneralOvernight"`)
}
