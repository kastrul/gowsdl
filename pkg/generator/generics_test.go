// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"context"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const genericTestWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://example.com/test"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             targetNamespace="http://example.com/test">

  <types>
    <xsd:schema targetNamespace="http://example.com/test">
      
      <xsd:element name="GetUserRequest">
        <xsd:complexType>
          <xsd:sequence>
            <xsd:element name="userId" type="xsd:int"/>
          </xsd:sequence>
        </xsd:complexType>
      </xsd:element>
      
      <xsd:element name="GetUserResponse">
        <xsd:complexType>
          <xsd:sequence>
            <xsd:element name="user" type="tns:User"/>
            <xsd:element name="users" type="tns:User" maxOccurs="unbounded"/>
          </xsd:sequence>
        </xsd:complexType>
      </xsd:element>
      
      <xsd:complexType name="User">
        <xsd:sequence>
          <xsd:element name="id" type="xsd:int"/>
          <xsd:element name="name" type="xsd:string"/>
          <xsd:element name="email" type="xsd:string"/>
        </xsd:sequence>
      </xsd:complexType>
      
    </xsd:schema>
  </types>

  <message name="GetUserRequestMessage">
    <part name="parameters" element="tns:GetUserRequest"/>
  </message>
  
  <message name="GetUserResponseMessage">
    <part name="parameters" element="tns:GetUserResponse"/>
  </message>

  <portType name="UserServicePortType">
    <operation name="GetUser">
      <input message="tns:GetUserRequestMessage"/>
      <output message="tns:GetUserResponseMessage"/>
    </operation>
  </portType>

  <binding name="UserServiceBinding" type="tns:UserServicePortType">
    <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
    <operation name="GetUser">
      <soap:operation soapAction="GetUser"/>
      <input>
        <soap:body use="literal"/>
      </input>
      <output>
        <soap:body use="literal"/>
      </output>
    </operation>
  </binding>

  <service name="UserService">
    <port name="UserServicePort" binding="tns:UserServiceBinding">
      <soap:address location="http://localhost:8080/userservice"/>
    </port>
  </service>

</definitions>`

func generateOperations(t *testing.T, wsdl string, opts ...Option) string {
	t.Helper()
	wsdlPath := filepath.Join(t.TempDir(), "service.wsdl")
	require.NoError(t, os.WriteFile(wsdlPath, []byte(wsdl), 0644))

	g, err := New(wsdlPath, append([]Option{WithPackage("test"), WithExportAllTypes(true)}, opts...)...)
	require.NoError(t, err)

	code, err := g.Generate(context.Background())
	require.NoError(t, err)
	return string(code["operations"])
}

func TestGenerator_Generics_WSDL11(t *testing.T) {
	genericSnippets := []string{
		"GetUserGeneric(ctx context.Context, request *GetUserRequest) (soap.Result[GetUserResponse], error)",
		"getUserClient *soap.GenericClient[GetUserRequest, GetUserResponse]",
		`getUserClient: soap.NewGenericClientFromClient[GetUserRequest, GetUserResponse](client, "GetUser")`,
		"func (service *userServicePortType) GetUserGeneric(ctx context.Context, request *GetUserRequest) (soap.Result[GetUserResponse], error)",
		"return soap.Result[GetUserResponse]{}, soap.ErrNilRequest",
		"return service.getUserClient.CallAsResult(ctx, *request), nil",
	}
	standardSnippet := "GetUser(ctx context.Context, request *GetUserRequest) (*GetUserResponse, error)"

	t.Run("enabled", func(t *testing.T) {
		code := generateOperations(t, genericTestWSDL, WithGenerics())
		assert.Contains(t, code, standardSnippet)
		for _, s := range genericSnippets {
			assert.Contains(t, code, s)
		}
	})

	t.Run("disabled", func(t *testing.T) {
		code := generateOperations(t, genericTestWSDL)
		assert.Contains(t, code, standardSnippet)
		assert.NotContains(t, code, "Generic")
		assert.NotContains(t, code, "soap.Result[")
	})

	t.Run("RPC port types are skipped", func(t *testing.T) {
		rpcWSDL := strings.Replace(genericTestWSDL, `style="document"`, `style="rpc"`, 1)
		require.NotEqual(t, genericTestWSDL, rpcWSDL)

		code := generateOperations(t, rpcWSDL, WithGenerics())
		assert.Contains(t, code, standardSnippet)
		assert.NotContains(t, code, "GenericClient")
		assert.NotContains(t, code, "GetUserGeneric")
	})

	t.Run("generics only", func(t *testing.T) {
		code := generateOperations(t, genericTestWSDL, WithGenericsOnly())
		assert.Contains(t, code, "GetUser(ctx context.Context, request *GetUserRequest) (soap.Result[GetUserResponse], error)")
		assert.Contains(t, code, "func (service *userServicePortType) GetUser(ctx context.Context, request *GetUserRequest) (soap.Result[GetUserResponse], error)")
		assert.Contains(t, code, "getUserClient *soap.GenericClient[GetUserRequest, GetUserResponse]")
		assert.Contains(t, code, "return service.getUserClient.CallAsResult(ctx, *request), nil")
		assert.NotContains(t, code, "GetUserGeneric")
		assert.NotContains(t, code, "(*GetUserResponse, error)")
	})

	t.Run("generics only keeps standard methods for RPC port types", func(t *testing.T) {
		rpcWSDL := strings.Replace(genericTestWSDL, `style="document"`, `style="rpc"`, 1)

		code := generateOperations(t, rpcWSDL, WithGenericsOnly())
		assert.Contains(t, code, standardSnippet)
		assert.NotContains(t, code, "GenericClient")
		assert.NotContains(t, code, "soap.Result[")
	})
}

func TestGenerator_Generics_DocCommentStaysAttached(t *testing.T) {
	documented := strings.Replace(genericTestWSDL,
		`<operation name="GetUser">`,
		`<operation name="GetUser">
      <documentation>Fetches a user.</documentation>`, 1)
	require.NotEqual(t, genericTestWSDL, documented)

	modes := map[string][]Option{
		"standard":      nil,
		"use generics":  {WithGenerics()},
		"generics only": {WithGenericsOnly()},
	}
	for name, opts := range modes {
		t.Run(name, func(t *testing.T) {
			code := generateOperations(t, documented, opts...)
			formatted, err := format.Source([]byte("package test\n" + code))
			require.NoError(t, err)
			assert.Contains(t, string(formatted), "// Fetches a user.\n\tGetUser(ctx context.Context")
		})
	}
}
