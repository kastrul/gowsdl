// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// minimalServiceWSDL is a one-operation document/literal WSDL, also used by
// the root package's tests.
const minimalServiceWSDL = "testdata/minimal_service.wsdl"

// writeWSDL writes wsdl to a file in a new temporary directory and returns
// the file's path.
func writeWSDL(tb testing.TB, wsdl string) string {
	tb.Helper()

	path := filepath.Join(tb.TempDir(), "service.wsdl")
	require.NoError(tb, os.WriteFile(path, []byte(wsdl), 0o600))
	return path
}

// generate runs the generator on the WSDL file at path with exactly opts and
// fails the test if construction or generation fails or the client is not
// valid Go.
func generate(tb testing.TB, path string, opts ...Option) generated {
	tb.Helper()

	out, err := tryGenerate(tb, path, opts...)
	require.NoError(tb, err)
	out.client(tb)
	return out
}

// tryGenerate runs the generator on the WSDL file at path with exactly opts
// and returns the error from New or Generate.
func tryGenerate(tb testing.TB, path string, opts ...Option) (generated, error) {
	tb.Helper()

	g, err := New(path, opts...)
	if err != nil {
		return nil, err
	}
	out, err := g.Generate(context.Background())
	return generated(out), err
}

// generated is the generator's output, keyed by part.
type generated map[string][]byte

// types returns the generated types part.
func (g generated) types() string { return string(g["types"]) }

// operations returns the generated operations part.
func (g generated) operations() string { return string(g["operations"]) }

// client returns the header, types and operations parts formatted by gofmt,
// failing the test if they are not valid Go.
func (g generated) client(tb testing.TB) string {
	tb.Helper()

	code := string(g["header"]) + string(g["types"]) + string(g["operations"])
	formatted, err := format.Source([]byte(code))
	require.NoError(tb, err, "generated code should be valid Go:\n%s", code)
	return string(formatted)
}

// decl returns one declaration of the formatted client: a type or function
// by name, or a method as "Recv.Method" with the receiver's type name.
func (g generated) decl(tb testing.TB, name string) string {
	tb.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "client.go", g.client(tb), 0)
	require.NoError(tb, err)

	recv, fn, isMethod := strings.Cut(name, ".")
	if !isMethod {
		fn = name
	}
	var node any
	for _, d := range file.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok && !isMethod && ts.Name.Name == name {
					node = &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{ts}}
				}
			}
		case *ast.FuncDecl:
			if d.Name.Name == fn && receiverName(d) == recv {
				node = d
			}
		}
	}
	require.NotNil(tb, node, "generated client has no declaration %s", name)

	var buf bytes.Buffer
	require.NoError(tb, printer.Fprint(&buf, fset, node))
	return buf.String()
}

// typeNames returns the names of the types the formatted client declares, in
// declaration order.
func (g generated) typeNames(tb testing.TB) []string {
	tb.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "client.go", g.client(tb), 0)
	require.NoError(tb, err)

	var names []string
	for _, d := range file.Decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
			for _, spec := range gd.Specs {
				names = append(names, spec.(*ast.TypeSpec).Name.Name)
			}
		}
	}
	return names
}

// declNames returns the names of the formatted client's type and function
// declarations, with methods as "Recv.Method", in declaration order.
func (g generated) declNames(tb testing.TB) []string {
	tb.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "client.go", g.client(tb), 0)
	require.NoError(tb, err)

	var names []string
	for _, d := range file.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					names = append(names, ts.Name.Name)
				}
			}
		case *ast.FuncDecl:
			if recv := receiverName(d); recv != "" {
				names = append(names, recv+"."+d.Name.Name)
			} else {
				names = append(names, d.Name.Name)
			}
		}
	}
	return names
}

// moduleFiles returns the unformatted output as files for goModule:
// client.go, and server.go when server code was generated, under dir.
func (g generated) moduleFiles(dir string) map[string]string {
	files := map[string]string{
		filepath.Join(dir, "client.go"): string(g["header"]) + string(g["types"]) + string(g["operations"]),
	}
	if len(g["server"]) > 0 {
		files[filepath.Join(dir, "server.go")] = string(g["server_header"]) + string(g["server_wsdl"]) + string(g["server"])
	}
	return files
}

