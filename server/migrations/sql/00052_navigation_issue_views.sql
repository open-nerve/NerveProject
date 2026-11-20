-- issue_views：Plane 的 issue_views 表按 M7 设计 3.7、4.2 与 P3b spec 2.3 建。project_id 为空即工作区视图（Plane 相同）。
-- 存名字、说明、图标、Nerve 格式的三项列表设置与所有者；所有者是建它的账户，只有他能改（D29）。删去 Plane 的 query、
-- rich_filters、access、is_locked、sort_order、archived_at（社区版的视图都是公开、未锁，D27）。建时由用例给出筛选、
-- 显示与时刻，表上不设默认值。
-- 已存的筛选键按种类（spec 2.3）：项目视图 12 个（shared.ListFilters 的 10 个与 cycle_id、module_id），工作区视图 11 个
-- （10 个与 project_id）；sub_issue 只在 display_filters，archived 不存。数据库只管形状，词由 issue 的 ListSettings 与
-- WorkspaceListSettings 按页面校验。
-- 删项目、工作区由 navigation.delete_rows 软删（工作区那一步只删 project_id 为空的行，spec 2.7）；60 天清理在 issue 之前清它（module.go）。

-- +goose Up
CREATE TABLE issue_views (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    -- 为空即工作区视图
    project_id uuid REFERENCES projects ON DELETE CASCADE,
    -- 去首尾空白后非空、按字符数至多 255 由领域层校验
    name varchar(255) NOT NULL CHECK (name <> ''),
    -- 纯文本，没有资产，不设长度上限（Plane 相同）
    description text NOT NULL DEFAULT '',
    -- {} 表示没有图标；形状同项目的图标（00010）
    logo_props jsonb NOT NULL DEFAULT '{}',
    filters jsonb NOT NULL,
    display_filters jsonb NOT NULL CHECK (CASE WHEN jsonb_typeof(display_filters) = 'object' THEN
            display_filters ?& array['layout','group_by','sub_group_by','order_by','sub_issue','show_empty_groups','calendar']
            AND display_filters - array['layout','group_by','sub_group_by','order_by','sub_issue','show_empty_groups','calendar'] = '{}'::jsonb
            AND jsonb_typeof(display_filters->'layout') = 'string'
            AND jsonb_typeof(display_filters->'group_by') IN ('string','null')
            AND jsonb_typeof(display_filters->'sub_group_by') IN ('string','null')
            AND jsonb_typeof(display_filters->'order_by') = 'string'
            AND jsonb_typeof(display_filters->'sub_issue') = 'boolean'
            AND jsonb_typeof(display_filters->'show_empty_groups') = 'boolean'
            AND CASE WHEN jsonb_typeof(display_filters->'calendar') = 'object' THEN
                (display_filters->'calendar') ?& array['layout','show_weekends']
                AND (display_filters->'calendar') - array['layout','show_weekends'] = '{}'::jsonb
                AND jsonb_typeof(display_filters->'calendar'->'layout') = 'string'
                AND jsonb_typeof(display_filters->'calendar'->'show_weekends') = 'boolean'
            ELSE false END
        ELSE false END),
    display_properties jsonb NOT NULL CHECK (CASE WHEN jsonb_typeof(display_properties) = 'object' THEN
            display_properties ?& array['assignee','start_date','due_date','labels','key','priority','state','sub_issue_count','link','attachment_count','created_on','updated_on','cycle','modules']
            AND display_properties - array['assignee','start_date','due_date','labels','key','priority','state','sub_issue_count','link','attachment_count','created_on','updated_on','cycle','modules'] = '{}'::jsonb
            AND NOT jsonb_path_exists(display_properties, 'strict $.* ? (@.type() != "boolean")')
        ELSE false END),
    -- 照搬；v0 不物理删除账户，不触发
    owned_by_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    -- 键集按种类：项目视图（project_id 不为空）与工作区视图各一种；每个值是字符串的数组
    CONSTRAINT issue_views_filters_check CHECK (CASE WHEN jsonb_typeof(filters) = 'object' THEN
            filters - CASE WHEN project_id IS NULL
                THEN array['state_id','state_group','priority','label_id','assignee_id','created_by_id','mention_id','subscriber_id',
                    'start_date','target_date','project_id']
                ELSE array['state_id','state_group','priority','label_id','assignee_id','created_by_id','mention_id','subscriber_id',
                    'start_date','target_date','cycle_id','module_id'] END = '{}'::jsonb
            AND CASE WHEN NOT jsonb_path_exists(filters, 'strict $.* ? (@.type() != "array")') THEN
                NOT jsonb_path_exists(filters, 'strict $.*[*] ? (@.type() != "string")')
            ELSE false END
        ELSE false END),
    -- 同 projects_logo_props_check：是对象、键的集合、出现的每个键的值类型，嵌套的 emoji、icon 也查到底
    CONSTRAINT issue_views_logo_props_check CHECK (CASE WHEN jsonb_typeof(logo_props) = 'object' THEN
            (logo_props - array['in_use', 'emoji', 'icon']) = '{}'::jsonb
        AND (NOT logo_props ? 'in_use' OR logo_props -> 'in_use' IN ('"emoji"'::jsonb, '"icon"'::jsonb))
        AND (NOT logo_props ? 'emoji' OR CASE WHEN jsonb_typeof(logo_props -> 'emoji') = 'object' THEN
                ((logo_props -> 'emoji') - array['value', 'url']) = '{}'::jsonb
            AND (NOT (logo_props -> 'emoji') ? 'value' OR jsonb_typeof(logo_props -> 'emoji' -> 'value') = 'string')
            AND (NOT (logo_props -> 'emoji') ? 'url' OR jsonb_typeof(logo_props -> 'emoji' -> 'url') = 'string')
          ELSE false END)
        AND (NOT logo_props ? 'icon' OR CASE WHEN jsonb_typeof(logo_props -> 'icon') = 'object' THEN
                ((logo_props -> 'icon') - array['name', 'color', 'background_color']) = '{}'::jsonb
            AND (NOT (logo_props -> 'icon') ? 'name' OR jsonb_typeof(logo_props -> 'icon' -> 'name') = 'string')
            AND (NOT (logo_props -> 'icon') ? 'color' OR jsonb_typeof(logo_props -> 'icon' -> 'color') = 'string')
            AND (NOT (logo_props -> 'icon') ? 'background_color'
                 OR jsonb_typeof(logo_props -> 'icon' -> 'background_color') = 'string')
          ELSE false END)
      ELSE false END)
);
-- 项目的视图列表与工作区的视图列表（新的在前、同一时刻按 id，spec 2.4），各对一条
CREATE INDEX issue_views_project_id_created_at_idx ON issue_views (project_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL AND project_id IS NOT NULL;
CREATE INDEX issue_views_workspace_id_created_at_idx ON issue_views (workspace_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL AND project_id IS NULL;
-- 物理级联按它们找子行，所以不带条件
CREATE INDEX issue_views_workspace_id_idx ON issue_views (workspace_id);
CREATE INDEX issue_views_project_id_idx ON issue_views (project_id);
CREATE INDEX issue_views_owned_by_id_idx ON issue_views (owned_by_id);
-- 60 天清理找候选
CREATE INDEX issue_views_deleted_at_idx ON issue_views (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP TABLE issue_views;
