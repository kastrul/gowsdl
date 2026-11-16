// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// extensionChoiceWSDL declares complexContent extensions whose own content
// holds a choice: at the top of the extension, inside its sequence, nested
// in further sequences and choices, and with an inline complex type as a
// branch. ModelsRequest is the anonymous type of a global element,
// ModelsResult a named type, and Holder's item an inline named type, so all
// the template sites that render an extension are covered.
const extensionChoiceWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="urn:extchoice"
             xmlns:xs="http://www.w3.org/2001/XMLSchema"
             targetNamespace="urn:extchoice">
  <types>
    <xs:schema targetNamespace="urn:extchoice" elementFormDefault="qualified">
      <xs:complexType name="BaseRequest">
        <xs:sequence><xs:element name="locale" type="xs:string"/></xs:sequence>
      </xs:complexType>
      <xs:complexType name="BaseResult">
        <xs:sequence><xs:element name="status" type="xs:string"/></xs:sequence>
      </xs:complexType>
      <xs:element name="ModelsRequest">
        <xs:complexType>
          <xs:complexContent>
            <xs:extension base="tns:BaseRequest">
              <xs:sequence>
                <xs:element name="modelYear" type="xs:int"/>
                <xs:choice>
                  <xs:element name="divisionId" type="xs:int"/>
                  <xs:element name="subdivisionId" type="xs:int"/>
                </xs:choice>
              </xs:sequence>
            </xs:extension>
          </xs:complexContent>
        </xs:complexType>
      </xs:element>
      <xs:element name="ModelsResponse">
        <xs:complexType>
          <xs:sequence><xs:element name="result" type="tns:ModelsResult"/></xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:complexType name="ModelsResult">
        <xs:complexContent>
          <xs:extension base="tns:BaseResult">
            <xs:choice>
              <xs:element name="modelId" type="xs:int"/>
              <xs:element name="message" type="xs:string"/>
              <xs:element name="detail">
                <xs:complexType>
                  <xs:sequence>
                    <xs:element name="code" type="xs:int"/>
                    <xs:element name="text" type="xs:string"/>
                  </xs:sequence>
                </xs:complexType>
              </xs:element>
            </xs:choice>
          </xs:extension>
        </xs:complexContent>
      </xs:complexType>
      <xs:complexType name="Holder">
        <xs:sequence>
          <xs:element name="item">
            <xs:complexType>
              <xs:complexContent>
                <xs:extension base="tns:BaseResult">
                  <xs:choice>
                    <xs:element name="byName" type="xs:string"/>
                    <xs:element name="byId" type="xs:int"/>
                    <xs:element name="nested">
                      <xs:complexType>
                        <xs:sequence><xs:element name="n" type="xs:int"/></xs:sequence>
                      </xs:complexType>
                    </xs:element>
                  </xs:choice>
                </xs:extension>
              </xs:complexContent>
            </xs:complexType>
          </xs:element>
        </xs:sequence>
      </xs:complexType>
      <xs:complexType name="Described">
        <xs:complexContent>
          <xs:extension base="tns:BaseRequest">
            <xs:sequence>
              <xs:choice>
                <xs:sequence>
                  <xs:element name="year" type="xs:int"/>
                  <xs:element name="make" type="xs:string"/>
                </xs:sequence>
                <xs:element name="styleId" type="xs:int"/>
              </xs:choice>
              <xs:sequence>
                <xs:element name="trim" type="xs:int"/>
              </xs:sequence>
            </xs:sequence>
          </xs:extension>
        </xs:complexContent>
      </xs:complexType>
    </xs:schema>
  </types>
  <message name="ModelsIn"><part name="parameters" element="tns:ModelsRequest"/></message>
  <message name="ModelsOut"><part name="parameters" element="tns:ModelsResponse"/></message>
  <portType name="ModelsPort">
    <operation name="GetModels"><input message="tns:ModelsIn"/><output message="tns:ModelsOut"/></operation>
  </portType>
  <binding name="ModelsBinding" type="tns:ModelsPort">
    <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
    <operation name="GetModels">
      <soap:operation soapAction="urn:extchoice:GetModels"/>
      <input><soap:body use="literal"/></input>
      <output><soap:body use="literal"/></output>
    </operation>
  </binding>
  <service name="ModelsService">
    <port name="ModelsPort" binding="tns:ModelsBinding"><soap:address location="http://localhost/"/></port>
  </service>
