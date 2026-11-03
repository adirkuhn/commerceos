package integrations

import (
	"context"
	"fmt"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ListSalesOrderOriginsUseCase reads the origins of the orders an API
// response shows (COM-15): the order holds no integration data, the
// response still carries its source, external order id and order key.
// Read only.
type ListSalesOrderOriginsUseCase struct {
	db   *gorm.DB
	repo integrationsDomain.SalesOrderOriginRepository
}

func NewListSalesOrderOriginsUseCase(db *gorm.DB, repo integrationsDomain.SalesOrderOriginRepository) *ListSalesOrderOriginsUseCase {
	return &ListSalesOrderOriginsUseCase{db: db, repo: repo}
}

// Execute returns the origins by sales order id.
func (uc *ListSalesOrderOriginsUseCase) Execute(ctx context.Context, salesOrderIDs []string) (map[string]integrationsDomain.SalesOrderOrigin, error) {
	out := map[string]integrationsDomain.SalesOrderOrigin{}
	if len(salesOrderIDs) == 0 {
		return out, nil
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		origins, err := uc.repo.ListBySalesOrderIDs(ctx, salesOrderIDs)
		for _, o := range origins {
			out[o.SalesOrderID] = o
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listSalesOrderOrigins: %w", err)
	}
	return out, nil
}