// structBody returns the source between the braces of the struct type name
// declared in code, without the first newline and with runs of spaces and
// tabs collapsed to one space.
func structBody(t *testing.T, code, name string) string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "generated.go", code, parser.ParseComments)
	require.NoError(t, err)

	var fields *ast.FieldList
	ast.Inspect(file, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == name {
			if st, ok := ts.Type.(*ast.StructType); ok {
				fields = st.Fields
			}
		}
		return fields == nil
	})
	require.NotNil(t, fields, "struct %s not found in generated code", name)

	body := code[fset.Position(fields.Opening).Offset+1 : fset.Position(fields.Closing).Offset]
	body = strings.TrimPrefix(body, "\n")
	return regexp.MustCompile(`[ \t]+`).ReplaceAllString(body, " ")
}

// declaredTypes returns the sorted names of every type declared in the
// generated types part src, including types local to function bodies.
func declaredTypes(src string) []string {
	file, err := parser.ParseFile(token.NewFileSet(), "types.go", "package p\n"+src, 0)
	if err != nil {
		panic(fmt.Sprintf("generated types do not parse: %v", err))
	}
	var names []string
	ast.Inspect(file, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			names = append(names, ts.Name.Name)
		}
		return true
	})
	sort.Strings(names)
	return names
}

// receiverName returns the type name of fn's receiver, or "" for a function.
func receiverName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	typ := fn.Recv.List[0].Type
	if star, ok := typ.(*ast.StarExpr); ok {
		typ = star.X
	}
	if id, ok := typ.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// goModule writes files, keyed by slash-separated path, into a new temporary
// Go module named module that requires this repository through a replace
// directive, and returns the module's directory. Using the module needs the
// go toolchain, so goModule skips the test in short mode.
func goModule(tb testing.TB, module string, files map[string]string) string {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping go toolchain run of generated code in short mode")
	}

	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(tb, err)
	goSum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	require.NoError(tb, err)

	dir := tb.TempDir()
	all := map[string]string{
		"go.mod": "module " + module + "\n\ngo 1.21\n\nrequire github.com/kastrul/gowsdl v0.0.0\n\nreplace github.com/kastrul/gowsdl => " + root + "\n",
		"go.sum": string(goSum),
	}
	for name, content := range files {
		all[name] = content
	}
	for name, content := range all {
		path := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(tb, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(tb, os.WriteFile(path, []byte(content), 0o600))
	}
	return dir
}

// runGo runs the go tool with args in dir, adding -mod=mod after the
// subcommand, and returns its standard output. It fails the test with the
// command's output when the command fails.
func runGo(tb testing.TB, dir string, args ...string) string {
	tb.Helper()

	stdout, err := tryGo(dir, args...)
	require.NoError(tb, err, "go %s failed:\n%s", strings.Join(args, " "), stdout)
	return stdout
}

// tryGo is runGo returning the failure instead: on error, the returned
// string holds the command's standard output and standard error.
func tryGo(dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "go", append([]string{args[0], "-mod=mod"}, args[1:]...)...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err != nil {
		return string(stdout) + stderr.String(), err
	}
	return string(stdout), nil
}

// assertGeneratedCodeBuilds builds the generator output, client and server,
// as a temporary module.
func assertGeneratedCodeBuilds(tb testing.TB, module string, out generated) {
	tb.Helper()
	runGo(tb, goModule(tb, module, out.moduleFiles("")), "build", "./...")
}

// runGeneratedModuleTests runs go test on the generator output, client and
// server, together with the extra files.
func runGeneratedModuleTests(tb testing.TB, module string, out generated, extra map[string]string) {
	tb.Helper()

	files := out.moduleFiles("")
	for name, content := range extra {
		files[name] = content
	}
	runGo(tb, goModule(tb, module, files), "test", "-count=1", "./...")
}
