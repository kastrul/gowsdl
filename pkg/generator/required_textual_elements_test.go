// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// requiredTextualWSDL declares required and optional elements and attributes
// of string, enumeration, union and date and time types.
const requiredTextualWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="urn:textual"
             xmlns:xs="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:textual">
  <types>
    <xs:schema targetNamespace="urn:textual" elementFormDefault="qualified">
      <xs:simpleType name="Color">
        <xs:restriction base="xs:string"><xs:enumeration value="red"/><xs:enumeration value="blue"/></xs:restriction>
      </xs:simpleType>
      <xs:simpleType name="Code">
        <xs:restriction base="xs:token"><xs:maxLength value="8"/></xs:restriction>
      </xs:simpleType>
      <xs:simpleType name="Either"><xs:union memberTypes="xs:int xs:string"/></xs:simpleType>
      <xs:element name="reference" type="xs:string"/>
      <xs:element name="Booking">
        <xs:complexType>
          <xs:sequence>
            <xs:element name="name" type="xs:string"/>
            <xs:element name="note" type="xs:string" minOccurs="0"/>
            <xs:element name="color" type="tns:Color"/>
            <xs:element name="shade" type="tns:Color" minOccurs="0"/>
            <xs:element name="code" type="tns:Code"/>
            <xs:element name="either" type="tns:Either"/>
            <xs:element name="year" type="xs:gYear"/>
            <xs:element name="link" type="xs:anyURI"/>
            <xs:element name="label">
              <xs:simpleType><xs:restriction base="xs:string"><xs:maxLength value="20"/></xs:restriction></xs:simpleType>
            </xs:element>
            <xs:element ref="tns:reference"/>
            <xs:element name="at" type="xs:dateTime"/>
            <xs:element name="until" type="xs:dateTime" minOccurs="0"/>
            <xs:element name="day" type="xs:date"/>
            <xs:element name="hour" type="xs:time"/>
            <xs:element name="blob" type="xs:base64Binary"/>
            <xs:element name="alias" type="xs:string" nillable="true"/>
            <xs:element name="tags" type="xs:string" maxOccurs="unbounded"/>
            <xs:sequence minOccurs="0">
              <xs:element name="inOptionalSequence" type="xs:string"/>
            </xs:sequence>
            <xs:choice>
              <xs:element name="email" type="xs:string"/>
              <xs:element name="phone" type="xs:string"/>
            </xs:choice>
          </xs:sequence>
          <xs:attribute name="channel" type="xs:string" use="required"/>
          <xs:attribute name="locale" type="xs:string"/>
          <xs:attribute name="mode" type="tns:Color" use="required"/>
          <xs:attribute name="on" type="xs:date" use="required"/>
        </xs:complexType>
      </xs:element>
      <xs:element name="BookingResponse">
        <xs:complexType>
          <xs:sequence><xs:element name="id" type="xs:string"/></xs:sequence>
        </xs:complexType>
      </xs:element>
    </xs:schema>
  </types>
  <message name="BookingIn"><part name="parameters" element="tns:Booking"/></message>
  <message name="BookingOut"><part name="parameters" element="tns:BookingResponse"/></message>
  <portType name="BookingPort">
    <operation name="Book"><input message="tns:BookingIn"/><output message="tns:BookingOut"/></operation>
  </portType>
  <binding name="BookingBinding" type="tns:BookingPort">
    <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
    <operation name="Book">
      <soap:operation soapAction="urn:textual:Book"/>
      <input><soap:body use="literal"/></input>
      <output><soap:body use="literal"/></output>
    </operation>
  </binding>
  <service name="BookingService">
    <port name="BookingPort" binding="tns:BookingBinding"><soap:address location="http://localhost/"/></port>
  </service>
