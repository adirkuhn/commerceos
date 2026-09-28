package themecatalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gentheme "github.com/adirkuhn/commerceos/api/theme"
	"github.com/adirkuhn/commerceos/infrastructure/themecatalog"
)

// The storefront app checks its component registry against this file, so
// it must match the contract.
func TestExportedCatalogIsUpToDate(t *testing.T) {
	spec, err := gentheme.GetSpec()
	require.NoError(t, err)
	want, err := themecatalog.ExportJSON(spec)
	require.NoError(t, err)

	got, err := os.ReadFile(filepath.Join("..", "..", "..", "storefront", "src", "components", "contract.gen.json"))
	require.NoError(t, err, "run: go run ./cmd/export-theme-catalog")
	assert.Equal(t, string(want), strings.ReplaceAll(string(got), "\r\n", "\n"), "stale -- run: go run ./cmd/export-theme-catalog")
}
