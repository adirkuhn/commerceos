-- +goose Up
ALTER TABLE products ADD COLUMN tags JSONB NOT NULL DEFAULT '[]';

CREATE INDEX products_tags_idx ON products USING GIN (tags);

-- +goose Down
DROP INDEX products_tags_idx;
ALTER TABLE products DROP COLUMN tags;
