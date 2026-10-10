// Package textrender fills text templates -- a small Mustache subset --
// shared by merchant-authored Message Center templates and CommerceOS's
// own translated notification texts.
package textrender

import (
	"fmt"
	"regexp"
	"strconv"
)

// Render fills a template -- a small Mustache subset -- from data:
//
//	{{name}}               a value; one with no matching key stays visible
//	                       as-is, so a typo shows in the sent message
//	{{#items}}..{{/items}} repeated once per element when items is a list
//	                       (each element's fields are in scope inside);
//	                       shown once when it is any other non-empty value
//	{{^name}}..{{/name}}   shown only when name is missing or empty
//
// Empty means missing, nil, false, 0, "" or an empty list. escape is applied
// to every value (never to the template's own text), e.g. html.EscapeString
// for email so product names cannot break the merchant's HTML.
//
// An element with data-each="name" (e.g. <tr data-each="items">) repeats
// like {{#name}}...{{/name}} around it -- the form visual HTML editors
// keep intact where loose {{#name}} text between table rows would be moved
// out of the table. See ExpandEach.
func Render(template string, data map[string]any, escape func(string) string) string {
	nodes := parseTemplate(ExpandEach(template))
	r := renderer{escape: escape}
	r.render(nodes, []map[string]any{data})
	return string(r.out)
}

var tagPattern = regexp.MustCompile(`\{\{\s*([#^/]?)\s*([A-Za-z0-9_.]+)\s*\}\}`)

type nodeKind int

const (
	textNode nodeKind = iota
	valueNode
	sectionNode
)

type templateNode struct {
	kind     nodeKind
	text     string // textNode: the text; valueNode/sectionNode: the raw opening tag
	name     string
	inverted bool
	children []templateNode
}

// parseTemplate builds the node tree. A section never closed, or a closing
// tag with no opening one, is kept as literal text.
func parseTemplate(template string) []templateNode {
	type frame struct {
		node  templateNode
		nodes []templateNode
	}
	stack := []frame{{}}
	appendNode := func(n templateNode) {
		top := &stack[len(stack)-1]
		top.nodes = append(top.nodes, n)
	}

	pos := 0
	for _, m := range tagPattern.FindAllStringSubmatchIndex(template, -1) {
		if m[0] > pos {
			appendNode(templateNode{kind: textNode, text: template[pos:m[0]]})
		}
		raw, sigil, name := template[m[0]:m[1]], template[m[2]:m[3]], template[m[4]:m[5]]
		pos = m[1]
		switch sigil {
		case "#", "^":
			stack = append(stack, frame{node: templateNode{kind: sectionNode, text: raw, name: name, inverted: sigil == "^"}})
		case "/":
			if len(stack) == 1 || stack[len(stack)-1].node.name != name {
				appendNode(templateNode{kind: textNode, text: raw})
				continue
			}
			done := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			done.node.children = done.nodes
			appendNode(done.node)
		default:
			appendNode(templateNode{kind: valueNode, text: raw, name: name})
		}
	}
	if pos < len(template) {
		appendNode(templateNode{kind: textNode, text: template[pos:]})
	}
	for len(stack) > 1 {
		open := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		appendNode(templateNode{kind: textNode, text: open.node.text})
		for _, n := range open.nodes {
			appendNode(n)
		}
	}
	return stack[0].nodes
}

type renderer struct {
	escape func(string) string
	out    []byte
}

func (r *renderer) render(nodes []templateNode, scopes []map[string]any) {
	for _, n := range nodes {
		switch n.kind {
		case textNode:
			r.out = append(r.out, n.text...)
		case valueNode:
			v, ok := lookup(scopes, n.name)
			if !ok {
				r.out = append(r.out, n.text...)
				continue
			}
			r.out = append(r.out, r.escape(formatValue(v))...)
		case sectionNode:
			v, _ := lookup(scopes, n.name)
			if n.inverted {
				if isEmpty(v) {
					r.render(n.children, scopes)
				}
				continue
			}
			if isEmpty(v) {
				continue
			}
			if list, ok := v.([]any); ok {
				for _, item := range list {
					r.render(n.children, withScope(scopes, item))
				}
				continue
			}
			r.render(n.children, withScope(scopes, v))
		}
	}
}

// withScope puts a map value's fields in scope; any other value keeps the
// current scopes.
func withScope(scopes []map[string]any, v any) []map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return scopes
	}
	out := make([]map[string]any, len(scopes), len(scopes)+1)
	copy(out, scopes)
	return append(out, m)
}

// lookup finds name in the innermost scope that has it.
func lookup(scopes []map[string]any, name string) (any, bool) {
	for i := len(scopes) - 1; i >= 0; i-- {
		if v, ok := scopes[i][name]; ok {
			return v, true
		}
	}
	return nil, false
}

func isEmpty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case bool:
		return !x
	case string:
		return x == ""
	case float64:
		return x == 0
	case int:
		return x == 0
	case int64:
		return x == 0
	case []any:
		return len(x) == 0
	default:
		return false
	}
}

func formatValue(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return fmt.Sprint(x)
	}
}

var eachAttr = regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9]*)\b[^>]*?\sdata-each=["']([A-Za-z0-9_.]+)["'][^>]*>`)

// ExpandEach rewrites every element carrying data-each="name" into
// {{#name}}<element without the attribute>{{/name}}. An element whose
// closing tag can't be found is left untouched.
func ExpandEach(html string) string {
	for {
		m := eachAttr.FindStringSubmatchIndex(html)
		if m == nil {
			return html
		}
		tag, name := html[m[2]:m[3]], html[m[4]:m[5]]
		end := matchingClose(html, m[1], tag)
		if end < 0 {
			return html
		}
		open := regexp.MustCompile(`\sdata-each=["'][^"']*["']`).ReplaceAllString(html[m[0]:m[1]], "")
		html = html[:m[0]] + "{{#" + name + "}}" + open + html[m[1]:end] + "{{/" + name + "}}" + html[end:]
	}
}

// matchingClose returns the index just after the </tag> closing the element
// whose start tag ends at from, counting nested same-name elements; -1 when
// there is none.
func matchingClose(html string, from int, tag string) int {
	tagRe := regexp.MustCompile(`(?i)<(/?)` + regexp.QuoteMeta(tag) + `\b[^>]*>`)
	depth := 1
	for _, m := range tagRe.FindAllStringSubmatchIndex(html[from:], -1) {
		if html[from+m[2]:from+m[3]] == "/" {
			depth--
			if depth == 0 {
				return from + m[1]
			}
		} else {
			depth++
		}
	}
	return -1
}
