// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package parser

import (
	"encoding/xml"
	"io"
	"sort"
	"strconv"
	"strings"
)

const xmlschema11 = "http://www.w3.org/2001/XMLSchema"

// XSDSchema represents an entire Schema structure.
type XSDSchema struct {
	XMLName            xml.Name          `xml:"schema"`
	Xmlns              map[string]string `xml:"-"` // prefix -> namespace URI; "" holds the default namespace
	Tns                string            `xml:"xmlns tns,attr"`
	Xs                 string            `xml:"xmlns xs,attr"`
	Version            string            `xml:"version,attr"`
	TargetNamespace    string            `xml:"targetNamespace,attr"`
	ElementFormDefault string            `xml:"elementFormDefault,attr"`
	Includes           []*XSDInclude     `xml:"include"`
	Imports            []*XSDImport      `xml:"import"`
	Elements           []*XSDElement     `xml:"element"`
	Attributes         []*XSDAttribute   `xml:"attribute"`
	ComplexTypes       []*XSDComplexType `xml:"complexType"` // global
	SimpleType         []*XSDSimpleType  `xml:"simpleType"`
	// Groups holds the schema's named model group definitions, xs:group
	// with a name, which complex types include with xs:group ref.
	Groups []*XSDGroup `xml:"group"`
}

// UnmarshalXML implements interface xml.Unmarshaler for XSDSchema.
func (s *XSDSchema) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	s.Xmlns = make(map[string]string)
	s.XMLName = start.Name
	for _, attr := range start.Attr {
		if attr.Name.Space == "xmlns" {
			s.Xmlns[attr.Name.Local] = attr.Value
			continue
		}
		if attr.Name.Space == "" && attr.Name.Local == "xmlns" {
			s.Xmlns[""] = attr.Value
			continue
		}

		switch attr.Name.Local {
		case "version":
			s.Version = attr.Value
		case "targetNamespace":
			s.TargetNamespace = attr.Value
		case "elementFormDefault":
			s.ElementFormDefault = attr.Value
		}
	}

Loop:
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Space != xmlschema11 {
				if err := d.Skip(); err != nil {
					return err
				}
				continue Loop
			}

			switch t.Name.Local {
			case "include":
				x := new(XSDInclude)
				if err := d.DecodeElement(x, &t); err != nil {
					return err
				}
				s.Includes = append(s.Includes, x)
			case "import":
				x := new(XSDImport)
				if err := d.DecodeElement(x, &t); err != nil {
					return err
				}
				s.Imports = append(s.Imports, x)
			case "element":
				x := new(XSDElement)
				if err := d.DecodeElement(x, &t); err != nil {
					return err
				}
				s.Elements = append(s.Elements, x)
			case "attribute":
				x := new(XSDAttribute)
				if err := d.DecodeElement(x, &t); err != nil {
					return err
				}
				s.Attributes = append(s.Attributes, x)
			case "complexType":
				x := new(XSDComplexType)
				if err := d.DecodeElement(x, &t); err != nil {
					return err
				}
				s.ComplexTypes = append(s.ComplexTypes, x)
			case "simpleType":
				x := new(XSDSimpleType)
				if err := d.DecodeElement(x, &t); err != nil {
					return err
				}
				s.SimpleType = append(s.SimpleType, x)
			case "group":
				x := new(XSDGroup)
				if err := d.DecodeElement(x, &t); err != nil {
					return err
				}
				s.Groups = append(s.Groups, x)
			default:
				if err := d.Skip(); err != nil {
					return err
				}
				continue Loop
			}
		case xml.EndElement:
			break Loop
		}
	}

	return nil
}

// XSDInclude represents schema includes.
type XSDInclude struct {
	SchemaLocation string `xml:"schemaLocation,attr"`
}

// XSDImport represents XSD imports within the main schema.
type XSDImport struct {
	XMLName        xml.Name `xml:"import"`
	SchemaLocation string   `xml:"schemaLocation,attr"`
	Namespace      string   `xml:"namespace,attr"`
}

