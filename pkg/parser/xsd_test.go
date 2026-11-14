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
