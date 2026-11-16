// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	xsdparser "github.com/kastrul/gowsdl/pkg/parser"
)

// boundedRepeatWSDL declares elements whose maxOccurs is a number greater
// than 1, of every kind of type an element field can have, next to elements
// that occur at most once (maxOccurs="1") or never (maxOccurs="0").
const boundedRepeatWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="urn:bounded"
             xmlns:xs="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:bounded">
  <types>
    <xs:schema targetNamespace="urn:bounded" elementFormDefault="qualified">
      <xs:complexType name="Item">
        <xs:sequence><xs:element name="sku" type="xs:string"/></xs:sequence>
      </xs:complexType>
      <xs:complexType name="Tree">
        <xs:sequence>
          <xs:element name="label" type="xs:string"/>
          <xs:element name="child" type="tns:Tree" minOccurs="0" maxOccurs="2"/>
        </xs:sequence>
      </xs:complexType>
      <xs:element name="tag" type="xs:string"/>
      <xs:element name="Order">
        <xs:complexType>
          <xs:sequence>
            <xs:element name="note" type="xs:string" maxOccurs="3"/>
            <xs:element name="item" type="tns:Item" maxOccurs="5"/>
            <xs:element name="optItem" type="tns:Item" minOccurs="0" maxOccurs="2"/>
            <xs:element ref="tns:tag" maxOccurs="2"/>
            <xs:element name="line" maxOccurs="4">
              <xs:complexType><xs:sequence><xs:element name="qty" type="xs:int"/></xs:sequence></xs:complexType>
            </xs:element>
            <xs:element name="score" maxOccurs="2">
              <xs:simpleType><xs:restriction base="xs:int"><xs:maxInclusive value="10"/></xs:restriction></xs:simpleType>
            </xs:element>
            <xs:element name="extra" maxOccurs="3"/>
            <xs:element name="tree" type="tns:Tree"/>
            <xs:choice>
              <xs:element name="gift" type="tns:Item" maxOccurs="2"/>
              <xs:element name="none" type="xs:boolean"/>
            </xs:choice>
            <xs:element name="once" type="xs:string" maxOccurs="1"/>
            <xs:element name="never" type="xs:string" minOccurs="0" maxOccurs="0"/>
          </xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:element name="OrderResponse">
        <xs:complexType>
          <xs:sequence>
            <xs:element name="code" type="xs:int" minOccurs="1" maxOccurs="3"/>
            <xs:element name="item" type="tns:Item" minOccurs="0" maxOccurs="4"/>
          </xs:sequence>
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
      <soap:operation soapAction="urn:bounded:PlaceOrder"/>
      <input><soap:body use="literal"/></input>
      <output><soap:body use="literal"/></output>
    </operation>
  </binding>
  <service name="OrderService">
    <port name="OrderPort" binding="tns:OrderBinding"><soap:address location="http://localhost/"/></port>
  </service>
</definitions>`

func TestGenerator_BoundedRepeat_IsSlice(t *testing.T) {
	out := generate(t, writeWSDL(t, boundedRepeatWSDL), WithPackage("bounded"))
	code := out.client(t)

	assert.Equal(t, []string{
		"XMLName xml.Name `xml:\"urn:bounded Order\"`",
		"Note []string `xml:\"note,omitempty\" json:\"note,omitempty\"`",
		"Item []Item `xml:\"item,omitempty\" json:\"item,omitempty\"`",
		"OptItem []Item `xml:\"optItem,omitempty\" json:\"optItem,omitempty\"`",
		"Tag []Tag `xml:\"tag,omitempty\" json:\"tag,omitempty\"`",
		"Line []OrderLineType `xml:\"line,omitempty\" json:\"line,omitempty\"`",
		"Score []int32 `xml:\"score,omitempty\" json:\"score,omitempty\"`",
		"Extra []soap.AnyType `xml:\"extra,omitempty\" json:\"extra,omitempty\"`",
		"Tree Tree `xml:\"tree,omitempty\" json:\"tree,omitzero\"`",
		"Gift []Item `xml:\"gift,omitempty\" json:\"gift,omitempty\"`",
		"None bool `xml:\"none,omitempty\" json:\"none,omitempty\"`",
		"Once string `xml:\"once\" json:\"once,omitempty\"`",
		"Never string `xml:\"never,omitempty\" json:\"never,omitempty\"`",
	}, fieldLines(t, code, "Order"))

	assert.Equal(t, []string{
		"Label string `xml:\"label\" json:\"label,omitempty\"`",
		"Child []Tree `xml:\"child,omitempty\" json:\"child,omitempty\"`",
	}, fieldLines(t, code, "Tree"), "a slice already breaks the cycle, so no pointer is added")

	assert.Equal(t, []string{
		"XMLName xml.Name `xml:\"urn:bounded OrderResponse\"`",
		"Code []int32 `xml:\"code,omitempty\" json:\"code,omitempty\"`",
		"Item []Item `xml:\"item,omitempty\" json:\"item,omitempty\"`",
	}, fieldLines(t, code, "OrderResponse"))
}

