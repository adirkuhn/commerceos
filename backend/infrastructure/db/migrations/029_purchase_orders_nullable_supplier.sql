-- +goose Up
-- Supplier is now optional at creation time: uploading a document without
-- picking a supplier up front creates a needs_review PO (see
-- application/procurement's RunDocumentParsingJobUseCase and
-- CreateSupplierAndAssignToPurchaseOrderUseCase) rather than forcing a
-- guess. ON DELETE RESTRICT is unaffected -- a NULL fk value is simply
-- exempt from the constraint.
ALTER TABLE purchase_orders ALTER COLUMN supplier_id DROP NOT NULL;

-- +goose Down
ALTER TABLE purchase_orders ALTER COLUMN supplier_id SET NOT NULL;
