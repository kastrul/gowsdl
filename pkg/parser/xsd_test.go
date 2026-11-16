// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package parser

import (
	"encoding/xml"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func elementNames(elements []*XSDElement) []string {
	names := make([]string, 0, len(elements))
	for _, el := range elements {
		names = append(names, el.Name)
	}
	return names
}

func TestXSDComplexType_NestedSequenceAndChoice(t *testing.T) {
	tests := []struct {
		name               string
		body               string
		wantSequence       []string
		wantSequenceChoice []string
		wantChoice         []string
	}{
		{
			name: "choice and sequence nested in sequence",
			body: `<sequence>
					<choice>
						<element name="categoryId" type="int"/>
						<element name="definition" type="tns:CategoryDefinition"/>
					</choice>
					<sequence>
						<element name="styleId" type="int" minOccurs="0" maxOccurs="unbounded"/>
						<element name="installed" type="tns:InstallationCause" minOccurs="0"/>
					</sequence>
				</sequence>`,
			wantSequence:       []string{"styleId", "installed"},
			wantSequenceChoice: []string{"categoryId", "definition"},
		},
		{
			name: "direct elements are kept alongside nested groups",
			body: `<sequence>
					<element name="first" type="string"/>
					<sequence>
						<element name="second" type="string"/>
					</sequence>
				</sequence>`,
			wantSequence: []string{"first", "second"},
		},
		{
			name: "deeply nested sequences and choices",
			body: `<sequence>
					<sequence>
						<sequence>
							<element name="deep" type="string"/>
						</sequence>
						<choice>
							<element name="optA" type="string"/>
							<sequence>
								<element name="optB" type="string"/>
							</sequence>
						</choice>
					</sequence>
					<choice>
						<choice>
							<element name="optC" type="string"/>
						</choice>
						<sequence>
							<element name="optD" type="string"/>
						</sequence>
					</choice>
				</sequence>`,
			wantSequence:       []string{"deep"},
			wantSequenceChoice: []string{"optC", "optD", "optA", "optB"},
		},
		{
			name: "groups nested in top-level choice",
			body: `<choice>
					<element name="a" type="string"/>
					<sequence>
						<element name="b" type="string"/>
						<element name="c" type="string"/>
					</sequence>
					<choice>
						<element name="d" type="string"/>
					</choice>
				</choice>`,
			wantChoice: []string{"a", "b", "c", "d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := `<complexType xmlns="http://www.w3.org/2001/XMLSchema" name="T" abstract="true">` +
				tt.body + `<attribute name="attr" type="string"/></complexType>`

			var ct XSDComplexType
			require.NoError(t, xml.Unmarshal([]byte(src), &ct))

			assert.Equal(t, "T", ct.Name)
			assert.True(t, ct.Abstract)
			assert.Equal(t, "complexType", ct.XMLName.Local)
			require.Len(t, ct.Attributes, 1)
			assert.Equal(t, "attr", ct.Attributes[0].Name)

			assert.ElementsMatch(t, tt.wantSequence, elementNames(ct.Sequence), "Sequence")
			assert.ElementsMatch(t, tt.wantSequenceChoice, elementNames(ct.SequenceChoice), "SequenceChoice")
			assert.ElementsMatch(t, tt.wantChoice, elementNames(ct.Choice), "Choice")
		})
	}
}

func TestXSDComplexType_NestedGroupsInSchema(t *testing.T) {
	src := `<schema xmlns="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:test">
		<complexType name="Outer">
			<sequence>
				<element name="inner">
					<complexType>
						<sequence>
							<sequence>
								<element name="nested" type="string"/>
							</sequence>
						</sequence>
					</complexType>
				</element>
			</sequence>
		</complexType>
	</schema>`

	var schema XSDSchema
	require.NoError(t, xml.Unmarshal([]byte(src), &schema))
	require.Len(t, schema.ComplexTypes, 1)

	outer := schema.ComplexTypes[0]
	require.Len(t, outer.Sequence, 1)
	inner := outer.Sequence[0].ComplexType
	require.NotNil(t, inner)
	assert.Equal(t, []string{"nested"}, elementNames(inner.Sequence))
}

func TestXSDComplexType_ComplexContentRestriction(t *testing.T) {
	src := `<complexType xmlns="http://www.w3.org/2001/XMLSchema" name="ClientIdentifierType">
		<complexContent>
			<restriction base="id:XRoadIdentifierType">
				<sequence>
					<element ref="id:xRoadInstance"/>
					<element ref="id:memberCode"/>
					<choice>
						<element ref="id:subsystemCode"/>
						<element ref="id:groupCode"/>
					</choice>
				</sequence>
				<attribute ref="id:objectType" use="required"/>
			</restriction>
		</complexContent>
	</complexType>`

	var ct XSDComplexType
	require.NoError(t, xml.Unmarshal([]byte(src), &ct))

	refs := func(elements []*XSDElement) []string {
		out := make([]string, 0, len(elements))
		for _, el := range elements {
			out = append(out, el.Ref)
		}
		return out
	}
	assert.Equal(t, []string{"id:xRoadInstance", "id:memberCode"}, refs(ct.Sequence))
	assert.Equal(t, []string{"id:subsystemCode", "id:groupCode"}, refs(ct.SequenceChoice))
	require.Len(t, ct.Attributes, 1)
	assert.Equal(t, "id:objectType", ct.Attributes[0].Ref)
	assert.Equal(t, "complexContent", ct.ComplexContent.XMLName.Local)
	assert.Empty(t, ct.ComplexContent.Extension.Base, "a restriction must not be read as an extension of its base")
}

func TestXSDComplexType_ComplexContentExtensionStillParsed(t *testing.T) {
	src := `<complexType xmlns="http://www.w3.org/2001/XMLSchema" name="Order">
		<complexContent>
			<extension base="tns:BaseEntity">
				<sequence><element name="status" type="string"/></sequence>
				<attribute name="priority" type="int"/>
			</extension>
		</complexContent>
	</complexType>`

	var ct XSDComplexType
	require.NoError(t, xml.Unmarshal([]byte(src), &ct))

	assert.Equal(t, "tns:BaseEntity", ct.ComplexContent.Extension.Base)
	assert.Equal(t, []string{"status"}, elementNames(ct.ComplexContent.Extension.Sequence))
	require.Len(t, ct.ComplexContent.Extension.Attributes, 1)
	assert.Empty(t, ct.Sequence)
	assert.Empty(t, ct.Attributes)
}

func TestXSDComplexType_InOptionalGroup(t *testing.T) {
	const src = `<xs:complexType name="T" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:sequence>
    <xs:element name="required"/>
    <xs:element name="ownMinOccurs" minOccurs="0"/>
    <xs:sequence minOccurs="1"><xs:element name="inRequiredSequence"/></xs:sequence>
    <xs:sequence minOccurs="0">
      <xs:element name="inOptionalSequence"/>
      <xs:sequence><xs:element name="deep"/></xs:sequence>
    </xs:sequence>
    <xs:choice><xs:element name="branch"/><xs:sequence><xs:element name="inBranch"/></xs:sequence></xs:choice>
    <xs:element name="nested">
      <xs:complexType><xs:sequence><xs:element name="inNested"/></xs:sequence></xs:complexType>
    </xs:element>
  </xs:sequence>
</xs:complexType>`
	var ct XSDComplexType
	require.NoError(t, xml.Unmarshal([]byte(src), &ct))

	got := map[string]bool{}
	for _, list := range [][]*XSDElement{ct.Sequence, ct.SequenceChoice, ct.Choice} {
		for _, el := range list {
			got[el.Name] = el.InOptionalGroup
			if el.ComplexType != nil {
				for _, inner := range el.ComplexType.Sequence {
					got[inner.Name] = inner.InOptionalGroup
				}
			}
		}
	}
	assert.Equal(t, map[string]bool{
		"required":           false,
		"ownMinOccurs":       false,
		"inRequiredSequence": false,
		"inOptionalSequence": true,
		"deep":               true,
		"branch":             true,
		"inBranch":           true,
		"nested":             false,
		"inNested":           false,
	}, got)
}

func TestXSDComplexType_InOptionalGroupResetAtNestedType(t *testing.T) {
	const src = `<xs:complexType name="T" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:choice>
    <xs:element name="branch">
      <xs:complexType><xs:sequence><xs:element name="inner"/></xs:sequence></xs:complexType>
    </xs:element>
  </xs:choice>
</xs:complexType>`
	var ct XSDComplexType
	require.NoError(t, xml.Unmarshal([]byte(src), &ct))
	require.Len(t, ct.Choice, 1)
	assert.True(t, ct.Choice[0].InOptionalGroup)
	require.Len(t, ct.Choice[0].ComplexType.Sequence, 1)
	assert.False(t, ct.Choice[0].ComplexType.Sequence[0].InOptionalGroup, "an element is required relative to its own type")
}

func TestXSDExtension_NestedGroupsFlattened(t *testing.T) {
	tests := []struct {
		name               string
		body               string
		wantSequence       []string
		wantChoice         []string
		wantSequenceChoice []string
	}{
		{
			name:       "choice",
			body:       `<xs:choice><xs:element name="a"/><xs:element name="b"/></xs:choice>`,
			wantChoice: []string{"a", "b"},
		},
		{
			name:               "sequence>choice",
			body:               `<xs:sequence><xs:element name="year"/><xs:choice><xs:element name="a"/><xs:element name="b"/></xs:choice></xs:sequence>`,
			wantSequence:       []string{"year"},
			wantSequenceChoice: []string{"a", "b"},
		},
		{
			name:               "sequence>sequence",
			body:               `<xs:sequence><xs:element name="a"/><xs:sequence><xs:element name="b"/><xs:choice><xs:element name="c"/></xs:choice></xs:sequence></xs:sequence>`,
			wantSequence:       []string{"a", "b"},
			wantSequenceChoice: []string{"c"},
		},
		{
			name:               "sequence>choice>sequence and choice",
			body:               `<xs:sequence><xs:choice><xs:sequence><xs:element name="a"/><xs:element name="b"/></xs:sequence><xs:element name="c"/><xs:choice><xs:element name="d"/></xs:choice></xs:choice><xs:element name="e"/></xs:sequence>`,
			wantSequence:       []string{"e"},
			wantSequenceChoice: []string{"c", "a", "b", "d"},
		},
		{
			name:       "choice>sequence and choice",
			body:       `<xs:choice><xs:element name="a"/><xs:sequence><xs:element name="b"/></xs:sequence><xs:choice><xs:element name="c"/></xs:choice></xs:choice>`,
			wantChoice: []string{"a", "b", "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := `<xs:complexType name="T" xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:complexContent><xs:extension base="Base">` +
				tt.body + `</xs:extension></xs:complexContent></xs:complexType>`
			var ct XSDComplexType
			require.NoError(t, xml.Unmarshal([]byte(src), &ct))
			ext := ct.ComplexContent.Extension
			assert.ElementsMatch(t, tt.wantSequence, elementNames(ext.Sequence), "Sequence")
			assert.ElementsMatch(t, tt.wantChoice, elementNames(ext.Choice), "Choice")
			assert.ElementsMatch(t, tt.wantSequenceChoice, elementNames(ext.SequenceChoice), "SequenceChoice")
			assert.Empty(t, ct.Sequence)
			assert.Empty(t, ct.Choice)
			assert.Empty(t, ct.SequenceChoice)
		})
	}
}

func TestXSDExtension_ChoiceMembersInOptionalGroup(t *testing.T) {
	const src = `<xs:complexType name="T" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexContent>
    <xs:extension base="Base">
      <xs:sequence>
        <xs:element name="required"/>
        <xs:sequence><xs:element name="inRequiredSequence"/></xs:sequence>
        <xs:choice>
          <xs:element name="branch"/>
          <xs:sequence><xs:element name="inBranch"/></xs:sequence>
        </xs:choice>
      </xs:sequence>
    </xs:extension>
  </xs:complexContent>
</xs:complexType>`
	var ct XSDComplexType
	require.NoError(t, xml.Unmarshal([]byte(src), &ct))

	ext := ct.ComplexContent.Extension
	got := map[string]bool{}
	for _, list := range [][]*XSDElement{ext.Sequence, ext.Choice, ext.SequenceChoice} {
		for _, el := range list {
			got[el.Name] = el.InOptionalGroup
		}
	}
	assert.Equal(t, map[string]bool{
		"required":           false,
		"inRequiredSequence": false,
		"branch":             true,
		"inBranch":           true,
	}, got)

	const topChoice = `<xs:complexType name="T" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexContent><xs:extension base="Base"><xs:choice><xs:element name="a"/></xs:choice></xs:extension></xs:complexContent>
</xs:complexType>`
	var choice XSDComplexType
	require.NoError(t, xml.Unmarshal([]byte(topChoice), &choice))
	require.Len(t, choice.ComplexContent.Extension.Choice, 1)
	assert.True(t, choice.ComplexContent.Extension.Choice[0].InOptionalGroup)
}

// byPosition returns the names of the elements of lists ordered by Position.
func byPosition(lists ...[]*XSDElement) []string {
	var all []*XSDElement
	for _, list := range lists {
		all = append(all, list...)
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].Position < all[j].Position })
	return elementNames(all)
}

