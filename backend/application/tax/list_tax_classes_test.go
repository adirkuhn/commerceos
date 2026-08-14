package tax_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptax "github.com/adirkuhn/commerceos/application/tax"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

func TestListTaxClasses_returnsTheRepoList(t *testing.T) {
	repo := &stubTaxClassRepo{list: []taxDomain.TaxClass{{ID: "class-1", Name: "Standard"}}}
	uc := apptax.NewListTaxClassesUseCase(newMockGormDB(t, 1), repo)

	classes, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, repo.list, classes)
}

func TestListTaxClasses_propagatesRepoError(t *testing.T) {
	repo := &stubTaxClassRepo{listErr: assert.AnError}
	uc := apptax.NewListTaxClassesUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}
