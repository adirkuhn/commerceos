-- +goose Up
-- Raw merchant-authored CSS injected as-is into the storefront-frontend's
-- page (see domain/theme.ThemePage's doc comment) -- same trust boundary
-- as a custom_html block's raw markup, just page-scoped instead of
-- block-scoped.
ALTER TABLE theme_pages ADD COLUMN custom_css TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE theme_pages DROP COLUMN custom_css;
