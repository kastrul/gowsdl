// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// groupRefsWSDL refers to named model groups from a global element's
// anonymous type, a complexContent extension and a group, from a sequence
// and from a choice, with minOccurs="0" on the reference, and from a group
// whose element closes a recursive cycle.
const groupRefsWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="urn:groups"
             xmlns:xs="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:groups">
  <types>
    <xs:schema targetNamespace="urn:groups" elementFormDefault="qualified">
      <xs:complexType name="Address">
        <xs:sequence><xs:element name="city" type="xs:string"/></xs:sequence>
      </xs:complexType>
      <xs:complexType name="Card">
        <xs:sequence><xs:element name="number" type="xs:string"/></xs:sequence>
      </xs:complexType>
      <xs:group name="Contact">
        <xs:sequence>
          <xs:element name="email" type="xs:string"/>
          <xs:element name="phone" type="xs:string" minOccurs="0"/>
          <xs:element name="address" type="tns:Address"/>
        </xs:sequence>
      </xs:group>
      <xs:group name="Payment">
        <xs:choice>
          <xs:element name="card" type="tns:Card"/>
          <xs:element name="iban" type="xs:string"/>
        </xs:choice>
      </xs:group>
      <xs:group name="Backup">
        <xs:sequence>
          <xs:element name="backupEmail" type="xs:string"/>
          <xs:element name="backupAddress" type="tns:Address"/>
        </xs:sequence>
      </xs:group>
      <xs:group name="Audit">
        <xs:sequence>
          <xs:element name="createdBy" type="xs:string"/>
          <xs:group ref="tns:Stamp"/>
        </xs:sequence>
      </xs:group>
      <xs:group name="Stamp">
        <xs:sequence><xs:element name="stampedAt" type="xs:int"/></xs:sequence>
      </xs:group>
      <xs:group name="Extra">
        <xs:sequence>
          <xs:element name="extra">
            <xs:complexType><xs:sequence><xs:element name="level" type="xs:int"/></xs:sequence></xs:complexType>
          </xs:element>
        </xs:sequence>
      </xs:group>
      <xs:group name="Shared">
        <xs:sequence>
          <xs:element name="box">
            <xs:complexType><xs:sequence><xs:element name="width" type="xs:int"/></xs:sequence></xs:complexType>
          </xs:element>
        </xs:sequence>
      </xs:group>
      <xs:complexType name="UsesShared">
        <xs:sequence><xs:group ref="tns:Shared"/></xs:sequence>
      </xs:complexType>
      <xs:complexType name="AlsoUsesShared">
        <xs:sequence><xs:group ref="tns:Shared" minOccurs="0"/></xs:sequence>
      </xs:complexType>
      <xs:group name="Tagging">
        <xs:sequence><xs:element name="tag" type="xs:string" maxOccurs="unbounded"/></xs:sequence>
      </xs:group>
      <xs:group name="Link">
        <xs:sequence><xs:element name="next" type="tns:Node"/></xs:sequence>
      </xs:group>
      <xs:complexType name="Node">
        <xs:sequence>
          <xs:element name="label" type="xs:string"/>
          <xs:group ref="tns:Link"/>
        </xs:sequence>
      </xs:complexType>
      <xs:complexType name="Base">
        <xs:sequence><xs:element name="baseId" type="xs:int"/></xs:sequence>
      </xs:complexType>
      <xs:complexType name="Tagged">
        <xs:complexContent>
          <xs:extension base="tns:Base">
            <xs:sequence>
              <xs:element name="extFirst" type="xs:string"/>
              <xs:group ref="tns:Tagging"/>
              <xs:element name="extLast" type="xs:string"/>
            </xs:sequence>
          </xs:extension>
        </xs:complexContent>
      </xs:complexType>
      <xs:element name="Customer">
        <xs:complexType>
          <xs:sequence>
            <xs:element name="id" type="xs:int"/>
            <xs:group ref="tns:Contact"/>
            <xs:group ref="tns:Payment"/>
            <xs:group ref="tns:Backup" minOccurs="0"/>
            <xs:choice>
              <xs:group ref="tns:Audit"/>
              <xs:element name="anonymous" type="xs:boolean"/>
            </xs:choice>
            <xs:group ref="tns:Extra"/>
            <xs:element name="tagged" type="tns:Tagged"/>
            <xs:element name="node" type="tns:Node" minOccurs="0"/>
            <xs:element name="last" type="xs:string"/>
          </xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:element name="CustomerResponse">
        <xs:complexType>
          <xs:sequence><xs:group ref="tns:Contact"/></xs:sequence>
        </xs:complexType>
      </xs:element>
    </xs:schema>
  </types>
  <message name="CustomerIn"><part name="parameters" element="tns:Customer"/></message>
  <message name="CustomerOut"><part name="parameters" element="tns:CustomerResponse"/></message>
  <portType name="CustomerPort">
    <operation name="SaveCustomer"><input message="tns:CustomerIn"/><output message="tns:CustomerOut"/></operation>
  </portType>
  <binding name="CustomerBinding" type="tns:CustomerPort">
    <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
    <operation name="SaveCustomer">
      <soap:operation soapAction="urn:groups:SaveCustomer"/>
      <input><soap:body use="literal"/></input>
      <output><soap:body use="literal"/></output>
    </operation>
  </binding>
  <service name="CustomerService">
    <port name="CustomerPort" binding="tns:CustomerBinding"><soap:address location="http://localhost/"/></port>
  </service>
