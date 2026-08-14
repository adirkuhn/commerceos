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

func TestUpdateTaxClass_success(t *testing.T) {
	repo := &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1", Name: "Standard"}}
	uc := apptax.NewUpdateTaxClassUseCase(newMockGormDB(t, 1), repo)

	class, err := uc.Execute(context.Background(), apptax.UpdateTaxClassInput{ID: "class-1", Name: "Standard rate"})

	require.NoError(t, err)
	assert.Equal(t, "Standard rate", class.Name)
	require.Len(t, repo.updated, 1)
}

func TestUpdateTaxClass_rejectsEmptyName(t *testing.T) {
	repo := &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1", Name: "Standard"}}
	uc := apptax.NewUpdateTaxClassUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), apptax.UpdateTaxClassInput{ID: "class-1", Name: ""})

	assert.ErrorIs(t, err, taxDomain.ErrTaxClassNameRequired)
}

func TestUpdateTaxClass_propagatesNotFound(t *testing.T) {
	repo := &stubTaxClassRepo{findErr: taxDomain.ErrTaxClassNotFound}
	uc := apptax.NewUpdateTaxClassUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), apptax.UpdateTaxClassInput{ID: "missing", Name: "X"})

	assert.ErrorIs(t, err, taxDomain.ErrTaxClassNotFound)
}

func TestUpdateTaxClass_settingDefaultUnsetsThePreviousDefault(t *testing.T) {
	repo := &stubTaxClassRepo{
		findResult: &taxDomain.TaxClass{ID: "class-2", Name: "Reduced rate"},
		list: []taxDomain.TaxClass{
			{ID: "class-1", Name: "Standard", IsDefault: true},
			{ID: "class-2", Name: "Reduced rate", IsDefault: false},
		},
	}
	uc := apptax.NewUpdateTaxClassUseCase(newMockGormDB(t, 1), repo)

	class, err := uc.Execute(context.Background(), apptax.UpdateTaxClassInput{ID: "class-2", Name: "Reduced rate", IsDefault: true})

	require.NoError(t, err)
	assert.True(t, class.IsDefault)
	require.Len(t, repo.updated, 2)
	assert.Equal(t, "class-1", repo.updated[0].ID)
	assert.False(t, repo.updated[0].IsDefault)
	assert.Equal(t, "class-2", repo.updated[1].ID)
	assert.True(t, repo.updated[1].IsDefault)
}

func TestUpdateTaxClass_reSavingTheCurrentDefaultDoesNotDoubleUnset(t *testing.T) {
	repo := &stubTaxClassRepo{
		findResult: &taxDomain.TaxClass{ID: "class-1", Name: "Standard", IsDefault: true},
		list: []taxDomain.TaxClass{
			{ID: "class-1", Name: "Standard", IsDefault: true},
		},
	}
	uc := apptax.NewUpdateTaxClassUseCase(newMockGormDB(t, 1), repo)

	class, err := uc.Execute(context.Background(), apptax.UpdateTaxClassInput{ID: "class-1", Name: "Standard", IsDefault: true})

	require.NoError(t, err)
	assert.True(t, class.IsDefault)
	require.Len(t, repo.updated, 1)
}

func TestUpdateTaxClass_propagatesRepoError(t *testing.T) {
	repo := &stubTaxClassRepo{
		findResult: &taxDomain.TaxClass{ID: "class-1", Name: "Standard"},
		updateErr:  errors.New("db down"),
	}
	uc := apptax.NewUpdateTaxClassUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), apptax.UpdateTaxClassInput{ID: "class-1", Name: "Standard rate"})

	require.Error(t, err)
}
