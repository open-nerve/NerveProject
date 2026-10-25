-- Description versions are saved atomically with the issue and its collaboration rows.
-- +goose Up
CREATE TABLE issue_description_versions (
 id uuid PRIMARY KEY,
 workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
 project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
 issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
 description_html text NOT NULL DEFAULT '<p></p>',
 description_stripped text,
 last_saved_at timestamptz NOT NULL,
 owned_by_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
 created_by_id uuid REFERENCES users ON DELETE SET NULL,
 updated_by_id uuid REFERENCES users ON DELETE SET NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 deleted_at timestamptz
);
CREATE INDEX issue_description_versions_issue_id_last_saved_at_idx ON issue_description_versions (issue_id, last_saved_at, id);
CREATE INDEX issue_description_versions_workspace_id_idx ON issue_description_versions (workspace_id);
CREATE INDEX issue_description_versions_project_id_idx ON issue_description_versions (project_id);
-- +goose Down
DROP TABLE issue_description_versions;
