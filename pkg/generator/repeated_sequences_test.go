// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"go/format"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// formattedDecl returns the declaration name of out as gofmt prints it,
// without blank lines.
func formattedDecl(t *testing.T, out generated, name string) string {
	t.Helper()
	compact := regexp.MustCompile(`\n\s*\n`).ReplaceAllString(out.decl(t, name), "\n")
	src, err := format.Source([]byte(compact))
	require.NoError(t, err)
	return string(src)
}

// repeatedSequencesWSDL declares one operation per type below, each taking
// and returning a global element of the type, so that every type is also
// embedded by an element type:
//
//   - R1Type: two required elements;
//   - R2Type: an optional member;
//   - R3Type: required and optional complex members;
//   - R4Type: a single member;
//   - R5Type: a repeated sequence nested in another;
//   - R6Type: a choice inside;
//   - R7Type: a repeated reference to a group whose model group is a sequence;
//   - R8Type: elements before and after;
//   - R9Type: the same in a complexContent extension;
//   - R10Type: a member of the type holding the sequence;
//   - R11Type: a field name taken by an element, a wildcard and an attribute;
const repeatedSequencesWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/" xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/" xmlns:tns="urn:repseq" xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:repseq">
  <types>
    <xs:schema targetNamespace="urn:repseq" elementFormDefault="qualified">
      <xs:complexType name="Item"><xs:sequence><xs:element name="sku" type="xs:string"/></xs:sequence><xs:attribute name="code" type="xs:string"/></xs:complexType>
      <xs:complexType name="Base"><xs:sequence><xs:element name="baseId" type="xs:int"/></xs:sequence></xs:complexType>
      <xs:group name="Pair"><xs:sequence><xs:element name="x" type="xs:string"/><xs:element name="y" type="xs:int"/></xs:sequence></xs:group>
      <xs:complexType name="R1Type"><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:sequence></xs:complexType>
      <xs:element name="R1" type="tns:R1Type"/>
      <xs:element name="R1Response" type="tns:R1Type"/>
      <xs:complexType name="R2Type"><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType>
      <xs:element name="R2" type="tns:R2Type"/>
      <xs:element name="R2Response" type="tns:R2Type"/>
      <xs:complexType name="R3Type"><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:element name="item" type="tns:Item"/><xs:element name="extra" type="tns:Item" minOccurs="0"/></xs:sequence></xs:complexType>
      <xs:element name="R3" type="tns:R3Type"/>
      <xs:element name="R3Response" type="tns:R3Type"/>
      <xs:complexType name="R4Type"><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
      <xs:element name="R4" type="tns:R4Type"/>
      <xs:element name="R4Response" type="tns:R4Type"/>
      <xs:complexType name="R5Type"><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string"/><xs:element name="c" type="xs:string"/></xs:sequence></xs:sequence></xs:complexType>
      <xs:element name="R5" type="tns:R5Type"/>
      <xs:element name="R5Response" type="tns:R5Type"/>
      <xs:complexType name="R6Type"><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:choice><xs:element name="b" type="xs:string"/><xs:element name="c" type="tns:Item"/></xs:choice></xs:sequence></xs:complexType>
      <xs:element name="R6" type="tns:R6Type"/>
      <xs:element name="R6Response" type="tns:R6Type"/>
      <xs:complexType name="R7Type"><xs:sequence><xs:group ref="tns:Pair" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
      <xs:element name="R7" type="tns:R7Type"/>
      <xs:element name="R7Response" type="tns:R7Type"/>
      <xs:complexType name="R8Type"><xs:sequence><xs:element name="id" type="xs:int"/><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:sequence><xs:element name="last" type="xs:string"/></xs:sequence></xs:complexType>
      <xs:element name="R8" type="tns:R8Type"/>
      <xs:element name="R8Response" type="tns:R8Type"/>
      <xs:complexType name="R9Type"><xs:complexContent><xs:extension base="tns:Base"><xs:sequence><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:sequence><xs:element name="last" type="xs:string"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
      <xs:element name="R9" type="tns:R9Type"/>
      <xs:element name="R9Response" type="tns:R9Type"/>
      <xs:complexType name="R10Type"><xs:sequence><xs:sequence maxOccurs="unbounded"><xs:element name="label" type="xs:string"/><xs:element name="child" type="tns:R10Type"/></xs:sequence></xs:sequence></xs:complexType>
      <xs:element name="R10" type="tns:R10Type"/>
      <xs:element name="R10Response" type="tns:R10Type"/>
      <xs:complexType name="R11Type"><xs:sequence><xs:element name="AGroup" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:sequence><xs:any namespace="##other" processContents="lax" minOccurs="0" maxOccurs="unbounded"/></xs:sequence><xs:attribute name="code" type="xs:string"/></xs:complexType>
      <xs:element name="R11" type="tns:R11Type"/>
      <xs:element name="R11Response" type="tns:R11Type"/>
    </xs:schema>
  </types>
  <message name="R1In"><part name="parameters" element="tns:R1"/></message>
  <message name="R1Out"><part name="parameters" element="tns:R1Response"/></message>
  <message name="R2In"><part name="parameters" element="tns:R2"/></message>
  <message name="R2Out"><part name="parameters" element="tns:R2Response"/></message>
  <message name="R3In"><part name="parameters" element="tns:R3"/></message>
  <message name="R3Out"><part name="parameters" element="tns:R3Response"/></message>
  <message name="R4In"><part name="parameters" element="tns:R4"/></message>
  <message name="R4Out"><part name="parameters" element="tns:R4Response"/></message>
  <message name="R5In"><part name="parameters" element="tns:R5"/></message>
  <message name="R5Out"><part name="parameters" element="tns:R5Response"/></message>
  <message name="R6In"><part name="parameters" element="tns:R6"/></message>
  <message name="R6Out"><part name="parameters" element="tns:R6Response"/></message>
  <message name="R7In"><part name="parameters" element="tns:R7"/></message>
  <message name="R7Out"><part name="parameters" element="tns:R7Response"/></message>
  <message name="R8In"><part name="parameters" element="tns:R8"/></message>
  <message name="R8Out"><part name="parameters" element="tns:R8Response"/></message>
  <message name="R9In"><part name="parameters" element="tns:R9"/></message>
  <message name="R9Out"><part name="parameters" element="tns:R9Response"/></message>
  <message name="R10In"><part name="parameters" element="tns:R10"/></message>
  <message name="R10Out"><part name="parameters" element="tns:R10Response"/></message>
  <message name="R11In"><part name="parameters" element="tns:R11"/></message>
  <message name="R11Out"><part name="parameters" element="tns:R11Response"/></message>
  <portType name="SeqPort">
    <operation name="OpR1"><input message="tns:R1In"/><output message="tns:R1Out"/></operation>
    <operation name="OpR2"><input message="tns:R2In"/><output message="tns:R2Out"/></operation>
    <operation name="OpR3"><input message="tns:R3In"/><output message="tns:R3Out"/></operation>
    <operation name="OpR4"><input message="tns:R4In"/><output message="tns:R4Out"/></operation>
    <operation name="OpR5"><input message="tns:R5In"/><output message="tns:R5Out"/></operation>
    <operation name="OpR6"><input message="tns:R6In"/><output message="tns:R6Out"/></operation>
    <operation name="OpR7"><input message="tns:R7In"/><output message="tns:R7Out"/></operation>
    <operation name="OpR8"><input message="tns:R8In"/><output message="tns:R8Out"/></operation>
    <operation name="OpR9"><input message="tns:R9In"/><output message="tns:R9Out"/></operation>
    <operation name="OpR10"><input message="tns:R10In"/><output message="tns:R10Out"/></operation>
    <operation name="OpR11"><input message="tns:R11In"/><output message="tns:R11Out"/></operation>
  </portType>
  <binding name="SeqBinding" type="tns:SeqPort">
    <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
    <operation name="OpR1"><soap:operation soapAction="urn:repseq:R1"/><input><soap:body use="literal"/></input><output><soap:body use="literal"/></output></operation>
    <operation name="OpR2"><soap:operation soapAction="urn:repseq:R2"/><input><soap:body use="literal"/></input><output><soap:body use="literal"/></output></operation>
    <operation name="OpR3"><soap:operation soapAction="urn:repseq:R3"/><input><soap:body use="literal"/></input><output><soap:body use="literal"/></output></operation>
    <operation name="OpR4"><soap:operation soapAction="urn:repseq:R4"/><input><soap:body use="literal"/></input><output><soap:body use="literal"/></output></operation>
    <operation name="OpR5"><soap:operation soapAction="urn:repseq:R5"/><input><soap:body use="literal"/></input><output><soap:body use="literal"/></output></operation>
    <operation name="OpR6"><soap:operation soapAction="urn:repseq:R6"/><input><soap:body use="literal"/></input><output><soap:body use="literal"/></output></operation>
    <operation name="OpR7"><soap:operation soapAction="urn:repseq:R7"/><input><soap:body use="literal"/></input><output><soap:body use="literal"/></output></operation>
    <operation name="OpR8"><soap:operation soapAction="urn:repseq:R8"/><input><soap:body use="literal"/></input><output><soap:body use="literal"/></output></operation>
    <operation name="OpR9"><soap:operation soapAction="urn:repseq:R9"/><input><soap:body use="literal"/></input><output><soap:body use="literal"/></output></operation>
    <operation name="OpR10"><soap:operation soapAction="urn:repseq:R10"/><input><soap:body use="literal"/></input><output><soap:body use="literal"/></output></operation>
    <operation name="OpR11"><soap:operation soapAction="urn:repseq:R11"/><input><soap:body use="literal"/></input><output><soap:body use="literal"/></output></operation>
  </binding>
  <service name="SeqService"><port name="SeqPort" binding="tns:SeqBinding"><soap:address location="http://localhost/"/></port></service>
