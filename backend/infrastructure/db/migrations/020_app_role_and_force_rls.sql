-- +goose Up

-- Closes a real RLS bypass: every service (backend/relay/scheduler) has
-- always connected as the same Postgres role that owns every table and
-- bootstraps the database (POSTGRES_USER, default "commerceos"), which the
-- official postgres image always creates as a superuser. Postgres
-- superusers -- and any role with BYPASSRLS -- ignore RLS policies
-- unconditionally, regardless of policy definitions or FORCE ROW LEVEL
-- SECURITY. So despite every tenant-scoped table below having a
-- tenant_isolation policy, none of it was ever actually enforced by
-- Postgres itself; isolation only ever depended on application code
-- remembering to call tenant.WithTenant correctly.
--
-- The fix: a dedicated, non-superuser, non-BYPASSRLS role
-- (commerceos_app) that the running services connect as instead. The
-- migration/bootstrap role (POSTGRES_USER) remains superuser and keeps
-- owning the schema, but is no longer used for live traffic.
--
-- The password below is a placeholder -- deployments MUST rotate it via
-- `ALTER ROLE commerceos_app PASSWORD '...'` after this migration runs
-- (docker-compose's migrate service does this automatically from
-- APP_DB_PASSWORD; see docker-compose.yml). It is deliberately not a real
-- secret, the same way POSTGRES_PASSWORD's dev default isn't.
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'commerceos_app') THEN
        CREATE ROLE commerceos_app LOGIN PASSWORD 'changeme'
            NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE NOREPLICATION;
    END IF;
END
$$;
-- +goose StatementEnd

GRANT USAGE ON SCHEMA public TO commerceos_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO commerceos_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO commerceos_app;

-- So tables created by future migrations are usable by commerceos_app
-- without a manual GRANT in every migration file.
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO commerceos_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO commerceos_app;

-- ENABLE ROW LEVEL SECURITY (already set on every table below) does not
-- restrict a table's owner. FORCE makes the policy apply even to the
-- owning role -- the actual backstop, on top of commerceos_app no longer
-- being that owner in the first place.
ALTER TABLE iam_groups               FORCE ROW LEVEL SECURITY;
ALTER TABLE iam_user_groups          FORCE ROW LEVEL SECURITY;
ALTER TABLE iam_group_permissions    FORCE ROW LEVEL SECURITY;
ALTER TABLE integration_connections  FORCE ROW LEVEL SECURITY;
ALTER TABLE categories               FORCE ROW LEVEL SECURITY;
ALTER TABLE products                 FORCE ROW LEVEL SECURITY;
ALTER TABLE attribute_definitions    FORCE ROW LEVEL SECURITY;
ALTER TABLE variants                 FORCE ROW LEVEL SECURITY;
ALTER TABLE variant_attributes       FORCE ROW LEVEL SECURITY;
ALTER TABLE skus                     FORCE ROW LEVEL SECURITY;
ALTER TABLE media                    FORCE ROW LEVEL SECURITY;
ALTER TABLE user_permissions         FORCE ROW LEVEL SECURITY;
ALTER TABLE warehouses               FORCE ROW LEVEL SECURITY;
ALTER TABLE batches                  FORCE ROW LEVEL SECURITY;
ALTER TABLE stock_movements          FORCE ROW LEVEL SECURITY;
ALTER TABLE product_channel_listings FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_settings          FORCE ROW LEVEL SECURITY;
ALTER TABLE import_jobs              FORCE ROW LEVEL SECURITY;
ALTER TABLE notifications            FORCE ROW LEVEL SECURITY;
ALTER TABLE channel_sync_logs        FORCE ROW LEVEL SECURITY;
ALTER TABLE sync_schedules           FORCE ROW LEVEL SECURITY;

-- +goose Down
ALTER TABLE iam_groups               NO FORCE ROW LEVEL SECURITY;
ALTER TABLE iam_user_groups          NO FORCE ROW LEVEL SECURITY;
ALTER TABLE iam_group_permissions    NO FORCE ROW LEVEL SECURITY;
ALTER TABLE integration_connections  NO FORCE ROW LEVEL SECURITY;
ALTER TABLE categories               NO FORCE ROW LEVEL SECURITY;
ALTER TABLE products                 NO FORCE ROW LEVEL SECURITY;
ALTER TABLE attribute_definitions    NO FORCE ROW LEVEL SECURITY;
ALTER TABLE variants                 NO FORCE ROW LEVEL SECURITY;
ALTER TABLE variant_attributes       NO FORCE ROW LEVEL SECURITY;
ALTER TABLE skus                     NO FORCE ROW LEVEL SECURITY;
ALTER TABLE media                    NO FORCE ROW LEVEL SECURITY;
ALTER TABLE user_permissions         NO FORCE ROW LEVEL SECURITY;
ALTER TABLE warehouses               NO FORCE ROW LEVEL SECURITY;
ALTER TABLE batches                  NO FORCE ROW LEVEL SECURITY;
ALTER TABLE stock_movements          NO FORCE ROW LEVEL SECURITY;
ALTER TABLE product_channel_listings NO FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_settings          NO FORCE ROW LEVEL SECURITY;
ALTER TABLE import_jobs              NO FORCE ROW LEVEL SECURITY;
ALTER TABLE notifications            NO FORCE ROW LEVEL SECURITY;
ALTER TABLE channel_sync_logs        NO FORCE ROW LEVEL SECURITY;
ALTER TABLE sync_schedules           NO FORCE ROW LEVEL SECURITY;

REVOKE SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public FROM commerceos_app;
REVOKE USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public FROM commerceos_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM commerceos_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE USAGE, SELECT ON SEQUENCES FROM commerceos_app;
DROP ROLE IF EXISTS commerceos_app;
