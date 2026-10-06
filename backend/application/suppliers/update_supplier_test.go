package suppliers_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appsuppliers "github.com/adirkuhn/commerceos/application/suppliers"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
)

func TestUpdateSupplier_updatesFields(t *testing.T) {
	repo := &stubSupplierRepo{findResult: &suppliersDomain.Supplier{ID: "supplier-1", Name: "Old Name"}}
	uc := appsuppliers.NewUpdateSupplierUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	s, err := uc.Execute(context.Background(), appsuppliers.UpdateSupplierInput{
		ID: "supplier-1", Name: "New Name", Status: suppliersDomain.SupplierStatusInactive,
	})

	require.NoError(t, err)
	require.Equal(t, "New Name", s.Name)
	require.Equal(t, suppliersDomain.SupplierStatusInactive, s.Status)
}

func TestUpdateSupplier_propagatesNotFound(t *testing.T) {
	repo := &stubSupplierRepo{findErr: suppliersDomain.ErrSupplierNotFound}
	uc := appsuppliers.NewUpdateSupplierUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appsuppliers.UpdateSupplierInput{ID: "missing"})

	require.ErrorIs(t, err, suppliersDomain.ErrSupplierNotFound)
}
