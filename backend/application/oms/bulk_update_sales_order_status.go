package oms

import (
	"context"
	"fmt"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// BulkUpdateSalesOrderStatusInput applies the same Status and/or Fulfilled
// change to every listed order -- the Sales Orders page's bulk action bar.
// At least one of OrderIDs and one of Status/Fulfilled is required.
type BulkUpdateSalesOrderStatusInput struct {
	OrderIDs  []string
	Status    *omsDomain.SalesOrderStatus
	Fulfilled *bool
}

// BulkUpdateSalesOrderStatusFailure is one order that failed to update --
// e.g. an ID that no longer exists by the time the batch runs.
type BulkUpdateSalesOrderStatusFailure struct {
	OrderID string
	Error   string
}

type BulkUpdateSalesOrderStatusResult struct {
	Updated []string
	Failed  []BulkUpdateSalesOrderStatusFailure
}

// BulkUpdateSalesOrderStatusUseCase loops UpdateSalesOrderUseCase over a
// selection of orders, rather than reimplementing its status-transition
// logic (stock decrement/restock, the StockDecremented atomic swap,
// NeedsStockReview, the completed-implies-fulfilled rule -- see
// UpdateSalesOrderUseCase's doc). Each order update is its own transaction,
// same as if a human clicked "update" once per order: a real
// tenant.WithTenant around the whole batch isn't possible here anyway,
// since UpdateSalesOrderUseCase's own stock-movement phase already runs
// as separate transactions after its header-update transaction commits
// (its "two-phase" composition, see its doc) -- nesting that inside one
// more outer transaction would violate the no-nested-transactions rule
// every other multi-call composition in this package already follows
// (e.g. listLineItemMovements's doc). A failure partway through the batch
// is reported per-order in Failed rather than rolling back orders already
// applied; every ID is still attempted regardless of an earlier failure.
type BulkUpdateSalesOrderStatusUseCase struct {
	updateSalesOrder updateSalesOrderExecutor
}

// updateSalesOrderExecutor is the single-order use case's own Execute
// signature, defined here (ISP, point of use) so this package doesn't
// depend on UpdateSalesOrderUseCase's concrete type.
type updateSalesOrderExecutor interface {
	Execute(ctx context.Context, id string, in UpdateSalesOrderInput) (*omsDomain.SalesOrder, error)
}

func NewBulkUpdateSalesOrderStatusUseCase(updateSalesOrder updateSalesOrderExecutor) *BulkUpdateSalesOrderStatusUseCase {
	return &BulkUpdateSalesOrderStatusUseCase{updateSalesOrder: updateSalesOrder}
}

func (uc *BulkUpdateSalesOrderStatusUseCase) Execute(ctx context.Context, in BulkUpdateSalesOrderStatusInput) (*BulkUpdateSalesOrderStatusResult, error) {
	if len(in.OrderIDs) == 0 {
		return nil, fmt.Errorf("bulkUpdateSalesOrderStatus: %w", omsDomain.ErrBulkUpdateOrderIDsRequired)
	}
	if in.Status == nil && in.Fulfilled == nil {
		return nil, fmt.Errorf("bulkUpdateSalesOrderStatus: %w", omsDomain.ErrBulkUpdateNoFieldsGiven)
	}

	result := &BulkUpdateSalesOrderStatusResult{}
	for _, id := range in.OrderIDs {
		_, err := uc.updateSalesOrder.Execute(ctx, id, UpdateSalesOrderInput{Status: in.Status, Fulfilled: in.Fulfilled})
		if err != nil {
			result.Failed = append(result.Failed, BulkUpdateSalesOrderStatusFailure{OrderID: id, Error: err.Error()})
			continue
		}
		result.Updated = append(result.Updated, id)
	}
	return result, nil
}
