-- +goose Up
-- Telling Bol that a Bol order shipped (see
-- application/channels.ConfirmBolShipmentUseCase). Bol accepts the request
-- at once but processes it later, so each attempt is kept with Bol's
-- process_status_id and resolved by the check_bol_shipment_confirmations
-- job (application/channels.CheckBolShipmentConfirmationsUseCase) -- the
-- order page shows the result, so a shipment Bol refused is never silent.
CREATE TABLE bol_shipment_confirmations (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    sales_order_id    UUID        NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
    connection_id     UUID        NOT NULL,
    shipment_id       UUID        NOT NULL,
    external_order_id TEXT        NOT NULL,
    -- Empty when the request never reached Bol (stored already failed).
    process_status_id TEXT        NOT NULL DEFAULT '',
    transporter_code  TEXT        NOT NULL DEFAULT '',
    track_and_trace   TEXT        NOT NULL DEFAULT '',
    status            TEXT        NOT NULL CHECK (status IN ('pending', 'success', 'failure', 'timeout')),
    error_message     TEXT        NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at       TIMESTAMPTZ
);

CREATE INDEX bol_shipment_confirmations_order_idx ON bol_shipment_confirmations (tenant_id, sales_order_id, created_at DESC);
CREATE INDEX bol_shipment_confirmations_pending_idx ON bol_shipment_confirmations (created_at) WHERE status = 'pending';

ALTER TABLE bol_shipment_confirmations ENABLE ROW LEVEL SECURITY;
ALTER TABLE bol_shipment_confirmations FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON bol_shipment_confirmations
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- Global job: every 5 minutes, ask Bol for the result of each pending
-- confirmation. Cheap (only pending rows, bounded per run).
INSERT INTO scheduled_jobs (job_key, enabled, interval_minutes) VALUES
    ('check_bol_shipment_confirmations', true, 5);

-- +goose Down
DELETE FROM scheduled_jobs WHERE job_key = 'check_bol_shipment_confirmations';
DROP TABLE bol_shipment_confirmations;
