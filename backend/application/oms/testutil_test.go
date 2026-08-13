package oms_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func newMockGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.MatchExpectationsInOrder(false)
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	return gormDB
}

type stubSalesOrderRepo struct {
	findByExternalIDResult *omsDomain.SalesOrder
	findByExternalIDErr    error
	createErr              error
	created                []omsDomain.SalesOrder
}

func (s *stubSalesOrderRepo) Create(_ context.Context, o *omsDomain.SalesOrder) error {
	if s.createErr != nil {
		return s.createErr
	}
	o.ID = "sales-order-1"
	s.created = append(s.created, *o)
	return nil
}
func (s *stubSalesOrderRepo) FindByID(_ context.Context, _ string) (*omsDomain.SalesOrder, error) {
	return nil, omsDomain.ErrSalesOrderNotFound
}
func (s *stubSalesOrderRepo) FindByExternalID(_ context.Context, _ omsDomain.SalesOrderSource, _ string) (*omsDomain.SalesOrder, error) {
	return s.findByExternalIDResult, s.findByExternalIDErr
}

type stubSalesOrderLineItemRepo struct {
	createErr error
	created   []omsDomain.SalesOrderLineItem
}

func (s *stubSalesOrderLineItemRepo) Create(_ context.Context, li *omsDomain.SalesOrderLineItem) error {
	if s.createErr != nil {
		return s.createErr
	}
	li.ID = "sales-order-line-1"
	s.created = append(s.created, *li)
	return nil
}
func (s *stubSalesOrderLineItemRepo) ListBySalesOrder(_ context.Context, _ string) ([]omsDomain.SalesOrderLineItem, error) {
	return nil, nil
}

type stubRecordMovement struct {
	calls []appinventory.RecordStockMovementInput
	err   error
}

func (s *stubRecordMovement) Execute(_ context.Context, in appinventory.RecordStockMovementInput) (*invDomain.StockMovement, error) {
	s.calls = append(s.calls, in)
	if s.err != nil {
		return nil, s.err
	}
	return &invDomain.StockMovement{ID: "movement-1", SKUID: in.SKUID, Type: in.Type, QuantityDelta: in.QuantityDelta}, nil
}
