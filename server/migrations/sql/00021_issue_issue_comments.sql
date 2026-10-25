-- Comments contain sanitized HTML and its searchable text projection.
-- +goose Up
CREATE TABLE issue_comments (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    actor_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    comment_html text NOT NULL DEFAULT '<p></p>',
    comment_stripped text NOT NULL DEFAULT '',
    edited_at timestamptz,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX issue_comments_issue_id_idx ON issue_comments (issue_id);
CREATE INDEX issue_comments_issue_id_created_at_idx ON issue_comments (issue_id, created_at, id) WHERE deleted_at IS NULL;
CREATE INDEX issue_comments_workspace_id_idx ON issue_comments (workspace_id);
CREATE INDEX issue_comments_project_id_idx ON issue_comments (project_id);

-- +goose Down
DROP TABLE issue_comments;
