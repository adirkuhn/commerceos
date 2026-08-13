-- +goose Up
-- Staff can now create sales orders manually from the admin UI (not just
-- receive them from a POS push) -- widen the source enum to allow it.
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_source_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_source_check CHECK (source IN ('pos', 'manual'));

-- Whether stock has actually been decremented for this order, tracked
-- explicitly rather than inferred from status -- WooCommerce's own status
-- semantics decide when stock moves (pending: not yet: on-hold/processing/
-- completed: reduced; cancelled/failed: returned if it had been reduced),
-- and a status *transition* needs to know what already happened to decide
-- whether to decrement or restock. Defaults false; ReceivePOSSalesOrderUseCase
-- and CreateSalesOrderUseCase set it true after successfully decrementing.
ALTER TABLE sales_orders ADD COLUMN stock_decremented BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE sales_orders DROP COLUMN stock_decremented;
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_source_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_source_check CHECK (source IN ('pos'));
