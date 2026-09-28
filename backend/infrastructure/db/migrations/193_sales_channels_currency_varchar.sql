-- +goose Up
-- sales_channels.currency_code uses '' for "the tenant's default currency"
-- (158_multi_currency.sql). As CHAR(3), Postgres padded '' to '   ', which
-- the app read back as a currency code: every product listing on such a
-- channel failed its price lookup. VARCHAR keeps '' empty.
ALTER TABLE sales_channels
    ALTER COLUMN currency_code DROP DEFAULT,
    ALTER COLUMN currency_code TYPE VARCHAR(3) USING trim(currency_code),
    ALTER COLUMN currency_code SET DEFAULT '';

-- +goose Down
ALTER TABLE sales_channels
    ALTER COLUMN currency_code DROP DEFAULT,
    ALTER COLUMN currency_code TYPE CHAR(3) USING currency_code,
    ALTER COLUMN currency_code SET DEFAULT '';