</definitions>`

func TestGenerator_RepeatedSequences_Declarations(t *testing.T) {
	out := generate(t, writeWSDL(t, repeatedSequencesWSDL), WithPackage("repseq"))

	for name, want := range map[string]string{
		"R1Type":                  "type R1Type struct {\n\tAGroup []R1TypeAGroup `xml:\"a,omitempty\" json:\"aGroup,omitempty\"`\n}",
		"R1TypeAGroup":            "type R1TypeAGroup struct {\n\tA string `xml:\"a\" json:\"a,omitempty\"`\n\tB int32  `xml:\"b\" json:\"b,omitempty\"`\n}",
		"R2TypeAGroup":            "type R2TypeAGroup struct {\n\tA string `xml:\"a\" json:\"a,omitempty\"`\n\tB string `xml:\"b,omitempty\" json:\"b,omitempty\"`\n}",
		"R3TypeAGroup":            "type R3TypeAGroup struct {\n\tA     string `xml:\"a\" json:\"a,omitempty\"`\n\tItem  Item   `xml:\"item,omitempty\" json:\"item,omitzero\"`\n\tExtra *Item  `xml:\"extra,omitempty\" json:\"extra,omitempty\"`\n}",
		"R4Type":                  "type R4Type struct {\n\tA []string `xml:\"a,omitempty\" json:\"a,omitempty\"`\n}",
		"R5Type":                  "type R5Type struct {\n\tAGroup []R5TypeAGroup `xml:\"a,omitempty\" json:\"aGroup,omitempty\"`\n}",
		"R5TypeAGroup":            "type R5TypeAGroup struct {\n\tA      string         `xml:\"a\" json:\"a,omitempty\"`\n\tBGroup []R5TypeBGroup `xml:\"b,omitempty\" json:\"bGroup,omitempty\"`\n}",
		"R5TypeBGroup":            "type R5TypeBGroup struct {\n\tB string `xml:\"b\" json:\"b,omitempty\"`\n\tC string `xml:\"c\" json:\"c,omitempty\"`\n}",
		"R6TypeAGroup":            "type R6TypeAGroup struct {\n\tA string `xml:\"a\" json:\"a,omitempty\"`\n\tB string `xml:\"b,omitempty\" json:\"b,omitempty\"`\n\tC *Item  `xml:\"c,omitempty\" json:\"c,omitempty\"`\n}",
		"R7Type":                  "type R7Type struct {\n\tPairGroup []R7TypePairGroup `xml:\"x,omitempty\" json:\"PairGroup,omitempty\"`\n}",
		"R7TypePairGroup":         "type R7TypePairGroup struct {\n\tX string `xml:\"x\" json:\"x,omitempty\"`\n\tY int32  `xml:\"y\" json:\"y,omitempty\"`\n}",
		"R8Type":                  "type R8Type struct {\n\tId     int32          `xml:\"id\" json:\"id,omitempty\"`\n\tAGroup []R8TypeAGroup `xml:\"a,omitempty\" json:\"aGroup,omitempty\"`\n\tLast   string         `xml:\"last\" json:\"last,omitempty\"`\n}",
		"R9Type":                  "type R9Type struct {\n\tBase\n\tAGroup []R9TypeAGroup `xml:\"a,omitempty\" json:\"aGroup,omitempty\"`\n\tLast   string         `xml:\"last\" json:\"last,omitempty\"`\n}",
		"R10Type":                 "type R10Type struct {\n\tLabelGroup []R10TypeLabelGroup `xml:\"label,omitempty\" json:\"labelGroup,omitempty\"`\n}",
		"R10TypeLabelGroup":       "type R10TypeLabelGroup struct {\n\tLabel string  `xml:\"label\" json:\"label,omitempty\"`\n\tChild R10Type `xml:\"child,omitempty\" json:\"child,omitzero\"`\n}",
		"R11Type":                 "type R11Type struct {\n\tAGroup  string            `xml:\"AGroup\" json:\"AGroup,omitempty\"`\n\tAGroup_ []R11TypeAGroup   `xml:\"a,omitempty\" json:\"aGroup,omitempty\"`\n\tCode    string            `xml:\"code,attr,omitempty\" json:\"code,omitempty\"`\n\tItems   []soap.AnyElement `xml:\",any\"`\n}",
		"R1TypeAGroup.MarshalXML": "func (o R1TypeAGroup) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {\n\ttype plain R1TypeAGroup\n\treturn soap.MarshalModelGroup(e, plain(o))\n}",
		"R1Type.UnmarshalXML":     "func (t *R1Type) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {\n\treturn soap.UnmarshalModelGroups(d, start, &struct {\n\t\t*R1Type\n\t\tUnmarshalXML struct{}              `xml:\"-\"`\n\t\tM0           soap.ModelGroupMember `xml:\"a\"`\n\t\tM1           soap.ModelGroupMember `xml:\"b\"`\n\t}{\n\t\tR1Type: t,\n\t\tM0:     soap.GroupMember[R1TypeAGroup](0),\n\t\tM1:     soap.GroupMember[R1TypeAGroup](1),\n\t})\n}",
		"R1.UnmarshalXML":         "func (t *R1) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {\n\treturn soap.UnmarshalModelGroups(d, start, &struct {\n\t\t*R1\n\t\tUnmarshalXML struct{}              `xml:\"-\"`\n\t\tM0           soap.ModelGroupMember `xml:\"a\"`\n\t\tM1           soap.ModelGroupMember `xml:\"b\"`\n\t}{\n\t\tR1: t,\n\t\tM0: soap.GroupMember[R1TypeAGroup](0),\n\t\tM1: soap.GroupMember[R1TypeAGroup](1),\n\t})\n}",
		"R5Type.UnmarshalXML":     "func (t *R5Type) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {\n\treturn soap.UnmarshalModelGroups(d, start, &struct {\n\t\t*R5Type\n\t\tUnmarshalXML struct{}              `xml:\"-\"`\n\t\tM0           soap.ModelGroupMember `xml:\"a\"`\n\t\tM1           soap.ModelGroupMember `xml:\"b\"`\n\t\tM2           soap.ModelGroupMember `xml:\"c\"`\n\t}{\n\t\tR5Type: t,\n\t\tM0:     soap.GroupMember[R5TypeAGroup](0),\n\t\tM1:     soap.GroupMember[R5TypeAGroup](1, 0),\n\t\tM2:     soap.GroupMember[R5TypeAGroup](1, 1),\n\t})\n}",
	} {
		assert.Equal(t, want, formattedDecl(t, out, name), name)
	}

	names := out.declNames(t)
	for _, name := range []string{"R4Type.UnmarshalXML", "R4.UnmarshalXML", "Item.UnmarshalXML", "Base.UnmarshalXML", "R1TypeAGroup.UnmarshalXML", "R5TypeAGroup.UnmarshalXML", "R4TypeAGroup"} {
		assert.NotContains(t, names, name)
	}
	for _, name := range []string{"R9.UnmarshalXML", "R9Response.UnmarshalXML", "R10Type.UnmarshalXML", "R11Type.UnmarshalXML", "R5TypeBGroup.MarshalXML"} {
		assert.Contains(t, names, name)
	}
}

