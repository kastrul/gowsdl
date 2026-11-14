// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInlineComplexTypeGeneration(t *testing.T) {
	wsdl := `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://example.com/order"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/order">
    <types>
        <schema xmlns="http://www.w3.org/2001/XMLSchema"
                targetNamespace="http://example.com/order">
            
            <!-- Complex type with inline optional complex type -->
            <complexType name="OrderType">
                <sequence>
                    <element name="orderId" type="string"/>
                    <element name="shippingAddress" minOccurs="0">
                        <complexType>
                            <sequence>
                                <element name="street" type="string"/>
                                <element name="city" type="string"/>
                                <element name="zipCode" type="string" minOccurs="0"/>
                            </sequence>
                        </complexType>
                    </element>
                    <element name="billingAddress">
                        <complexType>
                            <sequence>
                                <element name="street" type="string"/>
                                <element name="city" type="string"/>
                            </sequence>
                        </complexType>
                    </element>
                    <element name="items" maxOccurs="unbounded">
                        <complexType>
                            <sequence>
                                <element name="itemId" type="string"/>
                                <element name="quantity" type="int"/>
                            </sequence>
                        </complexType>
                    </element>
                </sequence>
            </complexType>
            
            <!-- Element with inline complex type -->
            <element name="CreateOrderRequest">
                <complexType>
                    <sequence>
                        <element name="order" type="tns:OrderType"/>
                    </sequence>
                </complexType>
            </element>
            
            <element name="CreateOrderResponse">
                <complexType>
                    <sequence>
                        <element name="success" type="boolean"/>
                    </sequence>
                </complexType>
            </element>
        </schema>
    </types>
    
    <message name="CreateOrderRequestMessage">
        <part name="parameters" element="tns:CreateOrderRequest"/>
    </message>
    
    <message name="CreateOrderResponseMessage">
        <part name="parameters" element="tns:CreateOrderResponse"/>
    </message>
    
    <portType name="OrderServicePortType">
        <operation name="CreateOrder">
            <input message="tns:CreateOrderRequestMessage"/>
            <output message="tns:CreateOrderResponseMessage"/>
        </operation>
    </portType>
    
    <binding name="OrderServiceBinding" type="tns:OrderServicePortType">
        <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="CreateOrder">
            <soap:operation soapAction="http://example.com/order/CreateOrder"/>
            <input><soap:body use="literal"/></input>
            <output><soap:body use="literal"/></output>
        </operation>
    </binding>
</definitions>`

	typesStr := generate(t, writeWSDL(t, wsdl), WithPackage("order")).types()
	t.Logf("Generated types:\n%s", typesStr)

	// Check that named types are generated for inline complex types
	assert.Contains(t, typesStr, "type OrderTypeShippingAddressType struct", "Should generate named type for optional inline complex type")
	assert.Contains(t, typesStr, "type OrderTypeBillingAddressType struct", "Should generate named type for required inline complex type")
	assert.Contains(t, typesStr, "type OrderTypeItemsType struct", "Should generate named type for array inline complex type")

	// Check that the OrderType uses pointers to named types
	assert.Contains(t, typesStr, "ShippingAddress *OrderTypeShippingAddressType", "Optional field should be pointer to named type")
	assert.Contains(t, typesStr, "BillingAddress OrderTypeBillingAddressType", "Required field should be value type")
	assert.Contains(t, typesStr, "Items []OrderTypeItemsType", "Array field should use named type")

	// Check that inline types for elements are also handled
	assert.Contains(t, typesStr, "type CreateOrderRequestType struct", "Should generate named type for element's inline complex type")
	assert.Contains(t, typesStr, "type CreateOrderResponseType struct", "Should generate named type for element's inline complex type")
}

func TestNestedInlineComplexTypes(t *testing.T) {
	wsdl := `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://example.com/nested"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/nested">
    <types>
        <schema xmlns="http://www.w3.org/2001/XMLSchema"
                targetNamespace="http://example.com/nested">
            
            <complexType name="CompanyType">
                <sequence>
                    <element name="name" type="string"/>
                    <element name="headquarters" minOccurs="0">
                        <complexType>
                            <sequence>
                                <element name="address">
                                    <complexType>
                                        <sequence>
                                            <element name="street" type="string"/>
                                            <element name="city" type="string"/>
                                            <element name="coordinates" minOccurs="0">
                                                <complexType>
                                                    <sequence>
                                                        <element name="lat" type="decimal"/>
                                                        <element name="lon" type="decimal"/>
                                                    </sequence>
                                                </complexType>
                                            </element>
                                        </sequence>
                                    </complexType>
                                </element>
                                <element name="phone" type="string"/>
                            </sequence>
                        </complexType>
                    </element>
                </sequence>
            </complexType>
        </schema>
    </types>
</definitions>`

	typesStr := generate(t, writeWSDL(t, wsdl), WithPackage("nested")).types()
	t.Logf("Generated types:\n%s", typesStr)

	// Check nested type generation
	assert.Contains(t, typesStr, "type CompanyTypeHeadquartersType struct", "Should generate type for headquarters")
	assert.Contains(t, typesStr, "type CompanyTypeHeadquartersTypeAddressType struct", "Should generate type for nested address")
	assert.Contains(t, typesStr, "type CompanyTypeHeadquartersTypeAddressTypeCoordinatesType struct", "Should generate type for deeply nested coordinates")

	// Check proper usage in parent types
	assert.Contains(t, typesStr, "Headquarters *CompanyTypeHeadquartersType", "Optional headquarters should use pointer")
	assert.Contains(t, typesStr, "Address CompanyTypeHeadquartersTypeAddressType", "Required address should be value type")
	assert.Contains(t, typesStr, "Coordinates *CompanyTypeHeadquartersTypeAddressTypeCoordinatesType", "Optional coordinates should use pointer")
}
