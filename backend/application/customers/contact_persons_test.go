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

type stubContactPersonRepo struct {
	created  []customersDomain.ContactPerson
	updated  []customersDomain.ContactPerson
	deleted  []string
	found    *customersDomain.ContactPerson
	listed   []customersDomain.ContactPerson
	err      error
	findErr  error
	gotOwner string
}

func (s *stubContactPersonRepo) Create(_ context.Context, p *customersDomain.ContactPerson) error {
	if s.err != nil {
		return s.err
	}
	p.ID = "cp-new"
	s.created = append(s.created, *p)
	return nil
}

func (s *stubContactPersonRepo) FindByID(_ context.Context, customerID, _ string) (*customersDomain.ContactPerson, error) {
	s.gotOwner = customerID
	if s.findErr != nil {
		return nil, s.findErr
	}
	p := *s.found
	return &p, nil
}

func (s *stubContactPersonRepo) ListByCustomer(_ context.Context, customerID string) ([]customersDomain.ContactPerson, error) {
	s.gotOwner = customerID
	return s.listed, s.err
}

func (s *stubContactPersonRepo) Update(_ context.Context, p *customersDomain.ContactPerson) error {
	s.updated = append(s.updated, *p)
	return s.err
}

func (s *stubContactPersonRepo) Delete(_ context.Context, customerID, id string) error {
	s.gotOwner = customerID
	s.deleted = append(s.deleted, id)
	return s.err
}

func TestAddContactPerson_addsAndRecordsWithoutPersonalData(t *testing.T) {
	persons := &stubContactPersonRepo{}
	outbox := &eventrecordtest.Outbox{}
	uc := appcustomers.NewAddContactPersonUseCase(newMockGormDB(t), &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c-1"}}, persons, outbox)

	p, err := uc.Execute(context.Background(), "c-1", appcustomers.ContactPersonInput{FirstName: "Kylian", LastName: "Klockenbrink", Role: "Employee"})

	require.NoError(t, err)
	assert.Equal(t, "cp-new", p.ID)
	require.Len(t, persons.created, 1)
	assert.Equal(t, "c-1", persons.created[0].CustomerID)
	assert.Equal(t, "Employee", persons.created[0].Role)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "customers.contact_person.added", e.Subject)
	assert.Equal(t, map[string]any{"customer_id": "c-1"}, e.Data())
}

func TestAddContactPerson_needsAName(t *testing.T) {
	_, err := appcustomers.NewAddContactPersonUseCase(nil, &stubCustomerRepo{}, &stubContactPersonRepo{}, &eventrecordtest.Outbox{}).
		Execute(context.Background(), "c-1", appcustomers.ContactPersonInput{Email: "k@claus.nl"})

	assert.ErrorIs(t, err, customersDomain.ErrContactPersonNameRequired)
}

func TestAddContactPerson_unknownCustomer(t *testing.T) {
	persons := &stubContactPersonRepo{}
	_, err := appcustomers.NewAddContactPersonUseCase(newMockGormDB(t), &stubCustomerRepo{findErr: customersDomain.ErrCustomerNotFound}, persons, &eventrecordtest.Outbox{}).
		Execute(context.Background(), "missing", appcustomers.ContactPersonInput{FirstName: "Kylian"})

	assert.ErrorIs(t, err, customersDomain.ErrCustomerNotFound)
	assert.Empty(t, persons.created)
}

func TestUpdateContactPerson_changesFieldsAndNamesThem(t *testing.T) {
	persons := &stubContactPersonRepo{found: &customersDomain.ContactPerson{ID: "cp-1", CustomerID: "c-1", FirstName: "Kylian"}}
	outbox := &eventrecordtest.Outbox{}

	p, err := appcustomers.NewUpdateContactPersonUseCase(newMockGormDB(t), persons, outbox).
		Execute(context.Background(), "c-1", "cp-1", appcustomers.ContactPersonInput{FirstName: "Kylian", Phone: "0612345678"})

	require.NoError(t, err)
	assert.Equal(t, "0612345678", p.Phone)
	assert.Equal(t, "c-1", persons.gotOwner)
	require.Len(t, persons.updated, 1)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "customers.contact_person.updated", e.Subject)
	assert.Equal(t, []any{"phone"}, e.Data()["changed_fields"])
}

