package suppliers_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appsuppliers "github.com/adirkuhn/commerceos/application/suppliers"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
)

func TestCreateSupplier_setsFields(t *testing.T) {
	repo := &stubSupplierRepo{}
	uc := appsuppliers.NewCreateSupplierUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	leadTime := 14
	s, err := uc.Execute(context.Background(), appsuppliers.CreateSupplierInput{
		Name: "Acme Foods", Email: "hi@acme.example", Phone: "+1 555-0100", Country: "US",
		LeadTimeDays: &leadTime, Status: suppliersDomain.SupplierStatusActive, Notes: "net-30",
	})

	require.NoError(t, err)
	require.Equal(t, "Acme Foods", s.Name)
	require.Equal(t, "hi@acme.example", s.Email)
	require.Equal(t, 14, *s.LeadTimeDays)
	require.Equal(t, suppliersDomain.SupplierStatusActive, s.Status)
	require.True(t, repo.createCalled)
}

func TestCreateSupplier_propagatesRepoError(t *testing.T) {
	repo := &stubSupplierRepo{createErr: suppliersDomain.ErrSupplierNotFound}
	uc := appsuppliers.NewCreateSupplierUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appsuppliers.CreateSupplierInput{Name: "Acme Foods"})

	require.Error(t, err)
}
