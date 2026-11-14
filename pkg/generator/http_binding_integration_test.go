// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// emptyMessageMarshalTest is placed in the package generated from
// httpBindingWSDL11 to check that an empty message leaves the SOAP body empty.
const emptyMessageMarshalTest = `package %s

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/kastrul/gowsdl/soap"
)

func TestEmptyMessageMarshalsToEmptyBody(t *testing.T) {
	out, err := xml.Marshal(&soap.SOAPEnvelope{Body: soap.SOAPBody{Body: &GetAlertsSoapIn{}}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "GetAlertsSoapIn") {
		t.Fatalf("empty message must not produce an element: %%s", out)
	}
}
`

func TestHTTPBindingAndEmptyMessages_GeneratedCodeBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go run of the gowsdl command in short mode")
	}

	tmpDir, err := os.MkdirTemp(".", "tmp-httpbinding-")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	inlinePath := filepath.Join(tmpDir, "schedule.wsdl")
	require.NoError(t, os.WriteFile(inlinePath, []byte(httpBindingWSDL11), 0o644))
	ferryPath := filepath.Join("..", "..", "fixtures", "ferry.wsdl")

	tests := []struct {
		name      string
		wsdl      string
		extraArgs []string
		runTests  bool
	}{
		{name: "ferry", wsdl: ferryPath},
		{name: "ferry operations", wsdl: ferryPath, extraArgs: []string{"-operations", "GetAllAlerts,GetAllRoutes"}},
		{name: "ferry generics", wsdl: ferryPath, extraArgs: []string{"-use-generics"}},
		{name: "ferry generics only", wsdl: ferryPath, extraArgs: []string{"-generics-only"}},
		{name: "ferry generics operations", wsdl: ferryPath, extraArgs: []string{"-use-generics", "-operations", "GetAllAlerts,GetAllRoutes"}},
		{name: "ferry server", wsdl: ferryPath, extraArgs: []string{"-generate-server"}},
		{name: "inline", wsdl: inlinePath, runTests: true},
		{name: "inline generics", wsdl: inlinePath, extraArgs: []string{"-use-generics"}},
		{name: "inline generics only", wsdl: inlinePath, extraArgs: []string{"-generics-only"}},
		{name: "inline operations", wsdl: inlinePath, extraArgs: []string{"-operations", "GetAlerts,Ping"}},
		{name: "inline server", wsdl: inlinePath, extraArgs: []string{"-generate-server"}},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgName := fmt.Sprintf("httpbinding%d", i)
			args := []string{"run", "../../cmd/gowsdl", "-p", pkgName, "-o", "generated.go", "-d", tmpDir}
			args = append(args, tt.extraArgs...)
			args = append(args, tt.wsdl)

			output, err := exec.CommandContext(t.Context(), "go", args...).CombinedOutput()
			require.NoError(t, err, "failed to generate code: %s", output)

			pkgDir := filepath.Join(tmpDir, pkgName)
			if tt.runTests {
				src := fmt.Sprintf(emptyMessageMarshalTest, pkgName)
				require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "empty_message_test.go"), []byte(src), 0o644))
			}

			for _, step := range [][]string{{"build", "."}, {"vet", "."}} {
				cmd := exec.CommandContext(t.Context(), "go", step...)
				cmd.Dir = pkgDir
				output, err := cmd.CombinedOutput()
				require.NoError(t, err, "go %s failed: %s", step[0], output)
			}

			if tt.runTests {
				cmd := exec.CommandContext(t.Context(), "go", "test", ".")
				cmd.Dir = pkgDir
				output, err := cmd.CombinedOutput()
				require.NoError(t, err, "generated package tests failed: %s", output)
			}
		})
	}
}
