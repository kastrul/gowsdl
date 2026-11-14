// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenericCodeGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go run of the gowsdl command in short mode")
	}

	tmpDir, err := os.MkdirTemp(".", "tmp-generic-")
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("failed to remove %s: %v", tmpDir, err)
		}
	})

	wsdlPath := filepath.Join(tmpDir, "test.wsdl")
	require.NoError(t, os.WriteFile(wsdlPath, []byte(genericTestWSDL), 0644))

	tests := []struct {
		name            string
		pkgName         string
		extraArgs       []string
		checkContent    []string
		checkNotContent []string
	}{
		{
			name:    "Standard code generation",
			pkgName: "testpkg",
			checkContent: []string{
				"type UserServicePortType interface",
				"GetUser(ctx context.Context, request *GetUserRequest) (*GetUserResponse, error)",
			},
			checkNotContent: []string{
				"Generic",
				"soap.Result[",
				"CallResult",
			},
		},
		{
			name:      "Generic code generation",
			pkgName:   "testpkggen",
			extraArgs: []string{"-use-generics"},
			checkContent: []string{
				"type UserServicePortType interface",
				"GetUser(ctx context.Context, request *GetUserRequest) (*GetUserResponse, error)",
				"GetUserGeneric(ctx context.Context, request *GetUserRequest) (soap.Result[GetUserResponse], error)",
				`service.client.CallResult[GetUserResponse](ctx, "GetUser", request)`,
				"soap.ErrNilRequest",
			},
			checkNotContent: []string{
				"GenericClient",
			},
		},
		{
			name:      "Generic-only code generation",
			pkgName:   "testpkggenonly",
			extraArgs: []string{"-generics-only"},
			checkContent: []string{
				"type UserServicePortType interface",
				"GetUser(ctx context.Context, request *GetUserRequest) (soap.Result[GetUserResponse], error)",
				`service.client.CallResult[GetUserResponse](ctx, "GetUser", request)`,
			},
			checkNotContent: []string{
				"GetUserGeneric",
				"(*GetUserResponse, error)",
				"GenericClient",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{
				"run", "../../cmd/gowsdl",
				"-p", tt.pkgName,
				"-o", "generated.go",
				"-d", tmpDir,
			}
			args = append(args, tt.extraArgs...)
			args = append(args, wsdlPath)

			cmd := exec.Command("go", args...)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Failed to generate code: %s", string(output))

			pkgDir := filepath.Join(tmpDir, tt.pkgName)
			content, err := os.ReadFile(filepath.Join(pkgDir, "generated.go"))
			require.NoError(t, err)

			contentStr := string(content)
			for _, check := range tt.checkContent {
				assert.Contains(t, contentStr, check)
			}
			for _, check := range tt.checkNotContent {
				assert.NotContains(t, contentStr, check)
			}

			cmd = exec.Command("go", "build", ".")
			cmd.Dir = pkgDir
			output, err = cmd.CombinedOutput()
			require.NoError(t, err, "Generated code does not compile: %s", string(output))

			cmd = exec.Command("go", "vet", ".")
			cmd.Dir = pkgDir
			output, err = cmd.CombinedOutput()
			require.NoError(t, err, "Generated code fails go vet: %s", string(output))
		})
	}
}
