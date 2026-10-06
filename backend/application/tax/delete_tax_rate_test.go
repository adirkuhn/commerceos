package tax_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptax "github.com/adirkuhn/commerceos/application/tax"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

func TestDeleteTaxRate_success(t *testing.T) {
	repo := &stubTaxRateRepo{findResult: &taxDomain.TaxRate{ID: "rate-1"}}
	uc := apptax.NewDeleteTaxRateUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "rate-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"rate-1"}, repo.deleted)
}

func TestDeleteTaxRate_propagatesNotFound(t *testing.T) {
	repo := &stubTaxRateRepo{findErr: taxDomain.ErrTaxRateNotFound}
	uc := apptax.NewDeleteTaxRateUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, taxDomain.ErrTaxRateNotFound)
}
