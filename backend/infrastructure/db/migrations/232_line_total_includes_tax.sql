-- +goose Up
-- Whether a line's total_minor_units already contains its VAT
-- (total_tax_minor_units) or has it on top. Woo and the storefront store
-- totals without VAT; shop-priced lines store the price as entered.
ALTER TABLE sales_order_line_items ADD COLUMN total_includes_tax BOOLEAN NOT NULL DEFAULT false;

-- Manual (incl. converted quotes) and WhatsApp lines carry the shop's
-- prices: tax-inclusive per the tenant's setting (true when never saved).
UPDATE sales_order_line_items li
SET total_includes_tax = COALESCE(ts.prices_include_tax, true)
FROM sales_orders so
LEFT JOIN tenant_settings ts ON ts.tenant_id = so.tenant_id
WHERE so.id = li.sales_order_id AND so.source IN ('manual', 'whatsapp');

-- POS and Bol send totals with VAT.
UPDATE sales_order_line_items li
SET total_includes_tax = true
FROM sales_orders so
WHERE so.id = li.sales_order_id AND so.source IN ('pos', 'bol');

-- Items added while picking/packing were stored at the shop price with no
-- VAT split: compute it from the SKU's tax class in the tenant's country.
CREATE TEMP TABLE added_line_tax ON COMMIT DROP AS
SELECT x.id, x.sales_order_id, x.inclusive,
       CASE WHEN x.rate <= 0 THEN 0
            WHEN x.inclusive THEN x.total - (2 * x.total * 10000 + (10000 + x.rate)) / (2 * (10000 + x.rate))
            ELSE (2 * x.total * x.rate + 10000) / 20000
       END AS tax
FROM (
    SELECT li.id, li.sales_order_id, li.total_minor_units AS total,
           COALESCE(ts.prices_include_tax, true) AS inclusive,
           COALESCE(tr.rate_basis_points, 0)::BIGINT AS rate
    FROM sales_order_line_items li
    LEFT JOIN tenant_settings ts ON ts.tenant_id = li.tenant_id
    LEFT JOIN LATERAL (
        SELECT r.rate_basis_points FROM tax_rates r
        WHERE r.tenant_id = li.tenant_id
          AND lower(r.country) = lower(COALESCE(ts.default_tax_country, ''))
          AND r.tax_class_id = COALESCE(li.tax_class_id,
              (SELECT c.id FROM tax_classes c WHERE c.tenant_id = li.tenant_id AND c.is_default LIMIT 1))
        LIMIT 1
    ) tr ON true
    WHERE li.total_tax_minor_units = 0 AND li.total_minor_units > 0
      AND (EXISTS (SELECT 1 FROM pick_scans s JOIN picking_sessions ps ON ps.id = s.session_id
                   WHERE ps.sales_order_id = li.sales_order_id AND s.sku_id = li.sku_id AND s.quantity = li.quantity
                     AND s.action_type IN ('freebie', 'new_item') AND s.replayed_at IS NOT NULL)
        OR EXISTS (SELECT 1 FROM pack_scans s JOIN packing_sessions ps ON ps.id = s.session_id
                   WHERE ps.sales_order_id = li.sales_order_id AND s.sku_id = li.sku_id AND s.quantity = li.quantity
                     AND s.action_type IN ('freebie', 'new_item') AND s.replayed_at IS NOT NULL))
) x;

UPDATE sales_order_line_items li
SET total_includes_tax = a.inclusive, total_tax_minor_units = a.tax, subtotal_tax_minor_units = a.tax
FROM added_line_tax a
WHERE a.id = li.id;

-- The order's VAT now counts them; a tax-exclusive shop's total gets it on top.
UPDATE sales_orders so
SET tax_total_minor_units = so.tax_total_minor_units + t.tax,
    total_minor_units = so.total_minor_units + t.tax_on_top
FROM (
    SELECT sales_order_id, SUM(tax) AS tax, SUM(CASE WHEN inclusive THEN 0 ELSE tax END) AS tax_on_top
    FROM added_line_tax GROUP BY sales_order_id
) t
WHERE t.sales_order_id = so.id;

-- +goose Down
-- The VAT computed for added items stays: it was missing, not wrong.
ALTER TABLE sales_order_line_items DROP COLUMN IF EXISTS total_includes_tax;
