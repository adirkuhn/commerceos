package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	schedulerapp "github.com/adirkuhn/commerceos/application/scheduler"
	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
)

// jobConstValues maps domain/scheduler's Job* constant names to their keys.
func jobConstValues(t *testing.T) map[string]string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "../../domain/scheduler/job.go", nil, 0)
	require.NoError(t, err)
	out := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, name := range spec.Names {
			if i < len(spec.Values) {
				if lit, ok := spec.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING && strings.HasPrefix(name.Name, "Job") {
					v, _ := strconv.Unquote(lit.Value)
					out[name.Name] = v
				}
			}
		}
		return true
	})
	return out
}

// registeredJobKeys reads which keys main.go puts in jobRunners -- the map
// literal and every jobRunners[schedulerDomain.X] = ... assignment.
func registeredJobKeys(t *testing.T) []string {
	t.Helper()
	consts := jobConstValues(t)
	f, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	require.NoError(t, err)
	var keys []string
	add := func(e ast.Expr) {
		if sel, ok := e.(*ast.SelectorExpr); ok {
			if v, ok := consts[sel.Sel.Name]; ok {
				keys = append(keys, v)
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				if id, ok := lhs.(*ast.Ident); ok && id.Name == "jobRunners" && i < len(n.Rhs) {
					if lit, ok := n.Rhs[i].(*ast.CompositeLit); ok {
						for _, el := range lit.Elts {
							if kv, ok := el.(*ast.KeyValueExpr); ok {
								add(kv.Key)
							}
						}
					}
				}
				if idx, ok := lhs.(*ast.IndexExpr); ok {
					if id, ok := idx.X.(*ast.Ident); ok && id.Name == "jobRunners" {
						add(idx.Index)
					}
				}
			}
		}
		return true
	})
	return keys
}

// The 2026-11-03 bug class: a job the admin can see but nothing can run.
func TestJobRegistry_everyKnownJobHasARunner(t *testing.T) {
	assert.ElementsMatch(t, schedulerDomain.JobKeys(), registeredJobKeys(t))
}

func TestJobRegistry_everyJobConstantIsListedInJobKeys(t *testing.T) {
	var consts []string
	for _, v := range jobConstValues(t) {
		consts = append(consts, v)
	}
	assert.ElementsMatch(t, consts, schedulerDomain.JobKeys())
}

func TestJobRegistry_everyKnownJobIsSeededByAMigration(t *testing.T) {
	files, err := filepath.Glob("../../infrastructure/db/migrations/*.sql")
	require.NoError(t, err)
	var seeds strings.Builder
	for _, f := range files {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		if strings.Contains(string(b), "INSERT INTO scheduled_jobs") {
			seeds.Write(b)
		}
	}
	for _, key := range schedulerDomain.JobKeys() {
		assert.Contains(t, seeds.String(), "'"+key+"'", "no migration seeds job %s", key)
	}
}

func TestMissingJobRunners(t *testing.T) {
	runners := map[string]schedulerapp.JobRunner{"a": nil, "b": nil}

	assert.Empty(t, missingJobRunners(runners, []string{"a", "b"}))
	assert.Equal(t, []string{"c"}, missingJobRunners(runners, []string{"a", "c"}))
}
