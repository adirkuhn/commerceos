-- +goose Up

-- Per-(product, Bol connection) product-content fields Bol's Product
-- Content API needs to move a listing from "Onvoldoende" (insufficient)
-- to "Compleet" -- extracted directly from Bol's own product-content page
-- for a real product in this catalog. Deliberately excludes Productgroep
-- (Bol's category/GPC taxonomy) -- that needs mapping against Bol's real
-- category tree, tracked separately. Same connection-scoped pattern as
-- bol_product_overrides (095_bol_integration.sql) -- a tenant could have
-- more than one Bol connection.
CREATE TABLE bol_product_content (
    id                          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID        NOT NULL,
    product_id                  UUID        NOT NULL,
    connection_id               UUID        NOT NULL,

    -- Basisinformatie
    manufacturer_contact        TEXT,
    manufacturer_address        TEXT,
    manufacturer_name           TEXT,
    shelf_life_closed           TEXT,
    shelf_life_after_opening    TEXT,
    ingredients                 TEXT,
    content_amount              TEXT,
    content_unit                TEXT,
    country_of_origin           TEXT,
    substance                   TEXT,
    instruction_language        TEXT,
    packaging_language          TEXT,

    -- Optioneel
    items_per_package           INTEGER,
    package_insert_url          TEXT,
    is_gift                     BOOLEAN,
    dermatologically_tested     BOOLEAN,
    hazard_statements           TEXT,
    skin_type                   TEXT,
    hypoallergenic              BOOLEAN,
    contains_zinc_oxide         BOOLEAN,
    content_per_dispenser_amount TEXT,
    content_per_dispenser_unit  TEXT,
    brand                       TEXT,
    fragrance_free              BOOLEAN,
    ph_neutral                  BOOLEAN,
    cooling_effect              BOOLEAN,
    package_width_mm            INTEGER,
    package_height_mm           INTEGER,
    package_length_mm           INTEGER,
    package_weight_g            INTEGER,
    precaution_statements       TEXT,
    precautions                 TEXT,
    warnings                    TEXT,
    video_url                   TEXT,

    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (product_id, connection_id)
);

CREATE INDEX bol_product_content_tenant_id_idx ON bol_product_content (tenant_id);

ALTER TABLE bol_product_content ENABLE ROW LEVEL SECURITY;
ALTER TABLE bol_product_content FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON bol_product_content
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE bol_product_content;
