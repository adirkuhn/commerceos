package oms

import (
	"context"
	"fmt"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const (
	defaultSalesOrdersPerPage = 20
	maxSalesOrdersPerPage     = 200
)

type ListSalesOrdersInput struct {
	Page    int
	PerPage int
}

type ListSalesOrdersResult struct {
	SalesOrders []omsDomain.SalesOrder
	Total       int64
}

// ListSalesOrdersUseCase is the read side for staff viewing what came in
// from POS (or, later, any other channel) -- ReceivePOSSalesOrderUseCase is
// the only write path today, this never creates or mutates anything.
type ListSalesOrdersUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
}

func NewListSalesOrdersUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository) *ListSalesOrdersUseCase {
	return &ListSalesOrdersUseCase{db: db, salesOrderRepo: salesOrderRepo}
}

func (uc *ListSalesOrdersUseCase) Execute(ctx context.Context, in ListSalesOrdersInput) (*ListSalesOrdersResult, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	perPage := in.PerPage
	if perPage <= 0 {
		perPage = defaultSalesOrdersPerPage
	}
	if perPage > maxSalesOrdersPerPage {
		perPage = maxSalesOrdersPerPage
	}

	result := &ListSalesOrdersResult{}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		orders, total, err := uc.salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{Limit: perPage, Offset: (page - 1) * perPage})
		if err != nil {
			return err
		}
		result.SalesOrders = orders
		result.Total = total
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listSalesOrders: %w", err)
	}
	return result, nil
}
