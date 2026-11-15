# test_fixtures

Each case is a WSDL, the values of one call (`*_test_data.json`) and the
request that call must send (`*_request.xml`).
`TestGenerator_TestFixtures_RequestsMatchExpectedXML` in `pkg/generator`
generates a client, makes the call and compares what it sends with the
expected file, by namespace URI and local name (prefix choices do not
matter).

The expected files are derived from the WSDL and the specs, not from the
generator's output. The derivation of each one:

Specs used:

- WSDL 1.1 section 3.5 (`soap:body`): in document style the Body holds the
  parts directly; with `use="literal"` and a part declared with `element`,
  that is the element itself. In RPC style the Body holds one wrapper
  element named after the operation, in the namespace of the `soap:body`
  `namespace` attribute, containing one accessor element per part, named
  after the part.
- WS-I Basic Profile 1.1 R2735: RPC part accessors are in no namespace.
  R2737: their descendants are qualified as the schema of the part's type
  defines.
- XML Schema 1.0 Part 1, section 3.3.2: a local element declaration with no
  `form` takes the schema's `elementFormDefault`, which is `unqualified`
  when absent, so the element is in no namespace. Global elements are in
  the schema's `targetNamespace`.

## document_literal/simple

- Document style, `use="literal"`, one part `parameters` with
  `element="tns:GetUserRequest"`: the Body holds `GetUserRequest`, a global
  element, so in `http://example.com/simple`.
- `userId` and `includeDetails` are local elements; the schema has no
  `elementFormDefault`, so they are in no namespace.

## rpc_literal/basic_rpc

- RPC style; the input's `soap:body` has
  `namespace="http://example.com/rpc"`: the wrapper is `getUser` in
  `http://example.com/rpc`.
- Parts `userId` (`xsd:int`) and `includeDetails` (`xsd:boolean`): one
  accessor each, named after the part, in no namespace (R2735), holding the
  value as text.

## rpc_literal/complex_rpc

- RPC style; `soap:body namespace="http://example.com/complex-rpc"`: the
  wrapper is `GetUserDetails` in that namespace.
- One part, `parameters`, with `type="tns:GetUserDetailsRequest"`: one
  accessor `parameters` in no namespace (R2735). The wrapper sets a default
  namespace here, so the accessor undeclares it with `xmlns=""`.
- Its content is the sequence of `GetUserDetailsRequest`: `userId`,
  `includeAddresses`, `includeMetadata`, local elements of a schema with no
  `elementFormDefault`, so in no namespace (R2737).

## rpc_literal/namespace_edge_case

- RPC style; `soap:body namespace="http://example.com/namespace-test"`: the
  wrapper is `information` in that namespace. Whether a prefix or a default
  namespace declaration puts it there does not matter.
- One part, `parameters`, with `type="tns:InformationRequest"`: one accessor
  `parameters` in no namespace (R2735), holding `dataId` and
  `includeDetails`, local elements with no `elementFormDefault`, so in no
  namespace (R2737).

## document_literal/complex_document

- Document style, `use="literal"`, part `parameters` with
  `element="tns:ProcessOrderRequest"`: the Body holds `ProcessOrderRequest`
  in `http://example.com/complex-document`.
- Every other element is a local element of a schema with no
  `elementFormDefault`, so in no namespace, at any depth.
- Children follow each type's `xsd:sequence`: `orderId`, `customerId`,
  `items` (`item*`: `productId`, `quantity`, `unitPrice`,
  `customizations*`), `shippingAddress`, `billingAddress` (`name`,
  `street1`, `street2?`, `city`, `state`, `zipCode`, `country`), `options`
  (`expedited`, `giftWrap`, `giftMessage?`, `specialInstructions?`).
  Optional elements with no value (`street2` of the billing address,
  `giftMessage`) are absent.
- `giftWrap` has no `minOccurs`, so it is required even though its value is
  `false`; `default="false"` does not make an element optional.
- The test data's values are those the case's earlier expected file held;
  they are nested the way the schema nests them (`items` holds `item`,
  `options` holds `expedited`, `giftWrap` and `specialInstructions`).
