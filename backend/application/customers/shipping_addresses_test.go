package customers_test

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
)

// memShippingRepo is an in-memory ShippingAddressRepository keeping the
// real one's rules: one default, not-found errors.
type memShippingRepo struct {
	rows   []customersDomain.ShippingAddress
	nextID int
}

func (m *memShippingRepo) ListByCustomer(_ context.Context, customerID string) (customersDomain.ShippingBook, error) {
	var out customersDomain.ShippingBook
	for _, r := range m.rows {
		if r.CustomerID == customerID {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out, nil
}

func (m *memShippingRepo) Create(_ context.Context, a *customersDomain.ShippingAddress) error {
	m.nextID++
	a.ID = fmt.Sprintf("sa-%d", m.nextID)
	m.rows = append(m.rows, *a)
	return nil
}

func (m *memShippingRepo) find(customerID, id string) (int, error) {
	for i, r := range m.rows {
		if r.ID == id && r.CustomerID == customerID {
			return i, nil
		}
	}
	return -1, customersDomain.ErrShippingAddressNotFound
}

func (m *memShippingRepo) Update(_ context.Context, a *customersDomain.ShippingAddress) error {
	i, err := m.find(a.CustomerID, a.ID)
	if err != nil {
		return err
	}
	a.IsDefault = m.rows[i].IsDefault
	m.rows[i] = *a
	return nil
}

func (m *memShippingRepo) Delete(_ context.Context, customerID, id string) error {
	i, err := m.find(customerID, id)
	if err != nil {
		return err
	}
	m.rows = append(m.rows[:i], m.rows[i+1:]...)
	return nil
}

func (m *memShippingRepo) SetDefault(_ context.Context, customerID, id string) error {
	if _, err := m.find(customerID, id); err != nil {
		return err
	}
	for i := range m.rows {
		if m.rows[i].CustomerID == customerID {
			m.rows[i].IsDefault = m.rows[i].ID == id
		}
	}
	return nil
}

func (m *memShippingRepo) seed(customerID, line string, isDefault bool) string {
	a := customersDomain.ShippingAddress{CustomerID: customerID, Address: customersDomain.Address{Address1: line}, IsDefault: isDefault, Position: len(m.rows)}
	_ = m.Create(context.Background(), &a)
	return a.ID
}

type shippingHarness struct {
	customers *stubCustomerRepo
	addresses *memShippingRepo
	outbox    *eventrecordtest.Outbox
	deps      appcustomers.ShippingAddressDeps
}

func newShippingHarness(t *testing.T, c customersDomain.Customer) *shippingHarness {
	h := &shippingHarness{
		customers: &stubCustomerRepo{findResult: &c},
		addresses: &memShippingRepo{},
		outbox:    &eventrecordtest.Outbox{},
	}
	h.deps = appcustomers.ShippingAddressDeps{DB: newMockGormDB(t), Customers: h.customers, Addresses: h.addresses, Outbox: h.outbox}
	return h
}

// again gives the next use case call its own mock transaction.
func (h *shippingHarness) again(t *testing.T) appcustomers.ShippingAddressDeps {
	h.deps.DB = newMockGormDB(t)
	return h.deps
}

// lastCopy is the customer's Shipping copy as last saved.
func (h *shippingHarness) lastCopy(t *testing.T) customersDomain.Address {
	t.Helper()
	require.NotEmpty(t, h.customers.updated, "customer copy saved")
	return h.customers.updated[len(h.customers.updated)-1].Shipping
}

func TestAddShippingAddress_firstBecomesDefaultAndTheCustomersCopy(t *testing.T) {
	h := newShippingHarness(t, customersDomain.Customer{ID: "c-1", TenantID: "t-1"})

	a, err := appcustomers.NewAddShippingAddressUseCase(h.deps).Execute(context.Background(), "c-1", appcustomers.ShippingAddressInput{
		Label: "Warehouse", Address: customersDomain.Address{Address1: "Dock 2", City: "Utrecht", Email: "dropped@x.nl"},
	}, false)

	require.NoError(t, err)
	assert.Equal(t, "sa-1", a.ID)
	assert.True(t, a.IsDefault)
	assert.Empty(t, a.Address.Email, "a shipping address has no email")
	assert.Equal(t, "Dock 2", h.lastCopy(t).Address1)
	e, err := h.outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "customers.shipping_address.added", e.Subject)
	assert.Equal(t, map[string]any{"customer_id": "c-1", "default": true}, e.Data())
}

func TestAddShippingAddress_asDefaultMovesTheDefault(t *testing.T) {
	h := newShippingHarness(t, customersDomain.Customer{ID: "c-1", Shipping: customersDomain.Address{Address1: "Main 1"}})
	first := h.addresses.seed("c-1", "Main 1", true)

	a, err := appcustomers.NewAddShippingAddressUseCase(h.deps).Execute(context.Background(), "c-1",
		appcustomers.ShippingAddressInput{Address: customersDomain.Address{Address1: "Dock 2"}}, true)

	require.NoError(t, err)
	book, _ := h.addresses.ListByCustomer(context.Background(), "c-1")
	assert.Equal(t, a.ID, book.Default().ID)
	assert.NotEqual(t, first, book.Default().ID)
	assert.Equal(t, "Dock 2", h.lastCopy(t).Address1)
}

func TestAddShippingAddress_laterOneKeepsTheDefaultAndTheCopy(t *testing.T) {
	h := newShippingHarness(t, customersDomain.Customer{ID: "c-1", Shipping: customersDomain.Address{Address1: "Main 1"}})
	h.addresses.seed("c-1", "Main 1", true)

	_, err := appcustomers.NewAddShippingAddressUseCase(h.deps).Execute(context.Background(), "c-1",
		appcustomers.ShippingAddressInput{Address: customersDomain.Address{Address1: "Dock 2"}}, false)

	require.NoError(t, err)
	assert.Empty(t, h.customers.updated, "copy unchanged, customer not saved")
}

func TestAddShippingAddress_errors(t *testing.T) {
	h := newShippingHarness(t, customersDomain.Customer{ID: "c-1"})
	_, err := appcustomers.NewAddShippingAddressUseCase(h.deps).Execute(context.Background(), "c-1", appcustomers.ShippingAddressInput{}, false)
	assert.ErrorIs(t, err, customersDomain.ErrShippingAddressLineRequired)

	h.customers.findErr, h.customers.findResult = customersDomain.ErrCustomerNotFound, nil
	_, err = appcustomers.NewAddShippingAddressUseCase(h.again(t)).Execute(context.Background(), "missing",
		appcustomers.ShippingAddressInput{Address: customersDomain.Address{Address1: "Dock 2"}}, false)
	assert.ErrorIs(t, err, customersDomain.ErrCustomerNotFound)
	assert.Empty(t, h.addresses.rows)
}

func TestUpdateShippingAddress_editingTheDefaultRefreshesTheCopy(t *testing.T) {
	h := newShippingHarness(t, customersDomain.Customer{ID: "c-1", Shipping: customersDomain.Address{Address1: "Main 1"}})
	id := h.addresses.seed("c-1", "Main 1", true)

	a, err := appcustomers.NewUpdateShippingAddressUseCase(h.deps).Execute(context.Background(), "c-1", id,
		appcustomers.ShippingAddressInput{Label: "Home", Address: customersDomain.Address{Address1: "Main 1", City: "Delft"}})

	require.NoError(t, err)
	assert.True(t, a.IsDefault)
	assert.Equal(t, "Home", a.Label)
	assert.Equal(t, "Delft", h.lastCopy(t).City)
	e, err := h.outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "customers.shipping_address.updated", e.Subject)
	assert.Equal(t, map[string]any{"customer_id": "c-1", "changed_fields": []any{"city", "label"}}, e.Data())
}

