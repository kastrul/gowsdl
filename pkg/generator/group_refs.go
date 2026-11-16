// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"sort"

	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/utils"
)

// expandGroupRefs adds the elements of the model groups that complex types
// include with xs:group ref to the element lists of those types, so that
// everything walking a type's elements, from the types template to type
// pruning, sees them as if they were declared in place. Each element is a
// copy that is optional when the reference is: when the reference has
// minOccurs="0" or sits in a choice or optional group of the type; and
// repeated when the reference has maxOccurs above 1 or sits in a repeated
// sequence or choice. The
// positions of a type's elements are renumbered so that they sort into
// document order with the copies standing where their reference does.
// Groups are found by local name in any of the schemas; a reference to an
// unknown group adds nothing. Expanding a type twice adds nothing more.
func expandGroupRefs(schemas []*parser.XSDSchema) {
	e := &groupExpander{
		defs:     make(map[string]*parser.XSDGroup),
		expanded: make(map[*parser.XSDGroup][]*parser.XSDElement),
		visiting: make(map[*parser.XSDGroup]bool),
		done:     make(map[*parser.XSDComplexType]bool),
	}
	for _, schema := range schemas {
		for _, g := range schema.Groups {
			if _, dup := e.defs[g.Name]; g.Name != "" && !dup {
				e.defs[g.Name] = g
			}
		}
	}
	for _, schema := range schemas {
		for _, ct := range schema.ComplexTypes {
			e.complexType(ct)
		}
		for _, el := range schema.Elements {
			e.complexType(el.ComplexType)
		}
	}
}

// groupExpander holds the state of one expandGroupRefs run.
type groupExpander struct {
	// defs holds the group definitions by name.
	defs map[string]*parser.XSDGroup
	// expanded holds the elements of each group definition already
	// expanded, nested references included, in document order.
	expanded map[*parser.XSDGroup][]*parser.XSDElement
	// visiting holds the definitions being expanded, to stop at a group
	// that refers to itself.
	visiting map[*parser.XSDGroup]bool
	// done holds the complex types already expanded.
	done map[*parser.XSDComplexType]bool
}

// complexType expands the group references of ct, of its complexContent
// extension and of the anonymous types of its elements.
func (e *groupExpander) complexType(ct *parser.XSDComplexType) {
	if ct == nil || e.done[ct] {
		return
	}
	e.done[ct] = true

	ct.Sequence = append(ct.Sequence, e.expand(ct.GroupRefs, ct.Sequence, ct.Choice, ct.SequenceChoice, ct.All)...)
	ext := &ct.ComplexContent.Extension
	ext.Sequence = append(ext.Sequence, e.expand(ext.GroupRefs, ext.Sequence, ext.Choice, ext.SequenceChoice)...)

	for _, list := range [][]*parser.XSDElement{ct.Sequence, ct.Choice, ct.SequenceChoice, ct.All, ext.Sequence, ext.Choice, ext.SequenceChoice} {
		for _, el := range list {
			e.complexType(el.ComplexType)
		}
	}
}

// expand returns copies of the elements of the groups refs name, made
// optional or repeated when their reference is, and renumbers the positions of the
// copies and of the elements of own, the lists holding the content model
// the references belong to, so that all of them sort into document order.
func (e *groupExpander) expand(refs []*parser.XSDGroup, own ...[]*parser.XSDElement) []*parser.XSDElement {
	if len(refs) == 0 {
		return nil
	}

	type particle struct {
		pos, sub int
		el       *parser.XSDElement
	}
	var all []particle
	for _, el := range contentElements(own...) {
		all = append(all, particle{pos: el.Position, el: el})
	}
	var added []*parser.XSDElement
	nextID := maxCompositorID(refs, own) + 1
	for _, ref := range refs {
		optional := particleOptional(ref.MinOccurs, ref.InOptionalGroup)
		repeated := ref.InRepeatedGroup || maxOccursRepeats(ref.MaxOccurs)
		ids := make(map[int]int)
		for i, member := range e.groupElements(ref) {
			c := copyElement(member)
			c.InOptionalGroup = c.InOptionalGroup || optional
			c.InRepeatedGroup = c.InRepeatedGroup || repeated
			c.Compositors = referencedCompositors(ref, member.Compositors, ids, &nextID)
			added = append(added, c)
			all = append(all, particle{pos: ref.Position, sub: i + 1, el: c})
		}
	}

	sort.SliceStable(all, func(i, j int) bool {
		if all[i].pos != all[j].pos {
			return all[i].pos < all[j].pos
		}
		return all[i].sub < all[j].sub
	})
	for i, p := range all {
		p.el.Position = i + 1
	}
	return added
}

