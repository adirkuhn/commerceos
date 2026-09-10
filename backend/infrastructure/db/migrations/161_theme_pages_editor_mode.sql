-- +goose Up
-- editor_mode selects which document a page actually renders from --
-- 'freeform' (the existing absolutely-positioned canvas, in `blocks`) or
-- 'puck' (a document authored by storefront-frontend's own embedded Puck
-- editor, in `puck_data`). Both columns are kept regardless of the active
-- mode -- switching back and forth never loses the other one's content.
-- See domain/theme.EditorMode's doc comment.
ALTER TABLE theme_pages ADD COLUMN editor_mode TEXT NOT NULL DEFAULT 'freeform';
ALTER TABLE theme_pages ADD COLUMN puck_data JSONB;
ALTER TABLE theme_pages ADD CONSTRAINT theme_pages_editor_mode_check CHECK (editor_mode IN ('freeform', 'puck'));

-- +goose Down
ALTER TABLE theme_pages DROP CONSTRAINT theme_pages_editor_mode_check;
ALTER TABLE theme_pages DROP COLUMN puck_data;
ALTER TABLE theme_pages DROP COLUMN editor_mode;
