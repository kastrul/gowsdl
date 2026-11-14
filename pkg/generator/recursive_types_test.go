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
    </xs:schema>
  </types>
</definitions>`

func TestGenerator_RecursiveComplexTypes_Compile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rec.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(recursiveTypesWSDL), 0o600))

	out := generate(t, path, WithPackage("rec"))
	code := out.client(t)

	assert.Regexp(t, `Cause \*DiagnosticException `, structBody(t, code, "DiagnosticException"))
	owner := structBody(t, code, "Folder")
	home := structBody(t, code, "Person")
	ownerPtr := regexp.MustCompile(`Owner \*Person `).MatchString(owner)
	homePtr := regexp.MustCompile(`Home \*Folder `).MatchString(home)
	assert.True(t, ownerPtr != homePtr,
		"exactly one field of the folder/person cycle should become a pointer:\nFolder:%s\nPerson:%s", owner, home)

	assertGeneratedCodeBuilds(t, "rec", out)
}
