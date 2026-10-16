-- +goose Up
-- Claim status of a purchase order's supplier issues (domain/procurement.
-- SupplierIssueState). The issues themselves -- missing, damaged and lost
-- items -- are derived on demand from bookings and receiving flags, so a row
-- only exists once someone claims or resolves one; no row means open.
-- source + source_id identify the issue: the short PO line item
-- (booking_shortfall), the damaged/lost booking scan (booking_scan) or the
-- damaged receiving flag (receiving_discrepancy).
-- commerceos_app gets its grants from 020's ALTER DEFAULT PRIVILEGES.
CREATE TABLE purchase_order_supplier_issues (
    tenant_id         UUID        NOT NULL,
    purchase_order_id UUID        NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    source            TEXT        NOT NULL CHECK (source IN ('booking_shortfall', 'booking_scan', 'receiving_discrepancy')),
    source_id         UUID        NOT NULL,
    status            TEXT        NOT NULL CHECK (status IN ('open', 'claimed', 'resolved')),
    note              TEXT        NOT NULL DEFAULT '',
    updated_by        TEXT        NOT NULL DEFAULT '',
    updated_at        TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, source, source_id)
);

CREATE INDEX purchase_order_supplier_issues_po_idx ON purchase_order_supplier_issues (purchase_order_id);

ALTER TABLE purchase_order_supplier_issues ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_order_supplier_issues FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON purchase_order_supplier_issues
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- Emails sent to a supplier from the issues list show in Sent emails.
ALTER TABLE sent_emails DROP CONSTRAINT sent_emails_source_check;
ALTER TABLE sent_emails ADD CONSTRAINT sent_emails_source_check
    CHECK (source IN ('automation', 'notification', 'sign_in_link', 'newsletter', 'supplier_issues'));

-- +goose Down
DELETE FROM sent_emails WHERE source = 'supplier_issues';
ALTER TABLE sent_emails DROP CONSTRAINT sent_emails_source_check;
ALTER TABLE sent_emails ADD CONSTRAINT sent_emails_source_check
    CHECK (source IN ('automation', 'notification', 'sign_in_link', 'newsletter'));
DROP TABLE IF EXISTS purchase_order_supplier_issues;
