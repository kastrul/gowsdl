package generator

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionalComplexPointerTypes(t *testing.T) {
	wsdl := `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/" 
    xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
    xmlns:tns="http://example.com/test"
    targetNamespace="http://example.com/test">
    <types>
        <schema xmlns="http://www.w3.org/2001/XMLSchema" targetNamespace="http://example.com/test">
            <complexType name="OrderType">
                <sequence>
                    <element name="orderId" type="string" minOccurs="1"/>
                    <element name="shippingAddress" minOccurs="0">
                        <complexType>
                            <sequence>
                                <element name="street" type="string"/>
                                <element name="city" type="string"/>
                                <element name="zipCode" type="string"/>
                            </sequence>
                        </complexType>
                    </element>
                    <element name="billingAddress" minOccurs="1">
                        <complexType>
                            <sequence>
                                <element name="street" type="string"/>
                                <element name="city" type="string"/>
                            </sequence>
                        </complexType>
                    </element>
                    <element name="discount" minOccurs="0">
                        <complexType>
                            <simpleContent>
                                <extension base="decimal">
                                    <attribute name="currency" type="string"/>
                                </extension>
                            </simpleContent>
                        </complexType>
                    </element>
                </sequence>
            </complexType>
            
            <element name="createOrder" type="tns:OrderType"/>
            <element name="createOrderResponse" type="string"/>
        </schema>
    </types>
</definitions>`

	// Create temp file
	tmpFile, err := os.CreateTemp("", "test-wsdl-*.wsdl")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.Write([]byte(wsdl))
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	g, err := New(tmpFile.Name())
	require.NoError(t, err)

	code, err := g.Generate(context.TODO())
	require.NoError(t, err)

	// Get the generated code - concatenate all files
	var codeStr string
	for filename, content := range code {
		t.Logf("Generated file: %s", filename)
		codeStr += string(content) + "\n"
	}

	// Check that optional complex types use pointers to named types
	assert.Contains(t, codeStr, "ShippingAddress *OrderTypeShippingAddressType", "Optional complex type should be a pointer to named type")
	assert.Contains(t, codeStr, "BillingAddress OrderTypeBillingAddressType", "Required complex type should not be a pointer")
	assert.Contains(t, codeStr, "Discount *OrderTypeDiscountType", "Optional simple content should be a pointer")
}
