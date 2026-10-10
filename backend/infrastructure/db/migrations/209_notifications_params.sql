-- +goose Up

-- Values a notification's translated text is filled with (see
-- domain/notifications.Type.Render), so the bell and email show it in the
-- reader's language. Older rows keep {} and show their stored English text.
ALTER TABLE notifications ADD COLUMN params JSONB NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE notifications DROP COLUMN params;
