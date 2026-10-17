-- issue_activities：Plane 的 issue_activities 表（20 列）按 M4 设计 4.11 保留 17 列。
-- 删除 issue_comment_id（评论不写动态）、epoch（只写不读）、attachments（从不写）（4.11）。

-- +goose Up
CREATE TABLE issue_activities (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    -- 这一行的工作项的项目（3.13）
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    -- Plane 可空、DO_NOTHING：动态随工作项删除（3.6）
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    verb varchar(255) NOT NULL DEFAULT 'created' CHECK (verb IN ('created', 'updated', 'deleted')),
    field varchar(255),
    old_value text,
    new_value text,
    -- 不是外键：指向状态、标签、账户、链接、工作项
    old_identifier uuid,
    new_identifier uuid,
    comment text NOT NULL DEFAULT '',
    -- 为空表示 nerve 自己（决策点 4）
    actor_id uuid REFERENCES users ON DELETE SET NULL,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
-- 列表的游标、找"最新的一行"、物理级联（4.11），所以不带条件
CREATE INDEX issue_activities_issue_id_created_at_idx ON issue_activities (issue_id, created_at, id);
-- 物理级联按它们找子行（4 节开头）
CREATE INDEX issue_activities_workspace_id_idx ON issue_activities (workspace_id);
CREATE INDEX issue_activities_project_id_idx ON issue_activities (project_id);

-- +goose Down
DROP TABLE issue_activities;