</definitions>`

func TestGenerator_ExtensionSequenceChoice_Fields(t *testing.T) {
	code := generate(t, writeWSDL(t, extensionChoiceWSDL), WithPackage("extchoice")).client(t)

	assert.Equal(t, []string{"XMLName", "ModelYear", "DivisionId", "SubdivisionId"}, structFieldNames(t, code, "ModelsRequest"))
	assert.Equal(t, map[string]string{
		"ModelYear":     "`xml:\"modelYear\" json:\"modelYear,omitempty\"`",
		"DivisionId":    "`xml:\"divisionId,omitempty\" json:\"divisionId,omitempty\"`",
		"SubdivisionId": "`xml:\"subdivisionId,omitempty\" json:\"subdivisionId,omitempty\"`",
	}, without(fieldTags(t, code, "ModelsRequest"), "XMLName"))
	assert.Regexp(t, `\n BaseRequest\n`, structBody(t, code, "ModelsRequest"))
}

func TestGenerator_ExtensionChoice_Fields(t *testing.T) {
	code := generate(t, writeWSDL(t, extensionChoiceWSDL), WithPackage("extchoice")).client(t)

	assert.Equal(t, []string{"ModelId", "Message", "Detail"}, structFieldNames(t, code, "ModelsResult"))
	tags := fieldTags(t, code, "ModelsResult")
	assert.Equal(t, "`xml:\"modelId,omitempty\" json:\"modelId,omitempty\"`", tags["ModelId"])
	assert.Equal(t, "`xml:\"message,omitempty\" json:\"message,omitempty\"`", tags["Message"])
	assert.Equal(t, "`xml:\"detail,omitempty\" json:\"detail,omitempty\"`", tags["Detail"])
	assert.Regexp(t, `\n Detail \*ModelsResultDetailType `, structBody(t, code, "ModelsResult"))
}

func TestGenerator_ExtensionChoice_InlineComplexTypes(t *testing.T) {
	out := generate(t, writeWSDL(t, extensionChoiceWSDL), WithPackage("extchoice"))
	code := out.client(t)

	assert.Equal(t, []string{"Code", "Text"}, structFieldNames(t, code, "ModelsResultDetailType"))

	assert.Equal(t, []string{"ByName", "ById", "Nested"}, structFieldNames(t, code, "HolderItemType"))
	assert.Regexp(t, `^ BaseResult\n`, structBody(t, code, "HolderItemType"))
	assert.Regexp(t, `\n Nested \*HolderItemTypeNestedType `, structBody(t, code, "HolderItemType"))
	assert.Equal(t, []string{"N"}, structFieldNames(t, code, "HolderItemTypeNestedType"))
	assert.Equal(t, "`xml:\"n\" json:\"n,omitempty\"`", fieldTags(t, code, "HolderItemTypeNestedType")["N"])
}

func TestGenerator_ExtensionNestedGroups_Fields(t *testing.T) {
	code := generate(t, writeWSDL(t, extensionChoiceWSDL), WithPackage("extchoice")).client(t)

	assert.ElementsMatch(t, []string{"Trim", "Year", "Make", "StyleId"}, structFieldNames(t, code, "Described"))
	assert.Equal(t, map[string]string{
		"Trim":    "`xml:\"trim\" json:\"trim,omitempty\"`",
		"Year":    "`xml:\"year,omitempty\" json:\"year,omitempty\"`",
		"Make":    "`xml:\"make,omitempty\" json:\"make,omitempty\"`",
		"StyleId": "`xml:\"styleId,omitempty\" json:\"styleId,omitempty\"`",
	}, without(fieldTags(t, code, "Described"), "BaseRequest"))
}

