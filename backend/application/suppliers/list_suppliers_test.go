package suppliers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appsuppliers "github.com/adirkuhn/commerceos/application/suppliers"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
)

func TestListSuppliers_returnsRepoResult(t *testing.T) {
	repo := &stubSupplierRepo{suppliers: []suppliersDomain.Supplier{{ID: "supplier-1", Name: "Acme"}}}
	uc := appsuppliers.NewListSuppliersUseCase(newMockGormDB(t), repo)

	suppliers, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, suppliers, 1)
	require.Equal(t, "Acme", suppliers[0].Name)
}
