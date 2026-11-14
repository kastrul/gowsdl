package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const typePruningWSDL11 = `<?xml version="1.0" encoding="UTF-8"?>
<wsdl:definitions xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
                  xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"
                  xmlns:xsd="http://www.w3.org/2001/XMLSchema"
                  xmlns:tns="http://example.com/prune"
                  targetNamespace="http://example.com/prune">
	<wsdl:types>
		<xsd:schema targetNamespace="http://example.com/prune" xmlns:tns="http://example.com/prune">
			<xsd:element name="GetOrder">
				<xsd:complexType>
					<xsd:sequence>
						<xsd:element name="id" type="tns:OrderID"/>
						<xsd:element ref="tns:Trace"/>
					</xsd:sequence>
				</xsd:complexType>
			</xsd:element>
			<xsd:element name="GetOrderResponse">
				<xsd:complexType>
					<xsd:sequence>
						<xsd:element name="order" type="tns:Order"/>
					</xsd:sequence>
				</xsd:complexType>
			</xsd:element>
			<xsd:element name="Trace" type="tns:TraceInfo"/>
			<xsd:element name="OrderFault" type="tns:FaultDetail"/>
			<xsd:element name="AuthHeader" type="tns:Credentials"/>

			<xsd:complexType name="Order">
				<xsd:complexContent>
					<xsd:extension base="tns:BaseEntity">
						<xsd:sequence>
							<xsd:element name="status" type="tns:Status"/>
							<xsd:element name="tags" type="tns:TagList"/>
							<xsd:element name="amount" type="tns:Money"/>
							<xsd:element name="delivery" type="tns:Delivery"/>
						</xsd:sequence>
						<xsd:attribute name="priority" type="tns:Priority"/>
					</xsd:extension>
				</xsd:complexContent>
			</xsd:complexType>
			<xsd:complexType name="BaseEntity">
				<xsd:sequence><xsd:element name="created" type="xsd:dateTime"/></xsd:sequence>
			</xsd:complexType>
			<xsd:complexType name="Money">
				<xsd:simpleContent>
					<xsd:extension base="tns:Amount">
						<xsd:attribute name="currency" type="tns:Currency"/>
					</xsd:extension>
				</xsd:simpleContent>
			</xsd:complexType>
			<xsd:complexType name="Delivery">
				<xsd:sequence>
					<xsd:element name="shipping">
						<xsd:complexType>
							<xsd:sequence>
								<xsd:element name="address" type="tns:Address"/>
							</xsd:sequence>
						</xsd:complexType>
					</xsd:element>
				</xsd:sequence>
			</xsd:complexType>
			<xsd:complexType name="Address">
				<xsd:sequence><xsd:element name="city" type="xsd:string"/></xsd:sequence>
			</xsd:complexType>
			<xsd:complexType name="TraceInfo">
				<xsd:sequence><xsd:element name="span" type="xsd:string"/></xsd:sequence>
			</xsd:complexType>
			<xsd:complexType name="FaultDetail">
				<xsd:sequence><xsd:element name="reason" type="xsd:string"/></xsd:sequence>
			</xsd:complexType>
			<xsd:complexType name="Credentials">
				<xsd:sequence><xsd:element name="token" type="xsd:string"/></xsd:sequence>
			</xsd:complexType>
			<xsd:simpleType name="OrderID"><xsd:restriction base="tns:Identifier"/></xsd:simpleType>
			<xsd:simpleType name="Identifier"><xsd:restriction base="xsd:string"/></xsd:simpleType>
			<xsd:simpleType name="Status">
				<xsd:restriction base="xsd:string">
					<xsd:enumeration value="open"/>
					<xsd:enumeration value="closed"/>
				</xsd:restriction>
			</xsd:simpleType>
			<xsd:simpleType name="TagList"><xsd:list itemType="tns:Tag"/></xsd:simpleType>
			<xsd:simpleType name="Tag"><xsd:restriction base="xsd:string"/></xsd:simpleType>
			<xsd:simpleType name="Amount"><xsd:restriction base="xsd:decimal"/></xsd:simpleType>
			<xsd:simpleType name="Currency"><xsd:union memberTypes="tns:IsoCurrency tns:CustomCurrency"/></xsd:simpleType>
			<xsd:simpleType name="IsoCurrency"><xsd:restriction base="xsd:string"/></xsd:simpleType>
			<xsd:simpleType name="CustomCurrency"><xsd:restriction base="xsd:string"/></xsd:simpleType>
			<xsd:simpleType name="Priority"><xsd:restriction base="xsd:int"/></xsd:simpleType>

			<xsd:element name="ListCustomers">
				<xsd:complexType>
					<xsd:sequence><xsd:element name="filter" type="tns:CustomerFilter"/></xsd:sequence>
				</xsd:complexType>
			</xsd:element>
			<xsd:element name="ListCustomersResponse">
				<xsd:complexType>
					<xsd:sequence>
						<xsd:element name="customer" type="tns:Customer" maxOccurs="unbounded"/>
						<xsd:element name="extra">
							<xsd:complexType>
								<xsd:sequence><xsd:element name="note" type="xsd:string"/></xsd:sequence>
							</xsd:complexType>
						</xsd:element>
					</xsd:sequence>
				</xsd:complexType>
			</xsd:element>
			<xsd:complexType name="Customer">
				<xsd:sequence><xsd:element name="name" type="xsd:string"/></xsd:sequence>
			</xsd:complexType>
			<xsd:complexType name="CustomerFilter">
				<xsd:sequence><xsd:element name="segment" type="tns:Segment"/></xsd:sequence>
			</xsd:complexType>
			<xsd:simpleType name="Segment"><xsd:restriction base="xsd:string"/></xsd:simpleType>
		</xsd:schema>
	</wsdl:types>

	<wsdl:message name="GetOrderIn"><wsdl:part name="parameters" element="tns:GetOrder"/></wsdl:message>
	<wsdl:message name="GetOrderOut"><wsdl:part name="parameters" element="tns:GetOrderResponse"/></wsdl:message>
	<wsdl:message name="GetOrderFault"><wsdl:part name="fault" element="tns:OrderFault"/></wsdl:message>
	<wsdl:message name="AuthHeaderMsg"><wsdl:part name="auth" element="tns:AuthHeader"/></wsdl:message>
	<wsdl:message name="ListCustomersIn"><wsdl:part name="parameters" element="tns:ListCustomers"/></wsdl:message>
	<wsdl:message name="ListCustomersOut"><wsdl:part name="parameters" element="tns:ListCustomersResponse"/></wsdl:message>

	<wsdl:portType name="ShopPort">
		<wsdl:operation name="GetOrder">
			<wsdl:input message="tns:GetOrderIn"/>
			<wsdl:output message="tns:GetOrderOut"/>
			<wsdl:fault name="OrderFault" message="tns:GetOrderFault"/>
		</wsdl:operation>
		<wsdl:operation name="ListCustomers">
			<wsdl:input message="tns:ListCustomersIn"/>
			<wsdl:output message="tns:ListCustomersOut"/>
		</wsdl:operation>
	</wsdl:portType>
	<wsdl:binding name="ShopBinding" type="tns:ShopPort">
		<soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
		<wsdl:operation name="GetOrder">
			<soap:operation soapAction="urn:GetOrder"/>
			<wsdl:input>
				<soap:body use="literal"/>
				<soap:header message="tns:AuthHeaderMsg" part="auth" use="literal"/>
			</wsdl:input>
			<wsdl:output><soap:body use="literal"/></wsdl:output>
		</wsdl:operation>
		<wsdl:operation name="ListCustomers">
			<soap:operation soapAction="urn:ListCustomers"/>
			<wsdl:input><soap:body use="literal"/></wsdl:input>
			<wsdl:output><soap:body use="literal"/></wsdl:output>
		</wsdl:operation>
	</wsdl:binding>
</wsdl:definitions>`

