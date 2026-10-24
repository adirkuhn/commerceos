package customers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
)

func TestGetCustomer(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c-1", FirstName: "Ana"}}

	c, err := appcustomers.NewGetCustomerUseCase(newMockGormDB(t), repo).Execute(context.Background(), "c-1")

	require.NoError(t, err)
	assert.Equal(t, "Ana", c.FirstName)
}

func TestGetCustomer_notFound(t *testing.T) {
	repo := &stubCustomerRepo{findErr: customersDomain.ErrCustomerNotFound}

	_, err := appcustomers.NewGetCustomerUseCase(newMockGormDB(t), repo).Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, customersDomain.ErrCustomerNotFound)
}