func TestIsOptionalType_BoundedRepeatIsNotOptional(t *testing.T) {
	assert.False(t, IsOptionalType("0", "3"), "a repeated element is a slice, not an optional pointer")
	assert.True(t, IsOptionalType("0", "1"))
	assert.True(t, IsOptionalType("0", ""))
}

func TestElementRepeated(t *testing.T) {
	for maxOccurs, want := range map[string]bool{
		"": false, "1": false, " 1 ": false, "0": false,
		"2": true, "3": true, " 10 ": true, "unbounded": true,
	} {
		assert.Equal(t, want, elementRepeated(&xsdparser.XSDElement{MaxOccurs: maxOccurs}), "maxOccurs=%q", maxOccurs)
	}
}

// boundedRepeatClientTest sends several occurrences of each bounded element
// through the generated client and decodes several from the response.
const boundedRepeatClientTest = `package bounded

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kastrul/gowsdl/soap"
)

func TestSendsAndReceivesSeveral(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = io.WriteString(w, ` + "`" + `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><OrderResponse xmlns="urn:bounded"><code>1</code><code>2</code><code>3</code><item><sku>x</sku></item><item><sku>y</sku></item></OrderResponse></soap:Body></soap:Envelope>` + "`" + `)
	}))
	defer server.Close()

	req := &Order{
		Note:  []string{"n1", "n2", "n3"},
		Item:  []Item{{Sku: "a"}, {Sku: "b"}},
		Tag:   []Tag{"t1", "t2"},
		Line:  []OrderLineType{{Qty: 1}, {Qty: 2}},
		Score: []int32{4, 5},
		Tree:  Tree{Label: "root", Child: []Tree{{Label: "c1"}, {Label: "c2"}}},
		Gift:  []Item{{Sku: "g1"}, {Sku: "g2"}},
		Once:  "o",
	}
	resp, err := NewOrderPort(soap.NewClient(server.URL)).PlaceOrder(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"<note>n1</note><note>n2</note><note>n3</note>",
		"<item><sku>a</sku></item><item><sku>b</sku></item>",
		"<tag>t1</tag><tag>t2</tag>",
		"<line><qty>1</qty></line><line><qty>2</qty></line>",
		"<score>4</score><score>5</score>",
		"<child><label>c1</label></child><child><label>c2</label></child>",
		"<gift><sku>g1</sku></gift><gift><sku>g2</sku></gift>",
		"<once>o</once>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("request lacks %s:\n%s", want, body)
		}
	}
	for _, absent := range []string{"<optItem>", "<extra>", "<never>", "<none>"} {
		if strings.Contains(body, absent) {
			t.Errorf("request has unset %s:\n%s", absent, body)
		}
	}

	if len(resp.Code) != 3 || resp.Code[0] != 1 || resp.Code[2] != 3 {
		t.Errorf("decoded codes %v, want [1 2 3]", resp.Code)
	}
	if len(resp.Item) != 2 || resp.Item[0].Sku != "x" || resp.Item[1].Sku != "y" {
		t.Errorf("decoded items %+v, want x and y", resp.Item)
	}
}
`

func TestGenerator_BoundedRepeat_SendsAndReceivesSeveral(t *testing.T) {
	out := generate(t, writeWSDL(t, boundedRepeatWSDL), WithPackage("bounded"))
	runGeneratedModuleTests(t, "bounded", out, map[string]string{"bounded_test.go": boundedRepeatClientTest})
}

func TestGenerator_BoundedRepeat_ServerAndGenericsBuild(t *testing.T) {
	assertServerAndGenericsBuild(t, writeWSDL(t, boundedRepeatWSDL), "bounded")
}