func TestUpdateShippingAddress_errors(t *testing.T) {
	h := newShippingHarness(t, customersDomain.Customer{ID: "c-1"})
	id := h.addresses.seed("c-1", "Main 1", true)

	_, err := appcustomers.NewUpdateShippingAddressUseCase(h.deps).Execute(context.Background(), "c-1", "nope",
		appcustomers.ShippingAddressInput{Address: customersDomain.Address{Address1: "X"}})
	assert.ErrorIs(t, err, customersDomain.ErrShippingAddressNotFound)

	_, err = appcustomers.NewUpdateShippingAddressUseCase(h.again(t)).Execute(context.Background(), "c-1", id, appcustomers.ShippingAddressInput{})
	assert.ErrorIs(t, err, customersDomain.ErrShippingAddressLineRequired)
}

func TestRemoveShippingAddress_removingTheDefaultPromotesTheNext(t *testing.T) {
	h := newShippingHarness(t, customersDomain.Customer{ID: "c-1", Shipping: customersDomain.Address{Address1: "Main 1"}})
	first := h.addresses.seed("c-1", "Main 1", true)
	second := h.addresses.seed("c-1", "Dock 2", false)

	require.NoError(t, appcustomers.NewRemoveShippingAddressUseCase(h.deps).Execute(context.Background(), "c-1", first))

	book, _ := h.addresses.ListByCustomer(context.Background(), "c-1")
	require.Len(t, book, 1)
	assert.Equal(t, second, book.Default().ID)
	assert.Equal(t, "Dock 2", h.lastCopy(t).Address1)
	assert.Equal(t, []string{"customers.shipping_address.removed", "customers.shipping_address.default_changed"}, h.outbox.Subjects())
}

