// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"context"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const extensionInlineComplexWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://example.com/order"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/order">
    <types>
        <xsd:schema targetNamespace="http://example.com/order">
            <xsd:complexType name="BaseEntity">
                <xsd:sequence>
                    <xsd:element name="id" type="xsd:string"/>
                </xsd:sequence>
            </xsd:complexType>
            <xsd:complexType name="Order">
                <xsd:complexContent>
                    <xsd:extension base="tns:BaseEntity">
                        <xsd:sequence>
                            <xsd:element name="shipping">
                                <xsd:complexType>
                                    <xsd:sequence><xsd:element name="city" type="xsd:string"/></xsd:sequence>
                                </xsd:complexType>
                            </xsd:element>
                            <xsd:element name="billing" minOccurs="0">
                                <xsd:complexType>
                                    <xsd:sequence><xsd:element name="city" type="xsd:string"/></xsd:sequence>
                                </xsd:complexType>
                            </xsd:element>
                            <xsd:element name="lines" maxOccurs="unbounded">
                                <xsd:complexType>
                                    <xsd:sequence><xsd:element name="sku" type="xsd:string"/></xsd:sequence>
                                </xsd:complexType>
                            </xsd:element>
                        </xsd:sequence>
                    </xsd:extension>
                </xsd:complexContent>
            </xsd:complexType>
            <xsd:element name="GetOrderRequest">
                <xsd:complexType>
                    <xsd:sequence><xsd:element name="id" type="xsd:string"/></xsd:sequence>
                </xsd:complexType>
            </xsd:element>
            <xsd:element name="GetOrderResponse">
                <xsd:complexType>
                    <xsd:sequence><xsd:element name="order" type="tns:Order"/></xsd:sequence>
                </xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
    <message name="GetOrderRequestMessage">
        <part name="parameters" element="tns:GetOrderRequest"/>
    </message>
    <message name="GetOrderResponseMessage">
        <part name="parameters" element="tns:GetOrderResponse"/>
    </message>
    <portType name="OrderServicePortType">
        <operation name="GetOrder">
            <input message="tns:GetOrderRequestMessage"/>
            <output message="tns:GetOrderResponseMessage"/>
        </operation>
    </portType>
    <binding name="OrderServiceBinding" type="tns:OrderServicePortType">
        <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="GetOrder">
            <soap:operation soapAction="http://example.com/order/GetOrder"/>
            <input><soap:body use="literal"/></input>
            <output><soap:body use="literal"/></output>
        </operation>
    </binding>
</definitions>`

func TestGenerator_ComplexContentExtension_InlineComplexType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "order.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(extensionInlineComplexWSDL), 0o600))

	g, err := New(path, WithPackage("order"))
	require.NoError(t, err)

	out, err := g.Generate(context.Background())
	require.NoError(t, err)

	code := string(out["header"]) + string(out["types"]) + string(out["operations"])
	formatted, err := format.Source([]byte(code))
	require.NoError(t, err, "generated code should be valid Go:\n%s", code)
	typesStr := string(formatted)

	orderStruct := regexp.MustCompile(`(?s)type Order struct \{.*?\n\}`).FindString(typesStr)
	require.NotEmpty(t, orderStruct)
	assert.Len(t, regexp.MustCompile(`(?m)^\s*BaseEntity$`).FindAllString(orderStruct, -1), 1, "base type should be embedded exactly once")

	assert.Contains(t, typesStr, "type OrderShippingType struct")
	assert.Contains(t, typesStr, "type OrderBillingType struct")
	assert.Contains(t, typesStr, "type OrderLinesType struct")
	assert.Regexp(t, `Shipping\s+OrderShippingType\s+`+"`"+`xml:"shipping,omitempty"`, typesStr)
	assert.Regexp(t, `Billing\s+\*OrderBillingType\s+`+"`"+`xml:"billing,omitempty"`, typesStr)
	assert.Regexp(t, `Lines\s+\[\]OrderLinesType\s+`+"`"+`xml:"lines,omitempty"`, typesStr)
}
