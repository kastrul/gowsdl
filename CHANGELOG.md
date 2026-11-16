# Changelog

All notable changes to gowsdl are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Entries marked **Regenerate** change the generated code in a way callers can
see (a field is renamed, retyped or added); the rest only affect code that did
not compile or did not work before.

## [Unreleased]

### Added

- SOAP 1.2 client support. `soap.WithSOAP12()` makes a client, and
  `soap.ContextWithSOAP12(ctx)` a single call, send SOAP 1.2: the envelope is
  in the SOAP 1.2 namespace (`soap.XMLNsSoap12Env`) and the action travels as
  the `action` parameter of an `application/soap+xml` Content-Type instead of
  a `SOAPAction` header. SOAP 1.2 faults are decoded into `*soap.SOAPFault`
  (code value, first reason text, role and detail), including Sender faults
  sent with HTTP 400. Responses may use either envelope version. SOAP 1.2 is
  not supported together with MTOM or MIME multipart attachments.
- `parser.WSDLOperation.SOAP12Operation` and the `SOAP12Body` field of
  `parser.WSDLInput` and `parser.WSDLOutput` hold a SOAP 1.2 binding's
  `soap12:operation` and `soap12:body`.
- `soap.AnyType` holds an `xsd:anyType` element: `Decode(&v)` unmarshals a
  received element, `Value` is encoded when sending, and a zero value is
  omitted. Received content is kept as self-contained XML, so namespace
  prefixes the server declared on the SOAP envelope still resolve.
- `AnyType.Type` and `AnyElement.Type` return the element's `xsi:type` with
  its prefix resolved, even when the server declared the prefix on the SOAP
  envelope. `soap.NewDecoder` is the decoder the client uses for this; use it
  instead of `xml.NewDecoder` to get the same when decoding a message
  yourself.
- `soap.AnyElement` holds one element matched by an `xs:any` wildcard. It
  works like `soap.AnyType` and, when encoded, is named by its `XMLName` or by
  its `Value`.
- `soap.Result.Err` holds the error a call failed with: the `*SOAPFault` for a
  SOAP fault, the original error for a transport error. `SOAPFault.Unwrap`
  exposes the cause of a synthetic transport fault, so `errors.Is` and
  `errors.As` see `context.DeadlineExceeded` or an `*HTTPError`.
- A nil `response` makes `soap.Client.CallContext` and `Call` a one-way call:
  an empty reply, such as HTTP 202 Accepted with no body or Content-Type,
  succeeds; a fault or an error status is still reported, and any other
  reply content is ignored.

### Changed

- **Regenerate.** Fields for the elements of a `complexContent` extension's
  sequence now carry a json tag, like every other element field. Their JSON
  key changes from the Go field name to the element name.
- **Regenerate.** The `Value` field of a named complex type with
  `simpleContent` now has `json:"value"`.
- **Regenerate.** An attribute called `Id` or `id` declared in an extension
  now becomes the field `IdAttr`, as it already did when declared directly on
  a type. XML and JSON names are unchanged.
- **Regenerate.** An element with no type information at all (no `ref`, no
  `type`, no inline type) is an `xsd:anyType`, as XML Schema defines, and now
  gets exactly the field an element declared `type="xsd:anyType"` gets:
  `soap.AnyType`, or `[]soap.AnyType` when it repeats, with
  `json:"name,omitzero"` on the single field. Before, it got an undeclared
  type named after the element, a `string`, or no field at all, depending on
  where it stood; v0.2.6 made it `string` everywhere. Read a received value
  with `Decode` or `InnerXML` and send one as `soap.AnyType{Value: v}`; child
  elements in the content are now kept instead of being flattened to their
  text. In JSON the field is now an object (`{"innerXML":"..."}`) instead of
  a string, and an unset field is omitted.
