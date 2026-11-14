package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kastrul/gowsdl/pkg/parser"
)

func bindingNames(bindings []*parser.WSDLBinding) []string {
	names := make([]string, 0, len(bindings))
	for _, b := range bindings {
		names = append(names, b.Name)
	}
	return names
}

func TestBindingsForPortType(t *testing.T) {
	g := &Generator{wsdl: &parser.WSDL{Binding: []*parser.WSDLBinding{
		{Name: "OrderPortBinding", Type: "tns:OrderPort"},
		{Name: "OrderBinding", Type: "tns:Order"},
		{Name: "OrderBinding12", Type: "Order"},
		{Name: "StockBinding", Type: "tns:StockPortType"},
	}}}

	tests := []struct {
		name     string
		portType string
		expected []string
	}{
		{"exact match wins over an earlier fuzzy match", "Order", []string{"OrderBinding", "OrderBinding12"}},
		{"exact match with suffix", "OrderPort", []string{"OrderPortBinding"}},
		{"fuzzy fallback trims suffixes", "Stock", []string{"StockBinding"}},
		{"fuzzy fallback ignores case", "stockPort", []string{"StockBinding"}},
		{"exact match is case-sensitive, fuzzy fallback finds all", "order", []string{"OrderPortBinding", "OrderBinding", "OrderBinding12"}},
		{"no match", "Invoice", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, bindingNames(g.bindingsForPortType(tt.portType)))
		})
	}

	assert.Empty(t, (&Generator{}).bindingsForPortType("Order"))
}

func TestBindingsForInterface_MatchesExactLocalNameOnly(t *testing.T) {
	g := &Generator{wsdl2: &parser.WSDL2{Bindings: []*parser.WSDL2Binding{
		{Name: "A", Interface: "tns:OrderPortType"},
		{Name: "B", Interface: "tns:Order"},
	}}}

	got := g.bindingsForInterface("Order")
	require.Len(t, got, 1)
	assert.Equal(t, "B", got[0].Name)
	assert.Empty(t, g.bindingsForInterface("OrderPort"))
	assert.Empty(t, (&Generator{}).bindingsForInterface("Order"))
}

func TestIsRPCPortType_UsesTheSameMatchAsFindSOAPAction(t *testing.T) {
	g := &Generator{wsdl: &parser.WSDL{Binding: []*parser.WSDLBinding{
		{Name: "DocBinding", Type: "tns:OrderPort"},
		{
			Name:        "RPCBinding",
			Type:        "tns:order",
			SOAPBinding: parser.WSDLSOAPBinding{Style: "rpc"},
			Operations: []*parser.WSDLOperation{
				{Name: "Get", SOAPOperation: parser.WSDLSOAPOperation{SOAPAction: "urn:rpc/Get"}},
			},
		},
	}}}

	assert.True(t, g.isRPCPortType("order"))
	assert.Equal(t, "urn:rpc/Get", g.findSOAPAction("Get", "order"))
	assert.False(t, g.isRPCPortType("OrderPort"))
	assert.Equal(t, "", g.findSOAPAction("Get", "OrderPort"))
}

func TestGenerator_RPCPortType_LowercaseName_UsesRPCCalls(t *testing.T) {
	out := generate(t, writeWSDL(t, `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://example.com/rpc"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/rpc">
    <message name="searchRequest">
        <part name="query" type="xsd:string"/>
        <part name="maxResults" type="xsd:int"/>
    </message>
    <message name="searchResponse">
        <part name="count" type="xsd:int"/>
    </message>
    <portType name="searchService">
        <operation name="search">
            <input message="tns:searchRequest"/>
            <output message="tns:searchResponse"/>
        </operation>
    </portType>
    <binding name="searchServiceBinding" type="tns:searchService">
        <soap:binding style="rpc" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="search">
            <soap:operation soapAction="http://example.com/rpc/search"/>
            <input><soap:body use="literal"/></input>
            <output><soap:body use="literal"/></output>
        </operation>
    </binding>
</definitions>`), WithPackage("test"))

	types := string(out["types"])
	operations := string(out["operations"])

	assert.Contains(t, types, "type OperationSearchIn struct")
	assert.Contains(t, operations, "α := OperationSearchIn{", "an RPC port type must get RPC-style calls whatever the case of its name")
	assert.Contains(t, operations, "var γ OperationSearchOut")
	assert.NotContains(t, operations, "// Document-style operation")
}

func TestGenerator_PortTypesDifferingBySuffix_GetTheirOwnSOAPAction(t *testing.T) {
	out := generate(t, writeWSDL(t, `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://example.com/order"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/order">
    <types>
        <xsd:schema targetNamespace="http://example.com/order" elementFormDefault="qualified">
            <xsd:element name="GetOrder"><xsd:complexType><xsd:sequence><xsd:element name="id" type="xsd:string"/></xsd:sequence></xsd:complexType></xsd:element>
            <xsd:element name="GetOrderResponse"><xsd:complexType><xsd:sequence><xsd:element name="status" type="xsd:string"/></xsd:sequence></xsd:complexType></xsd:element>
            <xsd:element name="GetPort"><xsd:complexType><xsd:sequence><xsd:element name="id" type="xsd:string"/></xsd:sequence></xsd:complexType></xsd:element>
            <xsd:element name="GetPortResponse"><xsd:complexType><xsd:sequence><xsd:element name="status" type="xsd:string"/></xsd:sequence></xsd:complexType></xsd:element>
        </xsd:schema>
    </types>
    <message name="GetOrderIn"><part name="parameters" element="tns:GetOrder"/></message>
    <message name="GetOrderOut"><part name="parameters" element="tns:GetOrderResponse"/></message>
    <message name="GetPortIn"><part name="parameters" element="tns:GetPort"/></message>
    <message name="GetPortOut"><part name="parameters" element="tns:GetPortResponse"/></message>
    <portType name="Order">
        <operation name="Get">
            <input message="tns:GetOrderIn"/>
            <output message="tns:GetOrderOut"/>
        </operation>
    </portType>
    <portType name="OrderPort">
        <operation name="Get">
            <input message="tns:GetPortIn"/>
            <output message="tns:GetPortOut"/>
        </operation>
    </portType>
    <binding name="OrderPortBinding" type="tns:OrderPort">
        <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="Get">
            <soap:operation soapAction="urn:orderport/Get"/>
            <input><soap:body use="literal"/></input>
            <output><soap:body use="literal"/></output>
        </operation>
    </binding>
    <binding name="OrderBinding" type="tns:Order">
        <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="Get">
            <soap:operation soapAction="urn:order/Get"/>
            <input><soap:body use="literal"/></input>
            <output><soap:body use="literal"/></output>
        </operation>
    </binding>
</definitions>`), WithPackage("test"))

	operations := string(out["operations"])

	assert.Regexp(t, `(?s)func \(service \*order\) Get\(.*?CallContext\(ctx, "urn:order/Get"`, operations)
	assert.Regexp(t, `(?s)func \(service \*orderPort\) Get\(.*?CallContext\(ctx, "urn:orderport/Get"`, operations)
}
