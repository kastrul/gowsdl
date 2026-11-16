// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package soap

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// wireRow is one value of an XSD date or time type and the bytes it is
// marshalled to as an element (<v>) and as an attribute (a="...").
type wireRow struct {
	name string
	elem string
	attr string
}

func marshalElemAttr[T xml.Marshaler](t *testing.T, v T) (string, string) {
	t.Helper()
	var b strings.Builder
	enc := xml.NewEncoder(&b)
	if err := v.MarshalXML(enc, xml.StartElement{Name: xml.Name{Local: "v"}}); err != nil {
		t.Fatalf("MarshalXML: %v", err)
	}
	if err := enc.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	ma, ok := any(v).(xml.MarshalerAttr)
	if !ok {
		t.Fatalf("%T does not implement xml.MarshalerAttr", v)
	}
	attr, err := ma.MarshalXMLAttr(xml.Name{Local: "a"})
	if err != nil {
		t.Fatalf("MarshalXMLAttr: %v", err)
	}
	if attr.Name.Local == "" {
		return b.String(), "<none>"
	}
	return b.String(), fmt.Sprintf("%s=%q", attr.Name.Local, attr.Value)
}

func decodeElem[T any](t *testing.T, text string) T {
	t.Helper()
	var v struct {
		V T `xml:"v"`
	}
	if err := xml.Unmarshal([]byte("<p><v>"+text+"</v></p>"), &v); err != nil {
		t.Fatalf("decoding %q: %v", text, err)
	}
	return v.V
}

var (
	plus2  = time.FixedZone("+02", 2*60*60)
	minus8 = time.FixedZone("-08", -8*60*60)
	year1  = time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)
)

func stripped(v XSDDateTime) XSDDateTime { v.StripTz(); return v }
func strippedDate(v XSDDate) XSDDate     { v.StripTz(); return v }

func xsdDateTimeWireValues(t *testing.T) []struct {
	name string
	v    XSDDateTime
} {
	return []struct {
		name string
		v    XSDDateTime
	}{
		{"zero value", XSDDateTime{}},
		{"Create(time.Time{}, true)", CreateXsdDateTime(time.Time{}, true)},
		{"Create(time.Time{}, false)", CreateXsdDateTime(time.Time{}, false)},
		{"Create(time.Time{}.In(+02), true)", CreateXsdDateTime(time.Time{}.In(plus2), true)},
		{"Create(0001-01-01T00:00Z, true)", CreateXsdDateTime(year1, true)},
		{"Create(0001-01-01T02:00+02, true)", CreateXsdDateTime(time.Date(1, 1, 1, 2, 0, 0, 0, plus2), true)},
		{"Create(0001-01-01T02:00+02, false)", CreateXsdDateTime(time.Date(1, 1, 1, 2, 0, 0, 0, plus2), false)},
		{"Create(0001-01-01T00:00+02, true)", CreateXsdDateTime(time.Date(1, 1, 1, 0, 0, 0, 0, plus2), true)},
		{"Create(0001-01-01T00:00:00.000000001Z, true)", CreateXsdDateTime(year1.Add(1), true)},
		{"Create(2024-02-29T00:00Z, true)", CreateXsdDateTime(time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), true)},
		{"Create(2024-02-29T00:00Z, false)", CreateXsdDateTime(time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), false)},
		{"Create(1951-10-22T01:02:03.4-08, true)", CreateXsdDateTime(time.Date(1951, 10, 22, 1, 2, 3, 400000000, minus8), true)},
		{"Create(1951-10-22T01:02:03-08, true).In(UTC)", CreateXsdDateTime(time.Date(1951, 10, 22, 1, 2, 3, 0, minus8).In(time.UTC), true)},
		{"Create(9999-12-31T23:59:59Z, true)", CreateXsdDateTime(time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC), true)},
		{"StripTz(Create(0001-01-01T02:00+02, true))", stripped(CreateXsdDateTime(time.Date(1, 1, 1, 2, 0, 0, 0, plus2), true))},
		{"StripTz(Create(2024-02-29T10:00+02, true))", stripped(CreateXsdDateTime(time.Date(2024, 2, 29, 10, 0, 0, 0, plus2), true))},
		{`decoded ""`, decodeElem[XSDDateTime](t, "")},
		{`decoded "0001-01-01T00:00:00"`, decodeElem[XSDDateTime](t, "0001-01-01T00:00:00")},
		{`decoded "0001-01-01T00:00:00Z"`, decodeElem[XSDDateTime](t, "0001-01-01T00:00:00Z")},
		{`decoded "0001-01-01T00:00:00+00:00"`, decodeElem[XSDDateTime](t, "0001-01-01T00:00:00+00:00")},
		{`decoded "0001-01-01T00:00:00-00:00"`, decodeElem[XSDDateTime](t, "0001-01-01T00:00:00-00:00")},
		{`decoded "0001-01-01T00:00:00.000Z"`, decodeElem[XSDDateTime](t, "0001-01-01T00:00:00.000Z")},
		{`decoded "0001-01-01T02:00:00+02:00"`, decodeElem[XSDDateTime](t, "0001-01-01T02:00:00+02:00")},
		{`decoded "0001-01-01T00:00:00+02:00"`, decodeElem[XSDDateTime](t, "0001-01-01T00:00:00+02:00")},
		{`StripTz(decoded "0001-01-01T00:00:00Z")`, stripped(decodeElem[XSDDateTime](t, "0001-01-01T00:00:00Z"))},
		{`decoded "2024-02-29T00:00:00"`, decodeElem[XSDDateTime](t, "2024-02-29T00:00:00")},
		{`decoded "2024-02-29T00:00:00Z"`, decodeElem[XSDDateTime](t, "2024-02-29T00:00:00Z")},
		{`decoded "2024-02-29T00:00:00.5-08:00"`, decodeElem[XSDDateTime](t, "2024-02-29T00:00:00.5-08:00")},
	}
}

