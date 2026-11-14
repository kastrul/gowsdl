package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// simpleTypeFallbackWSDL declares named simpleTypes that have no base type
// the generator can map: an empty one, and ones whose restriction, list or
// union is given by a nested simpleType instead of an attribute.
const simpleTypeFallbackWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/" xmlns:xs="http://www.w3.org/2001/XMLSchema"
             xmlns:tns="urn:st" targetNamespace="urn:st">
  <types>
    <xs:schema targetNamespace="urn:st" elementFormDefault="qualified">
      <xs:simpleType name="Opaque"/>
      <xs:simpleType name="Code">
        <xs:restriction>
          <xs:simpleType><xs:restriction base="xs:string"><xs:maxLength value="8"/></xs:restriction></xs:simpleType>
          <xs:pattern value="[A-Z]+"/>
        </xs:restriction>
      </xs:simpleType>
      <xs:simpleType name="Codes">
        <xs:list><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:list>
      </xs:simpleType>
      <xs:simpleType name="SizeOrAuto">
        <xs:union>
          <xs:simpleType><xs:restriction base="xs:int"/></xs:simpleType>
          <xs:simpleType><xs:restriction base="xs:string"><xs:enumeration value="auto"/></xs:restriction></xs:simpleType>
        </xs:union>
      </xs:simpleType>
      <xs:complexType name="Payload">
        <xs:sequence>
          <xs:element name="opaque" type="tns:Opaque"/>
          <xs:element name="code" type="tns:Code"/>
          <xs:element name="codes" type="tns:Codes"/>
          <xs:element name="size" type="tns:SizeOrAuto"/>
        </xs:sequence>
        <xs:attribute name="kind" type="tns:Code"/>
      </xs:complexType>
    </xs:schema>
  </types>
</definitions>`

const simpleTypeFallbackRoundTrip = `package st

import (
	"encoding/xml"
	"testing"
)

func TestDecodeFallbackSimpleTypes(t *testing.T) {
	in := ` + "`" + `<Payload kind="ID"><opaque>x</opaque><code>AB</code><codes>a b c</codes><size>auto</size></Payload>` + "`" + `
	var p Payload
	if err := xml.Unmarshal([]byte(in), &p); err != nil {
		t.Fatal(err)
	}
	if p.Opaque != "x" || p.Code != "AB" || p.Codes != "a b c" || p.Size != "auto" || p.Kind != "ID" {
		t.Fatalf("unexpected decode: %+v", p)
	}
	out, err := xml.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var again Payload
	if err := xml.Unmarshal(out, &again); err != nil {
		t.Fatal(err)
	}
	if again != p {
		t.Fatalf("round trip changed value: %+v != %+v", again, p)
	}
}
`

func TestGenerator_SimpleTypeWithoutMappableBaseIsString(t *testing.T) {
	path := filepath.Join(t.TempDir(), "st.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(simpleTypeFallbackWSDL), 0o600))

	out := generate(t, path, WithPackage("st"))
	code := out.client(t)

	for _, name := range []string{"Opaque", "Code", "Codes", "SizeOrAuto"} {
		assert.Regexp(t, `(?m)^type `+name+` string$`, code)
	}

	runGeneratedModuleTests(t, "example.com/st", out, map[string]string{"st_test.go": simpleTypeFallbackRoundTrip})
}
