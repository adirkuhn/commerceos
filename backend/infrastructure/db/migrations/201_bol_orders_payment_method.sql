-- +goose Up
-- bol.com orders are paid to bol.com, which pays the store out later --
-- there's no payment transaction of their own. New imports stamp them
-- "paid via bol.com" (application/channels.ImportBolOrdersUseCase); this
-- backfills orders imported before that.
UPDATE sales_orders
SET payment_method = 'bol', payment_method_title = 'bol.com'
WHERE source = 'bol' AND payment_method = '';

-- +goose Down
UPDATE sales_orders
SET payment_method = '', payment_method_title = ''
WHERE source = 'bol' AND payment_method = 'bol' AND payment_method_title = 'bol.com';