func xsdDateWireValues(t *testing.T) []struct {
	name string
	v    XSDDate
} {
	return []struct {
		name string
		v    XSDDate
	}{
		{"zero value", XSDDate{}},
		{"Create(time.Time{}, true)", CreateXsdDate(time.Time{}, true)},
		{"Create(time.Time{}, false)", CreateXsdDate(time.Time{}, false)},
		{"Create(time.Time{}.In(+02), true)", CreateXsdDate(time.Time{}.In(plus2), true)},
		{"Create(0001-01-01Z, true)", CreateXsdDate(year1, true)},
		{"Create(0001-01-01+02, true)", CreateXsdDate(time.Date(1, 1, 1, 0, 0, 0, 0, plus2), true)},
		{"Create(0001-01-01-08, false)", CreateXsdDate(time.Date(1, 1, 1, 0, 0, 0, 0, minus8), false)},
		{"Create(2024-02-29Z, true)", CreateXsdDate(time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), true)},
		{"Create(2024-02-29Z, false)", CreateXsdDate(time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), false)},
		{"Create(1951-10-22-08, true)", CreateXsdDate(time.Date(1951, 10, 22, 0, 0, 0, 0, minus8), true)},
		{"Create(9999-12-31Z, true)", CreateXsdDate(time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC), true)},
		{"StripTz(Create(1951-10-22-08, true))", strippedDate(CreateXsdDate(time.Date(1951, 10, 22, 0, 0, 0, 0, minus8), true))},
		{`decoded ""`, decodeElem[XSDDate](t, "")},
		{`decoded "0001-01-01"`, decodeElem[XSDDate](t, "0001-01-01")},
		{`decoded "0001-01-01Z"`, decodeElem[XSDDate](t, "0001-01-01Z")},
		{`decoded "0001-01-01+00:00"`, decodeElem[XSDDate](t, "0001-01-01+00:00")},
		{`decoded "0001-01-01-00:00"`, decodeElem[XSDDate](t, "0001-01-01-00:00")},
		{`decoded "0001-01-01+02:00"`, decodeElem[XSDDate](t, "0001-01-01+02:00")},
		{`decoded "0001-01-01-02:00"`, decodeElem[XSDDate](t, "0001-01-01-02:00")},
		{`StripTz(decoded "0001-01-01Z")`, strippedDate(decodeElem[XSDDate](t, "0001-01-01Z"))},
		{`decoded "2024-02-29"`, decodeElem[XSDDate](t, "2024-02-29")},
		{`decoded "2024-02-29-08:00"`, decodeElem[XSDDate](t, "2024-02-29-08:00")},
	}
}