// XSDElement represents a Schema element.
type XSDElement struct {
	XMLName     xml.Name        `xml:"element"`
	Name        string          `xml:"name,attr"`
	Doc         string          `xml:"annotation>documentation"`
	Nillable    bool            `xml:"nillable,attr"`
	Type        string          `xml:"type,attr"`
	Ref         string          `xml:"ref,attr"`
	MinOccurs   string          `xml:"minOccurs,attr"`
	MaxOccurs   string          `xml:"maxOccurs,attr"`
	ComplexType *XSDComplexType `xml:"complexType"` // local
	SimpleType  *XSDSimpleType  `xml:"simpleType"`
	Groups      []*XSDGroup     `xml:"group"`
	// InOptionalGroup reports that a model group between the element and
	// the complex type declaring it lets the element be absent whatever its
	// own minOccurs: a choice, or a sequence or all with minOccurs="0". The
	// parser sets it; it is always false for global elements.
	InOptionalGroup bool `xml:"urn:gowsdl:parser inOptionalGroup,attr"`
	// InRepeatedGroup reports that a model group between the element and
	// the complex type declaring it may occur more than once: a sequence or
	// choice with maxOccurs "unbounded" or above 1. The element may then
	// occur several times in one instance whatever its own maxOccurs. The
	// parser sets it; it is always false for global elements.
	InRepeatedGroup bool `xml:"urn:gowsdl:parser inRepeatedGroup,attr"`
	// Position is the 1-based document order of the element's declaration
	// among the element declarations of the complex type declaring it, so
	// that its elements can be put in schema order whichever of Sequence,
	// Choice, SequenceChoice and All holds them. The parser sets it; it is 0
	// for global elements.
	Position int `xml:"urn:gowsdl:parser position,attr"`
}

// XSDAny represents a Schema element.
type XSDAny struct {
	XMLName         xml.Name `xml:"any"`
	Doc             string   `xml:"annotation>documentation"`
	MinOccurs       string   `xml:"minOccurs,attr"`
	MaxOccurs       string   `xml:"maxOccurs,attr"`
	Namespace       string   `xml:"namespace,attr"`
	ProcessContents string   `xml:"processContents,attr"`
}

// XSDComplexType represents a Schema complex type.
type XSDComplexType struct {
	XMLName        xml.Name          `xml:"complexType"`
	Abstract       bool              `xml:"abstract,attr"`
	Name           string            `xml:"name,attr"`
	Mixed          bool              `xml:"mixed,attr"`
	Sequence       []*XSDElement     `xml:"sequence>element"`
	Choice         []*XSDElement     `xml:"choice>element"`
	SequenceChoice []*XSDElement     `xml:"sequence>choice>element"`
	All            []*XSDElement     `xml:"all>element"`
	ComplexContent XSDComplexContent `xml:"complexContent"`
	SimpleContent  XSDSimpleContent  `xml:"simpleContent"`
	Attributes     []*XSDAttribute   `xml:"attribute"`
	Any            []*XSDAny         `xml:"sequence>any"`
	// GroupRefs holds the xs:group references in the type's own content
	// model, at any depth of its sequences, choices and alls and in a
	// complexContent restriction, in document order. The elements of the
	// groups they name are not in the lists above. The references of a
	// complexContent extension are in its own GroupRefs.
	GroupRefs []*XSDGroup `xml:"-"`
}

// xsdModelGroup captures a nested sequence or choice so that its elements can
// be flattened into the enclosing complex type.
type xsdModelGroup struct {
	Elements   []*XSDElement    `xml:"element"`
	Sequences  []*xsdModelGroup `xml:"sequence"`
	Choices    []*xsdModelGroup `xml:"choice"`
	Alls       []*xsdModelGroup `xml:"all"`
	Attributes []*XSDAttribute  `xml:"attribute"`
}

