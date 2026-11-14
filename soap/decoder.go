// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package soap

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

// XSINamespace is the XML Schema instance namespace of xsi:type and xsi:nil.
const XSINamespace = "http://www.w3.org/2001/XMLSchema-instance"

// NewDecoder returns a decoder for a SOAP message. It decodes like
// xml.NewDecoder, except that an element with an xsi:type whose prefix the
// message declared on an ancestor element, such as xsi:type="ns2:Item" with
// ns2 declared on the SOAP envelope, also carries that declaration itself. A
// value kept apart from its ancestors, such as AnyType or AnyElement, can
// then still resolve its type. Client decodes responses with it.
//
// The message is read in full on the first read. If it does not use the
// XML Schema instance namespace, or cannot be scanned, it is decoded as is.
func NewDecoder(r io.Reader) *xml.Decoder {
	return xml.NewDecoder(&xsiPrefixReader{r: r})
}

// xsiPrefixReader reads the whole message on first use and serves it with
// the missing xsi:type prefix declarations inserted.
type xsiPrefixReader struct {
	r    io.Reader
	data *bytes.Reader
	err  error
}

func (x *xsiPrefixReader) Read(p []byte) (int, error) {
	if x.data == nil {
		data, err := io.ReadAll(x.r)
		x.data = bytes.NewReader(declareXSITypePrefixes(data))
		x.err = err
	}
	n, err := x.data.Read(p)
	if errors.Is(err, io.EOF) && x.err != nil {
		return n, x.err
	}
	return n, err
}

// declareXSITypePrefixes inserts, into the start tag of each element whose
// xsi:type uses a prefix declared only on an ancestor, the declaration of
// that prefix. It returns data unchanged when there is nothing to insert or
// when data holds anything the scan does not handle, such as a DOCTYPE or a
// malformed tag; the decoder then reports any error as usual.
//
// The scan works on the raw bytes instead of xml.Decoder tokens, because it
// runs on every response that may use xsi:type and must cost far less than
// the decode that follows.
func declareXSITypePrefixes(data []byte) []byte {
	if !bytes.Contains(data, []byte(XSINamespace)) || !bytes.Contains(data, []byte(":type=")) {
		return data
	}
	s := xsiScanner{data: data}
	if !s.scan() || len(s.inserts) == 0 {
		return data
	}
	var out bytes.Buffer
	out.Grow(len(data) + len(s.inserts)*48)
	last := 0
	for _, ins := range s.inserts {
		out.Write(data[last:ins.at])
		out.WriteString(ins.text)
		last = ins.at
	}
	out.Write(data[last:])
	return out.Bytes()
}

type prefixBinding struct {
	prefix, uri string // uri as written, still escaped
}

type xsiInsertion struct {
	at   int
	text string
}

type xsiScanner struct {
	data     []byte
	pos      int
	bindings []prefixBinding // in scope, innermost last
	marks    []int           // len(bindings) when each open element started
	open     [][]byte        // names of the open elements
	inserts  []xsiInsertion
}

var (
	commentStart = []byte("<!--")
	commentEnd   = []byte("-->")
	cdataStart   = []byte("<![CDATA[")
	cdataEnd     = []byte("]]>")
	piStart      = []byte("<?")
	piEnd        = []byte("?>")
)

// scan walks every tag. It reports false when it meets something it does
// not handle.
func (s *xsiScanner) scan() bool {
	for {
		lt := bytes.IndexByte(s.data[s.pos:], '<')
		if lt < 0 {
			return len(s.open) == 0
		}
		s.pos += lt
		rest := s.data[s.pos:]
		switch {
		case bytes.HasPrefix(rest, commentStart):
			if !s.skipPast(commentEnd) {
				return false
			}
		case bytes.HasPrefix(rest, cdataStart):
			if !s.skipPast(cdataEnd) {
				return false
			}
		case bytes.HasPrefix(rest, piStart):
			if !s.skipPast(piEnd) {
				return false
			}
		case len(rest) > 1 && rest[1] == '!':
			return false // DOCTYPE or other declaration
		case len(rest) > 1 && rest[1] == '/':
			if !s.endTag() {
				return false
			}
		default:
			if !s.startTag() {
				return false
			}
		}
	}
}

func (s *xsiScanner) skipPast(end []byte) bool {
	i := bytes.Index(s.data[s.pos:], end)
	if i < 0 {
		return false
	}
	s.pos += i + len(end)
	return true
}

func (s *xsiScanner) endTag() bool {
	s.pos += 2
	name := s.name()
	s.space()
	if len(name) == 0 || s.pos >= len(s.data) || s.data[s.pos] != '>' {
		return false
	}
	s.pos++
	if len(s.open) == 0 || !bytes.Equal(s.open[len(s.open)-1], name) {
		return false
	}
	s.open = s.open[:len(s.open)-1]
	s.bindings = s.bindings[:s.marks[len(s.marks)-1]]
	s.marks = s.marks[:len(s.marks)-1]
	return true
}

