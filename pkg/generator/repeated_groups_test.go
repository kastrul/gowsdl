// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// repeatedGroupsWSDL declares elements that occur at most once themselves
// but sit in a model group that repeats: a sequence, a choice, a nested
// sequence and a group reference with maxOccurs above 1, a repeated sequence
// inside a group definition, a repeated choice in an extension, and a
// repeated sequence closing a recursive cycle, next to a sequence with
// maxOccurs="1".
const repeatedGroupsWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="urn:repgroups"
             xmlns:xs="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:repgroups">
  <types>
    <xs:schema targetNamespace="urn:repgroups" elementFormDefault="qualified">
      <xs:complexType name="Item">
        <xs:sequence><xs:element name="sku" type="xs:string"/></xs:sequence>
      </xs:complexType>
      <xs:complexType name="Node">
        <xs:sequence>
          <xs:element name="label" type="xs:string"/>
          <xs:sequence maxOccurs="unbounded">
            <xs:element name="child" type="tns:Node"/>
          </xs:sequence>
        </xs:sequence>
      </xs:complexType>
      <xs:group name="Line">
        <xs:sequence>
          <xs:element name="lineSku" type="xs:string"/>
          <xs:element name="lineQty" type="xs:int"/>
        </xs:sequence>
      </xs:group>
      <xs:group name="Tags">
        <xs:sequence>
          <xs:element name="tagged" type="xs:boolean"/>
          <xs:sequence maxOccurs="unbounded"><xs:element name="tag" type="xs:string"/></xs:sequence>
        </xs:sequence>
      </xs:group>
      <xs:complexType name="Base">
        <xs:sequence><xs:element name="baseId" type="xs:int"/></xs:sequence>
      </xs:complexType>
      <xs:complexType name="Derived">
        <xs:complexContent>
          <xs:extension base="tns:Base">
            <xs:sequence>
              <xs:choice maxOccurs="unbounded">
                <xs:element name="extA" type="xs:string"/>
                <xs:element name="extB" type="tns:Item"/>
              </xs:choice>
            </xs:sequence>
          </xs:extension>
        </xs:complexContent>
      </xs:complexType>
      <xs:element name="Order">
        <xs:complexType>
          <xs:sequence>
            <xs:element name="id" type="xs:int"/>
            <xs:sequence maxOccurs="unbounded">
              <xs:element name="name" type="xs:string"/>
              <xs:element name="qty" type="xs:int"/>
            </xs:sequence>
            <xs:choice maxOccurs="3">
              <xs:element name="note" type="xs:string"/>
              <xs:element name="gift" type="tns:Item"/>
              <xs:element name="part">
                <xs:complexType><xs:sequence><xs:element name="code" type="xs:string"/></xs:sequence></xs:complexType>
              </xs:element>
            </xs:choice>
            <xs:sequence maxOccurs="2">
              <xs:sequence><xs:element name="deep" type="xs:string"/></xs:sequence>
            </xs:sequence>
            <xs:group ref="tns:Line" maxOccurs="unbounded"/>
            <xs:group ref="tns:Tags"/>
            <xs:sequence maxOccurs="1"><xs:element name="single" type="xs:string"/></xs:sequence>
            <xs:element name="tree" type="tns:Node" minOccurs="0"/>
            <xs:element name="derived" type="tns:Derived" minOccurs="0"/>
            <xs:element name="last" type="xs:string"/>
          </xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:element name="OrderResponse">
        <xs:complexType>
          <xs:choice minOccurs="0" maxOccurs="unbounded">
            <xs:element name="event" type="tns:Item"/>
            <xs:element name="message" type="xs:string"/>
          </xs:choice>
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
      <soap:operation soapAction="urn:repgroups:PlaceOrder"/>
      <input><soap:body use="literal"/></input>
      <output><soap:body use="literal"/></output>
    </operation>
  </binding>
  <service name="OrderService">
    <port name="OrderPort" binding="tns:OrderBinding"><soap:address location="http://localhost/"/></port>
  </service>
