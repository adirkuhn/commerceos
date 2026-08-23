-- +goose Up

-- Immutable, effective-dated audit trail of a SKU's regular/sale price --
-- see domain/pms.SKUPriceHistory. A row is written only when price or sale
-- price actually changes (CreateSKUUseCase, UpdateSKUUseCase,
-- ExpireSKUSalePriceUseCase); valid_until is filled in retroactively when
-- superseded, never edited otherwise. NULL valid_until means "currently in
-- effect".
CREATE TABLE sku_price_history (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID        NOT NULL,
    sku_id                 UUID        NOT NULL REFERENCES skus(id) ON DELETE CASCADE,
    price_minor_units      BIGINT      NOT NULL,
    sale_price_minor_units BIGINT,
    currency_code          TEXT        NOT NULL,
    valid_from             TIMESTAMPTZ NOT NULL,
    valid_until            TIMESTAMPTZ,
    reason                 TEXT        NOT NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sku_price_history_tenant_id_sku_id_idx ON sku_price_history (tenant_id, sku_id, valid_from DESC);
-- Speeds up CloseOpen's "find the currently open interval for this SKU"
-- update -- at most one row per SKU can match (open intervals are always
-- closed before a new one opens), so a plain (non-unique) partial index is
-- enough.
CREATE INDEX sku_price_history_open_interval_idx ON sku_price_history (sku_id) WHERE valid_until IS NULL;

ALTER TABLE sku_price_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE sku_price_history FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sku_price_history
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE sku_price_history;