// repeatedSequencesClientTest sends and receives occurrences of the
// repeated sequences through the generated client.
const repeatedSequencesClientTest = `package repseq

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"testing"

	"github.com/kastrul/gowsdl/soap"
)

var bodyPattern = regexp.MustCompile("(?s)<soap:Body>(.*)</soap:Body>")

// exchange calls op with req against a server answering with response, and
// returns the body element sent.
func exchange[Req, Resp any](t *testing.T, req *Req, response string, op func(SeqPort, context.Context, *Req) (*Resp, error)) (string, *Resp) {
	t.Helper()
	var sent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		sent = string(data)
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = io.WriteString(w, ` + "`" + `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body>` + "`" + `+response+` + "`" + `</soap:Body></soap:Envelope>` + "`" + `)
	}))
	defer server.Close()
	resp, err := op(NewSeqPort(soap.NewClient(server.URL)), context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	m := bodyPattern.FindStringSubmatch(sent)
	if m == nil {
		t.Fatalf("no body in %s", sent)
	}
	return m[1], resp
}

func check(t *testing.T, what string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n got %#v\nwant %#v", what, got, want)
	}
}

func TestOptionalMember(t *testing.T) {
	req := &R2{R2Type: R2Type{AGroup: []R2TypeAGroup{{A: "a1", B: "b1"}, {A: "a2"}, {A: "a3", B: "b3"}}}}
	const in = ` + "`" + `<R2Response xmlns="urn:repseq"><a>a1</a><a>a2</a><b>b2</b><a>a3</a></R2Response>` + "`" + `
	sent, resp := exchange(t, req, in, SeqPort.OpR2)
	check(t, "sent", sent, ` + "`" + `<R2 xmlns="urn:repseq"><a>a1</a><b>b1</b><a>a2</a><a>a3</a><b>b3</b></R2>` + "`" + `)
	check(t, "decoded", resp.AGroup, []R2TypeAGroup{{A: "a1"}, {A: "a2", B: "b2"}, {A: "a3"}})

	// What was decoded is sent back as it came.
	again, _ := exchange(t, &R2{R2Type: resp.R2Type}, in, SeqPort.OpR2)
	check(t, "sent back", again, ` + "`" + `<R2 xmlns="urn:repseq"><a>a1</a><a>a2</a><b>b2</b><a>a3</a></R2>` + "`" + `)
}

func TestRequiredMembers(t *testing.T) {
	req := &R1{R1Type: R1Type{AGroup: []R1TypeAGroup{{A: "a1", B: 1}, {A: "", B: 0}, {A: "a3", B: 3}}}}
	sent, resp := exchange(t, req, ` + "`" + `<R1Response xmlns="urn:repseq"><a>a1</a><b>1</b><a></a><b>0</b></R1Response>` + "`" + `, SeqPort.OpR1)
	check(t, "sent", sent, ` + "`" + `<R1 xmlns="urn:repseq"><a>a1</a><b>1</b><a></a><b>0</b><a>a3</a><b>3</b></R1>` + "`" + `)
	check(t, "decoded", resp.AGroup, []R1TypeAGroup{{A: "a1", B: 1}, {}})
	check(t, "element name", resp.XMLName, xml.Name{Space: "urn:repseq", Local: "R1Response"})
}

func TestComplexMembers(t *testing.T) {
	req := &R3{R3Type: R3Type{AGroup: []R3TypeAGroup{{A: "a1", Item: Item{Sku: "i1"}}, {A: "a2", Item: Item{Sku: "i2", Code: "c"}, Extra: &Item{Sku: "e2"}}}}}
	in := ` + "`" + `<R3Response xmlns="urn:repseq"><a>a1</a><item><sku>i1</sku></item><extra><sku>e1</sku></extra><a>a2</a><item code="c"><sku>i2</sku></item></R3Response>` + "`" + `
	sent, resp := exchange(t, req, in, SeqPort.OpR3)
	check(t, "sent", sent, ` + "`" + `<R3 xmlns="urn:repseq"><a>a1</a><item><sku>i1</sku></item><a>a2</a><item code="c"><sku>i2</sku></item><extra><sku>e2</sku></extra></R3>` + "`" + `)
	check(t, "decoded", resp.AGroup, []R3TypeAGroup{{A: "a1", Item: Item{Sku: "i1"}, Extra: &Item{Sku: "e1"}}, {A: "a2", Item: Item{Sku: "i2", Code: "c"}}})
}

func TestSingleMember(t *testing.T) {
	sent, resp := exchange(t, &R4{R4Type: R4Type{A: []string{"a1", "a2"}}}, ` + "`" + `<R4Response xmlns="urn:repseq"><a>a1</a><a>a2</a></R4Response>` + "`" + `, SeqPort.OpR4)
	check(t, "sent", sent, ` + "`" + `<R4 xmlns="urn:repseq"><a>a1</a><a>a2</a></R4>` + "`" + `)
	check(t, "decoded", resp.A, []string{"a1", "a2"})
}

func TestNested(t *testing.T) {
	req := &R5{R5Type: R5Type{AGroup: []R5TypeAGroup{
		{A: "a1", BGroup: []R5TypeBGroup{{B: "b1", C: "c1"}, {B: "b2", C: "c2"}}},
		{A: "a2", BGroup: []R5TypeBGroup{{B: "b3", C: "c3"}}},
	}}}
	const body = ` + "`" + `<a>a1</a><b>b1</b><c>c1</c><b>b2</b><c>c2</c><a>a2</a><b>b3</b><c>c3</c>` + "`" + `
	sent, resp := exchange(t, req, ` + "`" + `<R5Response xmlns="urn:repseq">` + "`" + `+body+` + "`" + `</R5Response>` + "`" + `, SeqPort.OpR5)
	check(t, "sent", sent, ` + "`" + `<R5 xmlns="urn:repseq">` + "`" + `+body+` + "`" + `</R5>` + "`" + `)
	check(t, "decoded", resp.AGroup, req.AGroup)
}

func TestChoiceInside(t *testing.T) {
	req := &R6{R6Type: R6Type{AGroup: []R6TypeAGroup{{A: "a1", B: "b1"}, {A: "a2", C: &Item{Sku: "c2"}}}}}
	const body = ` + "`" + `<a>a1</a><b>b1</b><a>a2</a><c><sku>c2</sku></c>` + "`" + `
	sent, resp := exchange(t, req, ` + "`" + `<R6Response xmlns="urn:repseq">` + "`" + `+body+` + "`" + `</R6Response>` + "`" + `, SeqPort.OpR6)
	check(t, "sent", sent, ` + "`" + `<R6 xmlns="urn:repseq">` + "`" + `+body+` + "`" + `</R6>` + "`" + `)
	check(t, "decoded", resp.AGroup, req.AGroup)
}

func TestGroupRef(t *testing.T) {
	req := &R7{R7Type: R7Type{PairGroup: []R7TypePairGroup{{X: "x1", Y: 1}, {X: "x2", Y: 2}}}}
	const body = ` + "`" + `<x>x1</x><y>1</y><x>x2</x><y>2</y>` + "`" + `
	sent, resp := exchange(t, req, ` + "`" + `<R7Response xmlns="urn:repseq">` + "`" + `+body+` + "`" + `</R7Response>` + "`" + `, SeqPort.OpR7)
	check(t, "sent", sent, ` + "`" + `<R7 xmlns="urn:repseq">` + "`" + `+body+` + "`" + `</R7>` + "`" + `)
	check(t, "decoded", resp.PairGroup, req.PairGroup)
}

func TestBeforeAndAfter(t *testing.T) {
	req := &R8{R8Type: R8Type{Id: 7, AGroup: []R8TypeAGroup{{A: "a1", B: 1}, {A: "a2", B: 2}}, Last: "z"}}
	const body = ` + "`" + `<id>7</id><a>a1</a><b>1</b><a>a2</a><b>2</b><last>z</last>` + "`" + `
	sent, resp := exchange(t, req, ` + "`" + `<R8Response xmlns="urn:repseq">` + "`" + `+body+` + "`" + `</R8Response>` + "`" + `, SeqPort.OpR8)
	check(t, "sent", sent, ` + "`" + `<R8 xmlns="urn:repseq">` + "`" + `+body+` + "`" + `</R8>` + "`" + `)
	check(t, "decoded", resp.R8Type, req.R8Type)
}

func TestExtension(t *testing.T) {
	req := &R9{R9Type: R9Type{Base: Base{BaseId: 9}, AGroup: []R9TypeAGroup{{A: "a1", B: 1}, {A: "a2", B: 2}}, Last: "z"}}
	const body = ` + "`" + `<baseId>9</baseId><a>a1</a><b>1</b><a>a2</a><b>2</b><last>z</last>` + "`" + `
	sent, resp := exchange(t, req, ` + "`" + `<R9Response xmlns="urn:repseq">` + "`" + `+body+` + "`" + `</R9Response>` + "`" + `, SeqPort.OpR9)
	check(t, "sent", sent, ` + "`" + `<R9 xmlns="urn:repseq">` + "`" + `+body+` + "`" + `</R9>` + "`" + `)
	check(t, "decoded", resp.R9Type, req.R9Type)
}

func TestRecursive(t *testing.T) {
	req := &R10{R10Type: R10Type{LabelGroup: []R10TypeLabelGroup{
		{Label: "l1", Child: R10Type{LabelGroup: []R10TypeLabelGroup{{Label: "inner"}}}},
		{Label: "l2"},
	}}}
	const body = ` + "`" + `<label>l1</label><child><label>inner</label><child></child></child><label>l2</label><child></child>` + "`" + `
	sent, resp := exchange(t, req, ` + "`" + `<R10Response xmlns="urn:repseq">` + "`" + `+body+` + "`" + `</R10Response>` + "`" + `, SeqPort.OpR10)
	check(t, "sent", sent, ` + "`" + `<R10 xmlns="urn:repseq">` + "`" + `+body+` + "`" + `</R10>` + "`" + `)
	check(t, "decoded", resp.R10Type, req.R10Type)
}

func TestRenamedFieldWildcardAndAttribute(t *testing.T) {
	req := &R11{R11Type: R11Type{AGroup: "g", AGroup_: []R11TypeAGroup{{A: "a1", B: 1}, {A: "a2", B: 2}}, Code: "c"}}
	sent, resp := exchange(t, req, ` + "`" + `<R11Response xmlns="urn:repseq" code="c"><AGroup>g</AGroup><a>a1</a><b>1</b><a>a2</a><b>2</b><o:x xmlns:o="urn:other">w</o:x></R11Response>` + "`" + `, SeqPort.OpR11)
	check(t, "sent", sent, ` + "`" + `<R11 xmlns="urn:repseq" code="c"><AGroup>g</AGroup><a>a1</a><b>1</b><a>a2</a><b>2</b></R11>` + "`" + `)
	check(t, "decoded fields", []any{resp.AGroup, resp.AGroup_, resp.Code}, []any{req.AGroup, req.AGroup_, req.Code})
	if len(resp.Items) != 1 || resp.Items[0].XMLName.Local != "x" {
		t.Errorf("wildcard items %+v, want the one x element", resp.Items)
	}
}
`

