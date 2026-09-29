package themecatalog

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/adirkuhn/commerceos/domain/theme"
)

// ExportedComponent is one component as the storefront app sees it: the
// placement rules plus the prop names the contract accepts.
type ExportedComponent struct {
	Type            string   `json:"type"`
	Container       bool     `json:"container"`
	AllowedChildren []string `json:"allowed_children"`
	AllowedParents  []string `json:"allowed_parents"`
	DataContext     string   `json:"data_context"`
	RequiredIn      []string `json:"required_in"`
	Props           []string `json:"props"`
	RequiredProps   []string `json:"required_props"`
	// TranslatableProps may differ per language (node translations).
	TranslatableProps []string `json:"translatable_props"`
}

type Export struct {
	Components      []ExportedComponent `json:"components"`
	StyleProperties []string            `json:"style_properties"`
}

// ExportJSON renders the catalog for the storefront app's contract test
// (storefront/src/components/contract.gen.json). Regenerate with
// `go run ./cmd/export-theme-catalog` after changing api/theme.
func ExportJSON(spec *openapi3.T) ([]byte, error) {
	var out Export
	names := make([]string, 0, len(spec.Components.Schemas))
	for name := range spec.Components.Schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		schema := spec.Components.Schemas[name].Value
		raw, ok := schema.Extensions[extensionKey]
		if !ok {
			continue
		}
		ext, err := decode(raw)
		if err != nil {
			return nil, fmt.Errorf("themecatalog: %s: %w", name, err)
		}
		props := make([]string, 0, len(schema.Properties))
		for p := range schema.Properties {
			props = append(props, p)
		}
		sort.Strings(props)
		required := append([]string{}, schema.Required...)
		sort.Strings(required)
		out.Components = append(out.Components, ExportedComponent{
			Type: ext.Type, Container: ext.Container, AllowedChildren: nonNil(ext.AllowedChildren),
			AllowedParents: nonNil(ext.AllowedParents), DataContext: ext.DataContext, RequiredIn: nonNil(ext.RequiredIn),
			Props: props, RequiredProps: required, TranslatableProps: sortedCopy(ext.Translatable),
		})
	}
	out.StyleProperties = theme.StyleProperties()
	sort.Strings(out.StyleProperties)
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func sortedCopy(s []string) []string {
	out := append([]string{}, s...)
	sort.Strings(out)
	return out
}
