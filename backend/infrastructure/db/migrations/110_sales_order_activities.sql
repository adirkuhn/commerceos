-- +goose Up

-- Structured, append-only activity log for a sales order -- replaces
-- ad-hoc use of sales_orders.customer_note (a single freeform,
-- overwritable field meant for the customer's own note) for anything
-- system-recorded. One row per event (MyParcel order export, shipment
-- creation, label print, and their failures), tied to both the order and
-- the acting user so it can be listed from either side: the order detail
-- page, or that user's own activity (see the iam users/{user_id}/activities
-- endpoint). user_id is the external lm2 identity -- no local users table
-- to FK against, same convention as user_permissions/iam_user_groups (see
-- 074_unify_user_id_uuid.sql).
CREATE TABLE sales_order_activities (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID        NOT NULL,
    sales_order_id UUID        NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
    user_id        UUID        NOT NULL,
    activity_type  TEXT        NOT NULL,
    summary        TEXT        NOT NULL,
    metadata       JSONB       NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX sales_order_activities_order_idx ON sales_order_activities (sales_order_id, created_at DESC);
CREATE INDEX sales_order_activities_user_idx ON sales_order_activities (tenant_id, user_id, created_at DESC);

ALTER TABLE sales_order_activities ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_order_activities FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON sales_order_activities
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE sales_order_activities;
