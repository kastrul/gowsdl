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
	assert.NotRegexp(t, `(?m)^\s*(any|interface\{\})\s*$`, code, "anyType base must not be embedded")

	assertGeneratedCodeBuilds(t, "anyext", out)
}

const anyTypeFieldWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/" xmlns:xs="http://www.w3.org/2001/XMLSchema"
             xmlns:tns="urn:anyfield" targetNamespace="urn:anyfield">
  <types>
    <xs:schema targetNamespace="urn:anyfield" elementFormDefault="qualified">
      <xs:complexType name="Payload">
        <xs:sequence>
          <xs:element name="value" type="xs:anyType"/>
          <xs:element name="maybe" type="xs:anyType" nillable="true"/>
          <xs:element name="simple" type="xs:anySimpleType"/>
        </xs:sequence>
      </xs:complexType>
      <xs:element name="tags">
        <xs:simpleType>
          <xs:list itemType="xs:string"/>
        </xs:simpleType>
      </xs:element>
    </xs:schema>
  </types>
</definitions>`

func TestGenerator_AnyTypeFieldsUseAny(t *testing.T) {
	path := filepath.Join(t.TempDir(), "anyfield.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(anyTypeFieldWSDL), 0o600))

	code, out := generateCode(t, path, WithPackage("anyfield"))

	payload := structBody(t, code, "Payload")
	assert.Regexp(t, `Value\s+any\s`, payload)
	assert.Regexp(t, `Maybe\s+any\s`, payload, "nillable anyType must not become *any")
	assert.Regexp(t, `Simple\s+any\s`, payload)
	assert.Regexp(t, `(?m)^type Tags string$`, code)
	assert.NotContains(t, code, "interface{}")

	assertGeneratedCodeBuilds(t, "anyfield", out)
}
