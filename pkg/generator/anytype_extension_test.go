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

	out := generate(t, path, WithPackage("anyext"))
	code := out.client(t)

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
      <xs:element name="blob" type="xs:anyType"/>
      <xs:element name="tags">
        <xs:simpleType>
          <xs:list itemType="xs:string"/>
        </xs:simpleType>
      </xs:element>
    </xs:schema>
  </types>
</definitions>`

const anyTypeFieldRoundTrip = `package anyfield

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
)

type item struct {
	ID   int    ` + "`" + `xml:"urn:item id"` + "`" + `
	Name string ` + "`" + `xml:"name"` + "`" + `
}

func TestAnyTypeFields(t *testing.T) {
	in := ` + "`" + `<Envelope xmlns:i="urn:item"><Payload><value><i:id>7</i:id><name>Jane</name></value><simple>s</simple></Payload></Envelope>` + "`" + `
	var env struct {
		Payload Payload ` + "`" + `xml:"Payload"` + "`" + `
	}
	if err := xml.Unmarshal([]byte(in), &env); err != nil {
		t.Fatal(err)
	}
	p := env.Payload
	if p.Simple != "s" {
		t.Fatalf("Simple = %q", p.Simple)
	}
	var got item
	if err := p.Value.Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got != (item{ID: 7, Name: "Jane"}) {
		t.Fatalf("decoded %+v", got)
	}
	if !p.Maybe.IsZero() {
		t.Fatalf("Maybe should be unset: %+v", p.Maybe)
	}

	out, err := xml.Marshal(Payload{Value: p.Value})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "maybe") {
		t.Fatalf("unset anyType field was encoded: %s", out)
	}
	var again Payload
	if err := xml.Unmarshal(out, &again); err != nil {
		t.Fatal(err)
	}
	got = item{}
	if err := again.Value.Decode(&got); err != nil || got.ID != 7 {
		t.Fatalf("re-encoded value decoded to %+v, %v (xml %s)", got, err, out)
	}

	js, err := json.Marshal(Payload{Simple: "s"})
	if err != nil {
		t.Fatal(err)
	}
	if string(js) != ` + "`" + `{"simple":"s"}` + "`" + ` {
		t.Fatalf("json = %s", js)
	}
}

func TestAnyTypeGlobalElement(t *testing.T) {
	var b Blob
	if err := xml.Unmarshal([]byte(` + "`" + `<blob><id>3</id></blob>` + "`" + `), &b); err != nil {
		t.Fatal(err)
	}
	var got struct {
		ID int ` + "`" + `xml:"id"` + "`" + `
	}
	if err := b.Decode(&got); err != nil || got.ID != 3 {
		t.Fatalf("decoded %+v, %v", got, err)
	}
	out, err := xml.Marshal(Blob{Value: got})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "<Blob><id>3</id></Blob>" {
		t.Fatalf("encoded %s", out)
	}
}
`

func TestGenerator_AnyTypeFieldsUseSoapAnyType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "anyfield.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(anyTypeFieldWSDL), 0o600))

	out := generate(t, path, WithPackage("anyfield"))
	code := out.client(t)

	payload := structBody(t, code, "Payload")
	assert.Regexp(t, `Value\s+soap\.AnyType\s`, payload)
	assert.Regexp(t, `Maybe\s+soap\.AnyType\s`, payload, "nillable anyType must not become a pointer")
	assert.Regexp(t, `Simple\s+string\s`, payload)
	assert.Regexp(t, `(?m)^type Blob soap\.AnyType$`, code)
	assert.Contains(t, code, "func (a *Blob) UnmarshalXML(")
	assert.Regexp(t, `(?m)^type Tags string$`, code)
	assert.NotRegexp(t, `\sany\s+`+"`", code, "no generated field may be typed any")

	runGeneratedModuleTests(t, "example.com/anyfield", out, map[string]string{"anyfield_test.go": anyTypeFieldRoundTrip})
}