// UnmarshalXML implements xml.Unmarshaler for XSDComplexType. Elements of
// sequences and choices nested inside the top-level sequence or choice are
// flattened into Sequence, SequenceChoice and Choice. The particles and
// attributes of a complexContent restriction restate the type's whole content
// model, so they are read as if declared directly on the complex type.
func (ct *XSDComplexType) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type Base XSDComplexType
	aux := struct {
		*Base
		xsdNestedGroups
		ComplexContent *struct {
			XMLName     xml.Name       `xml:"complexContent"`
			Extension   XSDExtension   `xml:"extension"`
			Restriction *xsdModelGroup `xml:"restriction"`
		} `xml:"complexContent"`
	}{Base: (*Base)(ct)}

	tokens, err := readElementTokens(d, start)
	if err != nil {
		return err
	}
	markOptionalGroupMembers(tokens)
	markPositions(tokens)
	ct.GroupRefs = collectGroupRefs(tokens, "sequence", "choice", "all", "complexContent", "restriction")
	replay := xml.NewTokenDecoder(&tokenReplay{tokens: tokens})
	first, err := replay.Token()
	if err != nil {
		return err
	}
	replayStart, _ := first.(xml.StartElement)
	if err := replay.DecodeElement(&aux, &replayStart); err != nil {
		return err
	}
	ct.XMLName = start.Name

	if cc := aux.ComplexContent; cc != nil {
		ct.ComplexContent = XSDComplexContent{XMLName: cc.XMLName, Extension: cc.Extension}
		if r := cc.Restriction; r != nil {
			for _, seq := range r.Sequences {
				seq.flattenSequence(&ct.Sequence, &ct.SequenceChoice)
			}
			for _, choice := range r.Choices {
				choice.flattenAll(&ct.Choice)
			}
			for _, all := range r.Alls {
				all.flattenAll(&ct.All)
			}
			ct.Attributes = append(ct.Attributes, r.Attributes...)
		}
	}

	aux.flatten(&ct.Sequence, &ct.SequenceChoice, &ct.Choice)
	return nil
}

// xsdNestedGroups captures the sequences and choices nested inside the
// top-level sequence or choice of a complex type or of a complexContent
// extension, whose elements the struct tags of XSDComplexType and
// XSDExtension do not reach.
type xsdNestedGroups struct {
	SequenceSequences       []*xsdModelGroup `xml:"sequence>sequence"`
	SequenceChoiceSequences []*xsdModelGroup `xml:"sequence>choice>sequence"`
	SequenceChoiceChoices   []*xsdModelGroup `xml:"sequence>choice>choice"`
	ChoiceSequences         []*xsdModelGroup `xml:"choice>sequence"`
	ChoiceChoices           []*xsdModelGroup `xml:"choice>choice"`
}

// flatten appends the elements of the nested groups to seq, seqChoice and
// choice: those of a sequence nested in the top-level sequence to seq, and
// every element below a choice to seqChoice or choice, by whether that
// choice is inside the top-level sequence or is the top-level choice.
func (n *xsdNestedGroups) flatten(seq, seqChoice, choice *[]*XSDElement) {
	for _, g := range n.SequenceSequences {
		g.flattenSequence(seq, seqChoice)
	}
	for _, g := range append(n.SequenceChoiceSequences, n.SequenceChoiceChoices...) {
		g.flattenAll(seqChoice)
	}
	for _, g := range append(n.ChoiceSequences, n.ChoiceChoices...) {
		g.flattenAll(choice)
	}
}

// optionalGroupAttr is the attribute markOptionalGroupMembers adds to element
// declarations, read into XSDElement.InOptionalGroup.
var optionalGroupAttr = xml.Name{Space: "urn:gowsdl:parser", Local: "inOptionalGroup"}

// readElementTokens returns copies of start and of every token up to and
// including its matching end element, consuming them from d.
func readElementTokens(d *xml.Decoder, start xml.StartElement) ([]xml.Token, error) {
	tokens := []xml.Token{start.Copy()}
	for depth := 1; depth > 0; {
		t, err := d.Token()
		if err != nil {
			return nil, err
		}
		switch t.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		}
		tokens = append(tokens, xml.CopyToken(t))
	}
	return tokens, nil
}

