// Package eventcatalog builds docs/events.md -- every domain event in the
// system, from the code itself: typed events (a struct with an
// EventName() method in domain/events, its doc comment and JSON fields)
// and standard action events (events.ActionTypes()). Regenerate with
// `go run ./cmd/eventcatalog` after adding an event; a test fails while
// the doc is out of date.
package eventcatalog

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/adirkuhn/commerceos/domain/events"
)

// Entry is one event in the catalog.
type Entry struct {
	Subject     string
	Kind        string // "action" or "typed"
	GoType      string
	Description string
	Fields      []string
}

// Collect lists every event: typed ones parsed from eventsDir, plus every
// registered action type. Sorted by subject.
func Collect(eventsDir string) ([]Entry, error) {
	typed, err := typedEvents(eventsDir)
	if err != nil {
		return nil, err
	}
	out := typed
	for _, a := range events.ActionTypes() {
		out = append(out, Entry{Subject: a.Subject, Kind: "action", GoType: "EntityAction", Description: a.Description})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Subject < out[j].Subject })
	return out, nil
}

func typedEvents(dir string) ([]Entry, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", dir, err)
	}
	var out []Entry
	for _, pkg := range pkgs {
		structs := map[string]*ast.GenDecl{}
		specs := map[string]*ast.TypeSpec{}
		var names []struct{ typ, subject string }
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.GenDecl:
					for _, s := range d.Specs {
						if ts, ok := s.(*ast.TypeSpec); ok {
							structs[ts.Name.Name], specs[ts.Name.Name] = d, ts
						}
					}
				case *ast.FuncDecl:
					if d.Recv == nil || d.Name.Name != "EventName" || d.Body == nil {
						continue
					}
					if subject, ok := returnedLiteral(d.Body); ok {
						names = append(names, struct{ typ, subject string }{receiverType(d), subject})
					}
				}
			}
		}
		for _, n := range names {
			ts := specs[n.typ]
			if ts == nil {
				continue
			}
			comment := ts.Doc
			if comment == nil {
				comment = structs[n.typ].Doc
			}
			out = append(out, Entry{
				Subject: n.subject, Kind: "typed", GoType: n.typ,
				Description: firstSentence(comment.Text()), Fields: jsonFields(ts),
			})
		}
	}
	return out, nil
}

func returnedLiteral(body *ast.BlockStmt) (string, bool) {
	var subject string
	ast.Inspect(body, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING && subject == "" {
			subject, _ = strconv.Unquote(lit.Value)
		}
		return subject == ""
	})
	return subject, subject != ""
}

func receiverType(fn *ast.FuncDecl) string {
	t := fn.Recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// jsonFields lists a struct's JSON field names; the embedded Base adds
// event_id and occurred_at.
func jsonFields(ts *ast.TypeSpec) []string {
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil
	}
	var out []string
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			if id, ok := f.Type.(*ast.Ident); ok && id.Name == "Base" {
				out = append(out, "event_id", "occurred_at")
			}
			continue
		}
		if f.Tag == nil {
			continue
		}
		tag, _ := strconv.Unquote(f.Tag.Value)
		name := strings.Split(reflect.StructTag(tag).Get("json"), ",")[0]
		if name != "" && name != "-" {
			out = append(out, name)
		}
	}
	return out
}

// firstSentence is a doc comment's first sentence -- not cut at an
// abbreviation ("e.g.") or inside parentheses.
func firstSentence(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	depth := 0
	for i, r := range text {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case '.':
			end := i+1 == len(text) || text[i+1] == ' '
			if depth == 0 && end && !endsWithAbbreviation(text[:i+1]) {
				return text[:i+1]
			}
		}
	}
	return text
}

func endsWithAbbreviation(s string) bool {
	for _, a := range []string{"e.g.", "i.e.", "etc.", "vs."} {
		if strings.HasSuffix(s, a) {
			return true
		}
	}
	return false
}
