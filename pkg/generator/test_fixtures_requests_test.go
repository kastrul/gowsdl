package generator_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"

	"github.com/kastrul/gowsdl/pkg/generator"
	soaptesting "github.com/kastrul/gowsdl/pkg/testing"
)

// fixtureRequestBinding says how a test_fixtures case's test data maps onto a
// call of its operation, which the fixture files do not record themselves.
type fixtureRequestBinding struct {
	// valuesKey names the test-data key holding the request values; when
	// empty, the values are the top-level keys other than "operation" and
	// "description".
	valuesKey string
	// target is the field path, relative to the generated request message,
	// that the values are decoded into ("" for the message itself).
	target string
}

// fixtureRequestBindings lists every case under test_fixtures whose test data
// can drive the generated client.
var fixtureRequestBindings = map[string]fixtureRequestBinding{
	"document_literal/simple":         {valuesKey: "request"},
	"rpc_literal/basic_rpc":           {valuesKey: "request"},
	"rpc_literal/complex_rpc":         {target: ".Parameters"},
	"rpc_literal/namespace_edge_case": {target: ".Parameters"},
}

// undrivableFixtureRequests lists test_fixtures cases whose test data does not
// carry what a call needs, with what is missing. The test checks that their
// values still fail to decode into the generated request, so a case is moved
// to fixtureRequestBindings once its data is completed.
var undrivableFixtureRequests = map[string]string{
	"document_literal/array_namespace_edge_case": "the test data flattens every wrapper type: stringArray, intArray and " +
		"emptyArray are bare JSON arrays where the schema has the complex types StringArray (item*), IntArray (value*) " +
		"and EmptyArray, and nestedArrays is a bare array where NestedArrayContainer holds level1 elements",
	"document_literal/complex_document": "the test data has no values for shippingAddress (required) or billingAddress, " +
		"which the expected XML contains, gives items as a bare array where OrderItemArray holds item elements, and puts " +
		"expedited, giftWrap and specialInstructions at the top level instead of under options",
}

// knownFixtureRequestMismatches lists test_fixtures cases whose captured
// request is known to differ from the expected XML, with the differences and
// which side is wrong. The test fails when a listed case starts matching, so
// the table can only shrink.
var knownFixtureRequestMismatches = map[string]struct{ diff, diagnosis string }{
}

// fixtureRequestDriver calls one generated operation with values decoded from
// a JSON file. It exits with status 2 when the values do not decode into the
// request, and reports a failed call on stderr without failing, since only the
// outgoing request matters.
const fixtureRequestDriver = `package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/kastrul/gowsdl/soap"

	"fixturedriver/svc"
)

func main() {
	data, err := os.Open(os.Args[2])
	if err != nil {
		panic(err)
	}
	var request svc.%[1]s
	dec := json.NewDecoder(data)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&request%[2]s); err != nil {
		fmt.Fprintln(os.Stderr, "decode test data:", err)
		os.Exit(2)
	}
	client := svc.%[3]s(soap.NewClient(os.Args[1]))
	if _, err := client.%[4]s(context.Background(), &request); err != nil {
		fmt.Fprintln(os.Stderr, "call:", err)
	}
}
`

// fixtureCase is one WSDL, *_request.xml and *_test_data.json triple under
// test_fixtures, keyed by its directory and base name.
type fixtureCase struct {
	key, wsdl, expected, testData string
}

func testFixtureCases(t *testing.T, root string) []fixtureCase {
	t.Helper()

	wsdls, err := filepath.Glob(filepath.Join(root, "test_fixtures", "*", "*.wsdl"))
	require.NoError(t, err)
	require.NotEmpty(t, wsdls)
	sort.Strings(wsdls)

	cases := make([]fixtureCase, 0, len(wsdls))
	for _, wsdl := range wsdls {
		base := strings.TrimSuffix(wsdl, ".wsdl")
		cases = append(cases, fixtureCase{
			key:      filepath.Base(filepath.Dir(wsdl)) + "/" + filepath.Base(base),
			wsdl:     wsdl,
			expected: base + "_request.xml",
			testData: base + "_test_data.json",
		})
	}
	return cases
}

// fixtureOperation finds the generated client method for operation and
// returns its request type and the constructor of its service.
func fixtureOperation(t *testing.T, client []byte, operation string) (method, requestType, constructor string) {
	t.Helper()

	r := []rune(operation)
	r[0] = unicode.ToUpper(r[0])
	method = string(r)

	file, err := parser.ParseFile(token.NewFileSet(), "client.go", client, 0)
	require.NoError(t, err)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name.Name != method || len(fn.Type.Params.List) != 2 {
			continue
		}
		recv := fn.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		param := fn.Type.Params.List[1].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		r := []rune(recv)
		r[0] = unicode.ToUpper(r[0])
		return method, param, "New" + string(r)
	}
	t.Fatalf("generated client has no method %s for operation %q", method, operation)
	return "", "", ""
}

// fixtureRequestValues extracts the request values from a case's test data.
func fixtureRequestValues(t *testing.T, testData []byte, key string) (operation string, values []byte) {
	t.Helper()

	var data map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(testData, &data))
	require.NoError(t, json.Unmarshal(data["operation"], &operation), "test data needs an operation")

	if key != "" {
		require.Contains(t, data, key)
		return operation, data[key]
	}
	delete(data, "operation")
	delete(data, "description")
	values, err := json.Marshal(data)
	require.NoError(t, err)
	return operation, values
}

