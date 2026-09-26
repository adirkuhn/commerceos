package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/adirkuhn/commerceos/internal/txctx"
	"gorm.io/gorm"
)

// Product-merge adapters (application/pms.SKUReferenceReassigner /
// ProductReferenceReassigner): each moves one module's references from a
// duplicate SKU/product to the existing one, on the merge's transaction.
// Plain UPDATEs -- every table here is tenant-scoped under RLS, so a
// statement can only ever touch the current tenant's rows.

func mergeTx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("merge: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func execAll(tx *gorm.DB, where string, stmts []string, args ...any) error {
	for _, stmt := range stmts {
		if err := tx.Exec(stmt, args...).Error; err != nil {
			return fmt.Errorf("%s: %s: %w", where, stmt, err)
		}
	}
	return nil
}

// InventorySKUReferences moves stock: a duplicate batch with the same
// warehouse, lot number and expiry as an active batch of the existing SKU
// folds into that batch (everything pointing at it moves, the duplicate
// batch is removed); every other batch, and all movements and discards,
// simply change SKU.
type InventorySKUReferences struct{}

func NewInventorySKUReferences() *InventorySKUReferences { return &InventorySKUReferences{} }

func (InventorySKUReferences) ReassignSKU(ctx context.Context, from, to string) error {
	tx, err := mergeTx(ctx)
	if err != nil {
		return err
	}
	const twins = `SELECT s.id AS source_id, t.id AS target_id FROM batches s
		JOIN batches t ON t.sku_id = @to AND t.warehouse_id = s.warehouse_id AND t.lot_number = s.lot_number
			AND t.expiry_date IS NOT DISTINCT FROM s.expiry_date AND t.status = 'active'
		WHERE s.sku_id = @from AND s.lot_number <> ''`
	var pairs []struct{ SourceID, TargetID string }
	if err := tx.Raw(twins, map[string]any{"from": from, "to": to}).Scan(&pairs).Error; err != nil {
		return fmt.Errorf("inventory merge: find twin batches: %w", err)
	}
	for _, p := range pairs {
		if err := execAll(tx, "inventory merge: fold batch", []string{
			`UPDATE stock_movements SET batch_id = ? WHERE batch_id = ?`,
			`UPDATE booking_scans SET batch_id = ? WHERE batch_id = ?`,
			`UPDATE discard_records SET batch_id = ? WHERE batch_id = ?`,
			`UPDATE sales_order_line_item_stock_movements SET batch_id = ? WHERE batch_id = ?`,
			`UPDATE promotion_skus SET batch_id = ? WHERE batch_id = ?`,
		}, p.TargetID, p.SourceID); err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM batches WHERE id = ?`, p.SourceID).Error; err != nil {
			return fmt.Errorf("inventory merge: remove folded batch: %w", err)
		}
	}
	return execAll(tx, "inventory merge", []string{
		`UPDATE batches SET sku_id = ? WHERE sku_id = ?`,
		`UPDATE stock_movements SET sku_id = ? WHERE sku_id = ?`,
		`UPDATE discard_records SET sku_id = ? WHERE sku_id = ?`,
	}, to, from)
}

// ProcurementSKUReferences moves purchase-order lines, receiving scans and
// supplier aliases (their uniqueness is per supplier+raw key, not per SKU,
// so moving one never collides).
type ProcurementSKUReferences struct{}

func NewProcurementSKUReferences() *ProcurementSKUReferences { return &ProcurementSKUReferences{} }

func (ProcurementSKUReferences) ReassignSKU(ctx context.Context, from, to string) error {
	tx, err := mergeTx(ctx)
	if err != nil {
		return err
	}
	return execAll(tx, "procurement merge", []string{
		`UPDATE purchase_order_line_items SET sku_id = ? WHERE sku_id = ?`,
		`UPDATE booking_scans SET sku_id = ? WHERE sku_id = ?`,
		`UPDATE supplier_product_aliases SET sku_id = ? WHERE sku_id = ?`,
	}, to, from)
}

// OrderSKUReferences moves sales order lines (they keep their own name /
// SKU code snapshot as sold) and picking scans.
type OrderSKUReferences struct{}

func NewOrderSKUReferences() *OrderSKUReferences { return &OrderSKUReferences{} }

func (OrderSKUReferences) ReassignSKU(ctx context.Context, from, to string) error {
	tx, err := mergeTx(ctx)
	if err != nil {
		return err
	}
	return execAll(tx, "orders merge", []string{
		`UPDATE sales_order_line_items SET sku_id = ? WHERE sku_id = ?`,
		`UPDATE pick_scans SET sku_id = ? WHERE sku_id = ?`,
	}, to, from)
}

// PromotionSKUReferences moves promotion lines; where the existing SKU is
// already in the same promotion, the duplicate's line is dropped (one line
// per SKU per promotion).
type PromotionSKUReferences struct{}

func NewPromotionSKUReferences() *PromotionSKUReferences { return &PromotionSKUReferences{} }

func (PromotionSKUReferences) ReassignSKU(ctx context.Context, from, to string) error {
	tx, err := mergeTx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Exec(`DELETE FROM promotion_skus s WHERE s.sku_id = ?
		AND EXISTS (SELECT 1 FROM promotion_skus t WHERE t.promotion_id = s.promotion_id AND t.sku_id = ?)`, from, to).Error; err != nil {
		return fmt.Errorf("promotions merge: drop duplicate lines: %w", err)
	}
	return execAll(tx, "promotions merge", []string{`UPDATE promotion_skus SET sku_id = ? WHERE sku_id = ?`}, to, from)
}

// CatalogSKUReferences moves barcode/scale-code aliases and re-points SKUs
// that are a case or a portion of the duplicate. A link that would make
// the existing SKU a case/portion of itself is cleared instead.
type CatalogSKUReferences struct{}

func NewCatalogSKUReferences() *CatalogSKUReferences { return &CatalogSKUReferences{} }

func (CatalogSKUReferences) ReassignSKU(ctx context.Context, from, to string) error {
	tx, err := mergeTx(ctx)
	if err != nil {
		return err
	}
	if err := execAll(tx, "catalog merge: self links", []string{
		`UPDATE skus SET case_of_sku_id = NULL, units_per_case = NULL WHERE id = ? AND case_of_sku_id = ?`,
		`UPDATE skus SET portioned_from_sku_id = NULL WHERE id = ? AND portioned_from_sku_id = ?`,
	}, to, from); err != nil {
		return err
	}
	return execAll(tx, "catalog merge", []string{
		`UPDATE skus SET case_of_sku_id = ? WHERE case_of_sku_id = ?`,
		`UPDATE skus SET portioned_from_sku_id = ? WHERE portioned_from_sku_id = ?`,
		`UPDATE barcode_aliases SET sku_id = ? WHERE sku_id = ?`,
		`UPDATE scale_code_aliases SET sku_id = ? WHERE sku_id = ?`,
	}, to, from)
}

// CatalogProductReferences moves product-level data: images (unlinked from
// the duplicate's variants, which are deleted) and reviews join the
// existing product, wishlists point at it; the duplicate's Bol content and
// overrides are dropped -- the existing product's own stand.
type CatalogProductReferences struct{}

func NewCatalogProductReferences() *CatalogProductReferences { return &CatalogProductReferences{} }

func (CatalogProductReferences) ReassignProduct(ctx context.Context, from, to string) error {
	tx, err := mergeTx(ctx)
	if err != nil {
		return err
	}
	if err := execAll(tx, "catalog merge: product", []string{
		`UPDATE media SET product_id = ?, variant_id = NULL WHERE product_id = ?`,
		`UPDATE reviews SET product_id = ? WHERE product_id = ?`,
	}, to, from); err != nil {
		return err
	}
	// product_ids is a JSONB array of id strings; the duplicate's id is
	// replaced (deduplicated) by the existing product's.
	if err := tx.Exec(`UPDATE storefront_wishlists
		SET product_ids = (SELECT jsonb_agg(DISTINCT CASE WHEN e.pid = @from THEN @to ELSE e.pid END)
			FROM jsonb_array_elements_text(product_ids) AS e(pid))
		WHERE product_ids @> jsonb_build_array(CAST(@from AS text))`, map[string]any{"from": from, "to": to}).Error; err != nil {
		return fmt.Errorf("catalog merge: wishlists: %w", err)
	}
	return execAll(tx, "catalog merge: bol", []string{
		`DELETE FROM bol_product_content WHERE product_id = ?`,
		`DELETE FROM bol_product_overrides WHERE product_id = ?`,
	}, from)
}
