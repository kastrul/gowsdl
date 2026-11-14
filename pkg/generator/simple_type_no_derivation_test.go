package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const simpleTypeNoDerivationWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/" xmlns:xs="http://www.w3.org/2001/XMLSchema"
             xmlns:tns="urn:anyfield" targetNamespace="urn:anyfield">
  <types>
    <xs:schema targetNamespace="urn:anyfield" elementFormDefault="qualified">
      <xs:simpleType name="Opaque"/>
      <xs:complexType name="Payload">
        <xs:sequence>
          <xs:element name="opaque" type="tns:Opaque"/>
        </xs:sequence>
      </xs:complexType>
    </xs:schema>
  </types>
</definitions>`

func TestGenerator_SimpleTypeWithoutDerivationIsDeclared(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opaque.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(simpleTypeNoDerivationWSDL), 0o600))

	code, out := generateCode(t, path, WithPackage("opq"))

	assert.Regexp(t, `(?m)^type Opaque string$`, code)
	assert.Regexp(t, `Opaque Opaque `+"`"+`xml:"opaque,omitempty"`, structBody(t, code, "Payload"))

	assertGeneratedCodeBuilds(t, "opq", out)
}
