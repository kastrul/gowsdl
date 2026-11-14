// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"context"
	"encoding/json"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kastrul/gowsdl/pkg/parser"
)

var elementRefsFixture = filepath.Join("..", "..", "fixtures", "element_refs.wsdl")

func generateElementRefsFixture(t *testing.T) map[string][]byte {
	t.Helper()

	g, err := New(elementRefsFixture, WithPackage("refs"))
	require.NoError(t, err)

	out, err := g.Generate(context.Background())
	require.NoError(t, err)
	return out
}

func structBody(t *testing.T, code, name string) string {
	t.Helper()

	m := regexp.MustCompile(`(?ms)^type ` + regexp.QuoteMeta(name) + ` struct \{\n(.*?)^\}`).FindStringSubmatch(code)
	require.NotNil(t, m, "struct %s not found in generated code", name)
	return regexp.MustCompile(`[ \t]+`).ReplaceAllString(m[1], " ")
}

func TestGenerator_RefElementTagNames(t *testing.T) {
	out := generateElementRefsFixture(t)
	formatted, err := format.Source([]byte(string(out["header"]) + string(out["types"])))
	require.NoError(t, err)
	code := string(formatted)

	assert.NotContains(t, code, "`xml:\",omitempty\"", "ref elements must not produce empty xml tag names")
	assert.NotContains(t, code, "json:\",omitempty\"", "ref elements must not produce empty json tag names")

	tests := []struct {
		structName string
		fields     []string
	}{
		{
			structName: "PartyType",
			fields: []string{
				"PartyClass PartyClass `xml:\"partyClass,omitempty\" json:\"partyClass,omitempty\"`",
				"PartyCode PartyCode `xml:\"urn:refs:other partyCode,omitempty\" json:\"partyCode,omitempty\"`",
				"MemberTag []MemberTag `xml:\"memberTag,omitempty\" json:\"memberTag,omitempty\"`",
			},
		},
		{
			structName: "RepresentedPartyType",
			fields: []string{
				"PartyRole PartyRole `xml:\"urn:refs:repr partyRole,omitempty\" json:\"partyRole,omitempty\"`",
				"PartyId PartyId `xml:\"urn:refs:repr partyId,omitempty\" json:\"partyId,omitempty\"`",
			},
		},
		{
			structName: "ChoiceType",
			fields: []string{
				"ChosenOne ChosenOne `xml:\"chosenOne,omitempty\" json:\"chosenOne,omitempty\"`",
				"ExtraNote ExtraNote `xml:\"urn:refs:other extraNote,omitempty\" json:\"extraNote,omitempty\"`",
			},
		},
		{
			structName: "AllType",
			fields: []string{
				"AnyOrder AnyOrder `xml:\"anyOrder,omitempty\" json:\"anyOrder,omitempty\"`",
			},
		},
		{
			structName: "HolderTypeNestedType",
			fields: []string{
				"PartyClass PartyClass `xml:\"partyClass,omitempty\" json:\"partyClass,omitempty\"`",
				"PartyCode PartyCode `xml:\"urn:refs:other partyCode,omitempty\" json:\"partyCode,omitempty\"`",
			},
		},
		{
			structName: "HolderTypeNestedChoiceType",
			fields: []string{
				"ChosenOne ChosenOne `xml:\"chosenOne,omitempty\" json:\"chosenOne,omitempty\"`",
				"ExtraNote ExtraNote `xml:\"urn:refs:other extraNote,omitempty\" json:\"extraNote,omitempty\"`",
			},
		},
		{
			structName: "HolderTypeNestedAllType",
			fields: []string{
				"AnyOrder AnyOrder `xml:\"anyOrder,omitempty\" json:\"anyOrder,omitempty\"`",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.structName, func(t *testing.T) {
			body := structBody(t, code, tt.structName)
			for _, field := range tt.fields {
				assert.Contains(t, body, field)
			}
		})
	}
}

func TestGenerator_InlineSimpleContentJSONValue(t *testing.T) {
	out := generateElementRefsFixture(t)
	formatted, err := format.Source([]byte(string(out["header"]) + string(out["types"])))
	require.NoError(t, err)
	code := string(formatted)

	assert.NotContains(t, code, `json:"-,"`)
	assert.Contains(t, structBody(t, code, "HolderTypeHashType"), "Value string `xml:\",chardata\" json:\"value\"`")
}

func TestResolveElementRefNamespaces(t *testing.T) {
	sameNS := &parser.XSDElement{Ref: "tns:local"}
	otherNS := &parser.XSDElement{Ref: "oth:remote"}
	unprefixed := &parser.XSDElement{Ref: "bare"}
	defaultNS := &parser.XSDElement{Ref: "viaDefault"}
	unknownPrefix := &parser.XSDElement{Ref: "nope:missing"}
	nested := &parser.XSDElement{Ref: "oth:deep"}
	global := &parser.XSDElement{Ref: "oth:top"}
	extension := &parser.XSDElement{Ref: "oth:ext"}

	ct := &parser.XSDComplexType{
		Name:     "T",
		Sequence: []*parser.XSDElement{sameNS, otherNS, unprefixed, unknownPrefix},
		Choice: []*parser.XSDElement{{
			Name:        "inner",
			ComplexType: &parser.XSDComplexType{All: []*parser.XSDElement{nested}},
		}},
	}
	ct.ComplexContent.Extension.Sequence = []*parser.XSDElement{extension}

	defaultNSSchema := &parser.XSDSchema{
		TargetNamespace: "urn:c",
		Xmlns:           map[string]string{"": "urn:d"},
		ComplexTypes:    []*parser.XSDComplexType{{Name: "D", Sequence: []*parser.XSDElement{defaultNS}}},
	}

	schema := &parser.XSDSchema{
		TargetNamespace: "urn:a",
		Xmlns:           map[string]string{"tns": "urn:a", "oth": "urn:b"},
		ComplexTypes:    []*parser.XSDComplexType{ct},
		Elements: []*parser.XSDElement{{
			Name:        "wrapper",
			ComplexType: &parser.XSDComplexType{Sequence: []*parser.XSDElement{global}},
		}},
	}

	got := resolveElementRefNamespaces([]*parser.XSDSchema{schema, defaultNSSchema}, "urn:a")
	assert.Equal(t, map[*parser.XSDElement]string{
		defaultNS: "urn:d",
		otherNS:   "urn:b",
		nested:    "urn:b",
		global:    "urn:b",
		extension: "urn:b",
	}, got)

	g := &Generator{refNamespaces: got}
	assert.Equal(t, "local", g.refXMLName(sameNS))
	assert.Equal(t, "urn:b remote", g.refXMLName(otherNS))
	assert.Equal(t, "bare", g.refXMLName(unprefixed))
	assert.Equal(t, "urn:d viaDefault", g.refXMLName(defaultNS))
	assert.Equal(t, "missing", g.refXMLName(unknownPrefix))
	assert.Equal(t, "x", (&Generator{}).refXMLName(&parser.XSDElement{Ref: "p:x"}))
}

const refRoundTripProgram = `package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"

	"refs"
)

const incoming = ` + "`" + `<m:GetPartyResponse xmlns:m="urn:refs:main" xmlns:o="urn:refs:other">
	<m:holder>
		<m:nested><m:partyClass>COM</m:partyClass><o:partyCode>12345</o:partyCode></m:nested>
		<m:nestedChoice><o:extraNote>note</o:extraNote></m:nestedChoice>
		<m:nestedAll><m:anyOrder>first</m:anyOrder></m:nestedAll>
		<m:hash algorithmId="SHA-512">abc==</m:hash>
	</m:holder>
	<m:choice><m:chosenOne>picked</m:chosenOne></m:choice>
	<m:all><m:anyOrder>second</m:anyOrder></m:all>
</m:GetPartyResponse>` + "`" + `

const wrongNamespace = ` + "`" + `<GetPartyResponse xmlns="urn:refs:main">
	<holder><nested><partyClass>COM</partyClass><partyCode>12345</partyCode></nested></holder>
</GetPartyResponse>` + "`" + `

func main() {
	req := refs.GetParty{Party: refs.PartyType{
		PartyClass: "COM",
		PartyCode:  "12345",
		MemberTag:  []refs.MemberTag{"a", "b"},
		Represented: refs.RepresentedPartyType{
			PartyRole: "COM",
			PartyId:   "70000000",
		},
	}}
	marshaled, err := xml.Marshal(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var reqBack refs.GetParty
	if err := xml.Unmarshal(marshaled, &reqBack); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var resp refs.GetPartyResponse
	if err := xml.Unmarshal([]byte(incoming), &resp); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var wrong refs.GetPartyResponse
	if err := xml.Unmarshal([]byte(wrongNamespace), &wrong); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result := map[string]any{
		"marshaled":     string(marshaled),
		"requestBack":   reqBack.Party,
		"response":      resp,
		"responseJSON":  string(respJSON),
		"wrongNSHolder": wrong.Holder,
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
`

func TestGenerator_RefElementRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go run of generated code in short mode")
	}

	out := generateElementRefsFixture(t)

	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	goSum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	require.NoError(t, err)

	dir := t.TempDir()
	goMod := "module refs\n\ngo 1.21\n\nrequire github.com/kastrul/gowsdl v0.0.0\n\nreplace github.com/kastrul/gowsdl => " + root + "\n"
	client := string(out["header"]) + string(out["types"]) + string(out["operations"])
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.sum"), goSum, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "refs.go"), []byte(client), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "roundtrip"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", "roundtrip", "main.go"), []byte(refRoundTripProgram), 0o600))

	cmd := exec.Command("go", "run", "-mod=mod", "./cmd/roundtrip")
	cmd.Dir = dir
	cmd.Stderr = &strings.Builder{}
	stdout, err := cmd.Output()
	require.NoError(t, err, "running generated code failed:\n%s", cmd.Stderr)

	var result struct {
		Marshaled   string
		RequestBack map[string]any
		Response    struct {
			Holder struct {
				Nested       map[string]any
				NestedChoice map[string]any
				NestedAll    map[string]any
				Hash         map[string]any
			}
			Choice map[string]any
			All    map[string]any
		}
		ResponseJSON  string
		WrongNSHolder struct {
			Nested map[string]any
		}
	}
	require.NoError(t, json.Unmarshal(stdout, &result), "output: %s", stdout)

	t.Run("marshal uses schema element names and namespaces", func(t *testing.T) {
		assert.Contains(t, result.Marshaled, "<partyClass>COM</partyClass>")
		assert.Contains(t, result.Marshaled, `<partyCode xmlns="urn:refs:other">12345</partyCode>`)
		assert.Contains(t, result.Marshaled, "<memberTag>a</memberTag><memberTag>b</memberTag>")
		assert.NotContains(t, result.Marshaled, "<PartyClass>")
		assert.NotContains(t, result.Marshaled, "<PartyCode")
		assert.Contains(t, result.Marshaled, `<represented><partyRole xmlns="urn:refs:repr">COM</partyRole><partyId xmlns="urn:refs:repr">70000000</partyId></represented>`)
		assert.NotContains(t, result.Marshaled, "<PartyRole>")
	})

	t.Run("marshal then unmarshal preserves values", func(t *testing.T) {
		assert.Equal(t, map[string]any{
			"partyClass": "COM",
			"partyCode":  "12345",
			"memberTag":  []any{"a", "b"},
			"represented": map[string]any{
				"partyRole": "COM",
				"partyId":   "70000000",
			},
		}, result.RequestBack)
	})

	t.Run("unmarshal prefixed document", func(t *testing.T) {
		h := result.Response.Holder
		assert.Equal(t, map[string]any{"partyClass": "COM", "partyCode": "12345"}, h.Nested)
		assert.Equal(t, map[string]any{"extraNote": "note"}, h.NestedChoice)
		assert.Equal(t, map[string]any{"anyOrder": "first"}, h.NestedAll)
		assert.Equal(t, map[string]any{"chosenOne": "picked"}, result.Response.Choice)
		assert.Equal(t, map[string]any{"anyOrder": "second"}, result.Response.All)
	})

	t.Run("simple content value keeps its json key", func(t *testing.T) {
		assert.Equal(t, map[string]any{"value": "abc==", "algorithmId": "SHA-512"}, result.Response.Holder.Hash)
		assert.Contains(t, result.ResponseJSON, `"hash":{"value":"abc==","algorithmId":"SHA-512"}`)
	})

	t.Run("cross-namespace ref does not match element in wrong namespace", func(t *testing.T) {
		assert.Equal(t, map[string]any{"partyClass": "COM"}, result.WrongNSHolder.Nested)
	})
}
