-- +goose Up

-- Lets product search tolerate typos ("Wigdet" finding "Widget") via
-- trigram similarity, on top of the exact-substring unaccent()/ILIKE match
-- from 022_unaccent_extension.sql. See ProductRepo.List in
-- infrastructure/repositories/pms_repo.go.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- unaccent() is STABLE, not IMMUTABLE (its behavior technically depends on
-- the configured text search dictionary), so Postgres refuses to use it
-- directly in an index expression. Wrap it in a same-behavior function
-- explicitly marked IMMUTABLE -- safe here since this app never changes the
-- dictionary at runtime. This is the standard documented workaround for
-- indexing unaccent() output.
CREATE OR REPLACE FUNCTION immutable_unaccent(text) RETURNS text AS $$
  SELECT unaccent('unaccent', $1)
$$ LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT;

CREATE INDEX idx_products_name_trgm ON products USING gin (immutable_unaccent(name) gin_trgm_ops);

-- +goose Down
DROP INDEX IF EXISTS idx_products_name_trgm;
DROP FUNCTION IF EXISTS immutable_unaccent(text);
DROP EXTENSION IF EXISTS pg_trgm;
