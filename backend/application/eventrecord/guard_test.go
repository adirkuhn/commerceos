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
		pkg := parsePackage(t, filepath.Join(root, d.Name()))
		for _, v := range pkg.violations() {
			violators = append(violators, d.Name()+"/"+v)
		}
	}
	slices.Sort(violators)

	for _, v := range violators {
		assert.Contains(t, pendingEvents, v, "%s writes without raising an event -- record one (eventrecord.Record) or add an //events: directive to its file", v)
	}
	for _, p := range pendingEvents {
		assert.True(t, slices.Contains(violators, p), "%s raises its event now -- remove it from pendingEvents", p)
	}
}

// writeMethod: repository calls that change state. notWrites are
// write-looking names of non-persistence calls (time, maps, URLs, slices).
var (
	writeMethod = regexp.MustCompile(`^(Create|Update|Delete|Save|Upsert|Insert|Replace|Remove|Add|Set|Mark|Clear|Attach|Link|Unlink|Increment|Decrement|Append|Archive|Restore|Assign|Complete|Cancel|Reset|Move|Bulk|Revoke|Grant|Claim|Toggle|Approve|Reject|Publish|Unpublish|Activate|Deactivate|Enable|Disable|Expire|Confirm|Adjust|Transfer|Reserve|Release|Rename|Merge|Record|Touch|Purge|Prune|Dismiss|Acknowledge)`)
	notWrites  = []string{"Add", "AddDate", "Set", "SetString", "Record", "Replace"}
)

// pkgFuncs is one application package: every function/method body by
// name (methods as Type.Name), for following calls into unexported helpers.
type pkgFuncs struct {
	funcs     map[string]*ast.FuncDecl // "Type.method" or "func"
	files     map[*ast.FuncDecl]string
	directive map[string]bool // file has an //events: directive
}

func funcKey(fn *ast.FuncDecl) string {
	if fn.Recv == nil {
		return fn.Name.Name
	}
	return recvType(fn) + "." + fn.Name.Name
}

func recvType(fn *ast.FuncDecl) string {
	t := fn.Recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func recvName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List[0].Names) == 0 {
		return ""
	}
	return fn.Recv.List[0].Names[0].Name
}

func parsePackage(t *testing.T, dir string) pkgFuncs {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	p := pkgFuncs{funcs: map[string]*ast.FuncDecl{}, files: map[*ast.FuncDecl]string{}, directive: map[string]bool{}}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, e.Name()))
		require.NoError(t, err)
		f, err := parser.ParseFile(token.NewFileSet(), e.Name(), src, 0)
		require.NoError(t, err)
		p.directive[e.Name()] = strings.Contains(string(src), "//events:")
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
				p.funcs[funcKey(fn)] = fn
				p.files[fn] = e.Name()
			}
		}
	}
	return p
}

// violations are the files with an exported use-case method that, through
// itself or the package's unexported helpers it calls, writes inside
// tenant.WithTenant without recording an event.
func (p pkgFuncs) violations() []string {
	seen := map[string]bool{}
	var out []string
	for _, fn := range p.funcs {
		file := p.files[fn]
		if !fn.Name.IsExported() || fn.Recv == nil || p.directive[file] || seen[file] {
			continue
		}
		r := p.reach(fn, map[*ast.FuncDecl]bool{})
		if r.tx && r.writes && !r.records {
			seen[file] = true
			out = append(out, file)
		}
	}
	return out
}

type reach struct{ tx, writes, records bool }

// uniqueMethod is the "Type.name" key of the package's only method called
// name, or "" when no type or several types have one.
func (p pkgFuncs) uniqueMethod(name string) string {
	found := ""
	for key, fn := range p.funcs {
		if fn.Recv != nil && fn.Name.Name == name {
			if found != "" {
				return ""
			}
			found = key
		}
	}
	return found
}

// reach follows fn's body and the unexported helpers it calls: a bare
// call to a package function, or a call on fn's own receiver to another of
// its type's methods (exported names are entry points, not helpers).
func (p pkgFuncs) reach(fn *ast.FuncDecl, visiting map[*ast.FuncDecl]bool) reach {
	var r reach
	if visiting[fn] {
		return r
	}
	visiting[fn] = true
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		var name, callee string
		switch f := call.Fun.(type) {
		case *ast.Ident:
			name, callee = f.Name, f.Name
		case *ast.SelectorExpr:
			name = f.Sel.Name
			if x, ok := f.X.(*ast.Ident); ok && fn.Recv != nil && x.Name == recvName(fn) {
				callee = recvType(fn) + "." + name
			} else if !ast.IsExported(name) {
				callee = p.uniqueMethod(name) // e.g. uc.deps.changed()
			}
			if pkg, ok := f.X.(*ast.Ident); ok {
				switch {
				case pkg.Name == "tenant" && name == "WithTenant":
					r.tx = true
				case pkg.Name == "eventrecord" && name == "Record":
					r.records = true
				}
			}
		}
		switch {
		case strings.Contains(strings.ToLower(name), "enqueue"):
			r.records = true
		case writeMethod.MatchString(name) && !slices.Contains(notWrites, name):
			if _, isSel := call.Fun.(*ast.SelectorExpr); isSel {
				r.writes = true
			}
		}
		if target, ok := p.funcs[callee]; ok && !ast.IsExported(name) {
			sub := p.reach(target, visiting)
			r.tx, r.writes, r.records = r.tx || sub.tx, r.writes || sub.writes, r.records || sub.records
		}
		return true
	})
	return r
}
