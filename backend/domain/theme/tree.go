package theme

import (
	"fmt"
	"regexp"
)

// RootType is the implicit container at the top of every document.
const RootType = "root"

const (
	MaxTreeDepth = 16
	MaxTreeNodes = 1500
)

var nodeIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Node is one placed component. Props are component-specific and
// validated against the OpenAPI contract; the domain only owns structure
// and styles.
type Node struct {
	ID       string
	Type     string
	Props    map[string]any
	Style    NodeStyle
	Children []Node
}

func EmptyRoot() Node { return Node{ID: "root", Type: RootType} }

// ComponentRule is what the domain needs to know about a component type
// to keep a tree valid. The full component definitions (props, editor
// behaviour, renderers) live in the storefront app's registry; the backend
// builds its catalog from the same OpenAPI contract.
type ComponentRule struct {
	Type      string
	Container bool
	// AllowedChildren restricts a container's children; empty means any.
	AllowedChildren []string
	// AllowedParents restricts where the component can be placed; empty
	// means anywhere. RootType means "top level of a document".
	AllowedParents []string
	DataContext    DataContext
	// RequiredIn lists template keys that must contain this component
	// (e.g. checkout needs the checkout form).
	RequiredIn []string
	// ValidateProps checks the component's props; nil accepts any. The
	// backend plugs in the OpenAPI contract's per-component props schema.
	ValidateProps func(props map[string]any) error
}

type Catalog struct {
	rules map[string]ComponentRule
}

func NewCatalog(rules []ComponentRule) (*Catalog, error) {
	c := &Catalog{rules: make(map[string]ComponentRule, len(rules))}
	for _, r := range rules {
		if r.Type == "" || r.Type == RootType {
			return nil, fmt.Errorf("%w: reserved or empty type %q", ErrInvalidCatalog, r.Type)
		}
		if _, dup := c.rules[r.Type]; dup {
			return nil, fmt.Errorf("%w: duplicate type %q", ErrInvalidCatalog, r.Type)
		}
		if !r.Container && len(r.AllowedChildren) > 0 {
			return nil, fmt.Errorf("%w: leaf %q declares children", ErrInvalidCatalog, r.Type)
		}
		c.rules[r.Type] = r
	}
	return c, nil
}

func (c *Catalog) Rule(componentType string) (ComponentRule, bool) {
	r, ok := c.rules[componentType]
	return r, ok
}

// ValidateTree checks a document's node tree against the catalog: known
// types, nesting rules, data context, required components, unique ids,
// styles and size limits.
func ValidateTree(d Document, c *Catalog) error {
	if d.Root.Type != RootType {
		return &DocumentError{NodeID: d.Root.ID, Reason: "document root must be of type root"}
	}
	w := walker{doc: d, catalog: c, seen: map[string]bool{}, types: map[string]bool{}}
	if err := w.walk(d.Root, nil, 0); err != nil {
		return err
	}
	if d.Kind == KindTemplate {
		for _, r := range c.rules {
			for _, key := range r.RequiredIn {
				if key == d.Key && !w.types[r.Type] {
					return &DocumentError{Reason: fmt.Sprintf("template %q must contain a %q component", d.Key, r.Type)}
				}
			}
		}
	}
	return nil
}

type walker struct {
	doc     Document
	catalog *Catalog
	seen    map[string]bool
	types   map[string]bool
	count   int
}

func (w *walker) walk(n Node, parent *ComponentRule, depth int) error {
	w.count++
	if w.count > MaxTreeNodes {
		return &DocumentError{Reason: fmt.Sprintf("more than %d components", MaxTreeNodes)}
	}
	if depth > MaxTreeDepth {
		return &DocumentError{NodeID: n.ID, Reason: fmt.Sprintf("nested deeper than %d levels", MaxTreeDepth)}
	}
	if !nodeIDPattern.MatchString(n.ID) {
		return &DocumentError{NodeID: n.ID, Reason: "invalid node id"}
	}
	if w.seen[n.ID] {
		return &DocumentError{NodeID: n.ID, Reason: "duplicate node id"}
	}
	w.seen[n.ID] = true
	if err := ValidateStyle(n.Style); err != nil {
		return &DocumentError{NodeID: n.ID, Reason: err.Error()}
	}

	rule, err := w.ruleFor(n, parent)
	if err != nil {
		return err
	}
	w.types[n.Type] = true
	if rule.ValidateProps != nil {
		if err := rule.ValidateProps(n.Props); err != nil {
			return &DocumentError{NodeID: n.ID, Reason: err.Error()}
		}
	}
	if !rule.Container && len(n.Children) > 0 {
		return &DocumentError{NodeID: n.ID, Reason: fmt.Sprintf("%q cannot contain components", n.Type)}
	}
	for _, child := range n.Children {
		if len(rule.AllowedChildren) > 0 && !contains(rule.AllowedChildren, child.Type) {
			return &DocumentError{NodeID: child.ID, Reason: fmt.Sprintf("%q is not allowed inside %q", child.Type, n.Type)}
		}
		if err := w.walk(child, &rule, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (w *walker) ruleFor(n Node, parent *ComponentRule) (ComponentRule, error) {
	if parent == nil {
		return ComponentRule{Type: RootType, Container: true}, nil
	}
	if n.Type == RootType {
		return ComponentRule{}, &DocumentError{NodeID: n.ID, Reason: "root can only be the top of a document"}
	}
	rule, ok := w.catalog.Rule(n.Type)
	if !ok {
		return ComponentRule{}, &DocumentError{NodeID: n.ID, Reason: fmt.Sprintf("unknown component %q", n.Type)}
	}
	if len(rule.AllowedParents) > 0 && !contains(rule.AllowedParents, parent.Type) {
		return ComponentRule{}, &DocumentError{NodeID: n.ID, Reason: fmt.Sprintf("%q cannot be placed inside %q", n.Type, parent.Type)}
	}
	if !ContextAvailable(w.doc.Kind, w.doc.Key, rule.DataContext) {
		return ComponentRule{}, &DocumentError{NodeID: n.ID, Reason: fmt.Sprintf("%q needs %s data, which this page doesn't have", n.Type, rule.DataContext)}
	}
	return rule, nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
