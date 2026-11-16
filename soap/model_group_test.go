// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package soap

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The types below are written as the generator writes them for
//
//	<xs:sequence>
//	  <xs:element name="id" type="xs:int"/>
//	  <xs:sequence maxOccurs="unbounded">
//	    <xs:element name="a" type="xs:string"/>
//	    <xs:element name="b" type="xs:int" minOccurs="0"/>
//	    <xs:element name="item" type="tns:groupItem" minOccurs="0"/>
//	    <xs:sequence maxOccurs="unbounded">
//	      <xs:element name="x" type="xs:string"/>
//	      <xs:element name="y" type="xs:string" minOccurs="0"/>
//	    </xs:sequence>
//	  </xs:sequence>
//	  <xs:element name="last" type="xs:string"/>
//	</xs:sequence>

type groupItem struct {
	Sku  string `xml:"sku"`
	Code string `xml:"code,attr,omitempty"`
}

type groupInner struct {
	X string  `xml:"x"`
	Y *string `xml:"y,omitempty"`
}

func (o groupInner) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	type plain groupInner
	return MarshalModelGroup(e, plain(o))
}

type groupOuter struct {
	A     string       `xml:"a"`
	B     int32        `xml:"b,omitempty"`
	Item  *groupItem   `xml:"item,omitempty"`
	Inner []groupInner `xml:"x,omitempty"`
}

func (o groupOuter) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	type plain groupOuter
	return MarshalModelGroup(e, plain(o))
}

type GroupParentFixture struct {
	XMLName xml.Name     `xml:"urn:g Parent"`
	ID      int32        `xml:"id"`
	AGroup  []groupOuter `xml:"a,omitempty"`
	Last    string       `xml:"last"`
	Attr    string       `xml:"at,attr,omitempty"`
	Items   []AnyElement `xml:",any"`
}

func (t *GroupParentFixture) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	return UnmarshalModelGroups(d, start, &struct {
		*GroupParentFixture
		UnmarshalXML struct{}         `xml:"-"`
		M0           ModelGroupMember `xml:"a"`
		M1           ModelGroupMember `xml:"b"`
		M2           ModelGroupMember `xml:"item"`
		M3           ModelGroupMember `xml:"x"`
		M4           ModelGroupMember `xml:"y"`
	}{
		GroupParentFixture: t,
		M0:          GroupMember[groupOuter](0),
		M1:          GroupMember[groupOuter](1),
		M2:          GroupMember[groupOuter](2),
		M3:          GroupMember[groupOuter](3, 0),
		M4:          GroupMember[groupOuter](3, 1),
	})
}

// GroupDerivedFixture embeds GroupParentFixture, as a complexContent extension does, and
// adds a group of its own.
type GroupDerivedFixture struct {
	GroupParentFixture
	PGroup []groupPair `xml:"p,omitempty"`
}

type groupPair struct {
	P string `xml:"p"`
	Q string `xml:"q"`
}

func (o groupPair) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	type plain groupPair
	return MarshalModelGroup(e, plain(o))
}

func (t *GroupDerivedFixture) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	return UnmarshalModelGroups(d, start, &struct {
		*GroupDerivedFixture
		UnmarshalXML struct{}         `xml:"-"`
		M0           ModelGroupMember `xml:"a"`
		M1           ModelGroupMember `xml:"b"`
		M2           ModelGroupMember `xml:"item"`
		M3           ModelGroupMember `xml:"x"`
		M4           ModelGroupMember `xml:"y"`
		M5           ModelGroupMember `xml:"p"`
		M6           ModelGroupMember `xml:"q"`
	}{
		GroupDerivedFixture: t,
		M0:           GroupMember[groupOuter](0),
		M1:           GroupMember[groupOuter](1),
		M2:           GroupMember[groupOuter](2),
		M3:           GroupMember[groupOuter](3, 0),
		M4:           GroupMember[groupOuter](3, 1),
		M5:           GroupMember[groupPair](0),
		M6:           GroupMember[groupPair](1),
	})
}

// NamedFixture is exported because encoding/xml panics on an XMLName in an
// unexported embedded struct.
type NamedFixture struct {
	XMLName xml.Name `xml:"urn:g Parent"`
}

func strPtr(s string) *string { return &s }

