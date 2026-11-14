// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package soap

import (
	"encoding/json"
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type anyHolder struct {
	XMLName xml.Name `xml:"holder"`
	Value   AnyType  `xml:"value,omitempty"`
}

type anyItem struct {
	ID   int    `xml:"urn:item id"`
	Name string `xml:"name"`
}

type plainItem struct {
	ID int `xml:"id"`
}

func TestAnyType_ZeroValueIsOmitted(t *testing.T) {
	out, err := xml.Marshal(anyHolder{})
	require.NoError(t, err)
	assert.Equal(t, "<holder></holder>", string(out))
	assert.True(t, AnyType{}.IsZero())
}

func TestAnyType_EncodesValueWithAttrs(t *testing.T) {
	h := anyHolder{Value: AnyType{
		Value: plainItem{ID: 7},
		Attrs: []xml.Attr{{Name: xml.Name{Space: XSINamespace, Local: "type"}, Value: "Item"}},
	}}

	out, err := xml.Marshal(h)
	require.NoError(t, err)

	var back struct {
		Value struct {
			Type string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
			ID   int    `xml:"id"`
		} `xml:"value"`
	}
	require.NoError(t, xml.Unmarshal(out, &back), "output: %s", out)
	assert.Equal(t, "Item", back.Value.Type)
	assert.Equal(t, 7, back.Value.ID)
}

func TestAnyType_DecodesContentUsingAncestorNamespaces(t *testing.T) {
	// The prefixes are declared on the envelope, as SOAP servers commonly do,
	// and the content mixes qualified and unqualified children.
	in := `<Envelope xmlns:ns2="urn:item" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
		`<holder><value xsi:type="ns2:Item"><ns2:id>7</ns2:id><name>Jane &amp; co</name></value></holder>` +
		`</Envelope>`

	var env struct {
		Holder anyHolder `xml:"holder"`
	}
	require.NoError(t, xml.Unmarshal([]byte(in), &env))
	v := env.Holder.Value

	assert.Equal(t, "value", v.XMLName.Local)
	require.Len(t, v.Attrs, 1)
	assert.Equal(t, xml.Name{Space: XSINamespace, Local: "type"}, v.Attrs[0].Name)
	assert.Equal(t, "ns2:Item", v.Attrs[0].Value)
	assert.Contains(t, v.InnerXML, `="urn:item"`, "InnerXML must declare the ancestor namespace it uses")

	var item anyItem
	require.NoError(t, v.Decode(&item))
	assert.Equal(t, anyItem{ID: 7, Name: "Jane & co"}, item)
}

func TestAnyType_UnqualifiedChildStaysUnqualified(t *testing.T) {
	in := `<holder xmlns="urn:outer"><value><id>7</id></value></holder>`
	var h anyHolder
	require.NoError(t, xml.Unmarshal([]byte(in), &h))

	var qualified struct {
		ID int `xml:"urn:outer id"`
	}
	require.NoError(t, h.Value.Decode(&qualified))
	assert.Equal(t, 7, qualified.ID, "id was in urn:outer through the default namespace")

	in = `<o:holder xmlns:o="urn:outer"><o:value><id>7</id></o:value></o:holder>`
	require.NoError(t, xml.Unmarshal([]byte(in), &h))
	var wrong struct {
		ID int `xml:"urn:outer id"`
	}
	require.NoError(t, h.Value.Decode(&wrong))
	assert.Zero(t, wrong.ID, "id was in no namespace and must not pick one up")
}

func TestAnyType_ReceivedValueRoundTrips(t *testing.T) {
	in := `<Envelope xmlns:ns2="urn:item"><holder><value><ns2:id>7</ns2:id><name>x</name></value></holder></Envelope>`
	var env struct {
		Holder anyHolder `xml:"holder"`
	}
	require.NoError(t, xml.Unmarshal([]byte(in), &env))

	out, err := xml.Marshal(env.Holder)
	require.NoError(t, err)

	var again anyHolder
	require.NoError(t, xml.Unmarshal(out, &again), "output: %s", out)
	var item anyItem
	require.NoError(t, again.Value.Decode(&item))
	assert.Equal(t, anyItem{ID: 7, Name: "x"}, item)
}

func TestAnyType_EncodesInnerXMLVerbatim(t *testing.T) {
	out, err := xml.Marshal(anyHolder{Value: AnyType{InnerXML: "<id>7</id>"}})
	require.NoError(t, err)
	assert.Equal(t, "<holder><value><id>7</id></value></holder>", string(out))
}

func TestAnyType_DecodeOfConstructedValue(t *testing.T) {
	var item plainItem
	require.NoError(t, AnyType{InnerXML: "<id>7</id>"}.Decode(&item))
	assert.Equal(t, 7, item.ID)
}

func TestAnyType_KeepsCommentsAndEscapes(t *testing.T) {
	in := `<holder><value a="x &quot;y&quot;"><!-- note --><t>1 &lt; 2</t></value></holder>`
	var h anyHolder
	require.NoError(t, xml.Unmarshal([]byte(in), &h))
	assert.Contains(t, h.Value.InnerXML, "<!-- note -->")

	var got struct {
		A string `xml:"a,attr"`
		T string `xml:"t"`
	}
	require.NoError(t, h.Value.Decode(&got))
	assert.Equal(t, `x "y"`, got.A)
	assert.Equal(t, "1 < 2", got.T)
}

func TestAnyType_JSON(t *testing.T) {
	out, err := json.Marshal(struct {
		V AnyType `json:"v,omitzero"`
	}{})
	require.NoError(t, err)
	assert.Equal(t, `{}`, string(out))

	out, err = json.Marshal(AnyType{InnerXML: "<id>7</id>"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"innerXML":"<id>7</id>"}`, string(out))
}

