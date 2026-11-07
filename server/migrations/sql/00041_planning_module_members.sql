-- module_members：Plane 的 module_members 表（10 列）按 M6 设计 4.2 与 P2 spec 2.2 照搬 10 列。
-- 一个账户在一个模块的未删除行中至多一行；Plane 另有的 UNIQUE (module_id, member_id, deleted_at) 多余，删除。
-- 成员离开项目不回写模块（设计 3.2），所以这里不约束成员资格；资格只在写入时检查新加入的成员。

-- +goose Up
CREATE TABLE module_members (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    module_id uuid NOT NULL REFERENCES modules ON DELETE CASCADE,
    -- 照搬；v0 不物理删除账户，不触发
    member_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX module_members_module_id_member_id_key ON module_members (module_id, member_id) WHERE deleted_at IS NULL;
-- 物理级联按它们找子行，所以不带条件
CREATE INDEX module_members_module_id_idx ON module_members (module_id);
CREATE INDEX module_members_member_id_idx ON module_members (member_id);
CREATE INDEX module_members_workspace_id_idx ON module_members (workspace_id);
CREATE INDEX module_members_project_id_idx ON module_members (project_id);
-- 60 天清理找候选
CREATE INDEX module_members_deleted_at_idx ON module_members (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP TABLE module_members;
