// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package parser

import (
	"encoding/xml"
	"os"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	data, err := os.ReadFile("fixtures/vim.wsdl")
	if err != nil {
		t.Errorf("incorrect result\ngot:  %#v\nwant: %#v", err, nil)
	}

	v := WSDL{}
	err = xml.Unmarshal(data, &v)
	if err != nil {
		t.Errorf("incorrect result\ngot:  %#v\nwant: %#v", err, nil)
	}
}

func TestUnmarshal_BindingKinds(t *testing.T) {
	data := []byte(`<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"
		xmlns:wsdlsoap="http://schemas.xmlsoap.org/wsdl/soap/"
		xmlns:soap12="http://schemas.xmlsoap.org/wsdl/soap12/"
		xmlns:http="http://schemas.xmlsoap.org/wsdl/http/">
		<wsdl:binding name="Soap11" type="tns:PT"><wsdlsoap:binding style="rpc" transport="http://schemas.xmlsoap.org/soap/http"/></wsdl:binding>
		<wsdl:binding name="Soap12" type="tns:PT"><soap12:binding transport="http://schemas.xmlsoap.org/soap/http"/></wsdl:binding>
		<wsdl:binding name="HttpGet" type="tns:PTHttp"><http:binding verb="GET"/></wsdl:binding>
		<wsdl:binding name="Bare" type="tns:PT"/>
	</wsdl:definitions>`)

	var w WSDL
	if err := xml.Unmarshal(data, &w); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(w.Binding) != 4 {
		t.Fatalf("got %d bindings, want 4", len(w.Binding))
	}

	soap11, soap12, httpGet, bare := w.Binding[0], w.Binding[1], w.Binding[2], w.Binding[3]
	if soap11.SOAPBinding.Style != "rpc" || soap11.SOAP12Binding != nil || soap11.HTTPBinding != nil || soap11.IsHTTPOnly() {
		t.Errorf("SOAP 1.1 binding parsed incorrectly: %+v", soap11)
	}
	if soap12.SOAP12Binding == nil || soap12.SOAP12Binding.Transport != "http://schemas.xmlsoap.org/soap/http" || soap12.IsHTTPOnly() {
		t.Errorf("SOAP 1.2 binding parsed incorrectly: %+v", soap12)
	}
	if httpGet.HTTPBinding == nil || httpGet.HTTPBinding.Verb != "GET" || !httpGet.IsHTTPOnly() {
		t.Errorf("HTTP binding parsed incorrectly: %+v", httpGet)
	}
	if bare.IsHTTPOnly() {
		t.Errorf("binding without extensions must not be HTTP-only: %+v", bare)
	}
}
