// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package soap

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const typedEnvelope = `<Envelope xmlns:ns2="urn:item" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
	`<holder><value xsi:type="ns2:List"><entry xsi:type="ns2:Item"><id>7</id></entry></value></holder>` +
	`</Envelope>`

type typedEnvelopeDoc struct {
	Holder anyHolder `xml:"holder"`
}

func TestNewDecoder_ResolvesXSITypeDeclaredOnAncestor(t *testing.T) {
	var env typedEnvelopeDoc
	require.NoError(t, NewDecoder(strings.NewReader(typedEnvelope)).Decode(&env))

	typ, ok := env.Holder.Value.Type()
	require.True(t, ok)
	assert.Equal(t, xml.Name{Space: "urn:item", Local: "List"}, typ)

	var list struct {
		Entry AnyType `xml:"entry"`
	}
	require.NoError(t, env.Holder.Value.Decode(&list))
	typ, ok = list.Entry.Type()
	require.True(t, ok, "nested xsi:type must stay resolvable inside InnerXML: %s", env.Holder.Value.InnerXML)
	assert.Equal(t, xml.Name{Space: "urn:item", Local: "Item"}, typ)
}

func TestAnyType_TypeWithoutNewDecoder(t *testing.T) {
	var env typedEnvelopeDoc
	require.NoError(t, xml.Unmarshal([]byte(typedEnvelope), &env))
	_, ok := env.Holder.Value.Type()
	assert.False(t, ok, "xml.Unmarshal cannot see the envelope's declarations")

	_, ok = AnyType{}.Type()
	assert.False(t, ok, "no xsi:type")
}

func TestAnyType_TypeUnprefixed(t *testing.T) {
	a := AnyType{Attrs: []xml.Attr{
		{Name: xml.Name{Local: "xmlns"}, Value: "urn:d"},
		{Name: xml.Name{Space: XSINamespace, Local: "type"}, Value: "Item"},
	}}
	typ, ok := a.Type()
	require.True(t, ok)
	assert.Equal(t, xml.Name{Space: "urn:d", Local: "Item"}, typ)
}

func TestAnyType_ReencodedTypeStaysResolvable(t *testing.T) {
	var env typedEnvelopeDoc
	require.NoError(t, NewDecoder(strings.NewReader(typedEnvelope)).Decode(&env))

	out, err := xml.Marshal(env.Holder)
	require.NoError(t, err)

	var again anyHolder
	require.NoError(t, xml.Unmarshal(out, &again), "output: %s", out)
	typ, ok := again.Value.Type()
	require.True(t, ok, "output: %s", out)
	assert.Equal(t, xml.Name{Space: "urn:item", Local: "List"}, typ)
}

func TestDeclareXSITypePrefixes(t *testing.T) {
	const xsi = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`
	tests := []struct {
		name, in, want string
	}{
		{
			name: "declaration copied onto the typed element",
			in:   `<e xmlns:p="urn:p" ` + xsi + `><v xsi:type="p:T">x</v></e>`,
			want: `<e xmlns:p="urn:p" ` + xsi + `><v xsi:type="p:T" xmlns:p="urn:p">x</v></e>`,
		},
		{
			name: "self-closing element",
			in:   `<e xmlns:p="urn:p" ` + xsi + `><v xsi:type="p:T"/></e>`,
			want: `<e xmlns:p="urn:p" ` + xsi + `><v xsi:type="p:T" xmlns:p="urn:p"/></e>`,
		},
		{
			name: "prefix already declared on the element",
			in:   `<e xmlns:p="urn:p" ` + xsi + `><v xmlns:p="urn:q" xsi:type="p:T"/></e>`,
		},
		{
			name: "prefix declared on the same element as xsi:type",
			in:   `<v xmlns:p="urn:p" ` + xsi + ` xsi:type="p:T"/>`,
		},
		{
			name: "undeclared prefix is left alone",
			in:   `<e ` + xsi + `><v xsi:type="p:T"/></e>`,
		},
		{
			name: "attribute values other than xsi:type are left alone",
			in:   `<e xmlns:p="urn:p" ` + xsi + `><v other="p:T"/></e>`,
		},
		{
			name: "uri is escaped",
			in:   `<e xmlns:p="urn:a&amp;b" ` + xsi + `><v xsi:type="p:T"/></e>`,
			want: `<e xmlns:p="urn:a&amp;b" ` + xsi + `><v xsi:type="p:T" xmlns:p="urn:a&amp;b"/></e>`,
		},
		{
			name: "no xsi namespace",
			in:   `<e xmlns:p="urn:p"><v type="p:T"/></e>`,
		},
		{
			name: "malformed input is returned unchanged",
			in:   `<e xmlns:p="urn:p" ` + xsi + `><v xsi:type="p:T"></e>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := tt.want
			if want == "" {
				want = tt.in
			}
			assert.Equal(t, want, string(declareXSITypePrefixes([]byte(tt.in))))
		})
	}
}

func TestNewDecoder_KeepsInnerXML(t *testing.T) {
	var v struct {
		Inner string `xml:",innerxml"`
	}
	require.NoError(t, NewDecoder(strings.NewReader(`<a><b>1</b></a>`)).Decode(&v))
	assert.Equal(t, "<b>1</b>", v.Inner)
}

type failingReader struct {
	data string
	err  error
}

func (f *failingReader) Read(p []byte) (int, error) {
	if f.data == "" {
		return 0, f.err
	}
	n := copy(p, f.data)
	f.data = f.data[n:]
	return n, nil
}

func TestNewDecoder_ReportsReadError(t *testing.T) {
	boom := errors.New("connection reset")
	var v struct{}
	err := NewDecoder(&failingReader{data: "<a><b>", err: boom}).Decode(&v)
	assert.ErrorIs(t, err, boom)

	err = NewDecoder(&failingReader{data: "<a></a>", err: io.EOF}).Decode(&v)
	assert.NoError(t, err)
}

func TestClient_DecodesXSITypeFromEnvelopeDeclaration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"` +
			` xmlns:ns2="urn:item" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
			`<soap:Body><holder><value xsi:type="ns2:Item"><id>7</id></value></holder></soap:Body></soap:Envelope>`))
	}))
	defer server.Close()

	result := NewClient(server.URL).CallResult[anyHolder](context.Background(), "get", &anyHolder{})
	require.True(t, result.IsSuccess(), "unexpected fault: %+v", result.Fault)

	typ, ok := result.Value.Value.Type()
	require.True(t, ok)
	assert.Equal(t, xml.Name{Space: "urn:item", Local: "Item"}, typ)
	var item plainItem
	require.NoError(t, result.Value.Value.Decode(&item))
	assert.Equal(t, 7, item.ID)
}

// oracleDeclareXSITypePrefixes is the straightforward version of
// declareXSITypePrefixes, built on xml.Decoder tokens, that the byte scanner
// is checked against.
func oracleDeclareXSITypePrefixes(data []byte) []byte {
	type insertion struct {
		at   int
		text string
	}
	var inserts []insertion
	var scopes []map[string]string
	lookup := func(prefix string) (string, bool) {
		for i := len(scopes) - 1; i >= 0; i-- {
			if uri, ok := scopes[i][prefix]; ok {
				return uri, true
			}
		}
		return "", false
	}
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return data
		}
		switch t := tok.(type) {
		case xml.StartElement:
			declared := make(map[string]string)
			for _, attr := range t.Attr {
				if attr.Name.Space == "xmlns" {
					declared[attr.Name.Local] = attr.Value
				}
			}
			var decls strings.Builder
			for _, attr := range t.Attr {
				if attr.Name.Space != XSINamespace || attr.Name.Local != "type" {
					continue
				}
				prefix, _, ok := strings.Cut(strings.TrimSpace(attr.Value), ":")
				if !ok || prefix == "" || prefix == "xml" {
					continue
				}
				if _, done := declared[prefix]; done {
					continue
				}
				if uri, bound := lookup(prefix); bound {
					declared[prefix] = uri
					decls.WriteString(" xmlns:" + prefix + `="`)
					_ = xml.EscapeText(&decls, []byte(uri))
					decls.WriteString(`"`)
				}
			}
			scopes = append(scopes, declared)
			if decls.Len() > 0 {
				end := int(d.InputOffset())
				at := end - 1
				if data[end-2] == '/' {
					at = end - 2
				}
				inserts = append(inserts, insertion{at: at, text: decls.String()})
			}
		case xml.EndElement:
			scopes = scopes[:len(scopes)-1]
		}
	}
	var out bytes.Buffer
	last := 0
	for _, ins := range inserts {
		out.Write(data[last:ins.at])
		out.WriteString(ins.text)
		last = ins.at
	}
	out.Write(data[last:])
	return out.Bytes()
}