// referencedCompositors returns the compositors of a copy of a member of the
// group ref names: those of the reference, then those of the member within
// the group definition. The outermost of the latter, the definition's own
// model group, takes the reference's minOccurs and maxOccurs when they make
// it optional or repeated, as the reference stands for the model group. The
// member's compositors get IDs unused in the reference's type or
// definition: ids maps the definition's IDs, the same for every member of
// one reference, to new ones taken from next.
func referencedCompositors(ref *parser.XSDGroup, member parser.XSDCompositors, ids map[int]int, next *int) parser.XSDCompositors {
	out := make(parser.XSDCompositors, 0, len(ref.Compositors)+len(member))
	out = append(out, ref.Compositors...)
	for i, c := range member {
		if i == 0 {
			if particleOptional(ref.MinOccurs, false) {
				c.MinOccurs = "0"
			}
			if maxOccursRepeats(ref.MaxOccurs) {
				c.MaxOccurs = ref.MaxOccurs
			}
		}
		id, ok := ids[c.ID]
		if !ok {
			id = *next
			*next++
			ids[c.ID] = id
		}
		c.ID = id
		out = append(out, c)
	}
	return out
}

// maxCompositorID returns the largest compositor ID used by refs and by the
// elements of lists.
func maxCompositorID(refs []*parser.XSDGroup, lists [][]*parser.XSDElement) int {
	highest := 0
	note := func(path parser.XSDCompositors) {
		for _, c := range path {
			highest = max(highest, c.ID)
		}
	}
	for _, ref := range refs {
		note(ref.Compositors)
	}
	for _, list := range lists {
		for _, el := range list {
			note(el.Compositors)
		}
	}
	return highest
}

// groupElements returns the elements of the group definition ref names, the
// elements of the groups it refers to included, in document order, with
// optionality relative to the definition. The result is shared: callers copy
// the elements before changing them.
func (e *groupExpander) groupElements(ref *parser.XSDGroup) []*parser.XSDElement {
	def := e.defs[utils.LocalName(ref.Ref)]
	if def == nil || e.visiting[def] {
		return nil
	}
	if elements, ok := e.expanded[def]; ok {
		return elements
	}

	e.visiting[def] = true
	own := make([]*parser.XSDElement, 0, len(def.Elements))
	for _, el := range def.Elements {
		c := *el
		own = append(own, &c)
	}
	elements := contentElements(own, e.expand(def.GroupRefs, own))
	delete(e.visiting, def)

	e.expanded[def] = elements
	return elements
}

// copyElement returns a copy of el whose anonymous complex type, if any, is
// a copy too, so that each use of a group gets its own anonymous types, named
// after the type using the group as if they were declared there.
func copyElement(el *parser.XSDElement) *parser.XSDElement {
	c := *el
	if el.ComplexType != nil {
		c.ComplexType = copyComplexType(el.ComplexType)
	}
	return &c
}

// copyComplexType returns a copy of ct with copies of its elements, and of
// their anonymous types, in every element list. Everything else is shared.
func copyComplexType(ct *parser.XSDComplexType) *parser.XSDComplexType {
	c := *ct
	copyList := func(list []*parser.XSDElement) []*parser.XSDElement {
		if list == nil {
			return nil
		}
		out := make([]*parser.XSDElement, len(list))
		for i, el := range list {
			out[i] = copyElement(el)
		}
		return out
	}
	c.Sequence = copyList(ct.Sequence)
	c.Choice = copyList(ct.Choice)
	c.SequenceChoice = copyList(ct.SequenceChoice)
	c.All = copyList(ct.All)
	ext := &c.ComplexContent.Extension
	ext.Sequence = copyList(ext.Sequence)
	ext.Choice = copyList(ext.Choice)
	ext.SequenceChoice = copyList(ext.SequenceChoice)
	return &c
}
