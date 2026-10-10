package messagecenter

import "github.com/adirkuhn/commerceos/domain/textrender"

// Render fills a merchant's message template -- see textrender.Render for
// the supported syntax ({{value}}, {{#list}}..{{/list}}, {{^value}}..).
func Render(template string, data map[string]any, escape func(string) string) string {
	return textrender.Render(template, data, escape)
}
