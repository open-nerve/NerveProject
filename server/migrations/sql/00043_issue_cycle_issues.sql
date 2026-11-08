-- cycle_issues：Plane 的 cycle_issues 表（10 列）按 M6 设计 4.2 与 P3a spec 2.2 照搬 10 列，写法照 issue_labels（00017）。
-- 一个工作项最多在一个迭代（总体设计 5.3）：部分唯一 (issue_id)。Plane 只有部分唯一 (cycle_id, issue_id) 与多余的
-- UNIQUE (issue_id, cycle_id, deleted_at)，“一个迭代”只在代码里（差异清单）。移到别的迭代是软删旧行、插入新行（spec R4）。
-- 归属归 issue 模块（设计 4.1）；cycle_id 引用 planning 的 cycles，TestSQLCSchemaScope 允许（先例 issue_labels.label_id）。

-- +goose Up
CREATE TABLE cycle_issues (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    cycle_id uuid NOT NULL REFERENCES cycles ON DELETE CASCADE,
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX cycle_issues_issue_id_key ON cycle_issues (issue_id) WHERE deleted_at IS NULL;
-- 物理级联按它们找子行（包括已删除的行），所以不带条件
CREATE INDEX cycle_issues_cycle_id_idx ON cycle_issues (cycle_id);
CREATE INDEX cycle_issues_issue_id_idx ON cycle_issues (issue_id);
CREATE INDEX cycle_issues_workspace_id_idx ON cycle_issues (workspace_id);
CREATE INDEX cycle_issues_project_id_idx ON cycle_issues (project_id);
-- 60 天清理找候选
CREATE INDEX cycle_issues_deleted_at_idx ON cycle_issues (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP TABLE cycle_issues;