type rawAttr struct {
	name, value []byte
}

func (s *xsiScanner) startTag() bool {
	s.pos++
	name := s.name()
	if len(name) == 0 {
		return false
	}
	outer := len(s.bindings)
	var types []rawAttr
	for {
		s.space()
		if s.pos >= len(s.data) {
			return false
		}
		switch s.data[s.pos] {
		case '>':
			s.finishStart(name, outer, types, s.pos, false)
			s.pos++
			return true
		case '/':
			if s.pos+1 >= len(s.data) || s.data[s.pos+1] != '>' {
				return false
			}
			s.finishStart(name, outer, types, s.pos, true)
			s.pos += 2
			return true
		}
		attr, ok := s.attr()
		if !ok {
			return false
		}
		switch {
		case bytes.HasPrefix(attr.name, []byte("xmlns:")):
			s.bindings = append(s.bindings, prefixBinding{string(attr.name[len("xmlns:"):]), string(attr.value)})
		case bytes.HasSuffix(attr.name, []byte(":type")):
			types = append(types, attr)
		}
	}
}

// finishStart records the declarations the element's xsi:type attributes
// need, to be inserted at the end of its start tag.
func (s *xsiScanner) finishStart(name []byte, outer int, types []rawAttr, end int, selfClosing bool) {
	var decls strings.Builder
	for _, attr := range types {
		attrPrefix := string(attr.name[:len(attr.name)-len(":type")])
		if uri, _ := s.lookup(attrPrefix, len(s.bindings)); uri != XSINamespace {
			continue
		}
		prefix, _, ok := strings.Cut(strings.TrimSpace(string(attr.value)), ":")
		if !ok || prefix == "" || prefix == "xml" || declares(s.bindings[outer:], prefix) {
			continue
		}
		uri, ok := s.lookup(prefix, outer)
		if !ok || strings.ContainsRune(uri, '"') {
			continue
		}
		s.bindings = append(s.bindings, prefixBinding{prefix, uri})
		decls.WriteString(" xmlns:" + prefix + `="` + uri + `"`)
	}
	if decls.Len() > 0 {
		s.inserts = append(s.inserts, xsiInsertion{at: end, text: decls.String()})
	}
	if selfClosing {
		s.bindings = s.bindings[:outer]
		return
	}
	s.open = append(s.open, name)
	s.marks = append(s.marks, outer)
}

func (s *xsiScanner) lookup(prefix string, from int) (string, bool) {
	for i := from - 1; i >= 0; i-- {
		if s.bindings[i].prefix == prefix {
			return s.bindings[i].uri, true
		}
	}
	return "", false
}

// attr reads name="value" or name='value'.
func (s *xsiScanner) attr() (rawAttr, bool) {
	name := s.name()
	s.space()
	if len(name) == 0 || s.pos >= len(s.data) || s.data[s.pos] != '=' {
		return rawAttr{}, false
	}
	s.pos++
	s.space()
	if s.pos >= len(s.data) || (s.data[s.pos] != '"' && s.data[s.pos] != '\'') {
		return rawAttr{}, false
	}
	quote := s.data[s.pos]
	s.pos++
	end := bytes.IndexByte(s.data[s.pos:], quote)
	if end < 0 {
		return rawAttr{}, false
	}
	value := s.data[s.pos : s.pos+end]
	s.pos += end + 1
	return rawAttr{name: name, value: value}, true
}

func (s *xsiScanner) name() []byte {
	start := s.pos
	for s.pos < len(s.data) {
		switch s.data[s.pos] {
		case ' ', '\t', '\r', '\n', '/', '>', '=':
			return s.data[start:s.pos]
		}
		s.pos++
	}
	return s.data[start:s.pos]
}

func (s *xsiScanner) space() {
	for s.pos < len(s.data) {
		switch s.data[s.pos] {
		case ' ', '\t', '\r', '\n':
			s.pos++
		default:
			return
		}
	}
}

// declares reports whether bindings declares prefix.
func declares(bindings []prefixBinding, prefix string) bool {
	for _, b := range bindings {
		if b.prefix == prefix {
			return true
		}
	}
	return false
}

// resolveQName resolves a prefixed name against the namespace declarations
// in attrs. An unprefixed name takes the default namespace declared there,
// if any.
func resolveQName(value string, attrs []xml.Attr) (xml.Name, bool) {
	value = strings.TrimSpace(value)
	prefix, local, ok := strings.Cut(value, ":")
	if !ok {
		for _, attr := range attrs {
			if attr.Name.Space == "" && attr.Name.Local == "xmlns" {
				return xml.Name{Space: attr.Value, Local: value}, true
			}
		}
		return xml.Name{Local: value}, true
	}
	for _, attr := range attrs {
		if attr.Name.Space == "xmlns" && attr.Name.Local == prefix {
			return xml.Name{Space: attr.Value, Local: local}, true
		}
	}
	return xml.Name{}, false
}