// markOptionalGroupMembers sets optionalGroupAttr on each element declaration
// and group reference in tokens that a model group may leave out: one with a
// choice, or a sequence or all with minOccurs="0", between it and its nearest
// enclosing complexType. It sets repeatedGroupAttr on those a model group may
// repeat: one with a sequence or choice whose maxOccurs is "unbounded" or
// above 1 in between. Any such attribute already present is replaced, so the
// result is the same when tokens of a nested complex type are marked again.
func markOptionalGroupMembers(tokens []xml.Token) {
	var stack []xml.StartElement
	for i, t := range tokens {
		switch t := t.(type) {
		case xml.StartElement:
			if t.Name.Local == "element" || isGroupRef(t) {
				attrs := t.Attr[:0:0]
				for _, a := range t.Attr {
					if a.Name != optionalGroupAttr && a.Name != repeatedGroupAttr {
						attrs = append(attrs, a)
					}
				}
				if inOptionalGroup(stack) {
					attrs = append(attrs, xml.Attr{Name: optionalGroupAttr, Value: "true"})
				}
				if inRepeatedGroup(stack) {
					attrs = append(attrs, xml.Attr{Name: repeatedGroupAttr, Value: "true"})
				}
				t.Attr = attrs
				tokens[i] = t
			}
			stack = append(stack, t)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		}
	}
}

// repeatedGroupAttr is the attribute markOptionalGroupMembers adds to
// element declarations and group references inside a repeated model group,
// read into their InRepeatedGroup.
var repeatedGroupAttr = xml.Name{Space: "urn:gowsdl:parser", Local: "inRepeatedGroup"}

// inRepeatedGroup reports whether the open elements in stack, read from the
// innermost up to the nearest complexType, include a sequence or choice that
// may occur more than once.
func inRepeatedGroup(stack []xml.StartElement) bool {
	for i := len(stack) - 1; i >= 0; i-- {
		switch stack[i].Name.Local {
		case "complexType":
			return false
		case "sequence", "choice":
			for _, a := range stack[i].Attr {
				if a.Name.Local == "maxOccurs" && a.Name.Space == "" && maxOccursRepeats(a.Value) {
					return true
				}
			}
		}
	}
	return false
}

// maxOccursRepeats reports whether maxOccurs is "unbounded" or a number
// above 1. A value that is neither is taken to repeat.
func maxOccursRepeats(maxOccurs string) bool {
	maxOccurs = strings.TrimSpace(maxOccurs)
	if maxOccurs == "" {
		return false
	}
	n, err := strconv.Atoi(maxOccurs)
	return err != nil || n > 1
}

// positionAttr is the attribute markPositions adds to element declarations,
// read into XSDElement.Position.
var positionAttr = xml.Name{Space: "urn:gowsdl:parser", Local: "position"}

// markPositions sets positionAttr on each element declaration and group
// reference in tokens to its 1-based document order among them, replacing
// any positionAttr already present. Particles of a nested complex type are
// numbered along with the rest and renumbered when that type is
// unmarshalled; either way their order among themselves is their document
// order.
func markPositions(tokens []xml.Token) {
	n := 0
	for i, t := range tokens {
		t, ok := t.(xml.StartElement)
		if !ok || (t.Name.Local != "element" && !isGroupRef(t)) {
			continue
		}
		n++
		attrs := t.Attr[:0:0]
		for _, a := range t.Attr {
			if a.Name != positionAttr {
				attrs = append(attrs, a)
			}
		}
		attrs = append(attrs, xml.Attr{Name: positionAttr, Value: strconv.Itoa(n)})
		t.Attr = attrs
		tokens[i] = t
	}
}

