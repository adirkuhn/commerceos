package oms

import (
	"context"
	"fmt"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type GetSalesOrderResult struct {
	SalesOrder omsDomain.SalesOrder
	LineItems  []omsDomain.SalesOrderLineItem
}

type GetSalesOrderUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
}

func NewGetSalesOrderUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository) *GetSalesOrderUseCase {
	return &GetSalesOrderUseCase{db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo}
}

func (uc *GetSalesOrderUseCase) Execute(ctx context.Context, id string) (*GetSalesOrderResult, error) {
	var result GetSalesOrderResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		order, err := uc.salesOrderRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		lineItems, err := uc.lineItemRepo.ListBySalesOrder(ctx, id)
		if err != nil {
			return err
		}
		result = GetSalesOrderResult{SalesOrder: *order, LineItems: lineItems}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getSalesOrder: %w", err)
	}
	return &result, nil
}