func TestGenerator_TypePruning_KeepsReachableTypesOnly(t *testing.T) {
	code, err := tryGenerate(t, writeWSDL(t, typePruningWSDL11), WithPackage("svc"), WithExportAllTypes(true), WithOperations("GetOrder"))
	require.NoError(t, err)
	types := declaredTypes(code.types())

	for _, name := range []string{
		"GetOrder", "GetOrderResponse",
		"OrderID", "Identifier",
		"Trace", "TraceInfo",
		"Order", "BaseEntity", "Status", "TagList", "Tag",
		"Money", "Amount", "Currency", "IsoCurrency", "CustomCurrency", "Priority",
		"Delivery", "Address", "DeliveryShippingType",
		"OrderFault", "FaultDetail",
		"AuthHeader", "Credentials",
	} {
		assert.Contains(t, types, name, "reachable type %s must be generated", name)
	}
	for _, name := range []string{
		"ListCustomers", "ListCustomersResponse", "Customer", "CustomerFilter", "Segment",
		"ListCustomersResponseExtraType",
	} {
		assert.NotContains(t, types, name, "unused type %s must be pruned", name)
	}

	code.client(t)
}

func TestGenerator_TypePruning_KeptTypesMatchUnprunedOutput(t *testing.T) {
	pruned, err := tryGenerate(t, writeWSDL(t, typePruningWSDL11), WithPackage("svc"), WithExportAllTypes(true), WithOperations("ListCustomers"))
	require.NoError(t, err)
	full, err := tryGenerate(t, writeWSDL(t, typePruningWSDL11), WithPackage("svc"), WithExportAllTypes(true))
	require.NoError(t, err)

	prunedTypes := declaredTypes(pruned.types())
	fullTypes := declaredTypes(full.types())
	assert.Subset(t, fullTypes, prunedTypes)
	assert.Less(t, len(prunedTypes), len(fullTypes))
	assert.Contains(t, prunedTypes, "ListCustomersResponseExtraType")
	assert.Equal(t, full["operations"] != nil, pruned["operations"] != nil)

	pruned.client(t)
}

