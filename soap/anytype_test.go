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

const xsiNamespace = "http://www.w3.org/2001/XMLSchema-instance"

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
		Attrs: []xml.Attr{{Name: xml.Name{Space: xsiNamespace, Local: "type"}, Value: "Item"}},
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
	assert.Equal(t, xml.Name{Space: xsiNamespace, Local: "type"}, v.Attrs[0].Name)
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
