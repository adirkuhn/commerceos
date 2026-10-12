-- +goose Up
-- "Mark as aware" on the close-to-expiry screen (domain/inventory.
-- ExpiryAcknowledgement): someone has seen this batch while it was in the
-- window_days close-to-expiry window, so that window's list and email
-- report skip it. One row per batch and window; the next, smaller window
-- has no row and reports the batch again.
-- commerceos_app gets its grants from 020's ALTER DEFAULT PRIVILEGES.
CREATE TABLE batch_expiry_acknowledgements (
    tenant_id       UUID        NOT NULL,
    batch_id        UUID        NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
    window_days     INTEGER     NOT NULL,
    acknowledged_by TEXT        NOT NULL,
    acknowledged_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (batch_id, window_days)
);

CREATE INDEX batch_expiry_acknowledgements_tenant_idx ON batch_expiry_acknowledgements (tenant_id);

ALTER TABLE batch_expiry_acknowledgements ENABLE ROW LEVEL SECURITY;
ALTER TABLE batch_expiry_acknowledgements FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON batch_expiry_acknowledgements
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE IF EXISTS batch_expiry_acknowledgements;