func TestGenerator_TypePruning_Disabled(t *testing.T) {
	pruned, err := tryGenerate(t, writeWSDL(t, typePruningWSDL11), WithPackage("svc"), WithExportAllTypes(true), WithOperations("GetOrder"), WithTypePruning(false))
	require.NoError(t, err)
	full, err := tryGenerate(t, writeWSDL(t, typePruningWSDL11), WithPackage("svc"), WithExportAllTypes(true))
	require.NoError(t, err)

	assert.Equal(t, full.types(), pruned.types())
	assert.NotContains(t, pruned.operations(), "ListCustomers(ctx")
}

func TestGenerator_TypePruning_NoOperationFilter(t *testing.T) {
	code, err := tryGenerate(t, writeWSDL(t, typePruningWSDL11), WithPackage("svc"), WithExportAllTypes(true), WithTypePruning(true))
	require.NoError(t, err)
	assert.Contains(t, declaredTypes(code.types()), "CustomerFilter")
}

func TestGenerator_TypePruning_SkippedWithServerGeneration(t *testing.T) {
	code, err := tryGenerate(t, writeWSDL(t, typePruningWSDL11), WithPackage("svc"), WithExportAllTypes(true), WithOperations("GetOrder"), WithServerGeneration(true))
	require.NoError(t, err)
	assert.Contains(t, declaredTypes(code.types()), "CustomerFilter",
		"server code covers every operation, so every type is kept")
}

func TestGenerator_TypePruning_WSDL20(t *testing.T) {
	code, err := tryGenerate(t, writeWSDL(t, operationFilterWSDL20), WithPackage("svc"), WithExportAllTypes(true), WithOperations("GetUser"))
	require.NoError(t, err)
	types := declaredTypes(code.types())

	assert.Contains(t, types, "GetUser")
	assert.Contains(t, types, "GetUserResponse")
	assert.NotContains(t, types, "DeleteUser")
	assert.NotContains(t, types, "DeleteUserResponse")

	code.client(t)
}

const typePruningRPC = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             xmlns:tns="http://example.com/rpc"
             targetNamespace="http://example.com/rpc">
	<types>
		<xsd:schema targetNamespace="http://example.com/rpc">
			<xsd:complexType name="User">
				<xsd:sequence><xsd:element name="name" type="xsd:string"/></xsd:sequence>
			</xsd:complexType>
			<xsd:complexType name="Invoice">
				<xsd:sequence><xsd:element name="total" type="xsd:decimal"/></xsd:sequence>
			</xsd:complexType>
		</xsd:schema>
	</types>
	<message name="getUserRequest"><part name="id" type="xsd:int"/></message>
	<message name="getUserResponse"><part name="user" type="tns:User"/></message>
	<message name="getInvoiceRequest"><part name="id" type="xsd:int"/><part name="year" type="xsd:int"/></message>
	<message name="getInvoiceResponse"><part name="invoice" type="tns:Invoice"/></message>
	<portType name="BillingPort">
		<operation name="getUser">
			<input message="tns:getUserRequest"/>
			<output message="tns:getUserResponse"/>
		</operation>
		<operation name="getInvoice">
			<input message="tns:getInvoiceRequest"/>
			<output message="tns:getInvoiceResponse"/>
		</operation>
	</portType>
	<binding name="BillingBinding" type="tns:BillingPort">
		<soap:binding style="rpc" transport="http://schemas.xmlsoap.org/soap/http"/>
		<operation name="getUser">
			<soap:operation soapAction="getUser"/>
			<input><soap:body use="literal" namespace="http://example.com/rpc"/></input>
			<output><soap:body use="literal" namespace="http://example.com/rpc"/></output>
		</operation>
		<operation name="getInvoice">
			<soap:operation soapAction="getInvoice"/>
			<input><soap:body use="literal" namespace="http://example.com/rpc"/></input>
			<output><soap:body use="literal" namespace="http://example.com/rpc"/></output>
		</operation>
	</binding>
</definitions>`

func TestGenerator_TypePruning_RPC(t *testing.T) {
	code, err := tryGenerate(t, writeWSDL(t, typePruningRPC), WithPackage("svc"), WithExportAllTypes(true), WithOperations("getUser"))
	require.NoError(t, err)
	types := declaredTypes(code.types())

	assert.Contains(t, types, "User")
	assert.Contains(t, types, "GetUserRequest")
	assert.Contains(t, types, "GetUserResponse")
	assert.Contains(t, types, "OperationGetUserIn")
	assert.NotContains(t, types, "Invoice")
	assert.NotContains(t, types, "GetInvoiceRequest")
	assert.NotContains(t, types, "OperationGetInvoiceIn")

	code.client(t)
}