type wildcardDoc struct {
	XMLName xml.Name     `xml:"doc"`
	Known   string       `xml:"known"`
	Items   []AnyElement `xml:",any"`
}

func TestAnyType_WildcardItemsKeepEachUnknownElement(t *testing.T) {
	in := `<doc xmlns:e="urn:e"><known>1</known><e:a k="v">x</e:a><b><c>2</c></b></doc>`
	var doc wildcardDoc
	require.NoError(t, xml.Unmarshal([]byte(in), &doc))

	assert.Equal(t, "1", doc.Known)
	require.Len(t, doc.Items, 2)
	assert.Equal(t, xml.Name{Space: "urn:e", Local: "a"}, doc.Items[0].XMLName)
	assert.Equal(t, []xml.Attr{{Name: xml.Name{Local: "k"}, Value: "v"}}, doc.Items[0].Attrs)
	assert.Equal(t, xml.Name{Local: "b"}, doc.Items[1].XMLName)

	var text string
	require.NoError(t, doc.Items[0].Decode(&text))
	assert.Equal(t, "x", text)
	var b struct {
		C int `xml:"c"`
	}
	require.NoError(t, doc.Items[1].Decode(&b))
	assert.Equal(t, 2, b.C)

	out, err := xml.Marshal(doc)
	require.NoError(t, err)
	var again wildcardDoc
	require.NoError(t, xml.Unmarshal(out, &again), "output: %s", out)
	require.Len(t, again.Items, 2, "output: %s", out)
	assert.Equal(t, doc.Items[0].XMLName, again.Items[0].XMLName)
	assert.Equal(t, doc.Items[1].XMLName, again.Items[1].XMLName)
	require.NoError(t, again.Items[1].Decode(&b))
	assert.Equal(t, 2, b.C)
}

type extension struct {
	XMLName xml.Name `xml:"urn:e ext"`
	N       int      `xml:"n"`
}

func TestAnyType_WildcardItemNaming(t *testing.T) {
	doc := wildcardDoc{Items: []AnyElement{
		{Value: extension{N: 1}},
		{Value: extension{N: 2}, Attrs: []xml.Attr{{Name: xml.Name{Local: "k"}, Value: "v"}}},
		{XMLName: xml.Name{Local: "raw"}, InnerXML: "<z></z>"},
		{XMLName: xml.Name{Space: "urn:x", Local: "named"}, Value: plainItem{ID: 3}},
	}}

	out, err := xml.Marshal(doc)
	require.NoError(t, err)

	var back struct {
		Ext []struct {
			K string `xml:"k,attr"`
			N int    `xml:"n"`
		} `xml:"urn:e ext"`
		Raw   string `xml:"raw>z"`
		Named struct {
			ID int `xml:"id"`
		} `xml:"urn:x named"`
	}
	require.NoError(t, xml.Unmarshal(out, &back), "output: %s", out)
	require.Len(t, back.Ext, 2, "output: %s", out)
	assert.Equal(t, 1, back.Ext[0].N)
	assert.Equal(t, 2, back.Ext[1].N)
	assert.Equal(t, "v", back.Ext[1].K)
	assert.NotContains(t, string(out), "<AnyType", "an item must never be named after the Go type")
	assert.Contains(t, string(out), "<raw><z></z></raw>")
	assert.Equal(t, 3, back.Named.ID)
}

func TestAnyElement_NeedsAName(t *testing.T) {
	_, err := xml.Marshal(wildcardDoc{Items: []AnyElement{{InnerXML: "<z></z>"}}})
	require.ErrorIs(t, err, errAnyElementName)
}
