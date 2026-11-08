-- module_issues：Plane 的 module_issues 表（10 列）按 M6 设计 4.2 与 P3a spec 2.2 照搬 10 列，写法照 issue_labels（00017）。
-- 一个工作项可属于多个模块，同一对在未删除的行中至多一行：部分唯一 (issue_id, module_id)，同 Plane；Plane 另有的
-- UNIQUE (issue_id, module_id, deleted_at) 多余，删除。整体替换时移出的软删、新加入的插入、留下的不动（spec 2.3）。
-- 归属归 issue 模块（设计 4.1）；module_id 引用 planning 的 modules，TestSQLCSchemaScope 允许（先例 issue_labels.label_id）。

-- +goose Up
CREATE TABLE module_issues (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    module_id uuid NOT NULL REFERENCES modules ON DELETE CASCADE,
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX module_issues_issue_id_module_id_key ON module_issues (issue_id, module_id) WHERE deleted_at IS NULL;
-- 物理级联按它们找子行（包括已删除的行），所以不带条件
CREATE INDEX module_issues_module_id_idx ON module_issues (module_id);
CREATE INDEX module_issues_issue_id_idx ON module_issues (issue_id);
CREATE INDEX module_issues_workspace_id_idx ON module_issues (workspace_id);
CREATE INDEX module_issues_project_id_idx ON module_issues (project_id);
-- 60 天清理找候选
CREATE INDEX module_issues_deleted_at_idx ON module_issues (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP TABLE module_issues;
