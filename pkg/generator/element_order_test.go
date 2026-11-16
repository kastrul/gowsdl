// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// elementOrderWSDL declares content models whose elements sit in sequences
// and choices nested at several depths, in a global element's anonymous
// type, a named type, a complexContent extension and restriction, a
// top-level choice and an inline type. In each, the elements' names follow
// the alphabet in schema document order after their type's prefix.
const elementOrderWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="urn:order"
             xmlns:xs="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:order">
  <types>
    <xs:schema targetNamespace="urn:order" elementFormDefault="qualified">
      <xs:complexType name="Base">
        <xs:sequence><xs:element name="baseA" type="xs:string"/></xs:sequence>
      </xs:complexType>
      <xs:element name="Order">
        <xs:complexType>
          <xs:sequence>
            <xs:element name="a" type="xs:string"/>
            <xs:choice>
              <xs:element name="b" type="xs:string"/>
              <xs:element name="c" type="xs:string"/>
            </xs:choice>
            <xs:sequence>
              <xs:element name="d" type="xs:string"/>
              <xs:choice>
                <xs:element name="e" type="xs:string"/>
              </xs:choice>
              <xs:element name="f" type="xs:string"/>
            </xs:sequence>
            <xs:element name="g" type="xs:int"/>
            <xs:choice>
              <xs:sequence>
                <xs:element name="h" type="xs:string"/>
                <xs:element name="i" type="xs:string"/>
              </xs:sequence>
              <xs:element name="j" type="xs:string"/>
            </xs:choice>
            <xs:element name="k" type="tns:Detail"/>
            <xs:element name="l">
              <xs:complexType>
                <xs:sequence>
                  <xs:element name="lA" type="xs:string"/>
                  <xs:choice><xs:element name="lB" type="xs:string"/></xs:choice>
                  <xs:element name="lC" type="xs:string"/>
                </xs:sequence>
              </xs:complexType>
            </xs:element>
            <xs:element name="m" type="tns:TopChoice"/>
            <xs:element name="n" type="tns:Restricted"/>
            <xs:element name="o" type="xs:string"/>
          </xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:complexType name="Detail">
        <xs:complexContent>
          <xs:extension base="tns:Base">
            <xs:sequence>
              <xs:element name="extA" type="xs:string"/>
              <xs:choice><xs:element name="extB" type="xs:string"/></xs:choice>
              <xs:sequence><xs:element name="extC" type="xs:string"/></xs:sequence>
              <xs:element name="extD" type="xs:string"/>
            </xs:sequence>
          </xs:extension>
        </xs:complexContent>
      </xs:complexType>
      <xs:complexType name="TopChoice">
        <xs:choice>
          <xs:sequence>
            <xs:element name="tcA" type="xs:string"/>
            <xs:element name="tcB" type="xs:string"/>
          </xs:sequence>
          <xs:element name="tcC" type="xs:string"/>
        </xs:choice>
      </xs:complexType>
      <xs:complexType name="Restricted">
        <xs:complexContent>
          <xs:restriction base="xs:anyType">
            <xs:sequence>
              <xs:element name="rA" type="xs:string"/>
              <xs:sequence><xs:element name="rB" type="xs:string"/></xs:sequence>
              <xs:element name="rC" type="xs:string"/>
            </xs:sequence>
          </xs:restriction>
        </xs:complexContent>
      </xs:complexType>
      <xs:element name="OrderResponse">
        <xs:complexType>
          <xs:sequence><xs:element name="ok" type="xs:boolean"/></xs:sequence>
        </xs:complexType>
      </xs:element>
    </xs:schema>
  </types>
  <message name="OrderIn"><part name="parameters" element="tns:Order"/></message>
  <message name="OrderOut"><part name="parameters" element="tns:OrderResponse"/></message>
  <portType name="OrderPort">
    <operation name="PlaceOrder"><input message="tns:OrderIn"/><output message="tns:OrderOut"/></operation>
  </portType>
  <binding name="OrderBinding" type="tns:OrderPort">
    <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
    <operation name="PlaceOrder">
      <soap:operation soapAction="urn:order:PlaceOrder"/>
      <input><soap:body use="literal"/></input>
      <output><soap:body use="literal"/></output>
    </operation>
  </binding>
  <service name="OrderService">
    <port name="OrderPort" binding="tns:OrderBinding"><soap:address location="http://localhost/"/></port>
  </service>
