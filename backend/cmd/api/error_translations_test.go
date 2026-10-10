package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/errs"
)

// TestEveryErrorCodeIsTranslated: every coded domain error the API can
// return (domain/errs registry -- this package imports every module) has a
// text in the admin's errors.json, in every language. Skipped when the
// frontend isn't checked out next to the backend.
func TestEveryErrorCodeIsTranslated(t *testing.T) {
	for _, lang := range []string{"en", "pt-BR", "nl"} {
		path := filepath.Join("..", "..", "..", "frontend", "src", "i18n", "locales", lang, "errors.json")
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			t.Skip("frontend not checked out")
		}
		require.NoError(t, err)
		var texts map[string]string
		require.NoError(t, json.Unmarshal(raw, &texts))
		for _, code := range errs.Codes() {
			assert.NotEmpty(t, texts[code], "%s has no %s text in %s", code, lang, path)
		}
	}
}
