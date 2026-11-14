// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package parser

import (
	"encoding/xml"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const wsdl2SOAPBindingDoc = `<?xml version="1.0" encoding="UTF-8"?>
<description xmlns="http://www.w3.org/ns/wsdl/"
             xmlns:tns="http://example.com/stockquote"
             xmlns:wsoap="http://www.w3.org/ns/wsdl/soap"
             targetNamespace="http://example.com/stockquote">
  <interface name="StockQuoteInterface">
    <operation name="GetQuote" pattern="http://www.w3.org/ns/wsdl/in-out"/>
    <operation name="GetLegacy" pattern="http://www.w3.org/ns/wsdl/in-out"/>
    <operation name="GetBoth" pattern="http://www.w3.org/ns/wsdl/in-out"/>
    <operation name="GetNone" pattern="http://www.w3.org/ns/wsdl/in-out"/>
  </interface>
  <binding name="StockQuoteSoapBinding"
           interface="tns:StockQuoteInterface"
           type="http://www.w3.org/ns/wsdl/soap"
           wsoap:protocol="http://www.w3.org/2003/05/soap/bindings/HTTP/"
           wsoap:mepDefault="http://www.w3.org/2003/05/soap/mep/request-response">
    <operation ref="tns:GetQuote"
               wsoap:action="http://example.com/GetQuote"
               wsoap:mep="http://www.w3.org/2003/05/soap/mep/soap-response"/>
    <operation ref="tns:GetLegacy"
               wsoap:soapAction="http://example.com/GetLegacy"
               wsoap:mepDefault="http://example.com/legacy-mep"/>
    <operation ref="tns:GetBoth"
               wsoap:action="http://example.com/GetBoth"
               wsoap:soapAction="http://example.com/GetBothLegacy"
               wsoap:mep="http://example.com/spec-mep"
               wsoap:mepDefault="http://example.com/legacy-mep"/>
    <operation ref="tns:GetNone"/>
  </binding>
</description>`

func TestParser_WSDL20_SOAPBindingAttributes(t *testing.T) {
	var w WSDL2
	require.NoError(t, xml.Unmarshal([]byte(wsdl2SOAPBindingDoc), &w))
	require.Len(t, w.Bindings, 1)

	binding := w.Bindings[0]
	assert.Equal(t, "http://www.w3.org/2003/05/soap/mep/request-response", binding.MEPDefault)
	require.Len(t, binding.Operations, 4)

	ops := make(map[string]*WSDL2BindingOperation, len(binding.Operations))
	for _, op := range binding.Operations {
		ops[op.Ref] = op
	}

	tests := []struct {
		ref            string
		action         string
		mep            string
		legacyAction   string
		legacyMEP      string
		resolvedAction string
		resolvedMEP    string
	}{
		{
			ref:            "tns:GetQuote",
			action:         "http://example.com/GetQuote",
			mep:            "http://www.w3.org/2003/05/soap/mep/soap-response",
			resolvedAction: "http://example.com/GetQuote",
			resolvedMEP:    "http://www.w3.org/2003/05/soap/mep/soap-response",
		},
		{
			ref:            "tns:GetLegacy",
			legacyAction:   "http://example.com/GetLegacy",
			legacyMEP:      "http://example.com/legacy-mep",
			resolvedAction: "http://example.com/GetLegacy",
			resolvedMEP:    "http://example.com/legacy-mep",
		},
		{
			ref:            "tns:GetBoth",
			action:         "http://example.com/GetBoth",
			mep:            "http://example.com/spec-mep",
			legacyAction:   "http://example.com/GetBothLegacy",
			legacyMEP:      "http://example.com/legacy-mep",
			resolvedAction: "http://example.com/GetBoth",
			resolvedMEP:    "http://example.com/spec-mep",
		},
		{
			ref: "tns:GetNone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			op, ok := ops[tt.ref]
			require.True(t, ok, "binding operation %s not parsed", tt.ref)
			assert.Equal(t, tt.action, op.Action)
			assert.Equal(t, tt.mep, op.MEP)
			assert.Equal(t, tt.legacyAction, op.SOAPAction)
			assert.Equal(t, tt.legacyMEP, op.SOAPMEPDefault)
			assert.Equal(t, tt.resolvedAction, op.ResolvedSOAPAction())
			assert.Equal(t, tt.resolvedMEP, op.ResolvedSOAPMEP())
		})
	}
}