func TestGenerator_RepeatedSequences_SendAndReceive(t *testing.T) {
	out := generate(t, writeWSDL(t, repeatedSequencesWSDL), WithPackage("repseq"))
	runGeneratedModuleTests(t, "repseq", out, map[string]string{"repseq_test.go": repeatedSequencesClientTest})
}

func TestGenerator_RepeatedSequences_ServerAndGenericsBuild(t *testing.T) {
	assertServerAndGenericsBuild(t, writeWSDL(t, repeatedSequencesWSDL), "repseq")
}

func TestGenerator_RepeatedSequences_TypePruning(t *testing.T) {
	out := generate(t, writeWSDL(t, repeatedSequencesWSDL), WithPackage("repseq"), WithOperations("OpR5"))
	names := out.declNames(t)
	for _, name := range []string{"R5Type", "R5TypeAGroup", "R5TypeBGroup", "R5TypeAGroup.MarshalXML", "R5TypeBGroup.MarshalXML", "R5Type.UnmarshalXML", "R5.UnmarshalXML", "R5Response.UnmarshalXML"} {
		assert.Contains(t, names, name)
	}
	for _, name := range []string{"R1Type", "R1TypeAGroup", "R1TypeAGroup.MarshalXML", "R1Type.UnmarshalXML"} {
		assert.NotContains(t, names, name)
	}
	assertGeneratedCodeBuilds(t, "repseq", out)
}
