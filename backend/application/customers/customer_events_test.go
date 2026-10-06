package customers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
)

func int64Ptr(i int64) *int64 { return &i }

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

func TestCreateCustomer_RecordsCreatedWithoutPersonalData(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appcustomers.NewCreateCustomerUseCase(newMockGormDB(t), &stubCustomerRepo{}, outbox)

	_, err := uc.Execute(context.Background(), appcustomers.CreateCustomerInput{
		WooCustomerID: int64Ptr(7), Email: "ana@example.com", FirstName: "Ana", Role: "customer",
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "customers.customer.created", e.Subject)
	assert.Equal(t, "customer-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"woo_customer_id": float64(7), "role": "customer", "is_paying_customer": false}, e.Data())
	assert.NotContains(t, e.Payload, "changes")
}

// Personal fields are named, never valued.
func TestUpdateCustomer_RecordsChangesAndPersonalFieldNamesOnly(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{
		ID: "customer-1", Email: "ana@example.com", FirstName: "Ana", Role: "customer",
		Billing: customersDomain.Address{City: "Utrecht"},
	}}
	uc := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, outbox)

	_, err := uc.Execute(context.Background(), appcustomers.UpdateCustomerInput{
		ID: "customer-1", Email: "ana@new.example", FirstName: "Ana", Role: "customer", IsPayingCustomer: true,
		Billing: customersDomain.Address{City: "Amsterdam"},
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "customers.customer.updated", e.Subject)
	assert.Equal(t, map[string][2]any{"is_paying_customer": {false, true}}, e.Changes())
	assert.Equal(t, map[string]any{"changed_fields": []any{"billing", "email"}}, e.Data())
	assert.NotContains(t, e.Payload, "ana@new.example")
}

func TestUpdateCustomer_NoPersonalChangesNoChangedFields(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "customer-1", Role: "customer"}}
	uc := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, outbox)

	_, err := uc.Execute(context.Background(), appcustomers.UpdateCustomerInput{ID: "customer-1", Role: "subscriber"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, map[string][2]any{"role": {"customer", "subscriber"}}, e.Changes())
	assert.Nil(t, e.Data())
}

func TestDeleteCustomer_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "customer-1", WooCustomerID: int64Ptr(7)}}
	uc := appcustomers.NewDeleteCustomerUseCase(newMockGormDB(t), repo, outbox)

	require.NoError(t, uc.Execute(context.Background(), "customer-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "customers.customer.deleted", e.Subject)
	assert.Equal(t, map[string]any{"woo_customer_id": float64(7)}, e.Data())
}

func TestCustomerUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}
	found := func() *stubCustomerRepo {
		return &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "customer-1"}}
	}
	ctx := context.Background()

	_, err := appcustomers.NewCreateCustomerUseCase(newMockGormDB(t), &stubCustomerRepo{}, failing).Execute(ctx, appcustomers.CreateCustomerInput{})
	assert.ErrorIs(t, err, boom, "create")
	_, err = appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), found(), failing).Execute(ctx, appcustomers.UpdateCustomerInput{ID: "customer-1"})
	assert.ErrorIs(t, err, boom, "update")
	assert.ErrorIs(t, appcustomers.NewDeleteCustomerUseCase(newMockGormDB(t), found(), failing).Execute(ctx, "customer-1"), boom, "delete")
}