</definitions>`

func TestGenerator_GroupRefs_Fields(t *testing.T) {
	code := generate(t, writeWSDL(t, groupRefsWSDL), WithPackage("groups")).client(t)

	assert.Equal(t, []string{
		"XMLName xml.Name `xml:\"urn:groups Customer\"`",
		"Id int32 `xml:\"id\" json:\"id,omitempty\"`",
		"Email string `xml:\"email\" json:\"email,omitempty\"`",
		"Phone string `xml:\"phone,omitempty\" json:\"phone,omitempty\"`",
		"Address Address `xml:\"address,omitempty\" json:\"address,omitzero\"`",
		"Card *Card `xml:\"card,omitempty\" json:\"card,omitempty\"`",
		"Iban string `xml:\"iban,omitempty\" json:\"iban,omitempty\"`",
		"BackupEmail string `xml:\"backupEmail,omitempty\" json:\"backupEmail,omitempty\"`",
		"BackupAddress *Address `xml:\"backupAddress,omitempty\" json:\"backupAddress,omitempty\"`",
		"CreatedBy string `xml:\"createdBy,omitempty\" json:\"createdBy,omitempty\"`",
		"StampedAt int32 `xml:\"stampedAt,omitempty\" json:\"stampedAt,omitempty\"`",
		"Anonymous bool `xml:\"anonymous,omitempty\" json:\"anonymous,omitempty\"`",
		"Extra CustomerExtraType `xml:\"extra,omitempty\" json:\"extra,omitzero\"`",
		"Tagged Tagged `xml:\"tagged,omitempty\" json:\"tagged,omitzero\"`",
		"Node *Node `xml:\"node,omitempty\" json:\"node,omitempty\"`",
		"Last string `xml:\"last\" json:\"last,omitempty\"`",
	}, fieldLines(t, code, "Customer"))

	assert.Equal(t, []string{
		"Level int32 `xml:\"level\" json:\"level,omitempty\"`",
	}, fieldLines(t, code, "CustomerExtraType"))

	assert.Equal(t, []string{
		"Base",
		"ExtFirst string `xml:\"extFirst\" json:\"extFirst,omitempty\"`",
		"Tag []string `xml:\"tag,omitempty\" json:\"tag,omitempty\"`",
		"ExtLast string `xml:\"extLast\" json:\"extLast,omitempty\"`",
	}, fieldLines(t, code, "Tagged"))

	assert.Equal(t, []string{
		"Label string `xml:\"label\" json:\"label,omitempty\"`",
		"Next *Node `xml:\"next,omitempty\" json:\"next,omitempty\"`",
	}, fieldLines(t, code, "Node"), "the required group element closing the cycle becomes a pointer")

	assert.Equal(t, []string{
		"Box UsesSharedBoxType `xml:\"box,omitempty\" json:\"box,omitzero\"`",
	}, fieldLines(t, code, "UsesShared"), "each use of a group gets its own anonymous types")
	assert.Equal(t, []string{
		"Box *AlsoUsesSharedBoxType `xml:\"box,omitempty\" json:\"box,omitempty\"`",
	}, fieldLines(t, code, "AlsoUsesShared"))
	assert.Equal(t, []string{
		"Width int32 `xml:\"width\" json:\"width,omitempty\"`",
	}, fieldLines(t, code, "UsesSharedBoxType"))

	assert.Equal(t, []string{
		"XMLName xml.Name `xml:\"urn:groups CustomerResponse\"`",
		"Email string `xml:\"email\" json:\"email,omitempty\"`",
		"Phone string `xml:\"phone,omitempty\" json:\"phone,omitempty\"`",
		"Address Address `xml:\"address,omitempty\" json:\"address,omitzero\"`",
	}, fieldLines(t, code, "CustomerResponse"))
}

