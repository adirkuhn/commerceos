-- +goose Up

-- Lets product search match "acai" against "Açaí", "feijao" against
-- "Feijão", etc. — plain ILIKE is case-insensitive but accent-sensitive,
-- so it never matched across the accented/unaccented spelling of the same
-- word. See ProductRepo.List in infrastructure/repositories/pms_repo.go.
CREATE EXTENSION IF NOT EXISTS unaccent;

-- +goose Down
DROP EXTENSION IF EXISTS unaccent;
