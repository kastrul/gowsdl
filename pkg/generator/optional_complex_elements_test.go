// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// optionalComplexWSDL declares elements of complex type in every position
// that decides whether they may be left out of an instance: required, own
// minOccurs="0", inside an optional sequence or all, a member of a choice,
// nillable, repeated, referring to a global element, of a simpleContent type,
// part of a recursive cycle, and of an anonymous type.
const optionalComplexWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="urn:optcomplex"
             xmlns:xs="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:optcomplex">
  <types>
    <xs:schema targetNamespace="urn:optcomplex" elementFormDefault="qualified">
      <xs:complexType name="Address">
        <xs:sequence><xs:element name="city" type="xs:string"/></xs:sequence>
      </xs:complexType>
      <xs:complexType name="Money">
        <xs:simpleContent>
          <xs:extension base="xs:decimal"><xs:attribute name="currency" type="xs:string" use="required"/></xs:extension>
        </xs:simpleContent>
      </xs:complexType>
      <xs:complexType name="Node">
        <xs:sequence>
          <xs:element name="label" type="xs:string"/>
          <xs:element name="child" type="tns:Node" minOccurs="0"/>
          <xs:element name="partner" type="tns:Partner" minOccurs="0"/>
        </xs:sequence>
      </xs:complexType>
      <xs:complexType name="Partner">
        <xs:sequence><xs:element name="back" type="tns:Node" minOccurs="0"/></xs:sequence>
      </xs:complexType>
      <xs:complexType name="Ring">
        <xs:sequence><xs:element name="next" type="tns:Ring"/></xs:sequence>
      </xs:complexType>
      <xs:complexType name="Unordered">
        <xs:all minOccurs="0"><xs:element name="allAddr" type="tns:Address"/></xs:all>
      </xs:complexType>
      <xs:complexType name="Payment">
        <xs:choice>
          <xs:element name="cardAddr" type="tns:Address"/>
          <xs:element name="bankAddr" type="tns:Address"/>
          <xs:element name="voucher">
            <xs:complexType><xs:sequence><xs:element name="code" type="xs:string"/></xs:sequence></xs:complexType>
          </xs:element>
        </xs:choice>
      </xs:complexType>
      <xs:element name="homeAddr" type="tns:Address"/>
      <xs:element name="workAddr" type="tns:Address"/>
      <xs:element name="Probe">
        <xs:complexType>
          <xs:sequence>
            <xs:element name="reqAddr" type="tns:Address"/>
            <xs:element name="optAddr" type="tns:Address" minOccurs="0"/>
            <xs:element name="nilAddr" type="tns:Address" nillable="true"/>
            <xs:element name="optNilAddr" type="tns:Address" nillable="true" minOccurs="0"/>
            <xs:element name="manyAddr" type="tns:Address" minOccurs="0" maxOccurs="unbounded"/>
            <xs:element ref="tns:homeAddr"/>
            <xs:element ref="tns:workAddr" minOccurs="0"/>
            <xs:element name="reqMoney" type="tns:Money"/>
            <xs:element name="optMoney" type="tns:Money" minOccurs="0"/>
            <xs:element name="optNode" type="tns:Node" minOccurs="0"/>
            <xs:element name="reqInline">
              <xs:complexType><xs:sequence><xs:element name="code" type="xs:string"/></xs:sequence></xs:complexType>
            </xs:element>
            <xs:element name="optInline" minOccurs="0">
              <xs:complexType><xs:sequence><xs:element name="code" type="xs:string"/></xs:sequence></xs:complexType>
            </xs:element>
            <xs:element name="optAny" type="xs:anyType" minOccurs="0"/>
            <xs:element name="unordered" type="tns:Unordered"/>
            <xs:sequence minOccurs="0">
              <xs:element name="seqAddr" type="tns:Address"/>
            </xs:sequence>
          </xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:element name="ProbeResponse">
        <xs:complexType><xs:sequence><xs:element name="ok" type="xs:boolean"/></xs:sequence></xs:complexType>
      </xs:element>
    </xs:schema>
  </types>
  <message name="ProbeIn"><part name="parameters" element="tns:Probe"/></message>
  <message name="ProbeOut"><part name="parameters" element="tns:ProbeResponse"/></message>
  <portType name="ProbePort">
    <operation name="Probe"><input message="tns:ProbeIn"/><output message="tns:ProbeOut"/></operation>
  </portType>
  <binding name="ProbeBinding" type="tns:ProbePort">
    <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
    <operation name="Probe">
      <soap:operation soapAction="urn:optcomplex:Probe"/>
      <input><soap:body use="literal"/></input>
      <output><soap:body use="literal"/></output>
    </operation>
  </binding>
  <service name="ProbeService">
    <port name="ProbePort" binding="tns:ProbeBinding"><soap:address location="http://localhost/"/></port>
  </service>
