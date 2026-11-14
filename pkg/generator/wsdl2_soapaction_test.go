package generator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const wsdl2SOAPActionTemplate = `<?xml version="1.0" encoding="UTF-8"?>
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
          <xs:sequence>
            <xs:element name="symbol" type="xs:string"/>
          </xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:element name="GetQuoteResponse">
        <xs:complexType>
          <xs:sequence>
            <xs:element name="price" type="xs:decimal"/>
          </xs:sequence>
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
  <binding name="StockQuoteSoapBinding"
           interface="tns:StockQuoteInterface"
           type="http://www.w3.org/ns/wsdl/soap"
           wsoap:protocol="http://www.w3.org/2003/05/soap/bindings/HTTP/"
           wsoap:mepDefault="http://www.w3.org/2003/05/soap/mep/request-response">
    <operation ref="tns:GetQuote" %s/>
  </binding>
  <service name="StockQuoteService" interface="tns:StockQuoteInterface">
    <endpoint name="StockQuoteEndpoint" binding="tns:StockQuoteSoapBinding"
              address="http://example.com/stockquote"/>
  </service>
</description>`

func generateWSDL2Operations(t *testing.T, bindingOpAttrs string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "stockquote.wsdl")
	content := fmt.Sprintf(wsdl2SOAPActionTemplate, bindingOpAttrs)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	g, err := New(path, WithPackage("stockquote"))
	require.NoError(t, err)

	files, err := g.Generate(context.Background())
	require.NoError(t, err)

	ops, ok := files["operations"]
	require.True(t, ok, "operations file should be generated")
	return string(ops)
}

func TestGenerator_WSDL20_SOAPActionInOperations(t *testing.T) {
	tests := []struct {
		name     string
		attrs    string
		wantCall string
	}{
		{
			name:     "spec wsoap:action",
			attrs:    `wsoap:action="http://example.com/GetQuote" wsoap:mep="http://www.w3.org/2003/05/soap/mep/request-response"`,
			wantCall: `CallContext(ctx, "http://example.com/GetQuote", request, response)`,
		},
		{
			name:     "legacy wsoap:soapAction",
			attrs:    `wsoap:soapAction="http://example.com/GetQuoteLegacy"`,
			wantCall: `CallContext(ctx, "http://example.com/GetQuoteLegacy", request, response)`,
		},
		{
			name:     "spec name wins over legacy name",
			attrs:    `wsoap:action="http://example.com/GetQuote" wsoap:soapAction="http://example.com/GetQuoteLegacy"`,
			wantCall: `CallContext(ctx, "http://example.com/GetQuote", request, response)`,
		},
		{
			name:     "no action",
			attrs:    ``,
			wantCall: `CallContext(ctx, "", request, response)`,
		},
		{
			name:     "action is escaped as a Go string literal",
			attrs:    `wsoap:action="urn:a&quot;b\c"`,
			wantCall: `CallContext(ctx, "urn:a\"b\\c", request, response)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops := generateWSDL2Operations(t, tt.attrs)
			assert.Contains(t, ops, tt.wantCall)
		})
	}
}

func TestGenerator_findWSDL2SOAPAction(t *testing.T) {
	g := &Generator{
		wsdl2: &parser.WSDL2{
			Bindings: []*parser.WSDL2Binding{
				{
					Name:      "OtherBinding",
					Interface: "tns:OtherInterface",
					Operations: []*parser.WSDL2BindingOperation{
						{Ref: "tns:GetQuote", Action: "http://example.com/Other"},
					},
				},
				{
					Name:      "HTTPBinding",
					Interface: "tns:StockQuoteInterface",
					Operations: []*parser.WSDL2BindingOperation{
						{Ref: "tns:GetQuote"},
					},
				},
				{
					Name:      "SOAPBinding",
					Interface: "tns:StockQuoteInterface",
					Operations: []*parser.WSDL2BindingOperation{
						{Ref: "tns:GetQuote", Action: "http://example.com/GetQuote"},
						{Ref: "GetPrice", SOAPAction: "http://example.com/GetPrice"},
					},
				},
			},
		},
	}

	assert.Equal(t, "http://example.com/GetQuote", g.findWSDL2SOAPAction("GetQuote", "StockQuoteInterface"))
	assert.Equal(t, "http://example.com/GetPrice", g.findWSDL2SOAPAction("GetPrice", "StockQuoteInterface"))
	assert.Equal(t, "http://example.com/Other", g.findWSDL2SOAPAction("GetQuote", "OtherInterface"))
	assert.Empty(t, g.findWSDL2SOAPAction("Missing", "StockQuoteInterface"))
	assert.Empty(t, g.findWSDL2SOAPAction("GetQuote", "MissingInterface"))
	assert.Empty(t, (&Generator{}).findWSDL2SOAPAction("GetQuote", "StockQuoteInterface"))
}
