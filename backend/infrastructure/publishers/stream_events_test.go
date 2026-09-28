package publishers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestSubjectsCoverEveryDomainEvent discovers every domain/events type's
// EventName() subject by parsing the package source, so a new event under a
// new prefix fails here instead of silently stalling the relay. An unrouted
// subject makes js.Publish return nats.ErrNoResponders, which cmd/relay
// treats as a connectivity error: the row stays pending forever, and once 50
// such rows accumulate FetchPending(50) returns only them — no event of any
// kind is published again.
func TestSubjectsCoverEveryDomainEvent(t *testing.T) {
	names := domainEventNames(t)
	if len(names) == 0 {
		t.Fatal("found no EventName() declarations in domain/events — parser drifted")
	}
	for _, name := range names {
		if !subjectCovered(name, subjects) {
			t.Errorf("event subject %q has no matching stream subject in %v", name, subjects)
		}
	}
}

func subjectCovered(name string, streamSubjects []string) bool {
	for _, s := range streamSubjects {
		if strings.HasPrefix(name, strings.TrimSuffix(s, ">")) {
			return true
		}
	}
	return false
}

// domainEventNames returns the string literal returned by every
// `func (X) EventName() string` in backend/domain/events.
func domainEventNames(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join("..", "..", "domain", "events")
	pkgs, err := parser.ParseDir(token.NewFileSet(), dir, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", dir, err)
	}
	var names []string
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || fn.Name.Name != "EventName" || fn.Body == nil {
					continue
				}
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					ret, ok := n.(*ast.ReturnStmt)
					if !ok || len(ret.Results) != 1 {
						return true
					}
					if lit, ok := ret.Results[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						if v, err := strconv.Unquote(lit.Value); err == nil {
							names = append(names, v)
						}
					}
					return false
				})
			}
		}
	}
	return names
}
