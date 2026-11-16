// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// requiredElementsWSDL declares elements and attributes that are required or
// optional by XML Schema's rules: an element is required when its own
// minOccurs and that of every model group enclosing it is at least 1, and it
// is not a member of a choice; an attribute is required when use="required".
const requiredElementsWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="urn:required"
             xmlns:xs="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:required">
  <types>
    <xs:schema targetNamespace="urn:required" elementFormDefault="qualified">
      <xs:simpleType name="Level">
        <xs:restriction base="xs:int"><xs:enumeration value="1"/><xs:enumeration value="2"/></xs:restriction>
      </xs:simpleType>
      <xs:simpleType name="Percent">
        <xs:restriction base="xs:decimal"><xs:maxInclusive value="100"/></xs:restriction>
      </xs:simpleType>
      <xs:simpleType name="Color">
        <xs:restriction base="xs:string"><xs:enumeration value="red"/><xs:enumeration value="blue"/></xs:restriction>
      </xs:simpleType>
      <xs:simpleType name="Codes"><xs:list itemType="xs:int"/></xs:simpleType>
      <xs:complexType name="Address">
        <xs:sequence><xs:element name="city" type="xs:string"/></xs:sequence>
      </xs:complexType>
      <xs:element name="count" type="xs:int"/>
      <xs:element name="Order">
        <xs:complexType>
          <xs:sequence>
            <xs:element name="giftWrap" type="xs:boolean"/>
            <xs:element name="express" type="xs:boolean" minOccurs="0"/>
            <xs:element name="quantity" type="xs:int"/>
            <xs:element name="discount" type="xs:int" minOccurs="0"/>
            <xs:element name="total" type="xs:long" minOccurs="1"/>
            <xs:element name="weight" type="xs:double"/>
            <xs:element name="price" type="xs:decimal"/>
            <xs:element name="tax" type="xs:decimal" minOccurs="0"/>
            <xs:element name="level" type="tns:Level"/>
            <xs:element name="percent" type="tns:Percent"/>
            <xs:element name="bounded">
              <xs:simpleType><xs:restriction base="xs:unsignedShort"><xs:maxInclusive value="10"/></xs:restriction></xs:simpleType>
            </xs:element>
            <xs:element ref="tns:count"/>
            <xs:element name="color" type="tns:Color"/>
            <xs:element name="codes" type="tns:Codes"/>
            <xs:element name="address" type="tns:Address"/>
            <xs:element name="maybeFlag" type="xs:boolean" nillable="true"/>
            <xs:element name="flags" type="xs:boolean" maxOccurs="unbounded"/>
            <xs:element name="pair" type="xs:int" maxOccurs="2"/>
            <xs:sequence>
              <xs:element name="inRequiredSequence" type="xs:boolean"/>
            </xs:sequence>
            <xs:sequence minOccurs="0">
              <xs:element name="inOptionalSequence" type="xs:boolean"/>
              <xs:sequence>
                <xs:element name="deepInOptionalSequence" type="xs:int"/>
              </xs:sequence>
            </xs:sequence>
            <xs:choice>
              <xs:element name="byCard" type="xs:boolean"/>
              <xs:element name="installments" type="xs:int" minOccurs="1"/>
              <xs:sequence>
                <xs:element name="inChoiceSequence" type="xs:int"/>
              </xs:sequence>
            </xs:choice>
          </xs:sequence>
          <xs:attribute name="priority" type="xs:int" use="required"/>
          <xs:attribute name="urgent" type="xs:boolean" use="required"/>
          <xs:attribute name="tracked" type="xs:boolean"/>
          <xs:attribute name="retries" type="xs:int" use="optional"/>
        </xs:complexType>
      </xs:element>
      <xs:element name="OrderResponse">
        <xs:complexType>
          <xs:sequence><xs:element name="accepted" type="xs:boolean"/></xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:complexType name="OptionalContent">
        <xs:sequence minOccurs="0">
          <xs:element name="flag" type="xs:boolean"/>
        </xs:sequence>
      </xs:complexType>
      <xs:complexType name="Payment">
        <xs:choice>
          <xs:element name="cash" type="xs:boolean"/>
          <xs:element name="amount" type="xs:int"/>
        </xs:choice>
      </xs:complexType>
      <xs:complexType name="Unordered">
        <xs:all>
          <xs:element name="first" type="xs:int"/>
          <xs:element name="second" type="xs:int" minOccurs="0"/>
        </xs:all>
      </xs:complexType>
      <xs:complexType name="OptionalUnordered">
        <xs:all minOccurs="0">
          <xs:element name="only" type="xs:int"/>
        </xs:all>
      </xs:complexType>
      <xs:complexType name="Derived">
        <xs:complexContent>
          <xs:extension base="tns:Unordered">
            <xs:sequence>
              <xs:element name="extra" type="xs:boolean"/>
              <xs:element name="extraOptional" type="xs:boolean" minOccurs="0"/>
            </xs:sequence>
            <xs:attribute name="version" type="xs:int" use="required"/>
          </xs:extension>
        </xs:complexContent>
      </xs:complexType>
      <xs:complexType name="Restricted">
        <xs:complexContent>
          <xs:restriction base="xs:anyType">
            <xs:sequence>
              <xs:element name="kept" type="xs:boolean"/>
              <xs:sequence minOccurs="0">
                <xs:element name="dropped" type="xs:boolean"/>
              </xs:sequence>
            </xs:sequence>
          </xs:restriction>
        </xs:complexContent>
      </xs:complexType>
      <xs:complexType name="Outer">
        <xs:sequence minOccurs="0">
          <xs:element name="inner">
            <xs:complexType>
              <xs:sequence><xs:element name="innerFlag" type="xs:boolean"/></xs:sequence>
            </xs:complexType>
          </xs:element>
        </xs:sequence>
      </xs:complexType>
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
      <soap:operation soapAction="urn:required:PlaceOrder"/>
      <input><soap:body use="literal"/></input>
      <output><soap:body use="literal"/></output>
    </operation>
  </binding>
  <service name="OrderService">
    <port name="OrderPort" binding="tns:OrderBinding"><soap:address location="http://localhost/"/></port>
  </service>
