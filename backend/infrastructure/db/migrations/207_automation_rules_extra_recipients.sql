-- +goose Up

-- Email addresses that always get a rule's email on top of whoever the
-- trigger resolves to. JSON array of bare addresses, validated in
-- domain/messagecenter.NormalizeExtraRecipients.
ALTER TABLE automation_rules ADD COLUMN extra_recipients JSONB NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE automation_rules DROP COLUMN extra_recipients;