// isGroupRef reports whether t starts an xs:group reference.
func isGroupRef(t xml.StartElement) bool {
	if t.Name.Local != "group" {
		return false
	}
	for _, a := range t.Attr {
		if a.Name.Space == "" && a.Name.Local == "ref" {
			return true
		}
	}
	return false
}

// collectGroupRefs returns the group references in tokens, the tokens of one
// element, that are reached from it through elements named in through only,
// so that references inside a nested element declaration, or in a part of
// the content model another type records, are left out. The references are
// read from their attributes, including those markOptionalGroupMembers and
// markPositions added.
func collectGroupRefs(tokens []xml.Token, through ...string) []*XSDGroup {
	pass := make(map[string]bool, len(through))
	for _, name := range through {
		pass[name] = true
	}
	var refs []*XSDGroup
	blocked := 0
	depth := 0
	for _, t := range tokens {
		switch t := t.(type) {
		case xml.StartElement:
			depth++
			if depth == 1 {
				continue
			}
			if blocked == 0 && isGroupRef(t) {
				refs = append(refs, groupRefFromAttrs(t.Attr))
			}
			if blocked > 0 || !pass[t.Name.Local] {
				blocked++
			}
		case xml.EndElement:
			depth--
			if blocked > 0 {
				blocked--
			}
		}
	}
	return refs
}

// groupRefFromAttrs returns the XSDGroup for a group reference with attrs.
func groupRefFromAttrs(attrs []xml.Attr) *XSDGroup {
	g := new(XSDGroup)
	for _, a := range attrs {
		switch {
		case a.Name == optionalGroupAttr:
			g.InOptionalGroup = strings.TrimSpace(a.Value) == "true"
		case a.Name == repeatedGroupAttr:
			g.InRepeatedGroup = strings.TrimSpace(a.Value) == "true"
		case a.Name == positionAttr:
			g.Position, _ = strconv.Atoi(a.Value)
		case a.Name.Space != "":
		case a.Name.Local == "ref":
			g.Ref = a.Value
		case a.Name.Local == "name":
			g.Name = a.Value
		case a.Name.Local == "minOccurs":
			g.MinOccurs = a.Value
		case a.Name.Local == "maxOccurs":
			g.MaxOccurs = a.Value
		}
	}
	return g
}

// inOptionalGroup reports whether the open elements in stack, read from the
// innermost up to the nearest complexType, include a choice or a sequence or
// all with minOccurs="0".
func inOptionalGroup(stack []xml.StartElement) bool {
	for i := len(stack) - 1; i >= 0; i-- {
		switch stack[i].Name.Local {
		case "complexType":
			return false
		case "choice":
			return true
		case "sequence", "all":
			for _, a := range stack[i].Attr {
				if a.Name.Local == "minOccurs" && a.Name.Space == "" && strings.TrimSpace(a.Value) == "0" {
					return true
				}
			}
		}
	}
	return false
}

// tokenReplay is an xml.TokenReader returning a recorded token sequence.
type tokenReplay struct {
	tokens []xml.Token
}

// Token returns the next recorded token, or io.EOF after the last one.
func (r *tokenReplay) Token() (xml.Token, error) {
	if len(r.tokens) == 0 {
		return nil, io.EOF
	}
	t := r.tokens[0]
	r.tokens = r.tokens[1:]
	return t, nil
}

// flattenSequence appends the group's elements to seq, descending into nested
// sequences, and appends every element below a nested choice to choice.
func (g *xsdModelGroup) flattenSequence(seq, choice *[]*XSDElement) {
	*seq = append(*seq, g.Elements...)
	for _, s := range g.Sequences {
		s.flattenSequence(seq, choice)
	}
	for _, c := range g.Choices {
		c.flattenAll(choice)
	}
}

// flattenAll appends every element in the group and its nested groups to out.
func (g *xsdModelGroup) flattenAll(out *[]*XSDElement) {
	*out = append(*out, g.Elements...)
	for _, s := range g.Sequences {
		s.flattenAll(out)
	}
	for _, c := range g.Choices {
		c.flattenAll(out)
	}
}

