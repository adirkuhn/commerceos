package theme_test

import (
	"regexp"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genstorefront "github.com/adirkuhn/commerceos/api/storefront"
	gentheme "github.com/adirkuhn/commerceos/api/theme"
	themehandler "github.com/adirkuhn/commerceos/infrastructure/handlers/theme"
)

// Routes are mounted by hand (admin vs public groups need different auth),
// so this guards against an operation in the contract that nobody serves,
// and against a path that clashes with the storefront API on the shared
// public /api/v1 router (gin panics on conflicting routes).
func TestEveryContractOperationIsMounted(t *testing.T) {
	spec, err := gentheme.GetSpec()
	require.NoError(t, err)
	param := regexp.MustCompile(`\{([^}]+)\}`)
	var want []string
	for path, item := range spec.Paths.Map() {
		for method := range item.Operations() {
			want = append(want, method+" /api/v1"+param.ReplaceAllString(path, ":$1"))
		}
	}

	r := gin.New()
	var storefront genstorefront.ServerInterface
	require.NotPanics(t, func() {
		genstorefront.RegisterHandlers(r.Group("/api/v1"), storefront)
		h := themehandler.NewHandler(themehandler.UseCases{})
		themehandler.RegisterAdmin(r.Group("/api/v1"), h)
		themehandler.RegisterPublic(r.Group("/api/v1"), h)
	})
	mounted := map[string]bool{}
	for _, route := range r.Routes() {
		mounted[route.Method+" "+route.Path] = true
	}
	sort.Strings(want)
	for _, w := range want {
		assert.True(t, mounted[w], "not mounted: %s", w)
	}
}