- Generated code spells `interface{}` as `any`. The types are identical.
- **Regenerate.** A global element whose inline `simpleType` has neither a
  union nor a restriction base (an empty one, or an `xs:list`) is now declared
  as `string` instead of `any`. `encoding/xml` silently left the `any` value
  nil when decoding, so the element's text was lost.
- **Regenerate.** `xsd:anyType` fields are `soap.AnyType` instead of `any`,
  and a global element of type `anyType` is a defined type over it.
  `encoding/xml` silently left `any` fields nil when decoding, so their
  content was lost. Assign values as `soap.AnyType{Value: v}`.
- **Regenerate.** `xsd:anySimpleType` fields are `string` instead of `any`,
  for the same reason.
- **Regenerate.** The `Items` field of a type with an `xs:any` wildcard is
  `[]soap.AnyElement` instead of `[]string`. Each item keeps the unknown
  element's name, attributes and full content; before, only its text was
  kept, and items could not be sent with a name.
- Every generated `TargetNamespace` method has a doc comment.
- The library uses the Go 1.26/1.27 idioms suggested by `go fix`. Exported
  signatures are unchanged.
- **Regenerate.** A required element or attribute of a boolean or numeric
  type (`xs:boolean`, the integer types, `xs:float`, `xs:double`,
  `xs:decimal`, and simple types restricting them) is now sent when it holds
  `false` or `0`. Before, its xml tag had `omitempty`, so a client that set
  such a field to `false` or `0` sent no element or attribute at all, and the
  request did not satisfy its own schema. An element counts as required when
  its own `minOccurs` is absent or at least 1, it occurs at most once, it is
  not nillable, and no `xs:choice` and no `xs:sequence` or `xs:all` with
  `minOccurs="0"` encloses it within its type; an attribute when it has
  `use="required"`. Only the xml tag changes: field types, json tags and
  decoding are unchanged, and optional elements, choice members and repeated
  elements are still omitted when zero.
- **Regenerate.** A required element or attribute of a string type
  (`xs:string` and the other string-like built-ins, enumerations and other
  simple types restricting them, unions) now has no `omitempty` either, by the
  same rule as the boolean and numeric entry above. A field left unset is now
  sent as an empty element (`<name></name>`) or attribute (`code=""`), where
  before nothing was sent. That is what the schema asks for, since the element
  or attribute is required, but it can change how a server answers: an empty
  value can fail validation for an enumeration, a pattern or a `minLength`,
  where a lenient server accepted the missing element. Set every required
  string field before calling. Required `xs:dateTime`, `xs:date` and `xs:time`
  fields get the same tag, but `soap.XSDDateTime`, `soap.XSDDate` and
  `soap.XSDTime` still write nothing for a zero value, so for them nothing
  changes on the wire: a required date or time field left unset is still
  omitted from the request. This is deliberate. A zero `time.Time` is Go's
  "not set", and sending it would put the year-1 date `0001-01-01T00:00:00Z`
  in the request, which a server may accept and store, where a missing
  required element gets a validation fault from a server that checks. Set
  every required date and time field before calling.
- **Regenerate.** An optional element of a complex type that occurs at most
  once is now a pointer field, `*T` instead of `T`, with
  `xml:"name,omitempty" json:"name,omitempty"`, the form optional inline
  complex types already had. Optional means what it means for the required
  scalar entries above: `minOccurs="0"`, a member of an `xs:choice`, or inside
  an `xs:sequence` or `xs:all` with `minOccurs="0"`. It covers named complex
  types, `simpleContent` types (a value plus attributes), `ref`s to global
  elements of complex type, and anonymous types in a choice or optional
  group. Before, `encoding/xml` could not omit the struct value, so every
  request sent the element even when the caller never set it, with its
  required children (`<optAddr><city></city></optAddr>`), and a choice sent
  every branch. Now a nil field is not sent, and a response that leaves the
  element out decodes to nil. Required elements of complex type stay struct
  values and are always sent; repeated, nillable and `xs:anyType` fields are
  unchanged. This breaks code using such a field in two ways:

  - Assigning it needs a pointer:

    ```go
    // before
    req := shop.OrderType{ShippingAddress: shop.Address{City: "Tartu"}}
    // after
    req := shop.OrderType{ShippingAddress: &shop.Address{City: "Tartu"}}
    ```

  - Reading it needs a nil check. A response without the element gave a zero
    struct before and gives a nil pointer now, so `resp.ShippingAddress.City`
    panics where it used to read `""`:

    ```go
    // before
    city := resp.ShippingAddress.City
    // after
    city := ""
    if resp.ShippingAddress != nil {
        city = resp.ShippingAddress.City
    }
    ```

