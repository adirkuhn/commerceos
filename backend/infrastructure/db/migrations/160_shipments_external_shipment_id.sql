-- +goose Up
-- Generalize shipments.myparcel_shipment_id to external_shipment_id (TEXT)
-- so a second carrier's shipment id (not necessarily a bigint) can be
-- stored in the same column -- see domain/integrations.CarrierClient's doc
-- comment. The MyParcel adapter still deals in MyParcel's own int64 ids;
-- it converts to/from this column's string representation at its own
-- boundary, so no other carrier's adapter is forced into MyParcel's id
-- shape.
ALTER TABLE shipments RENAME COLUMN myparcel_shipment_id TO external_shipment_id;
ALTER TABLE shipments ALTER COLUMN external_shipment_id TYPE TEXT USING external_shipment_id::TEXT;

-- +goose Down
ALTER TABLE shipments ALTER COLUMN external_shipment_id TYPE BIGINT USING NULLIF(external_shipment_id, '')::BIGINT;
ALTER TABLE shipments RENAME COLUMN external_shipment_id TO myparcel_shipment_id;
