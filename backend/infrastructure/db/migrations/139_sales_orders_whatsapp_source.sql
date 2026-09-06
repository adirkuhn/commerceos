-- +goose Up
-- Sales orders can now also arrive from the WhatsApp sales channel
-- (application/whatsapp.OrderFlowUseCase, CreateSalesOrderInput.Source =
-- SalesOrderSourceWhatsApp) -- this constraint was never updated when that
-- feature was built, so every real WhatsApp order insert would have failed
-- (only caught now while adding SalesOrderRepository.FindByNumber for
-- payment webhooks, which operate on WhatsApp orders).
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_source_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_source_check CHECK (source IN ('pos', 'manual', 'woocommerce', 'bol', 'whatsapp'));

-- +goose Down
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_source_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_source_check CHECK (source IN ('pos', 'manual', 'woocommerce', 'bol'));
