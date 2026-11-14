package generator

import (
	"context"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// knownNonCompilingFixtures lists repository WSDL fixtures, relative to the
// repository root, whose generated client and server code is known not to
// compile, each with the reason. TestGenerator_AllFixtures_GeneratedCodeCompiles
// fails when a fixture listed here starts compiling, so entries must be removed
// as the underlying bugs are fixed and the table can only shrink.
var knownNonCompilingFixtures = map[string]string{}

// fixtureWSDLs returns every *.wsdl file in a fixtures or test_fixtures
// directory of the repository, relative to the repository root.
func fixtureWSDLs(t *testing.T, root string) []string {
	t.Helper()

	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() || filepath.Ext(path) != ".wsdl" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		for dir := range strings.SplitSeq(filepath.ToSlash(filepath.Dir(rel)), "/") {
			if dir == "fixtures" || dir == "test_fixtures" {
				files = append(files, filepath.ToSlash(rel))
				break
			}
		}
		return nil
	})
	require.NoError(t, err)
	sort.Strings(files)
	return files
}

// compileFixture generates client and server code for wsdl and builds it,
// returning a description of the first failure, or "" when it compiles.
func compileFixture(t *testing.T, wsdl string) string {
	t.Helper()

	g, err := New(wsdl, WithPackage("fixture"), WithServerGeneration(true))
	if err != nil {
		return fmt.Sprintf("generator.New: %v", err)
	}
	out, err := g.Generate(context.Background())
	if err != nil {
		return fmt.Sprintf("Generate: %v", err)
	}

	dir := writeGeneratedModule(t, "fixture", out, nil)
	cmd := exec.Command("go", "build", "-mod=mod", "./...")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Sprintf("go build: %v\n%s", err, output)
	}
	return ""
}

func TestGenerator_AllFixtures_GeneratedCodeCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go build of generated code in short mode")
	}

	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	fixtures := fixtureWSDLs(t, root)
	require.NotEmpty(t, fixtures)

	seen := make(map[string]bool, len(fixtures))
	for _, rel := range fixtures {
		seen[rel] = true
		t.Run(rel, func(t *testing.T) {
			t.Parallel()

			failure := compileFixture(t, filepath.Join(root, filepath.FromSlash(rel)))
			reason, known := knownNonCompilingFixtures[rel]
			switch {
			case failure != "" && !known:
				t.Errorf("generated code for %s does not compile:\n%s", rel, failure)
			case failure == "" && known:
				t.Errorf("%s now compiles; remove it from knownNonCompilingFixtures (was: %s)", rel, reason)
			case failure != "":
				t.Logf("known failure (%s):\n%s", reason, failure)
			}
		})
	}

	for rel := range knownNonCompilingFixtures {
		if !seen[rel] {
			t.Errorf("knownNonCompilingFixtures lists %s, which is not a fixture WSDL", rel)
		}
	}
}
