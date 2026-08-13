-- +goose Up
-- A POS desktop integration is bootstrapped by presenting an lm2-signed
-- license (RS256, verified against lm2's public key) -- see
-- infrastructure/lm2license. Bootstrap alone only creates a 'pending' row;
-- a human (customer_admin/customer_owner) must explicitly approve it before
-- an API key is minted, so a leaked license can't silently grant API
-- access -- see application/pos's doc comments for the full flow.
CREATE TABLE pos_connections (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID        NOT NULL,
    device_id           TEXT        NOT NULL,
    status              TEXT        NOT NULL DEFAULT 'pending',
    -- SHA-256 hex digest of the minted API key -- the raw key is never
    -- stored, only ever shown once to the POS via the poll-status response.
    api_key_hash        TEXT,
    license_id          TEXT        NOT NULL,
    license_fingerprint TEXT        NOT NULL DEFAULT '',
    -- Flips true once the POS has confirmed (via a separate ack call) that
    -- it received and stored the raw API key -- after that, the poll-status
    -- response never includes the raw key again, even if re-approved.
    key_acknowledged    BOOLEAN     NOT NULL DEFAULT FALSE,
    requested_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_at         TIMESTAMPTZ,
    approved_by         TEXT        NOT NULL DEFAULT '',
    revoked_at          TIMESTAMPTZ,
    last_used_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT pos_connections_status_check
        CHECK (status IN ('pending', 'active', 'revoked'))
);

CREATE INDEX pos_connections_tenant_id_idx ON pos_connections (tenant_id);
-- Looked up on every POS-authenticated request -- must be fast and must
-- only ever match an active connection's hash.
CREATE INDEX pos_connections_api_key_hash_idx ON pos_connections (api_key_hash) WHERE status = 'active';

ALTER TABLE pos_connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE pos_connections FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON pos_connections
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE pos_connections;
