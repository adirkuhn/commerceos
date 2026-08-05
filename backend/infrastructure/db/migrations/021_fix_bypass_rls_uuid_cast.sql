-- +goose Up

-- Every tenant_isolation policy so far reads:
--   current_setting('app.current_tenant', true) = 'bypass_rls'
--   OR tenant_id = current_setting('app.current_tenant', true)::uuid
--
-- This was never actually evaluated by Postgres before 020_app_role_and_-
-- force_rls.sql, because the app always connected as a superuser (which
-- bypasses RLS unconditionally -- the policy expression is never even
-- attached to the query plan). Now that a real, non-superuser role runs
-- these queries, the expression is genuinely evaluated, and Postgres does
-- not guarantee the OR short-circuits before the right-hand side's
-- ::uuid cast runs. When app.current_tenant is literally 'bypass_rls'
-- (the super_admin / system-job sentinel -- see internal/tenant), that
-- cast throws `invalid input syntax for type uuid: "bypass_rls"` --
-- confirmed live, this broke the scheduler's orphaned-import-job recovery
-- and pending-job polling on startup.
--
-- Fix: NULLIF(current_setting(...), 'bypass_rls') evaluates to NULL
-- whenever the setting is the sentinel, and NULL::uuid is a valid no-op
-- cast (not an error) -- so the right-hand comparison becomes
-- `tenant_id = NULL`, which is NULL, not an error, and the already-true
-- left-hand side still makes the whole OR true. For a real tenant UUID,
-- NULLIF passes the value through unchanged and behaves exactly as
-- before.

-- +goose StatementBegin
DO $$
DECLARE
    t RECORD;
BEGIN
    FOR t IN
        SELECT c.relname, p.polname
        FROM pg_policy p JOIN pg_class c ON c.oid = p.polrelid
        WHERE pg_get_expr(p.polqual, p.polrelid) LIKE '%bypass_rls%'
    LOOP
        EXECUTE format(
            'ALTER POLICY %I ON %I USING (current_setting(''app.current_tenant'', true) = ''bypass_rls'' OR tenant_id = NULLIF(current_setting(''app.current_tenant'', true), ''bypass_rls'')::uuid)',
            t.polname, t.relname
        );
    END LOOP;
END
$$;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DO $$
DECLARE
    t RECORD;
BEGIN
    FOR t IN
        SELECT c.relname, p.polname
        FROM pg_policy p JOIN pg_class c ON c.oid = p.polrelid
        WHERE pg_get_expr(p.polqual, p.polrelid) LIKE '%bypass_rls%'
    LOOP
        EXECUTE format(
            'ALTER POLICY %I ON %I USING (current_setting(''app.current_tenant'', true) = ''bypass_rls'' OR tenant_id = current_setting(''app.current_tenant'', true)::uuid)',
            t.polname, t.relname
        );
    END LOOP;
END
$$;
-- +goose StatementEnd
