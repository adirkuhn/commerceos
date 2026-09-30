package pms_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/locale"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubSetProductTranslations struct {
	gotID string
	got   map[locale.Locale]pmsDomain.ProductTexts
	err   error
}

func (s *stubSetProductTranslations) Execute(_ context.Context, id string, tr map[locale.Locale]pmsDomain.ProductTexts) (*pmsDomain.Product, error) {
	s.gotID, s.got = id, tr
	if s.err != nil {
		return nil, s.err
	}
	return &pmsDomain.Product{ID: id, Name: "Pão de queijo", Translations: tr}, nil
}

func TestSetProductTranslations_passesTextsAndReturnsTheProduct(t *testing.T) {
	s := &stubs{}
	w := doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/products/p1/translations",
		map[string]any{"translations": map[string]any{"en": map[string]any{"name": "Cheese bread", "description": "Warm"}}})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "p1", s.setTranslations.gotID)
	assert.Equal(t, pmsDomain.ProductTexts{Name: "Cheese bread", Description: "Warm"}, s.setTranslations.got[locale.English])
	assert.Contains(t, w.Body.String(), `"translations":{"en":{"description":"Warm","name":"Cheese bread"}}`)
}

func TestSetProductTranslations_invalidIs422AndMissingIs404(t *testing.T) {
	s := &stubs{setTranslations: stubSetProductTranslations{err: pmsDomain.ErrInvalidProductTranslation}}
	w := doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/products/p1/translations", map[string]any{"translations": map[string]any{}})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	s = &stubs{setTranslations: stubSetProductTranslations{err: pmsDomain.ErrProductNotFound}}
	w = doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/products/p1/translations", map[string]any{"translations": map[string]any{}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}
