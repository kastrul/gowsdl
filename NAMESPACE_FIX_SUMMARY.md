# Namespace Handling Fix Summary

## Problem
After PR #32 was merged, users reported `java.lang.NumberFormatException: For input string: "null"` errors when calling SOAP services. The issue was caused by incorrect namespace handling for schemas where `elementFormDefault` is not specified (defaults to "unqualified").

## Root Cause
When `elementFormDefault="unqualified"` (the default), only top-level elements should be namespace-qualified, while local/child elements should be unqualified. However, when using Go's default namespace declaration (`xmlns="..."`), all child elements inherit the namespace, making them qualified when they shouldn't be.

## Solution
Modified gowsdl to generate operation types with namespace prefixes (e.g., `tns:OperationName`) when `elementFormDefault` is unqualified. This prevents child elements from inheriting the namespace.

The `elementFormDefault` that counts is the one on the schema that declares the element or complex type, not a WSDL-wide setting (see [Per-Schema Qualification](#per-schema-qualification)).

## Changes Made

### 1. Enhanced SOAP Envelope (`soap/namespace_envelope.go`)
- Created `EnhancedSOAPEnvelope` struct that supports target namespace declarations
- Added `NewEnhancedSOAPEnvelope` function to create envelopes with proper namespace handling

### 2. Modified SOAP Client (`soap/soap.go`)
- Updated the `call` method to use `EnhancedSOAPEnvelope` when the request implements `TargetNamespace()` method
- Maintains backward compatibility by using standard envelope when target namespace is not available

### 3. Updated Types Template (`pkg/generator/templates/types.go`)
- Modified XMLName generation to use namespace prefix (`tns:`) when the declaring schema's `elementFormDefault` is unqualified
- Added `TargetNamespace()` method generation for operation types
- Keeps full namespace format when the declaring schema has `elementFormDefault="qualified"`
- The template asks `isQualified .` for each declaration; it is backed by `declarationNamespaces.isQualified` in `pkg/generator/schema_namespaces.go`

### 4. Added Comprehensive Tests
- `pkg/generator/xmlname_test.go`: `TestGenerator_XMLNameAndTargetNamespace` covers the XMLName and `TargetNamespace()` output per `elementFormDefault`
- `pkg/generator/schema_namespaces_test.go`: per-schema namespace and qualification tests
- `soap/namespace_envelope_simple_test.go`: Tests for the SOAP envelope structure

## Example

### Before (Incorrect)
```xml
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GOWebService_SendungsErstellung xmlns="https://wsdemo.ax4.com/ws/GeneralOvernight">
      <Versender>2611570</Versender>
      <!-- Child elements inherit namespace - WRONG! -->
    </GOWebService_SendungsErstellung>
  </soap:Body>
</soap:Envelope>
```

### After (Correct)
```xml
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" 
               xmlns:tns="https://wsdemo.ax4.com/ws/GeneralOvernight">
  <soap:Body>
    <tns:GOWebService_SendungsErstellung>
      <Versender>2611570</Versender>
      <!-- Child elements are unqualified - CORRECT! -->
    </tns:GOWebService_SendungsErstellung>
  </soap:Body>
</soap:Envelope>
```

## Per-Schema Qualification

### Problem
The first version of this fix decided qualification for the whole WSDL: if *any* schema had `elementFormDefault="qualified"`, every operation element was generated in qualified form. WSDLs that mix schemas broke. X-Road producer WSDLs are a common case: the service schema leaves `elementFormDefault` unset (unqualified), while a small inline X-Road schema next to it is qualified:

```xml
<xsd:schema targetNamespace="http://kirst.x-road.eu">
  <xsd:element name="kindlustatus_nt">…</xsd:element>
</xsd:schema>
<xsd:schema targetNamespace="http://x-road.eu/xsd/xroad.xsd" elementFormDefault="qualified">
  <xsd:element name="consentService" type="xroad:consentServiceType"/>
</xsd:schema>
```

The qualified X-Road schema made `kindlustatus_nt` qualified too. It was generated as `xml:"http://kirst.x-road.eu kindlustatus_nt"` with no `TargetNamespace()`, so its unqualified children inherited the service namespace:

```xml
<kindlustatus_nt xmlns="http://kirst.x-road.eu">
  <request><isikukood>…</isikukood></request>  <!-- WRONG: inherited namespace -->
</kindlustatus_nt>
```

### Fix
`resolveDeclarationNamespaces` records the `elementFormDefault` of the schema that declares each global element and complex type, alongside its target namespace. Each declaration is then generated according to its own schema:

- **Unqualified schema** (e.g. `kindlustatus_nt`): `xml:"tns:kindlustatus_nt"` plus `TargetNamespace()`, so `<request>` and `<isikukood>` stay unqualified.
- **Qualified schema** (e.g. `consentService`): full namespace format, `xml:"http://x-road.eu/xsd/xroad.xsd consentService"`.

A declaration that can't be traced to a schema with a `targetNamespace` falls back to the old rule (qualified if any schema is).

### Tests
- `TestGenerator_XRoad_MixedQualificationTypes` and `TestGenerator_XRoad_MixedQualificationEnvelope` (`pkg/generator/xroad_envelope_test.go`) generate a client from a mixed-qualification WSDL and check both the struct tags and the envelope actually sent.
- `TestGenerator_XRoad_UnqualifiedBodyBesideQualifiedImports` (`pkg/generator/xroad_fixture_test.go`) covers `fixtures/xroad`, whose unqualified service schema imports qualified header schemas.

## Usage
The fix is automatic. Generated code will now:
1. Use namespace prefixes for operation elements when appropriate
2. Generate `TargetNamespace()` methods on operation types
3. Create proper SOAP envelopes with namespace declarations

## Backward Compatibility
The changes maintain full backward compatibility:
- Existing code continues to work without modifications
- Declarations in schemas with `elementFormDefault="qualified"` continue to use full namespace format
- WSDLs whose schemas all share one `elementFormDefault` generate the same code as before; only mixed-qualification WSDLs change, and only for their unqualified schemas
- Standard SOAP envelope is used when target namespace is not available