func TestMarshalModelGroup_WritesOccurrencesInSchemaOrder(t *testing.T) {
	v := GroupParentFixture{
		ID: 1,
		AGroup: []groupOuter{
			{A: "a1", B: 2, Item: &groupItem{Sku: "s", Code: "c"}, Inner: []groupInner{{X: "x1", Y: strPtr("y1")}, {X: "x2"}}},
			{A: "a2"},
			{A: "", B: 3},
		},
		Last: "z",
	}
	out, err := xml.Marshal(v)
	require.NoError(t, err)
	assert.Equal(t, `<Parent xmlns="urn:g"><id>1</id>`+
		`<a>a1</a><b>2</b><item code="c"><sku>s</sku></item><x>x1</x><y>y1</y><x>x2</x>`+
		`<a>a2</a>`+
		`<a></a><b>3</b>`+
		`<last>z</last></Parent>`, string(out))
}

func TestMarshalModelGroup_WritesWhatEncodingXMLWrites(t *testing.T) {
	type withAttr struct {
		V string `xml:"urn:attrs v,attr"`
	}
	type occurrence struct {
		Q   string     `xml:"urn:other q"`
		P   string     `xml:"tns:p"`
		Opt *groupItem `xml:"opt,omitempty"`
		W   withAttr   `xml:"w"`
		N   string     `xml:"n"`
	}
	v := occurrence{Q: "1", P: "2", W: withAttr{V: "v"}, N: "a&b<"}

	// encoding/xml flattens an embedded struct into its parent: what it
	// writes for the parent is what the group must write.
	want, err := xml.Marshal(struct {
		XMLName xml.Name `xml:"root"`
		occurrence
	}{occurrence: v})
	require.NoError(t, err)

	var b strings.Builder
	e := xml.NewEncoder(&b)
	require.NoError(t, e.EncodeToken(xml.StartElement{Name: xml.Name{Local: "root"}}))
	require.NoError(t, MarshalModelGroup(e, v))
	require.NoError(t, e.EncodeToken(xml.EndElement{Name: xml.Name{Local: "root"}}))
	require.NoError(t, e.Flush())

	assert.Equal(t, string(want), b.String())
	assert.Equal(t, `<root><q xmlns="urn:other">1</q><tns:p>2</tns:p><w xmlns:_="urn:attrs" _:v="v"></w><n>a&amp;b&lt;</n></root>`, b.String())
}

func TestMarshalModelGroup_NewlinesInText(t *testing.T) {
	var b strings.Builder
	e := xml.NewEncoder(&b)
	require.NoError(t, MarshalModelGroup(e, struct {
		N string `xml:"n"`
	}{N: "a\nb\rc"}))
	require.NoError(t, e.Flush())
	var got struct {
		N string `xml:"n"`
	}
	require.NoError(t, xml.Unmarshal([]byte("<r>"+b.String()+"</r>"), &got))
	assert.Equal(t, "a\nb\rc", got.N, "the text reads back the same, though a newline is no longer written as a character reference")
}

func TestMarshalModelGroup_Indents(t *testing.T) {
	var b strings.Builder
	e := xml.NewEncoder(&b)
	e.Indent("", " ")
	require.NoError(t, e.Encode(GroupParentFixture{AGroup: []groupOuter{{A: "1"}, {A: "2", B: 3}}}))
	assert.Equal(t, "<Parent xmlns=\"urn:g\">\n <id>0</id>\n <a>1</a>\n <a>2</a>\n <b>3</b>\n <last></last>\n</Parent>", b.String())
}