const wsdl2TestDocument = `<?xml version="1.0" encoding="UTF-8"?>
<wsdl:description xmlns:wsdl="%s"
                  xmlns:tns="http://example.com/stockquote"
                  xmlns:wsoap="http://www.w3.org/ns/wsdl/soap"
                  targetNamespace="http://example.com/stockquote"
                  name="StockQuote">
  <wsdl:documentation>Stock quote service</wsdl:documentation>
  <wsdl:types>
    <xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
               targetNamespace="http://example.com/stockquote">
      <xs:element name="GetQuoteRequest" type="xs:string"/>
      <xs:element name="GetQuoteResponse" type="xs:decimal"/>
    </xs:schema>
  </wsdl:types>
  <wsdl:interface name="StockQuoteInterface">
    <wsdl:operation name="GetQuote" pattern="http://www.w3.org/ns/wsdl/in-out">
      <wsdl:input element="tns:GetQuoteRequest"/>
      <wsdl:output element="tns:GetQuoteResponse"/>
    </wsdl:operation>
  </wsdl:interface>
  <wsdl:binding name="StockQuoteSoapBinding" interface="tns:StockQuoteInterface"
                type="http://www.w3.org/ns/wsdl/soap">
    <wsdl:operation ref="tns:GetQuote"/>
  </wsdl:binding>
  <wsdl:service name="StockQuoteService" interface="tns:StockQuoteInterface">
    <wsdl:endpoint name="StockQuoteEndpoint" binding="tns:StockQuoteSoapBinding"
                   address="http://example.com/stockquote"/>
  </wsdl:service>
</wsdl:description>`

func TestWSDL2_Unmarshal_Namespaces(t *testing.T) {
	namespaces := map[string]string{
		"spec":   "http://www.w3.org/ns/wsdl",
		"legacy": "http://www.w3.org/ns/wsdl/",
	}

	for name, ns := range namespaces {
		t.Run(name, func(t *testing.T) {
			data := []byte(fmt.Sprintf(wsdl2TestDocument, ns))

			version, err := DetectWSDLVersion(data)
			require.NoError(t, err)
			assert.Equal(t, "2.0", version)

			var w WSDL2
			require.NoError(t, xml.Unmarshal(data, &w))

			assert.Equal(t, "StockQuote", w.Name)
			assert.Equal(t, "http://example.com/stockquote", w.TargetNamespace)
			assert.Equal(t, "Stock quote service", w.Doc)
			assert.Equal(t, ns, w.Xmlns["wsdl"])

			require.Len(t, w.Types.Schemas, 1)
			assert.Len(t, w.Types.Schemas[0].Elements, 2)
			assert.Equal(t, ns, w.Types.Schemas[0].Xmlns["wsdl"])

			require.Len(t, w.Interfaces, 1)
			assert.Equal(t, "StockQuoteInterface", w.Interfaces[0].Name)
			require.Len(t, w.Interfaces[0].Operations, 1)
			op := w.Interfaces[0].Operations[0]
			assert.Equal(t, "GetQuote", op.Name)
			require.NotNil(t, op.Input)
			assert.Equal(t, "tns:GetQuoteRequest", op.Input.Element)
			require.NotNil(t, op.Output)
			assert.Equal(t, "tns:GetQuoteResponse", op.Output.Element)

			require.Len(t, w.Bindings, 1)
			assert.Equal(t, "StockQuoteSoapBinding", w.Bindings[0].Name)
			require.Len(t, w.Bindings[0].Operations, 1)
			assert.Equal(t, "tns:GetQuote", w.Bindings[0].Operations[0].Ref)

			require.Len(t, w.Services, 1)
			assert.Equal(t, "StockQuoteService", w.Services[0].Name)
			require.Len(t, w.Services[0].Endpoints, 1)
			assert.Equal(t, "http://example.com/stockquote", w.Services[0].Endpoints[0].Address)
		})
	}
}

func TestWSDL2_Unmarshal_IgnoresForeignNamespace(t *testing.T) {
	data := []byte(`<description xmlns="http://www.w3.org/ns/wsdl" xmlns:other="http://example.com/other">
  <other:interface name="NotWSDL"/>
  <interface name="Real"/>
</description>`)

	var w WSDL2
	require.NoError(t, xml.Unmarshal(data, &w))
	require.Len(t, w.Interfaces, 1)
	assert.Equal(t, "Real", w.Interfaces[0].Name)
}

func TestIsWSDL2Namespace(t *testing.T) {
	assert.True(t, isWSDL2Namespace("http://www.w3.org/ns/wsdl"))
	assert.True(t, isWSDL2Namespace("http://www.w3.org/ns/wsdl/"))
	assert.False(t, isWSDL2Namespace("http://www.w3.org/ns/wsdl/soap"))
	assert.False(t, isWSDL2Namespace("http://schemas.xmlsoap.org/wsdl/"))
	assert.False(t, isWSDL2Namespace(""))
}
