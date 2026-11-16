// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package parser

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// compositorsByName returns the compositors of every element in lists, and
// of the elements of their anonymous types, by element name.
func compositorsByName(lists ...[]*XSDElement) map[string]string {
	got := map[string]string{}
	var walk func([]*XSDElement)
	walk = func(list []*XSDElement) {
		for _, el := range list {
			got[el.Name] = el.Compositors.String()
			if el.ComplexType != nil {
				walk(el.ComplexType.Sequence)
			}
		}
	}
	for _, list := range lists {
		walk(list)
	}
	return got
}

func TestXSDComplexType_Compositors(t *testing.T) {
	const src = `<xs:complexType name="T" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:tns="urn:t">
  <xs:sequence>
    <xs:element name="id"/>
    <xs:sequence minOccurs="0" maxOccurs="unbounded">
      <xs:element name="a"/>
      <xs:choice><xs:element name="b"/><xs:element name="c"/></xs:choice>
      <xs:sequence maxOccurs="3"><xs:element name="d"/><xs:element name="e"/></xs:sequence>
      <xs:group ref="tns:G" maxOccurs="2"/>
    </xs:sequence>
    <xs:element name="nested">
      <xs:complexType><xs:sequence maxOccurs="unbounded"><xs:element name="inner"/></xs:sequence></xs:complexType>
    </xs:element>
  </xs:sequence>
</xs:complexType>`
	var ct XSDComplexType
	require.NoError(t, xml.Unmarshal([]byte(src), &ct))

	assert.Equal(t, map[string]string{
		"id":     "sequence:1::",
		"a":      "sequence:1:: sequence:2:0:unbounded",
		"b":      "sequence:1:: sequence:2:0:unbounded choice:3::",
		"c":      "sequence:1:: sequence:2:0:unbounded choice:3::",
		"d":      "sequence:1:: sequence:2:0:unbounded sequence:4::3",
		"e":      "sequence:1:: sequence:2:0:unbounded sequence:4::3",
		"nested": "sequence:1::",
		"inner":  "sequence:1::unbounded",
	}, compositorsByName(ct.Sequence, ct.SequenceChoice, ct.Choice), "a nested type numbers its own compositors")

	require.Len(t, ct.GroupRefs, 1)
	assert.Equal(t, XSDCompositors{
		{Kind: "sequence", ID: 1},
		{Kind: "sequence", ID: 2, MinOccurs: "0", MaxOccurs: "unbounded"},
	}, ct.GroupRefs[0].Compositors)
}

func TestXSDExtension_Compositors(t *testing.T) {
	const src = `<xs:complexType name="D" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexContent>
    <xs:extension base="B">
      <xs:sequence>
        <xs:sequence maxOccurs="unbounded"><xs:element name="a"/><xs:element name="b"/></xs:sequence>
        <xs:element name="last"/>
      </xs:sequence>
    </xs:extension>
  </xs:complexContent>
</xs:complexType>`
	var ct XSDComplexType
	require.NoError(t, xml.Unmarshal([]byte(src), &ct))
	ext := ct.ComplexContent.Extension
	assert.Equal(t, map[string]string{
		"a":    "sequence:1:: sequence:2::unbounded",
		"b":    "sequence:1:: sequence:2::unbounded",
		"last": "sequence:1::",
	}, compositorsByName(ext.Sequence, ext.SequenceChoice, ext.Choice))
}

func TestXSDGroup_CompositorsNameTheDefinition(t *testing.T) {
	const src = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:tns="urn:g" targetNamespace="urn:g">
  <xs:group name="G">
    <xs:sequence>
      <xs:element name="x"/>
      <xs:sequence maxOccurs="unbounded"><xs:element name="y"/></xs:sequence>
      <xs:group ref="tns:H"/>
    </xs:sequence>
  </xs:group>
</xs:schema>`
	var s XSDSchema
	require.NoError(t, xml.Unmarshal([]byte(src), &s))
	require.Len(t, s.Groups, 1)
	g := s.Groups[0]
	require.Len(t, g.Elements, 2)
	assert.Equal(t, XSDCompositors{{Kind: "sequence", ID: 1, Group: "G"}}, g.Elements[0].Compositors)
	assert.Equal(t, XSDCompositors{{Kind: "sequence", ID: 1, Group: "G"}, {Kind: "sequence", ID: 2, MaxOccurs: "unbounded"}}, g.Elements[1].Compositors)
	require.Len(t, g.GroupRefs, 1)
	assert.Equal(t, XSDCompositors{{Kind: "sequence", ID: 1, Group: "G"}}, g.GroupRefs[0].Compositors)
}

func TestXSDCompositors_Attr(t *testing.T) {
	var c XSDCompositors
	require.NoError(t, c.UnmarshalXMLAttr(xml.Attr{Value: "sequence:1:: choice:2:0:unbounded"}))
	assert.Equal(t, XSDCompositors{{Kind: "sequence", ID: 1}, {Kind: "choice", ID: 2, MinOccurs: "0", MaxOccurs: "unbounded"}}, c)

	attr, err := c.MarshalXMLAttr(compositorsAttr)
	require.NoError(t, err)
	assert.Equal(t, xml.Attr{Name: compositorsAttr, Value: "sequence:1:: choice:2:0:unbounded"}, attr)

	attr, err = XSDCompositors(nil).MarshalXMLAttr(compositorsAttr)
	require.NoError(t, err)
	assert.Equal(t, xml.Attr{}, attr, "no attribute for no compositors")

	assert.Error(t, c.UnmarshalXMLAttr(xml.Attr{Value: "sequence:1"}))
	assert.Error(t, c.UnmarshalXMLAttr(xml.Attr{Value: "sequence:x::"}))

	el := XSDElement{Name: "e", Compositors: XSDCompositors{{Kind: "all", ID: 3}}}
	out, err := xml.Marshal(el)
	require.NoError(t, err)
	var back XSDElement
	require.NoError(t, xml.Unmarshal(out, &back))
	assert.Equal(t, el.Compositors, back.Compositors)
}