</definitions>`

func TestGenerator_RepeatedGroups_MembersAreSlices(t *testing.T) {
	code := generate(t, writeWSDL(t, repeatedGroupsWSDL), WithPackage("repgroups")).client(t)

	assert.Equal(t, []string{
		"XMLName xml.Name `xml:\"urn:repgroups Order\"`",
		"Id int32 `xml:\"id\" json:\"id,omitempty\"`",
		"NameGroup []OrderNameGroup `xml:\"name,omitempty\" json:\"nameGroup,omitempty\"`",
		"Note []string `xml:\"note,omitempty\" json:\"note,omitempty\"`",
		"Gift []Item `xml:\"gift,omitempty\" json:\"gift,omitempty\"`",
		"Part []OrderPartType `xml:\"part,omitempty\" json:\"part,omitempty\"`",
		"Deep []string `xml:\"deep,omitempty\" json:\"deep,omitempty\"`",
		"LineGroup []OrderLineGroup `xml:\"lineSku,omitempty\" json:\"LineGroup,omitempty\"`",
		"Tagged bool `xml:\"tagged\" json:\"tagged,omitempty\"`",
		"Tag []string `xml:\"tag,omitempty\" json:\"tag,omitempty\"`",
		"Single string `xml:\"single\" json:\"single,omitempty\"`",
		"Tree *Node `xml:\"tree,omitempty\" json:\"tree,omitempty\"`",
		"Derived *Derived `xml:\"derived,omitempty\" json:\"derived,omitempty\"`",
		"Last string `xml:\"last\" json:\"last,omitempty\"`",
	}, fieldLines(t, code, "Order"))

	assert.Equal(t, []string{
		"Label string `xml:\"label\" json:\"label,omitempty\"`",
		"Child []Node `xml:\"child,omitempty\" json:\"child,omitempty\"`",
	}, fieldLines(t, code, "Node"), "a slice already breaks the cycle, so no pointer is added")

	assert.Equal(t, []string{
		"Base",
		"ExtA []string `xml:\"extA,omitempty\" json:\"extA,omitempty\"`",
		"ExtB []Item `xml:\"extB,omitempty\" json:\"extB,omitempty\"`",
	}, fieldLines(t, code, "Derived"))

	assert.Equal(t, []string{
		"XMLName xml.Name `xml:\"urn:repgroups OrderResponse\"`",
		"Event []Item `xml:\"event,omitempty\" json:\"event,omitempty\"`",
		"Message []string `xml:\"message,omitempty\" json:\"message,omitempty\"`",
	}, fieldLines(t, code, "OrderResponse"))
}

// repeatedGroupsClientTest sends several members of repeated groups through
// the generated client and decodes several, interleaved, from the response.
const repeatedGroupsClientTest = `package repgroups

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
		_, _ = io.WriteString(w, ` + "`" + `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><OrderResponse xmlns="urn:repgroups"><event><sku>a</sku></event><message>m1</message><event><sku>b</sku></event><message>m2</message></OrderResponse></soap:Body></soap:Envelope>` + "`" + `)
	}))
	defer server.Close()

	req := &Order{
		Id:   1,
		NameGroup: []OrderNameGroup{{Name: "n1", Qty: 2}},
		Note: []string{"x", "y"}, Gift: []Item{{Sku: "g"}},
		Deep:    []string{"d1", "d2"},
		LineGroup: []OrderLineGroup{{LineSku: "s1", LineQty: 3}},
		Tag:     []string{"t1", "t2"},
		Tree:    &Node{Label: "root", Child: []Node{{Label: "c1"}, {Label: "c2"}}},
		Derived: &Derived{ExtA: []string{"e1", "e2"}, ExtB: []Item{{Sku: "eb"}}},
		Last:    "z",
	}
	resp, err := NewOrderPort(soap.NewClient(server.URL)).PlaceOrder(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"<id>1</id><name>n1</name><qty>2</qty><note>x</note><note>y</note><gift><sku>g</sku></gift>",
		"<deep>d1</deep><deep>d2</deep><lineSku>s1</lineSku><lineQty>3</lineQty>",
		"<tagged>false</tagged><tag>t1</tag><tag>t2</tag><single></single>",
		"<child><label>c1</label></child><child><label>c2</label></child>",
		"<extA>e1</extA><extA>e2</extA><extB><sku>eb</sku></extB>",
		"<last>z</last>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("request lacks %s:\n%s", want, body)
		}
	}
	if strings.Contains(body, "<part>") {
		t.Errorf("request has unset part:\n%s", body)
	}

	if len(resp.Event) != 2 || resp.Event[0].Sku != "a" || resp.Event[1].Sku != "b" {
		t.Errorf("decoded events %+v, want a and b", resp.Event)
	}
	if len(resp.Message) != 2 || resp.Message[0] != "m1" || resp.Message[1] != "m2" {
		t.Errorf("decoded messages %v, want [m1 m2]", resp.Message)
	}
}
`

func TestGenerator_RepeatedGroups_SendsAndReceivesSeveral(t *testing.T) {
	out := generate(t, writeWSDL(t, repeatedGroupsWSDL), WithPackage("repgroups"))
	runGeneratedModuleTests(t, "repgroups", out, map[string]string{"repgroups_test.go": repeatedGroupsClientTest})
}

func TestGenerator_RepeatedGroups_ServerAndGenericsBuild(t *testing.T) {
	assertServerAndGenericsBuild(t, writeWSDL(t, repeatedGroupsWSDL), "repgroups")
}
