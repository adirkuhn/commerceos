package pms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	"github.com/adirkuhn/commerceos/domain/locale"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubTranslationsRepo struct {
	product *pmsDomain.Product
	got     map[locale.Locale]pmsDomain.ProductTexts
	err     error
}

func (s *stubTranslationsRepo) FindByID(_ context.Context, _ string) (*pmsDomain.Product, error) {
	if s.product == nil {
		return nil, pmsDomain.ErrProductNotFound
	}
	p := *s.product
	p.Translations = s.got
	return &p, nil
}

func (s *stubTranslationsRepo) SetTranslations(_ context.Context, _ string, tr map[locale.Locale]pmsDomain.ProductTexts) error {
	s.got = tr
	return s.err
}

func TestSetProductTranslations_savesCleanTranslationsAndRaisesEvent(t *testing.T) {
	repo := &stubTranslationsRepo{product: &pmsDomain.Product{ID: "p1", Name: "Pão de queijo"}}
	out := &stubOutboxRepo{}
	uc := apppms.NewSetProductTranslationsUseCase(newMockGormDB(t), repo, out)

	p, err := uc.Execute(context.Background(), "p1", map[locale.Locale]pmsDomain.ProductTexts{
		locale.English: {Name: " Cheese bread "}, locale.Dutch: {},
	})

	require.NoError(t, err)
	assert.Equal(t, map[locale.Locale]pmsDomain.ProductTexts{locale.English: {Name: "Cheese bread"}}, repo.got)
	assert.Equal(t, "Cheese bread", p.Translations[locale.English].Name)
	assert.Equal(t, []string{"pms.product.translated"}, out.calls)
}

func TestSetProductTranslations_rejectsInvalidBeforeTouchingTheDatabase(t *testing.T) {
	repo := &stubTranslationsRepo{product: &pmsDomain.Product{ID: "p1"}}
	uc := apppms.NewSetProductTranslationsUseCase(nil, repo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), "p1", map[locale.Locale]pmsDomain.ProductTexts{"de": {Name: "Käsebrot"}})

	assert.ErrorIs(t, err, pmsDomain.ErrInvalidProductTranslation)
	assert.Nil(t, repo.got)
}

func TestSetProductTranslations_unknownProduct(t *testing.T) {
	repo := &stubTranslationsRepo{err: pmsDomain.ErrProductNotFound}
	uc := apppms.NewSetProductTranslationsUseCase(newMockGormDB(t), repo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), "missing", nil)

	assert.True(t, errors.Is(err, pmsDomain.ErrProductNotFound))
}
