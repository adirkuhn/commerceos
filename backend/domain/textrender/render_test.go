package textrender_test

import (
	"html"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	mc "github.com/adirkuhn/commerceos/domain/textrender"
)

func plain(s string) string { return s }

func TestRender_substitutesVariables(t *testing.T) {
	out := mc.Render("Hi {{customer_name}}, order {{sales_order_id}}", map[string]any{"customer_name": "Jane", "sales_order_id": "o-1"}, plain)
	assert.Equal(t, "Hi Jane, order o-1", out)
}

func TestRender_leavesUnknownVariableVisible(t *testing.T) {
	out := mc.Render("Hi {{customer_name}}, {{typo}}", map[string]any{"customer_name": "Jane"}, plain)
	assert.Equal(t, "Hi Jane, {{typo}}", out)
}

func TestRender_formatsNumbersAndNil(t *testing.T) {
	out := mc.Render("{{a}}|{{b}}|{{c}}", map[string]any{"a": float64(3), "b": 2.5, "c": nil}, plain)
	assert.Equal(t, "3|2.5|", out)
}

func TestRender_loopsOverAList(t *testing.T) {
	data := map[string]any{
		"window_days": 30,
		"items": []any{
			map[string]any{"name": "Guarana", "sku": "G-1"},
			map[string]any{"name": "Farofa", "sku": "F-2"},
		},
	}
	out := mc.Render("{{window_days}} days:{{#items}} [{{name}} {{sku}} ({{window_days}})]{{/items}}", data, plain)
	assert.Equal(t, "30 days: [Guarana G-1 (30)] [Farofa F-2 (30)]", out)
}

func TestRender_emptyListRendersNothing(t *testing.T) {
	out := mc.Render("a{{#items}}x{{/items}}b", map[string]any{"items": []any{}}, plain)
	assert.Equal(t, "ab", out)
}

func TestRender_sectionOnValueShowsWhenPresent(t *testing.T) {
	tmpl := "{{#sale_price}}<s>{{price}}</s> {{sale_price}}{{/sale_price}}{{^sale_price}}{{price}}{{/sale_price}}"

	assert.Equal(t, "<s>€ 5,00</s> € 4,00", mc.Render(tmpl, map[string]any{"price": "€ 5,00", "sale_price": "€ 4,00"}, plain))
	assert.Equal(t, "€ 5,00", mc.Render(tmpl, map[string]any{"price": "€ 5,00", "sale_price": ""}, plain))
	assert.Equal(t, "€ 5,00", mc.Render(tmpl, map[string]any{"price": "€ 5,00"}, plain), "missing counts as empty")
}

func TestRender_falseAndZeroAreEmpty(t *testing.T) {
	tmpl := "{{#flag}}yes{{/flag}}{{^flag}}no{{/flag}}"
	assert.Equal(t, "no", mc.Render(tmpl, map[string]any{"flag": false}, plain))
	assert.Equal(t, "no", mc.Render(tmpl, map[string]any{"flag": float64(0)}, plain))
	assert.Equal(t, "yes", mc.Render(tmpl, map[string]any{"flag": true}, plain))
}

func TestRender_nestedSections(t *testing.T) {
	data := map[string]any{"items": []any{
		map[string]any{"name": "A", "sale": "1"},
		map[string]any{"name": "B", "sale": ""},
	}}
	out := mc.Render("{{#items}}{{name}}{{#sale}}*{{/sale}};{{/items}}", data, plain)
	assert.Equal(t, "A*;B;", out)
}

func TestRender_escapesValuesNotTemplate(t *testing.T) {
	data := map[string]any{"items": []any{map[string]any{"name": `Pão & "queijo" <b>`}}}
	out := mc.Render("<td>{{#items}}{{name}}{{/items}}</td>", data, html.EscapeString)
	assert.Equal(t, "<td>Pão &amp; &#34;queijo&#34; &lt;b&gt;</td>", out)
}

func TestRender_unclosedSectionIsLeftVisible(t *testing.T) {
	out := mc.Render("a {{#items}}x", map[string]any{"items": []any{map[string]any{}}}, plain)
	assert.True(t, strings.Contains(out, "{{#items}}"), out)
}

func TestRender_dataEachRepeatsTheElement(t *testing.T) {
	data := map[string]any{"items": []any{map[string]any{"name": "A"}, map[string]any{"name": "B"}}}
	out := mc.Render(`<table><tr><th>N</th></tr><tr data-each="items" style="x"><td>{{name}}</td></tr></table>`, data, plain)
	assert.Equal(t, `<table><tr><th>N</th></tr><tr style="x"><td>A</td></tr><tr style="x"><td>B</td></tr></table>`, out)
}

func TestRender_dataEachHandlesNestedSameTags(t *testing.T) {
	data := map[string]any{"rows": []any{map[string]any{"v": "1"}}}
	out := mc.Render(`<div data-each="rows"><div>{{v}}</div></div><div>after</div>`, data, plain)
	assert.Equal(t, `<div><div>1</div></div><div>after</div>`, out)
}

func TestRender_dataEachOnEmptyListRendersNothing(t *testing.T) {
	out := mc.Render(`<ul><li data-each='items'>{{name}}</li></ul>`, map[string]any{"items": []any{}}, plain)
	assert.Equal(t, `<ul></ul>`, out)
}

func TestExpandEach(t *testing.T) {
	assert.Equal(t, `{{#items}}<tr><td>{{name}}</td></tr>{{/items}}`, mc.ExpandEach(`<tr data-each="items"><td>{{name}}</td></tr>`))
	assert.Equal(t, `no markers`, mc.ExpandEach(`no markers`))
	assert.Equal(t, `<tr data-each="x"><td>`, mc.ExpandEach(`<tr data-each="x"><td>`), "unclosed element left as is")
}
