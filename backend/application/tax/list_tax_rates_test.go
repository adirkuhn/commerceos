package tax_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptax "github.com/adirkuhn/commerceos/application/tax"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

func TestListTaxRates_returnsTheRepoList(t *testing.T) {
	repo := &stubTaxRateRepo{list: []taxDomain.TaxRate{{ID: "rate-1", Country: "NL"}}}
	uc := apptax.NewListTaxRatesUseCase(newMockGormDB(t, 1), repo)

	rates, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, repo.list, rates)
}

func TestListTaxRates_propagatesRepoError(t *testing.T) {
	repo := &stubTaxRateRepo{listErr: assert.AnError}
	uc := apptax.NewListTaxRatesUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}
