// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"encoding/xml"
	"slices"

	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/utils"
)

// repeatedSequenceName is the XMLName of the elements foldRepeatedSequences
// puts in place of the elements of a repeated sequence. No schema declares
// an element under it.
var repeatedSequenceName = xml.Name{Space: "urn:gowsdl:generator", Local: "repeatedSequence"}

// isRepeatedSequence reports whether el stands for a repeated sequence, as
// foldRepeatedSequences makes it. Its field is a slice of a generated type
// holding one occurrence of the sequence, the anonymous complex type of el,
// and its Name, with "Group" at the end, names that field.
func isRepeatedSequence(el *parser.XSDElement) bool {
	return el != nil && el.XMLName == repeatedSequenceName && el.ComplexType != nil
}

// foldRepeatedSequences replaces the elements of each xs:sequence that may
// occur more than once and holds more than one member, in the content model
// of every complex type, with one element standing for the sequence (see
// isRepeatedSequence), whose anonymous type holds the members. A member is
// an element, or a nested repeated sequence folded the same way; elements
// of nested choices and of sequences that occur at most once are members
// themselves. The members are made optional and repeated as they are within
// one occurrence of the sequence. A repeated sequence with one member is
// left alone: that member is a repeated element, or a nested sequence that
// repeats anyway. An xs:group reference with maxOccurs above 1 counts as a
// repeated sequence when the group's model group is a sequence. It runs
// after expandGroupRefs, which gives group members the compositors of
// their reference.
func foldRepeatedSequences(schemas []*parser.XSDSchema) {
	done := make(map[*parser.XSDComplexType]bool)
	var foldType func(ct *parser.XSDComplexType)
	var walk func(elements []*parser.XSDElement)
	walk = func(elements []*parser.XSDElement) {
		for _, el := range elements {
			if isRepeatedSequence(el) {
				walk(el.ComplexType.Sequence)
				continue
			}
			foldType(el.ComplexType)
		}
	}
	foldType = func(ct *parser.XSDComplexType) {
		if ct == nil || done[ct] {
			return
		}
		done[ct] = true
		foldContent(&ct.Sequence, &ct.Choice, &ct.SequenceChoice, &ct.All)
		ext := &ct.ComplexContent.Extension
		foldContent(&ext.Sequence, &ext.Choice, &ext.SequenceChoice)
		walk(contentElements(ct.Sequence, ct.Choice, ct.SequenceChoice, ct.All))
		walk(contentElements(ext.Sequence, ext.Choice, ext.SequenceChoice))
	}
	for _, schema := range schemas {
		for _, ct := range schema.ComplexTypes {
			foldType(ct)
		}
		for _, el := range schema.Elements {
			foldType(el.ComplexType)
		}
	}
}

// foldContent folds the repeated sequences of the content model whose
// elements lists hold. The elements of a folded sequence leave the lists,
// and the element standing for it joins the first list, at the position of
// its first member.
func foldContent(lists ...*[]*parser.XSDElement) {
	var all [][]*parser.XSDElement
	for _, list := range lists {
		all = append(all, *list)
	}
	elements := contentElements(all...)
	folded := foldRun(elements, 0)

	kept := make(map[*parser.XSDElement]bool, len(folded))
	for _, el := range folded {
		kept[el] = true
	}
	changed := false
	for _, list := range lists {
		n := len(*list)
		*list = slices.DeleteFunc(*list, func(el *parser.XSDElement) bool { return !kept[el] })
		changed = changed || len(*list) != n
	}
	if !changed {
		return
	}
	for _, el := range folded {
		if isRepeatedSequence(el) {
			*lists[0] = append(*lists[0], el)
		}
	}
}

// foldRun folds the repeated sequences among elements, a run of elements in
// document order, looking only at compositors from index depth of their
// Compositors on: those before enclose the whole run.
func foldRun(elements []*parser.XSDElement, depth int) []*parser.XSDElement {
	var out []*parser.XSDElement
	for i := 0; i < len(elements); {
		el := elements[i]
		k := repeatedSequenceIndex(el.Compositors, depth)
		if k < 0 {
			out = append(out, el)
			i++
			continue
		}
		id := el.Compositors[k].ID
		j := i + 1
		for j < len(elements) && len(elements[j].Compositors) > k && elements[j].Compositors[k].ID == id {
			j++
		}
		members := foldRun(elements[i:j], k+1)
		i = j
		if len(members) < 2 {
			out = append(out, members...)
			continue
		}
		for _, m := range members {
			if !isRepeatedSequence(m) {
				m.InOptionalGroup = optionalWithin(m.Compositors, k)
				m.InRepeatedGroup = repeatedWithin(m.Compositors, k)
			}
		}
		name := el.Compositors[k].Group
		if name == "" {
			name = firstMemberName(members[0])
		}
		out = append(out, &parser.XSDElement{
			XMLName:         repeatedSequenceName,
			Name:            name + "Group",
			MinOccurs:       "0",
			MaxOccurs:       "unbounded",
			InRepeatedGroup: true,
			Compositors:     slices.Clone(el.Compositors[:k]),
			Position:        members[0].Position,
			ComplexType:     &parser.XSDComplexType{Sequence: members},
		})
	}
	return out
}

// repeatedSequenceIndex returns the index of the first sequence in path,
// from index depth on, that may occur more than once, or -1.
func repeatedSequenceIndex(path parser.XSDCompositors, depth int) int {
	for k := depth; k < len(path); k++ {
		if path[k].Kind == "sequence" && maxOccursRepeats(path[k].MaxOccurs) {
			return k
		}
	}
	return -1
}

// optionalWithin reports whether a compositor in path after index k lets
// the element be absent from one occurrence of the compositor at k.
func optionalWithin(path parser.XSDCompositors, k int) bool {
	for _, c := range path[k+1:] {
		if particleOptional(c.MinOccurs, c.Kind == "choice") {
			return true
		}
	}
	return false
}

// repeatedWithin reports whether a sequence or choice in path after index
// k may occur more than once within one occurrence of the compositor at k.
func repeatedWithin(path parser.XSDCompositors, k int) bool {
	for _, c := range path[k+1:] {
		if c.Kind != "all" && maxOccursRepeats(c.MaxOccurs) {
			return true
		}
	}
	return false
}

// firstMemberName returns the name of the element a repeated sequence
// starting with el starts with.
func firstMemberName(el *parser.XSDElement) string {
	for isRepeatedSequence(el) {
		el = el.ComplexType.Sequence[0]
	}
	if el.Ref != "" {
		return utils.LocalName(el.Ref)
	}
	return el.Name
}

// repeatedSequenceXMLName returns the name in the xml tag of the field of
// el, a repeated sequence: that of its first element, which encoding/xml
// passes to the occurrence's MarshalXML and which never matches an element
// when decoding, since the generated UnmarshalXML hands every element of the
// sequence to a member field of that name first.
func (g *Generator) repeatedSequenceXMLName(el *parser.XSDElement) string {
	for isRepeatedSequence(el) {
		el = el.ComplexType.Sequence[0]
	}
	if el.Ref != "" {
		return g.refXMLName(el)
	}
	return el.Name
}
