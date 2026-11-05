-- Attachments join the issue display properties (M5/P7b): every stored choice
-- keeps its keys and shows the attachment count, as a new row does.
-- +goose Up
ALTER TABLE project_user_properties DROP CONSTRAINT project_user_properties_display_properties_check;
UPDATE project_user_properties SET display_properties = display_properties || '{"attachment_count":true}'::jsonb;
ALTER TABLE project_user_properties
    ALTER COLUMN display_properties SET DEFAULT
        '{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true}',
    ADD CONSTRAINT project_user_properties_display_properties_check CHECK (
        CASE WHEN jsonb_typeof(display_properties) = 'object' THEN
            display_properties ?& array['assignee','start_date','due_date','labels','key','priority','state','sub_issue_count','link','attachment_count','created_on','updated_on']
            AND display_properties - array['assignee','start_date','due_date','labels','key','priority','state','sub_issue_count','link','attachment_count','created_on','updated_on'] = '{}'::jsonb
            AND NOT jsonb_path_exists(display_properties, 'strict $.* ? (@.type() != "boolean")')
        ELSE false END);

-- +goose Down
ALTER TABLE project_user_properties DROP CONSTRAINT project_user_properties_display_properties_check;
UPDATE project_user_properties SET display_properties = display_properties - 'attachment_count';
ALTER TABLE project_user_properties
    ALTER COLUMN display_properties SET DEFAULT
        '{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"created_on":true,"updated_on":true}',
    ADD CONSTRAINT project_user_properties_display_properties_check CHECK (
        CASE WHEN jsonb_typeof(display_properties) = 'object' THEN
            display_properties ?& array['assignee','start_date','due_date','labels','key','priority','state','sub_issue_count','link','created_on','updated_on']
            AND display_properties - array['assignee','start_date','due_date','labels','key','priority','state','sub_issue_count','link','created_on','updated_on'] = '{}'::jsonb
            AND NOT jsonb_path_exists(display_properties, 'strict $.* ? (@.type() != "boolean")')
        ELSE false END);
