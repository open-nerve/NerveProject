-- issue_labels：Plane 的 issue_labels 表（10 列）按 M4 设计 4.4 照搬 10 列。
-- Plane 没有任何唯一约束，同一个标签可以重复；这里新加部分唯一（差异清单）。删除标签软删除它的行（3.6）。

-- +goose Up
CREATE TABLE issue_labels (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    label_id uuid NOT NULL REFERENCES labels ON DELETE CASCADE,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX issue_labels_issue_id_label_id_key ON issue_labels (issue_id, label_id) WHERE deleted_at IS NULL;
-- 物理级联按它们找子行（4 节开头），标签的也是
CREATE INDEX issue_labels_issue_id_idx ON issue_labels (issue_id);
CREATE INDEX issue_labels_label_id_idx ON issue_labels (label_id);
CREATE INDEX issue_labels_workspace_id_idx ON issue_labels (workspace_id);
CREATE INDEX issue_labels_project_id_idx ON issue_labels (project_id);

-- +goose Down
DROP TABLE issue_labels;
