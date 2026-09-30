-- project_user_properties：Plane 的 project_user_properties 表（15 列）按 M3 设计 4.8 保留 11 列。
-- 一个账户在一个项目的显示设置（3.18）：导航偏好和侧边栏里项目的顺序，随项目成员关系一起建出。
-- 工作项列表的筛选和显示列（filters、display_filters、display_properties、rich_filters）由它们的使用者 M4 按自己的格式加回（3.18）。

-- +goose Up
CREATE TABLE project_user_properties (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    -- 外层恰好一个键 navigation，它恰好有 default_tab（字符串）和 hide_in_more_menu（数组）；外层的 CASE 让数组、
    -- 标量和 JSON null 也得到 check_violation（M2 设计 3.13）。默认值来自模型，去掉 pages（文档页已砍）
    preferences jsonb NOT NULL DEFAULT '{"navigation": {"default_tab": "work_items", "hide_in_more_menu": []}}'
        CHECK (CASE WHEN jsonb_typeof(preferences) = 'object' THEN
                (preferences - 'navigation') = '{}'::jsonb
            AND CASE WHEN jsonb_typeof(preferences -> 'navigation') = 'object' THEN
                    (preferences -> 'navigation') ?& array['default_tab', 'hide_in_more_menu']
                AND ((preferences -> 'navigation') - array['default_tab', 'hide_in_more_menu']) = '{}'::jsonb
                AND jsonb_typeof(preferences -> 'navigation' -> 'default_tab') = 'string'
                AND jsonb_typeof(preferences -> 'navigation' -> 'hide_in_more_menu') = 'array'
              ELSE false END
          ELSE false END),
    -- 侧边栏里项目的顺序（3.18）
    sort_order double precision NOT NULL DEFAULT 65535,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX project_user_properties_project_id_user_id_key ON project_user_properties (project_id, user_id)
    WHERE deleted_at IS NULL;
-- 物理级联（M4 的 60 天清理）按它们找子行（4 节开头）
CREATE INDEX project_user_properties_workspace_id_idx ON project_user_properties (workspace_id);
CREATE INDEX project_user_properties_project_id_idx ON project_user_properties (project_id);

-- +goose Down
DROP TABLE project_user_properties;
