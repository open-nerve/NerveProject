-- workspace_user_properties：Plane 的 workspace_user_properties 表（14 列）按 M3 设计 4.5 保留 10 列。
-- 一个账户在一个工作区的项目导航偏好（3.18）。第一次修改时建出（INSERT … ON CONFLICT），没有这一行时
-- 接口答默认值，默认值与这里的 DEFAULT 相同；删除工作区时随之软删除。
-- 工作项列表的筛选和显示列（filters、display_filters、display_properties、rich_filters）由它们的使用者 M4 按自己的格式加回（M3 设计 3.18）。

-- +goose Up
CREATE TABLE workspace_user_properties (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    -- 侧边栏显示的项目数，0 表示不限（模型的 default=10）
    navigation_project_limit integer NOT NULL DEFAULT 10 CHECK (navigation_project_limit >= 0),
    -- 模型的 choices
    navigation_control_preference varchar(25) NOT NULL DEFAULT 'ACCORDION'
        CHECK (navigation_control_preference IN ('ACCORDION', 'TABBED')),
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
-- 一个账户在一个工作区至多一行未删除的；INSERT … ON CONFLICT 以它为冲突的目标（3.18）
CREATE UNIQUE INDEX workspace_user_properties_workspace_id_user_id_key ON workspace_user_properties (workspace_id, user_id)
    WHERE deleted_at IS NULL;
-- 物理级联（M4 的 60 天清理）按它找子行（4 节开头）
CREATE INDEX workspace_user_properties_workspace_id_idx ON workspace_user_properties (workspace_id);

-- +goose Down
DROP TABLE workspace_user_properties;
