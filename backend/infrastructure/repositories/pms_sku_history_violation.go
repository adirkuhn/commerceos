package repositories

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// skuHistoryViolation maps a delete blocked by one of the ON DELETE
// RESTRICT foreign keys that keep a SKU's history (receiving, sales,
// picking, discards) to its typed pms error -- deleting a product, a
// variant or a SKU cascades down to skus, so all three hit the same
// constraints. Returns nil for any other error.
func skuHistoryViolation(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}
	switch pgErr.ConstraintName {
	case "booking_scans_sku_id_fkey":
		return pmsDomain.ErrProductHasReceivingHistory
	case "sales_order_line_items_sku_id_fkey":
		return pmsDomain.ErrProductHasSalesOrderHistory
	case "pick_scans_sku_id_fkey":
		return pmsDomain.ErrProductHasPickingHistory
	case "discard_records_sku_id_fkey":
		return pmsDomain.ErrProductHasDiscardHistory
	}
	return nil
}