### Fixed

- A `soap.XSDDateTime` or `soap.XSDDate` decoded from a response whose text
  was the instant `0001-01-01T00:00:00Z` (such as `0001-01-01T00:00:00`,
  `0001-01-01T02:00:00+02:00` or the date `0001-01-01`; .NET sends
  `DateTime.MinValue` this way) is now written when sent back in a request,
  in the timezone form it was received in. Before, it decoded without error
  but was then marshalled as nothing, so echoing a received record dropped the
  element or attribute, even a required one. Values built with
  `CreateXsdDateTime` or `CreateXsdDate`, and unset ones, are marshalled
  exactly as before, a zero `time.Time` still as nothing. No regeneration is
  needed.
- Decoding an empty `xsd:time` element or attribute into a `soap.XSDTime`
  that already held a time now leaves the zero value, as decoding it into a
  fresh one does. Before, the old timezone flag was kept, so `Location()`
  returned UTC instead of nil. Nothing changes on the wire.
- A global element with no type information at all (no `type`, no inline
  type) is now declared as `xsd:anyType`, exactly as if it were declared
  `type="xsd:anyType"`: a defined type over `soap.AnyType`. Before, it was
  not declared at all, so code with a field referring to it (`ref`) did not
  compile.
- **Regenerate.** A global element whose anonymous `complexType` is a
  `complexContent` extension now has the fields of the extension's sequence
  and attributes, as a named type does. Before, its struct held only the
  embedded base type, so those elements and attributes were never sent and
  were dropped from responses. In the chromedata fixture, 14 request and
  response elements gain their fields, among them `VehicleDescription`,
  `VehicleDescriptionRequest`, `VersionInfo` and `Divisions`.
- **Regenerate.** A WSDL 1.1 port type bound only with SOAP 1.2 now gets a
  client that sends SOAP 1.2, with its SOAP actions from `soap12:operation`;
  a `soap12:binding style="rpc"` makes it RPC-style, with the wrapper
  namespace from `soap12:body`. Before, such a client sent SOAP 1.1 document
  messages with no action. A port type that also has a SOAP 1.1 binding still
  uses SOAP 1.1, and generated servers still answer in SOAP 1.1.
- **Regenerate.** The element wrapping an RPC-style operation's request or
  response is now in the namespace of that input's or output's `soap:body`,
  falling back to the WSDL `targetNamespace` when the binding declares none.
  It used to be in the target namespace of the first schema, so requests were
  sent in the wrong namespace, and responses failed to decode, whenever the
  two differed.
- **Regenerate.** In RPC-style requests, the accessor element of each message
  part is now in no namespace, as WS-I Basic Profile R2735 requires, and the
  content of a part declared with a type follows its schema's
  `elementFormDefault`. Before, the accessors and any unqualified content
  inherited the operation wrapper's default namespace. This changes the
  request XML sent to every RPC-style service: a part is now written as
  `<userId xmlns="">` instead of `<userId>` under `<getUser xmlns="...">`.
  The generated request wrappers implement `xml.Marshaler` using the new
  `soap.MarshalRPCTypePart` and `soap.MarshalRPCElementPart`. Responses are
  read as before, so both forms of accessor are accepted.