func TestXSDComplexType_PositionIsDocumentOrder(t *testing.T) {
	const src = `<xs:complexType name="T" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:sequence>
    <xs:element name="a"/>
    <xs:choice><xs:element name="b"/><xs:sequence><xs:element name="c"/></xs:sequence></xs:choice>
    <xs:sequence><xs:element name="d"/><xs:choice><xs:element name="e"/></xs:choice></xs:sequence>
    <xs:element name="f">
      <xs:complexType>
        <xs:sequence><xs:element name="fA"/><xs:choice><xs:element name="fB"/></xs:choice><xs:element name="fC"/></xs:sequence>
      </xs:complexType>
    </xs:element>
    <xs:element name="g"/>
  </xs:sequence>
</xs:complexType>`
	var ct XSDComplexType
	require.NoError(t, xml.Unmarshal([]byte(src), &ct))

	assert.Equal(t, []string{"a", "f", "g", "d"}, elementNames(ct.Sequence), "Sequence keeps its flattened order")
	assert.Equal(t, []string{"a", "b", "c", "d", "e", "f", "g"}, byPosition(ct.Sequence, ct.SequenceChoice))
	for _, el := range append(ct.Sequence, ct.SequenceChoice...) {
		assert.Positive(t, el.Position, el.Name)
	}

	var f *XSDElement
	for _, el := range ct.Sequence {
		if el.Name == "f" {
			f = el
		}
	}
	require.NotNil(t, f)
	assert.Equal(t, []string{"fA", "fB", "fC"}, byPosition(f.ComplexType.Sequence, f.ComplexType.SequenceChoice))
}

