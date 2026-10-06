package events_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/events"
)

func TestValidateSubject(t *testing.T) {
	for _, ok := range []string{
		"procurement.purchase_order.created", "pms.product.changed", "storefront.theme.rolled_back",
		"payments.refund.manual_review_required", "procurement.booking.item_scanned", "newsletter.subscriber.sent",
	} {
		assert.NoError(t, events.ValidateSubject(ok), ok)
	}
	for _, bad := range []string{
		"procurement.purchase_order",            // no verb
		"procurement.purchase_order.create",     // not past tense
		"Procurement.purchaseOrder.created",     // not snake_case
		"marketing.campaign.created",            // unknown domain
		"procurement.purchase_order.created.v2", // too many parts
		"procurement..created",
	} {
		assert.Error(t, events.ValidateSubject(bad), bad)
	}
}

// Every event in the system follows the naming standard (docs/events.md),
// and no two events share a subject.
func TestEverySubjectFollowsTheNamingStandard(t *testing.T) {
	seen := map[string]bool{}
	for _, subject := range AllSubjects(t) {
		assert.False(t, seen[subject], "duplicate subject %s", subject)
		seen[subject] = true
		if events.IsLegacySubject(subject) {
			continue
		}
		assert.NoError(t, events.ValidateSubject(subject))
	}
}

func TestActionTypesAreDescribed(t *testing.T) {
	require.NotEmpty(t, events.ActionTypes())
	for _, a := range events.ActionTypes() {
		assert.NotEmpty(t, a.Description, a.Subject)
	}
}

// AllSubjects is every typed event's EventName() literal plus every
// registered action type.
func AllSubjects(t *testing.T) []string {
	t.Helper()
	pkgs, err := parser.ParseDir(token.NewFileSet(), ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	require.NoError(t, err)
	var out []string
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || fn.Name.Name != "EventName" || fn.Body == nil {
					continue
				}
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						if v, err := strconv.Unquote(lit.Value); err == nil {
							out = append(out, v)
						}
					}
					return true
				})
			}
		}
	}
	for _, a := range events.ActionTypes() {
		out = append(out, a.Subject)
	}
	return out
}