</definitions>`

// fieldDecls returns each field line of a structBody result, without its
// leading whitespace, keyed by the field name.
func fieldDecls(t *testing.T, code, name string) map[string]string {
	t.Helper()
	decls := map[string]string{}
	for _, line := range strings.Split(structBody(t, code, name), "\n") {
		line = strings.TrimSpace(line)
		if field, _, ok := strings.Cut(line, " "); ok {
			decls[field] = line
		}
	}
	return decls
}

func TestGenerator_OptionalComplexElements_Declarations(t *testing.T) {
	code := generate(t, writeWSDL(t, optionalComplexWSDL), WithPackage("optcomplex")).client(t)

	for typ, want := range map[string]map[string]string{
		"Probe": {
			"ReqAddr":    "ReqAddr Address `xml:\"reqAddr,omitempty\" json:\"reqAddr,omitzero\"`",
			"OptAddr":    "OptAddr *Address `xml:\"optAddr,omitempty\" json:\"optAddr,omitempty\"`",
			"NilAddr":    "NilAddr *Address `xml:\"nilAddr,omitempty\" json:\"nilAddr,omitempty\"`",
			"OptNilAddr": "OptNilAddr *Address `xml:\"optNilAddr,omitempty\" json:\"optNilAddr,omitempty\"`",
			"ManyAddr":   "ManyAddr []Address `xml:\"manyAddr,omitempty\" json:\"manyAddr,omitempty\"`",
			"HomeAddr":   "HomeAddr HomeAddr `xml:\"homeAddr,omitempty\" json:\"homeAddr,omitzero\"`",
			"WorkAddr":   "WorkAddr *WorkAddr `xml:\"workAddr,omitempty\" json:\"workAddr,omitempty\"`",
			"ReqMoney":   "ReqMoney Money `xml:\"reqMoney,omitempty\" json:\"reqMoney,omitzero\"`",
			"OptMoney":   "OptMoney *Money `xml:\"optMoney,omitempty\" json:\"optMoney,omitempty\"`",
			"OptNode":    "OptNode *Node `xml:\"optNode,omitempty\" json:\"optNode,omitempty\"`",
			"ReqInline":  "ReqInline ProbeReqInlineType `xml:\"reqInline,omitempty\" json:\"reqInline,omitzero\"`",
			"OptInline":  "OptInline *ProbeOptInlineType `xml:\"optInline,omitempty\" json:\"optInline,omitempty\"`",
			"OptAny":     "OptAny soap.AnyType `xml:\"optAny,omitempty\" json:\"optAny,omitzero\"`",
			"Unordered":  "Unordered Unordered `xml:\"unordered,omitempty\" json:\"unordered,omitzero\"`",
			"SeqAddr":    "SeqAddr *Address `xml:\"seqAddr,omitempty\" json:\"seqAddr,omitempty\"`",
		},
		"Unordered": {
			"AllAddr": "AllAddr *Address `xml:\"allAddr,omitempty\" json:\"allAddr,omitempty\"`",
		},
		"Payment": {
			"CardAddr": "CardAddr *Address `xml:\"cardAddr,omitempty\" json:\"cardAddr,omitempty\"`",
			"BankAddr": "BankAddr *Address `xml:\"bankAddr,omitempty\" json:\"bankAddr,omitempty\"`",
			"Voucher":  "Voucher *PaymentVoucherType `xml:\"voucher,omitempty\" json:\"voucher,omitempty\"`",
		},
		"Node": {
			"Label":   "Label string `xml:\"label\" json:\"label,omitempty\"`",
			"Child":   "Child *Node `xml:\"child,omitempty\" json:\"child,omitempty\"`",
			"Partner": "Partner *Partner `xml:\"partner,omitempty\" json:\"partner,omitempty\"`",
		},
		"Partner": {
			"Back": "Back *Node `xml:\"back,omitempty\" json:\"back,omitempty\"`",
		},
		"Ring": {
			"Next": "Next *Ring `xml:\"next,omitempty\" json:\"next,omitempty\"`",
		},
	} {
		got := fieldDecls(t, code, typ)
		for field, decl := range want {
			assert.Equal(t, decl, got[field], "%s.%s", typ, field)
		}
	}
	assert.NotContains(t, code, "**", "no field may become a pointer to a pointer")
}

