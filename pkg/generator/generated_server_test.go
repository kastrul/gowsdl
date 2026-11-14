package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// replaceGeneratedStub swaps the body of a generated server stub so the server
// returns a real response instead of WSDLUndefinedError.
func replaceGeneratedStub(t *testing.T, out map[string][]byte, signature, body string) {
	t.Helper()

	server := string(out["server"])
	stub := signature + " {\n\treturn nil, WSDLUndefinedError\n}"
	require.Contains(t, server, stub, "generated server stub not found")
	out["server"] = []byte(strings.Replace(server, stub, signature+" {\n"+body+"\n}", 1))
}

const wsdl2ServerDispatchTest = `package quote

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEndpointDispatchesRequest(t *testing.T) {
	body := ` + "`" + `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><GetQuoteRequest xmlns="http://example.com/stockquote"><symbol>ACME</symbol></GetQuoteRequest></soap:Body></soap:Envelope>` + "`" + `
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	rec := httptest.NewRecorder()

	Endpoint(rec, req)

	got, _ := io.ReadAll(rec.Result().Body)
	if strings.Contains(string(got), "Fault") {
		t.Fatalf("valid request produced a fault: %s", got)
	}
	if !strings.Contains(string(got), "<price>42.5</price>") {
		t.Fatalf("response does not carry the handler result: %s", got)
	}
}

func TestEndpointFaultsOnUnknownOperation(t *testing.T) {
	body := ` + "`" + `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><NoSuchRequest xmlns="http://example.com/stockquote"/></soap:Body></soap:Envelope>` + "`" + `
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	rec := httptest.NewRecorder()

	Endpoint(rec, req)

	got, _ := io.ReadAll(rec.Result().Body)
	if !strings.Contains(string(got), "<faultcode>soap:Server</faultcode>") ||
		!strings.Contains(string(got), "Server was unable to process request.") {
		t.Fatalf("unknown operation should produce a server fault: %s", got)
	}
}
`

func TestGenerator_WSDL2Server_DispatchesValidRequest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quote.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(wsdl2StockQuote), 0o600))

	out := generate(t, path, WithPackage("quote"), WithServerGeneration(true))
	replaceGeneratedStub(t, out,
		"func (service *SOAPBodyRequest) GetQuoteRequestFunc(request *GetQuoteRequest) (*GetQuoteResponse, error)",
		"\treturn &GetQuoteResponse{Price: 42.5}, nil")

	runGeneratedModuleTests(t, "quote", out, map[string]string{"server_test.go": wsdl2ServerDispatchTest})
}

const wsdl2StockQuote = `<?xml version="1.0" encoding="UTF-8"?>
<description xmlns="http://www.w3.org/ns/wsdl/"
             xmlns:tns="http://example.com/stockquote"
             xmlns:wsoap="http://www.w3.org/ns/wsdl/soap"
             targetNamespace="http://example.com/stockquote">
  <types>
    <xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
               targetNamespace="http://example.com/stockquote"
               elementFormDefault="qualified">
      <xs:element name="GetQuoteRequest">
        <xs:complexType>
          <xs:sequence><xs:element name="symbol" type="xs:string"/></xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:element name="GetQuoteResponse">
        <xs:complexType>
          <xs:sequence><xs:element name="price" type="xs:decimal"/></xs:sequence>
        </xs:complexType>
      </xs:element>
    </xs:schema>
  </types>
  <interface name="StockQuoteInterface">
    <operation name="GetQuote" pattern="http://www.w3.org/ns/wsdl/in-out">
      <input element="tns:GetQuoteRequest"/>
      <output element="tns:GetQuoteResponse"/>
    </operation>
  </interface>
  <binding name="StockQuoteSoapBinding" interface="tns:StockQuoteInterface"
           type="http://www.w3.org/ns/wsdl/soap"
           wsoap:protocol="http://schemas.xmlsoap.org/soap/http">
    <operation ref="tns:GetQuote"/>
  </binding>
  <service name="StockQuoteService" interface="tns:StockQuoteInterface">
    <endpoint name="StockQuoteEndpoint" binding="tns:StockQuoteSoapBinding"
              address="http://example.com/stockquote"/>
  </service>
</description>`

const wsdl11ServerContentTypeTest = `package stock

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEndpointRejectsWrongContentTypeOnce(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	Endpoint(rec, req)

	got, _ := io.ReadAll(rec.Result().Body)
	if n := strings.Count(string(got), "<soap:Envelope"); n != 1 {
		t.Fatalf("expected exactly one envelope, got %d: %s", n, got)
	}
	if !strings.Contains(string(got), "<faultcode>soap:Client</faultcode>") {
		t.Fatalf("expected a client fault: %s", got)
	}
}
`

func TestGenerator_WSDL11Server_WrongContentTypeWritesOneEnvelope(t *testing.T) {
	out := generate(t, filepath.Join("..", "..", "fixtures", "stock.wsdl"),
		WithPackage("stock"), WithServerGeneration(true))

	runGeneratedModuleTests(t, "stock", out, map[string]string{"server_test.go": wsdl11ServerContentTypeTest})
}
