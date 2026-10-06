package suppliers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appsuppliers "github.com/adirkuhn/commerceos/application/suppliers"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
)

func intPtr(i int) *int { return &i }

func TestCreateSupplier_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appsuppliers.NewCreateSupplierUseCase(newMockGormDB(t), &stubSupplierRepo{}, outbox)

	_, err := uc.Execute(context.Background(), appsuppliers.CreateSupplierInput{Name: "Bauducco", Status: suppliersDomain.SupplierStatusActive})

	require.NoError(t, err)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "suppliers.supplier.created", e.Subject)
	assert.Equal(t, "supplier-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"name": "Bauducco", "status": "active"}, e.Data())
}

func TestUpdateSupplier_RecordsChangedFields(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubSupplierRepo{findResult: &suppliersDomain.Supplier{
		ID: "supplier-1", Name: "Bauducco", Email: "a@b.nl", LeadTimeDays: intPtr(14), Status: suppliersDomain.SupplierStatusActive,
	}}
	uc := appsuppliers.NewUpdateSupplierUseCase(newMockGormDB(t), repo, outbox)

	_, err := uc.Execute(context.Background(), appsuppliers.UpdateSupplierInput{
		ID: "supplier-1", Name: "Bauducco", Email: "a@b.nl", LeadTimeDays: intPtr(21), Status: suppliersDomain.SupplierStatusActive,
	})

	require.NoError(t, err)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "suppliers.supplier.updated", e.Subject)
	assert.Equal(t, map[string][2]any{"lead_time_days": {float64(14), float64(21)}}, e.Changes())
}

func TestDeleteSupplier_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubSupplierRepo{findResult: &suppliersDomain.Supplier{ID: "supplier-1", Name: "Bauducco"}}
	uc := appsuppliers.NewDeleteSupplierUseCase(newMockGormDB(t), repo, outbox)

	require.NoError(t, uc.Execute(context.Background(), "supplier-1"))

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "suppliers.supplier.deleted", e.Subject)
	assert.Equal(t, map[string]any{"name": "Bauducco"}, e.Data())
}

func TestSupplierUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	found := &suppliersDomain.Supplier{ID: "supplier-1"}

	_, err := appsuppliers.NewCreateSupplierUseCase(newMockGormDB(t), &stubSupplierRepo{}, &eventrecordtest.Outbox{Err: boom}).
		Execute(context.Background(), appsuppliers.CreateSupplierInput{Name: "x"})
	assert.ErrorIs(t, err, boom)
	_, err = appsuppliers.NewUpdateSupplierUseCase(newMockGormDB(t), &stubSupplierRepo{findResult: found}, &eventrecordtest.Outbox{Err: boom}).
		Execute(context.Background(), appsuppliers.UpdateSupplierInput{ID: "supplier-1", Name: "y"})
	assert.ErrorIs(t, err, boom)
	err = appsuppliers.NewDeleteSupplierUseCase(newMockGormDB(t), &stubSupplierRepo{findResult: found}, &eventrecordtest.Outbox{Err: boom}).
		Execute(context.Background(), "supplier-1")
	assert.ErrorIs(t, err, boom)
}
