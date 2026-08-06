package suppliers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appsuppliers "github.com/adirkuhn/commerceos/application/suppliers"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
)

func TestDeleteSupplier_deletesExisting(t *testing.T) {
	repo := &stubSupplierRepo{findResult: &suppliersDomain.Supplier{ID: "supplier-1"}}
	uc := appsuppliers.NewDeleteSupplierUseCase(newMockGormDB(t), repo)

	err := uc.Execute(context.Background(), "supplier-1")

	require.NoError(t, err)
	require.Equal(t, []string{"supplier-1"}, repo.deleted)
}

func TestDeleteSupplier_propagatesNotFound(t *testing.T) {
	repo := &stubSupplierRepo{findErr: suppliersDomain.ErrSupplierNotFound}
	uc := appsuppliers.NewDeleteSupplierUseCase(newMockGormDB(t), repo)

	err := uc.Execute(context.Background(), "missing")

	require.ErrorIs(t, err, suppliersDomain.ErrSupplierNotFound)
}
