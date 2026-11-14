// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// httpBindingWSDL11 has a SOAP port type bound with SOAP 1.1 and SOAP 1.2, an
// HTTP-only port type, an RPC port type and a port type without any binding.
// GetAlerts and Ping take input messages without parts.
const httpBindingWSDL11 = `<?xml version="1.0" encoding="UTF-8"?>
<wsdl:definitions xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
                  xmlns:soap12="http://schemas.xmlsoap.org/wsdl/soap12/"
                  xmlns:http="http://schemas.xmlsoap.org/wsdl/http/"
                  xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"
                  xmlns:xsd="http://www.w3.org/2001/XMLSchema"
                  xmlns:tns="http://example.com/schedule"
                  targetNamespace="http://example.com/schedule">
	<wsdl:types>
		<xsd:schema targetNamespace="http://example.com/schedule" elementFormDefault="qualified">
			<xsd:element name="GetAlertsResponse">
				<xsd:complexType>
					<xsd:sequence>
						<xsd:element name="Alert" type="xsd:string" minOccurs="0" maxOccurs="unbounded"/>
					</xsd:sequence>
				</xsd:complexType>
			</xsd:element>
			<xsd:element name="GetRoute">
				<xsd:complexType>
					<xsd:sequence>
						<xsd:element name="RouteID" type="xsd:int"/>
					</xsd:sequence>
				</xsd:complexType>
			</xsd:element>
			<xsd:element name="GetRouteResponse">
				<xsd:complexType>
					<xsd:sequence>
						<xsd:element name="Name" type="xsd:string"/>
					</xsd:sequence>
				</xsd:complexType>
			</xsd:element>
			<xsd:element name="ArrayOfAlert">
				<xsd:complexType>
					<xsd:sequence>
						<xsd:element name="Alert" type="xsd:string" minOccurs="0" maxOccurs="unbounded"/>
					</xsd:sequence>
				</xsd:complexType>
			</xsd:element>
			<xsd:element name="Echo" type="xsd:string"/>
			<xsd:element name="EchoResponse" type="xsd:string"/>
		</xsd:schema>
	</wsdl:types>

	<wsdl:message name="GetAlertsSoapIn"/>
	<wsdl:message name="GetAlertsSoapOut"><wsdl:part name="parameters" element="tns:GetAlertsResponse"/></wsdl:message>
	<wsdl:message name="GetRouteSoapIn"><wsdl:part name="parameters" element="tns:GetRoute"/></wsdl:message>
	<wsdl:message name="GetRouteSoapOut"><wsdl:part name="parameters" element="tns:GetRouteResponse"/></wsdl:message>
	<wsdl:message name="GetAlertsHttpGetIn"/>
	<wsdl:message name="GetAlertsHttpGetOut"><wsdl:part name="Body" element="tns:ArrayOfAlert"/></wsdl:message>
	<wsdl:message name="HttpOnlyHttpGetIn"><wsdl:part name="id" type="xsd:string"/></wsdl:message>
	<wsdl:message name="HttpOnlyHttpGetOut"><wsdl:part name="Body" element="tns:ArrayOfAlert"/></wsdl:message>
	<wsdl:message name="PingRequest"/>
	<wsdl:message name="PingResponse"><wsdl:part name="status" type="xsd:string"/></wsdl:message>
	<wsdl:message name="EchoIn"><wsdl:part name="parameters" element="tns:Echo"/></wsdl:message>
	<wsdl:message name="EchoOut"><wsdl:part name="parameters" element="tns:EchoResponse"/></wsdl:message>

	<wsdl:portType name="ScheduleSoap">
		<wsdl:operation name="GetAlerts">
			<wsdl:input message="tns:GetAlertsSoapIn"/>
			<wsdl:output message="tns:GetAlertsSoapOut"/>
		</wsdl:operation>
		<wsdl:operation name="GetRoute">
			<wsdl:input message="tns:GetRouteSoapIn"/>
			<wsdl:output message="tns:GetRouteSoapOut"/>
		</wsdl:operation>
	</wsdl:portType>
	<wsdl:portType name="ScheduleHttpGet">
		<wsdl:operation name="GetAlerts">
			<wsdl:input message="tns:GetAlertsHttpGetIn"/>
			<wsdl:output message="tns:GetAlertsHttpGetOut"/>
		</wsdl:operation>
		<wsdl:operation name="HttpOnly">
			<wsdl:input message="tns:HttpOnlyHttpGetIn"/>
			<wsdl:output message="tns:HttpOnlyHttpGetOut"/>
		</wsdl:operation>
	</wsdl:portType>
	<wsdl:portType name="LegacyRPC">
		<wsdl:operation name="Ping">
			<wsdl:input message="tns:PingRequest"/>
			<wsdl:output message="tns:PingResponse"/>
		</wsdl:operation>
	</wsdl:portType>
	<wsdl:portType name="Unbound">
		<wsdl:operation name="Echo">
			<wsdl:input message="tns:EchoIn"/>
			<wsdl:output message="tns:EchoOut"/>
		</wsdl:operation>
	</wsdl:portType>

	<wsdl:binding name="ScheduleSoap" type="tns:ScheduleSoap">
		<soap:binding transport="http://schemas.xmlsoap.org/soap/http"/>
		<wsdl:operation name="GetAlerts">
			<soap:operation soapAction="urn:GetAlerts" style="document"/>
			<wsdl:input><soap:body use="literal"/></wsdl:input>
			<wsdl:output><soap:body use="literal"/></wsdl:output>
		</wsdl:operation>
		<wsdl:operation name="GetRoute">
			<soap:operation soapAction="urn:GetRoute" style="document"/>
			<wsdl:input><soap:body use="literal"/></wsdl:input>
			<wsdl:output><soap:body use="literal"/></wsdl:output>
		</wsdl:operation>
	</wsdl:binding>
	<wsdl:binding name="ScheduleSoap12" type="tns:ScheduleSoap">
		<soap12:binding transport="http://schemas.xmlsoap.org/soap/http"/>
		<wsdl:operation name="GetAlerts">
			<soap12:operation soapAction="urn:GetAlerts" style="document"/>
			<wsdl:input><soap12:body use="literal"/></wsdl:input>
			<wsdl:output><soap12:body use="literal"/></wsdl:output>
		</wsdl:operation>
		<wsdl:operation name="GetRoute">
			<soap12:operation soapAction="urn:GetRoute" style="document"/>
			<wsdl:input><soap12:body use="literal"/></wsdl:input>
			<wsdl:output><soap12:body use="literal"/></wsdl:output>
		</wsdl:operation>
	</wsdl:binding>
	<wsdl:binding name="ScheduleHttpGet" type="tns:ScheduleHttpGet">
		<http:binding verb="GET"/>
		<wsdl:operation name="GetAlerts">
			<http:operation location="/GetAlerts"/>
			<wsdl:input><http:urlEncoded/></wsdl:input>
			<wsdl:output><mime:mimeXml xmlns:mime="http://schemas.xmlsoap.org/wsdl/mime/" part="Body"/></wsdl:output>
		</wsdl:operation>
		<wsdl:operation name="HttpOnly">
			<http:operation location="/HttpOnly"/>
			<wsdl:input><http:urlEncoded/></wsdl:input>
			<wsdl:output><mime:mimeXml xmlns:mime="http://schemas.xmlsoap.org/wsdl/mime/" part="Body"/></wsdl:output>
		</wsdl:operation>
	</wsdl:binding>
	<wsdl:binding name="LegacyRPCBinding" type="tns:LegacyRPC">
		<soap:binding style="rpc" transport="http://schemas.xmlsoap.org/soap/http"/>
		<wsdl:operation name="Ping">
			<soap:operation soapAction="urn:Ping"/>
			<wsdl:input><soap:body use="literal" namespace="http://example.com/schedule"/></wsdl:input>
			<wsdl:output><soap:body use="literal" namespace="http://example.com/schedule"/></wsdl:output>
		</wsdl:operation>
	</wsdl:binding>

	<wsdl:service name="Schedule">
		<wsdl:port name="ScheduleSoap" binding="tns:ScheduleSoap">
			<soap:address location="http://example.com/schedule.asmx"/>
		</wsdl:port>
		<wsdl:port name="ScheduleSoap12" binding="tns:ScheduleSoap12">
			<soap12:address location="http://example.com/schedule.asmx"/>
		</wsdl:port>
		<wsdl:port name="ScheduleHttpGet" binding="tns:ScheduleHttpGet">
			<http:address location="http://example.com/schedule.asmx"/>
		</wsdl:port>
		<wsdl:port name="LegacyRPC" binding="tns:LegacyRPCBinding">
			<soap:address location="http://example.com/legacy"/>
		</wsdl:port>
	</wsdl:service>
</wsdl:definitions>`

