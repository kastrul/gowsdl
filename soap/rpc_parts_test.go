// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package soap

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rpcTestItem struct {
	Text string `xml:"text"`
	Ref  *struct {
		Code string `xml:"code"`
	} `xml:"urn:other ref,omitempty"`
}

type rpcTestElement struct {
	XMLName xml.Name `xml:"urn:el Item"`
	Text    string   `xml:"text"`
}

type rpcTestPrefixedElement struct {
	XMLName xml.Name `xml:"tns:Item"`
	Text    string   `xml:"text"`
}

// rpcTestItemWithElement is a type of a qualified schema with a field
// holding an element named with the tns prefix.
type rpcTestItemWithElement struct {
	Text string                  `xml:"text"`
	Item *rpcTestPrefixedElement `xml:"tns:Item"`
}

type rpcTestSimple string

// rpcPartNames encodes write inside an operation wrapper in the namespace
// urn:op, as an RPC request body is, and returns the qualified names of the
// elements inside the wrapper, resolved as a receiver reads them.
func rpcPartNames(t *testing.T, write func(e *xml.Encoder) error) []string {
	t.Helper()

	var buf bytes.Buffer
	e := xml.NewEncoder(&buf)
	wrapper := xml.StartElement{
		Name: xml.Name{Space: "urn:op", Local: "Op"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "xmlns:tns"}, Value: "urn:tns"}},
	}
	require.NoError(t, e.EncodeToken(wrapper))
	require.NoError(t, write(e))
	require.NoError(t, e.EncodeToken(wrapper.End()))
	require.NoError(t, e.Flush())

	var names []string
	d := xml.NewDecoder(&buf)
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err, buf.String())
		if start, ok := tok.(xml.StartElement); ok && start.Name.Local != "Op" {
			names = append(names, "{"+start.Name.Space+"}"+start.Name.Local)
		}
	}
	return names
}

func TestMarshalRPCTypePart(t *testing.T) {
	item := &rpcTestItem{Text: "a", Ref: &struct {
		Code string `xml:"code"`
	}{Code: "c"}}

	tests := []struct {
		name      string
		namespace string
		value     any
		want      []string
	}{
		{
			name:  "simple type",
			value: 42,
			want:  []string{"{}count"},
		},
		{
			name:  "complex type of an unqualified schema",
			value: item,
			want:  []string{"{}count", "{}text", "{urn:other}ref", "{urn:other}code"},
		},
		{
			name:      "complex type of a qualified schema",
			namespace: "urn:q",
			value:     item,
			want:      []string{"{}count", "{urn:q}text", "{urn:other}ref", "{urn:other}code"},
		},
		{
			name:      "complex type of a qualified schema holding a tns-prefixed element",
			namespace: "urn:q",
			value:     &rpcTestItemWithElement{Text: "a", Item: &rpcTestPrefixedElement{Text: "b"}},
			want:      []string{"{}count", "{urn:q}text", "{urn:tns}Item", "{urn:q}text"},
		},
		{
			name:  "nil writes nothing",
			value: (*rpcTestItem)(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rpcPartNames(t, func(e *xml.Encoder) error {
				return MarshalRPCTypePart(e, "count", tt.namespace, tt.value)
			})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMarshalRPCElementPart(t *testing.T) {
	simple := rpcTestSimple("v")

	tests := []struct {
		name  string
		value any
		want  []string
	}{
		{
			name:  "element of a qualified schema",
			value: &rpcTestElement{Text: "a"},
			want:  []string{"{}item", "{urn:el}Item", "{urn:el}text"},
		},
		{
			name:  "element named with the tns prefix",
			value: &rpcTestPrefixedElement{Text: "a"},
			want:  []string{"{}item", "{urn:tns}Item", "{}text"},
		},
		{
			name:  "element of a simple type",
			value: &simple,
			want:  []string{"{}item", "{urn:el}Simple"},
		},
		{
			name:  "nil writes nothing",
			value: (*rpcTestElement)(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rpcPartNames(t, func(e *xml.Encoder) error {
				return MarshalRPCElementPart(e, "item", xml.Name{Space: "urn:el", Local: "Simple"}, tt.value)
			})
			assert.Equal(t, tt.want, got)
		})
	}
}
