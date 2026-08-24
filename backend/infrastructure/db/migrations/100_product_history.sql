-- +goose Up

-- Lightweight audit trail of name/status changes on a product or variant --
-- see domain/pms.ProductHistory. Shared by both entity types (rather than
-- two near-identical tables) since the shape is identical either way; no
-- FK on entity_id since it polymorphically references either products or
-- variants. A row is written only when the field actually changes
-- (UpdateProductUseCase, UpdateVariantUseCase).
CREATE TABLE product_history (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    entity_type TEXT        NOT NULL,
    entity_id   UUID        NOT NULL,
    field       TEXT        NOT NULL,
    old_value   TEXT        NOT NULL,
    new_value   TEXT        NOT NULL,
    changed_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX product_history_tenant_entity_idx ON product_history (tenant_id, entity_type, entity_id, changed_at DESC);

ALTER TABLE product_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_history FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON product_history
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE product_history;
