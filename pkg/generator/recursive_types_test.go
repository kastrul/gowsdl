package generator

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBreakRecursiveStructs_SelfReference(t *testing.T) {
	src := "type Node struct {\n\tCause Node `xml:\"cause,omitempty\"`\n\tName string `xml:\"name\"`\n}\n"

	got := string(breakRecursiveStructs([]byte(src)))

	assert.Contains(t, got, "Cause *Node `xml:\"cause,omitempty\"`")
	assert.Contains(t, got, "Name string `xml:\"name\"`")
}

func TestBreakRecursiveStructs_IndirectCycle(t *testing.T) {
	src := "type A struct {\n\tB B `xml:\"b\"`\n}\n" +
		"type B struct {\n\tC C `xml:\"c\"`\n}\n" +
		"type C struct {\n\tA A `xml:\"a\"`\n}\n"

	got := string(breakRecursiveStructs([]byte(src)))

	assert.Contains(t, got, "B B `xml:\"b\"`")
	assert.Contains(t, got, "C C `xml:\"c\"`")
	assert.Contains(t, got, "A *A `xml:\"a\"`")
}

func TestBreakRecursiveStructs_CycleThroughDefinedType(t *testing.T) {
	src := "type A struct {\n\tB Alias `xml:\"b\"`\n}\n" +
		"type Alias B\n" +
		"type B struct {\n\tA A `xml:\"a\"`\n}\n"

	got := string(breakRecursiveStructs([]byte(src)))

	assert.Contains(t, got, "A *A `xml:\"a\"`")
}

func TestBreakRecursiveStructs_LeavesValidCodeUnchanged(t *testing.T) {
	src := "type Node struct {\n" +
		"\tChildren []Node `xml:\"child\"`\n" +
		"\tParent *Node `xml:\"parent\"`\n" +
		"\tByName map[string]Node `xml:\"-\"`\n" +
		"\tLeaf Leaf `xml:\"leaf\"`\n" +
		"}\n" +
		"type Leaf struct {\n\tV string `xml:\"v\"`\n}\n"

	assert.Equal(t, src, string(breakRecursiveStructs([]byte(src))))
}

func TestBreakRecursiveStructs_OptionalPointerCountsAsValue(t *testing.T) {
	src := "type A struct {\n\tM M `xml:\"m\"`\n}\n" +
		"type L struct {\n\tM M `xml:\"m\"`\n}\n" +
		"type M struct {\n\tL " + optionalPointer("L") + " `xml:\"l,omitempty\"`\n}\n"

	got := string(breakRecursiveStructs([]byte(src)))

	assert.Contains(t, got, "type L struct {\n\tM *M `xml:\"m\"`", "the field closing the cycle is chosen as if the optional pointer were a value")
	assert.Contains(t, got, "L *(L) `xml:\"l,omitempty\"`", "an optional pointer is not made a pointer again")

	unwrapped := string(unwrapOptionalPointers([]byte(got)))
	assert.Contains(t, unwrapped, "L *L `xml:\"l,omitempty\"`")
	assert.NotContains(t, unwrapped, "*(")
}

func TestUnwrapOptionalPointers_LeavesOtherParenthesesAlone(t *testing.T) {
	src := "type T struct {\n\tA *(B) `xml:\"a\"`\n}\n" +
		"func f(p *T) any { return (*T)(p) }\n"

	assert.Equal(t, "type T struct {\n\tA *B `xml:\"a\"`\n}\n"+
		"func f(p *T) any { return (*T)(p) }\n", string(unwrapOptionalPointers([]byte(src))))
}

func TestBreakRecursiveStructs_UnparseableSourceUnchanged(t *testing.T) {
	src := "type Broken struct {\n\tinterface{}\n}\n"
	assert.Equal(t, src, string(breakRecursiveStructs([]byte(src))))
}

const recursiveTypesWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/" xmlns:xs="http://www.w3.org/2001/XMLSchema"
             xmlns:tns="urn:rec" targetNamespace="urn:rec">
  <types>
    <xs:schema targetNamespace="urn:rec" elementFormDefault="qualified">
      <xs:complexType name="diagnosticException">
        <xs:sequence>
          <xs:element minOccurs="0" name="cause" type="tns:diagnosticException"/>
          <xs:element minOccurs="0" name="message" type="xs:string"/>
        </xs:sequence>
      </xs:complexType>
      <xs:complexType name="folder">
        <xs:sequence>
          <xs:element minOccurs="0" name="owner" type="tns:person"/>
        </xs:sequence>
      </xs:complexType>
      <xs:complexType name="person">
        <xs:sequence>
          <xs:element minOccurs="0" name="home" type="tns:folder"/>
          <xs:element name="name" type="xs:string"/>
        </xs:sequence>
      </xs:complexType>
      <xs:complexType name="aEntry">
        <xs:sequence><xs:element name="first" type="tns:methodFault"/></xs:sequence>
      </xs:complexType>
      <xs:complexType name="localizedFault">
        <xs:sequence><xs:element name="fault" type="tns:methodFault"/></xs:sequence>
      </xs:complexType>
      <xs:complexType name="methodFault">
        <xs:sequence><xs:element name="cause" type="tns:localizedFault" minOccurs="0"/></xs:sequence>
      </xs:complexType>
      <xs:complexType name="left">
        <xs:sequence><xs:element name="right" type="tns:right"/></xs:sequence>
      </xs:complexType>
      <xs:complexType name="right">
        <xs:sequence><xs:element name="left" type="tns:left"/></xs:sequence>
      </xs:complexType>
    </xs:schema>
  </types>
</definitions>`

func TestGenerator_RecursiveComplexTypes_Compile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rec.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(recursiveTypesWSDL), 0o600))

	out := generate(t, path, WithPackage("rec"))
	code := out.client(t)

	assert.Regexp(t, `Cause \*DiagnosticException `, structBody(t, code, "DiagnosticException"))
	assert.Regexp(t, `Owner \*Person `, structBody(t, code, "Folder"), "optional elements of complex type are pointers")
	assert.Regexp(t, `Home \*Folder `, structBody(t, code, "Person"), "optional elements of complex type are pointers")
	left := structBody(t, code, "Left")
	right := structBody(t, code, "Right")
	rightPtr := regexp.MustCompile(`Right \*Right `).MatchString(left)
	leftPtr := regexp.MustCompile(`Left \*Left `).MatchString(right)
	assert.True(t, rightPtr != leftPtr,
		"exactly one field of the required left/right cycle should become a pointer:\nLeft:%s\nRight:%s", left, right)
	assert.Regexp(t, `Cause \*LocalizedFault `, structBody(t, code, "MethodFault"))
	assert.Regexp(t, `Fault \*MethodFault `, structBody(t, code, "LocalizedFault"),
		"a required field keeps the pointer it was given to break the cycle when the optional cause was a value")
	assert.Regexp(t, `First MethodFault `, structBody(t, code, "AEntry"))
	assert.NotContains(t, code, "**")
	assert.NotContains(t, code, "*(")

	assertGeneratedCodeBuilds(t, "rec", out)
}