func TestXSDExtension_PositionIsDocumentOrder(t *testing.T) {
	const src = `<xs:complexType name="T" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexContent>
    <xs:extension base="Base">
      <xs:sequence>
        <xs:element name="a"/>
        <xs:choice><xs:element name="b"/></xs:choice>
        <xs:sequence><xs:element name="c"/></xs:sequence>
        <xs:element name="d"/>
      </xs:sequence>
    </xs:extension>
  </xs:complexContent>
</xs:complexType>`
	var ct XSDComplexType
	require.NoError(t, xml.Unmarshal([]byte(src), &ct))
	ext := ct.ComplexContent.Extension
	assert.Equal(t, []string{"a", "b", "c", "d"}, byPosition(ext.Sequence, ext.Choice, ext.SequenceChoice))
}

func TestXSDSchema_GlobalElementHasNoPosition(t *testing.T) {
	const src = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="g" type="xs:string"/></xs:schema>`
	var s XSDSchema
	require.NoError(t, xml.Unmarshal([]byte(src), &s))
	require.Len(t, s.Elements, 1)
	assert.Zero(t, s.Elements[0].Position)
}

func TestXSDSchema_GroupDefinitions(t *testing.T) {
	const src = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:tns="urn:g" targetNamespace="urn:g">
  <xs:group name="G">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
      <xs:choice><xs:element name="b"/><xs:sequence><xs:element name="c"/></xs:sequence></xs:choice>
      <xs:group ref="tns:H" minOccurs="0" maxOccurs="2"/>
      <xs:element name="d">
        <xs:complexType><xs:sequence><xs:group ref="tns:Inner"/></xs:sequence></xs:complexType>
      </xs:element>
    </xs:sequence>
  </xs:group>
  <xs:group name="H"><xs:choice><xs:element name="h"/></xs:choice></xs:group>
</xs:schema>`
	var s XSDSchema
	require.NoError(t, xml.Unmarshal([]byte(src), &s))
	require.Len(t, s.Groups, 2)

	g := s.Groups[0]
	assert.Equal(t, "G", g.Name)
	assert.Equal(t, []string{"a", "d"}, elementNamesOf(g.Sequence), "Sequence is read as before")
	assert.Equal(t, []string{"a", "b", "c", "d"}, elementNames(g.Elements))
	optional := map[string]bool{}
	for _, el := range g.Elements {
		optional[el.Name] = el.InOptionalGroup
	}
	assert.Equal(t, map[string]bool{"a": false, "b": true, "c": true, "d": false}, optional)

	require.Len(t, g.GroupRefs, 1, "the reference inside d's type belongs to that type")
	ref := g.GroupRefs[0]
	assert.Equal(t, "tns:H", ref.Ref)
	assert.Equal(t, "0", ref.MinOccurs)
	assert.Equal(t, "2", ref.MaxOccurs)
	assert.False(t, ref.InOptionalGroup)
	assert.Greater(t, ref.Position, g.Elements[2].Position, "after c")
	assert.Less(t, ref.Position, g.Elements[3].Position, "before d")

	require.NotNil(t, g.Elements[3].ComplexType)
	require.Len(t, g.Elements[3].ComplexType.GroupRefs, 1)
	assert.Equal(t, "tns:Inner", g.Elements[3].ComplexType.GroupRefs[0].Ref)

	assert.Equal(t, []string{"h"}, elementNames(s.Groups[1].Elements))
	assert.True(t, s.Groups[1].Elements[0].InOptionalGroup)
}

