// Command export-theme-catalog writes the theme component catalog (from
// api/theme/openapi.yaml) to storefront/src/components/contract.gen.json,
// which the storefront app's contract test checks its registry against.
// Run from backend/: go run ./cmd/export-theme-catalog
package main

import (
	"log"
	"os"
	"path/filepath"

	gentheme "github.com/adirkuhn/commerceos/api/theme"
	"github.com/adirkuhn/commerceos/infrastructure/themecatalog"
)

func main() {
	spec, err := gentheme.GetSpec()
	if err != nil {
		log.Fatal(err)
	}
	b, err := themecatalog.ExportJSON(spec)
	if err != nil {
		log.Fatal(err)
	}
	out := filepath.Join("..", "storefront", "src", "components", "contract.gen.json")
	if err := os.WriteFile(out, b, 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s", out)
}
