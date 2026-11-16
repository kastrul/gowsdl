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

// A repeated model group, such as an xs:sequence with maxOccurs="unbounded",
// holds several elements per occurrence and has no element of its own:
// its occurrences follow one another among the other children of the
// enclosing element. Generated code holds such a group as a slice of
// structs, one per occurrence, whose fields and struct tags describe the
// group's elements as they would any other content. encoding/xml cannot
// write or read a slice of structs without an element around each, so the
// helpers below let encoding/xml do all the work per element and only take
// away, or put back, the boundary between occurrences.

// MarshalModelGroup writes v, a struct holding one occurrence of a repeated
// model group, as its elements alone: everything encoding/xml writes for v
// inside its start and end tags, with neither tag. The occurrence type's
// MarshalXML calls it with a value of a type that has the occurrence's
// fields but no methods, so that encoding/xml applies the struct tags and
// does not call MarshalXML again.
func MarshalModelGroup(e *xml.Encoder, v any) error {
	var buf bytes.Buffer
	inner := xml.NewEncoder(&buf)
	if err := inner.EncodeElement(v, xml.StartElement{Name: xml.Name{Local: "group"}}); err != nil {
		return fmt.Errorf("soap: encoding model group: %w", err)
	}
	if err := inner.Flush(); err != nil {
		return fmt.Errorf("soap: encoding model group: %w", err)
	}

	// Read the encoding back without resolving namespaces, so that every
	// name and namespace declaration is written again exactly as
	// encoding/xml wrote it, prefixes included.
	d := xml.NewDecoder(&buf)
	depth := 0
	for {
		tok, err := d.RawToken()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("soap: re-encoding model group: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if depth == 1 {
				continue
			}
			err = e.EncodeToken(rawStart(t))
		case xml.EndElement:
			depth--
			if depth == 0 {
				continue
			}
			err = e.EncodeToken(xml.EndElement{Name: rawName(t.Name)})
		case xml.CharData, xml.Comment:
			if depth > 0 {
				err = e.EncodeToken(xml.CopyToken(t))
			}
		}
		if err != nil {
			return fmt.Errorf("soap: re-encoding model group: %w", err)
		}
	}
}

// rawStart returns start, as read by RawToken, with every prefixed name
// turned into a local name holding the prefix, so that the encoder writes it
// unchanged.
func rawStart(start xml.StartElement) xml.StartElement {
	out := xml.StartElement{Name: rawName(start.Name), Attr: make([]xml.Attr, len(start.Attr))}
	for i, a := range start.Attr {
		out.Attr[i] = xml.Attr{Name: rawName(a.Name), Value: a.Value}
	}
	return out
}

func rawName(n xml.Name) xml.Name {
	if n.Space == "" {
		return n
	}
	return xml.Name{Local: n.Space + ":" + n.Local}
}

// ModelGroupMember is a field of the struct UnmarshalModelGroups decodes
// into. Its xml tag names one element of a repeated model group, and
// encoding/xml hands it every child element of that name, in document order,
// for it to decode into the right occurrence. Make one with GroupMember.
type ModelGroupMember struct {
	occurrence reflect.Type
	path       []int
	group      *modelGroupDecoder
}

// GroupMember returns the ModelGroupMember for the element held by the field
// of T, the occurrence type of a repeated model group, with the given index.
// When that field is itself a slice of occurrences of a nested repeated
// group, further indexes select the field within those, and so on.
func GroupMember[T any](path ...int) ModelGroupMember {
	return ModelGroupMember{occurrence: reflect.TypeFor[T](), path: path}
}

// UnmarshalXML implements xml.Unmarshaler. It decodes start into the field
// of the occurrence the element belongs to: the last occurrence decoded so
// far, or a new one when the element cannot follow the elements already in
// it, being a field that comes earlier in the occurrence type, or the same
// field when that field holds one element. A member must belong to a struct
// passed to UnmarshalModelGroups.
func (m *ModelGroupMember) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if m.group == nil {
		return fmt.Errorf("soap: element %s of a model group decoded outside UnmarshalModelGroups", start.Name.Local)
	}
	return m.group.decode(m.path, d, start)
}