func TestGenerator_ChromeData_ExtensionChoiceFields(t *testing.T) {
	code := generate(t, chromeDataWSDL(), WithPackage("chromedata")).client(t)

	models := structFieldNames(t, code, "ModelsRequest")
	assert.Subset(t, models, []string{"ModelYear", "DivisionId", "SubdivisionId"})
	tags := fieldTags(t, code, "ModelsRequest")
	assert.Equal(t, "`xml:\"divisionId,omitempty\" json:\"divisionId,omitempty\"`", tags["DivisionId"])
	assert.Equal(t, "`xml:\"subdivisionId,omitempty\" json:\"subdivisionId,omitempty\"`", tags["SubdivisionId"])

	assert.Subset(t, structFieldNames(t, code, "VehicleDescriptionRequest"),
		[]string{"ModelYear", "MakeName", "ModelName", "Vin", "ReducingStyleId", "StyleId", "TrimName"})
}

// without returns m without the given keys.
func without(m map[string]string, keys ...string) map[string]string {
	for _, k := range keys {
		delete(m, k)
	}
	return m
}

// extensionChoiceClientTest calls GetModels through the generated client
// against a server that records the request body and answers with a result
// holding one branch of the extension's choice.
const extensionChoiceClientTest = `package extchoice

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kastrul/gowsdl/soap"
)

func call(t *testing.T, req *ModelsRequest, result string) (string, *ModelsResponse) {
	t.Helper()
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = io.WriteString(w, ` + "`" + `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><ModelsResponse xmlns="urn:extchoice"><result><status>ok</status>` + "`" + `+result+` + "`" + `</result></ModelsResponse></soap:Body></soap:Envelope>` + "`" + `)
	}))
	defer server.Close()

	resp, err := NewModelsPort(soap.NewClient(server.URL)).GetModels(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	return body, resp
}

func TestRequestSendsOnlyTheBranchSet(t *testing.T) {
	body, _ := call(t, &ModelsRequest{BaseRequest: BaseRequest{Locale: "en"}, ModelYear: 2020, DivisionId: 7}, "")
	for _, want := range []string{"<locale>en</locale>", "<modelYear>2020</modelYear>", "<divisionId>7</divisionId>"} {
		if !strings.Contains(body, want) {
			t.Errorf("request lacks %s:\n%s", want, body)
		}
	}
	if strings.Contains(body, "subdivisionId") {
		t.Errorf("request has the unchosen branch:\n%s", body)
	}

	body, _ = call(t, &ModelsRequest{ModelYear: 2021, SubdivisionId: 9}, "")
	if !strings.Contains(body, "<subdivisionId>9</subdivisionId>") || strings.Contains(body, "divisionId>0") {
		t.Errorf("request should hold only subdivisionId:\n%s", body)
	}
}

func TestResponseDecodesTheBranch(t *testing.T) {
	_, resp := call(t, &ModelsRequest{ModelYear: 2020, DivisionId: 1}, "<modelId>42</modelId>")
	if resp.Result.Status != "ok" || resp.Result.ModelId != 42 || resp.Result.Message != "" {
		t.Errorf("decoded %+v", resp.Result)
	}

	_, resp = call(t, &ModelsRequest{ModelYear: 2020, DivisionId: 1}, "<detail><code>3</code><text>gone</text></detail>")
	if resp.Result.Detail.Code != 3 || resp.Result.Detail.Text != "gone" || resp.Result.ModelId != 0 {
		t.Errorf("decoded %+v", resp.Result)
	}
}
`

func TestGenerator_ExtensionChoice_ClientRoundTrip(t *testing.T) {
	out := generate(t, writeWSDL(t, extensionChoiceWSDL), WithPackage("extchoice"))
	require.Contains(t, out.client(t), "func NewModelsPort(")
	runGeneratedModuleTests(t, "extchoice", out, map[string]string{"extchoice_test.go": extensionChoiceClientTest})
}
