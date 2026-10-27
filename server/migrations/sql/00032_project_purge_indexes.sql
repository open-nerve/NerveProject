-- +goose Up
CREATE INDEX project_user_properties_deleted_at_idx ON project_user_properties(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX project_members_deleted_at_idx ON project_members(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX labels_deleted_at_idx ON labels(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX states_deleted_at_idx ON states(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX projects_deleted_at_idx ON projects(deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP INDEX projects_deleted_at_idx;
DROP INDEX states_deleted_at_idx;
DROP INDEX labels_deleted_at_idx;
DROP INDEX project_members_deleted_at_idx;
DROP INDEX project_user_properties_deleted_at_idx;
