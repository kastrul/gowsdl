// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package soap

import (
	"encoding/xml"
	"errors"
	"slices"
	"strconv"
	"strings"
)

// AnyType holds the content of an element declared as xsd:anyType.
//
// To send content, set Value to any value encoding/xml can marshal; it is
// encoded as the element's content. Attrs are added to the element, for
// example an xsi:type attribute.
//
// A received element is kept in XMLName, Attrs and InnerXML. Use Decode to
// unmarshal it into a concrete type, or inspect Attrs first to find its
// xsi:type.
//
// A zero AnyType is omitted when encoded, like a nil interface value. The
// element is named by the field it is in; see AnyElement for the items of
// an xs:any wildcard.
type AnyType struct {
	// Value is encoded as the element's content. Decoding does not set it.
	Value any `json:"value,omitempty"`
	// XMLName is the name of the received element.
	XMLName xml.Name `json:"-"`
	// Attrs are the attributes of the received element, including its
	// namespace declarations, or the attributes to add when encoding. When
	// the response was decoded by Client or NewDecoder, they also declare
	// the prefix of an xsi:type value, such as ns2 in xsi:type="ns2:Item";
	// see Type.
	Attrs []xml.Attr `json:"attrs,omitempty"`
	// InnerXML is the content of the element as XML. For a received element
	// it is self-contained: every namespace it uses is declared within it,
	// including namespaces the server declared on ancestor elements such as
	// the SOAP envelope. When Value is nil, InnerXML is encoded verbatim as
	// the element's content.
	InnerXML string `json:"innerXML,omitempty"`
}

type rawInnerXML struct {
	Inner string `xml:",innerxml"`
}

// IsZero reports whether a holds no value, attributes or content.
func (a AnyType) IsZero() bool {
	return a.Value == nil && a.XMLName == (xml.Name{}) && len(a.Attrs) == 0 && a.InnerXML == ""
}

// MarshalXML encodes Value, or else InnerXML, as the content of start. It
// writes nothing for a zero AnyType.
func (a AnyType) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if a.IsZero() {
		return nil
	}
	return a.encode(e, start)
}

func (a AnyType) encode(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = append(start.Attr, encodableAttrs(a.Attrs)...)
	if a.Value != nil {
		return e.EncodeElement(a.Value, start)
	}
	return e.EncodeElement(rawInnerXML{Inner: a.InnerXML}, start)
}

// Type returns the element's xsi:type with its prefix resolved, for example
// {Space: "urn:example", Local: "Item"} for xsi:type="ns2:Item". ok is false
// when there is no xsi:type or its prefix is not declared in Attrs. Responses
// decoded by Client, or by NewDecoder, carry the declaration in Attrs even
// when the server made it on an ancestor such as the SOAP envelope.
func (a AnyType) Type() (name xml.Name, ok bool) {
	for _, attr := range a.Attrs {
		if attr.Name.Space == XSINamespace && attr.Name.Local == "type" {
			return resolveQName(attr.Value, a.Attrs)
		}
	}
	return xml.Name{}, false
}

// UnmarshalXML records the element's name and attributes, and its content as
// self-contained XML.
func (a *AnyType) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var w nsWriter
	for depth := 0; ; {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			w.start(t)
		case xml.EndElement:
			if depth == 0 {
				*a = AnyType{
					XMLName:  start.Name,
					Attrs:    slices.Clone(start.Attr),
					InnerXML: w.String(),
				}
				return nil
			}
			depth--
			w.end()
		case xml.CharData:
			w.text(t)
		case xml.Comment:
			w.comment(t)
		case xml.ProcInst:
			w.procInst(t)
		}
	}
}

// Decode unmarshals the element into v, as xml.Unmarshal would if the
// element had been decoded into v directly. If v's type declares an XMLName
// with a different element name, Decode returns an error.
func (a AnyType) Decode(v any) error {
	name := a.XMLName
	if name.Local == "" {
		name.Local = "anyType"
	}
	var w nsWriter
	w.start(xml.StartElement{Name: name, Attr: a.Attrs})
	w.raw(a.InnerXML)
	w.end()
	return xml.Unmarshal([]byte(w.String()), v)
}

// AnyElement is an element matched by an xs:any wildcard, generated as
// Items []AnyElement `xml:",any"`. Each unknown element is received into one
// item, with the same fields and Decode method as AnyType.
//
// Unlike AnyType, an item names itself when encoded: by XMLName when set,
// otherwise by Value, through Value's own XMLName field or type name.
type AnyElement AnyType

// IsZero reports whether a holds no value, attributes or content.
func (a AnyElement) IsZero() bool {
	return AnyType(a).IsZero()
}

// Decode unmarshals the element into v. See AnyType.Decode.
func (a AnyElement) Decode(v any) error {
	return AnyType(a).Decode(v)
}

// Type returns the element's resolved xsi:type. See AnyType.Type.
func (a AnyElement) Type() (xml.Name, bool) {
	return AnyType(a).Type()
}

// UnmarshalXML records the element. See AnyType.UnmarshalXML.
func (a *AnyElement) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	return (*AnyType)(a).UnmarshalXML(d, start)
}

// MarshalXML encodes the element under its own name, ignoring start, whose
// name is only the Go field name of the wildcard. It writes nothing for a
// zero AnyElement.
func (a AnyElement) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	t := AnyType(a)
	switch {
	case t.IsZero():
		return nil
	case t.XMLName.Local != "":
		return t.encode(e, xml.StartElement{Name: t.XMLName})
	case t.Value == nil:
		return errAnyElementName
	}
	start, err := valueStart(t.Value)
	if err != nil {
		return err
	}
	return t.encode(e, start)
}

