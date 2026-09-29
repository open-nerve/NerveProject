-- workspace_members：Plane 的 workspace_members 表（17 列）按 M3 设计 4.3 保留 10 列。
-- 移出、离开、停用都只把 is_active 改为假，行不删除（Plane 相同）。

-- +goose Up
CREATE TABLE workspace_members (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    member_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    -- 访客 5、成员 15、管理员 20（Plane 的 ROLE_CHOICES；快照的 role >= 0 收紧为三个取值）
    role smallint NOT NULL DEFAULT 5 CHECK (role IN (5, 15, 20)),
    is_active boolean NOT NULL DEFAULT true,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX workspace_members_workspace_id_member_id_key ON workspace_members (workspace_id, member_id)
    WHERE deleted_at IS NULL;
-- 我的工作区、停用（4.3）
CREATE INDEX workspace_members_member_id_idx ON workspace_members (member_id) WHERE deleted_at IS NULL;
-- 物理级联（M4 的 60 天清理）按它找子行（4 节开头）
CREATE INDEX workspace_members_workspace_id_idx ON workspace_members (workspace_id);

-- +goose Down
DROP TABLE workspace_members;
