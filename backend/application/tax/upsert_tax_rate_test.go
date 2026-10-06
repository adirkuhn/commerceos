package tax_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptax "github.com/adirkuhn/commerceos/application/tax"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

func TestUpsertTaxRate_success(t *testing.T) {
	classRepo := &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1", Name: "Standard"}}
	rateRepo := &stubTaxRateRepo{}
	uc := apptax.NewUpsertTaxRateUseCase(newMockGormDB(t, 1), rateRepo, classRepo, &eventrecordtest.Outbox{})

	rate, err := uc.Execute(context.Background(), apptax.UpsertTaxRateInput{
		TaxClassID: "class-1", Country: "nl", RateBasisPoints: 2100,
	})

	require.NoError(t, err)
	assert.Equal(t, "NL", rate.Country)
	assert.Equal(t, int32(2100), rate.RateBasisPoints)
	require.Len(t, rateRepo.upserted, 1)
	assert.Equal(t, "NL", rateRepo.upserted[0].Country)
}

func TestUpsertTaxRate_rejectsInvalidCountryCode(t *testing.T) {
	classRepo := &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1"}}
	uc := apptax.NewUpsertTaxRateUseCase(newMockGormDB(t, 1), &stubTaxRateRepo{}, classRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apptax.UpsertTaxRateInput{
		TaxClassID: "class-1", Country: "Netherlands", RateBasisPoints: 2100,
	})

	assert.ErrorIs(t, err, taxDomain.ErrInvalidCountryCode)
}

func TestUpsertTaxRate_rejectsNegativeRate(t *testing.T) {
	classRepo := &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1"}}
	uc := apptax.NewUpsertTaxRateUseCase(newMockGormDB(t, 1), &stubTaxRateRepo{}, classRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apptax.UpsertTaxRateInput{
		TaxClassID: "class-1", Country: "NL", RateBasisPoints: -1,
	})

	assert.ErrorIs(t, err, taxDomain.ErrInvalidTaxRateValue)
}

func TestUpsertTaxRate_propagatesTaxClassNotFound(t *testing.T) {
	classRepo := &stubTaxClassRepo{findErr: taxDomain.ErrTaxClassNotFound}
	uc := apptax.NewUpsertTaxRateUseCase(newMockGormDB(t, 1), &stubTaxRateRepo{}, classRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apptax.UpsertTaxRateInput{
		TaxClassID: "missing", Country: "NL", RateBasisPoints: 2100,
	})

	assert.ErrorIs(t, err, taxDomain.ErrTaxClassNotFound)
}

func TestUpsertTaxRate_propagatesRepoError(t *testing.T) {
	classRepo := &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1"}}
	rateRepo := &stubTaxRateRepo{upsertErr: errors.New("db down")}
	uc := apptax.NewUpsertTaxRateUseCase(newMockGormDB(t, 1), rateRepo, classRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apptax.UpsertTaxRateInput{
		TaxClassID: "class-1", Country: "NL", RateBasisPoints: 2100,
	})

	require.Error(t, err)
}