func TestMarshalModelGroup_Errors(t *testing.T) {
	var b strings.Builder
	e := xml.NewEncoder(&b)
	err := MarshalModelGroup(e, struct {
		C chan int `xml:"c"`
	}{C: make(chan int)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "soap: encoding model group")

	err = MarshalModelGroup(e, struct {
		Raw string `xml:",innerxml"`
	}{Raw: "<a><b></a>"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "soap: re-encoding model group")
}

func TestUnmarshalModelGroups_RegroupsOccurrences(t *testing.T) {
	in := `<Parent xmlns="urn:g" at="q"><id>1</id>` +
		`<a>a1</a><b>2</b><item code="c"><sku>s</sku></item><x>x1</x><y>y1</y><x>x2</x>` +
		`<a>a2</a>` +
		`<a></a><b>3</b>` +
		`<extra>e</extra><last>z</last></Parent>`
	var v GroupParentFixture
	require.NoError(t, xml.Unmarshal([]byte(in), &v))

	assert.Equal(t, xml.Name{Space: "urn:g", Local: "Parent"}, v.XMLName)
	assert.Equal(t, int32(1), v.ID)
	assert.Equal(t, "z", v.Last)
	assert.Equal(t, "q", v.Attr)
	require.Len(t, v.Items, 1)
	assert.Equal(t, "extra", v.Items[0].XMLName.Local)
	assert.Equal(t, []groupOuter{
		{A: "a1", B: 2, Item: &groupItem{Sku: "s", Code: "c"}, Inner: []groupInner{{X: "x1", Y: strPtr("y1")}, {X: "x2"}}},
		{A: "a2"},
		{A: "", B: 3},
	}, v.AGroup)

	out, err := xml.Marshal(GroupParentFixture{ID: v.ID, AGroup: v.AGroup, Last: v.Last, Attr: v.Attr})
	require.NoError(t, err)
	assert.Equal(t, strings.Replace(in, "<extra>e</extra>", "", 1), string(out))
}

func TestUnmarshalModelGroups_OptionalMemberMissing(t *testing.T) {
	// b and item are optional: a b? item? repeated.
	in := `<Parent xmlns="urn:g"><a>1</a><item><sku>s1</sku></item><a>2</a><b>2</b><a>3</a><item><sku>s3</sku></item></Parent>`
	var v GroupParentFixture
	require.NoError(t, xml.Unmarshal([]byte(in), &v))
	assert.Equal(t, []groupOuter{
		{A: "1", Item: &groupItem{Sku: "s1"}},
		{A: "2", B: 2},
		{A: "3", Item: &groupItem{Sku: "s3"}},
	}, v.AGroup)
}

func TestUnmarshalModelGroups_StartsOccurrenceOnRepeatedSingleMember(t *testing.T) {
	// The second b cannot join the occurrence holding the first.
	in := `<Parent xmlns="urn:g"><b>1</b><b>2</b><x>x</x><x>y</x></Parent>`
	var v GroupParentFixture
	require.NoError(t, xml.Unmarshal([]byte(in), &v))
	assert.Equal(t, []groupOuter{
		{B: 1},
		{B: 2, Inner: []groupInner{{X: "x"}, {X: "y"}}},
	}, v.AGroup)
}

func TestUnmarshalModelGroups_Embedded(t *testing.T) {
	in := `<Parent xmlns="urn:g"><id>4</id><a>1</a><a>2</a><last>z</last><p>p1</p><q>q1</q><p>p2</p><q>q2</q></Parent>`
	var v GroupDerivedFixture
	require.NoError(t, xml.Unmarshal([]byte(in), &v))
	assert.Equal(t, int32(4), v.ID)
	assert.Equal(t, []groupOuter{{A: "1"}, {A: "2"}}, v.AGroup)
	assert.Equal(t, []groupPair{{P: "p1", Q: "q1"}, {P: "p2", Q: "q2"}}, v.PGroup)
	var plain struct{ NamedFixture }
	require.NoError(t, xml.Unmarshal([]byte(in), &plain))
	assert.Equal(t, plain.XMLName, v.XMLName, "an XMLName the struct only embeds is left as encoding/xml leaves it")

	out, err := xml.Marshal(v)
	require.NoError(t, err)
	assert.Equal(t, `<Parent xmlns="urn:g"><id>4</id><a>1</a><a>2</a><last>z</last><p>p1</p><q>q1</q><p>p2</p><q>q2</q></Parent>`, string(out))
}

func TestUnmarshalModelGroups_DecodesIntoFreshOccurrences(t *testing.T) {
	v := GroupParentFixture{AGroup: []groupOuter{{A: "old"}}}
	require.NoError(t, xml.Unmarshal([]byte(`<Parent xmlns="urn:g"><b>1</b></Parent>`), &v))
	assert.Equal(t, []groupOuter{{A: "old"}, {B: 1}}, v.AGroup, "decoding appends, as encoding/xml does to slices, and starts a new occurrence")
}

func TestUnmarshalModelGroups_MalformedInput(t *testing.T) {
	for name, in := range map[string]string{
		"truncated":       `<Parent xmlns="urn:g"><a>1</a><b>`,
		"bad member text": `<Parent xmlns="urn:g"><a>1</a><b>two</b></Parent>`,
		"mismatched tag":  `<Parent xmlns="urn:g"><a>1</b></Parent>`,
		"wrong element":   `<Other xmlns="urn:g"><a>1</a></Other>`,
	} {
		t.Run(name, func(t *testing.T) {
			var v GroupParentFixture
			assert.Error(t, xml.Unmarshal([]byte(in), &v))
		})
	}
}

func TestUnmarshalModelGroups_RejectsBadShadow(t *testing.T) {
	start := xml.StartElement{Name: xml.Name{Space: "urn:g", Local: "Parent"}}
	newDecoder := func() *xml.Decoder {
		d := xml.NewDecoder(strings.NewReader(`<Parent xmlns="urn:g"><a>1</a></Parent>`))
		_, _ = d.Token()
		return d
	}
	var p GroupParentFixture

	assert.ErrorContains(t, UnmarshalModelGroups(newDecoder(), start, p), "needs a pointer to a struct")
	assert.ErrorContains(t, UnmarshalModelGroups(newDecoder(), start, &struct{ X int }{}), "must embed")
	assert.ErrorContains(t, UnmarshalModelGroups(newDecoder(), start, &struct{ *GroupParentFixture }{}), "must embed a non-nil")
	assert.ErrorContains(t, UnmarshalModelGroups(newDecoder(), start, &struct{ *groupItem }{&groupItem{}}), "must embed a non-nil pointer to an exported struct")
	assert.ErrorContains(t, UnmarshalModelGroups(newDecoder(), start, &struct {
		*GroupParentFixture
		M ModelGroupMember `xml:"a"`
	}{GroupParentFixture: &p}), "was not made by GroupMember")
	assert.ErrorContains(t, UnmarshalModelGroups(newDecoder(), start, &struct {
		*GroupParentFixture
		M ModelGroupMember `xml:"a"`
	}{GroupParentFixture: &p, M: GroupMember[groupPair](0)}), "has no field of type []soap.groupPair")
	assert.ErrorContains(t, UnmarshalModelGroups(newDecoder(), start, &struct {
		*GroupParentFixture
		M ModelGroupMember `xml:"a"`
	}{GroupParentFixture: &p, M: GroupMember[groupOuter](9)}), "no exported field 9")
	assert.ErrorContains(t, UnmarshalModelGroups(newDecoder(), start, &struct {
		*GroupParentFixture
		M ModelGroupMember `xml:"a"`
	}{GroupParentFixture: &p, M: GroupMember[groupOuter](0, 0)}), "is not a slice")
	assert.ErrorContains(t, UnmarshalModelGroups(newDecoder(), start, &struct {
		*GroupParentFixture
		M ModelGroupMember `xml:"a"`
	}{GroupParentFixture: &p, M: GroupMember[groupOuter]()}), "empty field path")
}

func TestModelGroupMember_OutsideUnmarshalModelGroups(t *testing.T) {
	var v struct {
		M ModelGroupMember `xml:"a"`
	}
	err := xml.Unmarshal([]byte(`<r><a>1</a></r>`), &v)
	assert.ErrorContains(t, err, "decoded outside UnmarshalModelGroups")
}

// GroupPtrFixture embeds its base by pointer, which is allocated when the
// group is found.
type GroupPtrFixture struct {
	Other int `xml:"other"`
	*GroupParentFixture
}

func (t *GroupPtrFixture) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	return UnmarshalModelGroups(d, start, &struct {
		*GroupPtrFixture
		UnmarshalXML struct{}         `xml:"-"`
		M0           ModelGroupMember `xml:"a"`
	}{GroupPtrFixture: t, M0: GroupMember[groupOuter](0)})
}

func TestUnmarshalModelGroups_EmbeddedPointer(t *testing.T) {
	var v GroupPtrFixture
	require.NoError(t, xml.Unmarshal([]byte(`<Parent xmlns="urn:g"><a>1</a><a>2</a></Parent>`), &v))
	require.NotNil(t, v.GroupParentFixture)
	assert.Equal(t, []groupOuter{{A: "1"}, {A: "2"}}, v.AGroup)
}