// UnmarshalModelGroups decodes start into v, a pointer to a struct whose
// first field embeds a pointer to the struct to decode into, and whose other
// fields of type ModelGroupMember name the elements of the repeated model
// groups that struct holds, as slices of occurrence types, directly or in
// structs it embeds. Generated UnmarshalXML methods call it with a struct
// literal that also has a field named UnmarshalXML, which keeps the
// methods of the embedded struct from being used to decode v.
//
// encoding/xml decodes everything as it would decode the embedded struct,
// except that it hands the elements of model groups to the members, which
// keep them in their occurrences. As encoding/xml would, it sets the
// XMLName field the struct declares itself, if any, to the name of start.
func UnmarshalModelGroups(d *xml.Decoder, start xml.StartElement, v any) error {
	sv := reflect.ValueOf(v)
	if sv.Kind() != reflect.Pointer || sv.IsNil() || sv.Elem().Kind() != reflect.Struct || sv.Elem().NumField() == 0 {
		return fmt.Errorf("soap: UnmarshalModelGroups needs a pointer to a struct, not %T", v)
	}
	sv = sv.Elem()
	first := sv.Type().Field(0)
	target := sv.Field(0)
	if !first.Anonymous || !first.IsExported() || target.Kind() != reflect.Pointer || target.Type().Elem().Kind() != reflect.Struct || target.IsNil() {
		return fmt.Errorf("soap: the first field of %s must embed a non-nil pointer to an exported struct", sv.Type())
	}
	target = target.Elem()

	groups := make(map[reflect.Type]*modelGroupDecoder)
	memberType := reflect.TypeFor[ModelGroupMember]()
	for i := 1; i < sv.NumField(); i++ {
		if sv.Type().Field(i).Type != memberType {
			continue
		}
		m := sv.Field(i).Addr().Interface().(*ModelGroupMember)
		if m.occurrence == nil {
			return fmt.Errorf("soap: model group member %s of %s was not made by GroupMember", sv.Type().Field(i).Name, sv.Type())
		}
		g, ok := groups[m.occurrence]
		if !ok {
			slice, found := findSliceOf(target, m.occurrence)
			if !found {
				return fmt.Errorf("soap: %s has no field of type []%s", target.Type(), m.occurrence)
			}
			g = &modelGroupDecoder{slice: slice}
			groups[m.occurrence] = g
		}
		if err := checkMemberPath(m.occurrence, m.path); err != nil {
			return fmt.Errorf("soap: model group member %s of %s: %w", sv.Type().Field(i).Name, sv.Type(), err)
		}
		m.group = g
	}

	if err := d.DecodeElement(v, &start); err != nil {
		return err
	}
	if f, ok := target.Type().FieldByName("XMLName"); ok && len(f.Index) == 1 && f.Type == reflect.TypeFor[xml.Name]() {
		target.Field(f.Index[0]).Set(reflect.ValueOf(start.Name))
	}
	return nil
}

// findSliceOf returns the field of the struct v of type []elem, looking
// into embedded structs too, allocating embedded pointers on the way.
func findSliceOf(v reflect.Value, elem reflect.Type) (reflect.Value, bool) {
	want := reflect.SliceOf(elem)
	t := v.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Type == want && f.IsExported() {
			return v.Field(i), true
		}
	}
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.Anonymous {
			continue
		}
		fv := v.Field(i)
		if fv.Kind() == reflect.Pointer && fv.Type().Elem().Kind() == reflect.Struct {
			if fv.IsNil() {
				if !fv.CanSet() {
					continue
				}
				fv.Set(reflect.New(fv.Type().Elem()))
			}
			fv = fv.Elem()
		}
		if fv.Kind() != reflect.Struct {
			continue
		}
		if found, ok := findSliceOf(fv, elem); ok {
			return found, true
		}
	}
	return reflect.Value{}, false
}

// checkMemberPath reports whether path selects an exported field of the
// struct t, every index but the last selecting a slice of structs to
// continue in.
func checkMemberPath(t reflect.Type, path []int) error {
	if len(path) == 0 {
		return errors.New("empty field path")
	}
	for n, i := range path {
		if t.Kind() != reflect.Struct || i < 0 || i >= t.NumField() || !t.Field(i).IsExported() {
			return fmt.Errorf("no exported field %d in %s", i, t)
		}
		ft := t.Field(i).Type
		if n < len(path)-1 {
			if ft.Kind() != reflect.Slice {
				return fmt.Errorf("field %s of %s is not a slice of a nested model group", t.Field(i).Name, t)
			}
			t = ft.Elem()
		}
	}
	return nil
}

// modelGroupDecoder holds the slice of occurrences of one repeated model
// group being decoded and what its last occurrence holds so far.
type modelGroupDecoder struct {
	slice reflect.Value
	state occurrenceState
}

// occurrenceState records the last field decoded into the current
// occurrence of a group, and the state of the groups nested in it.
type occurrenceState struct {
	started bool
	last    int
	nested  map[int]*occurrenceState
}

func (g *modelGroupDecoder) decode(path []int, d *xml.Decoder, start xml.StartElement) error {
	return g.state.decode(g.slice, path, d, start)
}

// decode decodes start into the field path[0] of the last occurrence in
// slice, first appending a new occurrence when the field cannot follow what
// that occurrence already holds.
func (s *occurrenceState) decode(slice reflect.Value, path []int, d *xml.Decoder, start xml.StartElement) error {
	i := path[0]
	occurrence := slice.Type().Elem()
	if !s.started || slice.Len() == 0 || i < s.last || (i == s.last && !repeatable(occurrence.Field(i).Type)) {
		slice.Set(reflect.Append(slice, reflect.Zero(occurrence)))
		s.started = true
		s.nested = nil
	}
	s.last = i
	field := slice.Index(slice.Len() - 1).Field(i)
	if len(path) == 1 {
		return d.DecodeElement(field.Addr().Interface(), &start)
	}
	if s.nested == nil {
		s.nested = make(map[int]*occurrenceState)
	}
	inner := s.nested[i]
	if inner == nil {
		inner = new(occurrenceState)
		s.nested[i] = inner
	}
	return inner.decode(field, path[1:], d, start)
}

// repeatable reports whether a field of type t holds several elements: a
// slice other than []byte, which encoding/xml reads as text.
func repeatable(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && t.Elem().Kind() != reflect.Uint8
}
