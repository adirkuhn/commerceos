package customers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
)

func TestUpdateCustomer_updatesFields(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c1", Email: "old@example.com"}}
	uc := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo)

	c, err := uc.Execute(context.Background(), appcustomers.UpdateCustomerInput{
		ID: "c1", Email: "new@example.com", FirstName: "Jane", LastName: "Doe",
	})

	require.NoError(t, err)
	require.Equal(t, "new@example.com", c.Email)
	require.Len(t, repo.updated, 1)
	require.Equal(t, "new@example.com", repo.updated[0].Email)
}

func TestUpdateCustomer_propagatesNotFound(t *testing.T) {
	repo := &stubCustomerRepo{findErr: customersDomain.ErrCustomerNotFound}
	uc := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appcustomers.UpdateCustomerInput{ID: "missing"})

	require.ErrorIs(t, err, customersDomain.ErrCustomerNotFound)
}