- RPC-style operations whose message parts are declared with `element`
  instead of `type` now compile and work. The request and response are the
  part's element itself, and the element is sent inside an accessor element
  named after the part, as WSDL 1.1 section 3.5 specifies. Before, the client
  did not compile when a message had a single element part, and failed at run
  time with an encoding/xml name conflict when it had several; an element part
  of a simple type was sent as the accessor's text instead of inside it.
- A one-way operation (a WSDL 1.1 operation with no `output`, or a WSDL 2.0
  in-only operation) is now generated as a method returning only `error` that
  sends the request and expects no response. It used to be generated as
  returning `(*, error)`, which did not compile. Server code generated for a
  WSDL 1.1 service with a one-way operation now compiles too; as for WSDL 2.0,
  the server does not implement one-way operations.
- A named `simpleType` with no mappable base (a restriction, list or union of
  nested simple types, or an empty one) is now declared as `string` instead of
  being left out, which made every field using it fail to compile.
- In an extension sequence, an element with both a `type` attribute and an
  inline `complexType` now uses the inline type, as everywhere else.

## [0.2.5] - 2026-11-14

### Added

- `soap.Client.CallResult`, a generic method that generated `<Op>Generic` and
  `-generics-only` methods call directly on the shared client.

### Changed

- **Go 1.27 is now required**, for the library and for generated code.
- **Regenerate.** Generated generic operations no longer build one
  `soap.GenericClient` per operation. For large services this removes one
  struct field and one allocation per operation from the client constructor.
- **Regenerate.** An element with an inline `simpleType` now gets the base of
  its restriction at every site. In choices, alls and extension sequences the
  field used to be dropped, and in anonymous nested types it was always a
  `string`.
- **Regenerate.** Attributes from an extension on a named complex type now
  carry a json tag. Their JSON key changes from the Go field name to the
  attribute name.
- WSDL 1.1 and WSDL 2.0 servers are rendered from one template.
  `templates.ServerTemplate` and `templates.ServerWSDL2Template` keep their
  names but are now the same text, executed with the new
  `templates.ServerData`.
- MTOM and MMA share one multipart implementation. Wire bytes are unchanged.
- Inline complex type names are resolved once, by the type itself. A lookup
  that finds no name is now a generation error instead of a reference to a
  type that is never declared.
- The repository is linted with golangci-lint (including gosec and revive).

### Deprecated

- `soap.NewGenericClientFromClient`. It still works, so code generated by
  older versions keeps compiling.

### Fixed

- X-Road WSDLs with mixed qualification generate clients the services accept:
  `elementFormDefault` follows the schema that declares each element,
  `complexContent` restrictions are parsed, refs into imported schemas keep
  their namespace, and header-only messages marshal each part as its own
  header element.
- **Regenerate.** An attribute whose name starts with an underscore, on a
  global element with `simpleContent`, became an unexported field that
  `encoding/xml` ignored. It is now exported.
- An inline complex type inside a `complexContent` extension without a `base`
  was referenced but never declared.
- The `workday-time-min` and `vim` fixtures were incomplete; every fixture
  WSDL now generates code that compiles.

## [0.2.4] - 2026-11-14

### Changed

- `pkg/http.Config` and `HTTPClientConfig` build their client from one
  transport setup. The root package's `DefaultHTTPClientConfig` and the CLI's
  `-http-*` flag defaults come from `pkg/http.DefaultHTTPClientConfig`.
- Port types are matched to their bindings in one place: by exact local name,
  falling back to the suffix-trimming match only when nothing matches exactly.
- `pkg/generator/generator.go` is split into loader, render, funcmap and
  lookup files, and every template is rendered through one helper that wraps
  errors with the template name.
- Dead template copies, placeholder tests and unused `Generator` fields are
  removed.

### Added

- `utils.LocalName` returns the local part of a QName.
- A test compiles the generated client and server code of every fixture WSDL.

### Deprecated

- `pkg/http.Config`; use `HTTPClientConfig`.

### Fixed

