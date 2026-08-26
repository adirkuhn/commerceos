-- +goose Up
-- Bol's own "not for sale" reasons (GET /offers/{id}/not-for-sale-reasons,
-- v11 -- see infrastructure/bol.NotForSaleReasons) checked after every Bol
-- push: a push can 200/202 with no error at all yet Bol still won't list
-- the offer for sale (e.g. a still-missing mandatory content attribute).
-- Without this there was no way for the merchant to find out short of
-- checking Bol's own seller portal by hand. Stored on the listing (current
-- state), separate from last_sync_error (that's "did the push itself
-- fail," this is "did the push succeed but Bol still won't sell it") --
-- distinct failure modes, conflating them would misreport a clean push as
-- an error.
ALTER TABLE product_channel_listings ADD COLUMN not_for_sale_note TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE product_channel_listings DROP COLUMN not_for_sale_note;
