package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// inlineSimpleTypeFallbackWSDL declares global elements whose inline
// simpleType has neither a union nor a restriction base: an empty one and a
// list.
const inlineSimpleTypeFallbackWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/" xmlns:xs="http://www.w3.org/2001/XMLSchema"
             xmlns:tns="urn:ist" targetNamespace="urn:ist">
  <types>
    <xs:schema targetNamespace="urn:ist" elementFormDefault="qualified">
      <xs:element name="note">
        <xs:simpleType/>
      </xs:element>
      <xs:element name="tags">
        <xs:simpleType>
          <xs:list itemType="xs:string"/>
        </xs:simpleType>
      </xs:element>
    </xs:schema>
  </types>
</definitions>`

const inlineSimpleTypeFallbackRoundTrip = `package ist

import (
	"encoding/xml"
	"testing"
)

func TestDecodeInlineFallbackSimpleTypes(t *testing.T) {
	var n Note
	if err := xml.Unmarshal([]byte(` + "`" + `<note>x</note>` + "`" + `), &n); err != nil {
		t.Fatal(err)
	}
	var tags Tags
	if err := xml.Unmarshal([]byte(` + "`" + `<tags>a b c</tags>` + "`" + `), &tags); err != nil {
		t.Fatal(err)
	}
	if n != "x" || tags != "a b c" {
		t.Fatalf("unexpected decode: %q %q", n, tags)
	}
	out, err := xml.Marshal(tags)
	if err != nil {
		t.Fatal(err)
	}
	var again Tags
	if err := xml.Unmarshal(out, &again); err != nil {
		t.Fatal(err)
	}
	if again != tags {
		t.Fatalf("round trip changed value: %q != %q", again, tags)
	}
}
`

func TestGenerator_InlineSimpleTypeWithoutBaseIsString(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ist.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(inlineSimpleTypeFallbackWSDL), 0o600))

	out := generate(t, path, WithPackage("ist"))
	code := out.client(t)

	assert.Regexp(t, `(?m)^type Note string$`, code)
	assert.Regexp(t, `(?m)^type Tags string$`, code)

	runGeneratedModuleTests(t, "example.com/ist", out, map[string]string{"ist_test.go": inlineSimpleTypeFallbackRoundTrip})
}