// optionalComplexMarshalTest marshals and unmarshals the generated types: an
// unset optional element of complex type is not sent and decodes to nil, a
// set one is sent in full, a required one is sent even when zero, and a choice
// sends only the branch that is set.
const optionalComplexMarshalTest = `package optcomplex

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

func TestUnsetOptionalComplexElementsAreNotSent(t *testing.T) {
	got := marshal(t, Probe{})
	for _, want := range []string{
		"<reqAddr><city></city></reqAddr>",
		"<homeAddr xmlns=\"urn:optcomplex\"><city></city></homeAddr>",
		"<reqMoney currency=\"\">0</reqMoney>",
		"<reqInline><code></code></reqInline>",
		"<unordered></unordered>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("request lacks required %s:\n%s", want, got)
		}
	}
	for _, absent := range []string{
		"<optAddr", "<nilAddr", "<optNilAddr", "<manyAddr", "<workAddr", "<optMoney",
		"<optNode", "<optInline", "<optAny", "<allAddr", "<seqAddr",
	} {
		if strings.Contains(got, absent) {
			t.Errorf("request has unset optional %s:\n%s", absent, got)
		}
	}
}

func TestSetOptionalComplexElementsAreSentInFull(t *testing.T) {
	got := marshal(t, Probe{
		OptAddr:  &Address{City: "Tartu"},
		WorkAddr: &WorkAddr{Address: Address{City: "Narva"}},
		OptMoney: &Money{Value: 2.5, Currency: "EUR"},
		OptNode:  &Node{Label: "a", Child: &Node{Label: "b"}},
		SeqAddr:  &Address{},
	})
	for _, want := range []string{
		"<optAddr><city>Tartu</city></optAddr>",
		"<workAddr xmlns=\"urn:optcomplex\"><city>Narva</city></workAddr>",
		"<optMoney currency=\"EUR\">2.5</optMoney>",
		"<optNode><label>a</label><child><label>b</label></child></optNode>",
		"<seqAddr><city></city></seqAddr>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("request lacks %s:\n%s", want, got)
		}
	}
}

func TestChoiceSendsOnlyTheComplexBranchSet(t *testing.T) {
	for _, c := range []struct {
		in   Payment
		want string
	}{
		{Payment{}, "<Payment></Payment>"},
		{Payment{BankAddr: &Address{City: "Tallinn"}}, "<Payment><bankAddr><city>Tallinn</city></bankAddr></Payment>"},
		{Payment{CardAddr: &Address{}}, "<Payment><cardAddr><city></city></cardAddr></Payment>"},
		{Payment{Voucher: &PaymentVoucherType{Code: "X1"}}, "<Payment><voucher><code>X1</code></voucher></Payment>"},
	} {
		if got := marshal(t, c.in); got != c.want {
			t.Errorf("got %s, want %s", got, c.want)
		}
	}
}

func TestAbsentOptionalComplexElementsDecodeToNil(t *testing.T) {
	var p Probe
	if err := xml.Unmarshal([]byte(` + "`" + `<Probe xmlns="urn:optcomplex"><reqAddr><city>x</city></reqAddr></Probe>` + "`" + `), &p); err != nil {
		t.Fatal(err)
	}
	if p.ReqAddr.City != "x" {
		t.Errorf("reqAddr decoded as %+v", p.ReqAddr)
	}
	if p.OptAddr != nil || p.WorkAddr != nil || p.OptMoney != nil || p.OptNode != nil || p.OptInline != nil || p.SeqAddr != nil {
		t.Errorf("absent optional elements decoded as non-nil: %+v", p)
	}

	var pay Payment
	if err := xml.Unmarshal([]byte(` + "`" + `<Payment><bankAddr><city>y</city></bankAddr></Payment>` + "`" + `), &pay); err != nil {
		t.Fatal(err)
	}
	if pay.CardAddr != nil || pay.Voucher != nil || pay.BankAddr == nil || pay.BankAddr.City != "y" {
		t.Errorf("payment decoded as %+v", pay)
	}
}

func TestPresentOptionalComplexElementsDecodeToValues(t *testing.T) {
	var p Probe
	in := ` + "`" + `<Probe xmlns="urn:optcomplex"><optAddr><city>a</city></optAddr><workAddr><city>w</city></workAddr><optMoney currency="EUR">3</optMoney><optNode><label>n</label><partner><back><label>m</label></back></partner></optNode><seqAddr></seqAddr></Probe>` + "`" + `
	if err := xml.Unmarshal([]byte(in), &p); err != nil {
		t.Fatal(err)
	}
	if p.OptAddr == nil || p.OptAddr.City != "a" {
		t.Errorf("optAddr decoded as %+v", p.OptAddr)
	}
	if p.WorkAddr == nil || p.WorkAddr.City != "w" {
		t.Errorf("workAddr decoded as %+v", p.WorkAddr)
	}
	if p.OptMoney == nil || p.OptMoney.Value != 3 || p.OptMoney.Currency != "EUR" {
		t.Errorf("optMoney decoded as %+v", p.OptMoney)
	}
	if p.OptNode == nil || p.OptNode.Label != "n" || p.OptNode.Partner == nil || p.OptNode.Partner.Back == nil || p.OptNode.Partner.Back.Label != "m" {
		t.Errorf("optNode decoded as %+v", p.OptNode)
	}
	if p.SeqAddr == nil || p.SeqAddr.City != "" {
		t.Errorf("empty seqAddr decoded as %+v", p.SeqAddr)
	}
}
`

func TestGenerator_OptionalComplexElements_Marshal(t *testing.T) {
	out := generate(t, writeWSDL(t, optionalComplexWSDL), WithPackage("optcomplex"))
	runGeneratedModuleTests(t, "optcomplex", out, map[string]string{"optcomplex_test.go": optionalComplexMarshalTest})
}
