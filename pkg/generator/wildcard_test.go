// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const wildcardWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/" xmlns:xs="http://www.w3.org/2001/XMLSchema"
             xmlns:tns="urn:wild" targetNamespace="urn:wild">
  <types>
    <xs:schema targetNamespace="urn:wild">
      <xs:complexType name="Extensible">
        <xs:sequence>
          <xs:element name="known" type="xs:string"/>
          <xs:any namespace="##other" processContents="lax" minOccurs="0" maxOccurs="unbounded"/>
        </xs:sequence>
      </xs:complexType>
      <xs:element name="Holder">
        <xs:complexType>
          <xs:sequence>
            <xs:element name="inline">
              <xs:complexType>
                <xs:sequence>
                  <xs:any processContents="skip" maxOccurs="unbounded"/>
                </xs:sequence>
              </xs:complexType>
            </xs:element>
          </xs:sequence>
        </xs:complexType>
      </xs:element>
    </xs:schema>
  </types>
</definitions>`

const wildcardRoundTrip = `package wild

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/kastrul/gowsdl/soap"
)

type ext struct {
	XMLName xml.Name ` + "`" + `xml:"urn:ext flag"` + "`" + `
	On      bool     ` + "`" + `xml:"on"` + "`" + `
}

func TestWildcardItems(t *testing.T) {
	in := ` + "`" + `<Extensible xmlns:x="urn:ext"><known>k</known><x:flag><x:on>true</x:on></x:flag><note>hi</note></Extensible>` + "`" + `
	var e Extensible
	if err := xml.Unmarshal([]byte(in), &e); err != nil {
		t.Fatal(err)
	}
	if e.Known != "k" || len(e.Items) != 2 {
		t.Fatalf("decoded %+v", e)
	}
	if e.Items[0].XMLName.Local != "flag" || e.Items[0].XMLName.Space != "urn:ext" {
		t.Fatalf("first item name %v", e.Items[0].XMLName)
	}
	var f ext
	if err := e.Items[0].Decode(&f); err != nil || !f.On {
		t.Fatalf("decoded flag %+v, %v", f, err)
	}
	var note string
	if err := e.Items[1].Decode(&note); err != nil || note != "hi" {
		t.Fatalf("decoded note %q, %v", note, err)
	}

	out, err := xml.Marshal(Extensible{Known: "k", Items: []soap.AnyElement{{Value: ext{On: true}}}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "<Items") {
		t.Fatalf("item named after the Go field: %s", out)
	}
	var back Extensible
	if err := xml.Unmarshal(out, &back); err != nil || len(back.Items) != 1 {
		t.Fatalf("re-decoded %+v, %v (xml %s)", back, err, out)
	}
	f = ext{}
	if err := back.Items[0].Decode(&f); err != nil || !f.On {
		t.Fatalf("round trip lost the flag: %+v, %v (xml %s)", f, err, out)
	}
}
`

func TestGenerator_WildcardItemsUseAnyElement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wild.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(wildcardWSDL), 0o600))

	code, out := generateCode(t, path, WithPackage("wild"))

	assert.Regexp(t, `Items\s+\[\]soap\.AnyElement\s+`+"`"+`xml:",any"`+"`", structBody(t, code, "Extensible"))
	assert.Equal(t, 2, strings.Count(code, "Items []soap.AnyElement"), "named and inline complex types")
	assert.NotContains(t, code, "Items []string")

	runGeneratedModuleTests(t, "example.com/wild", out, map[string]string{"wild_test.go": wildcardRoundTrip})
}
