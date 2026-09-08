-- +goose Up
-- Multi-currency, scoped one-currency-per-channel (not a shopper-facing
-- currency picker -- see the AskUserQuestion decision in the 2026-09-08
-- session this shipped in): a SalesChannel can run in a currency other
-- than the tenant's default, priced either via a manual per-SKU override
-- or, absent one, an auto-converted fx_rates entry.

-- fx_rates: tenant-maintained rate FROM the tenant's default currency TO
-- another currency, expressed in basis points the same way
-- coupons.discount_value/tax_rates already are (10000 = 1:1 parity) --
-- e.g. EUR->USD at 1.0876 is stored as 10876. No live-rate API (third-party
-- gate in CLAUDE.md) -- a human sets/updates this.
CREATE TABLE fx_rates (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    currency_code     CHAR(3)     NOT NULL,
    rate_basis_points BIGINT      NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fx_rates_rate_positive_check CHECK (rate_basis_points > 0),
    CONSTRAINT fx_rates_tenant_currency_unique UNIQUE (tenant_id, currency_code)
);

CREATE INDEX fx_rates_tenant_id_idx ON fx_rates (tenant_id);

ALTER TABLE fx_rates ENABLE ROW LEVEL SECURITY;
ALTER TABLE fx_rates FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON fx_rates
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- sales_channels.currency_code: '' means "use the tenant's default
-- currency" (every existing channel keeps today's behavior unchanged).
ALTER TABLE sales_channels ADD COLUMN currency_code CHAR(3) NOT NULL DEFAULT '';

-- sku_currency_prices: a manual per-currency price override for one SKU --
-- absent one, the channel's currency falls back to fx_rates conversion of
-- the SKU's own price_minor_units/currency_code.
CREATE TABLE sku_currency_prices (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID        NOT NULL,
    sku_id                UUID        NOT NULL REFERENCES skus(id) ON DELETE CASCADE,
    currency_code         CHAR(3)     NOT NULL,
    price_minor_units     BIGINT      NOT NULL,
    sale_price_minor_units BIGINT,
    sale_end_date         DATE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sku_currency_prices_unique UNIQUE (sku_id, currency_code)
);

CREATE INDEX sku_currency_prices_tenant_id_idx ON sku_currency_prices (tenant_id);
CREATE INDEX sku_currency_prices_sku_id_idx ON sku_currency_prices (sku_id);

ALTER TABLE sku_currency_prices ENABLE ROW LEVEL SECURITY;
ALTER TABLE sku_currency_prices FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sku_currency_prices
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- coupon_currency_amounts: same override pattern for a Fixed-type
-- coupon's DiscountValue, since coupons are tenant-wide (not
-- channel-scoped) and so can be redeemed on channels running in
-- different currencies -- absent an override, fx_rates converts
-- discount_value the same way a SKU price would.
CREATE TABLE coupon_currency_amounts (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL,
    coupon_id     UUID        NOT NULL REFERENCES coupons(id) ON DELETE CASCADE,
    currency_code CHAR(3)     NOT NULL,
    amount_minor_units BIGINT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT coupon_currency_amounts_unique UNIQUE (coupon_id, currency_code)
);

CREATE INDEX coupon_currency_amounts_tenant_id_idx ON coupon_currency_amounts (tenant_id);
CREATE INDEX coupon_currency_amounts_coupon_id_idx ON coupon_currency_amounts (coupon_id);

ALTER TABLE coupon_currency_amounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE coupon_currency_amounts FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON coupon_currency_amounts
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE coupon_currency_amounts;
DROP TABLE sku_currency_prices;
ALTER TABLE sales_channels DROP COLUMN currency_code;
DROP TABLE fx_rates;