func TestGenerator_HTTPBinding_PortTypeSkipped(t *testing.T) {
	code, err := generateWithOperations(t, httpBindingWSDL11)
	require.NoError(t, err)
	ops := string(code["operations"])

	assert.Contains(t, ops, "type ScheduleSoap interface")
	assert.Contains(t, ops, "type LegacyRPC interface")
	assert.Contains(t, ops, "type Unbound interface", "port types without any binding are kept")
	assert.NotContains(t, ops, "ScheduleHttpGet", "HTTP-only port types get no SOAP client")
	assert.NotContains(t, ops, "GetAlertsHttpGetIn")
	assert.NotContains(t, ops, "HttpOnly")

	assertCompilableSource(t, code)
}

func TestGenerator_HTTPBinding_WithOperationFilter(t *testing.T) {
	code, err := generateWithOperations(t, httpBindingWSDL11, WithOperations("GetAlerts"))
	require.NoError(t, err)

	ops := string(code["operations"])
	assert.Contains(t, ops, "type ScheduleSoap interface")
	assert.Contains(t, ops, "GetAlerts(ctx context.Context, request *GetAlertsSoapIn) (*GetAlertsResponse, error)")
	assert.NotContains(t, ops, "ScheduleHttpGet")

	types := declaredTypes(string(code["types"]))
	assert.Contains(t, types, "GetAlertsSoapIn")
	assert.Contains(t, types, "GetAlertsResponse")
	assert.NotContains(t, types, "GetAlertsHttpGetIn")
	assert.NotContains(t, types, "ArrayOfAlert", "types used only by HTTP port types are pruned")

	assertCompilableSource(t, code)
}