</definitions>`

// fieldTags returns the formatted struct tag of each line of a structBody
// result, keyed by the field name.
func fieldTags(t *testing.T, code, name string) map[string]string {
	t.Helper()
	tags := map[string]string{}
	for _, line := range strings.Split(structBody(t, code, name), "\n") {
		field, _, _ := strings.Cut(strings.TrimSpace(line), " ")
		if i := strings.Index(line, "`"); i >= 0 && field != "" {
			tags[field] = line[i:]
		}
	}
	return tags
}

func TestGenerator_RequiredBoolAndNumericElements_Tags(t *testing.T) {
	code := generate(t, writeWSDL(t, requiredElementsWSDL), WithPackage("required")).client(t)

	order := fieldTags(t, code, "Order")
	for field, tag := range map[string]string{
		"GiftWrap":               "`xml:\"giftWrap\" json:\"giftWrap,omitempty\"`",
		"Express":                "`xml:\"express,omitempty\" json:\"express,omitempty\"`",
		"Quantity":               "`xml:\"quantity\" json:\"quantity,omitempty\"`",
		"Discount":               "`xml:\"discount,omitempty\" json:\"discount,omitempty\"`",
		"Total":                  "`xml:\"total\" json:\"total,omitempty\"`",
		"Weight":                 "`xml:\"weight\" json:\"weight,omitempty\"`",
		"Price":                  "`xml:\"price\" json:\"price,omitempty\"`",
		"Tax":                    "`xml:\"tax,omitempty\" json:\"tax,omitempty\"`",
		"Level":                  "`xml:\"level\" json:\"level,omitempty\"`",
		"Percent":                "`xml:\"percent\" json:\"percent,omitempty\"`",
		"Bounded":                "`xml:\"bounded\" json:\"bounded,omitempty\"`",
		"Count":                  "`xml:\"count\" json:\"count,omitempty\"`",
		"Codes":                  "`xml:\"codes,omitempty\" json:\"codes,omitempty\"`",
		"Address":                "`xml:\"address,omitempty\" json:\"address,omitzero\"`",
		"MaybeFlag":              "`xml:\"maybeFlag,omitempty\" json:\"maybeFlag,omitempty\"`",
		"Flags":                  "`xml:\"flags,omitempty\" json:\"flags,omitempty\"`",
		"Pair":                   "`xml:\"pair,omitempty\" json:\"pair,omitempty\"`",
		"InRequiredSequence":     "`xml:\"inRequiredSequence\" json:\"inRequiredSequence,omitempty\"`",
		"InOptionalSequence":     "`xml:\"inOptionalSequence,omitempty\" json:\"inOptionalSequence,omitempty\"`",
		"DeepInOptionalSequence": "`xml:\"deepInOptionalSequence,omitempty\" json:\"deepInOptionalSequence,omitempty\"`",
		"ByCard":                 "`xml:\"byCard,omitempty\" json:\"byCard,omitempty\"`",
		"Installments":           "`xml:\"installments,omitempty\" json:\"installments,omitempty\"`",
		"InChoiceSequence":       "`xml:\"inChoiceSequence,omitempty\" json:\"inChoiceSequence,omitempty\"`",
		"Priority":               "`xml:\"priority,attr\" json:\"priority,omitempty\"`",
		"Urgent":                 "`xml:\"urgent,attr\" json:\"urgent,omitempty\"`",
		"Tracked":                "`xml:\"tracked,attr,omitempty\" json:\"tracked,omitempty\"`",
		"Retries":                "`xml:\"retries,attr,omitempty\" json:\"retries,omitempty\"`",
	} {
		assert.Equal(t, tag, order[field], "Order.%s", field)
	}

	for typ, want := range map[string]map[string]string{
		"OptionalContent":   {"Flag": "`xml:\"flag,omitempty\" json:\"flag,omitempty\"`"},
		"Payment":           {"Cash": "`xml:\"cash,omitempty\" json:\"cash,omitempty\"`", "Amount": "`xml:\"amount,omitempty\" json:\"amount,omitempty\"`"},
		"Unordered":         {"First": "`xml:\"first\" json:\"first,omitempty\"`", "Second": "`xml:\"second,omitempty\" json:\"second,omitempty\"`"},
		"OptionalUnordered": {"Only": "`xml:\"only,omitempty\" json:\"only,omitempty\"`"},
		"Derived": {
			"Extra":         "`xml:\"extra\" json:\"extra,omitempty\"`",
			"ExtraOptional": "`xml:\"extraOptional,omitempty\" json:\"extraOptional,omitempty\"`",
			"Version":       "`xml:\"version,attr\" json:\"version,omitempty\"`",
		},
		"Restricted":     {"Kept": "`xml:\"kept\" json:\"kept,omitempty\"`", "Dropped": "`xml:\"dropped,omitempty\" json:\"dropped,omitempty\"`"},
		"OuterInnerType": {"InnerFlag": "`xml:\"innerFlag\" json:\"innerFlag,omitempty\"`"},
	} {
		tags := fieldTags(t, code, typ)
		for field, tag := range want {
			assert.Equal(t, tag, tags[field], "%s.%s", typ, field)
		}
	}
}