// xmlTokens returns the decoded token stream of data, or ok false if data
// is not well-formed.
func xmlTokens(data []byte) (tokens []xml.Token, ok bool) {
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return tokens, true
		}
		if err != nil {
			return nil, false
		}
		tokens = append(tokens, xml.CopyToken(tok))
	}
}

// checkScannerMatchesOracle fails when the byte scanner and the token-based
// oracle give documents that decode differently. Inputs the scanner
// deliberately leaves alone (entities, DOCTYPEs, single-quoted values) only
// need to stay well-formed.
func checkScannerMatchesOracle(t *testing.T, in []byte) {
	t.Helper()
	if _, ok := xmlTokens(in); !ok {
		return
	}
	got := declareXSITypePrefixes(in)
	gotTokens, ok := xmlTokens(got)
	require.True(t, ok, "scanner output is not well-formed:\n%s", got)
	if bytes.ContainsAny(in, "&'") || bytes.Contains(in, []byte("<!DOCTYPE")) ||
		!bytes.Contains(in, []byte(":type=")) {
		return
	}
	wantTokens, _ := xmlTokens(oracleDeclareXSITypePrefixes(in))
	require.Equal(t, wantTokens, gotTokens, "input:\n%s\nscanner output:\n%s", in, got)
}

var scannerCorpus = []string{
	typedEnvelope,
	`<?xml version="1.0"?><!-- <v xsi:type="p:T"> --><e xmlns:p="urn:p" xmlns:xsi="` + XSINamespace + `"><v xsi:type="p:T"/></e>`,
	`<e xmlns:p="urn:p" xmlns:x="` + XSINamespace + `"><a><b x:type=" p:T " c="d>e"><![CDATA[<v xsi:type="p:T">]]></b></a></e>`,
	`<e xmlns:p="urn:p" xmlns:xsi="` + XSINamespace + `"><v xmlns:p="urn:q"><w xsi:type="p:T"/></v><w xsi:type="p:T"></w></e>`,
	`<e xmlns:p="urn:p" xmlns:q="urn:q" xmlns:xsi="` + XSINamespace + `"><v xsi:type="p:T" q:type="q:U"><w xsi:type="q:U"/></v></e>`,
	`<e xmlns:xsi="` + XSINamespace + `"><v xmlns:p="urn:p"/><w xsi:type="p:T"/></e>`,
	`<e
  xmlns:p="urn:p"
  xmlns:xsi="` + XSINamespace + `"
><v
  xsi:type="p:T"
  /></e>`,
	`<e xmlns:p="urn:p" xmlns:xsi="` + XSINamespace + `"><v xsi:type="xsd:string"/><v xsi:type="T"/><v xsi:type="xml:lang"/></e>`,
}

