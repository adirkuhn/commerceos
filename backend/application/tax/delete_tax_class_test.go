package tax_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptax "github.com/adirkuhn/commerceos/application/tax"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

func TestDeleteTaxClass_success(t *testing.T) {
	repo := &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1", Name: "Reduced rate"}}
	checker := &stubSKUTaxClassChecker{exists: false}
	uc := apptax.NewDeleteTaxClassUseCase(newMockGormDB(t, 1), repo, checker)

	err := uc.Execute(context.Background(), "class-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"class-1"}, repo.deleted)
}

func TestDeleteTaxClass_propagatesNotFound(t *testing.T) {
	repo := &stubTaxClassRepo{findErr: taxDomain.ErrTaxClassNotFound}
	uc := apptax.NewDeleteTaxClassUseCase(newMockGormDB(t, 1), repo, &stubSKUTaxClassChecker{})

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, taxDomain.ErrTaxClassNotFound)
}

func TestDeleteTaxClass_rejectsDeletingTheDefaultClass(t *testing.T) {
	repo := &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1", Name: "Standard", IsDefault: true}}
	uc := apptax.NewDeleteTaxClassUseCase(newMockGormDB(t, 1), repo, &stubSKUTaxClassChecker{})

	err := uc.Execute(context.Background(), "class-1")

	assert.ErrorIs(t, err, taxDomain.ErrCannotDeleteDefaultTaxClass)
	assert.Empty(t, repo.deleted)
}

func TestDeleteTaxClass_rejectsWhenAssignedToASKU(t *testing.T) {
	repo := &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1", Name: "Reduced rate"}}
	checker := &stubSKUTaxClassChecker{exists: true}
	uc := apptax.NewDeleteTaxClassUseCase(newMockGormDB(t, 1), repo, checker)

	err := uc.Execute(context.Background(), "class-1")

	assert.ErrorIs(t, err, taxDomain.ErrTaxClassInUse)
	assert.Empty(t, repo.deleted)
}
