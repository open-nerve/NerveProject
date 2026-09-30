-- project_members：Plane 的 project_members 表（16 列）按 M3 设计 4.7 保留 11 列。
-- 移出、离开、停用都只把 is_active 改为假，行不删除；删除项目、删除工作区时随之软删除。

-- +goose Up
CREATE TABLE project_members (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    -- Plane 可空，改为非空：没有成员的成员关系没有意义
    member_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    -- 访客 5、成员 15、管理员 20，同 workspace_members.role
    role smallint NOT NULL DEFAULT 5 CHECK (role IN (5, 15, 20)),
    is_active boolean NOT NULL DEFAULT true,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX project_members_project_id_member_id_key ON project_members (project_id, member_id) WHERE deleted_at IS NULL;
-- 连带结束、降为访客、停用、我的项目角色（4.7）
CREATE INDEX project_members_member_id_idx ON project_members (member_id) WHERE deleted_at IS NULL;
-- 物理级联（M4 的 60 天清理）按它们找子行（4 节开头）
CREATE INDEX project_members_workspace_id_idx ON project_members (workspace_id);
CREATE INDEX project_members_project_id_idx ON project_members (project_id);

-- +goose Down
DROP TABLE project_members;
