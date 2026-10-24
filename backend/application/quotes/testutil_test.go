package quotes_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/quotes"
)

// newMockGormDB mocks only the tenant.WithTenant envelope; the stub repo
// does the rest.
func newMockGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)
	mock.MatchExpectationsInOrder(false)
	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	return gormDB
}

type stubRepo struct {
	byID      map[string]*quotes.Quote
	created   []*quotes.Quote
	updated   []*quotes.Quote
	deleted   []string
	listOut   []quotes.Summary
	listTotal int64
	gotFilter quotes.ListFilter
	err       error
}

func newStubRepo(qs ...*quotes.Quote) *stubRepo {
	r := &stubRepo{byID: map[string]*quotes.Quote{}}
	for _, q := range qs {
		r.byID[q.ID] = q
	}
	return r
}

func (r *stubRepo) Create(_ context.Context, q *quotes.Quote) error {
	if r.err != nil {
		return r.err
	}
	q.ID, q.Number = "q-new", 12
	r.created = append(r.created, q)
	return nil
}

func (r *stubRepo) FindByID(_ context.Context, id string) (*quotes.Quote, error) {
	q, ok := r.byID[id]
	if !ok {
		return nil, quotes.ErrQuoteNotFound
	}
	cp := *q
	cp.Lines = append([]quotes.Line(nil), q.Lines...)
	return &cp, nil
}

func (r *stubRepo) List(_ context.Context, f quotes.ListFilter) ([]quotes.Summary, int64, error) {
	r.gotFilter = f
	return r.listOut, r.listTotal, r.err
}

func (r *stubRepo) Update(_ context.Context, q *quotes.Quote) error {
	if r.err != nil {
		return r.err
	}
	r.updated = append(r.updated, q)
	return nil
}

func (r *stubRepo) Delete(_ context.Context, id string) error {
	if r.err != nil {
		return r.err
	}
	r.deleted = append(r.deleted, id)
	return nil
}

type stubCustomers struct {
	byID map[string]*customersDomain.Customer
}

func (s *stubCustomers) Execute(_ context.Context, id string) (*customersDomain.Customer, error) {
	if c, ok := s.byID[id]; ok {
		return c, nil
	}
	return nil, customersDomain.ErrCustomerNotFound
}

func newStubCustomers() *stubCustomers {
	return &stubCustomers{byID: map[string]*customersDomain.Customer{
		"c-1": {ID: "c-1", FirstName: "Ana", LastName: "Souza", Email: "ana@sabor.nl"},
	}}
}