var errAnyElementName = errors.New("soap: AnyElement needs XMLName or Value to name the element")

// valueStart returns the start element v is encoded with on its own.
func valueStart(v any) (xml.StartElement, error) {
	var buf strings.Builder
	if err := xml.NewEncoder(&buf).Encode(v); err != nil {
		return xml.StartElement{}, err
	}
	d := xml.NewDecoder(strings.NewReader(buf.String()))
	for {
		tok, err := d.Token()
		if err != nil {
			return xml.StartElement{}, err
		}
		if start, ok := tok.(xml.StartElement); ok {
			return xml.StartElement{Name: start.Name}, nil
		}
	}
}

// encodableAttrs prepares attributes for xml.Encoder. Prefixed namespace
// declarations are passed through as plain attributes, because the encoder
// would otherwise rename them, and default namespace declarations are
// dropped, because the encoder declares the element's namespace itself.
func encodableAttrs(attrs []xml.Attr) []xml.Attr {
	out := make([]xml.Attr, 0, len(attrs))
	for _, attr := range attrs {
		switch {
		case attr.Name.Space == "xmlns":
			out = append(out, xml.Attr{Name: xml.Name{Local: "xmlns:" + attr.Name.Local}, Value: attr.Value})
		case attr.Name.Space == "" && attr.Name.Local == "xmlns":
		default:
			out = append(out, attr)
		}
	}
	return out
}

const xmlNamespaceURI = "http://www.w3.org/XML/1998/namespace"

type nsBinding struct {
	prefix, uri string
}

// nsWriter writes decoded XML tokens, whose names carry namespace URIs, back
// out as XML that declares every namespace it uses. It never declares a
// default namespace, and resets it on top-level elements, so the output
// means the same wherever it is embedded.
type nsWriter struct {
	b      strings.Builder
	scopes [][]nsBinding
	names  []string
	next   int
}

func (w *nsWriter) String() string {
	return w.b.String()
}

func (w *nsWriter) resolve(prefix string) (string, bool) {
	for i := len(w.scopes) - 1; i >= 0; i-- {
		for j := len(w.scopes[i]) - 1; j >= 0; j-- {
			if w.scopes[i][j].prefix == prefix {
				return w.scopes[i][j].uri, true
			}
		}
	}
	return "", false
}

// prefixFor returns a prefix bound to uri, declaring a new one on the current
// element if no binding in scope can be used.
func (w *nsWriter) prefixFor(uri string) string {
	if uri == xmlNamespaceURI {
		return "xml"
	}
	for i := len(w.scopes) - 1; i >= 0; i-- {
		for _, b := range w.scopes[i] {
			if b.uri == uri {
				if bound, _ := w.resolve(b.prefix); bound == uri {
					return b.prefix
				}
			}
		}
	}
	for {
		w.next++
		prefix := "ns" + strconv.Itoa(w.next)
		if _, taken := w.resolve(prefix); !taken {
			last := len(w.scopes) - 1
			w.scopes[last] = append(w.scopes[last], nsBinding{prefix: prefix, uri: uri})
			return prefix
		}
	}
}

func (w *nsWriter) qualify(name xml.Name) string {
	if name.Space == "" {
		return name.Local
	}
	return w.prefixFor(name.Space) + ":" + name.Local
}

func (w *nsWriter) start(t xml.StartElement) {
	var scope []nsBinding
	var attrs []xml.Attr
	for _, attr := range t.Attr {
		switch {
		case attr.Name.Space == "xmlns":
			scope = append(scope, nsBinding{prefix: attr.Name.Local, uri: attr.Value})
		case attr.Name.Space == "" && attr.Name.Local == "xmlns":
		default:
			attrs = append(attrs, attr)
		}
	}
	top := len(w.scopes) == 0
	w.scopes = append(w.scopes, scope)

	name := w.qualify(t.Name)
	attrNames := make([]string, len(attrs))
	for i, attr := range attrs {
		attrNames[i] = w.qualify(attr.Name)
	}

	w.b.WriteString("<" + name)
	if top {
		w.b.WriteString(` xmlns=""`)
	}
	for _, b := range w.scopes[len(w.scopes)-1] {
		w.writeAttr("xmlns:"+b.prefix, b.uri)
	}
	for i, attr := range attrs {
		w.writeAttr(attrNames[i], attr.Value)
	}
	w.b.WriteString(">")
	w.names = append(w.names, name)
}

func (w *nsWriter) writeAttr(name, value string) {
	w.b.WriteString(" " + name + `="`)
	_ = xml.EscapeText(&w.b, []byte(value))
	w.b.WriteString(`"`)
}

func (w *nsWriter) end() {
	last := len(w.names) - 1
	w.b.WriteString("</" + w.names[last] + ">")
	w.names = w.names[:last]
	w.scopes = w.scopes[:len(w.scopes)-1]
}

func (w *nsWriter) text(data xml.CharData) {
	_ = xml.EscapeText(&w.b, data)
}

func (w *nsWriter) comment(data xml.Comment) {
	w.b.WriteString("<!--")
	w.b.Write(data)
	w.b.WriteString("-->")
}

func (w *nsWriter) procInst(p xml.ProcInst) {
	w.b.WriteString("<?" + p.Target)
	if len(p.Inst) > 0 {
		w.b.WriteString(" ")
		w.b.Write(p.Inst)
	}
	w.b.WriteString("?>")
}

func (w *nsWriter) raw(s string) {
	w.b.WriteString(s)
}
