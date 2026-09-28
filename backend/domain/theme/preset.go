package theme

// PresetDocuments returns the documents a new theme's first draft starts
// with. The default preset gives every template and section group an
// empty canvas; its designed content arrives with the default theme.
func PresetDocuments(presetKey string) ([]Document, error) {
	if !presets[presetKey] {
		return nil, ErrUnknownPreset
	}
	var docs []Document
	for _, key := range templateKeys {
		docs = append(docs, Document{Kind: KindTemplate, Key: key, Root: EmptyRoot()})
	}
	for _, key := range sectionGroupKeys {
		docs = append(docs, Document{Kind: KindSectionGroup, Key: key, Root: EmptyRoot()})
	}
	return docs, nil
}