// captureFixtureRequest generates the client for wsdl, calls operation with
// values through a driver program against a local server and returns the
// request body the server received. ok is false when the values did not
// decode into the request.
func captureFixtureRequest(t *testing.T, root, wsdl, operation string, values []byte, target string) (body string, ok bool) {
	t.Helper()

	g, err := generator.New(wsdl, generator.WithPackage("svc"))
	require.NoError(t, err)
	out, err := g.Generate(context.Background())
	require.NoError(t, err)
	client, err := format.Source(bytes.Join([][]byte{out["header"], out["types"], out["operations"]}, nil))
	require.NoError(t, err)

	method, requestType, constructor := fixtureOperation(t, client, operation)

	dir := t.TempDir()
	goSum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	require.NoError(t, err)
	goMod := "module fixturedriver\n\ngo 1.21\n\nrequire github.com/kastrul/gowsdl v0.0.0\n\nreplace github.com/kastrul/gowsdl => " + root + "\n"
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "svc"), 0o750))
	for name, content := range map[string][]byte{
		"go.mod":        []byte(goMod),
		"go.sum":        goSum,
		"svc/client.go": client,
		"main.go":       fmt.Appendf(nil, fixtureRequestDriver, requestType, target, constructor, method),
		"values.json":   values,
	} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), content, 0o600))
	}

	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		mu.Lock()
		body = string(data)
		mu.Unlock()
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = io.WriteString(w, `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body/></soap:Envelope>`)
	}))
	defer server.Close()

	build := exec.CommandContext(t.Context(), "go", "build", "-mod=mod", "-o", "driver", ".")
	build.Dir = dir
	output, err := build.CombinedOutput()
	require.NoError(t, err, "driver does not build:\n%s", output)

	cmd := exec.CommandContext(t.Context(), filepath.Join(dir, "driver"), server.URL, "values.json")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	if exit, isExit := errors.AsType[*exec.ExitError](err); isExit && exit.ExitCode() == 2 {
		t.Logf("driver: %s", strings.TrimSpace(stderr.String()))
		return "", false
	}
	require.NoError(t, err, "driver failed:\n%s", stderr.String())
	if stderr.Len() > 0 {
		t.Logf("driver: %s", strings.TrimSpace(stderr.String()))
	}

	mu.Lock()
	defer mu.Unlock()
	require.NotEmpty(t, body, "driver sent no request")
	return body, true
}

// requestDifferences compares a captured request with the expected one and
// returns the differences, one per line in a stable order.
func requestDifferences(t *testing.T, expected, actual string) string {
	t.Helper()

	result, err := soaptesting.NewXMLComparator().CompareSOAPRequests(expected, actual)
	require.NoError(t, err)
	lines := make([]string, 0, len(result.Differences))
	for _, d := range result.Differences {
		lines = append(lines, fmt.Sprintf("%s %s: expected %q, actual %q", d.Type, d.Path, d.Expected, d.Actual))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func TestGenerator_TestFixtures_RequestsMatchExpectedXML(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go run of generated clients in short mode")
	}

	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	cases := testFixtureCases(t, root)

	seen := make(map[string]bool, len(cases))
	for _, c := range cases {
		seen[c.key] = true
		t.Run(c.key, func(t *testing.T) {
			t.Parallel()

			binding, drivable := fixtureRequestBindings[c.key]
			missing, undrivable := undrivableFixtureRequests[c.key]
			require.True(t, drivable != undrivable, "%s must be listed in exactly one of fixtureRequestBindings and undrivableFixtureRequests", c.key)

			testData, err := os.ReadFile(c.testData)
			require.NoError(t, err)
			expected, err := os.ReadFile(c.expected)
			require.NoError(t, err)

			operation, values := fixtureRequestValues(t, testData, binding.valuesKey)
			actual, decoded := captureFixtureRequest(t, root, c.wsdl, operation, values, binding.target)
			if undrivable {
				require.False(t, decoded, "test data of %s now decodes into the request; move it to fixtureRequestBindings (was: %s)", c.key, missing)
				t.Logf("not drivable: %s", missing)
				return
			}
			require.True(t, decoded, "test data of %s does not decode into the request", c.key)

			diff := requestDifferences(t, string(expected), actual)
			known, isKnown := knownFixtureRequestMismatches[c.key]
			switch {
			case diff != "" && !isKnown:
				t.Errorf("request for %s does not match %s:\n%s\nactual request:\n%s", c.key, filepath.Base(c.expected), diff, actual)
			case diff == "" && isKnown:
				t.Errorf("request for %s now matches; remove it from knownFixtureRequestMismatches", c.key)
			case diff != known.diff:
				t.Errorf("request for %s differs from %s differently than recorded in knownFixtureRequestMismatches:\ngot:\n%s\nrecorded:\n%s\nactual request:\n%s", c.key, filepath.Base(c.expected), diff, known.diff, actual)
			case diff != "":
				t.Logf("known mismatch (%s):\n%s", known.diagnosis, diff)
			}
		})
	}

	for _, table := range []map[string]bool{keys(fixtureRequestBindings), keys(undrivableFixtureRequests), keys(knownFixtureRequestMismatches)} {
		for key := range table {
			if !seen[key] {
				t.Errorf("%s is listed but is not a case under test_fixtures", key)
			}
		}
	}
}

func keys[V any](m map[string]V) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}
