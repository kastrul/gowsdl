package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const anyTypeExtensionWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/" xmlns:xs="http://www.w3.org/2001/XMLSchema"
             xmlns:tns="urn:any" targetNamespace="urn:any">
  <types>
    <xs:schema targetNamespace="urn:any" elementFormDefault="qualified">
      <xs:complexType name="Attribute">
        <xs:complexContent mixed="true">
          <xs:extension base="xs:anyType">
            <xs:attribute name="id" type="xs:anyURI" use="required"/>
          </xs:extension>
        </xs:complexContent>
      </xs:complexType>
      <xs:element name="holder">
        <xs:complexType>
          <xs:sequence>
            <xs:element name="inline">
              <xs:complexType>
                <xs:complexContent>
                  <xs:extension base="xs:anyType">
                    <xs:attribute name="kind" type="xs:string"/>
                  </xs:extension>
                </xs:complexContent>
              </xs:complexType>
            </xs:element>
          </xs:sequence>
        </xs:complexType>
      </xs:element>
    </xs:schema>
  </types>
</definitions>`

func TestGenerator_ComplexContentExtensionOfAnyType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "any.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(anyTypeExtensionWSDL), 0o600))

	code, out := generateCode(t, path, WithPackage("anyext"))

	attr := structBody(t, code, "Attribute")
	assert.Regexp(t, `InnerXML string `+"`"+`xml:",innerxml"`+"`", attr)
	assert.Regexp(t, `IdAttr string `+"`"+`xml:"id,attr`, attr)
	assert.NotContains(t, code, "interface{}\n")

	assertGeneratedCodeBuilds(t, "anyext", out)
}
