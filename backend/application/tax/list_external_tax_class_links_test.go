package tax_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptax "github.com/adirkuhn/commerceos/application/tax"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

func TestListExternalTaxClassLinks_returnsTheRepoList(t *testing.T) {
	repo := &stubExternalTaxClassLinkRepo{list: []taxDomain.ExternalTaxClassLink{{ID: "link-1", Provider: "woocommerce", ExternalSlug: "reduced-rate"}}}
	uc := apptax.NewListExternalTaxClassLinksUseCase(newMockGormDB(t, 1), repo)

	links, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, repo.list, links)
}

func TestListExternalTaxClassLinks_propagatesRepoError(t *testing.T) {
	repo := &stubExternalTaxClassLinkRepo{listErr: assert.AnError}
	uc := apptax.NewListExternalTaxClassLinksUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}
