package customers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
)

func TestListCustomers_returnsRepoResult(t *testing.T) {
	repo := &stubCustomerRepo{customers: []customersDomain.Customer{{ID: "c1", Email: "a@b.com"}}}
	uc := appcustomers.NewListCustomersUseCase(newMockGormDB(t), repo)

	out, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "a@b.com", out[0].Email)
}