func TestGenerator_HTTPBinding_OperationOnlyOnHTTPIsUnknown(t *testing.T) {
	_, err := generateWithOperations(t, httpBindingWSDL11, WithOperations("HttpOnly"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown operation(s) HttpOnly")
}

func TestGenerator_HTTPBinding_ServerSkipsHTTPPortType(t *testing.T) {
	code, err := generateWithOperations(t, httpBindingWSDL11, WithServerGeneration(true))
	require.NoError(t, err)
	server := string(code["server"])

	assert.Contains(t, server, "GetAlertsSoapIn *GetAlertsSoapIn")
	assert.Contains(t, server, "GetRoute *GetRoute")
	assert.NotContains(t, server, "HttpGet")
	assert.NotContains(t, server, "HttpOnly")
}

func TestGenerator_EmptyInputMessage_Document(t *testing.T) {
	code, err := generateWithOperations(t, httpBindingWSDL11)
	require.NoError(t, err)
	types := string(code["types"])

	assert.Contains(t, types, "type GetAlertsSoapIn struct{}")
	assert.Contains(t, types, "func (GetAlertsSoapIn) MarshalXML(*xml.Encoder, xml.StartElement) error { return nil }")
	assert.NotContains(t, types, "type GetAlertsHttpGetIn", "empty messages of skipped port types are not declared")
	assert.Contains(t, string(code["operations"]), "GetAlerts(ctx context.Context, request *GetAlertsSoapIn) (*GetAlertsResponse, error)")

	assertCompilableSource(t, code)
}

func TestGenerator_EmptyInputMessage_RPC(t *testing.T) {
	code, err := generateWithOperations(t, httpBindingWSDL11)
	require.NoError(t, err)
	types := string(code["types"])

	assert.Contains(t, types, "type PingRequest struct{}")
	assert.Contains(t, types, "type OperationPingIn struct")
	assert.Contains(t, types, "type OperationPingOut struct")
	assert.Contains(t, string(code["operations"]), "Ping(ctx context.Context, request *PingRequest) (*PingResponse, error)")

	assertCompilableSource(t, code)
}

func TestGenerator_EmptyInputMessage_Generics(t *testing.T) {
	for name, opt := range map[string]Option{"generics": WithGenerics(), "generics-only": WithGenericsOnly()} {
		t.Run(name, func(t *testing.T) {
			code, err := generateWithOperations(t, httpBindingWSDL11, opt)
			require.NoError(t, err)
			ops := string(code["operations"])

			assert.Contains(t, ops, "service.client.CallResult[GetAlertsResponse](ctx, ")
			assert.NotContains(t, ops, "HttpGet")

			assertCompilableSource(t, code)
		})
	}
}
