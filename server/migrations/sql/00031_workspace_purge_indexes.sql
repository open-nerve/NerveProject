-- +goose Up
CREATE INDEX workspace_user_properties_deleted_at_idx ON workspace_user_properties(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX workspace_member_invites_deleted_at_idx ON workspace_member_invites(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX workspace_members_deleted_at_idx ON workspace_members(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX workspaces_deleted_at_idx ON workspaces(deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP INDEX workspaces_deleted_at_idx;
DROP INDEX workspace_members_deleted_at_idx;
DROP INDEX workspace_member_invites_deleted_at_idx;
DROP INDEX workspace_user_properties_deleted_at_idx;
