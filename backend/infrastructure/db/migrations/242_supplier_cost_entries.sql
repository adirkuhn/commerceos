-- +goose Up
-- COM-11: a SKU's cost history by supplier -- one append-only row per
-- received purchase order line (plus reversal/correction rows -- history is
-- never edited). Written at goods receipt from procurement.goods.received,
-- the cost price follows the last receipt, the averages are computed from
-- these rows (domain/procurement.SummarizeSupplierCosts).
CREATE TABLE supplier_cost_entries (
    id                           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                    UUID        NOT NULL,
    sku_id                       UUID        NOT NULL REFERENCES skus(id) ON DELETE CASCADE,
    supplier_id                  UUID        REFERENCES suppliers(id) ON DELETE SET NULL,
    purchase_order_id            UUID        NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    purchase_order_line_item_id  UUID        NOT NULL REFERENCES purchase_order_line_items(id) ON DELETE CASCADE,
    batch_id                     UUID        REFERENCES batches(id) ON DELETE SET NULL,
    kind                         TEXT        NOT NULL CHECK (kind IN ('receipt', 'reversal', 'correction')),
    quantity                     BIGINT      NOT NULL,
    unit_cost_minor_units        BIGINT      NOT NULL,
    landed_unit_cost_minor_units BIGINT      NOT NULL,
    currency_code                TEXT        NOT NULL,
    received_at                  TIMESTAMPTZ NOT NULL,
    source_key                   TEXT        NOT NULL,
    created_at                   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX supplier_cost_entries_source_key_idx ON supplier_cost_entries (tenant_id, source_key);
CREATE INDEX supplier_cost_entries_sku_idx ON supplier_cost_entries (tenant_id, sku_id, received_at);
CREATE INDEX supplier_cost_entries_line_idx ON supplier_cost_entries (purchase_order_line_item_id);

ALTER TABLE supplier_cost_entries ENABLE ROW LEVEL SECURITY;
ALTER TABLE supplier_cost_entries FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON supplier_cost_entries
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- Backfill 1: every past receipt into a batch booked against a PO line with
-- a landed cost -- one row per receipt movement. Batches without a landed
-- cost can't be costed and are left out.
INSERT INTO supplier_cost_entries (tenant_id, sku_id, supplier_id, purchase_order_id, purchase_order_line_item_id,
    batch_id, kind, quantity, unit_cost_minor_units, landed_unit_cost_minor_units, currency_code, received_at, source_key)
SELECT b.tenant_id, b.sku_id, po.supplier_id, po.id, li.id,
    b.id, 'receipt', m.quantity_delta, li.unit_cost_minor_units, b.landed_cost_minor_units, po.currency_code, m.occurred_at,
    'backfill:movement:' || m.id
FROM stock_movements m
JOIN batches b ON b.id = m.batch_id
JOIN purchase_order_line_items li ON li.id = b.purchase_order_line_item_id
JOIN purchase_orders po ON po.id = li.purchase_order_id
WHERE m.type = 'receipt' AND m.quantity_delta > 0 AND b.landed_cost_minor_units IS NOT NULL
ON CONFLICT (tenant_id, source_key) DO NOTHING;

-- Backfill 2: product lines of received / partially received POs that no
-- batch accounts for (marked received without booking stock). Landed cost
-- mirrors application/procurement.AllocateLandedUnitCost: the line's price
-- plus its value share of the PO's shipping/fee/other-charge lines,
-- rounded half up in integer math (sums cast back to bigint: SUM(bigint)
-- is numeric). Quantity mirrors ListReceivedCostLinesForSKUUseCase.
INSERT INTO supplier_cost_entries (tenant_id, sku_id, supplier_id, purchase_order_id, purchase_order_line_item_id,
    kind, quantity, unit_cost_minor_units, landed_unit_cost_minor_units, currency_code, received_at, source_key)
SELECT li.tenant_id, li.sku_id, po.supplier_id, po.id, li.id,
    'receipt', li.quantity * GREATEST(li.units_per_purchased_item, 1), li.unit_cost_minor_units,
    li.unit_cost_minor_units + CASE
        WHEN t.product_sum > 0 AND li.quantity > 0 AND t.charges <> 0 THEN
            ((li.unit_cost_minor_units * li.quantity * t.charges + t.product_sum / 2) / t.product_sum + li.quantity / 2) / li.quantity
        ELSE 0 END,
    po.currency_code, po.updated_at, 'backfill:line:' || li.id
FROM purchase_order_line_items li
JOIN purchase_orders po ON po.id = li.purchase_order_id
JOIN (
    SELECT purchase_order_id,
        COALESCE(SUM(unit_cost_minor_units * quantity) FILTER (WHERE line_type IN ('product', '')), 0)::bigint AS product_sum,
        COALESCE(SUM(unit_cost_minor_units * quantity) FILTER (WHERE line_type IN ('shipping', 'fee', 'other_charge')), 0)::bigint AS charges
    FROM purchase_order_line_items GROUP BY purchase_order_id
) t ON t.purchase_order_id = po.id
WHERE li.sku_id IS NOT NULL AND li.line_type IN ('product', '') AND li.quantity > 0
  AND po.status IN ('received', 'partially_received')
  AND NOT EXISTS (SELECT 1 FROM batches b WHERE b.purchase_order_line_item_id = li.id)
ON CONFLICT (tenant_id, source_key) DO NOTHING;

-- +goose Down
DROP TABLE supplier_cost_entries;
