-- Project-owned display settings: the database checks shape, the issue
-- settings adapter checks words. Each top-level field is replaced whole.
-- +goose Up
ALTER TABLE project_user_properties
    ADD COLUMN filters jsonb NOT NULL DEFAULT '{}' CHECK (
        CASE WHEN jsonb_typeof(filters) = 'object' THEN
            filters - array['state_id','state_group','priority','label_id','assignee_id','created_by_id',
                'mention_id','subscriber_id','start_date','target_date'] = '{}'::jsonb
            AND CASE WHEN NOT jsonb_path_exists(filters, 'strict $.* ? (@.type() != "array")') THEN
                NOT jsonb_path_exists(filters, 'strict $.*[*] ? (@.type() != "string")')
            ELSE false END
        ELSE false END),
    ADD COLUMN display_filters jsonb NOT NULL DEFAULT
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
    ADD COLUMN display_properties jsonb NOT NULL DEFAULT
        '{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"created_on":true,"updated_on":true}'
        CHECK (CASE WHEN jsonb_typeof(display_properties) = 'object' THEN
            display_properties ?& array['assignee','start_date','due_date','labels','key','priority','state','sub_issue_count','link','created_on','updated_on']
            AND display_properties - array['assignee','start_date','due_date','labels','key','priority','state','sub_issue_count','link','created_on','updated_on'] = '{}'::jsonb
            AND NOT jsonb_path_exists(display_properties, 'strict $.* ? (@.type() != "boolean")')
        ELSE false END);

-- +goose Down
ALTER TABLE project_user_properties
    DROP COLUMN filters,
    DROP COLUMN display_filters,
    DROP COLUMN display_properties;