// XSDGroup element is used to define a group of elements to be used in complex type definitions.
type XSDGroup struct {
	Name     string       `xml:"name,attr"`
	Ref      string       `xml:"ref,attr"`
	Sequence []XSDElement `xml:"sequence>element"`
	Choice   []XSDElement `xml:"choice>element"`
	All      []XSDElement `xml:"all>element"`
	// MinOccurs and MaxOccurs are those of a group reference.
	MinOccurs string `xml:"minOccurs,attr"`
	MaxOccurs string `xml:"maxOccurs,attr"`
	// InOptionalGroup reports, for a group reference in a complex type or
	// in a group definition, that a choice, or a sequence or all with
	// minOccurs="0", between the reference and the type or definition lets
	// the group's elements be absent whatever the reference's own minOccurs.
	InOptionalGroup bool `xml:"urn:gowsdl:parser inOptionalGroup,attr"`
	// InRepeatedGroup reports, for a group reference, that a sequence or
	// choice between the reference and its type or definition may occur
	// more than once, so that the group's elements may repeat whatever the
	// reference's own maxOccurs.
	InRepeatedGroup bool `xml:"urn:gowsdl:parser inRepeatedGroup,attr"`
	// Position is a group reference's document order among the element
	// declarations and group references of its complex type or group
	// definition, comparable with their XSDElement.Position.
	Position int `xml:"urn:gowsdl:parser position,attr"`
	// Elements holds every element declaration of a group definition's
	// model group, at any depth of its sequences and choices, in document
	// order. Their InOptionalGroup and Position are relative to the
	// definition. Elements of the groups it refers to are not included.
	Elements []*XSDElement `xml:"-"`
	// GroupRefs holds the group references in a group definition's model
	// group, at any depth, in document order.
	GroupRefs []*XSDGroup `xml:"-"`
}

// UnmarshalXML implements xml.Unmarshaler for XSDGroup. Besides the fields
// read from attributes and from the top-level sequence, choice or all, it
// fills Elements and GroupRefs for a group definition.
func (g *XSDGroup) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	tokens, err := readElementTokens(d, start)
	if err != nil {
		return err
	}
	inner := tokens[1 : len(tokens)-1]
	markOptionalGroupMembers(inner)
	markPositions(inner)

	type Base XSDGroup
	if err := decodeTokens(tokens, (*Base)(g)); err != nil {
		return err
	}
	var content struct {
		Sequences []*xsdModelGroup `xml:"sequence"`
		Choices   []*xsdModelGroup `xml:"choice"`
		Alls      []*xsdModelGroup `xml:"all"`
	}
	if err := decodeTokens(tokens, &content); err != nil {
		return err
	}
	for _, groups := range [][]*xsdModelGroup{content.Sequences, content.Choices, content.Alls} {
		for _, mg := range groups {
			mg.flattenAll(&g.Elements)
		}
	}
	sort.SliceStable(g.Elements, func(i, j int) bool { return g.Elements[i].Position < g.Elements[j].Position })
	g.GroupRefs = collectGroupRefs(tokens, "sequence", "choice", "all")
	return nil
}

// decodeTokens decodes tokens, the tokens of one element, into v.
func decodeTokens(tokens []xml.Token, v any) error {
	replay := xml.NewTokenDecoder(&tokenReplay{tokens: append([]xml.Token(nil), tokens...)})
	first, err := replay.Token()
	if err != nil {
		return err
	}
	start, _ := first.(xml.StartElement)
	return replay.DecodeElement(v, &start)
}

// XSDComplexContent element defines extensions or restrictions on a complex
// type that contains mixed content or elements only.
type XSDComplexContent struct {
	XMLName   xml.Name     `xml:"complexContent"`
	Extension XSDExtension `xml:"extension"`
}

// XSDSimpleContent element contains extensions or restrictions on a text-only
// complex type or on a simple type as content and contains no elements.
type XSDSimpleContent struct {
	XMLName   xml.Name     `xml:"simpleContent"`
	Extension XSDExtension `xml:"extension"`
}

