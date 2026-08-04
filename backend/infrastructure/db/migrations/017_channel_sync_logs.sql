-- +goose Up

-- Immutable audit trail of every sync attempt between CommerceOS and a
-- channel. product_channel_listings holds only the *current* state
-- (SyncStatus/LastSyncedAt/LastSyncError) -- this is the history of how it
-- got there: what was synced, when, in which direction, and what changed
-- since the previous attempt for the same listing.
CREATE TABLE channel_sync_logs (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL,
    listing_id    UUID        NOT NULL REFERENCES product_channel_listings(id) ON DELETE CASCADE,
    product_id    UUID        NOT NULL,
    connection_id UUID        NOT NULL,
    provider      TEXT        NOT NULL,
    external_id   TEXT        NOT NULL,
    direction     TEXT        NOT NULL,
    status        TEXT        NOT NULL,
    error_message TEXT        NOT NULL DEFAULT '',
    fields        JSONB       NOT NULL DEFAULT '{}',
    changes       JSONB       NOT NULL DEFAULT '{}',
    synced_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT channel_sync_logs_direction_check CHECK (direction IN ('push', 'import')),
    CONSTRAINT channel_sync_logs_status_check CHECK (status IN ('success', 'error'))
);

CREATE INDEX channel_sync_logs_listing_id_idx ON channel_sync_logs (listing_id, synced_at DESC);

ALTER TABLE channel_sync_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON channel_sync_logs
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

-- +goose Down
DROP TABLE channel_sync_logs;
