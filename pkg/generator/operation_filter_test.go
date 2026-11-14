package generator

import (
	"context"
	"go/format"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const operationFilterWSDL11 = `<?xml version="1.0" encoding="UTF-8"?>
<wsdl:definitions xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
                  xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"
                  xmlns:xsd="http://www.w3.org/2001/XMLSchema"
                  xmlns:tns="http://example.com/test"
                  targetNamespace="http://example.com/test">
	<wsdl:types>
		<xsd:schema targetNamespace="http://example.com/test">
			<xsd:element name="GetUser" type="xsd:string"/>
			<xsd:element name="GetUserResponse" type="xsd:string"/>
			<xsd:element name="DeleteUser" type="xsd:string"/>
			<xsd:element name="DeleteUserResponse" type="xsd:string"/>
			<xsd:element name="Ping" type="xsd:string"/>
			<xsd:element name="PingResponse" type="xsd:string"/>
		</xsd:schema>
	</wsdl:types>
	<wsdl:message name="GetUserIn"><wsdl:part name="parameters" element="tns:GetUser"/></wsdl:message>
	<wsdl:message name="GetUserOut"><wsdl:part name="parameters" element="tns:GetUserResponse"/></wsdl:message>
	<wsdl:message name="DeleteUserIn"><wsdl:part name="parameters" element="tns:DeleteUser"/></wsdl:message>
	<wsdl:message name="DeleteUserOut"><wsdl:part name="parameters" element="tns:DeleteUserResponse"/></wsdl:message>
	<wsdl:message name="PingIn"><wsdl:part name="parameters" element="tns:Ping"/></wsdl:message>
	<wsdl:message name="PingOut"><wsdl:part name="parameters" element="tns:PingResponse"/></wsdl:message>
	<wsdl:portType name="UserPort">
		<wsdl:operation name="GetUser">
			<wsdl:input message="tns:GetUserIn"/>
			<wsdl:output message="tns:GetUserOut"/>
		</wsdl:operation>
		<wsdl:operation name="DeleteUser">
			<wsdl:input message="tns:DeleteUserIn"/>
			<wsdl:output message="tns:DeleteUserOut"/>
		</wsdl:operation>
	</wsdl:portType>
	<wsdl:portType name="HealthPort">
		<wsdl:operation name="Ping">
			<wsdl:input message="tns:PingIn"/>
			<wsdl:output message="tns:PingOut"/>
		</wsdl:operation>
	</wsdl:portType>
	<wsdl:binding name="UserBinding" type="tns:UserPort">
		<soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
		<wsdl:operation name="GetUser"><soap:operation soapAction="urn:GetUser"/></wsdl:operation>
		<wsdl:operation name="DeleteUser"><soap:operation soapAction="urn:DeleteUser"/></wsdl:operation>
	</wsdl:binding>
	<wsdl:binding name="HealthBinding" type="tns:HealthPort">
		<soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
		<wsdl:operation name="Ping"><soap:operation soapAction="urn:Ping"/></wsdl:operation>
	</wsdl:binding>
</wsdl:definitions>`

const operationFilterWSDL20 = `<?xml version="1.0" encoding="UTF-8"?>
<description xmlns="http://www.w3.org/ns/wsdl/"
             xmlns:tns="http://example.com/test"
             xmlns:xs="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/test">
	<types>
		<xs:schema targetNamespace="http://example.com/test">
			<xs:element name="GetUser" type="xs:string"/>
			<xs:element name="GetUserResponse" type="xs:string"/>
			<xs:element name="DeleteUser" type="xs:string"/>
			<xs:element name="DeleteUserResponse" type="xs:string"/>
		</xs:schema>
	</types>
	<interface name="UserInterface">
		<operation name="GetUser" pattern="http://www.w3.org/ns/wsdl/in-out">
			<input messageLabel="In" element="tns:GetUser"/>
			<output messageLabel="Out" element="tns:GetUserResponse"/>
		</operation>
		<operation name="DeleteUser" pattern="http://www.w3.org/ns/wsdl/in-out">
			<input messageLabel="In" element="tns:DeleteUser"/>
			<output messageLabel="Out" element="tns:DeleteUserResponse"/>
		</operation>
	</interface>
</description>`

func generateWithOperations(t *testing.T, wsdl string, opts ...Option) (map[string][]byte, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "service.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(wsdl), 0o644))

	g, err := New(path, append([]Option{WithPackage("svc"), WithExportAllTypes(true)}, opts...)...)
	require.NoError(t, err)
	return g.Generate(context.Background())
}

func assertCompilableSource(t *testing.T, code map[string][]byte) {
	t.Helper()
	src := append(append(append([]byte{}, code["header"]...), code["types"]...), code["operations"]...)
	_, err := format.Source(src)
	require.NoError(t, err, "generated code must be valid Go")
}

func TestGenerator_OperationFilter_WSDL11(t *testing.T) {
	code, err := generateWithOperations(t, operationFilterWSDL11, WithOperations("GetUser"))
	require.NoError(t, err)
	ops := string(code["operations"])

	assert.Contains(t, ops, "GetUser(ctx context.Context")
	assert.NotContains(t, ops, "DeleteUser(ctx context.Context")
	assert.NotContains(t, ops, "type HealthPort interface", "port type with no selected operations should be omitted")
	assert.Contains(t, ops, "urn:GetUser")

	types := string(code["types"])
	assert.NotContains(t, types, "DeleteUser", "types of excluded operations are pruned by default")

	assertCompilableSource(t, code)
}

func TestGenerator_OperationFilter_WSDL11_AcrossPortTypes(t *testing.T) {
	code, err := generateWithOperations(t, operationFilterWSDL11, WithOperations("DeleteUser", "Ping"))
	require.NoError(t, err)
	ops := string(code["operations"])

	assert.Contains(t, ops, "type UserPort interface")
	assert.Contains(t, ops, "type HealthPort interface")
	assert.Contains(t, ops, "DeleteUser(ctx context.Context")
	assert.Contains(t, ops, "Ping(ctx context.Context")
	assert.NotContains(t, ops, "GetUser(ctx context.Context")

	assertCompilableSource(t, code)
}

func TestGenerator_OperationFilter_WSDL20(t *testing.T) {
	code, err := generateWithOperations(t, operationFilterWSDL20, WithOperations("DeleteUser"))
	require.NoError(t, err)
	ops := string(code["operations"])

	assert.Contains(t, ops, "DeleteUser(ctx context.Context")
	assert.NotContains(t, ops, "GetUser(ctx context.Context")

	assertCompilableSource(t, code)
}

func TestGenerator_OperationFilter_NoFilterGeneratesAll(t *testing.T) {
	code, err := generateWithOperations(t, operationFilterWSDL11, WithOperations())
	require.NoError(t, err)
	ops := string(code["operations"])

	for _, name := range []string{"GetUser", "DeleteUser", "Ping"} {
		assert.Contains(t, ops, name+"(ctx context.Context")
	}
}

func TestGenerator_OperationFilter_DoesNotAffectServer(t *testing.T) {
	code, err := generateWithOperations(t, operationFilterWSDL11, WithOperations("GetUser"), WithServerGeneration(true))
	require.NoError(t, err)
	assert.Contains(t, string(code["server"]), "DeleteUser")
}

func TestGenerator_OperationFilter_UnknownOperation(t *testing.T) {
	_, err := generateWithOperations(t, operationFilterWSDL11, WithOperations("GetUser", "Nope", "getuser"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown operation(s) Nope, getuser")
	assert.Contains(t, err.Error(), "available operations: DeleteUser, GetUser, Ping")
}

func TestWithOperations_IgnoresBlankNames(t *testing.T) {
	g := &Generator{}
	require.NoError(t, WithOperations(" GetUser ", "", "  ")(g))
	assert.Equal(t, map[string]bool{"GetUser": true}, g.operations)

	g = &Generator{}
	require.NoError(t, WithOperations("", " ")(g))
	assert.Nil(t, g.operations)
}

func TestParseOperationList(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{" , ,", nil},
		{"GetUser", []string{"GetUser"}},
		{"GetUser, DeleteUser ,Ping,", []string{"GetUser", "DeleteUser", "Ping"}},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ParseOperationList(tt.in), "input %q", tt.in)
	}
}
