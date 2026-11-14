// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"

	gowsdlhttp "github.com/kastrul/gowsdl/pkg/http"
	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/types"
)

func (g *Generator) unmarshal(ctx context.Context) error {
	data, err := g.fetchFile(ctx, g.loc)
	if err != nil {
		return &types.WSDLError{
			Op:   "fetch",
			Path: g.loc.String(),
			Err:  err,
		}
	}

	// Detect WSDL version
	version, err := parser.DetectWSDLVersion(data)
	if err != nil {
		return &types.WSDLError{
			Op:   "detect_version",
			Path: g.loc.String(),
			Err:  err,
		}
	}
	g.wsdlVersion = version

	// Parse based on version
	if version == "2.0" {
		g.wsdl2 = new(parser.WSDL2)
		err = xml.Unmarshal(data, g.wsdl2)
		if err != nil {
			return &types.WSDLError{
				Op:   "parse",
				Path: g.loc.String(),
				Err:  fmt.Errorf("failed to unmarshal WSDL 2.0: %w", err),
			}
		}
	} else {
		g.wsdl = new(parser.WSDL)
		err = xml.Unmarshal(data, g.wsdl)
		if err != nil {
			return &types.WSDLError{
				Op:   "parse",
				Path: g.loc.String(),
				Err:  fmt.Errorf("failed to unmarshal WSDL 1.1: %w", err),
			}
		}
	}

	g.resolveSchemaExternals(ctx)
	ResolveAttributeRefs(g.getSchemas())
	g.refNamespaces = resolveElementRefNamespaces(g.getSchemas(), g.getTargetNamespace())
	g.attrNamespaces = resolveAttributeRefNamespaces(g.getSchemas(), g.getTargetNamespace())

	return nil
}

// maxSchemaImportDepth bounds how deep nested xsd:import/xsd:include chains are followed.
const maxSchemaImportDepth = 10

// resolveSchemaExternals loads schemas referenced through xsd:import and
// xsd:include and appends them to the WSDL's schema list. Schemas that cannot
// be fetched or parsed are skipped with a warning.
func (g *Generator) resolveSchemaExternals(ctx context.Context) {
	resolved := make(map[string]bool)
	for _, schema := range g.getSchemas() {
		g.resolveXSDExternals(ctx, schema, g.loc, resolved, 0)
	}
}

func (g *Generator) resolveXSDExternals(ctx context.Context, schema *parser.XSDSchema, base *gowsdlhttp.Location, resolved map[string]bool, depth int) {
	if depth >= maxSchemaImportDepth {
		log.Printf("[WARN] schema import depth limit (%d) reached at %s", maxSchemaImportDepth, base)
		return
	}

	refs := make([]string, 0, len(schema.Imports)+len(schema.Includes))
	for _, imp := range schema.Imports {
		if imp.SchemaLocation == "" {
			continue
		}
		refs = append(refs, imp.SchemaLocation)
	}
	for _, incl := range schema.Includes {
		if incl.SchemaLocation == "" {
			continue
		}
		refs = append(refs, incl.SchemaLocation)
	}

	for _, ref := range refs {
		loc, err := base.Parse(ref)
		if err != nil {
			log.Printf("[WARN] invalid schemaLocation %q: %v", ref, err)
			continue
		}
		key := loc.String()
		if resolved[key] {
			continue
		}
		resolved[key] = true

		data, err := g.fetchFile(ctx, loc)
		if err != nil {
			log.Printf("[WARN] failed to fetch schema %s: %v", key, err)
			continue
		}

		external := new(parser.XSDSchema)
		if err := xml.Unmarshal(data, external); err != nil {
			log.Printf("[WARN] failed to parse schema %s: %v", key, err)
			continue
		}

		g.resolveXSDExternals(ctx, external, loc, resolved, depth+1)
		g.appendSchema(external)
	}
}

func (g *Generator) appendSchema(schema *parser.XSDSchema) {
	if g.wsdlVersion == "2.0" && g.wsdl2 != nil {
		g.wsdl2.Types.Schemas = append(g.wsdl2.Types.Schemas, schema)
	} else if g.wsdl != nil {
		g.wsdl.Types.Schemas = append(g.wsdl.Types.Schemas, schema)
	}
}

func (g *Generator) fetchFile(ctx context.Context, loc *gowsdlhttp.Location) ([]byte, error) {
	if loc.IsLocal() {
		return loc.ReadFile()
	}
	return gowsdlhttp.DownloadFile(ctx, loc.URL(), g.httpConfig)
}
