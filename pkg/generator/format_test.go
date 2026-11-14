// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"go/format"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatSource_CollapsesTemplateWhitespace(t *testing.T) {
	src := "package p\n\n" +
		"import (\n\t\"context\"\n\n\t\"encoding/xml\"\n)\n\n" +
		"type T struct {\n\n\tA string `xml:\"a\"`\n\n\n\tLong int `xml:\"long\"`\n\n}\n\n" +
		"const (\n\tX T = 1\n\n\tY T = 2\n)\n\n" +
		"type S interface {\n\n\t//\n\t//\n\t//\n\tDo(ctx context.Context) error\n}\n\n" +
		"var _ = map[string]int{\n\t\"a\": 1,\n\n\t\"b\": 2,\n}\n\n" +
		"var _ xml.Name\n"

	want := "package p\n\n" +
		"import (\n\t\"context\"\n\n\t\"encoding/xml\"\n)\n\n" +
		"type T struct {\n\tA    string `xml:\"a\"`\n\tLong int    `xml:\"long\"`\n}\n\n" +
		"const (\n\tX T = 1\n\tY T = 2\n)\n\n" +
		"type S interface {\n\tDo(ctx context.Context) error\n}\n\n" +
		"var _ = map[string]int{\n\t\"a\": 1,\n\t\"b\": 2,\n}\n\n" +
		"var _ xml.Name\n"

	got, err := FormatSource([]byte(src))
	require.NoError(t, err)
	assert.Equal(t, want, string(got))
}

func TestFormatSource_KeepsCommentsAndBodies(t *testing.T) {
	src := "package p\n\n" +
		"// Doc for T.\n//\n" +
		"type T struct {\n\t// A is documented.\n\t//\n\tA string\n\n\t// free comment\n\n\tB string // trailing\n}\n\n" +
		"var s = `x\n\ny`\n\n" +
		"func f() {\n\ta := 1\n\n\t_ = a\n\n\t_ = []func(){\n\t\tfunc() {\n\t\t\tprintln()\n\n\t\t\tprintln()\n\t\t},\n\t}\n}\n"

	want := "package p\n\n" +
		"// Doc for T.\n" +
		"type T struct {\n\t// A is documented.\n\tA string\n\t// free comment\n\tB string // trailing\n}\n\n" +
		"var s = `x\n\ny`\n\n" +
		"func f() {\n\ta := 1\n\n\t_ = a\n\n\t_ = []func(){\n\t\tfunc() {\n\t\t\tprintln()\n\n\t\t\tprintln()\n\t\t},\n\t}\n}\n"

	got, err := FormatSource([]byte(src))
	require.NoError(t, err)
	assert.Equal(t, want, string(got))
}

func TestFormatSource_IsGofmtStable(t *testing.T) {
	src := "package p\n\ntype T struct {\n\n\tA string\n\n\tB struct {\n\n\t\tC int\n\n\t}\n}\n"

	got, err := FormatSource([]byte(src))
	require.NoError(t, err)

	again, err := format.Source(got)
	require.NoError(t, err)
	assert.Equal(t, string(got), string(again))
	assert.Equal(t, "package p\n\ntype T struct {\n\tA string\n\tB struct {\n\t\tC int\n\t}\n}\n", string(got))
}

func TestFormatSource_InvalidSource(t *testing.T) {
	_, err := FormatSource([]byte("package p\n\nfunc {"))
	require.Error(t, err)
}

func TestFormatSource_EmptyCommentOnFirstLine(t *testing.T) {
	got, err := FormatSource([]byte("//\npackage p\n\ntype T struct {\n\n\tA int\n}\n"))
	require.NoError(t, err)
	assert.Equal(t, "package p\n\ntype T struct {\n\tA int\n}\n", string(got))
}
