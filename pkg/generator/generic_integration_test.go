// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build integration
// +build integration

package generator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/kastrul/gowsdl/pkg/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenericCodeGeneration(t *testing.T) {
	rootDir := moduleRoot(t)

	// The generated packages must live inside the module so that they
	// resolve github.com/kastrul/gowsdl/soap without a go.mod.
	tmpDir, err := os.MkdirTemp(rootDir, "tmp-generic-test-")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	// Write WSDL file
	wsdlPath := filepath.Join(tmpDir, "test.wsdl")
	err = os.WriteFile(wsdlPath, []byte(genericTestWSDL), 0644)
	require.NoError(t, err)

	tests := []struct {
		name         string
		useGenerics  bool
		checkContent []string
	}{
		{
			name:        "Standard code generation",
			useGenerics: false,
			checkContent: []string{
				"type UserServicePortType interface",
				"GetUser(ctx context.Context, request *GetUserRequest) (*GetUserResponse, error)",
			},
		},
		{
			name:        "Generic code generation",
			useGenerics: true,
			checkContent: []string{
				"type UserServicePortType interface",
				"GetUser(ctx context.Context, request *GetUserRequest) (*GetUserResponse, error)",
				"GetUserGeneric(request *GetUserRequest) (soap.Result[GetUserResponse]",
				"GetUserGenericContext(ctx context.Context",
				"getUserClient *soap.GenericClient[GetUserRequest, GetUserResponse]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create package directory
			pkgName := "testpkg"
			if tt.useGenerics {
				pkgName = "testpkggen"
			}

			// Generate code
			args := []string{
				"run", "cmd/gowsdl/main.go",
				"-p", pkgName,
				"-o", "generated.go",
				"-d", tmpDir,
			}
			if tt.useGenerics {
				args = append(args, "-use-generics")
			}
			args = append(args, wsdlPath)

			cmd := exec.Command("go", args...)
			cmd.Dir = rootDir
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Failed to generate code: %s", string(output))

			// Read generated file
			pkgDir := filepath.Join(tmpDir, pkgName)
			content, err := os.ReadFile(filepath.Join(pkgDir, "generated.go"))
			require.NoError(t, err)

			// Check content
			contentStr := string(content)
			for _, check := range tt.checkContent {
				assert.Contains(t, contentStr, check, "Expected content not found: %s", check)
			}

			// Verify it compiles
			cmd = exec.Command("go", "build", ".")
			cmd.Dir = pkgDir
			output, err = cmd.CombinedOutput()
			require.NoError(t, err, "Generated code does not compile: %s", string(output))
		})
	}
}

func TestGenericClientUsage(t *testing.T) {
	wsdlPath := filepath.Join(t.TempDir(), "test_generic.wsdl")
	require.NoError(t, os.WriteFile(wsdlPath, []byte(genericTestWSDL), 0644))

	g, err := New(wsdlPath,
		WithPackage("testgen"),
		WithHTTPConfig(http.DefaultHTTPClientConfig()),
		WithExportAllTypes(true),
		WithServerGeneration(true),
		WithGenerics(),
	)
	require.NoError(t, err)

	code, err := g.Generate(context.Background())
	require.NoError(t, err)

	operationsCode := string(code["operations"])
	assert.Contains(t, operationsCode, "soap.GenericClient")
	assert.Contains(t, operationsCode, "soap.Result[")
	assert.Contains(t, operationsCode, "CallAsResult")
}

// moduleRoot returns the directory containing the module's go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "go.mod not found")
		dir = parent
	}
}

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
