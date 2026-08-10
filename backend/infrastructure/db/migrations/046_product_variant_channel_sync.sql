-- +goose Up
-- sync_variants_as_woo_variations: false (default) means each Variant
-- pushes as its own separate simple Woo product; true means all Variants
-- push as ONE Woo "variable" product with one Woo variation per Variant.
-- Only matters when products.has_variants is true. See
-- domain/pms.Product's doc comment for the full picture, and
-- application/channels.SyncProductToChannelsUseCase for the migration that
-- runs when this flips on an already-pushed product.
ALTER TABLE products ADD COLUMN sync_variants_as_woo_variations BOOLEAN NOT NULL DEFAULT false;

-- variant_id/is_variation let one product_channel_listings row represent
-- either the whole product (variant_id NULL -- a simple product, or the
-- parent of a variable product) or one specific Variant (variant_id set --
-- either its own standalone simple Woo product, or a Woo variation nested
-- under the sibling product-level row's external_id as parent, per
-- is_variation). See domain/channels.ProductChannelListing's doc comment.
ALTER TABLE product_channel_listings ADD COLUMN variant_id UUID NULL REFERENCES variants(id) ON DELETE CASCADE;
ALTER TABLE product_channel_listings ADD COLUMN is_variation BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX product_channel_listings_variant_id_idx ON product_channel_listings (variant_id);

-- +goose Down
DROP INDEX IF EXISTS product_channel_listings_variant_id_idx;
ALTER TABLE product_channel_listings DROP COLUMN is_variation;
ALTER TABLE product_channel_listings DROP COLUMN variant_id;
ALTER TABLE products DROP COLUMN sync_variants_as_woo_variations;
