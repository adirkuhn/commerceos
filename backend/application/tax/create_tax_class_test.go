package tax_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptax "github.com/adirkuhn/commerceos/application/tax"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

func TestCreateTaxClass_success(t *testing.T) {
	repo := &stubTaxClassRepo{}
	uc := apptax.NewCreateTaxClassUseCase(newMockGormDB(t, 1), repo)

	class, err := uc.Execute(context.Background(), apptax.CreateTaxClassInput{Name: "Standard"})

	require.NoError(t, err)
	assert.Equal(t, "Standard", class.Name)
	assert.False(t, class.IsDefault)
	require.Len(t, repo.created, 1)
}

func TestCreateTaxClass_rejectsEmptyName(t *testing.T) {
	uc := apptax.NewCreateTaxClassUseCase(newMockGormDB(t, 0), &stubTaxClassRepo{})

	_, err := uc.Execute(context.Background(), apptax.CreateTaxClassInput{Name: "  "})

	assert.ErrorIs(t, err, taxDomain.ErrTaxClassNameRequired)
}

func TestCreateTaxClass_settingDefaultUnsetsThePreviousDefault(t *testing.T) {
	repo := &stubTaxClassRepo{list: []taxDomain.TaxClass{
		{ID: "class-old", Name: "Standard", IsDefault: true},
		{ID: "class-other", Name: "Other", IsDefault: false},
	}}
	uc := apptax.NewCreateTaxClassUseCase(newMockGormDB(t, 1), repo)

	class, err := uc.Execute(context.Background(), apptax.CreateTaxClassInput{Name: "Zero rate", IsDefault: true})

	require.NoError(t, err)
	assert.True(t, class.IsDefault)
	require.Len(t, repo.updated, 1)
	assert.Equal(t, "class-old", repo.updated[0].ID)
	assert.False(t, repo.updated[0].IsDefault)
}

func TestCreateTaxClass_propagatesRepoError(t *testing.T) {
	repo := &stubTaxClassRepo{createErr: errors.New("db down")}
	uc := apptax.NewCreateTaxClassUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), apptax.CreateTaxClassInput{Name: "Standard"})

	require.Error(t, err)
}
