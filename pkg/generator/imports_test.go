package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReferencedPackages(t *testing.T) {
	tests := []struct {
		name string
		code []string
		want map[string]bool
	}{
		{
			name: "qualified identifiers across fragments",
			code: []string{
				"type T struct { At soap.XSDDateTime }",
				"func (c *C) Do(ctx context.Context) error { return nil }",
			},
			want: map[string]bool{"soap": true, "context": true},
		},
		{
			name: "no qualified identifiers",
			code: []string{"type T struct { Name string }", ""},
			want: map[string]bool{},
		},
		{
			name: "parse error keeps context and soap",
			code: []string{"type T struct {"},
			want: map[string]bool{"context": true, "soap": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := make([][]byte, len(tt.code))
			for i, c := range tt.code {
				code[i] = []byte(c)
			}
			assert.Equal(t, tt.want, referencedPackages(code...))
		})
	}
}