</definitions>`

func TestGenerator_RequiredTextualElements_Tags(t *testing.T) {
	code := generate(t, writeWSDL(t, requiredTextualWSDL), WithPackage("textual")).client(t)

	booking := fieldTags(t, code, "Booking")
	for field, tag := range map[string]string{
		"Name":               "`xml:\"name\" json:\"name,omitempty\"`",
		"Note":               "`xml:\"note,omitempty\" json:\"note,omitempty\"`",
		"Color":              "`xml:\"color\" json:\"color,omitempty\"`",
		"Shade":              "`xml:\"shade,omitempty\" json:\"shade,omitempty\"`",
		"Code":               "`xml:\"code\" json:\"code,omitempty\"`",
		"Either":             "`xml:\"either\" json:\"either,omitempty\"`",
		"Year":               "`xml:\"year\" json:\"year,omitempty\"`",
		"Link":               "`xml:\"link\" json:\"link,omitempty\"`",
		"Label":              "`xml:\"label\" json:\"label,omitempty\"`",
		"Reference":          "`xml:\"reference\" json:\"reference,omitempty\"`",
		"At":                 "`xml:\"at\" json:\"at,omitzero\"`",
		"Until":              "`xml:\"until,omitempty\" json:\"until,omitzero\"`",
		"Day":                "`xml:\"day\" json:\"day,omitzero\"`",
		"Hour":               "`xml:\"hour\" json:\"hour,omitzero\"`",
		"Blob":               "`xml:\"blob,omitempty\" json:\"blob,omitempty\"`",
		"Alias":              "`xml:\"alias,omitempty\" json:\"alias,omitempty\"`",
		"Tags":               "`xml:\"tags,omitempty\" json:\"tags,omitempty\"`",
		"InOptionalSequence": "`xml:\"inOptionalSequence,omitempty\" json:\"inOptionalSequence,omitempty\"`",
		"Email":              "`xml:\"email,omitempty\" json:\"email,omitempty\"`",
		"Phone":              "`xml:\"phone,omitempty\" json:\"phone,omitempty\"`",
		"Channel":            "`xml:\"channel,attr\" json:\"channel,omitempty\"`",
		"Locale":             "`xml:\"locale,attr,omitempty\" json:\"locale,omitempty\"`",
		"Mode":               "`xml:\"mode,attr\" json:\"mode,omitempty\"`",
		"On":                 "`xml:\"on,attr\" json:\"on,omitzero\"`",
	} {
		assert.Equal(t, tag, booking[field], "Booking.%s", field)
	}
}

// requiredTextualMarshalTest marshals the generated Booking request: required
// strings, enumerations and unions left unset are sent empty, optional ones
// are not sent, and zero dates and times are still not sent at all, since
// soap.XSDDateTime, XSDDate and XSDTime write nothing for a zero value.
const requiredTextualMarshalTest = `package textual

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestRequiredTextualZeroValues(t *testing.T) {
	b, err := xml.Marshal(Booking{Email: "a@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{
		"<name></name>", "<color></color>", "<code></code>", "<either></either>",
		"<year></year>", "<link></link>", "<label></label>", "<reference></reference>",
		"<email>a@example.com</email>",
		` + "`" + `channel=""` + "`" + `, ` + "`" + `mode=""` + "`" + `,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("request lacks %s:\n%s", want, got)
		}
	}
	for _, absent := range []string{
		"<note>", "<shade>", "<until>", "<at>", "<day>", "<hour>", "<blob>", "<alias>", "<tags>",
		"<inOptionalSequence>", "<phone>", "locale=", "on=",
	} {
		if strings.Contains(got, absent) {
			t.Errorf("request has %s:\n%s", absent, got)
		}
	}
}

func TestTextualResponsesDecodeAsBefore(t *testing.T) {
	var got Booking
	in := ` + "`" + `<Booking xmlns="urn:textual" channel="web" mode="red"><name>Ann</name><color>blue</color><at>2026-01-02T03:04:05Z</at><phone>1</phone></Booking>` + "`" + `
	if err := xml.Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "Ann" || got.Color != ColorBlue || got.Channel != "web" || got.Mode != ColorRed || got.Phone != "1" || got.At.ToGoTime().Year() != 2026 {
		t.Errorf("decoded %+v", got)
	}
}
`

func TestGenerator_RequiredTextualElements_Marshal(t *testing.T) {
	out := generate(t, writeWSDL(t, requiredTextualWSDL), WithPackage("textual"))
	runGeneratedModuleTests(t, "textual", out, map[string]string{"textual_test.go": requiredTextualMarshalTest})
}
