// Package themecatalog builds the domain's component catalog from the theme
// OpenAPI contract: every schema carrying x-commerceos-component is one
// component type -- the extension gives its placement rules, the schema
// validates its props.
package themecatalog

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/adirkuhn/commerceos/domain/theme"
)

const extensionKey = "x-commerceos-component"

type extension struct {
	Type            string   `json:"type"`
	Container       bool     `json:"container"`
	AllowedChildren []string `json:"allowed_children"`
	AllowedParents  []string `json:"allowed_parents"`
	DataContext     string   `json:"data_context"`
	RequiredIn      []string `json:"required_in"`
	// Translatable: props (merchant texts) a node may translate per language.
	Translatable []string `json:"translatable"`
}

func FromSpec(spec *openapi3.T) (*theme.Catalog, error) {
	names := make([]string, 0, len(spec.Components.Schemas))
	for name := range spec.Components.Schemas {
		names = append(names, name)
	}
	sort.Strings(names)

	var rules []theme.ComponentRule
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
		for _, p := range ext.Translatable {
			if _, ok := schema.Properties[p]; !ok {
				return nil, fmt.Errorf("themecatalog: %s: translatable prop %q is not a property", name, p)
			}
		}
		rules = append(rules, theme.ComponentRule{
			Type: ext.Type, Container: ext.Container, AllowedChildren: ext.AllowedChildren,
			AllowedParents: ext.AllowedParents, DataContext: theme.DataContext(ext.DataContext),
			RequiredIn: ext.RequiredIn, Translatable: ext.Translatable, ValidateProps: propsValidator(schema),
		})
	}
	return theme.NewCatalog(rules)
}

func decode(raw any) (extension, error) {
	var ext extension
	b, err := json.Marshal(raw)
	if err != nil {
		return ext, err
	}
	if err := json.Unmarshal(b, &ext); err != nil {
		return ext, err
	}
	if ext.Type == "" {
		return ext, fmt.Errorf("%s has no type", extensionKey)
	}
	return ext, nil
}

func propsValidator(schema *openapi3.Schema) func(map[string]any) error {
	return func(props map[string]any) error {
		// Round-trip through JSON so the validator sees plain JSON values
		// (e.g. []any rather than []map[string]any).
		var value any = map[string]any{}
		if props != nil {
			b, err := json.Marshal(props)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(b, &value); err != nil {
				return err
			}
		}
		if err := schema.VisitJSON(value, openapi3.MultiErrors()); err != nil {
			return fmt.Errorf("invalid props: %w", err)
		}
		return nil
	}
}
