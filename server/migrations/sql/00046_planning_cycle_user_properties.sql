-- cycle_user_properties：Plane 的 cycle_user_properties 表按 M6/P3b spec 2.4 照 project_user_properties（00012、00026、
-- 00045）去掉 preferences 与 sort_order：一个账户在一个迭代页的显示设置，由调用者自己经 /me/cycles/{id}/preferences 读写。
-- 数据库只管形状，词由 issue 的列表词汇校验。筛选是项目的 12 个键去掉本页的 cycle_id（spec R8）；显示筛选、显示属性与项目相同。
-- 删迭代时随之软删；删项目、工作区由 planning.delete_rows 软删；60 天清理在 cycles 之前清它（module.go）。

-- +goose Up
CREATE TABLE cycle_user_properties (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    cycle_id uuid NOT NULL REFERENCES cycles ON DELETE CASCADE,
    -- 同 project_user_properties；v0 不物理删除账户，不触发
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    filters jsonb NOT NULL DEFAULT '{}' CHECK (
        CASE WHEN jsonb_typeof(filters) = 'object' THEN
            filters - array['state_id','state_group','priority','label_id','assignee_id','created_by_id',
                'mention_id','subscriber_id','start_date','target_date','module_id'] = '{}'::jsonb
            AND CASE WHEN NOT jsonb_path_exists(filters, 'strict $.* ? (@.type() != "array")') THEN
                NOT jsonb_path_exists(filters, 'strict $.*[*] ? (@.type() != "string")')
            ELSE false END
        ELSE false END),
    display_filters jsonb NOT NULL DEFAULT
        '{"layout":"list","group_by":null,"sub_group_by":null,"order_by":"-created_at","sub_issue":true,"show_empty_groups":true,"calendar":{"layout":"month","show_weekends":false}}'
        CHECK (CASE WHEN jsonb_typeof(display_filters) = 'object' THEN
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
    display_properties jsonb NOT NULL DEFAULT
        '{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true,"cycle":true,"modules":true}'
        CHECK (CASE WHEN jsonb_typeof(display_properties) = 'object' THEN
            display_properties ?& array['assignee','start_date','due_date','labels','key','priority','state','sub_issue_count','link','attachment_count','created_on','updated_on','cycle','modules']
            AND display_properties - array['assignee','start_date','due_date','labels','key','priority','state','sub_issue_count','link','attachment_count','created_on','updated_on','cycle','modules'] = '{}'::jsonb
            AND NOT jsonb_path_exists(display_properties, 'strict $.* ? (@.type() != "boolean")')
        ELSE false END),
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
-- 一个账户在一个迭代页至多一行未删除的；Plane 的 UNIQUE (cycle_id, user_id, deleted_at) 多余，删除
CREATE UNIQUE INDEX cycle_user_properties_cycle_id_user_id_key ON cycle_user_properties (cycle_id, user_id) WHERE deleted_at IS NULL;
-- 物理级联按它们找子行，所以不带条件
CREATE INDEX cycle_user_properties_cycle_id_idx ON cycle_user_properties (cycle_id);
CREATE INDEX cycle_user_properties_user_id_idx ON cycle_user_properties (user_id);
CREATE INDEX cycle_user_properties_workspace_id_idx ON cycle_user_properties (workspace_id);
CREATE INDEX cycle_user_properties_project_id_idx ON cycle_user_properties (project_id);
-- 60 天清理找候选
CREATE INDEX cycle_user_properties_deleted_at_idx ON cycle_user_properties (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP TABLE cycle_user_properties;
