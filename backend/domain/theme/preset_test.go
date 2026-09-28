package theme_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/theme"
)

func TestPresetDocuments_defaultHasEveryTemplateAndSectionGroup(t *testing.T) {
	docs, err := theme.PresetDocuments(theme.PresetDefault)
	require.NoError(t, err)

	got := map[theme.DocumentKind][]string{}
	for _, d := range docs {
		got[d.Kind] = append(got[d.Kind], d.Key)
		assert.Equal(t, theme.RootType, d.Root.Type)
	}
	assert.ElementsMatch(t, theme.TemplateKeys(), got[theme.KindTemplate])
	assert.ElementsMatch(t, theme.SectionGroupKeys(), got[theme.KindSectionGroup])
	assert.Empty(t, got[theme.KindPage])
}

func TestPresetDocuments_unknownPreset(t *testing.T) {
	_, err := theme.PresetDocuments("bogus")
	assert.ErrorIs(t, err, theme.ErrUnknownPreset)
}
