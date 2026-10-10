-- +goose Up

-- text_body: the WhatsApp/SMS text ('' = use body, as before).
-- design: the visual email editor's project JSON that body was built from.
-- default_key: the built-in default a template was created from, so it can
-- be reset to it ('' = none). See domain/messagecenter.MessageTemplate.
ALTER TABLE message_templates
    ADD COLUMN text_body TEXT NOT NULL DEFAULT '',
    ADD COLUMN design JSONB,
    ADD COLUMN default_key TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE message_templates
    DROP COLUMN default_key,
    DROP COLUMN design,
    DROP COLUMN text_body;