func TestUpdateContactPerson_notFoundAndNameRequired(t *testing.T) {
	_, err := appcustomers.NewUpdateContactPersonUseCase(newMockGormDB(t), &stubContactPersonRepo{findErr: customersDomain.ErrContactPersonNotFound}, &eventrecordtest.Outbox{}).
		Execute(context.Background(), "c-1", "cp-x", appcustomers.ContactPersonInput{FirstName: "A"})
	assert.ErrorIs(t, err, customersDomain.ErrContactPersonNotFound)

	persons := &stubContactPersonRepo{found: &customersDomain.ContactPerson{ID: "cp-1", FirstName: "Kylian"}}
	_, err = appcustomers.NewUpdateContactPersonUseCase(newMockGormDB(t), persons, &eventrecordtest.Outbox{}).
		Execute(context.Background(), "c-1", "cp-1", appcustomers.ContactPersonInput{})
	assert.ErrorIs(t, err, customersDomain.ErrContactPersonNameRequired)
	assert.Empty(t, persons.updated)
}

func TestRemoveContactPerson_deletesAndRecords(t *testing.T) {
	persons := &stubContactPersonRepo{}
	outbox := &eventrecordtest.Outbox{}

	err := appcustomers.NewRemoveContactPersonUseCase(newMockGormDB(t), persons, outbox).Execute(context.Background(), "c-1", "cp-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"cp-1"}, persons.deleted)
	assert.Equal(t, "c-1", persons.gotOwner)
	assert.Equal(t, []string{"customers.contact_person.removed"}, outbox.Subjects())
}

func TestRemoveContactPerson_propagatesNotFound(t *testing.T) {
	err := appcustomers.NewRemoveContactPersonUseCase(newMockGormDB(t), &stubContactPersonRepo{err: customersDomain.ErrContactPersonNotFound}, &eventrecordtest.Outbox{}).
		Execute(context.Background(), "c-1", "cp-x")

	assert.ErrorIs(t, err, customersDomain.ErrContactPersonNotFound)
}

func TestListContactPersons(t *testing.T) {
	persons := &stubContactPersonRepo{listed: []customersDomain.ContactPerson{{ID: "cp-1", FirstName: "Kylian"}}}
	out, err := appcustomers.NewListContactPersonsUseCase(newMockGormDB(t), &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c-1"}}, persons).
		Execute(context.Background(), "c-1")

	require.NoError(t, err)
	assert.Len(t, out, 1)
	assert.Equal(t, "c-1", persons.gotOwner)

	_, err = appcustomers.NewListContactPersonsUseCase(newMockGormDB(t), &stubCustomerRepo{findErr: customersDomain.ErrCustomerNotFound}, persons).
		Execute(context.Background(), "missing")
	assert.ErrorIs(t, err, customersDomain.ErrCustomerNotFound)

	_, err = appcustomers.NewListContactPersonsUseCase(newMockGormDB(t), &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c-1"}}, &stubContactPersonRepo{err: errors.New("boom")}).
		Execute(context.Background(), "c-1")
	assert.Error(t, err)
}

func TestCreateCustomer_withItsFirstContactPerson(t *testing.T) {
	repo := &stubCustomerRepo{}
	persons := &stubContactPersonRepo{}
	outbox := &eventrecordtest.Outbox{}
	uc := appcustomers.NewCreateCustomerUseCase(newMockGormDB(t), repo, outbox).WithContactPersons(persons)

	_, err := uc.Execute(context.Background(), appcustomers.CreateCustomerInput{
		Type: customersDomain.TypeCompany, CompanyName: "Claus Hoofddorp B.V.",
		ContactPersons: []appcustomers.ContactPersonInput{{FirstName: "Kylian", LastName: "Klockenbrink"}},
	})

	require.NoError(t, err)
	require.Len(t, persons.created, 1)
	assert.Equal(t, "Kylian", persons.created[0].FirstName)
	assert.Equal(t, []string{"customers.customer.created", "customers.contact_person.added"}, outbox.Subjects())
}

func TestCreateCustomer_rejectsANamelessContactPersonBeforeSaving(t *testing.T) {
	repo := &stubCustomerRepo{}
	_, err := appcustomers.NewCreateCustomerUseCase(nil, repo, &eventrecordtest.Outbox{}).WithContactPersons(&stubContactPersonRepo{}).
		Execute(context.Background(), appcustomers.CreateCustomerInput{Type: customersDomain.TypeCompany, CompanyName: "Claus",
			ContactPersons: []appcustomers.ContactPersonInput{{Email: "k@claus.nl"}}})

	assert.ErrorIs(t, err, customersDomain.ErrContactPersonNameRequired)
	assert.False(t, repo.createCalled)
}
