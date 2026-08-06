-- +goose Up
-- Populated by RunDocumentParsingJobUseCase when a PDF upload's AI
-- extraction found a supplier name / currency the document itself states --
-- purely informational suggestions for the review UI (see
-- CreateSupplierAndAssignToPurchaseOrderUseCase, which still requires an
-- explicit human click, and the header's own currency_code, which this
-- never overwrites). Empty string means no suggestion, same convention as
-- every other required-with-default TEXT column in this schema.
ALTER TABLE purchase_orders ADD COLUMN suggested_supplier_name TEXT NOT NULL DEFAULT '';
ALTER TABLE purchase_orders ADD COLUMN suggested_currency_code TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE purchase_orders DROP COLUMN suggested_supplier_name;
ALTER TABLE purchase_orders DROP COLUMN suggested_currency_code;