// groupRefsClientTest sends a Customer through the generated client, checks
// that the group elements are sent in schema order and that optional ones
// left unset are not sent, and decodes them from the response.
const groupRefsClientTest = `package groups

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/kastrul/gowsdl/soap"
)

func call(t *testing.T, req *Customer) (string, *CustomerResponse) {
	t.Helper()
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = io.WriteString(w, ` + "`" + `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><CustomerResponse xmlns="urn:groups"><email>e@x</email><phone>555</phone><address><city>Tartu</city></address></CustomerResponse></soap:Body></soap:Envelope>` + "`" + `)
	}))
	defer server.Close()

	resp, err := NewCustomerPort(soap.NewClient(server.URL)).SaveCustomer(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	return body, resp
}

func children(t *testing.T, body, parent string) []string {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(body))
	var names []string
	depth := -1
	for {
		tok, err := dec.Token()
		if err != nil {
			return names
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			if depth >= 0 {
				depth++
				if depth == 1 {
					names = append(names, tok.Name.Local)
				}
			} else if tok.Name.Local == parent {
				depth = 0
			}
		case xml.EndElement:
			if depth == 0 {
				return names
			}
			if depth > 0 {
				depth--
			}
		}
	}
}

func TestGroupElementsAreSentInSchemaOrder(t *testing.T) {
	body, _ := call(t, &Customer{
		Id: 1, Email: "e@x", Phone: "555", Address: Address{City: "Tartu"},
		Card: &Card{Number: "4111"}, BackupEmail: "b@x", BackupAddress: &Address{City: "Tallinn"},
		CreatedBy: "me", StampedAt: 9, Extra: CustomerExtraType{Level: 2},
		Tagged: Tagged{Base: Base{BaseId: 3}, ExtFirst: "f", Tag: []string{"t1", "t2"}, ExtLast: "l"},
		Last: "z",
	})
	want := []string{"id", "email", "phone", "address", "card", "backupEmail", "backupAddress", "createdBy", "stampedAt", "extra", "tagged", "last"}
	if got := children(t, body, "Customer"); !reflect.DeepEqual(got, want) {
		t.Errorf("Customer children\ngot  %v\nwant %v\n%s", got, want, body)
	}
	want = []string{"baseId", "extFirst", "tag", "tag", "extLast"}
	if got := children(t, body, "tagged"); !reflect.DeepEqual(got, want) {
		t.Errorf("tagged children\ngot  %v\nwant %v\n%s", got, want, body)
	}
}

func TestOptionalGroupElementsAreOmitted(t *testing.T) {
	body, _ := call(t, &Customer{Id: 1, Iban: "EE00", Anonymous: true})
	want := []string{"id", "email", "address", "iban", "anonymous", "extra", "tagged", "last"}
	if got := children(t, body, "Customer"); !reflect.DeepEqual(got, want) {
		t.Errorf("Customer children\ngot  %v\nwant %v\n%s", got, want, body)
	}
}

func TestGroupElementsAreDecoded(t *testing.T) {
	_, resp := call(t, &Customer{})
	if resp.Email != "e@x" || resp.Phone != "555" || resp.Address.City != "Tartu" {
		t.Errorf("decoded %+v", resp)
	}
}
`

func TestGenerator_GroupRefs_ClientRoundTrip(t *testing.T) {
	out := generate(t, writeWSDL(t, groupRefsWSDL), WithPackage("groups"))
	runGeneratedModuleTests(t, "groups", out, map[string]string{"groups_test.go": groupRefsClientTest})
}

func TestGenerator_GroupRefs_ServerAndGenericsBuild(t *testing.T) {
	assertServerAndGenericsBuild(t, writeWSDL(t, groupRefsWSDL), "groups")
}

// TestGenerator_GroupRefs_TypePruning checks that a type used only by an
// element of a referenced group is kept when types are pruned.
func TestGenerator_GroupRefs_TypePruning(t *testing.T) {
	out := generate(t, writeWSDL(t, groupRefsWSDL), WithPackage("groups"), WithOperations("SaveCustomer"))
	types := strings.Join(out.typeNames(t), " ")
	for _, name := range []string{"Card", "Address", "Node", "CustomerExtraType"} {
		assert.Contains(t, " "+types+" ", " "+name+" ")
	}
}

func TestExpandGroupRefs_ComposesCompositors(t *testing.T) {
	const src = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:tns="urn:g" targetNamespace="urn:g">
  <xs:group name="Inner"><xs:sequence><xs:element name="i"/></xs:sequence></xs:group>
  <xs:group name="Pair">
    <xs:sequence>
      <xs:element name="x"/>
      <xs:group ref="tns:Inner" maxOccurs="2"/>
    </xs:sequence>
  </xs:group>
  <xs:complexType name="T">
    <xs:sequence>
      <xs:element name="id"/>
      <xs:group ref="tns:Pair" minOccurs="0" maxOccurs="unbounded"/>
      <xs:choice><xs:group ref="tns:Pair"/></xs:choice>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`
	var s parser.XSDSchema
	require.NoError(t, xml.Unmarshal([]byte(src), &s))
	expandGroupRefs([]*parser.XSDSchema{&s})

	ct := s.ComplexTypes[0]
	var got []string
	for _, el := range contentElements(ct.Sequence, ct.Choice, ct.SequenceChoice) {
		got = append(got, el.Name+" "+el.Compositors.String())
	}
	assert.Equal(t, []string{
		"id sequence:1::",
		"x sequence:1:: sequence:3:0:unbounded",
		"i sequence:1:: sequence:3:0:unbounded sequence:4::2",
		"x sequence:1:: choice:2:: sequence:5::",
		"i sequence:1:: choice:2:: sequence:5:: sequence:6::2",
	}, got, "each reference gets its own IDs, and the definition's model group takes its occurrence")
	assert.Equal(t, "Pair", contentElements(ct.Sequence)[1].Compositors[1].Group)
	assert.Equal(t, "Inner", contentElements(ct.Sequence)[2].Compositors[2].Group)
}
