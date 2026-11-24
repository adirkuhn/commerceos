-- +goose Up
-- Settings > Appearance now picks a default look (wallet or graphite) instead
-- of a font and two colours. Backward compatible with the code still deployed:
-- the old columns stay (old code still writes them) and get defaults so the
-- new code can leave them out. A later migration drops them.
ALTER TABLE tenant_appearance ADD COLUMN default_look TEXT NOT NULL DEFAULT 'wallet';
ALTER TABLE tenant_appearance
    ALTER COLUMN font SET DEFAULT 'atkinson',
    ALTER COLUMN accent_color SET DEFAULT '#2F33B8',
    ALTER COLUMN sidebar_color SET DEFAULT '#14163F';

-- +goose Down
ALTER TABLE tenant_appearance
    ALTER COLUMN font DROP DEFAULT,
    ALTER COLUMN accent_color DROP DEFAULT,
    ALTER COLUMN sidebar_color DROP DEFAULT;
ALTER TABLE tenant_appearance DROP COLUMN IF EXISTS default_look;
