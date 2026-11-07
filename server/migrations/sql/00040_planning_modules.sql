-- modules：Plane 的 modules 表（22 列）按 M6 设计 4.2 与 P2 spec 2.2 保留 16 列。
-- 状态只有六种、默认 planned，由 CHECK 约束（Plane 只在 Python 的 choices）；起止日期各自可空，两者都有时开始不晚于目标
-- （Plane 只在序列化器检查）。名称在项目内未删除的模块（含已归档）中不分大小写唯一（Plane 的唯一键 (name, project_id)
-- 区分大小写）。lead_id 同 Plane 为 SET NULL。
-- view_props、logo_props、external_source、external_id、description_text、description_html 没有读者，删除。

-- +goose Up
CREATE TABLE modules (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    name varchar(255) NOT NULL CHECK (name <> ''),
    description text NOT NULL DEFAULT '',
    start_date date,
    target_date date,
    status varchar(20) NOT NULL DEFAULT 'planned'
        CHECK (status IN ('backlog', 'planned', 'in-progress', 'paused', 'completed', 'cancelled')),
    lead_id uuid REFERENCES users ON DELETE SET NULL,
    sort_order double precision NOT NULL DEFAULT 65535,
    archived_at timestamptz,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT modules_dates_check CHECK (start_date IS NULL OR target_date IS NULL OR start_date <= target_date)
);
-- 名称唯一：项目内未删除（含已归档）的模块，不分大小写；写入在项目行锁下，冲突由适配器映射为 409
CREATE UNIQUE INDEX modules_project_id_name_key ON modules (project_id, lower(name)) WHERE deleted_at IS NULL;
-- 项目的模块列表（默认顺序 sort_order, created_at, id）
CREATE INDEX modules_project_list_idx ON modules (project_id, sort_order, created_at, id) WHERE deleted_at IS NULL;
-- 物理级联（60 天清理）按它们找子行，所以不带条件
CREATE INDEX modules_workspace_id_idx ON modules (workspace_id);
CREATE INDEX modules_project_id_idx ON modules (project_id);
-- 60 天清理找候选
CREATE INDEX modules_deleted_at_idx ON modules (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP TABLE modules;
