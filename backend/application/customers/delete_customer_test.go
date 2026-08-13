package customers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
)

func TestDeleteCustomer_deletesWhenFound(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c1"}}
	uc := appcustomers.NewDeleteCustomerUseCase(newMockGormDB(t), repo)

	err := uc.Execute(context.Background(), "c1")

	require.NoError(t, err)
	require.Equal(t, []string{"c1"}, repo.deleted)
}

func TestDeleteCustomer_propagatesNotFound(t *testing.T) {
	repo := &stubCustomerRepo{findErr: customersDomain.ErrCustomerNotFound}
	uc := appcustomers.NewDeleteCustomerUseCase(newMockGormDB(t), repo)

	err := uc.Execute(context.Background(), "missing")

	require.ErrorIs(t, err, customersDomain.ErrCustomerNotFound)
	require.Empty(t, repo.deleted)
}
