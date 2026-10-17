-- issue_assignees：Plane 的 issue_assignees 表（10 列）按 M4 设计 4.3 照搬 10 列。
-- 负责人是项目的有效成员、角色不低于成员（3.7）；离开项目的人仍是已有工作项的负责人（3.6）。

-- +goose Up
CREATE TABLE issue_assignees (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    -- 照搬；v0 不物理删除账户，不触发
    assignee_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX issue_assignees_issue_id_assignee_id_key ON issue_assignees (issue_id, assignee_id) WHERE deleted_at IS NULL;
-- 个人主页的"已分配"、按负责人筛选（4.3）
CREATE INDEX issue_assignees_assignee_id_idx ON issue_assignees (assignee_id) WHERE deleted_at IS NULL;
-- 物理级联按它们找子行（4 节开头）
CREATE INDEX issue_assignees_issue_id_idx ON issue_assignees (issue_id);
CREATE INDEX issue_assignees_workspace_id_idx ON issue_assignees (workspace_id);
CREATE INDEX issue_assignees_project_id_idx ON issue_assignees (project_id);

-- +goose Down
DROP TABLE issue_assignees;
