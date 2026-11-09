-- Cycles and modules join a project's issue list settings (M6/P3b spec R4):
-- the filters take cycle_id and module_id, and every stored choice keeps its
-- keys and shows a work item's cycle and modules, as a new row does.
-- +goose Up
ALTER TABLE project_user_properties
    DROP CONSTRAINT project_user_properties_filters_check,
    ADD CONSTRAINT project_user_properties_filters_check CHECK (
        CASE WHEN jsonb_typeof(filters) = 'object' THEN
            filters - array['state_id','state_group','priority','label_id','assignee_id','created_by_id',
                'mention_id','subscriber_id','start_date','target_date','cycle_id','module_id'] = '{}'::jsonb
            AND CASE WHEN NOT jsonb_path_exists(filters, 'strict $.* ? (@.type() != "array")') THEN
                NOT jsonb_path_exists(filters, 'strict $.*[*] ? (@.type() != "string")')
            ELSE false END
        ELSE false END);
ALTER TABLE project_user_properties DROP CONSTRAINT project_user_properties_display_properties_check;
UPDATE project_user_properties SET display_properties = display_properties || '{"cycle":true,"modules":true}'::jsonb;
ALTER TABLE project_user_properties
    ALTER COLUMN display_properties SET DEFAULT
        '{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true,"cycle":true,"modules":true}',
    ADD CONSTRAINT project_user_properties_display_properties_check CHECK (
        CASE WHEN jsonb_typeof(display_properties) = 'object' THEN
            display_properties ?& array['assignee','start_date','due_date','labels','key','priority','state','sub_issue_count','link','attachment_count','created_on','updated_on','cycle','modules']
            AND display_properties - array['assignee','start_date','due_date','labels','key','priority','state','sub_issue_count','link','attachment_count','created_on','updated_on','cycle','modules'] = '{}'::jsonb
            AND NOT jsonb_path_exists(display_properties, 'strict $.* ? (@.type() != "boolean")')
        ELSE false END);

-- +goose Down
-- The new checks go first: stripping the keys breaks the fourteen-property
-- one, and the old checks hold only once no row has the keys.
ALTER TABLE project_user_properties
    DROP CONSTRAINT project_user_properties_display_properties_check,
    DROP CONSTRAINT project_user_properties_filters_check;
UPDATE project_user_properties SET display_properties = display_properties - array['cycle','modules'],
    filters = filters - array['cycle_id','module_id'];
ALTER TABLE project_user_properties
    ALTER COLUMN display_properties SET DEFAULT
        '{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true}',
    ADD CONSTRAINT project_user_properties_filters_check CHECK (
        CASE WHEN jsonb_typeof(filters) = 'object' THEN
            filters - array['state_id','state_group','priority','label_id','assignee_id','created_by_id',
                'mention_id','subscriber_id','start_date','target_date'] = '{}'::jsonb
            AND CASE WHEN NOT jsonb_path_exists(filters, 'strict $.* ? (@.type() != "array")') THEN
                NOT jsonb_path_exists(filters, 'strict $.*[*] ? (@.type() != "string")')
            ELSE false END
        ELSE false END),
    ADD CONSTRAINT project_user_properties_display_properties_check CHECK (
        CASE WHEN jsonb_typeof(display_properties) = 'object' THEN
            display_properties ?& array['assignee','start_date','due_date','labels','key','priority','state','sub_issue_count','link','attachment_count','created_on','updated_on']
            AND display_properties - array['assignee','start_date','due_date','labels','key','priority','state','sub_issue_count','link','attachment_count','created_on','updated_on'] = '{}'::jsonb
            AND NOT jsonb_path_exists(display_properties, 'strict $.* ? (@.type() != "boolean")')
        ELSE false END);
