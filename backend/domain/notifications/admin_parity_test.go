package notifications_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/notifications"
)

// TestTypeTexts_matchTheAdminTranslations keeps the admin's bell texts
// (frontend notifications.json, rendered in each user's own language) and
// the email texts (this package, rendered in the company language) word for
// word the same. Skipped when the frontend isn't checked out next to the
// backend (e.g. a backend-only build context).
func TestTypeTexts_matchTheAdminTranslations(t *testing.T) {
	for _, lang := range languages {
		path := filepath.Join("..", "..", "..", "frontend", "src", "i18n", "locales", string(lang), "notifications.json")
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			t.Skip("frontend not checked out")
		}
		require.NoError(t, err)
		var file struct {
			Types map[string]any `json:"types"`
		}
		require.NoError(t, json.Unmarshal(raw, &file))

		for typ, byLang := range notifications.TypeTexts() {
			want := byLang[lang]
			got := lookup(file.Types, strings.Split(string(typ), "."))
			require.NotNil(t, got, "%s missing in %s", typ, path)
			assert.Equal(t, want.Title, got["title"], "%s title (%s)", typ, lang)
			assert.Equal(t, want.Message, got["message"], "%s message (%s)", typ, lang)
			if want.MessageOne != "" {
				assert.Equal(t, want.MessageOne, got["messageOne"], "%s messageOne (%s)", typ, lang)
			}
			for _, p := range want.Parts {
				parts, _ := got["parts"].(map[string]any)
				assert.Equal(t, p.Text, parts[p.Param], "%s part %s (%s)", typ, p.Param, lang)
			}
		}
	}
}

func lookup(node map[string]any, path []string) map[string]any {
	for _, seg := range path {
		next, ok := node[seg].(map[string]any)
		if !ok {
			return nil
		}
		node = next
	}
	return node
}
