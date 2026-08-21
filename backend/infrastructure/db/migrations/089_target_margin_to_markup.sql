-- +goose Up

-- Switches the tenant-default and per-category target-profit setting from
-- MARGIN (profit / price, mathematically capped under 100%) to MARKUP
-- (profit / cost, no upper bound) -- a merchant who wants "300% profit"
-- means 300% markup (price = 4x cost), not 300% margin (impossible: margin
-- can never reach, let alone exceed, 100%). Renamed, not just
-- reinterpreted -- a same-named bps column silently changing what it means
-- is exactly the kind of bug this migration exists to prevent recurring.
--
-- Existing stored values are margin bps and must be converted, not just
-- relabeled: markup = margin / (1 - margin), i.e.
-- markup_bps = round(margin_bps * 10000 / (10000 - margin_bps)).
-- margin_bps is always < 10000 here (enforced by the CHECK constraints
-- this migration replaces), so the denominator is never zero/negative.

ALTER TABLE categories RENAME COLUMN target_margin_basis_points TO target_markup_basis_points;
ALTER TABLE tenant_settings RENAME COLUMN default_target_margin_basis_points TO default_target_markup_basis_points;

ALTER TABLE categories DROP CONSTRAINT categories_target_margin_basis_points_check;
ALTER TABLE tenant_settings DROP CONSTRAINT tenant_settings_default_target_margin_basis_points_check;

UPDATE categories SET target_markup_basis_points =
    ROUND((target_markup_basis_points::numeric * 10000) / (10000 - target_markup_basis_points))
    WHERE target_markup_basis_points IS NOT NULL AND target_markup_basis_points > 0;

UPDATE tenant_settings SET default_target_markup_basis_points =
    ROUND((default_target_markup_basis_points::numeric * 10000) / (10000 - default_target_markup_basis_points))
    WHERE default_target_markup_basis_points > 0;

ALTER TABLE categories
    ADD CONSTRAINT categories_target_markup_basis_points_check
        CHECK (target_markup_basis_points IS NULL OR target_markup_basis_points >= 0);
ALTER TABLE tenant_settings
    ADD CONSTRAINT tenant_settings_default_target_markup_basis_points_check
        CHECK (default_target_markup_basis_points >= 0);

-- +goose Down

ALTER TABLE categories DROP CONSTRAINT categories_target_markup_basis_points_check;
ALTER TABLE tenant_settings DROP CONSTRAINT tenant_settings_default_target_markup_basis_points_check;

-- Reverse conversion: margin = markup / (1 + markup). A markup value above
-- what a <100%-margin column can represent has no exact inverse under the
-- old CHECK (< 10000); clamp rather than fail the rollback outright.
UPDATE categories SET target_markup_basis_points =
    LEAST(9999, ROUND((target_markup_basis_points::numeric * 10000) / (10000 + target_markup_basis_points)))
    WHERE target_markup_basis_points IS NOT NULL AND target_markup_basis_points > 0;

UPDATE tenant_settings SET default_target_markup_basis_points =
    LEAST(9999, ROUND((default_target_markup_basis_points::numeric * 10000) / (10000 + default_target_markup_basis_points)))
    WHERE default_target_markup_basis_points > 0;

ALTER TABLE categories
    ADD CONSTRAINT categories_target_margin_basis_points_check
        CHECK (target_markup_basis_points IS NULL OR (target_markup_basis_points >= 0 AND target_markup_basis_points < 10000));
ALTER TABLE tenant_settings
    ADD CONSTRAINT tenant_settings_default_target_margin_basis_points_check
        CHECK (default_target_markup_basis_points >= 0 AND default_target_markup_basis_points < 10000);

ALTER TABLE categories RENAME COLUMN target_markup_basis_points TO target_margin_basis_points;
ALTER TABLE tenant_settings RENAME COLUMN default_target_markup_basis_points TO default_target_margin_basis_points;
