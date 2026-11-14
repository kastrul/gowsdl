package generator

import (
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixStructOmitempty_NamedStruct(t *testing.T) {
	src := "type Bar struct {\n\tX string `xml:\"x,omitempty\" json:\"x,omitempty\"`\n}\n" +
		"type Foo struct {\n\tB Bar `xml:\"b,omitempty\" json:\"b,omitempty\"`\n}\n"

	got := string(fixStructOmitempty([]byte(src)))

	assert.Contains(t, got, "B Bar `xml:\"b,omitempty\" json:\"b,omitzero\"`")
	assert.Contains(t, got, "X string `xml:\"x,omitempty\" json:\"x,omitempty\"`")
}

func TestFixStructOmitempty_DateTime(t *testing.T) {
	src := "type Foo struct {\n" +
		"\tA soap.XSDDateTime `xml:\"a,omitempty\" json:\"a,omitempty\"`\n" +
		"\tB soap.XSDDate `xml:\"b,omitempty\" json:\"b,omitempty\"`\n" +
		"\tC soap.XSDTime `xml:\"c,attr,omitempty\" json:\"c,omitempty\"`\n" +
		"}\n"

	got := string(fixStructOmitempty([]byte(src)))

	assert.Contains(t, got, "json:\"a,omitzero\"")
	assert.Contains(t, got, "json:\"b,omitzero\"")
	assert.Contains(t, got, "`xml:\"c,attr,omitempty\" json:\"c,omitzero\"`")
}

func TestFixStructOmitempty_LeavesNonStructs(t *testing.T) {
	src := "type Bar struct{}\n" +
		"type Code string\n" +
		"type Foo struct {\n" +
		"\tS string `json:\"s,omitempty\"`\n" +
		"\tP *Bar `json:\"p,omitempty\"`\n" +
		"\tL []Bar `json:\"l,omitempty\"`\n" +
		"\tC Code `json:\"c,omitempty\"`\n" +
		"\tI interface{} `json:\"i,omitempty\"`\n" +
		"\tT *soap.XSDDateTime `json:\"t,omitempty\"`\n" +
		"}\n"

	got := fixStructOmitempty([]byte(src))

	assert.Equal(t, src, string(got))
}

func TestFixStructOmitempty_AliasChain(t *testing.T) {
	src := "type A B\n" +
		"type B = C\n" +
		"type C struct{}\n" +
		"type D soap.XSDDateTime\n" +
		"type Foo struct {\n" +
		"\tA A `json:\"a,omitempty\"`\n" +
		"\tD D `json:\"d,omitempty\"`\n" +
		"}\n"

	got := string(fixStructOmitempty([]byte(src)))

	assert.Contains(t, got, "json:\"a,omitzero\"")
	assert.Contains(t, got, "json:\"d,omitzero\"")
}

func TestFixStructOmitempty_AnonymousStruct(t *testing.T) {
	src := "type Foo struct {\n" +
		"\tInner struct {\n\t\tX string `json:\"x,omitempty\"`\n\t} `xml:\"inner,omitempty\" json:\"inner,omitempty\"`\n" +
		"}\n"

	got := string(fixStructOmitempty([]byte(src)))

	assert.Contains(t, got, "} `xml:\"inner,omitempty\" json:\"inner,omitzero\"`")
	assert.Contains(t, got, "X string `json:\"x,omitempty\"`")
}

func TestFixStructOmitempty_ExistingOmitzero(t *testing.T) {
	src := "type Bar struct{}\ntype Foo struct {\n\tB Bar `json:\"b,omitempty,omitzero\"`\n}\n"

	got := string(fixStructOmitempty([]byte(src)))

	assert.Contains(t, got, "`json:\"b,omitzero\"`")
}

func TestFixStructOmitempty_InvalidSource(t *testing.T) {
	src := "type Foo struct {\n\tB Bar `json:\"b,omitempty\"`\n"

	assert.Equal(t, src, string(fixStructOmitempty([]byte(src))))
}

func TestGenerator_EC2_StructFieldsUseOmitzero(t *testing.T) {
	code := generate(t, filepath.Join("..", "..", "fixtures", "ec2.wsdl"), WithPackage("ec2")).client(t)

	assert.Regexp(t, `LaunchTime\s+soap\.XSDDateTime\s+`+"`"+`xml:"launchTime,omitempty" json:"launchTime,omitzero"`, code)
	assert.Regexp(t, `\[\]\w+\s+`+"`"+`xml:"\w+,omitempty" json:"\w+,omitempty"`, code)
	assert.NotRegexp(t, regexp.MustCompile(`soap\.XSDDateTime\s+`+"`"+`[^`+"`"+`]*json:"[^"]*omitempty`), code)
}
