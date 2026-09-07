-- +goose Up
-- A sales order can now also arrive from the built-in internal storefront
-- checkout (application/storefront.CheckoutCartUseCase,
-- CreateSalesOrderInput.Source = SalesOrderSourceInternal), same widening
-- pattern as migration 139 added SalesOrderSourceWhatsApp.
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_source_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_source_check CHECK (source IN ('pos', 'manual', 'woocommerce', 'bol', 'whatsapp', 'internal'));

-- +goose Down
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_source_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_source_check CHECK (source IN ('pos', 'manual', 'woocommerce', 'bol', 'whatsapp'));
