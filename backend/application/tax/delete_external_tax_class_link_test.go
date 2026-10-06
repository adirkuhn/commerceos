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

func TestDeleteExternalTaxClassLink_success(t *testing.T) {
	repo := &stubExternalTaxClassLinkRepo{findResult: &taxDomain.ExternalTaxClassLink{ID: "link-1"}}
	uc := apptax.NewDeleteExternalTaxClassLinkUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "link-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"link-1"}, repo.deleted)
}

func TestDeleteExternalTaxClassLink_propagatesNotFound(t *testing.T) {
	repo := &stubExternalTaxClassLinkRepo{findErr: taxDomain.ErrExternalTaxClassLinkNotFound}
	uc := apptax.NewDeleteExternalTaxClassLinkUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, taxDomain.ErrExternalTaxClassLinkNotFound)
}
