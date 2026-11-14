package parser

import (
	"encoding/xml"
	"strings"
)

type traverseMode int32

const (
	refResolution traverseMode = iota
)

// Traverser walks an XSD schema and resolves element references against a
// set of schemas. Create one with NewTraverser.
type Traverser struct {
	c   *XSDSchema
	all []*XSDSchema
	tm  traverseMode
}

// NewTraverser creates a new traverser for schema resolution
func NewTraverser(c *XSDSchema, all []*XSDSchema) *Traverser {
	return &Traverser{
		c:   c,
		all: all,
		tm:  refResolution, // default traverse mode is refResolution
	}
}

// Traverse performs the traversal
func (t *Traverser) Traverse() {
	t.tm = refResolution

	for _, ct := range t.c.ComplexTypes {
		t.traverseComplexType(ct)
	}
	for _, st := range t.c.SimpleType {
		t.traverseSimpleType(st)
	}
	for _, elm := range t.c.Elements {
		t.traverseElement(elm)
	}
}

func (t *Traverser) traverseElements(ct []*XSDElement) {
	for _, elm := range ct {
		t.traverseElement(elm)
	}
}

func (t *Traverser) traverseElement(elm *XSDElement) {
	// Check if we are in ref resolution mode
	if t.tm == refResolution && elm.Ref != "" {
		refElm := t.getGlobalElement(elm.Ref)
		if refElm != nil && refElm.Ref == "" {
			// Copy properties from referenced element
			t.traverseElement(refElm)
			elm.Name = refElm.Name
			elm.Type = refElm.Type
			elm.Nillable = refElm.Nillable
			if elm.MinOccurs == "" {
				elm.MinOccurs = refElm.MinOccurs
			}
			if elm.MaxOccurs == "" {
				elm.MaxOccurs = refElm.MaxOccurs
			}
			// Copy complex/simple type if element doesn't have its own
			if elm.ComplexType == nil {
				elm.ComplexType = refElm.ComplexType
			}
			if elm.SimpleType == nil {
				elm.SimpleType = refElm.SimpleType
			}
		}
	}

	if elm.ComplexType != nil {
		t.traverseComplexType(elm.ComplexType)
	}
	if elm.SimpleType != nil {
		t.traverseSimpleType(elm.SimpleType)
	}
}

func (t *Traverser) traverseSimpleType(_ *XSDSimpleType) {
}

func (t *Traverser) traverseComplexType(ct *XSDComplexType) {
	t.traverseElements(ct.Sequence)
	t.traverseElements(ct.Choice)
	t.traverseElements(ct.SequenceChoice)
	t.traverseElements(ct.All)
	t.traverseAttributes(ct.Attributes)
	t.traverseAttributes(ct.ComplexContent.Extension.Attributes)
	t.traverseElements(ct.ComplexContent.Extension.Sequence)
	t.traverseElements(ct.ComplexContent.Extension.Choice)
	t.traverseElements(ct.ComplexContent.Extension.SequenceChoice)
	t.traverseAttributes(ct.SimpleContent.Extension.Attributes)
}

func (t *Traverser) traverseAttributes(attrs []*XSDAttribute) {
	for _, attr := range attrs {
		t.traverseAttribute(attr)
	}
}

func (t *Traverser) traverseAttribute(attr *XSDAttribute) {
	// Check if we are in ref resolution mode
	if t.tm != refResolution {
		return
	}

	if attr.Ref != "" {
		refAttr := t.getGlobalAttribute(attr.Ref)
		if refAttr != nil && refAttr.Ref == "" {
			t.traverseAttribute(refAttr)
			attr.Name = refAttr.Name
			attr.Type = refAttr.Type
			if attr.Fixed == "" {
				attr.Fixed = refAttr.Fixed
			}
		}
	} else if attr.Type == "" {
		if attr.SimpleType != nil {
			t.traverseSimpleType(attr.SimpleType)
			attr.Type = attr.SimpleType.Restriction.Base
		}
	}
}

func (t *Traverser) getGlobalAttribute(name string) *XSDAttribute {
	ref := t.qname(name)

	for _, schema := range t.all {
		if schema.TargetNamespace == ref.Space {
			for _, attr := range schema.Attributes {
				if attr.Name == ref.Local {
					return attr
				}
			}
		}
	}

	return nil
}

func (t *Traverser) getGlobalElement(name string) *XSDElement {
	ref := t.qname(name)

	for _, schema := range t.all {
		if schema.TargetNamespace == ref.Space {
			for _, elm := range schema.Elements {
				if elm.Name == ref.Local {
					return elm
				}
			}
		}
	}

	return nil
}

// qname resolves QName into xml.Name.
func (t *Traverser) qname(name string) (qname xml.Name) {
	x := strings.SplitN(name, ":", 2)
	if len(x) == 1 {
		qname.Local = x[0]
	} else {
		qname.Local = x[1]
		qname.Space = x[0]
		if ns, ok := t.c.Xmlns[qname.Space]; ok {
			qname.Space = ns
		}
	}

	return qname
}