func xsdTimeWireValues(t *testing.T) []struct {
	name string
	v    XSDTime
} {
	return []struct {
		name string
		v    XSDTime
	}{
		{"zero value", XSDTime{}},
		{"Create(0,0,0,0,nil)", CreateXsdTime(0, 0, 0, 0, nil)},
		{"Create(0,0,0,0,UTC)", CreateXsdTime(0, 0, 0, 0, time.UTC)},
		{"Create(0,0,0,0,+02)", CreateXsdTime(0, 0, 0, 0, plus2)},
		{"Create(0,0,0,0,-08)", CreateXsdTime(0, 0, 0, 0, minus8)},
		{"Create(23,59,59,999999999,nil)", CreateXsdTime(23, 59, 59, 999999999, nil)},
		{"Create(12,13,14,4,-08)", CreateXsdTime(12, 13, 14, 4, minus8)},
		{`decoded ""`, decodeElem[XSDTime](t, "")},
		{`decoded "00:00:00"`, decodeElem[XSDTime](t, "00:00:00")},
		{`decoded "00:00:00Z"`, decodeElem[XSDTime](t, "00:00:00Z")},
		{`decoded "00:00:00+00:00"`, decodeElem[XSDTime](t, "00:00:00+00:00")},
		{`decoded "00:00:00-00:00"`, decodeElem[XSDTime](t, "00:00:00-00:00")},
		{`decoded "00:00:00+02:00"`, decodeElem[XSDTime](t, "00:00:00+02:00")},
		{`decoded "00:00:00-08:00"`, decodeElem[XSDTime](t, "00:00:00-08:00")},
		{`decoded "00:00:00.000"`, decodeElem[XSDTime](t, "00:00:00.000")},
		{`decoded "12:13:14.5"`, decodeElem[XSDTime](t, "12:13:14.5")},
	}
}

// TestXSDDateTimeTypes_DumpWire prints every row of the wire tables. Set
// GOWSDL_DUMP_XSD_WIRE=1 to see it.
func TestXSDDateTimeTypes_DumpWire(t *testing.T) {
	if os.Getenv("GOWSDL_DUMP_XSD_WIRE") == "" {
		t.Skip("set GOWSDL_DUMP_XSD_WIRE=1 to print the wire tables")
	}
	for _, r := range xsdDateTimeWireValues(t) {
		e, a := marshalElemAttr(t, r.v)
		fmt.Printf("XSDDateTime | %-46s | %-40s | %s\n", r.name, e, a)
	}
	for _, r := range xsdDateWireValues(t) {
		e, a := marshalElemAttr(t, r.v)
		fmt.Printf("XSDDate     | %-46s | %-40s | %s\n", r.name, e, a)
	}
	for _, r := range xsdTimeWireValues(t) {
		e, a := marshalElemAttr(t, r.v)
		fmt.Printf("XSDTime     | %-46s | %-40s | %s\n", r.name, e, a)
	}
}

// The want tables pin the bytes each value is marshalled to, as an element
// and as an attribute. The only rows that changed when decoded values started
// to round-trip are the decoded ones whose instant is 0001-01-01T00:00:00Z
// (.NET's DateTime.MinValue among them): they used to be written as nothing.
// Every value built with a constructor or left unset is written exactly as
// before.

