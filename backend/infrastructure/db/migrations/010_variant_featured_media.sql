-- +goose Up

ALTER TABLE variants ADD COLUMN featured_media_id UUID NULL REFERENCES media(id) ON DELETE SET NULL;

-- +goose Down

ALTER TABLE variants DROP COLUMN featured_media_id;