// requiredElementsMarshalTest marshals and unmarshals the generated Order
// request: required booleans and numbers holding their zero value are sent,
// optional ones left unset are not, and a choice sends only the branch set.
const requiredElementsMarshalTest = `package required

import (
	"encoding/xml"
	"strings"
	"testing"
)

func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := xml.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRequiredZeroValuesAreSent(t *testing.T) {
	got := marshal(t, Order{Installments: 3})
	for _, want := range []string{
		"<giftWrap>false</giftWrap>",
		"<quantity>0</quantity>",
		"<total>0</total>",
		"<weight>0</weight>",
		"<price>0</price>",
		"<level>0</level>",
		"<percent>0</percent>",
		"<bounded>0</bounded>",
		"<count>0</count>",
		"<inRequiredSequence>false</inRequiredSequence>",
		"<installments>3</installments>",
		` + "`" + `priority="0"` + "`" + `,
		` + "`" + `urgent="false"` + "`" + `,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("request lacks %s:\n%s", want, got)
		}
	}
	for _, absent := range []string{
		"<express>", "<discount>", "<tax>", "<maybeFlag>", "<flags>", "<pair>",
		"<inOptionalSequence>", "<deepInOptionalSequence>",
		"<byCard>", "<inChoiceSequence>",
		"tracked=", "retries=",
	} {
		if strings.Contains(got, absent) {
			t.Errorf("request has optional or unchosen %s:\n%s", absent, got)
		}
	}
}

func TestChoiceSendsOnlyTheBranchSet(t *testing.T) {
	got := marshal(t, Payment{Amount: 5})
	if want := "<Payment><amount>5</amount></Payment>"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	got = marshal(t, Payment{Cash: true})
	if want := "<Payment><cash>true</cash></Payment>"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestOptionalAndAllGroups(t *testing.T) {
	if got, want := marshal(t, OptionalContent{}), "<OptionalContent></OptionalContent>"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	if got, want := marshal(t, Unordered{}), "<Unordered><first>0</first></Unordered>"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	if got, want := marshal(t, Derived{}), ` + "`" + `<Derived version="0"><first>0</first><extra>false</extra></Derived>` + "`" + `; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestResponsesDecodeAsBefore(t *testing.T) {
	var full Order
	in := ` + "`" + `<Order xmlns="urn:required" priority="2" urgent="true" tracked="true"><giftWrap>true</giftWrap><express>false</express><quantity>4</quantity><price>1.5</price><byCard>true</byCard></Order>` + "`" + `
	if err := xml.Unmarshal([]byte(in), &full); err != nil {
		t.Fatal(err)
	}
	if !full.GiftWrap || full.Express || full.Quantity != 4 || full.Price != 1.5 || !full.ByCard || full.Priority != 2 || !full.Urgent || !full.Tracked {
		t.Errorf("decoded %+v", full)
	}

	var sparse Order
	if err := xml.Unmarshal([]byte(` + "`" + `<Order xmlns="urn:required"></Order>` + "`" + `), &sparse); err != nil {
		t.Fatal(err)
	}
	if sparse.GiftWrap || sparse.Quantity != 0 || sparse.Priority != 0 {
		t.Errorf("decoded %+v from an element without content", sparse)
	}
}
`

func TestGenerator_RequiredBoolAndNumericElements_Marshal(t *testing.T) {
	out := generate(t, writeWSDL(t, requiredElementsWSDL), WithPackage("required"))
	runGeneratedModuleTests(t, "required", out, map[string]string{"required_test.go": requiredElementsMarshalTest})
}