var wantXSDDateTimeWire = []wireRow{
	{`zero value`, ``, `<none>`},
	{`Create(time.Time{}, true)`, ``, `<none>`},
	{`Create(time.Time{}, false)`, ``, `<none>`},
	{`Create(time.Time{}.In(+02), true)`, ``, `<none>`},
	{`Create(0001-01-01T00:00Z, true)`, ``, `<none>`},
	{`Create(0001-01-01T02:00+02, true)`, ``, `<none>`},
	{`Create(0001-01-01T02:00+02, false)`, ``, `<none>`},
	{`Create(0001-01-01T00:00+02, true)`, `<v>0001-01-01T00:00:00+02:00</v>`, `a="0001-01-01T00:00:00+02:00"`},
	{`Create(0001-01-01T00:00:00.000000001Z, true)`, `<v>0001-01-01T00:00:00.000000001Z</v>`, `a="0001-01-01T00:00:00.000000001Z"`},
	{`Create(2024-02-29T00:00Z, true)`, `<v>2024-02-29T00:00:00Z</v>`, `a="2024-02-29T00:00:00Z"`},
	{`Create(2024-02-29T00:00Z, false)`, `<v>2024-02-29T00:00:00</v>`, `a="2024-02-29T00:00:00"`},
	{`Create(1951-10-22T01:02:03.4-08, true)`, `<v>1951-10-22T01:02:03.4-08:00</v>`, `a="1951-10-22T01:02:03.4-08:00"`},
	{`Create(1951-10-22T01:02:03-08, true).In(UTC)`, `<v>1951-10-22T09:02:03Z</v>`, `a="1951-10-22T09:02:03Z"`},
	{`Create(9999-12-31T23:59:59Z, true)`, `<v>9999-12-31T23:59:59Z</v>`, `a="9999-12-31T23:59:59Z"`},
	{`StripTz(Create(0001-01-01T02:00+02, true))`, ``, `<none>`},
	{`StripTz(Create(2024-02-29T10:00+02, true))`, `<v>2024-02-29T10:00:00</v>`, `a="2024-02-29T10:00:00"`},
	{`decoded ""`, ``, `<none>`},
	{`decoded "0001-01-01T00:00:00"`, `<v>0001-01-01T00:00:00</v>`, `a="0001-01-01T00:00:00"`},
	{`decoded "0001-01-01T00:00:00Z"`, `<v>0001-01-01T00:00:00Z</v>`, `a="0001-01-01T00:00:00Z"`},
	{`decoded "0001-01-01T00:00:00+00:00"`, `<v>0001-01-01T00:00:00Z</v>`, `a="0001-01-01T00:00:00Z"`},
	{`decoded "0001-01-01T00:00:00-00:00"`, `<v>0001-01-01T00:00:00Z</v>`, `a="0001-01-01T00:00:00Z"`},
	{`decoded "0001-01-01T00:00:00.000Z"`, `<v>0001-01-01T00:00:00Z</v>`, `a="0001-01-01T00:00:00Z"`},
	{`decoded "0001-01-01T02:00:00+02:00"`, `<v>0001-01-01T02:00:00+02:00</v>`, `a="0001-01-01T02:00:00+02:00"`},
	{`decoded "0001-01-01T00:00:00+02:00"`, `<v>0001-01-01T00:00:00+02:00</v>`, `a="0001-01-01T00:00:00+02:00"`},
	{`StripTz(decoded "0001-01-01T00:00:00Z")`, `<v>0001-01-01T00:00:00</v>`, `a="0001-01-01T00:00:00"`},
	{`decoded "2024-02-29T00:00:00"`, `<v>2024-02-29T00:00:00</v>`, `a="2024-02-29T00:00:00"`},
	{`decoded "2024-02-29T00:00:00Z"`, `<v>2024-02-29T00:00:00Z</v>`, `a="2024-02-29T00:00:00Z"`},
	{`decoded "2024-02-29T00:00:00.5-08:00"`, `<v>2024-02-29T00:00:00.5-08:00</v>`, `a="2024-02-29T00:00:00.5-08:00"`},
}