func TestDeclareXSITypePrefixes_MatchesOracle(t *testing.T) {
	for _, in := range scannerCorpus {
		checkScannerMatchesOracle(t, []byte(in))
	}
	files, err := filepath.Glob("../fixtures/**/*.wsdl")
	require.NoError(t, err)
	top, err := filepath.Glob("../fixtures/*.wsdl")
	require.NoError(t, err)
	for _, f := range append(files, top...) {
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		// Give every element of the fixture an xsi:type using the
		// prefixes the fixture declares on its root.
		withTypes := bytes.ReplaceAll(data, []byte("<xsd:element "), []byte(`<xsd:element xsi:type="tns:Fixture" `))
		withTypes = bytes.Replace(withTypes, []byte("<definitions "), []byte(`<definitions xmlns:xsi="`+XSINamespace+`" `), 1)
		withTypes = bytes.Replace(withTypes, []byte("<wsdl:definitions "), []byte(`<wsdl:definitions xmlns:xsi="`+XSINamespace+`" `), 1)
		checkScannerMatchesOracle(t, withTypes)
	}
}

func FuzzDeclareXSITypePrefixes(f *testing.F) {
	for _, in := range scannerCorpus {
		f.Add(in)
	}
	f.Fuzz(func(t *testing.T, in string) {
		checkScannerMatchesOracle(t, []byte(in))
	})
}
