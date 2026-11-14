package parser

import (
	"encoding/xml"
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