var wantXSDDateWire = []wireRow{
	{`zero value`, ``, `<none>`},
	{`Create(time.Time{}, true)`, ``, `<none>`},
	{`Create(time.Time{}, false)`, ``, `<none>`},
	{`Create(time.Time{}.In(+02), true)`, ``, `<none>`},
	{`Create(0001-01-01Z, true)`, ``, `<none>`},
	{`Create(0001-01-01+02, true)`, `<v>0001-01-01+02:00</v>`, `a="0001-01-01+02:00"`},
	{`Create(0001-01-01-08, false)`, `<v>0001-01-01</v>`, `a="0001-01-01"`},
	{`Create(2024-02-29Z, true)`, `<v>2024-02-29Z</v>`, `a="2024-02-29Z"`},
	{`Create(2024-02-29Z, false)`, `<v>2024-02-29</v>`, `a="2024-02-29"`},
	{`Create(1951-10-22-08, true)`, `<v>1951-10-22-08:00</v>`, `a="1951-10-22-08:00"`},
	{`Create(9999-12-31Z, true)`, `<v>9999-12-31Z</v>`, `a="9999-12-31Z"`},
	{`StripTz(Create(1951-10-22-08, true))`, `<v>1951-10-22</v>`, `a="1951-10-22"`},
	{`decoded ""`, ``, `<none>`},
	{`decoded "0001-01-01"`, `<v>0001-01-01</v>`, `a="0001-01-01"`},
	{`decoded "0001-01-01Z"`, `<v>0001-01-01Z</v>`, `a="0001-01-01Z"`},
	{`decoded "0001-01-01+00:00"`, `<v>0001-01-01Z</v>`, `a="0001-01-01Z"`},
	{`decoded "0001-01-01-00:00"`, `<v>0001-01-01Z</v>`, `a="0001-01-01Z"`},
	{`decoded "0001-01-01+02:00"`, `<v>0001-01-01+02:00</v>`, `a="0001-01-01+02:00"`},
	{`decoded "0001-01-01-02:00"`, `<v>0001-01-01-02:00</v>`, `a="0001-01-01-02:00"`},
	{`StripTz(decoded "0001-01-01Z")`, `<v>0001-01-01</v>`, `a="0001-01-01"`},
	{`decoded "2024-02-29"`, `<v>2024-02-29</v>`, `a="2024-02-29"`},
	{`decoded "2024-02-29-08:00"`, `<v>2024-02-29-08:00</v>`, `a="2024-02-29-08:00"`},
}

var wantXSDTimeWire = []wireRow{
	{`zero value`, ``, `<none>`},
	{`Create(0,0,0,0,nil)`, `<v>00:00:00</v>`, `a="00:00:00"`},
	{`Create(0,0,0,0,UTC)`, `<v>00:00:00Z</v>`, `a="00:00:00Z"`},
	{`Create(0,0,0,0,+02)`, `<v>00:00:00+02:00</v>`, `a="00:00:00+02:00"`},
	{`Create(0,0,0,0,-08)`, `<v>00:00:00-08:00</v>`, `a="00:00:00-08:00"`},
	{`Create(23,59,59,999999999,nil)`, `<v>23:59:59.999999999</v>`, `a="23:59:59.999999999"`},
	{`Create(12,13,14,4,-08)`, `<v>12:13:14.000000004-08:00</v>`, `a="12:13:14.000000004-08:00"`},
	{`decoded ""`, ``, `<none>`},
	{`decoded "00:00:00"`, `<v>00:00:00</v>`, `a="00:00:00"`},
	{`decoded "00:00:00Z"`, `<v>00:00:00Z</v>`, `a="00:00:00Z"`},
	{`decoded "00:00:00+00:00"`, `<v>00:00:00Z</v>`, `a="00:00:00Z"`},
	{`decoded "00:00:00-00:00"`, `<v>00:00:00Z</v>`, `a="00:00:00Z"`},
	{`decoded "00:00:00+02:00"`, `<v>00:00:00+02:00</v>`, `a="00:00:00+02:00"`},
	{`decoded "00:00:00-08:00"`, `<v>00:00:00-08:00</v>`, `a="00:00:00-08:00"`},
	{`decoded "00:00:00.000"`, `<v>00:00:00</v>`, `a="00:00:00"`},
	{`decoded "12:13:14.5"`, `<v>12:13:14.5</v>`, `a="12:13:14.5"`},
}