func TestXSDComplexType_GroupRefs(t *testing.T) {
	const src = `<xs:complexType name="T" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:sequence>
    <xs:element name="a"/>
    <xs:group ref="tns:First"/>
    <xs:choice>
      <xs:group ref="tns:InChoice"/>
      <xs:element name="b"/>
    </xs:choice>
    <xs:sequence minOccurs="0"><xs:group ref="tns:InOptional"/></xs:sequence>
    <xs:group ref="tns:Optional" minOccurs="0"/>
    <xs:element name="c">
      <xs:complexType><xs:sequence><xs:group ref="tns:Nested"/></xs:sequence></xs:complexType>
    </xs:element>
  </xs:sequence>
</xs:complexType>`
	var ct XSDComplexType
	require.NoError(t, xml.Unmarshal([]byte(src), &ct))

	var refs []string
	optional := map[string]bool{}
	for _, r := range ct.GroupRefs {
		refs = append(refs, r.Ref)
		optional[r.Ref] = r.InOptionalGroup
	}
	assert.Equal(t, []string{"tns:First", "tns:InChoice", "tns:InOptional", "tns:Optional"}, refs)
	assert.Equal(t, map[string]bool{"tns:First": false, "tns:InChoice": true, "tns:InOptional": true, "tns:Optional": false}, optional)
	assert.Equal(t, "0", ct.GroupRefs[3].MinOccurs)

	positions := map[string]int{}
	for _, el := range append(ct.Sequence, ct.SequenceChoice...) {
		positions[el.Name] = el.Position
	}
	for _, r := range ct.GroupRefs {
		positions[r.Ref] = r.Position
	}
	assert.Less(t, positions["a"], positions["tns:First"])
	assert.Less(t, positions["tns:First"], positions["tns:InChoice"])
	assert.Less(t, positions["tns:InChoice"], positions["b"])
	assert.Less(t, positions["tns:Optional"], positions["c"])

	require.Len(t, ct.Sequence, 2)
	require.Len(t, ct.Sequence[1].ComplexType.GroupRefs, 1)
	assert.Equal(t, "tns:Nested", ct.Sequence[1].ComplexType.GroupRefs[0].Ref)
}

