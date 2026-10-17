-- issue_subscribers：Plane 的 issue_subscribers 表（10 列）按 M4 设计 4.5 照搬 10 列。
-- 写入只有自动关注（3.12）。

-- +goose Up
CREATE TABLE issue_subscribers (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    -- 照搬；v0 不物理删除账户，不触发
    subscriber_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX issue_subscribers_issue_id_subscriber_id_key ON issue_subscribers (issue_id, subscriber_id) WHERE deleted_at IS NULL;
-- "已关注"、按关注者筛选（4.5）
CREATE INDEX issue_subscribers_subscriber_id_idx ON issue_subscribers (subscriber_id) WHERE deleted_at IS NULL;
-- 物理级联按它们找子行（4 节开头）
CREATE INDEX issue_subscribers_issue_id_idx ON issue_subscribers (issue_id);
CREATE INDEX issue_subscribers_workspace_id_idx ON issue_subscribers (workspace_id);
CREATE INDEX issue_subscribers_project_id_idx ON issue_subscribers (project_id);

-- +goose Down
DROP TABLE issue_subscribers;