- **Regenerate.** An RPC port type whose name starts with a lowercase letter
  got RPC wrapper types but document-style calls.
- **Regenerate.** Port types that differ only by a `PortType`/`Port`/`Type`
  suffix and sit on separate bindings all got the first binding's SOAPAction.
- **Regenerate.** A struct field that makes a generated type contain itself is
  now a pointer, so recursive schemas compile.
- A `complexContent` extension of `xsd:anyType` no longer embeds
  `interface{}`.
- Generated WSDL 2.0 servers no longer panic while dispatching, and return a
  fault for an unknown operation. Generated WSDL 1.1 servers no longer write
  the envelope twice on a bad Content-Type.
- `soap.Binary` is encoded as base64 when MTOM is not used, and MTOM encoding
  handles `[]*soap.Binary`, inline values and nil fields.
- MMA responses are accepted whatever their `start` parameter, the root part
  is recognised by its media type, and attachments are no longer written
  through a nil pointer.
- A cancelled `CallBatch` no longer leaks a goroutine.
- The downloader resets its retry error on success, honours cancellation
  between retries and closes the response body on a cancelled context.
- `ParseLocation` returns an error for unparseable input.
- The generator uses the default HTTP config when given none or a nil one.

## [0.2.3] - 2026-11-14

### Changed

- **Regenerate.** Struct-valued fields use `omitzero` instead of `omitempty`
  in their json tag, since `encoding/json` ignores `omitempty` on structs.
- Generated files import `context` and the soap package only when they use
  them.

### Fixed

- `ref` elements no longer get empty xml and json tag names.
- An element's `XMLName` tag uses the namespace of the schema that declares
  it, not of the first schema in the WSDL.
- Global elements with `simpleContent` get their value and attribute fields.
- Fields for inline complex types resolve to the generated type names.
- Nested choice and sequence groups generate fields.

## [0.2.2] - 2026-11-14

### Changed

- Generated code no longer has a blank line between every struct field, enum
  constant and interface method. `generator.FormatSource` does the formatting.

### Fixed

- **Regenerate.** XSD names starting with an underscore became unexported
  struct fields; they are now exported.
- Port types bound only through HTTP GET/POST are skipped instead of producing
  SOAP clients that do not compile, and operation messages without parts are
  declared as empty types.

## [0.2.1] - 2026-11-14

### Added

- `-generics-only` (`WithGenericsOnly`) replaces the standard methods of WSDL
  1.1 document-style operations with generic ones returning `soap.Result`.

### Fixed

- `-use-generics` generates generic clients for WSDL 1.1 again.

## [0.2.0] - 2026-11-14

### Added

- `-operations` (`WithOperations`) generates client methods only for the named
  operations, and `-prune-types` (`WithTypePruning`) keeps only the types they
  use.

### Fixed

- The WSDL 2.0 namespace is accepted with or without a trailing slash.
- WSDL 2.0 operations use `wsoap:action` as their SOAP action.
- Inline complex types inside `complexContent` extensions are generated.

## [0.1.0] - 2026-11-14

### Changed

- The module is now `github.com/kastrul/gowsdl` and requires Go 1.24.
- Releases can be started manually from the Release workflow.

### Fixed

- X-Road WSDLs generate code that compiles.
- The HTTP client honours `HTTP_PROXY`, `HTTPS_PROXY` and `NO_PROXY` by
  default.

[Unreleased]: https://github.com/kastrul/gowsdl/compare/v0.2.5...HEAD
[0.2.5]: https://github.com/kastrul/gowsdl/compare/v0.2.4...v0.2.5
[0.2.4]: https://github.com/kastrul/gowsdl/compare/v0.2.3...v0.2.4
[0.2.3]: https://github.com/kastrul/gowsdl/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/kastrul/gowsdl/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/kastrul/gowsdl/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/kastrul/gowsdl/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/kastrul/gowsdl/releases/tag/v0.1.0
