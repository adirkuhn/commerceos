-- +goose Up

CREATE TABLE sync_schedules (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    connection_id     UUID        NOT NULL REFERENCES integration_connections(id) ON DELETE CASCADE,
    entity_type       TEXT        NOT NULL,
    direction         TEXT        NOT NULL,
    enabled           BOOLEAN     NOT NULL DEFAULT false,
    interval_minutes  INT         NOT NULL,
    default_warehouse_id UUID,
    default_currency  TEXT,
    last_run_at       TIMESTAMPTZ,
    last_run_status   TEXT        NOT NULL DEFAULT '',
    last_run_error    TEXT        NOT NULL DEFAULT '',
    next_run_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sync_schedules_entity_type_check CHECK (entity_type IN ('products', 'stock', 'orders', 'categories')),
    CONSTRAINT sync_schedules_direction_check CHECK (direction IN ('import', 'export')),
    UNIQUE (connection_id, entity_type, direction)
);

CREATE INDEX sync_schedules_next_run_idx ON sync_schedules (next_run_at) WHERE enabled;

ALTER TABLE sync_schedules ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON sync_schedules
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

-- +goose Down

DROP TABLE IF EXISTS sync_schedules;
