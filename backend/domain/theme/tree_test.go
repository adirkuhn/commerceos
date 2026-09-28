package theme_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/theme"
)

func testCatalog(t *testing.T) *theme.Catalog {
	t.Helper()
	c, err := theme.NewCatalog([]theme.ComponentRule{
		{Type: "section", Container: true, AllowedParents: []string{theme.RootType}},
		{Type: "columns", Container: true, AllowedChildren: []string{"column"}},
		{Type: "column", Container: true, AllowedParents: []string{"columns"}},
		{Type: "heading"},
		{Type: "add-to-cart", DataContext: theme.ContextProduct, RequiredIn: []string{"product"}},
		{Type: "cart-icon", DataContext: theme.ContextCart},
	})
	require.NoError(t, err)
	return c
}

func node(id, typ string, children ...theme.Node) theme.Node {
	return theme.Node{ID: id, Type: typ, Children: children}
}

func doc(kind theme.DocumentKind, key string, children ...theme.Node) theme.Document {
	return theme.Document{Kind: kind, Key: key, Root: node("root", theme.RootType, children...)}
}

func TestValidateTree_acceptsAValidTree(t *testing.T) {
	d := doc(theme.KindTemplate, "product",
		node("s1", "section", node("h1", "heading"), node("atc", "add-to-cart"),
			node("c", "columns", node("c1", "column", node("h2", "heading")))))

	assert.NoError(t, theme.ValidateTree(d, testCatalog(t)))
}

func TestValidateTree_rejections(t *testing.T) {
	cases := map[string]theme.Document{
		"unknown type":            doc(theme.KindPage, "about", node("x", "marquee")),
		"root must be root":       {Kind: theme.KindPage, Key: "about", Root: node("r", "section")},
		"nested root":             doc(theme.KindPage, "about", node("s", "section", node("r2", theme.RootType))),
		"leaf with children":      doc(theme.KindPage, "about", node("s", "section", node("h", "heading", node("h2", "heading")))),
		"disallowed parent":       doc(theme.KindPage, "about", node("s", "section", node("s2", "section"))),
		"disallowed child":        doc(theme.KindPage, "about", node("s", "section", node("c", "columns", node("h", "heading")))),
		"orphan column":           doc(theme.KindPage, "about", node("s", "section", node("c1", "column"))),
		"duplicate id":            doc(theme.KindPage, "about", node("s", "section", node("s", "heading"))),
		"empty id":                doc(theme.KindPage, "about", node("", "section")),
		"bad id chars":            doc(theme.KindPage, "about", node("a b", "section")),
		"context not available":   doc(theme.KindTemplate, "home", node("s", "section", node("atc", "add-to-cart"))),
		"required component gone": doc(theme.KindTemplate, "product", node("s", "section")),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			assert.ErrorIs(t, theme.ValidateTree(d, testCatalog(t)), theme.ErrInvalidDocument)
		})
	}
}

func TestValidateTree_sessionContextsWorkAnywhere(t *testing.T) {
	d := doc(theme.KindSectionGroup, "header", node("s", "section", node("ci", "cart-icon")))
	assert.NoError(t, theme.ValidateTree(d, testCatalog(t)))
}

func TestValidateTree_errorNamesTheOffendingNode(t *testing.T) {
	d := doc(theme.KindPage, "about", node("s", "section", node("bad-one", "marquee")))

	err := theme.ValidateTree(d, testCatalog(t))

	var docErr *theme.DocumentError
	require.ErrorAs(t, err, &docErr)
	assert.Equal(t, "bad-one", docErr.NodeID)
}

func TestValidateTree_limitsDepthAndSize(t *testing.T) {
	c, err := theme.NewCatalog([]theme.ComponentRule{{Type: "box", Container: true}, {Type: "heading"}})
	require.NoError(t, err)

	deep := node("leaf", "heading")
	for i := 0; i < theme.MaxTreeDepth+1; i++ {
		deep = node(fmt.Sprintf("b%d", i), "box", deep)
	}
	assert.ErrorIs(t, theme.ValidateTree(doc(theme.KindPage, "about", deep), c), theme.ErrInvalidDocument)

	var many []theme.Node
	for i := 0; i < theme.MaxTreeNodes; i++ {
		many = append(many, node(fmt.Sprintf("h%d", i), "heading"))
	}
	assert.ErrorIs(t, theme.ValidateTree(doc(theme.KindPage, "about", node("b", "box", many...)), c), theme.ErrInvalidDocument)
}

func TestValidateTree_checksStyles(t *testing.T) {
	n := node("s", "section")
	n.Style = theme.NodeStyle{Base: map[string]string{"padding_y": "huge"}}

	assert.ErrorIs(t, theme.ValidateTree(doc(theme.KindPage, "about", n), testCatalog(t)), theme.ErrInvalidDocument)
}

func TestNewCatalog_rejectsBadRules(t *testing.T) {
	_, err := theme.NewCatalog([]theme.ComponentRule{{Type: "a"}, {Type: "a"}})
	assert.ErrorIs(t, err, theme.ErrInvalidCatalog)

	_, err = theme.NewCatalog([]theme.ComponentRule{{Type: theme.RootType}})
	assert.ErrorIs(t, err, theme.ErrInvalidCatalog)

	_, err = theme.NewCatalog([]theme.ComponentRule{{Type: "a", AllowedChildren: []string{"b"}}})
	assert.ErrorIs(t, err, theme.ErrInvalidCatalog, "leaf may not declare children")
}

func TestValidateTree_runsComponentPropsValidation(t *testing.T) {
	c, err := theme.NewCatalog([]theme.ComponentRule{{
		Type: "heading",
		ValidateProps: func(p map[string]any) error {
			if _, ok := p["text"].(string); !ok {
				return fmt.Errorf("text is required")
			}
			return nil
		},
	}})
	require.NoError(t, err)

	bad := doc(theme.KindPage, "about", theme.Node{ID: "h", Type: "heading", Props: map[string]any{"text": 3}})
	err = theme.ValidateTree(bad, c)
	var docErr *theme.DocumentError
	require.ErrorAs(t, err, &docErr)
	assert.Equal(t, "h", docErr.NodeID)
	assert.Contains(t, docErr.Reason, "text is required")

	good := doc(theme.KindPage, "about", theme.Node{ID: "h", Type: "heading", Props: map[string]any{"text": "Hi"}})
	assert.NoError(t, theme.ValidateTree(good, c))
}