func checkWire[T xml.Marshaler](t *testing.T, values []struct {
	name string
	v    T
}, want []wireRow) {
	t.Helper()
	if len(values) != len(want) {
		t.Fatalf("%d values, %d want rows", len(values), len(want))
	}
	for i, r := range values {
		if r.name != want[i].name {
			t.Fatalf("row %d: value %q, want row %q", i, r.name, want[i].name)
		}
		e, a := marshalElemAttr(t, r.v)
		if e != want[i].elem || a != want[i].attr {
			t.Errorf("%T %s: got element %q attribute %s, want element %q attribute %s",
				r.v, r.name, e, a, want[i].elem, want[i].attr)
		}
	}
}

func TestXSDDateTime_WireBytes(t *testing.T) {
	checkWire(t, xsdDateTimeWireValues(t), wantXSDDateTimeWire)
}

func TestXSDDate_WireBytes(t *testing.T) {
	checkWire(t, xsdDateWireValues(t), wantXSDDateWire)
}

func TestXSDTime_WireBytes(t *testing.T) {
	checkWire(t, xsdTimeWireValues(t), wantXSDTimeWire)
}

// TestXSDDateTimeTypes_RoundTripMinValue decodes year-1 values from a response
// element and attribute and sends them back: every one must be written again,
// with the same timezone form it was received in.
func TestXSDDateTimeTypes_RoundTripMinValue(t *testing.T) {
	type dtMsg struct {
		XMLName xml.Name    `xml:"m"`
		A       XSDDateTime `xml:"a,attr"`
		V       XSDDateTime `xml:"v"`
	}
	type dMsg struct {
		XMLName xml.Name `xml:"m"`
		A       XSDDate  `xml:"a,attr"`
		V       XSDDate  `xml:"v"`
	}
	for _, in := range []string{"0001-01-01T00:00:00", "0001-01-01T00:00:00Z", "0001-01-01T02:00:00+02:00"} {
		doc := `<m a="` + in + `"><v>` + in + `</v></m>`
		var m dtMsg
		if err := xml.Unmarshal([]byte(doc), &m); err != nil {
			t.Fatalf("decoding %s: %v", doc, err)
		}
		out, err := xml.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != doc {
			t.Errorf("xsd:dateTime %q: sent back %s, want %s", in, out, doc)
		}
	}
	for _, in := range []string{"0001-01-01", "0001-01-01Z"} {
		doc := `<m a="` + in + `"><v>` + in + `</v></m>`
		var m dMsg
		if err := xml.Unmarshal([]byte(doc), &m); err != nil {
			t.Fatalf("decoding %s: %v", doc, err)
		}
		out, err := xml.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != doc {
			t.Errorf("xsd:date %q: sent back %s, want %s", in, out, doc)
		}
	}
}

// TestXSDDateTime_DecodedMinValueKeepsToGoTime checks that decoding a
// year-1 dateTime without a timezone still gives the zero time.Time from
// ToGoTime, whatever the local timezone, as it did before such values
// round-tripped.
func TestXSDDateTime_DecodedMinValueKeepsToGoTime(t *testing.T) {
	v := decodeElem[XSDDateTime](t, "0001-01-01T00:00:00")
	if got := v.ToGoTime(); !got.IsZero() || got.Location() != time.UTC {
		t.Errorf("ToGoTime() = %v (%v), want the zero time in UTC", got, got.Location())
	}
}

// TestXSDTime_DecodeEmptyResets checks that decoding an empty xsd:time into a
// value that held a time leaves the zero value, as decoding it into a fresh
// one does, instead of keeping the previous value's timezone flag.
func TestXSDTime_DecodeEmptyResets(t *testing.T) {
	var v struct {
		V XSDTime `xml:"v"`
	}
	if err := xml.Unmarshal([]byte("<p><v>10:00:00+02:00</v></p>"), &v); err != nil {
		t.Fatal(err)
	}
	if err := xml.Unmarshal([]byte("<p><v></v></p>"), &v); err != nil {
		t.Fatal(err)
	}
	if v.V != (XSDTime{}) {
		t.Errorf("after decoding an empty element: %+v, want the zero value", v.V)
	}
	if loc := v.V.Location(); loc != nil {
		t.Errorf("Location() = %v, want nil", loc)
	}
}
