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

func TestUpsertExternalTaxClassLink_success(t *testing.T) {
	classRepo := &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1", Name: "Reduced rate"}}
	linkRepo := &stubExternalTaxClassLinkRepo{}
	uc := apptax.NewUpsertExternalTaxClassLinkUseCase(newMockGormDB(t, 1), linkRepo, classRepo)

	link, err := uc.Execute(context.Background(), apptax.UpsertExternalTaxClassLinkInput{
		Provider: "woocommerce", ExternalSlug: "reduced-rate", TaxClassID: "class-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "woocommerce", link.Provider)
	assert.Equal(t, "reduced-rate", link.ExternalSlug)
	assert.Equal(t, "class-1", link.TaxClassID)
	require.Len(t, linkRepo.upserted, 1)
}

func TestUpsertExternalTaxClassLink_allowsEmptyExternalSlug(t *testing.T) {
	// "" represents WooCommerce's own Standard rate -- must not be
	// rejected as "missing".
	classRepo := &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1", Name: "Standard"}}
	linkRepo := &stubExternalTaxClassLinkRepo{}
	uc := apptax.NewUpsertExternalTaxClassLinkUseCase(newMockGormDB(t, 1), linkRepo, classRepo)

	link, err := uc.Execute(context.Background(), apptax.UpsertExternalTaxClassLinkInput{
		Provider: "woocommerce", ExternalSlug: "", TaxClassID: "class-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "", link.ExternalSlug)
}

func TestUpsertExternalTaxClassLink_rejectsEmptyProvider(t *testing.T) {
	uc := apptax.NewUpsertExternalTaxClassLinkUseCase(newMockGormDB(t, 0), &stubExternalTaxClassLinkRepo{}, &stubTaxClassRepo{})

	_, err := uc.Execute(context.Background(), apptax.UpsertExternalTaxClassLinkInput{
		Provider: "  ", ExternalSlug: "reduced-rate", TaxClassID: "class-1",
	})

	assert.ErrorIs(t, err, taxDomain.ErrProviderRequired)
}

func TestUpsertExternalTaxClassLink_propagatesTaxClassNotFound(t *testing.T) {
	classRepo := &stubTaxClassRepo{findErr: taxDomain.ErrTaxClassNotFound}
	uc := apptax.NewUpsertExternalTaxClassLinkUseCase(newMockGormDB(t, 1), &stubExternalTaxClassLinkRepo{}, classRepo)

	_, err := uc.Execute(context.Background(), apptax.UpsertExternalTaxClassLinkInput{
		Provider: "woocommerce", ExternalSlug: "reduced-rate", TaxClassID: "missing",
	})

	assert.ErrorIs(t, err, taxDomain.ErrTaxClassNotFound)
}

func TestUpsertExternalTaxClassLink_propagatesRepoError(t *testing.T) {
	classRepo := &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1"}}
	linkRepo := &stubExternalTaxClassLinkRepo{upsertErr: errors.New("db down")}
	uc := apptax.NewUpsertExternalTaxClassLinkUseCase(newMockGormDB(t, 1), linkRepo, classRepo)

	_, err := uc.Execute(context.Background(), apptax.UpsertExternalTaxClassLinkInput{
		Provider: "woocommerce", ExternalSlug: "reduced-rate", TaxClassID: "class-1",
	})

	require.Error(t, err)
}