func TestXSDExtension_GroupRefs(t *testing.T) {
	const src = `<xs:complexType name="T" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexContent>
    <xs:extension base="Base">
      <xs:sequence>
        <xs:element name="a"/>
        <xs:group ref="tns:G"/>
        <xs:choice><xs:group ref="tns:H"/></xs:choice>
      </xs:sequence>
    </xs:extension>
  </xs:complexContent>
</xs:complexType>`
	var ct XSDComplexType
	require.NoError(t, xml.Unmarshal([]byte(src), &ct))
	assert.Empty(t, ct.GroupRefs, "the extension's references are its own")
	ext := ct.ComplexContent.Extension
	require.Len(t, ext.GroupRefs, 2)
	assert.Equal(t, "tns:G", ext.GroupRefs[0].Ref)
	assert.False(t, ext.GroupRefs[0].InOptionalGroup)
	assert.Equal(t, "tns:H", ext.GroupRefs[1].Ref)
	assert.True(t, ext.GroupRefs[1].InOptionalGroup)
	assert.Equal(t, []string{"a"}, elementNames(ext.Sequence))
	assert.Less(t, ext.Sequence[0].Position, ext.GroupRefs[0].Position)
}

func TestXSDComplexType_RestrictionGroupRefs(t *testing.T) {
	const src = `<xs:complexType name="T" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexContent>
    <xs:restriction base="xs:anyType">
      <xs:sequence><xs:group ref="tns:G"/></xs:sequence>
    </xs:restriction>
  </xs:complexContent>
</xs:complexType>`
	var ct XSDComplexType
	require.NoError(t, xml.Unmarshal([]byte(src), &ct))
	require.Len(t, ct.GroupRefs, 1)
	assert.Equal(t, "tns:G", ct.GroupRefs[0].Ref)
}

// elementNamesOf returns the names of elements.
func elementNamesOf(elements []XSDElement) []string {
	names := make([]string, 0, len(elements))
	for _, el := range elements {
		names = append(names, el.Name)
	}
	return names
}