</definitions>`

// elementOrderSchemaSequence is every element of a fully populated Order
// instance in the order the schema puts them: the elements of Order's
// content model in document order, each followed by its own content.
var elementOrderSchemaSequence = []string{
	"Order",
	"a", "b", "c", "d", "e", "f", "g", "h", "i", "j",
	"k", "baseA", "extA", "extB", "extC", "extD",
	"l", "lA", "lB", "lC",
	"m", "tcA", "tcB", "tcC",
	"n", "rA", "rB", "rC",
	"o",
}

func TestGenerator_ElementOrder_FollowsSchemaDocumentOrder(t *testing.T) {
	out := generate(t, writeWSDL(t, elementOrderWSDL), WithPackage("order"))

	code := out.client(t)

	assert.Equal(t, []string{
		"XMLName xml.Name `xml:\"urn:order Order\"`",
		"A string `xml:\"a\" json:\"a,omitempty\"`",
		"B string `xml:\"b,omitempty\" json:\"b,omitempty\"`",
		"C string `xml:\"c,omitempty\" json:\"c,omitempty\"`",
		"D string `xml:\"d\" json:\"d,omitempty\"`",
		"E string `xml:\"e,omitempty\" json:\"e,omitempty\"`",
		"F string `xml:\"f\" json:\"f,omitempty\"`",
		"G int32 `xml:\"g\" json:\"g,omitempty\"`",
		"H string `xml:\"h,omitempty\" json:\"h,omitempty\"`",
		"I string `xml:\"i,omitempty\" json:\"i,omitempty\"`",
		"J string `xml:\"j,omitempty\" json:\"j,omitempty\"`",
		"K Detail `xml:\"k,omitempty\" json:\"k,omitzero\"`",
		"L OrderLType `xml:\"l,omitempty\" json:\"l,omitzero\"`",
		"M TopChoice `xml:\"m,omitempty\" json:\"m,omitzero\"`",
		"N Restricted `xml:\"n,omitempty\" json:\"n,omitzero\"`",
		"O string `xml:\"o\" json:\"o,omitempty\"`",
	}, fieldLines(t, code, "Order"))

	assert.Equal(t, []string{
		"Base",
		"ExtA string `xml:\"extA\" json:\"extA,omitempty\"`",
		"ExtB string `xml:\"extB,omitempty\" json:\"extB,omitempty\"`",
		"ExtC string `xml:\"extC\" json:\"extC,omitempty\"`",
		"ExtD string `xml:\"extD\" json:\"extD,omitempty\"`",
	}, fieldLines(t, code, "Detail"))

	assert.Equal(t, []string{
		"LA string `xml:\"lA\" json:\"lA,omitempty\"`",
		"LB string `xml:\"lB,omitempty\" json:\"lB,omitempty\"`",
		"LC string `xml:\"lC\" json:\"lC,omitempty\"`",
	}, fieldLines(t, code, "OrderLType"))

	assert.Equal(t, []string{
		"TcA string `xml:\"tcA,omitempty\" json:\"tcA,omitempty\"`",
		"TcB string `xml:\"tcB,omitempty\" json:\"tcB,omitempty\"`",
		"TcC string `xml:\"tcC,omitempty\" json:\"tcC,omitempty\"`",
	}, fieldLines(t, code, "TopChoice"))

	assert.Equal(t, []string{
		"RA string `xml:\"rA\" json:\"rA,omitempty\"`",
		"RB string `xml:\"rB\" json:\"rB,omitempty\"`",
		"RC string `xml:\"rC\" json:\"rC,omitempty\"`",
	}, fieldLines(t, code, "Restricted"))
}

// elementOrderMarshalTest marshals a fully populated Order and compares the
// sequence of its elements with the schema's, then decodes it back.
const elementOrderMarshalTest = `package order

import (
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
)

func TestMarshalledElementsFollowSchemaOrder(t *testing.T) {
	in := Order{
		A: "a", B: "b", C: "c", D: "d", E: "e", F: "f", G: 7, H: "h", I: "i", J: "j",
		K: Detail{Base: Base{BaseA: "baseA"}, ExtA: "extA", ExtB: "extB", ExtC: "extC", ExtD: "extD"},
		L: OrderLType{LA: "lA", LB: "lB", LC: "lC"},
		M: TopChoice{TcA: "tcA", TcB: "tcB", TcC: "tcC"},
		N: Restricted{RA: "rA", RB: "rB", RC: "rC"},
		O: "o",
	}
	b, err := xml.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	dec := xml.NewDecoder(strings.NewReader(string(b)))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok {
			got = append(got, se.Name.Local)
		}
	}
	want := strings.Split("SCHEMA_ORDER", ",")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("element order\ngot  %v\nwant %v\nrequest: %s", got, want, b)
	}

	var back Order
	if err := xml.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	back.XMLName = in.XMLName
	if !reflect.DeepEqual(back, in) {
		t.Errorf("round trip\ngot  %+v\nwant %+v", back, in)
	}
}
`

// fieldLines returns the non-blank lines between the braces of the struct
// type name declared in code, trimmed and with runs of blanks collapsed.
func fieldLines(t *testing.T, code, name string) []string {
	t.Helper()
	var lines []string
	for _, line := range strings.Split(structBody(t, code, name), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func TestGenerator_ElementOrder_MarshalledSequenceMatchesSchema(t *testing.T) {
	out := generate(t, writeWSDL(t, elementOrderWSDL), WithPackage("order"))
	test := strings.Replace(elementOrderMarshalTest, "SCHEMA_ORDER", strings.Join(elementOrderSchemaSequence, ","), 1)
	runGeneratedModuleTests(t, "order", out, map[string]string{"order_test.go": test})
}

func TestGenerator_ElementOrder_ServerAndGenericsBuild(t *testing.T) {
	assertServerAndGenericsBuild(t, writeWSDL(t, elementOrderWSDL), "order")
}
