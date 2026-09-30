package theme

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/adirkuhn/commerceos/domain/locale"
)

// Presets are written in English, the source language. A new theme starts
// in its store's language: each preset text is looked up in
// presettext/<lang>.json (English text -> translation, the gettext way),
// so translators work from the English wording.

//go:embed presettext/*.json
var presetTextFiles embed.FS

var presetTranslations = func() map[locale.Locale]map[string]string {
	out := map[locale.Locale]map[string]string{}
	for _, l := range []locale.Locale{locale.Portuguese, locale.Dutch} {
		raw, err := presetTextFiles.ReadFile("presettext/" + string(l) + ".json")
		if err != nil {
			panic(fmt.Sprintf("theme: missing %s preset texts: %v", l, err))
		}
		m := map[string]string{}
		if err := json.Unmarshal(raw, &m); err != nil {
			panic(fmt.Sprintf("theme: bad %s preset texts: %v", l, err))
		}
		out[l] = m
	}
	return out
}()

// Props whose value is text a shopper reads.
var textProps = []string{"text", "badge", "title", "subtitle", "label", "button_label", "link_label", "placeholder", "empty_message", "home_label", "consent_label"}

// HasPresetTranslation reports whether an English preset text has a
// translation in the language.
func HasPresetTranslation(l locale.Locale, english string) bool {
	return presetTranslations[l][english] != ""
}

// PresetTexts lists every distinct text the documents contain, sorted.
func PresetTexts(docs []Document) []string {
	seen := map[string]bool{}
	for _, d := range docs {
		walkTexts(d.Root, func(s string) string { seen[s] = true; return s })
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func localizeDocuments(docs []Document, l locale.Locale) []Document {
	tr := presetTranslations[l]
	if tr == nil {
		return docs
	}
	for i := range docs {
		docs[i].Root = walkTexts(docs[i].Root, func(s string) string {
			if t := tr[s]; t != "" {
				return t
			}
			return s
		})
	}
	return docs
}

// walkTexts returns a copy of the tree with every text passed through fn:
// text props, menu item labels and rich-text runs. Empty texts (meaning
// "the store language's default") are left alone.
func walkTexts(n Node, fn func(string) string) Node {
	if len(n.Props) > 0 {
		props := make(map[string]any, len(n.Props))
		for k, v := range n.Props {
			props[k] = v
		}
		for _, k := range textProps {
			if s, ok := props[k].(string); ok && s != "" {
				props[k] = fn(s)
			}
		}
		if items, ok := props["items"].([]any); ok {
			copied := make([]any, len(items))
			for i, it := range items {
				m, ok := it.(map[string]any)
				if !ok {
					copied[i] = it
					continue
				}
				c := make(map[string]any, len(m))
				for k, v := range m {
					c[k] = v
				}
				if s, ok := c["label"].(string); ok && s != "" {
					c["label"] = fn(s)
				}
				copied[i] = c
			}
			props["items"] = copied
		}
		if content, ok := props["content"].([]any); ok {
			props["content"] = walkRichText(content, fn)
		}
		n.Props = props
	}
	if len(n.Children) > 0 {
		children := make([]Node, len(n.Children))
		for i, c := range n.Children {
			children[i] = walkTexts(c, fn)
		}
		n.Children = children
	}
	return n
}

func walkRichText(blocks []any, fn func(string) string) []any {
	out := make([]any, len(blocks))
	for i, b := range blocks {
		block, ok := b.(map[string]any)
		if !ok {
			out[i] = b
			continue
		}
		copied := make(map[string]any, len(block))
		for k, v := range block {
			copied[k] = v
		}
		if runs, ok := block["children"].([]any); ok {
			newRuns := make([]any, len(runs))
			for j, r := range runs {
				run, ok := r.(map[string]any)
				if !ok {
					newRuns[j] = r
					continue
				}
				c := make(map[string]any, len(run))
				for k, v := range run {
					c[k] = v
				}
				if s, ok := c["text"].(string); ok && s != "" {
					c["text"] = fn(s)
				}
				newRuns[j] = c
			}
			copied["children"] = newRuns
		}
		out[i] = copied
	}
	return out
}
