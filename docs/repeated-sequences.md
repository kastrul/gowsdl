# Repeated Sequences in gowsdl

## Overview

An `xs:sequence` with `maxOccurs="unbounded"` (or a number above 1) holds
whole occurrences of its elements, in order: for a sequence of `name` and
`qty`, the content is name, qty, name, qty, and so on. The generator gives
such a sequence one slice field whose element type holds a single
occurrence, so that a request is sent in that order and a response keeps
which elements belong together.

```xml
<xs:complexType name="Order">
  <xs:sequence>
    <xs:element name="id" type="xs:int"/>
    <xs:sequence maxOccurs="unbounded">
      <xs:element name="name" type="xs:string"/>
      <xs:element name="qty" type="xs:int" minOccurs="0"/>
    </xs:sequence>
    <xs:element name="note" type="xs:string"/>
  </xs:sequence>
</xs:complexType>
```

```go
type Order struct {
	Id        int32            `xml:"id" json:"id,omitempty"`
	NameGroup []OrderNameGroup `xml:"name,omitempty" json:"nameGroup,omitempty"`
	Note      string           `xml:"note" json:"note,omitempty"`
}

// OrderNameGroup holds one occurrence of a repeated sequence.
type OrderNameGroup struct {
	Name string `xml:"name" json:"name,omitempty"`
	Qty  int32  `xml:"qty,omitempty" json:"qty,omitempty"`
}
```

`Order{Id: 1, NameGroup: []OrderNameGroup{{Name: "a", Qty: 2}, {Name: "b"}}, Note: "n"}`
is sent as `<id>1</id><name>a</name><qty>2</qty><name>b</name><note>n</note>`.

## Which sequences

- A sequence that may occur more than once and holds more than one member.
  A member is an element, or a nested repeated sequence, which gets its own
  occurrence type and is a slice field of the outer one. The elements of
  choices, and of sequences that occur at most once, inside the repeated
  sequence are members themselves.
- An `xs:group ref` with `maxOccurs` above 1 to a group whose model group is
  a sequence, which stands for that sequence repeated.
- A repeated sequence of one element is not changed: its field is a slice of
  that element.
- Repeated choices are not changed either: each element is a separate slice.
  The order of elements of different branches is not kept.

## Names

The field is named after the group, for a group reference, or after the
first element of the sequence, followed by `Group`: `NameGroup`, `LineGroup`.
If another field of the type has that name, it gets the usual `_` suffix.
The occurrence type is the enclosing type's name followed by the field's
(`OrderNameGroup`), with a number added if a type of that name exists
already. The anonymous types of the members keep the names they would have
without the sequence.

## Field rules within an occurrence

Each element of the occurrence type follows the rules for any element, taken
relative to one occurrence: a required scalar is always sent, even when
zero; an optional element, or an element of a choice, is omitted when zero
and is a pointer if of complex type; an element that repeats within the
occurrence is a slice.

## How it is encoded and decoded

The struct tags of the occurrence type decide everything about its elements.
Its `MarshalXML` method calls `soap.MarshalModelGroup`, which lets
`encoding/xml` encode the occurrence and writes what it wrote without the
enclosing element.

Every type holding such a field, or embedding a type that does (such as the
type of a global element, or a `complexContent` extension), gets an
`UnmarshalXML` method calling `soap.UnmarshalModelGroups`. It decodes the
type as `encoding/xml` would, with one extra field per element of the
sequence, tagged as that element's field in the occurrence type, which puts
each element into the right occurrence. An element starts a new occurrence
when it cannot follow the elements already in the current one: it comes
earlier in the sequence, or it is the same element and the occurrence holds
only one of it. Otherwise it joins the current occurrence. Decoding and then
encoding a value writes the elements back in the order they came.

## Limits

- Grouping is decided from the order of the elements only. Two alternatives
  of a choice inside the sequence that follow one another in schema order go
  into one occurrence, and a repeated choice inside the sequence whose
  elements come back in a different order starts a new occurrence. In both
  cases the elements are sent back in the order they were received.
- If the same element name appears twice in the sequences of one type, every
  element of that name is decoded into the first.
