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

func TestTaxRateResolver_returnsTheClassRateForTheCountry(t *testing.T) {
	classes := &stubTaxClassRepo{list: []taxDomain.TaxClass{{ID: "standard", IsDefault: true}}}
	rates := &stubTaxRateRepo{list: []taxDomain.TaxRate{{TaxClassID: "standard", Country: "NL", RateBasisPoints: 2100}}}

	bp, err := apptax.NewTaxRateResolver(classes, rates).RateBasisPoints(context.Background(), nil, "NL")

	require.NoError(t, err)
	assert.Equal(t, int64(2100), bp)
}

func TestTaxRateResolver_noRateConfiguredIsZero(t *testing.T) {
	bp, err := apptax.NewTaxRateResolver(&stubTaxClassRepo{}, &stubTaxRateRepo{}).RateBasisPoints(context.Background(), nil, "NL")

	require.NoError(t, err)
	assert.Zero(t, bp)
}

func TestTaxRateResolver_propagatesRepoErrors(t *testing.T) {
	boom := errors.New("db down")
	_, err := apptax.NewTaxRateResolver(&stubTaxClassRepo{listErr: boom}, &stubTaxRateRepo{}).RateBasisPoints(context.Background(), nil, "NL")
	assert.ErrorIs(t, err, boom)

	_, err = apptax.NewTaxRateResolver(&stubTaxClassRepo{}, &stubTaxRateRepo{listErr: boom}).RateBasisPoints(context.Background(), nil, "NL")
	assert.ErrorIs(t, err, boom)
}
