// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package soap

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"reflect"
)

// MarshalRPCTypePart writes an RPC message part declared with a type: an
// accessor element named after the part, in no namespace as WS-I Basic
// Profile R2735 requires, whose content is value. namespace is the target
// namespace of a qualified schema declaring value's type, whose local
// elements are in that namespace, or "" for a simple type or an unqualified
// schema. A nil value writes nothing.
func MarshalRPCTypePart(e *xml.Encoder, part, namespace string, value any) error {
	if isNilValue(value) {
		return nil
	}
	accessor := unqualifiedStart(part)
	if namespace == "" {
		return e.EncodeElement(value, accessor)
	}

	// The fields of a qualified type inherit their namespace from the
	// enclosing element, so encode them under an accessor in that namespace,
	// then write every element with the namespace it was encoded in, under
	// an accessor in none.
	var buf bytes.Buffer
	inner := xml.NewEncoder(&buf)
	if err := inner.EncodeElement(value, xml.StartElement{Name: xml.Name{Space: namespace, Local: part}}); err != nil {
		return fmt.Errorf("soap: encoding part %s: %w", part, err)
	}
	if err := inner.Flush(); err != nil {
		return fmt.Errorf("soap: encoding part %s: %w", part, err)
	}

	// The inner encoder writes an element's namespace only as a default
	// namespace declaration, so a decoded name whose namespace is not the
	// default in scope had a literal prefix in its local name, such as the
	// generator's "tns:" names, which the decoder could not resolve.
	d := xml.NewDecoder(&buf)
	var defaults []string
	var open []xml.StartElement
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("soap: re-encoding part %s: %w", part, err)
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			def := ""
			if len(defaults) > 0 {
				def = defaults[len(defaults)-1]
			}
			for _, a := range tok.Attr {
				if a.Name.Space == "" && a.Name.Local == "xmlns" {
					def = a.Value
				}
			}
			defaults = append(defaults, def)

			if len(open) == 0 {
				tok = accessor
			} else {
				tok = withoutNamespaceDeclarations(tok)
				if tok.Name.Space != def {
					tok.Name = xml.Name{Local: tok.Name.Space + ":" + tok.Name.Local}
				}
			}
			open = append(open, tok)
			err = e.EncodeToken(tok)
		case xml.EndElement:
			start := open[len(open)-1]
			open, defaults = open[:len(open)-1], defaults[:len(defaults)-1]
			err = e.EncodeToken(start.End())
		case xml.CharData:
			err = e.EncodeToken(tok.Copy())
		}
		if err != nil {
			return fmt.Errorf("soap: re-encoding part %s: %w", part, err)
		}
	}
}

// MarshalRPCElementPart writes an RPC message part declared with an element:
// an accessor element named after the part, in no namespace as WS-I Basic
// Profile R2735 requires, holding the element. A value whose type declares
// an XMLName field is written under that name; any other value, such as an
// element of a simple type, under name. A nil value writes nothing.
func MarshalRPCElementPart(e *xml.Encoder, part string, name xml.Name, value any) error {
	if isNilValue(value) {
		return nil
	}
	accessor := unqualifiedStart(part)
	if err := e.EncodeToken(accessor); err != nil {
		return err
	}
	var err error
	if hasXMLName(reflect.TypeOf(value)) {
		err = e.Encode(value)
	} else {
		err = e.EncodeElement(value, xml.StartElement{Name: name})
	}
	if err != nil {
		return fmt.Errorf("soap: encoding part %s: %w", part, err)
	}
	return e.EncodeToken(accessor.End())
}

// unqualifiedStart returns the start of an element named local in no
// namespace, undeclaring any default namespace it would inherit.
func unqualifiedStart(local string) xml.StartElement {
	return xml.StartElement{
		Name: xml.Name{Local: local},
		Attr: []xml.Attr{{Name: xml.Name{Local: "xmlns"}}},
	}
}

// withoutNamespaceDeclarations returns start without its xmlns attributes,
// which the encoder writes again from the element's namespace.
func withoutNamespaceDeclarations(start xml.StartElement) xml.StartElement {
	attrs := make([]xml.Attr, 0, len(start.Attr))
	for _, a := range start.Attr {
		if a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns") {
			continue
		}
		attrs = append(attrs, a)
	}
	start.Attr = attrs
	return start
}

func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

func hasXMLName(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	f, ok := t.FieldByName("XMLName")
	return ok && f.Type == reflect.TypeFor[xml.Name]()
}
