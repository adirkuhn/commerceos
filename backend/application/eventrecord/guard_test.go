package eventrecord_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every state-changing use case raises a domain event (docs/events.md).
// A file under application/ that writes inside tenant.WithTenant must
// record one -- eventrecord.Record, or an outbox Enqueue for a typed
// event -- or say why not with a directive comment:
//
//	//events:delegated <use case that records it>
//	//events:none <why this write is not a business action>
//
// pendingEvents are the files written before this rule; it only shrinks.
func TestEveryWriteRaisesAnEvent(t *testing.T) {
	root := filepath.Join("..")
	var violators []string
	dirs, err := os.ReadDir(root)
	require.NoError(t, err)
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		files := parsePackage(t, filepath.Join(root, d.Name()))
		recorders := recordingFuncs(files)
		for name, f := range files {
			if needsEvent(f, recorders) {
				violators = append(violators, d.Name()+"/"+name)
			}
		}
	}

	for _, v := range violators {
		assert.Contains(t, pendingEvents, v, "%s writes without raising an event -- record one (eventrecord.Record) or add an //events: directive", v)
	}
	for _, p := range pendingEvents {
		assert.True(t, slices.Contains(violators, p), "%s raises its event now -- remove it from pendingEvents", p)
	}
}

type parsedFile struct {
	ast       *ast.File
	directive bool
}

func parsePackage(t *testing.T, dir string) map[string]parsedFile {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	out := map[string]parsedFile{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, e.Name()))
		require.NoError(t, err)
		f, err := parser.ParseFile(token.NewFileSet(), e.Name(), src, 0)
		require.NoError(t, err)
		out[e.Name()] = parsedFile{ast: f, directive: strings.Contains(string(src), "//events:")}
	}
	return out
}

// recordingFuncs are the package's unexported helpers that record an event
// themselves -- calling one counts as recording (e.g. collections'
// Deps.changed). Exported names (a use case's Execute) don't: a call can't
// be told apart from any other type's Execute, so delegating to another
// use case is declared with //events:delegated instead.
func recordingFuncs(files map[string]parsedFile) map[string]bool {
	out := map[string]bool{}
	for _, f := range files {
		for _, decl := range f.ast.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil && !fn.Name.IsExported() && callsRecord(fn.Body, nil) {
				out[fn.Name.Name] = true
			}
		}
	}
	return out
}

func callsRecord(n ast.Node, recorders map[string]bool) bool {
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok && isRecordCall(call, recorders) {
			found = true
		}
		return !found
	})
	return found
}

func isRecordCall(call *ast.CallExpr, recorders map[string]bool) bool {
	var name string
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		name = fn.Name
	case *ast.SelectorExpr:
		name = fn.Sel.Name
		if pkg, ok := fn.X.(*ast.Ident); ok && pkg.Name == "eventrecord" && name == "Record" {
			return true
		}
	}
	return strings.Contains(strings.ToLower(name), "enqueue") || recorders[name]
}

var writeMethod = regexp.MustCompile(`^(Create|Update|Delete|Save|Upsert|Insert|Replace|Remove|Add|Set|Mark|Clear|Attach|Link|Unlink|Increment|Decrement|Append|Archive|Restore|Assign|Complete|Cancel|Reset|Move|Bulk)`)

// notWrites are write-looking names of non-persistence calls (time, maps, URLs).
var notWrites = []string{"Add", "AddDate", "Set", "SetString"}

func needsEvent(f parsedFile, recorders map[string]bool) bool {
	if f.directive {
		return false
	}
	var inTx, writes bool
	ast.Inspect(f.ast, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "tenant" && sel.Sel.Name == "WithTenant" {
			inTx = true
		} else if writeMethod.MatchString(sel.Sel.Name) && !slices.Contains(notWrites, sel.Sel.Name) {
			writes = true
		}
		return true
	})
	return inTx && writes && !callsRecord(f.ast, recorders)
}
