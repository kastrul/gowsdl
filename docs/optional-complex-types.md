# Optional Complex Types in gowsdl

## Overview

An element of a complex type that an instance may leave out, and that occurs
at most once, is generated as a pointer field. A nil pointer is not sent, and
an element missing from a response decodes to nil, so "absent" and "present
but empty" can be told apart. An element may be left out when:

- its own `minOccurs` is `0`,
- it is a member of an `xs:choice`, or
- an `xs:sequence` or `xs:all` with `minOccurs="0"` encloses it within its
  type, or
- it comes from a named model group included with `xs:group ref`, and that
  reference has `minOccurs="0"` or stands in an `xs:choice` or an optional
  `xs:sequence` or `xs:all`. The group's own content counts too: a member of
  a choice in the group is optional wherever the group is used.

This holds for named complex types, complex types with `simpleContent`, `ref`s
to global elements of complex type, and anonymous (inline) complex types.
Required elements of complex type stay struct values and are always sent.
Repeated elements (`maxOccurs="unbounded"` or a number greater than 1) are
slices, and nillable elements are pointers whether optional or not.

## Behavior

### WSDL Definition
```xml
<complexType name="Address">
    <sequence>
        <element name="street" type="string"/>
        <element name="city" type="string"/>
    </sequence>
</complexType>
<complexType name="OrderType">
    <sequence>
        <element name="billingAddress" type="tns:Address"/>
        <element name="shippingAddress" type="tns:Address" minOccurs="0"/>
        <element name="giftNote" minOccurs="0">
            <complexType>
                <sequence><element name="text" type="string"/></sequence>
            </complexType>
        </element>
        <choice>
            <element name="card" type="tns:Address"/>
            <element name="invoice" type="tns:Address"/>
        </choice>
    </sequence>
</complexType>
```

### Generated Go Code
```go
type Address struct {
    Street string `xml:"street" json:"street,omitempty"`
    City   string `xml:"city" json:"city,omitempty"`
}

type OrderType struct {
    // Required - a struct value, always sent
    BillingAddress Address `xml:"billingAddress,omitempty" json:"billingAddress,omitzero"`

    // Optional (minOccurs="0") - a pointer, sent only when set
    ShippingAddress *Address `xml:"shippingAddress,omitempty" json:"shippingAddress,omitempty"`

    // Optional inline complex type - a pointer to its named type
    GiftNote *OrderTypeGiftNoteType `xml:"giftNote,omitempty" json:"giftNote,omitempty"`

    // Choice members - pointers, so only the branch that is set is sent
    Card    *Address `xml:"card,omitempty" json:"card,omitempty"`
    Invoice *Address `xml:"invoice,omitempty" json:"invoice,omitempty"`
}

type OrderTypeGiftNoteType struct {
    Text string `xml:"text" json:"text,omitempty"`
}
```

Inline complex types get named types; see `inline-complex-types-solution.md`.

## XML Marshaling Behavior

### With nil pointers (fields omitted)
```go
order := OrderType{
    BillingAddress: Address{Street: "1 Main St", City: "Tartu"},
    Invoice:        &Address{Street: "2 Side St", City: "Narva"},
}
// <OrderType>
//   <billingAddress><street>1 Main St</street><city>Tartu</city></billingAddress>
//   <invoice><street>2 Side St</street><city>Narva</city></invoice>
// </OrderType>
// No shippingAddress, giftNote or card element.
```

### With a non-nil pointer (field sent in full)
```go
order.ShippingAddress = &Address{Street: "123 Main St", City: "New York"}
// ...<shippingAddress><street>123 Main St</street><city>New York</city></shippingAddress>...
```

A required struct field is sent even when it is zero: `OrderType{}` sends
`<billingAddress><street></street><city></city></billingAddress>`.

## Important: Nil Safety

**When reading optional fields, always check for nil.** An element the server
leaves out of a response decodes to a nil pointer.

```go
// WRONG - panics if the response has no shippingAddress
street := order.ShippingAddress.Street

// CORRECT - check for nil first
if order.ShippingAddress != nil {
    street := order.ShippingAddress.Street
}
```

## Rationale

- `encoding/xml` cannot omit a struct value, even with `omitempty`; it can
  omit a nil pointer.
- A pointer tells a missing element (nil) apart from an empty one (non-nil,
  zero fields).
- No custom MarshalXML methods are needed.

## Migration Guide

Before this change, only optional inline complex types were pointers;
optional elements of a named complex type, `ref`s, choice members and
elements in optional groups were struct values. After regenerating:

### Assigning
```go
// Before
order := OrderType{ShippingAddress: Address{City: "Tartu"}}

// After
order := OrderType{ShippingAddress: &Address{City: "Tartu"}}
```

### Reading
```go
// Before - a missing element read as a zero struct
if order.ShippingAddress.Street != "" {
    // Process shipping address
}

// After - a missing element is nil
if order.ShippingAddress != nil && order.ShippingAddress.Street != "" {
    // Process shipping address
}
```