// XSDExtension element extends an existing simpleType or complexType element.
type XSDExtension struct {
	XMLName        xml.Name        `xml:"extension"`
	Base           string          `xml:"base,attr"`
	Attributes     []*XSDAttribute `xml:"attribute"`
	Sequence       []*XSDElement   `xml:"sequence>element"`
	Choice         []*XSDElement   `xml:"choice>element"`
	SequenceChoice []*XSDElement   `xml:"sequence>choice>element"`
	// GroupRefs holds the xs:group references in the extension's content
	// model, at any depth of its sequences and choices, in document order.
	// The elements of the groups they name are not in the lists above.
	GroupRefs []*XSDGroup `xml:"-"`
}

// UnmarshalXML implements xml.Unmarshaler for XSDExtension. Elements of
// sequences and choices nested inside the extension's top-level sequence or
// choice are flattened into Sequence, SequenceChoice and Choice, as they are
// for the content of an XSDComplexType.
func (e *XSDExtension) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type Base XSDExtension
	aux := struct {
		*Base
		xsdNestedGroups
	}{Base: (*Base)(e)}
	tokens, err := readElementTokens(d, start)
	if err != nil {
		return err
	}
	if err := decodeTokens(tokens, &aux); err != nil {
		return err
	}
	aux.flatten(&e.Sequence, &e.SequenceChoice, &e.Choice)
	e.GroupRefs = collectGroupRefs(tokens, "sequence", "choice", "all")
	return nil
}

// XSDAttribute represent an element attribute. Simple elements cannot have
// attributes. If an element has attributes, it is considered to be of a
// complex type. But the attribute itself is always declared as a simple type.
type XSDAttribute struct {
	Doc        string         `xml:"annotation>documentation"`
	Name       string         `xml:"name,attr"`
	Ref        string         `xml:"ref,attr"`
	Type       string         `xml:"type,attr"`
	Use        string         `xml:"use,attr"`
	Fixed      string         `xml:"fixed,attr"`
	SimpleType *XSDSimpleType `xml:"simpleType"`
}

// XSDSimpleType element defines a simple type and specifies the constraints
// and information about the values of attributes or text-only elements.
type XSDSimpleType struct {
	Name        string         `xml:"name,attr"`
	Doc         string         `xml:"annotation>documentation"`
	Restriction XSDRestriction `xml:"restriction"`
	List        XSDList        `xml:"list"`
	Union       XSDUnion       `xml:"union"`
	Final       string         `xml:"final"`
}

// XSDList represents a element list
type XSDList struct {
	Doc        string         `xml:"annotation>documentation"`
	ItemType   string         `xml:"itemType,attr"`
	SimpleType *XSDSimpleType `xml:"simpleType"`
}

// XSDUnion represents a union element
type XSDUnion struct {
	SimpleType  []*XSDSimpleType `xml:"simpleType,omitempty"`
	MemberTypes string           `xml:"memberTypes,attr"`
}

// XSDRestriction defines restrictions on a simpleType, simpleContent, or complexContent definition.
type XSDRestriction struct {
	Base         string                `xml:"base,attr"`
	Enumeration  []XSDRestrictionValue `xml:"enumeration"`
	Pattern      XSDRestrictionValue   `xml:"pattern"`
	MinInclusive XSDRestrictionValue   `xml:"minInclusive"`
	MaxInclusive XSDRestrictionValue   `xml:"maxInclusive"`
	WhiteSpace   XSDRestrictionValue   `xml:"whitespace"`
	Length       XSDRestrictionValue   `xml:"length"`
	MinLength    XSDRestrictionValue   `xml:"minLength"`
	MaxLength    XSDRestrictionValue   `xml:"maxLength"`
}

// XSDRestrictionValue represents a restriction value.
type XSDRestrictionValue struct {
	Doc   string `xml:"annotation>documentation"`
	Value string `xml:"value,attr"`
}
