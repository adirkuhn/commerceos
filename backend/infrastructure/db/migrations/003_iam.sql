-- +goose Up

CREATE TABLE iam_groups (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

ALTER TABLE iam_groups ENABLE ROW LEVEL SECURITY;
CREATE POLICY iam_groups_tenant_isolation ON iam_groups
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

CREATE TABLE iam_user_groups (
    group_id    UUID        NOT NULL REFERENCES iam_groups(id) ON DELETE CASCADE,
    user_id     TEXT        NOT NULL,
    tenant_id   UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, user_id)
);

ALTER TABLE iam_user_groups ENABLE ROW LEVEL SECURITY;
CREATE POLICY iam_user_groups_tenant_isolation ON iam_user_groups
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

CREATE TABLE iam_group_permissions (
    group_id    UUID        NOT NULL REFERENCES iam_groups(id) ON DELETE CASCADE,
    tenant_id   UUID        NOT NULL,
    rule        TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, rule)
);

ALTER TABLE iam_group_permissions ENABLE ROW LEVEL SECURITY;
CREATE POLICY iam_group_permissions_tenant_isolation ON iam_group_permissions
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

CREATE INDEX idx_iam_user_groups_user ON iam_user_groups (user_id, tenant_id);

-- +goose Down

DROP TABLE IF EXISTS iam_group_permissions;
DROP TABLE IF EXISTS iam_user_groups;
DROP TABLE IF EXISTS iam_groups;
