package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeGeneratedModule writes the generator output and any extra files into a
// temporary Go module that depends on this repository and returns its directory.
func writeGeneratedModule(t *testing.T, module string, out map[string][]byte, extra map[string]string) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)

	dir := t.TempDir()
	client := string(out["header"]) + string(out["types"]) + string(out["operations"])
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client.go"), []byte(client), 0o600))
	if len(out["server"]) > 0 {
		server := string(out["server_header"]) + string(out["server_wsdl"]) + string(out["server"])
		require.NoError(t, os.WriteFile(filepath.Join(dir, "server.go"), []byte(server), 0o600))
	}
	for name, content := range extra {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}

	goMod := "module " + module + "\n\ngo 1.21\n\nrequire github.com/kastrul/gowsdl v0.0.0\n\nreplace github.com/kastrul/gowsdl => " + root + "\n"
	goSum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.sum"), goSum, 0o600))
	return dir
}

// runGeneratedModuleTests runs go test inside a module built by writeGeneratedModule.
func runGeneratedModuleTests(t *testing.T, module string, out map[string][]byte, extra map[string]string) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping go test of generated code in short mode")
	}

	dir := writeGeneratedModule(t, module, out, extra)
	cmd := exec.Command("go", "test", "-mod=mod", "-count=1", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "tests against generated code failed:\n%s", output)
}

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
`

func TestGenerator_WSDL2Server_DispatchesValidRequest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quote.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(wsdl2StockQuote), 0o600))

	_, out := generateCode(t, path, WithPackage("quote"), WithServerGeneration(true))
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