func TestRemoveShippingAddress_theLastOneEmptiesTheCopy(t *testing.T) {
	h := newShippingHarness(t, customersDomain.Customer{ID: "c-1", Shipping: customersDomain.Address{Address1: "Main 1"}})
	id := h.addresses.seed("c-1", "Main 1", true)

	require.NoError(t, appcustomers.NewRemoveShippingAddressUseCase(h.deps).Execute(context.Background(), "c-1", id))

	assert.Equal(t, customersDomain.Address{}, h.lastCopy(t))
	assert.Equal(t, []string{"customers.shipping_address.removed"}, h.outbox.Subjects())
}

func TestRemoveShippingAddress_notFound(t *testing.T) {
	h := newShippingHarness(t, customersDomain.Customer{ID: "c-1"})
	err := appcustomers.NewRemoveShippingAddressUseCase(h.deps).Execute(context.Background(), "c-1", "nope")
	assert.ErrorIs(t, err, customersDomain.ErrShippingAddressNotFound)
}

func TestSetDefaultShippingAddress_movesTheDefaultAndTheCopy(t *testing.T) {
	h := newShippingHarness(t, customersDomain.Customer{ID: "c-1", Shipping: customersDomain.Address{Address1: "Main 1"}})
	h.addresses.seed("c-1", "Main 1", true)
	second := h.addresses.seed("c-1", "Dock 2", false)

	book, err := appcustomers.NewSetDefaultShippingAddressUseCase(h.deps).Execute(context.Background(), "c-1", second)

	require.NoError(t, err)
	assert.Equal(t, second, book.Default().ID)
	assert.Equal(t, "Dock 2", h.lastCopy(t).Address1)
	e, err := h.outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "customers.shipping_address.default_changed", e.Subject)
	assert.Equal(t, second, e.Payload["entity_id"])

	_, err = appcustomers.NewSetDefaultShippingAddressUseCase(h.again(t)).Execute(context.Background(), "c-1", "nope")
	assert.ErrorIs(t, err, customersDomain.ErrShippingAddressNotFound)
}

func TestShippingAddresses_shipToBillingKeepsTheCopyOnBilling(t *testing.T) {
	billing := customersDomain.Address{Address1: "Bill 1", City: "Delft"}
	h := newShippingHarness(t, customersDomain.Customer{ID: "c-1", Billing: billing, ShipToBilling: true, Shipping: billing})

	_, err := appcustomers.NewAddShippingAddressUseCase(h.deps).Execute(context.Background(), "c-1",
		appcustomers.ShippingAddressInput{Address: customersDomain.Address{Address1: "Dock 2"}}, true)

	require.NoError(t, err)
	assert.Empty(t, h.customers.updated, "goods still go to billing")
}

func TestListShippingAddresses(t *testing.T) {
	h := newShippingHarness(t, customersDomain.Customer{ID: "c-1"})
	h.addresses.seed("c-1", "Main 1", true)
	h.addresses.seed("other", "Elsewhere", true)

	book, err := appcustomers.NewListShippingAddressesUseCase(h.deps).Execute(context.Background(), "c-1")

	require.NoError(t, err)
	require.Len(t, book, 1)
	assert.Equal(t, "Main 1", book[0].Address.Address1)

	h.customers.findErr, h.customers.findResult = customersDomain.ErrCustomerNotFound, nil
	_, err = appcustomers.NewListShippingAddressesUseCase(h.again(t)).Execute(context.Background(), "missing")
	assert.ErrorIs(t, err, customersDomain.ErrCustomerNotFound)
}
