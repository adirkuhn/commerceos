package fx_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
)

func newMockGormDB(t *testing.T, n int) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.MatchExpectationsInOrder(false)
	for i := 0; i < n; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}
	return gormDB
}

type fakeExchangeRateRepo struct {
	byCurrency map[string]fxDomain.ExchangeRate
	upserted   *fxDomain.ExchangeRate
	deletedCcy string
	upsertErr  error
}

func (f *fakeExchangeRateRepo) FindByCurrency(_ context.Context, currencyCode string) (*fxDomain.ExchangeRate, error) {
	if r, ok := f.byCurrency[currencyCode]; ok {
		cp := r
		return &cp, nil
	}
	return nil, nil
}

func (f *fakeExchangeRateRepo) ListByTenant(_ context.Context) ([]fxDomain.ExchangeRate, error) {
	out := make([]fxDomain.ExchangeRate, 0, len(f.byCurrency))
	for _, r := range f.byCurrency {
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeExchangeRateRepo) Upsert(_ context.Context, r *fxDomain.ExchangeRate) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	r.ID = "rate-1"
	f.upserted = r
	return nil
}

func (f *fakeExchangeRateRepo) Delete(_ context.Context, currencyCode string) error {
	f.deletedCcy = currencyCode
	return nil
}
